package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/seaskyland/openclaw4j-backend-go/internal/contextx"
	"github.com/seaskyland/openclaw4j-backend-go/internal/dao"
)

func TestServiceAuthenticateConsoleTokenBuildsRequestContext(t *testing.T) {
	now := time.Date(2026, 7, 18, 10, 30, 0, 0, time.UTC)
	source := "console"
	accountDAO := &fakeAccountDAO{
		byID: map[string]*dao.Account{
			"acct_1": {
				AccountID: "acct_1",
				Username:  "alice",
				Type:      "admin",
			},
		},
	}
	workspaceDAO := &fakeWorkspaceDAO{
		byAccountID: map[string]*dao.Workspace{
			"acct_1": {WorkspaceID: "ws_1", AccountID: "acct_1"},
		},
	}
	tokenDAO := &fakeTokenSessionDAO{
		byTokenAndType: map[string]*dao.TokenSession{
			tokenSessionKey("access-token", TokenTypeAccess): {
				AccountID: "acct_1",
				TokenType: TokenTypeAccess,
				Source:    &source,
			},
		},
	}

	service := NewService(ServiceOptions{
		Accounts:      accountDAO,
		Workspaces:    workspaceDAO,
		TokenSessions: tokenDAO,
		Clock:         func() time.Time { return now },
		NewRequestID:  func() string { return "req_1" },
	})

	rc, err := service.AuthenticateConsoleToken(context.Background(), "access-token", "127.0.0.1")
	if err != nil {
		t.Fatalf("AuthenticateConsoleToken returned error: %v", err)
	}

	if rc.RequestID != "req_1" || rc.StartTime != now {
		t.Fatalf("unexpected request metadata: %#v", rc)
	}
	if rc.AccountID != "acct_1" || rc.Username != "alice" || rc.AccountType != "admin" {
		t.Fatalf("unexpected account context: %#v", rc)
	}
	if rc.WorkspaceID != "ws_1" || rc.CallerIP != "127.0.0.1" {
		t.Fatalf("unexpected workspace/caller context: %#v", rc)
	}
	if rc.Source != "console" || rc.AuthType != contextx.AuthTypeConsoleToken {
		t.Fatalf("unexpected auth context: %#v", rc)
	}
	if tokenDAO.lastToken != "access-token" || tokenDAO.lastType != TokenTypeAccess || !tokenDAO.lastNow.Equal(now) {
		t.Fatalf("token lookup did not use expected arguments")
	}
}

func TestServiceAuthenticateConsoleTokenRejectsUnknownToken(t *testing.T) {
	service := NewService(ServiceOptions{
		Accounts:      &fakeAccountDAO{},
		Workspaces:    &fakeWorkspaceDAO{},
		TokenSessions: &fakeTokenSessionDAO{},
		Clock:         fixedClock,
		NewRequestID:  fixedRequestID,
	})

	_, err := service.AuthenticateConsoleToken(context.Background(), "missing-token", "127.0.0.1")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestServiceAuthenticateAPIKeyPrefersHashLookup(t *testing.T) {
	service := NewService(ServiceOptions{
		Accounts: &fakeAccountDAO{byID: map[string]*dao.Account{
			"acct_2": {AccountID: "acct_2", Username: "bob", Type: "normal"},
		}},
		Workspaces: &fakeWorkspaceDAO{byAccountID: map[string]*dao.Workspace{
			"acct_2": {WorkspaceID: "ws_2", AccountID: "acct_2"},
		}},
		APIKeys: &fakeAPIKeyDAO{byHash: map[string]*dao.APIKey{
			HashToken("sk-live"): {AccountID: "acct_2"},
		}},
		LegacyAPIKeyEncryptor: &fakeLegacyEncryptor{encrypted: "legacy-cipher"},
		Clock:                 fixedClock,
		NewRequestID:          fixedRequestID,
	})

	rc, err := service.AuthenticateAPIKey(context.Background(), "sk-live", "10.0.0.8")
	if err != nil {
		t.Fatalf("AuthenticateAPIKey returned error: %v", err)
	}

	if rc.AccountID != "acct_2" || rc.WorkspaceID != "ws_2" {
		t.Fatalf("unexpected API key context: %#v", rc)
	}
	if rc.Source != SourceOpenAPI || rc.AuthType != contextx.AuthTypeAPIKey {
		t.Fatalf("unexpected auth context: %#v", rc)
	}
}

func TestServiceAuthenticateAPIKeyFallsBackToLegacyEncryptedLookup(t *testing.T) {
	apiKeyDAO := &fakeAPIKeyDAO{
		byEncrypted: map[string]*dao.APIKey{
			"legacy-cipher": {AccountID: "acct_3"},
		},
	}
	encryptor := &fakeLegacyEncryptor{encrypted: "legacy-cipher"}
	service := NewService(ServiceOptions{
		Accounts: &fakeAccountDAO{byID: map[string]*dao.Account{
			"acct_3": {AccountID: "acct_3", Username: "carol", Type: "normal"},
		}},
		Workspaces: &fakeWorkspaceDAO{byAccountID: map[string]*dao.Workspace{
			"acct_3": {WorkspaceID: "ws_3", AccountID: "acct_3"},
		}},
		APIKeys:               apiKeyDAO,
		LegacyAPIKeyEncryptor: encryptor,
		Clock:                 fixedClock,
		NewRequestID:          fixedRequestID,
	})

	rc, err := service.AuthenticateAPIKey(context.Background(), "legacy-key", "10.0.0.9")
	if err != nil {
		t.Fatalf("AuthenticateAPIKey returned error: %v", err)
	}

	if rc.AccountID != "acct_3" || rc.WorkspaceID != "ws_3" {
		t.Fatalf("unexpected legacy API key context: %#v", rc)
	}
	if encryptor.lastPlain != "legacy-key" || apiKeyDAO.lastEncrypted != "legacy-cipher" {
		t.Fatalf("legacy encrypted lookup was not used")
	}
}

func TestServiceAuthenticateAPIKeyRejectsUnknownKey(t *testing.T) {
	service := NewService(ServiceOptions{
		Accounts:              &fakeAccountDAO{},
		Workspaces:            &fakeWorkspaceDAO{},
		APIKeys:               &fakeAPIKeyDAO{},
		LegacyAPIKeyEncryptor: &fakeLegacyEncryptor{encrypted: "missing-cipher"},
		Clock:                 fixedClock,
		NewRequestID:          fixedRequestID,
	})

	_, err := service.AuthenticateAPIKey(context.Background(), "missing-key", "10.0.0.8")
	if !errors.Is(err, ErrInvalidAPIKey) {
		t.Fatalf("expected ErrInvalidAPIKey, got %v", err)
	}
}

func fixedClock() time.Time {
	return time.Date(2026, 7, 18, 10, 30, 0, 0, time.UTC)
}

func fixedRequestID() string {
	return "req_fixed"
}

type fakeAccountDAO struct {
	byID               map[string]*dao.Account
	byUsername         map[string]*dao.Account
	lastLoginAccountID string
	lastLoginTime      time.Time
}

func (f *fakeAccountDAO) FindActiveByAccountID(_ context.Context, accountID string) (*dao.Account, error) {
	account, ok := f.byID[accountID]
	if !ok {
		return nil, dao.ErrNotFound
	}
	return account, nil
}

func (f *fakeAccountDAO) FindActiveByUsername(_ context.Context, username string) (*dao.Account, error) {
	if f.byUsername == nil {
		return nil, dao.ErrNotFound
	}
	account, ok := f.byUsername[username]
	if !ok {
		return nil, dao.ErrNotFound
	}
	return account, nil
}

func (f *fakeAccountDAO) UpdateLastLogin(_ context.Context, accountID string, lastLogin time.Time) error {
	f.lastLoginAccountID = accountID
	f.lastLoginTime = lastLogin
	return nil
}

type fakeWorkspaceDAO struct {
	byAccountID map[string]*dao.Workspace
}

func (f *fakeWorkspaceDAO) FindDefaultByAccountID(_ context.Context, accountID string) (*dao.Workspace, error) {
	workspace, ok := f.byAccountID[accountID]
	if !ok {
		return nil, dao.ErrNotFound
	}
	return workspace, nil
}

type fakeAPIKeyDAO struct {
	byHash        map[string]*dao.APIKey
	byEncrypted   map[string]*dao.APIKey
	lastHash      string
	lastEncrypted string
}

func (f *fakeAPIKeyDAO) FindActiveByHash(_ context.Context, keyHash string) (*dao.APIKey, error) {
	f.lastHash = keyHash
	apiKey, ok := f.byHash[keyHash]
	if !ok {
		return nil, dao.ErrNotFound
	}
	return apiKey, nil
}

func (f *fakeAPIKeyDAO) FindActiveByEncryptedKey(_ context.Context, encryptedKey string) (*dao.APIKey, error) {
	f.lastEncrypted = encryptedKey
	apiKey, ok := f.byEncrypted[encryptedKey]
	if !ok {
		return nil, dao.ErrNotFound
	}
	return apiKey, nil
}

func (f *fakeAPIKeyDAO) CountActiveByAccountID(context.Context, string) (int64, error) {
	return 0, nil
}

func (f *fakeAPIKeyDAO) ListActiveByAccountID(context.Context, string, int32, int32) ([]dao.APIKey, error) {
	return nil, nil
}

type fakeTokenSessionDAO struct {
	byTokenAndType   map[string]*dao.TokenSession
	lastToken        string
	lastType         string
	lastNow          time.Time
	creates          []dao.TokenSession
	lastRevokedToken string
	revokeErr        error
}

func (f *fakeTokenSessionDAO) Create(_ context.Context, session dao.TokenSession) error {
	f.creates = append(f.creates, session)
	return nil
}

func (f *fakeTokenSessionDAO) FindActiveByToken(_ context.Context, token string, tokenType string, now time.Time) (*dao.TokenSession, error) {
	f.lastToken = token
	f.lastType = tokenType
	f.lastNow = now
	session, ok := f.byTokenAndType[tokenSessionKey(token, tokenType)]
	if !ok {
		return nil, dao.ErrNotFound
	}
	return session, nil
}

func (f *fakeTokenSessionDAO) RevokeByToken(_ context.Context, token string) error {
	f.lastRevokedToken = token
	return f.revokeErr
}

func (f *fakeTokenSessionDAO) RevokeAccountTokens(context.Context, string, string) error {
	return nil
}

func (f *fakeTokenSessionDAO) DeleteExpired(context.Context, time.Time) error {
	return nil
}

type fakeLegacyEncryptor struct {
	encrypted string
	lastPlain string
}

func (f *fakeLegacyEncryptor) Encrypt(plain string) (string, error) {
	f.lastPlain = plain
	return f.encrypted, nil
}

func tokenSessionKey(tokenHash string, tokenType string) string {
	return tokenHash + ":" + tokenType
}
