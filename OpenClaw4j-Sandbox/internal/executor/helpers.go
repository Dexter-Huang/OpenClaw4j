package executor

import (
	"path"
	"strings"
	"unicode/utf8"

	"github.com/seaskyland/openclaw4j-sandbox/internal/config"
)

func SanitizeName(value string) string {
	var builder strings.Builder
	for _, ch := range value {
		if builder.Len() >= 48 {
			break
		}
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' {
			builder.WriteRune(ch)
		} else {
			builder.WriteByte('-')
		}
	}
	return builder.String()
}

func LimitBytes(bytes []byte, maxBytes int) string {
	if len(bytes) <= maxBytes {
		return string(bytes)
	}
	truncated := bytes[:maxBytes]
	for !utf8.Valid(truncated) && len(truncated) > 0 {
		truncated = truncated[:len(truncated)-1]
	}
	return string(truncated) + "\n... truncated ..."
}

func SandboxReadPaths(cfg config.Config) []string {
	return []string{"/usr", "/lib", "/lib64", "/bin", "/etc", cfg.DepsDir}
}

func SandboxEnvVars(cfg config.Config) map[string]string {
	pythonRuntime := strings.ReplaceAll(cfg.PythonRuntime, `\`, `/`)
	pythonBin := path.Dir(pythonRuntime)
	if pythonBin == "." || pythonBin == "/" {
		pythonBin = "/usr/local/bin"
	}
	return map[string]string{
		"PATH":             pythonBin + ":/usr/local/bin:/usr/bin:/bin",
		"LANG":             "C.UTF-8",
		"PYTHONIOENCODING": "utf-8",
		"NODE_PATH":        cfg.NodePath,
	}
}
