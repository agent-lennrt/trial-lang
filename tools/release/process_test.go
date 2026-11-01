package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestCommandTreeHelper(t *testing.T) {
	if os.Getenv("TRIAL_RELEASE_PROCESS_HELPER") != "1" {
		return
	}
	args := os.Args[len(os.Args)-4:]
	if args[0] == "signal-parent" {
		ctx, stop := interruptContext(context.Background())
		defer stop()
		_, err := runCommand(ctx, 10*time.Second, os.Args[0], "-test.run=^TestCommandTreeHelper$", "--", "parent", args[1], args[2], args[3])
		if !errors.Is(err, context.Canceled) {
			os.Exit(6)
		}
		os.Exit(0)
	}
	if args[0] == "parent" {
		child := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestCommandTreeHelper$", "--", "child", args[1], args[2], args[3])
		child.Stdout, child.Stderr = os.Stdout, os.Stderr
		if err := child.Run(); err != nil {
			os.Exit(2)
		}
		os.Exit(0)
	}
	if err := os.WriteFile(args[1], []byte("ready"), 0600); err != nil {
		os.Exit(3)
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(args[2]); err == nil {
			if err := os.WriteFile(args[3], []byte("survived"), 0600); err != nil {
				os.Exit(4)
			}
			os.Exit(0)
		}
		time.Sleep(5 * time.Millisecond)
	}
	os.Exit(5)
}

func TestCommandCancellationStopsDescendants(t *testing.T) {
	t.Setenv("TRIAL_RELEASE_PROCESS_HELPER", "1")
	dir := t.TempDir()
	ready, trigger, marker := filepath.Join(dir, "ready"), filepath.Join(dir, "trigger"), filepath.Join(dir, "marker")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := runCommand(ctx, 15*time.Second, os.Args[0], "-test.run=^TestCommandTreeHelper$", "--", "parent", ready, trigger, marker)
		done <- err
	}()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("descendant did not become ready")
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("lost cancellation: %v", err)
		}
	case <-time.After(7 * time.Second):
		t.Fatal("command did not stop")
	}
	if err := os.WriteFile(trigger, []byte("continue"), 0600); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(marker); err == nil {
			t.Fatal("descendant continued after cancellation")
		}
		time.Sleep(5 * time.Millisecond)
	}
}
