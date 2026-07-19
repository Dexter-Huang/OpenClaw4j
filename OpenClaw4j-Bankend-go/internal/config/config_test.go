package config

import (
	"errors"
	"os"
	"path/filepath"
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
	if cfg.RedisHost != DefaultRedisHost || cfg.RedisPort != DefaultRedisPort || cfg.RedisDatabase != DefaultRedisDatabase || cfg.SandboxBaseURL != DefaultSandboxBaseURL || cfg.SandboxTimeoutMs != DefaultSandboxTimeout {
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
		case EnvSandboxBaseURL:
			return "http://sandbox.local:9010/"
		case EnvSandboxTimeoutMs:
			return "1234"
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
	if cfg.SandboxBaseURL != "http://sandbox.local:9010" || cfg.SandboxTimeoutMs != 1234 {
		t.Fatalf("unexpected sandbox config: %#v", cfg)
	}
}

func TestLoadRequiresDatabaseDSN(t *testing.T) {
	_, err := LoadFromEnv(func(string) string { return "" })
	if !errors.Is(err, ErrMissingDatabaseDSN) {
		t.Fatalf("expected ErrMissingDatabaseDSN, got %v", err)
	}
}

func TestLoadFromEnvFileUsesFileValues(t *testing.T) {
	path := writeEnvFile(t, "# local development\nOPENCLAW_DATABASE_DSN='postgres://file-user:file-pass@localhost:5432/openclaw?sslmode=disable'\nOPENCLAW_REDIS_PORT=6380\n")

	cfg, err := LoadFromEnvFile(path, func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatalf("LoadFromEnvFile returned error: %v", err)
	}
	if cfg.DatabaseDSN != "postgres://file-user:file-pass@localhost:5432/openclaw?sslmode=disable" || cfg.RedisPort != 6380 {
		t.Fatalf("environment file values were not loaded: %#v", cfg)
	}
}

func TestLoadFromEnvFilePrefersProcessEnvironment(t *testing.T) {
	path := writeEnvFile(t, "OPENCLAW_DATABASE_DSN=postgres://file-user:file-pass@localhost:5432/openclaw\nOPENCLAW_REDIS_HOST=file-redis\n")

	cfg, err := LoadFromEnvFile(path, func(key string) (string, bool) {
		if key == EnvDatabaseDSN {
			return "postgres://env-user:env-pass@localhost:5432/openclaw", true
		}
		return "", false
	})
	if err != nil {
		t.Fatalf("LoadFromEnvFile returned error: %v", err)
	}
	if cfg.DatabaseDSN != "postgres://env-user:env-pass@localhost:5432/openclaw" || cfg.RedisHost != "file-redis" {
		t.Fatalf("process environment must override only its configured values: %#v", cfg)
	}
}

func TestLoadFromEnvFileRejectsMalformedLine(t *testing.T) {
	path := writeEnvFile(t, "OPENCLAW_DATABASE_DSN postgres://localhost/openclaw\n")

	_, err := LoadFromEnvFile(path, func(string) (string, bool) { return "", false })
	if err == nil {
		t.Fatal("expected malformed environment file error")
	}
}

func writeEnvFile(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write environment file: %v", err)
	}
	return path
}
