package pathguard

import (
	"path/filepath"
	"testing"
)

func TestResolveAllowsPathsInsideWorkspace(t *testing.T) {
	root := t.TempDir()
	guard := New(root)

	relative, err := guard.Resolve(filepath.Join("reports", "out.txt"))
	if err != nil {
		t.Fatalf("relative path should resolve: %v", err)
	}
	if relative != filepath.Join(root, "reports", "out.txt") {
		t.Fatalf("unexpected relative resolution: %s", relative)
	}

	inside := filepath.Join(root, "reports", "out.txt")
	absolute, err := guard.Resolve(inside)
	if err != nil {
		t.Fatalf("absolute path inside workspace should resolve: %v", err)
	}
	if absolute != inside {
		t.Fatalf("unexpected absolute resolution: %s", absolute)
	}
}

func TestResolveRejectsWorkspaceEscape(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret.txt")
	guard := New(root)

	for _, input := range []string{filepath.Join("..", "secret.txt"), outside} {
		_, err := guard.Resolve(input)
		if err == nil {
			t.Fatalf("expected %s to be rejected", input)
		}
		guardErr, ok := err.(Error)
		if !ok {
			t.Fatalf("expected pathguard.Error, got %T", err)
		}
		if guardErr.ErrorType != "invalid_path" {
			t.Fatalf("unexpected error type: %s", guardErr.ErrorType)
		}
	}
}
