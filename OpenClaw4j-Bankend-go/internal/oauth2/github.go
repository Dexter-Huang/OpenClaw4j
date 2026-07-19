package oauth2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/seaskyland/openclaw4j-backend-go/internal/auth"
)

const (
	defaultAuthorizeURL = "https://github.com/login/oauth/authorize"
	defaultTokenURL     = "https://github.com/login/oauth/access_token"
	defaultUserInfoURL  = "https://api.github.com/user"
)

var ErrNotConfigured = errors.New("github oauth2 is not configured")

// GitHubConfig 与 Java OAuth2Config.Github 对齐。外部 URL 保留为可配置项，便于私有 GitHub
// Enterprise 和本地协议测试；生产默认仍使用 github.com。
type GitHubConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	AuthorizeURL string
	TokenURL     string
	UserInfoURL  string
}

// GitHubService 只处理 OAuth2 协议，不关心账号创建与 token 签发。
type GitHubService struct {
	config GitHubConfig
	client *http.Client
}

func NewGitHubService(config GitHubConfig, client *http.Client) *GitHubService {
	if strings.TrimSpace(config.AuthorizeURL) == "" {
		config.AuthorizeURL = defaultAuthorizeURL
	}
	if strings.TrimSpace(config.TokenURL) == "" {
		config.TokenURL = defaultTokenURL
	}
	if strings.TrimSpace(config.UserInfoURL) == "" {
		config.UserInfoURL = defaultUserInfoURL
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &GitHubService{config: config, client: client}
}

func (s *GitHubService) AuthorizationURL() (string, error) {
	if !s.configured() {
		return "", ErrNotConfigured
	}
	endpoint, err := url.Parse(s.config.AuthorizeURL)
	if err != nil {
		return "", fmt.Errorf("parse github authorize url: %w", err)
	}
	query := endpoint.Query()
	query.Set("client_id", s.config.ClientID)
	query.Set("redirect_uri", s.config.RedirectURI)
	query.Set("scope", "user:email")
	endpoint.RawQuery = query.Encode()
	return endpoint.String(), nil
}

func (s *GitHubService) Authenticate(ctx context.Context, code string) (auth.OAuthUser, error) {
	if !s.configured() {
		return auth.OAuthUser{}, ErrNotConfigured
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return auth.OAuthUser{}, errors.New("github authorization code is missing")
	}

	accessToken, err := s.exchangeCode(ctx, code)
	if err != nil {
		return auth.OAuthUser{}, err
	}
	return s.fetchUser(ctx, accessToken)
}

func (s *GitHubService) exchangeCode(ctx context.Context, code string) (string, error) {
	payload, err := json.Marshal(map[string]string{
		"client_id":     s.config.ClientID,
		"client_secret": s.config.ClientSecret,
		"code":          code,
		"redirect_uri":  s.config.RedirectURI,
	})
	if err != nil {
		return "", fmt.Errorf("encode github token request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.config.TokenURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("build github token request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	response, err := s.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("call github token endpoint: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("github token endpoint returned status %d", response.StatusCode)
	}
	var body struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&body); err != nil {
		return "", fmt.Errorf("decode github token response: %w", err)
	}
	if body.Error != "" || strings.TrimSpace(body.AccessToken) == "" {
		return "", errors.New("github token response did not contain an access token")
	}
	return body.AccessToken, nil
}

func (s *GitHubService) fetchUser(ctx context.Context, accessToken string) (auth.OAuthUser, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.config.UserInfoURL, nil)
	if err != nil {
		return auth.OAuthUser{}, fmt.Errorf("build github user request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response, err := s.client.Do(request)
	if err != nil {
		return auth.OAuthUser{}, fmt.Errorf("call github user endpoint: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return auth.OAuthUser{}, fmt.Errorf("github user endpoint returned status %d", response.StatusCode)
	}
	var body struct {
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&body); err != nil {
		return auth.OAuthUser{}, fmt.Errorf("decode github user response: %w", err)
	}
	if strings.TrimSpace(body.Login) == "" {
		return auth.OAuthUser{}, errors.New("github user response did not contain login")
	}
	return auth.OAuthUser{Username: body.Login, Name: body.Name, Email: body.Email, Icon: body.AvatarURL}, nil
}

func (s *GitHubService) configured() bool {
	return strings.TrimSpace(s.config.ClientID) != "" && strings.TrimSpace(s.config.ClientSecret) != "" && strings.TrimSpace(s.config.RedirectURI) != ""
}
