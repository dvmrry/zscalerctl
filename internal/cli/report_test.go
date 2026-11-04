package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"zscalerctl/internal/cli"
	"zscalerctl/internal/resources"
	"zscalerctl/internal/zscaler"
)

// fakeReportReader answers zdx/apps reports and records the query.
type fakeReportReader struct {
	query zscaler.ReportQuery
	id    string
	calls int
}

func (r *fakeReportReader) List(context.Context, resources.Product, string) ([]resources.SourceRecord, error) {
	return nil, nil
}

func (r *fakeReportReader) Get(context.Context, resources.Product, string, string) (resources.SourceRecord, error) {
	return resources.SourceRecord{}, nil
}

func (r *fakeReportReader) Show(context.Context, resources.Product, string) (resources.SourceRecord, error) {
	return resources.SourceRecord{}, nil
}

func (r *fakeReportReader) Report(_ context.Context, _ resources.Product, _ string, id string, query zscaler.ReportQuery) ([]resources.SourceRecord, error) {
	r.calls++
	r.query, r.id = query, id
	return []resources.SourceRecord{resources.NewSourceRecord(map[string]any{
		"id": 7, "name": "app-name", "score": json.Number("87.3"), "total_users": 12,
		"most_impacted_region": map[string]any{"id": "region-id-canary", "city": "city-canary", "country": "country-name"},
	})}, nil
}

func runReport(t *testing.T, reader *fakeReportReader, args ...string) (string, string, error) {
	t.Helper()
	var out, errOut bytes.Buffer
	app := cli.NewWithOptions(&out, &errOut, []string{"ZSCALERCTL_CLOUD=example"}, cli.Options{Reader: reader})
	err := app.Run(context.Background(), args)
	return out.String(), errOut.String(), err
}

// TestZDXReportEnvelopeWindowAndRedaction: a report's JSON names its window
// first, sends the resolved window and filters to the reader, and projects
// records like any read (city and region id standard only).
func TestZDXReportEnvelopeWindowAndRedaction(t *testing.T) {
	t.Parallel()

	reader := &fakeReportReader{}
	out, _, err := runReport(t, reader, "zdx", "apps", "report", "--from", "2026-10-26T10:00:00Z", "--to", "2026-10-26T12:00:00Z",
		"--department", "4", "--location", "5", "--location", "6", "--format", "json", "--redaction", "share")
	if err != nil {
		t.Fatalf("report error = %v", err)
	}
	var envelope struct {
		Window  struct{ From, To string } `json:"window"`
		Records []map[string]any          `json:"records"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("output %q is not a report envelope: %v", out, err)
	}
	if envelope.Window.From != "2026-10-26T10:00:00Z" || envelope.Window.To != "2026-10-26T12:00:00Z" {
		t.Errorf("window = %+v, want the requested window", envelope.Window)
	}
	if got := reader.query; got.From.Format(time.RFC3339) != "2026-10-26T10:00:00Z" || got.To.Format(time.RFC3339) != "2026-10-26T12:00:00Z" ||
		len(got.Departments) != 1 || got.Departments[0] != 4 || len(got.Locations) != 2 || got.Locations[1] != 6 {
		t.Errorf("reader query = %+v, want the window and filters", got)
	}
	if strings.Contains(out, "city-canary") || strings.Contains(out, "region-id-canary") || !strings.Contains(out, "country-name") ||
		!strings.Contains(out, `"score": 87.3`) && !strings.Contains(out, `"score":87.3`) {
		t.Errorf("share output = %s, want country and score but not city or region id", out)
	}
}

// TestZDXReportRules: an id reaches the reader, --fields narrows, list and
// row filters are refused, and a report resource has no list.
func TestZDXReportRules(t *testing.T) {
	t.Parallel()

	reader := &fakeReportReader{}
	out, _, err := runReport(t, reader, "zdx", "apps", "report", "7", "--since", "24h", "--fields", "name", "--format", "json")
	if err != nil || reader.id != "7" || strings.Contains(out, "total_users") || !strings.Contains(out, "app-name") {
		t.Errorf("report 7 --fields name = %s (id %q, err %v), want only name for id 7", out, reader.id, err)
	}
	for _, args := range [][]string{
		{"zdx", "apps", "report", "--filter", "name=x"},
		{"zdx", "apps", "report", "--limit", "1"},
		{"zdx", "apps", "report", "--since", "2h", "--from", "2026-10-26T10:00:00Z", "--to", "2026-10-26T12:00:00Z"},
		{"zdx", "apps", "report", "a", "b"},
		{"zdx", "apps", "list"},
		{"zia", "locations", "report"},
		{"zdx", "apps", "report", "--fields", "bogus"},
		{"zdx", "departments", "report", "3"},
		{"zdx", "departments", "report", "--department", "4"},
		{"zdx", "apps", "report", "--since", "213504d"},
	} {
		before := reader.calls
		if _, _, err := runReport(t, reader, args...); err == nil {
			t.Errorf("%v error = nil, want a usage error", args)
		}
		if reader.calls != before {
			t.Errorf("%v reached the reader", args)
		}
	}
}

// TestZDXReportsStayOutOfSnapshots: dump cannot select a report resource and
// the machine manifest does not list one.
func TestZDXReportsStayOutOfSnapshots(t *testing.T) {
	t.Parallel()

	reader := &fakeReportReader{}
	if _, _, err := runReport(t, reader, "dump", "--products", "zdx", "--output", t.TempDir()+"/d"); err == nil {
		t.Error("dump --products zdx error = nil, want an empty-selection error")
	}
	out, _, err := runReport(t, reader, "machine", "manifest")
	if err != nil {
		t.Fatalf("machine manifest error = %v", err)
	}
	if strings.Contains(out, `"zdx"`) || strings.Contains(out, "report") {
		t.Errorf("machine manifest mentions zdx or report: %s", out)
	}
	if reader.calls != 0 {
		t.Errorf("reader calls = %d, want 0", reader.calls)
	}
}

// TestZDXReportRefsRefused: refs names a report resource as a report.
func TestZDXReportRefsRefused(t *testing.T) {
	t.Parallel()

	reader := &fakeReportReader{}
	_, _, err := runReport(t, reader, "refs", "check", "zdx", "apps")
	if err == nil || !strings.Contains(err.Error(), "time-windowed report") || reader.calls != 0 {
		t.Errorf("refs check zdx apps error = %v (calls %d), want the report refusal and no read", err, reader.calls)
	}
}

// TestZDXReportCountBy: --count-by counts report records; --search is refused.
func TestZDXReportCountBy(t *testing.T) {
	t.Parallel()

	reader := &fakeReportReader{}
	out, _, err := runReport(t, reader, "zdx", "apps", "report", "--count-by", "name", "--format", "json")
	if err != nil || !strings.Contains(out, `"count_by"`) || !strings.Contains(out, "app-name") {
		t.Errorf("report --count-by name = %s, err %v; want counts", out, err)
	}
	if _, _, err := runReport(t, reader, "zdx", "apps", "report", "--search", "x"); err == nil {
		t.Error("report --search error = nil, want a usage error")
	}
}

// TestZDXReportHelpShowsPolicyUsage: resource help prints the report's id,
// window and filter usage from its policy.
func TestZDXReportHelpShowsPolicyUsage(t *testing.T) {
	t.Parallel()

	for resource, want := range map[string]string{
		"apps":        "zscalerctl zdx apps report [id] [--since 2h | --from T --to T] [--department ID] [--location ID]",
		"departments": "zscalerctl zdx departments report [--since 2h | --from T --to T]\n",
	} {
		out, _, _ := runReport(t, &fakeReportReader{}, "zdx", resource, "report", "--help")
		if !strings.Contains(out, want) {
			t.Errorf("zdx %s report --help = %q, want %q", resource, out, want)
		}
	}
}
