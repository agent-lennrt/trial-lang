package court

import (
	"errors"
	"strings"
	"testing"

	"github.com/lennrt/trial-lang/internal/docket"
	"github.com/lennrt/trial-lang/internal/gregor"
)

func TestImportedSourceProvenance(t *testing.T) {
	const statute = "FORM S-1.\nIN THE MATTER OF: dependency.\nTHE OFFICE OF helper, CONCERNING n.\nSHOULD n EXCEED 0, LET IT BE RECORDED THAT marked IS 1.\nREMAND WITH n."
	for _, tc := range []struct {
		name, declarations, body, offices, source, message string
		line                                               int
	}{
		{"duplicate office", "", "ADJOURN INDEFINITELY.", "\nTHE OFFICE OF helper. REMAND.", "dependency", "established more than once", 3},
		{"nested imported statement", "HEREINAFTER, marked SHALL MEAN 2.\n", "ADJOURN INDEFINITELY.", "", "dependency", "cannot be overwritten", 4},
		{"root call", "", "PROCLAIM THE FINDING OF helper REGARDING 1 AND 2.", "", "", "expects 1 argument(s)", 5},
		{"root deferred referral", "", "REFER TO ARTICLE 99.", "", "", "no ARTICLE 99", 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			log := docket.NewMemoryLog()
			defer log.Close()
			if _, _, err := Enact(t.Context(), log, statute); err != nil {
				t.Fatal(err)
			}
			_, err := File(t.Context(), log, "FORM K-1.\nIN THE MATTER OF: root.\nINCORPORATE BY REFERENCE dependency.\n"+tc.declarations+"ARTICLE 1.\n"+tc.body+tc.offices)
			assertSourceRejection(t, err, tc.source, tc.line, tc.message)
		})
	}
}

func TestStoredSourceProvenanceFailures(t *testing.T) {
	// Inject stored text directly to exercise malformed or incomplete dependency
	// histories that Enact would normally reject before publishing.
	for _, tc := range []struct {
		name, dependency, nested, source, message string
		line                                      int
	}{
		{"missing root import", "", "", "", "no statute named", 3},
		{"missing transitive import", "FORM S-1.\nIN THE MATTER OF: dependency.\nINCORPORATE BY REFERENCE nested.\nTHE OFFICE OF helper. REMAND.", "", "dependency", "no statute named", 3},
		{"imported parse failure", "FORM S-1.\nIN THE MATTER OF: dependency.\n😀", "", "dependency", "unsupported character", 3},
		{"transitive parse failure", "FORM S-1.\nIN THE MATTER OF: dependency.\nINCORPORATE BY REFERENCE nested.\nTHE OFFICE OF helper. REMAND.", "FORM S-1.\nIN THE MATTER OF: nested.\n😀", "nested", "unsupported character", 3},
		{"imported deferred section", "FORM S-1.\nIN THE MATTER OF: dependency.\nTHE OFFICE OF helper.\nREFER TO SECTION 99.", "", "dependency", "no SECTION 99", 4},
		{"imported static examiner", "FORM S-1.\nIN THE MATTER OF: dependency.\nTHE OFFICE OF helper.\nASSIGN THE LETTERS FOR invention TO \"recipient\".\nPROCLAIM THE PRACTICE OF invention.", "", "dependency", "use after assignment", 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			log := docket.NewMemoryLog()
			defer log.Close()
			for _, entry := range []struct{ name, source string }{{"dependency", tc.dependency}, {"nested", tc.nested}} {
				if entry.source == "" {
					continue
				}
				topic := docket.StatuteTopic(entry.name)
				if err := log.EnsureTopic(t.Context(), topic); err != nil {
					t.Fatal(err)
				}
				appends := []docket.StepAppend{{Topic: topic, Key: []byte(enactmentKey), Value: []byte("ENACTMENT 1.")}}
				for line := range strings.SplitSeq(entry.source, "\n") {
					appends = append(appends, docket.StepAppend{Topic: topic, Value: []byte(line)})
				}
				if _, err := log.AppendBatch(t.Context(), appends); err != nil {
					t.Fatal(err)
				}
			}
			_, err := File(t.Context(), log, "FORM K-1.\nIN THE MATTER OF: root.\nINCORPORATE BY REFERENCE dependency.\nARTICLE 1. ADJOURN INDEFINITELY.")
			assertSourceRejection(t, err, tc.source, tc.line, tc.message)
		})
	}
}

func assertSourceRejection(t *testing.T, err error, source string, line int, message string) {
	t.Helper()
	rejection, ok := errors.AsType[*gregor.RejectedFiling](err)
	if !ok || rejection.Source != source || rejection.Line != line || rejection.Col != 1 || !strings.Contains(rejection.Particulars, message) {
		t.Fatalf("unexpected rejection: %+v (%v), want source=%q line=%d %q", rejection, err, source, line, message)
	}
}
