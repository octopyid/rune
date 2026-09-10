//go:build !windows

package executor

import (
	"os"
	"os/exec"
	"syscall"
)

func setProcessGroup(cmd *exec.Cmd) {
	// Child processes share the parent's process group so that interactive
	// programs (e.g. flutter, vim, bash) have foreground terminal access (TTY)
	// and receive keyboard input (avoiding SIGTTIN) and terminal signals (SIGINT).
}

func forwardSignal(cmd *exec.Cmd, sig os.Signal) {
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Signal(sig)
	}
}

func exitStatus(exitErr *exec.ExitError) int {
	if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
		if status.Signaled() {
			return 128 + int(status.Signal())
		}
		return status.ExitStatus()
	}
	return exitErr.ExitCode()
}
