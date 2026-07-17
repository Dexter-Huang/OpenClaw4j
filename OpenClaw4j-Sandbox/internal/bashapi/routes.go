package bashapi

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
)

type Route struct {
	Method  string
	Path    string
	Handler app.HandlerFunc
}

func Routes(service *Service) []Route {
	return []Route{
		{Method: "POST", Path: "/v1/bash/exec", Handler: bindHandler(service.Exec)},
		{Method: "POST", Path: "/v1/bash/output", Handler: bindHandler(service.Output)},
		{Method: "POST", Path: "/v1/bash/write", Handler: bindHandler(service.Write)},
		{Method: "POST", Path: "/v1/bash/kill", Handler: bindHandler(service.Kill)},
		{Method: "GET", Path: "/v1/bash/sessions", Handler: func(ctx context.Context, c *app.RequestContext) {
			c.JSON(http.StatusOK, service.Sessions())
		}},
		{Method: "POST", Path: "/v1/bash/sessions/create", Handler: bindHandler(service.CreateSession)},
		{Method: "POST", Path: "/v1/bash/sessions/:session_id/close", Handler: func(ctx context.Context, c *app.RequestContext) {
			c.JSON(http.StatusOK, service.CloseSession(c.Param("session_id")))
		}},
	}
}

func RegisterRoutes(h *server.Hertz, service *Service) {
	for _, route := range Routes(service) {
		switch route.Method {
		case "GET":
			h.GET(route.Path, route.Handler)
		case "POST":
			h.POST(route.Path, route.Handler)
		}
	}
}

func bindHandler[T any](handler func(T) Response) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		var req T
		if err := c.BindAndValidate(&req); err != nil {
			c.JSON(http.StatusOK, Response{
				Success: false,
				Message: err.Error(),
				Data: map[string]any{
					"error_type": "invalid_request",
					"message":    err.Error(),
					"retryable":  false,
				},
			})
			return
		}
		c.JSON(http.StatusOK, handler(req))
	}
}
