package chat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/seaskyland/openclaw4j-backend-go/internal/dao"
)

// StreamEvent maps one upstream OpenAI-compatible SSE delta into the public chat
// shape. Done is explicit so every transport can emit a terminal event reliably.
type StreamEvent struct {
	Response Response `json:"response"`
	Done     bool     `json:"done"`
	Error    string   `json:"error,omitempty"`
}

// Streamer is intentionally optional. Existing fakes and custom chat managers keep
// working while production Service exposes true incremental model output.
type Streamer interface {
	CompleteStream(context.Context, string, Request) (<-chan StreamEvent, error)
}

func (s *Service) CompleteStream(ctx context.Context, workspaceID string, input Request) (<-chan StreamEvent, error) {
	if strings.TrimSpace(input.AppID) == "" {
		return nil, ErrAppIDRequired
	}
	if len(input.Messages) == 0 {
		return nil, ErrMessagesRequired
	}
	app, err := s.apps.Get(ctx, workspaceID, input.AppID)
	if err != nil {
		return nil, err
	}
	configuration := app.Config
	if !input.IsDraft {
		if len(app.PubConfig) == 0 {
			return nil, ErrPublishedAppRequired
		}
		configuration = app.PubConfig
	}
	providerCode := stringValue(configuration["model_provider"])
	modelID := modelID(configuration["model"])
	if providerCode == "" {
		return nil, ErrModelProviderMissing
	}
	if modelID == "" {
		return nil, ErrModelMissing
	}
	if _, err := s.models.FindByProviderAndIDAndWorkspace(ctx, providerCode, modelID, workspaceID); err != nil {
		if errors.Is(err, dao.ErrNotFound) {
			return nil, ErrModelMissing
		}
		return nil, err
	}
	provider, err := s.providers.FindByCodeAndWorkspace(ctx, providerCode, workspaceID)
	if err != nil {
		return nil, err
	}
	credential, err := s.credential(provider.Credential)
	if err != nil {
		return nil, err
	}
	endpoint, err := completionEndpoint(stringValue(credential["endpoint"]), stringValue(credential["completions_path"]))
	if err != nil {
		return nil, err
	}
	body := map[string]any{"model": modelID, "messages": input.Messages, "stream": true, "stream_options": map[string]any{"include_usage": true}}
	for key, value := range mapValue(configuration["parameter"]) {
		body[key] = value
	}
	for key, value := range input.Parameter {
		body[key] = value
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "text/event-stream")
	request.Header.Set("Authorization", "Bearer "+credential["api_key"])
	response, err := s.client.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		response.Body.Close()
		return nil, fmt.Errorf("model stream request failed with status %d", response.StatusCode)
	}
	events := make(chan StreamEvent)
	go func() {
		defer close(events)
		defer response.Body.Close()
		scanner := bufio.NewScanner(response.Body)
		scanner.Buffer(make([]byte, 16*1024), 2<<20)
		requestID := ""
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if payload == "[DONE]" {
				events <- StreamEvent{Response: Response{RequestID: requestID, ConversationID: input.ConversationID}, Done: true}
				return
			}
			value, ok := parseStreamChunk(payload, input.ConversationID)
			if !ok {
				continue
			}
			if value.RequestID != "" {
				requestID = value.RequestID
			}
			select {
			case events <- StreamEvent{Response: value}:
			case <-ctx.Done():
				return
			}
		}
		// A few compatible providers omit [DONE]. Closing the HTTP stream is still a
		// successful terminal condition, and clients must not wait indefinitely.
		select {
		case events <- StreamEvent{Response: Response{RequestID: requestID, ConversationID: input.ConversationID}, Done: true}:
		case <-ctx.Done():
		}
	}()
	return events, nil
}

func parseStreamChunk(payload, conversationID string) (Response, bool) {
	var upstream struct {
		ID      string `json:"id"`
		Choices []struct {
			Delta struct {
				Role      string     `json:"role"`
				Content   any        `json:"content"`
				Reasoning string     `json:"reasoning_content"`
				ToolCalls []ToolCall `json:"tool_calls"`
			} `json:"delta"`
		} `json:"choices"`
		Usage map[string]any `json:"usage"`
	}
	if json.Unmarshal([]byte(payload), &upstream) != nil {
		return Response{}, false
	}
	if len(upstream.Choices) == 0 && upstream.Usage == nil {
		return Response{}, false
	}
	message := Message{Role: "assistant"}
	if len(upstream.Choices) > 0 {
		delta := upstream.Choices[0].Delta
		if delta.Role != "" {
			message.Role = delta.Role
		}
		message.Content = delta.Content
		message.ReasoningContent = delta.Reasoning
		message.ToolCalls = delta.ToolCalls
	}
	return Response{RequestID: upstream.ID, ConversationID: conversationID, Message: message, Usage: upstream.Usage}, true
}

var _ Streamer = (*Service)(nil)
