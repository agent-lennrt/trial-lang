package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/lennrt/trial-lang/internal/court"
	"github.com/lennrt/trial-lang/internal/docket"
	"github.com/lennrt/trial-lang/internal/gregor"
)

func checkCmd(ctx context.Context, args []string) int {
	return checkLocalCommandWithRunner(ctx, args, os.Stdout, os.Stderr, func(ctx context.Context, action func() error, diagnostics io.Writer) int {
		return localProcessAction(ctx, action, diagnostics, "check")
	})
}

func checkLocalCommand(ctx context.Context, args []string, out, diagnostics io.Writer) int {
	return checkLocalCommandWithRunner(ctx, args, out, diagnostics, func(_ context.Context, action func() error, diagnostics io.Writer) int {
		return reportCommandResult("check", action(), diagnostics)
	})
}

func checkLocalCommandWithRunner(ctx context.Context, args []string, out, diagnostics io.Writer, perform func(context.Context, func() error, io.Writer) int) int {
	fs := commandFlags("check")
	fs.SetOutput(diagnostics)
	useCanon := fs.Bool("canon", false, "enact the bundled canon first")
	timeout := fs.Duration("timeout", 30*time.Second, "positive limit for loading and checking")
	var enactments []string
	fs.Func("enact", "statute path to enact; repeat for each dependency", func(path string) error {
		if path == "" || len(enactments) >= maxRunEnactments {
			return fmt.Errorf("provide a nonempty statute path, at most %d times", maxRunEnactments)
		}
		enactments = append(enactments, path)
		return nil
	})
	path, ok := parseFirstArg(fs, args)
	if !ok {
		return 2
	}
	if path == "" || fs.NArg() != 0 {
		_, _ = fmt.Fprintln(diagnostics, "trial check: provide exactly one source path. See 'trial help check'.")
		return 2
	}
	if *timeout <= 0 {
		_, _ = fmt.Fprintln(diagnostics, "trial check: --timeout must be positive, for example 30s.")
		return 2
	}
	stdinCount := 0
	for _, sourcePath := range append([]string{path}, enactments...) {
		if sourcePath == "-" {
			stdinCount++
		}
	}
	if stdinCount > 1 {
		_, _ = fmt.Fprintln(diagnostics, "trial check: standard input can supply only one source.")
		return 2
	}

	checkCtx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()
	return perform(checkCtx, func() error {
		source, statutes, err := loadLocalSources(checkCtx, path, enactments, *useCanon)
		if err != nil {
			return err
		}
		if err := checkLocalSource(checkCtx, source, statutes); err != nil {
			return err
		}
		if err := checkCtx.Err(); err != nil {
			return err
		}
		const accepted = "Source accepted.\n"
		n, err := io.WriteString(out, accepted)
		if err == nil && n != len(accepted) {
			err = io.ErrShortWrite
		}
		if err != nil {
			return fmt.Errorf("write result: %w", err)
		}
		return checkCtx.Err()
	}, diagnostics)
}

// checkLocalSource uses the same acceptance rules as real filings. Its log is
// temporary and no Court proceeds, so source cannot produce runtime effects.
func checkLocalSource(ctx context.Context, source string, statutes []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	log := docket.NewMemoryLog()
	defer log.Close()
	for i, statute := range statutes {
		if _, _, err := court.Enact(ctx, log, statute); err != nil {
			return fmt.Errorf("enact statute %d: %w", i+1, err)
		}
	}
	program, err := gregor.Parse(source)
	if err != nil {
		return err
	}
	switch program.Form {
	case "K-1":
		_, err = court.File(ctx, log, source)
	case "S-1":
		_, _, err = court.Enact(ctx, log, source)
	default:
		return errors.New("checking Form K-2 requires an existing case; check supports Form K-1 and Form S-1")
	}
	if err != nil {
		return err
	}
	return ctx.Err()
}
