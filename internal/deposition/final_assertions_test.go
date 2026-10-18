package deposition

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestDepositionFinalAssertions(t *testing.T) {
	const source = `FORM K-1.
IN THE MATTER OF: final-assertions.
ARTICLE 1.
    LET IT BE RECORDED THAT answer IS 7.
    PROCLAIM "ready".
    ADJOURN INDEFINITELY.
`
	verdictSource := strings.Replace(source, "ADJOURN INDEFINITELY.", `HOLD "the witness" IN CONTEMPT.`, 1)
	for _, tc := range []struct {
		name          string
		verdict       bool
		expectations  string
		contradiction string
	}{
		{"output only", false, "EXPECT PROCLAMATION: ready.\nEXPECT ADJOURNMENT.", ""},
		{"unexpected output", false, "EXPECT ADJOURNMENT.", `proclamation 1: unexpected "ready"`},
		{"wrong output", false, "EXPECT PROCLAMATION: wrong.", `proclamation 1: got "ready"; want "wrong"`},
		{"missing output", false, "EXPECT PROCLAMATION: ready.\nEXPECT PROCLAMATION: missing.", `proclamation 2: missing "missing"`},
		{"wrong outcome", false, "EXPECT PROCLAMATION: ready.\nEXPECT VERDICT.", "the case ended in adjournment; the deposition expected verdict"},
		{"unexpected verdict", true, "EXPECT PROCLAMATION: ready.", "the case ended in a verdict the deposition did not expect"},
		{"verdict without citation", true, "EXPECT PROCLAMATION: ready.\nEXPECT VERDICT.", ""},
		{"record only", false, "EXPECT PROCLAMATION: ready.\nEXPECT RECORD answer: 7.", ""},
		{"wrong record", false, "EXPECT PROCLAMATION: ready.\nEXPECT RECORD answer: 8.", `the record "answer" reads 7; the deposition expected 8`},
		{"missing record", false, "EXPECT PROCLAMATION: ready.\nEXPECT RECORD missing: 7.", `record "missing" is missing`},
		{"citation only", true, "EXPECT PROCLAMATION: ready.\nEXPECT VERDICT CITING the witness.", ""},
		{"wrong citation", true, "EXPECT PROCLAMATION: ready.\nEXPECT VERDICT CITING another witness.", `the deposition expected it to cite "another witness"`},
		{"citation without verdict", false, "EXPECT PROCLAMATION: ready.\nEXPECT VERDICT CITING the witness.", "no verdict was reached"},
		{"record and citation", true, "EXPECT PROCLAMATION: ready.\nEXPECT RECORD answer: 7.\nEXPECT VERDICT CITING the witness.", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, err := Parse("DEPOSITION OF: final-assertions.trial.\n" + tc.expectations)
			if err != nil {
				t.Fatal(err)
			}
			program := source
			if tc.verdict {
				program = verdictSource
			}
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			result := Run(ctx, program, d)
			if !slices.Equal(result.Said, []string{"ready"}) {
				t.Fatalf("observed output = %q, want [ready]", result.Said)
			}
			if tc.contradiction == "" {
				if !result.OK() {
					t.Fatalf("unexpected contradictions: %q", result.Contradictions)
				}
			} else if result.OK() || !strings.Contains(strings.Join(result.Contradictions, "\n"), tc.contradiction) {
				t.Fatalf("contradictions = %q, want %q", result.Contradictions, tc.contradiction)
			}
		})
	}
}

func TestDepositionOutputOnlyCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	result := Run(ctx, countdown, &Deposition{AllowDays: 1})
	if result.OK() || !strings.Contains(strings.Join(result.Contradictions, "\n"), "context canceled") {
		t.Fatalf("canceled run must fail: %q", result.Contradictions)
	}
}

// Keep a nontrivial execution history: output assertions need its proclamation,
// but do not need to replay every intermediate record, stack, or call frame.
func BenchmarkDepositionOutputOnly(b *testing.B) {
	const source = `FORM K-1.
IN THE MATTER OF: output-only.
ARTICLE 1.
    LET IT BE RECORDED THAT n IS 256.
ARTICLE 2.
    LET IT BE RECORDED THAT n IS n LESS 1.
    SHOULD n EXCEED 0, REFER TO ARTICLE 2.
    PROCLAIM n.
    ADJOURN INDEFINITELY.
`
	d := &Deposition{AllowDays: 15, Outcome: "adjournment", Proclamations: []string{"0"}}
	b.ReportAllocs()
	for b.Loop() {
		result := Run(b.Context(), source, d)
		if !result.OK() {
			b.Fatalf("deposition failed: %q", result.Contradictions)
		}
	}
}
