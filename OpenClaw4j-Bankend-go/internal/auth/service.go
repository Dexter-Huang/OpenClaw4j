package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/seaskyland/openclaw4j-backend-go/internal/contextx"
	"github.com/seaskyland/openclaw4j-backend-go/internal/dao"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"

	SourceConsole = "console"
	SourceOpenAPI = "openapi"

	defaultAccessTokenTTL  = 2 * time.Hour
	defaultRefreshTokenTTL = 30 * 24 * time.Hour
)

var (
	ErrInvalidToken             = errors.New("invalid token")
	ErrInvalidAPIKey            = errors.New("invalid api key")
	ErrInvalidCredentials       = errors.New("invalid credentials")
	ErrInvalidRefreshToken      = errors.New("invalid refresh token")
	ErrAccountNotFound          = errors.New("account not found")
	ErrDefaultWorkspaceNotFound = errors.New("default workspace not found")
	ErrTokenSessionUnavailable  = errors.New("token session store is unavailable")
	ErrOAuthUserNotFound        = errors.New("oauth user is missing")
)

// OAuthUser 是外部身份提供方返回、且业务认证链路真正需要的最小用户信息。
// 不把 GitHub 等提供方类型泄露到 auth 包，后续扩展提供方时仍复用同一账号落库逻辑。
type OAuthUser struct {
	Username string
	Name     string
	Email    string
	Icon     string
}

type LegacyAPIKeyEncryptor interface {
	Encrypt(plain string) (string, error)
	Decrypt(encrypted string) (string, error)
}

type ServiceOptions struct {
	Accounts              dao.AccountDAO
	Workspaces            dao.WorkspaceDAO
	APIKeys               dao.APIKeyDAO
	TokenSessions         dao.TokenSessionDAO
	LegacyAPIKeyEncryptor LegacyAPIKeyEncryptor
	Clock                 func() time.Time
	NewRequestID          func() string
}

type Service struct {
	accounts              dao.AccountDAO
	workspaces            dao.WorkspaceDAO
	apiKeys               dao.APIKeyDAO
	tokenSessions         dao.TokenSessionDAO
	legacyAPIKeyEncryptor LegacyAPIKeyEncryptor
	clock                 func() time.Time
	newRequestID          func() string
}

// AccountProfile 是 console profile 接口的脱敏账户视图，绝不能包含 password。
type AccountProfile struct {
	AccountID          string     `json:"account_id"`
	DefaultWorkspaceID string     `json:"default_workspace_id"`
	Username           string     `json:"username"`
	Email              *string    `json:"email"`
	Mobile             *string    `json:"mobile"`
	Status             int16      `json:"status"`
	Type               string     `json:"type"`
	Nickname           *string    `json:"nickname"`
	Icon               *string    `json:"icon"`
	Creator            string     `json:"creator"`
	Modifier           string     `json:"modifier"`
	GmtCreate          time.Time  `json:"gmt_create"`
	GmtModified        time.Time  `json:"gmt_modified"`
	GmtLastLogin       *time.Time `json:"gmt_last_login"`
}

func NewService(options ServiceOptions) *Service {
	clock := options.Clock
	if clock == nil {
		clock = time.Now
	}
	newRequestID := options.NewRequestID
	if newRequestID == nil {
		newRequestID = randomRequestID
	}
	return &Service{
		accounts:              options.Accounts,
		workspaces:            options.Workspaces,
		apiKeys:               options.APIKeys,
		tokenSessions:         options.TokenSessions,
		legacyAPIKeyEncryptor: options.LegacyAPIKeyEncryptor,
		clock:                 clock,
		newRequestID:          newRequestID,
	}
}

func (s *Service) AuthenticateConsoleToken(ctx context.Context, token string, callerIP string) (*contextx.RequestContext, error) {
	if token == "" || s.tokenSessions == nil {
		return nil, ErrInvalidToken
	}
	now := s.clock()
	session, err := s.tokenSessions.FindActiveByToken(ctx, token, TokenTypeAccess, now)
	if err != nil {
		if errors.Is(err, dao.ErrNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("find active token session: %w", err)
	}
	if session == nil {
		return nil, ErrInvalidToken
	}
	source := SourceConsole
	if session.Source != nil && *session.Source != "" {
		source = *session.Source
	}
	return s.buildRequestContext(ctx, session.AccountID, contextx.AuthTypeConsoleToken, source, callerIP, now)
}

func (s *Service) AuthenticateAPIKey(ctx context.Context, apiKey string, callerIP string) (*contextx.RequestContext, error) {
	if apiKey == "" || s.apiKeys == nil {
		return nil, ErrInvalidAPIKey
	}
	record, err := s.apiKeys.FindActiveByHash(ctx, HashToken(apiKey))
	if err != nil && !errors.Is(err, dao.ErrNotFound) {
		return nil, fmt.Errorf("find active api key by hash: %w", err)
	}
	if record == nil && errors.Is(err, dao.ErrNotFound) && s.legacyAPIKeyEncryptor != nil {
		encrypted, encryptErr := s.legacyAPIKeyEncryptor.Encrypt(apiKey)
		if encryptErr != nil {
			return nil, fmt.Errorf("encrypt legacy api key: %w", encryptErr)
		}
		record, err = s.apiKeys.FindActiveByEncryptedKey(ctx, encrypted)
		if err != nil && !errors.Is(err, dao.ErrNotFound) {
			return nil, fmt.Errorf("find active api key by encrypted key: %w", err)
		}
	}
	if record == nil {
		return nil, ErrInvalidAPIKey
	}
	return s.buildRequestContext(ctx, record.AccountID, contextx.AuthTypeAPIKey, SourceOpenAPI, callerIP, s.clock())
}

func (s *Service) Login(ctx context.Context, username string, password string, callerIP string, userAgent string) (*TokenResponse, error) {
	username = strings.TrimSpace(username)
	if username == "" || strings.TrimSpace(password) == "" || s.accounts == nil {
		return nil, ErrInvalidCredentials
	}

	account, err := s.accounts.FindActiveByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, dao.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("find active account by username: %w", err)
	}
	if account == nil || !VerifyPassword(password, account.Password) {
		return nil, ErrInvalidCredentials
	}

	now := s.clock()
	if _, err := s.workspaceForAccount(ctx, account.AccountID); err != nil {
		return nil, err
	}
	if err := s.accounts.UpdateLastLogin(ctx, account.AccountID, now); err != nil {
		return nil, fmt.Errorf("update account last login: %w", err)
	}

	return s.issueConsoleTokens(ctx, account, SourceConsole, callerIP, userAgent, now)
}

// LoginOAuth 对齐 Java AccountService.login(Oauth2User)：外部用户首次登录时创建普通账号和
// 默认工作区，随后签发与密码登录相同的 console token。账号密码是随机且不可由 OAuth 回调返回，
// 因此外部身份不会意外变成可猜测的密码登录入口。
func (s *Service) LoginOAuth(ctx context.Context, user OAuthUser, callerIP string, userAgent string) (*TokenResponse, error) {
	username := strings.TrimSpace(user.Username)
	if username == "" || s.accounts == nil || s.workspaces == nil {
		return nil, ErrOAuthUserNotFound
	}

	account, err := s.accounts.FindActiveByUsername(ctx, username)
	if err != nil && !errors.Is(err, dao.ErrNotFound) {
		return nil, fmt.Errorf("find oauth account: %w", err)
	}
	if errors.Is(err, dao.ErrNotFound) || account == nil {
		account, err = s.createOAuthAccount(ctx, OAuthUser{
			Username: username,
			Name:     strings.TrimSpace(user.Name),
			Email:    strings.TrimSpace(user.Email),
			Icon:     strings.TrimSpace(user.Icon),
		})
		if err != nil {
			return nil, err
		}
	}

	now := s.clock()
	if _, err := s.workspaceForAccount(ctx, account.AccountID); err != nil {
		return nil, err
	}
	return s.issueConsoleTokens(ctx, account, SourceConsole, callerIP, userAgent, now)
}

func (s *Service) createOAuthAccount(ctx context.Context, user OAuthUser) (*dao.Account, error) {
	accountID, err := randomIdentifier()
	if err != nil {
		return nil, err
	}
	randomPassword, err := randomToken("pwd_")
	if err != nil {
		return nil, err
	}
	password, err := HashPassword(randomPassword)
	if err != nil {
		return nil, err
	}
	now := s.clock()
	account := &dao.Account{
		AccountID:   accountID,
		Username:    user.Username,
		Email:       optionalPointer(user.Email),
		Password:    password,
		Nickname:    optionalPointer(user.Name),
		Icon:        optionalPointer(user.Icon),
		Type:        "user",
		Status:      1,
		GmtCreate:   now,
		GmtModified: now,
		Creator:     accountID,
		Modifier:    accountID,
	}
	if err := s.accounts.Create(ctx, *account); err != nil {
		// GitHub 同一用户的并发回调可能同时通过首次查询。唯一约束获胜后复查，
		// 避免把正常的并发登录暴露成 500。
		existing, findErr := s.accounts.FindActiveByUsername(ctx, user.Username)
		if findErr == nil && existing != nil {
			return existing, nil
		}
		return nil, fmt.Errorf("create oauth account: %w", err)
	}

	workspaceID, err := randomIdentifier()
	if err != nil {
		return nil, err
	}
	description := "Default workspace"
	if err := s.workspaces.Create(ctx, dao.Workspace{
		WorkspaceID: workspaceID,
		AccountID:   accountID,
		Status:      1,
		Name:        "Default Workspace",
		Description: &description,
		GmtCreate:   now,
		GmtModified: now,
		Creator:     accountID,
		Modifier:    accountID,
	}); err != nil {
		return nil, fmt.Errorf("create oauth default workspace: %w", err)
	}
	return account, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string, callerIP string, userAgent string) (*TokenResponse, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" || s.tokenSessions == nil {
		return nil, ErrInvalidRefreshToken
	}

	now := s.clock()
	session, err := s.tokenSessions.FindActiveByToken(ctx, refreshToken, TokenTypeRefresh, now)
	if err != nil {
		if errors.Is(err, dao.ErrNotFound) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, fmt.Errorf("find active refresh token session: %w", err)
	}
	if session == nil {
		return nil, ErrInvalidRefreshToken
	}

	account, err := s.accountForID(ctx, session.AccountID)
	if err != nil {
		return nil, err
	}
	if _, err := s.workspaceForAccount(ctx, account.AccountID); err != nil {
		return nil, err
	}

	source := SourceConsole
	if session.Source != nil && *session.Source != "" {
		source = *session.Source
	}
	response, err := s.issueConsoleTokens(ctx, account, source, callerIP, userAgent, now)
	if err != nil {
		return nil, err
	}
	if err := s.tokenSessions.RevokeByToken(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("revoke refresh token session: %w", err)
	}
	return response, nil
}

// 保持 Java 兼容契约：缺失凭证时幂等成功，携带 access token 时立即撤销对应会话。
func (s *Service) Logout(ctx context.Context, accessToken string) error {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return nil
	}
	if s.tokenSessions == nil {
		return ErrTokenSessionUnavailable
	}
	if err := s.tokenSessions.RevokeByToken(ctx, accessToken); err != nil {
		return fmt.Errorf("revoke access token session: %w", err)
	}
	return nil
}

func (s *Service) GetAccountProfile(ctx context.Context, accountID string) (*AccountProfile, error) {
	account, err := s.accountForID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	workspace, err := s.workspaceForAccount(ctx, account.AccountID)
	if err != nil {
		return nil, err
	}
	return &AccountProfile{
		AccountID:          account.AccountID,
		DefaultWorkspaceID: workspace.WorkspaceID,
		Username:           account.Username,
		Email:              account.Email,
		Mobile:             account.Mobile,
		Status:             account.Status,
		Type:               account.Type,
		Nickname:           account.Nickname,
		Icon:               account.Icon,
		Creator:            account.Creator,
		Modifier:           account.Modifier,
		GmtCreate:          account.GmtCreate,
		GmtModified:        account.GmtModified,
		GmtLastLogin:       account.GmtLastLogin,
	}, nil
}

func (s *Service) buildRequestContext(
	ctx context.Context,
	accountID string,
	authType contextx.AuthType,
	source string,
	callerIP string,
	now time.Time,
) (*contextx.RequestContext, error) {
	if s.accounts == nil {
		return nil, ErrAccountNotFound
	}
	account, err := s.accounts.FindActiveByAccountID(ctx, accountID)
	if err != nil {
		if errors.Is(err, dao.ErrNotFound) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("find active account: %w", err)
	}
	if account == nil {
		return nil, ErrAccountNotFound
	}
	if s.workspaces == nil {
		return nil, ErrDefaultWorkspaceNotFound
	}
	workspace, err := s.workspaces.FindDefaultByAccountID(ctx, accountID)
	if err != nil {
		if errors.Is(err, dao.ErrNotFound) {
			return nil, ErrDefaultWorkspaceNotFound
		}
		return nil, fmt.Errorf("find default workspace: %w", err)
	}
	if workspace == nil {
		return nil, ErrDefaultWorkspaceNotFound
	}
	return &contextx.RequestContext{
		StartTime:   now,
		RequestID:   s.newRequestID(),
		AccountID:   account.AccountID,
		Username:    account.Username,
		AccountType: account.Type,
		WorkspaceID: workspace.WorkspaceID,
		CallerIP:    callerIP,
		Source:      source,
		AuthType:    authType,
	}, nil
}

func (s *Service) accountForID(ctx context.Context, accountID string) (*dao.Account, error) {
	if s.accounts == nil {
		return nil, ErrAccountNotFound
	}
	account, err := s.accounts.FindActiveByAccountID(ctx, accountID)
	if err != nil {
		if errors.Is(err, dao.ErrNotFound) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("find active account: %w", err)
	}
	if account == nil {
		return nil, ErrAccountNotFound
	}
	return account, nil
}

func (s *Service) workspaceForAccount(ctx context.Context, accountID string) (*dao.Workspace, error) {
	if s.workspaces == nil {
		return nil, ErrDefaultWorkspaceNotFound
	}
	workspace, err := s.workspaces.FindDefaultByAccountID(ctx, accountID)
	if err != nil {
		if errors.Is(err, dao.ErrNotFound) {
			return nil, ErrDefaultWorkspaceNotFound
		}
		return nil, fmt.Errorf("find default workspace: %w", err)
	}
	if workspace == nil {
		return nil, ErrDefaultWorkspaceNotFound
	}
	return workspace, nil
}

func (s *Service) issueConsoleTokens(
	ctx context.Context,
	account *dao.Account,
	source string,
	callerIP string,
	userAgent string,
	now time.Time,
) (*TokenResponse, error) {
	if s.tokenSessions == nil {
		return nil, ErrTokenSessionUnavailable
	}

	accessToken, err := randomToken("acc_")
	if err != nil {
		return nil, err
	}
	refreshToken, err := randomToken("ref_")
	if err != nil {
		return nil, err
	}

	accessExpiresAt := now.Add(defaultAccessTokenTTL)
	refreshExpiresAt := now.Add(defaultRefreshTokenTTL)
	if err := s.createTokenSession(ctx, account, TokenTypeAccess, accessToken, accessExpiresAt, source, callerIP, userAgent); err != nil {
		return nil, err
	}
	if err := s.createTokenSession(ctx, account, TokenTypeRefresh, refreshToken, refreshExpiresAt, source, callerIP, userAgent); err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    accessExpiresAt.Unix(),
	}, nil
}

func (s *Service) createTokenSession(
	ctx context.Context,
	account *dao.Account,
	tokenType string,
	token string,
	expiresAt time.Time,
	source string,
	callerIP string,
	userAgent string,
) error {
	sourceValue := source
	callerIPValue := strings.TrimSpace(callerIP)
	userAgentValue := strings.TrimSpace(userAgent)
	if err := s.tokenSessions.Create(ctx, dao.TokenSession{
		TokenID:   token,
		AccountID: account.AccountID,
		TokenType: tokenType,
		TokenHash: HashToken(token),
		ExpiresAt: expiresAt,
		Source:    optionalPointer(sourceValue),
		CallerIP:  optionalPointer(callerIPValue),
		UserAgent: optionalPointer(userAgentValue),
		TenantID:  account.TenantID,
	}); err != nil {
		return fmt.Errorf("create %s token session: %w", tokenType, err)
	}
	return nil
}

func randomToken(prefix string) (string, error) {
	var data [32]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return prefix + hex.EncodeToString(data[:]), nil
}

// randomIdentifier 生成与现有 account_id、workspace_id 兼容的 24 位十六进制标识。
// 这些数据库字段限制为 varchar(64)，不能复用用于 token 的 64 位随机串再叠加业务前缀。
func randomIdentifier() (string, error) {
	var data [12]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", fmt.Errorf("generate identifier: %w", err)
	}
	return hex.EncodeToString(data[:]), nil
}

func optionalPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func randomRequestID() string {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return fmt.Sprintf("req_%d", time.Now().UnixNano())
	}
	return "req_" + hex.EncodeToString(data[:])
}
