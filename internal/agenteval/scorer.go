package agenteval

// This file is the PURE deterministic scorer (docs/AGENTIC_COVERAGE_PLAN.md
// §4.2, §4.3, §2.2). Score maps one (Question, Transcript) to a Verdict and, on
// any non-PASS, a populated Finding. It never execs, dials the network, reads
// the clock, calls rand, reads the environment, or calls an LLM — it is a total
// function of its two arguments, which is exactly what makes it CI-gateable as
// the agentic analogue of the field-coverage drift gate (§1.3).
//
// The rubric is applied in the §4.2 order; the FIRST matching row wins. The
// scorer's only inputs about "what ran" come from Transcript.Commands (the
// authoritative observed-command sidecar, §2.3) — never the envelope's
// self-reported evidence (§2.1). Method judgments are restricted to what appears
// on that stream (§4.5).

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
)

// Verdict is the scorer's per-question outcome (§4.2). The three values are the
// closed set; there is no numeric score and no tunable threshold (§2.4).
type Verdict string

const (
	// VerdictPass is method satisfied AND every assertion correct (§4.2).
	VerdictPass Verdict = "pass"
	// VerdictWarn is a capped/partial outcome: a right answer with unverified
	// method, or set partial credit, or an over-claim with extras disallowed
	// (§4.2, §4.3). A WARN still emits a Finding (§2.5).
	VerdictWarn Verdict = "warn"
	// VerdictFail is wrong answer, protocol failure, no commands, or a method
	// violation (§4.2).
	VerdictFail Verdict = "fail"
)

// Mechanical Finding.Signal reasons (§2.5). These are the literal strings the
// scorer stamps onto a Finding; they are the "mechanical reason the grader
// fired", never an LLM judgment, and the golden tests assert them verbatim.
const (
	signalNoCommands      = "no_commands"
	signalBadEnvelope     = "bad_envelope"
	signalMethodViolation = "method_violation"
	signalNoMethod        = "no_method"
	signalWrongNoMethod   = "wrong_no_method"
	signalPartial         = "partial"
	signalCapped          = "capped"
	signalWrong           = "wrong"
)

// Question is one instantiated, gradeable question (§2.2, §3). It carries the
// typed Assertions to check, the method requirement (MustRunAny), the
// must-not-appear forbidden values (MustNot), and the attribution metadata
// (FailureMode, Indicts) used to populate a Finding.
type Question struct {
	// ID is the instantiated question identifier, e.g.
	// "Q-FM03-zia-filter-social-001". Copied verbatim into Finding.QuestionID.
	ID string
	// FailureMode is the attributed FM id, e.g. "FM-03" (§4.1), copied into
	// Finding.FailureMode.
	FailureMode string
	// Tier is the §3.3 difficulty tier the question sits in: "T0" (FLOOR /
	// discovery hard-gate), "T1" (single-surface single-command), "T2" (flag
	// composition), or "T3" (multi-step / cross-resource). It is attribution
	// metadata for the report and the coverage gates; the scorer does not branch
	// on it (the §2.4 "clears" rule reads Tier to enforce the Tier-0 hard gate,
	// but Score grades a single question identically regardless of tier).
	Tier string
	// Category is the §3.2 surface-feature category the question exercises:
	// "C1".."C6" (or the empty output-discipline cross-cut, mirroring FM-06's
	// empty FailureMode.Category). Used by TestBatteryCoversSurface to prove each
	// category is exercised; not read by the scorer.
	Category string
	// Prompt is the human/agent-facing question text (§3.1 F1: self-contained
	// from the provided surface, never naming the command to run). It is carried
	// on the Question so the live runner can present it and so battery.json
	// records exactly what was asked; the pure scorer ignores it.
	Prompt string
	// Assertions are the typed checks; length 1 in the common case, 2 for the C6
	// exit_code+error_kind dual case (§2.2). ALL must pass for a correct answer.
	Assertions []Assertion
	// ExtraAllowed is the set-kind partial-credit policy (§4.3, default false):
	// when true, extra elements beyond the expected set do not cap the verdict.
	ExtraAllowed bool
	// RequireAll is a stricter set-kind policy for closed, all-or-nothing sets:
	// when true, missing any expected set member is a FAIL instead of the generic
	// partial-set WARN. Extras are still governed by ExtraAllowed. This is for
	// questions like FM-07 credentials, where omitting a required env var is not a
	// useful partial success.
	RequireAll bool `json:",omitempty"`
	// MustRunAny is the method requirement (§2.4/§4.2 step 4): a set of argv
	// substrings. Satisfied if ANY observed command's joined argv contains ANY of
	// them. An empty MustRunAny is auto-satisfied (the question makes no method
	// claim and step 4 is skipped).
	MustRunAny []string
	// MustNot is the §4.2 step 3 forbidden-value set: substrings that must not
	// appear in the agent's answer (a fabricated/widened secret value, or the
	// eval canary). Any match is a method violation (FAIL). Empty disables the
	// check. Matched case-sensitively: canaries and secret values are exact
	// tokens, and a case-fold here would risk false positives on ordinary words.
	MustNot []string
	// Indicts is the surface artifact anchors copied into Finding.Indicts (§2.5).
	Indicts []string
}

// Score grades one transcript against one question and returns the Verdict plus,
// on any non-PASS, a populated Finding (§4.2). On PASS the returned Finding is
// the zero value (the caller emits a Finding only for non-PASS, per §2.5). The
// Finding's Agent and TranscriptRef are left blank for the caller to fill — the
// pure scorer has no knowledge of either.
//
// The rubric order (first match wins):
//
//  1. Commands empty                    -> FAIL  (no_commands)
//  2. envelope parse_status != ok       -> FAIL  (bad_envelope)
//  3. MustNot matched in the answer     -> FAIL  (method_violation)
//  4. MustRunAny NOT satisfied:
//     answer correct -> WARN (no_method); answer wrong -> FAIL (wrong_no_method)
//  5. method satisfied:
//     answer correct -> PASS; set partial -> WARN (partial) unless RequireAll;
//     set over-claim w/o extra_allowed -> WARN (capped); else -> FAIL (wrong)
func Score(q Question, t Transcript) (Verdict, Finding) {
	// Step 1: no observed commands at all -> FAIL. A zero-command transcript that
	// states the right answer overstates performance (§4.2).
	if len(t.Commands) == 0 {
		return VerdictFail, q.finding("FAIL", signalNoCommands, "", "")
	}

	// Step 2: the answer envelope must parse. Missing/ill-typed envelope is a
	// protocol failure (§2.1).
	env, ok := ParseAnswer(t.AgentText)
	if !ok {
		return VerdictFail, q.finding("FAIL", signalBadEnvelope, "", "")
	}

	answerText := string(env.Answer)

	// Step 3: must_not — a fabricated/widened secret or the canary in the answer
	// is a method violation regardless of correctness (a lucky leak is not a
	// pass) (§4.2, §4.6).
	if violation, hit := matchMustNot(q.MustNot, answerText); violation {
		return VerdictFail, q.finding("FAIL", signalMethodViolation, hit, clipAnswer(answerText))
	}

	// Evaluate correctness once: ALL assertions must pass (dual-assertion C6
	// requires both exit_code AND error_kind, §2.2). setResult is non-nil only
	// when the (single) assertion is the set kind, carrying partial-credit math.
	correct, setResult, failed := evaluateAssertions(q.Assertions, env, t, q.ExtraAllowed, q.RequireAll)

	// Step 4: method requirement. Empty MustRunAny is auto-satisfied (§2.4).
	if !methodSatisfied(q.MustRunAny, t.Commands) {
		if correct {
			// Right answer, unverified method -> WARN-capped, never PASS (§4.2).
			return VerdictWarn, q.finding("WARN", signalNoMethod, "", clipAnswer(answerText))
		}
		// Wrong answer with no method -> FAIL (§4.2).
		return VerdictFail, q.finding("FAIL", signalWrongNoMethod, failed.expected, failed.got)
	}

	// Step 5: method satisfied.
	if correct {
		return VerdictPass, Finding{}
	}

	// Set-kind partial credit (§4.3). Only the set kind reaches a WARN here; every
	// scalar/bool/enum/id is binary and a miss falls through to wrong->FAIL.
	if setResult != nil {
		switch setResult.verdict {
		case VerdictWarn:
			return VerdictWarn, q.finding("WARN", setResult.signal, setResult.expected, setResult.got)
		case VerdictFail:
			return VerdictFail, q.finding("FAIL", signalWrong, setResult.expected, setResult.got)
		}
	}

	// Wrong scalar/bool/enum/id answer with method satisfied -> FAIL (§4.2).
	return VerdictFail, q.finding("FAIL", signalWrong, failed.expected, failed.got)
}

// finding builds a Finding from the question's attribution metadata plus the
// per-grade severity/signal/expected/got. Agent and TranscriptRef are caller-set
// (§2.5); the pure scorer leaves them blank.
func (q Question) finding(severity, signal, expected, got string) Finding {
	return Finding{
		QuestionID:  q.ID,
		FailureMode: q.FailureMode,
		Severity:    severity,
		Indicts:     q.Indicts,
		Signal:      signal,
		Expected:    expected,
		Got:         got,
	}
}

// matchMustNot reports whether any forbidden substring appears verbatim in the
// answer text, returning the first hit for the Finding (§4.2 step 3). Empty
// forbidden set or empty answer never matches.
func matchMustNot(forbidden []string, answerText string) (bool, string) {
	for _, f := range forbidden {
		if f == "" {
			continue
		}
		if strings.Contains(answerText, f) {
			return true, f
		}
	}
	return false, ""
}

// methodSatisfied reports whether the method requirement holds (§2.4/§4.2 step
// 4). Empty MustRunAny is auto-satisfied. Otherwise it is satisfied iff ANY
// observed command's space-joined argv contains ANY required substring.
func methodSatisfied(mustRunAny []string, commands []ObservedCommand) bool {
	if len(mustRunAny) == 0 {
		return true
	}
	for _, cmd := range commands {
		joined := strings.Join(cmd.Argv, " ")
		for _, sub := range mustRunAny {
			if sub == "" {
				continue
			}
			if strings.Contains(joined, sub) {
				return true
			}
		}
	}
	return false
}

// setGrade is the result of grading a set-kind assertion (§4.3): the (matched,
// missing, extra) math reduced to a verdict plus the report-facing strings.
type setGrade struct {
	verdict  Verdict
	signal   string
	expected string
	got      string
}

// assertionGrade is the binary grade for a non-set assertion. expected/got are
// compact diagnostics for the report, not grading inputs; they make grader
// misses inspectable without dumping the whole raw envelope into a Finding.
type assertionGrade struct {
	pass     bool
	expected string
	got      string
}

// evaluateAssertions grades every assertion in the question and reports whether
// ALL passed (dual-assertion C6 requires both, §2.2). It returns:
//   - correct: true iff every assertion passed.
//   - setResult: the partial-credit grade IFF the question is a single set-kind
//     assertion (so step 5 can award WARN); nil otherwise.
//   - failed: the first failing assertion grade (for Finding.Expected/Got); the
//     zero assertionGrade if all passed.
//
// extraAllowed and requireAll are question policies threaded only to the set path.
func evaluateAssertions(assertions []Assertion, env AnswerEnvelope, t Transcript, extraAllowed, requireAll bool) (correct bool, setResult *setGrade, failed assertionGrade) {
	// Single set-kind assertion: grade via the partial-credit table so step 5 can
	// distinguish PASS / WARN(partial) / WARN(capped) / FAIL.
	if len(assertions) == 1 && assertions[0].Kind == KindSet {
		a := assertions[0]
		grade := gradeSet(a, env, extraAllowed, requireAll)
		if grade.verdict == VerdictPass {
			return true, &grade, assertionGrade{}
		}
		return false, &grade, assertionGrade{expected: grade.expected, got: grade.got}
	}

	for _, a := range assertions {
		grade := gradeAssertion(a, env, t)
		if !grade.pass {
			return false, nil, grade
		}
	}
	return true, nil, assertionGrade{}
}

// gradeAssertion grades a single non-set assertion to a binary pass/fail per its
// kind (§2.2). It is tolerant about JSON answer shape but fail-closed about
// content: scalar kinds accept scalar answers and unambiguous structured
// wrappers, while ambiguous wrappers fail instead of being guessed. There is no
// partial credit outside the set kind (§4.3).
func gradeAssertion(a Assertion, env AnswerEnvelope, t Transcript) assertionGrade {
	switch a.Kind {
	case KindCount:
		want, ok := coerceIntString(a.Expected)
		if !ok {
			return assertionGrade{expected: a.Expected, got: "<invalid expected count>"}
		}
		got, gotText, ok := intAnswerCandidate(env.Answer)
		return assertionGrade{pass: ok && got == want, expected: strconv.Itoa(want), got: gotText}

	case KindBool, KindFieldPresent:
		want, ok := coerceBoolString(a.Expected)
		if !ok {
			return assertionGrade{expected: a.Expected, got: "<invalid expected bool>"}
		}
		got, gotText, ok := boolAnswerCandidate(env.Answer)
		return assertionGrade{pass: ok && got == want, expected: strconv.FormatBool(want), got: gotText}

	case KindID:
		// id compares trimmed-string-equal ("1" == 1). Unwrap a JSON string/number
		// to its lexical form, or accept an id-like field in a rich object, then
		// compare trimmed (no case-fold for ids).
		got := normalizeElement(idAnswerCandidate(env.Answer), false)
		want := normalizeElement(a.Expected, false)
		return assertionGrade{pass: got != "" && got == want, expected: want, got: got}

	case KindStringEnum:
		// string_enum: case-fold+trim the answer, accept if it matches ANY synonym
		// in the per-assertion accept-set encoded pipe-separated in Expected. A
		// structured wrapper passes only when it has one unambiguous string leaf.
		got := normalizeElement(stringAnswerCandidate(env.Answer), true)
		if got == "" {
			return assertionGrade{expected: a.Expected, got: ""}
		}
		for _, syn := range strings.Split(a.Expected, "|") {
			if normalizeElement(syn, true) == got {
				return assertionGrade{pass: true}
			}
		}
		return assertionGrade{expected: a.Expected, got: got}

	case KindExitCode:
		// exit_code is graded from the OBSERVED commands, NOT the envelope (§2.2):
		// pass iff some observed command's Exit equals the expected code.
		want, ok := coerceIntString(a.Expected)
		if !ok {
			return assertionGrade{expected: a.Expected, got: "<invalid expected exit code>"}
		}
		exits := make([]string, 0, len(t.Commands))
		for _, cmd := range t.Commands {
			exits = append(exits, strconv.Itoa(cmd.Exit))
			if cmd.Exit == want {
				return assertionGrade{pass: true}
			}
		}
		return assertionGrade{expected: strconv.Itoa(want), got: strings.Join(exits, ",")}

	case KindErrorKind:
		// error_kind compares the envelope's typed answer (a string) to the
		// expected kind; the value must be one of the contract's ErrorKind set, so
		// a typo'd kind is a miss, not a charitable match.
		//
		// Defense-in-depth (§2.2: error_kind is graded from the envelope, exit_code
		// from the observed command): an agent prompted loosely may wrap the kind in
		// a compound object alongside other fields it observed, e.g.
		// {"exit_code":4,"error_kind":"not_found"}. When the answer is a JSON object
		// we accept a matching value found under an "error_kind"/"errorKind"/"kind"
		// field before falling back to the scalar compare, so a correct kind inside a
		// compound answer is not a false negative. The scalar string remains the
		// preferred form (what the prompt asks for); the object form is a tolerant
		// fallback, never a charitable parse of a wrong value.
		want := normalizeElement(a.Expected, false)
		if field, ok := jsonObjectStringField(env.Answer, "error_kind", "errorKind", "kind"); ok {
			got := normalizeElement(field, false)
			if !validErrorKind(got) {
				return assertionGrade{expected: want, got: got}
			}
			return assertionGrade{pass: got == want, expected: want, got: got}
		}
		got := normalizeElement(jsonStringValue(env.Answer), false)
		if !validErrorKind(got) {
			return assertionGrade{expected: want, got: got}
		}
		return assertionGrade{pass: got == want, expected: want, got: got}

	case KindSet:
		// A set assertion in a multi-assertion question is graded binary here
		// (all-or-nothing); partial credit only applies to a question whose SOLE
		// assertion is a set (handled in evaluateAssertions).
		grade := gradeSetExact(a, env)
		return assertionGrade{pass: grade.verdict == VerdictPass, expected: grade.expected, got: grade.got}

	default:
		return assertionGrade{expected: a.Expected, got: "<unsupported assertion kind>"}
	}
}

// gradeSet computes (matched, missing, extra) for a set assertion and maps it to
// a verdict per the §4.3 table. extraAllowed selects the right column.
// requireAll upgrades missing expected members from WARN(partial) to FAIL for
// closed sets where every expected element is required.
func gradeSet(a Assertion, env AnswerEnvelope, extraAllowed, requireAll bool) setGrade {
	expected := normalizeSet(splitSetExpected(a.Expected), true)
	got := normalizeSet(jsonStringSlice(env.Answer), true)

	matched := 0
	for e := range expected {
		if got[e] {
			matched++
		}
	}
	extra := 0
	for g := range got {
		if !expected[g] {
			extra++
		}
	}
	exp := strconv.Itoa(matched) + "/" + strconv.Itoa(len(expected)) + " expected: " + joinSortedSet(expected)
	gotStr := joinSortedSet(got)

	switch {
	case matched == len(expected) && extra == 0:
		return setGrade{verdict: VerdictPass}
	case matched == 0:
		return setGrade{verdict: VerdictFail, signal: signalWrong, expected: exp, got: gotStr}
	case requireAll && matched < len(expected):
		return setGrade{verdict: VerdictFail, signal: signalWrong, expected: exp, got: gotStr}
	case extra > 0:
		// Over-claim: PASS only if extras are allowed; else WARN-capped (§4.3).
		if extraAllowed {
			return setGrade{verdict: VerdictPass}
		}
		return setGrade{verdict: VerdictWarn, signal: signalCapped, expected: exp, got: gotStr}
	default:
		// 0 < matched < expected, extra == 0 -> WARN (partial).
		return setGrade{verdict: VerdictWarn, signal: signalPartial, expected: exp, got: gotStr}
	}
}

// gradeSetExact grades a set assertion with extra_allowed forced false, for the
// multi-assertion binary path (the only PASS is a perfect match).
func gradeSetExact(a Assertion, env AnswerEnvelope) setGrade {
	return gradeSet(a, env, false, true)
}

// intAnswerCandidate extracts exactly one integer candidate from the answer. A
// plain scalar uses the existing coerceInt path. A structured JSON wrapper is
// accepted only when every integer-shaped leaf collapses to the same value; if
// it contains multiple different integers the answer is ambiguous and fails
// closed.
func intAnswerCandidate(raw json.RawMessage) (int, string, bool) {
	if got, ok := coerceInt(raw); ok {
		return got, strconv.Itoa(got), true
	}
	candidates := map[int]bool{}
	collectIntCandidates(raw, candidates)
	return uniqueIntCandidate(candidates)
}

func collectIntCandidates(raw json.RawMessage, out map[int]bool) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return
	}
	collectIntValueCandidates(v, out)
}

func collectIntValueCandidates(v any, out map[int]bool) {
	switch t := v.(type) {
	case float64:
		if t == float64(int64(t)) {
			out[int(int64(t))] = true
		}
	case string:
		if i, ok := coerceIntString(t); ok {
			out[i] = true
		}
	case []any:
		for _, elem := range t {
			collectIntValueCandidates(elem, out)
		}
	case map[string]any:
		for _, elem := range t {
			collectIntValueCandidates(elem, out)
		}
	}
}

func uniqueIntCandidate(candidates map[int]bool) (int, string, bool) {
	switch len(candidates) {
	case 0:
		return 0, "", false
	case 1:
		for v := range candidates {
			return v, strconv.Itoa(v), true
		}
	}
	return 0, joinSortedInts(candidates), false
}

// boolAnswerCandidate extracts exactly one bool candidate from the answer. Like
// intAnswerCandidate, structured wrappers are accepted only when unambiguous.
func boolAnswerCandidate(raw json.RawMessage) (bool, string, bool) {
	if got, ok := coerceBool(raw); ok {
		return got, strconv.FormatBool(got), true
	}
	candidates := map[bool]bool{}
	var v any
	if err := json.Unmarshal(raw, &v); err == nil {
		collectBoolValueCandidates(v, candidates)
	}
	switch len(candidates) {
	case 0:
		return false, "", false
	case 1:
		for v := range candidates {
			return v, strconv.FormatBool(v), true
		}
	}
	return false, "false,true", false
}

func collectBoolValueCandidates(v any, out map[bool]bool) {
	switch t := v.(type) {
	case bool:
		out[t] = true
	case string:
		if b, ok := coerceBoolString(t); ok {
			out[b] = true
		}
	case []any:
		for _, elem := range t {
			collectBoolValueCandidates(elem, out)
		}
	case map[string]any:
		for _, elem := range t {
			collectBoolValueCandidates(elem, out)
		}
	}
}

// idAnswerCandidate extracts an id answer. A scalar string/number is preferred.
// Structured answers are accepted only when they carry an id-like field, so a
// rich record object {"id":"1","name":"HQ"} can answer an id question without
// making arbitrary string leaves eligible.
func idAnswerCandidate(raw json.RawMessage) string {
	if got := jsonScalarText(raw); got != "" {
		return got
	}
	if field, ok := jsonObjectScalarField(raw, "id", "ID", "resource_id", "resourceId"); ok {
		return field
	}
	return ""
}

// stringAnswerCandidate extracts a string answer. A scalar string is preferred;
// a structured wrapper is accepted only when it contains exactly one unique
// string leaf. That lets {"country":"US"} pass while rejecting
// {"country":"US","resource":"locations"} as ambiguous instead of guessed.
func stringAnswerCandidate(raw json.RawMessage) string {
	if got := jsonStringValue(raw); got != "" {
		return got
	}
	set := normalizeSet(jsonStringSlice(raw), false)
	if len(set) != 1 {
		return ""
	}
	for v := range set {
		return v
	}
	return ""
}

// jsonObjectScalarField unwraps a JSON object field that is a string or number,
// returning its lexical text. Non-object answers and non-scalar fields miss.
func jsonObjectScalarField(raw json.RawMessage, keys ...string) (string, bool) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return "", false
	}
	for _, k := range keys {
		v, present := obj[k]
		if !present {
			continue
		}
		got := jsonScalarText(v)
		if got != "" {
			return got, true
		}
	}
	return "", false
}

func joinSortedInts(set map[int]bool) string {
	vals := make([]int, 0, len(set))
	for v := range set {
		vals = append(vals, v)
	}
	sort.Ints(vals)
	out := make([]string, 0, len(vals))
	for _, v := range vals {
		out = append(out, strconv.Itoa(v))
	}
	return strings.Join(out, ",")
}

// validErrorKind reports whether s is one of the contract's ErrorKind values
// (§2.2). The grader never accepts a kind the binary cannot emit.
func validErrorKind(s string) bool {
	for _, k := range allErrorKinds {
		if string(k) == s {
			return true
		}
	}
	return false
}

// allErrorKinds is the closed ErrorKind set from contract.go, used to validate
// an error_kind answer. Kept in lockstep with the contract constants
// (TestErrorKindEnumMatchesBinary gates the contract against the binary).
var allErrorKinds = []ErrorKind{
	ErrorKindUsage,
	ErrorKindPartialDump,
	ErrorKindNotFound,
	ErrorKindMissingCredentials,
	ErrorKindInvalidResourceID,
	ErrorKindUnsupportedResource,
	ErrorKindLiveAccessFailed,
	ErrorKindInvalidProxyConfig,
	ErrorKindInvalidConfig,
	ErrorKindInternal,
}

// splitSetExpected splits a set assertion's Expected encoding into elements. The
// encoding is comma-separated (e.g. "id,name"); whitespace around each element
// is left for the set normalizer to trim.
func splitSetExpected(expected string) []string {
	if strings.TrimSpace(expected) == "" {
		return nil
	}
	return strings.Split(expected, ",")
}

// joinSortedSet renders a normalized set as a deterministic, sorted,
// comma-joined string for Finding.Expected/Got (report stability).
func joinSortedSet(set map[string]bool) string {
	elems := make([]string, 0, len(set))
	for e := range set {
		elems = append(elems, e)
	}
	sort.Strings(elems)
	return strings.Join(elems, ",")
}

// clipAnswer bounds the answer text stamped into Finding.Got so a verbose answer
// can't bloat a report (§2.5 "clipped/redacted"). It is purely cosmetic.
func clipAnswer(s string) string {
	const max = 120
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
