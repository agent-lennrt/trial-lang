package main

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lennrt/trial-lang/internal/counsel"
)

type counselFile struct {
	path string
	uri  string
}

func normalizeCounselFile(path string) (counselFile, error) {
	if path == "" || path == "-" {
		return counselFile{}, fmt.Errorf("--enact needs a file path; standard input is reserved for the LSP connection")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return counselFile{}, err
	}
	uriPath := filepath.ToSlash(absolute)
	u := url.URL{Scheme: "file", Path: uriPath}
	if runtime.GOOS == "windows" && strings.HasPrefix(uriPath, "//") {
		// UNC paths use the server as the file URI's authority.
		host, rest, _ := strings.Cut(strings.TrimPrefix(uriPath, "//"), "/")
		u.Host, u.Path = host, "/"+rest
	} else if !strings.HasPrefix(uriPath, "/") {
		// A Windows drive path must not become file:C:/path.
		u.Path = "/" + uriPath
	}
	return counselFile{path: absolute, uri: u.String()}, nil
}

func counselPathKey(path string) string {
	if runtime.GOOS == "windows" {
		return strings.ToLower(path)
	}
	return path
}

// loadCounselSources owns the explicit file allowlist and its saved byte total.
// Editor URIs are lookup keys only; they are never converted into filesystem paths.
func loadCounselSources(ctx context.Context, files []counselFile, maxBytes int) ([]counsel.Source, func(string) (string, error), error) {
	sources := make([]counsel.Source, 0, len(files))
	paths := make(map[string]string, len(files))
	sizes := make(map[string]int, len(files))
	total := 0
	for _, file := range files {
		data, err := readSource(ctx, file.path)
		if err != nil {
			return nil, nil, fmt.Errorf("read statute %q: %w", file.path, err)
		}
		if len(data) > maxBytes-total {
			return nil, nil, fmt.Errorf("configured statutes exceed %d bytes", maxBytes)
		}
		total += len(data)
		paths[file.uri], sizes[file.uri] = file.path, len(data)
		sources = append(sources, counsel.Source{URI: file.uri, Text: string(data)})
	}
	reload := func(uri string) (string, error) {
		path, ok := paths[uri]
		if !ok {
			return "", fmt.Errorf("source URI is not a configured statute")
		}
		data, err := readSource(ctx, path)
		if err != nil {
			return "", fmt.Errorf("read statute %q: %w", path, err)
		}
		previous := sizes[uri]
		if len(data) > maxBytes-(total-previous) {
			return "", fmt.Errorf("configured statutes exceed %d bytes", maxBytes)
		}
		total += len(data) - previous
		sizes[uri] = len(data)
		return string(data), nil
	}
	return sources, reload, nil
}
