package agenteval

// This file is the single source of truth for the failure-mode (FM) taxonomy
// of the agentic-coverage eval (docs/AGENTIC_COVERAGE_PLAN.md §3.2). The FM
// layer is the *attribution* layer: each FM names the surface weakness a wrong
// answer would indict, and every other file in the eval refers to an FM by its
// ID rather than re-stating the taxonomy.
//
// The taxonomy is closed: exactly FM-01..FM-08, mapped onto the six surface
// categories (C1..C6) plus the output-discipline cross-cut. Adding, renaming,
// or dropping an FM here ripples through the traceability gate
// (TestEveryAgentSurfacePromiseHasAnFM, §3.6), which asserts that every promise
// in surface_promises.json points at a valid FM and that every FM here is
// claimed by at least one promise. Nothing in this file execs, dials the
// network, reads the clock, or calls an LLM — it is pure data.

// FailureMode is one entry in the closed FM taxonomy.
//
// Category is the surface-feature category the FM attributes to (C1..C6, or the
// empty/cross-cut bucket for output discipline), matching the §3.2 table. Desc
// is the human-readable weakness the FM names; it is documentation, never parsed
// or matched against prose.
type FailureMode struct {
	// ID is the stable taxonomy identifier, e.g. "FM-03".
	ID string
	// Category is the §3.2 surface category, e.g. "C3". FM-06 (output
	// discipline) is a cross-cut and carries the empty string deliberately.
	Category string
	// Desc is the human-readable weakness this FM names.
	Desc string
}

// FailureModes is the closed FM taxonomy: exactly FM-01..FM-08 (§3.2). Order is
// presentation-only; the taxonomy is a set keyed by ID. FM-06 is the
// output-discipline cross-cut and intentionally carries no single Cn category
// (it indicts the same "parse output, not prose" promise as C2).
var FailureModes = []FailureMode{
	{ID: "FM-01", Category: "C1", Desc: "can't discover resource names"},
	{ID: "FM-02", Category: "C2", Desc: "mis-parses JSON output (parse output not prose)"},
	{ID: "FM-03", Category: "C3", Desc: "can't compose narrowing (--filter/--search)"},
	{ID: "FM-04", Category: "C6", Desc: "mishandles exit code / error kind"},
	{ID: "FM-05", Category: "C1", Desc: "can't discover --fields"},
	{ID: "FM-06", Category: "", Desc: "over-trusts pretty vs machine output"},
	{ID: "FM-07", Category: "C6", Desc: "can't find credentials"},
	{ID: "FM-08", Category: "C5", Desc: "over-reads the fail-closed/redacted boundary"},
}

// FailureModeIDs returns the set of valid FM IDs as a lookup map. Callers (the
// traceability gate, future graders) use it to validate that an FM reference
// names a taxonomy member without re-stating the closed list.
func FailureModeIDs() map[string]bool {
	ids := make(map[string]bool, len(FailureModes))
	for _, fm := range FailureModes {
		ids[fm.ID] = true
	}
	return ids
}
