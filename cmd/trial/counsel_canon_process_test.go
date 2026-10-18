package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"
)

func TestCLICounselCanonDiagnostics(t *testing.T) {
	const source = "FORM K-1.\nIN THE MATTER OF: editor.\nINCORPORATE BY REFERENCE statutes-of-arithmetic.\nARTICLE 1.\nREFER TO ARTICLE 99.\n"
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("canon=%t", enabled), func(t *testing.T) {
			args := []string{"counsel"}
			if enabled {
				args = append(args, "--canon")
			}
			cmd := cliProcess(t, args...)
			cmd.Env = append(cmd.Env, "TRIAL_BROKER=,")
			var input, output, diagnostics bytes.Buffer
			for _, request := range []map[string]any{
				{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}},
				{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{
					"textDocument": map[string]any{"uri": "file:///editor.trial", "text": source, "version": 1},
				}},
				{"jsonrpc": "2.0", "method": "textDocument/didChange", "params": map[string]any{
					"textDocument":   map[string]any{"uri": "file:///editor.trial", "version": 2},
					"contentChanges": []map[string]any{{"text": strings.Replace(source, "ARTICLE 99", "ARTICLE 1", 1)}},
				}},
			} {
				body, err := json.Marshal(request)
				if err != nil {
					t.Fatal(err)
				}
				fmt.Fprintf(&input, "Content-Length: %d\r\n\r\n%s", len(body), body)
			}
			cmd.Stdin, cmd.Stdout, cmd.Stderr = &input, &output, &diagnostics
			if err := cmd.Run(); err != nil {
				t.Fatalf("counsel failed: %v stderr=%s", err, &diagnostics)
			}
			reader := bufio.NewReader(&output)
			for i := range 3 {
				header, err := reader.ReadString('\n')
				if err != nil {
					t.Fatal(err)
				}
				length, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(header, "Content-Length:")))
				if err != nil || length < 1 || length > 1<<20 {
					t.Fatalf("invalid frame header %q: %v", header, err)
				}
				if blank, err := reader.ReadString('\n'); err != nil || blank != "\r\n" {
					t.Fatalf("frame separator %q: %v", blank, err)
				}
				body := make([]byte, length)
				if _, err := io.ReadFull(reader, body); err != nil {
					t.Fatal(err)
				}
				var reply struct {
					Result json.RawMessage `json:"result"`
					Method string          `json:"method"`
					Params struct {
						Diagnostics []struct {
							Message string `json:"message"`
							Range   struct {
								Start struct{ Line, Character int } `json:"start"`
							} `json:"range"`
						} `json:"diagnostics"`
					} `json:"params"`
				}
				if err := json.Unmarshal(body, &reply); err != nil {
					t.Fatal(err)
				}
				if i == 0 {
					if len(reply.Result) == 0 {
						t.Fatalf("initialization failed: %s", body)
					}
					continue
				}
				if reply.Method != "textDocument/publishDiagnostics" {
					t.Fatalf("expected diagnostics: %s", body)
				}
				got := reply.Params.Diagnostics
				if enabled && i == 1 {
					if len(got) != 1 || !strings.Contains(got[0].Message, "ARTICLE 99") || got[0].Range.Start.Line != 4 || got[0].Range.Start.Character != 0 {
						t.Fatalf("missing article diagnostics: %s", body)
					}
				} else if len(got) != 0 {
					t.Fatalf("unexpected diagnostics: %s", body)
				}
			}
			if extra, err := io.ReadAll(reader); err != nil || len(extra) != 0 {
				t.Fatalf("unexpected trailing protocol output %q: %v", extra, err)
			}
		})
	}
}
