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
	"strconv"
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
		{"branch page", "https://github.com/owner/name/tree/develop", Repo{Owner: "owner", Name: "name"}},
		{"file page", "https://github.com/owner/name/blob/main/README.md", Repo{Owner: "owner", Name: "name"}},
		{"tab query", "https://github.com/owner/name?tab=readme-ov-file", Repo{Owner: "owner", Name: "name"}},
		{"anchor", "https://github.com/owner/name#install", Repo{Owner: "owner", Name: "name"}},
		{"issues", "https://github.com/owner/name/issues", Repo{Owner: "owner", Name: "name"}},
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
	argsPath := filepath.Join(dir, "railpack-args")
	script := `#!/bin/sh
plan=""
info=""
args_file="` + argsPath + `"
printf '%s\n' "$@" > "$args_file"
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
	args, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(args), "prepare\n"+dir+"\n--plan-out\n"; !strings.HasPrefix(got, want) {
		t.Fatalf("Prepare() args = %q, want prefix %q", got, want)
	}
	if a, b := strings.Index(string(args), "A=one"), strings.Index(string(args), "B=two"); a < 0 || b < 0 || a > b {
		t.Fatalf("Prepare() did not pass ordered environment values: %q", args)
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

// entry describes one tar record for tarWith.
type entry struct {
	name     string
	typeflag byte
	body     string
	linkname string
}

func tarWith(t *testing.T, entries []entry) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		hdr := &tar.Header{Name: e.name, Typeflag: e.typeflag, Linkname: e.linkname, Mode: 0o644}
		if e.typeflag == tar.TypeDir {
			hdr.Mode = 0o755
		}
		if e.typeflag == tar.TypeReg {
			hdr.Size = int64(len(e.body))
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if e.typeflag == tar.TypeReg {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func writeTar(t *testing.T, root string, data []byte) string {
	t.Helper()
	path := filepath.Join(root, "archive.tar.gz")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// Real repositories carry symlinks: sveltejs/kit, vitejs/vite and
// prisma/prisma all do. Refusing them refused the repository.
func TestExtractTarballKeepsInTreeSymlinks(t *testing.T) {
	root := t.TempDir()
	data := tarWith(t, []entry{
		{name: "repo/", typeflag: tar.TypeDir},
		{name: "repo/real.txt", typeflag: tar.TypeReg, body: "hello"},
		{name: "repo/link.txt", typeflag: tar.TypeSymlink, linkname: "real.txt"},
		{name: "repo/nested/", typeflag: tar.TypeDir},
		{name: "repo/nested/up.txt", typeflag: tar.TypeSymlink, linkname: "../real.txt"},
	})
	if err := extractTarball(writeTar(t, root, data), root); err != nil {
		t.Fatalf("extractTarball() error = %v, want nil", err)
	}
	for _, name := range []string{"repo/link.txt", "repo/nested/up.txt"} {
		path := filepath.Join(root, filepath.FromSlash(name))
		if _, err := os.Lstat(path); err != nil {
			t.Fatalf("%s was not created: %v", name, err)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s does not resolve: %v", name, err)
		}
		if string(body) != "hello" {
			t.Fatalf("%s resolved to %q, want %q", name, body, "hello")
		}
	}
}

// An escaping link is dropped, not fatal. Every link that survives points
// inside the tree, so nothing written later can leave it.
func TestExtractTarballDropsEscapingLinks(t *testing.T) {
	root := t.TempDir()
	data := tarWith(t, []entry{
		{name: "repo/", typeflag: tar.TypeDir},
		{name: "repo/ok.txt", typeflag: tar.TypeReg, body: "kept"},
		{name: "repo/escape", typeflag: tar.TypeSymlink, linkname: "../../../../etc/passwd"},
		{name: "repo/absolute", typeflag: tar.TypeSymlink, linkname: "/etc/passwd"},
		{name: "repo/hardescape", typeflag: tar.TypeLink, linkname: "../../../../etc/passwd"},
	})
	if err := extractTarball(writeTar(t, root, data), root); err != nil {
		t.Fatalf("extractTarball() error = %v, want nil", err)
	}
	for _, name := range []string{"repo/escape", "repo/absolute", "repo/hardescape"} {
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(name))); !os.IsNotExist(err) {
			t.Fatalf("%s should have been dropped, Lstat error = %v", name, err)
		}
	}
	if body, err := os.ReadFile(filepath.Join(root, "repo", "ok.txt")); err != nil || string(body) != "kept" {
		t.Fatalf("extraction did not continue past the dropped links: %q %v", body, err)
	}
}

// A planted symlink must not become a hole to write through. The file entry
// replaces the link instead of following it.
func TestExtractTarballDoesNotWriteThroughPlantedLink(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	data := tarWith(t, []entry{
		{name: "repo/", typeflag: tar.TypeDir},
		{name: "repo/hole", typeflag: tar.TypeSymlink, linkname: outside},
		{name: "repo/hole", typeflag: tar.TypeReg, body: "overwritten"},
	})
	if err := extractTarball(writeTar(t, root, data), root); err != nil {
		t.Fatalf("extractTarball() error = %v, want nil", err)
	}
	body, err := os.ReadFile(outside)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "original" {
		t.Fatalf("archive wrote through a symlink: outside file = %q", body)
	}
	if body, err := os.ReadFile(filepath.Join(root, "repo", "hole")); err != nil || string(body) != "overwritten" {
		t.Fatalf("duplicate entry did not replace the link in place: %q %v", body, err)
	}
}

// The dashboard asks for a branch head every time it renders. Without a memo
// each render was one network round trip per repository service.
func TestResolveSHACachedReusesRecentAnswers(t *testing.T) {
	dir := t.TempDir()
	countPath := filepath.Join(dir, "calls")
	stub := filepath.Join(dir, "fake-git")
	script := "#!/bin/sh\nprintf x >> " + countPath + "\nprintf '%s\\t%s\\n' " + strings.Repeat("a", 40) + " refs/heads/main\n"
	if err := os.WriteFile(stub, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	oldBinary := gitBinary
	gitBinary = stub
	t.Cleanup(func() { gitBinary = oldBinary })

	branchSHACache.Lock()
	branchSHACache.entries = map[string]branchSHAEntry{}
	branchSHACache.Unlock()

	repo := Repo{Owner: "owner", Name: "demo", Branch: "main"}
	for range 3 {
		if _, err := ResolveSHACached(context.Background(), repo); err != nil {
			t.Fatalf("ResolveSHACached() error = %v", err)
		}
	}
	calls, err := os.ReadFile(countPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 {
		t.Fatalf("resolved the branch %d times, want 1", len(calls))
	}

	// A different branch of the same repository is a different question.
	if _, err := ResolveSHACached(context.Background(), Repo{Owner: "owner", Name: "demo", Branch: "next"}); err != nil {
		t.Fatalf("ResolveSHACached() error = %v", err)
	}
	if calls, _ = os.ReadFile(countPath); len(calls) != 2 {
		t.Fatalf("a second branch reused the first branch's answer: %d calls", len(calls))
	}

	// An expired entry is resolved again, and expired neighbours are dropped.
	branchSHACache.Lock()
	for key, entry := range branchSHACache.entries {
		entry.resolved = entry.resolved.Add(-2 * BranchSHATTL)
		branchSHACache.entries[key] = entry
	}
	branchSHACache.Unlock()
	if _, err := ResolveSHACached(context.Background(), repo); err != nil {
		t.Fatalf("ResolveSHACached() error = %v", err)
	}
	if calls, _ = os.ReadFile(countPath); len(calls) != 3 {
		t.Fatalf("expired entry was not resolved again: %d calls", len(calls))
	}
	branchSHACache.Lock()
	remaining := len(branchSHACache.entries)
	branchSHACache.Unlock()
	if remaining != 1 {
		t.Fatalf("expired entries were not pruned: %d remain", remaining)
	}
}

// GitHub refuses to distinguish a missing repository from a private one, so
// both arrive as a request to authenticate. With prompts disabled that is what
// git reports, and it must not read as "GitHub is down".
func TestResolveSHAClassifiesUnreadableRepositories(t *testing.T) {
	cases := map[string]struct {
		stderr string
		want   error
	}{
		"auth prompt": {"fatal: could not read Username for 'https://github.com': terminal prompts disabled", ErrRepoNotFound},
		"not found":   {"remote: Repository not found.", ErrRepoNotFound},
		"network":     {"fatal: unable to access 'https://github.com/': Could not resolve host", ErrRemoteUnavailable},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			stub := filepath.Join(dir, "fake-git")
			if err := os.WriteFile(stub, []byte("#!/bin/sh\necho "+strconv.Quote(tc.stderr)+" >&2\nexit 128\n"), 0o700); err != nil {
				t.Fatal(err)
			}
			oldBinary := gitBinary
			gitBinary = stub
			t.Cleanup(func() { gitBinary = oldBinary })

			_, err := ResolveSHA(context.Background(), Repo{Owner: "owner", Name: "demo", Branch: "main"})
			if !errors.Is(err, tc.want) {
				t.Fatalf("ResolveSHA() error = %v, want %v", err, tc.want)
			}
		})
	}
}
