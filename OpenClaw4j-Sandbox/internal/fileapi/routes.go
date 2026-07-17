package fileapi

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

func Routes(service Service) []Route {
	return []Route{
		{Method: "POST", Path: "/v1/file/read", Handler: bindHandler(service.Read)},
		{Method: "POST", Path: "/v1/file/write", Handler: bindHandler(service.Write)},
		{Method: "POST", Path: "/v1/file/replace", Handler: bindHandler(service.Replace)},
		{Method: "POST", Path: "/v1/file/list", Handler: bindHandler(service.List)},
	}
}

func RegisterRoutes(h *server.Hertz, service Service) {
	for _, route := range Routes(service) {
		if route.Method == "POST" {
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
