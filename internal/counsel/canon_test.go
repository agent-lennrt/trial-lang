package counsel

import (
	"bufio"
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func canonProgram(body string) string {
	return "FORM K-1.\nIN THE MATTER OF: editor.\nINCORPORATE BY REFERENCE statutes-of-arithmetic.\nARTICLE 1.\n" + body
}

func TestCanonDiagnostics(t *testing.T) {
	valid := canonProgram("PROCLAIM THE FINDING OF maximum REGARDING 1 AND 2.")
	for _, tc := range []struct {
		name   string
		source string
	}{
		{"imported office", valid},
		{"transitive office", strings.Replace(valid, "statutes-of-arithmetic", "statutes-of-schedules", 1)},
		{"multiple imports", strings.Replace(valid, "ARTICLE 1.", "INCORPORATE BY REFERENCE statutes-of-schedules.\nARTICLE 1.", 1)},
		{"statute", "FORM S-1.\nIN THE MATTER OF: custom.\nINCORPORATE BY REFERENCE statutes-of-arithmetic.\nTHE OFFICE OF greatest, CONCERNING n.\nREMAND WITH THE FINDING OF maximum REGARDING n AND 0."},
		{"no execution", canonProgram("REFER TO ARTICLE 1.")},
		{"unknown import fallback", strings.Replace(canonProgram("REFER TO ARTICLE 99."), "statutes-of-arithmetic", "custom", 1)},
		{"mixed import fallback", strings.Replace(canonProgram("REFER TO ARTICLE 99."), "ARTICLE 1.", "INCORPORATE BY REFERENCE custom.\nARTICLE 1.", 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if diagnostics := diagnose(tc.source, true); len(diagnostics) != 0 {
				t.Fatalf("unexpected diagnostics: %v", diagnostics)
			}
		})
	}
}

func TestCanonDiagnosticLocations(t *testing.T) {
	for _, tc := range []struct {
		name          string
		source        string
		message       string
		start, end    position
		documentLevel bool
	}{
		{"local referral", canonProgram("REFER TO ARTICLE 99."), "ARTICLE 99", position{Line: 4}, position{Line: 4, Character: 1}, false},
		{"imported arity", canonProgram("PROCLAIM THE FINDING OF maximum REGARDING 1."), "expects 2 argument(s); received 1", position{}, position{}, true},
		{"imported collision", canonProgram("ADJOURN INDEFINITELY.\nTHE OFFICE OF maximum, CONCERNING a AND b.\nREMAND WITH a."), "established more than once", position{}, position{}, true},
		{"imported concern conflict", "FORM K-1.\nIN THE MATTER OF: editor.\nINCORPORATE BY REFERENCE statutes-of-arithmetic.\nHEREINAFTER, n SHALL MEAN 1.\nARTICLE 1.\nADJOURN INDEFINITELY.", "uses defined term", position{}, position{}, true},
		{"UTF16 parse range", canonProgram("PROCLAIM \"😀\" EXTRA."), "expected", position{Line: 4, Character: 14}, position{Line: 4, Character: 15}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			diagnostics := diagnose(tc.source, true)
			if len(diagnostics) != 1 || !strings.Contains(diagnostics[0]["message"].(string), tc.message) {
				t.Fatalf("expected %q: %v", tc.message, diagnostics)
			}
			rangeValue := diagnostics[0]["range"].(map[string]any)
			if rangeValue["start"] != tc.start || rangeValue["end"] != tc.end {
				t.Fatalf("range = %v, want %+v..%+v", rangeValue, tc.start, tc.end)
			}
			contextual := strings.HasPrefix(diagnostics[0]["message"].(string), "While checking bundled canon imports:")
			if contextual != tc.documentLevel {
				t.Fatalf("document-level context = %t, want %t: %v", contextual, tc.documentLevel, diagnostics)
			}
		})
	}
}

func TestCanonKeepsDefaultAndParseDiagnostics(t *testing.T) {
	if diagnostics := Diagnose(canonProgram("REFER TO ARTICLE 99.")); len(diagnostics) != 0 {
		t.Fatalf("default diagnostics started compiling imports: %v", diagnostics)
	}
	for _, source := range []string{
		"FORM K-1.\nIN THE MATTER OF: editor.\nARTICLE 1.\nREFER TO ARTICLE 99.",
		"FORM K-2.\nIN THE MATTER OF: editor.\nARTICLE 2.\nREFER TO ARTICLE 99.",
		canonProgram("PROCLAIM \"😀\" EXTRA."),
		strings.Replace(canonProgram("PROCLAIM ."), "statutes-of-arithmetic", "custom", 1),
		strings.Replace(canonProgram("ADJOURN INDEFINITELY."), "FORM K-1", "FORM K-2", 1),
	} {
		if ordinary, enabled := Diagnose(source), diagnose(source, true); !reflect.DeepEqual(ordinary, enabled) {
			t.Fatalf("--canon changed non-import compilation or parse error: default=%v enabled=%v", ordinary, enabled)
		}
	}
}

func TestCanonDocumentChecksDoNotShareStatutes(t *testing.T) {
	// A checked S-1 document must not replace the canon in the next check or
	// turn an unknown custom import into a resolved editor dependency.
	sources := []string{
		"FORM S-1.\nIN THE MATTER OF: statutes-of-arithmetic.\nINCORPORATE BY REFERENCE statutes-of-strings.\nTHE OFFICE OF maximum, CONCERNING n.\nREMAND WITH n.",
		canonProgram("PROCLAIM THE FINDING OF maximum REGARDING 1 AND 2."),
		"FORM S-1.\nIN THE MATTER OF: custom.\nINCORPORATE BY REFERENCE statutes-of-arithmetic.\nTHE OFFICE OF local.\nREMAND WITH 1.",
		strings.Replace(canonProgram("REFER TO ARTICLE 99."), "statutes-of-arithmetic", "custom", 1),
	}
	var in, out bytes.Buffer
	for _, source := range sources {
		in.WriteString(frame(map[string]any{
			"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{
				"textDocument": map[string]any{"uri": "file:///current.trial", "text": source},
			},
		}))
	}
	s := &Server{Canon: true, In: &in, Out: &out}
	if err := s.Serve(t.Context()); err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(&out)
	for range sources {
		body, err := readMessage(reader)
		if err != nil {
			t.Fatal(err)
		}
		var reply struct {
			Params struct {
				Diagnostics []any `json:"diagnostics"`
			} `json:"params"`
		}
		if err := json.Unmarshal(body, &reply); err != nil || len(reply.Params.Diagnostics) != 0 {
			t.Fatalf("state leaked between checks: %s (%v)", body, err)
		}
	}
}

func BenchmarkDiagnoseCanon(b *testing.B) {
	source := strings.Replace(canonProgram("PROCLAIM THE FINDING OF maximum REGARDING 1 AND 2."), "statutes-of-arithmetic", "statutes-of-schedules", 1)
	b.ReportAllocs()
	for b.Loop() {
		if diagnostics := diagnose(source, true); len(diagnostics) != 0 {
			b.Fatal(diagnostics)
		}
	}
}
