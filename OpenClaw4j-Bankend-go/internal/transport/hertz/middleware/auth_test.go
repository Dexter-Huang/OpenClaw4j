package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/seaskyland/openclaw4j-backend-go/internal/auth"
	"github.com/seaskyland/openclaw4j-backend-go/internal/contextx"
)

func TestConsoleAuthBindsRequestContextFromXSaaToken(t *testing.T) {
	requestContext := &contextx.RequestContext{
		RequestID: "req_1",
		AccountID: "acct_1",
		AuthType:  contextx.AuthTypeConsoleToken,
	}
	authenticator := &fakeAuthenticator{consoleContext: requestContext}

	called, rc := runMiddleware(t, ConsoleAuth(authenticator), func(context.Context, *app.RequestContext) {})
	if !called {
		t.Fatalf("next handler was not called")
	}

	if rc != requestContext {
		t.Fatalf("request context was not bound: %#v", rc)
	}
	if authenticator.consoleToken != "console-token" {
		t.Fatalf("console token was not extracted: %#v", authenticator)
	}
}

func TestConsoleAuthAcceptsAccessTokenQueryForSSECompatibility(t *testing.T) {
	authenticator := &fakeAuthenticator{consoleContext: &contextx.RequestContext{RequestID: "req_1"}}
	ctx := newHertzContext()
	ctx.Request.SetRequestURI("/console/v1/events?access_token=query-token")
	ctx.SetHandlers(app.HandlersChain{
		ConsoleAuth(authenticator),
		func(context.Context, *app.RequestContext) {},
	})

	ctx.Next(context.Background())

	if authenticator.consoleToken != "query-token" {
		t.Fatalf("query access_token was not used: %#v", authenticator)
	}
	if ctx.Response.StatusCode() >= consts.StatusBadRequest {
		t.Fatalf("unexpected status: %d", ctx.Response.StatusCode())
	}
}

func TestConsoleAuthRejectsMissingToken(t *testing.T) {
	authenticator := &fakeAuthenticator{consoleErr: auth.ErrInvalidToken}
	ctx := newHertzContext()
	ctx.Request.Header.Del(HeaderXSaaToken)
	ctx.SetHandlers(app.HandlersChain{
		ConsoleAuth(authenticator),
		func(context.Context, *app.RequestContext) {
			t.Fatalf("next handler should not run")
		},
	})

	ctx.Next(context.Background())

	if ctx.Response.StatusCode() != consts.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", ctx.Response.StatusCode())
	}
}

func TestAPIKeyAuthBindsRequestContextFromAuthorizationBearer(t *testing.T) {
	requestContext := &contextx.RequestContext{
		RequestID: "req_api",
		AccountID: "acct_api",
		AuthType:  contextx.AuthTypeAPIKey,
	}
	authenticator := &fakeAuthenticator{apiKeyContext: requestContext}
	ctx := newHertzContext()
	ctx.Request.Header.Set(HeaderAuthorization, "Bearer sk-live")
	var rc *contextx.RequestContext
	ctx.SetHandlers(app.HandlersChain{
		APIKeyAuth(authenticator),
		func(c context.Context, _ *app.RequestContext) {
			rc, _ = contextx.From(c)
		},
	})

	ctx.Next(context.Background())

	if rc != requestContext {
		t.Fatalf("request context was not bound: %#v", rc)
	}
	if authenticator.apiKey != "sk-live" {
		t.Fatalf("API key was not extracted: %#v", authenticator)
	}
}

func TestAPIKeyAuthAcceptsXAPIKeyHeader(t *testing.T) {
	authenticator := &fakeAuthenticator{apiKeyContext: &contextx.RequestContext{RequestID: "req_api"}}
	ctx := newHertzContext()
	ctx.Request.Header.Set(HeaderXAPIKey, "sk-header")
	ctx.SetHandlers(app.HandlersChain{
		APIKeyAuth(authenticator),
		func(context.Context, *app.RequestContext) {},
	})

	ctx.Next(context.Background())

	if authenticator.apiKey != "sk-header" {
		t.Fatalf("X-API-Key was not used: %#v", authenticator)
	}
}

func TestAuthMiddlewareSkipsOptions(t *testing.T) {
	authenticator := &fakeAuthenticator{consoleErr: errors.New("should not be called")}
	ctx := newHertzContext()
	ctx.Request.Header.SetMethod(consts.MethodOptions)
	called := false
	ctx.SetHandlers(app.HandlersChain{
		ConsoleAuth(authenticator),
		func(context.Context, *app.RequestContext) {
			called = true
		},
	})

	ctx.Next(context.Background())

	if !called {
		t.Fatalf("OPTIONS request should pass through")
	}
	if authenticator.consoleCalled || authenticator.apiKeyCalled {
		t.Fatalf("authenticator should not be called for OPTIONS")
	}
}

func runMiddleware(t *testing.T, middleware app.HandlerFunc, next app.HandlerFunc) (bool, *contextx.RequestContext) {
	t.Helper()
	ctx := newHertzContext()
	ctx.Request.Header.Set(HeaderXSaaToken, "Bearer console-token")
	var rc *contextx.RequestContext
	called := false
	ctx.SetHandlers(app.HandlersChain{
		middleware,
		func(c context.Context, request *app.RequestContext) {
			called = true
			next(c, request)
			rc, _ = contextx.From(c)
		},
	})
	ctx.Next(context.Background())
	return called, rc
}

func newHertzContext() *app.RequestContext {
	ctx := app.NewContext(0)
	ctx.Request.SetRequestURI("/")
	ctx.Request.Header.SetMethod(consts.MethodGet)
	ctx.Request.Header.Set("X-Real-IP", "127.0.0.1")
	return ctx
}

type fakeAuthenticator struct {
	consoleContext *contextx.RequestContext
	apiKeyContext  *contextx.RequestContext
	consoleErr     error
	apiKeyErr      error

	consoleToken  string
	apiKey        string
	callerIP      string
	consoleCalled bool
	apiKeyCalled  bool
}

func (f *fakeAuthenticator) AuthenticateConsoleToken(_ context.Context, token string, callerIP string) (*contextx.RequestContext, error) {
	f.consoleCalled = true
	f.consoleToken = token
	f.callerIP = callerIP
	if f.consoleErr != nil {
		return nil, f.consoleErr
	}
	return f.consoleContext, nil
}

func (f *fakeAuthenticator) AuthenticateAPIKey(_ context.Context, apiKey string, callerIP string) (*contextx.RequestContext, error) {
	f.apiKeyCalled = true
	f.apiKey = apiKey
	f.callerIP = callerIP
	if f.apiKeyErr != nil {
		return nil, f.apiKeyErr
	}
	return f.apiKeyContext, nil
}
