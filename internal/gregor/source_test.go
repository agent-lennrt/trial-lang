package gregor

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestSourceParseRejections(t *testing.T) {
	for _, src := range []string{"FORM K-1.\n😀", "FORM K-9."} {
		_, ordinaryErr := Parse(src)
		_, namedErr := ParseSource(src, "dependency")
		ordinary, ordinaryOK := errors.AsType[*RejectedFiling](ordinaryErr)
		named, namedOK := errors.AsType[*RejectedFiling](namedErr)
		if !ordinaryOK || !namedOK || ordinary.Source != "" || named.Source != "dependency" {
			t.Fatalf("unexpected source identities: ordinary=%+v named=%+v", ordinary, named)
		}
		if ordinaryErr.Error() != namedErr.Error() || ordinary.Line != named.Line || ordinary.Col != named.Col {
			t.Fatalf("source identity changed rejection: ordinary=%v named=%v", ordinaryErr, namedErr)
		}
		ordinaryJSON, err := json.Marshal(ordinary)
		if err != nil {
			t.Fatal(err)
		}
		namedJSON, err := json.Marshal(named)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(ordinaryJSON, namedJSON) || bytes.Contains(namedJSON, []byte("dependency")) {
			t.Fatalf("source identity leaked into JSON: %s != %s", ordinaryJSON, namedJSON)
		}
	}
}

func TestSourceImportedCompilerRejections(t *testing.T) {
	const prefix = "FORM S-1.\nIN THE MATTER OF: dependency.\n"
	for _, tc := range []struct {
		name, body, message string
		line                int
	}{
		{"constant collision", "HEREINAFTER, value SHALL MEAN 1.\nHEREINAFTER, value SHALL MEAN 2.\nTHE OFFICE OF helper. REMAND.", "defined more than once", 4},
		{"exhibit collision", "THE EXHIBIT OF pair, COMPRISING value.\nTHE EXHIBIT OF pair, COMPRISING other.\nTHE OFFICE OF helper. REMAND.", "established more than once", 4},
		{"office collision", "THE OFFICE OF helper. REMAND.\nTHE OFFICE OF helper. REMAND.", "established more than once", 4},
		{"office concern", "THE OFFICE OF helper, CONCERNING value AND value. REMAND.", "concern", 3},
		{"nested expression", "THE OFFICE OF helper.\nREMAND WITH 1 PLUS THE FINDING OF absent REGARDING 2.", "not defined", 4},
		{"nested section expression", "THE OFFICE OF helper.\nSECTION 1.\nSHOULD 1 EQUAL 1, PROCLAIM THE FINDING OF absent REGARDING 2.", "not defined", 5},
		{"deferred section", "THE OFFICE OF helper.\nSHOULD 1 EQUAL 1, REFER TO SECTION 99.", "no SECTION 99", 4},
		{"section collision", "THE OFFICE OF helper.\nSECTION 1. REMAND.\nSECTION 1. REMAND.", "appears twice", 5},
		{"examiner body", "THE OFFICE OF helper.\nASSIGN THE LETTERS FOR invention TO \"recipient\".\nSHOULD 1 EQUAL 1, PROCLAIM THE PRACTICE OF invention.", "use after assignment", 5},
		{"examiner section", "THE OFFICE OF helper.\nSECTION 1.\nASSIGN THE LETTERS FOR invention TO \"recipient\".\nPROCLAIM THE PRACTICE OF invention.", "use after assignment", 6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var baseline string
			for _, identity := range []string{"", "dependency"} {
				root, err := ParseSource("FORM K-1.\nIN THE MATTER OF: root.\nARTICLE 1. ADJOURN INDEFINITELY.", "root")
				if err != nil {
					t.Fatal(err)
				}
				dep, err := ParseSource(prefix+tc.body, identity)
				if err != nil {
					t.Fatal(err)
				}
				root.Constants = append(root.Constants, dep.Constants...)
				root.Exhibits = append(root.Exhibits, dep.Exhibits...)
				root.Offices = append(root.Offices, dep.Offices...)
				_, err = Compile(root)
				rejection, ok := errors.AsType[*RejectedFiling](err)
				if !ok || rejection.Source != identity || rejection.Line != tc.line || rejection.Col != 1 || !strings.Contains(rejection.Particulars, tc.message) {
					t.Fatalf("unexpected rejection: %+v (%v), want %q line %d %q", rejection, err, identity, tc.line, tc.message)
				}
				if identity == "" {
					baseline = err.Error()
				} else if err.Error() != baseline {
					t.Fatalf("source identity changed error text: %q != %q", err, baseline)
				}
			}
		})
	}
}

func TestSourceRootArticleRejectionsAfterImportedOffices(t *testing.T) {
	for _, body := range []string{
		"REFER TO ARTICLE 99.",
		"FILE A MOTION TO RECONSIDER, REFERRING TO ARTICLE 99.",
		"PROCLAIM THE FINDING OF absent REGARDING 1.",
		"ASSIGN THE LETTERS FOR invention TO \"recipient\".\nPROCLAIM THE PRACTICE OF invention.",
	} {
		root, err := ParseSource("FORM K-1.\nIN THE MATTER OF: root.\nARTICLE 1.\n"+body, "root")
		if err != nil {
			t.Fatal(err)
		}
		dep, err := ParseSource("FORM S-1.\nIN THE MATTER OF: dependency.\nTHE OFFICE OF helper. REMAND.", "dependency")
		if err != nil {
			t.Fatal(err)
		}
		root.Offices = append(root.Offices, dep.Offices...)
		_, err = Compile(root)
		if rejection, ok := errors.AsType[*RejectedFiling](err); !ok || rejection.Source != "root" {
			t.Fatalf("root rejection acquired imported ownership: %+v (%v)", rejection, err)
		}
	}
}

func TestSourceMetadataDoesNotChangeInstructions(t *testing.T) {
	const source = `FORM K-1.
IN THE MATTER OF: positions.
HEREINAFTER, offset SHALL MEAN 2.
THE EXHIBIT OF pair, COMPRISING first AND second.
ARTICLE 1.
LET IT BE RECORDED THAT value IS AN EXHIBIT OF pair WHEREIN first IS 1 AND second IS 2.
PROCLAIM THE FINDING OF twice REGARDING offset.
LET IT BE RECORDED THAT authority IS A POWER OF ATTORNEY OVER THE OFFICE OF twice.
REFER TO ARTICLE 2.
ARTICLE 2.
ADJOURN INDEFINITELY.
THE OFFICE OF twice, CONCERNING n.
REFER TO SECTION 1.
SECTION 1.
REMAND WITH n PLUS n.`
	plain, err := Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	named, err := ParseSource(source, "file:///example.trial")
	if err != nil {
		t.Fatal(err)
	}
	plainInstructions, err := Compile(plain)
	if err != nil {
		t.Fatal(err)
	}
	namedInstructions, err := Compile(named)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plainInstructions, namedInstructions) {
		t.Fatalf("metadata changed instructions: plain=%+v named=%+v", plainInstructions, namedInstructions)
	}
	plainJSON, err := json.Marshal(plainInstructions)
	if err != nil {
		t.Fatal(err)
	}
	namedJSON, err := json.Marshal(namedInstructions)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(plainJSON, namedJSON) || bytes.Contains(namedJSON, []byte("example.trial")) {
		t.Fatalf("metadata changed instruction JSON: %s != %s", plainJSON, namedJSON)
	}
}
