// Package source resolves public GitHub repositories and prepares Railpack
// build plans without coupling the source workflow to HTTP handlers or the DB.
package source

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const MaxTarballSize int64 = 500 * 1024 * 1024

var (
	ErrInvalidRepoURL    = errors.New("invalid GitHub repository URL")
	ErrInvalidBranch     = errors.New("invalid branch name")
	ErrBranchNotFound    = errors.New("branch not found")
	ErrRemoteUnavailable = errors.New("remote repository unavailable")
	ErrSourceTooLarge    = errors.New("repository tarball exceeds 500 MB")
	ErrUnsupportedSource = errors.New("Railpack could not determine how to build this repository")

	gitBinary            = "git"
	railpackBinary       = "railpack"
	codeloadBaseURL      = "https://codeload.github.com"
	httpClient           = http.DefaultClient
	githubSegmentPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)
	commitPattern        = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)
)

type Repo struct {
	Owner  string
	Name   string
	Branch string
}

// Plan keeps the exact JSON produced by railpack. The build frontend consumes
// this document directly, so normalising it into a Go struct would be lossy.
type Plan struct {
	Raw json.RawMessage
}

type Info struct {
	Provider        string            `json:"provider"`
	RuntimeVersions map[string]string `json:"runtime_versions"`
	StartCommand    string            `json:"start_command,omitempty"`
}

type Analysis struct {
	Repo Repo
	SHA  string
	Plan Plan
	Info Info
}

// Analyzer makes the HTTP layer testable without substituting source analysis
// results supplied by a client. Production uses DefaultAnalyzer.
type Analyzer interface {
	Analyze(ctx context.Context, repo Repo) (Analysis, error)
}

// EnvAnalyzer is implemented by analyzers that can make build plans using the
// service's decrypted environment values.
type EnvAnalyzer interface {
	AnalyzeWithEnv(ctx context.Context, repo Repo, env map[string]string) (Analysis, error)
}

type DefaultAnalyzer struct{}

func (a DefaultAnalyzer) Analyze(ctx context.Context, repo Repo) (Analysis, error) {
	return a.AnalyzeWithEnv(ctx, repo, nil)
}

func (DefaultAnalyzer) AnalyzeWithEnv(ctx context.Context, repo Repo, env map[string]string) (Analysis, error) {
	sha, err := ResolveSHA(ctx, repo)
	if err != nil {
		return Analysis{}, err
	}
	dir, cleanup, err := Fetch(ctx, repo, sha)
	if err != nil {
		return Analysis{}, err
	}
	defer cleanup()
	plan, info, err := Prepare(ctx, dir, env)
	if err != nil {
		return Analysis{}, err
	}
	return Analysis{Repo: repo, SHA: sha, Plan: plan, Info: info}, nil
}

type commandError struct {
	name   string
	output string
	err    error
}

func (e *commandError) Error() string {
	if e.output == "" {
		return fmt.Sprintf("%s failed: %v", e.name, e.err)
	}
	return fmt.Sprintf("%s failed: %v: %s", e.name, e.err, strings.TrimSpace(e.output))
}

func (e *commandError) Unwrap() error { return e.err }

// ParseRepoURL accepts the canonical HTTPS URL, a host without a scheme, and
// the short owner/name form. Only public github.com repositories are allowed.
func ParseRepoURL(raw string) (Repo, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.ContainsAny(raw, "\r\n") {
		return Repo{}, ErrInvalidRepoURL
	}
	if !strings.Contains(raw, "://") {
		if strings.HasPrefix(strings.ToLower(raw), "github.com/") {
			raw = "https://" + raw
		} else if strings.Count(strings.Trim(raw, "/"), "/") == 1 {
			raw = "https://github.com/" + raw
		} else {
			raw = "https://" + raw
		}
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || !strings.EqualFold(u.Host, "github.com") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return Repo{}, ErrInvalidRepoURL
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return Repo{}, ErrInvalidRepoURL
	}
	name := strings.TrimSuffix(parts[1], ".git")
	if !githubSegmentPattern.MatchString(parts[0]) || !githubSegmentPattern.MatchString(name) {
		return Repo{}, ErrInvalidRepoURL
	}
	return Repo{Owner: parts[0], Name: name}, nil
}

func (r Repo) cloneURL() string {
	return "https://github.com/" + r.Owner + "/" + r.Name + ".git"
}

func (r Repo) branch() (string, error) {
	branch := strings.TrimSpace(r.Branch)
	if branch == "" {
		return "main", nil
	}
	if strings.ContainsAny(branch, "\r\n") || strings.HasPrefix(branch, "-") || strings.Contains(branch, "..") || strings.Contains(branch, "//") {
		return "", ErrInvalidBranch
	}
	return branch, nil
}

// ResolveSHA resolves a branch through git's wire protocol. It does not use
// the GitHub API, so GitHub's unauthenticated API quota is not consumed.
func ResolveSHA(ctx context.Context, r Repo) (string, error) {
	branch, err := r.branch()
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, gitBinary, "ls-remote", r.cloneURL(), "refs/heads/"+branch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", &commandError{name: "git ls-remote", output: string(out), err: fmt.Errorf("%w: %v", ErrRemoteUnavailable, err)}
	}
	fields := strings.Fields(string(out))
	if len(fields) == 0 || !commitPattern.MatchString(fields[0]) {
		return "", fmt.Errorf("%w: %s", ErrBranchNotFound, branch)
	}
	return strings.ToLower(fields[0]), nil
}

// Fetch downloads and safely extracts the SHA-addressed GitHub tarball. The
// returned cleanup function removes the complete temporary tree.
func Fetch(ctx context.Context, r Repo, sha string) (dir string, cleanup func(), err error) {
	if !commitPattern.MatchString(sha) {
		return "", nil, fmt.Errorf("invalid commit SHA")
	}
	root, err := os.MkdirTemp("", "uploy-source-")
	if err != nil {
		return "", nil, err
	}
	removeRoot := func() { _ = os.RemoveAll(root) }

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, codeloadBaseURL+"/"+url.PathEscape(r.Owner)+"/"+url.PathEscape(r.Name)+"/tar.gz/"+sha, nil)
	if err != nil {
		removeRoot()
		return "", nil, fmt.Errorf("%w: %v", ErrRemoteUnavailable, err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		removeRoot()
		return "", nil, fmt.Errorf("%w: %v", ErrRemoteUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		removeRoot()
		return "", nil, fmt.Errorf("%w: codeload returned HTTP %d", ErrRemoteUnavailable, resp.StatusCode)
	}
	if resp.ContentLength > MaxTarballSize {
		removeRoot()
		return "", nil, ErrSourceTooLarge
	}

	tarPath := filepath.Join(root, "source.tar.gz")
	f, err := os.OpenFile(tarPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		removeRoot()
		return "", nil, err
	}
	n, copyErr := io.Copy(f, io.LimitReader(resp.Body, MaxTarballSize+1))
	closeErr := f.Close()
	if copyErr != nil {
		removeRoot()
		return "", nil, fmt.Errorf("%w: %v", ErrRemoteUnavailable, copyErr)
	}
	if closeErr != nil {
		removeRoot()
		return "", nil, closeErr
	}
	if n > MaxTarballSize {
		removeRoot()
		return "", nil, ErrSourceTooLarge
	}

	if err := extractTarball(tarPath, root); err != nil {
		removeRoot()
		return "", nil, err
	}
	expected := filepath.Join(root, r.Name+"-"+sha)
	if info, statErr := os.Stat(expected); statErr != nil || !info.IsDir() {
		removeRoot()
		return "", nil, fmt.Errorf("%w: tarball did not contain %s", ErrRemoteUnavailable, r.Name+"-"+sha)
	}
	_ = os.Remove(tarPath)
	return expected, removeRoot, nil
}

func extractTarball(tarPath, root string) error {
	f, err := os.Open(tarPath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("invalid repository tarball: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("invalid repository tarball: %w", err)
		}
		name := filepath.Clean(filepath.FromSlash(hdr.Name))
		if name == "." || filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
			return fmt.Errorf("invalid repository tarball path %q", hdr.Name)
		}
		target := filepath.Join(root, name)
		if !within(root, target) {
			return fmt.Errorf("invalid repository tarball path %q", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeXGlobalHeader, tar.TypeXHeader, tar.TypeGNULongName, tar.TypeGNULongLink:
			// Metadata entries are consumed by archive/tar and do not represent
			// files that belong in the extracted source tree.
			continue
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			mode := hdr.FileInfo().Mode().Perm()
			if mode == 0 {
				mode = 0o644
			}
			out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|os.O_EXCL, mode)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, tr)
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		case tar.TypeSymlink, tar.TypeLink:
			return fmt.Errorf("repository tarball contains unsupported link %q", hdr.Name)
		default:
			return fmt.Errorf("repository tarball contains unsupported entry %q", hdr.Name)
		}
	}
}

func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// Prepare runs Railpack's analysis phase and parses only the fields used by
// the API. The raw plan is retained verbatim for the later build phase.
func Prepare(ctx context.Context, dir string, env map[string]string) (Plan, Info, error) {
	planFile, err := os.CreateTemp("", "uploy-plan-*.json")
	if err != nil {
		return Plan{}, Info{}, err
	}
	planPath := planFile.Name()
	if err := planFile.Close(); err != nil {
		return Plan{}, Info{}, err
	}
	defer os.Remove(planPath)

	infoFile, err := os.CreateTemp("", "uploy-info-*.json")
	if err != nil {
		return Plan{}, Info{}, err
	}
	infoPath := infoFile.Name()
	if err := infoFile.Close(); err != nil {
		return Plan{}, Info{}, err
	}
	defer os.Remove(infoPath)

	args := []string{"prepare", dir, "--plan-out", planPath, "--info-out", infoPath, "--hide-pretty-plan"}
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !validEnvName(key) {
			return Plan{}, Info{}, fmt.Errorf("invalid environment variable name %q", key)
		}
		args = append(args, "--env", key+"="+env[key])
	}
	cmd := exec.CommandContext(ctx, railpackBinary, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Plan{}, Info{}, &commandError{name: "railpack prepare", output: string(out), err: err}
	}

	rawPlan, err := os.ReadFile(planPath)
	if err != nil {
		return Plan{}, Info{}, fmt.Errorf("read railpack plan: %w", err)
	}
	if !json.Valid(rawPlan) {
		return Plan{}, Info{}, fmt.Errorf("railpack produced invalid plan JSON")
	}
	rawInfo, err := os.ReadFile(infoPath)
	if err != nil {
		return Plan{}, Info{}, fmt.Errorf("read railpack info: %w", err)
	}
	info, err := parseInfo(rawInfo, rawPlan)
	if err != nil {
		return Plan{}, Info{}, err
	}
	return Plan{Raw: json.RawMessage(rawPlan)}, info, nil
}

func parseInfo(rawInfo, rawPlan []byte) (Info, error) {
	var decoded struct {
		ResolvedPackages map[string]struct {
			ResolvedVersion string `json:"resolvedVersion"`
		} `json:"resolvedPackages"`
		Metadata struct {
			Providers string `json:"providers"`
		} `json:"metadata"`
		DetectedProviders []string `json:"detectedProviders"`
	}
	if err := json.Unmarshal(rawInfo, &decoded); err != nil {
		return Info{}, fmt.Errorf("parse railpack info: %w", err)
	}
	provider := ""
	if len(decoded.DetectedProviders) > 0 {
		provider = decoded.DetectedProviders[0]
	}
	if provider == "" {
		provider = strings.TrimSpace(strings.Split(decoded.Metadata.Providers, ",")[0])
	}
	if provider == "" {
		return Info{}, ErrUnsupportedSource
	}
	versions := make(map[string]string)
	for _, name := range strings.Split(decoded.Metadata.Providers, ",") {
		name = strings.TrimSpace(name)
		if pkg, ok := decoded.ResolvedPackages[name]; ok && pkg.ResolvedVersion != "" {
			versions[name] = pkg.ResolvedVersion
		}
	}
	if len(versions) == 0 {
		if pkg, ok := decoded.ResolvedPackages[provider]; ok && pkg.ResolvedVersion != "" {
			versions[provider] = pkg.ResolvedVersion
		}
	}
	var plan struct {
		Deploy struct {
			StartCommand string `json:"startCommand"`
		} `json:"deploy"`
	}
	if err := json.Unmarshal(rawPlan, &plan); err != nil {
		return Info{}, fmt.Errorf("parse railpack plan: %w", err)
	}
	return Info{Provider: provider, RuntimeVersions: versions, StartCommand: plan.Deploy.StartCommand}, nil
}

func validEnvName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if !(r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || i > 0 && r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

func SuggestedName(repoName string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(repoName) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			if b.Len() > 0 {
				b.WriteByte('-')
			}
		}
	}
	name := strings.Trim(b.String(), "-")
	if name == "" {
		return "service"
	}
	return name
}

func SuggestedPort(provider string) int {
	provider = strings.ToLower(strings.TrimSpace(strings.Split(provider, ",")[0]))
	switch provider {
	case "node", "bun", "deno":
		return 3000
	case "python":
		return 8000
	case "go", "java":
		return 8080
	case "staticfile", "staticfiles", "static-file":
		return 80
	default:
		return 3000
	}
}
