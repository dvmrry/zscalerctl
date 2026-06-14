package agenteval

// aggregate.go is the PURE multi-sampling aggregator (IMPROVEMENT #1: de-noise
// the floor). The live half is non-deterministic — the single-run floor was
// observed to flip across three runs — so the runner can sample each
// (backend, question) N times (--samples N) and grade each sample. This file
// folds those N per-sample QuestionResults into ONE aggregated QuestionResult
// the way Clears/Floor/Render expect.
//
// It is deterministic and total: AggregateVerdicts is a function of its input
// slice alone. It never execs, dials the network, reads the clock, calls rand,
// reads the environment, or calls an LLM — same posture as report.go/scorer.go,
// so the aggregation is CI-unit-testable even though the sampling that feeds it
// is not.
//
// The rule (so that "clears" means RELIABLE, not lucky):
//
//   - the aggregated verdict is the MODE (majority) of the N sample verdicts;
//   - on a TIE between modes, the WORSE verdict wins (Fail > Warn > Pass) — a
//     question that is flaky between pass and fail should not be credited as a
//     clean pass;
//   - the representative Finding is taken from a sample whose verdict equals the
//     aggregated verdict (so the report's named violation is a real observed
//     sample, not a synthesised one);
//   - Samples = N and Passes = count of VerdictPass samples record the
//     per-question consistency ("passes M/N") the report surfaces.

// verdictSeverityRank orders verdicts worst-first for the tie-break: a higher
// rank is WORSE. Fail (2) > Warn (1) > Pass (0). On a tie between equally-common
// verdicts the highest-rank (worst) one is chosen, because a reliable "clears"
// must not be awarded to a question that is only sometimes correct.
func verdictSeverityRank(v Verdict) int {
	switch v {
	case VerdictFail:
		return 2
	case VerdictWarn:
		return 1
	case VerdictPass:
		return 0
	default:
		// An unknown verdict is treated as the worst so it can never silently win a
		// tie by ranking below a real verdict.
		return 3
	}
}

// AggregateVerdicts folds N per-sample QuestionResults for the SAME question
// into one aggregated QuestionResult (IMPROVEMENT #1). The samples must all be
// for one (agent, question); the aggregate carries the first sample's Question
// (they are identical by construction).
//
// The aggregated Verdict is the mode of the sample verdicts, worse-on-tie
// (Fail > Warn > Pass). The aggregated Finding is a representative sample's
// Finding whose verdict equals the aggregate (the zero Finding when the
// aggregate is a PASS — a clean PASS sample carries no Finding, §2.5). Samples
// is len(samples) and Passes is the count of VerdictPass samples, so the report
// can show "passes M/N" and surface non-unanimous (flaky) questions.
//
// Edge cases (total function):
//   - empty input -> the zero QuestionResult (nothing to aggregate);
//   - a single sample -> a passthrough: same Question/Verdict/Finding, with
//     Samples=1 and Passes set to 0/1 so the report knows it was a 1-sample run.
func AggregateVerdicts(samples []QuestionResult) QuestionResult {
	if len(samples) == 0 {
		return QuestionResult{}
	}

	// Tally verdict frequencies and count passes for the consistency metric.
	counts := map[Verdict]int{}
	passes := 0
	for _, s := range samples {
		counts[s.Verdict]++
		if s.Verdict == VerdictPass {
			passes++
		}
	}

	agg := modeVerdict(counts)

	out := QuestionResult{
		Question: samples[0].Question,
		Verdict:  agg,
		Finding:  representativeFinding(samples, agg),
		Samples:  len(samples),
		Passes:   passes,
	}
	return out
}

// modeVerdict returns the most frequent verdict in counts, breaking a tie toward
// the WORSE verdict (higher verdictSeverityRank). counts is non-empty by the
// caller's contract (AggregateVerdicts guards the empty case). Iteration order
// over the map is irrelevant: the (count desc, severity desc) comparison is a
// total order over the distinct verdicts present, so the result is deterministic.
func modeVerdict(counts map[Verdict]int) Verdict {
	var best Verdict
	bestCount := -1
	for v, c := range counts {
		switch {
		case c > bestCount:
			best, bestCount = v, c
		case c == bestCount && verdictSeverityRank(v) > verdictSeverityRank(best):
			// Tie on frequency: the worse verdict wins (reliable "clears").
			best = v
		}
	}
	return best
}

// representativeFinding returns the Finding of the first sample whose verdict
// equals the aggregated verdict — so the report's named violation is a real
// observed sample, not a synthesised one. When the aggregate is a PASS the
// matching sample carries the zero Finding (a clean PASS emits no Finding,
// §2.5), which is exactly what a PASS QuestionResult should hold. If — defensively
// — no sample matches (cannot happen for a mode drawn from the same samples), the
// zero Finding is returned.
func representativeFinding(samples []QuestionResult, agg Verdict) Finding {
	for _, s := range samples {
		if s.Verdict == agg {
			return s.Finding
		}
	}
	return Finding{}
}
