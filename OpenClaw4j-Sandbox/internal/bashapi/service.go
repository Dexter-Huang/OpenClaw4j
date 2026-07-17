package bashapi

import (
	"context"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/seaskyland/openclaw4j-sandbox/internal/pathguard"
)

type Service struct {
	guard            pathguard.Guard
	defaultTimeoutMs uint64
	hardTimeoutMs    uint64
	outputLimitBytes int

	mu       sync.Mutex
	sessions map[string]*sessionState
}

type Response struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
}

type ExecRequest struct {
	Command         string             `json:"command"`
	SessionID       string             `json:"session_id,omitempty"`
	ExecDir         string             `json:"exec_dir,omitempty"`
	Env             map[string]*string `json:"env,omitempty"`
	AsyncMode       *bool              `json:"async_mode,omitempty"`
	Timeout         *float64           `json:"timeout,omitempty"`
	HardTimeout     *float64           `json:"hard_timeout,omitempty"`
	MaxOutputLength *int               `json:"max_output_length,omitempty"`
}

type OutputRequest struct {
	SessionID    string   `json:"session_id"`
	CommandID    string   `json:"command_id,omitempty"`
	Offset       int      `json:"offset,omitempty"`
	StderrOffset int      `json:"stderr_offset,omitempty"`
	Wait         *bool    `json:"wait,omitempty"`
	WaitTimeout  *float64 `json:"wait_timeout,omitempty"`
}

type WriteRequest struct {
	SessionID string `json:"session_id"`
	CommandID string `json:"command_id,omitempty"`
	Input     string `json:"input"`
}

type KillRequest struct {
	SessionID string `json:"session_id"`
	CommandID string `json:"command_id,omitempty"`
	Signal    string `json:"signal,omitempty"`
}

type CreateSessionRequest struct {
	ExecDir string `json:"exec_dir,omitempty"`
}

type SessionInfo struct {
	SessionID string `json:"session_id"`
	ExecDir   string `json:"exec_dir"`
	Status    string `json:"status"`
}

type sessionState struct {
	sessionID        string
	execDir          string
	status           string
	currentCommandID string
	commands         map[string]*commandState
}

type commandState struct {
	mu       sync.Mutex
	id       string
	command  string
	status   string
	stdout   []byte
	stderr   []byte
	exitCode *int
	stdin    io.WriteCloser
	cancel   context.CancelFunc
	done     chan struct{}
}

func NewService(guard pathguard.Guard, defaultTimeoutMs uint64, hardTimeoutMs uint64, outputLimitBytes int) *Service {
	return &Service{
		guard:            guard,
		defaultTimeoutMs: defaultTimeoutMs,
		hardTimeoutMs:    hardTimeoutMs,
		outputLimitBytes: outputLimitBytes,
		sessions:         make(map[string]*sessionState),
	}
}

func (s *Service) Exec(req ExecRequest) Response {
	if strings.TrimSpace(req.Command) == "" {
		return fail("command cannot be empty", map[string]any{"error_type": "invalid_request"})
	}

	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = uuid.NewString()
	}
	execDir, err := s.resolveExecDir(req.ExecDir, sessionID)
	if err != nil {
		return bashPathError(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cmd := shellCommand(ctx, req.Command)
	cmd.Dir = execDir
	for key, value := range req.Env {
		if value != nil {
			cmd.Env = append(cmd.Env, key+"="+*value)
		}
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return fail(err.Error(), map[string]any{"error_type": "spawn_error"})
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return fail(err.Error(), map[string]any{"error_type": "spawn_error"})
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return fail(err.Error(), map[string]any{"error_type": "spawn_error"})
	}
	if err := cmd.Start(); err != nil {
		cancel()
		return fail(err.Error(), map[string]any{"error_type": "spawn_error"})
	}

	state := &commandState{
		id:      uuid.NewString(),
		command: req.Command,
		status:  "running",
		stdin:   stdin,
		cancel:  cancel,
		done:    make(chan struct{}),
	}
	s.registerCommand(sessionID, execDir, state)

	var readers sync.WaitGroup
	readers.Add(2)
	go readPipe(stdout, func(chunk []byte) { state.appendStdout(chunk, s.outputLimitBytes) }, &readers)
	go readPipe(stderr, func(chunk []byte) { state.appendStderr(chunk, s.outputLimitBytes) }, &readers)
	go s.waitCommand(cmd, state, &readers)
	go enforceHardTimeout(state, secondsToDuration(req.HardTimeout, s.hardTimeoutMs))

	if boolValue(req.AsyncMode) {
		return s.commandResponse(sessionID, state, req.MaxOutputLength, 0, 0)
	}

	select {
	case <-state.done:
	case <-time.After(secondsToDuration(req.Timeout, s.defaultTimeoutMs)):
	}
	return s.commandResponse(sessionID, state, req.MaxOutputLength, 0, 0)
}

func (s *Service) Output(req OutputRequest) Response {
	state, sessionID, ok := s.findCommand(req.SessionID, req.CommandID)
	if !ok {
		return fail("command not found", map[string]any{"error_type": "not_found"})
	}

	if boolValue(req.Wait) {
		timeout := secondsToDuration(req.WaitTimeout, 1000)
		deadline := time.After(timeout)
		for {
			state.mu.Lock()
			hasNewOutput := len(state.stdout) > req.Offset || len(state.stderr) > req.StderrOffset || state.status != "running"
			state.mu.Unlock()
			if hasNewOutput {
				break
			}
			select {
			case <-state.done:
				return s.commandResponse(sessionID, state, nil, req.Offset, req.StderrOffset)
			case <-deadline:
				return s.commandResponse(sessionID, state, nil, req.Offset, req.StderrOffset)
			case <-time.After(20 * time.Millisecond):
			}
		}
	}

	return s.commandResponse(sessionID, state, nil, req.Offset, req.StderrOffset)
}

func (s *Service) Write(req WriteRequest) Response {
	state, _, found := s.findCommand(req.SessionID, req.CommandID)
	if !found {
		return fail("command not found", map[string]any{"error_type": "not_found"})
	}

	state.mu.Lock()
	stdin := state.stdin
	state.mu.Unlock()
	if stdin == nil {
		return fail("stdin is closed", map[string]any{"error_type": "stdin_closed"})
	}
	n, err := io.WriteString(stdin, req.Input)
	if err != nil {
		return fail(err.Error(), map[string]any{"error_type": "stdin_write_error"})
	}
	return ok("Input written successfully", map[string]any{"bytes_written": n})
}

func (s *Service) Kill(req KillRequest) Response {
	state, sessionID, found := s.findCommand(req.SessionID, req.CommandID)
	if !found {
		return fail("command not found", map[string]any{"error_type": "not_found"})
	}

	state.mu.Lock()
	if state.status == "running" {
		state.status = "killed"
		state.cancel()
	}
	status := state.status
	state.mu.Unlock()

	return ok("Command killed successfully", map[string]any{
		"session_id": sessionID,
		"command_id": state.id,
		"status":     status,
	})
}

func (s *Service) Sessions() Response {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessions := make([]SessionInfo, 0, len(s.sessions))
	for _, session := range s.sessions {
		if session.status == "ready" {
			sessions = append(sessions, SessionInfo{
				SessionID: session.sessionID,
				ExecDir:   session.execDir,
				Status:    session.status,
			})
		}
	}
	return ok("Sessions listed successfully", map[string]any{"sessions": sessions})
}

func (s *Service) CreateSession(req CreateSessionRequest) Response {
	execDir := s.guard.Root()
	if req.ExecDir != "" {
		resolved, err := s.guard.Resolve(req.ExecDir)
		if err != nil {
			return bashPathError(err)
		}
		execDir = resolved
	}
	sessionID := uuid.NewString()

	s.mu.Lock()
	s.sessions[sessionID] = &sessionState{
		sessionID: sessionID,
		execDir:   execDir,
		status:    "ready",
		commands:  make(map[string]*commandState),
	}
	s.mu.Unlock()

	return ok("Session created successfully", map[string]any{
		"session_id": sessionID,
		"exec_dir":   execDir,
		"status":     "ready",
	})
}

func (s *Service) CloseSession(sessionID string) Response {
	s.mu.Lock()
	session, found := s.sessions[sessionID]
	if found {
		session.status = "closed"
		delete(s.sessions, sessionID)
	}
	s.mu.Unlock()
	if !found {
		return fail("session not found", map[string]any{"error_type": "not_found"})
	}

	for _, command := range session.commands {
		command.mu.Lock()
		if command.status == "running" {
			command.status = "killed"
			command.cancel()
		}
		command.mu.Unlock()
	}

	return ok("Session closed successfully", map[string]any{"session_id": sessionID, "status": "closed"})
}

func (s *Service) resolveExecDir(input string, sessionID string) (string, error) {
	if input != "" {
		return s.guard.Resolve(input)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if session, ok := s.sessions[sessionID]; ok {
		return session.execDir, nil
	}
	return s.guard.Root(), nil
}

func (s *Service) registerCommand(sessionID string, execDir string, command *commandState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[sessionID]
	if !ok {
		session = &sessionState{
			sessionID: sessionID,
			execDir:   execDir,
			status:    "ready",
			commands:  make(map[string]*commandState),
		}
		s.sessions[sessionID] = session
	}
	session.currentCommandID = command.id
	session.commands[command.id] = command
}

func (s *Service) findCommand(sessionID string, commandID string) (*commandState, string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, "", false
	}
	if commandID == "" {
		commandID = session.currentCommandID
	}
	command, ok := session.commands[commandID]
	return command, sessionID, ok
}

func (s *Service) waitCommand(cmd *exec.Cmd, state *commandState, readers *sync.WaitGroup) {
	err := cmd.Wait()
	readers.Wait()

	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	} else if err != nil {
		exitCode = 1
	}

	state.mu.Lock()
	if state.status == "running" {
		state.status = "completed"
		state.exitCode = &exitCode
	}
	if state.exitCode == nil && state.status == "completed" {
		state.exitCode = &exitCode
	}
	state.mu.Unlock()
	close(state.done)
}

func (s *Service) commandResponse(sessionID string, state *commandState, maxOutputLength *int, offset int, stderrOffset int) Response {
	state.mu.Lock()
	defer state.mu.Unlock()

	stdout := sliceOutput(state.stdout, offset)
	stderr := sliceOutput(state.stderr, stderrOffset)
	if maxOutputLength != nil && *maxOutputLength >= 0 {
		stdout = truncate(stdout, *maxOutputLength)
		stderr = truncate(stderr, *maxOutputLength)
	}
	exitCode := any(nil)
	if state.exitCode != nil {
		exitCode = *state.exitCode
	}

	return ok("Command output collected successfully", map[string]any{
		"session_id":    sessionID,
		"command_id":    state.id,
		"command":       state.command,
		"status":        state.status,
		"stdout":        string(stdout),
		"stderr":        string(stderr),
		"exit_code":     exitCode,
		"offset":        len(state.stdout),
		"stderr_offset": len(state.stderr),
	})
}

func readPipe(reader io.Reader, appendChunk func([]byte), wg *sync.WaitGroup) {
	defer wg.Done()
	buffer := make([]byte, 4096)
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buffer[:n])
			appendChunk(chunk)
		}
		if err != nil {
			return
		}
	}
}

func (c *commandState) appendStdout(chunk []byte, limit int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stdout = appendLimited(c.stdout, chunk, limit)
}

func (c *commandState) appendStderr(chunk []byte, limit int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stderr = appendLimited(c.stderr, chunk, limit)
}

func enforceHardTimeout(state *commandState, duration time.Duration) {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-state.done:
		return
	case <-timer.C:
		state.mu.Lock()
		if state.status == "running" {
			state.status = "timed_out"
			state.cancel()
		}
		state.mu.Unlock()
	}
}

func shellCommand(ctx context.Context, command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "powershell", "-NoLogo", "-NoProfile", "-Command", command)
	}
	return exec.CommandContext(ctx, "/bin/bash", "-lc", command)
}

func appendLimited(current []byte, chunk []byte, limit int) []byte {
	if limit <= 0 {
		return append(current, chunk...)
	}
	combined := append(current, chunk...)
	if len(combined) <= limit {
		return combined
	}
	return combined[len(combined)-limit:]
}

func sliceOutput(output []byte, offset int) []byte {
	if offset < 0 {
		offset = 0
	}
	if offset > len(output) {
		offset = len(output)
	}
	return output[offset:]
}

func truncate(output []byte, limit int) []byte {
	if limit < 0 || len(output) <= limit {
		return output
	}
	return output[len(output)-limit:]
}

func secondsToDuration(seconds *float64, fallbackMs uint64) time.Duration {
	if seconds == nil {
		return time.Duration(fallbackMs) * time.Millisecond
	}
	return time.Duration(*seconds * float64(time.Second))
}

func boolValue(value *bool) bool {
	return value != nil && *value
}

func ok(message string, data map[string]any) Response {
	return Response{Success: true, Message: message, Data: data}
}

func fail(message string, data map[string]any) Response {
	if data == nil {
		data = map[string]any{}
	}
	data["message"] = message
	return Response{Success: false, Message: message, Data: data}
}

func bashPathError(err error) Response {
	return fail(err.Error(), map[string]any{
		"error_type": "invalid_path",
		"retryable":  false,
	})
}
