package pgdao

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/seaskyland/openclaw4j-backend-go/internal/dao"
	"github.com/seaskyland/openclaw4j-backend-go/internal/db"
)

func (d *AccountDAO) Create(ctx context.Context, v dao.Account) error {
	return d.q.CreateAccount(ctx, db.CreateAccountParams{AccountID: v.AccountID, Username: v.Username, Email: pgOptionalText(v.Email), Mobile: pgOptionalText(v.Mobile), Password: v.Password, Nickname: pgOptionalText(v.Nickname), Icon: pgOptionalText(v.Icon), Type: v.Type, Status: v.Status, GmtCreate: pgTimestamp(v.GmtCreate), GmtModified: pgTimestamp(v.GmtModified), Creator: v.Creator, Modifier: v.Modifier, TenantID: pgOptionalInt8(v.TenantID)})
}
func (d *AccountDAO) Update(ctx context.Context, v dao.Account) error {
	return d.q.UpdateAccount(ctx, db.UpdateAccountParams{AccountID: v.AccountID, Email: pgOptionalText(v.Email), Mobile: pgOptionalText(v.Mobile), Password: v.Password, Nickname: pgOptionalText(v.Nickname), Icon: pgOptionalText(v.Icon), GmtModified: pgTimestamp(v.GmtModified), Modifier: v.Modifier})
}
func (d *AccountDAO) SoftDelete(ctx context.Context, id, m string, at time.Time) error {
	return d.q.SoftDeleteAccount(ctx, db.SoftDeleteAccountParams{AccountID: id, Modifier: m, GmtModified: pgTimestamp(at)})
}
func (d *AccountDAO) ListActiveUsers(ctx context.Context, n string, l, o int32) ([]dao.Account, error) {
	rows, e := d.q.ListActiveUserAccounts(ctx, db.ListActiveUserAccountsParams{Column1: n, Limit: l, Offset: o})
	if e != nil {
		return nil, e
	}
	out := make([]dao.Account, 0, len(rows))
	for _, v := range rows {
		out = append(out, *mapAccount(v))
	}
	return out, nil
}
func (d *AccountDAO) CountActiveUsers(ctx context.Context, n string) (int64, error) {
	return d.q.CountActiveUserAccounts(ctx, n)
}
func (d *WorkspaceDAO) Create(ctx context.Context, v dao.Workspace) error {
	return d.q.CreateWorkspace(ctx, db.CreateWorkspaceParams{WorkspaceID: v.WorkspaceID, AccountID: v.AccountID, Status: v.Status, Name: v.Name, Description: pgOptionalText(v.Description), Config: pgOptionalText(v.Config), GmtCreate: pgTimestamp(v.GmtCreate), GmtModified: pgTimestamp(v.GmtModified), Creator: v.Creator, Modifier: v.Modifier, TenantID: pgOptionalInt8(v.TenantID)})
}
func (d *WorkspaceDAO) FindActiveByIDAndAccountID(ctx context.Context, id, a string) (*dao.Workspace, error) {
	v, e := d.q.FindActiveWorkspaceByIDAndAccountID(ctx, db.FindActiveWorkspaceByIDAndAccountIDParams{WorkspaceID: id, AccountID: a})
	if e != nil {
		return nil, convertNotFound(e)
	}
	return mapWorkspace(v), nil
}
func (d *WorkspaceDAO) FindActiveByNameAndAccountID(ctx context.Context, n, a string) (*dao.Workspace, error) {
	v, e := d.q.FindActiveWorkspaceByNameAndAccountID(ctx, db.FindActiveWorkspaceByNameAndAccountIDParams{Name: n, AccountID: a})
	if e != nil {
		return nil, convertNotFound(e)
	}
	return mapWorkspace(v), nil
}
func (d *WorkspaceDAO) CountActiveByAccountID(ctx context.Context, a string) (int64, error) {
	return d.q.CountActiveWorkspacesByAccountID(ctx, a)
}
func (d *WorkspaceDAO) ListActiveByAccountID(ctx context.Context, a string, l, o int32) ([]dao.Workspace, error) {
	rows, e := d.q.ListActiveWorkspacesByAccountID(ctx, db.ListActiveWorkspacesByAccountIDParams{AccountID: a, Limit: l, Offset: o})
	if e != nil {
		return nil, e
	}
	out := make([]dao.Workspace, 0, len(rows))
	for _, v := range rows {
		out = append(out, *mapWorkspace(v))
	}
	return out, nil
}
func (d *WorkspaceDAO) Update(ctx context.Context, v dao.Workspace) error {
	return d.q.UpdateWorkspace(ctx, db.UpdateWorkspaceParams{WorkspaceID: v.WorkspaceID, AccountID: v.AccountID, Name: v.Name, Description: pgOptionalText(v.Description), Config: pgOptionalText(v.Config), Modifier: v.Modifier, GmtModified: pgTimestamp(v.GmtModified)})
}
func (d *WorkspaceDAO) SoftDelete(ctx context.Context, id, a, m string, at time.Time) error {
	return d.q.SoftDeleteWorkspace(ctx, db.SoftDeleteWorkspaceParams{WorkspaceID: id, AccountID: a, Modifier: m, GmtModified: pgTimestamp(at)})
}
func (d *APIKeyDAO) Create(ctx context.Context, v dao.APIKey) (int64, error) {
	return d.q.CreateAPIKey(ctx, db.CreateAPIKeyParams{AccountID: v.AccountID, ApiKey: v.APIKey, Status: v.Status, Description: pgOptionalText(v.Description), GmtCreate: pgTimestamp(v.GmtCreate), GmtModified: pgTimestamp(v.GmtModified), Creator: v.Creator, Modifier: v.Modifier, TenantID: pgOptionalInt8(v.TenantID)})
}
func (d *APIKeyDAO) FindActiveByIDAndAccountID(ctx context.Context, id int64, a string) (*dao.APIKey, error) {
	v, e := d.q.FindActiveAPIKeyByIDAndAccountID(ctx, db.FindActiveAPIKeyByIDAndAccountIDParams{ID: id, AccountID: a})
	if e != nil {
		return nil, convertNotFound(e)
	}
	return mapAPIKey(v.ID, v.AccountID, v.ApiKey, v.Status, v.Description, v.GmtCreate, v.GmtModified, v.Creator, v.Modifier, v.TenantID), nil
}
func (d *APIKeyDAO) UpdateDescription(ctx context.Context, id int64, a, s, m string, at time.Time) error {
	return d.q.UpdateAPIKeyDescription(ctx, db.UpdateAPIKeyDescriptionParams{ID: id, AccountID: a, Description: pgText(s), Modifier: m, GmtModified: pgTimestamp(at)})
}
func (d *APIKeyDAO) SoftDelete(ctx context.Context, id int64, a, m string, at time.Time) error {
	return d.q.SoftDeleteAPIKey(ctx, db.SoftDeleteAPIKeyParams{ID: id, AccountID: a, Modifier: m, GmtModified: pgTimestamp(at)})
}

type ProviderDAO struct{ q db.Querier }

func NewProviderDAO(q db.Querier) *ProviderDAO { return &ProviderDAO{q: q} }
func (d *ProviderDAO) Create(c context.Context, v dao.Provider) error {
	return d.q.CreateProvider(c, db.CreateProviderParams{WorkspaceID: pgText(v.WorkspaceID), Icon: pgOptionalText(v.Icon), Name: pgOptionalText(v.Name), Description: pgOptionalText(v.Description), Provider: v.Provider, Enable: boolInt2(v.Enable), Source: v.Source, Credential: pgOptionalText(v.Credential), SupportedModelTypes: pgOptionalText(v.SupportedModelTypes), Protocol: pgOptionalText(v.Protocol), GmtCreate: pgTimestamp(v.GmtCreate), GmtModified: pgTimestamp(v.GmtModified), Creator: pgOptionalText(v.Creator), Modifier: pgOptionalText(v.Modifier), TenantID: pgOptionalInt8(v.TenantID)})
}
func (d *ProviderDAO) FindByCodeAndWorkspace(c context.Context, p, w string) (*dao.Provider, error) {
	v, e := d.q.FindProviderByCodeAndWorkspace(c, db.FindProviderByCodeAndWorkspaceParams{Provider: p, WorkspaceID: pgText(w)})
	if e != nil {
		return nil, convertNotFound(e)
	}
	return providerMap(v), nil
}
func (d *ProviderDAO) ListByWorkspace(c context.Context, w, n string) ([]dao.Provider, error) {
	rows, e := d.q.ListProvidersByWorkspace(c, db.ListProvidersByWorkspaceParams{WorkspaceID: pgText(w), Column2: n})
	out := make([]dao.Provider, 0, len(rows))
	for _, v := range rows {
		out = append(out, *providerMap(v))
	}
	return out, e
}
func (d *ProviderDAO) Update(c context.Context, v dao.Provider) error {
	return d.q.UpdateProvider(c, db.UpdateProviderParams{Provider: v.Provider, WorkspaceID: pgText(v.WorkspaceID), Icon: pgOptionalText(v.Icon), Name: pgOptionalText(v.Name), Description: pgOptionalText(v.Description), Enable: boolInt2(v.Enable), Credential: pgOptionalText(v.Credential), SupportedModelTypes: pgOptionalText(v.SupportedModelTypes), Protocol: pgOptionalText(v.Protocol), GmtModified: pgTimestamp(v.GmtModified), Modifier: pgOptionalText(v.Modifier)})
}
func (d *ProviderDAO) Delete(c context.Context, p, w string) error {
	return d.q.DeleteProvider(c, db.DeleteProviderParams{Provider: p, WorkspaceID: pgText(w)})
}
func (d *ProviderDAO) CountModels(c context.Context, p, w string) (int64, error) {
	return d.q.CountModelsByProviderAndWorkspace(c, db.CountModelsByProviderAndWorkspaceParams{Provider: p, WorkspaceID: pgText(w)})
}
func boolInt2(v bool) pgtype.Int2 {
	if v {
		return pgtype.Int2{Int16: 1, Valid: true}
	}
	return pgtype.Int2{Int16: 0, Valid: true}
}
func providerMap(v db.Provider) *dao.Provider {
	return &dao.Provider{ID: v.ID, WorkspaceID: textString(v.WorkspaceID), Icon: optionalString(v.Icon), Name: optionalString(v.Name), Description: optionalString(v.Description), Provider: v.Provider, Enable: v.Enable.Valid && v.Enable.Int16 != 0, Source: v.Source, Credential: optionalString(v.Credential), SupportedModelTypes: optionalString(v.SupportedModelTypes), Protocol: optionalString(v.Protocol), GmtCreate: v.GmtCreate.Time, GmtModified: v.GmtModified.Time, Creator: optionalString(v.Creator), Modifier: optionalString(v.Modifier), TenantID: optionalInt8(v.TenantID)}
}
func textString(v pgtype.Text) string {
	if v.Valid {
		return v.String
	}
	return ""
}

type ModelDAO struct{ q db.Querier }

func NewModelDAO(q db.Querier) *ModelDAO { return &ModelDAO{q: q} }

func (d *ModelDAO) Create(ctx context.Context, value dao.Model) error {
	return d.q.CreateModel(ctx, db.CreateModelParams{
		WorkspaceID: pgText(value.WorkspaceID), Icon: pgOptionalText(value.Icon), Name: pgOptionalText(value.Name),
		Type: pgOptionalText(value.Type), Mode: pgOptionalText(value.Mode), ModelID: value.ModelID, Provider: value.Provider,
		Enable: boolInt2(value.Enable), Tags: pgOptionalText(value.Tags), Source: value.Source,
		GmtCreate: pgTimestamp(value.GmtCreate), GmtModified: pgTimestamp(value.GmtModified),
		Creator: pgOptionalText(value.Creator), Modifier: pgOptionalText(value.Modifier), TenantID: pgOptionalInt8(value.TenantID),
	})
}

func (d *ModelDAO) FindByProviderAndIDAndWorkspace(ctx context.Context, provider, modelID, workspaceID string) (*dao.Model, error) {
	value, err := d.q.FindModelByProviderAndIDAndWorkspace(ctx, db.FindModelByProviderAndIDAndWorkspaceParams{
		Provider: provider, ModelID: modelID, WorkspaceID: pgText(workspaceID),
	})
	if err != nil {
		return nil, convertNotFound(err)
	}
	return modelMap(value), nil
}

func (d *ModelDAO) ListByProviderAndWorkspace(ctx context.Context, provider, workspaceID string) ([]dao.Model, error) {
	rows, err := d.q.ListModelsByProviderAndWorkspace(ctx, db.ListModelsByProviderAndWorkspaceParams{Provider: provider, WorkspaceID: pgText(workspaceID)})
	if err != nil {
		return nil, err
	}
	return modelList(rows), nil
}

func (d *ModelDAO) ListByWorkspace(ctx context.Context, workspaceID string) ([]dao.Model, error) {
	rows, err := d.q.ListModelsByWorkspace(ctx, pgText(workspaceID))
	if err != nil {
		return nil, err
	}
	return modelList(rows), nil
}

func (d *ModelDAO) Update(ctx context.Context, value dao.Model) error {
	return d.q.UpdateModel(ctx, db.UpdateModelParams{
		Provider: value.Provider, ModelID: value.ModelID, WorkspaceID: pgText(value.WorkspaceID),
		Icon: pgOptionalText(value.Icon), Name: pgOptionalText(value.Name), Enable: boolInt2(value.Enable), Tags: pgOptionalText(value.Tags),
		GmtModified: pgTimestamp(value.GmtModified), Modifier: pgOptionalText(value.Modifier),
	})
}

func (d *ModelDAO) Delete(ctx context.Context, provider, modelID, workspaceID string) error {
	return d.q.DeleteModel(ctx, db.DeleteModelParams{Provider: provider, ModelID: modelID, WorkspaceID: pgText(workspaceID)})
}

func modelList(values []db.Model) []dao.Model {
	result := make([]dao.Model, 0, len(values))
	for _, value := range values {
		result = append(result, *modelMap(value))
	}
	return result
}

func modelMap(value db.Model) *dao.Model {
	return &dao.Model{
		ID: value.ID, WorkspaceID: textString(value.WorkspaceID), Icon: optionalString(value.Icon), Name: optionalString(value.Name),
		Type: optionalString(value.Type), Mode: optionalString(value.Mode), ModelID: value.ModelID, Provider: value.Provider,
		Enable: value.Enable.Valid && value.Enable.Int16 != 0, Tags: optionalString(value.Tags), Source: value.Source,
		GmtCreate: value.GmtCreate.Time, GmtModified: value.GmtModified.Time, Creator: optionalString(value.Creator),
		Modifier: optionalString(value.Modifier), TenantID: optionalInt8(value.TenantID),
	}
}

type ApplicationDAO struct{ q db.Querier }

func NewApplicationDAO(q db.Querier) *ApplicationDAO { return &ApplicationDAO{q: q} }

func (d *ApplicationDAO) Create(ctx context.Context, value dao.Application) error {
	return d.q.CreateApplication(ctx, db.CreateApplicationParams{
		WorkspaceID: value.WorkspaceID, AppID: value.AppID, Name: value.Name, Description: pgOptionalText(value.Description),
		Icon: pgOptionalText(value.Icon), Source: value.Source, Type: value.Type, Status: value.Status,
		GmtCreate: pgTimestamp(value.GmtCreate), GmtModified: pgTimestamp(value.GmtModified),
		Creator: value.Creator, Modifier: value.Modifier, TenantID: pgOptionalInt8(value.TenantID),
	})
}

func (d *ApplicationDAO) CreateVersion(ctx context.Context, value dao.ApplicationVersion) error {
	return d.q.CreateApplicationVersion(ctx, db.CreateApplicationVersionParams{
		AppID: value.AppID, WorkspaceID: value.WorkspaceID, Config: pgOptionalText(value.Config), Status: value.Status,
		Version: value.Version, Description: pgOptionalText(value.Description), GmtCreate: pgTimestamp(value.GmtCreate),
		GmtModified: pgTimestamp(value.GmtModified), Creator: value.Creator, Modifier: value.Modifier, TenantID: pgOptionalInt8(value.TenantID),
	})
}

func (d *ApplicationDAO) FindByID(ctx context.Context, appID, workspaceID string) (*dao.Application, error) {
	value, err := d.q.FindActiveApplicationByIDAndWorkspace(ctx, db.FindActiveApplicationByIDAndWorkspaceParams{AppID: appID, WorkspaceID: workspaceID})
	if err != nil {
		return nil, convertNotFound(err)
	}
	return applicationMap(value), nil
}

func (d *ApplicationDAO) FindByName(ctx context.Context, name, workspaceID string) (*dao.Application, error) {
	value, err := d.q.FindActiveApplicationByNameAndWorkspace(ctx, db.FindActiveApplicationByNameAndWorkspaceParams{Name: name, WorkspaceID: workspaceID})
	if err != nil {
		return nil, convertNotFound(err)
	}
	return applicationMap(value), nil
}

func (d *ApplicationDAO) FindLatestVersion(ctx context.Context, appID, workspaceID string) (*dao.ApplicationVersion, error) {
	value, err := d.q.FindLatestActiveApplicationVersion(ctx, db.FindLatestActiveApplicationVersionParams{AppID: appID, WorkspaceID: workspaceID})
	if err != nil {
		return nil, convertNotFound(err)
	}
	return applicationVersionMap(value), nil
}

func (d *ApplicationDAO) FindLastPublishedVersion(ctx context.Context, appID, workspaceID string) (*dao.ApplicationVersion, error) {
	value, err := d.q.FindLastPublishedApplicationVersion(ctx, db.FindLastPublishedApplicationVersionParams{AppID: appID, WorkspaceID: workspaceID})
	if err != nil {
		return nil, convertNotFound(err)
	}
	return applicationVersionMap(value), nil
}

func (d *ApplicationDAO) FindVersion(ctx context.Context, appID, workspaceID, version string) (*dao.ApplicationVersion, error) {
	value, err := d.q.FindApplicationVersionByVersion(ctx, db.FindApplicationVersionByVersionParams{AppID: appID, WorkspaceID: workspaceID, Version: version})
	if err != nil {
		return nil, convertNotFound(err)
	}
	return applicationVersionMap(value), nil
}

func (d *ApplicationDAO) List(ctx context.Context, workspaceID, name, appType string, status int16, limit, offset int32) ([]dao.Application, error) {
	rows, err := d.q.ListActiveApplicationsByWorkspace(ctx, db.ListActiveApplicationsByWorkspaceParams{WorkspaceID: workspaceID, Column2: name, Column3: appType, Column4: status, Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	result := make([]dao.Application, 0, len(rows))
	for _, value := range rows {
		result = append(result, *applicationMap(value))
	}
	return result, nil
}

func (d *ApplicationDAO) Count(ctx context.Context, workspaceID, name, appType string, status int16) (int64, error) {
	return d.q.CountActiveApplicationsByWorkspace(ctx, db.CountActiveApplicationsByWorkspaceParams{WorkspaceID: workspaceID, Column2: name, Column3: appType, Column4: status})
}

func (d *ApplicationDAO) ListVersions(ctx context.Context, appID, workspaceID string, status int16, limit, offset int32) ([]dao.ApplicationVersion, error) {
	rows, err := d.q.ListActiveApplicationVersions(ctx, db.ListActiveApplicationVersionsParams{AppID: appID, WorkspaceID: workspaceID, Column3: status, Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	result := make([]dao.ApplicationVersion, 0, len(rows))
	for _, value := range rows {
		result = append(result, *applicationVersionMap(value))
	}
	return result, nil
}

func (d *ApplicationDAO) CountVersions(ctx context.Context, appID, workspaceID string, status int16) (int64, error) {
	return d.q.CountActiveApplicationVersions(ctx, db.CountActiveApplicationVersionsParams{AppID: appID, WorkspaceID: workspaceID, Column3: status})
}

func (d *ApplicationDAO) Update(ctx context.Context, value dao.Application) error {
	return d.q.UpdateApplication(ctx, db.UpdateApplicationParams{AppID: value.AppID, WorkspaceID: value.WorkspaceID, Name: value.Name, Description: pgOptionalText(value.Description), Icon: pgOptionalText(value.Icon), Type: value.Type, Status: value.Status, GmtModified: pgTimestamp(value.GmtModified), Modifier: value.Modifier})
}

func (d *ApplicationDAO) UpdateVersion(ctx context.Context, value dao.ApplicationVersion) error {
	return d.q.UpdateApplicationVersion(ctx, db.UpdateApplicationVersionParams{AppID: value.AppID, WorkspaceID: value.WorkspaceID, Version: value.Version, Config: pgOptionalText(value.Config), Status: value.Status, Description: pgOptionalText(value.Description), GmtModified: pgTimestamp(value.GmtModified), Modifier: value.Modifier})
}

func (d *ApplicationDAO) Delete(ctx context.Context, appID, workspaceID, modifier string, modifiedAt time.Time) error {
	params := db.MarkApplicationVersionsDeletedParams{AppID: appID, WorkspaceID: workspaceID, Modifier: modifier, GmtModified: pgTimestamp(modifiedAt)}
	if err := d.q.MarkApplicationVersionsDeleted(ctx, params); err != nil {
		return err
	}
	return d.q.MarkApplicationDeleted(ctx, db.MarkApplicationDeletedParams{AppID: appID, WorkspaceID: workspaceID, Modifier: modifier, GmtModified: pgTimestamp(modifiedAt)})
}

func applicationMap(value db.Application) *dao.Application {
	return &dao.Application{ID: value.ID, WorkspaceID: value.WorkspaceID, AppID: value.AppID, Name: value.Name, Description: optionalString(value.Description), Icon: optionalString(value.Icon), Source: value.Source, Type: value.Type, Status: value.Status, GmtCreate: value.GmtCreate.Time, GmtModified: value.GmtModified.Time, Creator: value.Creator, Modifier: value.Modifier, TenantID: optionalInt8(value.TenantID)}
}

func applicationVersionMap(value db.ApplicationVersion) *dao.ApplicationVersion {
	return &dao.ApplicationVersion{ID: value.ID, AppID: value.AppID, WorkspaceID: value.WorkspaceID, Config: optionalString(value.Config), Status: value.Status, Version: value.Version, Description: optionalString(value.Description), GmtCreate: value.GmtCreate.Time, GmtModified: value.GmtModified.Time, Creator: value.Creator, Modifier: value.Modifier, TenantID: optionalInt8(value.TenantID)}
}

type AgentSchemaDAO struct{ q db.Querier }

func NewAgentSchemaDAO(q db.Querier) *AgentSchemaDAO { return &AgentSchemaDAO{q: q} }

func (d *AgentSchemaDAO) Create(ctx context.Context, value dao.AgentSchema) (int64, error) {
	return d.q.CreateAgentSchema(ctx, db.CreateAgentSchemaParams{AgentID: pgOptionalText(value.AgentID), WorkspaceID: value.WorkspaceID, Name: value.Name, Description: pgOptionalText(value.Description), Type: value.Type, Instruction: pgOptionalText(value.Instruction), InputKeys: pgOptionalText(value.InputKeys), OutputKey: pgOptionalText(value.OutputKey), Handle: pgOptionalText(value.Handle), SubAgents: pgOptionalText(value.SubAgents), YamlSchema: pgOptionalText(value.YamlSchema), Status: value.Status, Enabled: boolSmallint(value.Enabled), GmtCreate: pgTimestamp(value.GmtCreate), GmtModified: pgTimestamp(value.GmtModified), Creator: value.Creator, Modifier: value.Modifier, TenantID: pgOptionalInt8(value.TenantID)})
}

func (d *AgentSchemaDAO) Find(ctx context.Context, id int64, workspaceID string) (*dao.AgentSchema, error) {
	value, err := d.q.FindAgentSchemaByIDAndWorkspace(ctx, db.FindAgentSchemaByIDAndWorkspaceParams{ID: id, WorkspaceID: workspaceID})
	if err != nil {
		return nil, convertNotFound(err)
	}
	return agentSchemaMap(value), nil
}

func (d *AgentSchemaDAO) FindByName(ctx context.Context, name, workspaceID string) (*dao.AgentSchema, error) {
	value, err := d.q.FindAgentSchemaByNameAndWorkspace(ctx, db.FindAgentSchemaByNameAndWorkspaceParams{Name: name, WorkspaceID: workspaceID})
	if err != nil {
		return nil, convertNotFound(err)
	}
	return agentSchemaMap(value), nil
}

func (d *AgentSchemaDAO) List(ctx context.Context, workspaceID, name string) ([]dao.AgentSchema, error) {
	values, err := d.q.ListAgentSchemasByWorkspace(ctx, db.ListAgentSchemasByWorkspaceParams{WorkspaceID: workspaceID, Column2: name})
	if err != nil {
		return nil, err
	}
	result := make([]dao.AgentSchema, 0, len(values))
	for _, value := range values {
		result = append(result, *agentSchemaMap(value))
	}
	return result, nil
}

func (d *AgentSchemaDAO) Update(ctx context.Context, value dao.AgentSchema) error {
	return d.q.UpdateAgentSchema(ctx, db.UpdateAgentSchemaParams{ID: value.ID, WorkspaceID: value.WorkspaceID, Name: value.Name, Description: pgOptionalText(value.Description), Type: value.Type, Instruction: pgOptionalText(value.Instruction), InputKeys: pgOptionalText(value.InputKeys), OutputKey: pgOptionalText(value.OutputKey), Handle: pgOptionalText(value.Handle), SubAgents: pgOptionalText(value.SubAgents), YamlSchema: pgOptionalText(value.YamlSchema), Status: value.Status, Enabled: boolSmallint(value.Enabled), GmtModified: pgTimestamp(value.GmtModified), Modifier: value.Modifier})
}

func (d *AgentSchemaDAO) Delete(ctx context.Context, id int64, workspaceID string) error {
	return d.q.DeleteAgentSchema(ctx, db.DeleteAgentSchemaParams{ID: id, WorkspaceID: workspaceID})
}

func agentSchemaMap(value db.AgentSchema) *dao.AgentSchema {
	return &dao.AgentSchema{ID: value.ID, AgentID: optionalString(value.AgentID), WorkspaceID: value.WorkspaceID, Name: value.Name, Description: optionalString(value.Description), Type: value.Type, Instruction: optionalString(value.Instruction), InputKeys: optionalString(value.InputKeys), OutputKey: optionalString(value.OutputKey), Handle: optionalString(value.Handle), SubAgents: optionalString(value.SubAgents), YamlSchema: optionalString(value.YamlSchema), Status: value.Status, Enabled: value.Enabled != 0, GmtCreate: value.GmtCreate.Time, GmtModified: value.GmtModified.Time, Creator: value.Creator, Modifier: value.Modifier, TenantID: optionalInt8(value.TenantID)}
}

func boolSmallint(value bool) int16 {
	if value {
		return 1
	}
	return 0
}

type ToolDAO struct{ q db.Querier }

func NewToolDAO(q db.Querier) *ToolDAO { return &ToolDAO{q: q} }

func (d *ToolDAO) Create(ctx context.Context, value dao.Tool) (int64, error) {
	return d.q.CreateTool(ctx, db.CreateToolParams{PluginID: value.PluginID, ToolID: value.ToolID, WorkspaceID: value.WorkspaceID, Status: value.Status, Enabled: boolSmallint(value.Enabled), TestStatus: value.TestStatus, Name: value.Name, Description: pgOptionalText(value.Description), Config: value.Config, ApiSchema: value.APISchema, GmtCreate: pgTimestamp(value.GmtCreate), GmtModified: pgTimestamp(value.GmtModified), Creator: value.Creator, Modifier: value.Modifier, TenantID: pgOptionalInt8(value.TenantID)})
}

func (d *ToolDAO) Find(ctx context.Context, id int64, workspaceID string) (*dao.Tool, error) {
	value, err := d.q.FindToolByIDAndWorkspace(ctx, db.FindToolByIDAndWorkspaceParams{ID: id, WorkspaceID: workspaceID})
	if err != nil {
		return nil, convertNotFound(err)
	}
	return toolMap(value), nil
}

func (d *ToolDAO) List(ctx context.Context, workspaceID, name, pluginID string) ([]dao.Tool, error) {
	values, err := d.q.ListToolsByWorkspace(ctx, db.ListToolsByWorkspaceParams{WorkspaceID: workspaceID, Column2: name, Column3: pluginID})
	if err != nil {
		return nil, err
	}
	result := make([]dao.Tool, 0, len(values))
	for _, value := range values {
		result = append(result, *toolMap(value))
	}
	return result, nil
}

func (d *ToolDAO) Update(ctx context.Context, value dao.Tool) error {
	return d.q.UpdateTool(ctx, db.UpdateToolParams{ID: value.ID, WorkspaceID: value.WorkspaceID, PluginID: value.PluginID, Status: value.Status, Enabled: boolSmallint(value.Enabled), TestStatus: value.TestStatus, Name: value.Name, Description: pgOptionalText(value.Description), Config: value.Config, ApiSchema: value.APISchema, GmtModified: pgTimestamp(value.GmtModified), Modifier: value.Modifier})
}

func (d *ToolDAO) Delete(ctx context.Context, id int64, workspaceID string) error {
	return d.q.DeleteTool(ctx, db.DeleteToolParams{ID: id, WorkspaceID: workspaceID})
}

type KnowledgeBaseDAO struct{ q db.Querier }

func NewKnowledgeBaseDAO(q db.Querier) *KnowledgeBaseDAO { return &KnowledgeBaseDAO{q: q} }
func (d *KnowledgeBaseDAO) Create(c context.Context, v dao.KnowledgeBase) error {
	return d.q.CreateKnowledgeBase(c, db.CreateKnowledgeBaseParams{WorkspaceID: v.WorkspaceID, KbID: v.KbID, Type: v.Type, Status: v.Status, Name: v.Name, Description: pgOptionalText(v.Description), ProcessConfig: pgOptionalText(v.ProcessConfig), IndexConfig: pgOptionalText(v.IndexConfig), SearchConfig: pgOptionalText(v.SearchConfig), TotalDocs: v.TotalDocs, GmtCreate: pgTimestamp(v.GmtCreate), GmtModified: pgTimestamp(v.GmtModified), Creator: v.Creator, Modifier: v.Modifier, TenantID: pgOptionalInt8(v.TenantID)})
}
func (d *KnowledgeBaseDAO) Find(c context.Context, id, w string) (*dao.KnowledgeBase, error) {
	v, e := d.q.FindActiveKnowledgeBase(c, db.FindActiveKnowledgeBaseParams{KbID: id, WorkspaceID: w})
	if e != nil {
		return nil, convertNotFound(e)
	}
	return kbMap(v), nil
}
func (d *KnowledgeBaseDAO) List(c context.Context, w, n string) ([]dao.KnowledgeBase, error) {
	vs, e := d.q.ListActiveKnowledgeBases(c, db.ListActiveKnowledgeBasesParams{WorkspaceID: w, Column2: n})
	out := make([]dao.KnowledgeBase, 0, len(vs))
	for _, v := range vs {
		out = append(out, *kbMap(v))
	}
	return out, e
}
func (d *KnowledgeBaseDAO) Update(c context.Context, v dao.KnowledgeBase) error {
	return d.q.UpdateKnowledgeBase(c, db.UpdateKnowledgeBaseParams{KbID: v.KbID, WorkspaceID: v.WorkspaceID, Type: v.Type, Name: v.Name, Description: pgOptionalText(v.Description), ProcessConfig: pgOptionalText(v.ProcessConfig), IndexConfig: pgOptionalText(v.IndexConfig), SearchConfig: pgOptionalText(v.SearchConfig), GmtModified: pgTimestamp(v.GmtModified), Modifier: v.Modifier})
}
func (d *KnowledgeBaseDAO) Delete(c context.Context, id, w, m string, t time.Time) error {
	return d.q.DeleteKnowledgeBase(c, db.DeleteKnowledgeBaseParams{KbID: id, WorkspaceID: w, Modifier: m, GmtModified: pgTimestamp(t)})
}
func kbMap(v db.KnowledgeBase) *dao.KnowledgeBase {
	return &dao.KnowledgeBase{ID: v.ID, WorkspaceID: v.WorkspaceID, KbID: v.KbID, Type: v.Type, Status: v.Status, Name: v.Name, Description: optionalString(v.Description), ProcessConfig: optionalString(v.ProcessConfig), IndexConfig: optionalString(v.IndexConfig), SearchConfig: optionalString(v.SearchConfig), TotalDocs: v.TotalDocs, GmtCreate: v.GmtCreate.Time, GmtModified: v.GmtModified.Time, Creator: v.Creator, Modifier: v.Modifier, TenantID: optionalInt8(v.TenantID)}
}

type McpServerDAO struct{ q db.Querier }

func NewMcpServerDAO(q db.Querier) *McpServerDAO { return &McpServerDAO{q: q} }

func (d *McpServerDAO) Create(ctx context.Context, value dao.McpServer) error {
	return d.q.CreateMcpServer(ctx, db.CreateMcpServerParams{
		GmtCreate: pgTimestamp(value.GmtCreate), GmtModified: pgTimestamp(value.GmtModified),
		ServerCode: value.ServerCode, Name: value.Name, Description: pgOptionalText(value.Description),
		Source: pgOptionalText(value.Source), Type: value.Type, DeployConfig: value.DeployConfig,
		WorkspaceID: pgText(value.WorkspaceID), AccountID: pgText(value.AccountID), Status: value.Status,
		DetailConfig: pgOptionalText(value.DetailConfig), InstallType: pgOptionalText(value.InstallType),
		TenantID: pgOptionalInt8(value.TenantID),
	})
}

func (d *McpServerDAO) Find(ctx context.Context, serverCode, workspaceID string) (*dao.McpServer, error) {
	value, err := d.q.FindActiveMcpServer(ctx, db.FindActiveMcpServerParams{ServerCode: serverCode, WorkspaceID: pgText(workspaceID)})
	if err != nil {
		return nil, convertNotFound(err)
	}
	return mcpServerMap(value), nil
}

func (d *McpServerDAO) List(ctx context.Context, workspaceID, name string) ([]dao.McpServer, error) {
	values, err := d.q.ListActiveMcpServers(ctx, db.ListActiveMcpServersParams{WorkspaceID: pgText(workspaceID), Column2: name})
	if err != nil {
		return nil, err
	}
	result := make([]dao.McpServer, 0, len(values))
	for _, value := range values {
		result = append(result, *mcpServerMap(value))
	}
	return result, nil
}

func (d *McpServerDAO) Update(ctx context.Context, value dao.McpServer) error {
	return d.q.UpdateMcpServer(ctx, db.UpdateMcpServerParams{
		ServerCode: value.ServerCode, WorkspaceID: pgText(value.WorkspaceID), GmtModified: pgTimestamp(value.GmtModified),
		Name: value.Name, Description: pgOptionalText(value.Description), Source: pgOptionalText(value.Source),
		Type: value.Type, DeployConfig: value.DeployConfig, Status: value.Status,
		DetailConfig: pgOptionalText(value.DetailConfig), InstallType: pgOptionalText(value.InstallType),
	})
}

func (d *McpServerDAO) Delete(ctx context.Context, serverCode, workspaceID string, modifiedAt time.Time) error {
	return d.q.DeleteMcpServer(ctx, db.DeleteMcpServerParams{ServerCode: serverCode, WorkspaceID: pgText(workspaceID), GmtModified: pgTimestamp(modifiedAt)})
}

func mcpServerMap(value db.McpServer) *dao.McpServer {
	return &dao.McpServer{
		ServerCode: value.ServerCode, WorkspaceID: textString(value.WorkspaceID), AccountID: textString(value.AccountID),
		Name: value.Name, DeployConfig: value.DeployConfig, DetailConfig: optionalString(value.DetailConfig),
		Status: value.Status, Type: value.Type, BizType: optionalString(value.BizType),
		Description: optionalString(value.Description), InstallType: optionalString(value.InstallType),
		DeployEnv: optionalString(value.DeployEnv), Source: optionalString(value.Source), Host: optionalString(value.Host),
		GmtCreate: value.GmtCreate.Time, GmtModified: value.GmtModified.Time, TenantID: optionalInt8(value.TenantID),
	}
}

type SkillDAO struct{ q db.Querier }

func NewSkillDAO(q db.Querier) *SkillDAO { return &SkillDAO{q: q} }

func (d *SkillDAO) Create(ctx context.Context, value dao.Skill) error {
	return d.q.CreateSkill(ctx, db.CreateSkillParams{SkillCode: value.SkillCode, WorkspaceID: value.WorkspaceID, AccountID: pgOptionalText(value.AccountID), Name: value.Name, Description: pgOptionalText(value.Description), Source: value.Source, Status: value.Status, Tags: pgOptionalText(value.Tags), GmtCreate: pgTimestamp(value.GmtCreate), GmtModified: pgTimestamp(value.GmtModified), Creator: pgOptionalText(value.Creator), Modifier: pgOptionalText(value.Modifier)})
}

func (d *SkillDAO) CreateVersion(ctx context.Context, value dao.SkillVersion) error {
	return d.q.CreateSkillVersion(ctx, db.CreateSkillVersionParams{SkillCode: value.SkillCode, WorkspaceID: value.WorkspaceID, Version: value.Version, Description: pgOptionalText(value.Description), MainFilePath: value.MainFilePath, Manifest: pgOptionalText(value.Manifest), ContentHash: pgOptionalText(value.ContentHash), StorageType: value.StorageType, StorageBucket: pgOptionalText(value.StorageBucket), StoragePrefix: value.StoragePrefix, PackageObjectKey: pgOptionalText(value.PackageObjectKey), FileCount: pgOptionalInt4(value.FileCount), TotalSizeBytes: pgOptionalInt8Pointer(value.TotalSizeBytes), Status: value.Status, GmtCreate: pgTimestamp(value.GmtCreate), GmtModified: pgTimestamp(value.GmtModified), Creator: pgOptionalText(value.Creator), Modifier: pgOptionalText(value.Modifier)})
}

func (d *SkillDAO) Find(ctx context.Context, skillCode, workspaceID string) (*dao.Skill, error) {
	value, err := d.q.FindActiveSkill(ctx, db.FindActiveSkillParams{SkillCode: skillCode, WorkspaceID: workspaceID})
	if err != nil {
		return nil, convertNotFound(err)
	}
	return skillMap(value), nil
}

func (d *SkillDAO) FindVersion(ctx context.Context, skillCode, workspaceID, version string) (*dao.SkillVersion, error) {
	value, err := d.q.FindActiveSkillVersion(ctx, db.FindActiveSkillVersionParams{SkillCode: skillCode, WorkspaceID: workspaceID, Version: version})
	if err != nil {
		return nil, convertNotFound(err)
	}
	return skillVersionMap(value), nil
}

func (d *SkillDAO) FindLatestVersion(ctx context.Context, skillCode, workspaceID string) (*dao.SkillVersion, error) {
	value, err := d.q.FindLatestActiveSkillVersion(ctx, db.FindLatestActiveSkillVersionParams{SkillCode: skillCode, WorkspaceID: workspaceID})
	if err != nil {
		return nil, convertNotFound(err)
	}
	return skillVersionMap(value), nil
}

func (d *SkillDAO) List(ctx context.Context, workspaceID, name string, status int16) ([]dao.Skill, error) {
	values, err := d.q.ListActiveSkills(ctx, db.ListActiveSkillsParams{WorkspaceID: workspaceID, Column2: name, Column3: status})
	if err != nil {
		return nil, err
	}
	result := make([]dao.Skill, 0, len(values))
	for _, value := range values {
		result = append(result, *skillMap(value))
	}
	return result, nil
}

func (d *SkillDAO) Update(ctx context.Context, value dao.Skill) error {
	return d.q.UpdateSkill(ctx, db.UpdateSkillParams{SkillCode: value.SkillCode, WorkspaceID: value.WorkspaceID, Name: value.Name, Description: pgOptionalText(value.Description), Source: value.Source, Status: value.Status, Tags: pgOptionalText(value.Tags), GmtModified: pgTimestamp(value.GmtModified), Modifier: pgOptionalText(value.Modifier)})
}

func (d *SkillDAO) UpdateVersion(ctx context.Context, value dao.SkillVersion) error {
	return d.q.UpdateSkillVersion(ctx, db.UpdateSkillVersionParams{SkillCode: value.SkillCode, WorkspaceID: value.WorkspaceID, Version: value.Version, Description: pgOptionalText(value.Description), MainFilePath: value.MainFilePath, Manifest: pgOptionalText(value.Manifest), ContentHash: pgOptionalText(value.ContentHash), StorageType: value.StorageType, StorageBucket: pgOptionalText(value.StorageBucket), StoragePrefix: value.StoragePrefix, PackageObjectKey: pgOptionalText(value.PackageObjectKey), FileCount: pgOptionalInt4(value.FileCount), TotalSizeBytes: pgOptionalInt8Pointer(value.TotalSizeBytes), Status: value.Status, GmtModified: pgTimestamp(value.GmtModified), Modifier: pgOptionalText(value.Modifier)})
}

func (d *SkillDAO) Delete(ctx context.Context, skillCode, workspaceID, modifier string, modifiedAt time.Time) error {
	params := db.MarkSkillDeletedParams{SkillCode: skillCode, WorkspaceID: workspaceID, Modifier: pgText(modifier), GmtModified: pgTimestamp(modifiedAt)}
	if err := d.q.MarkSkillDeleted(ctx, params); err != nil {
		return err
	}
	return d.q.MarkSkillVersionsDeleted(ctx, db.MarkSkillVersionsDeletedParams{SkillCode: skillCode, WorkspaceID: workspaceID, Modifier: pgText(modifier), GmtModified: pgTimestamp(modifiedAt)})
}

func skillMap(value db.Skill) *dao.Skill {
	return &dao.Skill{SkillCode: value.SkillCode, WorkspaceID: value.WorkspaceID, AccountID: optionalString(value.AccountID), Name: value.Name, Description: optionalString(value.Description), Source: value.Source, Status: value.Status, Tags: optionalString(value.Tags), GmtCreate: value.GmtCreate.Time, GmtModified: value.GmtModified.Time, Creator: optionalString(value.Creator), Modifier: optionalString(value.Modifier), TenantID: optionalInt8(value.TenantID)}
}

func skillVersionMap(value db.SkillVersion) *dao.SkillVersion {
	return &dao.SkillVersion{SkillCode: value.SkillCode, WorkspaceID: value.WorkspaceID, Version: value.Version, Description: optionalString(value.Description), MainFilePath: value.MainFilePath, Manifest: optionalString(value.Manifest), ContentHash: optionalString(value.ContentHash), StorageType: value.StorageType, StorageBucket: optionalString(value.StorageBucket), StoragePrefix: value.StoragePrefix, PackageObjectKey: optionalString(value.PackageObjectKey), FileCount: optionalInt4(value.FileCount), TotalSizeBytes: optionalInt8Pointer(value.TotalSizeBytes), Status: value.Status, GmtCreate: value.GmtCreate.Time, GmtModified: value.GmtModified.Time, Creator: optionalString(value.Creator), Modifier: optionalString(value.Modifier), TenantID: optionalInt8(value.TenantID)}
}

type PluginDAO struct{ q db.Querier }

func NewPluginDAO(q db.Querier) *PluginDAO { return &PluginDAO{q: q} }

func (d *PluginDAO) Create(ctx context.Context, value dao.Plugin) error {
	return d.q.CreatePlugin(ctx, db.CreatePluginParams{PluginID: value.PluginID, WorkspaceID: value.WorkspaceID, Type: value.Type, Status: value.Status, Name: value.Name, Description: pgOptionalText(value.Description), Config: pgOptionalText(value.Config), Source: value.Source, GmtCreate: pgTimestamp(value.GmtCreate), GmtModified: pgTimestamp(value.GmtModified), Creator: value.Creator, Modifier: value.Modifier, TenantID: pgOptionalInt8(value.TenantID)})
}

func (d *PluginDAO) Find(ctx context.Context, pluginID, workspaceID string) (*dao.Plugin, error) {
	value, err := d.q.FindActivePlugin(ctx, db.FindActivePluginParams{PluginID: pluginID, WorkspaceID: workspaceID})
	if err != nil {
		return nil, convertNotFound(err)
	}
	return pluginMap(value), nil
}

func (d *PluginDAO) List(ctx context.Context, workspaceID, name string, status int16) ([]dao.Plugin, error) {
	values, err := d.q.ListActivePlugins(ctx, db.ListActivePluginsParams{WorkspaceID: workspaceID, Column2: name, Column3: status})
	if err != nil {
		return nil, err
	}
	result := make([]dao.Plugin, 0, len(values))
	for _, value := range values {
		result = append(result, *pluginMap(value))
	}
	return result, nil
}

func (d *PluginDAO) Update(ctx context.Context, value dao.Plugin) error {
	return d.q.UpdatePlugin(ctx, db.UpdatePluginParams{PluginID: value.PluginID, WorkspaceID: value.WorkspaceID, Type: value.Type, Status: value.Status, Name: value.Name, Description: pgOptionalText(value.Description), Config: pgOptionalText(value.Config), Source: value.Source, GmtModified: pgTimestamp(value.GmtModified), Modifier: value.Modifier})
}

func (d *PluginDAO) Delete(ctx context.Context, pluginID, workspaceID string) error {
	if err := d.q.DeletePluginTools(ctx, db.DeletePluginToolsParams{PluginID: pluginID, WorkspaceID: workspaceID}); err != nil {
		return err
	}
	return d.q.DeletePlugin(ctx, db.DeletePluginParams{PluginID: pluginID, WorkspaceID: workspaceID})
}

func (d *PluginDAO) CreateTool(ctx context.Context, value dao.Tool) error {
	return d.q.CreatePluginTool(ctx, db.CreatePluginToolParams{PluginID: value.PluginID, ToolID: value.ToolID, WorkspaceID: value.WorkspaceID, Status: value.Status, Enabled: boolSmallint(value.Enabled), TestStatus: value.TestStatus, Name: value.Name, Description: pgOptionalText(value.Description), Config: value.Config, ApiSchema: value.APISchema, GmtCreate: pgTimestamp(value.GmtCreate), GmtModified: pgTimestamp(value.GmtModified), Creator: value.Creator, Modifier: value.Modifier, TenantID: pgOptionalInt8(value.TenantID)})
}

func (d *PluginDAO) FindTool(ctx context.Context, pluginID, toolID, workspaceID string) (*dao.Tool, error) {
	value, err := d.q.FindPluginTool(ctx, db.FindPluginToolParams{PluginID: pluginID, ToolID: toolID, WorkspaceID: workspaceID})
	if err != nil {
		return nil, convertNotFound(err)
	}
	return toolMap(value), nil
}

func (d *PluginDAO) FindToolByID(ctx context.Context, toolID, workspaceID string) (*dao.Tool, error) {
	value, err := d.q.FindPluginToolByID(ctx, db.FindPluginToolByIDParams{ToolID: toolID, WorkspaceID: workspaceID})
	if err != nil {
		return nil, convertNotFound(err)
	}
	return toolMap(value), nil
}

func (d *PluginDAO) ListTools(ctx context.Context, pluginID, workspaceID, name string) ([]dao.Tool, error) {
	values, err := d.q.ListPluginTools(ctx, db.ListPluginToolsParams{PluginID: pluginID, WorkspaceID: workspaceID, Column3: name})
	if err != nil {
		return nil, err
	}
	result := make([]dao.Tool, 0, len(values))
	for _, value := range values {
		result = append(result, *toolMap(value))
	}
	return result, nil
}

func (d *PluginDAO) ListToolsByIDs(ctx context.Context, workspaceID string, toolIDs []string) ([]dao.Tool, error) {
	values, err := d.q.ListPluginToolsByIDs(ctx, db.ListPluginToolsByIDsParams{WorkspaceID: workspaceID, Column2: toolIDs})
	if err != nil {
		return nil, err
	}
	result := make([]dao.Tool, 0, len(values))
	for _, value := range values {
		result = append(result, *toolMap(value))
	}
	return result, nil
}

func (d *PluginDAO) UpdateTool(ctx context.Context, value dao.Tool) error {
	return d.q.UpdatePluginTool(ctx, db.UpdatePluginToolParams{PluginID: value.PluginID, ToolID: value.ToolID, WorkspaceID: value.WorkspaceID, Status: value.Status, Enabled: boolSmallint(value.Enabled), TestStatus: value.TestStatus, Name: value.Name, Description: pgOptionalText(value.Description), Config: value.Config, ApiSchema: value.APISchema, GmtModified: pgTimestamp(value.GmtModified), Modifier: value.Modifier})
}

func (d *PluginDAO) DeleteTool(ctx context.Context, pluginID, toolID, workspaceID string) error {
	return d.q.DeletePluginTool(ctx, db.DeletePluginToolParams{PluginID: pluginID, ToolID: toolID, WorkspaceID: workspaceID})
}

func (d *PluginDAO) DeleteTools(ctx context.Context, pluginID, workspaceID string) error {
	return d.q.DeletePluginTools(ctx, db.DeletePluginToolsParams{PluginID: pluginID, WorkspaceID: workspaceID})
}

func pluginMap(value db.Plugin) *dao.Plugin {
	return &dao.Plugin{PluginID: value.PluginID, WorkspaceID: value.WorkspaceID, Type: value.Type, Status: value.Status, Name: value.Name, Description: optionalString(value.Description), Config: optionalString(value.Config), Source: value.Source, GmtCreate: value.GmtCreate.Time, GmtModified: value.GmtModified.Time, Creator: value.Creator, Modifier: value.Modifier, TenantID: optionalInt8(value.TenantID)}
}

func toolMap(value db.Tool) *dao.Tool {
	return &dao.Tool{ID: value.ID, PluginID: value.PluginID, ToolID: value.ToolID, WorkspaceID: value.WorkspaceID, Status: value.Status, Enabled: value.Enabled != 0, TestStatus: value.TestStatus, Name: value.Name, Description: optionalString(value.Description), Config: value.Config, APISchema: value.ApiSchema, GmtCreate: value.GmtCreate.Time, GmtModified: value.GmtModified.Time, Creator: value.Creator, Modifier: value.Modifier, TenantID: optionalInt8(value.TenantID)}
}

type DocumentDAO struct{ q db.Querier }

func NewDocumentDAO(q db.Querier) *DocumentDAO { return &DocumentDAO{q: q} }

func (d *DocumentDAO) Create(ctx context.Context, value dao.Document) error {
	return d.q.CreateDocument(ctx, db.CreateDocumentParams{WorkspaceID: value.WorkspaceID, KbID: value.KbID, DocID: value.DocID, Type: value.Type, Status: value.Status, Enabled: boolSmallint(value.Enabled), Name: value.Name, Format: value.Format, Size: value.Size, Metadata: pgOptionalText(value.Metadata), IndexStatus: value.IndexStatus, Path: value.Path, ParsedPath: pgOptionalText(value.ParsedPath), ProcessConfig: pgOptionalText(value.ProcessConfig), Source: pgOptionalText(value.Source), Error: pgOptionalText(value.Error), GmtCreate: pgTimestamp(value.GmtCreate), GmtModified: pgTimestamp(value.GmtModified), Creator: value.Creator, Modifier: value.Modifier, TenantID: pgOptionalInt8(value.TenantID)})
}

func (d *DocumentDAO) Find(ctx context.Context, workspaceID, kbID, docID string) (*dao.Document, error) {
	value, err := d.q.FindActiveDocument(ctx, db.FindActiveDocumentParams{WorkspaceID: workspaceID, KbID: kbID, DocID: docID})
	if err != nil {
		return nil, convertNotFound(err)
	}
	return documentMap(value), nil
}

func (d *DocumentDAO) FindByDocID(ctx context.Context, workspaceID, docID string) (*dao.Document, error) {
	value, err := d.q.FindActiveDocumentByDocID(ctx, db.FindActiveDocumentByDocIDParams{WorkspaceID: workspaceID, DocID: docID})
	if err != nil {
		return nil, convertNotFound(err)
	}
	return documentMap(value), nil
}

func (d *DocumentDAO) List(ctx context.Context, workspaceID, kbID, name string, indexStatus int16) ([]dao.Document, error) {
	values, err := d.q.ListActiveDocuments(ctx, db.ListActiveDocumentsParams{WorkspaceID: workspaceID, KbID: kbID, Column3: name, Column4: indexStatus})
	if err != nil {
		return nil, err
	}
	result := make([]dao.Document, 0, len(values))
	for _, value := range values {
		result = append(result, *documentMap(value))
	}
	return result, nil
}

func (d *DocumentDAO) Update(ctx context.Context, value dao.Document) error {
	return d.q.UpdateDocument(ctx, db.UpdateDocumentParams{WorkspaceID: value.WorkspaceID, KbID: value.KbID, DocID: value.DocID, Enabled: boolSmallint(value.Enabled), Name: value.Name, Format: value.Format, Size: value.Size, Metadata: pgOptionalText(value.Metadata), IndexStatus: value.IndexStatus, Path: value.Path, ParsedPath: pgOptionalText(value.ParsedPath), ProcessConfig: pgOptionalText(value.ProcessConfig), Source: pgOptionalText(value.Source), Error: pgOptionalText(value.Error), GmtModified: pgTimestamp(value.GmtModified), Modifier: value.Modifier})
}

func (d *DocumentDAO) Delete(ctx context.Context, workspaceID, kbID, docID, modifier string, modifiedAt time.Time) error {
	return d.q.DeleteDocument(ctx, db.DeleteDocumentParams{WorkspaceID: workspaceID, KbID: kbID, DocID: docID, Modifier: modifier, GmtModified: pgTimestamp(modifiedAt)})
}

func (d *DocumentDAO) CreateChunk(ctx context.Context, value dao.DocumentChunk) error {
	return d.q.CreateDocumentChunk(ctx, db.CreateDocumentChunkParams{WorkspaceID: value.WorkspaceID, KbID: value.KbID, DocID: value.DocID, ChunkID: value.ChunkID, DocName: value.DocName, Title: value.Title, Text: value.Text, Score: pgOptionalFloat8(value.Score), PageNumber: pgOptionalInt4(value.PageNumber), Enabled: boolSmallint(value.Enabled), GmtCreate: pgTimestamp(value.GmtCreate), GmtModified: pgTimestamp(value.GmtModified), Creator: value.Creator, Modifier: value.Modifier, TenantID: pgOptionalInt8(value.TenantID)})
}

func (d *DocumentDAO) FindChunk(ctx context.Context, workspaceID, docID, chunkID string) (*dao.DocumentChunk, error) {
	value, err := d.q.FindActiveDocumentChunk(ctx, db.FindActiveDocumentChunkParams{WorkspaceID: workspaceID, DocID: docID, ChunkID: chunkID})
	if err != nil {
		return nil, convertNotFound(err)
	}
	return documentChunkMap(value), nil
}

func (d *DocumentDAO) ListChunks(ctx context.Context, workspaceID, docID string) ([]dao.DocumentChunk, error) {
	values, err := d.q.ListActiveDocumentChunks(ctx, db.ListActiveDocumentChunksParams{WorkspaceID: workspaceID, DocID: docID})
	if err != nil {
		return nil, err
	}
	result := make([]dao.DocumentChunk, 0, len(values))
	for _, value := range values {
		result = append(result, *documentChunkMap(value))
	}
	return result, nil
}

func (d *DocumentDAO) UpdateChunk(ctx context.Context, value dao.DocumentChunk) error {
	return d.q.UpdateDocumentChunk(ctx, db.UpdateDocumentChunkParams{WorkspaceID: value.WorkspaceID, DocID: value.DocID, ChunkID: value.ChunkID, Title: value.Title, Text: value.Text, Score: pgOptionalFloat8(value.Score), PageNumber: pgOptionalInt4(value.PageNumber), Enabled: boolSmallint(value.Enabled), GmtModified: pgTimestamp(value.GmtModified), Modifier: value.Modifier})
}

func (d *DocumentDAO) SetChunksEnabled(ctx context.Context, workspaceID, docID string, chunkIDs []string, enabled bool, modifier string, modifiedAt time.Time) error {
	return d.q.UpdateDocumentChunksEnabled(ctx, db.UpdateDocumentChunksEnabledParams{WorkspaceID: workspaceID, DocID: docID, Column3: chunkIDs, Enabled: boolSmallint(enabled), GmtModified: pgTimestamp(modifiedAt), Modifier: modifier})
}

func (d *DocumentDAO) DeleteChunks(ctx context.Context, workspaceID, docID string, chunkIDs []string, modifier string, modifiedAt time.Time) error {
	return d.q.DeleteDocumentChunks(ctx, db.DeleteDocumentChunksParams{WorkspaceID: workspaceID, DocID: docID, Column3: chunkIDs, GmtModified: pgTimestamp(modifiedAt), Modifier: modifier})
}

func documentMap(value db.Document) *dao.Document {
	return &dao.Document{WorkspaceID: value.WorkspaceID, KbID: value.KbID, DocID: value.DocID, Type: value.Type, Status: value.Status, Enabled: value.Enabled != 0, Name: value.Name, Format: value.Format, Size: value.Size, Metadata: optionalString(value.Metadata), IndexStatus: value.IndexStatus, Path: value.Path, ParsedPath: optionalString(value.ParsedPath), ProcessConfig: optionalString(value.ProcessConfig), Source: optionalString(value.Source), Error: optionalString(value.Error), GmtCreate: value.GmtCreate.Time, GmtModified: value.GmtModified.Time, Creator: value.Creator, Modifier: value.Modifier, TenantID: optionalInt8(value.TenantID)}
}

func documentChunkMap(value db.DocumentChunk) *dao.DocumentChunk {
	return &dao.DocumentChunk{WorkspaceID: value.WorkspaceID, KbID: value.KbID, DocID: value.DocID, ChunkID: value.ChunkID, DocName: value.DocName, Title: value.Title, Text: value.Text, Score: optionalFloat8(value.Score), PageNumber: optionalInt4(value.PageNumber), Enabled: value.Enabled != 0, GmtCreate: value.GmtCreate.Time, GmtModified: value.GmtModified.Time, Creator: value.Creator, Modifier: value.Modifier, TenantID: optionalInt8(value.TenantID)}
}

type ApplicationComponentDAO struct{ q db.Querier }

func NewApplicationComponentDAO(q db.Querier) *ApplicationComponentDAO {
	return &ApplicationComponentDAO{q: q}
}

func (d *ApplicationComponentDAO) Create(ctx context.Context, value dao.ApplicationComponent) error {
	return d.q.CreateApplicationComponent(ctx, db.CreateApplicationComponentParams{GmtCreate: pgTimestamp(value.GmtCreate), GmtModified: pgTimestamp(value.GmtModified), Code: value.Code, Name: value.Name, WorkspaceID: value.WorkspaceID, Type: value.Type, AppID: pgOptionalText(value.AppID), Config: pgOptionalText(value.Config), Description: pgOptionalText(value.Description), Status: pgtype.Int2{Int16: value.Status, Valid: true}, Creator: pgOptionalText(value.Creator), Modifier: pgOptionalText(value.Modifier), TenantID: pgOptionalInt8(value.TenantID)})
}

func (d *ApplicationComponentDAO) Find(ctx context.Context, workspaceID, code string) (*dao.ApplicationComponent, error) {
	value, err := d.q.FindActiveApplicationComponent(ctx, db.FindActiveApplicationComponentParams{WorkspaceID: workspaceID, Code: code})
	if err != nil {
		return nil, convertNotFound(err)
	}
	return applicationComponentMap(value), nil
}

func (d *ApplicationComponentDAO) FindByAppID(ctx context.Context, workspaceID, appID string) (*dao.ApplicationComponent, error) {
	value, err := d.q.FindActiveApplicationComponentByAppID(ctx, db.FindActiveApplicationComponentByAppIDParams{WorkspaceID: workspaceID, AppID: pgText(appID)})
	if err != nil {
		return nil, convertNotFound(err)
	}
	return applicationComponentMap(value), nil
}

func (d *ApplicationComponentDAO) List(ctx context.Context, workspaceID, name, componentType, appID string, status int16) ([]dao.ApplicationComponent, error) {
	values, err := d.q.ListActiveApplicationComponents(ctx, db.ListActiveApplicationComponentsParams{WorkspaceID: workspaceID, Column2: name, Column3: componentType, Column4: appID, Column5: status})
	if err != nil {
		return nil, err
	}
	result := make([]dao.ApplicationComponent, 0, len(values))
	for _, value := range values {
		result = append(result, *applicationComponentMap(value))
	}
	return result, nil
}

func (d *ApplicationComponentDAO) Update(ctx context.Context, value dao.ApplicationComponent) error {
	return d.q.UpdateApplicationComponent(ctx, db.UpdateApplicationComponentParams{WorkspaceID: value.WorkspaceID, Code: value.Code, Name: value.Name, Type: value.Type, AppID: pgOptionalText(value.AppID), Config: pgOptionalText(value.Config), Description: pgOptionalText(value.Description), Status: pgtype.Int2{Int16: value.Status, Valid: true}, GmtModified: pgTimestamp(value.GmtModified), Modifier: pgOptionalText(value.Modifier)})
}

func (d *ApplicationComponentDAO) Delete(ctx context.Context, workspaceID, code, modifier string, modifiedAt time.Time) error {
	return d.q.DeleteApplicationComponent(ctx, db.DeleteApplicationComponentParams{WorkspaceID: workspaceID, Code: code, GmtModified: pgTimestamp(modifiedAt), Modifier: pgText(modifier)})
}

func applicationComponentMap(value db.ApplicationComponent) *dao.ApplicationComponent {
	return &dao.ApplicationComponent{Code: value.Code, Name: value.Name, WorkspaceID: value.WorkspaceID, Type: value.Type, AppID: optionalString(value.AppID), Config: optionalString(value.Config), Description: optionalString(value.Description), Status: value.Status.Int16, GmtCreate: value.GmtCreate.Time, GmtModified: value.GmtModified.Time, Creator: optionalString(value.Creator), Modifier: optionalString(value.Modifier), TenantID: optionalInt8(value.TenantID)}
}
