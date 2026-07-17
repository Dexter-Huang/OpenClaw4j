package bashapi

import (
	"testing"

	"github.com/seaskyland/openclaw4j-sandbox/internal/pathguard"
)

func TestRoutesExposeBashEndpoints(t *testing.T) {
	service := NewService(pathguard.New(t.TempDir()), 3000, 5000, 64*1024)
	routes := Routes(service)

	seen := make(map[string]string, len(routes))
	for _, route := range routes {
		seen[route.Path] = route.Method
	}

	want := map[string]string{
		"/v1/bash/exec":                       "POST",
		"/v1/bash/output":                     "POST",
		"/v1/bash/write":                      "POST",
		"/v1/bash/kill":                       "POST",
		"/v1/bash/sessions":                   "GET",
		"/v1/bash/sessions/create":            "POST",
		"/v1/bash/sessions/:session_id/close": "POST",
	}
	for path, method := range want {
		if seen[path] != method {
			t.Fatalf("route %s = %q, want %s", path, seen[path], method)
		}
	}
}
