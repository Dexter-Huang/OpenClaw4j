// Package scriptsandbox provides the workflow-facing client for the isolated script service.
package scriptsandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultTimeout = 30 * time.Second

type Client struct {
	baseURL string
	http    *http.Client
	timeout uint64
}

type Executor interface {
	Execute(context.Context, string, string, map[string]any, string) (*Result, error)
}

type Result struct {
	Success bool
	Data    any
	Message string
	Code    string
}

type executeRequest struct {
	RequestID string         `json:"request_id,omitempty"`
	Language  string         `json:"language"`
	Code      string         `json:"code"`
	Params    map[string]any `json:"params"`
	TimeoutMs uint64         `json:"timeout_ms"`
}

type executeResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		http:    &http.Client{Timeout: timeout},
		timeout: uint64(timeout.Milliseconds()),
	}
}

func (c *Client) Execute(ctx context.Context, language, code string, params map[string]any, requestID string) (*Result, error) {
	if c == nil || c.baseURL == "" {
		return nil, fmt.Errorf("script sandbox is not configured")
	}
	body, err := json.Marshal(executeRequest{
		RequestID: requestID,
		Language:  language,
		Code:      code,
		Params:    params,
		TimeoutMs: c.timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("encode sandbox request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/execute", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create sandbox request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	response, err := c.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("call script sandbox: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("script sandbox returned HTTP %d", response.StatusCode)
	}

	var payload executeResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode sandbox response: %w", err)
	}
	return &Result{Success: payload.Success, Data: payload.Data, Message: payload.Message, Code: payload.Code}, nil
}
