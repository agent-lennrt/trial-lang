package main

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCounselFileURI(t *testing.T) {
	path := filepath.Join(t.TempDir(), "statute #100%.trial")
	file, err := normalizeCounselFile(path)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(file.uri)
	if err != nil || u.Scheme != "file" || u.Fragment != "" || u.RawQuery != "" || !strings.HasPrefix(u.Path, "/") {
		t.Fatalf("invalid file URI %q: %+v, %v", file.uri, u, err)
	}
	if !strings.Contains(file.uri, "statute%20%23100%25.trial") {
		t.Fatalf("URI does not escape filename: %q", file.uri)
	}
	alias := filepath.Join(filepath.Dir(path), "sub") + string(filepath.Separator) + ".." + string(filepath.Separator) + filepath.Base(path)
	normalized, err := normalizeCounselFile(alias)
	if err != nil || normalized != file {
		t.Fatalf("lexical alias = %+v, %v; want %+v", normalized, err, file)
	}
	if runtime.GOOS == "windows" && !strings.HasPrefix(u.Path, "/"+filepath.VolumeName(path)+"/") {
		t.Fatalf("Windows URI lacks drive-leading slash: %q", file.uri)
	}
	if runtime.GOOS == "windows" {
		unc, err := normalizeCounselFile(`\\server\share\statute name.trial`)
		if err != nil || unc.uri != "file://server/share/statute%20name.trial" {
			t.Fatalf("UNC URI = %q, %v", unc.uri, err)
		}
	}
}

func writeCounselFixture(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func counselFixtureFile(t *testing.T, name, text string) counselFile {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	writeCounselFixture(t, path, text)
	file, err := normalizeCounselFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

func TestCounselSourceReloadAllowlistAndBounds(t *testing.T) {
	a := counselFixtureFile(t, "a.trial", "aaa")
	b := counselFixtureFile(t, "b.trial", "bbb")
	sources, reload, err := loadCounselSources(t.Context(), []counselFile{a, b}, 10)
	if err != nil || len(sources) != 2 || sources[0].URI != a.uri || sources[0].Text != "aaa" || sources[1].URI != b.uri {
		t.Fatalf("loaded sources = %+v, %v", sources, err)
	}
	unknown := counselFixtureFile(t, "not-allowed.trial", "secret")
	if _, err := reload(unknown.uri); err == nil || !strings.Contains(err.Error(), "not a configured statute") {
		t.Fatalf("unconfigured URI must be rejected: %v", err)
	}
	writeCounselFixture(t, a.path, "aaaaaaaa")
	if _, err := reload(a.uri); err == nil || !strings.Contains(err.Error(), "exceed 10 bytes") {
		t.Fatalf("aggregate growth must be rejected: %v", err)
	}
	writeCounselFixture(t, b.path, "b")
	if got, err := reload(b.uri); err != nil || got != "b" {
		t.Fatalf("reload smaller source = %q, %v", got, err)
	}
	if got, err := reload(a.uri); err != nil || got != "aaaaaaaa" {
		t.Fatalf("reload after freeing bytes = %q, %v", got, err)
	}
	writeCounselFixture(t, a.path, strings.Repeat("x", maxSourceBytes+1))
	if _, err := reload(a.uri); err == nil || !strings.Contains(err.Error(), "source limit") {
		t.Fatalf("oversized reload must fail: %v", err)
	}
	if err := os.Remove(a.path); err != nil {
		t.Fatal(err)
	}
	if _, err := reload(a.uri); err == nil {
		t.Fatal("deleted configured source must fail to reload")
	}
}

func TestCounselSourceStartupBounds(t *testing.T) {
	a := counselFixtureFile(t, "a.trial", "invalid syntax")
	b := counselFixtureFile(t, "b.trial", "also invalid")
	if sources, _, err := loadCounselSources(t.Context(), []counselFile{a}, 100); err != nil || sources[0].Text != "invalid syntax" {
		t.Fatalf("syntax errors belong in editor diagnostics: %+v, %v", sources, err)
	}
	if _, _, err := loadCounselSources(t.Context(), []counselFile{a, b}, 20); err == nil || !strings.Contains(err.Error(), "exceed 20 bytes") {
		t.Fatalf("aggregate startup bound not enforced: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, _, err := loadCounselSources(ctx, []counselFile{a}, 100); err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("canceled startup must fail: %v", err)
	}
}

func TestCounselSourceReloadCancellation(t *testing.T) {
	file := counselFixtureFile(t, "saved.trial", "source")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	_, reload, err := loadCounselSources(ctx, []counselFile{file}, 100)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	if _, err := reload(file.uri); err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("canceled reload must fail: %v", err)
	}
}
