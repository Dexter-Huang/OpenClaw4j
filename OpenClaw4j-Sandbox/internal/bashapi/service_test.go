package bashapi

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/seaskyland/openclaw4j-sandbox/internal/pathguard"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	return NewService(pathguard.New(t.TempDir()), 3000, 5000, 64*1024)
}

func TestServiceExecCompletesShortCommand(t *testing.T) {
	service := newTestService(t)

	result := service.Exec(ExecRequest{Command: echoCommand("hello")})
	if !result.Success {
		t.Fatalf("exec failed: %#v", result)
	}
	if result.Data["status"] != "completed" {
		t.Fatalf("status = %#v", result.Data["status"])
	}
	if result.Data["exit_code"] != 0 {
		t.Fatalf("exit_code = %#v", result.Data["exit_code"])
	}
	if !strings.Contains(result.Data["stdout"].(string), "hello") {
		t.Fatalf("stdout = %#v", result.Data["stdout"])
	}
}

func TestServiceExecCapturesStderrAndNonZeroExit(t *testing.T) {
	service := newTestService(t)

	result := service.Exec(ExecRequest{Command: stderrExitCommand("boom", 7)})
	if !result.Success {
		t.Fatalf("exec failed: %#v", result)
	}
	if result.Data["status"] != "completed" {
		t.Fatalf("status = %#v", result.Data["status"])
	}
	if result.Data["exit_code"] != 7 {
		t.Fatalf("exit_code = %#v", result.Data["exit_code"])
	}
	if !strings.Contains(result.Data["stderr"].(string), "boom") {
		t.Fatalf("stderr = %#v", result.Data["stderr"])
	}
}

func TestServiceAsyncOutputAndWrite(t *testing.T) {
	service := newTestService(t)

	start := service.Exec(ExecRequest{
		Command:   readLineCommand(),
		AsyncMode: boolPtr(true),
	})
	if !start.Success {
		t.Fatalf("async exec failed: %#v", start)
	}
	sessionID := start.Data["session_id"].(string)
	commandID := start.Data["command_id"].(string)

	written := service.Write(WriteRequest{
		SessionID: sessionID,
		CommandID: commandID,
		Input:     "agent\n",
	})
	if !written.Success {
		t.Fatalf("write failed: %#v", written)
	}

	var output Response
	for i := 0; i < 20; i++ {
		output = service.Output(OutputRequest{
			SessionID: sessionID,
			CommandID: commandID,
		})
		if output.Data["status"] == "completed" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if output.Data["status"] != "completed" {
		t.Fatalf("status = %#v output=%#v", output.Data["status"], output)
	}
	if !strings.Contains(output.Data["stdout"].(string), "got:agent") {
		t.Fatalf("stdout = %#v", output.Data["stdout"])
	}
}

func TestServiceKillRunningCommand(t *testing.T) {
	service := newTestService(t)

	start := service.Exec(ExecRequest{
		Command:   sleepCommand(),
		AsyncMode: boolPtr(true),
	})
	if !start.Success {
		t.Fatalf("async exec failed: %#v", start)
	}
	sessionID := start.Data["session_id"].(string)
	commandID := start.Data["command_id"].(string)

	killed := service.Kill(KillRequest{SessionID: sessionID, CommandID: commandID})
	if !killed.Success {
		t.Fatalf("kill failed: %#v", killed)
	}
	if killed.Data["status"] != "killed" {
		t.Fatalf("status = %#v", killed.Data["status"])
	}
}

func TestServiceSessionsCreateAndClose(t *testing.T) {
	service := newTestService(t)

	created := service.CreateSession(CreateSessionRequest{})
	if !created.Success {
		t.Fatalf("create session failed: %#v", created)
	}
	sessionID := created.Data["session_id"].(string)

	sessions := service.Sessions()
	if !sessions.Success || len(sessions.Data["sessions"].([]SessionInfo)) != 1 {
		t.Fatalf("sessions = %#v", sessions)
	}

	closed := service.CloseSession(sessionID)
	if !closed.Success {
		t.Fatalf("close session failed: %#v", closed)
	}
}

func boolPtr(value bool) *bool {
	return &value
}

func echoCommand(text string) string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("Write-Output %q", text)
	}
	return fmt.Sprintf("echo %q", text)
}

func stderrExitCommand(text string, code int) string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("Write-Error %q; exit %d", text, code)
	}
	return fmt.Sprintf("echo %q >&2; exit %d", text, code)
}

func readLineCommand() string {
	if runtime.GOOS == "windows" {
		return "$line = [Console]::In.ReadLine(); Write-Output \"got:$line\""
	}
	return "read line; echo got:$line"
}

func sleepCommand() string {
	if runtime.GOOS == "windows" {
		return "Start-Sleep -Seconds 5"
	}
	return "sleep 5"
}
