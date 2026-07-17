package pathguard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Guard struct {
	root string
}

type Error struct {
	Input     string
	Message   string
	ErrorType string
}

func (e Error) Error() string {
	return e.Message
}

func New(root string) Guard {
	return Guard{root: filepath.Clean(root)}
}

func (g Guard) Root() string {
	return g.root
}

func (g Guard) EnsureWorkspace() error {
	return os.MkdirAll(g.root, 0o755)
}

func (g Guard) Resolve(input string) (string, error) {
	cleanInput := filepath.Clean(input)
	candidate := cleanInput
	if !filepath.IsAbs(cleanInput) {
		candidate = filepath.Join(g.root, cleanInput)
	}
	candidate = filepath.Clean(candidate)

	rel, err := filepath.Rel(g.root, candidate)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", Error{
			Input:     input,
			Message:   fmt.Sprintf("path must stay inside workspace %s", g.root),
			ErrorType: "invalid_path",
		}
	}

	return candidate, nil
}
