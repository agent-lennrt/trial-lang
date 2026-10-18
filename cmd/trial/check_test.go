package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func checkFixture(t *testing.T, source string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "source.trial")
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCheckDoesNotExecute(t *testing.T) {
	for _, body := range []string{
		`PROCLAIM "must not print". AWAIT SUMMONS, FILED UNDER pending.`,
		`REFER TO ARTICLE 1.`,
		`HOLD "must not become a verdict" IN CONTEMPT.`,
		`COMMENCE PROCEEDINGS UPON "invalid child source", FILED UNDER child.`,
	} {
		t.Run(body, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
			defer cancel()
			var out, diagnostics bytes.Buffer
			code := checkLocalCommand(ctx, []string{checkFixture(t, localSource(body))}, &out, &diagnostics)
			if code != 0 || out.String() != "Source accepted.\n" || diagnostics.Len() != 0 {
				t.Fatalf("exit=%d stdout=%q stderr=%q", code, &out, &diagnostics)
			}
		})
	}
}

func TestCheckLocalDependenciesAndForms(t *testing.T) {
	base := `FORM S-1.
IN THE MATTER OF: foundation.
THE OFFICE OF base, CONCERNING n.
REMAND WITH n TIMES 2.`
	mid := `FORM S-1.
IN THE MATTER OF: building.
INCORPORATE BY REFERENCE foundation.
THE OFFICE OF twice, CONCERNING n.
REMAND WITH THE FINDING OF base REGARDING n.`
	program := `FORM K-1.
IN THE MATTER OF: checking.
INCORPORATE BY REFERENCE building.
ARTICLE 1.
PROCLAIM THE FINDING OF twice REGARDING 21.`
	basePath, midPath, programPath := checkFixture(t, base), checkFixture(t, mid), checkFixture(t, program)
	for _, tc := range []struct {
		name string
		args []string
		code int
	}{
		{"standalone statute", []string{basePath}, 0},
		{"dependent statute", []string{midPath, "--enact", basePath}, 0},
		{"transitive case", []string{"--enact", basePath, programPath, "--enact", midPath}, 0},
		{"missing dependency", []string{programPath}, 1},
		{"wrong enactment order", []string{programPath, "--enact", midPath, "--enact", basePath}, 1},
		{"canon", []string{"--canon", "../../examples/the-wave-chamber.trial"}, 0},
		{"no canon", []string{"../../examples/the-wave-chamber.trial"}, 1},
		{"invalid statute", []string{programPath, "--enact", checkFixture(t, "invalid")}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, diagnostics bytes.Buffer
			code := checkLocalCommand(t.Context(), tc.args, &out, &diagnostics)
			if code != tc.code || (code == 0 && (out.String() != "Source accepted.\n" || diagnostics.Len() != 0)) || (code != 0 && (out.Len() != 0 || diagnostics.Len() == 0)) {
				t.Fatalf("exit=%d stdout=%q stderr=%q", code, &out, &diagnostics)
			}
		})
	}
}

func TestCheckRejectedSources(t *testing.T) {
	for _, tc := range []struct{ source, diagnostic string }{
		{"", "expected"},
		{localSource("PROCLAIM ."), "expected"},
		{localSource("REFER TO ARTICLE 99."), "99"},
		{"FORM K-2.\nIN THE MATTER OF: amendment.\nARTICLE 2.\nPROCLAIM 1.", "existing case"},
		{"FORM S-1.\nIN THE MATTER OF: broken.\nTHE OFFICE OF twice, CONCERNING n.\nREMAND WITH THE FINDING OF missing REGARDING n.", "missing"},
		{strings.Repeat("x", maxSourceBytes+1), "exceeds"},
	} {
		var out, diagnostics bytes.Buffer
		code := checkLocalCommand(t.Context(), []string{checkFixture(t, tc.source)}, &out, &diagnostics)
		if code != 1 || out.Len() != 0 || !strings.Contains(diagnostics.String(), tc.diagnostic) {
			t.Fatalf("exit=%d stdout=%q stderr=%q; want %q", code, &out, &diagnostics, tc.diagnostic)
		}
	}
}

func TestCheckArguments(t *testing.T) {
	for _, args := range [][]string{
		nil,
		{"a.trial", "b.trial"},
		{"--timeout=0", "missing.trial"},
		{"missing.trial", "--timeout=-1s"},
		{"--timeout=forever", "missing.trial"},
		{"--broker=localhost:9092", "missing.trial"},
		{"--serve=1", "missing.trial"},
		{"--enact=", "missing.trial"},
		{"--enact=-", "-"},
		{"--enact=-", "--enact=-", "missing.trial"},
		append([]string{"missing.trial"}, strings.Fields(strings.Repeat("--enact=x ", maxRunEnactments+1))...),
	} {
		var out, diagnostics bytes.Buffer
		if code := checkLocalCommand(t.Context(), args, &out, &diagnostics); code != 2 || out.Len() != 0 || diagnostics.Len() == 0 {
			t.Fatalf("args=%q exit=%d stdout=%q stderr=%q", args, code, &out, &diagnostics)
		}
	}
}

func TestCheckReadAndCancellationErrors(t *testing.T) {
	path := checkFixture(t, localSource(`PROCLAIM 1.`))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := checkLocalSource(ctx, localSource(`PROCLAIM 1.`), nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled validation: %v", err)
	}
	for _, tc := range []struct {
		ctx  context.Context
		args []string
	}{
		{ctx, []string{path}},
		{t.Context(), []string{path, "--timeout=1ns"}},
		{t.Context(), []string{filepath.Join(t.TempDir(), "absent.trial")}},
	} {
		var out, diagnostics bytes.Buffer
		if code := checkLocalCommand(tc.ctx, tc.args, &out, &diagnostics); code != 1 || out.Len() != 0 || diagnostics.Len() == 0 {
			t.Fatalf("exit=%d stdout=%q stderr=%q", code, &out, &diagnostics)
		}
	}
}

type checkResultWriter struct{ err error }

func (w checkResultWriter) Write([]byte) (int, error) { return 0, w.err }

func TestCheckOutputFailure(t *testing.T) {
	path := checkFixture(t, localSource(`PROCLAIM 1.`))
	for _, err := range []error{nil, io.ErrClosedPipe} {
		var diagnostics bytes.Buffer
		code := checkLocalCommand(t.Context(), []string{path}, checkResultWriter{err}, &diagnostics)
		if code != 1 || !strings.Contains(diagnostics.String(), "write result") {
			t.Fatalf("exit=%d stderr=%q", code, &diagnostics)
		}
	}
}

func TestCLICheckStdin(t *testing.T) {
	cmd := cliProcess(t, "check", "-")
	cmd.Stdin = strings.NewReader(localSource(`PROCLAIM "not executed". AWAIT SUMMONS, FILED UNDER value.`))
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil || stdout.String() != "Source accepted.\n" || stderr.Len() != 0 {
		t.Fatalf("check: %v stdout=%q stderr=%q", err, &stdout, &stderr)
	}
}

func TestCLICheckStatuteFromStdin(t *testing.T) {
	path := checkFixture(t, `FORM K-1.
IN THE MATTER OF: stdin-statute.
INCORPORATE BY REFERENCE supplied.
ARTICLE 1.
PROCLAIM THE FINDING OF answer.`)
	cmd := cliProcess(t, "check", path, "--enact", "-")
	cmd.Stdin = strings.NewReader(`FORM S-1.
IN THE MATTER OF: supplied.
THE OFFICE OF answer.
REMAND WITH 42.`)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil || stdout.String() != "Source accepted.\n" || stderr.Len() != 0 {
		t.Fatalf("check: %v stdout=%q stderr=%q", err, &stdout, &stderr)
	}
}

func TestCLICheckTimeoutOnOpenStdin(t *testing.T) {
	cmd := cliProcess(t, "check", "-", "--timeout=50ms")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = stdin.Close() }()
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	started := time.Now()
	if err := cmd.Run(); err == nil || cmd.ProcessState.ExitCode() != 1 {
		t.Fatalf("check exit=%v stderr=%q", err, &stderr)
	}
	if elapsed := time.Since(started); elapsed > 10*time.Second {
		t.Fatalf("check deadline did not bound stdin: %s", elapsed)
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "context deadline exceeded") || strings.Count(stderr.String(), "trial check:") != 1 {
		t.Fatalf("stdout=%q stderr=%q", &stdout, &stderr)
	}
}
