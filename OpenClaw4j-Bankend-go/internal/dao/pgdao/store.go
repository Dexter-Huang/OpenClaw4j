package pgdao

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/seaskyland/openclaw4j-backend-go/internal/dao"
	"github.com/seaskyland/openclaw4j-backend-go/internal/db"
)

type AccountQueries interface {
	CreateAccount(ctx context.Context, arg db.CreateAccountParams) error
	FindActiveAccountByID(ctx context.Context, accountID string) (db.Account, error)
	FindActiveAccountByUsername(ctx context.Context, username string) (db.Account, error)
	UpdateAccountLastLogin(ctx context.Context, arg db.UpdateAccountLastLoginParams) error
	UpdateAccount(ctx context.Context, arg db.UpdateAccountParams) error
	SoftDeleteAccount(ctx context.Context, arg db.SoftDeleteAccountParams) error
	ListActiveUserAccounts(ctx context.Context, arg db.ListActiveUserAccountsParams) ([]db.Account, error)
	CountActiveUserAccounts(ctx context.Context, name string) (int64, error)
}

type WorkspaceQueries interface {
	CreateWorkspace(ctx context.Context, arg db.CreateWorkspaceParams) error
	FindDefaultWorkspaceByAccountID(ctx context.Context, accountID string) (db.Workspace, error)
	FindActiveWorkspaceByIDAndAccountID(ctx context.Context, arg db.FindActiveWorkspaceByIDAndAccountIDParams) (db.Workspace, error)
	FindActiveWorkspaceByNameAndAccountID(ctx context.Context, arg db.FindActiveWorkspaceByNameAndAccountIDParams) (db.Workspace, error)
	CountActiveWorkspacesByAccountID(ctx context.Context, accountID string) (int64, error)
	ListActiveWorkspacesByAccountID(ctx context.Context, arg db.ListActiveWorkspacesByAccountIDParams) ([]db.Workspace, error)
	UpdateWorkspace(ctx context.Context, arg db.UpdateWorkspaceParams) error
	SoftDeleteWorkspace(ctx context.Context, arg db.SoftDeleteWorkspaceParams) error
}

type APIKeyQueries interface {
	CreateAPIKey(ctx context.Context, arg db.CreateAPIKeyParams) (int64, error)
	FindActiveAPIKeyByEncryptedKey(ctx context.Context, apiKey string) (db.FindActiveAPIKeyByEncryptedKeyRow, error)
	FindActiveAPIKeyByHash(ctx context.Context, apiKeyHash pgtype.Text) (db.FindActiveAPIKeyByHashRow, error)
	CountActiveAPIKeysByAccountID(ctx context.Context, accountID string) (int64, error)
	ListActiveAPIKeysByAccountID(ctx context.Context, arg db.ListActiveAPIKeysByAccountIDParams) ([]db.ListActiveAPIKeysByAccountIDRow, error)
	FindActiveAPIKeyByIDAndAccountID(ctx context.Context, arg db.FindActiveAPIKeyByIDAndAccountIDParams) (db.FindActiveAPIKeyByIDAndAccountIDRow, error)
	UpdateAPIKeyDescription(ctx context.Context, arg db.UpdateAPIKeyDescriptionParams) error
	SoftDeleteAPIKey(ctx context.Context, arg db.SoftDeleteAPIKeyParams) error
}

type TokenSessionQueries interface {
	CreateTokenSession(ctx context.Context, arg db.CreateTokenSessionParams) error
	FindActiveTokenSessionByHash(ctx context.Context, arg db.FindActiveTokenSessionByHashParams) (db.AuthTokenSession, error)
	RevokeTokenSessionByHash(ctx context.Context, tokenHash string) error
	RevokeAccountTokens(ctx context.Context, arg db.RevokeAccountTokensParams) error
	DeleteExpiredTokenSessions(ctx context.Context, expiresAt pgtype.Timestamp) error
}

type AccountDAO struct {
	q AccountQueries
}

func NewAccountDAO(q AccountQueries) *AccountDAO {
	return &AccountDAO{q: q}
}

func (d *AccountDAO) FindActiveByAccountID(ctx context.Context, accountID string) (*dao.Account, error) {
	account, err := d.q.FindActiveAccountByID(ctx, accountID)
	if err != nil {
		return nil, convertNotFound(err)
	}
	return mapAccount(account), nil
}

func (d *AccountDAO) FindActiveByUsername(ctx context.Context, username string) (*dao.Account, error) {
	account, err := d.q.FindActiveAccountByUsername(ctx, username)
	if err != nil {
		return nil, convertNotFound(err)
	}
	return mapAccount(account), nil
}

func (d *AccountDAO) UpdateLastLogin(ctx context.Context, accountID string, lastLogin time.Time) error {
	return convertNotFound(d.q.UpdateAccountLastLogin(ctx, db.UpdateAccountLastLoginParams{
		AccountID:    accountID,
		GmtLastLogin: pgTimestamp(lastLogin),
	}))
}

type WorkspaceDAO struct {
	q WorkspaceQueries
}

func NewWorkspaceDAO(q WorkspaceQueries) *WorkspaceDAO {
	return &WorkspaceDAO{q: q}
}

func (d *WorkspaceDAO) FindDefaultByAccountID(ctx context.Context, accountID string) (*dao.Workspace, error) {
	workspace, err := d.q.FindDefaultWorkspaceByAccountID(ctx, accountID)
	if err != nil {
		return nil, convertNotFound(err)
	}
	return mapWorkspace(workspace), nil
}

type APIKeyDAO struct {
	q APIKeyQueries
}

func NewAPIKeyDAO(q APIKeyQueries) *APIKeyDAO {
	return &APIKeyDAO{q: q}
}

func (d *APIKeyDAO) FindActiveByEncryptedKey(ctx context.Context, encryptedKey string) (*dao.APIKey, error) {
	apiKey, err := d.q.FindActiveAPIKeyByEncryptedKey(ctx, encryptedKey)
	if err != nil {
		return nil, convertNotFound(err)
	}
	return mapAPIKey(apiKey.ID, apiKey.AccountID, apiKey.ApiKey, apiKey.Status, apiKey.Description, apiKey.GmtCreate, apiKey.GmtModified, apiKey.Creator, apiKey.Modifier, apiKey.TenantID), nil
}

func (d *APIKeyDAO) FindActiveByHash(ctx context.Context, keyHash string) (*dao.APIKey, error) {
	apiKey, err := d.q.FindActiveAPIKeyByHash(ctx, pgText(keyHash))
	if err != nil {
		return nil, convertAPIKeyHashNotFound(err)
	}
	return mapAPIKey(apiKey.ID, apiKey.AccountID, apiKey.ApiKey, apiKey.Status, apiKey.Description, apiKey.GmtCreate, apiKey.GmtModified, apiKey.Creator, apiKey.Modifier, apiKey.TenantID), nil
}

func convertAPIKeyHashNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return dao.ErrNotFound
	}
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && postgresError.Code == "42703" {
		return dao.ErrNotFound
	}
	return err
}

func (d *APIKeyDAO) CountActiveByAccountID(ctx context.Context, accountID string) (int64, error) {
	count, err := d.q.CountActiveAPIKeysByAccountID(ctx, accountID)
	return count, convertNotFound(err)
}

func (d *APIKeyDAO) ListActiveByAccountID(ctx context.Context, accountID string, limit int32, offset int32) ([]dao.APIKey, error) {
	rows, err := d.q.ListActiveAPIKeysByAccountID(ctx, db.ListActiveAPIKeysByAccountIDParams{
		AccountID: accountID,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, convertNotFound(err)
	}
	apiKeys := make([]dao.APIKey, 0, len(rows))
	for _, row := range rows {
		apiKey := mapAPIKey(row.ID, row.AccountID, row.ApiKey, row.Status, row.Description, row.GmtCreate, row.GmtModified, row.Creator, row.Modifier, row.TenantID)
		apiKeys = append(apiKeys, *apiKey)
	}
	return apiKeys, nil
}

type TokenSessionDAO struct {
	q TokenSessionQueries
}

func NewTokenSessionDAO(q TokenSessionQueries) *TokenSessionDAO {
	return &TokenSessionDAO{q: q}
}

func (d *TokenSessionDAO) Create(ctx context.Context, session dao.TokenSession) error {
	return convertNotFound(d.q.CreateTokenSession(ctx, db.CreateTokenSessionParams{
		TokenID:   session.TokenID,
		AccountID: session.AccountID,
		TokenType: session.TokenType,
		TokenHash: session.TokenHash,
		ExpiresAt: pgTimestamp(session.ExpiresAt),
		Source:    pgOptionalText(session.Source),
		CallerIp:  pgOptionalText(session.CallerIP),
		UserAgent: pgOptionalText(session.UserAgent),
		TenantID:  pgOptionalInt8(session.TenantID),
	}))
}

func (d *TokenSessionDAO) FindActiveByToken(ctx context.Context, token string, tokenType string, now time.Time) (*dao.TokenSession, error) {
	session, err := d.q.FindActiveTokenSessionByHash(ctx, db.FindActiveTokenSessionByHashParams{
		TokenHash: hashToken(token),
		TokenType: tokenType,
		ExpiresAt: pgTimestamp(now),
	})
	if err != nil {
		return nil, convertNotFound(err)
	}
	return mapTokenSession(session), nil
}

func (d *TokenSessionDAO) RevokeByToken(ctx context.Context, token string) error {
	return convertNotFound(d.q.RevokeTokenSessionByHash(ctx, hashToken(token)))
}

func (d *TokenSessionDAO) RevokeAccountTokens(ctx context.Context, accountID string, tokenType string) error {
	return convertNotFound(d.q.RevokeAccountTokens(ctx, db.RevokeAccountTokensParams{
		AccountID: accountID,
		TokenType: tokenType,
	}))
}

func (d *TokenSessionDAO) DeleteExpired(ctx context.Context, before time.Time) error {
	return convertNotFound(d.q.DeleteExpiredTokenSessions(ctx, pgTimestamp(before)))
}

func mapAccount(row db.Account) *dao.Account {
	return &dao.Account{
		ID:           row.ID,
		AccountID:    row.AccountID,
		Username:     row.Username,
		Email:        optionalString(row.Email),
		Mobile:       optionalString(row.Mobile),
		Password:     row.Password,
		Nickname:     optionalString(row.Nickname),
		Icon:         optionalString(row.Icon),
		Type:         row.Type,
		Status:       row.Status,
		GmtCreate:    row.GmtCreate.Time,
		GmtModified:  row.GmtModified.Time,
		GmtLastLogin: optionalTime(row.GmtLastLogin),
		Creator:      row.Creator,
		Modifier:     row.Modifier,
		TenantID:     optionalInt8(row.TenantID),
	}
}

func mapWorkspace(row db.Workspace) *dao.Workspace {
	return &dao.Workspace{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
		AccountID:   row.AccountID,
		Status:      row.Status,
		Name:        row.Name,
		Description: optionalString(row.Description),
		Config:      optionalString(row.Config),
		GmtCreate:   row.GmtCreate.Time,
		GmtModified: row.GmtModified.Time,
		Creator:     row.Creator,
		Modifier:    row.Modifier,
		TenantID:    optionalInt8(row.TenantID),
	}
}

func mapAPIKey(
	id int64,
	accountID string,
	apiKey string,
	status int16,
	description pgtype.Text,
	gmtCreate pgtype.Timestamp,
	gmtModified pgtype.Timestamp,
	creator string,
	modifier string,
	tenantID pgtype.Int8,
) *dao.APIKey {
	return &dao.APIKey{
		ID:          id,
		AccountID:   accountID,
		APIKey:      apiKey,
		Status:      status,
		Description: optionalString(description),
		GmtCreate:   gmtCreate.Time,
		GmtModified: gmtModified.Time,
		Creator:     creator,
		Modifier:    modifier,
		TenantID:    optionalInt8(tenantID),
	}
}

func mapTokenSession(row db.AuthTokenSession) *dao.TokenSession {
	return &dao.TokenSession{
		ID:          row.ID,
		TokenID:     row.TokenID,
		AccountID:   row.AccountID,
		TokenType:   row.TokenType,
		TokenHash:   row.TokenHash,
		ExpiresAt:   row.ExpiresAt.Time,
		Revoked:     row.Revoked,
		Source:      optionalString(row.Source),
		CallerIP:    optionalString(row.CallerIp),
		UserAgent:   optionalString(row.UserAgent),
		GmtCreate:   row.GmtCreate.Time,
		GmtModified: row.GmtModified.Time,
		TenantID:    optionalInt8(row.TenantID),
	}
}

func convertNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return dao.ErrNotFound
	}
	return err
}

func optionalString(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func optionalTime(value pgtype.Timestamp) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func optionalInt8(value pgtype.Int8) int64 {
	if !value.Valid {
		return 0
	}
	return value.Int64
}

func pgText(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: true}
}

func pgOptionalText(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}
	return pgText(*value)
}

func pgTimestamp(value time.Time) pgtype.Timestamp {
	return pgtype.Timestamp{Time: value, Valid: true}
}

func pgOptionalInt8(value int64) pgtype.Int8 {
	return pgtype.Int8{Int64: value, Valid: true}
}

func pgOptionalInt8Pointer(value *int64) pgtype.Int8 {
	if value == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *value, Valid: true}
}

func optionalInt4(value pgtype.Int4) *int32 {
	if !value.Valid {
		return nil
	}
	return &value.Int32
}

func pgOptionalInt4(value *int32) pgtype.Int4 {
	if value == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: *value, Valid: true}
}

func optionalFloat8(value pgtype.Float8) *float64 {
	if !value.Valid {
		return nil
	}
	return &value.Float64
}

func pgOptionalFloat8(value *float64) pgtype.Float8 {
	if value == nil {
		return pgtype.Float8{}
	}
	return pgtype.Float8{Float64: *value, Valid: true}
}

func optionalInt8Pointer(value pgtype.Int8) *int64 {
	if !value.Valid {
		return nil
	}
	return &value.Int64
}

var (
	_ dao.AccountDAO      = (*AccountDAO)(nil)
	_ dao.WorkspaceDAO    = (*WorkspaceDAO)(nil)
	_ dao.APIKeyDAO       = (*APIKeyDAO)(nil)
	_ dao.TokenSessionDAO = (*TokenSessionDAO)(nil)
)

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
