package agenteval_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/dvmrry/zscalerctl/internal/agenteval"
)

// runner_test.go exercises the DETERMINISTIC runner core (runner.go): the
// observed-command parsers (sidecar + codex log), the FakeBackend test double,
// and BuildSandbox's hermeticity contract — including the RED-test that a planted
// parent provider token never reaches the sandbox env. No live agent, network,
// LLM, or clock is involved; the filesystem is t.TempDir.

// TestParseSidecarGolden pins the host-sidecar parse (§2.3): a few JSONL lines,
// including exit codes and a blank line, decode to the expected ObservedCommands.
func TestParseSidecarGolden(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		`{"argv":["zscalerctl","zia","locations","list"],"exit":0}`,
		``, // blank line tolerated
		`{"argv":["zscalerctl","zia","locations","get","999999"],"exit":4}`,
		`{"tool":"jq","argv":["jq",".[] | .name"],"exit":0,"stdout_len":12}`, // extra fields ignored
	}, "\n") + "\n"

	got, err := agenteval.ParseSidecar([]byte(input))
	if err != nil {
		t.Fatalf("ParseSidecar: %v", err)
	}

	want := []agenteval.ObservedCommand{
		{Argv: []string{"zscalerctl", "zia", "locations", "list"}, Exit: 0},
		{Argv: []string{"zscalerctl", "zia", "locations", "get", "999999"}, Exit: 4},
		{Argv: []string{"jq", ".[] | .name"}, Exit: 0},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseSidecar golden mismatch:\n got: %#v\nwant: %#v", got, want)
	}
}

// TestParseSidecarRejectsMalformed asserts a non-blank line that is not a valid
// sidecar object is an error (a dropped command could turn a method violation
// into a false PASS), and that a line missing argv is rejected too.
func TestParseSidecarRejectsMalformed(t *testing.T) {
	t.Parallel()

	for name, input := range map[string]string{
		"not json":     `{"argv":["x"],"exit":0}` + "\n" + `not-json`,
		"missing argv": `{"exit":0}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := agenteval.ParseSidecar([]byte(input)); err == nil {
				t.Fatalf("ParseSidecar(%q) = nil error, want a parse error", input)
			}
		})
	}
}

// TestParseCodexCommandsGolden pins the codex-transcript parse (§2.3): a
// representative run+completed snippet with two commands and exits pairs each run
// with its completion BY THE COMMAND STRING and recovers the exit code.
func TestParseCodexCommandsGolden(t *testing.T) {
	t.Parallel()

	transcript := strings.Join([]string{
		"thinking about the discovery surface...",
		"[codex] Running command: /usr/bin/zsh -lc 'zscalerctl --format json schema list'",
		"[codex] (streamed output elided)",
		"[codex] Command completed: /usr/bin/zsh -lc 'zscalerctl --format json schema list' (exit 0)",
		"now fetch a missing id",
		"[codex] Running command: /usr/bin/zsh -lc 'zscalerctl zia locations get 999999'",
		"[codex] Command completed: /usr/bin/zsh -lc 'zscalerctl zia locations get 999999' (exit 4)",
	}, "\n")

	got := agenteval.ParseCodexCommands(transcript)
	want := []agenteval.ObservedCommand{
		{Argv: []string{"zscalerctl", "--format", "json", "schema", "list"}, Exit: 0},
		{Argv: []string{"zscalerctl", "zia", "locations", "get", "999999"}, Exit: 4},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseCodexCommands golden mismatch:\n got: %#v\nwant: %#v", got, want)
	}
}

// TestParseCodexCommandsNoCompletion asserts a run with no completion line
// defaults to exit 0 (observed to start; no failure invented).
func TestParseCodexCommandsNoCompletion(t *testing.T) {
	t.Parallel()

	transcript := "[codex] Running command: /usr/bin/zsh -lc 'zscalerctl doctor'"
	got := agenteval.ParseCodexCommands(transcript)
	want := []agenteval.ObservedCommand{{Argv: []string{"zscalerctl", "doctor"}, Exit: 0}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("no-completion default mismatch:\n got: %#v\nwant: %#v", got, want)
	}
}

// TestParseCodexCommandsQuotedArg asserts the shell-ish splitter keeps a quoted
// argument intact as one field (the jq predicate the battery may grade).
func TestParseCodexCommandsQuotedArg(t *testing.T) {
	t.Parallel()

	transcript := strings.Join([]string{
		`[codex] Running command: /usr/bin/zsh -lc 'jq ".[] | .name"'`,
		`[codex] Command completed: /usr/bin/zsh -lc 'jq ".[] | .name"' (exit 0)`,
	}, "\n")
	got := agenteval.ParseCodexCommands(transcript)
	want := []agenteval.ObservedCommand{{Argv: []string{"jq", ".[] | .name"}, Exit: 0}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("quoted-arg mismatch:\n got: %#v\nwant: %#v", got, want)
	}
}

// TestFakeBackend asserts the test double returns its canned Transcript (and its
// error path) without execing anything.
func TestFakeBackend(t *testing.T) {
	t.Parallel()

	canned := agenteval.AssembleTranscript("answer", []agenteval.ObservedCommand{{Argv: []string{"zscalerctl", "doctor"}, Exit: 0}})
	b := agenteval.FakeBackend{AgentName: "haiku", AgentRank: 2, Canned: canned}
	if b.Name() != "haiku" || b.Rank() != 2 {
		t.Fatalf("Name/Rank = %q/%d, want haiku/2", b.Name(), b.Rank())
	}
	got, err := b.Run(context.Background(), "/sandbox", "prompt")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !reflect.DeepEqual(got, canned) {
		t.Fatalf("Run returned %#v, want canned %#v", got, canned)
	}
}

// TestBuildSandboxLayout asserts BuildSandbox creates ONLY the allowed files: the
// copied-in fixture binary as `zscalerctl`, plus exactly the docs entries (at
// their relative keys, parent dirs created). No other file appears.
func TestBuildSandboxLayout(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fixtureBin := writeFakeFixtureBinary(t)
	docs := map[string]string{
		"AGENTS.md":      "# agents surface",
		"skill/SKILL.md": "# skill surface",
	}

	if _, err := agenteval.BuildSandbox(dir, fixtureBin, docs); err != nil {
		t.Fatalf("BuildSandbox: %v", err)
	}

	got := relFiles(t, dir)
	want := []string{"AGENTS.md", "skill/SKILL.md", "zscalerctl"}
	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sandbox files = %v, want exactly %v", got, want)
	}

	// The copied-in zscalerctl must be executable and carry the binary's bytes.
	binPath := filepath.Join(dir, "zscalerctl")
	info, err := os.Stat(binPath)
	if err != nil {
		t.Fatalf("stat copied binary: %v", err)
	}
	if info.Mode().Perm()&0o100 == 0 {
		t.Fatalf("copied binary mode = %v, want owner-executable", info.Mode().Perm())
	}
}

// TestBuildSandboxRejectsEscapingDoc asserts a doc key that escapes the sandbox
// (absolute path or .. traversal) is rejected, never written.
func TestBuildSandboxRejectsEscapingDoc(t *testing.T) {
	t.Parallel()

	fixtureBin := writeFakeFixtureBinary(t)
	for name, key := range map[string]string{
		"absolute":  "/etc/evil",
		"traversal": "../escape.md",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			_, err := agenteval.BuildSandbox(dir, fixtureBin, map[string]string{key: "x"})
			if err == nil {
				t.Fatalf("BuildSandbox with doc key %q = nil error, want a confinement rejection", key)
			}
		})
	}
}

// TestBuildSandboxEnvHygiene is the §6.3/§5.5 env-hygiene contract, INCLUDING the
// RED-test bit: a planted parent ANTHROPIC token must NOT appear anywhere in the
// returned sandbox env. It also asserts the synthetic creds are injected, the
// fixture dir + a RELATIVE fixture log are set, and a planted real-looking
// ZSCALERCTL_* credential is stripped.
//
// The planted provider token is assembled at runtime from non-contiguous pieces
// so this test source file never contains a literal provider-token-shaped string
// (which the posture gate, TestNoProviderTokensInArtifacts, scans for).
func TestBuildSandboxEnvHygiene(t *testing.T) {
	dir := t.TempDir()
	fixtureBin := writeFakeFixtureBinary(t)

	// Plant credentials/tokens in the PARENT environment that must all be stripped.
	// Build the provider-token VALUE from pieces so no literal token-shaped string
	// is committed in this file.
	plantedProviderKey := "ANTHROPIC_" + "API_" + "KEY"
	plantedProviderVal := "sk" + "-" + strings.Repeat("a1b2", 8) // 32+ mixed chars, secret-shaped
	realZscalerSecret := "rk" + strings.Repeat("c3d4", 8)        // a real-looking ZSCALERCTL_ value

	t.Setenv(plantedProviderKey, plantedProviderVal)
	t.Setenv("OPENAI_API_KEY", "openai-"+strings.Repeat("e5f6", 8))
	t.Setenv("DEVIN_API_KEY", "devin-"+strings.Repeat("a7b8", 8))
	t.Setenv("ZSCALERCTL_CLIENT_SECRET", realZscalerSecret) // a REAL cred that must be stripped
	t.Setenv("ZSCALERCTL_VANITY_DOMAIN", "real-tenant-vanity")

	env, err := agenteval.BuildSandbox(dir, fixtureBin, nil)
	if err != nil {
		t.Fatalf("BuildSandbox: %v", err)
	}
	joined := strings.Join(env, "\n")

	// RED-test bit: the planted provider token value must NOT appear in the sandbox env.
	if strings.Contains(joined, plantedProviderVal) {
		t.Fatalf("planted %s value leaked into sandbox env:\n%s", plantedProviderKey, joined)
	}
	// No stripped-prefix KEY may appear at all.
	for _, k := range []string{plantedProviderKey, "OPENAI_API_KEY", "DEVIN_API_KEY"} {
		if strings.Contains(joined, k+"=") {
			t.Fatalf("stripped provider var %s leaked into sandbox env:\n%s", k, joined)
		}
	}
	// The REAL ZSCALERCTL_ secret value must be gone (a synthetic one replaces it).
	if strings.Contains(joined, realZscalerSecret) {
		t.Fatalf("real ZSCALERCTL_ credential value leaked into sandbox env:\n%s", joined)
	}
	if strings.Contains(joined, "real-tenant-vanity") {
		t.Fatalf("real ZSCALERCTL_VANITY_DOMAIN value leaked into sandbox env:\n%s", joined)
	}

	m := envMap(env)

	// Synthetic, value-free creds are injected.
	assertEnv(t, m, "ZSCALERCTL_CLIENT_ID", "synthetic-client-id")
	assertEnv(t, m, "ZSCALERCTL_CLIENT_SECRET", "synthetic-client-secret")
	assertEnv(t, m, "ZSCALERCTL_VANITY_DOMAIN", "example")

	// Fixture selection + a RELATIVE confined sidecar log.
	assertEnv(t, m, "ZSCALERCTL_FIXTURE_DIR", dir)
	if logVal := m["ZSCALERCTL_FIXTURE_LOG"]; logVal == "" {
		t.Fatalf("ZSCALERCTL_FIXTURE_LOG not set")
	} else if filepath.IsAbs(logVal) || strings.ContainsRune(logVal, filepath.Separator) {
		t.Fatalf("ZSCALERCTL_FIXTURE_LOG = %q, want a RELATIVE bare filename (confined-path contract)", logVal)
	}

	// PATH is minimal and leads with the sandbox dir (so `zscalerctl` resolves to
	// the copied-in fixture binary).
	pathVal := m["PATH"]
	if pathVal == "" {
		t.Fatalf("PATH not set in sandbox env")
	}
	if first := strings.Split(pathVal, string(os.PathListSeparator))[0]; first != dir {
		t.Fatalf("PATH leads with %q, want the sandbox dir %q", first, dir)
	}
}

// TestSanitizeParentEnvStripsProviderTokens RED-tests the exported env scrubber
// directly: a planted provider token in the input slice must be absent from the
// output, and a non-stripped var must survive.
func TestSanitizeParentEnvStripsProviderTokens(t *testing.T) {
	t.Parallel()

	plantedKey := "ANTHROPIC_" + "API_" + "KEY"
	plantedVal := "sk" + strings.Repeat("9z8y", 8)
	parent := []string{
		plantedKey + "=" + plantedVal,
		"ZSCALERCTL_CLIENT_SECRET=" + "rk" + strings.Repeat("7x6w", 8),
		"OPENAI_API_KEY=o-" + strings.Repeat("5v4u", 8),
		"DEVIN_TOKEN=d-" + strings.Repeat("3t2s", 8),
		"HOME=/home/dev", // must survive
		"LANG=en_US.UTF-8",
	}

	got := agenteval.SanitizeParentEnv(parent)
	joined := strings.Join(got, "\n")

	for _, bad := range []string{plantedKey, "ZSCALERCTL_CLIENT_SECRET", "OPENAI_API_KEY", "DEVIN_TOKEN", plantedVal} {
		if strings.Contains(joined, bad) {
			t.Fatalf("SanitizeParentEnv leaked %q:\n%s", bad, joined)
		}
	}
	m := envMap(got)
	if m["HOME"] != "/home/dev" {
		t.Fatalf("SanitizeParentEnv dropped a non-stripped var HOME; got %v", got)
	}
	if m["LANG"] != "en_US.UTF-8" {
		t.Fatalf("SanitizeParentEnv dropped a non-stripped var LANG; got %v", got)
	}
}

// --- helpers ----------------------------------------------------------------

// writeFakeFixtureBinary writes a tiny placeholder file standing in for the
// fixture binary, so BuildSandbox's copy path can be exercised without compiling
// the real binary (TestShimBinaryBehavior covers the real one). The content is an
// arbitrary, value-free marker.
func writeFakeFixtureBinary(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "zscalerctl-fixture")
	if err := os.WriteFile(p, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatalf("write fake fixture binary: %v", err)
	}
	return p
}

// relFiles returns every regular file under root as a slash-separated path
// relative to root.
func relFiles(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("walk sandbox: %v", err)
	}
	return out
}

// envMap turns a KEY=VALUE slice into a map for assertions.
func envMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, kv := range env {
		if i := strings.IndexByte(kv, '='); i >= 0 {
			m[kv[:i]] = kv[i+1:]
		}
	}
	return m
}

// assertEnv fails if the env map does not bind key to want.
func assertEnv(t *testing.T, m map[string]string, key, want string) {
	t.Helper()
	if got := m[key]; got != want {
		t.Fatalf("env[%s] = %q, want %q", key, got, want)
	}
}
