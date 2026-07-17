//go:build linux

package main

import "github.com/seaskyland/openclaw4j-sandbox/internal/executor"

func defaultRuntime() executor.Runtime {
	return executor.SandlockRuntime{}
}
