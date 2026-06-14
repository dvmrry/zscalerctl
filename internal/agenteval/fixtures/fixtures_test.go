package fixtures

import (
	"context"
	"errors"
	"testing"

	"github.com/dvmrry/zscalerctl/internal/resources"
	"github.com/dvmrry/zscalerctl/internal/zscaler"
)

// projectName pulls the "name" field back out of a SourceRecord for assertions.
// SourceRecord is opaque, so the test round-trips through ProjectRecords with
// the real catalog spec in standard mode and reads the projected JSON map.
func projectName(t *testing.T, rec resources.SourceRecord) string {
	t.Helper()
	spec, ok := resources.FindSpec(resources.ProductZIA, "locations")
	if !ok {
		t.Fatal("zia/locations spec missing from catalog")
	}
	projected, _, err := resources.ProjectRecords(spec, "standard", []resources.SourceRecord{rec})
	if err != nil {
		t.Fatalf("ProjectRecords: %v", err)
	}
	marshaled, err := projected.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	return string(marshaled)
}

func TestListReturnsMultipleRecords(t *testing.T) {
	r := NewReader()
	got, err := r.List(context.Background(), resources.ProductZIA, "locations")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) <= 1 {
		t.Fatalf("List count = %d, want > 1 (cardinality mandate, plan §3.5)", len(got))
	}
}

func TestListRecordsHaveDistinctIdsAndNames(t *testing.T) {
	r := NewReader()
	got, err := r.List(context.Background(), resources.ProductZIA, "locations")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	seenName := map[string]bool{}
	for _, rec := range got {
		name := projectName(t, rec)
		if seenName[name] {
			t.Fatalf("duplicate projected record %q; records must be distinct", name)
		}
		seenName[name] = true
	}
	// The corpus ids must also be distinct.
	ids := corpus[corpusKey{resources.ProductZIA, "locations"}]
	seenID := map[string]bool{}
	for _, rec := range ids {
		if seenID[rec.id] {
			t.Fatalf("duplicate corpus id %q", rec.id)
		}
		seenID[rec.id] = true
	}
}

func TestGetKnownReturnsExpectedRecord(t *testing.T) {
	r := NewReader()
	rec, err := r.Get(context.Background(), resources.ProductZIA, "locations", "1")
	if err != nil {
		t.Fatalf("Get(known id 1): %v", err)
	}
	got := projectName(t, rec)
	if want := `"HQ"`; !contains(got, want) {
		t.Fatalf("Get(1) projected = %s, want it to contain %s", got, want)
	}
	// A different known id returns a different record.
	rec2, err := r.Get(context.Background(), resources.ProductZIA, "locations", "2")
	if err != nil {
		t.Fatalf("Get(known id 2): %v", err)
	}
	if want := `"Branch office"`; !contains(projectName(t, rec2), want) {
		t.Fatalf("Get(2) projected = %s, want it to contain %s", projectName(t, rec2), want)
	}
}

func TestGetUnknownIDReturnsNotFoundSentinel(t *testing.T) {
	r := NewReader()
	_, err := r.Get(context.Background(), resources.ProductZIA, "locations", "999999")
	if err == nil {
		t.Fatal("Get(unknown id): want error, got nil")
	}
	if !errors.Is(err, zscaler.ErrResourceNotFound) {
		t.Fatalf("Get(unknown id) error = %v, want errors.Is(err, zscaler.ErrResourceNotFound)", err)
	}
}

func TestUnknownResourceIsUnsupported(t *testing.T) {
	r := NewReader()
	_, listErr := r.List(context.Background(), resources.ProductZIA, "no-such-resource")
	if !errors.Is(listErr, zscaler.ErrUnsupportedResource) {
		t.Fatalf("List(unknown resource) error = %v, want errors.Is(err, zscaler.ErrUnsupportedResource)", listErr)
	}
	_, getErr := r.Get(context.Background(), resources.ProductZIA, "no-such-resource", "1")
	if !errors.Is(getErr, zscaler.ErrUnsupportedResource) {
		t.Fatalf("Get(unknown resource) error = %v, want errors.Is(err, zscaler.ErrUnsupportedResource)", getErr)
	}
}

// TestSecretCanaryRecordPresent proves the corpus carries the secret-classified
// preSharedKey field with the obviously-synthetic canary placeholder, so a
// downstream scenario can assert projection drops it.
func TestSecretCanaryRecordPresent(t *testing.T) {
	records := corpus[corpusKey{resources.ProductZIA, "locations"}]
	var found bool
	for _, rec := range records {
		v, ok := rec.fields["preSharedKey"]
		if !ok {
			continue
		}
		found = true
		if v != SecretCanary {
			t.Fatalf("preSharedKey = %v, want the synthetic canary %q", v, SecretCanary)
		}
	}
	if !found {
		t.Fatalf("no record carries a preSharedKey field; secret-field coverage missing (plan §3.5)")
	}

	// Defense in depth: the canary must be obviously synthetic, never realistic.
	if !contains(SecretCanary, "CANARY") {
		t.Fatalf("SecretCanary = %q is not obviously synthetic", SecretCanary)
	}

	// And projection must NOT carry the secret value into output.
	r := NewReader()
	rec, err := r.Get(context.Background(), resources.ProductZIA, "locations", "1")
	if err != nil {
		t.Fatalf("Get(1): %v", err)
	}
	if projected := projectName(t, rec); contains(projected, SecretCanary) {
		t.Fatalf("projected output leaked the secret canary: %s", projected)
	}
}

func contains(haystack, needle string) bool {
	return len(needle) == 0 || (len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
