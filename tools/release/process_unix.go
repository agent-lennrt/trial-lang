//go:build unix

package main

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

// Give each command its own process group so cancellation also stops make's
// compilers, tests, and other descendants before a retry can begin.
func configureCancellation(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
}
