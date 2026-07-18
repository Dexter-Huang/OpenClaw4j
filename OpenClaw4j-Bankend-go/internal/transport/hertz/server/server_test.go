package server

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/seaskyland/openclaw4j-backend-go/internal/contextx"
)

func TestNewRegistersPublicHealthRoute(t *testing.T) {
	h := New(Options{})

	ctx := request(consts.MethodGet, "/healthz")
	ctx.Request.Header.Set("Origin", "http://127.0.0.1:8000")
	h.ServeHTTP(context.Background(), ctx)

	if ctx.Response.StatusCode() != consts.StatusOK {
		t.Fatalf("expected 200, got %d", ctx.Response.StatusCode())
	}
	if !strings.Contains(string(ctx.Response.Body()), `"status":"ok"`) {
		t.Fatalf("unexpected body: %s", ctx.Response.Body())
	}
	if string(ctx.Response.Header.Peek("Access-Control-Allow-Origin")) != "http://127.0.0.1:8000" {
		t.Fatalf("cors origin header was not set: %s", ctx.Response.Header.Header())
	}
}

func TestNewHandlesCORSPreflightForUnknownConsoleRoute(t *testing.T) {
	h := New(Options{})

	ctx := request(consts.MethodOptions, "/console/v1/auth/login")
	ctx.Request.Header.Set("Origin", "http://127.0.0.1:8000")
	ctx.Request.Header.Set("Access-Control-Request-Headers", "content-type, x-saa-token")
	h.ServeHTTP(context.Background(), ctx)

	if ctx.Response.StatusCode() != consts.StatusNoContent {
		t.Fatalf("expected 204, got %d body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	headers := string(ctx.Response.Header.Header())
	if !strings.Contains(headers, "Access-Control-Allow-Origin: http://127.0.0.1:8000") {
		t.Fatalf("cors origin header was not set: %s", headers)
	}
	if !strings.Contains(strings.ToLower(headers), "x-saa-token") || !strings.Contains(strings.ToLower(headers), "authorization") {
		t.Fatalf("cors allowed headers were not set: %s", headers)
	}
}

func TestNewMountsConsoleGroupWithAuthMiddleware(t *testing.T) {
	authenticator := &fakeAuthenticator{
		consoleContext: &contextx.RequestContext{
			AccountID: "acct_console",
			AuthType:  contextx.AuthTypeConsoleToken,
		},
	}
	h := New(Options{
		Authenticator: authenticator,
		RegisterConsoleRoutes: func(group *route.RouterGroup) {
			group.GET("/whoami", func(ctx context.Context, c *app.RequestContext) {
				rc := contextx.MustFrom(ctx)
				c.JSON(consts.StatusOK, map[string]string{
					"account_id": rc.AccountID,
					"auth_type":  string(rc.AuthType),
				})
			})
		},
	})

	ctx := request(consts.MethodGet, "/console/v1/whoami")
	ctx.Request.Header.Set("X-SAA-TOKEN", "Bearer console-token")
	h.ServeHTTP(context.Background(), ctx)

	if ctx.Response.StatusCode() != consts.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	body := string(ctx.Response.Body())
	if !strings.Contains(body, `"account_id":"acct_console"`) || !strings.Contains(body, `"auth_type":"console_token"`) {
		t.Fatalf("unexpected body: %s", body)
	}
	if authenticator.consoleToken != "console-token" {
		t.Fatalf("console token was not passed to authenticator: %#v", authenticator)
	}
}

func TestNewMountsAPIGroupWithAPIKeyMiddleware(t *testing.T) {
	authenticator := &fakeAuthenticator{
		apiContext: &contextx.RequestContext{
			AccountID: "acct_api",
			AuthType:  contextx.AuthTypeAPIKey,
		},
	}
	h := New(Options{
		Authenticator: authenticator,
		RegisterAPIRoutes: func(group *route.RouterGroup) {
			group.GET("/whoami", func(ctx context.Context, c *app.RequestContext) {
				rc := contextx.MustFrom(ctx)
				c.JSON(consts.StatusOK, map[string]string{
					"account_id": rc.AccountID,
					"auth_type":  string(rc.AuthType),
				})
			})
		},
	})

	ctx := request(consts.MethodGet, "/api/v1/whoami")
	ctx.Request.Header.Set("Authorization", "Bearer sk-live")
	h.ServeHTTP(context.Background(), ctx)

	if ctx.Response.StatusCode() != consts.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	body := string(ctx.Response.Body())
	if !strings.Contains(body, `"account_id":"acct_api"`) || !strings.Contains(body, `"auth_type":"api_key"`) {
		t.Fatalf("unexpected body: %s", body)
	}
	if authenticator.apiKey != "sk-live" {
		t.Fatalf("api key was not passed to authenticator: %#v", authenticator)
	}
}

func TestNewReturnsUnauthorizedWhenGroupRouteHasNoCredential(t *testing.T) {
	h := New(Options{
		Authenticator: &fakeAuthenticator{},
		RegisterConsoleRoutes: func(group *route.RouterGroup) {
			group.GET("/whoami", func(context.Context, *app.RequestContext) {
				t.Fatalf("console handler should not run")
			})
		},
	})

	ctx := request(consts.MethodGet, "/console/v1/whoami")
	h.ServeHTTP(context.Background(), ctx)

	if ctx.Response.StatusCode() != consts.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", ctx.Response.StatusCode())
	}
}

func request(method string, uri string) *app.RequestContext {
	ctx := app.NewContext(0)
	ctx.Request.SetRequestURI(uri)
	ctx.Request.Header.SetMethod(method)
	ctx.Request.Header.Set("X-Real-IP", "127.0.0.1")
	return ctx
}

type fakeAuthenticator struct {
	consoleContext *contextx.RequestContext
	apiContext     *contextx.RequestContext
	consoleToken   string
	apiKey         string
}

func (f *fakeAuthenticator) AuthenticateConsoleToken(_ context.Context, token string, _ string) (*contextx.RequestContext, error) {
	f.consoleToken = token
	return f.consoleContext, nil
}

func (f *fakeAuthenticator) AuthenticateAPIKey(_ context.Context, apiKey string, _ string) (*contextx.RequestContext, error) {
	f.apiKey = apiKey
	return f.apiContext, nil
}
