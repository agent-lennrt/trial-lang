package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

type counselDiagnostic struct {
	Message string `json:"message"`
	Range   struct {
		Start struct{ Line, Character int } `json:"start"`
	} `json:"range"`
}

type counselProcessSession struct {
	t           *testing.T
	in          io.WriteCloser
	out         *bufio.Reader
	nextID      int
	versions    map[string]int
	diagnostics map[string][]counselDiagnostic
	finish      func()
}

func startCounselProcess(t *testing.T, args ...string) *counselProcessSession {
	t.Helper()
	cmd := cliProcess(t, append([]string{"counsel"}, args...)...)
	cmd.Env = append(cmd.Env, "TRIAL_BROKER=,")
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	finished := false
	finish := func() {
		if finished {
			return
		}
		finished = true
		_ = in.Close()
		if err := cmd.Wait(); err != nil {
			t.Errorf("Counsel exited: %v; stderr=%s", err, &stderr)
		}
	}
	t.Cleanup(finish)
	s := &counselProcessSession{t: t, in: in, out: bufio.NewReader(out), versions: make(map[string]int), diagnostics: make(map[string][]counselDiagnostic), finish: finish}
	s.request("initialize", map[string]any{})
	return s
}

func (s *counselProcessSession) send(message map[string]any) {
	s.t.Helper()
	message["jsonrpc"] = "2.0"
	body, err := json.Marshal(message)
	if err != nil {
		s.t.Fatal(err)
	}
	if _, err := fmt.Fprintf(s.in, "Content-Length: %d\r\n\r\n%s", len(body), body); err != nil {
		s.t.Fatal(err)
	}
}

// Requests act as protocol barriers: the single connection processes earlier
// document notifications and all diagnostic publications before this response.
func (s *counselProcessSession) request(method string, params map[string]any) {
	s.t.Helper()
	s.nextID++
	s.send(map[string]any{"id": s.nextID, "method": method, "params": params})
	for {
		header, err := s.out.ReadString('\n')
		if err != nil {
			s.t.Fatal(err)
		}
		length, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(header, "Content-Length:")))
		if err != nil || length < 1 || length > 16<<20 {
			s.t.Fatalf("invalid frame header %q: %v", header, err)
		}
		if line, err := s.out.ReadString('\n'); err != nil || line != "\r\n" {
			s.t.Fatalf("invalid frame separator %q: %v", line, err)
		}
		body := make([]byte, length)
		if _, err := io.ReadFull(s.out, body); err != nil {
			s.t.Fatal(err)
		}
		var reply struct {
			ID     int             `json:"id"`
			Method string          `json:"method"`
			Error  json.RawMessage `json:"error"`
			Params struct {
				URI         string              `json:"uri"`
				Diagnostics []counselDiagnostic `json:"diagnostics"`
			} `json:"params"`
		}
		if err := json.Unmarshal(body, &reply); err != nil || len(reply.Error) != 0 {
			s.t.Fatalf("invalid response %s: %v", body, err)
		}
		if reply.Method == "textDocument/publishDiagnostics" {
			s.diagnostics[reply.Params.URI] = reply.Params.Diagnostics
			continue
		}
		if reply.ID != s.nextID {
			s.t.Fatalf("unexpected response %s", body)
		}
		return
	}
}

func (s *counselProcessSession) document(method, uri, text string) {
	s.t.Helper()
	document := map[string]any{"uri": uri}
	params := map[string]any{"textDocument": document}
	switch method {
	case "didOpen":
		s.versions[uri] = 1
		document["text"], document["version"] = text, s.versions[uri]
	case "didChange":
		s.versions[uri]++
		document["version"] = s.versions[uri]
		params["contentChanges"] = []map[string]any{{"text": text}}
	case "didClose":
		delete(s.versions, uri)
	}
	s.send(map[string]any{"method": "textDocument/" + method, "params": params})
	s.request("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]int{"line": 0, "character": 0}})
}

const editorStatute = `FORM S-1.
IN THE MATTER OF: custom.
THE OFFICE OF twice, CONCERNING n.
REMAND WITH n TIMES 2.
`

const editorCase = `FORM K-1.
IN THE MATTER OF: editor.
INCORPORATE BY REFERENCE custom.
ARTICLE 1.
PROCLAIM THE FINDING OF twice REGARDING 3.
ADJOURN INDEFINITELY.
`

func TestCLICounselCustomImports(t *testing.T) {
	file := counselFixtureFile(t, "custom #1.trial", editorStatute)
	for _, useCanon := range []bool{false, true} {
		t.Run(fmt.Sprintf("canon=%t", useCanon), func(t *testing.T) {
			args := []string{"--enact", file.path}
			source := editorCase
			if useCanon {
				args = append(args, "--canon")
				source = strings.Replace(source, "ARTICLE 1.", "INCORPORATE BY REFERENCE statutes-of-arithmetic.\nARTICLE 1.", 1)
			}
			s := startCounselProcess(t, args...)
			defer s.finish()
			const uri = "file:///custom-editor.trial"
			s.document("didOpen", uri, source)
			for owner, diagnostics := range s.diagnostics {
				if len(diagnostics) != 0 {
					t.Fatalf("valid custom import has diagnostics at %s: %+v", owner, diagnostics)
				}
			}
			s.document("didChange", uri, strings.Replace(source, "REFERENCE custom.", "REFERENCE missing.", 1))
			if got := s.diagnostics[uri]; len(got) != 1 || !strings.Contains(got[0].Message, "missing") {
				t.Fatalf("missing import diagnostics = %+v", s.diagnostics)
			}
		})
	}
}

func TestCLICounselCustomOverlayAndDiskRestore(t *testing.T) {
	broken := strings.Replace(editorStatute, "n TIMES 2", "THE FINDING OF absent REGARDING n", 1)
	file := counselFixtureFile(t, "dependency with spaces.trial", broken)
	s := startCounselProcess(t, "--enact", file.path)
	defer s.finish()
	const uri = "file:///dependent-editor.trial"
	s.document("didOpen", uri, editorCase)
	assertBroken := func() {
		t.Helper()
		got := s.diagnostics[file.uri]
		if len(got) != 1 || !strings.Contains(got[0].Message, "absent") || got[0].Range.Start.Line != 3 {
			t.Fatalf("dependency error must use its URI and line: %+v", s.diagnostics)
		}
	}
	assertBroken()
	s.document("didOpen", file.uri, editorStatute)
	if len(s.diagnostics[file.uri]) != 0 || len(s.diagnostics[uri]) != 0 {
		t.Fatalf("valid overlay did not clear errors: %+v", s.diagnostics)
	}
	s.document("didChange", file.uri, broken)
	assertBroken()
	s.document("didChange", file.uri, editorStatute)
	if len(s.diagnostics[file.uri]) != 0 || len(s.diagnostics[uri]) != 0 {
		t.Fatalf("second overlay repair did not clear errors: %+v", s.diagnostics)
	}
	s.document("didClose", file.uri, "")
	assertBroken()
	// Closing must reread only the configured file, including edits saved since startup.
	s.document("didOpen", file.uri, broken)
	writeCounselFixture(t, file.path, editorStatute)
	s.document("didClose", file.uri, "")
	if len(s.diagnostics[file.uri]) != 0 || len(s.diagnostics[uri]) != 0 {
		t.Fatalf("closing did not restore updated disk contents: %+v", s.diagnostics)
	}
}

func TestCLICounselCustomAliasOverlay(t *testing.T) {
	broken := strings.Replace(editorStatute, "n TIMES 2", "THE FINDING OF absent REGARDING n", 1)
	file := counselFixtureFile(t, "Dependency with spaces.trial", broken)
	alias := strings.Replace(file.uri, "Dependency", "%44ependency", 1)
	if runtime.GOOS == "windows" {
		alias = strings.ToLower(alias)
	}
	s := startCounselProcess(t, "--enact", file.path)
	defer s.finish()
	const uri = "file:///alias-imports.trial"
	s.document("didOpen", uri, editorCase)
	if got := s.diagnostics[file.uri]; len(got) == 0 || !strings.Contains(got[0].Message, "absent") {
		t.Fatalf("missing initial dependency diagnostic: %+v", s.diagnostics)
	}
	s.document("didOpen", alias, editorStatute)
	for owner, diagnostics := range s.diagnostics {
		if len(diagnostics) != 0 {
			t.Fatalf("configured alias overlay left diagnostics at %s: %+v", owner, diagnostics)
		}
	}
	s.document("didClose", alias, "")
	if got := s.diagnostics[file.uri]; len(got) == 0 || !strings.Contains(got[0].Message, "absent") || len(s.diagnostics[alias]) != 0 {
		t.Fatalf("closing alias did not reload the configured path and clear overlay: %+v", s.diagnostics)
	}
}

func TestCLICounselConfiguredSourceErrors(t *testing.T) {
	for _, source := range []string{"not valid source", editorCase} {
		t.Run(source, func(t *testing.T) {
			file := counselFixtureFile(t, "invalid.trial", source)
			s := startCounselProcess(t, "--enact", file.path)
			defer s.finish()
			s.document("didOpen", "file:///imports-invalid.trial", editorCase)
			if len(s.diagnostics[file.uri]) == 0 {
				t.Fatalf("configured syntax/form error must reach its URI: %+v", s.diagnostics)
			}
		})
	}
}

func TestCLICounselEnactDependencyOrder(t *testing.T) {
	foundation := counselFixtureFile(t, "foundation.trial", `FORM S-1.
IN THE MATTER OF: foundation.
THE OFFICE OF base, CONCERNING n.
REMAND WITH n TIMES 2.
`)
	custom := counselFixtureFile(t, "custom.trial", `FORM S-1.
IN THE MATTER OF: custom.
INCORPORATE BY REFERENCE foundation.
THE OFFICE OF twice, CONCERNING n.
REMAND WITH THE FINDING OF base REGARDING n.
`)
	for _, reverse := range []bool{false, true} {
		t.Run(fmt.Sprintf("reverse=%t", reverse), func(t *testing.T) {
			first, second := foundation, custom
			if reverse {
				first, second = second, first
			}
			s := startCounselProcess(t, "--enact", first.path, "--enact", second.path)
			defer s.finish()
			s.document("didOpen", "file:///ordered.trial", editorCase)
			if reverse {
				if got := s.diagnostics[custom.uri]; len(got) == 0 || !strings.Contains(got[0].Message, "foundation") {
					t.Fatalf("wrong dependency order did not fail at custom source: %+v", s.diagnostics)
				}
			} else {
				for owner, diagnostics := range s.diagnostics {
					if len(diagnostics) != 0 {
						t.Fatalf("ordered imports fail at %s: %+v", owner, diagnostics)
					}
				}
			}
		})
	}
}

func TestCLICounselEnactArgumentsAndReads(t *testing.T) {
	file := counselFixtureFile(t, "valid.trial", editorStatute)
	oversized := counselFixtureFile(t, "large.trial", strings.Repeat("x", maxSourceBytes+1))
	alias := filepath.Join(filepath.Dir(file.path), "sub") + string(filepath.Separator) + ".." + string(filepath.Separator) + filepath.Base(file.path)
	tooMany := []string{"counsel"}
	for i := range maxRunEnactments + 1 {
		tooMany = append(tooMany, "--enact", fmt.Sprintf("statute-%d.trial", i))
	}
	for _, tc := range []struct {
		name       string
		args       []string
		code       int
		diagnostic string
	}{
		{"stdin", []string{"counsel", "--enact", "-"}, 2, "standard input is reserved"},
		{"empty", []string{"counsel", "--enact="}, 2, "file path"},
		{"duplicate", []string{"counsel", "--enact", file.path, "--enact", alias}, 2, "more than once"},
		{"count", tooMany, 2, "at most 100"},
		{"missing", []string{"counsel", "--enact", filepath.Join(t.TempDir(), "absent.trial")}, 1, "read statute"},
		{"directory", []string{"counsel", "--enact", t.TempDir()}, 1, "read statute"},
		{"oversized", []string{"counsel", "--enact", oversized.path}, 1, "source limit"},
		{"broker", []string{"counsel", "--broker=localhost:9092"}, 2, "flag provided but not defined"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := cliProcess(t, tc.args...)
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			if err := cmd.Run(); err == nil || cmd.ProcessState.ExitCode() != tc.code || stdout.Len() != 0 || !strings.Contains(stderr.String(), tc.diagnostic) {
				t.Fatalf("exit=%v stdout=%q stderr=%q; want %d and %q", err, &stdout, &stderr, tc.code, tc.diagnostic)
			}
		})
	}
}
