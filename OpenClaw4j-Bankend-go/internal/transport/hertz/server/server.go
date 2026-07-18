package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/seaskyland/openclaw4j-backend-go/internal/auth"
	"github.com/seaskyland/openclaw4j-backend-go/internal/contextx"
	authmiddleware "github.com/seaskyland/openclaw4j-backend-go/internal/transport/hertz/middleware"
)

type Options struct {
	HertzOptions          []config.Option
	Authenticator         authmiddleware.Authenticator
	SessionIssuer         SessionIssuer
	RegisterConsoleRoutes func(group *route.RouterGroup)
	RegisterAPIRoutes     func(group *route.RouterGroup)
}

type SessionIssuer interface {
	Login(ctx context.Context, username string, password string, callerIP string, userAgent string) (*auth.TokenResponse, error)
	RefreshToken(ctx context.Context, refreshToken string, callerIP string, userAgent string) (*auth.TokenResponse, error)
	Logout(ctx context.Context, accessToken string) error
}

func New(options Options) *server.Hertz {
	h := server.New(options.HertzOptions...)
	h.Use(corsMiddleware())
	registerCORSPreflightRoute(h)
	registerHealthRoute(h)
	registerPublicConsoleRoutes(h, options)
	registerConsoleGroup(h, options)
	registerPublicAPIRoutes(h)
	registerAPIGroup(h, options)
	return h
}

func registerHealthRoute(h *server.Hertz) {
	h.Engine.Handle(consts.MethodGet, "/healthz", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(consts.StatusOK, map[string]string{"status": "ok"})
	})
}

func registerCORSPreflightRoute(h *server.Hertz) {
	h.Engine.Handle(consts.MethodOptions, "/*path", func(ctx context.Context, c *app.RequestContext) {
		setCORSHeaders(c)
		c.AbortWithStatus(consts.StatusNoContent)
	})
}

func registerPublicConsoleRoutes(h *server.Hertz, options Options) {
	h.Engine.Handle(consts.MethodPost, "/console/v1/auth/login", loginHandler(options.SessionIssuer))
	h.Engine.Handle(consts.MethodPost, "/console/v1/auth/refresh-token", refreshTokenHandler(options.SessionIssuer))
	h.Engine.Handle(consts.MethodPost, "/console/v1/auth/logout", logoutHandler(options.SessionIssuer))
	h.Engine.Handle(consts.MethodGet, "/console/v1/system/global-config", func(_ context.Context, c *app.RequestContext) {
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), globalConfigResponse{
			LoginMethod:  "preset_account",
			UploadMethod: "file",
		}))
	})
}

func registerConsoleGroup(h *server.Hertz, options Options) {
	console := h.Group("/console/v1")
	console.Use(authmiddleware.ConsoleAuth(options.Authenticator))
	console.GET("/accounts/profile", accountProfileHandler)
	console.GET("/models/:modelType/selector", modelSelectorHandler)
	if options.RegisterConsoleRoutes != nil {
		options.RegisterConsoleRoutes(console)
	}
}

func registerPublicAPIRoutes(h *server.Hertz) {
	h.Engine.Handle(consts.MethodGet, "/api/models", apiModelsHandler)
}

func registerAPIGroup(h *server.Hertz, options Options) {
	api := h.Group("/api/v1")
	api.Use(authmiddleware.APIKeyAuth(options.Authenticator))
	if options.RegisterAPIRoutes != nil {
		options.RegisterAPIRoutes(api)
	}
}

func corsMiddleware() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		setCORSHeaders(c)
		c.Next(ctx)
	}
}

func callerIP(c *app.RequestContext) string {
	if realIP := strings.TrimSpace(c.Request.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	if forwardedFor := strings.TrimSpace(c.Request.Header.Get("X-Forwarded-For")); forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		return strings.TrimSpace(parts[0])
	}
	return c.ClientIP()
}

func loginHandler(issuer SessionIssuer) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if issuer == nil {
			c.JSON(consts.StatusServiceUnavailable, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "session issuer is unavailable"))
			return
		}

		var payload loginRequest
		if err := json.Unmarshal(c.Request.Body(), &payload); err != nil {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_JSON", "request body must be valid json"))
			return
		}

		response, err := issuer.Login(ctx, payload.Username, payload.Password, callerIP(c), strings.TrimSpace(c.Request.Header.Get("User-Agent")))
		if err != nil {
			status, code, message := mapSessionError(err)
			c.JSON(status, errorEnvelope(newRequestID(), code, message))
			return
		}

		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), response))
	}
}

func refreshTokenHandler(issuer SessionIssuer) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if issuer == nil {
			c.JSON(consts.StatusServiceUnavailable, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "session issuer is unavailable"))
			return
		}

		var payload refreshTokenRequest
		if err := json.Unmarshal(c.Request.Body(), &payload); err != nil {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_JSON", "request body must be valid json"))
			return
		}

		response, err := issuer.RefreshToken(ctx, payload.RefreshToken, callerIP(c), strings.TrimSpace(c.Request.Header.Get("User-Agent")))
		if err != nil {
			status, code, message := mapSessionError(err)
			c.JSON(status, errorEnvelope(newRequestID(), code, message))
			return
		}

		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), response))
	}
}

func logoutHandler(issuer SessionIssuer) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if issuer == nil {
			c.JSON(consts.StatusServiceUnavailable, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "session issuer is unavailable"))
			return
		}

		accessToken := strings.TrimSpace(c.Request.Header.Get(authmiddleware.HeaderAuthorization))
		accessToken = strings.TrimSpace(strings.TrimPrefix(accessToken, "Bearer "))
		if err := issuer.Logout(ctx, accessToken); err != nil {
			c.JSON(consts.StatusInternalServerError, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "An internal error has occurred, please try again later or contact service support."))
			return
		}

		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func accountProfileHandler(ctx context.Context, c *app.RequestContext) {
	rc := contextx.MustFrom(ctx)
	c.JSON(consts.StatusOK, successEnvelope(newRequestID(), accountProfileResponse{
		AccountID: rc.AccountID,
		Username:  rc.Username,
		Email:     "",
		Type:      rc.AccountType,
	}))
}

func modelSelectorHandler(_ context.Context, c *app.RequestContext) {
	modelType := strings.TrimSpace(c.Param("modelType"))
	c.JSON(consts.StatusOK, successEnvelope(newRequestID(), modelSelectorForType(modelType)))
}

func apiModelsHandler(_ context.Context, c *app.RequestContext) {
	page := parsePositiveInt(c.Query("page"), 1)
	size := parsePositiveInt(c.Query("size"), 10)
	c.JSON(consts.StatusOK, successEnvelope(newRequestID(), adminPage[map[string]any]{
		TotalCount: 0,
		TotalPage:  0,
		PageNumber: int64(page),
		PageSize:   int64(size),
		PageItems:  []map[string]any{},
	}))
}

func setCORSHeaders(c *app.RequestContext) {
	origin := strings.TrimSpace(c.Request.Header.Get("Origin"))
	if origin == "" {
		return
	}
	c.Response.Header.Set("Access-Control-Allow-Origin", origin)
	c.Response.Header.Set("Vary", "Origin")
	c.Response.Header.Set("Access-Control-Allow-Credentials", "true")
	c.Response.Header.Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
	c.Response.Header.Set("Access-Control-Allow-Headers", "Authorization,X-SAA-TOKEN,X-API-Key,Content-Type,Accept")
}

func successEnvelope(requestID string, data any) apiEnvelope {
	return apiEnvelope{
		Code:      200,
		Message:   "success",
		Data:      data,
		RequestID: requestID,
	}
}

func errorEnvelope(requestID string, code string, message string) apiEnvelope {
	return apiEnvelope{
		Code:      code,
		Message:   message,
		RequestID: requestID,
	}
}

func mapSessionError(err error) (int, string, string) {
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials):
		return consts.StatusUnauthorized, "ACCOUNT_LOGIN_ERROR", "Login error, please check username and password."
	case errors.Is(err, auth.ErrInvalidRefreshToken):
		return consts.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "Refresh token is invalid."
	case errors.Is(err, auth.ErrDefaultWorkspaceNotFound):
		return consts.StatusNotFound, "DEFAULT_WORKSPACE_NOT_FOUND", "Default workspace can not be found."
	case errors.Is(err, auth.ErrAccountNotFound):
		return consts.StatusNotFound, "ACCOUNT_NOT_FOUND", "Account can not be found."
	default:
		return consts.StatusInternalServerError, "SYSTEM_ERROR", "An internal error has occurred, please try again later or contact service support."
	}
}

func modelSelectorForType(modelType string) []modelProviderGroup {
	if modelType != "llm" && modelType != "text_embedding" {
		return []modelProviderGroup{}
	}

	provider := providerConfigInfo{
		Name:                "OpenAI",
		Provider:            "OpenAI",
		Description:         "OpenAI",
		SupportedModelTypes: []string{"llm", "text_embedding"},
		Protocol:            "OpenAI",
		Enable:              true,
		Source:              "preset",
	}

	switch modelType {
	case "llm":
		return []modelProviderGroup{{
			Provider: provider,
			Models: []modelConfigInfo{
				{ModelID: "gpt-4.1", Name: "gpt-4.1", Provider: "OpenAI", Type: "llm", Mode: "chat", Enable: true, Source: "preset"},
				{ModelID: "gpt-4o-mini", Name: "gpt-4o-mini", Provider: "OpenAI", Type: "llm", Mode: "chat", Enable: true, Source: "preset"},
			},
		}}
	case "text_embedding":
		return []modelProviderGroup{{
			Provider: provider,
			Models: []modelConfigInfo{
				{ModelID: "text-embedding-3-small", Name: "text-embedding-3-small", Provider: "OpenAI", Type: "text_embedding", Mode: "chat", Enable: true, Source: "preset"},
			},
		}}
	default:
		return []modelProviderGroup{}
	}
}

func parsePositiveInt(value string, fallback int) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	var parsed int
	for _, r := range value {
		if r < '0' || r > '9' {
			return fallback
		}
		parsed = parsed*10 + int(r-'0')
	}
	if parsed <= 0 {
		return fallback
	}
	return parsed
}

func newRequestID() string {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return fmt.Sprintf("req_%d", time.Now().UnixNano())
	}
	return "req_" + hex.EncodeToString(data[:])
}

type apiEnvelope struct {
	Code      any    `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	RequestID string `json:"request_id"`
}

type globalConfigResponse struct {
	LoginMethod  string `json:"login_method"`
	UploadMethod string `json:"upload_method"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type refreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type accountProfileResponse struct {
	AccountID string `json:"account_id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Type      string `json:"type"`
	Logo      string `json:"logo,omitempty"`
}

type providerConfigInfo struct {
	Name                string   `json:"name"`
	Provider            string   `json:"provider"`
	Description         string   `json:"description,omitempty"`
	Icon                string   `json:"icon,omitempty"`
	SupportedModelTypes []string `json:"supported_model_types,omitempty"`
	Protocol            string   `json:"protocol,omitempty"`
	Enable              bool     `json:"enable,omitempty"`
	Source              string   `json:"source,omitempty"`
}

type modelConfigInfo struct {
	ModelID  string   `json:"model_id"`
	Name     string   `json:"name"`
	Provider string   `json:"provider"`
	Icon     string   `json:"icon,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Type     string   `json:"type"`
	Mode     string   `json:"mode,omitempty"`
	Enable   bool     `json:"enable,omitempty"`
	Source   string   `json:"source,omitempty"`
}

type modelProviderGroup struct {
	Provider providerConfigInfo `json:"provider"`
	Models   []modelConfigInfo  `json:"models"`
}

type adminPage[T any] struct {
	TotalCount int64 `json:"totalCount"`
	TotalPage  int64 `json:"totalPage"`
	PageNumber int64 `json:"pageNumber"`
	PageSize   int64 `json:"pageSize"`
	PageItems  []T   `json:"pageItems"`
}
