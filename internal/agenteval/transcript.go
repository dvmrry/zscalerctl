package agenteval

// This file is the typed v1 transcript record plus the envelope parser and the
// per-kind coercion helpers (docs/AGENTIC_COVERAGE_PLAN.md §2.1, §2.2). It is
// PURE: it never execs, dials the network, reads the clock, calls rand, or calls
// an LLM. Everything here is a deterministic function of its string/slice input.
//
// The transcript is the deterministic-half's view of one (agent, question) run:
// the agent's raw text (from which the answer envelope is mined) plus the
// authoritative observed-command stream (the §2.3 sidecar, already decoded into
// ObservedCommand records by the runner). The scorer (scorer.go) consumes this
// record and nothing else — it does not see prose beyond the envelope, and it
// trusts Commands, never the envelope's self-reported "evidence".

import (
	"encoding/json"
	"strings"
	"unicode"
)

// ObservedCommand is one entry on the observed-command sidecar (§2.3). It is the
// authoritative record of a single tool invocation the harness wrapped and
// logged — NOT the agent's self-reported evidence. The JSON tags match the
// sidecar line format the fixture binary and the jq wrapper emit
// ({"argv":[…],"exit":N}); other sidecar fields (tool, stdout_sha256, …) are
// not needed by the scorer and are intentionally dropped on decode.
type ObservedCommand struct {
	// Argv is the full argument vector of the invocation, argv[0] first.
	Argv []string `json:"argv"`
	// Exit is the process exit code the wrapper observed.
	Exit int `json:"exit"`
}

// Transcript is the typed v1 record the scorer grades (§4.2). AgentText is the
// agent's raw output, from which ParseAnswer mines the LAST answer envelope;
// Commands is the authoritative observed-command stream (§2.3). A nil/empty
// Commands is the no_commands condition (§4.2 step 1), distinct from a present
// stream with a parse-failed envelope.
type Transcript struct {
	// AgentText is the agent's raw textual output for this question.
	AgentText string
	// Commands is the authoritative observed-command stream for this run.
	Commands []ObservedCommand
}

// AnswerEnvelope is the parsed body of the one answer protocol's fenced block
// (§2.1). Answer is left as json.RawMessage so each AnswerKind's coercion
// (coerceInt/coerceBool/the set normalizer) can decide how to interpret it —
// "12", 12, and " 12 " are all valid count answers, and only the kind knows
// which coercion applies. Evidence is diagnostic only and never verdict-affecting
// (§2.1); the scorer ignores it.
type AnswerEnvelope struct {
	// Answer is the typed answer value, interpreted per the question's AnswerKind.
	Answer json.RawMessage `json:"answer"`
	// Evidence is the agent's self-reported commands; diagnostic only (§2.1).
	Evidence []string `json:"evidence"`
}

// ParseAnswer extracts the LAST AnswerOpen…AnswerClose block from agentText,
// trims it, and json.Unmarshals the body into an AnswerEnvelope (§2.1).
//
// Last-block-wins: an earlier "thinking out loud" draft is ignored in favour of
// the final block, exactly as the protocol promises. ok is false (a FAIL signal
// for the scorer, via parse_status != ok) when there is no complete
// AnswerOpen…AnswerClose pair or when the body between them is not valid JSON.
// It is fail-closed: ambiguity is a miss, never a charitable parse.
//
// Delimiter scan: AnswerOpen ("<<<ZSCTL_ANSWER") is a strict superstring of
// AnswerClose ("ZSCTL_ANSWER"), so naively searching for AnswerClose would match
// inside the opener. We therefore find each opener, then find the NEXT
// AnswerClose that begins strictly after that opener's own text. The body is the
// text between them. Scanning all opener positions and keeping the last
// well-formed block implements last-block-wins. CRLF is tolerated because the
// delimiters are matched as substrings and the JSON body's surrounding
// whitespace (including \r) is trimmed before unmarshal.
func ParseAnswer(agentText string) (AnswerEnvelope, bool) {
	var (
		bestBody  string
		bestFound bool
	)

	searchFrom := 0
	for {
		openIdx := strings.Index(agentText[searchFrom:], AnswerOpen)
		if openIdx < 0 {
			break
		}
		openIdx += searchFrom
		bodyStart := openIdx + len(AnswerOpen)

		// Find the closing delimiter strictly after the opener's own text, so the
		// opener (a superstring of the closer) can't match itself.
		closeRel := strings.Index(agentText[bodyStart:], AnswerClose)
		if closeRel < 0 {
			// Opener with no matching closer: not a complete block. Advance past
			// this opener and keep looking (there may be a complete later one, but
			// an opener without a closer can't itself contribute).
			searchFrom = bodyStart
			continue
		}
		bodyEnd := bodyStart + closeRel

		bestBody = agentText[bodyStart:bodyEnd]
		bestFound = true

		// Continue past this closer to honour last-block-wins.
		searchFrom = bodyEnd + len(AnswerClose)
	}

	if !bestFound {
		return AnswerEnvelope{}, false
	}

	body := strings.TrimSpace(bestBody)
	if body == "" {
		return AnswerEnvelope{}, false
	}

	var env AnswerEnvelope
	if err := json.Unmarshal([]byte(body), &env); err != nil {
		return AnswerEnvelope{}, false
	}
	return env, true
}

// smallNumberWords maps spelled-out small cardinals to their integer value, an
// optional convenience for coerceInt (§2.2 notes "twelve" optional). Kept small
// and exact — no fuzzy parsing — so it can never turn an ambiguous word into a
// charitable number.
var smallNumberWords = map[string]int{
	"zero": 0, "one": 1, "two": 2, "three": 3, "four": 4, "five": 5,
	"six": 6, "seven": 7, "eight": 8, "nine": 9, "ten": 10,
	"eleven": 11, "twelve": 12,
}

// coerceInt interprets a raw answer value as an integer per §2.2. It accepts a
// JSON number (12, and integral floats like 12.0), a JSON string that trims to a
// decimal integer ("12", " 12 "), and an optional small spelled-out cardinal
// ("twelve"). It is tolerant within the kind but fail-closed on ambiguity:
// anything else returns ok=false (a miss, not a charitable parse). raw is the
// envelope's Answer RawMessage.
func coerceInt(raw json.RawMessage) (int, bool) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return 0, false
	}

	// JSON number form (including integral floats such as 12.0).
	var num json.Number
	if err := json.Unmarshal(raw, &num); err == nil {
		if i, ierr := num.Int64(); ierr == nil {
			return int(i), true
		}
		if f, ferr := num.Float64(); ferr == nil && f == float64(int64(f)) {
			return int(int64(f)), true
		}
		return 0, false
	}

	// JSON string form: unwrap, then parse the inner text.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return coerceIntString(s)
	}

	// Bare (unquoted) token, e.g. when a caller passes a non-JSON snippet.
	return coerceIntString(trimmed)
}

// coerceIntString parses a single string into an int: decimal integer after
// trim, else a small spelled-out cardinal (case-fold). Fail-closed otherwise.
func coerceIntString(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	if n, ok := parseDecimalInt(s); ok {
		return n, true
	}
	if n, ok := smallNumberWords[strings.ToLower(s)]; ok {
		return n, true
	}
	return 0, false
}

// parseDecimalInt parses an optionally-signed base-10 integer with no extra
// characters. It avoids strconv's acceptance of underscores and other bases by
// validating the digit run itself.
func parseDecimalInt(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	neg := false
	body := s
	switch body[0] {
	case '+':
		body = body[1:]
	case '-':
		neg = true
		body = body[1:]
	}
	if body == "" {
		return 0, false
	}
	n := 0
	for _, r := range body {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}
	if neg {
		n = -n
	}
	return n, true
}

// coerceBool interprets a raw answer value as a truthiness-normalized bool per
// §2.2. It accepts a JSON bool (true/false), a JSON number (0 false, nonzero
// true), and a JSON/bare string from a small fixed truthy/falsey vocabulary
// (case-fold+trim: true/false, yes/no, t/f, y/n, 1/0, on/off). Anything outside
// the vocabulary returns ok=false — fail-closed, no charitable guess.
func coerceBool(raw json.RawMessage) (bool, bool) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return false, false
	}

	var b bool
	if err := json.Unmarshal(raw, &b); err == nil {
		return b, true
	}

	var num json.Number
	if err := json.Unmarshal(raw, &num); err == nil {
		if i, ierr := num.Int64(); ierr == nil {
			return i != 0, true
		}
		if f, ferr := num.Float64(); ferr == nil {
			return f != 0, true
		}
		return false, false
	}

	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return coerceBoolString(s)
	}

	return coerceBoolString(trimmed)
}

// coerceBoolString maps a single string to a bool via the fixed truthy/falsey
// vocabulary (case-fold+trim). Unknown words are a miss.
func coerceBoolString(s string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "yes", "t", "y", "1", "on":
		return true, true
	case "false", "no", "f", "n", "0", "off":
		// Recognized falsey: value false, ok TRUE. (The bare `return false, false`
		// would conflate "recognized as false" with "unrecognized".)
		return false, true
	}
	return false, false
}

// normalizeElement applies the shared, total, deterministic per-element pipeline
// of §2.2 to one string: trim → collapse internal whitespace to a single space →
// (NFC) → optional case-fold. There is no fuzzy/Levenshtein step. casefold is
// declared per comparison (the set normalizer and string_enum both pass true;
// id comparison normalizes without case-fold).
//
// NFC note: §2.2 lists Unicode NFC as a pipeline step. The eval's answer domain
// is value-free and ASCII by construction (F4: HQ, US, RFC-5737 addresses,
// RFC-2606 names, exit codes, the error_kind vocabulary) — NFC is a no-op over
// that domain. The pure grading core therefore does not pull in
// golang.org/x/text (not a direct/vendored dependency of this module) just to
// run an identity transform; the normalize pipeline's shape (trim → collapse →
// case-fold) is preserved, and the NFC slot is documented here so a future
// non-ASCII answer kind knows exactly where it belongs.
func normalizeElement(s string, casefold bool) string {
	s = collapseWhitespace(s)
	s = strings.TrimSpace(s)
	if casefold {
		s = strings.ToLower(s)
	}
	return s
}

// collapseWhitespace replaces every run of Unicode whitespace with a single
// ASCII space. Leading/trailing whitespace is left for TrimSpace to remove.
func collapseWhitespace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !inSpace {
				b.WriteByte(' ')
				inSpace = true
			}
			continue
		}
		b.WriteRune(r)
		inSpace = false
	}
	return b.String()
}

// jsonStringValue unwraps a raw answer that is a JSON string to its inner text.
// A non-string (number, bool, object, array) yields "" — string_enum and
// error_kind answers are strings by contract, and a non-string there is a miss.
func jsonStringValue(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return ""
}

// jsonScalarText renders a raw answer scalar (string OR number) as its lexical
// text, used by the id kind where "1" == 1. A JSON string yields its inner text;
// a JSON number yields its canonical decimal form; anything else yields "".
func jsonScalarText(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var num json.Number
	if err := json.Unmarshal(raw, &num); err == nil {
		return num.String()
	}
	return ""
}

// jsonObjectStringField unwraps a raw answer that is a JSON object and returns
// the first present string-valued field among keys, with ok=true. It is the
// tolerant fallback for a compound answer (§2.2 defense-in-depth): an agent that
// reports the error kind inside an object alongside other observed values, e.g.
// {"exit_code":4,"error_kind":"not_found"}, still has its kind extracted. ok is
// false when the answer is not a JSON object, or when none of the named keys is
// present with a string value — so a non-object scalar falls through to the
// normal scalar path and a wrong/absent field is never charitably matched.
func jsonObjectStringField(raw json.RawMessage, keys ...string) (string, bool) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return "", false
	}
	if obj == nil {
		return "", false
	}
	for _, k := range keys {
		v, present := obj[k]
		if !present {
			continue
		}
		var s string
		if err := json.Unmarshal(v, &s); err == nil {
			return s, true
		}
	}
	return "", false
}

// jsonStringSlice unwraps a raw answer that is a JSON array of strings. A scalar
// string is tolerated as a singleton array (an agent that answered one element
// without bracketing it). Non-string array elements and other shapes yield nil —
// they are a miss for the set kind, never a charitable parse.
func jsonStringSlice(raw json.RawMessage) []string {
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr
	}
	if s := jsonStringValue(raw); s != "" {
		return []string{s}
	}
	return nil
}

// normalizeSet turns a slice of raw strings into an order-insensitive, deduped,
// element-normalized set per §2.2 (the set kind's element normalizer). Each
// element is run through normalizeElement(casefold); empties after normalization
// are dropped; duplicates collapse. The returned map is a set (presence-keyed)
// so set comparison (§4.3) is membership math, order-free by construction.
func normalizeSet(elems []string, casefold bool) map[string]bool {
	out := make(map[string]bool, len(elems))
	for _, e := range elems {
		n := normalizeElement(e, casefold)
		if n == "" {
			continue
		}
		out[n] = true
	}
	return out
}
