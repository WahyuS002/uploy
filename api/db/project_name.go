package db

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/WahyuS002/uploy/db/sqlcgen"
)

// projectNameAdjectives and projectNameNouns are curated ASCII, lowercase,
// single-word lists used to build Railway-style project names of the form
// `adjective-noun` (e.g. `reliable-luck`). Keep entries friendly, neutral,
// and unambiguous; avoid words that combine awkwardly or carry loaded
// meanings.
var projectNameAdjectives = []string{
	"able", "ace", "agile", "alert", "amber", "ample", "apt",
	"balmy", "bold", "brave", "breezy", "brisk", "bright", "calm",
	"cheery", "civic", "clean", "clear", "clever", "cosmic", "cozy",
	"crisp", "daring", "deft", "eager", "early", "easy", "epic",
	"fair", "fancy", "fast", "fine", "firm", "fluent", "fond",
	"free", "fresh", "funky", "gentle", "glad", "golden", "good",
	"grand", "green", "handy", "happy", "hardy", "jolly", "keen",
	"kind", "loyal", "lucky", "mellow", "merry", "mighty", "modern",
	"neat", "nice", "noble", "plucky", "polite", "proud", "quick",
	"quiet", "rapid", "ready", "reliable", "rich", "royal", "sharp",
	"silky", "silver", "simple", "smart", "snappy", "solid", "sound",
	"spry", "steady", "stellar", "strong", "sturdy", "subtle", "sunny",
	"super", "swift", "tidy", "trusty", "urban", "vivid", "warm",
	"wise", "witty", "zesty",
}

var projectNameNouns = []string{
	"acorn", "agent", "amber", "anchor", "apex", "arrow", "atlas",
	"beacon", "boulder", "branch", "breeze", "brook", "canyon", "cascade",
	"cedar", "cinder", "cipher", "cobble", "comet", "crest", "cyclone",
	"delta", "dune", "echo", "ember", "falcon", "feather", "fern",
	"field", "finch", "flame", "fleet", "flint", "forest", "fox",
	"garnet", "glade", "glider", "granite", "grove", "harbor", "hawk",
	"haze", "heron", "horizon", "ivy", "lagoon", "lake", "lantern",
	"leaf", "ledger", "luck", "marble", "meadow", "meteor", "moss",
	"nebula", "nest", "oak", "orbit", "otter", "panda", "pebble",
	"petal", "pine", "pixel", "planet", "plateau", "plume", "prairie",
	"quartz", "quokka", "raven", "reef", "ridge", "river", "robin",
	"saddle", "sage", "sail", "shore", "signal", "spark", "spire",
	"sprout", "stone", "summit", "thistle", "tide", "timber", "torch",
	"trail", "valley", "vista", "willow", "wolf", "yacht", "zephyr",
}

// maxGeneratorAttempts bounds how many unique adjective+noun candidates are
// tried before falling back to a numeric suffix. With the curated lists above
// we have >7k combinations so collisions in the base form are rare.
const maxGeneratorAttempts = 16

// pickWord returns a random element from words using crypto/rand so
// generation is not biased by a predictable seed.
func pickWord(words []string) (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(words))))
	if err != nil {
		return "", err
	}
	return words[n.Int64()], nil
}

// generateCandidateName returns a single `adjective-noun` candidate. It is
// exported via the unique-name helpers; tests should call
// GenerateUniqueProjectNameForTest to exercise collision handling.
func generateCandidateName() (string, error) {
	adj, err := pickWord(projectNameAdjectives)
	if err != nil {
		return "", err
	}
	noun, err := pickWord(projectNameNouns)
	if err != nil {
		return "", err
	}
	return adj + "-" + noun, nil
}

// nameExistsFn reports whether a project name is already taken within the
// caller's scope. It is injected so the generation logic can be unit tested
// without a live database.
type nameExistsFn func(ctx context.Context, name string) (bool, error)

// generateUniqueProjectName produces a Railway-style project name guaranteed
// to be unique according to the supplied existence check. It first tries up
// to maxGeneratorAttempts distinct `adjective-noun` pairs. If every candidate
// collides it anchors on the last candidate and appends `-2`, `-3`, ... until
// a free name is found.
func generateUniqueProjectName(ctx context.Context, exists nameExistsFn) (string, error) {
	var last string
	for i := 0; i < maxGeneratorAttempts; i++ {
		candidate, err := generateCandidateName()
		if err != nil {
			return "", err
		}
		last = candidate
		taken, err := exists(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
	}
	for suffix := 2; ; suffix++ {
		candidate := fmt.Sprintf("%s-%d", last, suffix)
		taken, err := exists(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
	}
}

// workspaceNameExists builds an existence check bound to a workspace and a
// specific sqlc Queries handle (so the caller can pass a tx-scoped Queries
// and keep the check inside the same transaction as the insert).
func workspaceNameExists(q *sqlcgen.Queries, workspaceID string) nameExistsFn {
	return func(ctx context.Context, name string) (bool, error) {
		return q.ProjectNameExistsInWorkspace(ctx, sqlcgen.ProjectNameExistsInWorkspaceParams{
			WorkspaceID: workspaceID,
			Name:        name,
		})
	}
}
