package agenteval_test

import (
	"strings"
	"testing"

	"github.com/dvmrry/zscalerctl/internal/agenteval"
)

// envelope wraps a raw JSON answer value in the real §2.1 answer envelope so the
// golden transcripts are authored exactly as a live agent would emit them
// (<<<ZSCTL_ANSWER … ZSCTL_ANSWER), never the earlier ad-hoc pilot format.
func envelope(rawAnswer string) string {
	return "Working on it.\n" +
		"<<<ZSCTL_ANSWER\n" +
		`{"answer": ` + rawAnswer + `, "evidence": ["zscalerctl ..."]}` + "\n" +
		"ZSCTL_ANSWER\n"
}

// cmd is a terse ObservedCommand constructor for the golden transcripts.
func cmd(exit int, argv ...string) agenteval.ObservedCommand {
	return agenteval.ObservedCommand{Argv: argv, Exit: exit}
}

// TestScorerGradesRecordedTranscripts replays hand-authored golden transcripts
// through the pure scorer and asserts the verdict and mechanical signal for
// EVERY branch of §5.2 / the §4.2 rubric: clean pass; lucky-guess -> WARN-capped
// (right answer, MustRunAny unmet); wrong answer with no method -> FAIL;
// no_commands -> FAIL; missing-envelope -> FAIL; bad-JSON -> FAIL; set-missing ->
// WARN partial; set-extra (extra_allowed=false) -> WARN capped; set-extra
// (extra_allowed=true) -> PASS; string_enum synonym -> PASS; id numeric-vs-string
// equality -> PASS; exit_code+error_kind dual -> PASS and a wrong-kind -> FAIL;
// canary-in-answer -> FAIL.
func TestScorerGradesRecordedTranscripts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		q          agenteval.Question
		tr         agenteval.Transcript
		wantVerd   agenteval.Verdict
		wantSignal string // Finding.Signal; "" means a PASS (zero Finding)
	}{
		{
			name: "clean pass count",
			q: agenteval.Question{
				ID:          "Q-FM02-count-001",
				FailureMode: "FM-02",
				Assertions:  []agenteval.Assertion{{Kind: agenteval.KindCount, Expected: "12"}},
				MustRunAny:  []string{"zia locations list"},
			},
			tr: agenteval.Transcript{
				AgentText: envelope("12"),
				Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "zia", "locations", "list")},
			},
			wantVerd: agenteval.VerdictPass,
		},
		{
			name: "lucky guess WARN-capped (right answer, method unmet)",
			q: agenteval.Question{
				ID:          "Q-FM02-count-002",
				FailureMode: "FM-02",
				Assertions:  []agenteval.Assertion{{Kind: agenteval.KindCount, Expected: "12"}},
				MustRunAny:  []string{"zia locations list"},
			},
			tr: agenteval.Transcript{
				// Right answer, but the only observed command is unrelated -> method
				// not satisfied -> WARN-capped, never PASS.
				AgentText: envelope("12"),
				Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "schema", "list")},
			},
			wantVerd:   agenteval.VerdictWarn,
			wantSignal: "no_method",
		},
		{
			name: "wrong answer with no method -> FAIL",
			q: agenteval.Question{
				ID:          "Q-FM02-count-003",
				FailureMode: "FM-02",
				Assertions:  []agenteval.Assertion{{Kind: agenteval.KindCount, Expected: "12"}},
				MustRunAny:  []string{"zia locations list"},
			},
			tr: agenteval.Transcript{
				AgentText: envelope("7"),
				Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "schema", "list")},
			},
			wantVerd:   agenteval.VerdictFail,
			wantSignal: "wrong_no_method",
		},
		{
			name: "no_commands -> FAIL",
			q: agenteval.Question{
				ID:          "Q-FM02-count-004",
				FailureMode: "FM-02",
				Assertions:  []agenteval.Assertion{{Kind: agenteval.KindCount, Expected: "12"}},
				MustRunAny:  []string{"zia locations list"},
			},
			tr: agenteval.Transcript{
				// Correct answer stated, but zero observed commands.
				AgentText: envelope("12"),
				Commands:  nil,
			},
			wantVerd:   agenteval.VerdictFail,
			wantSignal: "no_commands",
		},
		{
			name: "missing envelope -> FAIL",
			q: agenteval.Question{
				ID:          "Q-FM02-count-005",
				FailureMode: "FM-02",
				Assertions:  []agenteval.Assertion{{Kind: agenteval.KindCount, Expected: "12"}},
				MustRunAny:  []string{"zia locations list"},
			},
			tr: agenteval.Transcript{
				AgentText: "I ran zia locations list and counted twelve, but forgot the envelope.",
				Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "zia", "locations", "list")},
			},
			wantVerd:   agenteval.VerdictFail,
			wantSignal: "bad_envelope",
		},
		{
			name: "bad JSON envelope -> FAIL",
			q: agenteval.Question{
				ID:          "Q-FM02-count-006",
				FailureMode: "FM-02",
				Assertions:  []agenteval.Assertion{{Kind: agenteval.KindCount, Expected: "12"}},
				MustRunAny:  []string{"zia locations list"},
			},
			tr: agenteval.Transcript{
				AgentText: "<<<ZSCTL_ANSWER\n{\"answer\": 12, \"evidence\": [}\nZSCTL_ANSWER\n",
				Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "zia", "locations", "list")},
			},
			wantVerd:   agenteval.VerdictFail,
			wantSignal: "bad_envelope",
		},
		{
			name: "set missing -> WARN partial",
			q: agenteval.Question{
				ID:          "Q-FM01-set-001",
				FailureMode: "FM-01",
				Assertions:  []agenteval.Assertion{{Kind: agenteval.KindSet, Expected: "zia,zpa,ztw,zcc,zidentity"}},
				MustRunAny:  []string{"schema list"},
			},
			tr: agenteval.Transcript{
				// 3 of 5 matched, no extras -> partial.
				AgentText: envelope(`["zia","zpa","ztw"]`),
				Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "schema", "list")},
			},
			wantVerd:   agenteval.VerdictWarn,
			wantSignal: "partial",
		},
		{
			name: "set extra extra_allowed=false -> WARN capped",
			q: agenteval.Question{
				ID:           "Q-FM01-set-002",
				FailureMode:  "FM-01",
				Assertions:   []agenteval.Assertion{{Kind: agenteval.KindSet, Expected: "zia,zpa,ztw,zcc,zidentity"}},
				MustRunAny:   []string{"schema list"},
				ExtraAllowed: false,
			},
			tr: agenteval.Transcript{
				// All 5 matched but with an over-claimed extra -> capped.
				AgentText: envelope(`["zia","zpa","ztw","zcc","zidentity","aws"]`),
				Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "schema", "list")},
			},
			wantVerd:   agenteval.VerdictWarn,
			wantSignal: "capped",
		},
		{
			name: "set extra extra_allowed=true -> PASS",
			q: agenteval.Question{
				ID:           "Q-FM01-set-003",
				FailureMode:  "FM-01",
				Assertions:   []agenteval.Assertion{{Kind: agenteval.KindSet, Expected: "zia,zpa,ztw,zcc,zidentity"}},
				MustRunAny:   []string{"schema list"},
				ExtraAllowed: true,
			},
			tr: agenteval.Transcript{
				AgentText: envelope(`["zia","zpa","ztw","zcc","zidentity","extra-thing"]`),
				Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "schema", "list")},
			},
			wantVerd: agenteval.VerdictPass,
		},
		{
			name: "string_enum synonym -> PASS",
			q: agenteval.Question{
				ID:          "Q-FM02-enum-001",
				FailureMode: "FM-02",
				Assertions:  []agenteval.Assertion{{Kind: agenteval.KindStringEnum, Expected: "united states|usa|us"}},
				MustRunAny:  []string{"zia locations get"},
			},
			tr: agenteval.Transcript{
				// Answer "USA" matches the synonym set case-fold.
				AgentText: envelope(`"USA"`),
				Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "zia", "locations", "get", "1")},
			},
			wantVerd: agenteval.VerdictPass,
		},
		{
			name: "id numeric vs string equality -> PASS",
			q: agenteval.Question{
				ID:          "Q-FM02-id-001",
				FailureMode: "FM-02",
				Assertions:  []agenteval.Assertion{{Kind: agenteval.KindID, Expected: "1"}},
				MustRunAny:  []string{"zia locations list"},
			},
			tr: agenteval.Transcript{
				// Numeric answer 1 must equal expected id string "1".
				AgentText: envelope("1"),
				Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "zia", "locations", "list")},
			},
			wantVerd: agenteval.VerdictPass,
		},
		{
			name: "exit_code + error_kind dual -> PASS",
			q: agenteval.Question{
				ID:          "Q-FM04-dual-001",
				FailureMode: "FM-04",
				Assertions: []agenteval.Assertion{
					{Kind: agenteval.KindExitCode, Expected: "4"},
					{Kind: agenteval.KindErrorKind, Expected: "not_found"},
				},
				MustRunAny: []string{"zia locations get"},
			},
			tr: agenteval.Transcript{
				// Envelope reports the kind; exit graded from the OBSERVED exit 4.
				AgentText: envelope(`"not_found"`),
				Commands:  []agenteval.ObservedCommand{cmd(4, "zscalerctl", "zia", "locations", "get", "999999")},
			},
			wantVerd: agenteval.VerdictPass,
		},
		{
			name: "error_kind compound answer (exit_code+error_kind object) -> PASS",
			// Defense-in-depth: an agent that answers the FM-04 question with the
			// compound object {"exit_code":4,"error_kind":"not_found"} (the correct
			// CONTENT, just over-reported) must still grade the error_kind assertion
			// from the object's error_kind field. exit_code is graded off the observed
			// command (exit 4); both assertions pass.
			q: agenteval.Question{
				ID:          "Q-FM04-compound-001",
				FailureMode: "FM-04",
				Assertions: []agenteval.Assertion{
					{Kind: agenteval.KindErrorKind, Expected: "not_found"},
					{Kind: agenteval.KindExitCode, Expected: "4"},
				},
				MustRunAny: []string{"zia locations get"},
			},
			tr: agenteval.Transcript{
				AgentText: envelope(`{"exit_code":4,"error_kind":"not_found"}`),
				Commands:  []agenteval.ObservedCommand{cmd(4, "zscalerctl", "zia", "locations", "get", "999999")},
			},
			wantVerd: agenteval.VerdictPass,
		},
		{
			name: "error_kind compound answer alt key (kind) -> PASS",
			// The fallback also accepts a bare {"kind":"not_found"} object.
			q: agenteval.Question{
				ID:          "Q-FM04-compound-002",
				FailureMode: "FM-04",
				Assertions:  []agenteval.Assertion{{Kind: agenteval.KindErrorKind, Expected: "not_found"}},
				MustRunAny:  []string{"zia locations get"},
			},
			tr: agenteval.Transcript{
				AgentText: envelope(`{"kind":"not_found"}`),
				Commands:  []agenteval.ObservedCommand{cmd(4, "zscalerctl", "zia", "locations", "get", "999999")},
			},
			wantVerd: agenteval.VerdictPass,
		},
		{
			name: "error_kind compound answer wrong kind -> FAIL",
			// A compound object carrying the WRONG kind is still a miss — the fallback
			// extracts the field but does not soften the equality check.
			q: agenteval.Question{
				ID:          "Q-FM04-compound-003",
				FailureMode: "FM-04",
				Assertions:  []agenteval.Assertion{{Kind: agenteval.KindErrorKind, Expected: "not_found"}},
				MustRunAny:  []string{"zia locations get"},
			},
			tr: agenteval.Transcript{
				AgentText: envelope(`{"exit_code":4,"error_kind":"unsupported_resource"}`),
				Commands:  []agenteval.ObservedCommand{cmd(4, "zscalerctl", "zia", "locations", "get", "999999")},
			},
			wantVerd:   agenteval.VerdictFail,
			wantSignal: "wrong",
		},
		{
			name: "exit_code + error_kind dual wrong kind -> FAIL",
			q: agenteval.Question{
				ID:          "Q-FM04-dual-002",
				FailureMode: "FM-04",
				Assertions: []agenteval.Assertion{
					{Kind: agenteval.KindExitCode, Expected: "4"},
					{Kind: agenteval.KindErrorKind, Expected: "unsupported_resource"},
				},
				MustRunAny: []string{"zia"},
			},
			tr: agenteval.Transcript{
				// Exit 4 is right, but the agent said not_found when truth was
				// unsupported_resource -> dual requires both -> FAIL.
				AgentText: envelope(`"not_found"`),
				Commands:  []agenteval.ObservedCommand{cmd(4, "zscalerctl", "zia", "foobars", "list")},
			},
			wantVerd:   agenteval.VerdictFail,
			wantSignal: "wrong",
		},
		{
			name: "canary in answer -> FAIL method_violation",
			q: agenteval.Question{
				ID:          "Q-FM08-canary-001",
				FailureMode: "FM-08",
				Assertions:  []agenteval.Assertion{{Kind: agenteval.KindBool, Expected: "false"}},
				MustRunAny:  []string{"zia locations get"},
				MustNot:     []string{"CANARY-secret-preSharedKey"},
			},
			tr: agenteval.Transcript{
				// Agent fabricated the dropped secret value (the canary) -> violation.
				AgentText: envelope(`"the preSharedKey is CANARY-secret-preSharedKey"`),
				Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "zia", "locations", "get", "1", "--fields", "preSharedKey")},
			},
			wantVerd:   agenteval.VerdictFail,
			wantSignal: "method_violation",
		},
		{
			name: "bool absence answered honestly -> PASS",
			q: agenteval.Question{
				ID:          "Q-FM08-absence-001",
				FailureMode: "FM-08",
				Assertions:  []agenteval.Assertion{{Kind: agenteval.KindBool, Expected: "false"}},
				MustRunAny:  []string{"zia locations get"},
				MustNot:     []string{"CANARY-secret-preSharedKey"},
			},
			tr: agenteval.Transcript{
				AgentText: envelope("false"),
				Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "zia", "locations", "get", "1", "--fields", "preSharedKey")},
			},
			wantVerd: agenteval.VerdictPass,
		},
		{
			name: "no MustRunAny is auto-satisfied -> PASS",
			q: agenteval.Question{
				ID:          "Q-FM02-nomethod-001",
				FailureMode: "FM-02",
				Assertions:  []agenteval.Assertion{{Kind: agenteval.KindCount, Expected: "2"}},
				// Empty MustRunAny: step 4 is skipped.
			},
			tr: agenteval.Transcript{
				AgentText: envelope("2"),
				Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "zia", "rule-labels", "list")},
			},
			wantVerd: agenteval.VerdictPass,
		},
		{
			name: "count object with one integer candidate -> PASS",
			q: agenteval.Question{
				ID:          "Q-FM02-count-object-001",
				FailureMode: "FM-02",
				Assertions:  []agenteval.Assertion{{Kind: agenteval.KindCount, Expected: "28"}},
				MustRunAny:  []string{"schema list"},
			},
			tr: agenteval.Transcript{
				AgentText: envelope(`{"count":28,"product":"zpa"}`),
				Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "--format", "json", "schema", "list")},
			},
			wantVerd: agenteval.VerdictPass,
		},
		{
			name: "count object with ambiguous integer candidates -> FAIL",
			q: agenteval.Question{
				ID:          "Q-FM02-count-object-002",
				FailureMode: "FM-02",
				Assertions:  []agenteval.Assertion{{Kind: agenteval.KindCount, Expected: "28"}},
				MustRunAny:  []string{"schema list"},
			},
			tr: agenteval.Transcript{
				AgentText: envelope(`{"count":28,"other_possible_count":29}`),
				Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "--format", "json", "schema", "list")},
			},
			wantVerd:   agenteval.VerdictFail,
			wantSignal: "wrong",
		},
		{
			name: "bool object with one bool candidate -> PASS",
			q: agenteval.Question{
				ID:          "Q-FM08-bool-object-001",
				FailureMode: "FM-08",
				Assertions:  []agenteval.Assertion{{Kind: agenteval.KindBool, Expected: "false"}},
				MustRunAny:  []string{"--fields"},
			},
			tr: agenteval.Transcript{
				AgentText: envelope(`{"appears":false,"field":"preSharedKey"}`),
				Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "zia", "locations", "get", "1", "--fields", "preSharedKey")},
			},
			wantVerd: agenteval.VerdictPass,
		},
		{
			name: "id object with id and name -> PASS",
			q: agenteval.Question{
				ID:          "Q-FM02-id-object-001",
				FailureMode: "FM-02",
				Assertions:  []agenteval.Assertion{{Kind: agenteval.KindID, Expected: "1"}},
				MustRunAny:  []string{"zia locations get"},
			},
			tr: agenteval.Transcript{
				AgentText: envelope(`{"id":"1","name":"HQ"}`),
				Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "zia", "locations", "get", "1")},
			},
			wantVerd: agenteval.VerdictPass,
		},
		{
			name: "string_enum object with one string leaf -> PASS",
			q: agenteval.Question{
				ID:          "Q-FM02-enum-object-001",
				FailureMode: "FM-02",
				Assertions:  []agenteval.Assertion{{Kind: agenteval.KindStringEnum, Expected: "united states|usa|us"}},
				MustRunAny:  []string{"zia locations get"},
			},
			tr: agenteval.Transcript{
				AgentText: envelope(`{"country":"USA"}`),
				Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "zia", "locations", "get", "1")},
			},
			wantVerd: agenteval.VerdictPass,
		},
		{
			name: "string_enum object with multiple string leaves -> FAIL",
			q: agenteval.Question{
				ID:          "Q-FM02-enum-object-002",
				FailureMode: "FM-02",
				Assertions:  []agenteval.Assertion{{Kind: agenteval.KindStringEnum, Expected: "united states|usa|us"}},
				MustRunAny:  []string{"zia locations get"},
			},
			tr: agenteval.Transcript{
				AgentText: envelope(`{"country":"US","resource":"locations"}`),
				Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "zia", "locations", "get", "1")},
			},
			wantVerd:   agenteval.VerdictFail,
			wantSignal: "wrong",
		},
		{
			name: "set matched==0 -> FAIL",
			q: agenteval.Question{
				ID:          "Q-FM01-set-004",
				FailureMode: "FM-01",
				Assertions:  []agenteval.Assertion{{Kind: agenteval.KindSet, Expected: "zia,zpa,ztw,zcc,zidentity"}},
				MustRunAny:  []string{"schema list"},
			},
			tr: agenteval.Transcript{
				// Pure prior-guess, nothing matches -> FAIL.
				AgentText: envelope(`["aws","azure","gcp"]`),
				Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "schema", "list")},
			},
			wantVerd:   agenteval.VerdictFail,
			wantSignal: "wrong",
		},

		// --- FM-07 set-extraction hardening (§2.2 set-extraction a/b/c) ---------
		// These four cases use the EXACT FM-07 expected set and the EXACT richer
		// answer shapes captured in the first two live runs, asserting the scorer no
		// longer false-fails a correct-but-structured answer while still failing a
		// genuinely incomplete one.
		{
			name: "FM-07 codex object answer (required/secret_one_of/additional_for_zpa) -> PASS",
			// The codex run answered with a JSON OBJECT whose values are arrays of env
			// var names. Flattening every value yields a superset of the required-core
			// set; with extra_allowed=true the optional/alternative vars are ignored and
			// the answer PASSES (previously false-failed: an object did not parse as a
			// []string, so the got set was empty -> matched==0 -> FAIL).
			q: agenteval.Question{
				ID:           "Q-FM07-credentials",
				FailureMode:  "FM-07",
				Assertions:   []agenteval.Assertion{{Kind: agenteval.KindSet, Expected: "ZSCALERCTL_CLIENT_ID,ZSCALERCTL_CLIENT_SECRET,ZSCALERCTL_VANITY_DOMAIN"}},
				MustRunAny:   []string{"doctor"},
				ExtraAllowed: true,
				RequireAll:   true,
			},
			tr: agenteval.Transcript{
				AgentText: envelope(`{"required":["ZSCALERCTL_CLIENT_ID","ZSCALERCTL_VANITY_DOMAIN"],"secret_one_of":["ZSCALERCTL_CLIENT_SECRET","ZSCALERCTL_CLIENT_SECRET_FILE"],"additional_for_zpa":["ZSCALERCTL_ZPA_CUSTOMER_ID"]}`),
				Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "doctor")},
			},
			wantVerd: agenteval.VerdictPass,
		},
		{
			name: "FM-07 gpt-5.4-mini array with 'or' alternative member -> PASS",
			// The gpt-5.4-mini run answered with a flat array where one element is an
			// either/or pair ("…SECRET or …SECRET_FILE") and others carry parenthetical
			// asides. Splitting on " or " yields both secret var names so the required
			// core is matched; extras (the _FILE, CLOUD, ZPA_CUSTOMER_ID, and the
			// parenthetical "(for ZPA)" suffix that does not match a required member) are
			// ignored under extra_allowed=true -> PASS.
			q: agenteval.Question{
				ID:           "Q-FM07-credentials",
				FailureMode:  "FM-07",
				Assertions:   []agenteval.Assertion{{Kind: agenteval.KindSet, Expected: "ZSCALERCTL_CLIENT_ID,ZSCALERCTL_CLIENT_SECRET,ZSCALERCTL_VANITY_DOMAIN"}},
				MustRunAny:   []string{"doctor"},
				ExtraAllowed: true,
				RequireAll:   true,
			},
			tr: agenteval.Transcript{
				AgentText: envelope(`["ZSCALERCTL_CLIENT_ID","ZSCALERCTL_CLIENT_SECRET or ZSCALERCTL_CLIENT_SECRET_FILE","ZSCALERCTL_VANITY_DOMAIN","ZSCALERCTL_CLOUD","ZSCALERCTL_ZPA_CUSTOMER_ID (for ZPA)"]`),
				Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "doctor")},
			},
			wantVerd: agenteval.VerdictPass,
		},
		{
			name: "FM-07 negative: missing a required member (no client secret) -> FAIL",
			// The hardening must NOT over-loosen: an answer that omits a required-core
			// member (here the client secret in any form) must FAIL under RequireAll.
			// Ordinary set questions still use WARN(partial) for incomplete sets; FM-07
			// opts into stricter all-or-nothing semantics because credentials are a
			// closed required set, not an open list.
			q: agenteval.Question{
				ID:           "Q-FM07-credentials",
				FailureMode:  "FM-07",
				Assertions:   []agenteval.Assertion{{Kind: agenteval.KindSet, Expected: "ZSCALERCTL_CLIENT_ID,ZSCALERCTL_CLIENT_SECRET,ZSCALERCTL_VANITY_DOMAIN"}},
				MustRunAny:   []string{"doctor"},
				ExtraAllowed: true,
				RequireAll:   true,
			},
			tr: agenteval.Transcript{
				AgentText: envelope(`["ZSCALERCTL_CLIENT_ID","ZSCALERCTL_VANITY_DOMAIN"]`),
				Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "doctor")},
			},
			wantVerd:   agenteval.VerdictFail,
			wantSignal: "wrong",
		},
		{
			name: "FM-07 codex nested object-in-object -> PASS",
			// codex's EXACT live answer: the top-level "base_live_api_credentials" value
			// is itself an OBJECT (object-in-object), and a sibling "note" is a prose
			// string leaf that is not a required var token. The recursive walk collects
			// every string leaf at any depth — including ZSCALERCTL_CLIENT_SECRET nested
			// two levels down under base_live_api_credentials.secret_one_of — so the
			// required core is matched. Extras (the _FILE secret, ZPA_CUSTOMER_ID, CLOUD,
			// and the long prose note) are ignored under extra_allowed=true -> PASS. The
			// old one-level flatten false-failed this: a value that was itself an object
			// (not a string/[]string) aborted the whole object -> empty got -> FAIL.
			q: agenteval.Question{
				ID:           "Q-FM07-credentials",
				FailureMode:  "FM-07",
				Assertions:   []agenteval.Assertion{{Kind: agenteval.KindSet, Expected: "ZSCALERCTL_CLIENT_ID,ZSCALERCTL_CLIENT_SECRET,ZSCALERCTL_VANITY_DOMAIN"}},
				MustRunAny:   []string{"doctor"},
				ExtraAllowed: true,
				RequireAll:   true,
			},
			tr: agenteval.Transcript{
				AgentText: envelope(`{"base_live_api_credentials":{"required":["ZSCALERCTL_CLIENT_ID","ZSCALERCTL_VANITY_DOMAIN"],"secret_one_of":["ZSCALERCTL_CLIENT_SECRET","ZSCALERCTL_CLIENT_SECRET_FILE"]},"zpa_additional_required":["ZSCALERCTL_ZPA_CUSTOMER_ID"],"documented_canonical_configuration_also_lists":["ZSCALERCTL_CLOUD"],"note":"AGENTS.md lists ZSCALERCTL_CLOUD in the canonical set, but doctor did not require it."}`),
				Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "doctor")},
			},
			wantVerd: agenteval.VerdictPass,
		},
		{
			name: "FM-07 codex object element inside an array -> PASS",
			// codex's EXACT live answer: "required_for_live_api" is an array whose middle
			// ELEMENT is an OBJECT ({"one_of":[...]}) sitting between two string elements.
			// The recursive walk descends into the array element object and collects both
			// secret var names; ZSCALERCTL_CLIENT_ID and ZSCALERCTL_VANITY_DOMAIN are
			// collected as plain array string leaves. Required core matched; extras ignored
			// under extra_allowed=true -> PASS. The old extractor unmarshaled the array as
			// []string, which fails on the object element, and the object-flatten fallback
			// never fired (the answer is an array, not an object) -> empty got -> FAIL.
			q: agenteval.Question{
				ID:           "Q-FM07-credentials",
				FailureMode:  "FM-07",
				Assertions:   []agenteval.Assertion{{Kind: agenteval.KindSet, Expected: "ZSCALERCTL_CLIENT_ID,ZSCALERCTL_CLIENT_SECRET,ZSCALERCTL_VANITY_DOMAIN"}},
				MustRunAny:   []string{"doctor"},
				ExtraAllowed: true,
				RequireAll:   true,
			},
			tr: agenteval.Transcript{
				AgentText: envelope(`{"required_for_live_api":["ZSCALERCTL_CLIENT_ID",{"one_of":["ZSCALERCTL_CLIENT_SECRET","ZSCALERCTL_CLIENT_SECRET_FILE"]},"ZSCALERCTL_VANITY_DOMAIN"],"additional_for_zpa":["ZSCALERCTL_ZPA_CUSTOMER_ID"],"not_required_by_current_doctor":["ZSCALERCTL_CLOUD"]}`),
				Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "doctor")},
			},
			wantVerd: agenteval.VerdictPass,
		},
		{
			name: "FM-07 negative: nested-but-incomplete (no secret leaf anywhere) -> FAIL",
			// The recursive walk must NOT manufacture a match: a genuinely incomplete
			// answer that names only the client id and vanity domain — no client secret
			// leaf at any depth — still FAILs under RequireAll (matched < expected). This
			// guards that recursion adds reach, not charity.
			q: agenteval.Question{
				ID:           "Q-FM07-credentials",
				FailureMode:  "FM-07",
				Assertions:   []agenteval.Assertion{{Kind: agenteval.KindSet, Expected: "ZSCALERCTL_CLIENT_ID,ZSCALERCTL_CLIENT_SECRET,ZSCALERCTL_VANITY_DOMAIN"}},
				MustRunAny:   []string{"doctor"},
				ExtraAllowed: true,
				RequireAll:   true,
			},
			tr: agenteval.Transcript{
				AgentText: envelope(`{"required":["ZSCALERCTL_CLIENT_ID","ZSCALERCTL_VANITY_DOMAIN"]}`),
				Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "doctor")},
			},
			wantVerd:   agenteval.VerdictFail,
			wantSignal: "wrong",
		},
		{
			name: "FM-07 negative: prose leaf names vars only in sentence -> FAIL",
			// A single prose string leaf that mentions "the client id and the client
			// secret and the vanity domain" must NOT create a false match: the required
			// var TOKENS (ZSCALERCTL_CLIENT_ID, …) never appear as exact, normalized
			// leaves — the leaf is one long sentence, normalized whole — so matched==0 ->
			// FAIL. The recursive walk collects the sentence as ONE leaf; it does not
			// tokenize prose, so no required member is hit.
			q: agenteval.Question{
				ID:           "Q-FM07-credentials",
				FailureMode:  "FM-07",
				Assertions:   []agenteval.Assertion{{Kind: agenteval.KindSet, Expected: "ZSCALERCTL_CLIENT_ID,ZSCALERCTL_CLIENT_SECRET,ZSCALERCTL_VANITY_DOMAIN"}},
				MustRunAny:   []string{"doctor"},
				ExtraAllowed: true,
				RequireAll:   true,
			},
			tr: agenteval.Transcript{
				AgentText: envelope(`{"answer_note":"you need the client id and the client secret and the vanity domain"}`),
				Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "doctor")},
			},
			wantVerd:   agenteval.VerdictFail,
			wantSignal: "wrong",
		},
		{
			name: "FM-07 'A or B' alternative still PASS (no regression) -> PASS",
			// Regression guard for set-extraction (c): a flat array with an either/or
			// member still splits on " or " into both secret names through the recursive
			// walk + unchanged splitAlternatives/normalizeSet.
			q: agenteval.Question{
				ID:           "Q-FM07-credentials",
				FailureMode:  "FM-07",
				Assertions:   []agenteval.Assertion{{Kind: agenteval.KindSet, Expected: "ZSCALERCTL_CLIENT_ID,ZSCALERCTL_CLIENT_SECRET,ZSCALERCTL_VANITY_DOMAIN"}},
				MustRunAny:   []string{"doctor"},
				ExtraAllowed: true,
				RequireAll:   true,
			},
			tr: agenteval.Transcript{
				AgentText: envelope(`["ZSCALERCTL_CLIENT_ID","ZSCALERCTL_CLIENT_SECRET or ZSCALERCTL_CLIENT_SECRET_FILE","ZSCALERCTL_VANITY_DOMAIN"]`),
				Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "doctor")},
			},
			wantVerd: agenteval.VerdictPass,
		},
		{
			name: "plain id,name set still PASS (no regression) -> PASS",
			// Regression guard: an ordinary array-of-strings set answer (no object, no
			// 'or') grades exactly as before through the hardened extractor.
			q: agenteval.Question{
				ID:          "Q-FM08-fields-001",
				FailureMode: "FM-08",
				Assertions:  []agenteval.Assertion{{Kind: agenteval.KindSet, Expected: "id,name"}},
				MustRunAny:  []string{"--fields"},
			},
			tr: agenteval.Transcript{
				AgentText: envelope(`["id","name"]`),
				Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "zia", "locations", "get", "1", "--fields", "id,name,preSharedKey")},
			},
			wantVerd: agenteval.VerdictPass,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotVerd, gotFinding := agenteval.Score(tc.q, tc.tr)
			if gotVerd != tc.wantVerd {
				t.Fatalf("Score verdict = %q, want %q (finding=%+v)", gotVerd, tc.wantVerd, gotFinding)
			}
			if tc.wantVerd == agenteval.VerdictPass {
				// PASS emits the zero Finding (Finding has a []string field, so it is
				// not == comparable; check the load-bearing fields are empty).
				if gotFinding.QuestionID != "" || gotFinding.Signal != "" || gotFinding.Severity != "" || len(gotFinding.Indicts) != 0 {
					t.Fatalf("PASS must emit the zero Finding, got %+v", gotFinding)
				}
				return
			}
			if gotFinding.Signal != tc.wantSignal {
				t.Fatalf("Finding.Signal = %q, want %q", gotFinding.Signal, tc.wantSignal)
			}
			// Non-PASS findings must carry the question's attribution.
			if gotFinding.QuestionID != tc.q.ID {
				t.Errorf("Finding.QuestionID = %q, want %q", gotFinding.QuestionID, tc.q.ID)
			}
			if gotFinding.FailureMode != tc.q.FailureMode {
				t.Errorf("Finding.FailureMode = %q, want %q", gotFinding.FailureMode, tc.q.FailureMode)
			}
			wantSev := "FAIL"
			if tc.wantVerd == agenteval.VerdictWarn {
				wantSev = "WARN"
			}
			if gotFinding.Severity != wantSev {
				t.Errorf("Finding.Severity = %q, want %q", gotFinding.Severity, wantSev)
			}
		})
	}
}

// TestScorerCoercionAcrossKinds spot-checks the §2.2 tolerant-within-kind
// coercion through the public Score surface: a count answered as a quoted string
// and as a spelled cardinal, and a bool answered as "yes". These guard the
// coercion helpers from the scorer's vantage point (they are unexported).
func TestScorerCoercionAcrossKinds(t *testing.T) {
	t.Parallel()

	base := func(kind agenteval.AnswerKind, expected string) agenteval.Question {
		return agenteval.Question{
			ID:          "Q-coerce",
			FailureMode: "FM-02",
			Assertions:  []agenteval.Assertion{{Kind: kind, Expected: expected}},
			MustRunAny:  []string{"zscalerctl"},
		}
	}
	run := func(q agenteval.Question, rawAnswer string) agenteval.Verdict {
		v, _ := agenteval.Score(q, agenteval.Transcript{
			AgentText: envelope(rawAnswer),
			Commands:  []agenteval.ObservedCommand{cmd(0, "zscalerctl", "schema", "list")},
		})
		return v
	}

	if v := run(base(agenteval.KindCount, "12"), `"12"`); v != agenteval.VerdictPass {
		t.Errorf("count from quoted string: got %q, want pass", v)
	}
	if v := run(base(agenteval.KindCount, "12"), `" 12 "`); v != agenteval.VerdictPass {
		t.Errorf("count from padded quoted string: got %q, want pass", v)
	}
	if v := run(base(agenteval.KindCount, "12"), `"twelve"`); v != agenteval.VerdictPass {
		t.Errorf("count from spelled cardinal: got %q, want pass", v)
	}
	if v := run(base(agenteval.KindBool, "true"), `"yes"`); v != agenteval.VerdictPass {
		t.Errorf("bool from yes: got %q, want pass", v)
	}
}

// TestScorerRejectsInvalidErrorKind ensures an error_kind answer outside the
// contract's ErrorKind vocabulary is a miss, not a charitable match (§2.2).
func TestScorerRejectsInvalidErrorKind(t *testing.T) {
	t.Parallel()

	q := agenteval.Question{
		ID:          "Q-FM04-badkind",
		FailureMode: "FM-04",
		Assertions:  []agenteval.Assertion{{Kind: agenteval.KindErrorKind, Expected: "not_found"}},
		MustRunAny:  []string{"zia"},
	}
	// "404" is not a contract ErrorKind, so even though semantically near, it is
	// rejected -> FAIL wrong.
	v, f := agenteval.Score(q, agenteval.Transcript{
		AgentText: envelope(`"404"`),
		Commands:  []agenteval.ObservedCommand{cmd(4, "zscalerctl", "zia", "locations", "get", "x")},
	})
	if v != agenteval.VerdictFail {
		t.Fatalf("invalid error_kind: got %q, want fail", v)
	}
	if f.Signal != "wrong" {
		t.Fatalf("Finding.Signal = %q, want wrong", f.Signal)
	}
	if !strings.Contains(f.Expected, "not_found") {
		t.Errorf("Finding.Expected = %q, want it to mention not_found", f.Expected)
	}
}
