package agenteval_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// posture_test.go turns the eval's "value-free" posture into a measured number
// (§5.5) instead of a prose claim. It statically scans every committed text
// artifact under internal/agenteval/ for secret-shaped and provider-token-shaped
// strings. The regexes are deliberately conservative: they require
// high-entropy, contiguous, mixed-character runs so they do not fire on normal
// Go identifiers, import paths, long CamelCase test names, or prose. The only
// secret-shaped tokens that may appear are the obviously-synthetic, explicitly
// allow-listed sentinels below.

// agentevalScanExtensions is the set of committed text artifact types the
// posture gates scan. Binary artifacts and directories are skipped.
var agentevalScanExtensions = map[string]bool{
	".go":     true,
	".json":   true,
	".md":     true,
	".ndjson": true,
}

// allowedSyntheticTokens are the obviously-synthetic, value-free sentinels that
// are permitted to appear in the corpus and tests. They are NOT secrets: the
// canary is a fixed CANARY-prefixed placeholder whose whole purpose is to prove
// projection drops a ClassSecret field without risking a real leak, and the
// synthetic-* creds are literal placeholders that pass validation but never
// reach an endpoint (the fixture reader serves the data). Each is allow-listed
// explicitly so the gate's "what counts as fake" is auditable.
var allowedSyntheticTokens = []string{
	"CANARY-secret-preSharedKey", // fixtures.SecretCanary
	"synthetic-client-id",        // shim_binary_test.go synthetic OneAPI creds
	"synthetic-client-secret",    // shim_binary_test.go synthetic OneAPI creds
}

// secretShapedToken matches a high-entropy, secret-shaped run: a contiguous
// base64/hex-ish token of >=32 chars drawn from the base64+url-safe alphabet.
// On its own this would flag long CamelCase identifiers, so a found token is
// only treated as a secret if it ALSO clears the entropy heuristic in
// looksLikeSecret (mixed case + digits, or a long pure-hex run).
var secretShapedToken = regexp.MustCompile(`[A-Za-z0-9+/=_-]{32,}`)

// longHexRun matches a contiguous hexadecimal run of >=32 chars — the shape of
// an SHA-256 digest, an API hash, or a hex-encoded key.
var longHexRun = regexp.MustCompile(`\b[0-9a-fA-F]{32,}\b`)

// providerToken matches a provider API token: an ANTHROPIC_/OPENAI_/DEVIN_
// prefix immediately followed by a token-shaped run. This is conservative — it
// requires the literal prefix, so it cannot fire on ordinary prose mentioning
// the providers.
var providerToken = regexp.MustCompile(`(?:ANTHROPIC|OPENAI|DEVIN)_[A-Za-z0-9_]*[A-Za-z0-9]{8,}`)

// TestFixturesContainNoRealLookingSecrets asserts no committed artifact under
// internal/agenteval/ carries a secret-shaped token other than the explicitly
// allow-listed synthetic sentinels (§5.5).
func TestFixturesContainNoRealLookingSecrets(t *testing.T) {
	t.Parallel()

	forEachAgentevalArtifact(t, func(t *testing.T, rel, content string) {
		for _, line := range strings.Split(content, "\n") {
			scrubbed := stripAllowed(line)

			for _, tok := range secretShapedToken.FindAllString(scrubbed, -1) {
				if looksLikeSecret(tok) {
					t.Errorf("%s: secret-shaped token %q is not an allow-listed synthetic sentinel; value-free posture requires no real-looking secrets", rel, clip(tok))
				}
			}
			for _, tok := range longHexRun.FindAllString(scrubbed, -1) {
				t.Errorf("%s: hex run %q looks like a digest/key; value-free posture requires no real-looking secrets", rel, clip(tok))
			}
		}
	})
}

// TestNoProviderTokensInArtifacts asserts no committed artifact under
// internal/agenteval/ contains an ANTHROPIC_/OPENAI_/DEVIN_-prefixed
// token-shaped string — the static backstop for the runtime token scrub (§5.5).
func TestNoProviderTokensInArtifacts(t *testing.T) {
	t.Parallel()

	forEachAgentevalArtifact(t, func(t *testing.T, rel, content string) {
		for _, tok := range providerToken.FindAllString(content, -1) {
			t.Errorf("%s: provider-token-shaped string %q must never appear in a committed artifact (runtime scrub backstop)", rel, clip(tok))
		}
	})
}

// looksLikeSecret applies the entropy heuristic that separates a real-looking
// secret from a benign long identifier. A token qualifies only if it is either
// a long pure-hex run, OR mixes upper-case, lower-case, AND digits (the shape of
// a random base64 key, never a CamelCase Go identifier, which lacks digits).
func looksLikeSecret(tok string) bool {
	if longHexRun.MatchString(tok) {
		return true
	}
	var hasUpper, hasLower, hasDigit bool
	for _, r := range tok {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= '0' && r <= '9':
			hasDigit = true
		}
	}
	return hasUpper && hasLower && hasDigit
}

// stripAllowed removes every allow-listed synthetic sentinel from a line before
// the secret scan, so the canary and synthetic creds can never trip the gate
// while any real secret-shaped neighbour still would.
func stripAllowed(line string) string {
	for _, tok := range allowedSyntheticTokens {
		line = strings.ReplaceAll(line, tok, "")
	}
	return line
}

// forEachAgentevalArtifact walks internal/agenteval/ under the repo root and
// invokes fn for each committed text artifact (by extension). It excludes
// nothing within those extensions: goldens, verdicts, fixtures, and battery
// manifests are all in scope as they land.
func forEachAgentevalArtifact(t *testing.T, fn func(t *testing.T, rel, content string)) {
	t.Helper()
	root := repoRoot(t)
	base := filepath.Join(root, "internal", "agenteval")

	err := filepath.WalkDir(base, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !agentevalScanExtensions[filepath.Ext(path)] {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		fn(t, rel, string(data))
		return nil
	})
	if err != nil {
		t.Fatalf("walk internal/agenteval: %v", err)
	}
}

// clip shortens a token for failure messages so a genuinely long secret is not
// echoed in full into CI logs.
func clip(s string) string {
	const max = 16
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(" + strconv.Itoa(len(s)) + " chars)"
}
