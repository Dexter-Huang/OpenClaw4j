package config

import (
	"path/filepath"
	"testing"
)

func TestClampTimeout(t *testing.T) {
	c := Config{DefaultTimeoutMs: 30000, MaxTimeoutMs: 60000}
	value := uint64(90000)

	if got := c.ClampTimeout(&value); got != 60000 {
		t.Fatalf("ClampTimeout() = %d, want 60000", got)
	}
	if got := c.ClampTimeout(nil); got != 30000 {
		t.Fatalf("ClampTimeout(nil) = %d, want 30000", got)
	}
}

func TestFromEnvUsesDefaults(t *testing.T) {
	t.Setenv("OPENCLAW_SANDBOX_BIND", "")
	c := FromEnv()

	if c.Bind != "0.0.0.0:9010" {
		t.Fatalf("Bind = %q, want default", c.Bind)
	}
	if filepath.ToSlash(c.WorkDir) != "/tmp/openclaw4j-sandbox" {
		t.Fatalf("WorkDir = %q", c.WorkDir)
	}
	if c.PythonRuntime != "/opt/openclaw4j-sandbox/deps/python/bin/python" {
		t.Fatalf("PythonRuntime = %q", c.PythonRuntime)
	}
}

func TestFromEnvUsesAgentCompatibilityDefaults(t *testing.T) {
	t.Setenv("OPENCLAW_SANDBOX_WORKSPACE_DIR", "")
	t.Setenv("OPENCLAW_SANDBOX_BASH_DEFAULT_TIMEOUT_MS", "")
	t.Setenv("OPENCLAW_SANDBOX_BASH_HARD_TIMEOUT_MS", "")
	t.Setenv("OPENCLAW_SANDBOX_BASH_OUTPUT_LIMIT_BYTES", "")
	t.Setenv("OPENCLAW_SANDBOX_FILE_READ_LIMIT_BYTES", "")

	c := FromEnv()

	if filepath.ToSlash(c.WorkspaceDir) != "/tmp/openclaw4j-workspace" {
		t.Fatalf("WorkspaceDir = %q", c.WorkspaceDir)
	}
	if c.BashDefaultTimeoutMs != 30000 {
		t.Fatalf("BashDefaultTimeoutMs = %d", c.BashDefaultTimeoutMs)
	}
	if c.BashHardTimeoutMs != 120000 {
		t.Fatalf("BashHardTimeoutMs = %d", c.BashHardTimeoutMs)
	}
	if c.BashOutputLimitBytes != 65536 {
		t.Fatalf("BashOutputLimitBytes = %d", c.BashOutputLimitBytes)
	}
	if c.FileReadLimitBytes != 1048576 {
		t.Fatalf("FileReadLimitBytes = %d", c.FileReadLimitBytes)
	}
}
