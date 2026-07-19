package appcomponent

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/seaskyland/openclaw4j-backend-go/internal/application"
	"github.com/seaskyland/openclaw4j-backend-go/internal/dao"
)

const (
	statusPublished int16 = 1
	statusDeleted   int16 = 3
)

var (
	ErrTypeRequired        = errors.New("type is required")
	ErrNameRequired        = errors.New("name is required")
	ErrConfigRequired      = errors.New("config is required")
	ErrAppIDRequired       = errors.New("app_id is required")
	ErrDescriptionRequired = errors.New("description is required")
	ErrNotFound            = errors.New("application component not found")
)

type Input struct {
	Code        string          `json:"code"`
	Name        string          `json:"name"`
	AppName     *string         `json:"app_name"`
	Type        string          `json:"type"`
	AppID       string          `json:"app_id"`
	Config      json.RawMessage `json:"config"`
	Description string          `json:"description"`
	Status      *int16          `json:"status"`
	Codes       []string        `json:"codes"`
}

type Component struct {
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	AppName     *string   `json:"app_name,omitempty"`
	Type        string    `json:"type"`
	AppID       string    `json:"app_id"`
	Config      string    `json:"config"`
	Description string    `json:"description"`
	Status      int16     `json:"status"`
	NeedUpdate  int16     `json:"need_update"`
	GmtCreate   time.Time `json:"gmt_create"`
	GmtModified time.Time `json:"gmt_modified"`
}

type Page[T any] struct {
	Current int64 `json:"current"`
	Size    int64 `json:"size"`
	Total   int64 `json:"total"`
	Records []T   `json:"records"`
}

type ApplicationReader interface {
	Get(context.Context, string, string) (*application.Application, error)
	List(context.Context, string, string, string, string, int64, int64) (*application.Page[application.Application], error)
}

type Service struct {
	dao  dao.ApplicationComponentDAO
	apps ApplicationReader
	now  func() time.Time
}

func NewService(data dao.ApplicationComponentDAO, apps ApplicationReader, clock func() time.Time) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{dao: data, apps: apps, now: clock}
}

func (s *Service) Create(ctx context.Context, workspaceID, accountID string, input Input) (string, error) {
	if err := validate(input); err != nil {
		return "", err
	}
	if _, err := s.apps.Get(ctx, workspaceID, input.AppID); err != nil {
		return "", application.ErrNotFound
	}
	code, err := newCode()
	if err != nil {
		return "", err
	}
	now := s.now()
	value := dao.ApplicationComponent{Code: code, Name: strings.TrimSpace(input.Name), WorkspaceID: workspaceID, Type: strings.TrimSpace(input.Type), AppID: stringPtr(input.AppID), Config: rawJSONText(input.Config), Description: stringPtr(strings.TrimSpace(input.Description)), Status: statusPublished, GmtCreate: now, GmtModified: now, Creator: stringPtr(accountID), Modifier: stringPtr(accountID)}
	if err := s.dao.Create(ctx, value); err != nil {
		return "", err
	}
	return code, nil
}

func (s *Service) Update(ctx context.Context, workspaceID, accountID, code string, input Input) error {
	if strings.TrimSpace(code) == "" {
		return ErrNotFound
	}
	if err := validate(input); err != nil {
		return err
	}
	value, err := s.dao.Find(ctx, workspaceID, code)
	if errors.Is(err, dao.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if _, err := s.apps.Get(ctx, workspaceID, input.AppID); err != nil {
		return application.ErrNotFound
	}
	value.Name, value.Type, value.AppID, value.Config, value.Description = strings.TrimSpace(input.Name), strings.TrimSpace(input.Type), stringPtr(input.AppID), rawJSONText(input.Config), stringPtr(strings.TrimSpace(input.Description))
	value.Status, value.GmtModified, value.Modifier = statusPublished, s.now(), stringPtr(accountID)
	return s.dao.Update(ctx, *value)
}

func (s *Service) Delete(ctx context.Context, workspaceID, accountID, code string) error {
	if _, err := s.Get(ctx, workspaceID, code); err != nil {
		return err
	}
	return s.dao.Delete(ctx, workspaceID, code, accountID, s.now())
}

func (s *Service) Get(ctx context.Context, workspaceID, code string) (*Component, error) {
	value, err := s.dao.Find(ctx, workspaceID, code)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.toComponent(ctx, workspaceID, *value)
}

func (s *Service) GetByAppID(ctx context.Context, workspaceID, appID string) (*Component, error) {
	value, err := s.dao.FindByAppID(ctx, workspaceID, appID)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.toComponent(ctx, workspaceID, *value)
}

func (s *Service) List(ctx context.Context, workspaceID, name, componentType, appID string, status, current, size int64) (*Page[Component], error) {
	current, size = page(current, size)
	values, err := s.dao.List(ctx, workspaceID, strings.TrimSpace(name), strings.TrimSpace(componentType), strings.TrimSpace(appID), int16(status))
	if err != nil {
		return nil, err
	}
	all := make([]Component, 0, len(values))
	for _, value := range values {
		mapped, err := s.toComponent(ctx, workspaceID, value)
		if err != nil {
			return nil, err
		}
		all = append(all, *mapped)
	}
	start := (current - 1) * size
	if start >= int64(len(all)) {
		return &Page[Component]{Current: current, Size: size, Total: int64(len(all)), Records: []Component{}}, nil
	}
	end := start + size
	if end > int64(len(all)) {
		end = int64(len(all))
	}
	return &Page[Component]{Current: current, Size: size, Total: int64(len(all)), Records: all[start:end]}, nil
}

func (s *Service) ListByCodes(ctx context.Context, workspaceID string, codes []string) ([]Component, error) {
	values, err := s.dao.List(ctx, workspaceID, "", "", "", -1)
	if err != nil {
		return nil, err
	}
	wanted := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		wanted[strings.TrimSpace(code)] = struct{}{}
	}
	result := make([]Component, 0, len(codes))
	for _, value := range values {
		if _, ok := wanted[value.Code]; ok {
			component, err := s.toComponent(ctx, workspaceID, value)
			if err != nil {
				return nil, err
			}
			result = append(result, *component)
		}
	}
	return result, nil
}

func (s *Service) Publishable(ctx context.Context, workspaceID, componentType, appName string, current, size int64) (*Page[application.Application], error) {
	if strings.TrimSpace(componentType) == "" {
		return nil, ErrTypeRequired
	}
	apps, err := s.apps.List(ctx, workspaceID, appName, componentType, "published", 1, 100)
	if err != nil {
		return nil, err
	}
	components, err := s.dao.List(ctx, workspaceID, "", componentType, "", statusPublished)
	if err != nil {
		return nil, err
	}
	existing := make(map[string]struct{}, len(components))
	for _, component := range components {
		if component.AppID != nil {
			existing[*component.AppID] = struct{}{}
		}
	}
	available := make([]application.Application, 0, len(apps.Records))
	for _, app := range apps.Records {
		if _, found := existing[app.AppID]; !found {
			available = append(available, app)
		}
	}
	current, size = page(current, size)
	start := (current - 1) * size
	if start >= int64(len(available)) {
		return &Page[application.Application]{Current: current, Size: size, Total: int64(len(available)), Records: []application.Application{}}, nil
	}
	end := start + size
	if end > int64(len(available)) {
		end = int64(len(available))
	}
	return &Page[application.Application]{Current: current, Size: size, Total: int64(len(available)), Records: available[start:end]}, nil
}

func (s *Service) QueryConfig(ctx context.Context, workspaceID, appID string) (*Component, error) {
	app, err := s.apps.Get(ctx, workspaceID, appID)
	if err != nil {
		return nil, err
	}
	config, _ := json.Marshal(app.Config)
	return &Component{AppID: app.AppID, AppName: stringPtr(app.Name), Config: string(config)}, nil
}

func (s *Service) Refer(ctx context.Context, workspaceID, code string) ([]Component, error) {
	if _, err := s.Get(ctx, workspaceID, code); err != nil {
		return nil, err
	}
	apps, err := s.apps.List(ctx, workspaceID, "", "", "", 1, 100)
	if err != nil {
		return nil, err
	}
	result := make([]Component, 0)
	for _, app := range apps.Records {
		if appConfigReferences(app.Config, code) {
			result = append(result, Component{AppID: app.AppID, Name: app.Name, Type: app.Type})
		}
	}
	return result, nil
}

func (s *Service) Schema(ctx context.Context, workspaceID, code string) (map[string]any, error) {
	component, err := s.Get(ctx, workspaceID, code)
	if err != nil {
		return nil, err
	}
	var config map[string]any
	if json.Unmarshal([]byte(component.Config), &config) != nil {
		return map[string]any{"input": []any{}, "output": []any{}, "output_type": "text"}, nil
	}
	result := map[string]any{"input": []any{}, "output": []any{}, "output_type": "text"}
	if value, ok := config["input"]; ok {
		result["input"] = value
	}
	if value, ok := config["output"]; ok {
		result["output"] = value
	}
	if value, ok := config["output_type"]; ok {
		result["output_type"] = value
	}
	return result, nil
}

func (s *Service) Schemas(ctx context.Context, workspaceID string, codes []string) (map[string]any, error) {
	result := make(map[string]any, len(codes))
	for _, code := range codes {
		schema, err := s.Schema(ctx, workspaceID, code)
		if err != nil {
			return nil, err
		}
		result[code] = schema
	}
	return result, nil
}

func (s *Service) toComponent(ctx context.Context, workspaceID string, value dao.ApplicationComponent) (*Component, error) {
	appID := ""
	if value.AppID != nil {
		appID = *value.AppID
	}
	config := "{}"
	if value.Config != nil {
		config = *value.Config
	}
	description := ""
	if value.Description != nil {
		description = *value.Description
	}
	component := &Component{Code: value.Code, Name: value.Name, Type: value.Type, AppID: appID, Config: config, Description: description, Status: value.Status, GmtCreate: value.GmtCreate, GmtModified: value.GmtModified}
	if appID != "" {
		if app, err := s.apps.Get(ctx, workspaceID, appID); err == nil {
			component.AppName = stringPtr(app.Name)
		}
	}
	return component, nil
}

func appConfigReferences(config map[string]any, code string) bool {
	for _, key := range []string{"agent_components", "workflow_components"} {
		values, ok := config[key].([]any)
		if !ok {
			continue
		}
		for _, value := range values {
			if candidate, ok := value.(string); ok && candidate == code {
				return true
			}
		}
	}
	return false
}

func validate(input Input) error {
	if strings.TrimSpace(input.Type) == "" {
		return ErrTypeRequired
	}
	if strings.TrimSpace(input.Name) == "" {
		return ErrNameRequired
	}
	if len(input.Config) == 0 || string(input.Config) == "null" {
		return ErrConfigRequired
	}
	if strings.TrimSpace(input.AppID) == "" {
		return ErrAppIDRequired
	}
	if strings.TrimSpace(input.Description) == "" {
		return ErrDescriptionRequired
	}
	return nil
}
func rawJSONText(value json.RawMessage) *string {
	var data any
	if json.Unmarshal(value, &data) != nil {
		return nil
	}
	text, err := json.Marshal(data)
	if err != nil {
		return nil
	}
	result := string(text)
	return &result
}
func stringPtr(value string) *string { return &value }
func page(current, size int64) (int64, int64) {
	if current < 1 {
		current = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	return current, size
}
func newCode() (string, error) {
	var data [12]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	return "component_" + hex.EncodeToString(data[:]), nil
}

var _ = statusDeleted
