package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

const (
	EnvHTTPAddr      = "OPENCLAW_HTTP_ADDR"
	EnvDatabaseDSN   = "OPENCLAW_DATABASE_DSN"
	EnvRedisHost     = "OPENCLAW_REDIS_HOST"
	EnvRedisPort     = "OPENCLAW_REDIS_PORT"
	EnvRedisPassword = "OPENCLAW_REDIS_PASSWORD"
	EnvRedisDatabase = "OPENCLAW_REDIS_DATABASE"

	DefaultHTTPAddr      = ":9004"
	DefaultRedisHost     = "127.0.0.1"
	DefaultRedisPort     = 6379
	DefaultRedisDatabase = 0
)

var (
	ErrMissingDatabaseDSN   = errors.New("OPENCLAW_DATABASE_DSN is required")
	ErrInvalidRedisPort     = errors.New("OPENCLAW_REDIS_PORT must be a valid integer")
	ErrInvalidRedisDatabase = errors.New("OPENCLAW_REDIS_DATABASE must be a valid integer")
)

type Config struct {
	HTTPAddr      string
	DatabaseDSN   string
	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDatabase int
}

func Load() (Config, error) {
	return LoadFromEnv(os.Getenv)
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

	redisDatabase := DefaultRedisDatabase
	if raw := strings.TrimSpace(getenv(EnvRedisDatabase)); raw != "" {
		db, err := strconv.Atoi(raw)
		if err != nil {
			return Config{}, ErrInvalidRedisDatabase
		}
		redisDatabase = db
	}

	return Config{
		HTTPAddr:      httpAddr,
		DatabaseDSN:   databaseDSN,
		RedisHost:     redisHost,
		RedisPort:     redisPort,
		RedisPassword: redisPassword,
		RedisDatabase: redisDatabase,
	}, nil
}
