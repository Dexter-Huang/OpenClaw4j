package pgdao

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/seaskyland/openclaw4j-backend-go/internal/dao"
	"github.com/seaskyland/openclaw4j-backend-go/internal/db"
)

func TestAccountDAOFindActiveByAccountIDMapsRow(t *testing.T) {
	now := time.Date(2026, 7, 18, 10, 30, 0, 0, time.UTC)
	query := &fakeAccountQueries{
		account: db.Account{
			ID:           1,
			AccountID:    "acct_1",
			Username:     "alice",
			Email:        textValue("alice@example.com"),
			Mobile:       textValue("13800000000"),
			Password:     "hash",
			Nickname:     textValue("Alice"),
			Icon:         textValue("avatar.png"),
			Type:         "admin",
			Status:       1,
			GmtCreate:    timestampValue(now),
			GmtModified:  timestampValue(now),
			GmtLastLogin: timestampValue(now),
			Creator:      "system",
			Modifier:     "system",
			TenantID:     int8Value(7),
		},
	}

	account, err := NewAccountDAO(query).FindActiveByAccountID(context.Background(), "acct_1")
	if err != nil {
		t.Fatalf("FindActiveByAccountID returned error: %v", err)
	}

	if account.AccountID != "acct_1" || account.Username != "alice" || account.Type != "admin" {
		t.Fatalf("unexpected account: %#v", account)
	}
	if account.Email == nil || *account.Email != "alice@example.com" {
		t.Fatalf("email was not mapped: %#v", account)
	}
	if account.GmtLastLogin == nil || !account.GmtLastLogin.Equal(now) {
		t.Fatalf("last login was not mapped: %#v", account)
	}
	if account.TenantID != 7 {
		t.Fatalf("tenant id was not mapped: %#v", account)
	}
}

func TestAccountDAOConvertsNoRows(t *testing.T) {
	query := &fakeAccountQueries{err: pgx.ErrNoRows}

	_, err := NewAccountDAO(query).FindActiveByUsername(context.Background(), "missing")
	if !errors.Is(err, dao.ErrNotFound) {
		t.Fatalf("expected dao.ErrNotFound, got %v", err)
	}
}

func TestAccountDAOUpdateLastLoginPassesTimestamp(t *testing.T) {
	lastLogin := time.Date(2026, 7, 18, 11, 0, 0, 0, time.UTC)
	query := &fakeAccountQueries{}

	err := NewAccountDAO(query).UpdateLastLogin(context.Background(), "acct_1", lastLogin)
	if err != nil {
		t.Fatalf("UpdateLastLogin returned error: %v", err)
	}

	if query.lastUpdate.AccountID != "acct_1" || !query.lastUpdate.GmtLastLogin.Time.Equal(lastLogin) || !query.lastUpdate.GmtLastLogin.Valid {
		t.Fatalf("unexpected update params: %#v", query.lastUpdate)
	}
}

func TestWorkspaceDAOFindDefaultByAccountIDMapsRow(t *testing.T) {
	now := time.Date(2026, 7, 18, 10, 30, 0, 0, time.UTC)
	query := &fakeWorkspaceQueries{workspace: db.Workspace{
		ID:          9,
		WorkspaceID: "ws_1",
		AccountID:   "acct_1",
		Status:      1,
		Name:        "default",
		Description: textValue("desc"),
		Config:      textValue("{}"),
		GmtCreate:   timestampValue(now),
		GmtModified: timestampValue(now),
		Creator:     "system",
		Modifier:    "system",
		TenantID:    int8Value(7),
	}}

	workspace, err := NewWorkspaceDAO(query).FindDefaultByAccountID(context.Background(), "acct_1")
	if err != nil {
		t.Fatalf("FindDefaultByAccountID returned error: %v", err)
	}

	if workspace.WorkspaceID != "ws_1" || workspace.AccountID != "acct_1" || workspace.Name != "default" {
		t.Fatalf("unexpected workspace: %#v", workspace)
	}
	if workspace.Description == nil || *workspace.Description != "desc" {
		t.Fatalf("description was not mapped: %#v", workspace)
	}
}

func TestAPIKeyDAOMapsHashEncryptedAndListRows(t *testing.T) {
	now := time.Date(2026, 7, 18, 10, 30, 0, 0, time.UTC)
	row := db.FindActiveAPIKeyByHashRow{
		ID:          3,
		AccountID:   "acct_1",
		ApiKey:      "cipher",
		Status:      1,
		Description: textValue("key"),
		GmtCreate:   timestampValue(now),
		GmtModified: timestampValue(now),
		Creator:     "system",
		Modifier:    "system",
		TenantID:    int8Value(7),
	}
	query := &fakeAPIKeyQueries{
		hashRow: row,
		encryptedRow: db.FindActiveAPIKeyByEncryptedKeyRow{
			ID:          row.ID,
			AccountID:   row.AccountID,
			ApiKey:      row.ApiKey,
			Status:      row.Status,
			Description: row.Description,
			GmtCreate:   row.GmtCreate,
			GmtModified: row.GmtModified,
			Creator:     row.Creator,
			Modifier:    row.Modifier,
			TenantID:    row.TenantID,
		},
		listRows: []db.ListActiveAPIKeysByAccountIDRow{{
			ID:          row.ID,
			AccountID:   row.AccountID,
			ApiKey:      row.ApiKey,
			Status:      row.Status,
			Description: row.Description,
			GmtCreate:   row.GmtCreate,
			GmtModified: row.GmtModified,
			Creator:     row.Creator,
			Modifier:    row.Modifier,
			TenantID:    row.TenantID,
		}},
		count: 1,
	}
	apiKeyDAO := NewAPIKeyDAO(query)

	apiKey, err := apiKeyDAO.FindActiveByHash(context.Background(), "hash")
	if err != nil {
		t.Fatalf("FindActiveByHash returned error: %v", err)
	}
	if query.lastHash.String != "hash" || !query.lastHash.Valid || apiKey.AccountID != "acct_1" {
		t.Fatalf("hash lookup was not mapped: params=%#v apiKey=%#v", query.lastHash, apiKey)
	}

	apiKey, err = apiKeyDAO.FindActiveByEncryptedKey(context.Background(), "cipher")
	if err != nil {
		t.Fatalf("FindActiveByEncryptedKey returned error: %v", err)
	}
	if query.lastEncrypted != "cipher" || apiKey.APIKey != "cipher" {
		t.Fatalf("encrypted lookup was not mapped: params=%#v apiKey=%#v", query.lastEncrypted, apiKey)
	}

	count, err := apiKeyDAO.CountActiveByAccountID(context.Background(), "acct_1")
	if err != nil || count != 1 {
		t.Fatalf("unexpected count: count=%d err=%v", count, err)
	}

	items, err := apiKeyDAO.ListActiveByAccountID(context.Background(), "acct_1", 20, 40)
	if err != nil {
		t.Fatalf("ListActiveByAccountID returned error: %v", err)
	}
	if len(items) != 1 || items[0].Description == nil || *items[0].Description != "key" {
		t.Fatalf("list rows were not mapped: %#v", items)
	}
	if query.lastList.AccountID != "acct_1" || query.lastList.Limit != 20 || query.lastList.Offset != 40 {
		t.Fatalf("list params were not passed: %#v", query.lastList)
	}
}

func TestTokenSessionDAOMapsCreateFindAndMutations(t *testing.T) {
	now := time.Date(2026, 7, 18, 10, 30, 0, 0, time.UTC)
	source := "console"
	callerIP := "127.0.0.1"
	userAgent := "ua"
	query := &fakeTokenSessionQueries{
		session: db.AuthTokenSession{
			ID:          5,
			TokenID:     "token_1",
			AccountID:   "acct_1",
			TokenType:   "access",
			TokenHash:   "hash",
			ExpiresAt:   timestampValue(now.Add(time.Hour)),
			Revoked:     0,
			Source:      textValue(source),
			CallerIp:    textValue(callerIP),
			UserAgent:   textValue(userAgent),
			GmtCreate:   timestampValue(now),
			GmtModified: timestampValue(now),
			TenantID:    int8Value(7),
		},
	}
	tokenDAO := NewTokenSessionDAO(query)

	err := tokenDAO.Create(context.Background(), dao.TokenSession{
		TokenID:   "token_1",
		AccountID: "acct_1",
		TokenType: "access",
		TokenHash: "hash",
		ExpiresAt: now.Add(time.Hour),
		Source:    &source,
		CallerIP:  &callerIP,
		UserAgent: &userAgent,
		TenantID:  7,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if query.lastCreate.TokenID != "token_1" || !query.lastCreate.Source.Valid || query.lastCreate.TenantID.Int64 != 7 {
		t.Fatalf("create params were not mapped: %#v", query.lastCreate)
	}

	session, err := tokenDAO.FindActiveByToken(context.Background(), "hash", "access", now)
	if err != nil {
		t.Fatalf("FindActiveByToken returned error: %v", err)
	}
	if session.TokenID != "token_1" || session.Source == nil || *session.Source != source {
		t.Fatalf("session was not mapped: %#v", session)
	}
	if query.lastFind.TokenHash == "" || query.lastFind.TokenType != "access" || !query.lastFind.ExpiresAt.Time.Equal(now) {
		t.Fatalf("find params were not passed: %#v", query.lastFind)
	}

	if err := tokenDAO.RevokeByToken(context.Background(), "hash"); err != nil {
		t.Fatalf("RevokeByToken returned error: %v", err)
	}
	if err := tokenDAO.RevokeAccountTokens(context.Background(), "acct_1", "access"); err != nil {
		t.Fatalf("RevokeAccountTokens returned error: %v", err)
	}
	if err := tokenDAO.DeleteExpired(context.Background(), now); err != nil {
		t.Fatalf("DeleteExpired returned error: %v", err)
	}
	if query.lastRevokedHash == "" || query.lastRevokeAccount.AccountID != "acct_1" || !query.lastDeleteExpired.Time.Equal(now) {
		t.Fatalf("mutation params were not passed")
	}
}

func TestTokenSessionDAOConvertsNoRows(t *testing.T) {
	query := &fakeTokenSessionQueries{err: pgx.ErrNoRows}

	_, err := NewTokenSessionDAO(query).FindActiveByToken(context.Background(), "hash", "access", time.Now())
	if !errors.Is(err, dao.ErrNotFound) {
		t.Fatalf("expected dao.ErrNotFound, got %v", err)
	}
}

type fakeAccountQueries struct {
	account    db.Account
	err        error
	lastUpdate db.UpdateAccountLastLoginParams
}

func (f *fakeAccountQueries) FindActiveAccountByID(context.Context, string) (db.Account, error) {
	return f.account, f.err
}

func (f *fakeAccountQueries) FindActiveAccountByUsername(context.Context, string) (db.Account, error) {
	return f.account, f.err
}

func (f *fakeAccountQueries) UpdateAccountLastLogin(_ context.Context, arg db.UpdateAccountLastLoginParams) error {
	f.lastUpdate = arg
	return f.err
}

type fakeWorkspaceQueries struct {
	workspace db.Workspace
	err       error
}

func (f *fakeWorkspaceQueries) FindDefaultWorkspaceByAccountID(context.Context, string) (db.Workspace, error) {
	return f.workspace, f.err
}

type fakeAPIKeyQueries struct {
	hashRow       db.FindActiveAPIKeyByHashRow
	encryptedRow  db.FindActiveAPIKeyByEncryptedKeyRow
	listRows      []db.ListActiveAPIKeysByAccountIDRow
	count         int64
	err           error
	lastHash      pgtype.Text
	lastEncrypted string
	lastList      db.ListActiveAPIKeysByAccountIDParams
}

func (f *fakeAPIKeyQueries) FindActiveAPIKeyByHash(_ context.Context, apiKeyHash pgtype.Text) (db.FindActiveAPIKeyByHashRow, error) {
	f.lastHash = apiKeyHash
	return f.hashRow, f.err
}

func (f *fakeAPIKeyQueries) FindActiveAPIKeyByEncryptedKey(_ context.Context, apiKey string) (db.FindActiveAPIKeyByEncryptedKeyRow, error) {
	f.lastEncrypted = apiKey
	return f.encryptedRow, f.err
}

func (f *fakeAPIKeyQueries) CountActiveAPIKeysByAccountID(context.Context, string) (int64, error) {
	return f.count, f.err
}

func (f *fakeAPIKeyQueries) ListActiveAPIKeysByAccountID(_ context.Context, arg db.ListActiveAPIKeysByAccountIDParams) ([]db.ListActiveAPIKeysByAccountIDRow, error) {
	f.lastList = arg
	return f.listRows, f.err
}

type fakeTokenSessionQueries struct {
	session           db.AuthTokenSession
	err               error
	lastCreate        db.CreateTokenSessionParams
	lastFind          db.FindActiveTokenSessionByHashParams
	lastRevokedHash   string
	lastRevokeAccount db.RevokeAccountTokensParams
	lastDeleteExpired pgtype.Timestamp
}

func (f *fakeTokenSessionQueries) CreateTokenSession(_ context.Context, arg db.CreateTokenSessionParams) error {
	f.lastCreate = arg
	return f.err
}

func (f *fakeTokenSessionQueries) FindActiveTokenSessionByHash(_ context.Context, arg db.FindActiveTokenSessionByHashParams) (db.AuthTokenSession, error) {
	f.lastFind = arg
	return f.session, f.err
}

func (f *fakeTokenSessionQueries) RevokeTokenSessionByHash(_ context.Context, tokenHash string) error {
	f.lastRevokedHash = tokenHash
	return f.err
}

func (f *fakeTokenSessionQueries) RevokeAccountTokens(_ context.Context, arg db.RevokeAccountTokensParams) error {
	f.lastRevokeAccount = arg
	return f.err
}

func (f *fakeTokenSessionQueries) DeleteExpiredTokenSessions(_ context.Context, expiresAt pgtype.Timestamp) error {
	f.lastDeleteExpired = expiresAt
	return f.err
}

func textValue(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: true}
}

func timestampValue(value time.Time) pgtype.Timestamp {
	return pgtype.Timestamp{Time: value, Valid: true}
}

func int8Value(value int64) pgtype.Int8 {
	return pgtype.Int8{Int64: value, Valid: true}
}
