package diff

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"zscalerctl/internal/dump"
	"zscalerctl/internal/redact"
	"zscalerctl/internal/resources"
)

const (
	benchmarkSelectedRecords   = 64
	benchmarkUnselectedRecords = 10_000
)

func BenchmarkCompareContextNarrowSelection(b *testing.B) {
	selected := benchmarkDiffSpec("selected-locations")
	unselected := benchmarkDiffSpec("unselected-locations")
	catalog := resources.ResourceCatalog{selected, unselected}
	selectedPayload := benchmarkDiffPayload(benchmarkSelectedRecords)
	unselectedPayload := benchmarkDiffPayload(benchmarkUnselectedRecords)
	dir := writeBenchmarkDiffDump(b, []benchmarkDiffEntry{
		{spec: selected, payload: selectedPayload, records: benchmarkSelectedRecords},
		{spec: unselected, payload: unselectedPayload, records: benchmarkUnselectedRecords},
	})
	opts := Options{
		Catalog: catalog,
		Resources: map[ResourceKey]bool{
			{Product: selected.Product, Name: selected.Name}: true,
		},
	}

	b.ReportAllocs()
	b.SetBytes(2 * int64(len(selectedPayload)+len(unselectedPayload)))
	b.ResetTimer()
	var retained Report
	for b.Loop() {
		var err error
		retained, err = CompareContext(context.Background(), dir, dir, opts, nil)
		if err != nil {
			b.Fatalf("CompareContext(narrow selection) error = %v, want nil", err)
		}
	}
	b.ReportMetric(float64(benchmarkUnselectedRecords), "unselected-records/op")
	runtime.KeepAlive(retained)
}

func BenchmarkUnselectedResourceParsing(b *testing.B) {
	for _, records := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("records-%d", records), func(b *testing.B) {
			benchmarkUnselectedResourceParsing(b, records)
		})
	}
}

func benchmarkUnselectedResourceParsing(b *testing.B, records int) {
	b.Helper()
	spec := benchmarkDiffSpec("unselected-locations")
	payload := benchmarkDiffPayload(records)
	dir := writeBenchmarkDiffDump(b, []benchmarkDiffEntry{{
		spec: spec, payload: payload, records: records,
	}})
	root, err := os.OpenRoot(dir)
	if err != nil {
		b.Fatalf("os.OpenRoot(%q) error = %v, want nil", dir, err)
	}
	b.Cleanup(func() {
		if err := root.Close(); err != nil {
			b.Errorf("os.Root.Close() error = %v, want nil", err)
		}
	})
	mr := dump.ManifestResource{
		Product: string(spec.Product),
		Name:    spec.Name,
		Status:  "ok",
		Path:    filepath.ToSlash(filepath.Join("resources", string(spec.Product), spec.Name+".json")),
		Records: records,
	}

	b.Run("full-decode", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(payload)))
		var retained []map[string]any
		for b.Loop() {
			var readErr error
			retained, readErr = readResource(context.Background(), root, mr, spec)
			if readErr != nil {
				b.Fatalf("readResource(%d records) error = %v, want nil", records, readErr)
			}
		}
		runtime.KeepAlive(retained)
	})

	b.Run("stream-count", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(payload)))
		var retained int
		for b.Loop() {
			var readErr error
			retained, readErr = readUnselectedResourceCount(context.Background(), root, mr, spec)
			if readErr != nil {
				b.Fatalf("readUnselectedResourceCount(%d records) error = %v, want nil", records, readErr)
			}
		}
		runtime.KeepAlive(retained)
	})
}

type benchmarkDiffEntry struct {
	spec    resources.ResourceSpec
	payload []byte
	records int
}

// writeBenchmarkDiffDump writes a valid dump through the same helper the diff
// tests use, so the benchmark fixture keeps up with artifact admission.
func writeBenchmarkDiffDump(b *testing.B, entries []benchmarkDiffEntry) string {
	b.Helper()
	catalog := make(resources.ResourceCatalog, 0, len(entries))
	fixture := dumpFixture{}
	for _, entry := range entries {
		catalog = append(catalog, entry.spec)
		fixture.entries = append(fixture.entries, dumpEntryFixture{spec: entry.spec, payload: string(entry.payload)})
	}
	return writeTestDump(b, catalog, fixture)
}

func benchmarkDiffPayload(records int) []byte {
	var body bytes.Buffer
	body.Grow(records * 160)
	body.WriteByte('[')
	for i := range records {
		if i > 0 {
			body.WriteByte(',')
		}
		fmt.Fprintf(
			&body,
			`{"id":%d,"name":"branch-%06d","enabled":true,"ports":[80,443,8443],"metadata":{"region":"us-east","labels":["branch","managed","benchmark"]}}`,
			i,
			i,
		)
	}
	body.WriteByte(']')
	return body.Bytes()
}

func benchmarkDiffSpec(name string) resources.ResourceSpec {
	standard := []redact.Mode{redact.ModeStandard}
	return resources.ResourceSpec{
		Product:    resources.ProductZIA,
		Name:       name,
		Operations: resources.ReadOperations(),
		Fields: []resources.FieldSpec{
			{Name: "id", Classification: resources.ClassOperational, AllowedModes: standard},
			{Name: "name", Classification: resources.ClassTenantConfig, AllowedModes: standard},
			{Name: "enabled", Classification: resources.ClassTenantConfig, AllowedModes: standard},
			{Name: "ports", Classification: resources.ClassOperational, AllowedModes: standard},
			{
				Name:           "metadata",
				Classification: resources.ClassOperational,
				AllowedModes:   standard,
				Fields: []resources.FieldSpec{
					{Name: "region", Classification: resources.ClassOperational, AllowedModes: standard},
					{Name: "labels", Classification: resources.ClassOperational, AllowedModes: standard},
				},
			},
		},
	}
}
