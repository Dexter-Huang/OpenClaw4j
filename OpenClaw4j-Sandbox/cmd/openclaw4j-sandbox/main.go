package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/seaskyland/openclaw4j-sandbox/internal/config"
	"github.com/seaskyland/openclaw4j-sandbox/internal/executor"
	"github.com/seaskyland/openclaw4j-sandbox/internal/fileapi"
	"github.com/seaskyland/openclaw4j-sandbox/internal/model"
	"github.com/seaskyland/openclaw4j-sandbox/internal/pathguard"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := config.FromEnv()
	exec := executor.New(cfg, defaultRuntime())
	guard := pathguard.New(cfg.WorkspaceDir)
	if err := guard.EnsureWorkspace(); err != nil {
		slog.Error("failed to prepare sandbox workspace", "workspace", cfg.WorkspaceDir, "error", err)
		os.Exit(1)
	}
	fileService := fileapi.NewService(guard, cfg.FileReadLimitBytes)
	h := server.Default(server.WithHostPorts(cfg.Bind))

	h.GET("/health", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(http.StatusOK, utils.H{"status": "UP"})
	})

	h.POST("/v1/execute", func(ctx context.Context, c *app.RequestContext) {
		var req model.ExecuteRequest
		if err := c.BindAndValidate(&req); err != nil {
			res := model.Error("INVALID_REQUEST", err.Error(), "", "", nil, 0)
			c.JSON(http.StatusOK, res)
			return
		}
		c.JSON(http.StatusOK, exec.Execute(ctx, req))
	})

	fileapi.RegisterRoutes(h, fileService)

	slog.Info("OpenClaw4j sandbox listening", "bind", cfg.Bind, "workspace", cfg.WorkspaceDir)
	h.Spin()
}
