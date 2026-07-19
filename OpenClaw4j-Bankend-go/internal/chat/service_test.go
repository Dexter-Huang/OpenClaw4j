package chat

import "testing"

func TestCompletionEndpointPreservesProviderBasePath(t *testing.T) {
	tests := []struct {
		name           string
		endpoint       string
		completionPath string
		want           string
	}{
		{
			name:           "dashscope compatible mode",
			endpoint:       "https://dashscope.aliyuncs.com/compatible-mode/v1",
			completionPath: "/v1/chat/completions",
			want:           "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions",
		},
		{
			name:     "openai default path",
			endpoint: "https://api.openai.com/v1",
			want:     "https://api.openai.com/v1/chat/completions",
		},
		{
			name:           "provider relative custom path",
			endpoint:       "https://model.example.com/gateway",
			completionPath: "/custom/chat",
			want:           "https://model.example.com/gateway/custom/chat",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := completionEndpoint(test.endpoint, test.completionPath)
			if err != nil {
				t.Fatalf("completionEndpoint() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("completionEndpoint() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestParseCompletionPreservesToolCalls(t *testing.T) {
	response, err := parseCompletion([]byte(`{
		"id": "request-1",
		"choices": [{
			"message": {
				"role": "assistant",
				"content": null,
				"tool_calls": [{
					"id": "call-1",
					"type": "function",
					"function": {"name": "weather", "arguments": "{\"city\":\"Shanghai\"}"}
				}]
			}
		}]
	}`), "conversation-1")
	if err != nil {
		t.Fatal(err)
	}
	if response.RequestID != "request-1" || response.ConversationID != "conversation-1" || len(response.Message.ToolCalls) != 1 {
		t.Fatalf("parsed response = %#v", response)
	}
	call := response.Message.ToolCalls[0]
	if call.ID != "call-1" || call.Type != "function" || call.Function.Name != "weather" || call.Function.Arguments != `{"city":"Shanghai"}` {
		t.Fatalf("parsed tool call = %#v", call)
	}
}
