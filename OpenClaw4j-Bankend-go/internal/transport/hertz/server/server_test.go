package server

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/seaskyland/openclaw4j-backend-go/internal/application"
	"github.com/seaskyland/openclaw4j-backend-go/internal/auth"
	"github.com/seaskyland/openclaw4j-backend-go/internal/chat"
	"github.com/seaskyland/openclaw4j-backend-go/internal/contextx"
	"github.com/seaskyland/openclaw4j-backend-go/internal/mcpserver"
	"github.com/seaskyland/openclaw4j-backend-go/internal/workflow"
)

type fakeStreamingChatManager struct{}

func (fakeStreamingChatManager) Complete(context.Context, string, chat.Request) (*chat.Response, error) {
	return nil, nil
}

func (fakeStreamingChatManager) CompleteStream(_ context.Context, _ string, _ chat.Request) (<-chan chat.StreamEvent, error) {
	events := make(chan chat.StreamEvent, 3)
	events <- chat.StreamEvent{Response: chat.Response{RequestID: "r", ConversationID: "c", Message: chat.Message{Role: "assistant", Content: "hello"}}}
	events <- chat.StreamEvent{Response: chat.Response{RequestID: "r", ConversationID: "c", Message: chat.Message{Role: "assistant", Content: " world"}}}
	events <- chat.StreamEvent{Response: chat.Response{RequestID: "r", ConversationID: "c"}, Done: true}
	close(events)
	return events, nil
}

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

func TestNewServesFrontendDistWithSPAFallback(t *testing.T) {
	distDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(distDir, "index.html"), []byte("<html>frontend</html>"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(distDir, "umi.js"), []byte("console.log('frontend')"), 0o600); err != nil {
		t.Fatal(err)
	}
	h := New(Options{FrontendDistDir: distDir})

	for _, testCase := range []struct {
		path string
		body string
	}{
		{path: "/", body: "<html>frontend</html>"},
		{path: "/umi.js", body: "console.log('frontend')"},
		{path: "/app/workflow/example", body: "<html>frontend</html>"},
	} {
		ctx := request(consts.MethodGet, testCase.path)
		h.ServeHTTP(context.Background(), ctx)
		if ctx.Response.StatusCode() != consts.StatusOK {
			t.Fatalf("GET %s expected 200, got %d body=%s", testCase.path, ctx.Response.StatusCode(), ctx.Response.Body())
		}
		if string(ctx.Response.Body()) != testCase.body {
			t.Fatalf("GET %s returned unexpected body: %s", testCase.path, ctx.Response.Body())
		}
	}

	missingAsset := request(consts.MethodGet, "/missing.js")
	h.ServeHTTP(context.Background(), missingAsset)
	if missingAsset.Response.StatusCode() != consts.StatusNotFound {
		t.Fatalf("missing static asset expected 404, got %d", missingAsset.Response.StatusCode())
	}

	missingAPI := request(consts.MethodGet, "/console/v1/missing")
	h.ServeHTTP(context.Background(), missingAPI)
	if missingAPI.Response.StatusCode() != consts.StatusNotFound {
		t.Fatalf("missing API route expected 404, got %d", missingAPI.Response.StatusCode())
	}
}

func TestNewHandlesCORSPreflightForUnknownConsoleRoute(t *testing.T) {
	h := New(Options{})

	ctx := request(consts.MethodOptions, "/console/v1/auth/login")
	ctx.Request.Header.Set("Origin", "http://127.0.0.1:8000")
	ctx.Request.Header.Set("Access-Control-Request-Headers", "content-type, x-saa-token, x-client-version, x-request-id")
	h.ServeHTTP(context.Background(), ctx)

	if ctx.Response.StatusCode() != consts.StatusNoContent {
		t.Fatalf("expected 204, got %d body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	headers := string(ctx.Response.Header.Header())
	if !strings.Contains(headers, "Access-Control-Allow-Origin: http://127.0.0.1:8000") {
		t.Fatalf("cors origin header was not set: %s", headers)
	}
	if !strings.Contains(strings.ToLower(headers), "x-saa-token") || !strings.Contains(strings.ToLower(headers), "x-client-version") || !strings.Contains(strings.ToLower(headers), "x-request-id") || !strings.Contains(strings.ToLower(headers), "authorization") {
		t.Fatalf("cors allowed headers were not set: %s", headers)
	}
}

func TestChatCompletionHandlerStreamsEveryModelDelta(t *testing.T) {
	context := contextx.With(context.Background(), &contextx.RequestContext{WorkspaceID: "ws", RequestID: "request"})
	ctx := request(consts.MethodPost, "/apps/chat/completions")
	ctx.Request.SetBodyString(`{"app_id":"app","stream":true,"messages":[{"role":"user","content":"hello"}]}`)
	chatCompletionHandler(fakeStreamingChatManager{})(context, ctx)
	body, err := ctx.Response.BodyE()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ctx.Response.Header.Peek("Content-Type")), "text/event-stream") || strings.Count(string(body), "data: ") != 3 || !strings.Contains(string(body), `"request_id":"r"`) || strings.Contains(string(body), `"response":`) || strings.Contains(string(body), `"done":`) {
		t.Fatalf("unexpected SSE response: headers=%s body=%s", ctx.Response.Header.Header(), body)
	}
}

func TestChatSSEPayloadUsesFrontendErrorShape(t *testing.T) {
	payload := newChatSSEPayload(chat.StreamEvent{
		Response: chat.Response{RequestID: "r", ConversationID: "c"},
		Done:     true,
		Error:    "model stream request failed with status 429",
	})

	if payload.RequestID != "r" || payload.Error == nil || payload.Error.Message != "model stream request failed with status 429" {
		t.Fatalf("unexpected frontend SSE payload: %#v", payload)
	}
}

func TestNewServesGitHubOAuth2ControllerRoutes(t *testing.T) {
	provider := &fakeOAuth2Provider{
		authorizationURL: "https://github.example/login/oauth/authorize?client_id=client_1",
		user:             auth.OAuthUser{Username: "octocat", Name: "The Octocat"},
	}
	issuer := &fakeOAuth2SessionIssuer{response: &auth.TokenResponse{AccessToken: "acc_1", RefreshToken: "ref_1", ExpiresIn: 123456}}
	h := New(Options{OAuth2Provider: provider, SessionIssuer: issuer})

	loginCtx := request(consts.MethodGet, "/oauth2/login/github")
	h.ServeHTTP(context.Background(), loginCtx)
	if loginCtx.Response.StatusCode() != consts.StatusOK || !strings.Contains(string(loginCtx.Response.Body()), `client_id=client_1`) {
		t.Fatalf("GitHub login URL did not preserve Java response shape: status=%d body=%s", loginCtx.Response.StatusCode(), loginCtx.Response.Body())
	}

	callbackCtx := request(consts.MethodGet, "/oauth2/callback/github?code=code_1")
	callbackCtx.Request.Header.Set("User-Agent", "test-agent")
	h.ServeHTTP(context.Background(), callbackCtx)
	if callbackCtx.Response.StatusCode() != consts.StatusFound {
		t.Fatalf("GitHub callback expected 302, got %d body=%s", callbackCtx.Response.StatusCode(), callbackCtx.Response.Body())
	}
	location := string(callbackCtx.Response.Header.Peek("Location"))
	if location != "/?access_token=acc_1&refresh_token=ref_1&expires_in=123456" {
		t.Fatalf("unexpected GitHub callback redirect: %q", location)
	}
	if provider.lastCode != "code_1" || issuer.user.Username != "octocat" || issuer.callerIP != "127.0.0.1" || issuer.userAgent != "test-agent" {
		t.Fatalf("OAuth2 callback dependencies did not receive the expected input: %#v %#v", provider, issuer)
	}
}

func TestNewRejectsGitHubOAuth2WhenUnavailable(t *testing.T) {
	h := New(Options{})
	ctx := request(consts.MethodGet, "/oauth2/login/github")
	h.ServeHTTP(context.Background(), ctx)
	if ctx.Response.StatusCode() != consts.StatusServiceUnavailable || !strings.Contains(string(ctx.Response.Body()), `"code":"OAUTH2_UNAVAILABLE"`) {
		t.Fatalf("unconfigured GitHub OAuth2 expected 503, got status=%d body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
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

func TestNewServesWorkflowDebugControllerRoutes(t *testing.T) {
	manager := workflow.NewService(nil, nil, nil)
	h := New(Options{
		Authenticator: &fakeAuthenticator{consoleContext: &contextx.RequestContext{
			AccountID: "acct_saa", WorkspaceID: "ws_saa", RequestID: "request_saa", AuthType: contextx.AuthTypeConsoleToken,
		}},
		WorkflowManager: manager,
	})

	runCtx := request(consts.MethodPost, "/console/v1/apps/workflow/debug/part-graph/run-task")
	runCtx.Request.Header.Set("X-SAA-TOKEN", "Bearer console-token")
	runCtx.Request.SetBodyString(`{"app_id":"app","nodes":[{"id":"start","type":"Start","config":{}},{"id":"end","type":"End","config":{"node_param":{"text_template":"ok"}}}],"edges":[{"source":"start","target":"end"}]}`)
	h.ServeHTTP(context.Background(), runCtx)
	if runCtx.Response.StatusCode() != consts.StatusOK || !strings.Contains(string(runCtx.Response.Body()), `"task_id"`) {
		t.Fatalf("run workflow debug task expected 200, got status=%d body=%s", runCtx.Response.StatusCode(), runCtx.Response.Body())
	}
}

func TestNewStreamsWorkflowNodeEventsAsTheyAreProduced(t *testing.T) {
	manager := workflow.NewService(streamingWorkflowApplicationReader{}, nil, nil)
	h := New(Options{
		Authenticator: &fakeAuthenticator{consoleContext: &contextx.RequestContext{
			AccountID: "acct_saa", WorkspaceID: "ws_saa", RequestID: "request_saa", AuthType: contextx.AuthTypeConsoleToken,
		}},
		WorkflowManager: manager,
	})

	ctx := request(consts.MethodPost, "/console/v1/apps/workflow/app_stream/run_stream")
	ctx.Request.Header.Set("X-SAA-TOKEN", "Bearer console-token")
	ctx.Request.SetBodyString(`{"conversation_id":"conversation_stream","input_params":[]}`)
	h.ServeHTTP(context.Background(), ctx)
	body, err := ctx.Response.BodyE()
	if err != nil {
		t.Fatal(err)
	}
	response := string(body)
	if !strings.Contains(string(ctx.Response.Header.Peek("Content-Type")), "text/event-stream") || strings.Count(response, "data: ") != 5 {
		t.Fatalf("workflow SSE must contain per-node events and terminal event: headers=%s body=%s", ctx.Response.Header.Header(), response)
	}
	startExecuting := strings.Index(response, `"node_id":"start","node_name":"Start","node_type":"Start","node_status":"executing"`)
	startSuccess := strings.Index(response, `"node_id":"start","node_name":"Start","node_type":"Start","node_status":"success"`)
	finished := strings.Index(response, `"event":"Finished"`)
	if startExecuting < 0 || startSuccess < startExecuting || finished < startSuccess {
		t.Fatalf("workflow SSE event ordering is invalid: %s", response)
	}
}

func TestNewStreamsPublicWorkflowCompletionEvents(t *testing.T) {
	manager := workflow.NewService(streamingWorkflowApplicationReader{}, nil, nil)
	h := New(Options{
		Authenticator: &fakeAuthenticator{apiContext: &contextx.RequestContext{
			AccountID: "acct_api", WorkspaceID: "ws_api", RequestID: "request_api", AuthType: contextx.AuthTypeAPIKey,
		}},
		WorkflowManager: manager,
	})

	ctx := request(consts.MethodPost, "/api/v1/apps/workflow/completions")
	ctx.Request.Header.Set("Authorization", "Bearer sk-live")
	ctx.Request.SetBodyString(`{"app_id":"app_stream","stream":true,"messages":[{"role":"user","content":"hello"}]}`)
	h.ServeHTTP(context.Background(), ctx)
	body, err := ctx.Response.BodyE()
	if err != nil {
		t.Fatal(err)
	}
	response := string(body)
	if !strings.Contains(string(ctx.Response.Header.Peek("Content-Type")), "text/event-stream") || strings.Count(response, "data: ") != 5 || !strings.Contains(response, `"event":"Finished"`) {
		t.Fatalf("public workflow stream did not emit node events: headers=%s body=%s", ctx.Response.Header.Header(), response)
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

func TestNewServesApiExampleControllerRoutes(t *testing.T) {
	h := New(Options{})

	getCtx := request(consts.MethodGet, "/test/api/example/getOrder")
	h.ServeHTTP(context.Background(), getCtx)
	if getCtx.Response.StatusCode() != consts.StatusOK {
		t.Fatalf("get example expected 200, got %d body=%s", getCtx.Response.StatusCode(), getCtx.Response.Body())
	}
	if !strings.Contains(string(getCtx.Response.Body()), `"orderId":"100001"`) || !strings.Contains(string(getCtx.Response.Body()), `"company":"顺丰快递"`) {
		t.Fatalf("get example did not preserve Java response shape: %s", getCtx.Response.Body())
	}
	if strings.Contains(string(getCtx.Response.Body()), `"items"`) {
		t.Fatalf("GET example must not include the POST-only items field: %s", getCtx.Response.Body())
	}

	postCtx := request(consts.MethodPost, "/test/api/example/getOrder")
	postCtx.Request.SetBodyString(`{"orderId":"order_123"}`)
	h.ServeHTTP(context.Background(), postCtx)
	if postCtx.Response.StatusCode() != consts.StatusOK {
		t.Fatalf("post example expected 200, got %d body=%s", postCtx.Response.StatusCode(), postCtx.Response.Body())
	}
	if !strings.Contains(string(postCtx.Response.Body()), `"orderId":"order_123"`) || !strings.Contains(string(postCtx.Response.Body()), `"itemId":"2002"`) {
		t.Fatalf("post example did not preserve Java response shape: %s", postCtx.Response.Body())
	}

	pathCtx := request(consts.MethodPost, "/test/api/example/getOrder/order_456")
	pathCtx.Request.SetBodyString(`{"ignored":true}`)
	h.ServeHTTP(context.Background(), pathCtx)
	if pathCtx.Response.StatusCode() != consts.StatusOK || !strings.Contains(string(pathCtx.Response.Body()), `"orderId":"order_456"`) {
		t.Fatalf("path example expected Java-compatible response, got status=%d body=%s", pathCtx.Response.StatusCode(), pathCtx.Response.Body())
	}

	missingOrderIDCtx := request(consts.MethodPost, "/test/api/example/getOrder")
	missingOrderIDCtx.Request.SetBodyString(`{}`)
	h.ServeHTTP(context.Background(), missingOrderIDCtx)
	if missingOrderIDCtx.Response.StatusCode() != consts.StatusBadRequest || !strings.Contains(string(missingOrderIDCtx.Response.Body()), `"code":"MISSING_PARAMS"`) {
		t.Fatalf("missing orderId expected 400 MISSING_PARAMS, got status=%d body=%s", missingOrderIDCtx.Response.StatusCode(), missingOrderIDCtx.Response.Body())
	}
}

func TestNewServesMcpServerControllerRoutes(t *testing.T) {
	manager := &fakeMcpServerManager{}
	h := New(Options{
		Authenticator: &fakeAuthenticator{consoleContext: &contextx.RequestContext{
			AccountID: "acct_saa", WorkspaceID: "ws_saa", AuthType: contextx.AuthTypeConsoleToken,
		}},
		McpServerManager: manager,
	})

	createCtx := request(consts.MethodPost, "/console/v1/mcp-servers")
	createCtx.Request.Header.Set("X-SAA-TOKEN", "Bearer console-token")
	createCtx.Request.SetBodyString(`{"name":"Weather","deploy_config":"{\"url\":\"https://example.test/sse\"}"}`)
	h.ServeHTTP(context.Background(), createCtx)
	if createCtx.Response.StatusCode() != consts.StatusOK || !strings.Contains(string(createCtx.Response.Body()), `"mcp_created"`) {
		t.Fatalf("create MCP expected Java-compatible response, got status=%d body=%s", createCtx.Response.StatusCode(), createCtx.Response.Body())
	}
	if manager.createWorkspaceID != "ws_saa" || manager.createAccountID != "acct_saa" {
		t.Fatalf("create MCP did not receive request context: %#v", manager)
	}

	getCtx := request(consts.MethodGet, "/console/v1/mcp-servers/mcp_created?need_tools=true")
	getCtx.Request.Header.Set("X-SAA-TOKEN", "Bearer console-token")
	h.ServeHTTP(context.Background(), getCtx)
	if getCtx.Response.StatusCode() != consts.StatusOK || !strings.Contains(string(getCtx.Response.Body()), `"server_code":"mcp_created"`) {
		t.Fatalf("get MCP expected Java-compatible response, got status=%d body=%s", getCtx.Response.StatusCode(), getCtx.Response.Body())
	}
	if !manager.needTools {
		t.Fatal("need_tools query parameter was not forwarded")
	}

	queryCtx := request(consts.MethodPost, "/console/v1/mcp-servers/query-by-codes")
	queryCtx.Request.Header.Set("X-SAA-TOKEN", "Bearer console-token")
	queryCtx.Request.SetBodyString(`{"codes":["mcp_created"]}`)
	h.ServeHTTP(context.Background(), queryCtx)
	if queryCtx.Response.StatusCode() != consts.StatusOK || !strings.Contains(string(queryCtx.Response.Body()), `"server_code":"mcp_created"`) {
		t.Fatalf("query MCP expected Java-compatible response, got status=%d body=%s", queryCtx.Response.StatusCode(), queryCtx.Response.Body())
	}

	debugCtx := request(consts.MethodPost, "/console/v1/mcp-servers/debug-tools")
	debugCtx.Request.Header.Set("X-SAA-TOKEN", "Bearer console-token")
	debugCtx.Request.SetBodyString(`{"server_code":"mcp_created","tool_name":"weather","tool_params":{"city":"Shanghai"}}`)
	h.ServeHTTP(context.Background(), debugCtx)
	if debugCtx.Response.StatusCode() != consts.StatusOK || !strings.Contains(string(debugCtx.Response.Body()), `"is_error":false`) {
		t.Fatalf("debug MCP expected Java-compatible response, got status=%d body=%s", debugCtx.Response.StatusCode(), debugCtx.Response.Body())
	}

	invalidNeedToolsCtx := request(consts.MethodGet, "/console/v1/mcp-servers/mcp_created")
	invalidNeedToolsCtx.Request.Header.Set("X-SAA-TOKEN", "Bearer console-token")
	h.ServeHTTP(context.Background(), invalidNeedToolsCtx)
	if invalidNeedToolsCtx.Response.StatusCode() != consts.StatusBadRequest {
		t.Fatalf("need_tools must remain required, got status=%d body=%s", invalidNeedToolsCtx.Response.StatusCode(), invalidNeedToolsCtx.Response.Body())
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

type streamingWorkflowApplicationReader struct{}

func (streamingWorkflowApplicationReader) GetVersion(context.Context, string, string, string) (*application.Version, error) {
	return &application.Version{Config: `{
      "nodes": [
        {"id":"start","name":"Start","type":"Start","config":{}},
        {"id":"end","name":"End","type":"End","config":{"node_param":{"text_template":"done"}}}
      ],
      "edges": [{"source":"start","target":"end"}]
    }`}, nil
}

type fakeOAuth2Provider struct {
	authorizationURL string
	user             auth.OAuthUser
	lastCode         string
}

func (f *fakeOAuth2Provider) AuthorizationURL() (string, error) { return f.authorizationURL, nil }

func (f *fakeOAuth2Provider) Authenticate(_ context.Context, code string) (auth.OAuthUser, error) {
	f.lastCode = code
	return f.user, nil
}

type fakeOAuth2SessionIssuer struct {
	response  *auth.TokenResponse
	user      auth.OAuthUser
	callerIP  string
	userAgent string
}

func (f *fakeOAuth2SessionIssuer) Login(context.Context, string, string, string, string) (*auth.TokenResponse, error) {
	return f.response, nil
}

func (f *fakeOAuth2SessionIssuer) RefreshToken(context.Context, string, string, string) (*auth.TokenResponse, error) {
	return f.response, nil
}

func (f *fakeOAuth2SessionIssuer) Logout(context.Context, string) error { return nil }

func (f *fakeOAuth2SessionIssuer) LoginOAuth(_ context.Context, user auth.OAuthUser, callerIP string, userAgent string) (*auth.TokenResponse, error) {
	f.user = user
	f.callerIP = callerIP
	f.userAgent = userAgent
	return f.response, nil
}

type fakeMcpServerManager struct {
	createWorkspaceID string
	createAccountID   string
	needTools         bool
}

func (f *fakeMcpServerManager) Create(_ context.Context, workspaceID, accountID string, _ mcpserver.Input) (string, error) {
	f.createWorkspaceID = workspaceID
	f.createAccountID = accountID
	return "mcp_created", nil
}

func (f *fakeMcpServerManager) Update(context.Context, string, mcpserver.Input) error { return nil }

func (f *fakeMcpServerManager) Get(_ context.Context, _ string, serverCode string, needTools bool) (*mcpserver.Server, error) {
	f.needTools = needTools
	return &mcpserver.Server{ServerCode: serverCode, Name: "Weather", DeployConfig: "{}", Type: "sse", Status: 1}, nil
}

func (f *fakeMcpServerManager) List(context.Context, string, string, bool, int64, int64) (*mcpserver.Page, error) {
	return &mcpserver.Page{Current: 1, Size: 10, Records: []mcpserver.Server{}}, nil
}

func (f *fakeMcpServerManager) ListByCodes(_ context.Context, _ string, codes []string, needTools bool) ([]mcpserver.Server, error) {
	f.needTools = needTools
	result := make([]mcpserver.Server, 0, len(codes))
	for _, code := range codes {
		result = append(result, mcpserver.Server{ServerCode: code, Name: "Weather", DeployConfig: "{}", Type: "sse", Status: 1})
	}
	return result, nil
}

func (f *fakeMcpServerManager) Delete(context.Context, string, string) error { return nil }

func (f *fakeMcpServerManager) CallTool(context.Context, string, mcpserver.ToolCallInput) (*mcpserver.ToolCallResult, error) {
	return &mcpserver.ToolCallResult{IsError: false, Content: []map[string]any{{"type": "text", "text": "ok"}}}, nil
}

func (f *fakeAuthenticator) AuthenticateConsoleToken(_ context.Context, token string, _ string) (*contextx.RequestContext, error) {
	f.consoleToken = token
	return f.consoleContext, nil
}

func (f *fakeAuthenticator) AuthenticateAPIKey(_ context.Context, apiKey string, _ string) (*contextx.RequestContext, error) {
	f.apiKey = apiKey
	return f.apiContext, nil
}
