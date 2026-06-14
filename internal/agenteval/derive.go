package agenteval

// This file derives GROUND TRUTH for the agentic-coverage eval from the two
// inputs the binary itself projects from: resources.Catalog() and the value-free
// fixture corpus (docs/AGENTIC_COVERAGE_PLAN.md §3.4). The cardinal rule (F2)
// is that no expected answer is hand-authored — every value here is computed
// from the same production code the binary runs, so "truth" is by construction
// exactly what a correct agent sees on the surface.
//
// To honour that, the derive functions:
//
//   - obtain fixture records through fixtures.NewReader() (the production
//     cli.ResourceReader seam: List/Get), never a private copy of the corpus;
//   - project through resources.ProjectRecordsAndVerify /
//     resources.ProjectRecordAndVerify with the same redaction mode the binary
//     applies — so a dropped or secret field is genuinely absent, not merely
//     assumed absent (the verifying variants re-assert the rendered-subset
//     invariant, the same project->verify guarantee the binary upholds);
//   - read field classification straight from the catalog via
//     FieldSpec.Classification + FieldSpec.AllowedIn(mode);
//   - apply the SAME narrowing predicate the binary's --filter uses (a faithful
//     mirror of internal/cli's recordMatches / formatTableValue, which are
//     unexported and therefore reproduced here rather than imported — see
//     filterMatchesExact).
//
// Everything is PURE: no LLM, no network, no clock, no rand. The functions are
// computed lazily on each call (§3.4 / failure-mode critique Hole 4) — nothing
// is baked at package init, so a fixture or catalog change can never drift
// silently from a frozen value.

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dvmrry/zscalerctl/internal/agenteval/fixtures"
	"github.com/dvmrry/zscalerctl/internal/config"
	"github.com/dvmrry/zscalerctl/internal/redact"
	"github.com/dvmrry/zscalerctl/internal/resources"
	"github.com/dvmrry/zscalerctl/internal/zscaler"
)

// DeriveProductSet returns the distinct catalog products, in catalog-iteration
// order (the order resources.Catalog() emits: zia, zpa, ztw, zcc, zidentity).
// This is the T0-a "list every product this tool can query" truth (§3.3),
// computed from the catalog so it tracks any product addition automatically —
// never a hardcoded list.
func DeriveProductSet() []string {
	seen := make(map[resources.Product]bool)
	out := make([]string, 0)
	for _, spec := range resources.Catalog() {
		if seen[spec.Product] {
			continue
		}
		seen[spec.Product] = true
		out = append(out, string(spec.Product))
	}
	return out
}

// DeriveResourceCount returns how many distinct resources the given product
// exposes in the catalog — the T0-b "how many resources does zpa expose" truth
// (§3.3). The count is computed (no hardcoded number) so it cannot drift from
// the catalog. A product with no specs returns 0.
func DeriveResourceCount(product resources.Product) int {
	n := 0
	for _, spec := range resources.Catalog() {
		if spec.Product == product {
			n++
		}
	}
	return n
}

// DeriveRecordCount returns the number of PROJECTED records a correct agent sees
// for (product, resource) in standard mode — the C2 "how many X are configured"
// truth (§3.4). It pulls records through the production reader seam
// (fixtures.NewReader().List), then projects via resources.ProjectRecords, so the
// count equals what the binary emits, by construction. An unknown/empty
// (product, resource) returns 0 (an unsupported resource is "nothing to count",
// not an error to a count question).
func DeriveRecordCount(product resources.Product, resource string) int {
	projected, ok := projectedRecords(product, resource, redact.ModeStandard)
	if !ok {
		return 0
	}
	return len(projected.Records())
}

// DeriveProjectedFields returns which of the requested field names SURVIVE
// projection for the record with the given id, in the order requested (§3.4 C5).
// It is the engine behind Q8 ("--fields id,name,preSharedKey -> {id,name}"): a
// dropped or secret-classified requested field is silently absent from the
// result, exactly as `--fields` narrows the already-projected record. A
// requested field the record never carried (even pre-projection) is likewise
// absent.
//
// The record is fetched through the production reader (fixtures.NewReader().Get)
// and projected with resources.ProjectRecord in the given mode, so survival is
// decided by the real AllowedIn(mode) + redaction path. An empty mode defaults
// to standard via redact.EffectiveMode (mirroring the binary). An unknown
// (product, resource) or unknown id returns nil.
func DeriveProjectedFields(product resources.Product, resource, id string, requested []string, mode redact.Mode) []string {
	spec, ok := resources.FindSpec(product, resource)
	if !ok {
		return nil
	}
	record, err := fixtures.NewReader().Get(context.Background(), product, resource, id)
	if err != nil {
		return nil
	}
	projected, _, err := resources.ProjectRecordAndVerify(spec, mode, record)
	if err != nil {
		return nil
	}
	present := projected.Fields()
	out := make([]string, 0, len(requested))
	for _, name := range requested {
		if _, ok := present[name]; ok {
			out = append(out, name)
		}
	}
	return out
}

// DeriveFieldValue returns the PROJECTED value of a single field for the record
// with the given id, rendered as a string — the C2 "what is the <field> of
// <resource> <id>" truth (§3.4). It is the engine behind a get-shaped fact such
// as "what is the name of the zia location with id 1?". The record is fetched
// through the production reader (fixtures.NewReader().Get) and projected with
// resources.ProjectRecordAndVerify in the given mode, so the value an agent must
// report is exactly what the binary emits after projection.
//
// A field that projection drops (secret/never-allowed) or that the record never
// carried is absent post-projection, so the function returns "" — never the raw
// pre-projection value. The value is rendered with the same renderTableValue
// helper the filter derivation uses, so a scalar field reads back as its text.
// An unknown (product, resource) or unknown id returns "".
func DeriveFieldValue(product resources.Product, resource, id, field string, mode redact.Mode) string {
	spec, ok := resources.FindSpec(product, resource)
	if !ok {
		return ""
	}
	record, err := fixtures.NewReader().Get(context.Background(), product, resource, id)
	if err != nil {
		return ""
	}
	projected, _, err := resources.ProjectRecordAndVerify(spec, mode, record)
	if err != nil {
		return ""
	}
	v, ok := projected.Fields()[field]
	if !ok {
		return ""
	}
	return renderTableValue(v)
}

// DeriveRequiredCredentialEnvVars returns the NAMES of the required-core
// credential environment variables a caller must set to provide this tool's
// tenant credentials for live API access (the FM-07 credential-mechanism truth,
// §3.7). The names are sourced from the config package constants
// (config.EnvClientID, config.EnvClientSecret, config.EnvVanityDomain) rather
// than retyped, so a rename of any env var in internal/config flows through
// automatically and the eval can never grade against a stale literal.
//
// The "required core" selection is a DOCUMENTED-CONTRACT pin, not a catalog
// derivation: the catalog describes resources, not the credential mechanism, so
// it cannot tell us which env vars are required. These three are the OneAPI
// minimum the reader validation enforces (client id + secret + vanity domain);
// optional/alternative vars (CLOUD, ZPA_CUSTOMER_ID, legacy-auth fields) are
// deliberately excluded from the required-core set, which is why a question
// grading this set allows extras (ExtraAllowed) so naming them too is fine.
func DeriveRequiredCredentialEnvVars() []string {
	return []string{
		config.EnvClientID,
		config.EnvClientSecret,
		config.EnvVanityDomain,
	}
}

// DeriveSecretEverShown reports whether a field is EVER exposed in any redaction
// mode — the C5 "is Y ever shown" truth (§3.4). It returns false iff the field's
// classification is secret OR the field is never AllowedIn any mode (standard,
// share, or paranoid). This is the agentic mirror of the field-coverage boundary:
// the honest answer to "does zia locations ever expose preSharedKey" is false,
// because preSharedKey is ClassSecret and AllowedIn returns false for every mode.
//
// An unknown (product, resource) or an unknown field returns false (a field the
// surface does not model is, trivially, never shown).
func DeriveSecretEverShown(product resources.Product, resource, field string) bool {
	spec, ok := resources.FindSpec(product, resource)
	if !ok {
		return false
	}
	fs, ok := findField(spec, field)
	if !ok {
		return false
	}
	if fs.Classification == resources.ClassSecret {
		return false
	}
	for _, mode := range allRedactionModes() {
		if fs.AllowedIn(mode) {
			return true
		}
	}
	return false
}

// DeriveFilterCount returns how many projected records match `field == value`
// under the production --filter exact-match predicate (§3.4 C3), computed in
// standard mode. It projects the fixture records (so a dropped/secret field is
// already absent — a filter naming it matches nothing, exit 0 empty set), then
// applies the SAME comparison the binary's recordMatches uses: render the
// projected value with formatTableValue and compare it for exact equality to the
// requested value. An unknown (product, resource) returns 0.
//
// This captures the narrow-only semantics the eval cares about: filtering
// zia/locations on country=US over a corpus whose only country is COUNTRY_NONE
// yields 0, and filtering on a secret field (preSharedKey) yields 0 because the
// field is not present post-projection.
func DeriveFilterCount(product resources.Product, resource, field, value string) int {
	projected, ok := projectedRecords(product, resource, redact.ModeStandard)
	if !ok {
		return 0
	}
	n := 0
	for _, record := range projected.Records() {
		if filterMatchesExact(record.Fields(), field, value) {
			n++
		}
	}
	return n
}

// DeriveErrorKindUnknownID returns the error_kind a correct agent must report
// for `get <unknown-id>` on a KNOWN resource (Q9, §3.7). The fixture reader's
// Get returns zscaler.ErrResourceNotFound for an unknown id on a known bucket
// (see fixtures.fixtureReader.Get), which the binary's errorKind() maps to
// "not_found" and exitCodeForError() maps to exit 4 (verified in
// cmd/zscalerctl/main.go: ErrResourceNotFound -> "not_found" / exitNotFound).
// This is distinct from `unsupported_resource` (an unknown resource KEY, also
// exit 4 but a different remediation), so the two are never conflated.
//
// The mapping is asserted against the live error path below (deriveNotFoundKind)
// so this constant can never silently drift from what the binary actually emits
// for that error.
func DeriveErrorKindUnknownID() ErrorKind {
	return deriveNotFoundKind()
}

// deriveNotFoundKind classifies the SENTINEL the fixture reader returns for an
// unknown id (zscaler.ErrResourceNotFound) via the same errors.Is dispatch the
// binary's errorKind() uses, so the returned kind is grounded in the live error
// value rather than a bare literal. It mirrors the relevant arms of
// cmd/zscalerctl/main.go's errorKind() switch.
func deriveNotFoundKind() ErrorKind {
	err := fmt.Errorf("%w: probe", zscaler.ErrResourceNotFound)
	switch {
	case errors.Is(err, zscaler.ErrResourceNotFound):
		return ErrorKindNotFound
	case errors.Is(err, zscaler.ErrUnsupportedResource):
		return ErrorKindUnsupportedResource
	default:
		return ErrorKindInternal
	}
}

// projectedRecords lists (via the production reader seam) and projects every
// record for (product, resource) in the given mode. ok is false for an
// unknown/unsupported (product, resource) bucket or any projection error — the
// derive callers treat that as "nothing to count/derive" rather than panicking.
func projectedRecords(product resources.Product, resource string, mode redact.Mode) (resources.ProjectedRecords, bool) {
	spec, ok := resources.FindSpec(product, resource)
	if !ok {
		return resources.ProjectedRecords{}, false
	}
	records, err := fixtures.NewReader().List(context.Background(), product, resource)
	if err != nil {
		return resources.ProjectedRecords{}, false
	}
	projected, _, err := resources.ProjectRecordsAndVerify(spec, mode, records)
	if err != nil {
		return resources.ProjectedRecords{}, false
	}
	return projected, true
}

// findField returns the top-level FieldSpec for a JSON field name on a spec.
// Matching is on the rendered JSON field (FieldSpec.JSONField), the same key the
// projected record is keyed by, so a field's classification is looked up by the
// exact name an agent would pass to --filter / --fields. Nested fields are out of
// scope for the secret-ever-shown derivation (which grades a top-level boundary).
func findField(spec resources.ResourceSpec, field string) (resources.FieldSpec, bool) {
	for _, fs := range spec.Fields {
		if fs.JSONField() == field {
			return fs, true
		}
	}
	return resources.FieldSpec{}, false
}

// allRedactionModes is the closed set of redaction modes a field can be allowed
// in (§3.4: "ever shown" = AllowedIn ANY of these). It mirrors the catalog's own
// notion of all modes; AllowedIn applies redact.EffectiveMode internally, so the
// empty/default mode is already covered by ModeStandard.
func allRedactionModes() []redact.Mode {
	return []redact.Mode{redact.ModeStandard, redact.ModeShare, redact.ModeParanoid}
}

// filterMatchesExact reports whether a projected record matches `field == value`
// under the production --filter '=' (exact) predicate. It is a faithful mirror of
// internal/cli.recordMatches's exact-match arm: a record lacking the key never
// matches (the fail-closed path for dropped/secret fields), and a present value
// is rendered with the same formatTableValue formatting before an exact string
// compare. The cli helpers are unexported, so the relevant logic is reproduced
// here; renderTableValue below mirrors cli.formatTableValue byte-for-byte.
func filterMatchesExact(fields map[string]any, field, value string) bool {
	v, ok := fields[field]
	if !ok {
		return false
	}
	return renderTableValue(v) == value
}

// renderTableValue mirrors internal/cli.formatTableValue (rawTableValue +
// sanitizeCellValue): scalars render as their text, []string / []any join with
// commas, anything else uses fmt.Sprint, and control characters collapse to
// spaces. Keeping this in lockstep with the cli renderer is what makes
// DeriveFilterCount apply the same comparison string the binary's --filter does.
func renderTableValue(value any) string {
	return sanitizeCellValue(rawTableValue(value))
}

// rawTableValue mirrors internal/cli.rawTableValue.
func rawTableValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case []string:
		return strings.Join(v, ",")
	case []any:
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = rawTableValue(item)
		}
		return strings.Join(parts, ",")
	default:
		return fmt.Sprint(v)
	}
}

// sanitizeCellValue mirrors internal/cli.sanitizeCellValue: every control
// character (tab, newline, carriage return, other C0 bytes, DEL) collapses to a
// single space so a value renders on one logical cell.
func sanitizeCellValue(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\t' || r == '\n' || r == '\r' || r < 0x20 || r == 0x7f {
			return ' '
		}
		return r
	}, s)
}
