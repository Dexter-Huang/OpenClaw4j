package mcpapi

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
		{Method: "POST", Path: "/mcp", Handler: func(ctx context.Context, c *app.RequestContext) {
			var req JSONRPCRequest
			if err := c.BindAndValidate(&req); err != nil {
				c.JSON(http.StatusOK, rpcError(nil, -32600, err.Error()))
				return
			}
			c.JSON(http.StatusOK, service.HandleJSONRPC(req))
		}},
		{Method: "GET", Path: "/v1/mcp/servers", Handler: func(ctx context.Context, c *app.RequestContext) {
			c.JSON(http.StatusOK, map[string]any{"success": true, "data": []string{"sandbox"}})
		}},
		{Method: "GET", Path: "/v1/mcp/:server_name/tools", Handler: func(ctx context.Context, c *app.RequestContext) {
			if c.Param("server_name") != "sandbox" {
				c.JSON(http.StatusOK, map[string]any{"success": false, "message": "unknown MCP server"})
				return
			}
			c.JSON(http.StatusOK, map[string]any{"success": true, "data": map[string]any{"tools": ListTools()}})
		}},
		{Method: "POST", Path: "/v1/mcp/:server_name/tools/:tool_name", Handler: func(ctx context.Context, c *app.RequestContext) {
			if c.Param("server_name") != "sandbox" {
				c.JSON(http.StatusOK, map[string]any{"success": false, "message": "unknown MCP server"})
				return
			}
			var args map[string]any
			if err := c.BindAndValidate(&args); err != nil {
				c.JSON(http.StatusOK, map[string]any{"success": false, "message": err.Error()})
				return
			}
			c.JSON(http.StatusOK, service.CallTool(c.Param("tool_name"), args))
		}},
	}
}

func RegisterRoutes(h *server.Hertz, service Service) {
	for _, route := range Routes(service) {
		switch route.Method {
		case "GET":
			h.GET(route.Path, route.Handler)
		case "POST":
			h.POST(route.Path, route.Handler)
		}
	}
}
