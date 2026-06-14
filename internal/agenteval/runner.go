package agenteval

// runner.go is the DETERMINISTIC runner core (docs/AGENTIC_COVERAGE_PLAN.md §6,
// esp. §6.3) — the pure, testable scaffolding the live half plugs into next.
//
// CRITICAL SCOPE BOUNDARY: nothing in this file execs a live agent, dials a
// network, calls an LLM, reads the clock, or calls rand. It is the same gated
// half as the scorer (§1.3): every function here is a pure/deterministic
// transform of its arguments, with the single exception of BuildSandbox, which
// writes a hermetic working directory and reads the parent process environment
// (filesystem + os.Environ only — no exec, no net, no clock). The LIVE exec
// adapters that implement Backend.Run via os/exec land in the NEXT slice; here a
// Backend is only an interface, and the only implementation is FakeBackend,
// which returns a canned Transcript so the runner core can be unit-tested
// without an agent.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Backend is one roster agent the live half drives (§6.1). Name identifies it
// (matching roster.json's "agent"); Rank is the §1.2 floor ordering (lower =
// weaker). Run executes the agent against the prompt inside sandboxDir and
// returns the resulting Transcript.
//
// In THIS slice Run is only a contract: the live exec adapters that shell out to
// `command claude`/`codex exec`/`devin run` implement it next. The single
// implementation here is FakeBackend (a canned Transcript), which is what lets
// the deterministic runner core be tested without invoking any agent, network,
// LLM, or clock.
type Backend interface {
	// Name is the roster agent identifier (e.g. "haiku"), matching roster.json.
	Name() string
	// Rank is the §1.2 floor ordering; lower is a weaker agent.
	Rank() int
	// Run drives the agent against prompt inside sandboxDir and returns the
	// observed Transcript. Implemented by the live adapters in the next slice.
	Run(ctx context.Context, sandboxDir, prompt string) (Transcript, error)
}

// FakeBackend is the deterministic Backend used by tests (and by any pure
// runner-core exercise): Run ignores ctx/sandboxDir/prompt and returns a canned
// Transcript, so the runner core can be exercised without an agent, network,
// LLM, or clock. The live adapters replace this in the next slice; FakeBackend
// stays as the test double.
type FakeBackend struct {
	// AgentName is returned by Name().
	AgentName string
	// AgentRank is returned by Rank().
	AgentRank int
	// Canned is the Transcript Run returns verbatim.
	Canned Transcript
	// RunErr, when non-nil, is returned by Run instead of Canned (to exercise the
	// error path deterministically).
	RunErr error
}

// Name reports the canned agent name.
func (f FakeBackend) Name() string { return f.AgentName }

// Rank reports the canned agent rank.
func (f FakeBackend) Rank() int { return f.AgentRank }

// Run returns the canned Transcript (or RunErr). It never execs, dials the
// network, reads the clock, or calls an LLM — it is a pure function of f.
func (f FakeBackend) Run(_ context.Context, _ string, _ string) (Transcript, error) {
	if f.RunErr != nil {
		return Transcript{}, f.RunErr
	}
	return f.Canned, nil
}

// Sandbox env var names (§6.3). The fixture binary reads ZSCALERCTL_FIXTURE_DIR
// (its hard-fail gate) and ZSCALERCTL_FIXTURE_LOG (the confined relative sidecar
// path) from its own os.Getenv, so the runner sets exactly these.
const (
	// envFixtureDir selects + gates the fixture binary (§2.3). Unset/empty makes
	// the fixture binary hard-fail exit 1, so BuildSandbox always sets it.
	envFixtureDir = "ZSCALERCTL_FIXTURE_DIR"
	// envFixtureLog is the CONFINED relative sidecar filename (§2.3). It MUST be a
	// bare relative filename (no path separators, no "..") so the fixture binary,
	// which filepath.Cleans it and rejects absolute/".." values, resolves it
	// against the sandbox cwd.
	envFixtureLog = "ZSCALERCTL_FIXTURE_LOG"

	// sidecarLogName is the default confined sidecar filename BuildSandbox injects
	// for ZSCALERCTL_FIXTURE_LOG. A bare filename resolves against the sandbox cwd
	// (the agent WorkDir) per the §2.3 confined-path contract.
	sidecarLogName = "observed.jsonl"

	// fixtureBinName is the name the fixture binary is copied in as inside the
	// sandbox: the agent runs `zscalerctl` (on PATH), never the runner's tmp
	// BinPath (§6.3 "Prompt uses zscalerctl (on PATH)").
	fixtureBinName = "zscalerctl"
)

// Synthetic, value-free fixture credentials (§2.3 / §6.3). These are obviously
// synthetic placeholders that pass zscaler.ValidateReaderConfig so the fixture
// reader is injected, but never reach an endpoint (the fixture reader serves the
// data). They match the literals the shim test and posture allow-list use, so
// the value-free posture gate recognizes them as sentinels rather than secrets.
const (
	syntheticClientID     = "synthetic-client-id"
	syntheticClientSecret = "synthetic-client-secret" // #nosec G101 -- synthetic placeholder, never a real secret
	syntheticVanityDomain = "example"
)

// strippedEnvPrefixes are the parent-environment prefixes BuildSandbox strips so
// no real credential or provider token ever enters the sandbox (§6.3 env
// hygiene / §5.5). Every parent var whose key starts with one of these is
// dropped before the minimal sandbox env is assembled:
//
//   - ZSCALERCTL_ : the operator's REAL tenant credentials. Stripped so a real
//     secret is never present to leak; synthetic value-free creds are injected
//     instead (below).
//   - ANTHROPIC_ / OPENAI_ / DEVIN_ : provider API tokens for the agents
//     themselves. Stripped so a transcript/sandbox can never carry a provider
//     token (the runtime half of the §5.5 two-layer scrub).
var strippedEnvPrefixes = []string{
	"ZSCALERCTL_",
	"ANTHROPIC_",
	"OPENAI_",
	"DEVIN_",
}

// BuildSandbox creates a fresh hermetic WorkDir for one (agent, question) run
// (§6.3) and returns the MINIMAL sandbox environment the agent process must run
// with. It is the only function in this file that touches the filesystem or the
// parent environment; it still never execs, dials the network, reads the clock,
// or calls an LLM.
//
// It performs three things:
//
//  1. Copies the fixture binary at fixtureBinPath into dir as "zscalerctl", so
//     the agent invokes it via PATH as the plain command name (§6.3 — the prompt
//     uses `zscalerctl`, never the runner's tmp BinPath). The copy is marked
//     executable.
//  2. Writes each docs entry into dir at its (relative) key, creating parent
//     directories as needed (e.g. "AGENTS.md", "skill/SKILL.md"). A key with an
//     absolute path or a ".." traversal is rejected, so a caller can't write
//     outside the sandbox.
//  3. Returns the minimal sandbox env (see below).
//
// The returned env is assembled fresh, NOT inherited wholesale from the parent:
//
//   - every ZSCALERCTL_*/ANTHROPIC_*/OPENAI_*/DEVIN_* value in the parent is
//     STRIPPED (strippedEnvPrefixes) so no real credential or provider token
//     leaks in;
//   - obviously-synthetic, value-free ZSCALERCTL_* fixture credentials are
//     injected (client id/secret + vanity) so the fixture reader's real
//     credential-validation path passes;
//   - ZSCALERCTL_FIXTURE_DIR points at the fixture corpus directory and
//     ZSCALERCTL_FIXTURE_LOG is the CONFINED relative sidecar filename, resolved
//     against the sandbox cwd (dir);
//   - PATH is minimal: it leads with the sandbox dir (so `zscalerctl` resolves to
//     the copied-in fixture binary, and any future logging wrapper in dir wins)
//     followed by a small set of standard system bin dirs.
//
// Nothing else from the parent leaks: a parent var that is not a stripped prefix
// and not one of the explicitly-set sandbox vars simply does not appear.
//
// fixtureDir is where the corpus lives; the agent process's cwd is the caller's
// responsibility (the runner sets it to dir) — the returned ZSCALERCTL_FIXTURE_LOG
// is relative and only resolves correctly when cwd == dir.
func BuildSandbox(dir, fixtureBinPath string, docs map[string]string) (env []string, err error) {
	// (a) Copy the fixture binary in as the plain `zscalerctl` command.
	if err := copyExecutable(fixtureBinPath, filepath.Join(dir, fixtureBinName)); err != nil {
		return nil, err
	}

	// (b) Write each doc into the sandbox, confined under dir.
	for rel, content := range docs {
		if !confinedRel(rel) {
			return nil, &SandboxError{Op: "write doc", Path: rel, Reason: "doc path must be a confined relative path (no absolute path, no .. traversal)"}
		}
		dest := filepath.Join(dir, filepath.Clean(rel))
		if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
			return nil, err
		}
		if err := os.WriteFile(dest, []byte(content), 0o600); err != nil {
			return nil, err
		}
	}

	// (c) Assemble the minimal sandbox env. The corpus directory is the sandbox
	// itself in this slice (the runner stages everything in dir); a parent
	// ZSCALERCTL_FIXTURE_DIR is stripped and re-set to this value, never inherited.
	return buildSandboxEnv(dir, dir), nil
}

// buildSandboxEnv assembles the minimal sandbox environment FROM SCRATCH (§6.3 /
// §5.5): nothing from the parent process environment is inherited, so no real
// credential or provider token can leak in. The returned slice contains only the
// synthetic value-free fixture creds, ZSCALERCTL_FIXTURE_DIR, the CONFINED
// relative log filename (ZSCALERCTL_FIXTURE_LOG), and a minimal PATH leading with
// the sandbox dir. It is a pure function of (dir, corpusDir string). (The
// prefix-strip primitive SanitizeParentEnv is provided separately for the live
// adapter, which may need to hand a scrubbed PATH/HOME to the agent CLI.)
func buildSandboxEnv(dir, corpusDir string) []string {
	// Start empty; we add back ONLY what the sandbox needs. Nothing from the
	// parent is inherited wholesale.
	out := make([]string, 0, 8)

	// Synthetic, value-free fixture credentials so ValidateReaderConfig passes and
	// the fixture reader (never a live client) is injected.
	out = append(out,
		"ZSCALERCTL_CLIENT_ID="+syntheticClientID,
		"ZSCALERCTL_CLIENT_SECRET="+syntheticClientSecret,
		"ZSCALERCTL_VANITY_DOMAIN="+syntheticVanityDomain,
	)

	// Fixture selection + the confined relative sidecar log (resolved against the
	// sandbox cwd, which the runner sets to dir).
	out = append(out,
		envFixtureDir+"="+corpusDir,
		envFixtureLog+"="+sidecarLogName,
	)

	// Minimal PATH: the sandbox dir first (so `zscalerctl` resolves to the
	// copied-in fixture binary and any wrapper in dir wins), then a small set of
	// standard system bin dirs. The parent PATH is NOT inherited.
	out = append(out, "PATH="+strings.Join([]string{dir, "/usr/bin", "/bin"}, string(os.PathListSeparator)))

	sort.Strings(out)
	return out
}

// SanitizeParentEnv returns the parent process environment with every
// stripped-prefix variable removed (§6.3 / §5.5). It is exported and pure over
// its input so the env-stripping invariant can be unit-tested directly (the
// RED-test plants a provider token in the input and asserts it is gone from the
// output). buildSandboxEnv does not inherit the parent at all, so this is the
// belt-and-braces helper the live half uses when it must thread a curated subset
// of the parent (e.g. an agent CLI's own non-secret config) into a child.
func SanitizeParentEnv(parent []string) []string {
	out := make([]string, 0, len(parent))
	for _, kv := range parent {
		key := kv
		if i := strings.IndexByte(kv, '='); i >= 0 {
			key = kv[:i]
		}
		if hasStrippedPrefix(key) {
			continue
		}
		out = append(out, kv)
	}
	return out
}

// hasStrippedPrefix reports whether an env var key starts with one of the
// stripped credential/provider prefixes (§6.3).
func hasStrippedPrefix(key string) bool {
	for _, p := range strippedEnvPrefixes {
		if strings.HasPrefix(key, p) {
			return true
		}
	}
	return false
}

// confinedRel reports whether rel is a confined relative path: not absolute, and
// not escaping its root via "..". It mirrors the fixture binary's confined-path
// contract (§2.3) so a doc key can never be written outside the sandbox.
func confinedRel(rel string) bool {
	if rel == "" {
		return false
	}
	clean := filepath.Clean(rel)
	if filepath.IsAbs(clean) {
		return false
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

// copyExecutable copies the file at src to dst and marks dst executable. It is a
// plain filesystem copy — no exec, no network.
func copyExecutable(src, dst string) error {
	data, err := os.ReadFile(src) // #nosec G304 -- runner-supplied fixture binary path, eval tooling
	if err != nil {
		return &SandboxError{Op: "copy fixture binary", Path: src, Reason: err.Error()}
	}
	if err := os.WriteFile(dst, data, 0o700); err != nil { // #nosec G302 -- the agent must be able to exec the copied-in fixture binary
		return &SandboxError{Op: "copy fixture binary", Path: dst, Reason: err.Error()}
	}
	return nil
}

// SandboxError is a typed error from BuildSandbox, carrying the operation and
// path so a caller (and a test) can distinguish a confinement rejection from a
// filesystem failure.
type SandboxError struct {
	Op     string
	Path   string
	Reason string
}

// Error renders the sandbox error.
func (e *SandboxError) Error() string {
	return "agenteval sandbox: " + e.Op + " " + e.Path + ": " + e.Reason
}

// --- per-backend observed-command capture (§2.3) ----------------------------

// ParseSidecar parses the fixture binary's observed.jsonl sidecar — one JSON
// object per line, {"argv":[...],"exit":N} — into []ObservedCommand (§2.3). It
// is the capture path for HOST-EXECUTING agents whose tool calls go through the
// PATH-interposed fixture binary / wrappers to the host sidecar file.
//
// It is lenient about blank lines (skipped) but strict about content: a
// non-blank line that is not a valid sidecar object is an error, so a corrupt
// stream is surfaced rather than silently dropping observed commands (a dropped
// command could turn a method violation into a false PASS). Extra sidecar fields
// (tool, stdout_sha256, …) are ignored — only argv + exit reach the scorer.
func ParseSidecar(b []byte) ([]ObservedCommand, error) {
	var out []ObservedCommand
	for i, raw := range strings.Split(string(b), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		cmd, err := decodeSidecarLine(line)
		if err != nil {
			return nil, &SidecarError{Line: i + 1, Reason: err.Error()}
		}
		out = append(out, cmd)
	}
	return out, nil
}

// decodeSidecarLine decodes one sidecar line into an ObservedCommand. It uses a
// strict decoder so a line missing argv (or with the wrong shape) is rejected
// rather than yielding an empty-argv command. Extra fields are tolerated (only
// argv + exit are needed by the scorer); a present-but-non-array argv is an
// error.
func decodeSidecarLine(line string) (ObservedCommand, error) {
	// Distinguish "argv absent" from "argv present and empty": require the key.
	var probe map[string]json.RawMessage
	if err := json.Unmarshal([]byte(line), &probe); err != nil {
		return ObservedCommand{}, err
	}
	if _, ok := probe["argv"]; !ok {
		return ObservedCommand{}, errNoArgv
	}
	var cmd ObservedCommand
	if err := json.Unmarshal([]byte(line), &cmd); err != nil {
		return ObservedCommand{}, err
	}
	return cmd, nil
}

// errNoArgv is the sidecar-decode error for a line missing the argv key.
var errNoArgv = &sidecarFieldError{field: "argv"}

// sidecarFieldError reports a missing required sidecar field.
type sidecarFieldError struct{ field string }

func (e *sidecarFieldError) Error() string { return "missing required field " + strconv.Quote(e.field) }

// SidecarError reports a malformed sidecar line (1-based line number).
type SidecarError struct {
	Line   int
	Reason string
}

// Error renders the sidecar parse error.
func (e *SidecarError) Error() string {
	return "agenteval sidecar: line " + strconv.Itoa(e.Line) + ": " + e.Reason
}

// ParseCodexCommands parses codex's streamed log into []ObservedCommand (§2.3),
// the capture path for an FS-SANDBOXED codex sub-agent whose HOST sidecar file is
// invisible to the runner (the pilot's correction). Codex narrates each shell
// invocation on two lines:
//
//	[codex] Running command: /path/to/zsh -lc '<cmd>'
//	[codex] Command completed: /path/to/zsh -lc '<cmd>' (exit N)
//
// The parser:
//
//   - extracts the single-quoted <cmd> from each "Running command:" line (the
//     command codex actually ran inside `zsh -lc '…'`);
//   - pairs that run with its matching "Command completed:" line BY THE COMMAND
//     STRING to recover the exit code; if no completion line is seen for a run,
//     the exit defaults to 0 (the command was observed to start; absent a
//     completion we record success rather than inventing a failure);
//   - splits <cmd> into Argv with a simple, documented shell-ish tokenizer (see
//     splitShellish): whitespace-separated fields, with single- and double-quoted
//     runs kept intact and the quotes removed. This is deliberately NOT a full
//     POSIX shell parser — it handles the flat `zscalerctl … --filter k=v` /
//     `jq '...'` commands the battery grades, and the method check (§4.5) only
//     ever substring-matches the joined argv, so exact tokenization of exotic
//     quoting is not load-bearing.
//
// Commands appear in the order their "Running command:" lines appear. A line that
// is neither a run nor a completion is ignored, so interleaved agent prose never
// produces a phantom command.
func ParseCodexCommands(transcript string) []ObservedCommand {
	const (
		runMarker  = "Running command:"
		doneMarker = "Command completed:"
	)

	// First pass: collect every completion's exit code keyed by command string, so
	// a run can be paired with its completion regardless of intervening output.
	exitByCmd := map[string]int{}
	for _, line := range strings.Split(transcript, "\n") {
		idx := strings.Index(line, doneMarker)
		if idx < 0 {
			continue
		}
		rest := line[idx+len(doneMarker):]
		cmd, ok := extractQuotedCommand(rest)
		if !ok {
			continue
		}
		exitByCmd[cmd] = extractExitCode(rest)
	}

	// Second pass: emit one ObservedCommand per "Running command:" line, in order.
	var out []ObservedCommand
	for _, line := range strings.Split(transcript, "\n") {
		idx := strings.Index(line, runMarker)
		if idx < 0 {
			continue
		}
		rest := line[idx+len(runMarker):]
		cmd, ok := extractQuotedCommand(rest)
		if !ok {
			continue
		}
		exit := 0 // no completion seen -> default 0 (observed to start; no failure invented)
		if e, found := exitByCmd[cmd]; found {
			exit = e
		}
		out = append(out, ObservedCommand{Argv: splitShellish(cmd), Exit: exit})
	}
	return out
}

// extractQuotedCommand pulls the single-quoted command body out of a codex log
// tail like ` /path/zsh -lc '<cmd>' (exit N)`. Codex wraps the agent command in
// `zsh -lc '<cmd>'`, so the command is the text between the FIRST and LAST single
// quote on the line. Returns false if there is no quoted body.
func extractQuotedCommand(s string) (string, bool) {
	first := strings.IndexByte(s, '\'')
	if first < 0 {
		return "", false
	}
	last := strings.LastIndexByte(s, '\'')
	if last <= first {
		return "", false
	}
	return s[first+1 : last], true
}

// extractExitCode reads the trailing "(exit N)" from a completion line tail and
// returns N, or 0 if absent/unparseable (a completion without a parseable exit
// is treated as success, consistent with the no-completion default).
func extractExitCode(s string) int {
	const marker = "(exit "
	idx := strings.LastIndex(s, marker)
	if idx < 0 {
		return 0
	}
	rest := s[idx+len(marker):]
	end := strings.IndexByte(rest, ')')
	if end < 0 {
		return 0
	}
	n, ok := parseDecimalInt(strings.TrimSpace(rest[:end]))
	if !ok {
		return 0
	}
	return n
}

// splitShellish is the simple, documented shell-ish tokenizer ParseCodexCommands
// uses (§2.3 "a simple shell-ish split is fine; document it"). Rules:
//
//   - fields are separated by ASCII whitespace runs;
//   - a single- or double-quoted run is one field with the surrounding quotes
//     removed and inner whitespace preserved (so `jq '.[] | .name'` is the two
//     fields `jq` and `.[] | .name`);
//   - quotes may abut unquoted text within a field (e.g. country='US' -> country=US);
//   - there is NO backslash escaping, NO variable/glob expansion, NO operator
//     handling. It is intentionally minimal: the method check only substring-
//     matches the joined argv (§4.5), so this faithfully recovers the flat
//     commands the battery grades without a full POSIX parser.
func splitShellish(s string) []string {
	var (
		fields  []string
		cur     strings.Builder
		inField bool
		quote   byte // 0, '\'' or '"'
	)
	flush := func() {
		if inField {
			fields = append(fields, cur.String())
			cur.Reset()
			inField = false
		}
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			if c == quote {
				quote = 0
				continue
			}
			cur.WriteByte(c)
			inField = true
			continue
		}
		switch c {
		case '\'', '"':
			quote = c
			inField = true
		case ' ', '\t', '\r', '\n', '\v', '\f':
			flush()
		default:
			cur.WriteByte(c)
			inField = true
		}
	}
	flush()
	return fields
}

// AssembleTranscript is the small helper that builds a Transcript from an agent's
// final text and the observed commands captured for the run (§2.3). It is pure;
// it is the seam every backend's capture path funnels through (host sidecar via
// ParseSidecar, or codex log via ParseCodexCommands) before handing the result
// to the scorer.
func AssembleTranscript(agentFinalText string, commands []ObservedCommand) Transcript {
	return Transcript{AgentText: agentFinalText, Commands: commands}
}
