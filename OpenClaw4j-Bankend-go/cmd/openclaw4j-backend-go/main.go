package main

import (
	"context"
	"fmt"
	"log"
	"time"

	hertz "github.com/cloudwego/hertz/pkg/app/server"
	hertzconfig "github.com/cloudwego/hertz/pkg/common/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/seaskyland/openclaw4j-backend-go/internal/agent"
	"github.com/seaskyland/openclaw4j-backend-go/internal/bootstrap"
	appconfig "github.com/seaskyland/openclaw4j-backend-go/internal/config"
	"github.com/seaskyland/openclaw4j-backend-go/internal/dao/redisdao"
	"github.com/seaskyland/openclaw4j-backend-go/internal/legacycrypto"
	"github.com/seaskyland/openclaw4j-backend-go/internal/oauth2"
	"github.com/seaskyland/openclaw4j-backend-go/internal/scriptsandbox"
	"github.com/seaskyland/openclaw4j-backend-go/internal/workflow"
)

func main() {
	cfg, err := appconfig.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer pool.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDatabase,
	})
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("connect redis: %v", err)
	}
	defer redisClient.Close()

	legacyAPIKeyEncryptor, err := legacycrypto.NewJavaAPIKeyEncryptor()
	if err != nil {
		log.Fatalf("initialize legacy API key encryptor: %v", err)
	}

	app, err := bootstrap.New(bootstrap.Options{
		DB:                     pool,
		TokenSessions:          redisdao.NewTokenSessionDAO(redisClient),
		AgentMemory:            agent.NewRedisStore(redisClient, 0),
		WorkflowState:          workflow.NewRedisStateStore(redisClient, 0),
		LegacyAPIKeyEncryptor:  legacyAPIKeyEncryptor,
		HertzOptions:           []hertzconfig.Option{hertz.WithHostPorts(cfg.HTTPAddr)},
		FileStorageDir:         cfg.FileStorageDir,
		ProviderPrivateKeyFile: cfg.ProviderPrivateKeyFile,
		ScriptExecutor:         scriptsandbox.NewClient(cfg.SandboxBaseURL, time.Duration(cfg.SandboxTimeoutMs)*time.Millisecond),
		GitHubOAuth2Provider: oauth2.NewGitHubService(oauth2.GitHubConfig{
			ClientID:     cfg.GitHubClientID,
			ClientSecret: cfg.GitHubClientSecret,
			RedirectURI:  cfg.GitHubRedirectURI,
			AuthorizeURL: cfg.GitHubAuthorizeURL,
			TokenURL:     cfg.GitHubTokenURL,
			UserInfoURL:  cfg.GitHubUserInfoURL,
		}, nil),
	})
	if err != nil {
		log.Fatalf("bootstrap app: %v", err)
	}

	app.HTTP.Spin()
}
