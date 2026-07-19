package rag

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

	"github.com/seaskyland/openclaw4j-backend-go/internal/dao"
)

type Embedder interface {
	Embed(context.Context, string, string, string, []string) ([][]float32, error)
}

type OpenAIEmbedder struct {
	providers      dao.ProviderDAO
	models         dao.ModelDAO
	privateKeyFile string
	client         *http.Client
}

func NewOpenAIEmbedder(providers dao.ProviderDAO, models dao.ModelDAO, privateKeyFile string, client *http.Client) *OpenAIEmbedder {
	if client == nil {
		client = http.DefaultClient
	}
	return &OpenAIEmbedder{providers: providers, models: models, privateKeyFile: strings.TrimSpace(privateKeyFile), client: client}
}
func (s *OpenAIEmbedder) Embed(ctx context.Context, workspaceID, providerCode, modelID string, input []string) ([][]float32, error) {
	if len(input) == 0 {
		return [][]float32{}, nil
	}
	if s == nil || s.providers == nil || s.models == nil {
		return nil, errors.New("embedding provider is unavailable")
	}
	if _, err := s.models.FindByProviderAndIDAndWorkspace(ctx, providerCode, modelID, workspaceID); err != nil {
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
	endpoint, err := embeddingsEndpoint(credential["endpoint"], credential["embeddings_path"])
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(map[string]any{"model": modelID, "input": input, "encoding_format": "float"})
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
	payload, err := io.ReadAll(io.LimitReader(response.Body, 16<<20))
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("embedding request failed with status %d", response.StatusCode)
	}
	var decoded struct {
		Data []struct {
			Index     int       `json:"index"`
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, fmt.Errorf("decode embedding response: %w", err)
	}
	if len(decoded.Data) != len(input) {
		return nil, fmt.Errorf("embedding response count %d does not match input %d", len(decoded.Data), len(input))
	}
	result := make([][]float32, len(input))
	for _, item := range decoded.Data {
		if item.Index < 0 || item.Index >= len(result) || len(item.Embedding) == 0 {
			return nil, errors.New("embedding response contains an invalid item")
		}
		result[item.Index] = item.Embedding
	}
	return result, nil
}
func (s *OpenAIEmbedder) credential(raw *string) (map[string]string, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, errors.New("provider credential is required")
	}
	var value map[string]string
	if err := json.Unmarshal([]byte(*raw), &value); err != nil {
		return nil, fmt.Errorf("decode provider credential: %w", err)
	}
	if strings.TrimSpace(value["api_key"]) == "" {
		return nil, errors.New("provider api_key is required")
	}
	plain, err := s.decrypt(value["api_key"])
	if err != nil {
		return nil, err
	}
	value["api_key"] = plain
	return value, nil
}
func (s *OpenAIEmbedder) decrypt(value string) (string, error) {
	if s.privateKeyFile == "" {
		return "", errors.New("provider private key file is required")
	}
	contents, err := os.ReadFile(s.privateKeyFile)
	if err != nil {
		return "", err
	}
	block, _ := pem.Decode(contents)
	if block == nil {
		return "", errors.New("decode provider private key")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}
	privateKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return "", errors.New("provider private key is not RSA")
	}
	ciphertext, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	plain, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
func embeddingsEndpoint(raw, configuredPath string) (string, error) {
	base, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" {
		return "", errors.New("provider endpoint must be an http(s) URL")
	}
	path := strings.TrimSpace(configuredPath)
	if path == "" {
		path = "/v1/embeddings"
	}
	relative, err := url.Parse(path)
	if err != nil || relative.IsAbs() {
		return "", errors.New("provider embeddings_path must be a relative path")
	}
	basePath := strings.TrimRight(base.Path, "/")
	parts := strings.Split(strings.Trim(relative.Path, "/"), "/")
	baseParts := strings.Split(strings.Trim(basePath, "/"), "/")
	if len(parts) > 0 && len(baseParts) > 0 && parts[0] == baseParts[len(baseParts)-1] {
		parts = parts[1:]
	}
	base.Path = strings.TrimRight(basePath, "/") + "/" + strings.Join(parts, "/")
	base.RawPath = ""
	base.RawQuery = relative.RawQuery
	return base.String(), nil
}

var _ Embedder = (*OpenAIEmbedder)(nil)
