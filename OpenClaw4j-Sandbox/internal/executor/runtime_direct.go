//go:build !linux

package executor

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"time"

	"github.com/seaskyland/openclaw4j-sandbox/internal/config"
	"github.com/seaskyland/openclaw4j-sandbox/internal/wrapper"
)

type DirectRuntime struct{}

func (DirectRuntime) Run(parent context.Context, cfg config.Config, language wrapper.Language, workDir string, timeoutMs uint64) (RuntimeResult, error) {
	ctx, cancel := context.WithTimeout(parent, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	args := append([]string{}, language.RuntimeArgs()...)
	args = append(args, language.RunnerFileName())
	cmd := exec.CommandContext(ctx, language.RuntimeCommand(cfg.PythonRuntime), args...)
	cmd.Dir = workDir
	for key, value := range SandboxEnvVars(cfg) {
		cmd.Env = append(cmd.Env, key+"="+value)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return RuntimeResult{Timeout: true}, nil
	}
	exitCode := 0
	success := true
	if err != nil {
		success = false
		exitCode = 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return RuntimeResult{}, err
		}
	}
	return RuntimeResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), ExitCode: &exitCode, Success: success}, nil
}
