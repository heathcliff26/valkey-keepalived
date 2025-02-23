package cmd

import (
	"os"
	"os/exec"
	"testing"

	"github.com/heathcliff26/valkey-keepalived/pkg/version"
	"github.com/stretchr/testify/assert"
)

func TestNewRootCommand(t *testing.T) {
	cmd := NewRootCommand()

	assert.Equal(t, version.Name, cmd.Use)
}

func TestCommandRun(t *testing.T) {
	if os.Getenv("RUN_CRASH_TEST") == "1" {
		cmd := NewRootCommand()
		cmd.SetArgs([]string{"-c", "not-a-file.yaml"})
		_ = cmd.Execute()
	}
	execExitTest(t, "TestCommandRun", true)
}

func TestExecute(t *testing.T) {
	if os.Getenv("RUN_CRASH_TEST") == "1" {
		oldArgs := os.Args
		t.Cleanup(func() {
			os.Args = oldArgs
		})
		os.Args = append(os.Args[0:], "-c")
		Execute()
	}
	execExitTest(t, "TestExecute", true)
}

func execExitTest(t *testing.T, test string, exitsError bool) {
	cmd := exec.Command(os.Args[0], "-test.run="+test)
	cmd.Env = append(os.Environ(), "RUN_CRASH_TEST=1")
	err := cmd.Run()
	if exitsError && err == nil {
		t.Fatal("Process exited without error")
	} else if !exitsError && err == nil {
		return
	}
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}
