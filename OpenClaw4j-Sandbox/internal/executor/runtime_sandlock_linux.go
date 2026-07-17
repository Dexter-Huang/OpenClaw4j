//go:build linux

package executor

import (
	"context"
	"errors"
	"time"

	sandlock "github.com/multikernel/sandlock/go"
	"github.com/seaskyland/openclaw4j-sandbox/internal/config"
	"github.com/seaskyland/openclaw4j-sandbox/internal/wrapper"
)

type SandlockRuntime struct{}

func (SandlockRuntime) Run(parent context.Context, cfg config.Config, language wrapper.Language, workDir string, timeoutMs uint64) (RuntimeResult, error) {
	ctx, cancel := context.WithTimeout(parent, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	sb := &sandlock.Sandbox{
		FSReadable:   SandboxReadPaths(cfg),
		FSWritable:   []string{workDir},
		MaxMemory:    cfg.MemoryLimit,
		MaxProcesses: cfg.ProcessLimit,
		CleanEnv:     true,
		Env:          SandboxEnvVars(cfg),
		Cwd:          workDir,
	}

	cmd := append([]string{language.RuntimeCommand(cfg.PythonRuntime)}, language.RuntimeArgs()...)
	cmd = append(cmd, language.RunnerFileName())
	res, err := sb.Run(ctx, cmd...)
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return RuntimeResult{Timeout: true}, nil
	}
	if err != nil {
		return RuntimeResult{}, err
	}
	exitCode := res.ExitCode
	timeout := res.Reason == sandlock.ReasonTimeout
	return RuntimeResult{
		Stdout:   res.Stdout,
		Stderr:   res.Stderr,
		ExitCode: &exitCode,
		Success:  res.Success,
		Timeout:  timeout,
	}, nil
}
