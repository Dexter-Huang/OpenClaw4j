package dao

import (
	"context"
	"time"
)

type Account struct {
	ID           int64
	AccountID    string
	Username     string
	Email        *string
	Mobile       *string
	Password     string
	Nickname     *string
	Icon         *string
	Type         string
	Status       int16
	GmtCreate    time.Time
	GmtModified  time.Time
	GmtLastLogin *time.Time
	Creator      string
	Modifier     string
	TenantID     int64
}

type Workspace struct {
	ID          int64
	WorkspaceID string
	AccountID   string
	Status      int16
	Name        string
	Description *string
	Config      *string
	GmtCreate   time.Time
	GmtModified time.Time
	Creator     string
	Modifier    string
	TenantID    int64
}

type APIKey struct {
	ID          int64
	AccountID   string
	APIKey      string
	Status      int16
	Description *string
	GmtCreate   time.Time
	GmtModified time.Time
	Creator     string
	Modifier    string
	TenantID    int64
}

type TokenSession struct {
	ID          int64
	TokenID     string
	AccountID   string
	TokenType   string
	TokenHash   string
	ExpiresAt   time.Time
	Revoked     int16
	Source      *string
	CallerIP    *string
	UserAgent   *string
	GmtCreate   time.Time
	GmtModified time.Time
	TenantID    int64
}

type Provider struct {
	ID                  int64
	WorkspaceID         string
	Icon                *string
	Name                *string
	Description         *string
	Provider            string
	Enable              bool
	Source              string
	Credential          *string
	SupportedModelTypes *string
	Protocol            *string
	GmtCreate           time.Time
	GmtModified         time.Time
	Creator             *string
	Modifier            *string
	TenantID            int64
}

type Model struct {
	ID          int64
	WorkspaceID string
	Icon        *string
	Name        *string
	Type        *string
	Mode        *string
	ModelID     string
	Provider    string
	Enable      bool
	Tags        *string
	Source      string
	GmtCreate   time.Time
	GmtModified time.Time
	Creator     *string
	Modifier    *string
	TenantID    int64
}

type Application struct {
	ID          int64
	WorkspaceID string
	AppID       string
	Name        string
	Description *string
	Icon        *string
	Source      string
	Type        string
	Status      int16
	GmtCreate   time.Time
	GmtModified time.Time
	Creator     string
	Modifier    string
	TenantID    int64
}

type ApplicationVersion struct {
	ID          int64
	AppID       string
	WorkspaceID string
	Config      *string
	Status      int16
	Version     string
	Description *string
	GmtCreate   time.Time
	GmtModified time.Time
	Creator     string
	Modifier    string
	TenantID    int64
}

type AgentSchema struct {
	ID          int64
	AgentID     *string
	WorkspaceID string
	Name        string
	Description *string
	Type        string
	Instruction *string
	InputKeys   *string
	OutputKey   *string
	Handle      *string
	SubAgents   *string
	YamlSchema  *string
	Status      string
	Enabled     bool
	GmtCreate   time.Time
	GmtModified time.Time
	Creator     string
	Modifier    string
	TenantID    int64
}

type Tool struct {
	ID          int64
	PluginID    string
	ToolID      string
	WorkspaceID string
	Status      int16
	Enabled     bool
	TestStatus  int16
	Name        string
	Description *string
	Config      string
	APISchema   string
	GmtCreate   time.Time
	GmtModified time.Time
	Creator     string
	Modifier    string
	TenantID    int64
}
type KnowledgeBase struct {
	ID                                                    int64
	WorkspaceID, KbID, Type, Name                         string
	Status                                                int16
	Description, ProcessConfig, IndexConfig, SearchConfig *string
	TotalDocs                                             int64
	GmtCreate, GmtModified                                time.Time
	Creator, Modifier                                     string
	TenantID                                              int64
}

type McpServer struct {
	ServerCode   string
	WorkspaceID  string
	AccountID    string
	Name         string
	DeployConfig string
	DetailConfig *string
	Status       int16
	Type         string
	BizType      *string
	Description  *string
	InstallType  *string
	DeployEnv    *string
	Source       *string
	Host         *string
	GmtCreate    time.Time
	GmtModified  time.Time
	TenantID     int64
}

type Skill struct {
	SkillCode   string
	WorkspaceID string
	AccountID   *string
	Name        string
	Description *string
	Source      string
	Status      int16
	Tags        *string
	GmtCreate   time.Time
	GmtModified time.Time
	Creator     *string
	Modifier    *string
	TenantID    int64
}

type SkillVersion struct {
	SkillCode        string
	WorkspaceID      string
	Version          string
	Description      *string
	MainFilePath     string
	Manifest         *string
	ContentHash      *string
	StorageType      string
	StorageBucket    *string
	StoragePrefix    string
	PackageObjectKey *string
	FileCount        *int32
	TotalSizeBytes   *int64
	Status           int16
	GmtCreate        time.Time
	GmtModified      time.Time
	Creator          *string
	Modifier         *string
	TenantID         int64
}

type Plugin struct {
	PluginID    string
	WorkspaceID string
	Type        string
	Status      int16
	Name        string
	Description *string
	Config      *string
	Source      string
	GmtCreate   time.Time
	GmtModified time.Time
	Creator     string
	Modifier    string
	TenantID    int64
}

type Document struct {
	WorkspaceID   string
	KbID          string
	DocID         string
	Type          string
	Status        int16
	Enabled       bool
	Name          string
	Format        string
	Size          int64
	Metadata      *string
	IndexStatus   int16
	Path          string
	ParsedPath    *string
	ProcessConfig *string
	Source        *string
	Error         *string
	GmtCreate     time.Time
	GmtModified   time.Time
	Creator       string
	Modifier      string
	TenantID      int64
}

type DocumentChunk struct {
	WorkspaceID string
	KbID        string
	DocID       string
	ChunkID     string
	DocName     string
	Title       string
	Text        string
	Score       *float64
	PageNumber  *int32
	Enabled     bool
	GmtCreate   time.Time
	GmtModified time.Time
	Creator     string
	Modifier    string
	TenantID    int64
}

type ApplicationComponent struct {
	Code        string
	Name        string
	WorkspaceID string
	Type        string
	AppID       *string
	Config      *string
	Description *string
	Status      int16
	GmtCreate   time.Time
	GmtModified time.Time
	Creator     *string
	Modifier    *string
	TenantID    int64
}

type AccountDAO interface {
	Create(ctx context.Context, account Account) error
	FindActiveByAccountID(ctx context.Context, accountID string) (*Account, error)
	FindActiveByUsername(ctx context.Context, username string) (*Account, error)
	Update(ctx context.Context, account Account) error
	SoftDelete(ctx context.Context, accountID string, modifier string, modifiedAt time.Time) error
	ListActiveUsers(ctx context.Context, name string, limit int32, offset int32) ([]Account, error)
	CountActiveUsers(ctx context.Context, name string) (int64, error)
	UpdateLastLogin(ctx context.Context, accountID string, lastLogin time.Time) error
}

type WorkspaceDAO interface {
	Create(ctx context.Context, workspace Workspace) error
	FindDefaultByAccountID(ctx context.Context, accountID string) (*Workspace, error)
	FindActiveByIDAndAccountID(ctx context.Context, workspaceID string, accountID string) (*Workspace, error)
	FindActiveByNameAndAccountID(ctx context.Context, name string, accountID string) (*Workspace, error)
	CountActiveByAccountID(ctx context.Context, accountID string) (int64, error)
	ListActiveByAccountID(ctx context.Context, accountID string, limit int32, offset int32) ([]Workspace, error)
	Update(ctx context.Context, workspace Workspace) error
	SoftDelete(ctx context.Context, workspaceID string, accountID string, modifier string, modifiedAt time.Time) error
}

type APIKeyDAO interface {
	Create(ctx context.Context, apiKey APIKey) (int64, error)
	FindActiveByEncryptedKey(ctx context.Context, encryptedKey string) (*APIKey, error)
	FindActiveByHash(ctx context.Context, keyHash string) (*APIKey, error)
	FindActiveByIDAndAccountID(ctx context.Context, id int64, accountID string) (*APIKey, error)
	UpdateDescription(ctx context.Context, id int64, accountID string, description string, modifier string, modifiedAt time.Time) error
	SoftDelete(ctx context.Context, id int64, accountID string, modifier string, modifiedAt time.Time) error
	CountActiveByAccountID(ctx context.Context, accountID string) (int64, error)
	ListActiveByAccountID(ctx context.Context, accountID string, limit int32, offset int32) ([]APIKey, error)
}

type TokenSessionDAO interface {
	Create(ctx context.Context, session TokenSession) error
	FindActiveByToken(ctx context.Context, token string, tokenType string, now time.Time) (*TokenSession, error)
	RevokeByToken(ctx context.Context, token string) error
	RevokeAccountTokens(ctx context.Context, accountID string, tokenType string) error
	DeleteExpired(ctx context.Context, before time.Time) error
}

type ProviderDAO interface {
	Create(ctx context.Context, provider Provider) error
	FindByCodeAndWorkspace(ctx context.Context, provider, workspaceID string) (*Provider, error)
	ListByWorkspace(ctx context.Context, workspaceID, name string) ([]Provider, error)
	Update(ctx context.Context, provider Provider) error
	Delete(ctx context.Context, provider, workspaceID string) error
	CountModels(ctx context.Context, provider, workspaceID string) (int64, error)
}

type ModelDAO interface {
	Create(ctx context.Context, model Model) error
	FindByProviderAndIDAndWorkspace(ctx context.Context, provider, modelID, workspaceID string) (*Model, error)
	ListByProviderAndWorkspace(ctx context.Context, provider, workspaceID string) ([]Model, error)
	ListByWorkspace(ctx context.Context, workspaceID string) ([]Model, error)
	Update(ctx context.Context, model Model) error
	Delete(ctx context.Context, provider, modelID, workspaceID string) error
}

type ApplicationDAO interface {
	Create(ctx context.Context, application Application) error
	CreateVersion(ctx context.Context, version ApplicationVersion) error
	FindByID(ctx context.Context, appID, workspaceID string) (*Application, error)
	FindByName(ctx context.Context, name, workspaceID string) (*Application, error)
	FindLatestVersion(ctx context.Context, appID, workspaceID string) (*ApplicationVersion, error)
	FindLastPublishedVersion(ctx context.Context, appID, workspaceID string) (*ApplicationVersion, error)
	FindVersion(ctx context.Context, appID, workspaceID, version string) (*ApplicationVersion, error)
	List(ctx context.Context, workspaceID, name, appType string, status int16, limit, offset int32) ([]Application, error)
	Count(ctx context.Context, workspaceID, name, appType string, status int16) (int64, error)
	ListVersions(ctx context.Context, appID, workspaceID string, status int16, limit, offset int32) ([]ApplicationVersion, error)
	CountVersions(ctx context.Context, appID, workspaceID string, status int16) (int64, error)
	Update(ctx context.Context, application Application) error
	UpdateVersion(ctx context.Context, version ApplicationVersion) error
	Delete(ctx context.Context, appID, workspaceID, modifier string, modifiedAt time.Time) error
}

type AgentSchemaDAO interface {
	Create(context.Context, AgentSchema) (int64, error)
	Find(context.Context, int64, string) (*AgentSchema, error)
	FindByName(context.Context, string, string) (*AgentSchema, error)
	List(context.Context, string, string) ([]AgentSchema, error)
	Update(context.Context, AgentSchema) error
	Delete(context.Context, int64, string) error
}

type ToolDAO interface {
	Create(context.Context, Tool) (int64, error)
	Find(context.Context, int64, string) (*Tool, error)
	List(context.Context, string, string, string) ([]Tool, error)
	Update(context.Context, Tool) error
	Delete(context.Context, int64, string) error
}
type KnowledgeBaseDAO interface {
	Create(context.Context, KnowledgeBase) error
	Find(context.Context, string, string) (*KnowledgeBase, error)
	List(context.Context, string, string) ([]KnowledgeBase, error)
	Update(context.Context, KnowledgeBase) error
	Delete(context.Context, string, string, string, time.Time) error
}

type McpServerDAO interface {
	Create(context.Context, McpServer) error
	Find(context.Context, string, string) (*McpServer, error)
	List(context.Context, string, string) ([]McpServer, error)
	Update(context.Context, McpServer) error
	Delete(context.Context, string, string, time.Time) error
}

type SkillDAO interface {
	Create(context.Context, Skill) error
	CreateVersion(context.Context, SkillVersion) error
	Find(context.Context, string, string) (*Skill, error)
	FindVersion(context.Context, string, string, string) (*SkillVersion, error)
	FindLatestVersion(context.Context, string, string) (*SkillVersion, error)
	List(context.Context, string, string, int16) ([]Skill, error)
	Update(context.Context, Skill) error
	UpdateVersion(context.Context, SkillVersion) error
	Delete(context.Context, string, string, string, time.Time) error
}

type PluginDAO interface {
	Create(context.Context, Plugin) error
	Find(context.Context, string, string) (*Plugin, error)
	List(context.Context, string, string, int16) ([]Plugin, error)
	Update(context.Context, Plugin) error
	Delete(context.Context, string, string) error
	CreateTool(context.Context, Tool) error
	FindTool(context.Context, string, string, string) (*Tool, error)
	FindToolByID(context.Context, string, string) (*Tool, error)
	ListTools(context.Context, string, string, string) ([]Tool, error)
	ListToolsByIDs(context.Context, string, []string) ([]Tool, error)
	UpdateTool(context.Context, Tool) error
	DeleteTool(context.Context, string, string, string) error
	DeleteTools(context.Context, string, string) error
}

type DocumentDAO interface {
	Create(context.Context, Document) error
	Find(context.Context, string, string, string) (*Document, error)
	List(context.Context, string, string, string, int16) ([]Document, error)
	Update(context.Context, Document) error
	Delete(context.Context, string, string, string, string, time.Time) error
	FindByDocID(context.Context, string, string) (*Document, error)
	CreateChunk(context.Context, DocumentChunk) error
	FindChunk(context.Context, string, string, string) (*DocumentChunk, error)
	ListChunks(context.Context, string, string) ([]DocumentChunk, error)
	UpdateChunk(context.Context, DocumentChunk) error
	SetChunksEnabled(context.Context, string, string, []string, bool, string, time.Time) error
	DeleteChunks(context.Context, string, string, []string, string, time.Time) error
}

type ApplicationComponentDAO interface {
	Create(context.Context, ApplicationComponent) error
	Find(context.Context, string, string) (*ApplicationComponent, error)
	FindByAppID(context.Context, string, string) (*ApplicationComponent, error)
	List(context.Context, string, string, string, string, int16) ([]ApplicationComponent, error)
	Update(context.Context, ApplicationComponent) error
	Delete(context.Context, string, string, string, time.Time) error
}
