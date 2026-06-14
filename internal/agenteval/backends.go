package agenteval

// backends.go is the LIVE exec adapter layer (docs/AGENTIC_COVERAGE_PLAN.md §6.1,
// §6.3) — the ONLY impure code in the package. It is the seam where the
// deterministic runner core meets a real agent CLI: it shells out via os/exec,
// captures the agent's output, and funnels it back through the package's PURE
// parsers (ParseCodexJSON for the FS-sandboxed codex stream, ParseSidecar for the
// host-sidecar agents) into a Transcript the scorer grades.
//
// SCOPE BOUNDARY: this file and cmd/run/main.go are the ONLY files in the package
// permitted to import os/exec. Everything else (runner.go, scorer.go,
// transcript.go, report.go, battery.go, …) stays pure. These adapters are NOT
// unit-tested live (a test must never invoke codex/claude); the pure halves they
// call — ParseCodexJSON, ParseSidecar, BuildSandbox — carry the coverage, and the
// roster->backend factory below is unit-tested without exec.
//
// VERIFIED invocations (the operator ran these; we build to THESE, not guesses):
//
//	codex:  codex exec -m <MODEL> -s workspace-write --skip-git-repo-check \
//	          -C <SANDBOX_DIR> --json "<PROMPT>"
//	        Runs commands UNATTENDED in codex's OWN sandbox (no --dangerously, no
//	        -a — `-a` is rejected by `codex exec`). -m is OMITTED when the model is
//	        "" (codex's default = the strong tier). Observed commands + answer text
//	        are recovered from the --json event stream via ParseCodexJSON.
//
//	claude: <claudePath> -p --model <m> --permission-mode bypassPermissions \
//	          --allowedTools Bash --output-format text "<PROMPT>"
//	        cmd.Dir = SANDBOX; observed commands come from the HOST sidecar
//	        (BuildSandbox's observed.jsonl, decoded by ParseSidecar); AgentText is
//	        stdout.

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// codexBackend drives an OpenAI codex sub-agent via `codex exec --json` (§6.1
// ranks 1 and 3). It is FS-sandboxed by codex itself, so the host sidecar is
// invisible and observed commands are recovered from the --json stream
// (ParseCodexJSON). It implements Backend.
type codexBackend struct {
	// name is the roster agent identifier (e.g. "codex-gpt5.4-mini").
	name string
	// rank is the §1.2 floor ordering (lower = weaker).
	rank int
	// model is passed via `-m`; an EMPTY model omits -m so codex uses its default
	// (the strong tier). "gpt-5.4-mini" is the deliberately-weak floor.
	model string
	// bin is the codex executable; defaults to "codex" (resolved on PATH) when "".
	bin string
}

// Name reports the roster agent name.
func (b codexBackend) Name() string { return b.name }

// Rank reports the §1.2 floor rank.
func (b codexBackend) Rank() int { return b.rank }

// Run drives codex against prompt inside sandboxDir and returns the parsed
// Transcript. It builds the VERIFIED `codex exec` argv (omitting -m when model is
// ""), sets cmd.Dir = sandboxDir and cmd.Env = the BuildSandbox-supplied sandbox
// env (codex reads the commands' environment from cmd.Env), runs it, and parses
// stdout with ParseCodexJSON. A non-zero exit from codex itself is NOT an error
// here: codex exits non-zero on some turns even when it produced a usable stream,
// so the stream is parsed regardless and Run only errors if it could not start
// the process or the stream was structurally unparseable.
func (b codexBackend) Run(ctx context.Context, sandboxDir, prompt string, env []string) (Transcript, error) {
	bin := b.bin
	if bin == "" {
		bin = "codex"
	}

	args := codexArgs(b.model, sandboxDir, prompt)

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, bin, args...) // #nosec G204 -- eval tooling; bin is a fixed roster CLI, args are constructed not interpolated
	cmd.Dir = sandboxDir
	cmd.Env = env
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// codex's own non-zero exit is tolerated (it can exit non-zero on a completed
	// turn); a start failure is not.
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return Transcript{}, fmt.Errorf("agenteval codex backend %q: start: %w (stderr: %s)", b.name, err, stderr.String())
		}
	}

	t, err := ParseCodexJSON(stdout.Bytes())
	if err != nil {
		return Transcript{}, fmt.Errorf("agenteval codex backend %q: parse stream: %w", b.name, err)
	}
	return t, nil
}

// codexArgs builds the VERIFIED `codex exec` argv (§6.1). -m is included ONLY for
// a named model; an empty model omits -m so codex uses its default (the strong
// tier). -C points codex at the sandbox dir (cmd.Dir is also set so `zscalerctl`
// on PATH and relative sidecar/wrapper paths resolve). It is PURE so the argv
// shape — especially the -m omission — is unit-testable without exec.
func codexArgs(model, sandboxDir, prompt string) []string {
	args := []string{"exec"}
	if model != "" {
		args = append(args, "-m", model)
	}
	args = append(args,
		"-s", "workspace-write",
		"--skip-git-repo-check",
		"-C", sandboxDir,
		"--json",
		prompt,
	)
	return args
}

// claudeArgs builds the VERIFIED `claude -p` argv (§6.1): -p (print/non-
// interactive), --model, --permission-mode bypassPermissions, --allowedTools
// Bash, --output-format text, then the prompt. It is PURE so the argv shape is
// unit-testable without exec.
func claudeArgs(model, prompt string) []string {
	return []string{
		"-p",
		"--model", model,
		"--permission-mode", "bypassPermissions",
		"--allowedTools", "Bash",
		"--output-format", "text",
		prompt,
	}
}

// claudeBackend drives Anthropic's claude CLI via `claude -p` (§6.1 ranks 2/4).
// Claude executes tool calls (Bash) that go through the PATH-interposed fixture
// binary to the HOST sidecar, so observed commands are recovered from
// sandboxDir/observed.jsonl (ParseSidecar) and AgentText is stdout. It implements
// Backend.
//
// The claude path is GATED (it may be classifier-blocked in-session). The adapter
// is coded so the operator can run it on-demand; nothing here invokes it
// automatically and no test execs it.
type claudeBackend struct {
	// name is the roster agent identifier (e.g. "haiku", "sonnet").
	name string
	// rank is the §1.2 floor ordering (lower = weaker).
	rank int
	// model is passed via `--model`.
	model string
	// bin is the claude executable PATH. It MUST be the real binary, NOT the
	// nix-darwin fish wrapper named `claude` (which injects --remote-control and
	// disables features — see repo memory). The factory resolves it from the
	// roster/env; "" falls back to "claude" on PATH.
	bin string
}

// Name reports the roster agent name.
func (b claudeBackend) Name() string { return b.name }

// Rank reports the §1.2 floor rank.
func (b claudeBackend) Rank() int { return b.rank }

// Run drives claude against prompt inside sandboxDir and returns the Transcript.
// It builds the VERIFIED `claude -p … --allowedTools Bash --output-format text`
// argv, sets cmd.Dir = sandboxDir and cmd.Env = the sandbox env, runs it, then
// reads the HOST sidecar at sandboxDir/observed.jsonl (ParseSidecar) for the
// observed commands and uses stdout as AgentText. A claude non-zero exit is
// tolerated for the same reason as codex (the answer envelope may still be on
// stdout); Run errors only on a process start failure or a corrupt sidecar.
func (b claudeBackend) Run(ctx context.Context, sandboxDir, prompt string, env []string) (Transcript, error) {
	bin := b.bin
	if bin == "" {
		bin = "claude"
	}

	args := claudeArgs(b.model, prompt)

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, bin, args...) // #nosec G204 -- eval tooling; bin is a fixed roster CLI, args are constructed not interpolated
	cmd.Dir = sandboxDir
	cmd.Env = env
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return Transcript{}, fmt.Errorf("agenteval claude backend %q: start: %w (stderr: %s)", b.name, err, stderr.String())
		}
	}

	// Observed commands come from the HOST sidecar, not stdout. A missing sidecar
	// means the agent ran no wrapped commands (an empty stream), which the scorer
	// treats as the no_commands condition — distinct from a corrupt sidecar, which
	// is surfaced as an error so a dropped command can't become a false PASS.
	sidecarPath := filepath.Join(sandboxDir, sidecarLogName)
	commands, err := readSidecar(sidecarPath)
	if err != nil {
		return Transcript{}, fmt.Errorf("agenteval claude backend %q: read sidecar: %w", b.name, err)
	}
	return AssembleTranscript(stdout.String(), commands), nil
}

// readSidecar reads and parses the host observed-command sidecar at path. A
// non-existent sidecar yields an empty (nil) command slice with no error: the
// agent simply ran no wrapped commands. Any other read error, or a malformed
// line, is returned so a corrupt stream is surfaced rather than silently dropped.
func readSidecar(path string) ([]ObservedCommand, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- runner-confined sidecar path inside the eval sandbox
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return ParseSidecar(data)
}

// LiveBackend is the live exec contract the orchestrator drives: it extends the
// pure Backend's Name/Rank with a Run that ALSO takes the BuildSandbox env (the
// exec adapters need cmd.Env; the pure FakeBackend's Backend.Run does not). The
// orchestrator works against this richer interface; the factory returns it.
type LiveBackend interface {
	// Name is the roster agent identifier.
	Name() string
	// Rank is the §1.2 floor ordering (lower = weaker).
	Rank() int
	// Run drives the agent against prompt inside sandboxDir with the supplied
	// sandbox env and returns the observed Transcript.
	Run(ctx context.Context, sandboxDir, prompt string, env []string) (Transcript, error)
}

// Compile-time proof the adapters satisfy LiveBackend.
var (
	_ LiveBackend = codexBackend{}
	_ LiveBackend = claudeBackend{}
)

// BackendFactoryConfig carries the host-resolved executable paths the factory
// threads into the adapters (the roster.json entries declare the invocation shape
// but not absolute paths, which are host-specific and must never be committed).
// Empty paths fall back to the bare command name on PATH.
type BackendFactoryConfig struct {
	// CodexBin is the codex executable; "" -> "codex" on PATH.
	CodexBin string
	// ClaudeBin is the REAL claude executable (NOT the nix-darwin fish wrapper);
	// "" -> "claude" on PATH.
	ClaudeBin string
}

// BackendForRosterEntry maps one roster entry to its live exec adapter (§6.1),
// classifying by the entry's capture/invocation/model fields rather than by a
// hardcoded name list, so a new roster row with the same shape needs no code
// change. It is PURE over (entry, cfg) — it constructs an adapter value and never
// execs — so the roster->backend mapping is unit-testable without invoking any
// agent.
//
// Classification:
//
//   - capture == "codex-transcript"  -> codexBackend (the FS-sandboxed codex
//     stream is parsed via ParseCodexJSON). entry.Model selects the model; ""
//     means codex default.
//   - capture == "host-sidecar"      -> claudeBackend (tool calls go through the
//     PATH-interposed fixture binary to the host sidecar, parsed via ParseSidecar).
//
// An unrecognized capture is an error (a new capture mechanism is a deliberate
// code change, never a silent default to the wrong adapter).
func BackendForRosterEntry(entry RosterEntry, cfg BackendFactoryConfig) (LiveBackend, error) {
	switch entry.Capture {
	case captureCodexTranscript:
		return codexBackend{
			name:  entry.Agent,
			rank:  entry.Rank,
			model: entry.Model,
			bin:   cfg.CodexBin,
		}, nil
	case captureHostSidecar:
		return claudeBackend{
			name:  entry.Agent,
			rank:  entry.Rank,
			model: entry.Model,
			bin:   cfg.ClaudeBin,
		}, nil
	default:
		return nil, fmt.Errorf("agenteval: roster entry %q has unsupported capture %q (want %q or %q)",
			entry.Agent, entry.Capture, captureCodexTranscript, captureHostSidecar)
	}
}
