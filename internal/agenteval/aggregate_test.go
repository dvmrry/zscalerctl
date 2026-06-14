package agenteval_test

import (
	"testing"

	"github.com/dvmrry/zscalerctl/internal/agenteval"
)

// aggregate_test.go exercises the PURE multi-sampling aggregator
// (aggregate.go, IMPROVEMENT #1). AggregateVerdicts folds N per-sample
// QuestionResults into one: the mode verdict (worse-on-tie), a representative
// Finding matching that verdict, and the M/N pass-count consistency. Everything
// here is deterministic.

// sampleResult builds one per-sample QuestionResult for question id with verdict
// v. A non-PASS sample carries a Finding stamped with the verdict's signal so the
// representative-Finding assertions can tell samples apart.
func sampleResult(id string, v agenteval.Verdict, signal string) agenteval.QuestionResult {
	f := agenteval.Finding{}
	if v != agenteval.VerdictPass {
		f = agenteval.Finding{QuestionID: id, Signal: signal, Severity: severityFor(v)}
	}
	return agenteval.QuestionResult{
		Question: agenteval.Question{ID: id, Tier: "T1"},
		Verdict:  v,
		Finding:  f,
	}
}

// TestAggregateVerdicts covers the §aggregate.go rule: mode, worse-on-tie, the
// M/N consistency, and the edge cases (empty, single sample).
func TestAggregateVerdicts(t *testing.T) {
	t.Parallel()

	const id = "Q-X"
	P := func() agenteval.QuestionResult { return sampleResult(id, agenteval.VerdictPass, "") }
	W := func() agenteval.QuestionResult { return sampleResult(id, agenteval.VerdictWarn, "partial") }
	F := func() agenteval.QuestionResult { return sampleResult(id, agenteval.VerdictFail, "wrong") }

	tests := []struct {
		name        string
		samples     []agenteval.QuestionResult
		wantVerdict agenteval.Verdict
		wantSamples int
		wantPasses  int
	}{
		{
			// Majority pass: [pass,pass,fail] -> pass (2/3).
			name:        "majority pass 2 of 3",
			samples:     []agenteval.QuestionResult{P(), P(), F()},
			wantVerdict: agenteval.VerdictPass,
			wantSamples: 3,
			wantPasses:  2,
		},
		{
			// Majority fail: [fail,fail,pass] -> fail.
			name:        "majority fail 2 of 3",
			samples:     []agenteval.QuestionResult{F(), F(), P()},
			wantVerdict: agenteval.VerdictFail,
			wantSamples: 3,
			wantPasses:  1,
		},
		{
			// Tie pass/fail -> WORSE wins: [pass,fail] -> fail.
			name:        "tie pass-fail resolves to fail (worse)",
			samples:     []agenteval.QuestionResult{P(), F()},
			wantVerdict: agenteval.VerdictFail,
			wantSamples: 2,
			wantPasses:  1,
		},
		{
			// Majority warn: [warn,warn,pass] -> warn.
			name:        "majority warn 2 of 3",
			samples:     []agenteval.QuestionResult{W(), W(), P()},
			wantVerdict: agenteval.VerdictWarn,
			wantSamples: 3,
			wantPasses:  1,
		},
		{
			// Tie pass/warn -> WORSE wins (warn > pass): [pass,warn] -> warn.
			name:        "tie pass-warn resolves to warn (worse)",
			samples:     []agenteval.QuestionResult{P(), W()},
			wantVerdict: agenteval.VerdictWarn,
			wantSamples: 2,
			wantPasses:  1,
		},
		{
			// Tie warn/fail -> WORSE wins (fail > warn): [warn,fail] -> fail.
			name:        "tie warn-fail resolves to fail (worse)",
			samples:     []agenteval.QuestionResult{W(), F()},
			wantVerdict: agenteval.VerdictFail,
			wantSamples: 2,
			wantPasses:  0,
		},
		{
			// Three-way tie pass/warn/fail -> WORST wins: fail.
			name:        "three-way tie resolves to fail (worst)",
			samples:     []agenteval.QuestionResult{P(), W(), F()},
			wantVerdict: agenteval.VerdictFail,
			wantSamples: 3,
			wantPasses:  1,
		},
		{
			// Single sample -> passthrough (pass).
			name:        "single sample pass passthrough",
			samples:     []agenteval.QuestionResult{P()},
			wantVerdict: agenteval.VerdictPass,
			wantSamples: 1,
			wantPasses:  1,
		},
		{
			// Single sample -> passthrough (fail).
			name:        "single sample fail passthrough",
			samples:     []agenteval.QuestionResult{F()},
			wantVerdict: agenteval.VerdictFail,
			wantSamples: 1,
			wantPasses:  0,
		},
		{
			// Unanimous pass -> pass (3/3).
			name:        "unanimous pass",
			samples:     []agenteval.QuestionResult{P(), P(), P()},
			wantVerdict: agenteval.VerdictPass,
			wantSamples: 3,
			wantPasses:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := agenteval.AggregateVerdicts(tt.samples)

			if got.Verdict != tt.wantVerdict {
				t.Errorf("Verdict = %q, want %q", got.Verdict, tt.wantVerdict)
			}
			if got.Samples != tt.wantSamples {
				t.Errorf("Samples = %d, want %d", got.Samples, tt.wantSamples)
			}
			if got.Passes != tt.wantPasses {
				t.Errorf("Passes = %d, want %d", got.Passes, tt.wantPasses)
			}
			// The aggregate carries the question identity.
			if got.Question.ID != id {
				t.Errorf("Question.ID = %q, want %q", got.Question.ID, id)
			}
			// The representative Finding matches the aggregated verdict: a PASS
			// aggregate carries the zero Finding; a non-PASS aggregate carries a
			// populated Finding with the matching severity.
			if tt.wantVerdict == agenteval.VerdictPass {
				if got.Finding.Severity != "" || got.Finding.Signal != "" || got.Finding.QuestionID != "" {
					t.Errorf("PASS aggregate should carry the zero Finding, got %+v", got.Finding)
				}
			} else {
				if got.Finding.Severity != severityFor(tt.wantVerdict) {
					t.Errorf("Finding.Severity = %q, want %q (representative must match the aggregate)", got.Finding.Severity, severityFor(tt.wantVerdict))
				}
			}
		})
	}
}

// TestAggregateVerdictsEmpty asserts the empty-input edge: the zero
// QuestionResult, so the aggregator is total.
func TestAggregateVerdictsEmpty(t *testing.T) {
	t.Parallel()
	got := agenteval.AggregateVerdicts(nil)
	if got.Verdict != "" || got.Samples != 0 || got.Passes != 0 {
		t.Fatalf("empty aggregate = %+v, want the zero QuestionResult", got)
	}
}

// TestAggregateVerdictsRepresentativeFindingIsReal asserts the representative
// Finding is taken from a sample that actually produced the aggregated verdict —
// not synthesised. With [fail("a"), fail("b"), pass] the aggregate is fail and
// the Finding is the FIRST matching fail sample's.
func TestAggregateVerdictsRepresentativeFindingIsReal(t *testing.T) {
	t.Parallel()

	const id = "Q-Y"
	samples := []agenteval.QuestionResult{
		sampleResult(id, agenteval.VerdictFail, "first_fail"),
		sampleResult(id, agenteval.VerdictFail, "second_fail"),
		sampleResult(id, agenteval.VerdictPass, ""),
	}
	got := agenteval.AggregateVerdicts(samples)
	if got.Verdict != agenteval.VerdictFail {
		t.Fatalf("Verdict = %q, want fail", got.Verdict)
	}
	if got.Finding.Signal != "first_fail" {
		t.Fatalf("representative Finding signal = %q, want \"first_fail\" (first matching sample)", got.Finding.Signal)
	}
}
