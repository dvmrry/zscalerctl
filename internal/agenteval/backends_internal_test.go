package agenteval

// backends_internal_test.go asserts the CONCRETE adapter field mapping and the
// PURE argv builders from inside the package (the adapter structs are
// unexported). It NEVER execs codex/claude — only the deterministic
// roster->struct mapping and argv shape are checked; the live Run methods are
// validated on-demand by a real run.

import (
	"reflect"
	"testing"
)

// TestBackendForRosterEntryFieldMapping asserts the factory threads model + bin
// into the concrete adapter structs.
func TestBackendForRosterEntryFieldMapping(t *testing.T) {
	t.Parallel()

	cfg := BackendFactoryConfig{CodexBin: "/opt/codex", ClaudeBin: "/real/claude"}

	codexLB, err := BackendForRosterEntry(
		RosterEntry{Rank: 1, Agent: "codex-mini", Capture: captureCodexTranscript, Model: "gpt-5.4-mini"}, cfg)
	if err != nil {
		t.Fatalf("codex entry: %v", err)
	}
	cb, ok := codexLB.(codexBackend)
	if !ok {
		t.Fatalf("codex entry produced %T, want codexBackend", codexLB)
	}
	if cb.model != "gpt-5.4-mini" || cb.bin != "/opt/codex" || cb.name != "codex-mini" || cb.rank != 1 {
		t.Fatalf("codexBackend = %+v, want model=gpt-5.4-mini bin=/opt/codex name=codex-mini rank=1", cb)
	}

	claudeLB, err := BackendForRosterEntry(
		RosterEntry{Rank: 2, Agent: "haiku", Capture: captureHostSidecar, Model: "claude-haiku-4-5"}, cfg)
	if err != nil {
		t.Fatalf("claude entry: %v", err)
	}
	clb, ok := claudeLB.(claudeBackend)
	if !ok {
		t.Fatalf("claude entry produced %T, want claudeBackend", claudeLB)
	}
	if clb.model != "claude-haiku-4-5" || clb.bin != "/real/claude" || clb.name != "haiku" || clb.rank != 2 {
		t.Fatalf("claudeBackend = %+v, want model=claude-haiku-4-5 bin=/real/claude name=haiku rank=2", clb)
	}
}

// TestCodexArgsOmitsModelWhenEmpty asserts the load-bearing -m omission: a named
// model includes `-m <model>`; an empty model OMITS -m entirely (codex default =
// the strong tier). The rest of the VERIFIED argv is pinned.
func TestCodexArgsOmitsModelWhenEmpty(t *testing.T) {
	t.Parallel()

	withModel := codexArgs("gpt-5.4-mini", "/sandbox", "PROMPT")
	want := []string{"exec", "-m", "gpt-5.4-mini", "-s", "workspace-write", "--skip-git-repo-check", "-C", "/sandbox", "--json", "PROMPT"}
	if !reflect.DeepEqual(withModel, want) {
		t.Fatalf("codexArgs(model) = %v\nwant %v", withModel, want)
	}

	noModel := codexArgs("", "/sandbox", "PROMPT")
	wantNo := []string{"exec", "-s", "workspace-write", "--skip-git-repo-check", "-C", "/sandbox", "--json", "PROMPT"}
	if !reflect.DeepEqual(noModel, wantNo) {
		t.Fatalf("codexArgs(\"\") = %v\nwant %v (no -m)", noModel, wantNo)
	}
	for _, tok := range noModel {
		if tok == "-m" {
			t.Fatalf("codexArgs(\"\") contains -m, want it omitted: %v", noModel)
		}
	}
	// -a must NEVER be present (codex exec rejects it).
	for _, args := range [][]string{withModel, noModel} {
		for _, tok := range args {
			if tok == "-a" || tok == "--dangerously-bypass-approvals-and-sandbox" {
				t.Fatalf("codexArgs must not pass -a/--dangerously: %v", args)
			}
		}
	}
}

// TestClaudeArgsShape pins the VERIFIED claude -p argv.
func TestClaudeArgsShape(t *testing.T) {
	t.Parallel()

	got := claudeArgs("claude-haiku-4-5", "PROMPT")
	want := []string{
		"-p",
		"--model", "claude-haiku-4-5",
		"--permission-mode", "bypassPermissions",
		"--allowedTools", "Bash",
		"--output-format", "text",
		"PROMPT",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("claudeArgs = %v\nwant %v", got, want)
	}
}
