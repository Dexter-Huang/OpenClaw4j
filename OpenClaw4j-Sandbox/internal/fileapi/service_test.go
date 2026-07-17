package fileapi

import (
	"os"
	"path/filepath"
	"strings"
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

func TestServiceSearchesFileWithRegex(t *testing.T) {
	root, service := newTestService(t)
	if err := os.WriteFile(filepath.Join(root, "app.log"), []byte("alpha\nbeta\nalphabet\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := service.Search(SearchRequest{
		File:  "app.log",
		Regex: "alpha",
	})
	if !result.Success {
		t.Fatalf("search failed: %#v", result)
	}

	matches := result.Data["matches"].([]Match)
	if len(matches) != 2 {
		t.Fatalf("matches = %#v", matches)
	}
	if matches[0].Line != 0 || matches[0].Column != 0 || matches[0].Text != "alpha" {
		t.Fatalf("first match = %#v", matches[0])
	}
}

func TestServiceGrepsDirectory(t *testing.T) {
	root, service := newTestService(t)
	if err := os.MkdirAll(filepath.Join(root, "logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "logs", "a.txt"), []byte("Needle\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "logs", "b.txt"), []byte("needle again\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := service.Grep(GrepRequest{
		Path:            "logs",
		Pattern:         "needle",
		CaseInsensitive: boolPtr(true),
		MaxResults:      intPtr(1),
	})
	if !result.Success {
		t.Fatalf("grep failed: %#v", result)
	}

	matches := result.Data["matches"].([]Match)
	if len(matches) != 1 {
		t.Fatalf("matches = %#v", matches)
	}
	if !strings.HasSuffix(filepath.ToSlash(matches[0].File), "logs/a.txt") {
		t.Fatalf("file = %s", matches[0].File)
	}
}

func TestServiceGlobsWorkspace(t *testing.T) {
	root, service := newTestService(t)
	if err := os.MkdirAll(filepath.Join(root, "src", "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"src/main.go", "src/pkg/lib.go", "src/pkg/readme.md"} {
		if err := os.WriteFile(filepath.Join(root, path), []byte(path), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	result := service.Glob(GlobRequest{
		Path:    ".",
		Pattern: "**/*.go",
	})
	if !result.Success {
		t.Fatalf("glob failed: %#v", result)
	}

	paths := result.Data["paths"].([]string)
	if len(paths) != 2 {
		t.Fatalf("paths = %#v", paths)
	}
	if paths[0] != filepath.ToSlash(filepath.Join(root, "src/main.go")) {
		t.Fatalf("first path = %s", paths[0])
	}
}

func TestServiceStrReplaceEditorCreateViewReplaceInsertAndUndo(t *testing.T) {
	root, service := newTestService(t)

	create := service.StrReplaceEditor(StrReplaceEditorRequest{
		Command:  "create",
		Path:     "story.txt",
		FileText: "one\nthree\n",
	})
	if !create.Success {
		t.Fatalf("create failed: %#v", create)
	}

	view := service.StrReplaceEditor(StrReplaceEditorRequest{Command: "view", Path: "story.txt"})
	if !view.Success || view.Data["content"] != "one\nthree\n" {
		t.Fatalf("view = %#v", view)
	}

	replace := service.StrReplaceEditor(StrReplaceEditorRequest{
		Command: "str_replace",
		Path:    "story.txt",
		OldStr:  "three",
		NewStr:  "four",
	})
	if !replace.Success {
		t.Fatalf("replace failed: %#v", replace)
	}

	insert := service.StrReplaceEditor(StrReplaceEditorRequest{
		Command:    "insert",
		Path:       "story.txt",
		InsertLine: intPtr(1),
		NewStr:     "two\n",
	})
	if !insert.Success {
		t.Fatalf("insert failed: %#v", insert)
	}

	content, err := os.ReadFile(filepath.Join(root, "story.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "one\ntwo\nfour\n" {
		t.Fatalf("content = %q", content)
	}

	undo := service.StrReplaceEditor(StrReplaceEditorRequest{Command: "undo_edit", Path: "story.txt"})
	if undo.Success || undo.Data["error_type"] != "unsupported_operation" {
		t.Fatalf("undo = %#v", undo)
	}
}

func boolPtr(value bool) *bool {
	return &value
}

func intPtr(value int) *int {
	return &value
}
