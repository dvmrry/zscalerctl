package agenteval_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/dvmrry/zscalerctl/internal/agenteval"
)

// report_test.go exercises the PURE aggregation + rendering (report.go): the
// §2.4 Clears rule, the §1.2 Floor metric over a synthetic roster, and Render's
// floor-led, tracked-not-gated output. Everything is deterministic; the report
// date is a fixed parameter, never time.Now.

// qr builds a QuestionResult with a given tier, verdict, and finding signal.
func qr(id, tier string, v agenteval.Verdict, signal string) agenteval.QuestionResult {
	f := agenteval.Finding{}
	if v != agenteval.VerdictPass {
		f = agenteval.Finding{QuestionID: id, FailureMode: "FM-XX", Severity: severityFor(v), Signal: signal, Indicts: []string{"AGENTS.md#x"}}
	}
	return agenteval.QuestionResult{
		Question: agenteval.Question{ID: id, Tier: tier},
		Verdict:  v,
		Finding:  f,
	}
}

func severityFor(v agenteval.Verdict) string {
	if v == agenteval.VerdictFail {
		return "FAIL"
	}
	return "WARN"
}

// TestClears covers the §2.4 rule branches.
func TestClears(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  agenteval.AgentRun
		want bool
	}{
		{
			name: "all pass clears",
			run: agenteval.AgentRun{Agent: "a", Rank: 1, Results: []agenteval.QuestionResult{
				qr("t0", "T0", agenteval.VerdictPass, ""),
				qr("t1", "T1", agenteval.VerdictPass, ""),
			}},
			want: true,
		},
		{
			name: "non-T0 WARN still clears",
			run: agenteval.AgentRun{Agent: "a", Rank: 1, Results: []agenteval.QuestionResult{
				qr("t0", "T0", agenteval.VerdictPass, ""),
				qr("t1", "T1", agenteval.VerdictWarn, "partial"),
			}},
			want: true,
		},
		{
			name: "T0 fail does not clear",
			run: agenteval.AgentRun{Agent: "a", Rank: 1, Results: []agenteval.QuestionResult{
				qr("t0", "T0", agenteval.VerdictFail, "wrong"),
				qr("t1", "T1", agenteval.VerdictPass, ""),
			}},
			want: false,
		},
		{
			name: "T0 WARN does not clear (hard gate)",
			run: agenteval.AgentRun{Agent: "a", Rank: 1, Results: []agenteval.QuestionResult{
				qr("t0", "T0", agenteval.VerdictWarn, "partial"),
				qr("t1", "T1", agenteval.VerdictPass, ""),
			}},
			want: false,
		},
		{
			name: "method_violation does not clear",
			run: agenteval.AgentRun{Agent: "a", Rank: 1, Results: []agenteval.QuestionResult{
				qr("t0", "T0", agenteval.VerdictPass, ""),
				qr("t1", "T1", agenteval.VerdictFail, "method_violation"),
			}},
			want: false,
		},
		{
			name: "no_commands does not clear",
			run: agenteval.AgentRun{Agent: "a", Rank: 1, Results: []agenteval.QuestionResult{
				qr("t1", "T1", agenteval.VerdictFail, "no_commands"),
			}},
			want: false,
		},
		{
			name: "bad_envelope does not clear",
			run: agenteval.AgentRun{Agent: "a", Rank: 1, Results: []agenteval.QuestionResult{
				qr("t1", "T1", agenteval.VerdictFail, "bad_envelope"),
			}},
			want: false,
		},
		{
			name: "empty run does not clear",
			run:  agenteval.AgentRun{Agent: "a", Rank: 1},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := agenteval.Clears(tt.run); got != tt.want {
				t.Fatalf("Clears(%s) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// syntheticRoster builds a 3-agent roster with mixed verdicts:
//   - rank 1 (weak): FAILS a T0 question -> does not clear; its first violation is
//     the actionable gap.
//   - rank 2 (mid):  all pass-or-WARN, no T0 fail -> clears (the floor).
//   - rank 3 (high): all pass -> clears.
//
// So the floor is "mid" and the first violation is rank-1's T0 fail.
func syntheticRoster() []agenteval.AgentRun {
	return []agenteval.AgentRun{
		{Agent: "high", Rank: 3, Results: []agenteval.QuestionResult{
			qr("Q-t0", "T0", agenteval.VerdictPass, ""),
			qr("Q-t1", "T1", agenteval.VerdictPass, ""),
		}},
		{Agent: "weak", Rank: 1, Results: []agenteval.QuestionResult{
			qr("Q-t0", "T0", agenteval.VerdictFail, "wrong"),
			qr("Q-t1", "T1", agenteval.VerdictWarn, "partial"),
		}},
		{Agent: "mid", Rank: 2, Results: []agenteval.QuestionResult{
			qr("Q-t0", "T0", agenteval.VerdictPass, ""),
			qr("Q-t1", "T1", agenteval.VerdictWarn, "partial"),
		}},
	}
}

// TestFloor asserts the expected floor agent + first-violation over the synthetic
// roster (weakest-clears + actionable gap).
func TestFloor(t *testing.T) {
	t.Parallel()

	floor, violation := agenteval.Floor(syntheticRoster())
	if floor != "mid" {
		t.Fatalf("floor = %q, want \"mid\" (lowest-rank agent that clears)", floor)
	}
	if violation == nil {
		t.Fatalf("first violation = nil, want rank-1's T0 fail")
	}
	if violation.Agent != "weak" {
		t.Fatalf("first violation agent = %q, want \"weak\" (the weakest non-clearing agent)", violation.Agent)
	}
	if violation.QuestionID != "Q-t0" || violation.Signal != "wrong" {
		t.Fatalf("first violation = %+v, want the Q-t0 wrong FAIL", *violation)
	}
}

// TestFloorNobodyClears asserts the "nobody clears" edge: floor is "" and the
// first violation is the weakest agent's.
func TestFloorNobodyClears(t *testing.T) {
	t.Parallel()

	runs := []agenteval.AgentRun{
		{Agent: "weak", Rank: 1, Results: []agenteval.QuestionResult{qr("Q-t0", "T0", agenteval.VerdictFail, "wrong")}},
		{Agent: "strong", Rank: 2, Results: []agenteval.QuestionResult{qr("Q-t1", "T1", agenteval.VerdictFail, "method_violation")}},
	}
	floor, violation := agenteval.Floor(runs)
	if floor != "" {
		t.Fatalf("floor = %q, want \"\" (nobody clears)", floor)
	}
	if violation == nil || violation.Agent != "weak" {
		t.Fatalf("first violation = %+v, want weakest agent \"weak\"", violation)
	}
}

// TestFloorEverybodyClears asserts the "everybody clears" edge: floor is the
// weakest agent and there is no first violation.
func TestFloorEverybodyClears(t *testing.T) {
	t.Parallel()

	runs := []agenteval.AgentRun{
		{Agent: "strong", Rank: 2, Results: []agenteval.QuestionResult{qr("Q-t0", "T0", agenteval.VerdictPass, "")}},
		{Agent: "weak", Rank: 1, Results: []agenteval.QuestionResult{
			qr("Q-t0", "T0", agenteval.VerdictPass, ""),
			qr("Q-t1", "T1", agenteval.VerdictWarn, "partial"),
		}},
	}
	floor, violation := agenteval.Floor(runs)
	if floor != "weak" {
		t.Fatalf("floor = %q, want \"weak\" (weakest of all-clearing roster)", floor)
	}
	if violation != nil {
		t.Fatalf("first violation = %+v, want nil (everybody clears)", *violation)
	}
}

// TestRender asserts the Markdown leads with the FLOOR line, names a Finding, and
// carries the tracked-not-gated header; and that the JSON mirror is well-formed
// with the same floor + gated=false.
func TestRender(t *testing.T) {
	t.Parallel()

	const date = "2026-06-14"
	md, jsonBytes := agenteval.Render(syntheticRoster(), date)

	// Tracked, not gated header.
	if !strings.Contains(md, "TRACKED, NOT GATED") {
		t.Fatalf("Markdown missing the tracked-not-gated header:\n%s", md)
	}
	// Floor line leads.
	if !strings.Contains(md, "The floor is **mid**") {
		t.Fatalf("Markdown missing the floor line:\n%s", md)
	}
	// Floor section precedes the per-agent table (lead with the floor, not the ceiling).
	floorIdx := strings.Index(md, "## Floor")
	tableIdx := strings.Index(md, "## Per-agent results")
	if floorIdx < 0 || tableIdx < 0 || floorIdx > tableIdx {
		t.Fatalf("expected the Floor section before the per-agent table; floorIdx=%d tableIdx=%d", floorIdx, tableIdx)
	}
	// A Finding is enumerated.
	if !strings.Contains(md, "## Open findings") || !strings.Contains(md, "Q-t0") {
		t.Fatalf("Markdown missing the open-findings enumeration:\n%s", md)
	}
	// The date parameter appears.
	if !strings.Contains(md, date) {
		t.Fatalf("Markdown missing the run date %q", date)
	}

	// JSON mirror.
	var rep struct {
		Date           string `json:"date"`
		Tracked        bool   `json:"tracked"`
		Gated          bool   `json:"gated"`
		FloorAgent     string `json:"floor_agent"`
		FirstViolation *struct {
			Agent      string `json:"agent"`
			QuestionID string `json:"question_id"`
		} `json:"first_violation"`
		Agents []struct {
			Agent  string `json:"agent"`
			Clears bool   `json:"clears"`
		} `json:"agents"`
		Findings []struct {
			QuestionID string `json:"question_id"`
		} `json:"findings"`
	}
	if err := json.Unmarshal(jsonBytes, &rep); err != nil {
		t.Fatalf("report JSON did not parse: %v\n%s", err, jsonBytes)
	}
	if rep.Date != date {
		t.Fatalf("JSON date = %q, want %q", rep.Date, date)
	}
	if !rep.Tracked || rep.Gated {
		t.Fatalf("JSON tracked/gated = %v/%v, want true/false", rep.Tracked, rep.Gated)
	}
	if rep.FloorAgent != "mid" {
		t.Fatalf("JSON floor_agent = %q, want \"mid\"", rep.FloorAgent)
	}
	if rep.FirstViolation == nil || rep.FirstViolation.Agent != "weak" {
		t.Fatalf("JSON first_violation = %+v, want weak's", rep.FirstViolation)
	}
	if len(rep.Findings) == 0 {
		t.Fatalf("JSON findings empty; the report must enumerate open findings")
	}
	// Agents are weakest-first.
	if len(rep.Agents) != 3 || rep.Agents[0].Agent != "weak" {
		t.Fatalf("JSON agents not weakest-first: %+v", rep.Agents)
	}
}
