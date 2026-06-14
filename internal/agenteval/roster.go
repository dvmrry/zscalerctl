package agenteval

// roster.go is the PURE roster model + loader (docs/AGENTIC_COVERAGE_PLAN.md
// §6.1): the committed fixed roster (rank + per-agent capability/capture
// declaration) decoded into typed RosterEntry values. It reads a file (the same
// filesystem-only purity class as BuildSandbox) but never execs, dials the
// network, reads the clock, or calls an LLM. The live exec adapters that turn a
// RosterEntry into a running agent live in backends.go (the os/exec layer);
// mapping an entry to its adapter (BackendForRosterEntry) is pure and tested
// without exec.

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// Capture mechanism identifiers (§2.3, §6.1). They select which PURE parser the
// live adapter funnels the agent's output through:
//
//   - captureCodexTranscript: codex is FS-sandboxed, its host sidecar is
//     invisible, so observed commands are recovered from the `codex exec --json`
//     event stream (ParseCodexJSON).
//   - captureHostSidecar: the agent's tool calls go through the PATH-interposed
//     fixture binary to the host observed.jsonl sidecar (ParseSidecar); stdout is
//     the answer text.
const (
	captureCodexTranscript = "codex-transcript"
	captureHostSidecar     = "host-sidecar"
)

// RosterEntry is one committed roster row (§6.1). The exec/path details are NOT
// committed (host-specific); the factory threads absolute executable paths in via
// BackendFactoryConfig. The fields the live half branches on are Rank, Agent,
// Capture, Model, and Deferred; the rest (Invocation/Mode/ReadsLocalFiles/Notes)
// are human-facing documentation carried verbatim from the JSON.
type RosterEntry struct {
	// Rank is the §1.2 floor ordering (lower = weaker).
	Rank int `json:"rank"`
	// Agent is the roster agent identifier (e.g. "haiku"), the AgentRun.Agent name.
	Agent string `json:"agent"`
	// Capture selects the observed-command parser: captureCodexTranscript or
	// captureHostSidecar.
	Capture string `json:"capture"`
	// Model is the model id the adapter passes to the CLI. For codex an empty
	// Model omits -m (codex default = the strong tier); for claude it is the
	// --model value.
	Model string `json:"model"`
	// Deferred marks a roster row NOT in the initial run set (§6.1 decision 1):
	// sonnet (ceiling), devin, gemini. The orchestrator's default backend set is
	// every non-deferred entry.
	Deferred bool `json:"deferred"`
	// Invocation is the human-facing invocation shape (documentation).
	Invocation string `json:"invocation"`
	// Mode is "single-shot" or "session" (documentation).
	Mode string `json:"mode"`
	// ReadsLocalFiles is the declared local-file capability (documentation; a
	// JSON bool OR string in the committed roster, so it is decoded loosely).
	ReadsLocalFiles json.RawMessage `json:"readsLocalFiles"`
	// Notes is the per-agent caveat prose (documentation).
	Notes string `json:"notes"`
}

// LoadRoster reads and decodes the committed roster JSON at path into RosterEntry
// values, sorted by Rank ascending (weakest first) so the floor ordering is
// stable regardless of file order. It validates that every entry has a non-empty
// Agent and a recognized Capture, and that Agent names are unique — a malformed
// roster is a configuration error surfaced here, never a silent skip. It is PURE
// over the file contents (os.ReadFile + json.Unmarshal; no exec/net/clock).
func LoadRoster(path string) ([]RosterEntry, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- eval tooling; roster path is operator/flag supplied
	if err != nil {
		return nil, fmt.Errorf("agenteval: read roster %q: %w", path, err)
	}
	return ParseRoster(data)
}

// ParseRoster decodes roster JSON bytes into validated, rank-sorted RosterEntry
// values. It is split out from LoadRoster so the decode+validation can be
// unit-tested directly from a byte literal without touching the filesystem.
func ParseRoster(data []byte) ([]RosterEntry, error) {
	var entries []RosterEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("agenteval: decode roster: %w", err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("agenteval: roster is empty")
	}

	seen := make(map[string]bool, len(entries))
	for _, e := range entries {
		if e.Agent == "" {
			return nil, fmt.Errorf("agenteval: roster entry at rank %d has empty agent", e.Rank)
		}
		if seen[e.Agent] {
			return nil, fmt.Errorf("agenteval: roster has duplicate agent %q", e.Agent)
		}
		seen[e.Agent] = true
		if e.Capture != captureCodexTranscript && e.Capture != captureHostSidecar {
			return nil, fmt.Errorf("agenteval: roster entry %q has unsupported capture %q (want %q or %q)",
				e.Agent, e.Capture, captureCodexTranscript, captureHostSidecar)
		}
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Rank != entries[j].Rank {
			return entries[i].Rank < entries[j].Rank
		}
		return entries[i].Agent < entries[j].Agent
	})
	return entries, nil
}

// SelectBackends resolves the roster into the live backends the orchestrator will
// run, given the set of agent names to ENABLE. It is PURE over (entries, enable,
// cfg): it maps each selected entry to its adapter via BackendForRosterEntry and
// returns them in rank order, so the floor ordering is preserved. It never execs.
//
// enable selection:
//
//   - enable == nil: the DEFAULT — every NON-deferred entry (§6.1 decision 1:
//     sonnet/devin/gemini are deferred and skipped unless named explicitly).
//   - enable non-nil: exactly the named agents, deferred or not (so the operator
//     can opt a deferred agent in by name). An unknown name is an error.
func SelectBackends(entries []RosterEntry, enable []string, cfg BackendFactoryConfig) ([]LiveBackend, error) {
	byName := make(map[string]RosterEntry, len(entries))
	for _, e := range entries {
		byName[e.Agent] = e
	}

	var chosen []RosterEntry
	if enable == nil {
		for _, e := range entries {
			if !e.Deferred {
				chosen = append(chosen, e)
			}
		}
		if len(chosen) == 0 {
			return nil, fmt.Errorf("agenteval: no non-deferred roster entries to run")
		}
	} else {
		for _, name := range enable {
			e, ok := byName[name]
			if !ok {
				return nil, fmt.Errorf("agenteval: --backends names unknown agent %q (not in roster)", name)
			}
			chosen = append(chosen, e)
		}
	}

	out := make([]LiveBackend, 0, len(chosen))
	for _, e := range chosen {
		b, err := BackendForRosterEntry(e, cfg)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}
