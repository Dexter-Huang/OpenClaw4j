package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/seaskyland/openclaw4j-backend-go/internal/contextx"
)

func TestNewRequiresDatabaseWhenAuthenticatorIsNotProvided(t *testing.T) {
	_, err := New(Options{})
	if !errors.Is(err, ErrMissingDatabase) {
		t.Fatalf("expected ErrMissingDatabase, got %v", err)
	}
}

func TestNewUsesProvidedAuthenticatorForRouteGroups(t *testing.T) {
	authenticator := &fakeAuthenticator{
		consoleContext: &contextx.RequestContext{
			AccountID: "acct_1",
			AuthType:  contextx.AuthTypeConsoleToken,
		},
	}
	appInstance, err := New(Options{
		Authenticator: authenticator,
		RegisterConsoleRoutes: func(group *route.RouterGroup) {
			group.GET("/whoami", func(ctx context.Context, c *app.RequestContext) {
				rc := contextx.MustFrom(ctx)
				c.JSON(consts.StatusOK, map[string]string{"account_id": rc.AccountID})
			})
		},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	ctx := request(consts.MethodGet, "/console/v1/whoami")
	ctx.Request.Header.Set("X-SAA-TOKEN", "Bearer console-token")
	appInstance.HTTP.ServeHTTP(context.Background(), ctx)

	if ctx.Response.StatusCode() != consts.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if !strings.Contains(string(ctx.Response.Body()), `"account_id":"acct_1"`) {
		t.Fatalf("unexpected body: %s", ctx.Response.Body())
	}
	if appInstance.Authenticator != authenticator {
		t.Fatalf("provided authenticator was not retained")
	}
}

func TestNewBuildsAuthenticatorFromDatabase(t *testing.T) {
	dbtx := &fakeDBTX{}
	appInstance, err := New(Options{DB: dbtx})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if appInstance.Authenticator == nil {
		t.Fatalf("authenticator was not built from database")
	}
	if len(dbtx.execStatements) != 0 {
		t.Fatalf("startup should not execute schema migration, got %d statements", len(dbtx.execStatements))
	}

	ctx := request(consts.MethodGet, "/healthz")
	appInstance.HTTP.ServeHTTP(context.Background(), ctx)
	if ctx.Response.StatusCode() != consts.StatusOK {
		t.Fatalf("expected health route to be registered, got %d", ctx.Response.StatusCode())
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
}

func (f *fakeAuthenticator) AuthenticateConsoleToken(context.Context, string, string) (*contextx.RequestContext, error) {
	return f.consoleContext, nil
}

func (f *fakeAuthenticator) AuthenticateAPIKey(context.Context, string, string) (*contextx.RequestContext, error) {
	return f.apiContext, nil
}

type fakeDBTX struct {
	execStatements []string
}

func (f *fakeDBTX) Exec(_ context.Context, query string, _ ...interface{}) (pgconn.CommandTag, error) {
	f.execStatements = append(f.execStatements, query)
	return pgconn.CommandTag{}, nil
}

func (f *fakeDBTX) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	panic("fakeDBTX Query should not be called")
}

func (f *fakeDBTX) QueryRow(context.Context, string, ...interface{}) pgx.Row {
	panic("fakeDBTX QueryRow should not be called")
}
