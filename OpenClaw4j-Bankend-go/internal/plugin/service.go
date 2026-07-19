package plugin

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/seaskyland/openclaw4j-backend-go/internal/dao"
)

const (
	statusDraft     int16 = 1
	statusPublished int16 = 2
	testNotTested   int16 = 1
	testPassed      int16 = 2
)

var (
	ErrNameRequired        = errors.New("name is required")
	ErrDescriptionRequired = errors.New("description is required")
	ErrConfigRequired      = errors.New("config is required")
	ErrPluginNotFound      = errors.New("plugin not found")
	ErrToolNotFound        = errors.New("tool not found")
	ErrPluginNameExists    = errors.New("plugin name already exists")
	ErrToolNameExists      = errors.New("tool name already exists")
	ErrToolNotTested       = errors.New("tool must pass testing before publishing")
)

type PluginInput struct {
	PluginID    string          `json:"plugin_id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Config      json.RawMessage `json:"config"`
	Source      string          `json:"source"`
}

type Plugin struct {
	PluginID    string          `json:"plugin_id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Config      json.RawMessage `json:"config"`
	Source      string          `json:"source"`
	GmtCreate   time.Time       `json:"gmt_create"`
	GmtModified time.Time       `json:"gmt_modified"`
}

type ToolInput struct {
	PluginID    string          `json:"plugin_id"`
	ToolID      string          `json:"tool_id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Config      json.RawMessage `json:"config"`
	APISchema   json.RawMessage `json:"api_schema"`
	Enabled     *bool           `json:"enabled"`
}

type Tool struct {
	PluginID    string          `json:"plugin_id"`
	ToolID      string          `json:"tool_id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Config      json.RawMessage `json:"config"`
	APISchema   json.RawMessage `json:"api_schema"`
	Enabled     bool            `json:"enabled"`
	TestStatus  string          `json:"test_status"`
	Status      string          `json:"status"`
	GmtCreate   time.Time       `json:"gmt_create"`
	GmtModified time.Time       `json:"gmt_modified"`
}

type TestResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type Page[T any] struct {
	Current int64 `json:"current"`
	Size    int64 `json:"size"`
	Total   int64 `json:"total"`
	Records []T   `json:"records"`
}

type Service struct {
	dao   dao.PluginDAO
	clock func() time.Time
}

func NewService(data dao.PluginDAO, clock func() time.Time) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{dao: data, clock: clock}
}

func (s *Service) Create(ctx context.Context, workspaceID, accountID string, input PluginInput) (string, error) {
	config, err := validatePlugin(input)
	if err != nil {
		return "", err
	}
	values, err := s.dao.List(ctx, workspaceID, input.Name, -1)
	if err != nil {
		return "", err
	}
	for _, value := range values {
		if value.Name == strings.TrimSpace(input.Name) {
			return "", ErrPluginNameExists
		}
	}
	id, err := newID("plugin_")
	if err != nil {
		return "", err
	}
	now := s.clock()
	description := strings.TrimSpace(input.Description)
	return id, s.dao.Create(ctx, dao.Plugin{PluginID: id, WorkspaceID: workspaceID, Type: "custom", Status: statusDraft, Name: strings.TrimSpace(input.Name), Description: &description, Config: stringPointer(config), Source: defaultString(input.Source, "user"), GmtCreate: now, GmtModified: now, Creator: accountID, Modifier: accountID})
}

func (s *Service) Update(ctx context.Context, workspaceID, accountID, pluginID string, input PluginInput) error {
	config, err := validatePlugin(input)
	if err != nil {
		return err
	}
	value, err := s.dao.Find(ctx, pluginID, workspaceID)
	if errors.Is(err, dao.ErrNotFound) {
		return ErrPluginNotFound
	}
	if err != nil {
		return err
	}
	value.Name = strings.TrimSpace(input.Name)
	description := strings.TrimSpace(input.Description)
	value.Description = &description
	value.Config = stringPointer(config)
	value.Source = defaultString(input.Source, value.Source)
	value.Modifier = accountID
	value.GmtModified = s.clock()
	return s.dao.Update(ctx, *value)
}

func (s *Service) Delete(ctx context.Context, workspaceID, pluginID string) error {
	return s.dao.Delete(ctx, pluginID, workspaceID)
}
func (s *Service) Get(ctx context.Context, workspaceID, pluginID string) (*Plugin, error) {
	value, err := s.dao.Find(ctx, pluginID, workspaceID)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, ErrPluginNotFound
	}
	if err != nil {
		return nil, err
	}
	return mapPlugin(*value), nil
}
func (s *Service) List(ctx context.Context, workspaceID, name string, current, size int64) (*Page[Plugin], error) {
	values, err := s.dao.List(ctx, workspaceID, strings.TrimSpace(name), -1)
	if err != nil {
		return nil, err
	}
	items := make([]Plugin, 0, len(values))
	for _, value := range values {
		items = append(items, *mapPlugin(value))
	}
	return page(items, current, size), nil
}

func (s *Service) CreateTool(ctx context.Context, workspaceID, accountID, pluginID string, input ToolInput) (string, error) {
	if _, err := s.Get(ctx, workspaceID, pluginID); err != nil {
		return "", err
	}
	config, schema, err := validateTool(input)
	if err != nil {
		return "", err
	}
	tools, err := s.dao.ListTools(ctx, pluginID, workspaceID, input.Name)
	if err != nil {
		return "", err
	}
	for _, value := range tools {
		if value.Name == strings.TrimSpace(input.Name) {
			return "", ErrToolNameExists
		}
	}
	id, err := newID("tool_")
	if err != nil {
		return "", err
	}
	now := s.clock()
	description := strings.TrimSpace(input.Description)
	enabled := false
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	return id, s.dao.CreateTool(ctx, dao.Tool{PluginID: pluginID, ToolID: id, WorkspaceID: workspaceID, Status: statusDraft, Enabled: enabled, TestStatus: testNotTested, Name: strings.TrimSpace(input.Name), Description: &description, Config: config, APISchema: schema, GmtCreate: now, GmtModified: now, Creator: accountID, Modifier: accountID})
}

func (s *Service) UpdateTool(ctx context.Context, workspaceID, accountID, pluginID, toolID string, input ToolInput) error {
	config, schema, err := validateTool(input)
	if err != nil {
		return err
	}
	value, err := s.dao.FindTool(ctx, pluginID, toolID, workspaceID)
	if errors.Is(err, dao.ErrNotFound) {
		return ErrToolNotFound
	}
	if err != nil {
		return err
	}
	value.Name = strings.TrimSpace(input.Name)
	description := strings.TrimSpace(input.Description)
	value.Description = &description
	value.Config = config
	value.APISchema = schema
	if input.Enabled != nil {
		value.Enabled = *input.Enabled
	}
	if value.Status == statusPublished {
		value.Status = statusDraft
	}
	value.Modifier = accountID
	value.GmtModified = s.clock()
	return s.dao.UpdateTool(ctx, *value)
}

func (s *Service) DeleteTool(ctx context.Context, workspaceID, pluginID, toolID string) error {
	return s.dao.DeleteTool(ctx, pluginID, toolID, workspaceID)
}
func (s *Service) GetTool(ctx context.Context, workspaceID, pluginID, toolID string) (*Tool, error) {
	value, err := s.dao.FindTool(ctx, pluginID, toolID, workspaceID)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, ErrToolNotFound
	}
	if err != nil {
		return nil, err
	}
	return mapTool(*value), nil
}
func (s *Service) ListTools(ctx context.Context, workspaceID, pluginID, name string, current, size int64) (*Page[Tool], error) {
	values, err := s.dao.ListTools(ctx, pluginID, workspaceID, strings.TrimSpace(name))
	if err != nil {
		return nil, err
	}
	items := make([]Tool, 0, len(values))
	for _, value := range values {
		items = append(items, *mapTool(value))
	}
	return page(items, current, size), nil
}
func (s *Service) ListToolsByIDs(ctx context.Context, workspaceID string, ids []string) ([]Tool, error) {
	values, err := s.dao.ListToolsByIDs(ctx, workspaceID, ids)
	if err != nil {
		return nil, err
	}
	items := make([]Tool, 0, len(values))
	for _, value := range values {
		items = append(items, *mapTool(value))
	}
	return items, nil
}
func (s *Service) SetEnabled(ctx context.Context, workspaceID, accountID, toolID string, enabled bool) error {
	value, err := s.dao.FindToolByID(ctx, toolID, workspaceID)
	if errors.Is(err, dao.ErrNotFound) {
		return ErrToolNotFound
	}
	if err != nil {
		return err
	}
	value.Enabled = enabled
	value.Modifier = accountID
	value.GmtModified = s.clock()
	return s.dao.UpdateTool(ctx, *value)
}
func (s *Service) MarkTestPassed(ctx context.Context, workspaceID, accountID, pluginID, toolID string) error {
	value, err := s.dao.FindTool(ctx, pluginID, toolID, workspaceID)
	if errors.Is(err, dao.ErrNotFound) {
		return ErrToolNotFound
	}
	if err != nil {
		return err
	}
	value.TestStatus = testPassed
	value.Modifier = accountID
	value.GmtModified = s.clock()
	return s.dao.UpdateTool(ctx, *value)
}

func (s *Service) TestTool(ctx context.Context, workspaceID, accountID, pluginID, toolID string, arguments map[string]any) (*TestResult, error) {
	pluginValue, err := s.Get(ctx, workspaceID, pluginID)
	if err != nil {
		return nil, err
	}
	toolValue, err := s.GetTool(ctx, workspaceID, pluginID, toolID)
	if err != nil {
		return nil, err
	}
	var pluginConfig struct {
		Server string `json:"server"`
		Auth   struct {
			Type                  string `json:"type"`
			AuthorizationPosition string `json:"authorization_position"`
			AuthorizationType     string `json:"authorization_type"`
			AuthorizationKey      string `json:"authorization_key"`
			AuthorizationValue    string `json:"authorization_value"`
		} `json:"auth"`
	}
	var toolConfig struct {
		Path          string `json:"path"`
		RequestMethod string `json:"request_method"`
		ContentType   string `json:"content_type"`
	}
	if json.Unmarshal(pluginValue.Config, &pluginConfig) != nil || json.Unmarshal(toolValue.Config, &toolConfig) != nil {
		return nil, ErrConfigRequired
	}
	baseURL, err := url.Parse(pluginConfig.Server)
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" || (baseURL.Scheme != "http" && baseURL.Scheme != "https") {
		return nil, ErrConfigRequired
	}
	pathURL, err := url.Parse(toolConfig.Path)
	if err != nil {
		return nil, ErrConfigRequired
	}
	target := baseURL.ResolveReference(pathURL)
	method := strings.ToUpper(toolConfig.RequestMethod)
	var body io.Reader
	if method == http.MethodGet {
		query := target.Query()
		for key, value := range arguments {
			query.Set(key, fmt.Sprint(value))
		}
		target.RawQuery = query.Encode()
	} else {
		payload, marshalErr := json.Marshal(arguments)
		if marshalErr != nil {
			return nil, marshalErr
		}
		body = strings.NewReader(string(payload))
	}
	request, err := http.NewRequestWithContext(ctx, method, target.String(), body)
	if err != nil {
		return nil, err
	}
	if method != http.MethodGet {
		request.Header.Set("Content-Type", defaultString(toolConfig.ContentType, "application/json"))
	}
	applyAuth(request, pluginConfig.Auth.Type, pluginConfig.Auth.AuthorizationPosition, pluginConfig.Auth.AuthorizationType, pluginConfig.Auth.AuthorizationKey, pluginConfig.Auth.AuthorizationValue)
	response, err := (&http.Client{Timeout: 15 * time.Second}).Do(request)
	if err != nil {
		_ = s.markTestStatus(ctx, workspaceID, accountID, pluginID, toolID, false)
		return &TestResult{Success: false, Message: err.Error()}, nil
	}
	defer response.Body.Close()
	payload, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if readErr != nil {
		return nil, readErr
	}
	success := response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices
	if err := s.markTestStatus(ctx, workspaceID, accountID, pluginID, toolID, success); err != nil {
		return nil, err
	}
	result := &TestResult{Success: success, Message: response.Status}
	if json.Valid(payload) {
		var data any
		_ = json.Unmarshal(payload, &data)
		result.Data = data
	} else {
		result.Data = string(payload)
	}
	return result, nil
}

// InvokeTool 供工作流插件节点调用已发布工具。它复用插件的请求、认证和响应解析规则，
// 但不会修改测试状态或最后修改人，避免一次正常工作流运行改变插件的管理元数据。
func (s *Service) InvokeTool(ctx context.Context, workspaceID, pluginID, toolID string, arguments map[string]any) (*TestResult, error) {
	pluginValue, err := s.Get(ctx, workspaceID, pluginID)
	if err != nil {
		return nil, err
	}
	toolValue, err := s.GetTool(ctx, workspaceID, pluginID, toolID)
	if err != nil {
		return nil, err
	}
	var pluginConfig struct {
		Server string `json:"server"`
		Auth   struct {
			Type                  string `json:"type"`
			AuthorizationPosition string `json:"authorization_position"`
			AuthorizationType     string `json:"authorization_type"`
			AuthorizationKey      string `json:"authorization_key"`
			AuthorizationValue    string `json:"authorization_value"`
		} `json:"auth"`
	}
	var toolConfig struct {
		Path          string `json:"path"`
		RequestMethod string `json:"request_method"`
		ContentType   string `json:"content_type"`
	}
	if json.Unmarshal(pluginValue.Config, &pluginConfig) != nil || json.Unmarshal(toolValue.Config, &toolConfig) != nil {
		return nil, ErrConfigRequired
	}
	baseURL, err := url.Parse(pluginConfig.Server)
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" || (baseURL.Scheme != "http" && baseURL.Scheme != "https") {
		return nil, ErrConfigRequired
	}
	pathURL, err := url.Parse(toolConfig.Path)
	if err != nil {
		return nil, ErrConfigRequired
	}
	target := baseURL.ResolveReference(pathURL)
	method := strings.ToUpper(toolConfig.RequestMethod)
	var body io.Reader
	if method == http.MethodGet {
		query := target.Query()
		for key, value := range arguments {
			query.Set(key, fmt.Sprint(value))
		}
		target.RawQuery = query.Encode()
	} else {
		payload, marshalErr := json.Marshal(arguments)
		if marshalErr != nil {
			return nil, marshalErr
		}
		body = strings.NewReader(string(payload))
	}
	request, err := http.NewRequestWithContext(ctx, method, target.String(), body)
	if err != nil {
		return nil, err
	}
	if method != http.MethodGet {
		request.Header.Set("Content-Type", defaultString(toolConfig.ContentType, "application/json"))
	}
	applyAuth(request, pluginConfig.Auth.Type, pluginConfig.Auth.AuthorizationPosition, pluginConfig.Auth.AuthorizationType, pluginConfig.Auth.AuthorizationKey, pluginConfig.Auth.AuthorizationValue)
	response, err := (&http.Client{Timeout: 15 * time.Second}).Do(request)
	if err != nil {
		return &TestResult{Success: false, Message: err.Error()}, nil
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	result := &TestResult{Success: response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices, Message: response.Status}
	if json.Valid(payload) {
		_ = json.Unmarshal(payload, &result.Data)
	} else {
		result.Data = string(payload)
	}
	return result, nil
}

func (s *Service) markTestStatus(ctx context.Context, workspaceID, accountID, pluginID, toolID string, passed bool) error {
	value, err := s.dao.FindTool(ctx, pluginID, toolID, workspaceID)
	if errors.Is(err, dao.ErrNotFound) {
		return ErrToolNotFound
	}
	if err != nil {
		return err
	}
	if passed {
		value.TestStatus = testPassed
	} else {
		value.TestStatus = 3
	}
	value.Modifier, value.GmtModified = accountID, s.clock()
	return s.dao.UpdateTool(ctx, *value)
}

func applyAuth(request *http.Request, authType, position, authorizationType, key, value string) {
	if strings.EqualFold(authType, "none") || strings.TrimSpace(value) == "" {
		return
	}
	if strings.EqualFold(position, "query") {
		query := request.URL.Query()
		query.Set(defaultString(key, "api_key"), value)
		request.URL.RawQuery = query.Encode()
		return
	}
	if strings.EqualFold(authorizationType, "basic") {
		request.Header.Set("Authorization", "Basic "+value)
		return
	}
	if strings.EqualFold(authorizationType, "bearer") {
		request.Header.Set("Authorization", "Bearer "+value)
		return
	}
	request.Header.Set(defaultString(key, "Authorization"), value)
}
func (s *Service) PublishTool(ctx context.Context, workspaceID, accountID, pluginID, toolID string) error {
	value, err := s.dao.FindTool(ctx, pluginID, toolID, workspaceID)
	if errors.Is(err, dao.ErrNotFound) {
		return ErrToolNotFound
	}
	if err != nil {
		return err
	}
	if value.TestStatus != testPassed {
		return ErrToolNotTested
	}
	value.Status = statusPublished
	value.Modifier = accountID
	value.GmtModified = s.clock()
	return s.dao.UpdateTool(ctx, *value)
}

func validatePlugin(input PluginInput) (string, error) {
	if strings.TrimSpace(input.Name) == "" {
		return "", ErrNameRequired
	}
	if len(input.Config) == 0 || !json.Valid(input.Config) {
		return "", ErrConfigRequired
	}
	var config struct {
		Server string `json:"server"`
		Auth   *struct {
			Type string `json:"type"`
		} `json:"auth"`
	}
	if err := json.Unmarshal(input.Config, &config); err != nil || strings.TrimSpace(config.Server) == "" || config.Auth == nil || strings.TrimSpace(config.Auth.Type) == "" {
		return "", ErrConfigRequired
	}
	return compactJSON(input.Config), nil
}
func validateTool(input ToolInput) (string, string, error) {
	if strings.TrimSpace(input.Name) == "" {
		return "", "", ErrNameRequired
	}
	if strings.TrimSpace(input.Description) == "" {
		return "", "", ErrDescriptionRequired
	}
	if len(input.Config) == 0 || !json.Valid(input.Config) {
		return "", "", ErrConfigRequired
	}
	var config struct {
		Path          string `json:"path"`
		RequestMethod string `json:"request_method"`
	}
	if json.Unmarshal(input.Config, &config) != nil || !strings.HasPrefix(strings.TrimSpace(config.Path), "/") || !(strings.EqualFold(config.RequestMethod, "GET") || strings.EqualFold(config.RequestMethod, "POST")) {
		return "", "", ErrConfigRequired
	}
	schema := "{}"
	if len(input.APISchema) > 0 {
		if !json.Valid(input.APISchema) {
			return "", "", ErrConfigRequired
		}
		schema = compactJSON(input.APISchema)
	}
	return compactJSON(input.Config), schema, nil
}
func mapPlugin(value dao.Plugin) *Plugin {
	return &Plugin{PluginID: value.PluginID, Name: value.Name, Description: pointerString(value.Description), Config: rawJSON(value.Config), Source: value.Source, GmtCreate: value.GmtCreate, GmtModified: value.GmtModified}
}
func mapTool(value dao.Tool) *Tool {
	return &Tool{PluginID: value.PluginID, ToolID: value.ToolID, Name: value.Name, Description: pointerString(value.Description), Config: rawJSON(stringPointer(value.Config)), APISchema: rawJSON(stringPointer(value.APISchema)), Enabled: value.Enabled, TestStatus: testStatusName(value.TestStatus), Status: statusName(value.Status), GmtCreate: value.GmtCreate, GmtModified: value.GmtModified}
}
func page[T any](items []T, current, size int64) *Page[T] {
	if current < 1 {
		current = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	total := int64(len(items))
	start := (current - 1) * size
	if start >= total {
		return &Page[T]{Current: current, Size: size, Total: total, Records: []T{}}
	}
	end := start + size
	if end > total {
		end = total
	}
	return &Page[T]{Current: current, Size: size, Total: total, Records: items[start:end]}
}
func rawJSON(value *string) json.RawMessage {
	if value == nil || !json.Valid([]byte(*value)) {
		return json.RawMessage("{}")
	}
	return json.RawMessage(*value)
}
func compactJSON(value json.RawMessage) string {
	var dst bytes.Buffer
	_ = json.Compact(&dst, value)
	return dst.String()
}
func pointerString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
func stringPointer(value string) *string { return &value }
func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}
func statusName(value int16) string {
	if value == statusPublished {
		return "published"
	}
	return "draft"
}
func testStatusName(value int16) string {
	if value == testPassed {
		return "passed"
	}
	return "not_test"
}
func newID(prefix string) (string, error) {
	var data [12]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(data[:]), nil
}
