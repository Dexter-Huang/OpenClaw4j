package agent

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/seaskyland/openclaw4j-backend-go/internal/application"
	"github.com/seaskyland/openclaw4j-backend-go/internal/chat"
	"github.com/seaskyland/openclaw4j-backend-go/internal/document"
	"github.com/seaskyland/openclaw4j-backend-go/internal/mcpserver"
	"github.com/seaskyland/openclaw4j-backend-go/internal/plugin"
	"github.com/seaskyland/openclaw4j-backend-go/internal/skill"
)

type completerFunc func(context.Context, string, chat.Request) (*chat.Response, error)

func (f completerFunc) Complete(ctx context.Context, workspace string, request chat.Request) (*chat.Response, error) {
	return f(ctx, workspace, request)
}

func TestCompleteAddsHistoryAndAppendsOnlyNewTurn(t *testing.T) {
	store := NewInMemoryStore()
	if err := store.Append(context.Background(), "ws", "conv", chat.Message{Role: "user", Content: "old"}, chat.Message{Role: "assistant", Content: "answer"}); err != nil {
		t.Fatal(err)
	}
	service := New(completerFunc(func(_ context.Context, _ string, request chat.Request) (*chat.Response, error) {
		if len(request.Messages) != 3 {
			t.Fatalf("expected memory plus request, got %#v", request.Messages)
		}
		return &chat.Response{ConversationID: "conv", Message: chat.Message{Role: "assistant", Content: "new answer"}}, nil
	}), store, 20)
	if _, err := service.Complete(context.Background(), "ws", chat.Request{ConversationID: "conv", Messages: []chat.Message{{Role: "user", Content: "new"}}}); err != nil {
		t.Fatal(err)
	}
	history, err := store.Load(context.Background(), "ws", "conv", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 4 || history[2].Content != "new" || history[3].Content != "new answer" {
		t.Fatalf("unexpected memory: %#v", history)
	}
}

func TestCompleteAddsPublishedKnowledgeBaseContextWithoutPersistingIt(t *testing.T) {
	store := NewInMemoryStore()
	retriever := &fakeRetriever{chunks: []document.Chunk{{DocID: "doc-1", Title: "员工手册", Text: "年假为十天。"}}}
	service := NewWithRetrieval(completerFunc(func(_ context.Context, _ string, request chat.Request) (*chat.Response, error) {
		if len(request.Messages) != 2 || request.Messages[0].Role != "system" || request.Messages[1].Content != "年假有几天？" {
			t.Fatalf("retrieval-enriched request = %#v", request.Messages)
		}
		context, _ := request.Messages[0].Content.(string)
		if !containsAll(context, "不可信文本", "员工手册", "年假为十天", "标注对应的文档标题") {
			t.Fatalf("unexpected retrieval context: %q", context)
		}
		return &chat.Response{ConversationID: "conv", Message: chat.Message{Role: "assistant", Content: "十天"}}, nil
	}), store, 20, fakeApplicationReader{application: application.Application{AppID: "app", PubConfig: map[string]any{"file_search": map[string]any{"enable_search": true, "enable_citation": true, "kb_ids": []any{"kb-1"}, "top_k": float64(4), "retrieve_max_length": float64(100)}}}}, retriever)

	if _, err := service.Complete(context.Background(), "ws", chat.Request{AppID: "app", ConversationID: "conv", Messages: []chat.Message{{Role: "user", Content: "年假有几天？"}}}); err != nil {
		t.Fatal(err)
	}
	if retriever.workspaceID != "ws" || retriever.query != "年假有几天？" || retriever.limit != 4 || len(retriever.knowledgeBaseIDs) != 1 || retriever.knowledgeBaseIDs[0] != "kb-1" {
		t.Fatalf("retrieval request = %#v", retriever)
	}
	history, err := store.Load(context.Background(), "ws", "conv", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 || history[0].Role != "user" || history[1].Content != "十天" {
		t.Fatalf("retrieval context must not enter memory: %#v", history)
	}
}

func TestCompleteStreamAddsKnowledgeBaseContext(t *testing.T) {
	models := &fakeStreamingCompleter{}
	service := NewWithRetrieval(models, nil, 20, fakeApplicationReader{application: application.Application{AppID: "app", Config: map[string]any{"file_search": map[string]any{"enable_search": true, "kb_ids": []any{"kb-1"}}}}}, &fakeRetriever{chunks: []document.Chunk{{DocID: "doc-1", Text: "stream context"}}})

	events, err := service.CompleteStream(context.Background(), "ws", chat.Request{AppID: "app", IsDraft: true, Messages: []chat.Message{{Role: "user", Content: "question"}}})
	if err != nil {
		t.Fatal(err)
	}
	for range events {
	}
	if len(models.request.Messages) != 2 || models.request.Messages[0].Role != "system" || models.request.Messages[1].Content != "question" {
		t.Fatalf("streaming retrieval request = %#v", models.request.Messages)
	}
}

func TestCompleteExecutesConfiguredPluginAndMCPTools(t *testing.T) {
	models := &sequenceCompleter{responses: []chat.Response{
		{ConversationID: "conv", Message: chat.Message{Role: "assistant", ToolCalls: []chat.ToolCall{
			{ID: "plugin-call", Type: "function", Function: chat.ToolFunction{Name: "lookup_employee", Arguments: `{"employee_id":"A-1"}`}},
			{ID: "mcp-call", Type: "function", Function: chat.ToolFunction{Name: "weather", Arguments: `{"city":"Shanghai"}`}},
		}}},
		{ConversationID: "conv", Message: chat.Message{Role: "assistant", Content: "员工 A-1 所在城市天气晴朗。"}},
	}}
	plugins := &fakePluginToolService{tools: []plugin.Tool{{
		PluginID:    "plugin-1",
		ToolID:      "tool-employee",
		Name:        "lookup_employee",
		Description: "查询员工信息",
		Config:      json.RawMessage(`{"input_params":[{"key":"employee_id","type":"String","description":"员工编号","required":true}]}`),
		Enabled:     true,
		Status:      "published",
	}}}
	mcpServers := &fakeMCPToolService{servers: []mcpserver.Server{{
		ServerCode: "weather-server",
		Tools: []any{map[string]any{
			"name":        "weather",
			"description": "查询城市天气",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{"city": map[string]any{"type": "string"}}},
		}},
	}}}
	service := NewWithToolCalling(models, nil, 20, fakeApplicationReader{application: application.Application{
		AppID: "app",
		PubConfig: map[string]any{
			"tools":       []any{map[string]any{"id": "tool-employee"}},
			"mcp_servers": []any{map[string]any{"id": "weather-server"}},
		},
	}}, nil, plugins, mcpServers)

	response, err := service.Complete(context.Background(), "ws", chat.Request{AppID: "app", Messages: []chat.Message{{Role: "user", Content: "查询员工 A-1 和天气"}}})
	if err != nil {
		t.Fatal(err)
	}
	if response.Message.Content != "员工 A-1 所在城市天气晴朗。" || len(models.requests) != 2 {
		t.Fatalf("response = %#v, model requests = %#v", response, models.requests)
	}
	tools, ok := models.requests[0].Parameter["tools"].([]map[string]any)
	if !ok || len(tools) != 2 {
		t.Fatalf("published tools = %#v", models.requests[0].Parameter["tools"])
	}
	if len(models.requests[1].Messages) != 4 || models.requests[1].Messages[1].Role != "assistant" || models.requests[1].Messages[2].ToolCallID != "plugin-call" || models.requests[1].Messages[3].ToolCallID != "mcp-call" {
		t.Fatalf("tool follow-up messages = %#v", models.requests[1].Messages)
	}
	if len(plugins.calls) != 1 || plugins.calls[0].pluginID != "plugin-1" || plugins.calls[0].toolID != "tool-employee" || plugins.calls[0].arguments["employee_id"] != "A-1" {
		t.Fatalf("plugin calls = %#v", plugins.calls)
	}
	if len(mcpServers.calls) != 1 || mcpServers.calls[0].ServerCode != "weather-server" || mcpServers.calls[0].ToolName != "weather" || mcpServers.calls[0].ToolParams["city"] != "Shanghai" {
		t.Fatalf("MCP calls = %#v", mcpServers.calls)
	}
	if !containsAll(models.requests[1].Messages[2].Content.(string), `"success":true`, `"employee":"A-1"`) || !containsAll(models.requests[1].Messages[3].Content.(string), `"is_error":false`, "晴") {
		t.Fatalf("tool result messages = %#v", models.requests[1].Messages[2:])
	}
}

func TestCompleteStreamExecutesToolCallsAndStreamsFinalAnswer(t *testing.T) {
	index := 0
	toolCallStart := chat.ToolCall{Index: &index, ID: "call-employee", Type: "function", Function: chat.ToolFunction{Name: "lookup_employee", Arguments: `{"employee_`}}
	toolCallEnd := chat.ToolCall{Index: &index, Function: chat.ToolFunction{Arguments: `id":"A-1"}`}}
	models := &sequenceStreamingCompleter{streams: [][]chat.StreamEvent{
		{
			{Response: chat.Response{RequestID: "request-tool", ConversationID: "conv", Message: chat.Message{Role: "assistant", ToolCalls: []chat.ToolCall{toolCallStart}}}},
			{Response: chat.Response{RequestID: "request-tool", ConversationID: "conv", Message: chat.Message{Role: "assistant", ToolCalls: []chat.ToolCall{toolCallEnd}}}},
			{Response: chat.Response{RequestID: "request-tool", ConversationID: "conv"}, Done: true},
		},
		{
			{Response: chat.Response{RequestID: "request-answer", ConversationID: "conv", Message: chat.Message{Role: "assistant", Content: "员工 A-1 "}}},
			{Response: chat.Response{RequestID: "request-answer", ConversationID: "conv", Message: chat.Message{Role: "assistant", Content: "资料已查询。"}}},
			{Response: chat.Response{RequestID: "request-answer", ConversationID: "conv", Usage: map[string]any{"total_tokens": float64(9)}}, Done: true},
		},
	}}
	plugins := &fakePluginToolService{tools: []plugin.Tool{{
		PluginID: "plugin-1",
		ToolID:   "tool-employee",
		Name:     "lookup_employee",
		Config:   json.RawMessage(`{"input_params":[{"key":"employee_id","type":"String","required":true}]}`),
		Enabled:  true,
		Status:   "published",
	}}}
	store := NewInMemoryStore()
	service := NewWithToolCalling(models, store, 20, fakeApplicationReader{application: application.Application{
		AppID:     "app",
		PubConfig: map[string]any{"tools": []any{map[string]any{"id": "tool-employee"}}},
	}}, nil, plugins, nil)

	events, err := service.CompleteStream(context.Background(), "ws", chat.Request{AppID: "app", ConversationID: "conv", Messages: []chat.Message{{Role: "user", Content: "查询员工 A-1"}}})
	if err != nil {
		t.Fatal(err)
	}
	var answer strings.Builder
	var terminal chat.StreamEvent
	for event := range events {
		if text, ok := event.Response.Message.Content.(string); ok {
			answer.WriteString(text)
		}
		if event.Done {
			terminal = event
		}
		if len(event.Response.Message.ToolCalls) != 0 {
			t.Fatalf("tool call deltas must not reach client: %#v", event)
		}
	}
	if answer.String() != "员工 A-1 资料已查询。" || !terminal.Done || terminal.Error != "" || terminal.Response.Usage["total_tokens"] != float64(9) {
		t.Fatalf("stream result = answer:%q terminal:%#v", answer.String(), terminal)
	}
	if len(models.requests) != 2 || len(models.requests[1].Messages) != 3 || models.requests[1].Messages[1].Role != "assistant" || models.requests[1].Messages[1].ToolCalls[0].Function.Arguments != `{"employee_id":"A-1"}` || models.requests[1].Messages[2].ToolCallID != "call-employee" {
		t.Fatalf("stream model requests = %#v", models.requests)
	}
	if len(plugins.calls) != 1 || plugins.calls[0].arguments["employee_id"] != "A-1" {
		t.Fatalf("plugin calls = %#v", plugins.calls)
	}
	history, err := store.Load(context.Background(), "ws", "conv", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 || history[1].Content != "员工 A-1 资料已查询。" {
		t.Fatalf("stream memory = %#v", history)
	}
}

func TestCompleteAppliesAgentConfigurationAndRunsSkillAndComponentTools(t *testing.T) {
	store := NewInMemoryStore()
	_ = store.Append(context.Background(), "ws", "conv", chat.Message{Role: "user", Content: "old-1"}, chat.Message{Role: "assistant", Content: "old-2"}, chat.Message{Role: "user", Content: "old-3"})
	models := &sequenceCompleter{responses: []chat.Response{
		{ConversationID: "conv", Message: chat.Message{Role: "assistant", ToolCalls: []chat.ToolCall{{ID: "skill", Function: chat.ToolFunction{Name: "read_skill_file", Arguments: `{"skill_code":"skill-1","path":"SKILL.md"}`}}, {ID: "component", Function: chat.ToolFunction{Name: "component-agent", Arguments: `{"query":"run"}`}}}}},
		{ConversationID: "conv", Message: chat.Message{Role: "assistant", Content: "done"}},
	}}
	skills := &fakeSkillRuntimeService{items: []skill.Detail{{SkillCode: "skill-1", Name: "示例", MainFilePath: "SKILL.md"}}, content: "skill content"}
	components := &fakeComponentToolService{items: []ComponentTool{{Code: "component-agent", Name: "子 Agent", Parameters: map[string]any{"type": "object", "properties": map[string]any{}}}}}
	service := NewWithAgentRuntime(models, store, 20, fakeApplicationReader{application: application.Application{AppID: "app", PubConfig: map[string]any{
		"instructions": "你是 {{name}}", "prompt_variables": []any{map[string]any{"name": "name", "default_value": "默认助手"}}, "memory": map[string]any{"dialog_round": float64(2)}, "skills": []any{map[string]any{"id": "skill-1"}}, "agent_components": []any{"component-agent"},
	}}}, nil, nil, nil, skills)
	service.SetComponentTools(components)
	if _, err := service.Complete(context.Background(), "ws", chat.Request{AppID: "app", ConversationID: "conv", Parameter: map[string]any{"name": "测试助手"}, Messages: []chat.Message{{Role: "user", Content: "你好 {{name}}"}}}); err != nil {
		t.Fatal(err)
	}
	if len(models.requests) != 2 || len(models.requests[0].Messages) != 4 {
		t.Fatalf("configured request = %#v", models.requests)
	}
	context, _ := models.requests[0].Messages[0].Content.(string)
	if !containsAll(context, "你是 测试助手", "skill-1") || models.requests[0].Messages[3].Content != "你好 测试助手" {
		t.Fatalf("agent context = %#v", models.requests[0].Messages)
	}
	if skills.readCode != "skill-1" || components.calledCode != "component-agent" {
		t.Fatalf("tool services = %#v %#v", skills, components)
	}
}

type fakeApplicationReader struct{ application application.Application }

func (f fakeApplicationReader) Get(context.Context, string, string) (*application.Application, error) {
	return &f.application, nil
}

type fakeRetriever struct {
	chunks           []document.Chunk
	workspaceID      string
	knowledgeBaseIDs []string
	query            string
	limit            int
}

func (f *fakeRetriever) Search(_ context.Context, workspaceID string, knowledgeBaseIDs []string, query string, limit int) ([]document.Chunk, error) {
	f.workspaceID = workspaceID
	f.knowledgeBaseIDs = append([]string(nil), knowledgeBaseIDs...)
	f.query = query
	f.limit = limit
	return f.chunks, nil
}

type fakeStreamingCompleter struct{ request chat.Request }

func (f *fakeStreamingCompleter) Complete(context.Context, string, chat.Request) (*chat.Response, error) {
	return nil, nil
}

func (f *fakeStreamingCompleter) CompleteStream(_ context.Context, _ string, request chat.Request) (<-chan chat.StreamEvent, error) {
	f.request = request
	events := make(chan chat.StreamEvent, 1)
	events <- chat.StreamEvent{Response: chat.Response{ConversationID: "conv"}, Done: true}
	close(events)
	return events, nil
}

type sequenceStreamingCompleter struct {
	streams  [][]chat.StreamEvent
	requests []chat.Request
}

func (f *sequenceStreamingCompleter) Complete(context.Context, string, chat.Request) (*chat.Response, error) {
	return nil, errors.New("unexpected non-streaming model request")
}

func (f *sequenceStreamingCompleter) CompleteStream(_ context.Context, _ string, request chat.Request) (<-chan chat.StreamEvent, error) {
	if len(f.streams) == 0 {
		return nil, errors.New("unexpected streaming model request")
	}
	f.requests = append(f.requests, request)
	events := make(chan chat.StreamEvent, len(f.streams[0]))
	for _, event := range f.streams[0] {
		events <- event
	}
	close(events)
	f.streams = f.streams[1:]
	return events, nil
}

type sequenceCompleter struct {
	responses []chat.Response
	requests  []chat.Request
}

func (f *sequenceCompleter) Complete(_ context.Context, _ string, request chat.Request) (*chat.Response, error) {
	f.requests = append(f.requests, request)
	if len(f.responses) == 0 {
		return nil, errors.New("unexpected model request")
	}
	response := f.responses[0]
	f.responses = f.responses[1:]
	return &response, nil
}

type pluginToolCall struct {
	pluginID  string
	toolID    string
	arguments map[string]any
}

type fakePluginToolService struct {
	tools []plugin.Tool
	calls []pluginToolCall
}

func (f *fakePluginToolService) ListToolsByIDs(_ context.Context, _ string, _ []string) ([]plugin.Tool, error) {
	return f.tools, nil
}

func (f *fakePluginToolService) InvokeTool(_ context.Context, _ string, pluginID, toolID string, arguments map[string]any) (*plugin.TestResult, error) {
	f.calls = append(f.calls, pluginToolCall{pluginID: pluginID, toolID: toolID, arguments: arguments})
	return &plugin.TestResult{Success: true, Data: map[string]any{"employee": arguments["employee_id"]}}, nil
}

type fakeMCPToolService struct {
	servers []mcpserver.Server
	calls   []mcpserver.ToolCallInput
}

func (f *fakeMCPToolService) ListByCodes(_ context.Context, _ string, _ []string, _ bool) ([]mcpserver.Server, error) {
	return f.servers, nil
}

func (f *fakeMCPToolService) CallTool(_ context.Context, _ string, input mcpserver.ToolCallInput) (*mcpserver.ToolCallResult, error) {
	f.calls = append(f.calls, input)
	return &mcpserver.ToolCallResult{Content: []map[string]any{{"type": "text", "text": "晴"}}}, nil
}

type fakeSkillRuntimeService struct {
	items             []skill.Detail
	content, readCode string
}

func (f *fakeSkillRuntimeService) ListByCodes(context.Context, string, []string, bool) ([]skill.Detail, error) {
	return f.items, nil
}
func (f *fakeSkillRuntimeService) ReadFile(_ context.Context, _ string, code, _ string) (string, error) {
	f.readCode = code
	return f.content, nil
}

type fakeComponentToolService struct {
	items      []ComponentTool
	calledCode string
}

func (f *fakeComponentToolService) List(context.Context, string, []string, string) ([]ComponentTool, error) {
	return f.items, nil
}
func (f *fakeComponentToolService) Invoke(_ context.Context, _ string, code string, _ map[string]any) (any, error) {
	f.calledCode = code
	return "component result", nil
}

func containsAll(value string, values ...string) bool {
	for _, expected := range values {
		if !strings.Contains(value, expected) {
			return false
		}
	}
	return true
}
