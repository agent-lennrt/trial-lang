package counsel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"runtime"
	"slices"
	"strings"

	"github.com/lennrt/trial-lang/canon"
	"github.com/lennrt/trial-lang/internal/court"
	"github.com/lennrt/trial-lang/internal/docket"
	"github.com/lennrt/trial-lang/internal/gregor"
)

// Source is a snapshot of an explicitly configured statute. URI identifies its
// editor document; Counsel never interprets it as a path or reads it itself.
type Source struct {
	URI  string
	Text string
}

const maxEnactments = 100

func (s *Server) validateSources() error {
	if len(s.Enactments) > maxEnactments {
		return fmt.Errorf("at most %d statute sources may be configured", maxEnactments)
	}
	seen := make(map[string]bool)
	total := 0
	for _, source := range s.Enactments {
		if source.URI == "" || len(source.URI) > maxDocumentURIBytes {
			return fmt.Errorf("statute source URI must contain 1 to %d bytes", maxDocumentURIBytes)
		}
		key := sourceURIKey(source.URI)
		if seen[key] {
			return fmt.Errorf("statute source URI %q is configured more than once", source.URI)
		}
		seen[key] = true
		if len(source.Text) > maxDocumentBytes {
			return fmt.Errorf("statute source %q exceeds %d bytes", source.URI, maxDocumentBytes)
		}
		total += len(source.Text)
	}
	if total > docket.MaxReadBytes {
		return fmt.Errorf("statute sources exceed %d bytes", docket.MaxReadBytes)
	}
	return nil
}

func (s *Server) checkEffectiveSourceSize(uri, text string) error {
	total := 0
	for _, source := range s.Enactments {
		current := source.Text
		if overlay, ok := s.sourceOverlay(source.URI); ok {
			current = overlay.Text
		}
		if sourceURIKey(source.URI) == sourceURIKey(uri) {
			current = text
		}
		total += len(current)
	}
	if total > docket.MaxReadBytes {
		return fmt.Errorf("effective statute sources exceed %d bytes", docket.MaxReadBytes)
	}
	return nil
}

// Normalize identity only; no URI is converted into a filesystem read. On
// Windows, paths follow its usual case-insensitive file identity. Other
// file paths remain case-sensitive; non-file URIs retain their exact identity.
func sourceURIKey(uri string) string {
	return sourceURIKeyForPlatform(uri, runtime.GOOS == "windows")
}

func sourceURIKeyForPlatform(uri string, windows bool) string {
	parsed, err := url.Parse(uri)
	if err != nil || !strings.EqualFold(parsed.Scheme, "file") || parsed.Opaque != "" || !strings.HasPrefix(parsed.Path, "/") || parsed.ForceQuery || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil || parsed.Port() != "" {
		return uri
	}
	host := strings.ToLower(parsed.Host)
	if host == "localhost" {
		host = ""
	}
	name := path.Clean(parsed.Path)
	if windows {
		name = strings.ToLower(name)
	}
	return (&url.URL{Scheme: "file", Host: host, Path: name}).String()
}

func (s *Server) sourceOverlay(uri string) (Source, bool) {
	key := sourceURIKey(uri)
	for openURI, text := range s.docs {
		if sourceURIKey(openURI) == key {
			return Source{URI: openURI, Text: text}, true
		}
	}
	return Source{}, false
}

func (s *Server) checkSourceAlias(uri string) error {
	for _, source := range s.Enactments {
		if sourceURIKey(source.URI) != sourceURIKey(uri) {
			continue
		}
		if overlay, ok := s.sourceOverlay(uri); ok && overlay.URI != uri {
			return fmt.Errorf("configured statute is already open as %q", overlay.URI)
		}
	}
	return nil
}

func (s *Server) setVersion(uri string, version *int64) {
	if s.versions == nil {
		s.versions = make(map[string]*int64)
	}
	s.versions[uri] = version
}

func (s *Server) closeDocument(uri string) error {
	_, wasOpen := s.docs[uri]
	delete(s.docs, uri)
	delete(s.versions, uri)
	if wasOpen {
		for i := range s.Enactments {
			source := &s.Enactments[i]
			if sourceURIKey(source.URI) != sourceURIKey(uri) {
				continue
			}
			text := source.Text
			var err error
			if s.ReloadSource != nil {
				text, err = s.ReloadSource(source.URI)
			}
			if err == nil && len(text) > maxDocumentBytes {
				err = fmt.Errorf("statute source exceeds %d bytes", maxDocumentBytes)
			}
			if err == nil {
				err = s.checkEffectiveSourceSize(uri, text)
			}
			if s.reloadErrors == nil {
				s.reloadErrors = make(map[string]error)
			}
			if err != nil {
				s.reloadErrors[source.URI] = fmt.Errorf("reload configured statute: %w", err)
			} else {
				source.Text = text
				delete(s.reloadErrors, source.URI)
			}
			break
		}
	}
	return s.publishDiagnostics(uri)
}

// collectDiagnostics rebuilds the dependency context for each refresh. Checked
// documents cannot enact a new statute or retain temporary cases in this log.
func (s *Server) collectDiagnostics(ctx context.Context) (map[string][]map[string]any, error) {
	byURI := make(map[string][]map[string]any)
	log := docket.NewMemoryLog()
	defer log.Close()
	owners := make(map[string]Source)
	if s.Canon {
		for _, file := range canon.Files() {
			text, err := canon.FS.ReadFile(file)
			if err != nil {
				return nil, fmt.Errorf("read bundled canon: %w", err)
			}
			source := Source{URI: "trial-canon:///" + file, Text: string(text)}
			if err := enactDiagnosticSource(ctx, log, source, owners, byURI); err != nil {
				return nil, err
			}
		}
	}
	for _, source := range s.Enactments {
		if overlay, ok := s.sourceOverlay(source.URI); ok {
			source = overlay
		} else if err := s.reloadErrors[source.URI]; err != nil {
			byURI[source.URI] = append(byURI[source.URI], sourceDiagnostic("", err))
			continue
		}
		if err := enactDiagnosticSource(ctx, log, source, owners, byURI); err != nil {
			return nil, err
		}
	}
	rootURIs := make([]string, 0, len(s.docs))
	for uri := range s.docs {
		rootURIs = append(rootURIs, uri)
	}
	slices.Sort(rootURIs)
	for _, uri := range rootURIs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		source := Source{URI: uri, Text: s.docs[uri]}
		if byURI[uri] == nil {
			byURI[uri] = []map[string]any{}
		}
		prog, err := gregor.Parse(source.Text)
		if err == nil {
			switch {
			case len(prog.Incorporations) == 0:
				_, err = gregor.Compile(prog)
			case len(s.Enactments) != 0:
				err = court.Check(ctx, log, source.Text)
			case s.Canon && importsAvailable(prog, owners):
				err = court.Check(ctx, log, source.Text)
			}
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if err != nil {
			addSourceDiagnostic(byURI, owners, source, prog, err)
		}
	}
	deduplicateDiagnostics(byURI)
	return byURI, nil
}

func enactDiagnosticSource(ctx context.Context, log docket.Log, source Source, owners map[string]Source, byURI map[string][]map[string]any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	name, _, err := court.Enact(ctx, log, source.Text)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		// Parsing again only supplies direct import locations for related
		// information. Enact remains authoritative for validation.
		prog, _ := gregor.Parse(source.Text)
		addSourceDiagnostic(byURI, owners, source, prog, err)
	} else {
		owners[name] = source
	}
	return nil
}

func importsAvailable(prog *gregor.Program, owners map[string]Source) bool {
	for _, inc := range prog.Incorporations {
		if _, ok := owners[inc.Name]; !ok {
			return false
		}
	}
	return true
}

func addSourceDiagnostic(byURI map[string][]map[string]any, owners map[string]Source, root Source, prog *gregor.Program, err error) {
	owner := root
	if rejection, ok := errors.AsType[*gregor.RejectedFiling](err); ok && rejection.Source != "" {
		if imported, found := owners[rejection.Source]; found {
			owner = imported
		}
	}
	diagnostic := sourceDiagnostic(owner.Text, err)
	if owner.URI != root.URI {
		var start, end position
		if rejection, ok := errors.AsType[*gregor.RejectedFiling](err); ok && prog != nil {
			for _, inc := range prog.Incorporations {
				if inc.Name == rejection.Source {
					start, end = diagnosticRange(root.Text, inc.Line, 1)
					break
				}
			}
		}
		diagnostic["relatedInformation"] = []map[string]any{{
			"location": map[string]any{"uri": root.URI, "range": map[string]any{"start": start, "end": end}},
			"message":  "Imported while checking this document.",
		}}
		// Generic clients may not be able to open embedded virtual documents.
		// Keep their rejection visible in the editable importing document too.
		if strings.HasPrefix(owner.URI, "trial-canon:///") {
			location := owner.URI
			if rejection, ok := errors.AsType[*gregor.RejectedFiling](err); ok {
				location = fmt.Sprintf("%s at line %d", rejection.Source, rejection.Line)
			}
			byURI[root.URI] = append(byURI[root.URI], map[string]any{
				"range":    map[string]any{"start": start, "end": end},
				"severity": 1, "source": "gregor",
				"message": "While checking this document, bundled statute " + location + ": " + diagnostic["message"].(string),
				"relatedInformation": []map[string]any{{
					"location": map[string]any{"uri": owner.URI, "range": diagnostic["range"]},
					"message":  "The rejected declaration is in this bundled statute.",
				}},
			})
		}
	}
	byURI[owner.URI] = append(byURI[owner.URI], diagnostic)
}

func sourceDiagnostic(text string, err error) map[string]any {
	var start, end position
	message := err.Error()
	if rejection, ok := errors.AsType[*gregor.RejectedFiling](err); ok {
		start, end = diagnosticRange(text, rejection.Line, rejection.Col)
		message = rejection.Particulars
	}
	return map[string]any{
		"range":    map[string]any{"start": start, "end": end},
		"severity": 1, "source": "gregor",
		"message": message + " (Filing would be rejected pursuant to Article §4.2.)",
	}
}

func deduplicateDiagnostics(byURI map[string][]map[string]any) {
	for uri, diagnostics := range byURI {
		encoded := make(map[string]map[string]any)
		related := make(map[string]map[string]map[string]any)
		for _, diagnostic := range diagnostics {
			information, _ := diagnostic["relatedInformation"].([]map[string]any)
			delete(diagnostic, "relatedInformation")
			key, _ := json.Marshal(diagnostic)
			encoded[string(key)] = diagnostic
			if related[string(key)] == nil {
				related[string(key)] = make(map[string]map[string]any)
			}
			for _, item := range information {
				itemKey, _ := json.Marshal(item)
				related[string(key)][string(itemKey)] = item
			}
		}
		keys := make([]string, 0, len(encoded))
		for key := range encoded {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		byURI[uri] = make([]map[string]any, 0, len(keys))
		for _, key := range keys {
			items := related[key]
			itemKeys := make([]string, 0, len(items))
			for itemKey := range items {
				itemKeys = append(itemKeys, itemKey)
			}
			slices.Sort(itemKeys)
			if len(itemKeys) != 0 {
				information := make([]map[string]any, 0, len(itemKeys))
				for _, itemKey := range itemKeys {
					information = append(information, items[itemKey])
				}
				encoded[key]["relatedInformation"] = information
			}
			byURI[uri] = append(byURI[uri], encoded[key])
		}
	}
}

func sortedDiagnosticURIs(byURI map[string][]map[string]any) []string {
	uris := make([]string, 0, len(byURI))
	for uri := range byURI {
		uris = append(uris, uri)
	}
	slices.Sort(uris)
	return uris
}

func (s *Server) publishDiagnostics(changedURI string) error {
	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background() // Direct handler users have no Serve context.
	}
	byURI, err := s.collectDiagnostics(ctx)
	if err != nil {
		return err
	}
	for uri := range s.published {
		if _, ok := byURI[uri]; !ok {
			byURI[uri] = []map[string]any{}
		}
	}
	if _, ok := byURI[changedURI]; !ok {
		byURI[changedURI] = []map[string]any{}
	}
	s.published = make(map[string]bool)
	for _, uri := range sortedDiagnosticURIs(byURI) {
		params := map[string]any{"uri": uri, "diagnostics": byURI[uri]}
		if version := s.versions[uri]; version != nil {
			params["version"] = *version
		}
		if err := s.write(rpcResponse{Method: "textDocument/publishDiagnostics", Params: params}); err != nil {
			return err
		}
		if len(byURI[uri]) != 0 {
			s.published[uri] = true
		}
	}
	return nil
}
