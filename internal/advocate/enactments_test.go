package advocate

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/lennrt/trial-lang/internal/deposition"
	"github.com/lennrt/trial-lang/internal/docket"
)

// Every promoted method panics because the embedded log is nil. An isolated
// deposition must never consult any method of the server's persistent log.
type inaccessibleTestLog struct{ docket.Log }

func TestTrialTestInlineEnactments(t *testing.T) {
	const base = `FORM S-1.
IN THE MATTER OF: base.
THE OFFICE OF doubling, CONCERNING n.
    REMAND WITH n TIMES 2.
`
	const importing = `FORM S-1.
IN THE MATTER OF: importing.
INCORPORATE BY REFERENCE base.
THE OFFICE OF quadrupling, CONCERNING n.
    REMAND WITH THE FINDING OF doubling REGARDING n TIMES 2.
`
	const program = `FORM K-1.
IN THE MATTER OF: inline-dependencies.
INCORPORATE BY REFERENCE importing.
ARTICLE 1.
    PROCLAIM THE FINDING OF quadrupling REGARDING 21.
    ADJOURN INDEFINITELY.
`
	for _, tc := range []struct {
		name         string
		declarations string
		consistent   bool
	}{
		{"dependency order", "ENACT: z/base.trial.\nENACT: a/importing.trial.\n", true},
		{"repeated label", "ENACT: z/base.trial.\nENACT: z/base.trial.\nENACT: a/importing.trial.\n", true},
		{"wrong dependency order", "ENACT: a/importing.trial.\nENACT: z/base.trial.\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args, err := json.Marshal(map[string]any{
				"program_source": program,
				"deposition_source": "DEPOSITION OF: inline.trial.\n" + tc.declarations +
					"EXPECT PROCLAMATION: 84.\nEXPECT ADJOURNMENT.\nALLOW 2 COURT DAYS.\n",
				"enactments": map[string]string{
					"z/base.trial":      base,
					"a/importing.trial": importing,
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			s := &Server{Log: inaccessibleTestLog{}}
			result := s.call(t.Context(), "trial_test", args)
			if result.IsError {
				t.Fatalf("inline dependency input failed: %s", result.Content[0].Text)
			}
			var payload struct {
				Consistent     bool     `json:"consistent"`
				Contradictions []string `json:"contradictions"`
			}
			if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
				t.Fatal(err)
			}
			if payload.Consistent != tc.consistent {
				t.Fatalf("consistent = %v, want %v: %v", payload.Consistent, tc.consistent, payload.Contradictions)
			}
			if !tc.consistent && !strings.Contains(strings.Join(payload.Contradictions, "\n"), "enact statute") {
				t.Fatalf("wrong order did not fail during enactment: %v", payload.Contradictions)
			}
		})
	}
}

func TestTrialTestEnactmentArguments(t *testing.T) {
	const label = "statute.trial"
	const source = "FORM S-1. IN THE MATTER OF: statute. THE OFFICE OF answer. REMAND WITH 42."
	const program = "FORM K-1. IN THE MATTER OF: test. ARTICLE 1. ADJOURN INDEFINITELY."
	const declaration = "ENACT: " + label + ".\n"
	tooMany := make(map[string]string)
	for i := range maxToolEnactments + 1 {
		tooMany[fmt.Sprintf("statute-%d.trial", i)] = source
	}
	for _, tc := range []struct {
		name         string
		declarations string
		enactments   any
		wantError    string
	}{
		{"no dependencies", "", map[string]string{}, ""},
		{"missing source", declaration, map[string]string{}, "has no inline source"},
		{"unused source", "", map[string]string{label: source}, "has no ENACT declaration"},
		{"exact labels", declaration, map[string]string{"./" + label: source}, "has no ENACT declaration"},
		{"null object", declaration, nil, "must not be null"},
		{"array instead of object", declaration, []string{source}, "arguments could not be read"},
		{"string instead of object", declaration, source, "arguments could not be read"},
		{"null source", declaration, map[string]any{label: nil}, "must be a string, not null"},
		{"numeric source", declaration, map[string]any{label: 42}, "arguments could not be read"},
		{"boolean source", declaration, map[string]any{label: true}, "arguments could not be read"},
		{"array source", declaration, map[string]any{label: []string{source}}, "arguments could not be read"},
		{"object source", declaration, map[string]any{label: map[string]string{"source": source}}, "arguments could not be read"},
		{"too many sources", declaration, tooMany, "at most 100 inline statutes"},
		{"oversized source", declaration, map[string]string{label: strings.Repeat("x", maxSourceBytes+1)}, "exceeds the 4194304-byte limit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args, err := json.Marshal(map[string]any{
				"program_source":    program,
				"deposition_source": "DEPOSITION OF: test.trial.\n" + tc.declarations + "EXPECT ADJOURNMENT.\nALLOW 2 COURT DAYS.",
				"enactments":        tc.enactments,
			})
			if err != nil {
				t.Fatal(err)
			}
			result := (&Server{Log: inaccessibleTestLog{}}).call(t.Context(), "trial_test", args)
			text := result.Content[0].Text
			if tc.wantError == "" {
				if result.IsError || !strings.Contains(text, `"consistent": true`) {
					t.Fatalf("test without dependencies failed: %s", text)
				}
			} else if !result.IsError || !strings.Contains(text, tc.wantError) {
				t.Fatalf("result = %+v, want tool error containing %q", result, tc.wantError)
			}
		})
	}
}

func TestTrialTestInvalidStatuteIsContradiction(t *testing.T) {
	for _, source := range []string{"", "not a statute", "FORM K-1. IN THE MATTER OF: wrong-form. ARTICLE 1. ADJOURN INDEFINITELY."} {
		t.Run(source, func(t *testing.T) {
			args, err := json.Marshal(map[string]any{
				"program_source":    "", // A rejection expectation cannot conceal an invalid dependency.
				"deposition_source": "DEPOSITION OF: test.trial.\nENACT: statute.trial.\nEXPECT REJECTION.\n",
				"enactments":        map[string]string{"statute.trial": source},
			})
			if err != nil {
				t.Fatal(err)
			}
			result := (&Server{Log: inaccessibleTestLog{}}).call(t.Context(), "trial_test", args)
			text := result.Content[0].Text
			if result.IsError || !strings.Contains(text, `"consistent": false`) || !strings.Contains(text, "enact statute") {
				t.Fatalf("invalid statute was not an ordinary contradiction: %+v", result)
			}
		})
	}
}

func TestInlineEnactmentBounds(t *testing.T) {
	// Test source preparation without allocating giant JSON requests or
	// compiling 256 MiB of fixtures. Shared strings still count for each ENACT.
	source := strings.Repeat("x", maxSourceBytes)
	dep := &deposition.Deposition{Enacts: []string{"source"}}
	if err := setTestEnactments(dep, map[string]*string{"source": &source}); err != nil {
		t.Fatalf("source exactly at the byte limit: %v", err)
	}
	if len(dep.EnactSources) != 1 || dep.EnactSources[0] != source {
		t.Fatal("source exactly at the limit was not retained")
	}
	dep.Enacts = make([]string, docket.MaxReadBytes/maxSourceBytes)
	for i := range dep.Enacts {
		dep.Enacts[i] = "source"
	}
	if err := setTestEnactments(dep, map[string]*string{"source": &source}); err != nil {
		t.Fatalf("sequence exactly at the byte limit: %v", err)
	}
	if len(dep.EnactSources) != len(dep.Enacts) {
		t.Fatalf("repeated sources = %d, want %d", len(dep.EnactSources), len(dep.Enacts))
	}
	dep.Enacts = append(dep.Enacts, "source")
	if err := setTestEnactments(dep, map[string]*string{"source": &source}); err == nil || !strings.Contains(err.Error(), "enacted sources exceed") {
		t.Fatalf("oversized repeated sequence: %v", err)
	}
	if len(dep.EnactSources) != len(dep.Enacts)-1 {
		t.Fatal("invalid input replaced the previously resolved sources")
	}

	dep = &deposition.Deposition{}
	sources := make(map[string]*string)
	empty := ""
	for i := range maxToolEnactments {
		name := fmt.Sprintf("statute-%d.trial", i)
		dep.Enacts = append(dep.Enacts, name)
		sources[name] = &empty
	}
	if err := setTestEnactments(dep, sources); err != nil || len(dep.EnactSources) != maxToolEnactments {
		t.Fatalf("exact source count limit: %v, loaded %d", err, len(dep.EnactSources))
	}
}

func TestInlineEnactmentSchema(t *testing.T) {
	s := open(t)
	listed := s.rpc("tools/list", nil)["tools"].([]any)
	for _, entry := range listed {
		tool := entry.(map[string]any)
		if tool["name"] != "trial_test" {
			continue
		}
		schema := tool["inputSchema"].(map[string]any)
		required := schema["required"].([]any)
		if len(required) != 2 || required[0] != "program_source" || required[1] != "deposition_source" {
			t.Fatalf("required source fields changed: %v", required)
		}
		properties := schema["properties"].(map[string]any)
		enactments, ok := properties["enactments"].(map[string]any)
		if !ok || enactments["type"] != "object" || enactments["maxProperties"] != float64(maxToolEnactments) {
			t.Fatalf("inline enactments schema = %v", enactments)
		}
		values := enactments["additionalProperties"].(map[string]any)
		if values["type"] != "string" {
			t.Fatalf("inline source schema = %v", values)
		}
		return
	}
	t.Fatal("trial_test was not advertised")
}
