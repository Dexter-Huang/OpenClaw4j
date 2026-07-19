package modelprovider

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/seaskyland/openclaw4j-backend-go/internal/dao"
)

var (
	ErrProviderNameRequired = errors.New("provider name is required")
	ErrProviderNotFound     = errors.New("provider not found")
	ErrProviderExists       = errors.New("provider already exists")
	ErrModelIDRequired      = errors.New("model id is required")
	ErrModelNotFound        = errors.New("model not found")
	ErrModelExists          = errors.New("model already exists")
)

type Service struct {
	providers dao.ProviderDAO
	models    dao.ModelDAO
	clock     func() time.Time
}

type CreateProviderInput struct {
	Name                string         `json:"name"`
	Description         *string        `json:"description"`
	Icon                *string        `json:"icon"`
	CredentialConfig    map[string]any `json:"credential_config"`
	Protocol            *string        `json:"protocol"`
	SupportedModelTypes *string        `json:"supported_model_types"`
}

type UpdateProviderInput struct {
	Name                *string        `json:"name"`
	Description         *string        `json:"description"`
	Enable              *bool          `json:"enable"`
	Icon                *string        `json:"icon"`
	CredentialConfig    map[string]any `json:"credential_config"`
	Protocol            *string        `json:"protocol"`
	SupportedModelTypes *string        `json:"supported_model_types"`
}

type CreateModelInput struct {
	ModelID   string  `json:"model_id"`
	ModelName *string `json:"model_name"`
	Type      *string `json:"type"`
	Tags      *string `json:"tags"`
	Icon      *string `json:"icon"`
}

type UpdateModelInput struct {
	ModelName *string `json:"model_name"`
	Icon      *string `json:"icon"`
	Tags      *string `json:"tags"`
	Enable    *bool   `json:"enable"`
}

type Provider struct {
	Provider            string           `json:"provider"`
	Name                string           `json:"name"`
	Description         *string          `json:"description,omitempty"`
	Icon                *string          `json:"icon,omitempty"`
	Credential          map[string]any   `json:"credential,omitempty"`
	Enable              bool             `json:"enable"`
	Source              string           `json:"source"`
	Protocol            string           `json:"protocol"`
	SupportedModelTypes []string         `json:"supported_model_types"`
	ModelCount          int64            `json:"model_count,omitempty"`
	GmtCreate           time.Time        `json:"gmt_create,omitempty"`
	GmtModified         time.Time        `json:"gmt_modified,omitempty"`
	Creator             *string          `json:"creator,omitempty"`
	Modifier            *string          `json:"modifier,omitempty"`
	CredentialSpecs     []CredentialSpec `json:"credential_specs,omitempty"`
}

type CredentialSpec struct {
	Code        string `json:"code"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	PlaceHolder string `json:"place_holder"`
	Required    bool   `json:"required"`
	Sensitive   bool   `json:"sensitive"`
}

type Model struct {
	ModelID  string   `json:"model_id"`
	Name     string   `json:"name"`
	Provider string   `json:"provider"`
	Icon     *string  `json:"icon,omitempty"`
	Tags     []string `json:"tags"`
	Type     string   `json:"type"`
	Mode     string   `json:"mode,omitempty"`
	Enable   bool     `json:"enable"`
	Source   string   `json:"source"`
}

type SelectorGroup struct {
	Provider Provider `json:"provider"`
	Models   []Model  `json:"models"`
}

func NewService(providers dao.ProviderDAO, models dao.ModelDAO, clock func() time.Time) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{providers: providers, models: models, clock: clock}
}

func (s *Service) CreateProvider(ctx context.Context, workspaceID, accountID string, input CreateProviderInput) (string, error) {
	if s.providers == nil {
		return "", fmt.Errorf("provider service dependencies are unavailable")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return "", ErrProviderNameRequired
	}
	providerCode, err := newProviderCode()
	if err != nil {
		return "", err
	}
	credential, err := encodeCredential(input.CredentialConfig)
	if err != nil {
		return "", err
	}
	now := s.clock()
	if err := s.providers.Create(ctx, dao.Provider{
		WorkspaceID: workspaceID, Icon: trimPointer(input.Icon), Name: stringPointer(name), Description: trimPointer(input.Description), Provider: providerCode,
		Enable: true, Source: "custom", Credential: credential, SupportedModelTypes: stringPointer(strings.Join(normalizeCSV(input.SupportedModelTypes, "llm"), ",")),
		Protocol: stringPointer(normalizeProtocol(input.Protocol)), GmtCreate: now, GmtModified: now, Creator: stringPointer(accountID), Modifier: stringPointer(accountID),
	}); err != nil {
		return "", fmt.Errorf("create provider: %w", err)
	}
	return providerCode, nil
}

func (s *Service) UpdateProvider(ctx context.Context, workspaceID, accountID, providerCode string, input UpdateProviderInput) error {
	provider, err := s.findProvider(ctx, providerCode, workspaceID)
	if err != nil {
		return err
	}
	if name := trimPointer(input.Name); name != nil {
		provider.Name = name
	}
	if description := trimPointer(input.Description); description != nil {
		provider.Description = description
	}
	if icon := trimPointer(input.Icon); icon != nil {
		provider.Icon = icon
	}
	if input.Enable != nil {
		provider.Enable = *input.Enable
	}
	if input.SupportedModelTypes != nil && strings.TrimSpace(*input.SupportedModelTypes) != "" {
		provider.SupportedModelTypes = stringPointer(strings.Join(normalizeCSV(input.SupportedModelTypes, "llm"), ","))
	}
	if protocol := trimPointer(input.Protocol); protocol != nil {
		provider.Protocol = protocol
	}
	if input.CredentialConfig != nil {
		credential, err := encodeCredential(input.CredentialConfig)
		if err != nil {
			return err
		}
		provider.Credential = credential
	}
	provider.Modifier = stringPointer(accountID)
	provider.GmtModified = s.clock()
	if err := s.providers.Update(ctx, *provider); err != nil {
		return fmt.Errorf("update provider: %w", err)
	}
	return nil
}

func (s *Service) DeleteProvider(ctx context.Context, workspaceID, providerCode string) error {
	if _, err := s.findProvider(ctx, providerCode, workspaceID); err != nil {
		return err
	}
	if err := s.providers.Delete(ctx, providerCode, workspaceID); err != nil {
		return fmt.Errorf("delete provider: %w", err)
	}
	return nil
}

func (s *Service) ListProviders(ctx context.Context, workspaceID, name string) ([]Provider, error) {
	if s.providers == nil {
		return nil, fmt.Errorf("provider service dependencies are unavailable")
	}
	items, err := s.providers.ListByWorkspace(ctx, workspaceID, strings.TrimSpace(name))
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}
	result := make([]Provider, 0, len(items))
	for _, item := range items {
		count, err := s.providers.CountModels(ctx, item.Provider, workspaceID)
		if err != nil {
			return nil, fmt.Errorf("count provider models: %w", err)
		}
		result = append(result, toProvider(&item, false, count))
	}
	return result, nil
}

func (s *Service) GetProvider(ctx context.Context, workspaceID, providerCode string) (*Provider, error) {
	provider, err := s.findProvider(ctx, providerCode, workspaceID)
	if err != nil {
		return nil, err
	}
	count, err := s.providers.CountModels(ctx, providerCode, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("count provider models: %w", err)
	}
	result := toProvider(provider, true, count)
	return &result, nil
}

func (s *Service) CreateModel(ctx context.Context, workspaceID, accountID, providerCode string, input CreateModelInput) error {
	modelID := strings.TrimSpace(input.ModelID)
	if modelID == "" {
		return ErrModelIDRequired
	}
	if _, err := s.findProvider(ctx, providerCode, workspaceID); err != nil {
		return err
	}
	if existing, err := s.models.FindByProviderAndIDAndWorkspace(ctx, providerCode, modelID, workspaceID); err == nil && existing != nil {
		return ErrModelExists
	} else if !errors.Is(err, dao.ErrNotFound) {
		return fmt.Errorf("find model: %w", err)
	}
	now := s.clock()
	name := trimPointer(input.ModelName)
	if name == nil {
		name = stringPointer(modelID)
	}
	if err := s.models.Create(ctx, dao.Model{
		WorkspaceID: workspaceID, Icon: trimPointer(input.Icon), Name: name, Type: stringPointer(normalizeModelType(input.Type)), Mode: stringPointer("chat"), ModelID: modelID,
		Provider: providerCode, Enable: true, Tags: csvPointer(input.Tags), Source: "custom", GmtCreate: now, GmtModified: now, Creator: stringPointer(accountID), Modifier: stringPointer(accountID),
	}); err != nil {
		return fmt.Errorf("create model: %w", err)
	}
	return nil
}

func (s *Service) UpdateModel(ctx context.Context, workspaceID, accountID, providerCode, modelID string, input UpdateModelInput) error {
	model, err := s.findModel(ctx, providerCode, modelID, workspaceID)
	if err != nil {
		return err
	}
	if name := trimPointer(input.ModelName); name != nil {
		model.Name = name
	}
	if icon := trimPointer(input.Icon); icon != nil {
		model.Icon = icon
	}
	if input.Tags != nil && strings.TrimSpace(*input.Tags) != "" {
		model.Tags = csvPointer(input.Tags)
	}
	if input.Enable != nil {
		model.Enable = *input.Enable
	}
	model.Modifier = stringPointer(accountID)
	model.GmtModified = s.clock()
	if err := s.models.Update(ctx, *model); err != nil {
		return fmt.Errorf("update model: %w", err)
	}
	return nil
}

func (s *Service) DeleteModel(ctx context.Context, workspaceID, providerCode, modelID string) error {
	if _, err := s.findModel(ctx, providerCode, modelID, workspaceID); err != nil {
		return err
	}
	if err := s.models.Delete(ctx, providerCode, modelID, workspaceID); err != nil {
		return fmt.Errorf("delete model: %w", err)
	}
	return nil
}

func (s *Service) ListModels(ctx context.Context, workspaceID, providerCode string) ([]Model, error) {
	items, err := s.models.ListByProviderAndWorkspace(ctx, providerCode, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list models: %w", err)
	}
	result := make([]Model, 0, len(items))
	for _, item := range items {
		result = append(result, toModel(&item))
	}
	return result, nil
}

func (s *Service) GetModel(ctx context.Context, workspaceID, providerCode, modelID string) (*Model, error) {
	item, err := s.findModel(ctx, providerCode, modelID, workspaceID)
	if err != nil {
		return nil, err
	}
	result := toModel(item)
	return &result, nil
}

func (s *Service) Selector(ctx context.Context, workspaceID, modelType string) ([]SelectorGroup, error) {
	providers, err := s.ListProviders(ctx, workspaceID, "")
	if err != nil {
		return nil, err
	}
	models, err := s.models.ListByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list workspace models: %w", err)
	}
	byProvider := make(map[string][]Model)
	for _, item := range models {
		converted := toModel(&item)
		if converted.Type == modelType {
			byProvider[converted.Provider] = append(byProvider[converted.Provider], converted)
		}
	}
	result := make([]SelectorGroup, 0)
	for _, provider := range providers {
		if providerModels := byProvider[provider.Provider]; len(providerModels) > 0 {
			result = append(result, SelectorGroup{Provider: provider, Models: providerModels})
		}
	}
	return result, nil
}

func (s *Service) ParameterRules(ctx context.Context, workspaceID, providerCode, modelID string) ([]map[string]any, error) {
	model, err := s.findModel(ctx, providerCode, modelID, workspaceID)
	if err != nil {
		return nil, err
	}
	if valueOrDefault(model.Type, "llm") != "llm" {
		return []map[string]any{}, nil
	}
	return defaultLLMParameterRules(), nil
}

func defaultLLMParameterRules() []map[string]any {
	return []map[string]any{
		parameterRule("temperature", 0, 0, 1, 2, "Controls randomness.", "温度控制随机性。"),
		parameterRule("top_p", 1, 0, 1, 2, "Controls diversity via nucleus sampling.", "通过核心采样控制多样性。"),
		parameterRule("presence_penalty", 0, 0, 1, 2, "Applies a penalty to tokens already in the text.", "对文本中已有的标记施加惩罚。"),
		parameterRule("frequency_penalty", 0, 0, 1, 2, "Applies a penalty to tokens that appear in the text.", "对文本中出现的标记施加惩罚。"),
		parameterRule("max_tokens", 512, 1, 4096, 0, "Specifies the upper limit on the generated result length.", "指定生成结果长度的上限。"),
		{
			"code": "seed", "name": "seed", "type": "NUMBER", "required": false,
			"description": "Best-effort deterministic sampling seed.",
			"help":        map[string]string{"en_US": "Best-effort deterministic sampling seed.", "zh_Hans": "用于尽量确定性采样的种子。"},
		},
	}
}

func parameterRule(code string, defaultValue, min, max, precision int, description, chineseHelp string) map[string]any {
	return map[string]any{
		"code": code, "name": code, "type": "NUMBER", "default_value": defaultValue, "min": min, "max": max, "precision": precision, "required": false,
		"description": description, "help": map[string]string{"en_US": description, "zh_Hans": chineseHelp},
	}
	// Java 的 OpenAIProvider 对非 LLM 模型也返回空规则。
}

func (s *Service) findProvider(ctx context.Context, providerCode, workspaceID string) (*dao.Provider, error) {
	if s.providers == nil {
		return nil, fmt.Errorf("provider service dependencies are unavailable")
	}
	item, err := s.providers.FindByCodeAndWorkspace(ctx, strings.TrimSpace(providerCode), workspaceID)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, ErrProviderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find provider: %w", err)
	}
	return item, nil
}

func (s *Service) findModel(ctx context.Context, providerCode, modelID, workspaceID string) (*dao.Model, error) {
	if s.models == nil {
		return nil, fmt.Errorf("model service dependencies are unavailable")
	}
	item, err := s.models.FindByProviderAndIDAndWorkspace(ctx, strings.TrimSpace(providerCode), strings.TrimSpace(modelID), workspaceID)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, ErrModelNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find model: %w", err)
	}
	return item, nil
}

func toProvider(item *dao.Provider, includeCredential bool, count int64) Provider {
	result := Provider{
		Provider: item.Provider, Name: valueOrEmpty(item.Name), Description: item.Description, Icon: item.Icon, Enable: item.Enable, Source: item.Source,
		Protocol: valueOrDefault(item.Protocol, "OpenAI"), SupportedModelTypes: splitCSV(item.SupportedModelTypes), ModelCount: count,
		GmtCreate: item.GmtCreate, GmtModified: item.GmtModified, Creator: item.Creator, Modifier: item.Modifier,
	}
	if includeCredential {
		result.Credential = maskedCredential(item.Credential)
		result.CredentialSpecs = openAICredentialSpecs()
	}
	return result
}

func toModel(item *dao.Model) Model {
	return Model{ModelID: item.ModelID, Name: valueOrDefault(item.Name, item.ModelID), Provider: item.Provider, Icon: item.Icon, Tags: splitCSV(item.Tags), Type: valueOrDefault(item.Type, "llm"), Mode: valueOrDefault(item.Mode, "chat"), Enable: item.Enable, Source: item.Source}
}

func encodeCredential(input map[string]any) (*string, error) {
	if input == nil {
		return stringPointer("{}"), nil
	}
	credential := make(map[string]any, len(input))
	for key, value := range input {
		credential[key] = value
	}
	if key, ok := credential["api_key"].(string); ok && strings.TrimSpace(key) != "" {
		encrypted, err := encryptAPIKey(key)
		if err != nil {
			return nil, err
		}
		credential["api_key"] = encrypted
	}
	encoded, err := json.Marshal(credential)
	if err != nil {
		return nil, fmt.Errorf("encode provider credential: %w", err)
	}
	return stringPointer(string(encoded)), nil
}

func maskedCredential(encoded *string) map[string]any {
	if encoded == nil || strings.TrimSpace(*encoded) == "" {
		return map[string]any{}
	}
	credential := map[string]any{}
	if err := json.Unmarshal([]byte(*encoded), &credential); err != nil {
		return map[string]any{}
	}
	if value, ok := credential["api_key"].(string); ok && value != "" {
		credential["api_key"] = "********"
	}
	return credential
}

func encryptAPIKey(value string) (string, error) {
	block, _ := pem.Decode([]byte(javaCompatiblePublicKey))
	if block == nil {
		return "", errors.New("decode provider public key")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parse provider public key: %w", err)
	}
	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return "", errors.New("provider public key is not RSA")
	}
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaKey, []byte(value), nil)
	if err != nil {
		return "", fmt.Errorf("encrypt provider api key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func newProviderCode() (string, error) {
	var bytes [4]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("generate provider code: %w", err)
	}
	return hex.EncodeToString(bytes[:]), nil
}

func normalizeProtocol(value *string) string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return "OpenAI"
	}
	return strings.TrimSpace(*value)
}

func normalizeModelType(value *string) string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return "llm"
	}
	return strings.TrimSpace(*value)
}

func normalizeCSV(value *string, fallback string) []string {
	values := splitCSV(value)
	if len(values) == 0 {
		return []string{fallback}
	}
	return values
}

func splitCSV(value *string) []string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return []string{}
	}
	parts := strings.Split(*value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if item := strings.TrimSpace(part); item != "" {
			result = append(result, item)
		}
	}
	return result
}

func csvPointer(value *string) *string {
	items := splitCSV(value)
	if len(items) == 0 {
		return nil
	}
	return stringPointer(strings.Join(items, ","))
}

func trimPointer(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return stringPointer(trimmed)
}

func valueOrEmpty(value *string) string { return valueOrDefault(value, "") }
func valueOrDefault(value *string, fallback string) string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return fallback
	}
	return *value
}
func stringPointer(value string) *string { return &value }

func openAICredentialSpecs() []CredentialSpec {
	return []CredentialSpec{
		{Code: "endpoint", DisplayName: "Endpoint", Description: "OpenAI-compatible API endpoint", PlaceHolder: "https://api.openai.com/v1", Required: false},
		{Code: "api_key", DisplayName: "API Key", Description: "Provider API key", PlaceHolder: "sk-...", Required: true, Sensitive: true},
		{Code: "completions_path", DisplayName: "Completions Path", Description: "Optional chat completions path", PlaceHolder: "/chat/completions", Required: false},
		{Code: "embeddings_path", DisplayName: "Embeddings Path", Description: "Optional embeddings path", PlaceHolder: "/embeddings", Required: false},
	}
}

// 与 Java RSACryptUtils 使用同一公钥，确保 Go 创建的凭证可被仍在运行的 Java 后端读取。
const javaCompatiblePublicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAkgQdWAybEJoAPR7EZpfP
QjTssBGDKOR88r0lAgKIbxVoWywHS7VBvjMrXPiuPV6wYdfngQz57b2xkp+bQVda
RBc0tB3nXxkHd4AgPB78MbCNOtR/aAiyoMdTq7L0/Sgexq6PcmDD/J5rLM2ACTKD
gIroq3GnpN7Oq8ikNT9RlrKCGzXBuSF6LMFoogk7xk1v03LLycjtAQtKvjV3DlxW
n0QFnS30rspN+wAFscIuVTbwwET9WhCB7f6iPHbJ8prga1kGQa+pcAbf3iiA3aUF
oeQoPPv4BQaaG8/+7badEFZaCDmnc1/xRVR0LqXu5HzCuM0XkfjC5UhvI4UGeWlA
bwIDAQAB
-----END PUBLIC KEY-----`
