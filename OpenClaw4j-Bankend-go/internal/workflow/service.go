// Package workflow implements the controller-facing workflow task lifecycle.
package workflow

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/seaskyland/openclaw4j-backend-go/internal/appcomponent"
	"github.com/seaskyland/openclaw4j-backend-go/internal/application"
	"github.com/seaskyland/openclaw4j-backend-go/internal/chat"
	"github.com/seaskyland/openclaw4j-backend-go/internal/contextx"
	"github.com/seaskyland/openclaw4j-backend-go/internal/document"
	"github.com/seaskyland/openclaw4j-backend-go/internal/mcpserver"
	"github.com/seaskyland/openclaw4j-backend-go/internal/plugin"
	"github.com/seaskyland/openclaw4j-backend-go/internal/scriptsandbox"
)

const (
	statusExecuting = "executing"
	statusSuccess   = "success"
	statusFail      = "fail"
	statusPause     = "pause"
	statusStop      = "stop"
)

var (
	ErrAppIDRequired       = errors.New("app_id is required")
	ErrTaskIDRequired      = errors.New("task_id is required")
	ErrTaskNotFound        = errors.New("workflow task not found")
	ErrResumeNodeRequired  = errors.New("resume_node_id is required")
	ErrTaskNotPaused       = errors.New("workflow task is not paused")
	ErrNodesRequired       = errors.New("workflow nodes are required")
	ErrWorkflowConfig      = errors.New("workflow configuration is invalid")
	ErrUnsupportedNodeType = errors.New("workflow node type is not supported")
)

// ApplicationVersionReader keeps the workflow engine independent from persistence details.
type ApplicationVersionReader interface {
	GetVersion(context.Context, string, string, string) (*application.Version, error)
}

// ModelCompleter is deliberately the narrow internal capability required by LLM workflow nodes.
type ModelCompleter interface {
	CompleteWithModel(context.Context, string, string, string, []chat.Message, map[string]any, map[string]any, string) (*chat.Response, error)
}

// ApplicationCompleter 是应用组件节点调用已发布 Agent 应用所需的最小能力。
// 工作流节点仍只依赖该窄接口，避免反向依赖 HTTP 层。
type ApplicationCompleter interface {
	Complete(context.Context, string, chat.Request) (*chat.Response, error)
}

type Service struct {
	applications ApplicationVersionReader
	models       ModelCompleter
	scripts      scriptsandbox.Executor
	documents    *document.Service
	mcpServers   *mcpserver.Service
	plugins      *plugin.Service
	components   *appcomponent.Service
	httpClient   *http.Client
	clock        func() time.Time
	state        StateStore

	mu    sync.RWMutex
	tasks map[string]*task
}

type Param struct {
	Key          string `json:"key"`
	Type         string `json:"type,omitempty"`
	Desc         string `json:"desc,omitempty"`
	Value        any    `json:"value,omitempty"`
	ValueFrom    string `json:"value_from,omitempty"`
	Required     *bool  `json:"required,omitempty"`
	DefaultValue any    `json:"default_value,omitempty"`
	Source       string `json:"source,omitempty"`
}

type TaskRunRequest struct {
	AppID          string  `json:"app_id"`
	Inputs         []Param `json:"inputs"`
	ConversationID string  `json:"conversation_id"`
	Version        string  `json:"version"`
}

type TaskRunResponse struct {
	TaskID         string `json:"task_id"`
	ConversationID string `json:"conversation_id"`
	RequestID      string `json:"request_id"`
}

// CompletionRequest 对齐公开 /api/v1/apps/workflow/* 接口的请求体。
type CompletionRequest struct {
	AppID          string         `json:"app_id"`
	ConversationID string         `json:"conversation_id"`
	RequestID      string         `json:"request_id"`
	Messages       []chat.Message `json:"messages"`
	Stream         bool           `json:"stream"`
	Draft          bool           `json:"draft"`
	InputParams    []Param        `json:"input_params"`
}

type CompletionResponse struct {
	RequestID       string       `json:"request_id"`
	ConversationID  string       `json:"conversation_id"`
	TaskID          string       `json:"task_id"`
	NodeID          string       `json:"node_id,omitempty"`
	NodeName        string       `json:"node_name,omitempty"`
	NodeType        string       `json:"node_type,omitempty"`
	NodeStatus      string       `json:"node_status,omitempty"`
	NodeIsCompleted bool         `json:"node_is_completed"`
	Status          string       `json:"status"`
	Message         chat.Message `json:"message,omitempty"`
	Error           *Error       `json:"error,omitempty"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type InitRequest struct {
	AppID   string `json:"app_id"`
	Version string `json:"version"`
}

type FragmentRequest struct {
	AppID       string  `json:"app_id"`
	Nodes       []Node  `json:"nodes"`
	Edges       []Edge  `json:"edges"`
	InputParams []Param `json:"input_params"`
}

type ResumeRequest struct {
	AppID          string  `json:"app_id"`
	TaskID         string  `json:"task_id"`
	ConversationID string  `json:"conversation_id"`
	ResumeNodeID   string  `json:"resume_node_id"`
	ResumeParentID string  `json:"resume_parent_id"`
	InputParams    []Param `json:"input_params"`
}

type Node struct {
	ID     string     `json:"id"`
	Name   string     `json:"name"`
	Type   string     `json:"type"`
	Config NodeConfig `json:"config"`
}

type NodeConfig struct {
	InputParams  []Param        `json:"input_params"`
	OutputParams []Param        `json:"output_params"`
	NodeParam    map[string]any `json:"node_param"`
}

type Edge struct {
	ID           string `json:"id"`
	Source       string `json:"source"`
	SourceHandle string `json:"source_handle"`
	Target       string `json:"target"`
	TargetHandle string `json:"target_handle"`
}

type NodeResult struct {
	NodeID       string       `json:"node_id"`
	NodeName     string       `json:"node_name"`
	NodeType     string       `json:"node_type"`
	NodeStatus   string       `json:"node_status"`
	ErrorCode    string       `json:"error_code,omitempty"`
	ErrorInfo    string       `json:"error_info,omitempty"`
	ParentNodeID string       `json:"parent_node_id,omitempty"`
	Input        any          `json:"input,omitempty"`
	Output       any          `json:"output,omitempty"`
	Index        int          `json:"index,omitempty"`
	Batches      []NodeResult `json:"batches"`
}

type ProcessOutput struct {
	NodeType     string `json:"node_type"`
	NodeName     string `json:"node_name"`
	NodeID       string `json:"node_id"`
	ParentNodeID string `json:"parent_node_id,omitempty"`
	NodeContent  any    `json:"node_content"`
	NodeStatus   string `json:"node_status"`
	Index        int    `json:"index,omitempty"`
}

type ProcessResponse struct {
	TaskID         string          `json:"task_id"`
	RequestID      string          `json:"request_id"`
	ConversationID string          `json:"conversation_id"`
	TaskStatus     string          `json:"task_status"`
	TaskResults    []ProcessOutput `json:"task_results"`
	ErrorCode      string          `json:"error_code,omitempty"`
	ErrorInfo      string          `json:"error_info,omitempty"`
	TaskExecTime   string          `json:"task_exec_time,omitempty"`
	NodeResults    []NodeResult    `json:"node_results"`
}

type AsyncResponse struct {
	TaskID         string          `json:"task_id"`
	RequestID      string          `json:"request_id"`
	ConversationID string          `json:"conversation_id"`
	TaskStatus     string          `json:"task_status"`
	ErrorCode      string          `json:"error_code,omitempty"`
	ErrorInfo      string          `json:"error_info,omitempty"`
	TaskExecTime   string          `json:"task_exec_time,omitempty"`
	Outputs        []ProcessOutput `json:"outputs"`
}

// StreamEvent 在工作流推进时发出。字段名保持 Java Controller 的事件契约，
// 使已有 SparkChat 消费端无需因传输层替换而增加适配代码。
type StreamEvent struct {
	Event           string `json:"event"`
	TaskID          string `json:"task_id"`
	ConversationID  string `json:"conversation_id"`
	NodeID          string `json:"node_id,omitempty"`
	NodeName        string `json:"node_name,omitempty"`
	NodeType        string `json:"node_type,omitempty"`
	NodeStatus      string `json:"node_status,omitempty"`
	NodeMsgSeqID    int    `json:"node_msg_seq_id,omitempty"`
	NodeIsCompleted bool   `json:"node_is_completed"`
	TextContent     string `json:"text_content,omitempty"`
	ErrorCode       string `json:"error_code,omitempty"`
	ErrorMessage    string `json:"error_message,omitempty"`
	PauseType       string `json:"pause_type,omitempty"`
}

type config struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

type task struct {
	mu sync.RWMutex

	workspaceID    string
	appID          string
	id             string
	requestID      string
	conversationID string
	startedAt      time.Time
	config         config
	variables      map[string]any
	status         string
	errorCode      string
	errorInfo      string
	results        map[string]NodeResult
	order          []string
	pendingNodeID  string
	branchHandles  map[string]string
	subscribers    map[chan StreamEvent]struct{}
	eventSequence  int
	streamClosed   bool
}

func NewService(applications ApplicationVersionReader, models ModelCompleter, clock func() time.Time) *Service {
	return NewServiceWithScriptExecutor(applications, models, clock, nil)
}

// NewServiceWithScriptExecutor 保留原构造函数的兼容性，并允许启动装配和单测注入沙箱执行器。
func NewServiceWithScriptExecutor(applications ApplicationVersionReader, models ModelCompleter, clock func() time.Time, scripts scriptsandbox.Executor) *Service {
	return NewServiceWithNodeServices(applications, models, clock, scripts, nil, nil, nil, nil)
}

// NewServiceWithNodeServices 将外部节点需要的既有服务显式注入工作流引擎。保留旧构造函数，
// 使仅覆盖基础节点的调用方和已有测试不需要迁移。
func NewServiceWithNodeServices(applications ApplicationVersionReader, models ModelCompleter, clock func() time.Time, scripts scriptsandbox.Executor, documents *document.Service, mcpServers *mcpserver.Service, plugins *plugin.Service, components *appcomponent.Service) *Service {
	return NewServiceWithNodeServicesAndState(applications, models, clock, scripts, documents, mcpServers, plugins, components, nil)
}

// NewServiceWithNodeServicesAndState adds durable task state without changing the
// existing constructor used by controller compatibility tests.
func NewServiceWithNodeServicesAndState(applications ApplicationVersionReader, models ModelCompleter, clock func() time.Time, scripts scriptsandbox.Executor, documents *document.Service, mcpServers *mcpserver.Service, plugins *plugin.Service, components *appcomponent.Service, state StateStore) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{applications: applications, models: models, scripts: scripts, documents: documents, mcpServers: mcpServers, plugins: plugins, components: components, httpClient: &http.Client{Timeout: 30 * time.Second}, clock: clock, state: state, tasks: make(map[string]*task)}
}

func (s *Service) Init(ctx context.Context, workspaceID string, input InitRequest) ([]Param, error) {
	if strings.TrimSpace(input.AppID) == "" {
		return nil, ErrAppIDRequired
	}
	cfg, err := s.loadConfig(ctx, workspaceID, input.AppID, input.Version)
	if err != nil {
		return nil, err
	}
	params := make([]Param, 0)
	for _, node := range cfg.Nodes {
		if node.Type != "Start" {
			continue
		}
		for _, param := range node.Config.OutputParams {
			param.Source = "user"
			value := false
			param.Required = &value
			params = append(params, param)
		}
		break
	}
	optional := false
	params = append(params, Param{Key: "sys.query", Type: "String", Desc: "User query", Source: "sys", Required: &optional})
	return params, nil
}

func (s *Service) Run(ctx context.Context, workspaceID, requestID string, input TaskRunRequest) (*TaskRunResponse, error) {
	if strings.TrimSpace(input.AppID) == "" {
		return nil, ErrAppIDRequired
	}
	cfg, err := s.loadConfig(ctx, workspaceID, input.AppID, input.Version)
	if err != nil {
		return nil, err
	}
	return s.start(ctx, workspaceID, requestID, input.AppID, input.ConversationID, cfg, input.Inputs)
}

// RunStream 会在任务 goroutine 启动前完成订阅，避免极短工作流因先启动后订阅而丢失首个节点事件。
func (s *Service) RunStream(ctx context.Context, workspaceID, requestID string, input TaskRunRequest) (*TaskRunResponse, <-chan StreamEvent, func(), error) {
	if strings.TrimSpace(input.AppID) == "" {
		return nil, nil, nil, ErrAppIDRequired
	}
	cfg, err := s.loadConfig(ctx, workspaceID, input.AppID, input.Version)
	if err != nil {
		return nil, nil, nil, err
	}
	return s.startStream(ctx, workspaceID, requestID, input.AppID, input.ConversationID, cfg, input.Inputs)
}

func (s *Service) RunFragment(ctx context.Context, workspaceID, requestID string, input FragmentRequest) (*TaskRunResponse, error) {
	if len(input.Nodes) == 0 {
		return nil, ErrNodesRequired
	}
	return s.start(ctx, workspaceID, requestID, input.AppID, "", config{Nodes: input.Nodes, Edges: input.Edges}, input.InputParams)
}

func (s *Service) Resume(ctx context.Context, workspaceID, requestID string, input ResumeRequest) (*TaskRunResponse, error) {
	if strings.TrimSpace(input.TaskID) == "" {
		return nil, ErrTaskIDRequired
	}
	if strings.TrimSpace(input.ResumeNodeID) == "" {
		return nil, ErrResumeNodeRequired
	}
	task, err := s.getTask(workspaceID, input.TaskID)
	if err != nil {
		return nil, err
	}
	task.mu.Lock()
	if task.status != statusPause || task.pendingNodeID != input.ResumeNodeID {
		task.mu.Unlock()
		return nil, ErrTaskNotPaused
	}
	for _, param := range input.InputParams {
		task.variables[param.Key] = valueOrDefault(param.Value, param.DefaultValue)
	}
	node, ok := nodeByID(task.config.Nodes, input.ResumeNodeID)
	if !ok {
		task.mu.Unlock()
		return nil, ErrWorkflowConfig
	}
	output := map[string]any{}
	for _, param := range node.Config.OutputParams {
		value := task.variables[param.Key]
		output[param.Key] = value
		task.variables[node.ID+"."+param.Key] = value
	}
	task.recordLocked(NodeResult{NodeID: node.ID, NodeName: node.Name, NodeType: node.Type, NodeStatus: statusSuccess, Input: node.Config.OutputParams, Output: output})
	task.status = statusExecuting
	task.pendingNodeID = ""
	next := targets(task.config.Edges, node.ID, "")
	task.mu.Unlock()
	if err := s.saveTask(ctx, task); err != nil {
		return nil, err
	}
	go s.execute(contextx.Detach(ctx), task, next)
	return &TaskRunResponse{TaskID: task.id, ConversationID: task.conversationID, RequestID: requestID}, nil
}

func (s *Service) Stop(workspaceID, taskID string) (bool, error) {
	if strings.TrimSpace(taskID) == "" {
		return false, ErrTaskIDRequired
	}
	task, err := s.getTask(workspaceID, taskID)
	if err != nil {
		return false, err
	}
	task.mu.Lock()
	if task.status == statusSuccess || task.status == statusFail || task.status == statusStop {
		task.mu.Unlock()
		return false, nil
	}
	task.status = statusStop
	task.mu.Unlock()
	err = s.saveTask(context.Background(), task)
	if err == nil {
		task.finishStream()
	}
	return err == nil, err
}

func (s *Service) Process(workspaceID, taskID string) (*ProcessResponse, error) {
	task, err := s.getTask(workspaceID, taskID)
	if err != nil {
		return nil, err
	}
	task.mu.RLock()
	defer task.mu.RUnlock()
	results := make([]NodeResult, 0, len(task.order))
	outputs := make([]ProcessOutput, 0)
	for _, id := range task.order {
		result := task.results[id]
		results = append(results, result)
		if result.NodeType == "Output" || result.NodeType == "End" || result.NodeType == "Input" {
			outputs = append(outputs, ProcessOutput{NodeID: result.NodeID, NodeName: result.NodeName, NodeType: result.NodeType, NodeStatus: result.NodeStatus, ParentNodeID: result.ParentNodeID, NodeContent: result.Output, Index: result.Index})
		}
	}
	response := &ProcessResponse{TaskID: task.id, RequestID: task.requestID, ConversationID: task.conversationID, TaskStatus: task.status, TaskResults: outputs, NodeResults: results, ErrorCode: task.errorCode, ErrorInfo: task.errorInfo}
	if task.status == statusSuccess || task.status == statusFail || task.status == statusStop {
		response.TaskExecTime = time.Since(task.startedAt).Truncate(time.Millisecond).String()
	}
	return response, nil
}

func (s *Service) AsyncResult(workspaceID, taskID string) (*AsyncResponse, error) {
	process, err := s.Process(workspaceID, taskID)
	if err != nil {
		return nil, err
	}
	return &AsyncResponse{TaskID: process.TaskID, RequestID: process.RequestID, ConversationID: process.ConversationID, TaskStatus: process.TaskStatus, ErrorCode: process.ErrorCode, ErrorInfo: process.ErrorInfo, TaskExecTime: process.TaskExecTime, Outputs: process.TaskResults}, nil
}

// StartCompletion 启动公开 API 的工作流任务。消息列表中的最后一条 user 消息会映射到 Java
// 工作流约定的 sys.query，显式 input_params 仍可覆盖或补充其他开始节点变量。
func (s *Service) StartCompletion(ctx context.Context, workspaceID, requestID string, input CompletionRequest) (*TaskRunResponse, error) {
	return s.startCompletion(ctx, workspaceID, requestID, input)
}

// StartCompletionStream 是公开 API 的 RunStream 对应入口，保留 StartCompletion 的
// 输入映射规则，包括从最后一条 user 消息提取 sys.query。
func (s *Service) StartCompletionStream(ctx context.Context, workspaceID, requestID string, input CompletionRequest) (*TaskRunResponse, <-chan StreamEvent, func(), error) {
	return s.startCompletionStream(ctx, workspaceID, requestID, input)
}

func (s *Service) startCompletion(ctx context.Context, workspaceID, requestID string, input CompletionRequest) (*TaskRunResponse, error) {
	params := append([]Param(nil), input.InputParams...)
	for index := len(input.Messages) - 1; index >= 0; index-- {
		if input.Messages[index].Role == "user" {
			params = append(params, Param{Key: "sys.query", Value: input.Messages[index].Content})
			break
		}
	}
	version := "lastPublished"
	if input.Draft {
		version = "latest"
	}
	return s.Run(ctx, workspaceID, requestID, TaskRunRequest{AppID: input.AppID, ConversationID: input.ConversationID, Version: version, Inputs: params})
}

func (s *Service) startCompletionStream(ctx context.Context, workspaceID, requestID string, input CompletionRequest) (*TaskRunResponse, <-chan StreamEvent, func(), error) {
	params := append([]Param(nil), input.InputParams...)
	for index := len(input.Messages) - 1; index >= 0; index-- {
		if input.Messages[index].Role == "user" {
			params = append(params, Param{Key: "sys.query", Value: input.Messages[index].Content})
			break
		}
	}
	version := "lastPublished"
	if input.Draft {
		version = "latest"
	}
	return s.RunStream(ctx, workspaceID, requestID, TaskRunRequest{AppID: input.AppID, ConversationID: input.ConversationID, Version: version, Inputs: params})
}

// Complete 等待工作流进入稳定状态，并转换为公开 API 的 WorkflowResponse；流式处理器直接使用
// StartCompletionStream，不再先聚合完整结果。
func (s *Service) Complete(ctx context.Context, workspaceID, requestID string, input CompletionRequest) (*CompletionResponse, error) {
	run, err := s.StartCompletion(ctx, workspaceID, requestID, input)
	if err != nil {
		return nil, err
	}
	for {
		process, err := s.Process(workspaceID, run.TaskID)
		if err != nil {
			return nil, err
		}
		if process.TaskStatus != statusExecuting {
			return completionFromProcess(process), nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func completionFromProcess(process *ProcessResponse) *CompletionResponse {
	response := &CompletionResponse{RequestID: process.RequestID, ConversationID: process.ConversationID, TaskID: process.TaskID, Status: publicTaskStatus(process.TaskStatus)}
	if process.TaskStatus == statusFail || process.TaskStatus == statusStop {
		response.Error = &Error{Code: process.ErrorCode, Message: process.ErrorInfo}
		return response
	}
	if len(process.TaskResults) == 0 {
		return response
	}
	output := process.TaskResults[len(process.TaskResults)-1]
	response.NodeID = output.NodeID
	response.NodeName = output.NodeName
	response.NodeType = output.NodeType
	response.NodeStatus = output.NodeStatus
	response.NodeIsCompleted = output.NodeStatus == statusSuccess
	content := output.NodeContent
	if values, ok := content.(map[string]any); ok {
		content = values["output"]
	}
	response.Message = chat.Message{Role: "assistant", Content: content}
	return response
}

func publicTaskStatus(status string) string {
	switch status {
	case statusSuccess:
		return "completed"
	case statusFail, statusStop:
		return "failed"
	case statusPause:
		return "pause"
	default:
		return "in_progress"
	}
}

func (s *Service) start(ctx context.Context, workspaceID, requestID, appID, conversationID string, cfg config, inputs []Param) (*TaskRunResponse, error) {
	response, _, _, err := s.startWithSubscription(ctx, workspaceID, requestID, appID, conversationID, cfg, inputs, false)
	return response, err
}

func (s *Service) startStream(ctx context.Context, workspaceID, requestID, appID, conversationID string, cfg config, inputs []Param) (*TaskRunResponse, <-chan StreamEvent, func(), error) {
	return s.startWithSubscription(ctx, workspaceID, requestID, appID, conversationID, cfg, inputs, true)
}

func (s *Service) startWithSubscription(ctx context.Context, workspaceID, requestID, appID, conversationID string, cfg config, inputs []Param, stream bool) (*TaskRunResponse, <-chan StreamEvent, func(), error) {
	if len(cfg.Nodes) == 0 {
		return nil, nil, nil, ErrWorkflowConfig
	}
	taskID, err := identifier()
	if err != nil {
		return nil, nil, nil, err
	}
	if conversationID == "" {
		conversationID, err = identifier()
		if err != nil {
			return nil, nil, nil, err
		}
	}
	values := make(map[string]any, len(inputs))
	for _, input := range inputs {
		values[input.Key] = valueOrDefault(input.Value, input.DefaultValue)
	}
	task := &task{workspaceID: workspaceID, appID: appID, id: taskID, requestID: requestID, conversationID: conversationID, startedAt: s.clock(), config: cfg, variables: values, status: statusExecuting, results: make(map[string]NodeResult), branchHandles: make(map[string]string), subscribers: make(map[chan StreamEvent]struct{})}
	var events <-chan StreamEvent
	var unsubscribe func()
	if stream {
		events, unsubscribe = task.subscribe()
	}
	s.mu.Lock()
	s.tasks[taskKey(workspaceID, taskID)] = task
	s.mu.Unlock()
	if err := s.saveTask(ctx, task); err != nil {
		s.mu.Lock()
		delete(s.tasks, taskKey(workspaceID, taskID))
		s.mu.Unlock()
		if unsubscribe != nil {
			unsubscribe()
		}
		return nil, nil, nil, err
	}
	go s.execute(contextx.Detach(ctx), task, startNodes(cfg))
	return &TaskRunResponse{TaskID: taskID, ConversationID: conversationID, RequestID: requestID}, events, unsubscribe, nil
}

func (s *Service) execute(ctx context.Context, task *task, queue []string) {
	for len(queue) > 0 {
		if task.isTerminal() {
			_ = s.saveTask(ctx, task)
			task.finishStream()
			return
		}
		nodeID := queue[0]
		queue = queue[1:]
		node, found := nodeByID(task.config.Nodes, nodeID)
		if !found {
			task.fail("WORKFLOW_CONFIG_INVALID", "edge references a missing node")
			_ = s.saveTask(ctx, task)
			task.finishStream()
			return
		}
		if err := s.saveTask(ctx, task); err != nil {
			task.fail("WORKFLOW_STATE_STORE_FAILURE", err.Error())
			task.finishStream()
			return
		}
		continueWith, err := s.executeNode(ctx, task, node)
		if err != nil {
			task.failNode(node, "WORKFLOW_NODE_EXECUTION_FAIL", err.Error())
			_ = s.saveTask(ctx, task)
			task.publishNode(node.ID)
			task.finishStream()
			return
		}
		task.publishNode(node.ID)
		if !continueWith {
			_ = s.saveTask(ctx, task)
			task.finishStream()
			return
		}
		if err := s.saveTask(ctx, task); err != nil {
			task.fail("WORKFLOW_STATE_STORE_FAILURE", err.Error())
			task.finishStream()
			return
		}
		queue = append(queue, targets(task.config.Edges, node.ID, task.takeBranchHandle(node.ID))...)
	}
	task.mu.Lock()
	if task.status == statusExecuting {
		task.status = statusSuccess
	}
	task.mu.Unlock()
	_ = s.saveTask(ctx, task)
	task.finishStream()
}

func (s *Service) executeNode(ctx context.Context, task *task, node Node) (bool, error) {
	task.mu.Lock()
	if task.status != statusExecuting {
		task.mu.Unlock()
		return false, nil
	}
	task.recordLocked(NodeResult{NodeID: node.ID, NodeName: node.Name, NodeType: node.Type, NodeStatus: statusExecuting, Input: node.Config.InputParams})
	task.mu.Unlock()
	task.publishNode(node.ID)

	switch node.Type {
	case "Start":
		output := task.bindConfiguredOutputs(node)
		task.complete(node, output)
		return true, nil
	case "Input":
		task.mu.Lock()
		task.status = statusPause
		task.pendingNodeID = node.ID
		task.recordLocked(NodeResult{NodeID: node.ID, NodeName: node.Name, NodeType: node.Type, NodeStatus: statusPause, Input: node.Config.OutputParams})
		task.mu.Unlock()
		return false, nil
	case "Output", "End":
		output := map[string]any{"output": task.resolve(stringValue(node.Config.NodeParam["output"]))}
		if template := stringValue(node.Config.NodeParam["text_template"]); template != "" {
			output["output"] = task.resolve(template)
		}
		task.complete(node, output)
		return true, nil
	case "LLM":
		if s.models == nil {
			return false, errors.New("workflow model service is unavailable")
		}
		modelConfig := mapValue(node.Config.NodeParam["model_config"])
		provider := stringValue(modelConfig["provider"])
		modelID := stringValue(modelConfig["model_id"])
		messages := []chat.Message{}
		if prompt := task.resolve(stringValue(node.Config.NodeParam["sys_prompt_content"])); prompt != "" {
			messages = append(messages, chat.Message{Role: "system", Content: prompt})
		}
		messages = append(messages, chat.Message{Role: "user", Content: task.resolve(stringValue(node.Config.NodeParam["prompt_content"]))})
		response, err := s.models.CompleteWithModel(ctx, task.workspaceID, provider, modelID, messages, parameterValues(modelConfig["params"]), nil, task.conversationID)
		if err != nil {
			return false, err
		}
		content := response.Message.Content
		output := map[string]any{"output": content}
		for _, param := range node.Config.OutputParams {
			if param.Key == "output" {
				output[param.Key] = content
			}
		}
		task.bindOutput(node, output)
		task.complete(node, output)
		return true, nil
	case "Classifier":
		if s.models == nil {
			return false, errors.New("workflow model service is unavailable")
		}
		modelConfig := mapValue(node.Config.NodeParam["model_config"])
		provider := stringValue(modelConfig["provider"])
		modelID := stringValue(modelConfig["model_id"])
		conditions, ok := node.Config.NodeParam["conditions"].([]any)
		if !ok || len(conditions) == 0 {
			return false, ErrWorkflowConfig
		}
		subjects := make([]string, 0, len(conditions))
		defaultID := "default"
		for _, raw := range conditions {
			condition := mapValue(raw)
			if id := stringValue(condition["id"]); id == "default" {
				defaultID = id
			} else if subject := stringValue(condition["subject"]); subject != "" {
				subjects = append(subjects, subject)
			}
		}
		input := ""
		if len(node.Config.InputParams) > 0 {
			input = task.resolve(stringValue(node.Config.InputParams[0].Value))
		}
		instruction := task.resolve(stringValue(node.Config.NodeParam["instruction"]))
		prompt := fmt.Sprintf("%s\nChoose exactly one category from: %s. Reply with only the category text.\nInput: %s", instruction, strings.Join(subjects, ", "), input)
		response, err := s.models.CompleteWithModel(ctx, task.workspaceID, provider, modelID, []chat.Message{{Role: "user", Content: prompt}}, parameterValues(modelConfig["params"]), nil, task.conversationID)
		if err != nil {
			return false, err
		}
		decision := strings.TrimSpace(fmt.Sprint(response.Message.Content))
		selectedID := defaultID
		selectedSubject := ""
		for _, raw := range conditions {
			condition := mapValue(raw)
			subject := stringValue(condition["subject"])
			if subject != "" && strings.Contains(decision, subject) {
				selectedID = stringValue(condition["id"])
				selectedSubject = subject
				break
			}
		}
		output := map[string]any{"subject": selectedSubject, "thought": decision}
		task.bindOutput(node, output)
		task.setBranchHandle(node.ID, node.ID+"_"+selectedID)
		task.complete(node, output)
		return true, nil
	case "Script":
		output, err := task.executeScriptNode(ctx, node, s.scripts)
		if err != nil {
			return false, err
		}
		task.complete(node, output)
		return true, nil
	case "Judge":
		output, handle, err := task.executeJudgeNode(node)
		if err != nil {
			return false, err
		}
		task.setBranchHandle(node.ID, handle)
		task.complete(node, output)
		return true, nil
	case "VariableAssign":
		output, err := task.executeVariableAssignNode(node)
		if err != nil {
			return false, err
		}
		task.complete(node, output)
		return true, nil
	case "VariableHandle":
		output, err := task.executeVariableHandleNode(node)
		if err != nil {
			return false, err
		}
		task.complete(node, output)
		return true, nil
	case "ParameterExtractor":
		output, err := s.executeParameterExtractorNode(ctx, task, node)
		if err != nil {
			return false, err
		}
		task.complete(node, output)
		return true, nil
	case "API":
		output, err := s.executeAPINode(ctx, task, node)
		if err != nil {
			return false, err
		}
		task.complete(node, output)
		return true, nil
	case "Retrieval":
		output, err := s.executeRetrievalNode(ctx, task, node)
		if err != nil {
			return false, err
		}
		task.complete(node, output)
		return true, nil
	case "MCP":
		output, err := s.executeMCPNode(ctx, task, node)
		if err != nil {
			return false, err
		}
		task.complete(node, output)
		return true, nil
	case "Plugin":
		output, err := s.executePluginNode(ctx, task, node)
		if err != nil {
			return false, err
		}
		task.complete(node, output)
		return true, nil
	case "AppComponent":
		output, err := s.executeAppComponentNode(ctx, task, node)
		if err != nil {
			return false, err
		}
		task.complete(node, output)
		return true, nil
	case "Iterator", "Parallel":
		output, err := s.executeGroupNode(ctx, task, node)
		if err != nil {
			return false, err
		}
		task.complete(node, output)
		return true, nil
	case "IteratorStart", "IteratorEnd", "ParallelStart", "ParallelEnd":
		// 组内起止节点由容器执行器创建的子任务调度；单独调试时仍允许完成变量绑定。
		output := task.bindConfiguredValues(node)
		task.complete(node, output)
		return true, nil
	default:
		return false, fmt.Errorf("%w: %s", ErrUnsupportedNodeType, node.Type)
	}
}

func (t *task) bindConfiguredValues(node Node) map[string]any {
	output := t.inputValues(node.Config.OutputParams)
	if len(output) == 0 {
		output = t.inputValues(node.Config.InputParams)
	}
	return output
}

func (t *task) inputValues(params []Param) map[string]any {
	values := make(map[string]any, len(params))
	for _, param := range params {
		if param.Key == "" {
			continue
		}
		values[param.Key] = t.paramValue(param)
	}
	return values
}

func (t *task) paramValue(param Param) any {
	if strings.EqualFold(param.ValueFrom, "clear") {
		return nil
	}
	if strings.EqualFold(param.ValueFrom, "refer") {
		return t.resolveScriptReference(param.Value)
	}
	return valueOrDefault(param.Value, param.DefaultValue)
}

func (t *task) executeJudgeNode(node Node) (map[string]any, string, error) {
	branches, ok := node.Config.NodeParam["branches"].([]any)
	if !ok || len(branches) == 0 {
		return nil, "", ErrWorkflowConfig
	}
	defaultID := "default"
	for _, raw := range branches {
		branch := mapValue(raw)
		id := stringValue(branch["id"])
		if id == "" {
			continue
		}
		if id == "default" {
			defaultID = id
			continue
		}
		conditions, _ := branch["conditions"].([]any)
		matched := strings.EqualFold(stringValue(branch["logic"]), "or") == false
		for _, rawCondition := range conditions {
			condition := mapValue(rawCondition)
			result := judgeCondition(t, mapValue(condition["left"]), mapValue(condition["right"]), stringValue(condition["operator"]))
			if strings.EqualFold(stringValue(branch["logic"]), "or") {
				matched = matched || result
			} else {
				matched = matched && result
			}
		}
		if len(conditions) > 0 && matched {
			return map[string]any{"branch_id": id}, node.ID + "_" + id, nil
		}
	}
	return map[string]any{"branch_id": defaultID}, node.ID + "_" + defaultID, nil
}

func judgeCondition(task *task, left, right map[string]any, operator string) bool {
	leftValue := task.paramValue(Param{Value: left["value"], ValueFrom: stringValue(left["value_from"]), Type: stringValue(left["type"])})
	rightValue := task.paramValue(Param{Value: right["value"], ValueFrom: stringValue(right["value_from"]), Type: stringValue(right["type"])})
	leftText, rightText := fmt.Sprint(leftValue), fmt.Sprint(rightValue)
	length := func(value any) int {
		switch typed := value.(type) {
		case string:
			return len(typed)
		case []any:
			return len(typed)
		case map[string]any:
			return len(typed)
		default:
			return 0
		}
	}
	switch operator {
	case "isNull":
		return leftValue == nil || leftText == "" || length(leftValue) == 0
	case "isNotNull":
		return leftValue != nil && leftText != "" && (length(leftValue) != 0 || (leftText != "[]" && leftText != "map[]"))
	case "isTrue":
		return strings.EqualFold(leftText, "true")
	case "isFalse":
		return strings.EqualFold(leftText, "false")
	case "equals":
		return leftText == rightText
	case "notEquals":
		return leftText != rightText
	case "contains":
		return strings.Contains(leftText, rightText)
	case "notContains":
		return !strings.Contains(leftText, rightText)
	}
	leftNumber, leftErr := strconv.ParseFloat(leftText, 64)
	rightNumber, rightErr := strconv.ParseFloat(rightText, 64)
	if strings.HasPrefix(operator, "length") {
		leftNumber, leftErr = float64(length(leftValue)), nil
		rightNumber, rightErr = strconv.ParseFloat(rightText, 64)
	}
	if leftErr != nil || rightErr != nil {
		return false
	}
	switch operator {
	case "greater", "lengthGreater":
		return leftNumber > rightNumber
	case "greaterAndEqual", "lengthGreaterAndEqual":
		return leftNumber >= rightNumber
	case "less", "lengthLess":
		return leftNumber < rightNumber
	case "lessAndEqual", "lengthLessAndEqual":
		return leftNumber <= rightNumber
	case "lengthEquals":
		return leftNumber == rightNumber
	default:
		return false
	}
}

func (t *task) executeVariableAssignNode(node Node) (map[string]any, error) {
	assignments, ok := node.Config.NodeParam["inputs"].([]any)
	if !ok {
		return nil, ErrWorkflowConfig
	}
	output := make(map[string]any, len(assignments))
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, raw := range assignments {
		assignment := mapValue(raw)
		left, right := mapValue(assignment["left"]), mapValue(assignment["right"])
		key := variableKey(stringValue(left["value"]))
		if key == "" {
			return nil, ErrWorkflowConfig
		}
		value := t.paramValueLocked(Param{Value: right["value"], ValueFrom: stringValue(right["value_from"])})
		t.variables[key] = value
		output[key] = value
	}
	return output, nil
}

func (t *task) paramValueLocked(param Param) any {
	if strings.EqualFold(param.ValueFrom, "clear") {
		return nil
	}
	if strings.EqualFold(param.ValueFrom, "refer") {
		if key := variableKey(stringValue(param.Value)); key != "" {
			return t.variables[key]
		}
	}
	return param.Value
}

func variableKey(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
		return strings.TrimSuffix(strings.TrimPrefix(value, "${"), "}")
	}
	return value
}

func (t *task) executeVariableHandleNode(node Node) (map[string]any, error) {
	config := node.Config.NodeParam
	output := make(map[string]any)
	switch stringValue(config["type"]) {
	case "template":
		output["output"] = t.resolve(stringValue(config["template_content"]))
	case "json":
		for key, value := range t.inputValues(paramsValue(config["json_params"])) {
			output[key] = value
		}
	case "group":
		groups, _ := config["groups"].([]any)
		last := strings.EqualFold(stringValue(config["group_strategy"]), "lastNotNull")
		for _, raw := range groups {
			group := mapValue(raw)
			name := stringValue(group["group_name"])
			for _, rawVariable := range listValue(group["variables"]) {
				variable := mapValue(rawVariable)
				value := t.paramValue(Param{Value: variable["value"], ValueFrom: stringValue(variable["value_from"])})
				if value != nil && fmt.Sprint(value) != "" {
					if !last || output[name] == nil {
						output[name] = value
					}
				}
			}
		}
	default:
		return nil, ErrWorkflowConfig
	}
	for _, param := range node.Config.OutputParams {
		if _, exists := output[param.Key]; !exists && len(output) == 1 {
			for _, value := range output {
				output[param.Key] = value
			}
		}
	}
	return output, nil
}

func paramsValue(value any) []Param {
	items := listValue(value)
	result := make([]Param, 0, len(items))
	for _, item := range items {
		entry := mapValue(item)
		result = append(result, Param{Key: stringValue(entry["key"]), Value: entry["value"], ValueFrom: stringValue(entry["value_from"]), DefaultValue: entry["default_value"]})
	}
	return result
}

func listValue(value any) []any {
	items, _ := value.([]any)
	return items
}

func (s *Service) executeParameterExtractorNode(ctx context.Context, task *task, node Node) (map[string]any, error) {
	if s.models == nil {
		return nil, errors.New("workflow model service is unavailable")
	}
	modelConfig := mapValue(node.Config.NodeParam["model_config"])
	schema := make([]map[string]any, 0, len(node.Config.OutputParams))
	for _, param := range node.Config.OutputParams {
		if !strings.HasPrefix(param.Key, "_") {
			schema = append(schema, map[string]any{"key": param.Key, "type": param.Type, "description": param.Desc})
		}
	}
	input := ""
	if len(node.Config.InputParams) > 0 {
		input = fmt.Sprint(task.paramValue(node.Config.InputParams[0]))
	}
	prompt := fmt.Sprintf("%s\nExtract the requested fields from the input. Reply with one JSON object only. Fields: %s\nInput: %s", task.resolve(stringValue(node.Config.NodeParam["instruction"])), jsonText(schema), input)
	response, err := s.models.CompleteWithModel(ctx, task.workspaceID, stringValue(modelConfig["provider"]), stringValue(modelConfig["model_id"]), []chat.Message{{Role: "user", Content: prompt}}, parameterValues(modelConfig["params"]), nil, task.conversationID)
	if err != nil {
		return nil, err
	}
	output := make(map[string]any, len(node.Config.OutputParams))
	content := strings.TrimSpace(fmt.Sprint(response.Message.Content))
	content = strings.TrimPrefix(strings.TrimSuffix(content, "```"), "```json")
	if err := json.Unmarshal([]byte(strings.TrimSpace(content)), &output); err != nil {
		return nil, fmt.Errorf("parameter extractor model response is not JSON: %w", err)
	}
	if _, exists := output["_is_completed"]; !exists {
		output["_is_completed"] = true
	}
	return output, nil
}

func (s *Service) executeAPINode(ctx context.Context, task *task, node Node) (map[string]any, error) {
	config := node.Config.NodeParam
	target, err := url.Parse(task.resolve(stringValue(config["url"])))
	if err != nil || target.Scheme == "" || target.Host == "" || (target.Scheme != "http" && target.Scheme != "https") {
		return nil, errors.New("API node url must be an http(s) URL")
	}
	for _, param := range paramsValue(config["params"]) {
		target.Query().Set(param.Key, fmt.Sprint(task.paramValue(param)))
	}
	query := target.Query()
	for _, param := range paramsValue(config["params"]) {
		query.Set(param.Key, fmt.Sprint(task.paramValue(param)))
	}
	target.RawQuery = query.Encode()
	body, contentType, err := apiRequestBody(task, mapValue(config["body"]))
	if err != nil {
		return nil, err
	}
	timeout := 30 * time.Second
	if timeoutConfig := mapValue(config["timeout"]); timeoutConfig["read"] != nil {
		if seconds, parseErr := strconv.Atoi(fmt.Sprint(timeoutConfig["read"])); parseErr == nil && seconds > 0 {
			timeout = time.Duration(seconds) * time.Second
		}
	}
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, strings.ToUpper(defaultString(stringValue(config["method"]), http.MethodGet)), target.String(), body)
	if err != nil {
		return nil, err
	}
	for _, header := range paramsValue(config["headers"]) {
		request.Header.Set(header.Key, fmt.Sprint(task.paramValue(header)))
	}
	if contentType != "" && request.Header.Get("Content-Type") == "" {
		request.Header.Set("Content-Type", contentType)
	}
	authorization := mapValue(config["authorization"])
	applyAPIAuthorization(request, authorization)
	response, err := s.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("API node request failed with status %d", response.StatusCode)
	}
	var value any = string(payload)
	if strings.EqualFold(stringValue(config["output_type"]), "json") && json.Valid(payload) {
		if err := json.Unmarshal(payload, &value); err != nil {
			return nil, err
		}
	}
	return mapNodeOutput(node, value), nil
}

func apiRequestBody(task *task, config map[string]any) (io.Reader, string, error) {
	switch stringValue(config["type"]) {
	case "", "none":
		return nil, "", nil
	case "raw":
		return strings.NewReader(task.resolve(stringValue(config["data"]))), "text/plain", nil
	case "json":
		return strings.NewReader(task.resolve(stringValue(config["data"]))), "application/json", nil
	case "form-data":
		values := url.Values{}
		for _, param := range paramsValue(config["data"]) {
			values.Set(param.Key, fmt.Sprint(task.paramValue(param)))
		}
		return strings.NewReader(values.Encode()), "application/x-www-form-urlencoded", nil
	default:
		return nil, "", fmt.Errorf("unsupported API body type: %s", stringValue(config["type"]))
	}
}

func applyAPIAuthorization(request *http.Request, config map[string]any) {
	authConfig := mapValue(config["auth_config"])
	switch stringValue(config["auth_type"]) {
	case "BearerAuth":
		if token := stringValue(authConfig["token"]); token != "" {
			request.Header.Set("Authorization", "Bearer "+token)
		}
	case "ApiKeyAuth":
		for key, value := range authConfig {
			request.Header.Set(key, fmt.Sprint(value))
		}
	}
}

func (s *Service) executeRetrievalNode(ctx context.Context, task *task, node Node) (map[string]any, error) {
	if s.documents == nil {
		return nil, errors.New("knowledge retrieval service is unavailable")
	}
	config := node.Config.NodeParam
	ids := make([]string, 0)
	for _, item := range listValue(config["knowledge_base_ids"]) {
		if id := stringValue(item); id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 || len(node.Config.InputParams) == 0 {
		return nil, ErrWorkflowConfig
	}
	topK, _ := strconv.Atoi(fmt.Sprint(config["top_k"]))
	chunks, err := s.documents.Search(ctx, task.workspaceID, ids, fmt.Sprint(task.paramValue(node.Config.InputParams[0])), topK)
	if err != nil {
		return nil, err
	}
	return mapNodeOutput(node, chunks), nil
}

func (s *Service) executeMCPNode(ctx context.Context, task *task, node Node) (map[string]any, error) {
	if s.mcpServers == nil {
		return nil, errors.New("MCP service is unavailable")
	}
	config := node.Config.NodeParam
	result, err := s.mcpServers.CallTool(ctx, task.workspaceID, mcpserver.ToolCallInput{ServerCode: stringValue(config["server_code"]), ToolName: stringValue(config["tool_name"]), ToolParams: task.inputValues(node.Config.InputParams)})
	if err != nil {
		return nil, err
	}
	if result.IsError {
		return nil, fmt.Errorf("MCP tool returned an error: %v", result.Content)
	}
	return mapNodeOutput(node, result.Content), nil
}

func (s *Service) executePluginNode(ctx context.Context, task *task, node Node) (map[string]any, error) {
	if s.plugins == nil {
		return nil, errors.New("plugin service is unavailable")
	}
	config := node.Config.NodeParam
	result, err := s.plugins.InvokeTool(ctx, task.workspaceID, stringValue(config["plugin_id"]), stringValue(config["tool_id"]), task.inputValues(node.Config.InputParams))
	if err != nil {
		return nil, err
	}
	if !result.Success {
		return nil, errors.New(result.Message)
	}
	return mapNodeOutput(node, result.Data), nil
}

func (s *Service) executeAppComponentNode(ctx context.Context, task *task, node Node) (map[string]any, error) {
	if s.components == nil {
		return nil, errors.New("application component service is unavailable")
	}
	component, err := s.components.Get(ctx, task.workspaceID, stringValue(node.Config.NodeParam["code"]))
	if err != nil {
		return nil, err
	}
	inputs := task.inputValues(node.Config.InputParams)
	if strings.EqualFold(stringValue(node.Config.NodeParam["type"]), "workflow") {
		runInputs := make([]Param, 0, len(inputs))
		for key, value := range inputs {
			runInputs = append(runInputs, Param{Key: key, Value: value})
		}
		run, err := s.Run(ctx, task.workspaceID, task.requestID, TaskRunRequest{AppID: component.AppID, ConversationID: task.conversationID, Inputs: runInputs})
		if err != nil {
			return nil, err
		}
		for {
			process, processErr := s.Process(task.workspaceID, run.TaskID)
			if processErr != nil {
				return nil, processErr
			}
			if process.TaskStatus == statusSuccess {
				if len(process.TaskResults) == 0 {
					return mapNodeOutput(node, nil), nil
				}
				return mapNodeOutput(node, process.TaskResults[len(process.TaskResults)-1].NodeContent), nil
			}
			if process.TaskStatus != statusExecuting {
				return nil, errors.New(process.ErrorInfo)
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(10 * time.Millisecond):
			}
		}
	}
	completer, ok := s.models.(ApplicationCompleter)
	if !ok {
		return nil, errors.New("application component model service is unavailable")
	}
	response, err := completer.Complete(ctx, task.workspaceID, chat.Request{AppID: component.AppID, ConversationID: task.conversationID, Messages: []chat.Message{{Role: "user", Content: inputs}}, Parameter: inputs})
	if err != nil {
		return nil, err
	}
	return mapNodeOutput(node, response.Message.Content), nil
}

// executeGroupNode 按前端保存的 node_param.block 运行循环或批处理子图。每轮使用独立变量
// 快照，防止异步或嵌套任务污染父流程；完成后只把容器声明的输出聚合回父流程。
func (s *Service) executeGroupNode(ctx context.Context, task *task, node Node) (map[string]any, error) {
	block, err := parseBlock(node.Config.NodeParam["block"])
	if err != nil {
		return nil, err
	}
	iterations, err := task.groupIterations(node)
	if err != nil {
		return nil, err
	}
	output := make(map[string]any, len(node.Config.OutputParams))
	for _, param := range node.Config.OutputParams {
		output[param.Key] = []any{}
	}
	if node.Type == "Parallel" {
		return s.executeParallelBlock(ctx, task, node, block, iterations, output)
	}
	for index, values := range iterations {
		nested, err := s.executeBlock(ctx, task, block, values)
		if err != nil {
			return nil, fmt.Errorf("%s batch %d failed: %w", node.Type, index+1, err)
		}
		for _, param := range node.Config.OutputParams {
			value := nested.paramValue(param)
			output[param.Key] = append(output[param.Key].([]any), value)
		}
	}
	return output, nil
}

func (s *Service) executeParallelBlock(ctx context.Context, parent *task, node Node, block config, iterations []map[string]any, output map[string]any) (map[string]any, error) {
	concurrency, err := strconv.Atoi(fmt.Sprint(node.Config.NodeParam["concurrent_size"]))
	if err != nil || concurrency <= 0 {
		concurrency = 1
	}
	if concurrency > len(iterations) {
		concurrency = len(iterations)
	}
	if concurrency == 0 {
		return output, nil
	}
	type result struct {
		index int
		task  *task
		err   error
	}
	jobs := make(chan int)
	results := make(chan result, len(iterations))
	var workers sync.WaitGroup
	for worker := 0; worker < concurrency; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for index := range jobs {
				nested, nestedErr := s.executeBlock(ctx, parent, block, iterations[index])
				results <- result{index: index, task: nested, err: nestedErr}
			}
		}()
	}
	for index := range iterations {
		jobs <- index
	}
	close(jobs)
	workers.Wait()
	close(results)

	ordered := make([]result, len(iterations))
	for result := range results {
		ordered[result.index] = result
	}
	strategy := stringValue(node.Config.NodeParam["error_strategy"])
	for _, result := range ordered {
		if result.err != nil {
			if strategy == "terminated" || strategy == "" {
				return nil, fmt.Errorf("Parallel batch %d failed: %w", result.index+1, result.err)
			}
			if strategy == "continueOnError" {
				for _, param := range node.Config.OutputParams {
					output[param.Key] = append(output[param.Key].([]any), nil)
				}
			}
			continue
		}
		for _, param := range node.Config.OutputParams {
			output[param.Key] = append(output[param.Key].([]any), result.task.paramValue(param))
		}
	}
	return output, nil
}

func (s *Service) executeBlock(ctx context.Context, parent *task, block config, variables map[string]any) (*task, error) {
	nested := parent.newChild(block, variables)
	s.execute(ctx, nested, startNodes(block))
	if nested.status != statusSuccess {
		return nested, errors.New(nested.errorInfo)
	}
	return nested, nil
}

func parseBlock(value any) (config, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return config{}, err
	}
	var block config
	if err := json.Unmarshal(data, &block); err != nil || len(block.Nodes) == 0 {
		return config{}, ErrWorkflowConfig
	}
	return block, nil
}

func (t *task) newChild(block config, variables map[string]any) *task {
	t.mu.RLock()
	copyValues := make(map[string]any, len(t.variables)+len(variables))
	for key, value := range t.variables {
		copyValues[key] = value
	}
	t.mu.RUnlock()
	for key, value := range variables {
		copyValues[key] = value
	}
	return &task{workspaceID: t.workspaceID, appID: t.appID, id: t.id, requestID: t.requestID, conversationID: t.conversationID, startedAt: t.startedAt, config: block, variables: copyValues, status: statusExecuting, results: make(map[string]NodeResult), branchHandles: make(map[string]string)}
}

func (t *task) groupIterations(node Node) ([]map[string]any, error) {
	config := node.Config.NodeParam
	inputs := t.inputValues(node.Config.InputParams)
	count := 1
	if node.Type == "Iterator" && strings.EqualFold(stringValue(config["iterator_type"]), "byCount") {
		parsed, err := strconv.Atoi(fmt.Sprint(config["count_limit"]))
		if err != nil || parsed < 0 {
			return nil, ErrWorkflowConfig
		}
		count = parsed
	} else {
		count = -1
		for _, value := range inputs {
			items := arrayValue(value)
			if items == nil {
				return nil, errors.New("group node inputs must be arrays")
			}
			if count < 0 || len(items) < count {
				count = len(items)
			}
		}
		if count < 0 {
			count = 0
		}
	}
	if count > 500 {
		count = 500
	}
	intermediate := t.inputValues(paramsValue(config["variable_parameters"]))
	result := make([]map[string]any, 0, count)
	for index := 0; index < count; index++ {
		values := make(map[string]any, len(inputs)+len(intermediate)+1)
		for key, value := range intermediate {
			values[node.ID+"."+key] = value
		}
		for key, value := range inputs {
			if items := arrayValue(value); items != nil {
				values[node.ID+"."+key] = items[index]
			}
		}
		values[node.ID+".index"] = index + 1
		result = append(result, values)
	}
	return result, nil
}

func arrayValue(value any) []any {
	if items, ok := value.([]any); ok {
		return items
	}
	if text, ok := value.(string); ok {
		var items []any
		if json.Unmarshal([]byte(text), &items) == nil {
			return items
		}
	}
	return nil
}

func mapNodeOutput(node Node, value any) map[string]any {
	output := map[string]any{"output": value}
	for _, param := range node.Config.OutputParams {
		if param.Key != "" {
			output[param.Key] = value
		}
	}
	return output
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func (s *Service) loadConfig(ctx context.Context, workspaceID, appID, version string) (config, error) {
	if s.applications == nil {
		return config{}, ErrWorkflowConfig
	}
	if strings.TrimSpace(version) == "" {
		version = "latest"
	}
	value, err := s.applications.GetVersion(ctx, workspaceID, appID, version)
	if err != nil {
		return config{}, err
	}
	var cfg config
	if err := json.Unmarshal([]byte(value.Config), &cfg); err != nil || len(cfg.Nodes) == 0 {
		return config{}, ErrWorkflowConfig
	}
	return cfg, nil
}

func (s *Service) getTask(workspaceID, taskID string) (*task, error) {
	s.mu.RLock()
	value := s.tasks[taskKey(workspaceID, taskID)]
	s.mu.RUnlock()
	if value == nil {
		if s.state == nil {
			return nil, ErrTaskNotFound
		}
		raw, err := s.state.Load(context.Background(), workspaceID, taskID)
		if err != nil {
			return nil, err
		}
		value, err = taskFromSnapshot(raw)
		if err != nil {
			return nil, err
		}
		s.mu.Lock()
		s.tasks[taskKey(workspaceID, taskID)] = value
		s.mu.Unlock()
	}
	return value, nil
}

func (s *Service) saveTask(ctx context.Context, task *task) error {
	if s.state == nil {
		return nil
	}
	snapshot, err := task.snapshot()
	if err != nil {
		return err
	}
	return s.state.Save(ctx, task.workspaceID, task.id, snapshot)
}

func (t *task) isTerminal() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.status == statusStop || t.status == statusFail || t.status == statusPause || t.status == statusSuccess
}

func (t *task) fail(code, message string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.status == statusStop {
		return
	}
	t.status = statusFail
	t.errorCode = code
	t.errorInfo = message
}

// failNode keeps task and node state consistent. The frontend renders node_results
// independently from task_status, so leaving the last node as executing after a model
// error incorrectly suggests that polling should continue forever.
func (t *task) failNode(node Node, code, message string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.status == statusStop {
		return
	}
	t.status = statusFail
	t.errorCode = code
	t.errorInfo = message
	result, exists := t.results[node.ID]
	if !exists {
		result = NodeResult{NodeID: node.ID, NodeName: node.Name, NodeType: node.Type, Batches: []NodeResult{}}
	}
	result.NodeStatus = statusFail
	result.ErrorCode = code
	result.ErrorInfo = message
	t.recordLocked(result)
}

func (t *task) complete(node Node, output map[string]any) {
	t.bindOutput(node, output)
	t.mu.Lock()
	t.recordLocked(NodeResult{NodeID: node.ID, NodeName: node.Name, NodeType: node.Type, NodeStatus: statusSuccess, Input: node.Config.InputParams, Output: output})
	t.mu.Unlock()
}

// 订阅刻意限定在当前任务实例。Redis 持久化的是可恢复的任务状态，而 HTTP 响应归属于当前进程，
// 重连或跨实例后应继续使用既有轮询接口获取状态，不能伪造另一进程中的历史流。
func (t *task) subscribe() (<-chan StreamEvent, func()) {
	channel := make(chan StreamEvent, 512)
	t.mu.Lock()
	if t.streamClosed {
		close(channel)
	} else {
		if t.subscribers == nil {
			t.subscribers = make(map[chan StreamEvent]struct{})
		}
		t.subscribers[channel] = struct{}{}
	}
	t.mu.Unlock()
	return channel, func() {
		t.mu.Lock()
		if _, exists := t.subscribers[channel]; exists {
			delete(t.subscribers, channel)
			close(channel)
		}
		t.mu.Unlock()
	}
}

func (t *task) publishNode(nodeID string) {
	t.mu.Lock()
	result, exists := t.results[nodeID]
	if !exists || t.streamClosed {
		t.mu.Unlock()
		return
	}
	t.eventSequence++
	event := StreamEvent{
		Event:           "Message",
		TaskID:          t.id,
		ConversationID:  t.conversationID,
		NodeID:          result.NodeID,
		NodeName:        result.NodeName,
		NodeType:        result.NodeType,
		NodeStatus:      result.NodeStatus,
		NodeMsgSeqID:    t.eventSequence,
		NodeIsCompleted: result.NodeStatus == statusSuccess,
		TextContent:     streamText(result.Output),
		ErrorCode:       result.ErrorCode,
		ErrorMessage:    result.ErrorInfo,
	}
	t.publishLocked(event)
	t.mu.Unlock()
}

func (t *task) finishStream() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.streamClosed {
		return
	}
	event := StreamEvent{Event: "Finished", TaskID: t.id, ConversationID: t.conversationID}
	switch t.status {
	case statusPause:
		event.Event = "Paused"
		event.PauseType = "InputNodeInterrupt"
	case statusFail, statusStop:
		event.Event = "Error"
		event.ErrorCode = t.errorCode
		event.ErrorMessage = t.errorInfo
	}
	t.publishLocked(event)
	t.streamClosed = true
	for channel := range t.subscribers {
		close(channel)
		delete(t.subscribers, channel)
	}
}

func (t *task) publishLocked(event StreamEvent) {
	for channel := range t.subscribers {
		// writer 会经 io.Pipe 持续消费。有限缓冲防止已断开的客户端阻塞共享工作流执行器；
		// 普通节点事件溢出时可由轮询接口补齐，终态事件则淘汰最早事件后强制保留。
		select {
		case channel <- event:
		default:
			if event.Event == "Message" {
				continue
			}
			select {
			case <-channel:
			default:
			}
			select {
			case channel <- event:
			default:
			}
		}
	}
}

func streamText(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func (t *task) bindConfiguredOutputs(node Node) map[string]any {
	t.mu.Lock()
	defer t.mu.Unlock()
	output := map[string]any{}
	for _, param := range node.Config.OutputParams {
		value := t.variables[param.Key]
		if value == nil {
			value = param.DefaultValue
		}
		output[param.Key] = value
		t.variables[node.ID+"."+param.Key] = value
	}
	return output
}

func (t *task) bindOutput(node Node, output map[string]any) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for key, value := range output {
		t.variables[node.ID+"."+key] = value
	}
}

// executeScriptNode 与 Java ScriptExecuteProcessor 一样，将 Python 和 JavaScript 交给隔离的
// OpenClaw4j-Sandbox。工作流服务只负责编排、参数绑定和结果映射，不能在自身进程内执行用户代码。
func (t *task) executeScriptNode(ctx context.Context, node Node, executor scriptsandbox.Executor) (map[string]any, error) {
	scriptType := strings.ToLower(stringValue(node.Config.NodeParam["script_type"]))
	if scriptType != "javascript" && scriptType != "python" {
		return nil, fmt.Errorf("unsupported script type: %s, currently only python and javascript are supported", scriptType)
	}
	scriptContent := stringValue(node.Config.NodeParam["script_content"])
	if scriptContent == "" {
		return nil, errors.New("script_content is required")
	}
	if executor == nil {
		return nil, errors.New("script sandbox is unavailable")
	}

	inputs, err := t.scriptInputs(node.Config.InputParams)
	if err != nil {
		return nil, err
	}
	result, err := executor.Execute(ctx, scriptType, scriptContent, inputs, t.requestID)
	if err != nil {
		return nil, err
	}
	if !result.Success {
		if result.Code != "" {
			return nil, fmt.Errorf("script sandbox execution failed (%s): %s", result.Code, result.Message)
		}
		return nil, fmt.Errorf("script sandbox execution failed: %s", result.Message)
	}
	values, ok := result.Data.(map[string]any)
	if !ok {
		return nil, errors.New("script sandbox must return an object")
	}

	output := make(map[string]any, len(node.Config.OutputParams))
	for _, param := range node.Config.OutputParams {
		output[param.Key] = values[param.Key]
	}
	return output, nil
}

func (t *task) scriptInputs(params []Param) (map[string]any, error) {
	values := make(map[string]any, len(params))
	for _, param := range params {
		value := param.Value
		if strings.EqualFold(param.ValueFrom, "refer") {
			value = t.resolveScriptReference(value)
		}
		converted, err := convertScriptValue(param.Type, value)
		if err != nil {
			return nil, fmt.Errorf("invalid script input %q: %w", param.Key, err)
		}
		if converted != nil {
			values[param.Key] = converted
		}
	}
	return values, nil
}

func (t *task) resolveScriptReference(value any) any {
	text, ok := value.(string)
	if !ok {
		return value
	}
	matches := variableExpression.FindStringSubmatch(text)
	if len(matches) > 0 && matches[0] == text {
		key := strings.TrimSuffix(strings.TrimPrefix(text, "${"), "}")
		t.mu.RLock()
		resolved, found := t.variables[key]
		t.mu.RUnlock()
		if found {
			return resolved
		}
	}
	return t.resolve(text)
}

func convertScriptValue(valueType string, value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	switch strings.ToLower(valueType) {
	case "", "string":
		if valueType == "" {
			return value, nil
		}
		if _, ok := value.(string); ok {
			return value, nil
		}
		return fmt.Sprint(value), nil
	case "number":
		if _, ok := value.(float64); ok {
			return value, nil
		}
		if _, ok := value.(float32); ok {
			return value, nil
		}
		if _, ok := value.(int); ok {
			return value, nil
		}
		if _, ok := value.(int64); ok {
			return value, nil
		}
		text := strings.TrimSpace(fmt.Sprint(value))
		if integer, err := strconv.ParseInt(text, 10, 64); err == nil {
			return integer, nil
		}
		number, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return nil, fmt.Errorf("%q is not a number", text)
		}
		return number, nil
	case "boolean":
		if boolean, ok := value.(bool); ok {
			return boolean, nil
		}
		boolean, err := strconv.ParseBool(strings.TrimSpace(fmt.Sprint(value)))
		if err != nil {
			return nil, fmt.Errorf("%q is not a boolean", value)
		}
		return boolean, nil
	case "object", "arrayobject", "arraystring", "arraynumber", "arrayboolean", "arrayfile", "file":
		if _, ok := value.(string); !ok {
			return value, nil
		}
		var decoded any
		if err := json.Unmarshal([]byte(value.(string)), &decoded); err != nil {
			return nil, err
		}
		return decoded, nil
	default:
		return value, nil
	}
}

func (t *task) resolve(value string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return variableExpression.ReplaceAllStringFunc(value, func(expression string) string {
		key := strings.TrimSuffix(strings.TrimPrefix(expression, "${"), "}")
		if resolved, ok := t.variables[key]; ok && resolved != nil {
			return fmt.Sprint(resolved)
		}
		return ""
	})
}

func (t *task) recordLocked(result NodeResult) {
	// Java NodeResult 将输入与输出序列化为 JSON 字符串。前端的 JSONViewer/CodeMirror
	// 也依赖这个契约，直接返回 object 会导致结果面板抛出运行时异常。
	result.Input = jsonText(result.Input)
	result.Output = jsonText(result.Output)
	if result.Batches == nil {
		result.Batches = []NodeResult{}
	}
	if _, exists := t.results[result.NodeID]; !exists {
		t.order = append(t.order, result.NodeID)
	}
	t.results[result.NodeID] = result
}

func jsonText(value any) any {
	if value == nil {
		return nil
	}
	if _, ok := value.(string); ok {
		return value
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(encoded)
}

func (t *task) setBranchHandle(nodeID, handle string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.branchHandles[nodeID] = handle
}

func (t *task) takeBranchHandle(nodeID string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.branchHandles[nodeID]
}

var variableExpression = regexp.MustCompile(`\$\{[^}]+}`)

func startNodes(cfg config) []string {
	starts := make([]string, 0)
	for _, node := range cfg.Nodes {
		if node.Type == "Start" {
			starts = append(starts, node.ID)
		}
	}
	if len(starts) > 0 {
		return starts
	}
	incoming := make(map[string]bool)
	for _, edge := range cfg.Edges {
		incoming[edge.Target] = true
	}
	result := make([]string, 0)
	for _, node := range cfg.Nodes {
		if !incoming[node.ID] {
			result = append(result, node.ID)
		}
	}
	return result
}

func targets(edges []Edge, nodeID, handle string) []string {
	result := make([]string, 0)
	for _, edge := range edges {
		if edge.Source != nodeID {
			continue
		}
		if handle != "" && edge.SourceHandle != handle {
			continue
		}
		result = append(result, edge.Target)
	}
	return result
}

func nodeByID(nodes []Node, id string) (Node, bool) {
	for _, node := range nodes {
		if node.ID == id {
			return node, true
		}
	}
	return Node{}, false
}

func taskKey(workspaceID, taskID string) string { return workspaceID + ":" + taskID }

func identifier() (string, error) {
	var value [12]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}

func valueOrDefault(value, fallback any) any {
	if value != nil {
		return value
	}
	return fallback
}

func stringValue(value any) string {
	result, _ := value.(string)
	return strings.TrimSpace(result)
}

func mapValue(value any) map[string]any {
	result, _ := value.(map[string]any)
	return result
}

func parameterValues(value any) map[string]any {
	values, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make(map[string]any)
	for _, item := range values {
		parameter := mapValue(item)
		if key := stringValue(parameter["key"]); key != "" && parameter["value"] != nil {
			result[key] = parameter["value"]
		}
	}
	return result
}
