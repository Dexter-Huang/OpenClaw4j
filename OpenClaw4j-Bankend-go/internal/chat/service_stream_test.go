package chat

import "testing"

func TestParseStreamChunkPreservesDeltaAndUsage(t *testing.T) {
	response, ok := parseStreamChunk(`{"id":"request-1","choices":[{"delta":{"role":"assistant","content":"hello","reasoning_content":"think"}}]}`, "conversation-1")
	if !ok {
		t.Fatal("expected stream chunk")
	}
	if response.RequestID != "request-1" || response.ConversationID != "conversation-1" || response.Message.Content != "hello" || response.Message.ReasoningContent != "think" {
		t.Fatalf("unexpected response: %#v", response)
	}
	usage, ok := parseStreamChunk(`{"id":"request-1","usage":{"total_tokens":3}}`, "conversation-1")
	if !ok || usage.Usage["total_tokens"] != float64(3) {
		t.Fatalf("unexpected usage chunk: %#v", usage)
	}
}

func TestParseStreamChunkPreservesToolCallDelta(t *testing.T) {
	response, ok := parseStreamChunk(`{"id":"request-1","choices":[{"delta":{"tool_calls":[{"index":0,"id":"call-1","type":"function","function":{"name":"weather","arguments":"{\"city\""}}]}}]}`, "conversation-1")
	if !ok || len(response.Message.ToolCalls) != 1 {
		t.Fatalf("unexpected tool call stream response: %#v", response)
	}
	call := response.Message.ToolCalls[0]
	if call.Index == nil || *call.Index != 0 || call.ID != "call-1" || call.Function.Name != "weather" || call.Function.Arguments != `{"city"` {
		t.Fatalf("unexpected tool call delta: %#v", call)
	}
}
