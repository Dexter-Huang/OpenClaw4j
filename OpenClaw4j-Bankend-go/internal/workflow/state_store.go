package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// StateStore owns durable task snapshots. Implementations must atomically replace a
// whole snapshot so readers never combine variables from one node with results from
// another node.
type StateStore interface {
	Save(context.Context, string, string, []byte) error
	Load(context.Context, string, string) ([]byte, error)
}

type RedisStateStore struct {
	client redis.UniversalClient
	ttl    time.Duration
}

func NewRedisStateStore(client redis.UniversalClient, ttl time.Duration) *RedisStateStore {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &RedisStateStore{client: client, ttl: ttl}
}

func (s *RedisStateStore) Save(ctx context.Context, workspaceID, taskID string, snapshot []byte) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("workflow state redis client is unavailable")
	}
	if err := s.client.Set(ctx, workflowStateKey(workspaceID, taskID), snapshot, s.ttl).Err(); err != nil {
		return fmt.Errorf("save workflow state: %w", err)
	}
	return nil
}
func (s *RedisStateStore) Load(ctx context.Context, workspaceID, taskID string) ([]byte, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("workflow state redis client is unavailable")
	}
	value, err := s.client.Get(ctx, workflowStateKey(workspaceID, taskID)).Bytes()
	if err == redis.Nil {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load workflow state: %w", err)
	}
	return value, nil
}
func workflowStateKey(workspaceID, taskID string) string {
	return "workflow:task:" + workspaceID + ":" + taskID
}

type taskSnapshot struct {
	WorkspaceID    string                `json:"workspace_id"`
	AppID          string                `json:"app_id"`
	ID             string                `json:"id"`
	RequestID      string                `json:"request_id"`
	ConversationID string                `json:"conversation_id"`
	StartedAt      time.Time             `json:"started_at"`
	Config         config                `json:"config"`
	Variables      map[string]any        `json:"variables"`
	Status         string                `json:"status"`
	ErrorCode      string                `json:"error_code"`
	ErrorInfo      string                `json:"error_info"`
	Results        map[string]NodeResult `json:"results"`
	Order          []string              `json:"order"`
	PendingNodeID  string                `json:"pending_node_id"`
	BranchHandles  map[string]string     `json:"branch_handles"`
}

func (t *task) snapshot() ([]byte, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return json.Marshal(taskSnapshot{WorkspaceID: t.workspaceID, AppID: t.appID, ID: t.id, RequestID: t.requestID, ConversationID: t.conversationID, StartedAt: t.startedAt, Config: t.config, Variables: t.variables, Status: t.status, ErrorCode: t.errorCode, ErrorInfo: t.errorInfo, Results: t.results, Order: t.order, PendingNodeID: t.pendingNodeID, BranchHandles: t.branchHandles})
}
func taskFromSnapshot(raw []byte) (*task, error) {
	var value taskSnapshot
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("decode workflow state: %w", err)
	}
	if value.ID == "" || value.WorkspaceID == "" {
		return nil, ErrTaskNotFound
	}
	if value.Variables == nil {
		value.Variables = map[string]any{}
	}
	if value.Results == nil {
		value.Results = map[string]NodeResult{}
	}
	if value.BranchHandles == nil {
		value.BranchHandles = map[string]string{}
	}
	return &task{workspaceID: value.WorkspaceID, appID: value.AppID, id: value.ID, requestID: value.RequestID, conversationID: value.ConversationID, startedAt: value.StartedAt, config: value.Config, variables: value.Variables, status: value.Status, errorCode: value.ErrorCode, errorInfo: value.ErrorInfo, results: value.Results, order: value.Order, pendingNodeID: value.PendingNodeID, branchHandles: value.BranchHandles}, nil
}

var _ StateStore = (*RedisStateStore)(nil)
