package runtime

import (
	"context"
	"errors"
	"math"
	"testing"

	"zscalerctl/internal/redact"
	"zscalerctl/internal/resources"
)

type stubReportReader struct{ records []resources.SourceRecord }

func (stubReportReader) List(context.Context, resources.Product, string) ([]resources.SourceRecord, error) {
	return nil, nil
}

func (stubReportReader) Get(context.Context, resources.Product, string, string) (resources.SourceRecord, error) {
	return resources.SourceRecord{}, nil
}

func (stubReportReader) Show(context.Context, resources.Product, string) (resources.SourceRecord, error) {
	return resources.SourceRecord{}, nil
}

func (r stubReportReader) Report(context.Context, resources.Product, string, string, ReportQuery) ([]resources.SourceRecord, error) {
	return r.records, nil
}

// TestReadReportVerifiesTheProjection: reports go through the verified
// projection, so a value the allow-list check refuses fails the read
// instead of reaching a renderer that might drop or stringify it.
func TestReadReportVerifiesTheProjection(t *testing.T) {
	t.Parallel()

	spec, ok := resources.FindSpec(resources.ProductZDX, "apps")
	if !ok {
		t.Fatal("FindSpec(zdx, apps) ok = false")
	}
	reader := stubReportReader{records: []resources.SourceRecord{
		resources.NewSourceRecord(map[string]any{"id": 7, "name": "app-name", "score": math.NaN()}),
	}}
	_, err := ReadReportFromReader(context.Background(), reader, spec, redact.ModeStandard, "", ReportQuery{})
	if !errors.Is(err, resources.ErrInvalidProjectedValue) {
		t.Fatalf("ReadReportFromReader(NaN score) error = %v, want ErrInvalidProjectedValue", err)
	}
}
