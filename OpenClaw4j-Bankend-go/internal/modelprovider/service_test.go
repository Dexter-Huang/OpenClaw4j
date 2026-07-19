package modelprovider

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/seaskyland/openclaw4j-backend-go/internal/dao"
)

func TestCreateProviderEncryptsCredentialAndMasksDetail(t *testing.T) {
	providers := newFakeProviderDAO()
	service := NewService(providers, newFakeModelDAO(), fixedClock)

	providerCode, err := service.CreateProvider(context.Background(), "ws_1", "acct_1", CreateProviderInput{
		Name: "Compatible OpenAI",
		CredentialConfig: map[string]any{
			"endpoint": "https://example.test/v1",
			"api_key":  "sk-secret",
		},
	})
	if err != nil {
		t.Fatalf("CreateProvider returned error: %v", err)
	}
	stored := providers.items[providerKey(providerCode, "ws_1")]
	if stored == nil || stored.Credential == nil || strings.Contains(*stored.Credential, "sk-secret") {
		t.Fatalf("credential was not encrypted: %#v", stored)
	}

	listed, err := service.ListProviders(context.Background(), "ws_1", "")
	if err != nil || len(listed) != 1 || listed[0].Credential != nil {
		t.Fatalf("list must omit credentials: listed=%#v err=%v", listed, err)
	}
	detail, err := service.GetProvider(context.Background(), "ws_1", providerCode)
	if err != nil {
		t.Fatalf("GetProvider returned error: %v", err)
	}
	if detail.Credential["api_key"] != "********" || detail.Credential["endpoint"] != "https://example.test/v1" {
		t.Fatalf("provider detail did not mask the credential: %#v", detail.Credential)
	}
}

func TestModelsAndSelectorAreScopedToWorkspace(t *testing.T) {
	providers := newFakeProviderDAO()
	models := newFakeModelDAO()
	service := NewService(providers, models, fixedClock)
	providers.items[providerKey("p1", "ws_1")] = &dao.Provider{WorkspaceID: "ws_1", Provider: "p1", Name: pointer("provider 1"), Enable: true, Source: "custom", Protocol: pointer("OpenAI")}
	providers.items[providerKey("p2", "ws_2")] = &dao.Provider{WorkspaceID: "ws_2", Provider: "p2", Name: pointer("provider 2"), Enable: true, Source: "custom", Protocol: pointer("OpenAI")}

	if err := service.CreateModel(context.Background(), "ws_1", "acct_1", "p1", CreateModelInput{ModelID: "model-1"}); err != nil {
		t.Fatalf("CreateModel returned error: %v", err)
	}
	if err := service.CreateModel(context.Background(), "ws_2", "acct_2", "p2", CreateModelInput{ModelID: "model-2", Type: pointer("text_embedding")}); err != nil {
		t.Fatalf("CreateModel returned error: %v", err)
	}

	selector, err := service.Selector(context.Background(), "ws_1", "llm")
	if err != nil {
		t.Fatalf("Selector returned error: %v", err)
	}
	if len(selector) != 1 || selector[0].Provider.Provider != "p1" || len(selector[0].Models) != 1 || selector[0].Models[0].ModelID != "model-1" {
		t.Fatalf("selector leaked another workspace or returned wrong grouping: %#v", selector)
	}
	if _, err := service.GetModel(context.Background(), "ws_1", "p2", "model-2"); !errors.Is(err, ErrModelNotFound) {
		t.Fatalf("expected workspace-isolated model lookup to fail, got %v", err)
	}
	rules, err := service.ParameterRules(context.Background(), "ws_1", "p1", "model-1")
	if err != nil || len(rules) != 6 || rules[0]["code"] != "temperature" {
		t.Fatalf("unexpected LLM parameter rules: %#v err=%v", rules, err)
	}
}

func fixedClock() time.Time                           { return time.Date(2026, 7, 19, 3, 0, 0, 0, time.UTC) }
func pointer(value string) *string                    { return &value }
func providerKey(provider, workspaceID string) string { return workspaceID + ":" + provider }
func modelKey(provider, modelID, workspaceID string) string {
	return workspaceID + ":" + provider + ":" + modelID
}

type fakeProviderDAO struct {
	items map[string]*dao.Provider
}

func newFakeProviderDAO() *fakeProviderDAO {
	return &fakeProviderDAO{items: map[string]*dao.Provider{}}
}
func (d *fakeProviderDAO) Create(_ context.Context, provider dao.Provider) error {
	d.items[providerKey(provider.Provider, provider.WorkspaceID)] = &provider
	return nil
}
func (d *fakeProviderDAO) FindByCodeAndWorkspace(_ context.Context, provider, workspaceID string) (*dao.Provider, error) {
	item, ok := d.items[providerKey(provider, workspaceID)]
	if !ok {
		return nil, dao.ErrNotFound
	}
	copy := *item
	return &copy, nil
}
func (d *fakeProviderDAO) ListByWorkspace(_ context.Context, workspaceID, name string) ([]dao.Provider, error) {
	items := make([]dao.Provider, 0)
	for _, item := range d.items {
		if item.WorkspaceID == workspaceID && (name == "" || strings.Contains(strings.ToLower(valueOrEmpty(item.Name)), strings.ToLower(name))) {
			items = append(items, *item)
		}
	}
	return items, nil
}
func (d *fakeProviderDAO) Update(_ context.Context, provider dao.Provider) error {
	if _, ok := d.items[providerKey(provider.Provider, provider.WorkspaceID)]; !ok {
		return dao.ErrNotFound
	}
	d.items[providerKey(provider.Provider, provider.WorkspaceID)] = &provider
	return nil
}
func (d *fakeProviderDAO) Delete(_ context.Context, provider, workspaceID string) error {
	delete(d.items, providerKey(provider, workspaceID))
	return nil
}
func (d *fakeProviderDAO) CountModels(context.Context, string, string) (int64, error) { return 0, nil }

type fakeModelDAO struct {
	items map[string]*dao.Model
}

func newFakeModelDAO() *fakeModelDAO { return &fakeModelDAO{items: map[string]*dao.Model{}} }
func (d *fakeModelDAO) Create(_ context.Context, model dao.Model) error {
	d.items[modelKey(model.Provider, model.ModelID, model.WorkspaceID)] = &model
	return nil
}
func (d *fakeModelDAO) FindByProviderAndIDAndWorkspace(_ context.Context, provider, modelID, workspaceID string) (*dao.Model, error) {
	item, ok := d.items[modelKey(provider, modelID, workspaceID)]
	if !ok {
		return nil, dao.ErrNotFound
	}
	copy := *item
	return &copy, nil
}
func (d *fakeModelDAO) ListByProviderAndWorkspace(_ context.Context, provider, workspaceID string) ([]dao.Model, error) {
	items := make([]dao.Model, 0)
	for _, item := range d.items {
		if item.Provider == provider && item.WorkspaceID == workspaceID {
			items = append(items, *item)
		}
	}
	return items, nil
}
func (d *fakeModelDAO) ListByWorkspace(_ context.Context, workspaceID string) ([]dao.Model, error) {
	items := make([]dao.Model, 0)
	for _, item := range d.items {
		if item.WorkspaceID == workspaceID {
			items = append(items, *item)
		}
	}
	return items, nil
}
func (d *fakeModelDAO) Update(_ context.Context, model dao.Model) error {
	if _, ok := d.items[modelKey(model.Provider, model.ModelID, model.WorkspaceID)]; !ok {
		return dao.ErrNotFound
	}
	d.items[modelKey(model.Provider, model.ModelID, model.WorkspaceID)] = &model
	return nil
}
func (d *fakeModelDAO) Delete(_ context.Context, provider, modelID, workspaceID string) error {
	delete(d.items, modelKey(provider, modelID, workspaceID))
	return nil
}
