package server

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/seaskyland/openclaw4j-backend-go/internal/auth"
	"github.com/seaskyland/openclaw4j-backend-go/internal/contextx"
)

func TestNewServesPublicCompatRoutes(t *testing.T) {
	issuer := &fakeSessionIssuer{
		loginResponse: &auth.TokenResponse{
			AccessToken:  "access-token",
			RefreshToken: "refresh-token",
			ExpiresIn:    1234567890,
		},
		refreshResponse: &auth.TokenResponse{
			AccessToken:  "access-token-2",
			RefreshToken: "refresh-token-2",
			ExpiresIn:    1234567999,
		},
	}
	h := New(Options{
		Authenticator: &fakeAuthenticator{
			consoleContext: &contextx.RequestContext{
				AccountID:   "acct_saa",
				Username:    "saa",
				AccountType: "admin",
				AuthType:    contextx.AuthTypeConsoleToken,
			},
		},
		SessionIssuer: issuer,
		RegisterConsoleRoutes: func(group *route.RouterGroup) {
			group.GET("/whoami", func(ctx context.Context, c *app.RequestContext) {
				rc := contextx.MustFrom(ctx)
				c.JSON(consts.StatusOK, map[string]string{
					"account_id": rc.AccountID,
					"username":   rc.Username,
				})
			})
		},
	})

	loginCtx := request(consts.MethodPost, "/console/v1/auth/login")
	loginCtx.Request.SetBodyString(`{"username":"saa","password":"123456"}`)
	h.ServeHTTP(context.Background(), loginCtx)
	if loginCtx.Response.StatusCode() != consts.StatusOK {
		t.Fatalf("login expected 200, got %d body=%s", loginCtx.Response.StatusCode(), loginCtx.Response.Body())
	}
	if issuer.loginUsername != "saa" || issuer.loginPassword != "123456" {
		t.Fatalf("login credentials were not forwarded: %#v", issuer)
	}
	if !strings.Contains(string(loginCtx.Response.Body()), `"access_token":"access-token"`) {
		t.Fatalf("login response did not include tokens: %s", loginCtx.Response.Body())
	}

	refreshCtx := request(consts.MethodPost, "/console/v1/auth/refresh-token")
	refreshCtx.Request.SetBodyString(`{"refresh_token":"refresh-token"}`)
	h.ServeHTTP(context.Background(), refreshCtx)
	if refreshCtx.Response.StatusCode() != consts.StatusOK {
		t.Fatalf("refresh expected 200, got %d body=%s", refreshCtx.Response.StatusCode(), refreshCtx.Response.Body())
	}
	if issuer.refreshToken != "refresh-token" {
		t.Fatalf("refresh token was not forwarded: %#v", issuer)
	}

	logoutCtx := request(consts.MethodPost, "/console/v1/auth/logout")
	logoutCtx.Request.Header.Set("Authorization", "Bearer access-token")
	h.ServeHTTP(context.Background(), logoutCtx)
	if logoutCtx.Response.StatusCode() != consts.StatusOK {
		t.Fatalf("logout expected 200, got %d body=%s", logoutCtx.Response.StatusCode(), logoutCtx.Response.Body())
	}
	if issuer.logoutToken != "access-token" {
		t.Fatalf("logout token was not forwarded: %#v", issuer)
	}

	logoutWithoutTokenCtx := request(consts.MethodPost, "/console/v1/auth/logout")
	h.ServeHTTP(context.Background(), logoutWithoutTokenCtx)
	if logoutWithoutTokenCtx.Response.StatusCode() != consts.StatusOK {
		t.Fatalf("logout without token expected 200, got %d body=%s", logoutWithoutTokenCtx.Response.StatusCode(), logoutWithoutTokenCtx.Response.Body())
	}
	if issuer.logoutToken != "" {
		t.Fatalf("logout without token should forward an empty token: %#v", issuer)
	}

	globalCtx := request(consts.MethodGet, "/console/v1/system/global-config")
	h.ServeHTTP(context.Background(), globalCtx)
	if globalCtx.Response.StatusCode() != consts.StatusOK {
		t.Fatalf("global config expected 200, got %d body=%s", globalCtx.Response.StatusCode(), globalCtx.Response.Body())
	}
	if !strings.Contains(string(globalCtx.Response.Body()), `"login_method":"preset_account"`) {
		t.Fatalf("global config did not include login method: %s", globalCtx.Response.Body())
	}

	apiCtx := request(consts.MethodGet, "/api/models")
	h.ServeHTTP(context.Background(), apiCtx)
	if apiCtx.Response.StatusCode() != consts.StatusOK {
		t.Fatalf("api models expected 200, got %d body=%s", apiCtx.Response.StatusCode(), apiCtx.Response.Body())
	}

	profileCtx := request(consts.MethodGet, "/console/v1/accounts/profile")
	profileCtx.Request.Header.Set("X-SAA-TOKEN", "Bearer access-token")
	h.ServeHTTP(context.Background(), profileCtx)
	if profileCtx.Response.StatusCode() != consts.StatusOK {
		t.Fatalf("profile expected 200, got %d body=%s", profileCtx.Response.StatusCode(), profileCtx.Response.Body())
	}
	if !strings.Contains(string(profileCtx.Response.Body()), `"username":"saa"`) {
		t.Fatalf("profile response did not include account info: %s", profileCtx.Response.Body())
	}

	selectorCtx := request(consts.MethodGet, "/console/v1/models/llm/selector")
	selectorCtx.Request.Header.Set("X-SAA-TOKEN", "Bearer access-token")
	h.ServeHTTP(context.Background(), selectorCtx)
	if selectorCtx.Response.StatusCode() != consts.StatusOK {
		t.Fatalf("selector expected 200, got %d body=%s", selectorCtx.Response.StatusCode(), selectorCtx.Response.Body())
	}
	var selectorResponse map[string]any
	if err := json.Unmarshal(selectorCtx.Response.Body(), &selectorResponse); err != nil {
		t.Fatalf("selector response was not json: %v", err)
	}
}

type fakeSessionIssuer struct {
	loginUsername    string
	loginPassword    string
	loginCallerIP    string
	loginUserAgent   string
	refreshToken     string
	refreshCallerIP  string
	refreshUserAgent string
	logoutToken      string
	loginResponse    *auth.TokenResponse
	refreshResponse  *auth.TokenResponse
}

func (f *fakeSessionIssuer) Login(_ context.Context, username, password, callerIP, userAgent string) (*auth.TokenResponse, error) {
	f.loginUsername = username
	f.loginPassword = password
	f.loginCallerIP = callerIP
	f.loginUserAgent = userAgent
	return f.loginResponse, nil
}

func (f *fakeSessionIssuer) RefreshToken(_ context.Context, refreshToken, callerIP, userAgent string) (*auth.TokenResponse, error) {
	f.refreshToken = refreshToken
	f.refreshCallerIP = callerIP
	f.refreshUserAgent = userAgent
	return f.refreshResponse, nil
}

func (f *fakeSessionIssuer) Logout(_ context.Context, accessToken string) error {
	f.logoutToken = accessToken
	return nil
}
