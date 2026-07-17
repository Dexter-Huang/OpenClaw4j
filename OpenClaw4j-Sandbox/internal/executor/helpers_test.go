package executor

import (
	"reflect"
	"strings"
	"testing"

	"github.com/seaskyland/openclaw4j-sandbox/internal/config"
)

func testConfig() config.Config {
	return config.Config{
		WorkDir:          "/tmp/openclaw4j-sandbox",
		DefaultTimeoutMs: 30000,
		MaxTimeoutMs:     60000,
		MemoryLimit:      "512M",
		ProcessLimit:     16,
		StdoutLimitBytes: 1024,
		StderrLimitBytes: 1024,
		DepsDir:          "/opt/openclaw4j-sandbox/deps",
		PythonRuntime:    "/opt/openclaw4j-sandbox/deps/python/bin/python",
		NodePath:         "/opt/openclaw4j-sandbox/deps/node/node_modules",
	}
}

func TestSanitizeName(t *testing.T) {
	if got := SanitizeName("abc/../中文"); got != "abc------" {
		t.Fatalf("SanitizeName() = %q", got)
	}
}

func TestLimitBytes(t *testing.T) {
	if got := LimitBytes([]byte("abcdef"), 3); got != "abc\n... truncated ..." {
		t.Fatalf("LimitBytes() = %q", got)
	}
}

func TestSandboxReadPathsIncludesDeps(t *testing.T) {
	paths := SandboxReadPaths(testConfig())
	want := []string{"/usr", "/lib", "/lib64", "/bin", "/etc", "/opt/openclaw4j-sandbox/deps"}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("SandboxReadPaths() = %#v", paths)
	}
}

func TestSandboxEnvVars(t *testing.T) {
	env := SandboxEnvVars(testConfig())
	if env["NODE_PATH"] != "/opt/openclaw4j-sandbox/deps/node/node_modules" {
		t.Fatalf("NODE_PATH = %q", env["NODE_PATH"])
	}
	if env["PYTHONIOENCODING"] != "utf-8" {
		t.Fatalf("PYTHONIOENCODING = %q", env["PYTHONIOENCODING"])
	}
	if !strings.HasPrefix(env["PATH"], "/opt/openclaw4j-sandbox/deps/python/bin:") {
		t.Fatalf("PATH = %q", env["PATH"])
	}
}
