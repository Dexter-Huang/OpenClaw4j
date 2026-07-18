package middleware

import (
	"context"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/seaskyland/openclaw4j-backend-go/internal/contextx"
)

const (
	HeaderAuthorization = "Authorization"
	HeaderXSaaToken     = "X-SAA-TOKEN"
	HeaderXAPIKey       = "X-API-Key"

	QueryAccessToken = "access_token"
	tokenPrefix      = "Bearer "
)

type Authenticator interface {
	AuthenticateConsoleToken(ctx context.Context, token string, callerIP string) (*contextx.RequestContext, error)
	AuthenticateAPIKey(ctx context.Context, apiKey string, callerIP string) (*contextx.RequestContext, error)
}

func ConsoleAuth(authenticator Authenticator) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if isOptions(c) {
			c.Next(ctx)
			return
		}
		token, ok := consoleToken(c)
		if !ok || authenticator == nil {
			abortUnauthorized(c, "INVALID_TOKEN")
			return
		}
		rc, err := authenticator.AuthenticateConsoleToken(ctx, token, callerIP(c))
		if err != nil || rc == nil {
			abortUnauthorized(c, "INVALID_TOKEN")
			return
		}
		c.Next(contextx.With(ctx, rc))
	}
}

func APIKeyAuth(authenticator Authenticator) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if isOptions(c) {
			c.Next(ctx)
			return
		}
		apiKey, ok := apiKey(c)
		if !ok || authenticator == nil {
			abortUnauthorized(c, "INVALID_API_KEY")
			return
		}
		rc, err := authenticator.AuthenticateAPIKey(ctx, apiKey, callerIP(c))
		if err != nil || rc == nil {
			abortUnauthorized(c, "INVALID_API_KEY")
			return
		}
		c.Next(contextx.With(ctx, rc))
	}
}

func consoleToken(c *app.RequestContext) (string, bool) {
	header := strings.TrimSpace(c.Request.Header.Get(HeaderXSaaToken))
	if header != "" {
		return bearerValue(header)
	}
	queryToken := strings.TrimSpace(c.Query(QueryAccessToken))
	if queryToken == "" {
		return "", false
	}
	return strings.TrimPrefix(queryToken, tokenPrefix), true
}

func apiKey(c *app.RequestContext) (string, bool) {
	authorization := strings.TrimSpace(c.Request.Header.Get(HeaderAuthorization))
	if authorization != "" {
		return bearerValue(authorization)
	}
	xAPIKey := strings.TrimSpace(c.Request.Header.Get(HeaderXAPIKey))
	if xAPIKey == "" {
		return "", false
	}
	return xAPIKey, true
}

func bearerValue(value string) (string, bool) {
	if !strings.HasPrefix(value, tokenPrefix) {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(value, tokenPrefix))
	return token, token != ""
}

func isOptions(c *app.RequestContext) bool {
	return string(c.Request.Header.Method()) == consts.MethodOptions
}

func callerIP(c *app.RequestContext) string {
	if realIP := strings.TrimSpace(c.Request.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	if forwardedFor := strings.TrimSpace(c.Request.Header.Get("X-Forwarded-For")); forwardedFor != "" {
		return strings.TrimSpace(strings.Split(forwardedFor, ",")[0])
	}
	return c.ClientIP()
}

func abortUnauthorized(c *app.RequestContext, code string) {
	c.AbortWithStatusJSON(consts.StatusUnauthorized, map[string]any{
		"success": false,
		"code":    code,
	})
}
