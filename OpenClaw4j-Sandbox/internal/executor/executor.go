package executor

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/seaskyland/openclaw4j-sandbox/internal/config"
	"github.com/seaskyland/openclaw4j-sandbox/internal/model"
	"github.com/seaskyland/openclaw4j-sandbox/internal/wrapper"
)

type Executor struct {
	cfg     config.Config
	runtime Runtime
}

func New(cfg config.Config, runtime Runtime) *Executor {
	return &Executor{cfg: cfg, runtime: runtime}
}

func (e *Executor) Execute(ctx context.Context, req model.ExecuteRequest) model.ExecuteResponse {
	started := time.Now()
	requestID := "<none>"
	if req.RequestID != nil {
		requestID = *req.RequestID
	}
	slog.Debug("script execution request received", "request_id", requestID, "language", req.Language, "code_bytes", len(req.Code))

	language, err := wrapper.ValidateRequest(req)
	if err != nil {
		slog.Warn("script execution request rejected", "request_id", requestID, "language", req.Language, "error", err.Error())
		return model.Error("INVALID_REQUEST", err.Error(), "", "", nil, elapsedMs(started))
	}

	res, err := e.executeValidated(ctx, req, language, started)
	if err != nil {
		out := model.Error("SANDBOX_ERROR", err.Error(), "", "", nil, elapsedMs(started))
		logResponse(requestID, req.Language, out)
		return out
	}
	logResponse(requestID, req.Language, res)
	return res
}

func (e *Executor) executeValidated(ctx context.Context, req model.ExecuteRequest, language wrapper.Language, started time.Time) (model.ExecuteResponse, error) {
	if err := os.MkdirAll(e.cfg.WorkDir, 0o755); err != nil {
		return model.ExecuteResponse{}, err
	}
	prefix := uuid.NewString()
	if req.RequestID != nil {
		if sanitized := SanitizeName(*req.RequestID); sanitized != "" {
			prefix = sanitized
		}
	}
	workDir, err := os.MkdirTemp(e.cfg.WorkDir, prefix+"-")
	if err != nil {
		return model.ExecuteResponse{}, err
	}
	defer os.RemoveAll(workDir)

	if err := writeExecutionFiles(workDir, language, req); err != nil {
		return model.ExecuteResponse{}, err
	}

	timeoutMs := e.cfg.ClampTimeout(req.TimeoutMs)
	slog.Info("script execution started", "request_id", prefix, "language", string(language), "timeout_ms", timeoutMs, "memory_limit", e.cfg.MemoryLimit, "process_limit", e.cfg.ProcessLimit, "work_dir", workDir)

	runtimeResult, err := e.runtime.Run(ctx, e.cfg, language, workDir, timeoutMs)
	if err != nil {
		return model.ExecuteResponse{}, err
	}
	if runtimeResult.Timeout {
		return model.Error("TIMEOUT", "script execution timed out", "", "", nil, elapsedMs(started)), nil
	}

	stdout := LimitBytes(runtimeResult.Stdout, e.cfg.StdoutLimitBytes)
	stderr := LimitBytes(runtimeResult.Stderr, e.cfg.StderrLimitBytes)
	if !runtimeResult.Success {
		return model.Error("SCRIPT_ERROR", "script execution failed", stdout, stderr, runtimeResult.ExitCode, elapsedMs(started)), nil
	}

	resultText, err := os.ReadFile(filepath.Join(workDir, "result.json"))
	if err != nil {
		return model.ExecuteResponse{}, err
	}
	var data any
	if err := json.Unmarshal(resultText, &data); err != nil {
		return model.ExecuteResponse{}, err
	}
	exitCode := 0
	if runtimeResult.ExitCode != nil {
		exitCode = *runtimeResult.ExitCode
	}
	return model.Success(data, stdout, stderr, exitCode, elapsedMs(started)), nil
}

func writeExecutionFiles(workDir string, language wrapper.Language, req model.ExecuteRequest) error {
	if err := os.WriteFile(filepath.Join(workDir, language.UserFileName()), []byte(req.Code), 0o644); err != nil {
		return err
	}
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(workDir, "params.json"), paramsBytes, 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(workDir, language.RunnerFileName()), []byte(wrapper.BuildRunner(language, workDir)), 0o644)
}

func elapsedMs(started time.Time) uint64 {
	return uint64(time.Since(started).Milliseconds())
}

func logResponse(requestID string, language string, response model.ExecuteResponse) {
	attrs := []any{
		"request_id", requestID,
		"language", language,
		"duration_ms", response.DurationMs,
		"stdout_bytes", len(response.Stdout),
		"stderr_bytes", len(response.Stderr),
	}
	if response.ExitCode != nil {
		attrs = append(attrs, "exit_code", *response.ExitCode)
	}
	if response.Success {
		slog.Info("script execution completed", attrs...)
		return
	}
	if response.Code != nil {
		attrs = append(attrs, "code", *response.Code)
	}
	slog.Warn("script execution failed", attrs...)
}
