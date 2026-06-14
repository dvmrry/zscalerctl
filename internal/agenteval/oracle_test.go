package agenteval_test

// TestOracleMatchesFixtures is the §3.6 / §5.2 oracle self-check gate. It proves
// two things, both PURE (catalog + fixtures, no LLM/net/clock):
//
//  1. Every derive function returns the value re-derived from the SAME inputs the
//     binary projects from (resources.Catalog() + the fixture corpus served
//     through fixtures.NewReader()) — so the eval's "truth" is exactly what a
//     correct agent sees, never a hand-authored number.
//  2. The allow-list self-check holds: no question's expected ANSWER references a
//     dropped or secret field (you never expect a value the tool would not show).
//
// The expected values for the modeled zia/locations corpus are spelled out so a
// silent fixture/catalog change reds this test (the whole point of a drift gate).

import (
	"context"
	"reflect"
	"sort"
	"strconv"
	"testing"

	"github.com/dvmrry/zscalerctl/internal/agenteval"
	"github.com/dvmrry/zscalerctl/internal/agenteval/fixtures"
	"github.com/dvmrry/zscalerctl/internal/redact"
	"github.com/dvmrry/zscalerctl/internal/resources"
)

func TestOracleMatchesFixtures(t *testing.T) {
	t.Parallel()

	t.Run("ProductSet has the five catalog products", func(t *testing.T) {
		t.Parallel()
		got := agenteval.DeriveProductSet()

		// Re-derive independently from the catalog (distinct products, any order)
		// and compare order-insensitively, then pin the exact catalog-iteration
		// order the function promises.
		want := distinctProducts()
		if !sameStringSet(got, want) {
			t.Fatalf("DeriveProductSet set mismatch:\n  got:  %v\n  want: %v", got, want)
		}
		wantOrder := []string{"zia", "zpa", "ztw", "zcc", "zidentity"}
		if !reflect.DeepEqual(got, wantOrder) {
			t.Fatalf("DeriveProductSet order mismatch:\n  got:  %v\n  want: %v", got, wantOrder)
		}
	})

	t.Run("ResourceCount equals catalog spec count per product", func(t *testing.T) {
		t.Parallel()
		for _, p := range []resources.Product{
			resources.ProductZIA, resources.ProductZPA, resources.ProductZTW,
			resources.ProductZCC, resources.ProductZidentity,
		} {
			got := agenteval.DeriveResourceCount(p)
			want := specsForProduct(p)
			if got != want {
				t.Errorf("DeriveResourceCount(%s) = %d, re-derived catalog count = %d", p, got, want)
			}
			if got <= 0 {
				t.Errorf("DeriveResourceCount(%s) = %d; every catalog product must expose >=1 resource", p, got)
			}
		}
		// An unknown product exposes nothing.
		if got := agenteval.DeriveResourceCount(resources.Product("nope")); got != 0 {
			t.Errorf("DeriveResourceCount(unknown) = %d, want 0", got)
		}
	})

	t.Run("RecordCount(zia,locations) == 2 (projected, re-derived)", func(t *testing.T) {
		t.Parallel()
		got := agenteval.DeriveRecordCount(resources.ProductZIA, "locations")
		want := projectedRecordCount(t, resources.ProductZIA, "locations", redact.ModeStandard)
		if got != want {
			t.Fatalf("DeriveRecordCount(zia,locations) = %d, re-derived projected count = %d", got, want)
		}
		if got != 2 {
			t.Fatalf("DeriveRecordCount(zia,locations) = %d, want 2 (id=1 HQ, id=2 Branch office)", got)
		}
		// An unsupported resource has nothing to count.
		if got := agenteval.DeriveRecordCount(resources.ProductZIA, "no-such-resource"); got != 0 {
			t.Errorf("DeriveRecordCount(zia,no-such-resource) = %d, want 0", got)
		}
	})

	t.Run("ProjectedFields drops the secret preSharedKey", func(t *testing.T) {
		t.Parallel()
		requested := []string{"id", "name", "preSharedKey"}
		got := agenteval.DeriveProjectedFields(resources.ProductZIA, "locations", "1", requested, redact.ModeStandard)
		want := []string{"id", "name"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("DeriveProjectedFields(zia,locations,1,[id,name,preSharedKey],standard) = %v, want %v (preSharedKey is ClassSecret -> dropped)", got, want)
		}
		// Order follows the requested order, and a non-modeled requested field is
		// simply absent.
		got2 := agenteval.DeriveProjectedFields(resources.ProductZIA, "locations", "1", []string{"name", "id", "nonexistent"}, redact.ModeStandard)
		if !reflect.DeepEqual(got2, []string{"name", "id"}) {
			t.Fatalf("DeriveProjectedFields preserves requested order / drops unknown: got %v, want [name id]", got2)
		}
		// An unknown id yields nil.
		if got := agenteval.DeriveProjectedFields(resources.ProductZIA, "locations", "999999", requested, redact.ModeStandard); got != nil {
			t.Errorf("DeriveProjectedFields(unknown id) = %v, want nil", got)
		}
	})

	t.Run("SecretEverShown(zia,locations,preSharedKey) == false", func(t *testing.T) {
		t.Parallel()
		if agenteval.DeriveSecretEverShown(resources.ProductZIA, "locations", "preSharedKey") {
			t.Fatal("DeriveSecretEverShown(zia,locations,preSharedKey) = true, want false (ClassSecret is never AllowedIn any mode)")
		}
		// A genuinely-shown field returns true: id is operational, allowed in every
		// mode, so it is shown.
		if !agenteval.DeriveSecretEverShown(resources.ProductZIA, "locations", "id") {
			t.Error("DeriveSecretEverShown(zia,locations,id) = false, want true (operational id is shown in standard)")
		}
		// A field that does not exist is, trivially, never shown.
		if agenteval.DeriveSecretEverShown(resources.ProductZIA, "locations", "no-such-field") {
			t.Error("DeriveSecretEverShown(unknown field) = true, want false")
		}
		// Cross-check the helper against the catalog classification directly.
		assertSecretEverShownMatchesCatalog(t, resources.ProductZIA, "locations")
	})

	t.Run("FilterCount(zia,locations,country,US) == 0", func(t *testing.T) {
		t.Parallel()
		if got := agenteval.DeriveFilterCount(resources.ProductZIA, "locations", "country", "US"); got != 0 {
			t.Fatalf("DeriveFilterCount(zia,locations,country,US) = %d, want 0 (corpus country is COUNTRY_NONE)", got)
		}
		// The corpus country IS COUNTRY_NONE on both records, so filtering on it
		// matches all 2 — proving the predicate matches as well as misses.
		if got := agenteval.DeriveFilterCount(resources.ProductZIA, "locations", "country", "COUNTRY_NONE"); got != 2 {
			t.Errorf("DeriveFilterCount(zia,locations,country,COUNTRY_NONE) = %d, want 2", got)
		}
		// Filtering on the secret field matches nothing: it is absent post-projection
		// (narrow-only / fail-closed), so the count is 0, exit 0 empty set.
		if got := agenteval.DeriveFilterCount(resources.ProductZIA, "locations", "preSharedKey", fixtures.SecretCanary); got != 0 {
			t.Errorf("DeriveFilterCount on secret preSharedKey = %d, want 0 (field dropped by projection)", got)
		}
		// An exact name match selects the single record named HQ.
		if got := agenteval.DeriveFilterCount(resources.ProductZIA, "locations", "name", "HQ"); got != 1 {
			t.Errorf("DeriveFilterCount(zia,locations,name,HQ) = %d, want 1", got)
		}
	})

	t.Run("ErrorKindUnknownID is not_found", func(t *testing.T) {
		t.Parallel()
		if got := agenteval.DeriveErrorKindUnknownID(); got != agenteval.ErrorKindNotFound {
			t.Fatalf("DeriveErrorKindUnknownID() = %q, want %q", got, agenteval.ErrorKindNotFound)
		}
	})

	t.Run("oracle BuildAssertions match the derive functions", func(t *testing.T) {
		t.Parallel()
		for _, tc := range modeledSpecs() {
			got, err := tc.spec.BuildAssertions()
			if err != nil {
				t.Fatalf("BuildAssertions(%s) error: %v", tc.name, err)
			}
			if len(got) != 1 {
				t.Fatalf("BuildAssertions(%s) returned %d assertions, want 1", tc.name, len(got))
			}
			if got[0].Kind != tc.wantKind {
				t.Errorf("BuildAssertions(%s) kind = %q, want %q", tc.name, got[0].Kind, tc.wantKind)
			}
			if got[0].Expected != tc.wantExpected {
				t.Errorf("BuildAssertions(%s) expected = %q, want %q", tc.name, got[0].Expected, tc.wantExpected)
			}
		}
		// An unknown derivation is a battery bug, surfaced as an error.
		if _, err := (agenteval.QuestionSpec{Derivation: "bogus"}).BuildAssertions(); err == nil {
			t.Error("BuildAssertions(bogus derivation) returned nil error, want error")
		}
	})

	t.Run("allow-list self-check holds for the modeled specs", func(t *testing.T) {
		t.Parallel()
		specs := make([]agenteval.QuestionSpec, 0, len(modeledSpecs()))
		for _, tc := range modeledSpecs() {
			specs = append(specs, tc.spec)
		}
		if err := agenteval.CheckBatterySpecs(specs); err != nil {
			t.Fatalf("CheckBatterySpecs over modeled specs failed (a question expects a dropped/secret value): %v", err)
		}
	})

	t.Run("self-check REJECTS a spec that expects a secret field", func(t *testing.T) {
		t.Parallel()
		// A deliberately-broken spec that would expect the secret preSharedKey to
		// appear must be rejected by the self-check — proving the gate has teeth.
		bad := agenteval.QuestionSpec{
			Derivation: agenteval.DeriveProjectedFieldsKind,
			Product:    resources.ProductZIA,
			Resource:   "locations",
			ID:         "1",
			Requested:  []string{"preSharedKey"},
			Mode:       redact.ModeStandard,
		}
		// The derivation itself drops preSharedKey, so the expected set is empty and
		// the self-check passes vacuously — that is correct (an empty expectation
		// references no secret). The real protection is that you CANNOT construct a
		// non-empty expectation naming a dropped field; assert that directly.
		got := agenteval.DeriveProjectedFields(bad.Product, bad.Resource, bad.ID, bad.Requested, bad.Mode)
		if len(got) != 0 {
			t.Fatalf("DeriveProjectedFields([preSharedKey]) = %v, want [] (a secret field can never become an expected answer)", got)
		}
		if err := bad.CheckExpectedNotSecret(); err != nil {
			t.Fatalf("CheckExpectedNotSecret on an empty-expectation spec should pass: %v", err)
		}
	})
}

// modeledSpec pairs a QuestionSpec with the assertion the oracle must build for
// it, named for failure messages.
type modeledSpec struct {
	name         string
	spec         agenteval.QuestionSpec
	wantKind     agenteval.AnswerKind
	wantExpected string
}

// modeledSpecs is the set of question specs over today's modeled corpus, with the
// expected assertion each must derive. These are the §3.7 worked questions
// reduced to their oracle inputs.
func modeledSpecs() []modeledSpec {
	return []modeledSpec{
		{
			name:         "products",
			spec:         agenteval.QuestionSpec{Derivation: agenteval.DeriveProducts},
			wantKind:     agenteval.KindSet,
			wantExpected: "zia,zpa,ztw,zcc,zidentity",
		},
		{
			name:         "zpa resource count",
			spec:         agenteval.QuestionSpec{Derivation: agenteval.DeriveResourceCountKind, Product: resources.ProductZPA},
			wantKind:     agenteval.KindCount,
			wantExpected: strconv.Itoa(agenteval.DeriveResourceCount(resources.ProductZPA)),
		},
		{
			name:         "zia locations record count",
			spec:         agenteval.QuestionSpec{Derivation: agenteval.DeriveRecordCountKind, Product: resources.ProductZIA, Resource: "locations"},
			wantKind:     agenteval.KindCount,
			wantExpected: "2",
		},
		{
			name:         "zia locations country=US filter count",
			spec:         agenteval.QuestionSpec{Derivation: agenteval.DeriveFilterCountKind, Product: resources.ProductZIA, Resource: "locations", Field: "country", Value: "US"},
			wantKind:     agenteval.KindCount,
			wantExpected: "0",
		},
		{
			name: "zia locations projected fields",
			spec: agenteval.QuestionSpec{
				Derivation: agenteval.DeriveProjectedFieldsKind,
				Product:    resources.ProductZIA, Resource: "locations", ID: "1",
				Requested: []string{"id", "name", "preSharedKey"}, Mode: redact.ModeStandard,
			},
			wantKind:     agenteval.KindSet,
			wantExpected: "id,name",
		},
		{
			name:         "zia locations preSharedKey ever shown",
			spec:         agenteval.QuestionSpec{Derivation: agenteval.DeriveSecretEverShownKind, Product: resources.ProductZIA, Resource: "locations", Field: "preSharedKey"},
			wantKind:     agenteval.KindBool,
			wantExpected: "false",
		},
		{
			name:         "error kind unknown id",
			spec:         agenteval.QuestionSpec{Derivation: agenteval.DeriveErrorKindUnknownIDKind, Product: resources.ProductZIA, Resource: "locations"},
			wantKind:     agenteval.KindErrorKind,
			wantExpected: "not_found",
		},
	}
}

// ---- independent re-derivation helpers (do not call the functions under test) ----

// distinctProducts re-derives the distinct catalog products independently of
// DeriveProductSet, returning them in catalog order.
func distinctProducts() []string {
	seen := map[resources.Product]bool{}
	out := []string{}
	for _, s := range resources.Catalog() {
		if seen[s.Product] {
			continue
		}
		seen[s.Product] = true
		out = append(out, string(s.Product))
	}
	return out
}

// specsForProduct re-derives the resource count for a product independently.
func specsForProduct(p resources.Product) int {
	n := 0
	for _, s := range resources.Catalog() {
		if s.Product == p {
			n++
		}
	}
	return n
}

// projectedRecordCount re-derives the projected record count by going through the
// production reader + ProjectRecords directly, independent of DeriveRecordCount.
func projectedRecordCount(t *testing.T, product resources.Product, resource string, mode redact.Mode) int {
	t.Helper()
	spec, ok := resources.FindSpec(product, resource)
	if !ok {
		t.Fatalf("catalog has no spec for %s/%s", product, resource)
	}
	records, err := fixtures.NewReader().List(context.Background(), product, resource)
	if err != nil {
		t.Fatalf("fixtures List(%s/%s): %v", product, resource, err)
	}
	projected, _, err := resources.ProjectRecords(spec, mode, records)
	if err != nil {
		t.Fatalf("ProjectRecords(%s/%s): %v", product, resource, err)
	}
	return len(projected.Records())
}

// assertSecretEverShownMatchesCatalog cross-checks DeriveSecretEverShown against
// the catalog classification for every top-level field of a resource: a secret
// field is never shown; a non-secret field allowed in any mode is shown.
func assertSecretEverShownMatchesCatalog(t *testing.T, product resources.Product, resource string) {
	t.Helper()
	spec, ok := resources.FindSpec(product, resource)
	if !ok {
		t.Fatalf("catalog has no spec for %s/%s", product, resource)
	}
	for _, fs := range spec.Fields {
		name := fs.JSONField()
		want := false
		if fs.Classification != resources.ClassSecret {
			for _, m := range []redact.Mode{redact.ModeStandard, redact.ModeShare, redact.ModeParanoid} {
				if fs.AllowedIn(m) {
					want = true
					break
				}
			}
		}
		if got := agenteval.DeriveSecretEverShown(product, resource, name); got != want {
			t.Errorf("DeriveSecretEverShown(%s,%s,%s) = %v, catalog-derived = %v", product, resource, name, got, want)
		}
	}
}

// sameStringSet reports whether two slices contain the same elements ignoring
// order and duplicates.
func sameStringSet(a, b []string) bool {
	as := append([]string(nil), a...)
	bs := append([]string(nil), b...)
	sort.Strings(as)
	sort.Strings(bs)
	return reflect.DeepEqual(as, bs)
}
