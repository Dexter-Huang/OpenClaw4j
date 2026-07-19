package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/seaskyland/openclaw4j-backend-go/internal/scriptsandbox"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/seaskyland/openclaw4j-backend-go/internal/account"
	"github.com/seaskyland/openclaw4j-backend-go/internal/agent"
	"github.com/seaskyland/openclaw4j-backend-go/internal/agentschema"
	"github.com/seaskyland/openclaw4j-backend-go/internal/apikey"
	"github.com/seaskyland/openclaw4j-backend-go/internal/appcomponent"
	"github.com/seaskyland/openclaw4j-backend-go/internal/application"
	"github.com/seaskyland/openclaw4j-backend-go/internal/auth"
	"github.com/seaskyland/openclaw4j-backend-go/internal/chat"
	"github.com/seaskyland/openclaw4j-backend-go/internal/compat"
	"github.com/seaskyland/openclaw4j-backend-go/internal/dao"
	"github.com/seaskyland/openclaw4j-backend-go/internal/dao/pgdao"
	"github.com/seaskyland/openclaw4j-backend-go/internal/db"
	"github.com/seaskyland/openclaw4j-backend-go/internal/document"
	"github.com/seaskyland/openclaw4j-backend-go/internal/knowledgebase"
	"github.com/seaskyland/openclaw4j-backend-go/internal/mcpserver"
	"github.com/seaskyland/openclaw4j-backend-go/internal/modelprovider"
	"github.com/seaskyland/openclaw4j-backend-go/internal/oauth2"
	"github.com/seaskyland/openclaw4j-backend-go/internal/plugin"
	"github.com/seaskyland/openclaw4j-backend-go/internal/rag"
	"github.com/seaskyland/openclaw4j-backend-go/internal/skill"
	"github.com/seaskyland/openclaw4j-backend-go/internal/tool"
	authmiddleware "github.com/seaskyland/openclaw4j-backend-go/internal/transport/hertz/middleware"
	hertzserver "github.com/seaskyland/openclaw4j-backend-go/internal/transport/hertz/server"
	"github.com/seaskyland/openclaw4j-backend-go/internal/workflow"
	"github.com/seaskyland/openclaw4j-backend-go/internal/workspace"
)

var ErrMissingDatabase = errors.New("database is required when authenticator is not provided")

type Options struct {
	DB                     db.DBTX
	TokenSessions          dao.TokenSessionDAO
	AgentMemory            agent.MemoryStore
	HertzOptions           []config.Option
	Authenticator          authmiddleware.Authenticator
	AuthService            *auth.Service
	LegacyAPIKeyEncryptor  auth.LegacyAPIKeyEncryptor
	RegisterConsoleRoutes  func(group *route.RouterGroup)
	RegisterAPIRoutes      func(group *route.RouterGroup)
	FileStorageDir         string
	ProviderPrivateKeyFile string
	GitHubOAuth2Provider   *oauth2.GitHubService
	ScriptExecutor         scriptsandbox.Executor
	WorkflowState          workflow.StateStore
}

type App struct {
	HTTP                 *server.Hertz
	Authenticator        authmiddleware.Authenticator
	AuthService          *auth.Service
	APIKeyService        *apikey.Service
	WorkspaceService     *workspace.Service
	AccountService       *account.Service
	ModelProviderService *modelprovider.Service
	ApplicationService   *application.Service
	AppComponentService  *appcomponent.Service
	AgentSchemaService   *agentschema.Service
	ToolService          *tool.Service
	KnowledgeBaseService *knowledgebase.Service
	DocumentService      *document.Service
	McpServerService     *mcpserver.Service
	SkillService         *skill.Service
	PluginService        *plugin.Service
	ChatService          *chat.Service
	AgentService         *agent.Service
	WorkflowService      *workflow.Service
	LegacyAdminService   *compat.AdminService
	ObservabilityService *compat.ObservabilityService
	GitHubOAuth2Provider *oauth2.GitHubService
}

func New(options Options) (*App, error) {
	authService := options.AuthService
	if authService == nil && options.DB != nil {
		authService = buildAuthService(options.DB, options.TokenSessions, options.LegacyAPIKeyEncryptor)
	}

	authenticator := options.Authenticator
	if authenticator == nil {
		if authService == nil {
			if options.DB == nil {
				return nil, ErrMissingDatabase
			}
			authService = buildAuthService(options.DB, options.TokenSessions, options.LegacyAPIKeyEncryptor)
		}
		authenticator = authService
	}
	if authService == nil {
		if service, ok := authenticator.(*auth.Service); ok {
			authService = service
		}
	}
	if authenticator == nil {
		if options.DB == nil {
			return nil, ErrMissingDatabase
		}
		authenticator = buildAuthService(options.DB, options.TokenSessions, options.LegacyAPIKeyEncryptor)
		if authService == nil {
			if service, ok := authenticator.(*auth.Service); ok {
				authService = service
			}
		}
	}

	apiKeyService := buildAPIKeyService(options.DB, options.LegacyAPIKeyEncryptor)
	workspaceService := buildWorkspaceService(options.DB)
	accountService := buildAccountService(options.DB)
	modelProviderService := buildModelProviderService(options.DB)
	applicationService := buildApplicationService(options.DB)
	appComponentService := buildAppComponentService(options.DB, applicationService)
	agentSchemaService := buildAgentSchemaService(options.DB)
	toolService := buildToolService(options.DB)
	knowledgeBaseService := buildKnowledgeBaseService(options.DB)
	documentService := buildDocumentService(options.DB, options.ProviderPrivateKeyFile, options.FileStorageDir)
	mcpServerService := buildMcpServerService(options.DB)
	skillService := buildSkillService(options.DB, options.FileStorageDir)
	pluginService := buildPluginService(options.DB)
	chatService := buildChatService(options.DB, applicationService, options.ProviderPrivateKeyFile)
	agentService := agent.NewWithAgentRuntime(chatService, options.AgentMemory, 20, applicationService, documentService, pluginService, mcpServerService, skillService)
	workflowService := buildWorkflowService(applicationService, chatService, options.ScriptExecutor, documentService, mcpServerService, pluginService, appComponentService, options.WorkflowState)
	agentService.SetComponentTools(&componentToolExecutor{components: appComponentService, agents: agentService, workflows: workflowService})
	legacyAdminService := compat.NewAdminService(options.DB)
	observabilityService := compat.NewObservabilityService()
	httpServer := hertzserver.New(hertzserver.Options{
		HertzOptions:           options.HertzOptions,
		Authenticator:          authenticator,
		SessionIssuer:          authService,
		AccountProfileProvider: authService,
		APIKeyManager:          apiKeyService,
		WorkspaceManager:       workspaceService,
		AccountManager:         accountService,
		ModelProviderManager:   modelProviderService,
		ApplicationManager:     applicationService,
		AppComponentManager:    appComponentService,
		AgentSchemaManager:     agentSchemaService,
		ToolManager:            toolService,
		KnowledgeBaseManager:   knowledgeBaseService,
		DocumentManager:        documentService,
		McpServerManager:       mcpServerService,
		SkillManager:           skillService,
		PluginManager:          pluginService,
		ChatManager:            agentService,
		WorkflowManager:        workflowService,
		OAuth2Provider:         options.GitHubOAuth2Provider,
		LegacyAdminManager:     legacyAdminService,
		ObservabilityManager:   observabilityService,
		FileStorageDir:         options.FileStorageDir,
		RegisterConsoleRoutes:  options.RegisterConsoleRoutes,
		RegisterAPIRoutes:      options.RegisterAPIRoutes,
	})

	return &App{
		HTTP:                 httpServer,
		Authenticator:        authenticator,
		AuthService:          authService,
		APIKeyService:        apiKeyService,
		WorkspaceService:     workspaceService,
		AccountService:       accountService,
		ModelProviderService: modelProviderService,
		ApplicationService:   applicationService,
		AppComponentService:  appComponentService,
		AgentSchemaService:   agentSchemaService,
		ToolService:          toolService,
		KnowledgeBaseService: knowledgeBaseService,
		DocumentService:      documentService,
		McpServerService:     mcpServerService,
		SkillService:         skillService,
		PluginService:        pluginService,
		ChatService:          chatService,
		AgentService:         agentService,
		WorkflowService:      workflowService,
		LegacyAdminService:   legacyAdminService,
		ObservabilityService: observabilityService,
		GitHubOAuth2Provider: options.GitHubOAuth2Provider,
	}, nil
}

type componentToolExecutor struct {
	components *appcomponent.Service
	agents     *agent.Service
	workflows  *workflow.Service
}

func (e *componentToolExecutor) List(ctx context.Context, workspaceID string, codes []string, expectedType string) ([]agent.ComponentTool, error) {
	if e.components == nil {
		return nil, nil
	}
	components, err := e.components.ListByCodes(ctx, workspaceID, codes)
	if err != nil {
		return nil, err
	}
	result := make([]agent.ComponentTool, 0, len(components))
	for _, component := range components {
		if !strings.EqualFold(component.Type, expectedType) {
			continue
		}
		schema, schemaErr := e.components.Schema(ctx, workspaceID, component.Code)
		if schemaErr != nil {
			return nil, schemaErr
		}
		result = append(result, agent.ComponentTool{Code: component.Code, Name: component.Name, Description: component.Description, Parameters: componentParameters(schema)})
	}
	return result, nil
}

func (e *componentToolExecutor) Invoke(ctx context.Context, workspaceID, code string, arguments map[string]any) (any, error) {
	if e.components == nil {
		return nil, errors.New("application component service is unavailable")
	}
	component, err := e.components.Get(ctx, workspaceID, code)
	if err != nil {
		return nil, err
	}
	query := componentQuery(arguments)
	switch strings.ToLower(component.Type) {
	case "agent":
		if e.agents == nil {
			return nil, errors.New("agent service is unavailable")
		}
		response, callErr := e.agents.Complete(ctx, workspaceID, chat.Request{AppID: component.AppID, Messages: []chat.Message{{Role: "user", Content: query}}, Parameter: arguments})
		if callErr != nil {
			return nil, callErr
		}
		return response.Message.Content, nil
	case "workflow":
		if e.workflows == nil {
			return nil, errors.New("workflow service is unavailable")
		}
		params := make([]workflow.Param, 0, len(arguments))
		for key, value := range arguments {
			params = append(params, workflow.Param{Key: key, Value: value})
		}
		response, callErr := e.workflows.Complete(ctx, workspaceID, fmt.Sprintf("component-%d", time.Now().UnixNano()), workflow.CompletionRequest{AppID: component.AppID, Messages: []chat.Message{{Role: "user", Content: query}}, InputParams: params})
		if callErr != nil {
			return nil, callErr
		}
		if response.Error != nil {
			return nil, errors.New(response.Error.Message)
		}
		return response.Message.Content, nil
	default:
		return nil, fmt.Errorf("unsupported application component type: %s", component.Type)
	}
}

func componentParameters(schema map[string]any) map[string]any {
	parameters := map[string]any{"type": "object", "properties": map[string]any{}}
	properties := parameters["properties"].(map[string]any)
	if items, ok := schema["input"].([]any); ok {
		for _, item := range items {
			value, ok := item.(map[string]any)
			if !ok {
				continue
			}
			name := strings.TrimSpace(fmt.Sprint(value["key"]))
			if name == "" {
				name = strings.TrimSpace(fmt.Sprint(value["name"]))
			}
			if name == "" || name == "<nil>" {
				continue
			}
			properties[name] = map[string]any{"type": "string", "description": strings.TrimSpace(fmt.Sprint(value["description"]))}
		}
	}
	return parameters
}

func componentQuery(arguments map[string]any) string {
	if query, ok := arguments["query"].(string); ok && strings.TrimSpace(query) != "" {
		return query
	}
	data, _ := json.Marshal(arguments)
	return string(data)
}

func buildAccountService(database db.DBTX) *account.Service {
	if database == nil {
		return nil
	}
	return account.NewService(pgdao.NewAccountDAO(db.New(database)), buildWorkspaceService(database), nil)
}

func buildWorkspaceService(database db.DBTX) *workspace.Service {
	if database == nil {
		return nil
	}
	return workspace.NewService(pgdao.NewWorkspaceDAO(db.New(database)), nil)
}

func buildAPIKeyService(database db.DBTX, cipher auth.LegacyAPIKeyEncryptor) *apikey.Service {
	if database == nil || cipher == nil {
		return nil
	}
	return apikey.NewService(pgdao.NewAPIKeyDAO(db.New(database)), cipher, nil)
}

func buildModelProviderService(database db.DBTX) *modelprovider.Service {
	if database == nil {
		return nil
	}
	queries := db.New(database)
	return modelprovider.NewService(pgdao.NewProviderDAO(queries), pgdao.NewModelDAO(queries), nil)
}

func buildApplicationService(database db.DBTX) *application.Service {
	if database == nil {
		return nil
	}
	return application.NewService(pgdao.NewApplicationDAO(db.New(database)), nil)
}

func buildAppComponentService(database db.DBTX, apps *application.Service) *appcomponent.Service {
	if database == nil || apps == nil {
		return nil
	}
	return appcomponent.NewService(pgdao.NewApplicationComponentDAO(db.New(database)), apps, nil)
}

func buildAgentSchemaService(database db.DBTX) *agentschema.Service {
	if database == nil {
		return nil
	}
	return agentschema.NewService(pgdao.NewAgentSchemaDAO(db.New(database)), nil)
}

func buildToolService(database db.DBTX) *tool.Service {
	if database == nil {
		return nil
	}
	return tool.NewService(pgdao.NewToolDAO(db.New(database)), nil)
}
func buildKnowledgeBaseService(database db.DBTX) *knowledgebase.Service {
	if database == nil {
		return nil
	}
	return knowledgebase.NewService(pgdao.NewKnowledgeBaseDAO(db.New(database)), nil)
}

func buildDocumentService(database db.DBTX, privateKeyFile, fileStorageDir string) *document.Service {
	if database == nil {
		return nil
	}
	queries := db.New(database)
	return document.NewServiceWithRAG(pgdao.NewDocumentDAO(queries), pgdao.NewKnowledgeBaseDAO(queries), rag.NewOpenAIEmbedder(pgdao.NewProviderDAO(queries), pgdao.NewModelDAO(queries), privateKeyFile, nil), rag.NewPGVectorStore(database), fileStorageDir, nil)
}

func buildMcpServerService(database db.DBTX) *mcpserver.Service {
	if database == nil {
		return nil
	}
	return mcpserver.NewService(pgdao.NewMcpServerDAO(db.New(database)), nil)
}

func buildSkillService(database db.DBTX, fileStorageDir string) *skill.Service {
	if database == nil {
		return nil
	}
	return skill.NewServiceWithStorage(pgdao.NewSkillDAO(db.New(database)), nil, fileStorageDir)
}

func buildPluginService(database db.DBTX) *plugin.Service {
	if database == nil {
		return nil
	}
	return plugin.NewService(pgdao.NewPluginDAO(db.New(database)), nil)
}

func buildChatService(database db.DBTX, apps *application.Service, privateKeyFile string) *chat.Service {
	if database == nil || apps == nil {
		return nil
	}
	queries := db.New(database)
	return chat.NewService(apps, pgdao.NewProviderDAO(queries), pgdao.NewModelDAO(queries), privateKeyFile, nil)
}

func buildWorkflowService(apps *application.Service, models *chat.Service, scripts scriptsandbox.Executor, documents *document.Service, mcpServers *mcpserver.Service, plugins *plugin.Service, components *appcomponent.Service, state workflow.StateStore) *workflow.Service {
	if apps == nil {
		return nil
	}
	return workflow.NewServiceWithNodeServicesAndState(apps, models, nil, scripts, documents, mcpServers, plugins, components, state)
}

func buildAuthService(database db.DBTX, tokenSessions dao.TokenSessionDAO, legacyAPIKeyEncryptor auth.LegacyAPIKeyEncryptor) *auth.Service {
	queries := db.New(database)
	return auth.NewService(auth.ServiceOptions{
		Accounts:              pgdao.NewAccountDAO(queries),
		Workspaces:            pgdao.NewWorkspaceDAO(queries),
		APIKeys:               pgdao.NewAPIKeyDAO(queries),
		TokenSessions:         tokenSessions,
		LegacyAPIKeyEncryptor: legacyAPIKeyEncryptor,
	})
}
