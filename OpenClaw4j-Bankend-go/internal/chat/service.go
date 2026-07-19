package chat

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/seaskyland/openclaw4j-backend-go/internal/application"
	"github.com/seaskyland/openclaw4j-backend-go/internal/dao"
)

var (
	ErrAppIDRequired        = errors.New("app_id is required")
	ErrMessagesRequired     = errors.New("messages are required")
	ErrPublishedAppRequired = errors.New("published application configuration is required")
	ErrModelProviderMissing = errors.New("application model_provider is required")
	ErrModelMissing         = errors.New("application model is required")
	ErrCredentialMissing    = errors.New("provider api_key is required")
	ErrPrivateKeyMissing    = errors.New("provider private key file is required")
)

type ApplicationReader interface {
	Get(context.Context, string, string) (*application.Application, error)
}

type Message struct {
	Role             string     `json:"role"`
	Content          any        `json:"content"`
	ContentType      string     `json:"content_type,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
}

// ToolCall 保持 OpenAI-compatible chat/completions 的函数调用结构。Agent 层只依赖
// 此传输对象，具体 Plugin/MCP 的发现和执行仍在各自服务中完成。
type ToolCall struct {
	// Index 只会由流式响应携带，用于将同一工具调用的多个 delta 精确拼回。
	// 非流式 assistant 消息不需要该字段。
	Index    *int         `json:"index,omitempty"`
	ID       string       `json:"id,omitempty"`
	Type     string       `json:"type,omitempty"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Request struct {
	AppID          string         `json:"app_id"`
	Messages       []Message      `json:"messages"`
	ConversationID string         `json:"conversation_id"`
	Stream         bool           `json:"stream"`
	IsDraft        bool           `json:"is_draft"`
	Parameter      map[string]any `json:"parameter"`
}

type Response struct {
	RequestID      string         `json:"request_id"`
	ConversationID string         `json:"conversation_id"`
	Message        Message        `json:"message"`
	Usage          map[string]any `json:"usage,omitempty"`
}

type Service struct {
	apps           ApplicationReader
	providers      dao.ProviderDAO
	models         dao.ModelDAO
	privateKeyFile string
	client         *http.Client
	now            func() time.Time
}

func NewService(apps ApplicationReader, providers dao.ProviderDAO, models dao.ModelDAO, privateKeyFile string, clock func() time.Time) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{apps: apps, providers: providers, models: models, privateKeyFile: strings.TrimSpace(privateKeyFile), client: &http.Client{Timeout: 90 * time.Second}, now: clock}
}

func (s *Service) Complete(ctx context.Context, workspaceID string, input Request) (*Response, error) {
	if strings.TrimSpace(input.AppID) == "" {
		return nil, ErrAppIDRequired
	}
	if len(input.Messages) == 0 {
		return nil, ErrMessagesRequired
	}
	app, err := s.apps.Get(ctx, workspaceID, input.AppID)
	if err != nil {
		return nil, err
	}
	config := app.Config
	if !input.IsDraft {
		if len(app.PubConfig) == 0 {
			return nil, ErrPublishedAppRequired
		}
		config = app.PubConfig
	}
	providerCode := stringValue(config["model_provider"])
	if providerCode == "" {
		return nil, ErrModelProviderMissing
	}
	modelID := modelID(config["model"])
	if modelID == "" {
		return nil, ErrModelMissing
	}
	return s.CompleteWithModel(ctx, workspaceID, providerCode, modelID, input.Messages, mapValue(config["parameter"]), input.Parameter, input.ConversationID)
}

// CompleteWithModel 供工作流等需要按节点模型配置调用的内部服务使用。
// 它仍复用同一套模型登记校验、凭据解密和 OpenAI 兼容协议，避免工作流绕开 provider 配置。
func (s *Service) CompleteWithModel(ctx context.Context, workspaceID, providerCode, modelID string, messages []Message, modelParameters, requestParameters map[string]any, conversationID string) (*Response, error) {
	if strings.TrimSpace(providerCode) == "" {
		return nil, ErrModelProviderMissing
	}
	if strings.TrimSpace(modelID) == "" {
		return nil, ErrModelMissing
	}
	if len(messages) == 0 {
		return nil, ErrMessagesRequired
	}
	if _, err := s.models.FindByProviderAndIDAndWorkspace(ctx, providerCode, modelID, workspaceID); err != nil {
		if errors.Is(err, dao.ErrNotFound) {
			return nil, ErrModelMissing
		}
		return nil, err
	}
	provider, err := s.providers.FindByCodeAndWorkspace(ctx, providerCode, workspaceID)
	if err != nil {
		return nil, err
	}
	credential, err := s.credential(provider.Credential)
	if err != nil {
		return nil, err
	}
	endpoint, err := completionEndpoint(stringValue(credential["endpoint"]), stringValue(credential["completions_path"]))
	if err != nil {
		return nil, err
	}
	requestBody := map[string]any{"model": modelID, "messages": messages, "stream": false}
	for key, value := range modelParameters {
		requestBody[key] = value
	}
	for key, value := range requestParameters {
		requestBody[key] = value
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+credential["api_key"])
	response, err := s.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("model completion request failed with status %d", response.StatusCode)
	}
	return parseCompletion(responseBody, conversationID)
}

func (s *Service) credential(raw *string) (map[string]string, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, ErrCredentialMissing
	}
	var credential map[string]string
	if err := json.Unmarshal([]byte(*raw), &credential); err != nil {
		return nil, fmt.Errorf("invalid provider credential: %w", err)
	}
	apiKey := strings.TrimSpace(credential["api_key"])
	if apiKey == "" {
		return nil, ErrCredentialMissing
	}
	plain, err := s.decryptAPIKey(apiKey)
	if err != nil {
		return nil, err
	}
	credential["api_key"] = plain
	return credential, nil
}

func (s *Service) decryptAPIKey(value string) (string, error) {
	if s.privateKeyFile == "" {
		return "", ErrPrivateKeyMissing
	}
	contents, err := os.ReadFile(s.privateKeyFile)
	if err != nil {
		return "", fmt.Errorf("read provider private key: %w", err)
	}
	block, _ := pem.Decode(contents)
	if block == nil {
		return "", errors.New("decode provider private key")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parse provider private key: %w", err)
	}
	privateKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return "", errors.New("provider private key is not RSA")
	}
	ciphertext, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", fmt.Errorf("decode provider api key: %w", err)
	}
	plain, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt provider api key: %w", err)
	}
	return string(plain), nil
}

func completionEndpoint(endpoint, completionPath string) (string, error) {
	base, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" {
		return "", errors.New("provider endpoint must be an http(s) URL")
	}
	if strings.TrimSpace(completionPath) == "" {
		completionPath = "/v1/chat/completions"
	}
	relative, err := url.Parse(completionPath)
	if err != nil || relative.IsAbs() {
		return "", errors.New("provider completions_path must be a relative path")
	}

	// Java OpenAiChatOptions treats credential.endpoint as a baseUrl and appends the
	// completion route beneath it. url.ResolveReference would interpret a leading slash
	// as host-relative and discard a provider prefix such as /compatible-mode/v1.
	// Keep the base path and only remove a duplicated final /v1 segment.
	basePath := strings.TrimRight(base.Path, "/")
	completionSegments := strings.Split(strings.Trim(relative.Path, "/"), "/")
	if len(completionSegments) > 0 && completionSegments[0] != "" {
		baseSegments := strings.Split(strings.Trim(basePath, "/"), "/")
		if len(baseSegments) > 0 && baseSegments[len(baseSegments)-1] == completionSegments[0] {
			completionSegments = completionSegments[1:]
		}
	}
	completion := strings.Join(completionSegments, "/")
	if completion != "" {
		base.Path = strings.TrimRight(basePath, "/") + "/" + completion
	} else {
		base.Path = basePath
	}
	base.RawPath = ""
	base.RawQuery = relative.RawQuery
	base.Fragment = ""
	return base.String(), nil
}

func parseCompletion(body []byte, conversationID string) (*Response, error) {
	var upstream struct {
		ID      string `json:"id"`
		Choices []struct {
			Message struct {
				Role             string     `json:"role"`
				Content          any        `json:"content"`
				ReasoningContent string     `json:"reasoning_content"`
				ToolCalls        []ToolCall `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage map[string]any `json:"usage"`
	}
	if err := json.Unmarshal(body, &upstream); err != nil {
		return nil, fmt.Errorf("invalid model completion response: %w", err)
	}
	if len(upstream.Choices) == 0 {
		return nil, errors.New("model completion response has no choices")
	}
	message := upstream.Choices[0].Message
	if message.Role == "" {
		message.Role = "assistant"
	}
	requestID := upstream.ID
	if requestID == "" {
		requestID = fmt.Sprintf("chat-%d", time.Now().UnixNano())
	}
	if conversationID == "" {
		conversationID = requestID
	}
	return &Response{RequestID: requestID, ConversationID: conversationID, Message: Message{Role: message.Role, Content: message.Content, ReasoningContent: message.ReasoningContent, ToolCalls: message.ToolCalls}, Usage: upstream.Usage}, nil
}

func modelID(value any) string {
	if item := stringValue(value); item != "" {
		return item
	}
	item := mapValue(value)
	for _, key := range []string{"model_id", "id", "name"} {
		if result := stringValue(item[key]); result != "" {
			return result
		}
	}
	return ""
}

func mapValue(value any) map[string]any {
	result, _ := value.(map[string]any)
	return result
}

func stringValue(value any) string {
	result, _ := value.(string)
	return strings.TrimSpace(result)
}
