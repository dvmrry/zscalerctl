package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"zscalerctl/internal/output"
	"zscalerctl/internal/resources"
)

// writeHelp prints the global usage. runParsed calls it only for --help on an
// unknown command; recognized commands, including products, get Cobra's help.
func (a *App) writeHelp(w io.Writer) {
	a.writeUsage(w, a.resourceCatalog())
}

func (a *App) writeUsage(w io.Writer, catalog resources.ResourceCatalog) {
	fmt.Fprintln(w, "usage: zscalerctl [global flags] <command> [args]")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "products: %s\n", strings.Join(productNames(knownProducts(catalog)), ", "))
	fmt.Fprintln(w)
	fmt.Fprintln(w, "commands:")
	fmt.Fprintln(w, "  help [command]")
	fmt.Fprintln(w, "  doctor")
	fmt.Fprintln(w, "  auth status")
	fmt.Fprintln(w, "  config show")
	fmt.Fprintln(w, "  config init [--force]")
	fmt.Fprintln(w, "  zia url-lookup <url> [url...]")
	fmt.Fprintln(w, "  zia admin-audit-logs status")
	fmt.Fprintln(w, "  schema list")
	fmt.Fprintln(w, "  introspect")
	fmt.Fprintln(w, "  machine manifest")
	fmt.Fprintln(w, "  dump --output <dir> [--products names] [--resources names] [--continue-on-error] [--force]")
	fmt.Fprintln(w, "  refs to <product> <resource> <id>")
	fmt.Fprintln(w, "  refs check <product> <resource>")
	fmt.Fprintln(w, "  diff <old-dump-dir> <new-dump-dir> [--products names] [--resources names] [--ignore-operational] [--detail] [--allow-partial] [--fail-on-drift]")
	fmt.Fprintf(w, "  completion %s\n", completionShellNames())
	fmt.Fprintln(w, "  version")
	for _, product := range knownProducts(catalog) {
		fmt.Fprintf(w, "  %s <resource> %s\n", product, strings.Join(productReadOperationNames(product, catalog), "|"))
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "global flags:")
	fmt.Fprintln(w, "  --profile <name>")
	fmt.Fprintln(w, "  --config <path>")
	fmt.Fprintln(w, "  --format auto|table|json|ndjson|pretty")
	fmt.Fprintln(w, "  --limit <positive-count>         (list only; JSON page output)")
	fmt.Fprintln(w, "  --output <path>")
	fmt.Fprintln(w, "  --offset <nonnegative-count>     (list only; requires --limit)")
	fmt.Fprintln(w, "  --timeout <duration>")
	fmt.Fprintln(w, "  --redaction standard|share|paranoid")
	fmt.Fprintln(w, "  --color auto|always|never")
	fmt.Fprintln(w, "  --no-color")
	fmt.Fprintln(w, "  --no-cache")
	fmt.Fprintln(w, "  --log-level off|error|warn|info|debug")
	fmt.Fprintln(w, "  --fields <a,b,c>")
	fmt.Fprintln(w, "  --filter <key=value|key~value>   (list only; key may be a nested path; repeatable, all must match)")
	fmt.Fprintln(w, "  --search <term>                  (list only; case-insensitive, any field)")
	fmt.Fprintln(w, "  --count-by <a,b,c>               (list only; counts instead of records; lists count per element)")
}

func dumpUsage(catalog resources.ResourceCatalog) string {
	return fmt.Sprintf(
		"usage: zscalerctl dump --output <dir> [--products %s] [--resources names] [--continue-on-error] [--force]\n"+
			"tip: add --log-level info to see start, per-resource, and completion progress on stderr during a long dump",
		strings.Join(productNames(snapshotProducts(catalog)), ","),
	)
}

func diffUsage(catalog resources.ResourceCatalog) string {
	return fmt.Sprintf(
		"usage: zscalerctl diff <old-dump-dir> <new-dump-dir> [--products %s] [--resources names] [--ignore-operational] [--detail] [--allow-partial] [--fail-on-drift]",
		strings.Join(productNames(snapshotProducts(catalog)), ","),
	)
}

// helpWidth is the fixed width help and usage text is packed to, so error
// messages and non-terminal output stay deterministic.
const helpWidth = 80

// columnize lays out names in a left-aligned, column-major grid (alphabetical
// down each column, like `ls`) indented two spaces, packed to fit width
// columns. Returns the block without a trailing newline.
func columnize(names []string, width int) string {
	if len(names) == 0 {
		return ""
	}
	const indent, gap = 2, 2
	longest := 0
	for _, n := range names {
		if len(n) > longest {
			longest = len(n)
		}
	}
	colWidth := longest + gap
	cols := max((width-indent+gap)/colWidth, 1)
	rows := (len(names) + cols - 1) / cols
	var b strings.Builder
	for r := range rows {
		var line strings.Builder
		line.WriteString(strings.Repeat(" ", indent))
		for c := range cols {
			i := c*rows + r
			if i >= len(names) {
				break
			}
			line.WriteString(names[i])
			line.WriteString(strings.Repeat(" ", colWidth-len(names[i])))
		}
		b.WriteString(strings.TrimRight(line.String(), " "))
		if r < rows-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func productCommandUsage(product resources.Product, catalog resources.ResourceCatalog) string {
	// Enumerate the product's resources so a cold caller (human or agent) can
	// discover real names from --help or a usage error instead of guessing;
	// `schema list` remains the machine-readable source of truth.
	names := make([]string, 0, 64)
	for _, spec := range catalog {
		if spec.Product == product {
			names = append(names, spec.Name)
		}
	}
	sort.Strings(names)
	msg := fmt.Sprintf(
		"usage: zscalerctl %s <resource> %s\n\nresources (%d; see also: zscalerctl --format json schema list):\n%s",
		product,
		strings.Join(productReadOperationNames(product, catalog), "|"),
		len(names),
		columnize(names, helpWidth),
	)
	if product == resources.ProductZIA {
		msg += "\n\ndiagnostics:\n  zscalerctl zia url-lookup <url> [url...]\n  zscalerctl zia admin-audit-logs status"
	}
	return msg
}

// resourceUsage builds help for a known resource: its supported read operations
// plus the renderable field names (standard mode), so the operator can discover
// what to pass to --fields without consulting `schema list`.
func resourceUsage(product resources.Product, spec resources.ResourceSpec) string {
	// A zero-value Style renders without color, so this stays byte-identical
	// plain text (usage errors and envelopes must never carry ANSI).
	return newHelpRenderer(output.Style{}).renderResourceUsage(product, spec)
}
