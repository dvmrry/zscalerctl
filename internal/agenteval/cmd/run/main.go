// Command run is the agent-eval LIVE orchestrator (docs/AGENTIC_COVERAGE_PLAN.md
// §6, esp. §6.3/§6.4). It is the NON-DETERMINISTIC, TRACKED half: for each enabled
// roster backend × each battery question it stages a fresh hermetic sandbox,
// drives the real agent CLI against the composed prompt, grades the resulting
// transcript with the PURE scorer, aggregates per backend, and renders the floor
// report (docs/agentic-coverage.{md,json} shape) to --out.
//
// This binary is the os/exec entrypoint: together with internal/agenteval/
// backends.go it is the ONLY place the agenteval package shells out to a live
// agent. It is NEVER a CI gate (the deterministic battery+scorer drift gates run
// under `go test`); a failing floor here is tracked, never a build failure.
//
// Safety / pre-flight contract:
//
//   - It NEVER touches a live tenant: every sandbox env is the value-free
//     synthetic-credential env BuildSandbox assembles (real ZSCALERCTL_*/provider
//     tokens are stripped), and the binary it runs is the value-free fixture
//     binary, not the production CLI.
//   - It REFUSES to run if it cannot build or find the fixture binary (a missing
//     fixture binary would otherwise let an agent fall through to nothing useful;
//     pre-flight aborts instead).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dvmrry/zscalerctl/internal/agenteval"
)

// docKeyAgents and docKeySkill are the two doc paths staged into every sandbox
// (§6.3 "verbatim AGENTS.md + skills/zscalerctl/SKILL.md and nothing else"). The
// keys are the relative paths the docs are written at inside the sandbox; the
// agent reads exactly these.
const (
	docKeyAgents = "AGENTS.md"
	docKeySkill  = "skills/zscalerctl/SKILL.md"
)

// fixturePkg is the import path of the value-free fixture binary the
// orchestrator builds when --fixture-bin is not supplied.
const fixturePkg = "./internal/agenteval/cmd/zscalerctl-fixture"

// config is the parsed CLI configuration.
type config struct {
	rosterPath     string
	backends       string
	fixtureBin     string
	out            string
	agentsDoc      string
	skillDoc       string
	codexBin       string
	claudeBin      string
	reportDate     string
	transcriptsDir string
}

func main() {
	cfg := parseFlags()
	if err := run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "agent-eval:", err)
		os.Exit(1)
	}
}

// parseFlags wires the §6.4 flag surface. Defaults match the plan: roster at
// internal/agenteval/roster.json, all non-deferred backends, an auto-built
// fixture binary, and docs/agentic-coverage as the --out stem.
func parseFlags() config {
	var cfg config
	flag.StringVar(&cfg.rosterPath, "roster", "internal/agenteval/roster.json", "path to the committed roster JSON")
	flag.StringVar(&cfg.backends, "backends", "", "comma-separated agent names to ENABLE (default: all non-deferred roster entries)")
	flag.StringVar(&cfg.fixtureBin, "fixture-bin", "", "path to a prebuilt fixture binary (default: go build it to a temp file)")
	flag.StringVar(&cfg.out, "out", "docs/agentic-coverage", "output stem; writes <stem>.md and <stem>.json")
	flag.StringVar(&cfg.agentsDoc, "agents-doc", "AGENTS.md", "path to the AGENTS.md staged into each sandbox")
	flag.StringVar(&cfg.skillDoc, "skill-doc", "skills/zscalerctl/SKILL.md", "path to the SKILL.md staged into each sandbox")
	flag.StringVar(&cfg.codexBin, "codex-bin", "", "codex executable (default: codex on PATH)")
	flag.StringVar(&cfg.claudeBin, "claude-bin", "", "REAL claude executable, NOT the nix-darwin fish wrapper (default: claude on PATH)")
	flag.StringVar(&cfg.reportDate, "date", "", "report date (YYYY-MM-DD); required so the artifact is reproducible")
	flag.StringVar(&cfg.transcriptsDir, "transcripts", "", "if set, write a per-(agent,question) transcript JSON under <dir>/<agent>/<questionID>.json for triage (default: off). Intended for gitignored scratch.")
	flag.Parse()
	return cfg
}

// run is the orchestrator body, split from main so it can return an error.
func run(cfg config) error {
	if cfg.reportDate == "" {
		return fmt.Errorf("--date is required (the report date is a parameter, never time.Now, so the artifact is reproducible)")
	}

	// Load + validate the roster, then resolve the enabled backends.
	entries, err := agenteval.LoadRoster(cfg.rosterPath)
	if err != nil {
		return err
	}
	enable := parseEnable(cfg.backends)
	factoryCfg := agenteval.BackendFactoryConfig{CodexBin: cfg.codexBin, ClaudeBin: cfg.claudeBin}
	backends, err := agenteval.SelectBackends(entries, enable, factoryCfg)
	if err != nil {
		return err
	}

	// Pre-flight: stage the docs and the fixture binary. REFUSE to run without a
	// usable fixture binary (never touch a live tenant; never run a no-op sandbox).
	docs, err := loadDocs(cfg)
	if err != nil {
		return err
	}
	fixtureBin, cleanup, err := resolveFixtureBinary(cfg.fixtureBin)
	if err != nil {
		return fmt.Errorf("fixture binary pre-flight: %w", err)
	}
	defer cleanup()

	questions := agenteval.Battery()
	fmt.Fprintf(os.Stderr, "agent-eval: %d backends × %d questions; fixture=%s\n", len(backends), len(questions), fixtureBin)
	if cfg.transcriptsDir != "" {
		fmt.Fprintf(os.Stderr, "agent-eval: writing per-(agent,question) transcripts under %s\n", cfg.transcriptsDir)
	}

	// Drive every (backend, question). Each run gets a FRESH sandbox.
	ctx := context.Background()
	var runs []agenteval.AgentRun
	for _, backend := range backends {
		ar := agenteval.AgentRun{Agent: backend.Name(), Rank: backend.Rank()}
		for _, q := range questions {
			prompt, transcript, result := runOne(ctx, backend, q, fixtureBin, docs)
			ar.Results = append(ar.Results, result)
			if cfg.transcriptsDir != "" {
				if err := writeTranscript(cfg.transcriptsDir, backend.Name(), q, prompt, transcript, result); err != nil {
					// Transcript persistence is a debugging/goldens aid, never load-bearing
					// for the report — a write failure is logged, not fatal.
					fmt.Fprintf(os.Stderr, "agent-eval: %s %s: write transcript: %v\n", backend.Name(), q.ID, err)
				}
			}
		}
		runs = append(runs, ar)
		printAgentLine(ar)
	}

	// Render + write the two artifacts.
	md, jsonBytes := agenteval.Render(runs, cfg.reportDate)
	if err := writeArtifacts(cfg.out, md, jsonBytes); err != nil {
		return err
	}

	printFloor(runs)
	return nil
}

// runOne stages a fresh sandbox for one (backend, question), drives the agent,
// and grades the transcript into a QuestionResult. It also returns the composed
// prompt and the raw transcript so the caller can persist them for triage
// (--transcripts). A sandbox/exec error is NOT a surface FAIL: it is logged and
// the question is scored against an empty transcript, which the scorer treats as
// no_commands — a backend/infra problem surfaces as a (rerunnable) low result,
// not a phantom surface gap. (Capability smoke / BACKEND_UNFIT classification,
// §6.2, is a follow-up; here an exec error is simply visible in the per-agent
// tally.)
func runOne(ctx context.Context, backend agenteval.LiveBackend, q agenteval.Question, fixtureBin string, docs map[string]string) (string, agenteval.Transcript, agenteval.QuestionResult) {
	prompt := agenteval.ComposePrompt(q)

	sandbox, err := os.MkdirTemp("", "agent-eval-sandbox-")
	if err != nil {
		return prompt, agenteval.Transcript{}, scoreTranscript(q, agenteval.Transcript{})
	}
	defer os.RemoveAll(sandbox)

	env, err := agenteval.BuildSandbox(sandbox, fixtureBin, docs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "agent-eval: %s %s: build sandbox: %v\n", backend.Name(), q.ID, err)
		return prompt, agenteval.Transcript{}, scoreTranscript(q, agenteval.Transcript{})
	}

	t, err := backend.Run(ctx, sandbox, prompt, env)
	if err != nil {
		fmt.Fprintf(os.Stderr, "agent-eval: %s %s: run: %v\n", backend.Name(), q.ID, err)
		// Grade whatever transcript came back (may be empty -> no_commands).
	}
	return prompt, t, scoreTranscript(q, t)
}

// scoreTranscript grades one transcript against one question via the pure scorer
// and packages the QuestionResult (carrying the Question for the report's tier
// gate + finding attribution).
func scoreTranscript(q agenteval.Question, t agenteval.Transcript) agenteval.QuestionResult {
	verdict, finding := agenteval.Score(q, t)
	return agenteval.QuestionResult{Question: q, Verdict: verdict, Finding: finding}
}

// parseEnable turns the --backends comma list into the SelectBackends enable
// slice: an empty flag means nil (the default: all non-deferred). Whitespace
// around names is trimmed and empty fields are dropped.
func parseEnable(flagVal string) []string {
	flagVal = strings.TrimSpace(flagVal)
	if flagVal == "" {
		return nil
	}
	var out []string
	for _, name := range strings.Split(flagVal, ",") {
		if n := strings.TrimSpace(name); n != "" {
			out = append(out, n)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// loadDocs reads the two surface docs that get staged into every sandbox (§6.3).
// Both must exist — they ARE the surface under test, so a missing one is a
// configuration error, not a silent empty doc.
func loadDocs(cfg config) (map[string]string, error) {
	agentsContent, err := os.ReadFile(cfg.agentsDoc) // #nosec G304 -- flag-supplied surface doc path
	if err != nil {
		return nil, fmt.Errorf("read AGENTS.md (%s): %w", cfg.agentsDoc, err)
	}
	skillContent, err := os.ReadFile(cfg.skillDoc) // #nosec G304 -- flag-supplied surface doc path
	if err != nil {
		return nil, fmt.Errorf("read SKILL.md (%s): %w", cfg.skillDoc, err)
	}
	return map[string]string{
		docKeyAgents: string(agentsContent),
		docKeySkill:  string(skillContent),
	}, nil
}

// resolveFixtureBinary returns a path to the fixture binary, building it if
// --fixture-bin was not supplied, and a cleanup func to remove an auto-built
// temp binary. It REFUSES (returns an error) if a supplied path does not exist or
// if the build fails — the pre-flight that keeps the orchestrator from ever
// running a useless sandbox or falling through to a live reader.
func resolveFixtureBinary(supplied string) (path string, cleanup func(), err error) {
	noop := func() {}
	if supplied != "" {
		info, statErr := os.Stat(supplied)
		if statErr != nil {
			return "", noop, fmt.Errorf("supplied --fixture-bin %q: %w", supplied, statErr)
		}
		if info.IsDir() {
			return "", noop, fmt.Errorf("supplied --fixture-bin %q is a directory", supplied)
		}
		return supplied, noop, nil
	}

	tmp, err := os.MkdirTemp("", "agent-eval-fixture-")
	if err != nil {
		return "", noop, err
	}
	out := filepath.Join(tmp, "zscalerctl-fixture")
	build := exec.Command("go", "build", "-mod=vendor", "-o", out, fixturePkg) // #nosec G204 -- fixed go build of the in-repo fixture pkg
	build.Stderr = os.Stderr
	build.Stdout = os.Stderr
	if buildErr := build.Run(); buildErr != nil {
		os.RemoveAll(tmp)
		return "", noop, fmt.Errorf("go build %s: %w", fixturePkg, buildErr)
	}
	return out, func() { os.RemoveAll(tmp) }, nil
}

// writeArtifacts writes <stem>.md and <stem>.json, creating the parent directory
// as needed (so --out scratch/agent-eval writes scratch/agent-eval.{md,json}).
func writeArtifacts(stem, md string, jsonBytes []byte) error {
	if dir := filepath.Dir(stem); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil { // #nosec G301 -- ordinary artifact dir
			return fmt.Errorf("create out dir %q: %w", dir, err)
		}
	}
	if err := os.WriteFile(stem+".md", []byte(md), 0o644); err != nil { // #nosec G306 -- world-readable report
		return fmt.Errorf("write %s.md: %w", stem, err)
	}
	if err := os.WriteFile(stem+".json", jsonBytes, 0o644); err != nil { // #nosec G306 -- world-readable report
		return fmt.Errorf("write %s.json: %w", stem, err)
	}
	fmt.Fprintf(os.Stderr, "agent-eval: wrote %s.md and %s.json\n", stem, stem)
	return nil
}

// transcriptRecord is the per-(agent,question) triage artifact written under
// --transcripts (docs/AGENTIC_COVERAGE_PLAN.md §5.3: this makes every fail
// inspectable and can later seed the verdict-goldens). It captures everything a
// human (or a future golden-recorder) needs to replay and understand one run:
// the question metadata, the EXACT composed prompt the agent saw, the agent's
// raw text, the authoritative observed commands (argv + exit), the parsed answer
// (when the envelope parsed), and the scorer's Verdict + Finding.
//
// It is debugging/goldens scaffolding only — never consumed by the report — so
// it is written best-effort under gitignored scratch and a write failure is
// non-fatal.
type transcriptRecord struct {
	QuestionID  string                      `json:"question_id"`
	FailureMode string                      `json:"failure_mode"`
	Tier        string                      `json:"tier"`
	Agent       string                      `json:"agent"`
	Prompt      string                      `json:"prompt"`
	AgentText   string                      `json:"agent_text"`
	Commands    []agenteval.ObservedCommand `json:"commands"`
	AnswerOK    bool                        `json:"answer_ok"`
	Answer      *agenteval.AnswerEnvelope   `json:"answer,omitempty"`
	Verdict     agenteval.Verdict           `json:"verdict"`
	Finding     agenteval.Finding           `json:"finding"`
}

// writeTranscript persists one run's transcriptRecord to
// <dir>/<agent>/<questionID>.json (§5.3). It is pure os/file work — no exec, no
// network — so it stays in policy anywhere. The agent and question id are
// path-sanitized so a hostile/odd name can never escape the target dir, then the
// per-agent subdir is created and the record is written 2-space-indented for
// human triage. Returns an error (logged, non-fatal by the caller) on any I/O
// failure.
func writeTranscript(dir, agent string, q agenteval.Question, prompt string, t agenteval.Transcript, result agenteval.QuestionResult) error {
	rec := transcriptRecord{
		QuestionID:  q.ID,
		FailureMode: q.FailureMode,
		Tier:        q.Tier,
		Agent:       agent,
		Prompt:      prompt,
		AgentText:   t.AgentText,
		Commands:    t.Commands,
		Verdict:     result.Verdict,
		Finding:     result.Finding,
	}
	if env, ok := agenteval.ParseAnswer(t.AgentText); ok {
		rec.AnswerOK = true
		rec.Answer = &env
	}

	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal transcript: %w", err)
	}
	data = append(data, '\n')

	agentDir := filepath.Join(dir, sanitizePathComponent(agent))
	if mkErr := os.MkdirAll(agentDir, 0o755); mkErr != nil { // #nosec G301 -- ordinary scratch dir
		return fmt.Errorf("create transcript dir %q: %w", agentDir, mkErr)
	}
	path := filepath.Join(agentDir, sanitizePathComponent(q.ID)+".json")
	if wErr := os.WriteFile(path, data, 0o644); wErr != nil { // #nosec G306 -- non-secret triage artifact
		return fmt.Errorf("write transcript %q: %w", path, wErr)
	}
	return nil
}

// sanitizePathComponent reduces an agent/question name to a safe single path
// component: a path separator or `..` could otherwise let a name escape the
// target dir. Any os.PathSeparator (and the forward slash) collapses to '_', and
// a result of "." / ".." / "" is replaced with a safe placeholder. Agent and
// question ids are well-formed in practice; this is defense-in-depth so the
// env/flag-controlled output tree can never write outside <dir>.
func sanitizePathComponent(s string) string {
	mapped := strings.Map(func(r rune) rune {
		if r == '/' || r == os.PathSeparator {
			return '_'
		}
		return r
	}, s)
	switch mapped {
	case "", ".", "..":
		return "_"
	}
	return mapped
}

// printAgentLine prints one per-agent pass/warn/fail tally + clears bit to stdout
// (the operator-facing summary required by §6.4).
func printAgentLine(ar agenteval.AgentRun) {
	var pass, warn, fail int
	for _, r := range ar.Results {
		switch r.Verdict {
		case agenteval.VerdictPass:
			pass++
		case agenteval.VerdictWarn:
			warn++
		case agenteval.VerdictFail:
			fail++
		}
	}
	fmt.Printf("agent=%s rank=%d clears=%v pass=%d warn=%d fail=%d\n",
		ar.Agent, ar.Rank, agenteval.Clears(ar), pass, warn, fail)
}

// printFloor prints the headline FLOOR line (§1.2) to stdout: the weakest agent
// that clears (or that nobody does), plus the named first violation of the
// weakest non-clearing agent — the actionable gap.
func printFloor(runs []agenteval.AgentRun) {
	floorAgent, firstViolation := agenteval.Floor(runs)
	if floorAgent != "" {
		fmt.Printf("FLOOR: %s (weakest agent that clears the battery)\n", floorAgent)
	} else {
		fmt.Printf("FLOOR: none (no agent clears the battery)\n")
	}
	if firstViolation != nil {
		fmt.Printf("FIRST VIOLATION: severity=%s fm=%s agent=%s question=%s signal=%s\n",
			firstViolation.Severity, firstViolation.FailureMode, firstViolation.Agent,
			firstViolation.QuestionID, firstViolation.Signal)
	}
}
