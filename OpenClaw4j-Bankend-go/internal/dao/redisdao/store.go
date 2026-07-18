package redisdao

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/seaskyland/openclaw4j-backend-go/internal/dao"
)

const (
	tokenKeyAccessPrefix  = "access_token:"
	tokenKeyRefreshPrefix = "refresh_token:"
	scanBatchSize         = 128
)

type RedisClient interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Scan(ctx context.Context, cursor uint64, match string, count int64) *redis.ScanCmd
}

type TokenSessionDAO struct {
	client RedisClient
}

func NewTokenSessionDAO(client RedisClient) *TokenSessionDAO {
	return &TokenSessionDAO{client: client}
}

func (d *TokenSessionDAO) Create(ctx context.Context, session dao.TokenSession) error {
	token := session.TokenID
	if token == "" {
		token = session.TokenHash
	}
	if token == "" {
		return errors.New("token is required")
	}
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return errors.New("token session already expired")
	}

	payload, err := json.Marshal(redisTokenSession{
		AccountID: session.AccountID,
		TokenType: session.TokenType,
		TokenHash: session.TokenHash,
		ExpiresAt: session.ExpiresAt,
		Revoked:   session.Revoked,
		Source:    session.Source,
		CallerIP:  session.CallerIP,
		UserAgent: session.UserAgent,
		TenantID:  session.TenantID,
	})
	if err != nil {
		return fmt.Errorf("marshal token session: %w", err)
	}

	if err := d.client.Set(ctx, tokenKey(session.TokenType, token), payload, ttl).Err(); err != nil {
		return fmt.Errorf("store token session: %w", err)
	}
	return nil
}

func (d *TokenSessionDAO) FindActiveByToken(ctx context.Context, token string, tokenType string, now time.Time) (*dao.TokenSession, error) {
	if token == "" {
		return nil, dao.ErrNotFound
	}

	payload, err := d.client.Get(ctx, tokenKey(tokenType, token)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, dao.ErrNotFound
		}
		return nil, fmt.Errorf("load token session: %w", err)
	}

	record, err := decodeTokenSession(payload)
	if err != nil {
		return nil, err
	}
	if record.TokenType != "" && record.TokenType != tokenType {
		return nil, dao.ErrNotFound
	}
	if !record.ExpiresAt.IsZero() && !record.ExpiresAt.After(now) {
		_ = d.client.Del(ctx, tokenKey(tokenType, token)).Err()
		return nil, dao.ErrNotFound
	}
	return record.toDAO(token, tokenType), nil
}

func (d *TokenSessionDAO) RevokeByToken(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	if _, err := d.client.Del(ctx, tokenKey(tokenKeyPrefixForToken(token), token)).Result(); err != nil {
		return fmt.Errorf("revoke token session: %w", err)
	}
	return nil
}

func (d *TokenSessionDAO) RevokeAccountTokens(ctx context.Context, accountID string, tokenType string) error {
	return d.scanAndDelete(ctx, tokenType, func(record redisTokenSession) bool {
		return record.AccountID == accountID && record.TokenType == tokenType
	})
}

func (d *TokenSessionDAO) DeleteExpired(ctx context.Context, before time.Time) error {
	if err := d.scanAndDelete(ctx, daoTokenTypeAccess, func(record redisTokenSession) bool {
		return !record.ExpiresAt.IsZero() && !record.ExpiresAt.After(before)
	}); err != nil {
		return err
	}
	return d.scanAndDelete(ctx, daoTokenTypeRefresh, func(record redisTokenSession) bool {
		return !record.ExpiresAt.IsZero() && !record.ExpiresAt.After(before)
	})
}

func (d *TokenSessionDAO) scanAndDelete(ctx context.Context, tokenType string, predicate func(redisTokenSession) bool) error {
	pattern := tokenPattern(tokenType)
	var cursor uint64
	for {
		keys, next, err := d.client.Scan(ctx, cursor, pattern, scanBatchSize).Result()
		if err != nil {
			return fmt.Errorf("scan token sessions: %w", err)
		}
		for _, key := range keys {
			payload, err := d.client.Get(ctx, key).Bytes()
			if err != nil {
				if errors.Is(err, redis.Nil) {
					continue
				}
				return fmt.Errorf("load token session: %w", err)
			}
			record, err := decodeTokenSession(payload)
			if err != nil {
				return err
			}
			if predicate(*record) {
				if _, err := d.client.Del(ctx, key).Result(); err != nil {
					return fmt.Errorf("delete token session: %w", err)
				}
			}
		}
		if next == 0 {
			return nil
		}
		cursor = next
	}
}

type redisTokenSession struct {
	AccountID string    `json:"account_id"`
	TokenType string    `json:"token_type"`
	TokenHash string    `json:"token_hash"`
	ExpiresAt time.Time `json:"expires_at"`
	Revoked   int16     `json:"revoked"`
	Source    *string   `json:"source,omitempty"`
	CallerIP  *string   `json:"caller_ip,omitempty"`
	UserAgent *string   `json:"user_agent,omitempty"`
	TenantID  int64     `json:"tenant_id"`
}

func decodeTokenSession(payload []byte) (*redisTokenSession, error) {
	var record redisTokenSession
	if err := json.Unmarshal(payload, &record); err != nil {
		return nil, fmt.Errorf("decode token session: %w", err)
	}
	return &record, nil
}

func (r *redisTokenSession) toDAO(token string, tokenType string) *dao.TokenSession {
	return &dao.TokenSession{
		TokenID:   token,
		AccountID: r.AccountID,
		TokenType: tokenType,
		TokenHash: hashToken(token),
		ExpiresAt: r.ExpiresAt,
		Revoked:   r.Revoked,
		Source:    r.Source,
		CallerIP:  r.CallerIP,
		UserAgent: r.UserAgent,
		TenantID:  r.TenantID,
	}
}

func tokenKey(tokenType string, token string) string {
	return tokenKeyPrefixForType(tokenType) + token
}

func tokenPattern(tokenType string) string {
	prefix := tokenKeyPrefixForType(tokenType)
	return prefix + "*"
}

func tokenKeyPrefixForType(tokenType string) string {
	switch tokenType {
	case daoTokenTypeAccess:
		return tokenKeyAccessPrefix
	case daoTokenTypeRefresh:
		return tokenKeyRefreshPrefix
	default:
		return ""
	}
}

func tokenKeyPrefixForToken(token string) string {
	if len(token) >= 4 && token[:4] == "acc_" {
		return tokenKeyAccessPrefix
	}
	if len(token) >= 4 && token[:4] == "ref_" {
		return tokenKeyRefreshPrefix
	}
	return tokenKeyRefreshPrefix
}

const (
	daoTokenTypeAccess  = "access"
	daoTokenTypeRefresh = "refresh"
)

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

var _ dao.TokenSessionDAO = (*TokenSessionDAO)(nil)
