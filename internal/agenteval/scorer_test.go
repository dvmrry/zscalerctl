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
