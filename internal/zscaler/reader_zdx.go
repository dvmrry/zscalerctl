package zscaler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"time"

	zsdk "github.com/zscaler/zscaler-sdk-go/v3/zscaler"
	zdxadmin "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zdx/services/administration"
	zdxalerts "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zdx/services/alerts"
	zdxcommon "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zdx/services/common"
	zdxapps "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zdx/services/reports/applications"

	"zscalerctl/internal/resources"
)

// ReportQuery is one report read's resolved window and filters.
// From and To are absolute; the caller
// resolves --since before the read.
type ReportQuery struct {
	From        time.Time
	To          time.Time
	Departments []int
	Locations   []int
}

// ReportReader reads time-windowed report resources.
type ReportReader interface {
	Report(ctx context.Context, product resources.Product, resource, id string, query ReportQuery) ([]resources.SourceRecord, error)
}

var _ ReportReader = (*SDKReader)(nil)

// zdxAlertPageLimit bounds the alert cursor walk; ZDX allows 100 requests an
// hour, so a walk that needs more pages fails instead of spending the hour.
const zdxAlertPageLimit = 20

var errZDXRepeatedOffset = errors.New("read zdx alerts: the cursor repeated")

// ReportPolicy is what a report resource accepts: an id (one record's
// report) and department/location filters. Callers check it before
// building any client.
type ReportPolicy struct {
	AllowsID      bool
	AcceptsFilter bool
}

// ReportPolicyFor returns resource's policy, or false for no report.
func ReportPolicyFor(product resources.Product, resource string) (ReportPolicy, bool) {
	handler, ok := zdxReportHandlers[resource]
	if product != resources.ProductZDX || !ok {
		return ReportPolicy{}, false
	}
	return ReportPolicy{AllowsID: handler.allowsID, AcceptsFilter: handler.filters}, true
}

type zdxReportHandler struct {
	allowsID bool
	filters  bool
	read     func(ctx context.Context, service *zsdk.Service, id string, filters zdxcommon.GetFromToFilters) ([]resources.SourceRecord, error)
}

var zdxReportHandlers = map[string]zdxReportHandler{
	"apps":          {allowsID: true, filters: true, read: readZDXApps},
	"alerts":        {filters: true, read: zdxAlertReader(zdxalerts.GetOngoingAlerts)},
	"alert-history": {filters: true, read: zdxAlertReader(zdxalerts.GetHistoricalAlerts)},
	"departments":   {read: readZDXDepartments},
	"locations":     {read: readZDXLocations},
}

// Report reads one ZDX report resource over query's window. It makes one
// request per read (three for one app; at most zdxAlertPageLimit for alert
// pages) and never fans out per device or user.
func (r *SDKReader) Report(ctx context.Context, product resources.Product, resource, id string, query ReportQuery) ([]resources.SourceRecord, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: %s/%s", ErrUnsupportedResource, product, resource)
	}
	handler, ok := zdxReportHandlers[resource]
	if product != resources.ProductZDX || !ok {
		return nil, fmt.Errorf("%w: %s/%s", ErrUnsupportedResource, product, resource)
	}
	if id != "" && !handler.allowsID {
		return nil, fmt.Errorf("%w: %s/%s report takes no id", ErrInvalidResourceID, product, resource)
	}
	if !handler.filters && (len(query.Departments) > 0 || len(query.Locations) > 0) {
		return nil, fmt.Errorf("%w: %s/%s report takes no department or location filter", ErrInvalidResourceID, product, resource)
	}
	if query.From.Unix() <= 0 || query.To.IsZero() || !query.From.Before(query.To) {
		return nil, fmt.Errorf("%w: %s/%s report needs a window with from before to", ErrInvalidResourceID, product, resource)
	}
	service, cleanup, err := perCallService{cfg: r.cfg}.service(ctx, product)
	if err != nil {
		if isSetupError(err) {
			return nil, err
		}
		return nil, normalizeLiveError(ctx, "authenticate", product, resource, err)
	}
	defer cleanup()
	filters := zdxcommon.GetFromToFilters{
		From: int(query.From.Unix()),
		To:   int(query.To.Unix()),
		Dept: append([]int(nil), query.Departments...),
		Loc:  append([]int(nil), query.Locations...),
	}
	records, err := handler.read(ctx, service, id, filters)
	if err != nil {
		return nil, zdxReportError(ctx, product, resource, id, err)
	}
	return records, nil
}

// zdxReportError classifies a failed report read. One record's report for an
// id the API does not know is a missing record (exit 4), as for get; a
// sign-in 404 is a wrong host and a cancelled context is cancellation, so
// both stay live failures.
func zdxReportError(ctx context.Context, product resources.Product, resource, id string, err error) error {
	if errors.Is(err, ErrInvalidResourceID) {
		return err
	}
	if id != "" && effectiveContext(ctx).Err() == nil && sdkStatusCode(err) == http.StatusNotFound &&
		!errors.As(err, new(*signInError)) {
		return resourceNotFoundError{product: product, resource: resource}
	}
	return normalizeLiveError(ctx, "report", product, resource, err)
}

func readZDXApps(ctx context.Context, service *zsdk.Service, id string, filters zdxcommon.GetFromToFilters) ([]resources.SourceRecord, error) {
	if id == "" {
		apps, _, err := zdxapps.GetAllApps(ctx, service, filters)
		if err != nil {
			return nil, err
		}
		return structSourceRecords(apps), nil
	}
	appID, err := parsePositiveIntID(id)
	if err != nil {
		return nil, err
	}
	app, _, err := zdxapps.GetApp(ctx, service, strconv.Itoa(appID), filters)
	if err != nil {
		return nil, err
	}
	scores, _, err := zdxapps.GetAppScores(ctx, service, appID, filters)
	if err != nil {
		return nil, err
	}
	metrics, _, err := zdxapps.GetAppMetrics(ctx, service, appID, filters)
	if err != nil {
		return nil, err
	}
	fields := structMap(reflect.ValueOf(app))
	if fields == nil || app.ID != appID {
		return nil, errors.New("read zdx app: the response is not that app")
	}
	fields["score_trend"] = sourceValue(reflect.ValueOf(scores))
	fields["metrics"] = sourceValue(reflect.ValueOf(metrics))
	return []resources.SourceRecord{resources.NewSourceRecord(fields)}, nil
}

func zdxAlertReader(
	page func(context.Context, *zsdk.Service, zdxcommon.GetFromToFilters) (*zdxalerts.AlertsResponse, *http.Response, error),
) func(context.Context, *zsdk.Service, string, zdxcommon.GetFromToFilters) ([]resources.SourceRecord, error) {
	return func(ctx context.Context, service *zsdk.Service, _ string, filters zdxcommon.GetFromToFilters) ([]resources.SourceRecord, error) {
		var alerts []zdxalerts.Alert
		seen := map[string]bool{}
		for range zdxAlertPageLimit {
			response, _, err := page(ctx, service, filters)
			if err != nil {
				return nil, err
			}
			if response == nil {
				return nil, errors.New("read zdx alerts: empty response")
			}
			alerts = append(alerts, response.Alerts...)
			if response.NextOffset == "" {
				return structSourceRecords(alerts), nil
			}
			if seen[response.NextOffset] {
				return nil, errZDXRepeatedOffset
			}
			seen[response.NextOffset] = true
			filters.Offset = response.NextOffset
		}
		return nil, fmt.Errorf("read zdx alerts: more than %d pages", zdxAlertPageLimit)
	}
}

func readZDXDepartments(ctx context.Context, service *zsdk.Service, _ string, filters zdxcommon.GetFromToFilters) ([]resources.SourceRecord, error) {
	departments, _, err := zdxadmin.GetDepartments(ctx, service, zdxadmin.GetDepartmentsFilters{From: filters.From, To: filters.To})
	if err != nil {
		return nil, err
	}
	return structSourceRecords(departments), nil
}

func readZDXLocations(ctx context.Context, service *zsdk.Service, _ string, filters zdxcommon.GetFromToFilters) ([]resources.SourceRecord, error) {
	locations, _, err := zdxadmin.GetLocations(ctx, service, zdxadmin.GetLocationsFilters{From: filters.From, To: filters.To})
	if err != nil {
		return nil, err
	}
	return structSourceRecords(locations), nil
}

func structSourceRecords[T any](items []T) []resources.SourceRecord {
	records := make([]resources.SourceRecord, 0, len(items))
	for _, item := range items {
		records = append(records, structSourceRecord(item))
	}
	return records
}
