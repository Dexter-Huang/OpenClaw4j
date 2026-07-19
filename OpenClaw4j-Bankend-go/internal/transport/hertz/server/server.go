package server

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/seaskyland/openclaw4j-backend-go/internal/account"
	"github.com/seaskyland/openclaw4j-backend-go/internal/agentschema"
	"github.com/seaskyland/openclaw4j-backend-go/internal/apikey"
	"github.com/seaskyland/openclaw4j-backend-go/internal/appcomponent"
	"github.com/seaskyland/openclaw4j-backend-go/internal/application"
	"github.com/seaskyland/openclaw4j-backend-go/internal/auth"
	"github.com/seaskyland/openclaw4j-backend-go/internal/chat"
	"github.com/seaskyland/openclaw4j-backend-go/internal/compat"
	"github.com/seaskyland/openclaw4j-backend-go/internal/contextx"
	"github.com/seaskyland/openclaw4j-backend-go/internal/document"
	"github.com/seaskyland/openclaw4j-backend-go/internal/knowledgebase"
	"github.com/seaskyland/openclaw4j-backend-go/internal/mcpserver"
	"github.com/seaskyland/openclaw4j-backend-go/internal/modelprovider"
	"github.com/seaskyland/openclaw4j-backend-go/internal/plugin"
	"github.com/seaskyland/openclaw4j-backend-go/internal/skill"
	"github.com/seaskyland/openclaw4j-backend-go/internal/tool"
	authmiddleware "github.com/seaskyland/openclaw4j-backend-go/internal/transport/hertz/middleware"
	"github.com/seaskyland/openclaw4j-backend-go/internal/workflow"
	"github.com/seaskyland/openclaw4j-backend-go/internal/workspace"
)

type Options struct {
	HertzOptions           []config.Option
	Authenticator          authmiddleware.Authenticator
	SessionIssuer          SessionIssuer
	AccountProfileProvider AccountProfileProvider
	APIKeyManager          APIKeyManager
	WorkspaceManager       WorkspaceManager
	AccountManager         AccountManager
	ModelProviderManager   ModelProviderManager
	ApplicationManager     ApplicationManager
	AppComponentManager    AppComponentManager
	AgentSchemaManager     AgentSchemaManager
	ToolManager            ToolManager
	FileStorageDir         string
	KnowledgeBaseManager   KnowledgeBaseManager
	DocumentManager        DocumentManager
	McpServerManager       McpServerManager
	SkillManager           SkillManager
	PluginManager          PluginManager
	ChatManager            ChatManager
	WorkflowManager        WorkflowManager
	OAuth2Provider         OAuth2Provider
	LegacyAdminManager     *compat.AdminService
	ObservabilityManager   *compat.ObservabilityService
	RegisterConsoleRoutes  func(group *route.RouterGroup)
	RegisterAPIRoutes      func(group *route.RouterGroup)
}

type SessionIssuer interface {
	Login(ctx context.Context, username string, password string, callerIP string, userAgent string) (*auth.TokenResponse, error)
	RefreshToken(ctx context.Context, refreshToken string, callerIP string, userAgent string) (*auth.TokenResponse, error)
	Logout(ctx context.Context, accessToken string) error
}

// OAuth2Provider 将外部授权码转换为认证层可消费的用户信息。
type OAuth2Provider interface {
	AuthorizationURL() (string, error)
	Authenticate(ctx context.Context, code string) (auth.OAuthUser, error)
}

type OAuth2SessionIssuer interface {
	LoginOAuth(ctx context.Context, user auth.OAuthUser, callerIP string, userAgent string) (*auth.TokenResponse, error)
}

type AccountProfileProvider interface {
	GetAccountProfile(ctx context.Context, accountID string) (*auth.AccountProfile, error)
}

type ChatManager interface {
	Complete(context.Context, string, chat.Request) (*chat.Response, error)
}

// ChatStreamer is optional so integrations that only support request/response chat
// remain source compatible. The production agent service implements it.
type ChatStreamer interface {
	CompleteStream(context.Context, string, chat.Request) (<-chan chat.StreamEvent, error)
}

// WorkflowManager 覆盖 Java WorkflowController 使用的调试任务生命周期。
// HTTP 层只负责身份范围和协议封装，图解析、节点状态和任务缓存全部归 workflow 服务管理。
type WorkflowManager interface {
	Init(context.Context, string, workflow.InitRequest) ([]workflow.Param, error)
	Run(context.Context, string, string, workflow.TaskRunRequest) (*workflow.TaskRunResponse, error)
	RunFragment(context.Context, string, string, workflow.FragmentRequest) (*workflow.TaskRunResponse, error)
	Resume(context.Context, string, string, workflow.ResumeRequest) (*workflow.TaskRunResponse, error)
	Stop(string, string) (bool, error)
	Process(string, string) (*workflow.ProcessResponse, error)
	AsyncResult(string, string) (*workflow.AsyncResponse, error)
	StartCompletion(context.Context, string, string, workflow.CompletionRequest) (*workflow.TaskRunResponse, error)
	Complete(context.Context, string, string, workflow.CompletionRequest) (*workflow.CompletionResponse, error)
}

// WorkflowStreamer 是 WorkflowManager 的可选扩展。拆分接口可保持自定义 manager 的
// 源码兼容，同时让内置 Go 工作流服务在节点状态产生时立即推送。
type WorkflowStreamer interface {
	RunStream(context.Context, string, string, workflow.TaskRunRequest) (*workflow.TaskRunResponse, <-chan workflow.StreamEvent, func(), error)
	StartCompletionStream(context.Context, string, string, workflow.CompletionRequest) (*workflow.TaskRunResponse, <-chan workflow.StreamEvent, func(), error)
}

type APIKeyManager interface {
	Create(ctx context.Context, accountID string, description string) (int64, error)
	Get(ctx context.Context, accountID string, id int64) (*apikey.APIKey, error)
	List(ctx context.Context, accountID string, current int64, size int64) (*apikey.Page, error)
	UpdateDescription(ctx context.Context, accountID string, id int64, description string) error
	Delete(ctx context.Context, accountID string, id int64) error
}

type WorkspaceManager interface {
	Create(ctx context.Context, accountID string, name string, description *string, config *string) (string, error)
	Get(ctx context.Context, accountID string, workspaceID string) (*workspace.Workspace, error)
	List(ctx context.Context, accountID string, current int64, size int64) (*workspace.Page, error)
	Update(ctx context.Context, accountID string, workspaceID string, name string, description *string, config *string) error
	Delete(ctx context.Context, accountID string, workspaceID string) error
}

type AccountManager interface {
	Create(ctx context.Context, operatorID string, input account.Input) (string, error)
	Update(ctx context.Context, operatorID string, accountID string, input account.Input) error
	Delete(ctx context.Context, operatorID string, accountID string) error
	Get(ctx context.Context, accountID string) (*account.Account, error)
	List(ctx context.Context, operatorID string, name string, current int64, size int64) (*account.Page, error)
	ChangePassword(ctx context.Context, accountID string, password string, newPassword string) error
}

type ModelProviderManager interface {
	CreateProvider(ctx context.Context, workspaceID, accountID string, input modelprovider.CreateProviderInput) (string, error)
	UpdateProvider(ctx context.Context, workspaceID, accountID, providerCode string, input modelprovider.UpdateProviderInput) error
	DeleteProvider(ctx context.Context, workspaceID, providerCode string) error
	ListProviders(ctx context.Context, workspaceID, name string) ([]modelprovider.Provider, error)
	GetProvider(ctx context.Context, workspaceID, providerCode string) (*modelprovider.Provider, error)
	CreateModel(ctx context.Context, workspaceID, accountID, providerCode string, input modelprovider.CreateModelInput) error
	UpdateModel(ctx context.Context, workspaceID, accountID, providerCode, modelID string, input modelprovider.UpdateModelInput) error
	DeleteModel(ctx context.Context, workspaceID, providerCode, modelID string) error
	ListModels(ctx context.Context, workspaceID, providerCode string) ([]modelprovider.Model, error)
	GetModel(ctx context.Context, workspaceID, providerCode, modelID string) (*modelprovider.Model, error)
	Selector(ctx context.Context, workspaceID, modelType string) ([]modelprovider.SelectorGroup, error)
	ParameterRules(ctx context.Context, workspaceID, providerCode, modelID string) ([]map[string]any, error)
}

type ApplicationManager interface {
	Create(ctx context.Context, workspaceID, accountID string, input application.Input) (string, error)
	Update(ctx context.Context, workspaceID, accountID, appID string, input application.Input) error
	Delete(ctx context.Context, workspaceID, accountID, appID string) error
	Get(ctx context.Context, workspaceID, appID string) (*application.Application, error)
	List(ctx context.Context, workspaceID, name, appType, status string, current, size int64) (*application.Page[application.Application], error)
	Publish(ctx context.Context, workspaceID, accountID, appID string) error
	ListVersions(ctx context.Context, workspaceID, appID, status string, current, size int64) (*application.Page[application.Version], error)
	GetVersion(ctx context.Context, workspaceID, appID, version string) (*application.Version, error)
	Copy(ctx context.Context, workspaceID, accountID, appID string) (string, error)
}
type AppComponentManager interface {
	Create(context.Context, string, string, appcomponent.Input) (string, error)
	Update(context.Context, string, string, string, appcomponent.Input) error
	Delete(context.Context, string, string, string) error
	Get(context.Context, string, string) (*appcomponent.Component, error)
	GetByAppID(context.Context, string, string) (*appcomponent.Component, error)
	List(context.Context, string, string, string, string, int64, int64, int64) (*appcomponent.Page[appcomponent.Component], error)
	ListByCodes(context.Context, string, []string) ([]appcomponent.Component, error)
	Publishable(context.Context, string, string, string, int64, int64) (*appcomponent.Page[application.Application], error)
	QueryConfig(context.Context, string, string) (*appcomponent.Component, error)
	Refer(context.Context, string, string) ([]appcomponent.Component, error)
	Schema(context.Context, string, string) (map[string]any, error)
	Schemas(context.Context, string, []string) (map[string]any, error)
}

type AgentSchemaManager interface {
	Create(ctx context.Context, workspaceID, accountID string, input agentschema.Input) (*agentschema.Schema, error)
	Get(ctx context.Context, workspaceID string, id int64) (*agentschema.Schema, error)
	List(ctx context.Context, workspaceID, name string) ([]agentschema.Schema, error)
	Page(ctx context.Context, workspaceID, name string, current, size int64) (*agentschema.Page, error)
	Update(ctx context.Context, workspaceID, accountID string, id int64, input agentschema.Input) (*agentschema.Schema, error)
	SetEnabled(ctx context.Context, workspaceID, accountID string, id int64, enabled bool) error
	Delete(ctx context.Context, workspaceID string, id int64) error
}

type ToolManager interface {
	Create(context.Context, string, string, tool.Input) (*tool.Tool, error)
	Get(context.Context, string, int64) (*tool.Tool, error)
	List(context.Context, string, string, string) ([]tool.Tool, error)
	Page(context.Context, string, string, string, int64, int64) (*tool.Page, error)
	Update(context.Context, string, string, int64, tool.Input) (*tool.Tool, error)
	SetEnabled(context.Context, string, string, int64, bool) error
	Delete(context.Context, string, int64) error
}
type KnowledgeBaseManager interface {
	Create(context.Context, string, string, knowledgebase.Input) (string, error)
	Get(context.Context, string, string) (*knowledgebase.KnowledgeBase, error)
	List(context.Context, string, string, int64, int64) (*knowledgebase.Page, error)
	Update(context.Context, string, string, string, knowledgebase.Input) error
	Delete(context.Context, string, string, string) error
	ListByCodes(context.Context, string, []string) ([]knowledgebase.KnowledgeBase, error)
}
type DocumentManager interface {
	Create(context.Context, string, string, string, document.CreateInput) ([]string, error)
	Get(context.Context, string, string, string) (*document.Document, error)
	List(context.Context, string, string, string, int64, int64, int64) (*document.Page, error)
	Update(context.Context, string, string, string, string, document.UpdateInput) error
	ReIndex(context.Context, string, string, string, string, document.ReIndexInput) error
	Delete(context.Context, string, string, string, string) error
	DeleteBatch(context.Context, string, string, string, []string) error
	CreateChunk(context.Context, string, string, string, document.ChunkInput) (string, error)
	UpdateChunk(context.Context, string, string, string, string, document.ChunkInput) error
	DeleteChunks(context.Context, string, string, string, []string) error
	ListChunks(context.Context, string, string, int64, int64) (*document.PageChunks, error)
	PreviewChunks(context.Context, string, string) ([]document.Chunk, error)
	SetChunksEnabled(context.Context, string, string, string, document.ChunkStatusInput) error
}
type McpServerManager interface {
	Create(context.Context, string, string, mcpserver.Input) (string, error)
	Update(context.Context, string, mcpserver.Input) error
	Get(context.Context, string, string, bool) (*mcpserver.Server, error)
	List(context.Context, string, string, bool, int64, int64) (*mcpserver.Page, error)
	ListByCodes(context.Context, string, []string, bool) ([]mcpserver.Server, error)
	Delete(context.Context, string, string) error
	CallTool(context.Context, string, mcpserver.ToolCallInput) (*mcpserver.ToolCallResult, error)
}
type SkillManager interface {
	Create(context.Context, string, string, skill.Input) (string, error)
	Update(context.Context, string, string, skill.Input) error
	Publish(context.Context, string, string, string) error
	Delete(context.Context, string, string, string) error
	Get(context.Context, string, string, string, bool) (*skill.Detail, error)
	List(context.Context, string, string, int16, int64, int64) (*skill.Page, error)
	ListByCodes(context.Context, string, []string, bool) ([]skill.Detail, error)
}
type PluginManager interface {
	Create(context.Context, string, string, plugin.PluginInput) (string, error)
	Update(context.Context, string, string, string, plugin.PluginInput) error
	Delete(context.Context, string, string) error
	Get(context.Context, string, string) (*plugin.Plugin, error)
	List(context.Context, string, string, int64, int64) (*plugin.Page[plugin.Plugin], error)
	CreateTool(context.Context, string, string, string, plugin.ToolInput) (string, error)
	UpdateTool(context.Context, string, string, string, string, plugin.ToolInput) error
	DeleteTool(context.Context, string, string, string) error
	GetTool(context.Context, string, string, string) (*plugin.Tool, error)
	ListTools(context.Context, string, string, string, int64, int64) (*plugin.Page[plugin.Tool], error)
	ListToolsByIDs(context.Context, string, []string) ([]plugin.Tool, error)
	SetEnabled(context.Context, string, string, string, bool) error
	TestTool(context.Context, string, string, string, string, map[string]any) (*plugin.TestResult, error)
	PublishTool(context.Context, string, string, string, string) error
}

func New(options Options) *server.Hertz {
	h := server.New(options.HertzOptions...)
	h.Use(corsMiddleware())
	registerCORSPreflightRoute(h)
	registerHealthRoute(h)
	registerPublicExampleRoutes(h)
	registerPublicConsoleRoutes(h, options)
	registerLegacyCompatibilityRoutes(h, options)
	registerConsoleGroup(h, options)
	registerPublicAPIRoutes(h, options)
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
	h.Engine.Handle(consts.MethodGet, "/oauth2/login/github", githubOAuthLoginHandler(options.OAuth2Provider))
	h.Engine.Handle(consts.MethodGet, "/oauth2/callback/github", githubOAuthCallbackHandler(options.OAuth2Provider, options.SessionIssuer))
	h.Engine.Handle(consts.MethodGet, "/console/v1/system/global-config", func(_ context.Context, c *app.RequestContext) {
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), globalConfigResponse{
			LoginMethod:  "preset_account",
			UploadMethod: "file",
		}))
	})
	h.Engine.Handle(consts.MethodGet, "/console/v1/system/health", func(_ context.Context, c *app.RequestContext) {
		c.String(consts.StatusOK, "ok")
	})
}

func registerConsoleGroup(h *server.Hertz, options Options) {
	console := h.Group("/console/v1")
	console.Use(authmiddleware.ConsoleAuth(options.Authenticator))
	console.GET("/accounts/profile", accountProfileHandler(options.AccountProfileProvider))
	console.POST("/api-keys", apiKeyCreateHandler(options.APIKeyManager))
	console.GET("/api-keys", apiKeyListHandler(options.APIKeyManager))
	console.GET("/api-keys/:id", apiKeyGetHandler(options.APIKeyManager))
	console.PUT("/api-keys/:id", apiKeyUpdateHandler(options.APIKeyManager))
	console.DELETE("/api-keys/:id", apiKeyDeleteHandler(options.APIKeyManager))
	console.POST("/workspaces", workspaceCreateHandler(options.WorkspaceManager))
	console.GET("/workspaces", workspaceListHandler(options.WorkspaceManager))
	console.GET("/workspaces/:workspaceId", workspaceGetHandler(options.WorkspaceManager))
	console.PUT("/workspaces/:workspaceId", workspaceUpdateHandler(options.WorkspaceManager))
	console.DELETE("/workspaces/:workspaceId", workspaceDeleteHandler(options.WorkspaceManager))
	console.POST("/accounts", accountCreateHandler(options.AccountManager))
	console.GET("/accounts", accountListHandler(options.AccountManager))
	console.GET("/accounts/:accountId", accountGetHandler(options.AccountManager))
	console.PUT("/accounts/:accountId", accountUpdateHandler(options.AccountManager))
	console.DELETE("/accounts/:accountId", accountDeleteHandler(options.AccountManager))
	console.PUT("/accounts/change-password", accountChangePasswordHandler(options.AccountManager))
	console.POST("/providers", providerCreateHandler(options.ModelProviderManager))
	console.GET("/providers/protocols", providerProtocolsHandler)
	console.GET("/providers", providerListHandler(options.ModelProviderManager))
	console.GET("/providers/:provider", providerGetHandler(options.ModelProviderManager))
	console.PUT("/providers/:provider", providerUpdateHandler(options.ModelProviderManager))
	console.DELETE("/providers/:provider", providerDeleteHandler(options.ModelProviderManager))
	console.POST("/providers/:provider/models", providerModelCreateHandler(options.ModelProviderManager))
	console.GET("/providers/:provider/models", providerModelListHandler(options.ModelProviderManager))
	console.GET("/providers/:provider/models/:modelId", providerModelGetHandler(options.ModelProviderManager))
	console.PUT("/providers/:provider/models/:modelId", providerModelUpdateHandler(options.ModelProviderManager))
	console.DELETE("/providers/:provider/models/:modelId", providerModelDeleteHandler(options.ModelProviderManager))
	console.GET("/providers/:provider/models/:modelId/parameter_rules", providerModelParameterRulesHandler(options.ModelProviderManager))
	console.GET("/models/:modelType/selector", modelSelectorHandler(options.ModelProviderManager))
	console.POST("/apps", applicationCreateHandler(options.ApplicationManager))
	console.GET("/apps", applicationListHandler(options.ApplicationManager))
	console.GET("/apps/:appId", applicationGetHandler(options.ApplicationManager))
	console.PUT("/apps/:appId", applicationUpdateHandler(options.ApplicationManager))
	console.DELETE("/apps/:appId", applicationDeleteHandler(options.ApplicationManager))
	console.POST("/apps/:appId/publish", applicationPublishHandler(options.ApplicationManager))
	console.POST("/apps/:appId/copy", applicationCopyHandler(options.ApplicationManager))
	console.GET("/apps/:appId/versions", applicationVersionListHandler(options.ApplicationManager))
	console.GET("/apps/:appId/versions/:version", applicationVersionGetHandler(options.ApplicationManager))
	console.GET("/component-servers", appComponentListHandler(options.AppComponentManager))
	console.GET("/component-servers/app-publishable", appComponentPublishableHandler(options.AppComponentManager))
	console.POST("/component-servers", appComponentCreateHandler(options.AppComponentManager))
	console.POST("/component-servers/query-by-codes", appComponentByCodesHandler(options.AppComponentManager))
	console.POST("/component-servers/schema-by-codes", appComponentSchemasHandler(options.AppComponentManager))
	console.GET("/component-servers/:code/detail-by-code", appComponentGetHandler(options.AppComponentManager))
	console.GET("/component-servers/:appId/detail-by-appid", appComponentGetByAppIDHandler(options.AppComponentManager))
	console.GET("/component-servers/:code/query-refer", appComponentReferHandler(options.AppComponentManager))
	console.GET("/component-servers/:appId/query-config", appComponentConfigHandler(options.AppComponentManager))
	console.GET("/component-servers/:code/query-schema", appComponentSchemaHandler(options.AppComponentManager))
	console.PUT("/component-servers/:code", appComponentUpdateHandler(options.AppComponentManager))
	console.DELETE("/component-servers/:code", appComponentDeleteHandler(options.AppComponentManager))
	console.POST("/agent-schemas", agentSchemaCreateHandler(options.AgentSchemaManager))
	console.GET("/agent-schemas/page", agentSchemaPageHandler(options.AgentSchemaManager))
	console.GET("/agent-schemas/search", agentSchemaSearchHandler(options.AgentSchemaManager))
	console.GET("/agent-schemas", agentSchemaListHandler(options.AgentSchemaManager))
	console.GET("/agent-schemas/:id", agentSchemaGetHandler(options.AgentSchemaManager))
	console.PUT("/agent-schemas/:id", agentSchemaUpdateHandler(options.AgentSchemaManager))
	console.PATCH("/agent-schemas/:id/enabled", agentSchemaEnabledHandler(options.AgentSchemaManager))
	console.DELETE("/agent-schemas/:id", agentSchemaDeleteHandler(options.AgentSchemaManager))
	console.POST("/tools", toolCreateHandler(options.ToolManager))
	console.GET("/tools/page", toolPageHandler(options.ToolManager))
	console.GET("/tools/search", toolSearchHandler(options.ToolManager))
	console.GET("/tools/plugin/:pluginId", toolPluginListHandler(options.ToolManager))
	console.GET("/tools", toolListHandler(options.ToolManager))
	console.GET("/tools/:id", toolGetHandler(options.ToolManager))
	console.PUT("/tools/:id", toolUpdateHandler(options.ToolManager))
	console.PATCH("/tools/:id/enabled", toolEnabledHandler(options.ToolManager))
	console.DELETE("/tools/:id", toolDeleteHandler(options.ToolManager))
	console.POST("/files/upload", fileUploadHandler(options.FileStorageDir))
	console.POST("/files/upload-policies", fileUploadPoliciesHandler(options.FileStorageDir))
	console.GET("/files/download", fileDownloadHandler(options.FileStorageDir))
	console.GET("/files/get-preview-url", filePreviewURLHandler)
	console.POST("/knowledge-bases", knowledgeBaseCreateHandler(options.KnowledgeBaseManager))
	console.GET("/knowledge-bases", knowledgeBaseListHandler(options.KnowledgeBaseManager))
	console.GET("/knowledge-bases/:kbId", knowledgeBaseGetHandler(options.KnowledgeBaseManager))
	console.PUT("/knowledge-bases/:kbId", knowledgeBaseUpdateHandler(options.KnowledgeBaseManager))
	console.DELETE("/knowledge-bases/:kbId", knowledgeBaseDeleteHandler(options.KnowledgeBaseManager))
	console.POST("/knowledge-bases/query-by-codes", knowledgeBaseListByCodesHandler(options.KnowledgeBaseManager))
	console.POST("/knowledge-bases/retrieve", knowledgeBaseRetrieveHandler(options.KnowledgeBaseManager))
	console.POST("/knowledge-bases/:kbId/documents", documentCreateHandler(options.DocumentManager))
	console.GET("/knowledge-bases/:kbId/documents", documentListHandler(options.DocumentManager))
	console.DELETE("/knowledge-bases/:kbId/documents/batch-delete", documentBatchDeleteHandler(options.DocumentManager))
	console.GET("/knowledge-bases/:kbId/documents/:docId", documentGetHandler(options.DocumentManager))
	console.PUT("/knowledge-bases/:kbId/documents/:docId", documentUpdateHandler(options.DocumentManager))
	console.DELETE("/knowledge-bases/:kbId/documents/:docId", documentDeleteHandler(options.DocumentManager))
	console.PUT("/knowledge-bases/:kbId/documents/:docId/re-index", documentReIndexHandler(options.DocumentManager))
	console.POST("/documents/:docId/chunks", documentChunkCreateHandler(options.DocumentManager))
	console.GET("/documents/:docId/chunks", documentChunkListHandler(options.DocumentManager))
	console.DELETE("/documents/:docId/chunks/batch-delete", documentChunkBatchDeleteHandler(options.DocumentManager))
	console.POST("/documents/:docId/chunks/preview", documentChunkPreviewHandler(options.DocumentManager))
	console.PUT("/documents/:docId/chunks/update-status", documentChunkStatusHandler(options.DocumentManager))
	console.PUT("/documents/:docId/chunks/:chunkId", documentChunkUpdateHandler(options.DocumentManager))
	console.DELETE("/documents/:docId/chunks/:chunkId", documentChunkDeleteHandler(options.DocumentManager))
	console.POST("/mcp-servers", mcpServerCreateHandler(options.McpServerManager))
	console.PUT("/mcp-servers", mcpServerUpdateHandler(options.McpServerManager))
	console.DELETE("/mcp-servers/:serverCode", mcpServerDeleteHandler(options.McpServerManager))
	console.GET("/mcp-servers/:serverCode", mcpServerGetHandler(options.McpServerManager))
	console.GET("/mcp-servers", mcpServerListHandler(options.McpServerManager))
	console.POST("/mcp-servers/query-by-codes", mcpServerListByCodesHandler(options.McpServerManager))
	console.POST("/mcp-servers/debug-tools", mcpServerDebugToolHandler(options.McpServerManager))
	console.POST("/apps/chat/completions", chatCompletionHandler(options.ChatManager))
	console.POST("/apps/workflow/debug/init", workflowInitHandler(options.WorkflowManager))
	console.POST("/apps/workflow/debug/run-task", workflowRunHandler(options.WorkflowManager))
	console.POST("/apps/workflow/debug/get-task-process", workflowProcessHandler(options.WorkflowManager))
	console.POST("/apps/workflow/debug/resume-task", workflowResumeHandler(options.WorkflowManager))
	console.POST("/apps/workflow/debug/part-graph/run-task", workflowFragmentHandler(options.WorkflowManager))
	console.POST("/apps/workflow/debug/part-graph/stop-task", workflowStopHandler(options.WorkflowManager))
	console.POST("/apps/workflow/:appId/run_stream", workflowRunStreamHandler(options.WorkflowManager))
	console.POST("/skills", skillCreateHandler(options.SkillManager))
	console.POST("/skills/package", skillPackageUploadHandler(options.FileStorageDir))
	console.PUT("/skills", skillUpdateHandler(options.SkillManager))
	console.DELETE("/skills/:skillCode", skillDeleteHandler(options.SkillManager))
	console.POST("/skills/:skillCode/publish", skillPublishHandler(options.SkillManager))
	console.GET("/skills/:skillCode", skillGetHandler(options.SkillManager))
	console.GET("/skills", skillListHandler(options.SkillManager))
	console.POST("/skills/query-by-codes", skillListByCodesHandler(options.SkillManager))
	console.POST("/plugins", pluginCreateHandler(options.PluginManager))
	console.GET("/plugins", pluginListHandler(options.PluginManager))
	console.GET("/plugins/:pluginId", pluginGetHandler(options.PluginManager))
	console.PUT("/plugins/:pluginId", pluginUpdateHandler(options.PluginManager))
	console.DELETE("/plugins/:pluginId", pluginDeleteHandler(options.PluginManager))
	console.POST("/plugins/:pluginId/tools", pluginToolCreateHandler(options.PluginManager))
	console.GET("/plugins/:pluginId/tools", pluginToolListHandler(options.PluginManager))
	console.GET("/plugins/:pluginId/tools/:toolId", pluginToolGetHandler(options.PluginManager))
	console.PUT("/plugins/:pluginId/tools/:toolId", pluginToolUpdateHandler(options.PluginManager))
	console.DELETE("/plugins/:pluginId/tools/:toolId", pluginToolDeleteHandler(options.PluginManager))
	console.POST("/plugins/:pluginId/tools/:toolId/publish", pluginToolPublishHandler(options.PluginManager))
	console.POST("/plugins/:pluginId/tools/:toolId/test", pluginToolTestHandler(options.PluginManager))
	console.POST("/tools/:toolId/enable", pluginToolEnabledHandler(options.PluginManager, true))
	console.POST("/tools/:toolId/disable", pluginToolEnabledHandler(options.PluginManager, false))
	console.POST("/tools/query-by-ids", pluginToolsByIDsHandler(options.PluginManager))
	if options.RegisterConsoleRoutes != nil {
		options.RegisterConsoleRoutes(console)
	}
}

func registerPublicAPIRoutes(h *server.Hertz, options Options) {
	h.Engine.Handle(consts.MethodGet, "/api/models", apiModelsHandler(options.LegacyAdminManager))
}

func registerPublicExampleRoutes(h *server.Hertz) {
	h.Engine.Handle(consts.MethodGet, "/test/api/example/getOrder", exampleOrderGetHandler)
	h.Engine.Handle(consts.MethodPost, "/test/api/example/getOrder", exampleOrderPostHandler)
	h.Engine.Handle(consts.MethodPost, "/test/api/example/getOrder/:orderId", exampleOrderPathHandler)
}

func registerAPIGroup(h *server.Hertz, options Options) {
	api := h.Group("/api/v1")
	api.Use(authmiddleware.APIKeyAuth(options.Authenticator))
	api.POST("/apps/chat/completions", chatCompletionHandler(options.ChatManager))
	api.POST("/apps/workflow/completions", workflowCompletionHandler(options.WorkflowManager))
	api.POST("/apps/workflow/async-completions", workflowAsyncCompletionHandler(options.WorkflowManager))
	api.POST("/apps/workflow/stop-completions", workflowStopHandler(options.WorkflowManager))
	api.POST("/apps/workflow/async-results", workflowAsyncResultHandler(options.WorkflowManager))
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

func githubOAuthLoginHandler(provider OAuth2Provider) app.HandlerFunc {
	return func(_ context.Context, c *app.RequestContext) {
		if provider == nil {
			c.JSON(consts.StatusServiceUnavailable, errorEnvelope(newRequestID(), "OAUTH2_UNAVAILABLE", "GitHub OAuth2 is not configured."))
			return
		}
		authorizationURL, err := provider.AuthorizationURL()
		if err != nil {
			c.JSON(consts.StatusServiceUnavailable, errorEnvelope(newRequestID(), "OAUTH2_UNAVAILABLE", "GitHub OAuth2 is not configured."))
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), authorizationURL))
	}
}

func githubOAuthCallbackHandler(provider OAuth2Provider, issuer SessionIssuer) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if provider == nil {
			c.JSON(consts.StatusServiceUnavailable, errorEnvelope(newRequestID(), "OAUTH2_UNAVAILABLE", "GitHub OAuth2 is not configured."))
			return
		}
		oauthIssuer, ok := issuer.(OAuth2SessionIssuer)
		if !ok || oauthIssuer == nil {
			c.JSON(consts.StatusServiceUnavailable, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "OAuth2 session issuer is unavailable."))
			return
		}
		code := strings.TrimSpace(c.Query("code"))
		if code == "" {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", "code is required"))
			return
		}
		user, err := provider.Authenticate(ctx, code)
		if err != nil {
			c.JSON(consts.StatusBadGateway, errorEnvelope(newRequestID(), "OAUTH2_CALL_ERROR", "An internal error has occurred during oauth2 call, please try again later."))
			return
		}
		response, err := oauthIssuer.LoginOAuth(ctx, user, callerIP(c), strings.TrimSpace(c.Request.Header.Get("User-Agent")))
		if err != nil {
			status, responseCode, message := mapSessionError(err)
			c.JSON(status, errorEnvelope(newRequestID(), responseCode, message))
			return
		}
		redirectURL := fmt.Sprintf("/?access_token=%s&refresh_token=%s&expires_in=%d", response.AccessToken, response.RefreshToken, response.ExpiresIn)
		c.Response.Header.Set("Location", redirectURL)
		c.Response.SetStatusCode(consts.StatusFound)
	}
}

func accountProfileHandler(provider AccountProfileProvider) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		rc := contextx.MustFrom(ctx)
		if provider == nil {
			c.JSON(consts.StatusOK, successEnvelope(newRequestID(), auth.AccountProfile{
				AccountID: rc.AccountID,
				Username:  rc.Username,
				Type:      rc.AccountType,
			}))
			return
		}
		profile, err := provider.GetAccountProfile(ctx, rc.AccountID)
		if err != nil {
			status, code, message := mapSessionError(err)
			c.JSON(status, errorEnvelope(newRequestID(), code, message))
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), profile))
	}
}

func apiKeyCreateHandler(manager APIKeyManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			c.JSON(consts.StatusServiceUnavailable, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "api key service is unavailable"))
			return
		}
		var payload apiKeyDescriptionRequest
		if err := json.Unmarshal(c.Request.Body(), &payload); err != nil {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_JSON", "request body must be valid json"))
			return
		}
		id, err := manager.Create(ctx, contextx.MustFrom(ctx).AccountID, payload.Description)
		if err != nil {
			writeAPIKeyError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), fmt.Sprintf("%d", id)))
	}
}

func apiKeyListHandler(manager APIKeyManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			c.JSON(consts.StatusServiceUnavailable, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "api key service is unavailable"))
			return
		}
		page, err := manager.List(ctx, contextx.MustFrom(ctx).AccountID, int64(parsePositiveInt(c.Query("current"), 1)), int64(parsePositiveInt(c.Query("size"), 10)))
		if err != nil {
			writeAPIKeyError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), page))
	}
}

func apiKeyGetHandler(manager APIKeyManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		id, ok := parsePathID(c)
		if !ok {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", "id is invalid"))
			return
		}
		if manager == nil {
			c.JSON(consts.StatusServiceUnavailable, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "api key service is unavailable"))
			return
		}
		key, err := manager.Get(ctx, contextx.MustFrom(ctx).AccountID, id)
		if err != nil {
			writeAPIKeyError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), key))
	}
}

func apiKeyUpdateHandler(manager APIKeyManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		id, ok := parsePathID(c)
		if !ok {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", "id is invalid"))
			return
		}
		if manager == nil {
			c.JSON(consts.StatusServiceUnavailable, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "api key service is unavailable"))
			return
		}
		var payload apiKeyDescriptionRequest
		if err := json.Unmarshal(c.Request.Body(), &payload); err != nil {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_JSON", "request body must be valid json"))
			return
		}
		if err := manager.UpdateDescription(ctx, contextx.MustFrom(ctx).AccountID, id, payload.Description); err != nil {
			writeAPIKeyError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func apiKeyDeleteHandler(manager APIKeyManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		id, ok := parsePathID(c)
		if !ok {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", "id is invalid"))
			return
		}
		if manager == nil {
			c.JSON(consts.StatusServiceUnavailable, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "api key service is unavailable"))
			return
		}
		if err := manager.Delete(ctx, contextx.MustFrom(ctx).AccountID, id); err != nil {
			writeAPIKeyError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func workspaceCreateHandler(manager WorkspaceManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		var payload workspaceRequest
		if manager == nil {
			writeWorkspaceUnavailable(c)
			return
		}
		if err := json.Unmarshal(c.Request.Body(), &payload); err != nil {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_JSON", "request body must be valid json"))
			return
		}
		id, err := manager.Create(ctx, contextx.MustFrom(ctx).AccountID, payload.Name, optionalText(payload.Description), optionalText(payload.Config))
		if err != nil {
			writeWorkspaceError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), id))
	}
}

func workspaceListHandler(manager WorkspaceManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeWorkspaceUnavailable(c)
			return
		}
		items, err := manager.List(ctx, contextx.MustFrom(ctx).AccountID, int64(parsePositiveInt(c.Query("current"), 1)), int64(parsePositiveInt(c.Query("size"), 10)))
		if err != nil {
			writeWorkspaceError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), items))
	}
}

func workspaceGetHandler(manager WorkspaceManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeWorkspaceUnavailable(c)
			return
		}
		id := strings.TrimSpace(c.Param("workspaceId"))
		if id == "" {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", "workspaceId is required"))
			return
		}
		item, err := manager.Get(ctx, contextx.MustFrom(ctx).AccountID, id)
		if err != nil {
			writeWorkspaceError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), item))
	}
}

func workspaceUpdateHandler(manager WorkspaceManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeWorkspaceUnavailable(c)
			return
		}
		id := strings.TrimSpace(c.Param("workspaceId"))
		if id == "" {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", "workspaceId is required"))
			return
		}
		var payload workspaceRequest
		if err := json.Unmarshal(c.Request.Body(), &payload); err != nil {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_JSON", "request body must be valid json"))
			return
		}
		if err := manager.Update(ctx, contextx.MustFrom(ctx).AccountID, id, payload.Name, optionalText(payload.Description), optionalText(payload.Config)); err != nil {
			writeWorkspaceError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func workspaceDeleteHandler(manager WorkspaceManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeWorkspaceUnavailable(c)
			return
		}
		id := strings.TrimSpace(c.Param("workspaceId"))
		if id == "" {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", "workspaceId is required"))
			return
		}
		if err := manager.Delete(ctx, contextx.MustFrom(ctx).AccountID, id); err != nil {
			writeWorkspaceError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func accountCreateHandler(manager AccountManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeAccountUnavailable(c)
			return
		}
		var input account.Input
		if err := json.Unmarshal(c.Request.Body(), &input); err != nil {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_JSON", "request body must be valid json"))
			return
		}
		id, err := manager.Create(ctx, contextx.MustFrom(ctx).AccountID, input)
		if err != nil {
			writeAccountError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), id))
	}
}

func accountListHandler(manager AccountManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeAccountUnavailable(c)
			return
		}
		page, err := manager.List(ctx, contextx.MustFrom(ctx).AccountID, c.Query("name"), int64(parsePositiveInt(c.Query("current"), 1)), int64(parsePositiveInt(c.Query("size"), 10)))
		if err != nil {
			writeAccountError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), page))
	}
}

func accountGetHandler(manager AccountManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeAccountUnavailable(c)
			return
		}
		id := strings.TrimSpace(c.Param("accountId"))
		if id == "" {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", "accountId is required"))
			return
		}
		item, err := manager.Get(ctx, id)
		if err != nil {
			writeAccountError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), item))
	}
}

func accountUpdateHandler(manager AccountManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeAccountUnavailable(c)
			return
		}
		id := strings.TrimSpace(c.Param("accountId"))
		if id == "" {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", "accountId is required"))
			return
		}
		var input account.Input
		if err := json.Unmarshal(c.Request.Body(), &input); err != nil {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_JSON", "request body must be valid json"))
			return
		}
		if err := manager.Update(ctx, contextx.MustFrom(ctx).AccountID, id, input); err != nil {
			writeAccountError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func accountDeleteHandler(manager AccountManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeAccountUnavailable(c)
			return
		}
		id := strings.TrimSpace(c.Param("accountId"))
		if id == "" {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", "accountId is required"))
			return
		}
		if err := manager.Delete(ctx, contextx.MustFrom(ctx).AccountID, id); err != nil {
			writeAccountError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func accountChangePasswordHandler(manager AccountManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeAccountUnavailable(c)
			return
		}
		var payload changePasswordRequest
		if err := json.Unmarshal(c.Request.Body(), &payload); err != nil {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_JSON", "request body must be valid json"))
			return
		}
		if err := manager.ChangePassword(ctx, contextx.MustFrom(ctx).AccountID, payload.Password, payload.NewPassword); err != nil {
			writeAccountError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func providerCreateHandler(manager ModelProviderManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeModelProviderUnavailable(c)
			return
		}
		var input modelprovider.CreateProviderInput
		if err := json.Unmarshal(c.Request.Body(), &input); err != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		if _, err := manager.CreateProvider(ctx, rc.WorkspaceID, rc.AccountID, input); err != nil {
			writeModelProviderError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), true))
	}
}

func applicationCreateHandler(manager ApplicationManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeApplicationUnavailable(c)
			return
		}
		var in application.Input
		if json.Unmarshal(c.Request.Body(), &in) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		id, e := manager.Create(ctx, rc.WorkspaceID, rc.AccountID, in)
		if e != nil {
			writeApplicationError(c, e)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), id))
	}
}
func applicationListHandler(manager ApplicationManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeApplicationUnavailable(c)
			return
		}
		p, e := manager.List(ctx, contextx.MustFrom(ctx).WorkspaceID, c.Query("name"), c.Query("type"), c.Query("status"), int64(parsePositiveInt(c.Query("current"), 1)), int64(parsePositiveInt(c.Query("size"), 10)))
		if e != nil {
			writeApplicationError(c, e)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), p))
	}
}
func applicationGetHandler(manager ApplicationManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeApplicationUnavailable(c)
			return
		}
		id, ok := requiredPathParameter(c, "appId")
		if !ok {
			return
		}
		v, e := manager.Get(ctx, contextx.MustFrom(ctx).WorkspaceID, id)
		if e != nil {
			writeApplicationError(c, e)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), v))
	}
}
func applicationUpdateHandler(manager ApplicationManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeApplicationUnavailable(c)
			return
		}
		id, ok := requiredPathParameter(c, "appId")
		if !ok {
			return
		}
		var in application.Input
		if json.Unmarshal(c.Request.Body(), &in) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		if e := manager.Update(ctx, rc.WorkspaceID, rc.AccountID, id, in); e != nil {
			writeApplicationError(c, e)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}
func applicationDeleteHandler(manager ApplicationManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeApplicationUnavailable(c)
			return
		}
		id, ok := requiredPathParameter(c, "appId")
		if !ok {
			return
		}
		rc := contextx.MustFrom(ctx)
		if e := manager.Delete(ctx, rc.WorkspaceID, rc.AccountID, id); e != nil {
			writeApplicationError(c, e)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}
func applicationPublishHandler(manager ApplicationManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeApplicationUnavailable(c)
			return
		}
		id, ok := requiredPathParameter(c, "appId")
		if !ok {
			return
		}
		rc := contextx.MustFrom(ctx)
		if e := manager.Publish(ctx, rc.WorkspaceID, rc.AccountID, id); e != nil {
			writeApplicationError(c, e)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}
func applicationCopyHandler(manager ApplicationManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeApplicationUnavailable(c)
			return
		}
		id, ok := requiredPathParameter(c, "appId")
		if !ok {
			return
		}
		rc := contextx.MustFrom(ctx)
		copyID, e := manager.Copy(ctx, rc.WorkspaceID, rc.AccountID, id)
		if e != nil {
			writeApplicationError(c, e)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), copyID))
	}
}
func applicationVersionListHandler(manager ApplicationManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeApplicationUnavailable(c)
			return
		}
		id, ok := requiredPathParameter(c, "appId")
		if !ok {
			return
		}
		p, e := manager.ListVersions(ctx, contextx.MustFrom(ctx).WorkspaceID, id, c.Query("status"), int64(parsePositiveInt(c.Query("current"), 1)), int64(parsePositiveInt(c.Query("size"), 10)))
		if e != nil {
			writeApplicationError(c, e)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), p))
	}
}
func applicationVersionGetHandler(manager ApplicationManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeApplicationUnavailable(c)
			return
		}
		id, ok := requiredPathParameter(c, "appId")
		if !ok {
			return
		}
		version, ok := requiredPathParameter(c, "version")
		if !ok {
			return
		}
		v, e := manager.GetVersion(ctx, contextx.MustFrom(ctx).WorkspaceID, id, version)
		if e != nil {
			writeApplicationError(c, e)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), v))
	}
}
func writeApplicationUnavailable(c *app.RequestContext) {
	c.JSON(consts.StatusServiceUnavailable, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "application service is unavailable"))
}
func writeApplicationError(c *app.RequestContext, e error) {
	switch {
	case errors.Is(e, application.ErrNameRequired), errors.Is(e, application.ErrConfigRequired), errors.Is(e, application.ErrModelProviderRequired), errors.Is(e, application.ErrModelRequired):
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", e.Error()))
	case errors.Is(e, application.ErrNameExists):
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "APP_NAME_EXISTS", "Application name already exists."))
	case errors.Is(e, application.ErrNotFound):
		c.JSON(consts.StatusNotFound, errorEnvelope(newRequestID(), "APP_NOT_FOUND", "Application can not be found."))
	case errors.Is(e, application.ErrVersionNotFound):
		c.JSON(consts.StatusNotFound, errorEnvelope(newRequestID(), "APP_VERSION_NOT_FOUND", "Application version can not be found."))
	default:
		c.JSON(consts.StatusInternalServerError, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "An internal error has occurred, please try again later or contact service support."))
	}
}

func appComponentListHandler(manager AppComponentManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeAppComponentUnavailable(c)
			return
		}
		status := int64(-1)
		if value := strings.TrimSpace(c.Query("status")); value != "" {
			if parsed, err := strconv.ParseInt(value, 10, 16); err == nil {
				status = parsed
			}
		}
		value, err := manager.List(ctx, contextx.MustFrom(ctx).WorkspaceID, c.Query("name"), c.Query("type"), c.Query("app_id"), status, int64(parsePositiveInt(c.Query("current"), 1)), int64(parsePositiveInt(c.Query("size"), 10)))
		if err != nil {
			writeAppComponentError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func appComponentPublishableHandler(manager AppComponentManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeAppComponentUnavailable(c)
			return
		}
		value, err := manager.Publishable(ctx, contextx.MustFrom(ctx).WorkspaceID, c.Query("type"), c.Query("app_name"), int64(parsePositiveInt(c.Query("current"), 1)), int64(parsePositiveInt(c.Query("size"), 10)))
		if err != nil {
			writeAppComponentError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func appComponentCreateHandler(manager AppComponentManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeAppComponentUnavailable(c)
			return
		}
		var input appcomponent.Input
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		code, err := manager.Create(ctx, rc.WorkspaceID, rc.AccountID, input)
		if err != nil {
			writeAppComponentError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), code))
	}
}

func appComponentUpdateHandler(manager AppComponentManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeAppComponentUnavailable(c)
			return
		}
		code, ok := requiredPathParameter(c, "code")
		if !ok {
			return
		}
		var input appcomponent.Input
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		if err := manager.Update(ctx, rc.WorkspaceID, rc.AccountID, code, input); err != nil {
			writeAppComponentError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), "update success"))
	}
}

func appComponentDeleteHandler(manager AppComponentManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeAppComponentUnavailable(c)
			return
		}
		code, ok := requiredPathParameter(c, "code")
		if !ok {
			return
		}
		rc := contextx.MustFrom(ctx)
		if err := manager.Delete(ctx, rc.WorkspaceID, rc.AccountID, code); err != nil {
			writeAppComponentError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), true))
	}
}

func appComponentGetHandler(manager AppComponentManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeAppComponentUnavailable(c)
			return
		}
		code, ok := requiredPathParameter(c, "code")
		if !ok {
			return
		}
		value, err := manager.Get(ctx, contextx.MustFrom(ctx).WorkspaceID, code)
		if errors.Is(err, appcomponent.ErrNotFound) {
			c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
			return
		}
		if err != nil {
			writeAppComponentError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func appComponentGetByAppIDHandler(manager AppComponentManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeAppComponentUnavailable(c)
			return
		}
		appID, ok := requiredPathParameter(c, "appId")
		if !ok {
			return
		}
		value, err := manager.GetByAppID(ctx, contextx.MustFrom(ctx).WorkspaceID, appID)
		if errors.Is(err, appcomponent.ErrNotFound) {
			c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
			return
		}
		if err != nil {
			writeAppComponentError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func appComponentReferHandler(manager AppComponentManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeAppComponentUnavailable(c)
			return
		}
		code, ok := requiredPathParameter(c, "code")
		if !ok {
			return
		}
		value, err := manager.Refer(ctx, contextx.MustFrom(ctx).WorkspaceID, code)
		if err != nil {
			writeAppComponentError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func appComponentConfigHandler(manager AppComponentManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeAppComponentUnavailable(c)
			return
		}
		appID, ok := requiredPathParameter(c, "appId")
		if !ok {
			return
		}
		value, err := manager.QueryConfig(ctx, contextx.MustFrom(ctx).WorkspaceID, appID)
		if err != nil {
			writeAppComponentError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func appComponentByCodesHandler(manager AppComponentManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeAppComponentUnavailable(c)
			return
		}
		var input appcomponent.Input
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		if len(input.Codes) == 0 {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", "codes is required"))
			return
		}
		value, err := manager.ListByCodes(ctx, contextx.MustFrom(ctx).WorkspaceID, input.Codes)
		if err != nil {
			writeAppComponentError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func appComponentSchemaHandler(manager AppComponentManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeAppComponentUnavailable(c)
			return
		}
		code, ok := requiredPathParameter(c, "code")
		if !ok {
			return
		}
		value, err := manager.Schema(ctx, contextx.MustFrom(ctx).WorkspaceID, code)
		if err != nil {
			writeAppComponentError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func appComponentSchemasHandler(manager AppComponentManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeAppComponentUnavailable(c)
			return
		}
		var input appcomponent.Input
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		if len(input.Codes) == 0 {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", "codes is required"))
			return
		}
		value, err := manager.Schemas(ctx, contextx.MustFrom(ctx).WorkspaceID, input.Codes)
		if err != nil {
			writeAppComponentError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func writeAppComponentUnavailable(c *app.RequestContext) {
	c.JSON(consts.StatusServiceUnavailable, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "application component service is unavailable"))
}
func writeAppComponentError(c *app.RequestContext, err error) {
	switch {
	case errors.Is(err, appcomponent.ErrTypeRequired), errors.Is(err, appcomponent.ErrNameRequired), errors.Is(err, appcomponent.ErrConfigRequired), errors.Is(err, appcomponent.ErrAppIDRequired), errors.Is(err, appcomponent.ErrDescriptionRequired):
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", err.Error()))
	case errors.Is(err, appcomponent.ErrNotFound):
		c.JSON(consts.StatusNotFound, errorEnvelope(newRequestID(), "APP_COMPONENT_NOT_FOUND", "Application component can not be found."))
	case errors.Is(err, application.ErrNotFound):
		c.JSON(consts.StatusNotFound, errorEnvelope(newRequestID(), "APP_NOT_FOUND", "Application can not be found."))
	default:
		c.JSON(consts.StatusInternalServerError, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "An internal error has occurred, please try again later or contact service support."))
	}
}

func agentSchemaCreateHandler(manager AgentSchemaManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeAgentSchemaUnavailable(c)
			return
		}
		var input agentschema.Input
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		value, err := manager.Create(ctx, rc.WorkspaceID, rc.AccountID, input)
		if err != nil {
			writeAgentSchemaError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func agentSchemaListHandler(manager AgentSchemaManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeAgentSchemaUnavailable(c)
			return
		}
		values, err := manager.List(ctx, contextx.MustFrom(ctx).WorkspaceID, c.Query("name"))
		if err != nil {
			writeAgentSchemaError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), values))
	}
}

func agentSchemaPageHandler(manager AgentSchemaManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeAgentSchemaUnavailable(c)
			return
		}
		value, err := manager.Page(ctx, contextx.MustFrom(ctx).WorkspaceID, c.Query("name"), int64(parsePositiveInt(c.Query("current"), 1)), int64(parsePositiveInt(c.Query("size"), 10)))
		if err != nil {
			writeAgentSchemaError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func agentSchemaSearchHandler(manager AgentSchemaManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeAgentSchemaUnavailable(c)
			return
		}
		values, err := manager.List(ctx, contextx.MustFrom(ctx).WorkspaceID, c.Query("name"))
		if err != nil {
			writeAgentSchemaError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), values))
	}
}

func agentSchemaGetHandler(manager AgentSchemaManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeAgentSchemaUnavailable(c)
			return
		}
		id, ok := parsePathID(c)
		if !ok {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", "id is required"))
			return
		}
		value, err := manager.Get(ctx, contextx.MustFrom(ctx).WorkspaceID, id)
		if err != nil {
			writeAgentSchemaError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func agentSchemaUpdateHandler(manager AgentSchemaManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeAgentSchemaUnavailable(c)
			return
		}
		id, ok := parsePathID(c)
		if !ok {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", "id is required"))
			return
		}
		var input agentschema.Input
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		value, err := manager.Update(ctx, rc.WorkspaceID, rc.AccountID, id, input)
		if err != nil {
			writeAgentSchemaError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func agentSchemaEnabledHandler(manager AgentSchemaManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeAgentSchemaUnavailable(c)
			return
		}
		id, ok := parsePathID(c)
		if !ok {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", "id is required"))
			return
		}
		enabled, ok := parseBool(c.Query("enabled"))
		if !ok {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", "enabled must be true or false"))
			return
		}
		rc := contextx.MustFrom(ctx)
		if err := manager.SetEnabled(ctx, rc.WorkspaceID, rc.AccountID, id, enabled); err != nil {
			writeAgentSchemaError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func agentSchemaDeleteHandler(manager AgentSchemaManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeAgentSchemaUnavailable(c)
			return
		}
		id, ok := parsePathID(c)
		if !ok {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", "id is required"))
			return
		}
		if err := manager.Delete(ctx, contextx.MustFrom(ctx).WorkspaceID, id); err != nil {
			writeAgentSchemaError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func writeAgentSchemaUnavailable(c *app.RequestContext) {
	c.JSON(consts.StatusServiceUnavailable, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "agent schema service is unavailable"))
}

func writeAgentSchemaError(c *app.RequestContext, err error) {
	switch {
	case errors.Is(err, agentschema.ErrNameRequired), errors.Is(err, agentschema.ErrTypeRequired):
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", err.Error()))
	case errors.Is(err, agentschema.ErrNameExists):
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "AGENT_SCHEMA_NAME_EXISTS", "Agent schema name already exists."))
	case errors.Is(err, agentschema.ErrNotFound):
		c.JSON(consts.StatusNotFound, errorEnvelope(newRequestID(), "AGENT_SCHEMA_NOT_FOUND", "Agent schema can not be found."))
	default:
		c.JSON(consts.StatusInternalServerError, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "An internal error has occurred, please try again later or contact service support."))
	}
}

func toolCreateHandler(manager ToolManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeToolUnavailable(c)
			return
		}
		var input tool.Input
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		value, err := manager.Create(ctx, rc.WorkspaceID, rc.AccountID, input)
		if err != nil {
			writeToolError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func toolListHandler(manager ToolManager) app.HandlerFunc   { return toolListHandlerFor(manager, "") }
func toolSearchHandler(manager ToolManager) app.HandlerFunc { return toolListHandlerFor(manager, "") }
func toolPluginListHandler(manager ToolManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeToolUnavailable(c)
			return
		}
		values, err := manager.List(ctx, contextx.MustFrom(ctx).WorkspaceID, c.Query("name"), strings.TrimSpace(c.Param("pluginId")))
		if err != nil {
			writeToolError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), values))
	}
}
func toolListHandlerFor(manager ToolManager, pluginID string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeToolUnavailable(c)
			return
		}
		values, err := manager.List(ctx, contextx.MustFrom(ctx).WorkspaceID, c.Query("name"), pluginID)
		if err != nil {
			writeToolError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), values))
	}
}
func toolPageHandler(manager ToolManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeToolUnavailable(c)
			return
		}
		value, err := manager.Page(ctx, contextx.MustFrom(ctx).WorkspaceID, c.Query("name"), c.Query("pluginId"), int64(parsePositiveInt(c.Query("current"), 1)), int64(parsePositiveInt(c.Query("size"), 10)))
		if err != nil {
			writeToolError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}
func toolGetHandler(manager ToolManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeToolUnavailable(c)
			return
		}
		id, ok := parsePathID(c)
		if !ok {
			writeInvalidToolID(c)
			return
		}
		value, err := manager.Get(ctx, contextx.MustFrom(ctx).WorkspaceID, id)
		if err != nil {
			writeToolError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}
func toolUpdateHandler(manager ToolManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeToolUnavailable(c)
			return
		}
		id, ok := parsePathID(c)
		if !ok {
			writeInvalidToolID(c)
			return
		}
		var input tool.Input
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		value, err := manager.Update(ctx, rc.WorkspaceID, rc.AccountID, id, input)
		if err != nil {
			writeToolError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}
func toolEnabledHandler(manager ToolManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeToolUnavailable(c)
			return
		}
		id, ok := parsePathID(c)
		if !ok {
			writeInvalidToolID(c)
			return
		}
		enabled, ok := parseBool(c.Query("enabled"))
		if !ok {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", "enabled must be true or false"))
			return
		}
		rc := contextx.MustFrom(ctx)
		if err := manager.SetEnabled(ctx, rc.WorkspaceID, rc.AccountID, id, enabled); err != nil {
			writeToolError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}
func toolDeleteHandler(manager ToolManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeToolUnavailable(c)
			return
		}
		id, ok := parsePathID(c)
		if !ok {
			writeInvalidToolID(c)
			return
		}
		if err := manager.Delete(ctx, contextx.MustFrom(ctx).WorkspaceID, id); err != nil {
			writeToolError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}
func writeInvalidToolID(c *app.RequestContext) {
	c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", "id is required"))
}
func writeToolUnavailable(c *app.RequestContext) {
	c.JSON(consts.StatusServiceUnavailable, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "tool service is unavailable"))
}
func writeToolError(c *app.RequestContext, err error) {
	switch {
	case errors.Is(err, tool.ErrNameRequired), errors.Is(err, tool.ErrConfigRequired), errors.Is(err, tool.ErrSchemaRequired):
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", err.Error()))
	case errors.Is(err, tool.ErrNotFound):
		c.JSON(consts.StatusNotFound, errorEnvelope(newRequestID(), "TOOL_NOT_FOUND", "Tool can not be found."))
	default:
		c.JSON(consts.StatusInternalServerError, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "An internal error has occurred, please try again later or contact service support."))
	}
}

func fileUploadHandler(storageDir string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		form, err := c.MultipartForm()
		if err != nil {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", "multipart form is required"))
			return
		}
		categoryValues := form.Value["category"]
		category := ""
		if len(categoryValues) > 0 {
			category = strings.TrimSpace(categoryValues[0])
		}
		files := form.File["files"]
		if len(files) == 0 {
			files = form.File["file"]
		}
		key := ""
		if values := form.Value["key"]; len(values) > 0 {
			key = strings.TrimSpace(values[0])
		}
		if category == "" && key != "" {
			parts := strings.Split(filepath.ToSlash(key), "/")
			if len(parts) >= 3 {
				category = parts[1]
			}
		}
		if category == "" || len(files) == 0 {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", "category and files are required"))
			return
		}
		rc := contextx.MustFrom(ctx)
		result := make([]fileUploadPolicy, 0, len(files))
		for _, file := range files {
			var policy fileUploadPolicy
			var err error
			if key != "" && len(files) == 1 {
				policy, err = saveUploadedFileAt(storageDir, rc.AccountID, category, key, file)
			} else {
				policy, err = saveUploadedFile(storageDir, rc.AccountID, category, file)
			}
			if err != nil {
				c.JSON(consts.StatusInternalServerError, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "save uploaded file failed"))
				return
			}
			result = append(result, policy)
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), result))
	}
}

type webUploadPolicyRequest struct {
	Category string `json:"category"`
	Files    []struct {
		Name string `json:"name"`
	} `json:"files"`
}
type webUploadPolicy struct {
	fileUploadPolicy
	AccessID   string `json:"access_id"`
	Policy     string `json:"policy"`
	Host       string `json:"host"`
	Expire     int64  `json:"expire"`
	Signature  string `json:"signature"`
	UploadType string `json:"upload_type"`
}

func fileUploadPoliciesHandler(storageDir string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		var input webUploadPolicyRequest
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		category := filepath.Base(strings.TrimSpace(input.Category))
		if category == "." || category == "" || len(input.Files) == 0 {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", "category and files are required"))
			return
		}
		if storageDir == "" {
			storageDir = "data/files"
		}
		rc := contextx.MustFrom(ctx)
		result := make([]webUploadPolicy, 0, len(input.Files))
		for _, file := range input.Files {
			name := filepath.Base(strings.TrimSpace(file.Name))
			if name == "." || name == "" {
				c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", "file name is invalid"))
				return
			}
			path := filepath.ToSlash(filepath.Join(rc.AccountID, category, newRequestID()+filepath.Ext(name)))
			result = append(result, webUploadPolicy{fileUploadPolicy: fileUploadPolicy{Path: path, Name: name, Extension: strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), ".")}, Host: "/console/v1/files/upload", Expire: time.Now().Add(15 * time.Minute).Unix(), UploadType: "file"})
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), result))
	}
}

func fileDownloadHandler(storageDir string) app.HandlerFunc {
	return func(_ context.Context, c *app.RequestContext) {
		path, ok := resolveStoredFile(storageDir, c.Query("path"))
		if !ok {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", "path is invalid"))
			return
		}
		if _, err := os.Stat(path); err != nil {
			c.JSON(consts.StatusNotFound, errorEnvelope(newRequestID(), "FILE_NOT_FOUND", "File can not be found."))
			return
		}
		if c.Query("preview") == "true" {
			c.File(path)
			return
		}
		c.FileAttachment(path, filepath.Base(path))
	}
}

func filePreviewURLHandler(_ context.Context, c *app.RequestContext) {
	c.JSON(consts.StatusOK, successEnvelope(newRequestID(), "/console/v1/files/download?path="+c.Query("path")+"&preview=true"))
}

func knowledgeBaseCreateHandler(m KnowledgeBaseManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if m == nil {
			writeKBUnavailable(c)
			return
		}
		var in knowledgebase.Input
		if json.Unmarshal(c.Request.Body(), &in) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		id, e := m.Create(ctx, rc.WorkspaceID, rc.AccountID, in)
		if e != nil {
			writeKBError(c, e)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), id))
	}
}
func knowledgeBaseListHandler(m KnowledgeBaseManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if m == nil {
			writeKBUnavailable(c)
			return
		}
		v, e := m.List(ctx, contextx.MustFrom(ctx).WorkspaceID, c.Query("keyword"), int64(parsePositiveInt(c.Query("current"), 1)), int64(parsePositiveInt(c.Query("size"), 10)))
		if e != nil {
			writeKBError(c, e)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), v))
	}
}
func knowledgeBaseGetHandler(m KnowledgeBaseManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if m == nil {
			writeKBUnavailable(c)
			return
		}
		id, ok := requiredPathParameter(c, "kbId")
		if !ok {
			return
		}
		v, e := m.Get(ctx, contextx.MustFrom(ctx).WorkspaceID, id)
		if e != nil {
			writeKBError(c, e)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), v))
	}
}
func knowledgeBaseUpdateHandler(m KnowledgeBaseManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if m == nil {
			writeKBUnavailable(c)
			return
		}
		id, ok := requiredPathParameter(c, "kbId")
		if !ok {
			return
		}
		var in knowledgebase.Input
		if json.Unmarshal(c.Request.Body(), &in) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		if e := m.Update(ctx, rc.WorkspaceID, rc.AccountID, id, in); e != nil {
			writeKBError(c, e)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}
func knowledgeBaseDeleteHandler(m KnowledgeBaseManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if m == nil {
			writeKBUnavailable(c)
			return
		}
		id, ok := requiredPathParameter(c, "kbId")
		if !ok {
			return
		}
		rc := contextx.MustFrom(ctx)
		if e := m.Delete(ctx, rc.WorkspaceID, rc.AccountID, id); e != nil {
			writeKBError(c, e)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func knowledgeBaseListByCodesHandler(m KnowledgeBaseManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if m == nil {
			writeKBUnavailable(c)
			return
		}
		var input struct {
			KbIDs []string `json:"kb_ids"`
		}
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		if len(input.KbIDs) == 0 {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", "kb_ids is required"))
			return
		}
		value, err := m.ListByCodes(ctx, contextx.MustFrom(ctx).WorkspaceID, input.KbIDs)
		if err != nil {
			writeKBError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func knowledgeBaseRetrieveHandler(m KnowledgeBaseManager) app.HandlerFunc {
	return func(_ context.Context, c *app.RequestContext) {
		if m == nil {
			writeKBUnavailable(c)
			return
		}
		var input struct {
			Query         string `json:"query"`
			SearchOptions struct {
				KbIDs []string `json:"kb_ids"`
			} `json:"search_options"`
		}
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		if strings.TrimSpace(input.Query) == "" || len(input.SearchOptions.KbIDs) == 0 {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", "query and search_options.kb_ids are required"))
			return
		}
		// 文档索引尚未迁移到 Go；没有索引数据时 Java 检索也应返回空集合。
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), []any{}))
	}
}
func writeKBUnavailable(c *app.RequestContext) {
	c.JSON(consts.StatusServiceUnavailable, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "knowledge base service is unavailable"))
}
func writeKBError(c *app.RequestContext, e error) {
	switch {
	case errors.Is(e, knowledgebase.ErrNameRequired), errors.Is(e, knowledgebase.ErrIndexConfigRequired):
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", e.Error()))
	case errors.Is(e, knowledgebase.ErrNotFound):
		c.JSON(consts.StatusNotFound, errorEnvelope(newRequestID(), "KNOWLEDGE_BASE_NOT_FOUND", "Knowledge base can not be found."))
	default:
		c.JSON(consts.StatusInternalServerError, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "An internal error has occurred, please try again later or contact service support."))
	}
}

func documentCreateHandler(manager DocumentManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeDocumentUnavailable(c)
			return
		}
		kbID, ok := requiredPathParameter(c, "kbId")
		if !ok {
			return
		}
		var input document.CreateInput
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		value, err := manager.Create(ctx, rc.WorkspaceID, rc.AccountID, kbID, input)
		if err != nil {
			writeDocumentError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func documentListHandler(manager DocumentManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeDocumentUnavailable(c)
			return
		}
		kbID, ok := requiredPathParameter(c, "kbId")
		if !ok {
			return
		}
		indexStatus := int64(-1)
		if value := strings.TrimSpace(c.Query("index_status")); value != "" {
			if parsed, err := strconv.ParseInt(value, 10, 16); err == nil {
				indexStatus = parsed
			}
		}
		value, err := manager.List(ctx, contextx.MustFrom(ctx).WorkspaceID, kbID, c.Query("name"), indexStatus, int64(parsePositiveInt(c.Query("current"), 1)), int64(parsePositiveInt(c.Query("size"), 10)))
		if err != nil {
			writeDocumentError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func documentGetHandler(manager DocumentManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeDocumentUnavailable(c)
			return
		}
		kbID, ok := requiredPathParameter(c, "kbId")
		if !ok {
			return
		}
		docID, ok := requiredPathParameter(c, "docId")
		if !ok {
			return
		}
		value, err := manager.Get(ctx, contextx.MustFrom(ctx).WorkspaceID, kbID, docID)
		if err != nil {
			writeDocumentError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func documentUpdateHandler(manager DocumentManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeDocumentUnavailable(c)
			return
		}
		kbID, ok := requiredPathParameter(c, "kbId")
		if !ok {
			return
		}
		docID, ok := requiredPathParameter(c, "docId")
		if !ok {
			return
		}
		var input document.UpdateInput
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		if err := manager.Update(ctx, rc.WorkspaceID, rc.AccountID, kbID, docID, input); err != nil {
			writeDocumentError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func documentDeleteHandler(manager DocumentManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeDocumentUnavailable(c)
			return
		}
		kbID, ok := requiredPathParameter(c, "kbId")
		if !ok {
			return
		}
		docID, ok := requiredPathParameter(c, "docId")
		if !ok {
			return
		}
		rc := contextx.MustFrom(ctx)
		if err := manager.Delete(ctx, rc.WorkspaceID, rc.AccountID, kbID, docID); err != nil {
			writeDocumentError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func documentBatchDeleteHandler(manager DocumentManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeDocumentUnavailable(c)
			return
		}
		kbID, ok := requiredPathParameter(c, "kbId")
		if !ok {
			return
		}
		var input struct {
			DocIDs []string `json:"doc_ids"`
		}
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		if len(input.DocIDs) == 0 {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", "doc_ids is required"))
			return
		}
		rc := contextx.MustFrom(ctx)
		if err := manager.DeleteBatch(ctx, rc.WorkspaceID, rc.AccountID, kbID, input.DocIDs); err != nil {
			writeDocumentError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func documentReIndexHandler(manager DocumentManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeDocumentUnavailable(c)
			return
		}
		kbID, ok := requiredPathParameter(c, "kbId")
		if !ok {
			return
		}
		docID, ok := requiredPathParameter(c, "docId")
		if !ok {
			return
		}
		var input document.ReIndexInput
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		if err := manager.ReIndex(ctx, rc.WorkspaceID, rc.AccountID, kbID, docID, input); err != nil {
			writeDocumentError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func writeDocumentUnavailable(c *app.RequestContext) {
	c.JSON(consts.StatusServiceUnavailable, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "document service is unavailable"))
}

func writeDocumentError(c *app.RequestContext, err error) {
	switch {
	case errors.Is(err, document.ErrFilesRequired), errors.Is(err, document.ErrNameRequired):
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", err.Error()))
	case errors.Is(err, document.ErrDocumentNotFound):
		c.JSON(consts.StatusNotFound, errorEnvelope(newRequestID(), "DOCUMENT_NOT_FOUND", "Document can not be found."))
	case errors.Is(err, document.ErrChunkNotFound):
		c.JSON(consts.StatusNotFound, errorEnvelope(newRequestID(), "DOCUMENT_CHUNK_NOT_FOUND", "Document chunk can not be found."))
	case errors.Is(err, document.ErrTextRequired):
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", err.Error()))
	default:
		c.JSON(consts.StatusInternalServerError, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "An internal error has occurred, please try again later or contact service support."))
	}
}

func documentChunkCreateHandler(manager DocumentManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeDocumentUnavailable(c)
			return
		}
		docID, ok := requiredPathParameter(c, "docId")
		if !ok {
			return
		}
		var input document.ChunkInput
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		chunkID, err := manager.CreateChunk(ctx, rc.WorkspaceID, rc.AccountID, docID, input)
		if err != nil {
			writeDocumentError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), chunkID))
	}
}

func documentChunkListHandler(manager DocumentManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeDocumentUnavailable(c)
			return
		}
		docID, ok := requiredPathParameter(c, "docId")
		if !ok {
			return
		}
		value, err := manager.ListChunks(ctx, contextx.MustFrom(ctx).WorkspaceID, docID, int64(parsePositiveInt(c.Query("current"), 1)), int64(parsePositiveInt(c.Query("size"), 10)))
		if err != nil {
			writeDocumentError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func documentChunkUpdateHandler(manager DocumentManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeDocumentUnavailable(c)
			return
		}
		docID, ok := requiredPathParameter(c, "docId")
		if !ok {
			return
		}
		chunkID, ok := requiredPathParameter(c, "chunkId")
		if !ok {
			return
		}
		var input document.ChunkInput
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		if err := manager.UpdateChunk(ctx, rc.WorkspaceID, rc.AccountID, docID, chunkID, input); err != nil {
			writeDocumentError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func documentChunkDeleteHandler(manager DocumentManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeDocumentUnavailable(c)
			return
		}
		docID, ok := requiredPathParameter(c, "docId")
		if !ok {
			return
		}
		chunkID, ok := requiredPathParameter(c, "chunkId")
		if !ok {
			return
		}
		rc := contextx.MustFrom(ctx)
		if err := manager.DeleteChunks(ctx, rc.WorkspaceID, rc.AccountID, docID, []string{chunkID}); err != nil {
			writeDocumentError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func documentChunkBatchDeleteHandler(manager DocumentManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeDocumentUnavailable(c)
			return
		}
		docID, ok := requiredPathParameter(c, "docId")
		if !ok {
			return
		}
		var input struct {
			ChunkIDs []string `json:"chunk_ids"`
		}
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		if len(input.ChunkIDs) == 0 {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", "chunk_ids is required"))
			return
		}
		rc := contextx.MustFrom(ctx)
		if err := manager.DeleteChunks(ctx, rc.WorkspaceID, rc.AccountID, docID, input.ChunkIDs); err != nil {
			writeDocumentError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func documentChunkPreviewHandler(manager DocumentManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeDocumentUnavailable(c)
			return
		}
		docID, ok := requiredPathParameter(c, "docId")
		if !ok {
			return
		}
		value, err := manager.PreviewChunks(ctx, contextx.MustFrom(ctx).WorkspaceID, docID)
		if err != nil {
			writeDocumentError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func documentChunkStatusHandler(manager DocumentManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeDocumentUnavailable(c)
			return
		}
		docID, ok := requiredPathParameter(c, "docId")
		if !ok {
			return
		}
		var input document.ChunkStatusInput
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		if err := manager.SetChunksEnabled(ctx, rc.WorkspaceID, rc.AccountID, docID, input); err != nil {
			writeDocumentError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func mcpServerCreateHandler(manager McpServerManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeMcpServerUnavailable(c)
			return
		}
		var input mcpserver.Input
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		serverCode, err := manager.Create(ctx, rc.WorkspaceID, rc.AccountID, input)
		if err != nil {
			writeMcpServerError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), serverCode))
	}
}

func mcpServerUpdateHandler(manager McpServerManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeMcpServerUnavailable(c)
			return
		}
		var input mcpserver.Input
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		if err := manager.Update(ctx, contextx.MustFrom(ctx).WorkspaceID, input); err != nil {
			writeMcpServerError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func mcpServerDeleteHandler(manager McpServerManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeMcpServerUnavailable(c)
			return
		}
		serverCode, ok := requiredPathParameter(c, "serverCode")
		if !ok {
			return
		}
		if err := manager.Delete(ctx, contextx.MustFrom(ctx).WorkspaceID, serverCode); err != nil {
			writeMcpServerError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func chatCompletionHandler(manager ChatManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeChatUnavailable(c)
			return
		}
		var input chat.Request
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		if input.Stream {
			streamer, ok := manager.(ChatStreamer)
			if !ok {
				c.JSON(consts.StatusNotImplemented, errorEnvelope(newRequestID(), "STREAM_UNSUPPORTED", "chat manager does not support streaming"))
				return
			}
			events, err := streamer.CompleteStream(ctx, contextx.MustFrom(ctx).WorkspaceID, input)
			if err != nil {
				writeChatError(c, err)
				return
			}
			c.Response.Header.SetContentType("text/event-stream")
			c.Response.Header.Set("Cache-Control", "no-cache")
			c.Response.Header.Set("X-Accel-Buffering", "no")
			reader, writer := io.Pipe()
			c.SetBodyStream(reader, -1)
			go writeChatSSE(ctx, writer, events)
			return
		}
		response, err := manager.Complete(ctx, contextx.MustFrom(ctx).WorkspaceID, input)
		if err != nil {
			writeChatError(c, err)
			return
		}
		c.JSON(consts.StatusOK, response)
	}
}

func writeChatSSE(ctx context.Context, writer *io.PipeWriter, events <-chan chat.StreamEvent) {
	defer writer.Close()
	for event := range events {
		payload, err := json.Marshal(event)
		if err != nil {
			continue
		}
		if _, err := io.WriteString(writer, "data: "+string(payload)+"\n\n"); err != nil {
			return
		}
		if event.Done {
			return
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

func workflowInitHandler(manager WorkflowManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeWorkflowUnavailable(c)
			return
		}
		var input workflow.InitRequest
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		value, err := manager.Init(ctx, rc.WorkspaceID, input)
		if err != nil {
			writeWorkflowError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(rc.RequestID, value))
	}
}

func workflowRunHandler(manager WorkflowManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeWorkflowUnavailable(c)
			return
		}
		var input workflow.TaskRunRequest
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		value, err := manager.Run(ctx, rc.WorkspaceID, rc.RequestID, input)
		if err != nil {
			writeWorkflowError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(rc.RequestID, value))
	}
}

func workflowProcessHandler(manager WorkflowManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeWorkflowUnavailable(c)
			return
		}
		var input struct {
			TaskID string `json:"task_id"`
		}
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		value, err := manager.Process(contextx.MustFrom(ctx).WorkspaceID, input.TaskID)
		if err != nil {
			writeWorkflowError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(contextx.MustFrom(ctx).RequestID, value))
	}
}

func workflowResumeHandler(manager WorkflowManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeWorkflowUnavailable(c)
			return
		}
		var input workflow.ResumeRequest
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		value, err := manager.Resume(ctx, rc.WorkspaceID, rc.RequestID, input)
		if err != nil {
			writeWorkflowError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(rc.RequestID, value))
	}
}

func workflowFragmentHandler(manager WorkflowManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeWorkflowUnavailable(c)
			return
		}
		var input workflow.FragmentRequest
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		value, err := manager.RunFragment(ctx, rc.WorkspaceID, rc.RequestID, input)
		if err != nil {
			writeWorkflowError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(rc.RequestID, value))
	}
}

func workflowStopHandler(manager WorkflowManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeWorkflowUnavailable(c)
			return
		}
		var input struct {
			TaskID string `json:"task_id"`
		}
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		value, err := manager.Stop(contextx.MustFrom(ctx).WorkspaceID, input.TaskID)
		if err != nil {
			writeWorkflowError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(contextx.MustFrom(ctx).RequestID, value))
	}
}

func workflowCompletionHandler(manager WorkflowManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeWorkflowUnavailable(c)
			return
		}
		var input workflow.CompletionRequest
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		if input.Stream {
			if streamer, ok := manager.(WorkflowStreamer); ok {
				_, events, unsubscribe, err := streamer.StartCompletionStream(ctx, rc.WorkspaceID, rc.RequestID, input)
				if err != nil {
					writeWorkflowError(c, err)
					return
				}
				writeWorkflowSSE(ctx, c, events, unsubscribe)
				return
			}
		}
		value, err := manager.Complete(ctx, rc.WorkspaceID, rc.RequestID, input)
		if err != nil {
			writeWorkflowError(c, err)
			return
		}
		if input.Stream {
			payload, marshalErr := json.Marshal(value)
			if marshalErr != nil {
				c.JSON(consts.StatusInternalServerError, errorEnvelope(rc.RequestID, "SYSTEM_ERROR", "encode workflow response"))
				return
			}
			c.Response.Header.SetContentType("text/event-stream")
			c.Response.Header.Set("Cache-Control", "no-cache")
			c.Response.Header.Set("X-Accel-Buffering", "no")
			c.Response.SetBodyString("data: " + string(payload) + "\n\n")
			return
		}
		c.JSON(consts.StatusOK, value)
	}
}

func workflowAsyncCompletionHandler(manager WorkflowManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeWorkflowUnavailable(c)
			return
		}
		var input workflow.CompletionRequest
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		value, err := manager.StartCompletion(ctx, rc.WorkspaceID, rc.RequestID, input)
		if err != nil {
			writeWorkflowError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(rc.RequestID, value))
	}
}

func workflowAsyncResultHandler(manager WorkflowManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeWorkflowUnavailable(c)
			return
		}
		var input struct {
			TaskID string `json:"task_id"`
		}
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		value, err := manager.AsyncResult(contextx.MustFrom(ctx).WorkspaceID, input.TaskID)
		if err != nil {
			writeWorkflowError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(contextx.MustFrom(ctx).RequestID, value))
	}
}

// workflowRunStreamHandler 保持 Java WorkflowController 的 SSE 路径和事件字段。当前 Hertz
// handler 在请求生命周期内聚合任务状态后写出有序事件；任务本身仍由 workflow 服务异步执行，
// 因此调试轮询、停止和恢复不会被流式连接阻塞。
func workflowRunStreamHandler(manager WorkflowManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeWorkflowUnavailable(c)
			return
		}
		var input workflow.TaskRunRequest
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		input.AppID = c.Param("appId")
		input.Version = "lastPublished"
		rc := contextx.MustFrom(ctx)
		if streamer, ok := manager.(WorkflowStreamer); ok {
			_, events, unsubscribe, err := streamer.RunStream(ctx, rc.WorkspaceID, rc.RequestID, input)
			if err != nil {
				writeWorkflowError(c, err)
				return
			}
			writeWorkflowSSE(ctx, c, events, unsubscribe)
			return
		}
		run, err := manager.Run(ctx, rc.WorkspaceID, rc.RequestID, input)
		if err != nil {
			writeWorkflowError(c, err)
			return
		}

		deadline := time.Now().Add(90 * time.Second)
		for {
			process, processErr := manager.Process(rc.WorkspaceID, run.TaskID)
			if processErr != nil {
				writeWorkflowError(c, processErr)
				return
			}
			if process.TaskStatus != "executing" || time.Now().After(deadline) {
				messages := make([]string, 0, len(process.TaskResults)+1)
				for index, output := range process.TaskResults {
					content := output.NodeContent
					if values, ok := content.(map[string]any); ok {
						content = values["output"]
					}
					payload, marshalErr := json.Marshal(map[string]any{
						"event":             "Message",
						"task_id":           run.TaskID,
						"conversation_id":   run.ConversationID,
						"node_id":           output.NodeID,
						"node_name":         output.NodeName,
						"node_type":         output.NodeType,
						"node_status":       output.NodeStatus,
						"node_msg_seq_id":   index + 1,
						"node_is_completed": output.NodeStatus == "success",
						"text_content":      fmt.Sprint(content),
					})
					if marshalErr == nil {
						messages = append(messages, "data: "+string(payload)+"\n\n")
					}
				}
				event := "Finished"
				additional := map[string]any{"event": event, "task_id": run.TaskID, "conversation_id": run.ConversationID}
				if process.TaskStatus == "pause" {
					additional["event"] = "Paused"
					additional["pause_type"] = "InputNodeInterrupt"
				} else if process.TaskStatus != "success" {
					additional["event"] = "Error"
					additional["error_code"] = process.ErrorCode
					additional["error_message"] = process.ErrorInfo
				}
				if payload, marshalErr := json.Marshal(additional); marshalErr == nil {
					messages = append(messages, "data: "+string(payload)+"\n\n")
				}
				c.Response.Header.SetContentType("text/event-stream")
				c.Response.Header.Set("Cache-Control", "no-cache")
				c.Response.Header.Set("X-Accel-Buffering", "no")
				c.Response.SetBodyString(strings.Join(messages, ""))
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func writeWorkflowSSE(ctx context.Context, c *app.RequestContext, events <-chan workflow.StreamEvent, unsubscribe func()) {
	c.Response.Header.SetContentType("text/event-stream")
	c.Response.Header.Set("Cache-Control", "no-cache")
	c.Response.Header.Set("X-Accel-Buffering", "no")
	reader, writer := io.Pipe()
	c.SetBodyStream(reader, -1)
	go func() {
		defer writer.Close()
		if unsubscribe != nil {
			defer unsubscribe()
		}
		for event := range events {
			payload, err := json.Marshal(event)
			if err != nil {
				continue
			}
			if _, err := io.WriteString(writer, "data: "+string(payload)+"\n\n"); err != nil {
				return
			}
			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()
}

func writeWorkflowUnavailable(c *app.RequestContext) {
	c.JSON(consts.StatusServiceUnavailable, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "workflow service is unavailable"))
}

func writeWorkflowError(c *app.RequestContext, err error) {
	switch {
	case errors.Is(err, workflow.ErrAppIDRequired), errors.Is(err, workflow.ErrTaskIDRequired), errors.Is(err, workflow.ErrResumeNodeRequired), errors.Is(err, workflow.ErrNodesRequired):
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", err.Error()))
	case errors.Is(err, workflow.ErrTaskNotFound):
		c.JSON(consts.StatusNotFound, errorEnvelope(newRequestID(), "WORKFLOW_NODE_DEBUG_FAIL", err.Error()))
	case errors.Is(err, workflow.ErrTaskNotPaused), errors.Is(err, workflow.ErrWorkflowConfig):
		c.JSON(consts.StatusConflict, errorEnvelope(newRequestID(), "WORKFLOW_CONFIG_INVALID", err.Error()))
	default:
		c.JSON(consts.StatusInternalServerError, errorEnvelope(newRequestID(), "WORKFLOW_DEBUG_FAIL", err.Error()))
	}
}

func mcpServerGetHandler(manager McpServerManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeMcpServerUnavailable(c)
			return
		}
		serverCode, ok := requiredPathParameter(c, "serverCode")
		if !ok {
			return
		}
		needTools, valid := parseBool(c.Query("need_tools"))
		if !valid {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", "need_tools is required"))
			return
		}
		value, err := manager.Get(ctx, contextx.MustFrom(ctx).WorkspaceID, serverCode, needTools)
		if err != nil {
			writeMcpServerError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func mcpServerListHandler(manager McpServerManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeMcpServerUnavailable(c)
			return
		}
		needTools, valid := optionalBool(c.Query("need_tools"), false)
		if !valid {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", "need_tools must be a boolean"))
			return
		}
		value, err := manager.List(ctx, contextx.MustFrom(ctx).WorkspaceID, c.Query("name"), needTools, int64(parsePositiveInt(c.Query("current"), 1)), int64(parsePositiveInt(c.Query("size"), 10)))
		if err != nil {
			writeMcpServerError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func mcpServerListByCodesHandler(manager McpServerManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeMcpServerUnavailable(c)
			return
		}
		var input struct {
			Codes       []string `json:"codes"`
			ServerCodes []string `json:"server_codes"`
			NeedTools   bool     `json:"need_tools"`
		}
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		codes := input.ServerCodes
		if len(codes) == 0 {
			codes = input.Codes
		}
		if len(codes) == 0 {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", "server_codes is required"))
			return
		}
		value, err := manager.ListByCodes(ctx, contextx.MustFrom(ctx).WorkspaceID, codes, input.NeedTools)
		if err != nil {
			writeMcpServerError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func mcpServerDebugToolHandler(manager McpServerManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeMcpServerUnavailable(c)
			return
		}
		var input mcpserver.ToolCallInput
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		value, err := manager.CallTool(ctx, contextx.MustFrom(ctx).WorkspaceID, input)
		if err != nil {
			writeMcpServerError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func writeMcpServerUnavailable(c *app.RequestContext) {
	c.JSON(consts.StatusServiceUnavailable, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "mcp server service is unavailable"))
}

func writeChatUnavailable(c *app.RequestContext) {
	c.JSON(consts.StatusServiceUnavailable, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "chat service is unavailable"))
}

func writeChatError(c *app.RequestContext, err error) {
	switch {
	case errors.Is(err, chat.ErrAppIDRequired), errors.Is(err, chat.ErrMessagesRequired), errors.Is(err, chat.ErrModelProviderMissing), errors.Is(err, chat.ErrModelMissing), errors.Is(err, chat.ErrCredentialMissing), errors.Is(err, chat.ErrPrivateKeyMissing):
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", err.Error()))
	case errors.Is(err, chat.ErrPublishedAppRequired):
		c.JSON(consts.StatusConflict, errorEnvelope(newRequestID(), "APP_NOT_PUBLISHED", err.Error()))
	case errors.Is(err, application.ErrNotFound):
		c.JSON(consts.StatusNotFound, errorEnvelope(newRequestID(), "APP_NOT_FOUND", err.Error()))
	default:
		c.JSON(consts.StatusInternalServerError, errorEnvelope(newRequestID(), "SYSTEM_ERROR", err.Error()))
	}
}

func writeMcpServerError(c *app.RequestContext, err error) {
	switch {
	case errors.Is(err, mcpserver.ErrNameRequired), errors.Is(err, mcpserver.ErrDeployConfigRequired), errors.Is(err, mcpserver.ErrServerCodeRequired), errors.Is(err, mcpserver.ErrToolNameRequired), errors.Is(err, mcpserver.ErrRemoteAddressRequired):
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", err.Error()))
	case errors.Is(err, mcpserver.ErrNotFound):
		c.JSON(consts.StatusNotFound, errorEnvelope(newRequestID(), "MCP_SERVER_NOT_FOUND", "MCP server can not be found."))
	default:
		c.JSON(consts.StatusInternalServerError, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "An internal error has occurred, please try again later or contact service support."))
	}
}

func skillCreateHandler(manager SkillManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeSkillUnavailable(c)
			return
		}
		var input skill.Input
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		value, err := manager.Create(ctx, rc.WorkspaceID, rc.AccountID, input)
		if err != nil {
			writeSkillError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

const (
	maxSkillPackageBytes = int64(100 * 1024 * 1024)
	maxSkillFileCount    = 1000
)

type skillPackageUploadResult struct {
	StorageType      string `json:"storage_type"`
	StoragePrefix    string `json:"storage_prefix"`
	PackageObjectKey string `json:"package_object_key"`
	MainFilePath     string `json:"main_file_path"`
	Manifest         string `json:"manifest"`
	ContentHash      string `json:"content_hash"`
	FileCount        int    `json:"file_count"`
	TotalSizeBytes   int64  `json:"total_size_bytes"`
}

func skillPackageUploadHandler(storageDir string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		form, err := c.MultipartForm()
		if err != nil {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", "multipart form is required"))
			return
		}
		files := form.File["file"]
		if len(files) != 1 || files[0].Size <= 0 {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", "file is required"))
			return
		}
		file := files[0]
		if !strings.EqualFold(filepath.Ext(file.Filename), ".zip") {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", "skill package must be a zip file"))
			return
		}
		if file.Size > maxSkillPackageBytes {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", "skill package must be smaller than 100MB"))
			return
		}
		if storageDir == "" {
			storageDir = "data/files"
		}
		rc := contextx.MustFrom(ctx)
		packageID := newRequestID()
		prefix := filepath.ToSlash(filepath.Join("skills", rc.WorkspaceID, "packages", packageID)) + "/"
		root, err := filepath.Abs(storageDir)
		if err != nil {
			c.JSON(consts.StatusInternalServerError, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "resolve skill storage failed"))
			return
		}
		packageRoot := filepath.Join(root, filepath.FromSlash(prefix))
		if !strings.HasPrefix(filepath.Clean(packageRoot), root+string(os.PathSeparator)) {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", "storage prefix is invalid"))
			return
		}
		if err := os.MkdirAll(packageRoot, 0750); err != nil {
			c.JSON(consts.StatusInternalServerError, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "create skill storage failed"))
			return
		}
		zipPath := filepath.Join(packageRoot, "package.zip")
		if err := saveMultipartFile(zipPath, file); err != nil {
			_ = os.RemoveAll(packageRoot)
			c.JSON(consts.StatusInternalServerError, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "save skill package failed"))
			return
		}
		result, err := extractSkillPackage(zipPath, packageRoot, prefix)
		if err != nil {
			_ = os.RemoveAll(packageRoot)
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", err.Error()))
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), result))
	}
}

func saveMultipartFile(targetPath string, file *multipart.FileHeader) error {
	source, err := file.Open()
	if err != nil {
		return err
	}
	defer source.Close()
	target, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0640)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(target, io.LimitReader(source, maxSkillPackageBytes+1))
	closeErr := target.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func extractSkillPackage(zipPath, packageRoot, prefix string) (*skillPackageUploadResult, error) {
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, errors.New("skill package must be a valid zip file")
	}
	defer archive.Close()
	fileCount := 0
	var totalSize int64
	mainFile := false
	manifest := "{}"
	for _, entry := range archive.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		fileCount++
		if fileCount > maxSkillFileCount {
			return nil, errors.New("skill package contains too many files")
		}
		relative, err := cleanSkillArchivePath(entry.Name)
		if err != nil {
			return nil, err
		}
		target := filepath.Join(packageRoot, filepath.FromSlash(relative))
		if !strings.HasPrefix(filepath.Clean(target), packageRoot+string(os.PathSeparator)) {
			return nil, errors.New("skill package path is invalid")
		}
		if err := os.MkdirAll(filepath.Dir(target), 0750); err != nil {
			return nil, err
		}
		source, err := entry.Open()
		if err != nil {
			return nil, err
		}
		output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0640)
		if err != nil {
			source.Close()
			return nil, err
		}
		written, copyErr := io.Copy(output, io.LimitReader(source, maxSkillPackageBytes-totalSize+1))
		closeErr, sourceCloseErr := output.Close(), source.Close()
		if copyErr != nil {
			return nil, copyErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if sourceCloseErr != nil {
			return nil, sourceCloseErr
		}
		totalSize += written
		if totalSize > maxSkillPackageBytes {
			return nil, errors.New("skill package uncompressed size must be smaller than 100MB")
		}
		if relative == "SKILL.md" {
			mainFile = true
		}
		if relative == "manifest.json" {
			data, readErr := os.ReadFile(target)
			if readErr != nil {
				return nil, readErr
			}
			if !json.Valid(data) {
				return nil, errors.New("manifest.json must be valid JSON")
			}
			manifest = string(data)
		}
	}
	if fileCount == 0 {
		return nil, errors.New("skill package is empty")
	}
	if !mainFile {
		return nil, errors.New("skill package must contain SKILL.md")
	}
	data, err := os.ReadFile(zipPath)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(data)
	return &skillPackageUploadResult{StorageType: "file", StoragePrefix: prefix, PackageObjectKey: prefix + "package.zip", MainFilePath: "SKILL.md", Manifest: manifest, ContentHash: hex.EncodeToString(hash[:]), FileCount: fileCount, TotalSizeBytes: totalSize}, nil
}

func cleanSkillArchivePath(value string) (string, error) {
	value = strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	for strings.HasPrefix(value, "./") {
		value = strings.TrimPrefix(value, "./")
	}
	if value == "" || strings.HasPrefix(value, "/") {
		return "", errors.New("skill package path is invalid")
	}
	cleaned := filepath.ToSlash(filepath.Clean(value))
	if cleaned == "." || strings.HasPrefix(cleaned, "../") || cleaned == ".." {
		return "", errors.New("skill package path is invalid")
	}
	return cleaned, nil
}

func skillUpdateHandler(manager SkillManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeSkillUnavailable(c)
			return
		}
		var input skill.Input
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		if err := manager.Update(ctx, rc.WorkspaceID, rc.AccountID, input); err != nil {
			writeSkillError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func skillDeleteHandler(manager SkillManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeSkillUnavailable(c)
			return
		}
		skillCode, ok := requiredPathParameter(c, "skillCode")
		if !ok {
			return
		}
		rc := contextx.MustFrom(ctx)
		if err := manager.Delete(ctx, rc.WorkspaceID, rc.AccountID, skillCode); err != nil {
			writeSkillError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func skillPublishHandler(manager SkillManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeSkillUnavailable(c)
			return
		}
		skillCode, ok := requiredPathParameter(c, "skillCode")
		if !ok {
			return
		}
		rc := contextx.MustFrom(ctx)
		if err := manager.Publish(ctx, rc.WorkspaceID, rc.AccountID, skillCode); err != nil {
			writeSkillError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func skillGetHandler(manager SkillManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeSkillUnavailable(c)
			return
		}
		skillCode, ok := requiredPathParameter(c, "skillCode")
		if !ok {
			return
		}
		needFiles, valid := optionalBool(c.Query("need_files"), false)
		if !valid {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", "need_files must be true or false"))
			return
		}
		value, err := manager.Get(ctx, contextx.MustFrom(ctx).WorkspaceID, skillCode, c.Query("version"), needFiles)
		if err != nil {
			writeSkillError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func skillListHandler(manager SkillManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeSkillUnavailable(c)
			return
		}
		status, valid := optionalStatus(c.Query("status"))
		if !valid {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", "status must be an integer"))
			return
		}
		value, err := manager.List(ctx, contextx.MustFrom(ctx).WorkspaceID, c.Query("name"), status, int64(parsePositiveInt(c.Query("current"), 1)), int64(parsePositiveInt(c.Query("size"), 10)))
		if err != nil {
			writeSkillError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func skillListByCodesHandler(manager SkillManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeSkillUnavailable(c)
			return
		}
		var input struct {
			SkillCodes []string `json:"skill_codes"`
			NeedFiles  bool     `json:"need_files"`
		}
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		value, err := manager.ListByCodes(ctx, contextx.MustFrom(ctx).WorkspaceID, input.SkillCodes, input.NeedFiles)
		if err != nil {
			writeSkillError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func writeSkillUnavailable(c *app.RequestContext) {
	c.JSON(consts.StatusServiceUnavailable, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "skill service is unavailable"))
}

func writeSkillError(c *app.RequestContext, err error) {
	switch {
	case errors.Is(err, skill.ErrNameRequired), errors.Is(err, skill.ErrSkillCodeRequired):
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", err.Error()))
	case errors.Is(err, skill.ErrNotFound):
		c.JSON(consts.StatusNotFound, errorEnvelope(newRequestID(), "SKILL_NOT_FOUND", "Skill can not be found."))
	default:
		c.JSON(consts.StatusInternalServerError, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "An internal error has occurred, please try again later or contact service support."))
	}
}

func optionalBool(value string, fallback bool) (bool, bool) {
	if strings.TrimSpace(value) == "" {
		return fallback, true
	}
	return parseBool(value)
}

func optionalStatus(value string) (int16, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return -1, true
	}
	negative := false
	if strings.HasPrefix(value, "-") {
		negative, value = true, value[1:]
	}
	if value == "" {
		return 0, false
	}
	var result int16
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return 0, false
		}
		result = result*10 + int16(digit-'0')
	}
	if negative {
		result = -result
	}
	return result, true
}

func pluginCreateHandler(manager PluginManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writePluginUnavailable(c)
			return
		}
		var input plugin.PluginInput
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		value, err := manager.Create(ctx, rc.WorkspaceID, rc.AccountID, input)
		if err != nil {
			writePluginError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func pluginListHandler(manager PluginManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writePluginUnavailable(c)
			return
		}
		value, err := manager.List(ctx, contextx.MustFrom(ctx).WorkspaceID, c.Query("name"), int64(parsePositiveInt(c.Query("current"), 1)), int64(parsePositiveInt(c.Query("size"), 10)))
		if err != nil {
			writePluginError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func pluginGetHandler(manager PluginManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writePluginUnavailable(c)
			return
		}
		pluginID, ok := requiredPathParameter(c, "pluginId")
		if !ok {
			return
		}
		value, err := manager.Get(ctx, contextx.MustFrom(ctx).WorkspaceID, pluginID)
		if err != nil {
			writePluginError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func pluginUpdateHandler(manager PluginManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writePluginUnavailable(c)
			return
		}
		pluginID, ok := requiredPathParameter(c, "pluginId")
		if !ok {
			return
		}
		var input plugin.PluginInput
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		if err := manager.Update(ctx, rc.WorkspaceID, rc.AccountID, pluginID, input); err != nil {
			writePluginError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func pluginDeleteHandler(manager PluginManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writePluginUnavailable(c)
			return
		}
		pluginID, ok := requiredPathParameter(c, "pluginId")
		if !ok {
			return
		}
		if err := manager.Delete(ctx, contextx.MustFrom(ctx).WorkspaceID, pluginID); err != nil {
			writePluginError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func pluginToolCreateHandler(manager PluginManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writePluginUnavailable(c)
			return
		}
		pluginID, ok := requiredPathParameter(c, "pluginId")
		if !ok {
			return
		}
		var input plugin.ToolInput
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		value, err := manager.CreateTool(ctx, rc.WorkspaceID, rc.AccountID, pluginID, input)
		if err != nil {
			writePluginError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func pluginToolListHandler(manager PluginManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writePluginUnavailable(c)
			return
		}
		pluginID, ok := requiredPathParameter(c, "pluginId")
		if !ok {
			return
		}
		value, err := manager.ListTools(ctx, contextx.MustFrom(ctx).WorkspaceID, pluginID, c.Query("name"), int64(parsePositiveInt(c.Query("current"), 1)), int64(parsePositiveInt(c.Query("size"), 10)))
		if err != nil {
			writePluginError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func pluginToolGetHandler(manager PluginManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writePluginUnavailable(c)
			return
		}
		pluginID, ok := requiredPathParameter(c, "pluginId")
		if !ok {
			return
		}
		toolID, ok := requiredPathParameter(c, "toolId")
		if !ok {
			return
		}
		value, err := manager.GetTool(ctx, contextx.MustFrom(ctx).WorkspaceID, pluginID, toolID)
		if err != nil {
			writePluginError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func pluginToolUpdateHandler(manager PluginManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writePluginUnavailable(c)
			return
		}
		pluginID, ok := requiredPathParameter(c, "pluginId")
		if !ok {
			return
		}
		toolID, ok := requiredPathParameter(c, "toolId")
		if !ok {
			return
		}
		var input plugin.ToolInput
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		if err := manager.UpdateTool(ctx, rc.WorkspaceID, rc.AccountID, pluginID, toolID, input); err != nil {
			writePluginError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func pluginToolDeleteHandler(manager PluginManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writePluginUnavailable(c)
			return
		}
		pluginID, ok := requiredPathParameter(c, "pluginId")
		if !ok {
			return
		}
		toolID, ok := requiredPathParameter(c, "toolId")
		if !ok {
			return
		}
		if err := manager.DeleteTool(ctx, contextx.MustFrom(ctx).WorkspaceID, pluginID, toolID); err != nil {
			writePluginError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func pluginToolEnabledHandler(manager PluginManager, enabled bool) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writePluginUnavailable(c)
			return
		}
		toolID, ok := requiredPathParameter(c, "toolId")
		if !ok {
			return
		}
		rc := contextx.MustFrom(ctx)
		if err := manager.SetEnabled(ctx, rc.WorkspaceID, rc.AccountID, toolID, enabled); err != nil {
			writePluginError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func pluginToolPublishHandler(manager PluginManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writePluginUnavailable(c)
			return
		}
		pluginID, ok := requiredPathParameter(c, "pluginId")
		if !ok {
			return
		}
		toolID, ok := requiredPathParameter(c, "toolId")
		if !ok {
			return
		}
		rc := contextx.MustFrom(ctx)
		if err := manager.PublishTool(ctx, rc.WorkspaceID, rc.AccountID, pluginID, toolID); err != nil {
			writePluginError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), nil))
	}
}

func pluginToolTestHandler(manager PluginManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writePluginUnavailable(c)
			return
		}
		pluginID, ok := requiredPathParameter(c, "pluginId")
		if !ok {
			return
		}
		toolID, ok := requiredPathParameter(c, "toolId")
		if !ok {
			return
		}
		var input struct {
			Arguments map[string]any `json:"arguments"`
		}
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		value, err := manager.TestTool(ctx, rc.WorkspaceID, rc.AccountID, pluginID, toolID, input.Arguments)
		if err != nil {
			writePluginError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func pluginToolsByIDsHandler(manager PluginManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writePluginUnavailable(c)
			return
		}
		var input struct {
			ToolIDs []string `json:"tool_ids"`
		}
		if json.Unmarshal(c.Request.Body(), &input) != nil {
			writeInvalidJSON(c)
			return
		}
		if len(input.ToolIDs) == 0 {
			c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", "tool_ids is required"))
			return
		}
		value, err := manager.ListToolsByIDs(ctx, contextx.MustFrom(ctx).WorkspaceID, input.ToolIDs)
		if err != nil {
			writePluginError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}

func writePluginUnavailable(c *app.RequestContext) {
	c.JSON(consts.StatusServiceUnavailable, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "plugin service is unavailable"))
}
func writePluginError(c *app.RequestContext, err error) {
	switch {
	case errors.Is(err, plugin.ErrNameRequired), errors.Is(err, plugin.ErrDescriptionRequired), errors.Is(err, plugin.ErrConfigRequired):
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", err.Error()))
	case errors.Is(err, plugin.ErrPluginNameExists), errors.Is(err, plugin.ErrToolNameExists):
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_REQUEST", err.Error()))
	case errors.Is(err, plugin.ErrPluginNotFound):
		c.JSON(consts.StatusNotFound, errorEnvelope(newRequestID(), "PLUGIN_NOT_FOUND", "Plugin can not be found."))
	case errors.Is(err, plugin.ErrToolNotFound):
		c.JSON(consts.StatusNotFound, errorEnvelope(newRequestID(), "TOOL_NOT_FOUND", "Tool can not be found."))
	case errors.Is(err, plugin.ErrToolNotTested):
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_REQUEST", err.Error()))
	default:
		c.JSON(consts.StatusInternalServerError, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "An internal error has occurred, please try again later or contact service support."))
	}
}

type fileUploadPolicy struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	Extension   string `json:"extension"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
}

func saveUploadedFile(storageDir, accountID, category string, file *multipart.FileHeader) (fileUploadPolicy, error) {
	if storageDir == "" {
		storageDir = "data/files"
	}
	category = filepath.Base(category)
	if category == "." || category == "" {
		return fileUploadPolicy{}, errors.New("invalid category")
	}
	name := filepath.Base(file.Filename)
	extension := strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), ".")
	path := filepath.Join(storageDir, accountID, category, newRequestID()+filepath.Ext(name))
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return fileUploadPolicy{}, err
	}
	source, err := file.Open()
	if err != nil {
		return fileUploadPolicy{}, err
	}
	defer source.Close()
	target, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0640)
	if err != nil {
		return fileUploadPolicy{}, err
	}
	_, err = io.Copy(target, source)
	closeErr := target.Close()
	if err != nil {
		return fileUploadPolicy{}, err
	}
	if closeErr != nil {
		return fileUploadPolicy{}, closeErr
	}
	return fileUploadPolicy{Path: filepath.ToSlash(filepath.Join(accountID, category, filepath.Base(path))), Name: name, Extension: extension, ContentType: file.Header.Get("Content-Type"), Size: file.Size}, nil
}

func saveUploadedFileAt(storageDir, accountID, category, key string, file *multipart.FileHeader) (fileUploadPolicy, error) {
	key = filepath.ToSlash(filepath.Clean(filepath.FromSlash(key)))
	prefix := filepath.ToSlash(filepath.Join(accountID, category)) + "/"
	if !strings.HasPrefix(key, prefix) || strings.Contains(key, "..") {
		return fileUploadPolicy{}, errors.New("invalid upload key")
	}
	path, ok := resolveStoredFile(storageDir, key)
	if !ok {
		return fileUploadPolicy{}, errors.New("invalid upload key")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return fileUploadPolicy{}, err
	}
	source, err := file.Open()
	if err != nil {
		return fileUploadPolicy{}, err
	}
	defer source.Close()
	target, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0640)
	if err != nil {
		return fileUploadPolicy{}, err
	}
	_, copyErr := io.Copy(target, source)
	closeErr := target.Close()
	if copyErr != nil {
		return fileUploadPolicy{}, copyErr
	}
	if closeErr != nil {
		return fileUploadPolicy{}, closeErr
	}
	name := filepath.Base(file.Filename)
	return fileUploadPolicy{Path: key, Name: name, Extension: strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), "."), ContentType: file.Header.Get("Content-Type"), Size: file.Size}, nil
}
func resolveStoredFile(storageDir, value string) (string, bool) {
	value = filepath.Clean(filepath.FromSlash(strings.TrimSpace(value)))
	if value == "." || filepath.IsAbs(value) || strings.HasPrefix(value, "..") {
		return "", false
	}
	root, err := filepath.Abs(storageDir)
	if err != nil {
		return "", false
	}
	path, err := filepath.Abs(filepath.Join(root, value))
	if err != nil || !strings.HasPrefix(path, root+string(os.PathSeparator)) {
		return "", false
	}
	return path, true
}

func providerListHandler(manager ModelProviderManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeModelProviderUnavailable(c)
			return
		}
		items, err := manager.ListProviders(ctx, contextx.MustFrom(ctx).WorkspaceID, c.Query("name"))
		if err != nil {
			writeModelProviderError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), items))
	}
}

func providerGetHandler(manager ModelProviderManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeModelProviderUnavailable(c)
			return
		}
		providerCode, ok := requiredPathParameter(c, "provider")
		if !ok {
			return
		}
		item, err := manager.GetProvider(ctx, contextx.MustFrom(ctx).WorkspaceID, providerCode)
		if err != nil {
			writeModelProviderError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), item))
	}
}

func providerUpdateHandler(manager ModelProviderManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeModelProviderUnavailable(c)
			return
		}
		providerCode, ok := requiredPathParameter(c, "provider")
		if !ok {
			return
		}
		var input modelprovider.UpdateProviderInput
		if err := json.Unmarshal(c.Request.Body(), &input); err != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		if err := manager.UpdateProvider(ctx, rc.WorkspaceID, rc.AccountID, providerCode, input); err != nil {
			writeModelProviderError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), true))
	}
}

func providerDeleteHandler(manager ModelProviderManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeModelProviderUnavailable(c)
			return
		}
		providerCode, ok := requiredPathParameter(c, "provider")
		if !ok {
			return
		}
		if err := manager.DeleteProvider(ctx, contextx.MustFrom(ctx).WorkspaceID, providerCode); err != nil {
			writeModelProviderError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), true))
	}
}

func providerProtocolsHandler(_ context.Context, c *app.RequestContext) {
	c.JSON(consts.StatusOK, successEnvelope(newRequestID(), []string{"OpenAI"}))
}

func providerModelCreateHandler(manager ModelProviderManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeModelProviderUnavailable(c)
			return
		}
		providerCode, ok := requiredPathParameter(c, "provider")
		if !ok {
			return
		}
		var input modelprovider.CreateModelInput
		if err := json.Unmarshal(c.Request.Body(), &input); err != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		if err := manager.CreateModel(ctx, rc.WorkspaceID, rc.AccountID, providerCode, input); err != nil {
			writeModelProviderError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), true))
	}
}

func providerModelListHandler(manager ModelProviderManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeModelProviderUnavailable(c)
			return
		}
		providerCode, ok := requiredPathParameter(c, "provider")
		if !ok {
			return
		}
		items, err := manager.ListModels(ctx, contextx.MustFrom(ctx).WorkspaceID, providerCode)
		if err != nil {
			writeModelProviderError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), items))
	}
}

func providerModelGetHandler(manager ModelProviderManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeModelProviderUnavailable(c)
			return
		}
		providerCode, modelID, ok := requiredProviderModelParameters(c)
		if !ok {
			return
		}
		item, err := manager.GetModel(ctx, contextx.MustFrom(ctx).WorkspaceID, providerCode, modelID)
		if err != nil {
			writeModelProviderError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), item))
	}
}

func providerModelUpdateHandler(manager ModelProviderManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeModelProviderUnavailable(c)
			return
		}
		providerCode, modelID, ok := requiredProviderModelParameters(c)
		if !ok {
			return
		}
		var input modelprovider.UpdateModelInput
		if err := json.Unmarshal(c.Request.Body(), &input); err != nil {
			writeInvalidJSON(c)
			return
		}
		rc := contextx.MustFrom(ctx)
		if err := manager.UpdateModel(ctx, rc.WorkspaceID, rc.AccountID, providerCode, modelID, input); err != nil {
			writeModelProviderError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), true))
	}
}

func providerModelDeleteHandler(manager ModelProviderManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeModelProviderUnavailable(c)
			return
		}
		providerCode, modelID, ok := requiredProviderModelParameters(c)
		if !ok {
			return
		}
		if err := manager.DeleteModel(ctx, contextx.MustFrom(ctx).WorkspaceID, providerCode, modelID); err != nil {
			writeModelProviderError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), true))
	}
}

func providerModelParameterRulesHandler(manager ModelProviderManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			writeModelProviderUnavailable(c)
			return
		}
		providerCode, modelID, ok := requiredProviderModelParameters(c)
		if !ok {
			return
		}
		items, err := manager.ParameterRules(ctx, contextx.MustFrom(ctx).WorkspaceID, providerCode, modelID)
		if err != nil {
			writeModelProviderError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), items))
	}
}

func modelSelectorHandler(manager ModelProviderManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			// 兼容无数据库的轻量启动与健康检查场景；真实启动会注入持久化服务。
			c.JSON(consts.StatusOK, successEnvelope(newRequestID(), []modelprovider.SelectorGroup{}))
			return
		}
		items, err := manager.Selector(ctx, contextx.MustFrom(ctx).WorkspaceID, strings.TrimSpace(c.Param("modelType")))
		if err != nil {
			writeModelProviderError(c, err)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), items))
	}
}

func requiredPathParameter(c *app.RequestContext, name string) (string, bool) {
	value := strings.TrimSpace(c.Param(name))
	if value == "" {
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", name+" is required"))
		return "", false
	}
	return value, true
}

func requiredProviderModelParameters(c *app.RequestContext) (string, string, bool) {
	providerCode, ok := requiredPathParameter(c, "provider")
	if !ok {
		return "", "", false
	}
	modelID, ok := requiredPathParameter(c, "modelId")
	if !ok {
		return "", "", false
	}
	return providerCode, modelID, true
}

func writeInvalidJSON(c *app.RequestContext) {
	c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_JSON", "request body must be valid json"))
}

func writeModelProviderUnavailable(c *app.RequestContext) {
	c.JSON(consts.StatusServiceUnavailable, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "model provider service is unavailable"))
}

func writeModelProviderError(c *app.RequestContext, err error) {
	switch {
	case errors.Is(err, modelprovider.ErrProviderNameRequired), errors.Is(err, modelprovider.ErrModelIDRequired):
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", "required provider or model parameter is missing"))
	case errors.Is(err, modelprovider.ErrProviderExists), errors.Is(err, modelprovider.ErrModelExists):
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", "provider or model already exists"))
	case errors.Is(err, modelprovider.ErrProviderNotFound):
		c.JSON(consts.StatusNotFound, errorEnvelope(newRequestID(), "INVALID_PARAMS", "provider not found"))
	case errors.Is(err, modelprovider.ErrModelNotFound):
		c.JSON(consts.StatusNotFound, errorEnvelope(newRequestID(), "MODEL_NOT_FOUND", "model not found"))
	default:
		c.JSON(consts.StatusInternalServerError, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "An internal error has occurred, please try again later or contact service support."))
	}
}

func writeAccountUnavailable(c *app.RequestContext) {
	c.JSON(consts.StatusServiceUnavailable, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "account service is unavailable"))
}
func writeAccountError(c *app.RequestContext, err error) {
	switch {
	case errors.Is(err, account.ErrPermissionDenied):
		c.JSON(consts.StatusForbidden, errorEnvelope(newRequestID(), "PERMISSION_DENIED", "Permission denied."))
	case errors.Is(err, account.ErrUsernameRequired), errors.Is(err, account.ErrPasswordRequired):
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", "required account parameter is missing"))
	case errors.Is(err, account.ErrUsernameExists):
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "ACCOUNT_NAME_EXISTS", "Account name already exists."))
	case errors.Is(err, account.ErrPasswordNotMatch):
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "ACCOUNT_PASSWORD_NOT_MATCH", "Account password does not match."))
	case errors.Is(err, account.ErrNotFound):
		c.JSON(consts.StatusNotFound, errorEnvelope(newRequestID(), "ACCOUNT_NOT_FOUND", "Account can not be found."))
	default:
		c.JSON(consts.StatusInternalServerError, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "An internal error has occurred, please try again later or contact service support."))
	}
}

func writeWorkspaceUnavailable(c *app.RequestContext) {
	c.JSON(consts.StatusServiceUnavailable, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "workspace service is unavailable"))
}
func writeWorkspaceError(c *app.RequestContext, err error) {
	switch {
	case errors.Is(err, workspace.ErrNameRequired):
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", "name is required"))
	case errors.Is(err, workspace.ErrNameExists):
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "WORKSPACE_NAME_EXISTS", "Workspace name already exists."))
	case errors.Is(err, workspace.ErrLimitExceeded):
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_REQUEST", "workspace can not be more than 10."))
	case errors.Is(err, workspace.ErrNotFound):
		c.JSON(consts.StatusNotFound, errorEnvelope(newRequestID(), "WORKSPACE_NOT_FOUND", "Workspace can not be found."))
	default:
		c.JSON(consts.StatusInternalServerError, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "An internal error has occurred, please try again later or contact service support."))
	}
}

func optionalText(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func writeAPIKeyError(c *app.RequestContext, err error) {
	switch {
	case errors.Is(err, apikey.ErrDescriptionRequired):
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", "description is required"))
	case errors.Is(err, apikey.ErrLimitExceeded):
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_REQUEST", "api key can not be more than 20."))
	case errors.Is(err, apikey.ErrNotFound):
		c.JSON(consts.StatusNotFound, errorEnvelope(newRequestID(), "API_KEY_NOT_FOUND", "API key can not be found."))
	default:
		c.JSON(consts.StatusInternalServerError, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "An internal error has occurred, please try again later or contact service support."))
	}
}

func parsePathID(c *app.RequestContext) (int64, bool) {
	value := strings.TrimSpace(c.Param("id"))
	if value == "" {
		return 0, false
	}
	var id int64
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return 0, false
		}
		id = id*10 + int64(digit-'0')
	}
	return id, id > 0
}

func apiModelsHandler(manager *compat.AdminService) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		page := int64(parsePositiveInt(c.Query("page"), 1))
		size := int64(parsePositiveInt(c.Query("size"), 10))
		if manager == nil {
			c.JSON(consts.StatusOK, successEnvelope(newRequestID(), emptyLegacyPage(page, size)))
			return
		}
		value, err := manager.List(ctx, "model", "gmt_modified", page, size)
		legacyResult(c, value, err)
	}
}

func exampleOrderGetHandler(_ context.Context, c *app.RequestContext) {
	// Java 的 GET 示例只返回订单概要；items 仅属于两个 POST 示例接口。
	c.JSON(consts.StatusOK, exampleOrderSummary("100001"))
}

func exampleOrderPostHandler(_ context.Context, c *app.RequestContext) {
	var body struct {
		OrderID string `json:"orderId"`
	}
	if err := json.Unmarshal(c.Request.Body(), &body); err != nil {
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_JSON", "request body must be valid json"))
		return
	}
	if strings.TrimSpace(body.OrderID) == "" {
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", "orderId is required"))
		return
	}
	c.JSON(consts.StatusOK, exampleOrderResponse(body.OrderID, true))
}

func exampleOrderPathHandler(_ context.Context, c *app.RequestContext) {
	orderID := strings.TrimSpace(c.Param("orderId"))
	if orderID == "" {
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "MISSING_PARAMS", "orderId is required"))
		return
	}
	c.JSON(consts.StatusOK, exampleOrderResponse(orderID, false))
}

func exampleOrderResponse(orderID string, includeSecondItem bool) map[string]any {
	items := []map[string]any{{
		"itemId":   "2001",
		"itemName": "羽毛球拍",
		"price":    199.5,
	}}
	if includeSecondItem {
		items = append(items, map[string]any{
			"itemId":   "2002",
			"itemName": "亚狮龙7号",
			"price":    99.5,
		})
	}
	response := exampleOrderSummary(orderID)
	response["items"] = items
	return response
}

func exampleOrderSummary(orderID string) map[string]any {
	return map[string]any{
		"orderId":     orderID,
		"description": "订单详情为尤尼克斯羽毛球拍",
		"data": map[string]any{
			"company": "顺丰快递",
			"city":    "苏州",
		},
	}
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

func parseBool(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true":
		return true, true
	case "false":
		return false, true
	default:
		return false, false
	}
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

type apiKeyDescriptionRequest struct {
	Description string `json:"description"`
}

type workspaceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Config      string `json:"config"`
}

type changePasswordRequest struct {
	Password    string `json:"password"`
	NewPassword string `json:"new_password"`
}

type adminPage[T any] struct {
	TotalCount int64 `json:"totalCount"`
	TotalPage  int64 `json:"totalPage"`
	PageNumber int64 `json:"pageNumber"`
	PageSize   int64 `json:"pageSize"`
	PageItems  []T   `json:"pageItems"`
}
