package mcpserver

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/seaskyland/openclaw4j-backend-go/internal/dao"
)

var (
	ErrNameRequired          = errors.New("serverName is required")
	ErrDeployConfigRequired  = errors.New("deployConfig is required")
	ErrServerCodeRequired    = errors.New("serverCode is required")
	ErrNotFound              = errors.New("mcp server not found")
	ErrToolNameRequired      = errors.New("tool_name is required")
	ErrRemoteAddressRequired = errors.New("deploy_config.remote_address is required")
	ErrCommandRequired       = errors.New("deploy_config.command is required for stdio transport")
)

type Input struct {
	ServerCode   string  `json:"server_code"`
	Name         string  `json:"name"`
	DeployConfig string  `json:"deploy_config"`
	DetailConfig *string `json:"detail_config"`
	Status       *int16  `json:"status"`
	Type         *string `json:"type"`
	BizType      *string `json:"biz_type"`
	Description  *string `json:"description"`
	InstallType  *string `json:"install_type"`
	DeployEnv    *string `json:"deploy_env"`
	Source       *string `json:"source"`
	NeedTools    bool    `json:"need_tools"`
}

type Server struct {
	ServerCode   string    `json:"server_code"`
	Name         string    `json:"name"`
	DeployConfig string    `json:"deploy_config"`
	DetailConfig *string   `json:"detail_config,omitempty"`
	Status       int16     `json:"status"`
	Type         string    `json:"type"`
	BizType      *string   `json:"biz_type,omitempty"`
	Description  *string   `json:"description,omitempty"`
	InstallType  *string   `json:"install_type,omitempty"`
	DeployEnv    *string   `json:"deploy_env,omitempty"`
	Source       *string   `json:"source,omitempty"`
	Tools        []any     `json:"tools,omitempty"`
	GmtModified  time.Time `json:"gmt_modified"`
}

type Page struct {
	Current int64    `json:"current"`
	Size    int64    `json:"size"`
	Total   int64    `json:"total"`
	Records []Server `json:"records"`
}

type ToolCallInput struct {
	ServerCode string         `json:"server_code"`
	ToolName   string         `json:"tool_name"`
	ToolParams map[string]any `json:"tool_params"`
}

type ToolCallResult struct {
	IsError bool             `json:"is_error"`
	Content []map[string]any `json:"content"`
}

type Service struct {
	dao    dao.McpServerDAO
	clock  func() time.Time
	client *http.Client
}

func NewService(data dao.McpServerDAO, clock func() time.Time) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{dao: data, clock: clock, client: &http.Client{Timeout: 60 * time.Second}}
}

func (s *Service) CallTool(ctx context.Context, workspaceID string, input ToolCallInput) (*ToolCallResult, error) {
	if strings.TrimSpace(input.ServerCode) == "" {
		return nil, ErrServerCodeRequired
	}
	if strings.TrimSpace(input.ToolName) == "" {
		return nil, ErrToolNameRequired
	}
	server, err := s.dao.Find(ctx, input.ServerCode, workspaceID)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	body, err := s.call(ctx, *server, "tools/call", map[string]any{"name": input.ToolName, "arguments": input.ToolParams})
	if err != nil {
		return nil, err
	}
	return parseToolCallResponse(body)
}

func (s *Service) Create(ctx context.Context, workspaceID, accountID string, input Input) (string, error) {
	if strings.TrimSpace(input.Name) == "" {
		return "", ErrNameRequired
	}
	if strings.TrimSpace(input.DeployConfig) == "" {
		return "", ErrDeployConfigRequired
	}
	code, err := newServerCode()
	if err != nil {
		return "", err
	}
	now := s.clock()
	value := dao.McpServer{ServerCode: code, WorkspaceID: workspaceID, AccountID: accountID, Name: strings.TrimSpace(input.Name), DeployConfig: strings.TrimSpace(input.DeployConfig), DetailConfig: input.DetailConfig, Status: 1, Type: defaultValue(input.Type, "sse"), BizType: input.BizType, Description: input.Description, InstallType: defaultPointer(input.InstallType, "SSE"), DeployEnv: input.DeployEnv, Source: input.Source, GmtCreate: now, GmtModified: now}
	if input.Status != nil {
		value.Status = *input.Status
	}
	if err := s.dao.Create(ctx, value); err != nil {
		return "", err
	}
	return code, nil
}

func (s *Service) Update(ctx context.Context, workspaceID string, input Input) error {
	if strings.TrimSpace(input.ServerCode) == "" {
		return ErrServerCodeRequired
	}
	if strings.TrimSpace(input.Name) == "" {
		return ErrNameRequired
	}
	if strings.TrimSpace(input.DeployConfig) == "" {
		return ErrDeployConfigRequired
	}
	value, err := s.dao.Find(ctx, input.ServerCode, workspaceID)
	if errors.Is(err, dao.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	value.Name = strings.TrimSpace(input.Name)
	value.DeployConfig = strings.TrimSpace(input.DeployConfig)
	value.DetailConfig = input.DetailConfig
	value.Type = defaultValue(input.Type, value.Type)
	value.BizType = input.BizType
	value.Description = input.Description
	value.InstallType = defaultPointer(input.InstallType, pointerValue(value.InstallType, "SSE"))
	value.DeployEnv = input.DeployEnv
	value.Source = input.Source
	if input.Status != nil {
		value.Status = *input.Status
	}
	value.GmtModified = s.clock()
	return s.dao.Update(ctx, *value)
}

func (s *Service) Get(ctx context.Context, workspaceID, serverCode string, needTools bool) (*Server, error) {
	value, err := s.dao.Find(ctx, serverCode, workspaceID)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.toServer(ctx, *value, needTools)
}

func (s *Service) List(ctx context.Context, workspaceID, name string, needTools bool, current, size int64) (*Page, error) {
	if current < 1 {
		current = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	values, err := s.dao.List(ctx, workspaceID, strings.TrimSpace(name))
	if err != nil {
		return nil, err
	}
	all := make([]Server, 0, len(values))
	for _, value := range values {
		mapped, err := s.toServer(ctx, value, needTools)
		if err != nil {
			return nil, err
		}
		all = append(all, *mapped)
	}
	start := (current - 1) * size
	if start >= int64(len(all)) {
		return &Page{Current: current, Size: size, Total: int64(len(all)), Records: []Server{}}, nil
	}
	end := start + size
	if end > int64(len(all)) {
		end = int64(len(all))
	}
	return &Page{Current: current, Size: size, Total: int64(len(all)), Records: all[start:end]}, nil
}

func (s *Service) ListByCodes(ctx context.Context, workspaceID string, codes []string, needTools bool) ([]Server, error) {
	values, err := s.dao.List(ctx, workspaceID, "")
	if err != nil {
		return nil, err
	}
	selected := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		selected[strings.TrimSpace(code)] = struct{}{}
	}
	result := make([]Server, 0, len(codes))
	for _, value := range values {
		if _, ok := selected[value.ServerCode]; ok {
			mapped, err := s.toServer(ctx, value, needTools)
			if err != nil {
				return nil, err
			}
			result = append(result, *mapped)
		}
	}
	return result, nil
}

func (s *Service) Delete(ctx context.Context, workspaceID, serverCode string) error {
	if strings.TrimSpace(serverCode) == "" {
		return ErrServerCodeRequired
	}
	return s.dao.Delete(ctx, serverCode, workspaceID, s.clock())
}

func (s *Service) toServer(ctx context.Context, value dao.McpServer, needTools bool) (*Server, error) {
	result := mapServer(value)
	if !needTools {
		return result, nil
	}
	tools, err := s.listTools(ctx, value)
	if err != nil {
		return nil, err
	}
	result.Tools = tools
	return result, nil
}

func mapServer(value dao.McpServer) *Server {
	result := &Server{ServerCode: value.ServerCode, Name: value.Name, DeployConfig: value.DeployConfig, DetailConfig: value.DetailConfig, Status: value.Status, Type: value.Type, BizType: value.BizType, Description: value.Description, InstallType: value.InstallType, DeployEnv: value.DeployEnv, Source: value.Source, GmtModified: value.GmtModified}
	return result
}

func (s *Service) listTools(ctx context.Context, server dao.McpServer) ([]any, error) {
	body, err := s.call(ctx, server, "tools/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	return parseToolListResponse(body)
}

func defaultValue(value *string, fallback string) string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return fallback
	}
	return strings.TrimSpace(*value)
}

func defaultPointer(value *string, fallback string) *string {
	result := defaultValue(value, fallback)
	return &result
}

func pointerValue(value *string, fallback string) string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return fallback
	}
	return *value
}

func newServerCode() (string, error) {
	var data [12]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	return "mcp_" + hex.EncodeToString(data[:]), nil
}

type deployConfig struct {
	RemoteAddress  string            `json:"remote_address"`
	RemoteEndpoint string            `json:"remote_endpoint"`
	RemoteHeader   map[string]string `json:"remote_header"`
	Transport      string            `json:"transport"`
	Command        string            `json:"command"`
	Args           []string          `json:"args"`
	WorkingDir     string            `json:"working_dir"`
}

func parseDeployConfig(value string) (deployConfig, error) {
	var config deployConfig
	if err := json.Unmarshal([]byte(value), &config); err != nil {
		return deployConfig{}, fmt.Errorf("invalid deploy_config: %w", err)
	}
	return config, nil
}

func streamableEndpoint(address, endpoint string) (string, error) {
	base, err := url.Parse(strings.TrimSpace(address))
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" {
		return "", errors.New("deploy_config.remote_address must be an http(s) URL")
	}
	if strings.TrimSpace(endpoint) == "" {
		endpoint = "/mcp"
	}
	relative, err := url.Parse(endpoint)
	if err != nil || relative.IsAbs() {
		return "", errors.New("deploy_config.remote_endpoint must be a relative path")
	}
	return base.ResolveReference(relative).String(), nil
}

// call selects a JSON-RPC transport from the persisted server type/config. All
// transports return one JSON-RPC result body so tool parsing and workflow nodes do
// not need transport-specific branches.
func (s *Service) call(ctx context.Context, server dao.McpServer, method string, params map[string]any) ([]byte, error) {
	config, err := parseDeployConfig(server.DeployConfig)
	if err != nil {
		return nil, err
	}
	transport := resolveTransport(config, server)
	switch transport {
	case "stdio", "standard_io":
		return s.callStdio(ctx, config, method, params)
	case "sse":
		return s.callSSE(ctx, config, method, params)
	case "", "streamable-http", "streamable_http", "http":
		return s.callHTTP(ctx, config, method, params, "")
	default:
		return nil, fmt.Errorf("unsupported MCP transport: %s", transport)
	}
}

// resolveTransport 需要兼容 Java 侧已持久化的 MCP 记录：其 type 字段可能是
// CUSTOMER 等业务分类，并不表示传输协议。显式 transport 始终优先；只有 type
// 本身是已知协议时才采用，最后才回退到 install_type，避免旧记录在读取详情时
// 被误判为不支持的传输类型。
func resolveTransport(config deployConfig, server dao.McpServer) string {
	for _, value := range []string{config.Transport, server.Type, pointerValue(server.InstallType, "")} {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "stdio", "standard_io":
			return "stdio"
		case "sse":
			return "sse"
		case "streamable-http", "streamable_http", "http":
			return "streamable-http"
		}
	}
	return ""
}

func (s *Service) callHTTP(ctx context.Context, config deployConfig, method string, params map[string]any, endpointOverride string) ([]byte, error) {
	if strings.TrimSpace(config.RemoteAddress) == "" {
		return nil, ErrRemoteAddressRequired
	}
	endpoint := endpointOverride
	if endpoint == "" {
		var err error
		endpoint, err = streamableEndpoint(config.RemoteAddress, config.RemoteEndpoint)
		if err != nil {
			return nil, err
		}
	}
	payload, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": fmt.Sprintf("go-%d", s.clock().UnixNano()), "method": method, "params": params})
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	for name, value := range config.RemoteHeader {
		if strings.TrimSpace(name) != "" {
			request.Header.Set(name, value)
		}
	}
	response, err := s.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("MCP HTTP request failed with status %d", response.StatusCode)
	}
	return body, nil
}

// callSSE supports the legacy MCP SSE handshake: GET an event stream, read its
// endpoint event, POST JSON-RPC to that session endpoint, then read the resulting
// message event from the same stream. A configured endpoint that is not an SSE URL
// is still accepted as a direct POST endpoint for existing deployments.
func (s *Service) callSSE(ctx context.Context, config deployConfig, method string, params map[string]any) ([]byte, error) {
	if strings.TrimSpace(config.RemoteAddress) == "" {
		return nil, ErrRemoteAddressRequired
	}
	configuredEndpoint := strings.TrimSpace(config.RemoteEndpoint)
	if configuredEndpoint != "" && !strings.HasSuffix(strings.TrimRight(strings.ToLower(configuredEndpoint), "/"), "/sse") {
		return s.callHTTP(ctx, config, method, params, "")
	}
	handshakeEndpoint, err := sseHandshakeEndpoint(config.RemoteAddress, configuredEndpoint)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, handshakeEndpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "text/event-stream")
	for name, value := range config.RemoteHeader {
		if strings.TrimSpace(name) != "" {
			request.Header.Set(name, value)
		}
	}
	response, err := s.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("MCP SSE handshake failed with status %d", response.StatusCode)
	}
	endpoint, scanner, err := readSSEEndpoint(response.Body)
	if err != nil {
		return nil, err
	}
	relative, err := url.Parse(endpoint)
	if err != nil || relative.IsAbs() {
		return nil, errors.New("MCP SSE endpoint must be a relative path")
	}
	if _, err := s.callHTTP(ctx, config, method, params, response.Request.URL.ResolveReference(relative).String()); err != nil {
		return nil, err
	}
	return readSSEMessage(scanner)
}

func sseHandshakeEndpoint(address, endpoint string) (string, error) {
	base, err := url.Parse(strings.TrimSpace(address))
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" {
		return "", errors.New("deploy_config.remote_address must be an http(s) URL")
	}
	if endpoint == "" {
		base.Path = strings.TrimRight(base.Path, "/") + "/sse"
		return base.String(), nil
	}
	relative, err := url.Parse(endpoint)
	if err != nil || relative.IsAbs() {
		return "", errors.New("deploy_config.remote_endpoint must be a relative path")
	}
	return base.ResolveReference(relative).String(), nil
}

func readSSEEndpoint(reader io.Reader) (string, *bufio.Scanner, error) {
	scanner := bufio.NewScanner(io.LimitReader(reader, 64<<10))
	scanner.Buffer(make([]byte, 1024), 64<<10)
	event := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "event:") {
			event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if strings.HasPrefix(line, "data:") && (event == "" || event == "endpoint") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if value != "" {
				return value, scanner, nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", nil, err
	}
	return "", nil, errors.New("MCP SSE handshake did not provide an endpoint")
}

func readSSEMessage(scanner *bufio.Scanner) ([]byte, error) {
	event := ""
	data := make([]string, 0, 1)
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if line == "" {
			if event == "message" && len(data) > 0 {
				return []byte(strings.Join(data, "\n")), nil
			}
			event = ""
			data = data[:0]
			continue
		}
		if strings.HasPrefix(line, "event:") {
			event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if strings.HasPrefix(line, "data:") {
			data = append(data, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nil, errors.New("MCP SSE stream closed before responding")
}

func (s *Service) callStdio(ctx context.Context, config deployConfig, method string, params map[string]any) ([]byte, error) {
	if strings.TrimSpace(config.Command) == "" {
		return nil, ErrCommandRequired
	}
	command := exec.CommandContext(ctx, config.Command, config.Args...)
	command.Dir = strings.TrimSpace(config.WorkingDir)
	stdin, err := command.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("start MCP stdio process: %w", err)
	}
	defer func() {
		_ = stdin.Close()
		if command.Process != nil {
			_ = command.Process.Kill()
		}
		_ = command.Wait()
	}()
	scanner := bufio.NewScanner(io.LimitReader(stdout, 4<<20))
	scanner.Buffer(make([]byte, 16<<10), 1<<20)
	write := func(id any, rpcMethod string, rpcParams any) error {
		value, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": rpcMethod, "params": rpcParams})
		if err != nil {
			return err
		}
		_, err = stdin.Write(append(value, '\n'))
		return err
	}
	if err := write(1, "initialize", map[string]any{"protocolVersion": "2024-11-05", "capabilities": map[string]any{}, "clientInfo": map[string]string{"name": "openclaw4j-go", "version": "1"}}); err != nil {
		return nil, err
	}
	if _, err := readRPCResponse(scanner, 1); err != nil {
		return nil, err
	}
	notification, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"})
	if _, err := stdin.Write(append(notification, '\n')); err != nil {
		return nil, err
	}
	requestID := 2
	if err := write(requestID, method, params); err != nil {
		return nil, err
	}
	return readRPCResponse(scanner, requestID)
}

func readRPCResponse(scanner *bufio.Scanner, expectedID int) ([]byte, error) {
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var envelope struct {
			ID any `json:"id"`
		}
		if err := json.Unmarshal(line, &envelope); err != nil {
			return nil, fmt.Errorf("invalid MCP stdio response: %w", err)
		}
		if number, ok := envelope.ID.(float64); ok && int(number) == expectedID {
			return append([]byte(nil), line...), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nil, errors.New("MCP stdio process closed before responding")
}

func parseToolCallResponse(body []byte) (*ToolCallResult, error) {
	value := strings.TrimSpace(string(body))
	if strings.HasPrefix(value, "data:") || strings.HasPrefix(value, "event:") {
		value = firstSSEData(value)
	}
	if value == "" {
		return &ToolCallResult{IsError: true, Content: []map[string]any{{"type": "text", "text": "Empty MCP response"}}}, nil
	}
	var response struct {
		Error  map[string]any `json:"error"`
		Result struct {
			IsError    bool             `json:"isError"`
			SnakeError bool             `json:"is_error"`
			Content    []map[string]any `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(value), &response); err != nil {
		return nil, fmt.Errorf("invalid MCP response: %w", err)
	}
	if response.Error != nil {
		return &ToolCallResult{IsError: true, Content: []map[string]any{{"type": "text", "text": fmt.Sprint(response.Error["message"])}}}, nil
	}
	if response.Result.Content == nil {
		return &ToolCallResult{IsError: true, Content: []map[string]any{{"type": "text", "text": "Empty MCP response"}}}, nil
	}
	return &ToolCallResult{IsError: response.Result.IsError || response.Result.SnakeError, Content: response.Result.Content}, nil
}

func parseToolListResponse(body []byte) ([]any, error) {
	value := strings.TrimSpace(string(body))
	if strings.HasPrefix(value, "data:") || strings.HasPrefix(value, "event:") {
		value = firstSSEData(value)
	}
	var response struct {
		Error  map[string]any `json:"error"`
		Result struct {
			Tools []any `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(value), &response); err != nil {
		return nil, fmt.Errorf("invalid MCP response: %w", err)
	}
	if response.Error != nil {
		return nil, fmt.Errorf("MCP tools/list failed: %v", response.Error["message"])
	}
	if response.Result.Tools == nil {
		return []any{}, nil
	}
	return normalizeToolSchemas(response.Result.Tools), nil
}

// normalizeToolSchemas 保留 MCP 标准的 inputSchema，同时补充旧管理端表单使用的
// input_schema。工具来源由第三方 MCP Server 决定，不能要求其为某个管理端前端调整
// 字段命名；双字段输出可兼容标准客户端和既有管理端，而不会丢失原始 schema。
func normalizeToolSchemas(tools []any) []any {
	for index, rawTool := range tools {
		tool, ok := rawTool.(map[string]any)
		if !ok || tool["input_schema"] != nil {
			continue
		}
		if inputSchema, exists := tool["inputSchema"]; exists {
			tool["input_schema"] = inputSchema
			tools[index] = tool
		}
	}
	return tools
}

func firstSSEData(body string) string {
	var lines []string
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSuffix(line, "\r")
		if line == "" && len(lines) > 0 {
			break
		}
		if strings.HasPrefix(line, "data:") {
			lines = append(lines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	return strings.Join(lines, "\n")
}
