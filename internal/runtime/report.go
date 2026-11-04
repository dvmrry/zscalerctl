package runtime

import (
	"context"
	"fmt"

	"zscalerctl/internal/browser"
	"zscalerctl/internal/redact"
	"zscalerctl/internal/resources"
	"zscalerctl/internal/zscaler"
)

// ReportQuery is a report read's resolved window and filters.
type ReportQuery = zscaler.ReportQuery

// ReportPolicy is what a report resource accepts (an id, filters).
type ReportPolicy = zscaler.ReportPolicy

// ReportPolicyFor returns a report resource's static policy.
func ReportPolicyFor(product resources.Product, resource string) (ReportPolicy, bool) {
	return zscaler.ReportPolicyFor(product, resource)
}

// ReadReportFromReader reads one report resource through a trusted reader
// and projects it in mode. The spec must be a report resource; the window in
// query is the one the caller echoes in output.
func ReadReportFromReader(
	ctx context.Context,
	reader browser.RecordReader,
	spec resources.ResourceSpec,
	mode redact.Mode,
	id string,
	query ReportQuery,
) (resources.ProjectedRecords, error) {
	ctx = nonNilContext(ctx)
	if err := ctx.Err(); err != nil {
		return resources.ProjectedRecords{}, err
	}
	if reader == nil {
		return resources.ProjectedRecords{}, browser.ErrMissingReader
	}
	if spec.EffectiveShape() != resources.ShapeReport {
		return resources.ProjectedRecords{}, fmt.Errorf("%w: %s/%s is not a report resource", zscaler.ErrUnsupportedResource, spec.Product, spec.Name)
	}
	reportReader, ok := reader.(zscaler.ReportReader)
	if !ok {
		return resources.ProjectedRecords{}, fmt.Errorf("%w: %s/%s", zscaler.ErrUnsupportedResource, spec.Product, spec.Name)
	}
	records, err := reportReader.Report(ctx, spec.Product, spec.Name, id, query)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return resources.ProjectedRecords{}, ctxErr
	}
	if err != nil {
		return resources.ProjectedRecords{}, err
	}
	projected, _, err := resources.ProjectRecordsAndVerify(spec, redact.EffectiveMode(mode), records)
	if err != nil {
		return resources.ProjectedRecords{}, err
	}
	return projected, nil
}
