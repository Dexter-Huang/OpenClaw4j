package mcpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/sse"
)

type Route struct {
	Method  string
	Path    string
	Handler app.HandlerFunc
}

func Routes(service Service, broker *SSEBroker) []Route {
	return []Route{
		{Method: "POST", Path: "/mcp", Handler: func(ctx context.Context, c *app.RequestContext) {
			var req JSONRPCRequest
			if err := c.BindAndValidate(&req); err != nil {
				c.JSON(http.StatusOK, rpcError(nil, -32600, err.Error()))
				return
			}
			c.JSON(http.StatusOK, service.HandleJSONRPC(req))
		}},
		{Method: "GET", Path: "/mcp/sse", Handler: func(ctx context.Context, c *app.RequestContext) {
			session := broker.OpenSession()
			defer broker.CloseSession(session.ID)

			writer := sse.NewWriter(c)
			defer writer.Close()

			if err := writer.WriteEvent("", "endpoint", []byte(session.Endpoint)); err != nil {
				return
			}

			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case event, ok := <-session.Events:
					if !ok {
						return
					}
					if err := writer.WriteEvent(event.ID, event.Type, event.Data); err != nil {
						return
					}
				case <-ticker.C:
					if err := writer.WriteKeepAlive(); err != nil {
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}},
		{Method: "POST", Path: "/mcp/message", Handler: func(ctx context.Context, c *app.RequestContext) {
			sessionID := string(c.QueryArgs().Peek("session_id"))
			if sessionID == "" {
				sessionID = string(c.QueryArgs().Peek("sessionId"))
			}
			if sessionID == "" {
				c.JSON(http.StatusBadRequest, rpcError(nil, -32600, "missing SSE session id"))
				return
			}

			var req JSONRPCRequest
			if err := c.BindAndValidate(&req); err != nil {
				c.JSON(http.StatusBadRequest, rpcError(nil, -32600, err.Error()))
				return
			}
			if err := broker.Dispatch(sessionID, req); err != nil {
				status := http.StatusInternalServerError
				if errors.Is(err, ErrSSESessionNotFound) {
					status = http.StatusNotFound
				}
				if errors.Is(err, ErrSSESessionBackpressure) {
					status = http.StatusTooManyRequests
				}
				c.JSON(status, rpcError(req.ID, -32000, err.Error()))
				return
			}
			c.Status(http.StatusAccepted)
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
	broker := NewSSEBroker(service)
	for _, route := range Routes(service, broker) {
		switch route.Method {
		case "GET":
			h.GET(route.Path, route.Handler)
		case "POST":
			h.POST(route.Path, route.Handler)
		}
	}
}
