package agenteval_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/dvmrry/zscalerctl/internal/agenteval"
)

// backends_test.go covers the PURE roster->backend mapping and prompt-composition
// helpers of the live layer (roster.go, backends.go's factory, prompt.go). It
// NEVER invokes codex/claude — the os/exec adapters' Run methods are not exercised
// here (they are validated on-demand by a real run); only the deterministic
// mapping/selection/composition is tested.

// rosterJSON is a minimal two-capture roster used by the parse/select tests. It
// mirrors the committed roster's shape (rank/agent/capture/model/deferred) with a
// codex floor, a claude host-sidecar agent, a default-model codex, and a deferred
// ceiling.
const rosterJSON = `[
  {"rank":1,"agent":"codex-mini","capture":"codex-transcript","model":"gpt-5.4-mini","deferred":false},
  {"rank":2,"agent":"haiku","capture":"host-sidecar","model":"claude-haiku-4-5","deferred":false},
  {"rank":3,"agent":"codex-default","capture":"codex-transcript","model":"","deferred":false},
  {"rank":4,"agent":"sonnet","capture":"host-sidecar","model":"claude-sonnet-4-5","deferred":true}
]`

// TestParseRosterSortsAndValidates asserts ParseRoster decodes, rank-sorts, and
// validates the roster.
func TestParseRosterSortsAndValidates(t *testing.T) {
	t.Parallel()

	// Deliberately out of rank order to prove the sort.
	unsorted := `[
      {"rank":3,"agent":"codex-default","capture":"codex-transcript","model":"","deferred":false},
      {"rank":1,"agent":"codex-mini","capture":"codex-transcript","model":"gpt-5.4-mini","deferred":false},
      {"rank":2,"agent":"haiku","capture":"host-sidecar","model":"claude-haiku-4-5","deferred":false}
    ]`
	entries, err := agenteval.ParseRoster([]byte(unsorted))
	if err != nil {
		t.Fatalf("ParseRoster: %v", err)
	}
	gotOrder := []string{entries[0].Agent, entries[1].Agent, entries[2].Agent}
	wantOrder := []string{"codex-mini", "haiku", "codex-default"}
	if !reflect.DeepEqual(gotOrder, wantOrder) {
		t.Fatalf("rank order = %v, want %v", gotOrder, wantOrder)
	}
}

// TestParseRosterRejectsMalformed asserts the validation gate: empty agent,
// duplicate agent, unknown capture, and empty roster are all errors.
func TestParseRosterRejectsMalformed(t *testing.T) {
	t.Parallel()

	for name, in := range map[string]string{
		"empty roster":    `[]`,
		"empty agent":     `[{"rank":1,"agent":"","capture":"codex-transcript"}]`,
		"unknown capture": `[{"rank":1,"agent":"x","capture":"telepathy"}]`,
		"duplicate agent": `[{"rank":1,"agent":"x","capture":"host-sidecar"},{"rank":2,"agent":"x","capture":"host-sidecar"}]`,
		"not json":        `{`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := agenteval.ParseRoster([]byte(in)); err == nil {
				t.Fatalf("ParseRoster(%s) = nil error, want a validation error", name)
			}
		})
	}
}

// TestSelectBackendsDefaultSkipsDeferred asserts the default (nil enable) selects
// every NON-deferred entry, in rank order, and skips the deferred ceiling.
func TestSelectBackendsDefaultSkipsDeferred(t *testing.T) {
	t.Parallel()

	entries, err := agenteval.ParseRoster([]byte(rosterJSON))
	if err != nil {
		t.Fatalf("ParseRoster: %v", err)
	}
	backends, err := agenteval.SelectBackends(entries, nil, agenteval.BackendFactoryConfig{})
	if err != nil {
		t.Fatalf("SelectBackends: %v", err)
	}
	got := backendNames(backends)
	want := []string{"codex-mini", "haiku", "codex-default"} // sonnet deferred -> skipped
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("default backends = %v, want %v (deferred sonnet skipped, rank order)", got, want)
	}
	// Ranks come from the roster.
	if backends[0].Rank() != 1 || backends[1].Rank() != 2 || backends[2].Rank() != 3 {
		t.Fatalf("ranks = %d/%d/%d, want 1/2/3", backends[0].Rank(), backends[1].Rank(), backends[2].Rank())
	}
}

// TestSelectBackendsExplicitEnable asserts a non-nil enable list selects exactly
// the named agents (a DEFERRED one CAN be opted in by name), and an unknown name
// is an error.
func TestSelectBackendsExplicitEnable(t *testing.T) {
	t.Parallel()

	entries, err := agenteval.ParseRoster([]byte(rosterJSON))
	if err != nil {
		t.Fatalf("ParseRoster: %v", err)
	}

	// Deferred sonnet opted in explicitly + haiku.
	backends, err := agenteval.SelectBackends(entries, []string{"sonnet", "haiku"}, agenteval.BackendFactoryConfig{})
	if err != nil {
		t.Fatalf("SelectBackends: %v", err)
	}
	if got := backendNames(backends); !reflect.DeepEqual(got, []string{"sonnet", "haiku"}) {
		t.Fatalf("explicit backends = %v, want [sonnet haiku]", got)
	}

	// Unknown name is an error.
	if _, err := agenteval.SelectBackends(entries, []string{"nope"}, agenteval.BackendFactoryConfig{}); err == nil {
		t.Fatalf("SelectBackends with unknown agent = nil error, want an error")
	}
}

// TestSelectBackendsRejectsHostSidecarWithoutAdapter asserts deferred rows can
// only be opted in when their capture has a real local adapter. Devin/Gemini rows
// are documented future host-sidecars, but they must not silently run Claude.
func TestSelectBackendsRejectsHostSidecarWithoutAdapter(t *testing.T) {
	t.Parallel()

	entries, err := agenteval.ParseRoster([]byte(`[
      {"rank":1,"agent":"codex-mini","capture":"codex-transcript","model":"gpt-5.4-mini","deferred":false},
      {"rank":5,"agent":"devin","capture":"host-sidecar","model":"","deferred":true}
    ]`))
	if err != nil {
		t.Fatalf("ParseRoster: %v", err)
	}

	_, err = agenteval.SelectBackends(entries, []string{"devin"}, agenteval.BackendFactoryConfig{})
	if err == nil {
		t.Fatalf("SelectBackends(devin) = nil error, want missing-adapter error")
	}
	if !strings.Contains(err.Error(), "dedicated adapter") {
		t.Fatalf("SelectBackends(devin) error = %q, want dedicated-adapter guidance", err)
	}
}

// TestBackendForRosterEntryClassifies asserts the factory maps capture to the
// right adapter (by observable Name/Rank, and an unknown capture errors). Field
// mapping (model/bin) is asserted in the internal test where the concrete struct
// is visible.
func TestBackendForRosterEntryClassifies(t *testing.T) {
	t.Parallel()

	codex, err := agenteval.BackendForRosterEntry(
		agenteval.RosterEntry{Rank: 1, Agent: "codex-mini", Capture: "codex-transcript", Model: "gpt-5.4-mini"},
		agenteval.BackendFactoryConfig{},
	)
	if err != nil {
		t.Fatalf("codex entry: %v", err)
	}
	if codex.Name() != "codex-mini" || codex.Rank() != 1 {
		t.Fatalf("codex Name/Rank = %q/%d, want codex-mini/1", codex.Name(), codex.Rank())
	}

	claude, err := agenteval.BackendForRosterEntry(
		agenteval.RosterEntry{Rank: 2, Agent: "haiku", Capture: "host-sidecar", Model: "claude-haiku-4-5"},
		agenteval.BackendFactoryConfig{},
	)
	if err != nil {
		t.Fatalf("claude entry: %v", err)
	}
	if claude.Name() != "haiku" || claude.Rank() != 2 {
		t.Fatalf("claude Name/Rank = %q/%d, want haiku/2", claude.Name(), claude.Rank())
	}

	if _, err := agenteval.BackendForRosterEntry(
		agenteval.RosterEntry{Rank: 5, Agent: "gemini", Capture: "host-sidecar", Model: ""},
		agenteval.BackendFactoryConfig{},
	); err == nil {
		t.Fatalf("host-sidecar without adapter = nil error, want an error")
	}

	if _, err := agenteval.BackendForRosterEntry(
		agenteval.RosterEntry{Agent: "x", Capture: "telepathy"},
		agenteval.BackendFactoryConfig{},
	); err == nil {
		t.Fatalf("unsupported capture = nil error, want an error")
	}
}

// TestComposePrompt asserts the composed prompt contains the question text, the
// hard-rule confinement line, and the answer-envelope delimiters in order (the
// envelope LAST), and that the instructed delimiters match the parser's so an
// agent that follows the instructions produces a parseable block.
func TestComposePrompt(t *testing.T) {
	t.Parallel()

	q := agenteval.Question{ID: "Q-X", Prompt: "  How many locations are configured?  "}
	got := agenteval.ComposePrompt(q)

	// Question text is present (trimmed).
	if !strings.Contains(got, "How many locations are configured?") {
		t.Fatalf("composed prompt missing question text:\n%s", got)
	}
	// Hard-rule confinement line is present.
	if !strings.Contains(got, "only run ./zscalerctl") {
		t.Fatalf("composed prompt missing hard-rule line:\n%s", got)
	}
	// Both envelope delimiters present.
	if !strings.Contains(got, agenteval.AnswerOpen) || !strings.Contains(got, agenteval.AnswerClose) {
		t.Fatalf("composed prompt missing answer envelope delimiters:\n%s", got)
	}
	// Envelope instructions come LAST (after the hard rule).
	if strings.Index(got, agenteval.AnswerOpen) < strings.Index(got, "only run ./zscalerctl") {
		t.Fatalf("answer envelope should appear after the hard-rule line:\n%s", got)
	}

	// A block authored to these instructions must parse via ParseAnswer (the
	// instructed delimiters match the scanner's).
	authored := got + "\n" + agenteval.AnswerOpen + "\n" + `{"answer": 12, "evidence": ["zscalerctl zia locations list"]}` + "\n" + agenteval.AnswerClose + "\n"
	env, ok := agenteval.ParseAnswer(authored)
	if !ok {
		t.Fatalf("a block authored to ComposePrompt's instructions did not parse")
	}
	if strings.TrimSpace(string(env.Answer)) != "12" {
		t.Fatalf("parsed answer = %q, want 12", string(env.Answer))
	}
}

// backendNames returns the Name() of each backend, in order.
func backendNames(backends []agenteval.LiveBackend) []string {
	out := make([]string, len(backends))
	for i, b := range backends {
		out[i] = b.Name()
	}
	return out
}
