package main

import (
	"context"
	"os/exec"
	"strconv"
	"time"
)

func configureCancellation(cmd *exec.Cmd) {
	cmd.Cancel = func() error {
		// taskkill /T includes descendants. Bound the cleanup command itself, then
		// fall back to killing the parent if taskkill is unavailable or fails.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := exec.CommandContext(ctx, "taskkill", "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F").Run(); err == nil {
			return nil
		}
		return cmd.Process.Kill()
	}
}
