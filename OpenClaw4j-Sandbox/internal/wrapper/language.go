package wrapper

import (
	"fmt"
	"strings"

	"github.com/seaskyland/openclaw4j-sandbox/internal/model"
)

type Language string

const (
	Python     Language = "python"
	JavaScript Language = "javascript"
)

func ParseLanguage(value string) (Language, bool) {
	switch strings.ToLower(value) {
	case "python", "python3":
		return Python, true
	case "javascript", "js":
		return JavaScript, true
	default:
		return "", false
	}
}

func ValidateRequest(req model.ExecuteRequest) (Language, error) {
	if strings.TrimSpace(req.Code) == "" {
		return "", fmt.Errorf("script code cannot be empty")
	}
	language, ok := ParseLanguage(req.Language)
	if !ok {
		return "", fmt.Errorf("unsupported script language: %s", req.Language)
	}
	return language, nil
}

func (l Language) UserFileName() string {
	switch l {
	case Python:
		return "user.py"
	case JavaScript:
		return "user.js"
	default:
		return "user.txt"
	}
}

func (l Language) RunnerFileName() string {
	switch l {
	case Python:
		return "runner.py"
	case JavaScript:
		return "runner.js"
	default:
		return "runner.txt"
	}
}

func (l Language) RuntimeCommand(pythonRuntime string) string {
	if l == Python {
		return pythonRuntime
	}
	return "node"
}

func (l Language) RuntimeArgs() []string {
	if l == JavaScript {
		return []string{"--jitless"}
	}
	return nil
}
