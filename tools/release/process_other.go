//go:build !unix && !windows

package main

import "os/exec"

// Release preparation is supported on Unix and Windows. Preserve bounded
// direct-command cancellation on other Go platforms.
func configureCancellation(cmd *exec.Cmd) {}
