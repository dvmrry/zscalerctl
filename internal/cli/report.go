package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"zscalerctl/internal/config"
	"zscalerctl/internal/output"
	"zscalerctl/internal/resources"
	machineruntime "zscalerctl/internal/runtime"
)

// reportFlags are the report operation's window and filter flags.
// They exist only on products that have report resources, so no other
// command's help changes.
type reportFlags struct {
	since       string
	sinceSet    bool
	from        string
	to          string
	departments []int
	locations   []int
}

const (
	defaultReportSince = "2h"
	// maxReportSince bounds --since before any arithmetic can overflow;
	// ZDX keeps far less history than this.
	maxReportSince = 90 * 24 * time.Hour
)

func addReportFlags(cmd *cobra.Command, flags *reportFlags) {
	cmd.Flags().StringVar(&flags.since, "since", defaultReportSince, "report window ending now: a duration such as 2h, 90m or 7d")
	cmd.Flags().StringVar(&flags.from, "from", "", "report window start (RFC 3339); requires --to")
	cmd.Flags().StringVar(&flags.to, "to", "", "report window end (RFC 3339); requires --from")
	cmd.Flags().IntSliceVar(&flags.departments, "department", nil, "narrow a report to department ids (repeatable)")
	cmd.Flags().IntSliceVar(&flags.locations, "location", nil, "narrow a report to location ids (repeatable)")
}

// productHasReports reports whether any of product's resources is a report.
func productHasReports(catalog resources.ResourceCatalog, product resources.Product) bool {
	for _, spec := range catalog {
		if spec.Product == product && spec.EffectiveShape() == resources.ShapeReport {
			return true
		}
	}
	return false
}

// resolveReportWindow turns the flags into an absolute UTC window of whole
// seconds ending no later than now. --from/--to win only as a pair and
// exclude --since.
func resolveReportWindow(now time.Time, flags reportFlags) (time.Time, time.Time, error) {
	if flags.from != "" || flags.to != "" {
		if flags.from == "" || flags.to == "" {
			return time.Time{}, time.Time{}, UsageError{Message: "--from and --to go together"}
		}
		if flags.sinceSet {
			return time.Time{}, time.Time{}, UsageError{Message: "--since cannot be combined with --from and --to"}
		}
		from, err := time.Parse(time.RFC3339, flags.from)
		if err != nil {
			return time.Time{}, time.Time{}, UsageError{Message: "--from must be an RFC 3339 time, such as 2026-10-26T10:00:00Z"}
		}
		to, err := time.Parse(time.RFC3339, flags.to)
		if err != nil {
			return time.Time{}, time.Time{}, UsageError{Message: "--to must be an RFC 3339 time, such as 2026-10-26T12:00:00Z"}
		}
		from, to = from.UTC().Truncate(time.Second), to.UTC().Truncate(time.Second)
		if !from.Before(to) {
			return time.Time{}, time.Time{}, UsageError{Message: "--from must be at least a second before --to"}
		}
		if from.Unix() <= 0 {
			return time.Time{}, time.Time{}, UsageError{Message: "--from must be after 1970-01-01T00:00:00Z"}
		}
		if to.After(now.UTC().Add(time.Minute)) {
			return time.Time{}, time.Time{}, UsageError{Message: "--to cannot be in the future"}
		}
		return from, to, nil
	}
	since, err := parseReportSince(flags.since)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	to := now.UTC().Truncate(time.Second)
	return to.Add(-since), to, nil
}

func parseReportSince(value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	var since time.Duration
	if days, ok := strings.CutSuffix(value, "d"); ok {
		n, err := strconv.Atoi(days)
		if err != nil {
			return 0, UsageError{Message: "--since must be a duration such as 2h, 90m or 7d"}
		}
		// Bound before multiplying: a huge n would wrap to a short window.
		if n < 0 || n > int(maxReportSince/(24*time.Hour)) {
			return 0, UsageError{Message: "--since can be at most 90d"}
		}
		since = time.Duration(n) * 24 * time.Hour
	} else {
		parsed, err := time.ParseDuration(value)
		if err != nil {
			return 0, UsageError{Message: "--since must be a duration such as 2h, 90m or 7d"}
		}
		since = parsed
	}
	if since < time.Minute {
		return 0, UsageError{Message: "--since must be at least 1m"}
	}
	if since > maxReportSince {
		return 0, UsageError{Message: "--since can be at most 90d"}
	}
	return since, nil
}

// reportEnvelope is a report's JSON output: the resolved window first, then
// the projected records, so a result never reads as a snapshot.
type reportEnvelope struct {
	Window  reportWindow               `json:"window"`
	Records resources.ProjectedRecords `json:"records"`
}

type reportWindow struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func (reportEnvelope) OutputSafe() {}

func (a *App) runReport(
	ctx context.Context,
	cfg config.Config,
	opts globalOptions,
	spec resources.ResourceSpec,
	args []string,
) error {
	usage := reportUsage(spec)
	policy, ok := machineruntime.ReportPolicyFor(spec.Product, spec.Name)
	if !ok {
		return UsageError{Message: usage}
	}
	if len(args) < 2 || len(args) > 3 || (len(args) == 3 && !policy.AllowsID) {
		return UsageError{Message: usage}
	}
	if !policy.AcceptsFilter && (len(opts.report.departments) > 0 || len(opts.report.locations) > 0) {
		return UsageError{Message: fmt.Sprintf("%s/%s report takes no --department or --location\n%s", spec.Product, spec.Name, usage)}
	}
	if len(opts.filters) > 0 || opts.search != "" || opts.limitSet {
		return UsageError{Message: "report does not take --filter, --search or --limit\n" + usage}
	}
	id := ""
	if len(args) == 3 {
		id = args[2]
	}
	from, to, err := resolveReportWindow(a.clock(), opts.report)
	if err != nil {
		return err
	}
	mode := cfg.Defaults.Redaction
	// Validate narrowing before the read: ZDX allows 100 requests an hour.
	if len(opts.fields) > 0 {
		if err := resources.ValidateNarrowOptions(spec, resources.NarrowOptions{Fields: opts.fields}); err != nil {
			return UsageError{Message: err.Error()}
		}
	}
	if len(opts.countBy) > 0 {
		if err := resources.ValidateCountByPathsInMode(spec, mode, opts.countBy); err != nil {
			return UsageError{Message: err.Error()}
		}
	}
	query := machineruntime.ReportQuery{
		From:        from,
		To:          to,
		Departments: opts.report.departments,
		Locations:   opts.report.locations,
	}
	projected, err := callWithSpinner(a, opts, "contacting Zscaler", func() (resources.ProjectedRecords, error) {
		reader := a.reader
		if reader == nil {
			live, err := machineruntime.NewReaderFromConfig(ctx, cfg, machineruntime.Options{
				Timeout:    opts.timeout,
				Catalog:    a.resourceCatalog(),
				DiagLogger: a.sdkDiagLogger(opts),
				Notices:    &a.notices,
			})
			if err != nil {
				return resources.ProjectedRecords{}, err
			}
			reader = live
		}
		return machineruntime.ReadReportFromReader(ctx, reader, spec, mode, id, query)
	})
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return err
	}
	if len(opts.fields) > 0 {
		projected, err = resources.NarrowProjectedRecords(spec, mode, projected, resources.NarrowOptions{Fields: opts.fields})
		if err != nil {
			return UsageError{Message: err.Error()}
		}
	}
	window := reportWindow{From: from.Format(time.RFC3339), To: to.Format(time.RFC3339)}
	if len(opts.countBy) > 0 {
		fmt.Fprintf(&a.catalogNotices, "zscalerctl: report window %s to %s\n", window.From, window.To)
		return a.writeCounts(cfg, opts, projected)
	}
	a.noteOmittedFields(spec, mode, opts.fields, false)
	if opts.format == output.FormatJSON {
		return a.renderer(cfg, opts).WriteJSON(a.out, reportEnvelope{Window: window, Records: projected})
	}
	fmt.Fprintf(&a.catalogNotices, "zscalerctl: report window %s to %s\n", window.From, window.To)
	return a.writeProjectedRecords(cfg, opts, spec, projected)
}

// reportUsage is one report resource's usage line, from its static policy.
func reportUsage(spec resources.ResourceSpec) string {
	policy, _ := machineruntime.ReportPolicyFor(spec.Product, spec.Name)
	usage := fmt.Sprintf("usage: zscalerctl %s %s report", spec.Product, spec.Name)
	if policy.AllowsID {
		usage += " [id]"
	}
	usage += " [--since 2h | --from T --to T]"
	if policy.AcceptsFilter {
		usage += " [--department ID] [--location ID]"
	}
	return usage
}

func (a *App) clock() time.Time {
	if a.now != nil {
		return a.now()
	}
	return time.Now()
}
