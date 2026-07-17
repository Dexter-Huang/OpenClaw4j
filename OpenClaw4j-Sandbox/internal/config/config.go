package config

import (
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	Bind                 string
	WorkDir              string
	WorkspaceDir         string
	DefaultTimeoutMs     uint64
	MaxTimeoutMs         uint64
	MemoryLimit          string
	ProcessLimit         uint32
	StdoutLimitBytes     int
	StderrLimitBytes     int
	DepsDir              string
	PythonRuntime        string
	NodePath             string
	BashDefaultTimeoutMs uint64
	BashHardTimeoutMs    uint64
	BashOutputLimitBytes int
	FileReadLimitBytes   int
}

func FromEnv() Config {
	return Config{
		Bind:                 envValue("OPENCLAW_SANDBOX_BIND", "0.0.0.0:9010"),
		WorkDir:              filepath.Clean(envValue("OPENCLAW_SANDBOX_WORK_DIR", "/tmp/openclaw4j-sandbox")),
		WorkspaceDir:         filepath.Clean(envValue("OPENCLAW_SANDBOX_WORKSPACE_DIR", "/tmp/openclaw4j-workspace")),
		DefaultTimeoutMs:     envUint64("OPENCLAW_SANDBOX_DEFAULT_TIMEOUT_MS", 30000),
		MaxTimeoutMs:         envUint64("OPENCLAW_SANDBOX_MAX_TIMEOUT_MS", 120000),
		MemoryLimit:          envValue("OPENCLAW_SANDBOX_MEMORY_LIMIT", "256M"),
		ProcessLimit:         envUint32("OPENCLAW_SANDBOX_PROCESS_LIMIT", 16),
		StdoutLimitBytes:     envInt("OPENCLAW_SANDBOX_STDOUT_LIMIT_BYTES", 65536),
		StderrLimitBytes:     envInt("OPENCLAW_SANDBOX_STDERR_LIMIT_BYTES", 65536),
		DepsDir:              envValue("OPENCLAW_SANDBOX_DEPS_DIR", "/opt/openclaw4j-sandbox/deps"),
		PythonRuntime:        envValue("OPENCLAW_SANDBOX_PYTHON_RUNTIME", "/opt/openclaw4j-sandbox/deps/python/bin/python"),
		NodePath:             envValue("OPENCLAW_SANDBOX_NODE_PATH", "/opt/openclaw4j-sandbox/deps/node/node_modules"),
		BashDefaultTimeoutMs: envUint64("OPENCLAW_SANDBOX_BASH_DEFAULT_TIMEOUT_MS", 30000),
		BashHardTimeoutMs:    envUint64("OPENCLAW_SANDBOX_BASH_HARD_TIMEOUT_MS", 120000),
		BashOutputLimitBytes: envInt("OPENCLAW_SANDBOX_BASH_OUTPUT_LIMIT_BYTES", 65536),
		FileReadLimitBytes:   envInt("OPENCLAW_SANDBOX_FILE_READ_LIMIT_BYTES", 1048576),
	}
}

func (c Config) ClampTimeout(timeoutMs *uint64) uint64 {
	if timeoutMs == nil {
		return c.DefaultTimeoutMs
	}
	if *timeoutMs > c.MaxTimeoutMs {
		return c.MaxTimeoutMs
	}
	return *timeoutMs
}

func envValue(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envUint64(key string, fallback uint64) uint64 {
	value, err := strconv.ParseUint(os.Getenv(key), 10, 64)
	if err != nil {
		return fallback
	}
	return value
}

func envUint32(key string, fallback uint32) uint32 {
	value, err := strconv.ParseUint(os.Getenv(key), 10, 32)
	if err != nil {
		return fallback
	}
	return uint32(value)
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return value
}
