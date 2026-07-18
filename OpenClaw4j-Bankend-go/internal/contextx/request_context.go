package contextx

import (
	"context"
	"time"
)

type AuthType string

const (
	AuthTypeConsoleToken AuthType = "console_token"
	AuthTypeAPIKey       AuthType = "api_key"
)

// RequestContext 保存一次请求在鉴权后形成的身份与追踪信息。
// Go 版通过 context.Context 显式传递它，避免模拟不可靠的 goroutine-local storage。
type RequestContext struct {
	StartTime   time.Time `json:"start_time"`
	RequestID   string    `json:"request_id"`
	AccountID   string    `json:"account_id"`
	Username    string    `json:"username"`
	AccountType string    `json:"account_type"`
	WorkspaceID string    `json:"workspace_id"`
	CallerIP    string    `json:"caller_ip"`
	Source      string    `json:"source"`
	AuthType    AuthType  `json:"auth_type"`
}

type requestContextKey struct{}

func With(ctx context.Context, rc *RequestContext) context.Context {
	if rc == nil {
		return ctx
	}
	return context.WithValue(ctx, requestContextKey{}, rc)
}

func From(ctx context.Context) (*RequestContext, bool) {
	if ctx == nil {
		return nil, false
	}
	rc, ok := ctx.Value(requestContextKey{}).(*RequestContext)
	return rc, ok
}

func MustFrom(ctx context.Context) *RequestContext {
	rc, ok := From(ctx)
	if !ok {
		panic("request context is not bound")
	}
	return rc
}

// Detach 复制请求身份信息，但切断父 context 的取消信号。
// 用于必须在 HTTP/SSE 断开后继续运行的 workflow 或后台任务。
func Detach(ctx context.Context) context.Context {
	rc, ok := From(ctx)
	if !ok {
		return context.Background()
	}
	return With(context.Background(), rc)
}
