package db

import (
	"context"
	"regexp"
	"strings"
	"testing"
)

// TestGenerateCandidateName_FormatAndContent verifies the generated candidate
// is ASCII lowercase kebab-case in the exact `adjective-noun` form and that
// each half is drawn from the curated word lists.
func TestGenerateCandidateName_FormatAndContent(t *testing.T) {
	adjSet := wordSet(projectNameAdjectives)
	nounSet := wordSet(projectNameNouns)
	kebab := regexp.MustCompile(`^[a-z]+-[a-z]+$`)

	for i := 0; i < 200; i++ {
		name, err := generateCandidateName()
		if err != nil {
			t.Fatalf("generateCandidateName: %v", err)
		}
		if !kebab.MatchString(name) {
			t.Fatalf("name %q does not match adjective-noun kebab form", name)
		}
		parts := strings.SplitN(name, "-", 2)
		if _, ok := adjSet[parts[0]]; !ok {
			t.Fatalf("adjective %q not in curated list", parts[0])
		}
		if _, ok := nounSet[parts[1]]; !ok {
			t.Fatalf("noun %q not in curated list", parts[1])
		}
	}
}

// TestGenerateUniqueProjectName_AcceptsFirstFreeCandidate verifies the happy
// path: the very first candidate is not taken, so it is returned as-is.
func TestGenerateUniqueProjectName_AcceptsFirstFreeCandidate(t *testing.T) {
	var calls int
	exists := func(_ context.Context, _ string) (bool, error) {
		calls++
		return false, nil
	}
	name, err := generateUniqueProjectName(context.Background(), exists)
	if err != nil {
		t.Fatalf("generateUniqueProjectName: %v", err)
	}
	if name == "" {
		t.Fatal("expected non-empty name")
	}
	if calls != 1 {
		t.Fatalf("expected existence check to run once, ran %d", calls)
	}
}

// TestGenerateUniqueProjectName_RetriesNewPairsOnCollision verifies that when
// the first candidate is taken, the generator tries fresh `adjective-noun`
// pairs before falling back to a numeric suffix.
func TestGenerateUniqueProjectName_RetriesNewPairsOnCollision(t *testing.T) {
	var seen []string
	exists := func(_ context.Context, name string) (bool, error) {
		seen = append(seen, name)
		// first two candidates taken, third free
		return len(seen) <= 2, nil
	}
	name, err := generateUniqueProjectName(context.Background(), exists)
	if err != nil {
		t.Fatalf("generateUniqueProjectName: %v", err)
	}
	if len(seen) != 3 {
		t.Fatalf("expected 3 existence checks, got %d (%v)", len(seen), seen)
	}
	if name != seen[2] {
		t.Fatalf("returned name %q != last checked %q", name, seen[2])
	}
	// all three must be plain adjective-noun pairs (no numeric suffix phase)
	kebab := regexp.MustCompile(`^[a-z]+-[a-z]+$`)
	for _, n := range seen {
		if !kebab.MatchString(n) {
			t.Fatalf("candidate %q should still be plain adjective-noun form", n)
		}
	}
}

// TestGenerateUniqueProjectName_FallsBackToNumericSuffix verifies that when
// every adjective-noun pair collides for maxGeneratorAttempts, the generator
// anchors on the last candidate and appends `-2`, `-3`, ... until a free
// name is found.
func TestGenerateUniqueProjectName_FallsBackToNumericSuffix(t *testing.T) {
	var attempts []string
	exists := func(_ context.Context, name string) (bool, error) {
		attempts = append(attempts, name)
		// First maxGeneratorAttempts base candidates collide.
		if len(attempts) <= maxGeneratorAttempts {
			return true, nil
		}
		// Then `-2` also collides; `-3` is free.
		if strings.HasSuffix(name, "-2") {
			return true, nil
		}
		return false, nil
	}
	name, err := generateUniqueProjectName(context.Background(), exists)
	if err != nil {
		t.Fatalf("generateUniqueProjectName: %v", err)
	}
	if !strings.HasSuffix(name, "-3") {
		t.Fatalf("expected numeric suffix `-3`, got %q", name)
	}
	base := attempts[maxGeneratorAttempts-1]
	if name != base+"-3" {
		t.Fatalf("expected suffix anchored on last base %q, got %q", base, name)
	}
}

func wordSet(words []string) map[string]struct{} {
	s := make(map[string]struct{}, len(words))
	for _, w := range words {
		s[w] = struct{}{}
	}
	return s
}
