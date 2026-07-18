package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/argon2"

	"github.com/seaskyland/openclaw4j-backend-go/internal/dao"
)

func TestServiceLoginIssuesConsoleTokens(t *testing.T) {
	now := time.Date(2026, 7, 18, 10, 30, 0, 0, time.UTC)
	password := "123456"
	accountDAO := &fakeAccountDAO{
		byUsername: map[string]*dao.Account{
			"saa": {
				AccountID: "acct_saa",
				Username:  "saa",
				Password:  argon2HashForTest(t, password),
				Type:      "admin",
				TenantID:  7,
			},
		},
	}
	workspaceDAO := &fakeWorkspaceDAO{
		byAccountID: map[string]*dao.Workspace{
			"acct_saa": {WorkspaceID: "ws_saa", AccountID: "acct_saa"},
		},
	}
	tokenDAO := &fakeTokenSessionDAO{}
	service := NewService(ServiceOptions{
		Accounts:      accountDAO,
		Workspaces:    workspaceDAO,
		TokenSessions: tokenDAO,
		Clock:         func() time.Time { return now },
		NewRequestID:  func() string { return "req_login" },
	})

	response, err := service.Login(context.Background(), "saa", password, "127.0.0.1", "browser")
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if response.AccessToken == "" || response.RefreshToken == "" {
		t.Fatalf("expected tokens to be issued: %#v", response)
	}
	if response.AccessToken == response.RefreshToken {
		t.Fatalf("access and refresh token should differ: %#v", response)
	}
	if response.ExpiresIn <= now.Unix() {
		t.Fatalf("expected future expiry, got %#v", response)
	}
	if accountDAO.lastLoginAccountID != "acct_saa" || !accountDAO.lastLoginTime.Equal(now) {
		t.Fatalf("last login was not updated: %#v", accountDAO)
	}
	if len(tokenDAO.creates) != 2 {
		t.Fatalf("expected access and refresh sessions to be created, got %d", len(tokenDAO.creates))
	}
	if tokenDAO.creates[0].TokenType != TokenTypeAccess || tokenDAO.creates[1].TokenType != TokenTypeRefresh {
		t.Fatalf("unexpected token session types: %#v", tokenDAO.creates)
	}
	if tokenDAO.creates[0].AccountID != "acct_saa" || tokenDAO.creates[0].TenantID != 7 {
		t.Fatalf("unexpected access session payload: %#v", tokenDAO.creates[0])
	}
	if tokenDAO.creates[0].TokenHash != HashToken(response.AccessToken) || tokenDAO.creates[1].TokenHash != HashToken(response.RefreshToken) {
		t.Fatalf("token hashes did not match issued tokens: %#v", tokenDAO.creates)
	}
}

func TestServiceRefreshTokenRotatesConsoleTokens(t *testing.T) {
	now := time.Date(2026, 7, 18, 10, 30, 0, 0, time.UTC)
	refreshToken := "refresh-token"
	tokenDAO := &fakeTokenSessionDAO{
		byTokenAndType: map[string]*dao.TokenSession{
			tokenSessionKey(refreshToken, "refresh"): {
				AccountID: "acct_saa",
				TokenType: "refresh",
			},
		},
	}
	service := NewService(ServiceOptions{
		Accounts: &fakeAccountDAO{
			byID: map[string]*dao.Account{
				"acct_saa": {
					AccountID: "acct_saa",
					Username:  "saa",
					Password:  "unused",
					Type:      "admin",
					TenantID:  7,
				},
			},
		},
		Workspaces: &fakeWorkspaceDAO{
			byAccountID: map[string]*dao.Workspace{
				"acct_saa": {WorkspaceID: "ws_saa", AccountID: "acct_saa"},
			},
		},
		TokenSessions: tokenDAO,
		Clock:         func() time.Time { return now },
		NewRequestID:  func() string { return "req_refresh" },
	})

	response, err := service.RefreshToken(context.Background(), refreshToken, "127.0.0.1", "browser")
	if err != nil {
		t.Fatalf("RefreshToken returned error: %v", err)
	}
	if response.AccessToken == "" || response.RefreshToken == "" {
		t.Fatalf("expected new tokens to be issued: %#v", response)
	}
	if tokenDAO.lastRevokedToken != refreshToken {
		t.Fatalf("refresh token was not revoked: %#v", tokenDAO)
	}
	if len(tokenDAO.creates) != 2 {
		t.Fatalf("expected rotated access and refresh sessions to be created, got %d", len(tokenDAO.creates))
	}
	if !strings.HasPrefix(response.AccessToken, "acc_") || !strings.HasPrefix(response.RefreshToken, "ref_") {
		t.Fatalf("unexpected token prefixes: %#v", response)
	}
}

func TestServiceLogoutRevokesAccessTokenAndAllowsMissingToken(t *testing.T) {
	tokenDAO := &fakeTokenSessionDAO{}
	service := NewService(ServiceOptions{TokenSessions: tokenDAO})

	if err := service.Logout(context.Background(), "  acc_access-token  "); err != nil {
		t.Fatalf("Logout returned error: %v", err)
	}
	if tokenDAO.lastRevokedToken != "acc_access-token" {
		t.Fatalf("access token was not revoked: %#v", tokenDAO)
	}

	if err := service.Logout(context.Background(), "  "); err != nil {
		t.Fatalf("Logout without a token returned error: %v", err)
	}
	if tokenDAO.lastRevokedToken != "acc_access-token" {
		t.Fatalf("Logout without a token should be a no-op: %#v", tokenDAO)
	}
}

func TestServiceLogoutPropagatesSessionStoreFailure(t *testing.T) {
	revokeErr := errors.New("redis unavailable")
	service := NewService(ServiceOptions{TokenSessions: &fakeTokenSessionDAO{revokeErr: revokeErr}})

	err := service.Logout(context.Background(), "acc_access-token")
	if !errors.Is(err, revokeErr) {
		t.Fatalf("expected session store failure, got %v", err)
	}
}

func argon2HashForTest(t *testing.T, password string) string {
	t.Helper()

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		t.Fatalf("rand.Read failed: %v", err)
	}

	hash := argon2.IDKey([]byte(password), salt, 2, 66536, 1, 32)
	return fmt.Sprintf(
		"$argon2id$v=19$m=66536,t=2,p=1$%s$%s",
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)
}
