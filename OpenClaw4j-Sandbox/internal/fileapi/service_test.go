package fileapi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/seaskyland/openclaw4j-sandbox/internal/pathguard"
)

func newTestService(t *testing.T) (string, Service) {
	t.Helper()

	root := t.TempDir()
	service := NewService(pathguard.New(root), 1024*1024)
	return root, service
}

func TestServiceWritesAndReadsUTF8File(t *testing.T) {
	_, service := newTestService(t)

	write := service.Write(WriteRequest{
		File:            "notes/todo.txt",
		Content:         "hello",
		TrailingNewline: boolPtr(true),
	})
	if !write.Success {
		t.Fatalf("write failed: %#v", write)
	}
	if write.Data["bytes_written"] != 6 {
		t.Fatalf("bytes_written = %#v", write.Data["bytes_written"])
	}
	if write.Data["created"] != true {
		t.Fatalf("created = %#v", write.Data["created"])
	}

	read := service.Read(ReadRequest{File: "notes/todo.txt"})
	if !read.Success {
		t.Fatalf("read failed: %#v", read)
	}
	if read.Data["content"] != "hello\n" {
		t.Fatalf("content = %#v", read.Data["content"])
	}
	if read.Data["line_count"] != 1 {
		t.Fatalf("line_count = %#v", read.Data["line_count"])
	}
}

func TestServiceRejectsWorkspaceEscape(t *testing.T) {
	_, service := newTestService(t)

	read := service.Read(ReadRequest{File: filepath.Join("..", "secret.txt")})
	if read.Success {
		t.Fatalf("expected workspace escape to fail: %#v", read)
	}
	if read.Data["error_type"] != "invalid_path" {
		t.Fatalf("error_type = %#v", read.Data["error_type"])
	}
}

func TestServiceReplacesTextInFile(t *testing.T) {
	root, service := newTestService(t)
	target := filepath.Join(root, "app.txt")
	if err := os.WriteFile(target, []byte("hello world world"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := service.Replace(ReplaceRequest{
		File:       "app.txt",
		OldStr:     "world",
		NewStr:     "sandbox",
		ReplaceAll: boolPtr(true),
	})
	if !result.Success {
		t.Fatalf("replace failed: %#v", result)
	}

	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "hello sandbox sandbox" {
		t.Fatalf("content = %q", content)
	}
	if result.Data["replacements"] != 2 {
		t.Fatalf("replacements = %#v", result.Data["replacements"])
	}
}

func TestServiceListsDirectory(t *testing.T) {
	root, service := newTestService(t)
	if err := os.MkdirAll(filepath.Join(root, "docs", "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", ".hidden"), []byte("hidden"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := service.List(ListRequest{
		Path:        "docs",
		IncludeSize: boolPtr(true),
	})
	if !result.Success {
		t.Fatalf("list failed: %#v", result)
	}

	entries, ok := result.Data["entries"].([]Entry)
	if !ok {
		t.Fatalf("entries type = %T", result.Data["entries"])
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %#v", entries)
	}
	if entries[0].Name != "a.txt" || entries[0].Size == nil || *entries[0].Size != 1 {
		t.Fatalf("first entry = %#v", entries[0])
	}
	if entries[1].Name != "nested" || !entries[1].IsDir {
		t.Fatalf("second entry = %#v", entries[1])
	}
}

func boolPtr(value bool) *bool {
	return &value
}
