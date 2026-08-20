package source

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseRepoURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want Repo
	}{
		{"https", "https://github.com/owner/name", Repo{Owner: "owner", Name: "name"}},
		{"host", "github.com/owner/name", Repo{Owner: "owner", Name: "name"}},
		{"short", "owner/name", Repo{Owner: "owner", Name: "name"}},
		{"git suffix", "https://github.com/owner/name.git", Repo{Owner: "owner", Name: "name"}},
		{"trailing slash", "github.com/owner/name.git/", Repo{Owner: "owner", Name: "name"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRepoURL(tt.raw)
			if err != nil {
				t.Fatalf("ParseRepoURL() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("ParseRepoURL() = %+v, want %+v", got, tt.want)
			}
		})
	}

	for _, raw := range []string{
		"http://github.com/owner/name",
		"https://gitlab.com/owner/name",
		"https://github.com/owner/name/issues",
		"https://github.com/owner/name?tab=readme",
		"github.com/owner",
		"owner/name/extra",
		"github.com/owner/name%2Fother",
		"https://github.com/-bad/name",
	} {
		t.Run("reject "+raw, func(t *testing.T) {
			if _, err := ParseRepoURL(raw); !errors.Is(err, ErrInvalidRepoURL) {
				t.Fatalf("ParseRepoURL(%q) error = %v, want ErrInvalidRepoURL", raw, err)
			}
		})
	}
}

func TestFetchExtractsAndCleansUp(t *testing.T) {
	sha := strings.Repeat("a", 40)
	payload := tarGzip(t, "demo-"+sha+"/package.json", []byte(`{"name":"demo"}`))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/owner/demo/tar.gz/"+sha {
			t.Fatalf("request path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	oldBase, oldClient := codeloadBaseURL, httpClient
	codeloadBaseURL, httpClient = server.URL, server.Client()
	t.Cleanup(func() { codeloadBaseURL, httpClient = oldBase, oldClient })

	dir, cleanup, err := Fetch(context.Background(), Repo{Owner: "owner", Name: "demo"}, sha)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if got := filepath.Base(dir); got != "demo-"+sha {
		t.Fatalf("Fetch() dir = %q", dir)
	}
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err != nil {
		t.Fatalf("extracted file missing: %v", err)
	}
	cleanup()
	if _, err := os.Stat(filepath.Dir(dir)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cleanup left temporary tree: %v", err)
	}
}

func TestFetchRejectsOversizedTarball(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "524288001")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	oldBase, oldClient := codeloadBaseURL, httpClient
	codeloadBaseURL, httpClient = server.URL, server.Client()
	t.Cleanup(func() { codeloadBaseURL, httpClient = oldBase, oldClient })

	_, _, err := Fetch(context.Background(), Repo{Owner: "owner", Name: "demo"}, strings.Repeat("a", 40))
	if !errors.Is(err, ErrSourceTooLarge) {
		t.Fatalf("Fetch() error = %v, want ErrSourceTooLarge", err)
	}
}

func TestPrepareParsesRailpackOutput(t *testing.T) {
	dir := t.TempDir()
	planPath := filepath.Join(dir, "fake-railpack")
	script := `#!/bin/sh
plan=""
info=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --plan-out) plan="$2"; shift 2 ;;
    --info-out) info="$2"; shift 2 ;;
    *) shift ;;
  esac
done
printf '%s' '{"deploy":{"startCommand":"node server.js"}}' > "$plan"
printf '%s' '{"resolvedPackages":{"node":{"resolvedVersion":"22.11.0"}},"metadata":{"providers":"node"},"detectedProviders":["node"]}' > "$info"
`
	if err := os.WriteFile(planPath, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	oldBinary := railpackBinary
	railpackBinary = planPath
	t.Cleanup(func() { railpackBinary = oldBinary })

	plan, info, err := Prepare(context.Background(), dir, map[string]string{"B": "two", "A": "one"})
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if !json.Valid(plan.Raw) || info.Provider != "node" || info.RuntimeVersions["node"] != "22.11.0" || info.StartCommand != "node server.js" {
		t.Fatalf("Prepare() = plan %s, info %+v", plan.Raw, info)
	}
}

func tarGzip(t *testing.T, name string, body []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: filepath.ToSlash(filepath.Dir(name)) + "/", Mode: 0o755, Typeflag: tar.TypeDir}); err != nil {
		t.Fatal(err)
	}
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestSuggestedPort(t *testing.T) {
	for provider, want := range map[string]int{"node": 3000, "python": 8000, "go": 8080, "staticfile": 80, "unknown": 3000} {
		if got := SuggestedPort(provider); got != want {
			t.Errorf("SuggestedPort(%q) = %d, want %d", provider, got, want)
		}
	}
}
