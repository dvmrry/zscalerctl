package agenteval

// report.go is the PURE aggregation + rendering half of the runner core
// (docs/AGENTIC_COVERAGE_PLAN.md §1.2, §2.4, §7). It turns per-agent scored
// results into the floor metric and the tracked (NOT gated) report.
//
// It is deterministic: the report date is a PARAMETER (never time.Now), and
// nothing here execs, dials the network, reads the clock, calls rand, reads the
// environment, or calls an LLM. The aggregation is a total function of its
// []AgentRun input; Render is a total function of (runs, date).
//
// The headline is THE FLOOR (§1.2): the weakest agent that still clears the
// battery. A strong agent passing proves little; a weak agent passing proves the
// surface teaches itself. So the report leads with "the worst agent that clears
// it" and names the first violation of the weakest agent that did NOT clear —
// because that violation is the actionable surface gap.

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
)

// QuestionResult is one graded question for one agent: the Question that was
// asked, the scorer's Verdict, and the Finding (the zero Finding on a clean
// PASS, a populated Finding on any WARN/FAIL — §2.5). It is the unit Clears and
// the report aggregate over.
type QuestionResult struct {
	// Question is the instantiated question that was graded.
	Question Question
	// Verdict is the scorer's per-question outcome (§4.2).
	Verdict Verdict
	// Finding is the scorer's Finding: zero value on a clean PASS, populated on
	// any WARN or FAIL (§2.5).
	Finding Finding
}

// AgentRun is one roster agent's full battery result: its name, its §1.2 floor
// rank (lower = weaker), and the per-question results in battery order.
type AgentRun struct {
	// Agent is the roster agent name (e.g. "haiku").
	Agent string
	// Rank is the §1.2 floor ordering; lower is a weaker agent.
	Rank int
	// Results are the per-question graded results, one per battery question.
	Results []QuestionResult
}

// disqualifyingSignals are the Finding.Signal strings that, on a FAIL, hard-
// disqualify an agent from clearing regardless of any other result (§2.4): a
// no_commands FAIL, a method_violation FAIL, or a bad_envelope (protocol)
// failure. They are the "ZERO method violations AND ZERO no_commands" half of
// the clears rule, detected mechanically off the Finding the scorer stamped.
var disqualifyingSignals = map[string]bool{
	signalNoCommands:      true, // "no_commands"
	signalMethodViolation: true, // "method_violation"
	signalBadEnvelope:     true, // "bad_envelope"
}

// Clears implements the §2.4 definition of "clears the battery" EXACTLY, off the
// Verdict + Finding of each QuestionResult. An agent clears iff ALL of:
//
//  1. every Tier-0 question (Question.Tier == "T0") is VerdictPass — the hard
//     gate; the founding "a weak agent could not discover object names" concern
//     is never averaged away, and a WARN on a T0 question is NOT enough;
//  2. every OTHER question (Tier != "T0") is pass-or-WARN (i.e. not a FAIL);
//  3. ZERO method violations and ZERO no_commands and ZERO bad-envelope failures
//     anywhere — detected via the Finding's Signal on any FAIL
//     (disqualifyingSignals). A FAIL carrying one of those signals disqualifies
//     even if it somehow sat on a non-T0 question.
//
// The Verdict→clears mapping used:
//
//	T0 question:      PASS clears; WARN or FAIL does NOT.
//	non-T0 question:  PASS or WARN clears; FAIL does NOT.
//	any question:     a FAIL whose Finding.Signal is no_commands/method_violation/
//	                  bad_envelope disqualifies outright (these are the §2.4
//	                  zero-tolerance signals).
//
// A run with no results does not clear (there is nothing to prove the surface
// carried it).
func Clears(run AgentRun) bool {
	if len(run.Results) == 0 {
		return false
	}
	for _, r := range run.Results {
		// Zero-tolerance signals disqualify regardless of tier (§2.4).
		if r.Verdict == VerdictFail && disqualifyingSignals[r.Finding.Signal] {
			return false
		}
		if isTierZero(r.Question.Tier) {
			// T0 hard gate: must be a clean PASS.
			if r.Verdict != VerdictPass {
				return false
			}
			continue
		}
		// Non-T0: pass-or-WARN; a FAIL disqualifies.
		if r.Verdict == VerdictFail {
			return false
		}
	}
	return true
}

// isTierZero reports whether a question's tier is the Tier-0 floor gate. Tier
// strings are compared case-insensitively after trimming so a "t0"/" T0 " in a
// hand-authored fixture is not silently treated as non-floor.
func isTierZero(tier string) bool {
	return strings.EqualFold(strings.TrimSpace(tier), "T0")
}

// Floor implements the §1.2 floor metric. It returns:
//
//   - floorAgent: the name of the LOWEST-rank agent that Clears (the "worst agent
//     that clears it"). Empty string if NOBODY clears.
//   - firstViolation: the first violation (the lowest-severity-ordered Finding)
//     of the WEAKEST agent that did NOT clear — the actionable surface gap to
//     name in the report. Nil when there is no such agent (i.e. EVERYBODY clears,
//     or there are no runs).
//
// "Weakest agent that did not clear" is the lowest-rank non-clearing agent: per
// §7 triage, a small/weak agent failing is the strongest evidence the surface is
// not self-describing, so its first violation is the one worth surfacing.
//
// Both halves are computed independently, so the common case (some clear, some
// don't) returns both a floor AND the gap just below it; "nobody clears" returns
// ("", firstViolation-of-the-weakest); "everybody clears" returns
// (weakest-agent, nil).
func Floor(runs []AgentRun) (floorAgent string, firstViolation *Finding) {
	ordered := byRankAscending(runs)

	// Floor: the lowest-rank agent that clears.
	for _, run := range ordered {
		if Clears(run) {
			floorAgent = run.Agent
			break
		}
	}

	// First violation: the first violation of the weakest (lowest-rank) agent that
	// did NOT clear.
	for _, run := range ordered {
		if Clears(run) {
			continue
		}
		if f := firstViolationOf(run); f != nil {
			firstViolation = f
		}
		break
	}

	return floorAgent, firstViolation
}

// firstViolationOf returns the first violation Finding of a non-clearing agent,
// in severity order (§4.2): FAILs before WARNs, and within a severity in battery
// order. A non-clearing agent must have at least one FAIL, so a FAIL is
// preferred; the WARN fallback exists only for completeness. Returns nil if the
// run somehow has no WARN/FAIL Finding (shouldn't happen for a non-clearing run,
// but the function stays total).
func firstViolationOf(run AgentRun) *Finding {
	var firstFail, firstWarn *Finding
	for i := range run.Results {
		r := run.Results[i]
		switch r.Verdict {
		case VerdictFail:
			if firstFail == nil {
				f := run.Results[i].Finding
				stampAgent(&f, run.Agent)
				firstFail = &f
			}
		case VerdictWarn:
			if firstWarn == nil {
				f := run.Results[i].Finding
				stampAgent(&f, run.Agent)
				firstWarn = &f
			}
		}
	}
	if firstFail != nil {
		return firstFail
	}
	return firstWarn
}

// stampAgent fills the Finding.Agent (the pure scorer leaves it blank, §2.5) so
// the report's named violation says which agent hit it.
func stampAgent(f *Finding, agent string) {
	if f.Agent == "" {
		f.Agent = agent
	}
}

// byRankAscending returns runs sorted by Rank ascending (weakest first), ties
// broken by agent name so the ordering is deterministic regardless of input
// order.
func byRankAscending(runs []AgentRun) []AgentRun {
	out := make([]AgentRun, len(runs))
	copy(out, runs)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Rank != out[j].Rank {
			return out[i].Rank < out[j].Rank
		}
		return out[i].Agent < out[j].Agent
	})
	return out
}

// openFindings collects every non-PASS Finding across all runs, agent-stamped,
// in (agent-rank, battery) order — the report's required Findings enumeration
// (§2.5: no score without findings). FAILs are listed before WARNs within an
// agent so the most actionable gaps lead.
func openFindings(runs []AgentRun) []Finding {
	var out []Finding
	for _, run := range byRankAscending(runs) {
		var fails, warns []Finding
		for i := range run.Results {
			r := run.Results[i]
			if r.Verdict == VerdictPass {
				continue
			}
			f := r.Finding
			stampAgent(&f, run.Agent)
			if r.Verdict == VerdictFail {
				fails = append(fails, f)
			} else {
				warns = append(warns, f)
			}
		}
		out = append(out, fails...)
		out = append(out, warns...)
	}
	return out
}

// --- report rendering (§7) --------------------------------------------------

// reportJSON is the machine-readable docs/agentic-coverage.json shape. It leads
// with the floor and the tracked-not-gated posture, then the per-agent summary,
// then the open findings — the JSON mirror of the Markdown report.
type reportJSON struct {
	// Schema versions the report format.
	Schema string `json:"schema"`
	// Date is the run date, a parameter (never time.Now), so the artifact is
	// reproducible.
	Date string `json:"date"`
	// Tracked is always true and Gated always false: this artifact records the
	// non-deterministic live score and is NEVER a build gate (§1.3, §7).
	Tracked bool `json:"tracked"`
	Gated   bool `json:"gated"`
	// FloorAgent is the worst agent that clears, or "" if nobody clears.
	FloorAgent string `json:"floor_agent"`
	// FirstViolation is the named actionable gap (the weakest non-clearing agent's
	// first violation), or null when everybody clears.
	FirstViolation *Finding `json:"first_violation"`
	// Agents is the per-agent summary in weakest-first order.
	Agents []agentSummaryJSON `json:"agents"`
	// Findings is every open WARN/FAIL Finding (§2.5).
	Findings []Finding `json:"findings"`
}

// agentSummaryJSON is one row of the per-agent table.
type agentSummaryJSON struct {
	Agent  string `json:"agent"`
	Rank   int    `json:"rank"`
	Clears bool   `json:"clears"`
	Pass   int    `json:"pass"`
	Warn   int    `json:"warn"`
	Fail   int    `json:"fail"`
}

const reportSchema = "zscalerctl/agentic-coverage/v1"

// Render produces the two committed report artifacts docs/agentic-coverage.md
// and docs/agentic-coverage.json (§7), mirroring docs/FIELD_COVERAGE.md tone. It
// is pure: date is a parameter, never time.Now.
//
// The Markdown leads with the FLOOR (not the ceiling), then the per-agent table,
// then the open Findings. The header explicitly states this artifact is TRACKED,
// NOT GATED — the deterministic battery+scorer drift is gated; the live score is
// only tracked (§1.3, §7). jsonBytes is the machine mirror.
func Render(runs []AgentRun, date string) (md string, jsonBytes []byte) {
	ordered := byRankAscending(runs)
	floorAgent, firstViolation := Floor(runs)
	findings := openFindings(runs)

	md = renderMarkdown(ordered, date, floorAgent, firstViolation, findings)
	jsonBytes = renderReportJSON(ordered, date, floorAgent, firstViolation, findings)
	return md, jsonBytes
}

// renderMarkdown builds the Markdown report body (§7).
func renderMarkdown(ordered []AgentRun, date, floorAgent string, firstViolation *Finding, findings []Finding) string {
	var b strings.Builder

	b.WriteString("# Agentic Coverage\n\n")
	b.WriteString("> **TRACKED, NOT GATED.** This report records the non-deterministic live\n")
	b.WriteString("> multi-agent run. The battery + scorer drift gates are CI-gated under\n")
	b.WriteString("> `go test`; the live score below is tracked, never a build pass/fail\n")
	b.WriteString("> (docs/AGENTIC_COVERAGE_PLAN.md §1.3, §7).\n\n")
	b.WriteString("> Treat the floor as a self-describability signal, not a precise model\n")
	b.WriteString("> ranking. A `none` floor can come from strict method-proof requirements or\n")
	b.WriteString("> free-form answer shape, so read the findings before drawing product\n")
	b.WriteString("> conclusions.\n\n")
	b.WriteString("Run date: " + date + "\n\n")

	// Lead with the FLOOR (§1.2), not the ceiling.
	b.WriteString("## Floor\n\n")
	if floorAgent != "" {
		b.WriteString("The floor is **" + floorAgent + "** — the weakest agent that clears the battery.\n\n")
	} else {
		b.WriteString("**No agent clears the battery.** The floor is undefined: even the strongest\n")
		b.WriteString("agent in this run hit a disqualifying result.\n\n")
	}
	if firstViolation != nil {
		b.WriteString("First violation of the weakest agent that did not clear (the actionable gap):\n\n")
		b.WriteString(renderFindingLine(*firstViolation) + "\n\n")
	} else if floorAgent != "" {
		b.WriteString("Every agent in this run clears the battery; there is no open floor gap.\n\n")
	}

	// Per-agent table (§7).
	b.WriteString("## Per-agent results\n\n")
	b.WriteString("| rank | agent | clears | pass | warn | fail |\n")
	b.WriteString("|------|-------|--------|------|------|------|\n")
	for _, run := range ordered {
		s := summarize(run)
		b.WriteString("| " + strconv.Itoa(run.Rank) + " | " + run.Agent + " | " + yesNo(s.Clears) +
			" | " + strconv.Itoa(s.Pass) + " | " + strconv.Itoa(s.Warn) + " | " + strconv.Itoa(s.Fail) + " |\n")
	}
	b.WriteString("\n")

	// Open findings (§2.5: no score without findings).
	b.WriteString("## Open findings\n\n")
	if len(findings) == 0 {
		b.WriteString("None. Every question passed for every agent with zero warnings.\n")
	} else {
		for _, f := range findings {
			b.WriteString(renderFindingLine(f) + "\n")
		}
	}

	return b.String()
}

// renderFindingLine renders one Finding as a single Markdown list item naming the
// severity, FM, agent, question, signal, and indicted surface artifact(s) — the
// closed-loop record the report enumerates (§2.5).
func renderFindingLine(f Finding) string {
	var parts []string
	parts = append(parts, "**"+f.Severity+"**")
	if f.FailureMode != "" {
		parts = append(parts, f.FailureMode)
	}
	if f.Agent != "" {
		parts = append(parts, "agent="+f.Agent)
	}
	if f.QuestionID != "" {
		parts = append(parts, f.QuestionID)
	}
	if f.Signal != "" {
		parts = append(parts, "signal="+f.Signal)
	}
	line := "- " + strings.Join(parts, " — ")
	if f.Expected != "" || f.Got != "" {
		line += " (expected: " + f.Expected + "; got: " + f.Got + ")"
	}
	if len(f.Indicts) > 0 {
		line += " indicts: " + strings.Join(f.Indicts, ", ")
	}
	return line
}

// renderReportJSON builds the machine-readable report (§7), the JSON mirror of
// the Markdown. It encodes deterministically: a stable schema, the parameterized
// date, and the tracked/gated posture flags.
func renderReportJSON(ordered []AgentRun, date, floorAgent string, firstViolation *Finding, findings []Finding) []byte {
	rep := reportJSON{
		Schema:         reportSchema,
		Date:           date,
		Tracked:        true,
		Gated:          false,
		FloorAgent:     floorAgent,
		FirstViolation: firstViolation,
		Findings:       findings,
	}
	for _, run := range ordered {
		s := summarize(run)
		rep.Agents = append(rep.Agents, agentSummaryJSON{
			Agent:  run.Agent,
			Rank:   run.Rank,
			Clears: s.Clears,
			Pass:   s.Pass,
			Warn:   s.Warn,
			Fail:   s.Fail,
		})
	}
	return encodeReportJSON(rep)
}

// encodeReportJSON serializes the report manifest to deterministic bytes:
// 2-space indented, HTML escaping off, trailing newline — matching the
// field-coverage / battery.json encoding so the artifact is byte-stable.
func encodeReportJSON(rep reportJSON) []byte {
	var sb strings.Builder
	enc := json.NewEncoder(&sb)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(rep); err != nil {
		// rep is composed only of strings/ints/bools/slices of the same — it cannot
		// fail to marshal. Returning an empty slice keeps Render total.
		return nil
	}
	return []byte(sb.String())
}

// agentSummary is the per-agent verdict tally + clears bit used by both
// renderers.
type agentSummary struct {
	Clears           bool
	Pass, Warn, Fail int
}

// summarize tallies a run's verdicts and computes its clears bit.
func summarize(run AgentRun) agentSummary {
	s := agentSummary{Clears: Clears(run)}
	for _, r := range run.Results {
		switch r.Verdict {
		case VerdictPass:
			s.Pass++
		case VerdictWarn:
			s.Warn++
		case VerdictFail:
			s.Fail++
		}
	}
	return s
}

// yesNo renders a bool as a compact table cell.
func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
