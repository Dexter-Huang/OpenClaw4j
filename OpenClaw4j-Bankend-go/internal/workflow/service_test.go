package workflow

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/seaskyland/openclaw4j-backend-go/internal/application"
	"github.com/seaskyland/openclaw4j-backend-go/internal/chat"
	"github.com/seaskyland/openclaw4j-backend-go/internal/scriptsandbox"
)

func TestServiceRunsStartToEndGraph(t *testing.T) {
	service := NewService(fakeApplicationReader{version: application.Version{Config: `{
        "nodes": [
          {"id":"start","name":"Start","type":"Start","config":{"output_params":[{"key":"name","type":"String"}]}},
			{"id":"end","name":"End","type":"End","config":{"node_param":{"text_template":"Hello ${start.name}"}}}
        ],
        "edges": [{"source":"start","target":"end"}]
      }`}}, nil, nil)

	params, err := service.Init(context.Background(), "workspace", InitRequest{AppID: "app"})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if len(params) != 2 || params[0].Key != "name" || params[1].Key != "sys.query" {
		t.Fatalf("Init() params = %#v", params)
	}

	run, err := service.Run(context.Background(), "workspace", "request", TaskRunRequest{AppID: "app", Inputs: []Param{{Key: "name", Value: "Ada"}}})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	process := waitForProcess(t, service, "workspace", run.TaskID, statusSuccess)
	if process.TaskResults[0].NodeContent != `{"output":"Hello Ada"}` {
		t.Fatalf("workflow output = %#v", process.TaskResults)
	}
}

func TestServicePausesAndResumesInputNode(t *testing.T) {
	service := NewService(nil, nil, nil)
	run, err := service.RunFragment(context.Background(), "workspace", "request", FragmentRequest{
		Nodes: []Node{
			{ID: "start", Type: "Start"},
			{ID: "input", Type: "Input", Config: NodeConfig{OutputParams: []Param{{Key: "answer"}}}},
			{ID: "end", Type: "End", Config: NodeConfig{NodeParam: map[string]any{"text_template": "${input.answer}"}}},
		},
		Edges: []Edge{{Source: "start", Target: "input"}, {Source: "input", Target: "end"}},
	})
	if err != nil {
		t.Fatalf("RunFragment() error = %v", err)
	}
	waitForProcess(t, service, "workspace", run.TaskID, statusPause)
	if _, err := service.Resume(context.Background(), "workspace", "next", ResumeRequest{TaskID: run.TaskID, ResumeNodeID: "input", InputParams: []Param{{Key: "answer", Value: "done"}}}); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	process := waitForProcess(t, service, "workspace", run.TaskID, statusSuccess)
	if process.TaskResults[len(process.TaskResults)-1].NodeContent != `{"output":"done"}` {
		t.Fatalf("resume output = %#v", process.TaskResults)
	}
}

func TestServiceRunStreamPublishesNodeEventsBeforePause(t *testing.T) {
	service := NewService(fakeApplicationReader{version: application.Version{Config: `{
        "nodes": [
          {"id":"start","name":"Start","type":"Start","config":{}},
          {"id":"input","name":"Input","type":"Input","config":{"output_params":[{"key":"answer"}]}}
        ],
        "edges": [{"source":"start","target":"input"}]
      }`}}, nil, nil)

	_, events, unsubscribe, err := service.RunStream(context.Background(), "workspace", "request", TaskRunRequest{AppID: "app"})
	if err != nil {
		t.Fatalf("RunStream() error = %v", err)
	}
	defer unsubscribe()

	received := make([]StreamEvent, 0, 5)
	deadline := time.After(time.Second)
	for {
		select {
		case event, ok := <-events:
			if !ok {
				if len(received) != 5 {
					t.Fatalf("stream event count = %d, want 5: %#v", len(received), received)
				}
				if received[0].NodeID != "start" || received[0].NodeStatus != statusExecuting || received[1].NodeStatus != statusSuccess {
					t.Fatalf("start node events = %#v", received[:2])
				}
				if received[2].NodeID != "input" || received[2].NodeStatus != statusExecuting || received[3].NodeStatus != statusPause {
					t.Fatalf("input node events = %#v", received[2:4])
				}
				if received[4].Event != "Paused" || received[4].PauseType != "InputNodeInterrupt" {
					t.Fatalf("terminal event = %#v", received[4])
				}
				return
			}
			received = append(received, event)
		case <-deadline:
			t.Fatalf("timed out waiting for workflow stream events: %#v", received)
		}
	}
}

func TestServiceRestoresPausedTaskFromStateStore(t *testing.T) {
	store := &memoryStateStore{items: map[string][]byte{}}
	first := NewServiceWithNodeServicesAndState(nil, nil, nil, nil, nil, nil, nil, nil, store)
	run, err := first.RunFragment(context.Background(), "workspace", "request", FragmentRequest{
		Nodes: []Node{
			{ID: "start", Type: "Start"},
			{ID: "input", Type: "Input", Config: NodeConfig{OutputParams: []Param{{Key: "answer"}}}},
			{ID: "end", Type: "End", Config: NodeConfig{NodeParam: map[string]any{"text_template": "${input.answer}"}}},
		},
		Edges: []Edge{{Source: "start", Target: "input"}, {Source: "input", Target: "end"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	waitForProcess(t, first, "workspace", run.TaskID, statusPause)
	// The second service has no in-process task map, which models a different pod.
	second := NewServiceWithNodeServicesAndState(nil, nil, nil, nil, nil, nil, nil, nil, store)
	if _, err := second.Resume(context.Background(), "workspace", "request-2", ResumeRequest{TaskID: run.TaskID, ResumeNodeID: "input", InputParams: []Param{{Key: "answer", Value: "restored"}}}); err != nil {
		t.Fatal(err)
	}
	process := waitForProcess(t, second, "workspace", run.TaskID, statusSuccess)
	if process.TaskResults[len(process.TaskResults)-1].NodeContent != `{"output":"restored"}` {
		t.Fatalf("unexpected restored result: %#v", process.TaskResults)
	}
}

func TestServiceRunsLLMNodeThroughSharedCompleter(t *testing.T) {
	model := &fakeModelCompleter{}
	service := NewService(nil, model, nil)
	run, err := service.RunFragment(context.Background(), "workspace", "request", FragmentRequest{
		Nodes: []Node{
			{ID: "start", Type: "Start"},
			{ID: "llm", Type: "LLM", Config: NodeConfig{OutputParams: []Param{{Key: "output"}}, NodeParam: map[string]any{"model_config": map[string]any{"provider": "provider", "model_id": "model"}, "prompt_content": "Hello"}}},
			{ID: "end", Type: "End", Config: NodeConfig{NodeParam: map[string]any{"text_template": "${llm.output}"}}},
		},
		Edges: []Edge{{Source: "start", Target: "llm"}, {Source: "llm", Target: "end"}},
	})
	if err != nil {
		t.Fatalf("RunFragment() error = %v", err)
	}
	process := waitForProcess(t, service, "workspace", run.TaskID, statusSuccess)
	if !model.called || process.TaskResults[0].NodeContent != `{"output":"model response"}` {
		t.Fatalf("LLM workflow result = %#v, called=%t", process.TaskResults, model.called)
	}
}

func TestServiceRunsJavaScriptScriptNode(t *testing.T) {
	scripts := &fakeScriptExecutor{result: &scriptsandbox.Result{Success: true, Data: map[string]any{"output": int64(6111)}}}
	service := NewServiceWithScriptExecutor(nil, nil, nil, scripts)
	run, err := service.RunFragment(context.Background(), "workspace", "request", FragmentRequest{
		Nodes: []Node{{
			ID:   "Script_TF3N",
			Name: "脚本1",
			Type: "Script",
			Config: NodeConfig{
				InputParams: []Param{
					{Key: "input1", Type: "Number", Value: "5556", ValueFrom: "input"},
					{Key: "input2", Type: "Number", Value: "555", ValueFrom: "input"},
				},
				OutputParams: []Param{{Key: "output", Type: "Number"}},
				NodeParam: map[string]any{
					"script_type":    "javascript",
					"script_content": `function main(params) { return { output: params.input1 + params.input2 }; }`,
				},
			},
		}},
	})
	if err != nil {
		t.Fatalf("RunFragment() error = %v", err)
	}
	process := waitForProcess(t, service, "workspace", run.TaskID, statusSuccess)
	if len(process.NodeResults) != 1 || process.NodeResults[0].Output != `{"output":6111}` {
		t.Fatalf("script workflow result = %#v", process.NodeResults)
	}
	if scripts.language != "javascript" || scripts.requestID != "request" {
		t.Fatalf("script sandbox call = %#v", scripts)
	}
}

func TestServiceRunsPythonScriptNode(t *testing.T) {
	scripts := &fakeScriptExecutor{result: &scriptsandbox.Result{Success: true, Data: map[string]any{"output": float64(6)}}}
	service := NewServiceWithScriptExecutor(nil, nil, nil, scripts)
	run, err := service.RunFragment(context.Background(), "workspace", "request", FragmentRequest{
		Nodes: []Node{{
			ID: "python", Type: "Script",
			Config: NodeConfig{
				InputParams:  []Param{{Key: "number", Type: "Number", Value: "3"}},
				OutputParams: []Param{{Key: "output", Type: "Number"}},
				NodeParam: map[string]any{
					"script_type":    "python",
					"script_content": "def main(params):\n    return {'output': params['number'] * 2}",
				},
			},
		}},
	})
	if err != nil {
		t.Fatalf("RunFragment() error = %v", err)
	}
	process := waitForProcess(t, service, "workspace", run.TaskID, statusSuccess)
	if len(process.NodeResults) != 1 || process.NodeResults[0].Output != `{"output":6}` || scripts.language != "python" {
		t.Fatalf("python script result = %#v, call=%#v", process.NodeResults, scripts)
	}
}

func TestServiceRoutesJudgeBranch(t *testing.T) {
	service := NewService(nil, nil, nil)
	branches := []any{
		map[string]any{
			"id": "pass",
			"conditions": []any{map[string]any{
				"left":     map[string]any{"value_from": "refer", "value": "${start.score}"},
				"right":    map[string]any{"value_from": "input", "value": "60"},
				"operator": "greaterAndEqual",
			}},
		},
		map[string]any{"id": "default"},
	}
	run, err := service.RunFragment(context.Background(), "workspace", "request", FragmentRequest{
		Nodes: []Node{
			{ID: "start", Type: "Start", Config: NodeConfig{OutputParams: []Param{{Key: "score"}}}},
			{ID: "judge", Type: "Judge", Config: NodeConfig{NodeParam: map[string]any{"branches": branches}}},
			{ID: "pass", Type: "End", Config: NodeConfig{NodeParam: map[string]any{"text_template": "passed"}}},
			{ID: "fallback", Type: "End", Config: NodeConfig{NodeParam: map[string]any{"text_template": "failed"}}},
		},
		Edges: []Edge{
			{Source: "start", Target: "judge"},
			{Source: "judge", SourceHandle: "judge_pass", Target: "pass"},
			{Source: "judge", SourceHandle: "judge_default", Target: "fallback"},
		},
		InputParams: []Param{{Key: "score", Value: 80}},
	})
	if err != nil {
		t.Fatalf("RunFragment() error = %v", err)
	}
	process := waitForProcess(t, service, "workspace", run.TaskID, statusSuccess)
	if len(process.TaskResults) != 1 || process.TaskResults[0].NodeContent != `{"output":"passed"}` {
		t.Fatalf("judge workflow output = %#v", process.TaskResults)
	}
}

func TestServiceRunsIteratorBlockForEveryInput(t *testing.T) {
	scripts := &fakeScriptExecutor{result: &scriptsandbox.Result{Success: true, Data: map[string]any{"output": "done"}}}
	service := NewServiceWithScriptExecutor(nil, nil, nil, scripts)
	block := map[string]any{
		"nodes": []any{map[string]any{
			"id":   "script",
			"type": "Script",
			"config": map[string]any{
				"output_params": []any{map[string]any{"key": "output"}},
				"node_param":    map[string]any{"script_type": "javascript", "script_content": "function main() { return { output: 'done' }; }"},
			},
		}},
		"edges": []any{},
	}
	iterator := Node{ID: "iterator", Type: "Iterator", Config: NodeConfig{
		InputParams:  []Param{{Key: "item", Value: `["a","b"]`, ValueFrom: "input"}},
		OutputParams: []Param{{Key: "output", Value: "${script.output}", ValueFrom: "refer"}},
		NodeParam:    map[string]any{"iterator_type": "byArray", "block": block},
	}}
	run, err := service.RunFragment(context.Background(), "workspace", "request", FragmentRequest{Nodes: []Node{iterator}})
	if err != nil {
		t.Fatalf("RunFragment() error = %v", err)
	}
	process := waitForProcess(t, service, "workspace", run.TaskID, statusSuccess)
	if len(process.NodeResults) != 1 || process.NodeResults[0].Output != `{"output":["done","done"]}` {
		t.Fatalf("iterator output = %#v", process.NodeResults)
	}
}

func TestServiceMarksFailedModelNode(t *testing.T) {
	model := &fakeModelCompleter{err: errors.New("upstream returned 404")}
	service := NewService(nil, model, nil)
	run, err := service.RunFragment(context.Background(), "workspace", "request", FragmentRequest{
		Nodes: []Node{
			{ID: "start", Type: "Start"},
			{ID: "llm", Type: "LLM", Config: NodeConfig{NodeParam: map[string]any{"model_config": map[string]any{"provider": "provider", "model_id": "model"}, "prompt_content": "Hello"}}},
			{ID: "end", Type: "End"},
		},
		Edges: []Edge{{Source: "start", Target: "llm"}, {Source: "llm", Target: "end"}},
	})
	if err != nil {
		t.Fatalf("RunFragment() error = %v", err)
	}
	process := waitForProcess(t, service, "workspace", run.TaskID, statusFail)
	if process.ErrorCode != "WORKFLOW_NODE_EXECUTION_FAIL" {
		t.Fatalf("task error code = %q", process.ErrorCode)
	}
	failed := process.NodeResults[1]
	if failed.NodeStatus != statusFail || failed.ErrorInfo != "upstream returned 404" {
		t.Fatalf("failed node result = %#v", failed)
	}
}

func waitForProcess(t *testing.T, service *Service, workspaceID, taskID, expected string) *ProcessResponse {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		process, err := service.Process(workspaceID, taskID)
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}
		if process.TaskStatus == expected {
			return process
		}
		time.Sleep(time.Millisecond)
	}
	process, _ := service.Process(workspaceID, taskID)
	t.Fatalf("workflow task status = %s, want %s", process.TaskStatus, expected)
	return nil
}

type fakeApplicationReader struct{ version application.Version }

type memoryStateStore struct {
	mu    sync.Mutex
	items map[string][]byte
}

func (s *memoryStateStore) Save(_ context.Context, workspaceID, taskID string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[taskKey(workspaceID, taskID)] = append([]byte(nil), value...)
	return nil
}
func (s *memoryStateStore) Load(_ context.Context, workspaceID, taskID string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.items[taskKey(workspaceID, taskID)]
	if !ok {
		return nil, ErrTaskNotFound
	}
	return append([]byte(nil), value...), nil
}

func (f fakeApplicationReader) GetVersion(context.Context, string, string, string) (*application.Version, error) {
	return &f.version, nil
}

type fakeModelCompleter struct {
	called bool
	err    error
}

type fakeScriptExecutor struct {
	result    *scriptsandbox.Result
	err       error
	language  string
	code      string
	params    map[string]any
	requestID string
}

func (f *fakeScriptExecutor) Execute(_ context.Context, language, code string, params map[string]any, requestID string) (*scriptsandbox.Result, error) {
	f.language = language
	f.code = code
	f.params = params
	f.requestID = requestID
	return f.result, f.err
}

func (f *fakeModelCompleter) CompleteWithModel(_ context.Context, _ string, _, _ string, _ []chat.Message, _, _ map[string]any, conversationID string) (*chat.Response, error) {
	f.called = true
	if f.err != nil {
		return nil, f.err
	}
	return &chat.Response{ConversationID: conversationID, Message: chat.Message{Role: "assistant", Content: "model response"}}, nil
}
