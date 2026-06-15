// Package agenteval holds the shared substrate for the zscalerctl
// agentic-coverage eval (see docs/AGENTIC_COVERAGE_PLAN.md). It is the
// single source of truth referenced by every other file in the eval; no
// downstream file is permitted to reinvent the answer protocol, the typed
// answer contract, the definition of "clears", or the scorer output type.
//
// This file (§2 of the plan) is types + doc-comments only. It declares the
// contract; runtime behaviour (derivation, scoring, the fixture binary, the
// live runner) lands in later phases. Nothing here execs, dials the network,
// reads the clock, or calls an LLM.
//
// # The one answer protocol (§2.1)
//
// Every question prompt ends with a mandatory, fenced, machine-extractable
// block. The agent must emit it itself — the harness never injects markers.
//
//	When you have the answer, output EXACTLY one block, last, nothing after it:
//	<<<ZSCTL_ANSWER
//	{"answer": <typed value>, "evidence": ["<command you ran>", ...]}
//	ZSCTL_ANSWER
//
// Grading rules (deterministic, fail-closed):
//
//   - The grader extracts the LAST AnswerOpen…AnswerClose block and JSON-parses
//     its body. Last-block selection forgives "thinking out loud" with an
//     earlier draft.
//   - "answer" is typed per question by the question's declared AnswerKind.
//     Coercion is tolerant within a kind. Scalar answers may be emitted as the
//     scalar itself or as an unambiguous structured wrapper; ambiguity is a miss,
//     not a charitable parse.
//   - "evidence" is diagnostic only, never verdict-affecting. The authoritative
//     record of what ran is the observed-command sidecar (§2.3), not the
//     agent's self-report; a mismatch is logged as a warning, never a verdict
//     change.
//   - A missing, empty, or ill-typed envelope is a FAIL (the agent could not
//     follow the protocol — a real weakness signal for the floor).
//
// # The definition of "clears the battery" (§2.4)
//
// "Clears" is binary and has no tunable threshold. An agent clears iff:
//
//   - every Tier-0 / discovery question passes (a hard gate — the founding
//     "a weak agent could not discover object names" concern is never averaged
//     away), AND
//   - every other question is pass-or-WARN with zero method violations, AND
//   - there are zero no_commands failures.
//
// A method violation is defined only over signals observable on the one
// observed-command stream (§2.3): an answer derived from an unverified guessed
// resource name (Q3's documented recovery path is NOT a violation), a
// fabricated value the binary never emitted, or a canary leak. The tool is
// read-only, so "no write attempt" is vacuous. A single guessed-resource
// anywhere disqualifies — the claim being measured is "the surface is
// self-describing enough that this agent never had to guess."
//
// # Verdict ordering (§4.2)
//
// The scorer applies one ordered rubric (first matching row wins):
//
//  1. commands == []                                  -> FAIL (no_commands)
//  2. sidecar absent though agent claims commands      -> INCONCLUSIVE (re-run)
//  3. parse_status != ok                               -> FAIL (bad protocol)
//  4. method.must_not matched (guessed/fabricated)     -> FAIL
//  5. canary in binary output (any channel)            -> FAIL (eval-infra-leak)
//  6. method.must_run_any NOT satisfied:
//     answer correct -> WARN-capped (never PASS); answer wrong -> FAIL
//  7. method satisfied:
//     answer correct -> PASS; set within tolerance -> WARN (partial);
//     answer wrong -> FAIL. Closed required sets may opt into RequireAll,
//     which upgrades a missing expected member from WARN(partial) to FAIL.
package agenteval

// Envelope delimiters for the one answer protocol (§2.1). The grader scans for
// the LAST AnswerOpen line and the following AnswerClose line, then JSON-parses
// the body between them. These strings are part of the prompt contract and must
// not change without regenerating every recorded transcript golden.
const (
	// AnswerOpen opens the typed-answer envelope.
	AnswerOpen = "<<<ZSCTL_ANSWER"
	// AnswerClose closes the typed-answer envelope.
	AnswerClose = "ZSCTL_ANSWER"
)

// AnswerKind is the declared type of a question's canonical answer (§2.2). The
// grader extracts the envelope's "answer" value and compares it by the rule for
// this kind. Coercion is tolerant within a kind (e.g. "12", 12, "twelve" all
// coerce to the count 12 via the shared coerceInt). Scalar kinds may also accept
// unambiguous structured wrappers, but ambiguity is a miss.
type AnswerKind string

const (
	// KindCount is an exact integer, compared after coerceInt.
	KindCount AnswerKind = "count"
	// KindBool is a truthiness-normalized true/false.
	KindBool AnswerKind = "bool"
	// KindSet is an order-insensitive, deduped, element-normalized []string.
	// It is the workhorse kind and the only kind eligible for partial credit
	// (§4.3), graded as (matched, missing, extra).
	KindSet AnswerKind = "set"
	// KindStringEnum is a string compared case-fold+trim against a per-question
	// accept-set of synonyms.
	KindStringEnum AnswerKind = "string_enum"
	// KindID is a numeric/string identifier compared as trimmed strings, so
	// "1" == 1.
	KindID AnswerKind = "id"
	// KindFieldPresent answers "is field X in the emitted object?" as a bool.
	KindFieldPresent AnswerKind = "field_present"
	// KindExitCode is graded from the OBSERVED command's exit code (§2.3), not
	// from the envelope.
	KindExitCode AnswerKind = "exit_code"
	// KindErrorKind is the stderr error-envelope "kind" string, compared by
	// equality against ErrorKind (below).
	KindErrorKind AnswerKind = "error_kind"
)

// ErrorKind is the value the binary emits as the "kind" field of its stderr
// JSON error envelope.
//
// The constants below are the EXACT set of strings returned by errorKind() in
// cmd/zscalerctl/main.go, copied verbatim — note missing_credentials (not
// "credentials") and live_access_failed (not "live_api"). There is no
// translation layer: the grader compares against the literal envelope "kind"
// string, so a renamed enum would be a second place the vocabulary could drift
// from the binary.
//
// These MUST stay in lockstep with errorKind(). TestErrorKindEnumMatchesBinary
// (a future gate, §5.2) asserts this enum equals the set errorKind() produces
// and reds the build if a new kind is added to the binary without updating this
// list. Listed in errorKind() switch order; order is irrelevant for grading
// (equality, not position).
type ErrorKind string

const (
	// ErrorKindUsage corresponds to cli.ErrUsage (exit 2).
	ErrorKindUsage ErrorKind = "usage"
	// ErrorKindPartialDump corresponds to cli.ErrPartialDump (exit 6).
	ErrorKindPartialDump ErrorKind = "partial_dump"
	// ErrorKindNotFound corresponds to cli.ErrNotFound /
	// zscaler.ErrResourceNotFound (exit 4); "the id doesn't exist".
	ErrorKindNotFound ErrorKind = "not_found"
	// ErrorKindMissingCredentials corresponds to zscaler.ErrMissingCredentials
	// (exit 3).
	ErrorKindMissingCredentials ErrorKind = "missing_credentials"
	// ErrorKindInvalidResourceID corresponds to zscaler.ErrInvalidResourceID
	// (exit 2, NOT 4).
	ErrorKindInvalidResourceID ErrorKind = "invalid_resource_id"
	// ErrorKindUnsupportedResource corresponds to zscaler.ErrUnsupportedResource
	// (exit 4); "the resource key is wrong" — same exit as not_found, different
	// remediation.
	ErrorKindUnsupportedResource ErrorKind = "unsupported_resource"
	// ErrorKindLiveAccessFailed corresponds to zscaler.ErrLiveAccessFailed
	// (exit 5).
	ErrorKindLiveAccessFailed ErrorKind = "live_access_failed"
	// ErrorKindInvalidProxyConfig corresponds to zscaler.ErrInvalidProxyConfig
	// (exit 2).
	ErrorKindInvalidProxyConfig ErrorKind = "invalid_proxy_config"
	// ErrorKindInvalidConfig corresponds to config.ErrInvalidConfig (exit 2).
	ErrorKindInvalidConfig ErrorKind = "invalid_config"
	// ErrorKindInternal is the default fall-through (exit 1).
	ErrorKindInternal ErrorKind = "internal"
)

// Assertion is one typed check a question makes against a transcript (§2.2).
//
// A question carries one Assertion in the common case. The C6 dual-assertion
// case (e.g. Q9) carries two — an exit_code assertion AND an error_kind
// assertion on the same observed command — each graded independently on the
// observable channel; both must pass. This is modeled as two assertions on one
// question, never a new compound scalar kind.
type Assertion struct {
	// Kind selects the comparison rule (§2.2).
	Kind AnswerKind `json:"kind"`
	// Expected is the derived canonical answer for this assertion, serialized to
	// a string (catalog/fixture-derived, never hand-authored).
	Expected string `json:"expected"`
}

// Finding is the universal scorer output (§2.5): the single closed-loop record
// every wrong answer AND every unhealthy-path WARN emits, regardless of
// category. The report leads with the floor but is REQUIRED to enumerate open
// Findings — there is no score without findings, and only a zero-findings,
// zero-warns run is "clean".
//
// Indicts and Signal are populated by the question's own typed grader, never by
// an LLM.
type Finding struct {
	// QuestionID is the instantiated question identifier, e.g.
	// "Q-FM03-zia-filter-social-001".
	QuestionID string `json:"question_id"`
	// FailureMode is the attributed FM, e.g. "FM-03" (§4.1).
	FailureMode string `json:"failure_mode"`
	// Agent is the roster agent that produced the transcript, e.g. "haiku".
	Agent string `json:"agent"`
	// Severity is "FAIL" (wrong answer / violation) or "WARN" (right answer,
	// unhealthy path).
	Severity string `json:"severity"`
	// Indicts anchors the surface artifact(s) implicated, e.g.
	// ["AGENTS.md#narrowing-results"].
	Indicts []string `json:"indicts"`
	// Signal is the mechanical reason the grader fired (no LLM judgment).
	Signal string `json:"signal"`
	// Expected is the derived truth (from catalog/fixtures).
	Expected string `json:"expected"`
	// Got is the agent's extracted answer (clipped/redacted).
	Got string `json:"got"`
	// TranscriptRef is the path for replay.
	TranscriptRef string `json:"transcript_ref"`
}
