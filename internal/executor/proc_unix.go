//go:build !windows

package executor

import (
	"os"
	"os/exec"
	"syscall"
)

func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func forwardSignal(cmd *exec.Cmd, sig os.Signal) {
	if cmd != nil && cmd.Process != nil {
		if s, ok := sig.(syscall.Signal); ok {
			_ = syscall.Kill(-cmd.Process.Pid, s)
			return
		}
		_ = cmd.Process.Signal(sig)
	}
}
