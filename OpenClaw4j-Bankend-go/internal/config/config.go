package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"
)

const (
	EnvHTTPAddr               = "OPENCLAW_HTTP_ADDR"
	EnvDatabaseDSN            = "OPENCLAW_DATABASE_DSN"
	EnvRedisHost              = "OPENCLAW_REDIS_HOST"
	EnvRedisPort              = "OPENCLAW_REDIS_PORT"
	EnvRedisPassword          = "OPENCLAW_REDIS_PASSWORD"
	EnvRedisDatabase          = "OPENCLAW_REDIS_DATABASE"
	EnvFileStorageDir         = "OPENCLAW_FILE_STORAGE_DIR"
	EnvFrontendDistDir        = "OPENCLAW_FRONTEND_DIST_DIR"
	EnvProviderPrivateKeyFile = "OPENCLAW_PROVIDER_PRIVATE_KEY_FILE"
	EnvGitHubClientID         = "OPENCLAW_GITHUB_CLIENT_ID"
	EnvGitHubClientSecret     = "OPENCLAW_GITHUB_CLIENT_SECRET"
	EnvGitHubRedirectURI      = "OPENCLAW_GITHUB_REDIRECT_URI"
	EnvGitHubAuthorizeURL     = "OPENCLAW_GITHUB_AUTHORIZE_URL"
	EnvGitHubTokenURL         = "OPENCLAW_GITHUB_TOKEN_URL"
	EnvGitHubUserInfoURL      = "OPENCLAW_GITHUB_USER_INFO_URL"
	EnvSandboxBaseURL         = "OPENCLAW_SANDBOX_BASE_URL"
	EnvSandboxTimeoutMs       = "OPENCLAW_SANDBOX_TIMEOUT_MS"

	DefaultHTTPAddr        = ":9004"
	DefaultRedisHost       = "127.0.0.1"
	DefaultRedisPort       = 6379
	DefaultRedisDatabase   = 0
	DefaultFileStorageDir  = "data/files"
	DefaultFrontendDistDir = "dist"
	DefaultSandboxBaseURL  = "http://127.0.0.1:9010"
	DefaultSandboxTimeout  = 30000
	DefaultEnvFile         = ".env"
)

var (
	ErrMissingDatabaseDSN    = errors.New("OPENCLAW_DATABASE_DSN is required")
	ErrInvalidRedisPort      = errors.New("OPENCLAW_REDIS_PORT must be a valid integer")
	ErrInvalidRedisDatabase  = errors.New("OPENCLAW_REDIS_DATABASE must be a valid integer")
	ErrInvalidSandboxTimeout = errors.New("OPENCLAW_SANDBOX_TIMEOUT_MS must be a positive integer")
)

type Config struct {
	HTTPAddr               string
	DatabaseDSN            string
	RedisHost              string
	RedisPort              int
	RedisPassword          string
	RedisDatabase          int
	FileStorageDir         string
	FrontendDistDir        string
	ProviderPrivateKeyFile string
	GitHubClientID         string
	GitHubClientSecret     string
	GitHubRedirectURI      string
	GitHubAuthorizeURL     string
	GitHubTokenURL         string
	GitHubUserInfoURL      string
	SandboxBaseURL         string
	SandboxTimeoutMs       int
}

func Load() (Config, error) {
	return LoadFromEnvFile(DefaultEnvFile, os.LookupEnv)
}

func LoadFromEnv(getenv func(string) string) (Config, error) {
	httpAddr := strings.TrimSpace(getenv(EnvHTTPAddr))
	if httpAddr == "" {
		httpAddr = DefaultHTTPAddr
	}

	databaseDSN := strings.TrimSpace(getenv(EnvDatabaseDSN))
	if databaseDSN == "" {
		return Config{}, ErrMissingDatabaseDSN
	}

	redisHost := strings.TrimSpace(getenv(EnvRedisHost))
	if redisHost == "" {
		redisHost = DefaultRedisHost
	}

	redisPort := DefaultRedisPort
	if raw := strings.TrimSpace(getenv(EnvRedisPort)); raw != "" {
		port, err := strconv.Atoi(raw)
		if err != nil {
			return Config{}, ErrInvalidRedisPort
		}
		redisPort = port
	}

	redisPassword := strings.TrimSpace(getenv(EnvRedisPassword))
	fileStorageDir := strings.TrimSpace(getenv(EnvFileStorageDir))
	if fileStorageDir == "" {
		fileStorageDir = DefaultFileStorageDir
	}
	frontendDistDir := strings.TrimSpace(getenv(EnvFrontendDistDir))
	if frontendDistDir == "" {
		frontendDistDir = DefaultFrontendDistDir
	}
	providerPrivateKeyFile := strings.TrimSpace(getenv(EnvProviderPrivateKeyFile))

	redisDatabase := DefaultRedisDatabase
	if raw := strings.TrimSpace(getenv(EnvRedisDatabase)); raw != "" {
		db, err := strconv.Atoi(raw)
		if err != nil {
			return Config{}, ErrInvalidRedisDatabase
		}
		redisDatabase = db
	}

	sandboxTimeout := DefaultSandboxTimeout
	if raw := strings.TrimSpace(getenv(EnvSandboxTimeoutMs)); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			return Config{}, ErrInvalidSandboxTimeout
		}
		sandboxTimeout = value
	}
	sandboxBaseURL := strings.TrimRight(strings.TrimSpace(getenv(EnvSandboxBaseURL)), "/")
	if sandboxBaseURL == "" {
		sandboxBaseURL = DefaultSandboxBaseURL
	}

	return Config{
		HTTPAddr:               httpAddr,
		DatabaseDSN:            databaseDSN,
		RedisHost:              redisHost,
		RedisPort:              redisPort,
		RedisPassword:          redisPassword,
		RedisDatabase:          redisDatabase,
		FileStorageDir:         fileStorageDir,
		FrontendDistDir:        frontendDistDir,
		ProviderPrivateKeyFile: providerPrivateKeyFile,
		GitHubClientID:         strings.TrimSpace(getenv(EnvGitHubClientID)),
		GitHubClientSecret:     strings.TrimSpace(getenv(EnvGitHubClientSecret)),
		GitHubRedirectURI:      strings.TrimSpace(getenv(EnvGitHubRedirectURI)),
		GitHubAuthorizeURL:     strings.TrimSpace(getenv(EnvGitHubAuthorizeURL)),
		GitHubTokenURL:         strings.TrimSpace(getenv(EnvGitHubTokenURL)),
		GitHubUserInfoURL:      strings.TrimSpace(getenv(EnvGitHubUserInfoURL)),
		SandboxBaseURL:         sandboxBaseURL,
		SandboxTimeoutMs:       sandboxTimeout,
	}, nil
}

// LoadFromEnvFile 读取本地 .env 作为开发默认值。进程环境变量具有更高优先级，
// 因此容器、CI 和生产部署无需改变既有的环境变量配置方式。
func LoadFromEnvFile(path string, lookupEnv func(string) (string, bool)) (Config, error) {
	values, err := readEnvFile(path)
	if err != nil {
		return Config{}, err
	}

	return LoadFromEnv(func(key string) string {
		if value, found := lookupEnv(key); found {
			return value
		}
		return values[key]
	})
}

// readEnvFile 支持开发环境常用的 KEY=VALUE 格式，以及空行、注释和 export 前缀。
// 文件缺失不视为错误，保持部署环境仅通过系统环境变量启动的兼容性。
func readEnvFile(path string) (map[string]string, error) {
	contents, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read environment file %q: %w", path, err)
	}

	values := make(map[string]string)
	for index, rawLine := range strings.Split(string(contents), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		key, value, found := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !found || key == "" {
			return nil, fmt.Errorf("parse environment file %q: line %d must use KEY=VALUE", path, index+1)
		}

		value = strings.TrimSpace(value)
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}
		values[key] = value
	}

	return values, nil
}
