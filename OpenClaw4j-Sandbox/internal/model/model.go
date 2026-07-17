package model

type ExecuteRequest struct {
	RequestID *string `json:"request_id,omitempty"`
	Language  string  `json:"language"`
	Code      string  `json:"code"`
	Params    any     `json:"params,omitempty"`
	TimeoutMs *uint64 `json:"timeout_ms,omitempty"`
}

type ExecuteResponse struct {
	Success    bool    `json:"success"`
	Data       any     `json:"data,omitempty"`
	Message    *string `json:"message,omitempty"`
	Code       *string `json:"code,omitempty"`
	Stdout     string  `json:"stdout"`
	Stderr     string  `json:"stderr"`
	ExitCode   *int    `json:"exit_code,omitempty"`
	DurationMs uint64  `json:"duration_ms"`
}

func Success(data any, stdout string, stderr string, exitCode int, durationMs uint64) ExecuteResponse {
	return ExecuteResponse{
		Success:    true,
		Data:       data,
		Stdout:     stdout,
		Stderr:     stderr,
		ExitCode:   &exitCode,
		DurationMs: durationMs,
	}
}

func Error(code string, message string, stdout string, stderr string, exitCode *int, durationMs uint64) ExecuteResponse {
	return ExecuteResponse{
		Success:    false,
		Message:    &message,
		Code:       &code,
		Stdout:     stdout,
		Stderr:     stderr,
		ExitCode:   exitCode,
		DurationMs: durationMs,
	}
}
