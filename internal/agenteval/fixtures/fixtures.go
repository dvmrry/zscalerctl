// Package fixtures holds the value-free, synthetic resource corpus and a
// fixtureReader that serves it through the production
// cli.ResourceReader seam (see docs/AGENTIC_COVERAGE_PLAN.md §2.3, §3.5).
//
// This is the first slice of the agenteval fixture binary's data source. It is
// a NON-test (importable) package on purpose: the fixture main
// (internal/agenteval/cmd/zscalerctl-fixture) and the oracle consume it, and a
// `package main` cannot import the `_test.go` corpus that lives in
// internal/livesmoke/fake_runner_test.go. The corpus is therefore promoted
// here, not copied into a test file.
//
// Every value in this corpus is synthetic and value-free:
//
//   - names are obviously fake ("HQ", "Branch office") ;
//   - IP addresses come from the RFC 5737 documentation ranges
//     (192.0.2.0/24, 203.0.113.0/24) ;
//   - any hostname uses an RFC 2606 *.example.internal label ;
//   - the single secret-classified field (preSharedKey) carries an obviously
//     synthetic canary placeholder, NEVER a realistic key. Its presence lets a
//     scenario assert that a ClassSecret field is dropped by projection without
//     risking a real leak.
//
// The data source is the ONLY thing the fixture binary swaps; everything past
// the reader (projection, redaction, --fields, exit codes, the error envelope)
// is the exact production path. That is what keeps the eval honest, so the
// corpus must look like real records without carrying real values.
package fixtures

import (
	"context"
	"fmt"

	"github.com/dvmrry/zscalerctl/internal/cli"
	"github.com/dvmrry/zscalerctl/internal/resources"
	"github.com/dvmrry/zscalerctl/internal/zscaler"
)

// fixtureReader must satisfy the real production seam, not a local copy of it.
// This compile-time assertion fails the build if cli.ResourceReader's signature
// ever drifts from what fixtureReader implements.
var _ cli.ResourceReader = (*fixtureReader)(nil)

// SecretCanary is the obviously-synthetic placeholder stored in the one
// secret-classified field (zia/locations preSharedKey) in the corpus. It is NOT
// a realistic key: it is a fixed sentinel a scenario can assert never appears in
// projected output (ClassSecret fields are always dropped). Keeping it a named
// constant means the test and any future canary-leak assertion reference the
// same literal.
const SecretCanary = "CANARY-secret-preSharedKey"

// corpusRecord is the in-memory shape of one fixture record: its string id (the
// key get <id> resolves against) and the raw field map handed to
// resources.NewSourceRecord. The id is also present inside fields under "id";
// it is duplicated here only so Get can index without re-deriving the key.
type corpusRecord struct {
	id     string
	fields map[string]any
}

// corpusKey identifies a resource bucket by (product, resource-name) — the same
// pair the cli.ResourceReader keys List/Get/Show on. The resource name is the
// catalog Name (e.g. "locations"), not a product-qualified slug.
type corpusKey struct {
	product  resources.Product
	resource string
}

// corpus is the whole value-free dataset, keyed by (product, resource). Each
// bucket is an ordered slice so List output order is stable across runs. Field
// names match the resources catalog JSON field names (see catalog_zia.go) so
// projection sees exactly the fields it classifies.
var corpus = map[corpusKey][]corpusRecord{
	{resources.ProductZIA, "locations"}: {
		{
			id: "1",
			// One record carries the secret-classified preSharedKey with the
			// synthetic canary, so a scenario can prove projection drops it.
			fields: map[string]any{
				"id":           1,
				"name":         "HQ",
				"description":  "Example HQ location",
				"ipAddresses":  []any{"192.0.2.10"},
				"country":      "COUNTRY_NONE",
				"preSharedKey": SecretCanary,
			},
		},
		{
			id: "2",
			fields: map[string]any{
				"id":          2,
				"name":        "Branch office",
				"description": "Example branch location",
				"ipAddresses": []any{"203.0.113.20"},
				"country":     "COUNTRY_NONE",
			},
		},
	},
}

// fixtureReader serves the in-memory corpus through the production
// cli.ResourceReader contract. It dials no network, reads no clock, and holds no
// credentials: it is pure data. Construction mirrors the livesmoke fake — every
// returned record is built with resources.NewSourceRecord(map[string]any{...}),
// the only public constructor for the opaque SourceRecord.
//
// The compile-time assertion below pins fixtureReader to the real interface; if
// the cli.ResourceReader signature ever drifts, this package stops compiling.
type fixtureReader struct{}

// NewReader returns a fixtureReader serving the value-free corpus, typed as the
// production cli.ResourceReader seam so the fixture binary
// (internal/agenteval/cmd/zscalerctl-fixture) can hand it straight to
// cli.Options.Reader. The concrete type stays unexported; callers only ever see
// the interface.
func NewReader() cli.ResourceReader {
	return &fixtureReader{}
}

// List returns every record for (product, name) as []resources.SourceRecord. An
// unknown (product, name) bucket is an unsupported resource, mirroring the real
// reader's zscaler.ErrUnsupportedResource path (exit 4).
func (fixtureReader) List(_ context.Context, product resources.Product, name string) ([]resources.SourceRecord, error) {
	records, ok := corpus[corpusKey{product: product, resource: name}]
	if !ok {
		return nil, fmt.Errorf("%w: %s/%s", zscaler.ErrUnsupportedResource, product, name)
	}
	out := make([]resources.SourceRecord, 0, len(records))
	for _, rec := range records {
		out = append(out, resources.NewSourceRecord(rec.fields))
	}
	return out, nil
}

// Get returns the single record whose id matches. The signature mirrors the
// real reader exactly: (ctx, product, name, id). An unknown (product, name) is
// unsupported_resource; a known resource with an unknown id returns the
// not-found sentinel (errors.Is(err, zscaler.ErrResourceNotFound) == true),
// which the CLI maps to exit 4 + the not_found error envelope.
func (fixtureReader) Get(_ context.Context, product resources.Product, name string, id string) (resources.SourceRecord, error) {
	records, ok := corpus[corpusKey{product: product, resource: name}]
	if !ok {
		return resources.SourceRecord{}, fmt.Errorf("%w: %s/%s", zscaler.ErrUnsupportedResource, product, name)
	}
	for _, rec := range records {
		if rec.id == id {
			return resources.NewSourceRecord(rec.fields), nil
		}
	}
	return resources.SourceRecord{}, fmt.Errorf("%w: %s/%s id %s", zscaler.ErrResourceNotFound, product, name, id)
}

// Show returns the singleton record for (product, name). For a bucket with one
// record it returns that record directly; for a multi-record bucket it returns
// the first. An unknown bucket is unsupported_resource. List-shaped resources
// like zia/locations are not singletons, so Show on them is unusual — it is
// implemented for interface completeness and returns the first record.
func (fixtureReader) Show(_ context.Context, product resources.Product, name string) (resources.SourceRecord, error) {
	records, ok := corpus[corpusKey{product: product, resource: name}]
	if !ok || len(records) == 0 {
		return resources.SourceRecord{}, fmt.Errorf("%w: %s/%s", zscaler.ErrUnsupportedResource, product, name)
	}
	return resources.NewSourceRecord(records[0].fields), nil
}
