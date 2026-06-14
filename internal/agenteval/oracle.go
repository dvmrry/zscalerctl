package agenteval

// This file is the ORACLE: it assembles the typed Expected values a question
// asserts on, drawing exclusively from derive.go (catalog + fixtures), and it
// runs the fail-closed self-check that no expected ANSWER ever references a
// dropped or secret field (docs/AGENTIC_COVERAGE_PLAN.md §3.6
// TestOracleMatchesFixtures, §2.5 Finding).
//
// Why the oracle lives here and not in the battery: the battery phase (battery.go,
// a later phase) owns the []Question and the Templates; it will CALL this oracle
// to turn each question's declared derivation into concrete Assertions and to run
// the allow-list self-check. Putting the reusable derivation + self-check helpers
// here — keyed by a small QuestionSpec seam the battery fills in — avoids a
// circular dependency: the oracle never imports the battery, the battery imports
// nothing new to use the oracle. If the battery does not exist yet, these helpers
// are still exercised directly by TestOracleMatchesFixtures.
//
// The contract with the battery is the seam below: a QuestionSpec describes WHAT
// to derive (a derivation kind + its product/resource/field/id/mode/value
// inputs); BuildAssertions turns it into the typed Assertion(s) the scorer
// grades; CheckExpectedNotSecret asserts the Expected value is allow-listed in
// the active mode. Everything is PURE (catalog + fixtures only): no LLM, no
// network, no clock, no rand.

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dvmrry/zscalerctl/internal/redact"
	"github.com/dvmrry/zscalerctl/internal/resources"
)

// DerivationKind names HOW a question's expected answer is derived from the
// catalog+fixtures. It is the battery↔oracle seam: a battery template declares a
// DerivationKind plus the inputs it needs, and the oracle computes the typed
// Expected from derive.go. The kinds line up with §3.4's derive_fns.
type DerivationKind string

const (
	// DeriveProducts -> KindSet of catalog products (T0-a, §3.3).
	DeriveProducts DerivationKind = "products"
	// DeriveResourceCountKind -> KindCount of a product's resources (T0-b).
	DeriveResourceCountKind DerivationKind = "resource_count"
	// DeriveRecordCountKind -> KindCount of projected records (C2, §3.4).
	DeriveRecordCountKind DerivationKind = "record_count"
	// DeriveFilterCountKind -> KindCount after the production --filter predicate
	// (C3, §3.4).
	DeriveFilterCountKind DerivationKind = "filter_count"
	// DeriveProjectedFieldsKind -> KindSet of requested fields that survive
	// projection (C5/Q8, §3.4).
	DeriveProjectedFieldsKind DerivationKind = "projected_fields"
	// DeriveSecretEverShownKind -> KindBool "is field Y ever exposed" (C5/Q7,
	// §3.4); the honest answer for a secret/never-allowed field is false.
	DeriveSecretEverShownKind DerivationKind = "secret_ever_shown"
	// DeriveErrorKindUnknownIDKind -> KindErrorKind for get <unknown-id> on a
	// known resource (Q9, §3.4); not_found.
	DeriveErrorKindUnknownIDKind DerivationKind = "error_kind_unknown_id"
)

// QuestionSpec is the battery's declarative request for derived truth (the
// §3.4 derive_fn binding). The battery fills the fields its Derivation needs and
// hands the spec to the oracle; the oracle never authors a value, it only
// derives from the catalog+fixtures. Unused fields for a given Derivation are
// ignored (e.g. Field is irrelevant to a record-count question).
type QuestionSpec struct {
	// Derivation selects which derive_fn computes the expected answer.
	Derivation DerivationKind
	// Product is the catalog product the question targets (e.g. resources.ProductZIA).
	Product resources.Product
	// Resource is the catalog resource name (e.g. "locations").
	Resource string
	// ID is the record id for record-scoped derivations (projected_fields).
	ID string
	// Field is the field name for field-scoped derivations
	// (secret_ever_shown, filter_count).
	Field string
	// Value is the filter value for filter_count (the right-hand side of
	// --filter field=value).
	Value string
	// Requested is the requested --fields list for projected_fields, in order.
	Requested []string
	// Mode is the redaction mode the question pins (F5); the empty Mode defaults
	// to standard via redact.EffectiveMode, mirroring the binary.
	Mode redact.Mode
}

// BuildAssertions derives the typed Assertion(s) the scorer grades for this
// question (§2.2). A single-derivation question yields one Assertion; a future
// dual-assertion C6 question (exit_code + error_kind) would yield two, but the
// derivations here each produce exactly one — the battery composes a second
// assertion (the observed exit_code) alongside an error_kind derivation when it
// builds a Q9-style question. The returned Expected strings use the encodings
// the scorer expects: a count is decimal, a bool is "true"/"false", a set is
// comma-joined, an error_kind is the literal kind string.
//
// An unknown derivation kind is an error (a battery bug), never a silent empty
// assertion.
func (s QuestionSpec) BuildAssertions() ([]Assertion, error) {
	switch s.Derivation {
	case DeriveProducts:
		return []Assertion{{
			Kind:     KindSet,
			Expected: strings.Join(DeriveProductSet(), ","),
		}}, nil

	case DeriveResourceCountKind:
		return []Assertion{{
			Kind:     KindCount,
			Expected: strconv.Itoa(DeriveResourceCount(s.Product)),
		}}, nil

	case DeriveRecordCountKind:
		return []Assertion{{
			Kind:     KindCount,
			Expected: strconv.Itoa(DeriveRecordCount(s.Product, s.Resource)),
		}}, nil

	case DeriveFilterCountKind:
		return []Assertion{{
			Kind:     KindCount,
			Expected: strconv.Itoa(DeriveFilterCount(s.Product, s.Resource, s.Field, s.Value)),
		}}, nil

	case DeriveProjectedFieldsKind:
		fields := DeriveProjectedFields(s.Product, s.Resource, s.ID, s.Requested, s.Mode)
		return []Assertion{{
			Kind:     KindSet,
			Expected: strings.Join(fields, ","),
		}}, nil

	case DeriveSecretEverShownKind:
		return []Assertion{{
			Kind:     KindBool,
			Expected: strconv.FormatBool(DeriveSecretEverShown(s.Product, s.Resource, s.Field)),
		}}, nil

	case DeriveErrorKindUnknownIDKind:
		return []Assertion{{
			Kind:     KindErrorKind,
			Expected: string(DeriveErrorKindUnknownID()),
		}}, nil

	default:
		return nil, fmt.Errorf("oracle: unknown derivation kind %q", s.Derivation)
	}
}

// CheckExpectedNotSecret is the fail-closed self-check (§3.6): it asserts that a
// question's derived ANSWER never references a value the tool would not show —
// i.e. you never expect a dropped or secret field. A question whose expected
// answer NAMES a field (projected_fields) must only name fields that survive
// projection in the question's mode; a secret-ever-shown question must expect
// false for a secret/never-allowed field (expecting true would claim the tool
// exposes it). A leak here is a TEST BUG caught at construction, not a runtime
// agent miss.
//
// It returns nil when the question is safe. For derivations that cannot reference
// a field value (counts, products, error kinds), it is trivially safe.
func (s QuestionSpec) CheckExpectedNotSecret() error {
	switch s.Derivation {
	case DeriveProjectedFieldsKind:
		// Every field the oracle expects to appear must genuinely survive
		// projection in the pinned mode. Re-derive and confirm each expected field
		// is present in the projected record — a dropped/secret field can never be
		// an expected answer.
		expected := DeriveProjectedFields(s.Product, s.Resource, s.ID, s.Requested, s.Mode)
		present := projectedFieldSet(s.Product, s.Resource, s.ID, s.Mode)
		for _, f := range expected {
			if !present[f] {
				return fmt.Errorf(
					"oracle self-check: question expects field %q in %s/%s id %q (mode %s), but it is not in the projected record (dropped/secret)",
					f, s.Product, s.Resource, s.ID, effectiveModeName(s.Mode),
				)
			}
		}
		// Belt-and-braces: none of the expected fields may be a secret/never-shown
		// field, regardless of whether projection happened to drop it.
		for _, f := range expected {
			if !DeriveSecretEverShown(s.Product, s.Resource, f) {
				return fmt.Errorf(
					"oracle self-check: question expects field %q in %s/%s, but that field is never shown in any mode (secret/dropped)",
					f, s.Product, s.Resource,
				)
			}
		}
		return nil

	case DeriveSecretEverShownKind:
		// A secret-ever-shown question is safe iff its expected bool matches the
		// derived classification. The danger is expecting true for a field the tool
		// never exposes (which would invite the agent to fabricate it). Re-derive
		// and require the field NOT be claimed shown when it is in fact secret.
		shown := DeriveSecretEverShown(s.Product, s.Resource, s.Field)
		if shown {
			// Expecting "shown" is only safe when the field truly is shown; the
			// derivation guarantees this is grounded in the catalog, so a true here
			// is by construction a non-secret field. Nothing to reject.
			return nil
		}
		// Field is not shown -> the only safe expected answer is false, which the
		// BuildAssertions path produces. No secret value is ever in the answer.
		return nil

	case DeriveFilterCountKind:
		// A filter on a secret/never-shown field is a legitimate question (its
		// truth is 0 because the field is absent post-projection), so this is safe.
		// The ANSWER is a count, never the secret value, so nothing to reject.
		return nil

	default:
		// Counts, product sets, and error kinds never reference a field value.
		return nil
	}
}

// projectedFieldSet returns the set of fields present in the projected record for
// (product, resource, id) in the given mode — the post-projection field set the
// self-check compares an expected-fields answer against. An unknown record yields
// an empty set.
func projectedFieldSet(product resources.Product, resource, id string, mode redact.Mode) map[string]bool {
	// Reuse DeriveProjectedFields with the full modeled field order so the result
	// is exactly the projected keys, regardless of any narrowing the question
	// requested. We ask for every field the spec models so survival is decided
	// purely by projection.
	spec, ok := resources.FindSpec(product, resource)
	if !ok {
		return map[string]bool{}
	}
	all := make([]string, 0, len(spec.Fields))
	for _, fs := range spec.Fields {
		all = append(all, fs.JSONField())
	}
	present := DeriveProjectedFields(product, resource, id, all, mode)
	out := make(map[string]bool, len(present))
	for _, f := range present {
		out[f] = true
	}
	return out
}

// effectiveModeName renders a mode for error messages, applying the same
// empty->standard default the binary's projection uses.
func effectiveModeName(mode redact.Mode) string {
	return string(redact.EffectiveMode(mode))
}

// CheckBatterySpecs runs CheckExpectedNotSecret across a slice of question specs
// (the battery phase passes its instantiated specs here as the allow-list
// self-check). It returns the FIRST offending spec's error, so a battery that
// would leak a dropped/secret field fails construction with a precise reason.
// The battery phase calls this; TestOracleMatchesFixtures calls it directly with
// the modeled specs to prove the self-check holds today.
func CheckBatterySpecs(specs []QuestionSpec) error {
	for i, s := range specs {
		if err := s.CheckExpectedNotSecret(); err != nil {
			return fmt.Errorf("spec %d (%s %s/%s): %w", i, s.Derivation, s.Product, s.Resource, err)
		}
	}
	return nil
}
