package dao

import (
	"context"
	"time"
)

type Account struct {
	ID           int64
	AccountID    string
	Username     string
	Email        *string
	Mobile       *string
	Password     string
	Nickname     *string
	Icon         *string
	Type         string
	Status       int16
	GmtCreate    time.Time
	GmtModified  time.Time
	GmtLastLogin *time.Time
	Creator      string
	Modifier     string
	TenantID     int64
}

type Workspace struct {
	ID          int64
	WorkspaceID string
	AccountID   string
	Status      int16
	Name        string
	Description *string
	Config      *string
	GmtCreate   time.Time
	GmtModified time.Time
	Creator     string
	Modifier    string
	TenantID    int64
}

type APIKey struct {
	ID          int64
	AccountID   string
	APIKey      string
	Status      int16
	Description *string
	GmtCreate   time.Time
	GmtModified time.Time
	Creator     string
	Modifier    string
	TenantID    int64
}

type TokenSession struct {
	ID          int64
	TokenID     string
	AccountID   string
	TokenType   string
	TokenHash   string
	ExpiresAt   time.Time
	Revoked     int16
	Source      *string
	CallerIP    *string
	UserAgent   *string
	GmtCreate   time.Time
	GmtModified time.Time
	TenantID    int64
}

type AccountDAO interface {
	FindActiveByAccountID(ctx context.Context, accountID string) (*Account, error)
	FindActiveByUsername(ctx context.Context, username string) (*Account, error)
	UpdateLastLogin(ctx context.Context, accountID string, lastLogin time.Time) error
}

type WorkspaceDAO interface {
	FindDefaultByAccountID(ctx context.Context, accountID string) (*Workspace, error)
}

type APIKeyDAO interface {
	FindActiveByEncryptedKey(ctx context.Context, encryptedKey string) (*APIKey, error)
	FindActiveByHash(ctx context.Context, keyHash string) (*APIKey, error)
	CountActiveByAccountID(ctx context.Context, accountID string) (int64, error)
	ListActiveByAccountID(ctx context.Context, accountID string, limit int32, offset int32) ([]APIKey, error)
}

type TokenSessionDAO interface {
	Create(ctx context.Context, session TokenSession) error
	FindActiveByToken(ctx context.Context, token string, tokenType string, now time.Time) (*TokenSession, error)
	RevokeByToken(ctx context.Context, token string) error
	RevokeAccountTokens(ctx context.Context, accountID string, tokenType string) error
	DeleteExpired(ctx context.Context, before time.Time) error
}
