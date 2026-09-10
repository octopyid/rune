//go:build windows

package executor

import (
	"os"
	"os/exec"
)

func setProcessGroup(cmd *exec.Cmd) {
	// Process groups operate differently on Windows
}

func forwardSignal(cmd *exec.Cmd, sig os.Signal) {
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Signal(sig)
	}
}

func exitStatus(exitErr *exec.ExitError) int {
	return exitErr.ExitCode()
}
