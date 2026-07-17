package fileapi

import (
	"encoding/base64"
	"errors"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/seaskyland/openclaw4j-sandbox/internal/pathguard"
)

type Service struct {
	guard          pathguard.Guard
	readLimitBytes int
}

type Response struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
}

type ReadRequest struct {
	File      string `json:"file"`
	StartLine *int   `json:"start_line,omitempty"`
	EndLine   *int   `json:"end_line,omitempty"`
	Sudo      *bool  `json:"sudo,omitempty"`
}

type WriteRequest struct {
	File            string `json:"file"`
	Content         string `json:"content"`
	Encoding        string `json:"encoding,omitempty"`
	Append          *bool  `json:"append,omitempty"`
	LeadingNewline  *bool  `json:"leading_newline,omitempty"`
	TrailingNewline *bool  `json:"trailing_newline,omitempty"`
	Sudo            *bool  `json:"sudo,omitempty"`
}

type ReplaceRequest struct {
	File       string `json:"file"`
	OldStr     string `json:"old_str"`
	NewStr     string `json:"new_str"`
	ReplaceAll *bool  `json:"replace_all,omitempty"`
	Sudo       *bool  `json:"sudo,omitempty"`
}

type ListRequest struct {
	Path        string `json:"path"`
	Recursive   *bool  `json:"recursive,omitempty"`
	ShowHidden  *bool  `json:"show_hidden,omitempty"`
	IncludeSize *bool  `json:"include_size,omitempty"`
}

type SearchRequest struct {
	File            string `json:"file"`
	Regex           string `json:"regex"`
	CaseInsensitive *bool  `json:"case_insensitive,omitempty"`
	MaxResults      *int   `json:"max_results,omitempty"`
	Sudo            *bool  `json:"sudo,omitempty"`
}

type GrepRequest struct {
	Path            string `json:"path"`
	Pattern         string `json:"pattern"`
	CaseInsensitive *bool  `json:"case_insensitive,omitempty"`
	MaxResults      *int   `json:"max_results,omitempty"`
}

type FindRequest struct {
	Path       string `json:"path"`
	Pattern    string `json:"pattern"`
	MaxResults *int   `json:"max_results,omitempty"`
}

type GlobRequest struct {
	Path        string `json:"path"`
	Pattern     string `json:"pattern"`
	IncludeDirs *bool  `json:"include_dirs,omitempty"`
	ShowHidden  *bool  `json:"show_hidden,omitempty"`
	MaxResults  *int   `json:"max_results,omitempty"`
}

type StrReplaceEditorRequest struct {
	Command     string `json:"command"`
	Path        string `json:"path"`
	FileText    string `json:"file_text,omitempty"`
	OldStr      string `json:"old_str,omitempty"`
	NewStr      string `json:"new_str,omitempty"`
	InsertLine  *int   `json:"insert_line,omitempty"`
	ReplaceMode string `json:"replace_mode,omitempty"`
}

type Entry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
	Size  *int64 `json:"size,omitempty"`
}

type Match struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Text   string `json:"text"`
}

func NewService(guard pathguard.Guard, readLimitBytes int) Service {
	return Service{guard: guard, readLimitBytes: readLimitBytes}
}

func (s Service) Read(req ReadRequest) Response {
	path, err := s.guard.Resolve(req.File)
	if err != nil {
		return pathError("read", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return fileError("read", path, err)
	}
	if info.IsDir() {
		return businessError("read", path, "path is a directory", "is_directory")
	}
	if s.readLimitBytes > 0 && info.Size() > int64(s.readLimitBytes) && req.StartLine == nil {
		return businessError("read", path, "file is too large to read without line range", "too_large")
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		return fileError("read", path, err)
	}
	content := selectLines(string(bytes), req.StartLine, req.EndLine)

	return ok("File read successfully", map[string]any{
		"file":       path,
		"content":    content,
		"line_count": lineCount(content),
	})
}

func (s Service) Write(req WriteRequest) Response {
	path, err := s.guard.Resolve(req.File)
	if err != nil {
		return pathError("write", err)
	}
	created := !fileExists(path)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fileError("write", path, err)
	}

	bytes, err := decodeContent(req.Content, req.Encoding)
	if err != nil {
		return businessError("write", path, err.Error(), "decode_error")
	}
	if boolValue(req.LeadingNewline) {
		bytes = append([]byte("\n"), bytes...)
	}
	if boolValue(req.TrailingNewline) && !strings.HasSuffix(string(bytes), "\n") {
		bytes = append(bytes, '\n')
	}

	if boolValue(req.Append) {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return fileError("write", path, err)
		}
		defer file.Close()
		if _, err := file.Write(bytes); err != nil {
			return fileError("write", path, err)
		}
	} else if err := os.WriteFile(path, bytes, 0o644); err != nil {
		return fileError("write", path, err)
	}

	return ok("File written successfully", map[string]any{
		"file":          path,
		"bytes_written": len(bytes),
		"created":       created,
	})
}

func (s Service) Replace(req ReplaceRequest) Response {
	path, err := s.guard.Resolve(req.File)
	if err != nil {
		return pathError("replace", err)
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		return fileError("replace", path, err)
	}
	content := string(bytes)
	count := strings.Count(content, req.OldStr)
	if req.OldStr == "" || count == 0 {
		return businessError("replace", path, "old_str was not found", "not_found")
	}

	replacements := 1
	updated := strings.Replace(content, req.OldStr, req.NewStr, 1)
	if boolValue(req.ReplaceAll) {
		replacements = count
		updated = strings.ReplaceAll(content, req.OldStr, req.NewStr)
	}

	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return fileError("replace", path, err)
	}

	return ok("File replaced successfully", map[string]any{
		"file":         path,
		"replacements": replacements,
	})
}

func (s Service) List(req ListRequest) Response {
	path, err := s.guard.Resolve(req.Path)
	if err != nil {
		return pathError("list", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return fileError("list", path, err)
	}
	if !info.IsDir() {
		return businessError("list", path, "path is not a directory", "not_directory")
	}

	entries, err := listEntries(path, boolValue(req.Recursive), boolValue(req.ShowHidden), boolValue(req.IncludeSize))
	if err != nil {
		return fileError("list", path, err)
	}

	return ok("Directory listed successfully", map[string]any{
		"path":    path,
		"entries": entries,
	})
}

func (s Service) Search(req SearchRequest) Response {
	file, err := s.guard.Resolve(req.File)
	if err != nil {
		return pathError("search", err)
	}

	bytes, err := os.ReadFile(file)
	if err != nil {
		return fileError("search", file, err)
	}
	re, err := compilePattern(req.Regex, boolValue(req.CaseInsensitive))
	if err != nil {
		return businessError("search", file, err.Error(), "invalid_regex")
	}

	matches := findMatches(file, string(bytes), re, intValue(req.MaxResults))
	return ok("File searched successfully", map[string]any{
		"file":    file,
		"matches": matches,
	})
}

func (s Service) Grep(req GrepRequest) Response {
	root, err := s.guard.Resolve(req.Path)
	if err != nil {
		return pathError("grep", err)
	}

	re, err := compilePattern(req.Pattern, boolValue(req.CaseInsensitive))
	if err != nil {
		return businessError("grep", root, err.Error(), "invalid_regex")
	}

	limit := intValue(req.MaxResults)
	matches := make([]Match, 0)
	err = filepath.WalkDir(root, func(file string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if limit > 0 && len(matches) >= limit {
			return filepath.SkipAll
		}
		bytes, err := os.ReadFile(file)
		if err != nil {
			return nil
		}
		matches = append(matches, findMatches(file, string(bytes), re, limit-len(matches))...)
		return nil
	})
	if err != nil {
		return fileError("grep", root, err)
	}

	return ok("Directory grepped successfully", map[string]any{
		"path":    root,
		"matches": matches,
	})
}

func (s Service) Find(req FindRequest) Response {
	return s.Glob(GlobRequest{Path: req.Path, Pattern: req.Pattern, MaxResults: req.MaxResults})
}

func (s Service) Glob(req GlobRequest) Response {
	root, err := s.guard.Resolve(req.Path)
	if err != nil {
		return pathError("glob", err)
	}

	pattern := req.Pattern
	if pattern == "" {
		pattern = "*"
	}
	paths := make([]string, 0)
	limit := intValue(req.MaxResults)
	err = filepath.WalkDir(root, func(candidate string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if candidate == root {
			return nil
		}
		if !boolValue(req.ShowHidden) && strings.HasPrefix(entry.Name(), ".") {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() && !boolValue(req.IncludeDirs) {
			return nil
		}
		rel, err := filepath.Rel(root, candidate)
		if err != nil {
			return err
		}
		if globMatches(pattern, rel) {
			paths = append(paths, filepath.ToSlash(candidate))
			if limit > 0 && len(paths) >= limit {
				return filepath.SkipAll
			}
		}
		return nil
	})
	if err != nil {
		return fileError("glob", root, err)
	}
	sort.Strings(paths)

	return ok("Files globbed successfully", map[string]any{
		"path":  root,
		"paths": paths,
	})
}

func (s Service) StrReplaceEditor(req StrReplaceEditorRequest) Response {
	switch req.Command {
	case "view":
		path, err := s.guard.Resolve(req.Path)
		if err != nil {
			return pathError("str_replace_editor", err)
		}
		info, err := os.Stat(path)
		if err != nil {
			return fileError("str_replace_editor", path, err)
		}
		if info.IsDir() {
			return s.List(ListRequest{Path: req.Path})
		}
		return s.Read(ReadRequest{File: req.Path})
	case "create":
		path, err := s.guard.Resolve(req.Path)
		if err != nil {
			return pathError("str_replace_editor", err)
		}
		if fileExists(path) {
			return businessError("str_replace_editor", path, "file already exists", "already_exists")
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fileError("str_replace_editor", path, err)
		}
		if err := os.WriteFile(path, []byte(req.FileText), 0o644); err != nil {
			return fileError("str_replace_editor", path, err)
		}
		return ok("File created successfully", map[string]any{
			"file":          path,
			"bytes_written": len([]byte(req.FileText)),
		})
	case "str_replace":
		return s.editorReplace(req)
	case "insert":
		return s.editorInsert(req)
	case "undo_edit":
		return businessError("str_replace_editor", req.Path, "undo_edit is not supported", "unsupported_operation")
	default:
		return businessError("str_replace_editor", req.Path, "unsupported editor command", "unsupported_operation")
	}
}

func (s Service) editorReplace(req StrReplaceEditorRequest) Response {
	path, err := s.guard.Resolve(req.Path)
	if err != nil {
		return pathError("str_replace_editor", err)
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return fileError("str_replace_editor", path, err)
	}
	content := string(bytes)
	count := strings.Count(content, req.OldStr)
	if req.OldStr == "" || count == 0 {
		return businessError("str_replace_editor", path, "old_str was not found", "not_found")
	}

	mode := strings.ToUpper(req.ReplaceMode)
	if mode == "" && count != 1 {
		return businessError("str_replace_editor", path, "old_str must be unique when replace_mode is not set", "ambiguous_match")
	}

	replacements := 1
	updated := strings.Replace(content, req.OldStr, req.NewStr, 1)
	switch mode {
	case "ALL":
		replacements = count
		updated = strings.ReplaceAll(content, req.OldStr, req.NewStr)
	case "LAST":
		index := strings.LastIndex(content, req.OldStr)
		updated = content[:index] + req.NewStr + content[index+len(req.OldStr):]
	case "", "FIRST":
	default:
		return businessError("str_replace_editor", path, "unsupported replace_mode", "invalid_request")
	}

	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return fileError("str_replace_editor", path, err)
	}
	return ok("Text replaced successfully", map[string]any{
		"file":         path,
		"replacements": replacements,
	})
}

func (s Service) editorInsert(req StrReplaceEditorRequest) Response {
	if req.InsertLine == nil {
		return businessError("str_replace_editor", req.Path, "insert_line is required", "invalid_request")
	}
	path, err := s.guard.Resolve(req.Path)
	if err != nil {
		return pathError("str_replace_editor", err)
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return fileError("str_replace_editor", path, err)
	}

	lines := strings.SplitAfter(string(bytes), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	index := *req.InsertLine
	if index < 0 || index > len(lines) {
		return businessError("str_replace_editor", path, "insert_line is out of range", "invalid_request")
	}
	updatedLines := append(append([]string{}, lines[:index]...), append([]string{req.NewStr}, lines[index:]...)...)
	if err := os.WriteFile(path, []byte(strings.Join(updatedLines, "")), 0o644); err != nil {
		return fileError("str_replace_editor", path, err)
	}
	return ok("Text inserted successfully", map[string]any{
		"file": path,
	})
}

func listEntries(root string, recursive bool, showHidden bool, includeSize bool) ([]Entry, error) {
	entries := make([]Entry, 0)
	addEntry := func(path string, info os.FileInfo) {
		name := info.Name()
		if path != root {
			if rel, err := filepath.Rel(root, path); err == nil {
				name = rel
			}
		}
		entry := Entry{Name: filepath.ToSlash(name), Path: path, IsDir: info.IsDir()}
		if includeSize && !info.IsDir() {
			size := info.Size()
			entry.Size = &size
		}
		entries = append(entries, entry)
	}

	if recursive {
		err := filepath.WalkDir(root, func(path string, dirEntry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if path == root {
				return nil
			}
			if !showHidden && strings.HasPrefix(dirEntry.Name(), ".") {
				if dirEntry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			info, err := dirEntry.Info()
			if err != nil {
				return err
			}
			addEntry(path, info)
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		dirEntries, err := os.ReadDir(root)
		if err != nil {
			return nil, err
		}
		for _, dirEntry := range dirEntries {
			if !showHidden && strings.HasPrefix(dirEntry.Name(), ".") {
				continue
			}
			info, err := dirEntry.Info()
			if err != nil {
				return nil, err
			}
			addEntry(filepath.Join(root, dirEntry.Name()), info)
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})
	return entries, nil
}

func compilePattern(pattern string, caseInsensitive bool) (*regexp.Regexp, error) {
	if caseInsensitive {
		pattern = "(?i)" + pattern
	}
	return regexp.Compile(pattern)
}

func findMatches(file string, content string, re *regexp.Regexp, limit int) []Match {
	matches := make([]Match, 0)
	lines := strings.Split(content, "\n")
	for lineNo, line := range lines {
		indexes := re.FindAllStringIndex(line, -1)
		for _, index := range indexes {
			matches = append(matches, Match{
				File:   file,
				Line:   lineNo,
				Column: index[0],
				Text:   line[index[0]:index[1]],
			})
			if limit > 0 && len(matches) >= limit {
				return matches
			}
		}
	}
	return matches
}

func globMatches(pattern string, rel string) bool {
	pattern = filepath.ToSlash(pattern)
	rel = filepath.ToSlash(rel)
	if ok, _ := path.Match(pattern, rel); ok {
		return true
	}
	if strings.HasPrefix(pattern, "**/") {
		if ok, _ := path.Match(strings.TrimPrefix(pattern, "**/"), path.Base(rel)); ok {
			return true
		}
	}
	if ok, _ := path.Match(pattern, path.Base(rel)); ok {
		return true
	}
	return false
}

func selectLines(content string, start *int, end *int) string {
	if start == nil && end == nil {
		return content
	}

	lines := strings.SplitAfter(content, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	from := 0
	if start != nil && *start > 0 {
		from = *start
	}
	to := len(lines)
	if end != nil && *end < to {
		to = *end
	}
	if from > len(lines) || from > to {
		return ""
	}
	return strings.Join(lines[from:to], "")
}

func lineCount(content string) int {
	if content == "" {
		return 0
	}
	count := strings.Count(content, "\n")
	if !strings.HasSuffix(content, "\n") {
		count++
	}
	return count
}

func decodeContent(content string, encoding string) ([]byte, error) {
	switch strings.ToLower(encoding) {
	case "", "utf-8", "raw":
		return []byte(content), nil
	case "base64":
		return base64.StdEncoding.DecodeString(content)
	default:
		return nil, errors.New("unsupported encoding")
	}
}

func ok(message string, data map[string]any) Response {
	return Response{Success: true, Message: message, Data: data}
}

func pathError(operation string, err error) Response {
	var guardErr pathguard.Error
	if errors.As(err, &guardErr) {
		return Response{
			Success: false,
			Message: guardErr.Message,
			Data: map[string]any{
				"operation":  operation,
				"message":    guardErr.Message,
				"error_type": guardErr.ErrorType,
				"retryable":  false,
			},
		}
	}
	return businessError(operation, "", err.Error(), "invalid_path")
}

func fileError(operation string, path string, err error) Response {
	errorType := "io_error"
	switch {
	case errors.Is(err, os.ErrNotExist):
		errorType = "not_found"
	case errors.Is(err, os.ErrPermission):
		errorType = "permission_denied"
	}
	return Response{
		Success: false,
		Message: err.Error(),
		Data: map[string]any{
			"path":       path,
			"operation":  operation,
			"message":    err.Error(),
			"error_type": errorType,
			"retryable":  false,
		},
	}
}

func businessError(operation string, path string, message string, errorType string) Response {
	return Response{
		Success: false,
		Message: message,
		Data: map[string]any{
			"path":       path,
			"operation":  operation,
			"message":    message,
			"error_type": errorType,
			"retryable":  false,
		},
	}
}

func boolValue(value *bool) bool {
	return value != nil && *value
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
