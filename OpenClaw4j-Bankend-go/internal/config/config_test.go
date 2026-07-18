package config

import (
	"errors"
	"testing"
)

func TestLoadUsesDefaultHTTPAddr(t *testing.T) {
	cfg, err := LoadFromEnv(func(key string) string {
		if key == EnvDatabaseDSN {
			return "postgres://user:pass@localhost:5432/openclaw"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("LoadFromEnv returned error: %v", err)
	}

	if cfg.HTTPAddr != ":9004" {
		t.Fatalf("expected default addr %q, got %q", ":9004", cfg.HTTPAddr)
	}
	if cfg.DatabaseDSN == "" {
		t.Fatalf("database DSN was not loaded")
	}
	if cfg.RedisHost != DefaultRedisHost || cfg.RedisPort != DefaultRedisPort || cfg.RedisDatabase != DefaultRedisDatabase {
		t.Fatalf("unexpected redis defaults: %#v", cfg)
	}
}

func TestLoadUsesCustomHTTPAddr(t *testing.T) {
	cfg, err := LoadFromEnv(func(key string) string {
		switch key {
		case EnvHTTPAddr:
			return ":9090"
		case EnvDatabaseDSN:
			return "postgres://user:pass@localhost:5432/openclaw"
		case EnvRedisHost:
			return "redis.local"
		case EnvRedisPort:
			return "6380"
		case EnvRedisPassword:
			return "secret"
		case EnvRedisDatabase:
			return "2"
		default:
			return ""
		}
	})
	if err != nil {
		t.Fatalf("LoadFromEnv returned error: %v", err)
	}

	if cfg.HTTPAddr != ":9090" {
		t.Fatalf("expected custom addr, got %q", cfg.HTTPAddr)
	}
	if cfg.RedisHost != "redis.local" || cfg.RedisPort != 6380 || cfg.RedisPassword != "secret" || cfg.RedisDatabase != 2 {
		t.Fatalf("unexpected redis config: %#v", cfg)
	}
}

func TestLoadRequiresDatabaseDSN(t *testing.T) {
	_, err := LoadFromEnv(func(string) string { return "" })
	if !errors.Is(err, ErrMissingDatabaseDSN) {
		t.Fatalf("expected ErrMissingDatabaseDSN, got %v", err)
	}
}
