package agenteval_test

import (
	"os"
	"path/filepath"
	"testing"
)

// repoRoot locates the module root robustly by walking up from the test's
// working directory (which `go test` sets to the package dir,
// internal/agenteval) until it finds the directory containing go.mod. The
// integrity gates read committed docs and source from anchored repo-relative
// paths, so they must not assume any particular cwd depth.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("walked to filesystem root without finding go.mod (started from %q)", dir)
		}
		dir = parent
	}
}
