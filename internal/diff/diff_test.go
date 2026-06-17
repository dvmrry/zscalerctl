package diff

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dvmrry/zscalerctl/internal/dump"
	"github.com/dvmrry/zscalerctl/internal/redact"
	"github.com/dvmrry/zscalerctl/internal/resources"
)

func TestCompareKeyedResourceNormalizesIdentityAndReportsFieldChanges(t *testing.T) {
	catalog := resources.ResourceCatalog{testKeyedSpec()}
	oldDir := writeTestDump(t, catalog, dumpFixture{
		entries: []dumpEntryFixture{
			{
				spec:    testKeyedSpec(),
				payload: `[{"id":123,"name":"old","lastModifiedTime":"2026-01-01T00:00:00Z"}]`,
			},
		},
	})
	newDir := writeTestDump(t, catalog, dumpFixture{
		entries: []dumpEntryFixture{
			{
				spec:    testKeyedSpec(),
				payload: `[{"id":"123","name":"new","lastModifiedTime":"2026-01-01T00:00:00Z"}]`,
			},
		},
	})

	report, err := Compare(oldDir, newDir, Options{Catalog: catalog, IgnoreOperational: true})
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	resource := onlyResourceDiff(t, report)
	if resource.Identity.Mode != "get_key" || resource.Identity.Field != "id" {
		t.Fatalf("identity = %+v, want get_key/id", resource.Identity)
	}
	if len(resource.Added) != 0 || len(resource.Removed) != 0 || len(resource.Changed) != 1 {
		t.Fatalf("diff counts added=%d removed=%d changed=%d, want 0/0/1", len(resource.Added), len(resource.Removed), len(resource.Changed))
	}
	if resource.Changed[0].Key != "123" {
		t.Fatalf("changed key = %q, want 123", resource.Changed[0].Key)
	}
	assertChangedFields(t, resource.Changed[0], []string{"name"})
}

func TestCompareKeyedResourceReportsRecreatedRecordAsRemoveAndAdd(t *testing.T) {
	catalog := resources.ResourceCatalog{testKeyedSpec()}
	oldDir := writeTestDump(t, catalog, dumpFixture{
		entries: []dumpEntryFixture{
			{
				spec:    testKeyedSpec(),
				payload: `[{"id":"1","name":"policy"}]`,
			},
		},
	})
	newDir := writeTestDump(t, catalog, dumpFixture{
		entries: []dumpEntryFixture{
			{
				spec:    testKeyedSpec(),
				payload: `[{"id":"2","name":"policy"}]`,
			},
		},
	})

	report, err := Compare(oldDir, newDir, Options{Catalog: catalog})
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	resource := onlyResourceDiff(t, report)
	if len(resource.Added) != 1 || len(resource.Removed) != 1 || len(resource.Changed) != 0 {
		t.Fatalf("recreated record diff counts added=%d removed=%d changed=%d, want 1/1/0", len(resource.Added), len(resource.Removed), len(resource.Changed))
	}
	if resource.Removed[0].Key != "1" || resource.Added[0].Key != "2" {
		t.Fatalf("recreated record keys removed=%+v added=%+v, want removed key 1 and added key 2", resource.Removed, resource.Added)
	}
}

func TestCompareSingletonResourceReportsChanges(t *testing.T) {
	catalog := resources.ResourceCatalog{testSingletonSpec()}
	oldDir := writeTestDump(t, catalog, dumpFixture{
		entries: []dumpEntryFixture{{spec: testSingletonSpec(), payload: `{"enabled":false}`}},
	})
	newDir := writeTestDump(t, catalog, dumpFixture{
		entries: []dumpEntryFixture{{spec: testSingletonSpec(), payload: `{"enabled":true}`}},
	})

	report, err := Compare(oldDir, newDir, Options{Catalog: catalog})
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	resource := onlyResourceDiff(t, report)
	if resource.Identity.Mode != "singleton" {
		t.Fatalf("identity mode = %q, want singleton", resource.Identity.Mode)
	}
	if len(resource.Changed) != 1 || resource.Changed[0].Key != "singleton" {
		t.Fatalf("changed = %+v, want singleton change", resource.Changed)
	}
	assertChangedFields(t, resource.Changed[0], []string{"enabled"})
}

func TestCompareContentHashResourceIgnoresOperationalOnlyChanges(t *testing.T) {
	catalog := resources.ResourceCatalog{testIdentitylessSpec()}
	oldDir := writeTestDump(t, catalog, dumpFixture{
		entries: []dumpEntryFixture{
			{
				spec:    testIdentitylessSpec(),
				payload: `[{"name":"policy","lastModifiedTime":"2026-01-01T00:00:00Z"}]`,
			},
		},
	})
	newDir := writeTestDump(t, catalog, dumpFixture{
		entries: []dumpEntryFixture{
			{
				spec:    testIdentitylessSpec(),
				payload: `[{"name":"policy","lastModifiedTime":"2026-01-02T00:00:00Z"}]`,
			},
		},
	})

	report, err := Compare(oldDir, newDir, Options{Catalog: catalog})
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	resource := onlyResourceDiff(t, report)
	if resource.Identity.Mode != "content_hash" {
		t.Fatalf("identity mode = %q, want content_hash", resource.Identity.Mode)
	}
	if resource.HasDrift() {
		t.Fatalf("content hash reported drift for operational-only change: %+v", resource)
	}
}

func TestCompareContentHashResourceReportsEditAsRemoveAndAdd(t *testing.T) {
	catalog := resources.ResourceCatalog{testIdentitylessSpec()}
	oldDir := writeTestDump(t, catalog, dumpFixture{
		entries: []dumpEntryFixture{{spec: testIdentitylessSpec(), payload: `[{"name":"old"}]`}},
	})
	newDir := writeTestDump(t, catalog, dumpFixture{
		entries: []dumpEntryFixture{{spec: testIdentitylessSpec(), payload: `[{"name":"new"}]`}},
	})

	report, err := Compare(oldDir, newDir, Options{Catalog: catalog})
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	resource := onlyResourceDiff(t, report)
	if len(resource.Added) != 1 || len(resource.Removed) != 1 || len(resource.Changed) != 0 {
		t.Fatalf("content-hash edit counts added=%d removed=%d changed=%d, want 1/1/0", len(resource.Added), len(resource.Removed), len(resource.Changed))
	}
	if resource.Added[0].Hash == "" || resource.Removed[0].Hash == "" {
		t.Fatalf("content-hash refs must include hashes: added=%+v removed=%+v", resource.Added, resource.Removed)
	}
}

func TestCompareContentHashResourceCanonicalizesObjectKeyOrder(t *testing.T) {
	catalog := resources.ResourceCatalog{testIdentitylessSpec()}
	oldDir := writeTestDump(t, catalog, dumpFixture{
		entries: []dumpEntryFixture{{spec: testIdentitylessSpec(), payload: `[{"name":"same","nested":{"a":1,"b":2}}]`}},
	})
	newDir := writeTestDump(t, catalog, dumpFixture{
		entries: []dumpEntryFixture{{spec: testIdentitylessSpec(), payload: `[{"nested":{"b":2,"a":1},"name":"same"}]`}},
	})

	report, err := Compare(oldDir, newDir, Options{Catalog: catalog})
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	resource := onlyResourceDiff(t, report)
	if resource.HasDrift() {
		t.Fatalf("content hash reported drift for key-order-only change: %+v", resource)
	}
}

func TestCompareRejectsRedactionMismatch(t *testing.T) {
	catalog := resources.ResourceCatalog{testKeyedSpec()}
	oldDir := writeTestDump(t, catalog, dumpFixture{redaction: redact.ModeStandard})
	newDir := writeTestDump(t, catalog, dumpFixture{redaction: redact.ModeShare})

	_, err := Compare(oldDir, newDir, Options{Catalog: catalog})
	if !errors.Is(err, ErrRedactionMismatch) {
		t.Fatalf("Compare() error = %v, want ErrRedactionMismatch", err)
	}
}

func TestCompareRejectsPartialDumpUnlessAllowed(t *testing.T) {
	catalog := resources.ResourceCatalog{testKeyedSpec()}
	oldDir := writeTestDump(t, catalog, dumpFixture{status: "partial"})
	newDir := writeTestDump(t, catalog, dumpFixture{})

	_, err := Compare(oldDir, newDir, Options{Catalog: catalog})
	if !errors.Is(err, ErrPartialDumpInput) {
		t.Fatalf("Compare() error = %v, want ErrPartialDumpInput", err)
	}
	if _, err := Compare(oldDir, newDir, Options{Catalog: catalog, AllowPartial: true}); err != nil {
		t.Fatalf("Compare(... AllowPartial) error = %v", err)
	}
}

func TestCompareRejectsInvalidManifestStatus(t *testing.T) {
	catalog := resources.ResourceCatalog{testKeyedSpec()}
	oldDir := writeTestDump(t, catalog, dumpFixture{status: "degraded"})
	newDir := writeTestDump(t, catalog, dumpFixture{})

	_, err := Compare(oldDir, newDir, Options{Catalog: catalog})
	if !errors.Is(err, ErrInvalidDump) {
		t.Fatalf("Compare() error = %v, want ErrInvalidDump", err)
	}
	if !strings.Contains(err.Error(), "invalid manifest status") {
		t.Fatalf("Compare() error = %v, want invalid manifest status context", err)
	}
}

func TestCompareRejectsOversizedManifest(t *testing.T) {
	catalog := resources.ResourceCatalog{testKeyedSpec()}
	oldDir := writeTestDump(t, catalog, dumpFixture{})
	newDir := writeTestDump(t, catalog, dumpFixture{})
	truncateTestFile(t, filepath.Join(oldDir, "manifest.json"), maxManifestBytes+1)

	_, err := Compare(oldDir, newDir, Options{Catalog: catalog})
	if !errors.Is(err, ErrInvalidDump) {
		t.Fatalf("Compare() error = %v, want ErrInvalidDump", err)
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Fatalf("Compare() error = %v, want too large context", err)
	}
}

func TestCompareRejectsOversizedResource(t *testing.T) {
	catalog := resources.ResourceCatalog{testKeyedSpec()}
	oldDir := writeTestDump(t, catalog, dumpFixture{
		entries: []dumpEntryFixture{{spec: testKeyedSpec(), payload: `[{"id":"1","name":"old"}]`}},
	})
	newDir := writeTestDump(t, catalog, dumpFixture{
		entries: []dumpEntryFixture{{spec: testKeyedSpec(), payload: `[{"id":"1","name":"old"}]`}},
	})
	truncateTestFile(t, filepath.Join(oldDir, "resources", "zia", "rules.json"), maxResourceBytes+1)

	_, err := Compare(oldDir, newDir, Options{Catalog: catalog})
	if !errors.Is(err, ErrInvalidDump) {
		t.Fatalf("Compare() error = %v, want ErrInvalidDump", err)
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Fatalf("Compare() error = %v, want too large context", err)
	}
}

func TestCompareIgnoreOperationalSuppressesKeyedOperationalChanges(t *testing.T) {
	catalog := resources.ResourceCatalog{testKeyedSpec()}
	oldDir := writeTestDump(t, catalog, dumpFixture{
		entries: []dumpEntryFixture{{spec: testKeyedSpec(), payload: `[{"id":"1","name":"same","lastModifiedTime":"old"}]`}},
	})
	newDir := writeTestDump(t, catalog, dumpFixture{
		entries: []dumpEntryFixture{{spec: testKeyedSpec(), payload: `[{"id":"1","name":"same","lastModifiedTime":"new"}]`}},
	})

	report, err := Compare(oldDir, newDir, Options{Catalog: catalog})
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if !onlyResourceDiff(t, report).HasDrift() {
		t.Fatalf("Compare() without IgnoreOperational did not report operational drift")
	}
	report, err = Compare(oldDir, newDir, Options{Catalog: catalog, IgnoreOperational: true})
	if err != nil {
		t.Fatalf("Compare(... IgnoreOperational) error = %v", err)
	}
	if onlyResourceDiff(t, report).HasDrift() {
		t.Fatalf("Compare(... IgnoreOperational) reported operational drift: %+v", report.Resources[0])
	}
}

type dumpFixture struct {
	redaction redact.Mode
	status    string
	entries   []dumpEntryFixture
}

type dumpEntryFixture struct {
	spec    resources.ResourceSpec
	payload string
}

func writeTestDump(t *testing.T, catalog resources.ResourceCatalog, fixture dumpFixture) string {
	t.Helper()
	dir := t.TempDir()
	mode := fixture.redaction
	if mode == "" {
		mode = redact.ModeStandard
	}
	status := fixture.status
	if status == "" {
		status = "complete"
	}
	manifest := dump.Manifest{
		Schema:      dump.ManifestSchemaID,
		CollectedAt: "2026-01-01T00:00:00Z",
		ToolVersion: "test",
		Redaction:   string(mode),
		Warning:     "test fixture",
		Status:      status,
	}
	for _, entry := range fixture.entries {
		relPath := filepath.ToSlash(filepath.Join("resources", string(entry.spec.Product), entry.spec.Name+".json"))
		path := filepath.Join(dir, relPath)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatalf("MkdirAll(%s): %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(entry.payload), 0o600); err != nil {
			t.Fatalf("WriteFile(%s): %v", path, err)
		}
		manifest.Resources = append(manifest.Resources, dump.ManifestResource{
			Product: string(entry.spec.Product),
			Name:    entry.spec.Name,
			Shape:   string(entry.spec.EffectiveShape()),
			Status:  "ok",
			Path:    relPath,
			Records: countRecords(t, entry.payload),
		})
	}
	if status == "partial" {
		manifest.Errors = 1
		manifest.Resources = append(manifest.Resources, dump.ManifestResource{
			Product:   string(catalog[0].Product),
			Name:      catalog[0].Name,
			Status:    "error",
			Operation: "list",
			ErrorKind: "live_access_failed",
		})
	}
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent(manifest): %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), body, 0o600); err != nil {
		t.Fatalf("WriteFile(manifest): %v", err)
	}
	return dir
}

func truncateTestFile(t *testing.T, path string, size int64) {
	t.Helper()
	if err := os.Truncate(path, size); err != nil {
		t.Fatalf("Truncate(%s, %d): %v", path, size, err)
	}
}

func countRecords(t *testing.T, payload string) int {
	t.Helper()
	var raw any
	if err := json.Unmarshal([]byte(payload), &raw); err != nil {
		t.Fatalf("Unmarshal(%s): %v", payload, err)
	}
	switch value := raw.(type) {
	case []any:
		return len(value)
	case map[string]any:
		return 1
	default:
		t.Fatalf("payload %s is not an object or array", payload)
		return 0
	}
}

func onlyResourceDiff(t *testing.T, report Report) ResourceDiff {
	t.Helper()
	if len(report.Resources) != 1 {
		t.Fatalf("len(report.Resources) = %d, want 1; report=%+v", len(report.Resources), report)
	}
	return report.Resources[0]
}

func assertChangedFields(t *testing.T, change RecordChange, want []string) {
	t.Helper()
	if len(change.Changes) != len(want) {
		t.Fatalf("changed fields = %+v, want %v", change.Changes, want)
	}
	for i, field := range want {
		if change.Changes[i].Field != field {
			t.Fatalf("changed field %d = %q, want %q; changes=%+v", i, change.Changes[i].Field, field, change.Changes)
		}
	}
}

func testKeyedSpec() resources.ResourceSpec {
	return resources.ResourceSpec{
		Product:    resources.ProductZIA,
		Name:       "rules",
		Operations: resources.ReadOperations(),
		Fields: []resources.FieldSpec{
			{Name: "id", Classification: resources.ClassOperational},
			{Name: "name", Classification: resources.ClassTenantConfig},
			{Name: "lastModifiedTime", Classification: resources.ClassOperational},
		},
	}
}

func testSingletonSpec() resources.ResourceSpec {
	return resources.ResourceSpec{
		Product:    resources.ProductZIA,
		Name:       "advanced-settings",
		Shape:      resources.ShapeSingleton,
		Operations: resources.ShowOperation(),
		Fields: []resources.FieldSpec{
			{Name: "enabled", Classification: resources.ClassTenantConfig},
		},
	}
}

func testIdentitylessSpec() resources.ResourceSpec {
	return resources.ResourceSpec{
		Product:    resources.ProductZIA,
		Name:       "cloud-app-control",
		Operations: resources.ListOperations(),
		Fields: []resources.FieldSpec{
			{Name: "name", Classification: resources.ClassTenantConfig},
			{Name: "nested", Classification: resources.ClassTenantConfig},
			{Name: "lastModifiedTime", Classification: resources.ClassOperational},
		},
	}
}
