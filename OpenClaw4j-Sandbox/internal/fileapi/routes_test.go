package fileapi

import (
	"testing"

	"github.com/seaskyland/openclaw4j-sandbox/internal/pathguard"
)

func TestRoutesExposeBasicFileEndpoints(t *testing.T) {
	service := NewService(pathguard.New(t.TempDir()), 1024*1024)
	routes := Routes(service)

	seen := make(map[string]string, len(routes))
	for _, route := range routes {
		seen[route.Path] = route.Method
	}

	for _, path := range []string{
		"/v1/file/read",
		"/v1/file/write",
		"/v1/file/replace",
		"/v1/file/list",
	} {
		if seen[path] != "POST" {
			t.Fatalf("route %s = %q", path, seen[path])
		}
	}
}
