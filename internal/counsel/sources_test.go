package counsel

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"
	"testing"

	"github.com/lennrt/trial-lang/canon"
	"github.com/lennrt/trial-lang/internal/docket"
)

func customStatute(name, body string) string {
	return "FORM S-1.\nIN THE MATTER OF: " + name + ".\n" + body
}

func customCase(name, body string) string {
	return "FORM K-1.\nIN THE MATTER OF: editor.\nINCORPORATE BY REFERENCE " + name + ".\nARTICLE 1.\n" + body
}

const answerOffice = "THE OFFICE OF answer, CONCERNING n.\nREMAND WITH n."

type diagnosticPublication struct {
	URI         string `json:"uri"`
	Version     *int64 `json:"version"`
	Diagnostics []struct {
		Range   struct{ Start, End position } `json:"range"`
		Message string                        `json:"message"`
		Related []struct {
			Location struct {
				URI string `json:"uri"`
			} `json:"location"`
		} `json:"relatedInformation"`
	} `json:"diagnostics"`
}

func readPublications(t *testing.T, out *bytes.Buffer) map[string]diagnosticPublication {
	t.Helper()
	publications := make(map[string]diagnosticPublication)
	r := bufio.NewReader(bytes.NewReader(out.Bytes()))
	for {
		body, err := readMessage(r)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				t.Fatal(err)
			}
			break
		}
		var reply struct {
			Method string                `json:"method"`
			Params diagnosticPublication `json:"params"`
		}
		if err := json.Unmarshal(body, &reply); err != nil {
			t.Fatal(err)
		}
		if reply.Method != "textDocument/publishDiagnostics" {
			t.Fatalf("unexpected response: %s", body)
		}
		publications[reply.Params.URI] = reply.Params
	}
	out.Reset()
	return publications
}

func documentMessage(t *testing.T, s *Server, out *bytes.Buffer, method, uri, text string, version int64) map[string]diagnosticPublication {
	t.Helper()
	document := map[string]any{"uri": uri}
	params := map[string]any{"textDocument": document}
	switch method {
	case "didOpen":
		document["text"], document["version"] = text, version
	case "didChange":
		document["version"] = version
		params["contentChanges"] = []map[string]any{{"text": text}}
	}
	raw, err := json.Marshal(params)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.handle(&rpcRequest{Method: "textDocument/" + method, Params: raw}); err != nil {
		t.Fatal(err)
	}
	return readPublications(t, out)
}

func requireDiagnostic(t *testing.T, publications map[string]diagnosticPublication, uri, message string, start position) {
	t.Helper()
	p, ok := publications[uri]
	if !ok {
		t.Fatalf("no publication for %s: %+v", uri, publications)
	}
	for _, d := range p.Diagnostics {
		if strings.Contains(d.Message, message) {
			if d.Range.Start != start {
				t.Fatalf("%s starts at %+v, want %+v", d.Message, d.Range.Start, start)
			}
			return
		}
	}
	t.Fatalf("no %q diagnostic at %s: %+v", message, uri, p)
}

func requireClean(t *testing.T, publications map[string]diagnosticPublication, uri string) {
	t.Helper()
	p, ok := publications[uri]
	if !ok || len(p.Diagnostics) != 0 {
		t.Fatalf("expected cleared diagnostics for %s: %+v", uri, publications)
	}
}

func TestCanonImportedErrorsUseTheirSource(t *testing.T) {
	// This regression uses the pre-existing --canon API. Previously the error
	// was placed at the case's document start, losing the imported declaration.
	var out bytes.Buffer
	s := &Server{Canon: true, Out: &out}
	root := canonProgram("ADJOURN INDEFINITELY.\nTHE OFFICE OF maximum, CONCERNING a AND b.\nREMAND WITH a.")
	publications := documentMessage(t, s, &out, "didOpen", "file:///case.trial", root, 1)
	text, err := canon.FS.ReadFile("statutes-of-arithmetic.trial")
	if err != nil {
		t.Fatal(err)
	}
	line := strings.Count(string(text[:bytes.Index(text, []byte("THE OFFICE OF maximum"))]), "\n")
	uri := "trial-canon:///statutes-of-arithmetic.trial"
	requireDiagnostic(t, publications, uri, "established more than once", position{Line: line})
	requireDiagnostic(t, publications, "file:///case.trial", "bundled statute statutes-of-arithmetic", position{Line: 2})
	if publications[uri].Version != nil {
		t.Fatal("closed virtual source received document version")
	}
	if got := publications[uri].Diagnostics[0].Related; len(got) != 1 || got[0].Location.URI != "file:///case.trial" {
		t.Fatalf("importing document not identified: %+v", got)
	}
}

func TestCustomSourcesLocationsAndOrder(t *testing.T) {
	const rootURI, aURI, bURI = "file:///case.trial", "file:///a.trial", "file:///b.trial"
	a := Source{URI: aURI, Text: customStatute("base", answerOffice)}
	b := Source{URI: bURI, Text: customStatute("wrapper", "INCORPORATE BY REFERENCE base.\nTHE OFFICE OF wrapper, CONCERNING n.\nREMAND WITH THE FINDING OF answer REGARDING n.")}
	for _, tc := range []struct {
		name         string
		sources      []Source
		root         string
		uri, message string
		start        position
	}{
		{"valid transitive", []Source{a, b}, customCase("wrapper", "PROCLAIM THE FINDING OF wrapper REGARDING 42."), "", "", position{}},
		{"root arity", []Source{a, b}, customCase("wrapper", "PROCLAIM THE FINDING OF answer REGARDING 1 AND 2."), rootURI, "expects 1 argument(s); received 2", position{Line: 4}},
		{"source order", []Source{b, a}, customCase("wrapper", "ADJOURN INDEFINITELY."), bURI, "no statute named", position{Line: 2}},
		{"transitive missing", []Source{{URI: bURI, Text: strings.Replace(b.Text, "REFERENCE base", "REFERENCE absent", 1)}}, customCase("wrapper", "ADJOURN INDEFINITELY."), bURI, "no statute named", position{Line: 2}},
		{"source compile", []Source{{URI: aURI, Text: customStatute("base", "THE OFFICE OF answer.\nREFER TO SECTION 99.")}}, customCase("base", "ADJOURN INDEFINITELY."), aURI, "SECTION 99", position{Line: 3}},
		{"source UTF16 parse", []Source{{URI: aURI, Text: strings.ReplaceAll(customStatute("base", "THE OFFICE OF answer.\nPROCLAIM \"😀\" EXTRA."), "\n", "\r\n")}}, customCase("base", "ADJOURN INDEFINITELY."), aURI, "expected", position{Line: 3, Character: 14}},
		{"source form", []Source{{URI: aURI, Text: "FORM K-1.\nIN THE MATTER OF: base.\nARTICLE 1.\nADJOURN INDEFINITELY."}}, customCase("base", "ADJOURN INDEFINITELY."), aURI, "Form S-1", position{}},
		{"collision", []Source{a}, customCase("base", "ADJOURN INDEFINITELY.\nTHE OFFICE OF answer, CONCERNING n.\nREMAND WITH n."), aURI, "established more than once", position{Line: 2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			s := &Server{Enactments: tc.sources, Out: &out}
			publications := documentMessage(t, s, &out, "didOpen", rootURI, tc.root, 1)
			if tc.message != "" {
				requireDiagnostic(t, publications, tc.uri, tc.message, tc.start)
			} else {
				for uri := range publications {
					requireClean(t, publications, uri)
				}
			}
		})
	}
}

func TestCustomSourceOverlayLifecycleAndVersions(t *testing.T) {
	const rootURI, statuteURI = "file:///case.trial", "file:///statute.trial"
	initial := customStatute("base", answerOffice)
	reloaded := customStatute("base", strings.Replace(answerOffice, "CONCERNING n", "CONCERNING n AND m", 1))
	reloadCalls := 0
	var out bytes.Buffer
	s := &Server{Out: &out, Enactments: []Source{{URI: statuteURI, Text: initial}}, ReloadSource: func(uri string) (string, error) {
		reloadCalls++
		if uri != statuteURI {
			t.Fatalf("unconfigured reload %s", uri)
		}
		return reloaded, nil
	}}
	root := customCase("base", "PROCLAIM THE FINDING OF answer REGARDING 42.")
	requireClean(t, documentMessage(t, s, &out, "didOpen", rootURI, root, 1), rootURI)
	bad := customStatute("base", "THE OFFICE OF answer.\nREFER TO SECTION 99.")
	publications := documentMessage(t, s, &out, "didOpen", statuteURI, bad, 7)
	requireDiagnostic(t, publications, statuteURI, "SECTION 99", position{Line: 3})
	if publications[statuteURI].Version == nil || *publications[statuteURI].Version != 7 {
		t.Fatalf("wrong accepted version: %+v", publications)
	}
	if len(publications[statuteURI].Diagnostics) != 1 {
		t.Fatalf("same source error duplicated: %+v", publications[statuteURI])
	}
	requireDiagnostic(t, publications, rootURI, "no statute named", position{Line: 2})
	if got := documentMessage(t, s, &out, "didChange", statuteURI, initial, 6); len(got) != 0 {
		t.Fatalf("stale change published: %+v", got)
	}
	if s.docs[statuteURI] != bad {
		t.Fatal("stale change replaced accepted overlay")
	}
	publications = documentMessage(t, s, &out, "didChange", statuteURI, initial, 8)
	requireClean(t, publications, statuteURI)
	requireClean(t, publications, rootURI)
	publications = documentMessage(t, s, &out, "didClose", statuteURI, "", 0)
	if reloadCalls != 1 {
		t.Fatalf("reload calls=%d", reloadCalls)
	}
	requireDiagnostic(t, publications, rootURI, "expects 2 argument(s); received 1", position{Line: 4})
	requireClean(t, publications, statuteURI)
	if publications[statuteURI].Version != nil {
		t.Fatal("closed source retained stale version")
	}
	documentMessage(t, s, &out, "didClose", rootURI, "", 0)
	if reloadCalls != 1 {
		t.Fatal("arbitrary open document triggered reload")
	}
}

func TestSharedImportedDiagnosticsSurviveClosingOneRoot(t *testing.T) {
	const statuteURI = "file:///statute.trial"
	var out bytes.Buffer
	s := &Server{Out: &out, Enactments: []Source{{URI: statuteURI, Text: customStatute("base", answerOffice)}}}
	root := customCase("base", "ADJOURN INDEFINITELY.\n"+answerOffice)
	documentMessage(t, s, &out, "didOpen", "file:///one.trial", root, 1)
	publications := documentMessage(t, s, &out, "didOpen", "file:///two.trial", root, 2)
	requireDiagnostic(t, publications, statuteURI, "established more than once", position{Line: 2})
	if got := publications[statuteURI].Diagnostics; len(got) != 1 || len(got[0].Related) != 2 {
		t.Fatalf("shared error not merged: %+v", got)
	}
	publications = documentMessage(t, s, &out, "didClose", "file:///one.trial", "", 0)
	requireClean(t, publications, "file:///one.trial")
	requireDiagnostic(t, publications, statuteURI, "established more than once", position{Line: 2})
	if got := publications[statuteURI].Diagnostics[0].Related; len(got) != 1 || got[0].Location.URI != "file:///two.trial" {
		t.Fatalf("stale importing location: %+v", got)
	}
	publications = documentMessage(t, s, &out, "didClose", "file:///two.trial", "", 0)
	requireClean(t, publications, statuteURI)
}

func TestOpenConfiguredCanonCollisionHasOneImportLocation(t *testing.T) {
	const uri = "file:///statute.trial"
	source := customStatute("custom", "INCORPORATE BY REFERENCE statutes-of-arithmetic.\nTHE OFFICE OF maximum, CONCERNING a AND b.\nREMAND WITH a.")
	var out bytes.Buffer
	s := &Server{Canon: true, Out: &out, Enactments: []Source{{URI: uri, Text: source}}}
	publications := documentMessage(t, s, &out, "didOpen", uri, source, 1)
	requireDiagnostic(t, publications, uri, "bundled statute statutes-of-arithmetic", position{Line: 2})
	if len(publications[uri].Diagnostics) != 1 {
		t.Fatalf("configured/open source produced duplicate locations: %+v", publications[uri])
	}
	imported := publications["trial-canon:///statutes-of-arithmetic.trial"].Diagnostics
	if len(imported) != 1 || len(imported[0].Related) != 1 {
		t.Fatalf("duplicate related import locations: %+v", imported)
	}
}

func TestCustomSourceRenameAndReloadFailureDoNotUseOldEnactment(t *testing.T) {
	const statuteURI, rootURI = "file:///statute.trial", "file:///case.trial"
	initial := customStatute("base", answerOffice)
	var out bytes.Buffer
	s := &Server{Out: &out, Enactments: []Source{{URI: statuteURI, Text: initial}}, ReloadSource: func(string) (string, error) { return "", errors.New("file removed") }}
	documentMessage(t, s, &out, "didOpen", rootURI, customCase("base", "ADJOURN INDEFINITELY."), 1)
	publications := documentMessage(t, s, &out, "didOpen", statuteURI, strings.Replace(initial, "OF: base", "OF: renamed", 1), 1)
	requireDiagnostic(t, publications, rootURI, "no statute named", position{Line: 2})
	publications = documentMessage(t, s, &out, "didClose", statuteURI, "", 0)
	requireDiagnostic(t, publications, statuteURI, "file removed", position{})
	requireDiagnostic(t, publications, rootURI, "no statute named", position{Line: 2})
	publications = documentMessage(t, s, &out, "didOpen", statuteURI, initial, 2)
	requireClean(t, publications, statuteURI)
	requireClean(t, publications, rootURI)
}

func TestCustomContextIsExplicitAndCheckedRootsDoNotEnact(t *testing.T) {
	var out bytes.Buffer
	s := &Server{Canon: true, Out: &out, Enactments: []Source{{URI: "file:///base.trial", Text: customStatute("base", answerOffice)}}}
	documentMessage(t, s, &out, "didOpen", "file:///unconfigured.trial", customStatute("other", "INCORPORATE BY REFERENCE base.\nTHE OFFICE OF other.\nREMAND WITH 1."), 1)
	publications := documentMessage(t, s, &out, "didOpen", "file:///root.trial", customCase("other", "REFER TO ARTICLE 99."), 1)
	requireDiagnostic(t, publications, "file:///root.trial", "no statute named", position{Line: 2})
	publications = documentMessage(t, s, &out, "didOpen", "file:///canon.trial", canonProgram("PROCLAIM THE FINDING OF maximum REGARDING 1 AND 2."), 1)
	requireClean(t, publications, "file:///canon.trial")
}

func TestConfiguredSourceURIAliases(t *testing.T) {
	for _, tc := range []struct{ name, configured, opened, duplicate string }{
		{"escape", "file:///project/a%20b.trial", "file://localhost/project/a b.trial", "file:///project/./a%20b.trial"},
		{"Windows", "file:///C:/Project/Source.trial", "file:///c:/project/source.trial", "file:///C:/PROJECT/source.trial"},
		{"UNC", "file://SERVER/Share/Source.trial", "file://server/share/source.trial", "file://Server/SHARE/source.trial"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name != "escape" && runtime.GOOS != "windows" {
				t.Skip("Windows path identity; covered by platform-independent key tests below")
			}
			var out bytes.Buffer
			reloads := 0
			valid := customStatute("base", answerOffice)
			s := &Server{Out: &out, Enactments: []Source{{URI: tc.configured, Text: valid}}, ReloadSource: func(uri string) (string, error) {
				reloads++
				if uri != tc.configured {
					t.Fatalf("callback escaped allowlist: %q", uri)
				}
				return valid, nil
			}}
			documentMessage(t, s, &out, "didOpen", "file:///root.trial", customCase("base", "ADJOURN INDEFINITELY."), 1)
			invalid := customStatute("base", "THE OFFICE OF answer.\nREFER TO SECTION 99.")
			publications := documentMessage(t, s, &out, "didOpen", tc.opened, invalid, 1)
			requireDiagnostic(t, publications, tc.opened, "SECTION 99", position{Line: 3})
			requireDiagnostic(t, publications, "file:///root.trial", "no statute named", position{Line: 2})
			publications = documentMessage(t, s, &out, "didOpen", tc.duplicate, valid, 1)
			requireDiagnostic(t, publications, tc.duplicate, "already open", position{})
			if _, ok := s.docs[tc.duplicate]; ok {
				t.Fatal("second alias accepted")
			}
			documentMessage(t, s, &out, "didClose", tc.duplicate, "", 0)
			if reloads != 0 || s.docs[tc.opened] != invalid {
				t.Fatal("rejected alias close replaced real overlay")
			}
			publications = documentMessage(t, s, &out, "didClose", tc.opened, "", 0)
			requireClean(t, publications, "file:///root.trial")
			requireClean(t, publications, tc.opened)
			if reloads != 1 {
				t.Fatalf("reload count=%d", reloads)
			}
			duplicate := &Server{Enactments: []Source{{URI: tc.configured}, {URI: tc.opened}}}
			if duplicate.validateSources() == nil {
				t.Fatal("configured aliases accepted")
			}
		})
	}
	if sourceURIKeyForPlatform("file:///project/A.trial", false) == sourceURIKeyForPlatform("file:///project/a.trial", false) {
		t.Fatal("POSIX case sensitivity lost")
	}
}

func TestSourceURIKeysRespectPlatformAndURIShape(t *testing.T) {
	for _, pair := range [][2]string{{"file:///project/a%23b.trial", "file:///project/a#b.trial"}, {"file:///project/a%3Fb.trial", "file:///project/a?b.trial"}, {"file:///project/a%25b.trial", "file:///project/a%b.trial"}} {
		if sourceURIKeyForPlatform(pair[0], true) == sourceURIKeyForPlatform(pair[1], true) {
			t.Errorf("literal filename and URI delimiter incorrectly aliased: %q", pair)
		}
	}
	for _, uri := range []string{"file:relative", "file:other", "file:../a", "file:///a?", "file:///a?q=1", "file:///a#fragment", "file://user@host/a", "file://host:80/a"} {
		if got := sourceURIKeyForPlatform(uri, true); got != uri {
			t.Errorf("normalized unsupported URI %q to %q", uri, got)
		}
	}
	for _, pair := range [][2]string{{"file:///C:/Project/a.trial", "file:///c:/project/A.trial"}, {"file://HOST/Share/a.trial", "file://host/share/A.trial"}} {
		if sourceURIKeyForPlatform(pair[0], true) != sourceURIKeyForPlatform(pair[1], true) {
			t.Errorf("Windows aliases differ: %q", pair)
		}
		if sourceURIKeyForPlatform(pair[0], false) == sourceURIKeyForPlatform(pair[1], false) {
			t.Errorf("POSIX paths incorrectly aliased: %q", pair)
		}
	}
	if sourceURIKeyForPlatform("file:///host/a", true) == sourceURIKeyForPlatform("file://host/a", true) {
		t.Fatal("local path and remote authority aliased")
	}
}

func TestCustomEnactmentVersionsKeepOrderedSuccessfulContext(t *testing.T) {
	const rootURI, overrideURI = "file:///case.trial", "file:///override.trial"
	base := customStatute("base", answerOffice)
	invalid := customStatute("base", answerOffice+"\n"+answerOffice)
	var out bytes.Buffer
	s := &Server{Out: &out, Enactments: []Source{{URI: "file:///base.trial", Text: base}, {URI: overrideURI, Text: invalid}}}
	root := customCase("base", "PROCLAIM THE FINDING OF answer REGARDING 1.")
	publications := documentMessage(t, s, &out, "didOpen", rootURI, root, 1)
	requireDiagnostic(t, publications, overrideURI, "established more than once", position{Line: 4})
	requireClean(t, publications, rootURI)
	// A rejected new enactment cannot erase a successful earlier version, but
	// its own file stays diagnosed. Repairing it changes the effective context.
	twoArgs := customStatute("base", strings.Replace(answerOffice, "CONCERNING n", "CONCERNING n AND m", 1))
	publications = documentMessage(t, s, &out, "didOpen", overrideURI, twoArgs, 1)
	requireClean(t, publications, overrideURI)
	requireDiagnostic(t, publications, rootURI, "expects 2 argument(s); received 1", position{Line: 4})
	publications = documentMessage(t, s, &out, "didClose", overrideURI, "", 0)
	requireDiagnostic(t, publications, overrideURI, "established more than once", position{Line: 4})
	requireClean(t, publications, rootURI)
}

func TestReloadBoundsExcludeStaleConfiguredText(t *testing.T) {
	const uri = "file:///source.trial"
	var out bytes.Buffer
	s := &Server{Out: &out, Enactments: []Source{{URI: uri, Text: customStatute("base", answerOffice)}}, ReloadSource: func(string) (string, error) { return strings.Repeat("x", maxDocumentBytes+1), nil }}
	documentMessage(t, s, &out, "didOpen", "file:///case.trial", customCase("base", "ADJOURN INDEFINITELY."), 1)
	documentMessage(t, s, &out, "didOpen", uri, s.Enactments[0].Text, 1)
	publications := documentMessage(t, s, &out, "didClose", uri, "", 0)
	requireDiagnostic(t, publications, uri, "exceeds", position{})
	requireDiagnostic(t, publications, "file:///case.trial", "no statute named", position{Line: 2})
}

func TestCustomSourcesBoundsAndCancellation(t *testing.T) {
	for _, tc := range []struct {
		name    string
		sources []Source
	}{
		{"count", make([]Source, maxEnactments+1)},
		{"empty URI", []Source{{Text: "x"}}},
		{"long URI", []Source{{URI: strings.Repeat("x", maxDocumentURIBytes+1)}}},
		{"duplicate URI", []Source{{URI: "file:///a"}, {URI: "file:///a"}}},
		{"source bytes", []Source{{URI: "file:///a", Text: strings.Repeat("x", maxDocumentBytes+1)}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &Server{Enactments: tc.sources, In: strings.NewReader(""), Out: &bytes.Buffer{}}
			if err := s.Serve(t.Context()); err == nil {
				t.Fatal("invalid configuration accepted")
			}
		})
	}
	// Reuse one backing string to exercise aggregate bounds without allocating
	// hundreds of MiB for this validation-only check.
	text := strings.Repeat(" ", maxDocumentBytes)
	s := &Server{}
	for i := range docket.MaxReadBytes/maxDocumentBytes + 1 {
		s.Enactments = append(s.Enactments, Source{URI: fmt.Sprintf("file:///%d", i), Text: text})
	}
	if err := s.validateSources(); err == nil {
		t.Fatal("oversized aggregate accepted")
	}
	s.Enactments = s.Enactments[:len(s.Enactments)-1]
	s.Enactments[0].Text = ""
	if err := s.storeDocument(s.Enactments[0].URI, text, false); err != nil {
		t.Fatal(err)
	}
	if err := s.storeDocument(s.Enactments[0].URI, text+" ", true); err == nil {
		t.Fatal("oversized overlay accepted")
	}
	if s.docs[s.Enactments[0].URI] != text {
		t.Fatal("rejected overlay replaced text")
	}
	s.Enactments = append(s.Enactments, Source{URI: "file:///extra"})
	if err := s.storeDocument("file:///extra", "", false); err != nil {
		t.Fatal(err)
	}
	if err := s.storeDocument("file:///extra", "x", true); err == nil {
		t.Fatal("effective aggregate accepted beyond limit")
	}
	if s.docs["file:///extra"] != "" {
		t.Fatal("aggregate rejection replaced accepted text")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := (&Server{Canon: true}).collectDiagnostics(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled refresh=%v", err)
	}
}
