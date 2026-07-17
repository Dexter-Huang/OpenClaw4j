package executor

import (
	"context"

	"github.com/seaskyland/openclaw4j-sandbox/internal/config"
	"github.com/seaskyland/openclaw4j-sandbox/internal/wrapper"
)

type RuntimeResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode *int
	Success  bool
	Timeout  bool
}

type Runtime interface {
	Run(ctx context.Context, cfg config.Config, language wrapper.Language, workDir string, timeoutMs uint64) (RuntimeResult, error)
}
