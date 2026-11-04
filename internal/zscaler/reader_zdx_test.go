package zscaler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	zsdk "github.com/zscaler/zscaler-sdk-go/v3/zscaler"
	sdkerrorx "github.com/zscaler/zscaler-sdk-go/v3/zscaler/errorx"
	zdxcommon "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zdx/services/common"

	"zscalerctl/internal/redact"
	"zscalerctl/internal/resources"
)

func zdxRoutes() map[string]string {
	alert := func(id, offset string) string {
		return `{"alerts":[{"id":` + id + `,"rule_name":"rule-name-` + id + `","severity":"high","num_devices":0,` +
			`"application":{"id":3,"name":"alert-app"},"departments":[{"id":4,"name":"dept-name","num_devices":2}],` +
			`"geolocations":[{"id":"geo-id-canary","name":"geo-city-canary","num_devices":1}]}],"next_offset":"` + offset + `"}`
	}
	return map[string]string{
		"/zdx/v1/apps": `[{"id":7,"name":"app-name","score":87.3,"total_users":12,` +
			`"most_impacted_region":{"id":"region-id-canary","city":"city-canary","region":"region-name","country":"country-name"},` +
			`"stats":{"active_users":5,"active_devices":6,"num_poor":0,"num_okay":1,"num_good":4}}]`,
		"/zdx/v1/apps/7":                       `{"id":7,"name":"app-name","score":90}`,
		"/zdx/v1/apps/7/score":                 `[{"metric":"score","unit":"","datapoints":[{"timestamp":1761472800,"value":0},{"timestamp":1761476400,"value":91.5}]}]`,
		"/zdx/v1/apps/7/metrics":               `[{"metric":"pft","unit":"ms","datapoints":[{"timestamp":1761472800,"value":420}]}]`,
		"/zdx/v1/alerts/ongoing":               alert("1", "page-2"),
		"/zdx/v1/alerts/ongoing?offset=page-2": alert("2", ""),
		"/zdx/v1/alerts/historical":            alert("9", ""),
		"/zdx/v1/administration/departments":   `[{"id":4,"name":"dept-name"}]`,
		"/zdx/v1/administration/locations":     `[{"id":5,"name":"location-name"}]`,
	}
}

// newZDXTestService answers ZDX routes by path plus, when present, the
// cursor offset, and records each request's query.
func newZDXTestService(t *testing.T, routes map[string]string, queries map[string][]string) *zsdk.Service {
	t.Helper()
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body, status := `{"access_token":"test-token","expires_in":60}`, http.StatusOK
		key := request.URL.Path
		if offset := request.URL.Query().Get("offset"); offset != "" {
			key += "?offset=" + offset
		}
		if routeBody, ok := routes[key]; ok {
			queries[request.URL.Path] = append(queries[request.URL.Path], request.URL.RawQuery)
			body = routeBody
		} else if request.URL.Path != "/oauth2/v1/token" {
			status, body = http.StatusNotFound, `{}`
		}
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: request}, nil
	})
	cfg := newSDKConfiguration(context.Background(), validReaderConfig())
	for _, client := range []*http.Client{cfg.HTTPClient, cfg.ZDXHTTPClient} {
		client.Transport = transport
	}
	service, err := zsdk.NewOneAPIClient(cfg)
	if err != nil {
		t.Fatalf("NewOneAPIClient() error = %v", err)
	}
	t.Cleanup(service.Client.Close)
	return service
}

// TestZDXReportHandlersReadTheirRoutesAndRedactPerMode: each report reads its
// route with the window and filters, alerts follow the cursor, one app adds
// its score and metric series, zeros render, and geographic identifiers are
// standard only.
func TestZDXReportHandlersReadTheirRoutesAndRedactPerMode(t *testing.T) {
	t.Parallel()

	queries := map[string][]string{}
	service := newZDXTestService(t, zdxRoutes(), queries)
	filters := zdxcommon.GetFromToFilters{From: 1761472800, To: 1761480000, Dept: []int{4}, Loc: []int{5}}
	cases := []struct {
		name, id     string
		records      int
		standardOnly []string
		shared       []string
		everywhere   []string
	}{
		{name: "apps", records: 1, standardOnly: []string{"region-id-canary", "city-canary"},
			shared: []string{"app-name", "region-name", "country-name"}, everywhere: []string{`"score":87.3,`, `"num_poor":0`, `"total_users":12`}},
		{name: "apps", id: "7", records: 1, shared: []string{"app-name"},
			everywhere: []string{`"score":90`, `"value":0`, `"value":91.5`, `"metric":"pft"`, `"value":420`}},
		{name: "alerts", records: 2, standardOnly: []string{"geo-id-canary", "geo-city-canary"},
			shared: []string{"rule-name-1", "rule-name-2", "alert-app", "dept-name"}, everywhere: []string{`"severity":"high"`, `"num_devices":0`}},
		{name: "alert-history", records: 1, shared: []string{"rule-name-9"}},
		{name: "departments", records: 1, shared: []string{"dept-name"}},
		{name: "locations", records: 1, shared: []string{"location-name"}},
	}
	for _, c := range cases {
		records, err := zdxReportHandlers[c.name].read(context.Background(), service, c.id, filters)
		if err != nil {
			t.Fatalf("zdx/%s report(%q) error = %v", c.name, c.id, err)
		}
		if len(records) != c.records {
			t.Fatalf("zdx/%s report(%q) records = %d, want %d", c.name, c.id, len(records), c.records)
		}
		spec, ok := resources.FindSpec(resources.ProductZDX, c.name)
		if !ok {
			t.Fatalf("FindSpec(zdx, %s) ok = false", c.name)
		}
		for _, mode := range []redact.Mode{redact.ModeStandard, redact.ModeShare, redact.ModeParanoid} {
			projected, _, err := resources.ProjectRecords(spec, mode, records)
			if err != nil {
				t.Fatalf("ProjectRecords(zdx/%s, %s) error = %v", c.name, mode, err)
			}
			var all []map[string]any
			for _, record := range projected.Records() {
				if err := resources.AssertRenderedSubset(spec, mode, record.Fields()); err != nil {
					t.Errorf("AssertRenderedSubset(zdx/%s, %s) error = %v", c.name, mode, err)
				}
				all = append(all, record.Fields())
			}
			raw, _ := json.Marshal(all)
			rendered := string(raw)
			check := func(canaries []string, want bool) {
				for _, canary := range canaries {
					if got := strings.Contains(rendered, canary); got != want {
						t.Errorf("zdx/%s(%q) %s renders %q = %v, want %v: %s", c.name, c.id, mode, canary, got, want, rendered)
					}
				}
			}
			check(c.standardOnly, mode == redact.ModeStandard)
			check(c.shared, mode != redact.ModeParanoid)
			check(c.everywhere, true)
		}
	}
	for _, path := range []string{"/zdx/v1/administration/departments", "/zdx/v1/administration/locations"} {
		for _, query := range queries[path] {
			if !strings.Contains(query, "from=1761472800") || !strings.Contains(query, "to=1761480000") {
				t.Errorf("%s query %q lacks the window", path, query)
			}
		}
	}
	for _, path := range []string{"/zdx/v1/apps", "/zdx/v1/alerts/ongoing", "/zdx/v1/alerts/historical"} {
		for _, query := range queries[path] {
			for _, want := range []string{"from=1761472800", "to=1761480000", "dept=4", "loc=5"} {
				if !strings.Contains(query, want) {
					t.Errorf("%s query %q lacks %s", path, query, want)
				}
			}
		}
	}
	if len(queries["/zdx/v1/alerts/ongoing"]) != 2 {
		t.Errorf("ongoing alert requests = %d, want 2 (cursor followed)", len(queries["/zdx/v1/alerts/ongoing"]))
	}
}

// TestZDXAlertCursorFailsClosed: a repeated cursor, or a walk past the page
// cap with every cursor distinct, fails rather than returning a partial list.
func TestZDXAlertCursorFailsClosed(t *testing.T) {
	t.Parallel()

	repeated := zdxRoutes()
	repeated["/zdx/v1/alerts/ongoing?offset=page-2"] = `{"alerts":[],"next_offset":"page-2"}`
	endless := zdxRoutes()
	for i := 2; i <= zdxAlertPageLimit+1; i++ {
		endless[fmt.Sprintf("/zdx/v1/alerts/ongoing?offset=page-%d", i)] = fmt.Sprintf(`{"alerts":[{"id":%d}],"next_offset":"page-%d"}`, i, i+1)
	}
	for name, c := range map[string]struct {
		routes   map[string]string
		requests int
	}{
		// A repeated cursor stops at once; it does not spend the hourly
		// limit walking to the page cap.
		"repeated cursor":   {repeated, 2},
		"past the page cap": {endless, zdxAlertPageLimit},
	} {
		queries := map[string][]string{}
		service := newZDXTestService(t, c.routes, queries)
		if records, err := zdxReportHandlers["alerts"].read(context.Background(), service, "", zdxcommon.GetFromToFilters{From: 1, To: 2}); err == nil {
			t.Errorf("alerts with %s = %d records, want an error", name, len(records))
		}
		if got := len(queries["/zdx/v1/alerts/ongoing"]); got != c.requests {
			t.Errorf("alerts with %s made %d requests, want %d", name, got, c.requests)
		}
	}
}

// TestZDXReportRefusesBadRequests: Report checks id, window and resource
// before it builds any client.
func TestZDXReportRefusesBadRequests(t *testing.T) {
	t.Parallel()

	reader := &SDKReader{cfg: validReaderConfig()}
	from := time.Unix(1761472800, 0)
	window := ReportQuery{From: from, To: from.Add(2 * time.Hour)}
	for _, c := range []struct {
		product  resources.Product
		resource string
		id       string
		query    ReportQuery
		want     error
	}{
		{resources.ProductZDX, "alerts", "1", window, ErrInvalidResourceID},
		{resources.ProductZDX, "apps", "", ReportQuery{From: from, To: from}, ErrInvalidResourceID},
		{resources.ProductZDX, "apps", "", ReportQuery{}, ErrInvalidResourceID},
		{resources.ProductZDX, "devices", "", window, ErrUnsupportedResource},
		{resources.ProductZIA, "apps", "", window, ErrUnsupportedResource},
		{resources.ProductZDX, "departments", "", ReportQuery{From: window.From, To: window.To, Departments: []int{4}}, ErrInvalidResourceID},
		{resources.ProductZDX, "apps", "", ReportQuery{From: time.Unix(0, 0), To: from}, ErrInvalidResourceID},
	} {
		if _, err := reader.Report(context.Background(), c.product, c.resource, c.id, c.query); !errors.Is(err, c.want) {
			t.Errorf("Report(%s/%s, id %q) error = %v, want %v", c.product, c.resource, c.id, err, c.want)
		}
	}
	service := newZDXTestService(t, zdxRoutes(), map[string][]string{})
	if _, err := zdxReportHandlers["apps"].read(context.Background(), service, "7/score", zdxcommon.GetFromToFilters{From: 1, To: 2}); !errors.Is(err, ErrInvalidResourceID) {
		t.Errorf("apps report(\"7/score\") error = %v, want ErrInvalidResourceID", err)
	}
	routes := zdxRoutes()
	routes["/zdx/v1/apps/8"] = `{}`
	routes["/zdx/v1/apps/8/score"] = `[]`
	routes["/zdx/v1/apps/8/metrics"] = `[]`
	service = newZDXTestService(t, routes, map[string][]string{})
	if records, err := zdxReportHandlers["apps"].read(context.Background(), service, "8", zdxcommon.GetFromToFilters{From: 1, To: 2}); err == nil {
		t.Errorf("apps report(8) with an empty app body = %d records, want an error", len(records))
	}
}

// TestZDXReportErrorNotFoundOnlyForTheRecord: a 404 on one app's report is a
// missing record; a sign-in 404 (a wrong host), a list 404 and a cancelled
// read stay live failures.
func TestZDXReportErrorNotFoundOnlyForTheRecord(t *testing.T) {
	t.Parallel()

	notFound := &sdkerrorx.ErrorResponse{Response: &http.Response{StatusCode: http.StatusNotFound}}
	signIn := &signInError{status: http.StatusNotFound}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	for _, c := range []struct {
		name     string
		ctx      context.Context
		id       string
		err      error
		notFound bool
	}{
		{"record 404", context.Background(), "7", notFound, true},
		{"sign-in 404", context.Background(), "7", signIn, false},
		{"list 404", context.Background(), "", notFound, false},
		{"cancelled", canceled, "7", notFound, false},
	} {
		err := zdxReportError(c.ctx, resources.ProductZDX, "apps", c.id, c.err)
		if got := errors.As(err, new(resourceNotFoundError)); got != c.notFound {
			t.Errorf("%s: zdxReportError() = %v, not-found %v, want %v", c.name, err, got, c.notFound)
		}
	}
}
