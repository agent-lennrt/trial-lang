package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

type localMCPSession struct {
	t      *testing.T
	cmd    *exec.Cmd
	in     io.WriteCloser
	enc    *json.Encoder
	dec    *json.Decoder
	stderr bytes.Buffer
	id     int
	done   bool
}

func startLocalMCP(t *testing.T) *localMCPSession {
	t.Helper()
	cmd := cliProcess(t, "mcp", "--local")
	// Invalid even before dialing: local mode must ignore this environment
	// value rather than depending on a broker or network access in the test.
	cmd.Env = append(cmd.Env, "TRIAL_BROKER=,")
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		_ = in.Close()
		t.Fatal(err)
	}
	s := &localMCPSession{t: t, cmd: cmd, in: in, enc: json.NewEncoder(in), dec: json.NewDecoder(out)}
	cmd.Stderr = &s.stderr
	if err := cmd.Start(); err != nil {
		_ = in.Close()
		_ = out.Close()
		t.Fatal(err)
	}
	t.Cleanup(s.stop)
	return s
}

func (s *localMCPSession) stop() {
	_ = s.in.Close()
	if !s.done {
		_ = s.cmd.Process.Kill()
		_ = s.cmd.Wait()
		s.done = true
	}
}

func (s *localMCPSession) rpc(method string, params any) json.RawMessage {
	s.t.Helper()
	s.id++
	if err := s.enc.Encode(map[string]any{"jsonrpc": "2.0", "id": s.id, "method": method, "params": params}); err != nil {
		s.stop()
		s.t.Fatalf("send %s: %v; stderr=%s", method, err, &s.stderr)
	}
	var response struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      int             `json:"id"`
		Result  json.RawMessage `json:"result"`
		Error   json.RawMessage `json:"error"`
	}
	if err := s.dec.Decode(&response); err != nil {
		s.stop()
		s.t.Fatalf("read %s: %v; stderr=%s", method, err, &s.stderr)
	}
	if response.JSONRPC != "2.0" || response.ID != s.id || len(response.Error) != 0 || len(response.Result) == 0 {
		s.t.Fatalf("response to %s = %+v", method, response)
	}
	return response.Result
}

func (s *localMCPSession) call(name string, args any) (map[string]any, bool) {
	s.t.Helper()
	params := map[string]any{"name": name}
	if args != nil {
		params["arguments"] = args
	}
	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(s.rpc("tools/call", params), &result); err != nil {
		s.t.Fatal(err)
	}
	if len(result.Content) != 1 || result.Content[0].Type != "text" {
		s.t.Fatalf("tool %s content = %+v", name, result)
	}
	if result.IsError {
		return map[string]any{"error": result.Content[0].Text}, true
	}
	var value map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].Text), &value); err != nil {
		s.t.Fatalf("tool %s result: %v", name, err)
	}
	return value, false
}

func (s *localMCPSession) initialize() {
	s.t.Helper()
	var result struct {
		Instructions string `json:"instructions"`
	}
	if err := json.Unmarshal(s.rpc("initialize", map[string]any{"protocolVersion": "2025-06-18"}), &result); err != nil {
		s.t.Fatal(err)
	}
	for _, required := range []string{"temporary in-memory", "disappear", "new process", "isolated from session"} {
		if !strings.Contains(result.Instructions, required) {
			s.t.Fatalf("local initialization omits %q: %q", required, result.Instructions)
		}
	}
}

func (s *localMCPSession) close() {
	s.t.Helper()
	if err := s.in.Close(); err != nil {
		s.t.Fatal(err)
	}
	err := s.cmd.Wait()
	s.done = true
	if err != nil {
		s.t.Fatalf("EOF exit: %v; stderr=%s", err, &s.stderr)
	}
}

func TestCLILocalMCPWorkflow(t *testing.T) {
	s := startLocalMCP(t)
	s.initialize()
	var listed struct {
		Tools []struct {
			Name        string `json:"name"`
			InputSchema struct {
				Properties map[string]any `json:"properties"`
			} `json:"inputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(s.rpc("tools/list", map[string]any{}), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Tools) != 12 {
		t.Fatalf("local server advertised %d tools, want the full 12", len(listed.Tools))
	}
	foundTest := false
	for _, tool := range listed.Tools {
		if tool.Name == "trial_test" {
			foundTest = tool.InputSchema.Properties["enactments"] != nil
		}
	}
	if !foundTest {
		t.Fatal("inline enactment schema is missing from local discovery")
	}

	const statute = "FORM S-1. IN THE MATTER OF: doubling. THE OFFICE OF twice, CONCERNING n. REMAND WITH n TIMES 2."
	const program = "FORM K-1. IN THE MATTER OF: local. INCORPORATE BY REFERENCE doubling. ARTICLE 1. AWAIT SUMMONS, FILED UNDER n. PROCLAIM THE FINDING OF twice REGARDING n. ADJOURN INDEFINITELY."
	const deposition = "DEPOSITION OF: local.trial.\nENACT: doubling.trial.\nSERVE: 21.\nEXPECT PROCLAMATION: 42.\nEXPECT ADJOURNMENT.\nALLOW 2 COURT DAYS."
	result, failed := s.call("trial_test", map[string]any{
		"program_source": program, "deposition_source": deposition,
		"enactments": map[string]string{"doubling.trial": statute},
	})
	if failed || result["consistent"] != true {
		t.Fatalf("local inline test: %v", result)
	}
	assertEmptyLocalMCP(t, s)
	if result, failed = s.call("trial_enact", map[string]any{"source": statute}); failed || result["statute"] != "doubling" {
		t.Fatalf("local enactment: %v", result)
	}
	if result, failed = s.call("trial_file", map[string]any{"source": program}); failed {
		t.Fatalf("local filing: %v", result)
	}
	caseID, ok := result["case"].(string)
	if !ok || caseID == "" {
		t.Fatalf("local case identifier: %v", result)
	}
	if result, failed = s.call("trial_serve", map[string]any{"case": caseID, "values": []any{21}}); failed {
		t.Fatalf("local service: %v", result)
	}
	if result, failed = s.call("trial_proceed", map[string]any{"case": caseID, "for_at_most_court_days": 2}); failed || result["outcome"] != "adjourned indefinitely" || result["session_expired"] != false {
		t.Fatalf("local execution: %v", result)
	}
	result, failed = s.call("trial_observe", map[string]any{"case": caseID})
	if failed {
		t.Fatalf("local observation: %v", result)
	}
	proclamations, ok := result["proclamations"].([]any)
	if !ok || len(proclamations) != 1 || proclamations[0].(map[string]any)["text"] != "42" {
		t.Fatalf("local output: %v", result)
	}
	// Even after an enactment, trial_test must supply its own dependencies.
	result, failed = s.call("trial_test", map[string]any{
		"program_source": program,
		"deposition_source": "DEPOSITION OF: local.trial.\nSERVE: 21.\n" +
			"EXPECT PROCLAMATION: 42.\nEXPECT ADJOURNMENT.\nALLOW 2 COURT DAYS.",
	})
	if failed || result["consistent"] != false {
		t.Fatalf("trial_test borrowed a session statute: %v", result)
	}
	s.close()

	next := startLocalMCP(t)
	next.initialize()
	assertEmptyLocalMCP(t, next)
	if result, failed = next.call("trial_status", map[string]any{"case": caseID}); !failed {
		t.Fatalf("a new process retained the old case: %v", result)
	}
	next.close()
}

func assertEmptyLocalMCP(t *testing.T, s *localMCPSession) {
	t.Helper()
	for tool, field := range map[string]string{"trial_docket": "matters", "trial_statutes": "statutes"} {
		result, failed := s.call(tool, nil)
		value, exists := result[field]
		entries, list := value.([]any)
		// Existing tools encode an empty docket as [] and empty statutes as
		// null. Preserve both wire contracts while requiring an empty result.
		if failed || !exists || (value != nil && (!list || len(entries) != 0)) {
			t.Fatalf("local %s is not empty: %v", tool, result)
		}
	}
}

func TestCLILocalMCPArguments(t *testing.T) {
	for _, args := range [][]string{
		{"--local", "--broker=,"},
		{"--broker", ",", "--local"},
		{"--local=false", "--broker=,", "--local=true"},
	} {
		cmd := cliProcess(t, append([]string{"mcp"}, args...)...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		if err := cmd.Run(); err == nil || cmd.ProcessState.ExitCode() != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "cannot be combined") {
			t.Fatalf("args=%q exit=%v stdout=%q stderr=%q", args, err, &stdout, &stderr)
		}
	}
	// Explicit false retains normal Kafka setup, even after an earlier true.
	cmd := cliProcess(t, "mcp", "--local", "--local=false", "--broker=,")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err == nil || cmd.ProcessState.ExitCode() != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "empty address") {
		t.Fatalf("disabled local mode: exit=%v stdout=%q stderr=%q", err, &stdout, &stderr)
	}
}

func TestCLILocalMCPSignal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix process signals")
	}
	for _, signal := range []os.Signal{os.Interrupt, syscall.SIGTERM} {
		for _, partial := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/partial=%t", signal, partial), func(t *testing.T) {
				s := startLocalMCP(t)
				s.initialize() // Readiness canary: signal handling is installed.
				if partial {
					if _, err := io.WriteString(s.in, "{"); err != nil {
						t.Fatal(err)
					}
				}
				if err := s.cmd.Process.Signal(signal); err != nil {
					t.Fatal(err)
				}
				err := s.cmd.Wait()
				s.done = true
				if err == nil || s.cmd.ProcessState.ExitCode() != 1 {
					t.Fatalf("signal exit=%v state=%v stderr=%s", err, s.cmd.ProcessState, &s.stderr)
				}
			})
		}
	}
}
