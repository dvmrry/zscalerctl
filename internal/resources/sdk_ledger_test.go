package resources

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
)

// TestSDKLedgerTargetsExist keeps every ledger entry pointing at something
// real: exactly one disposition, gaps declared under the package's own
// product by surface name (not alias), resources in the catalog, and
// Pending, Excluded or OutOfScope (never for a modeled product) where no
// product exists. Whether the ledger lists every SDK read
// package is scripts/verify-sdk-ledger.sh's job; it needs the SDK module.
func TestSDKLedgerTargetsExist(t *testing.T) {
	t.Parallel()

	catalog := Catalog()
	surfaces := map[Product][]string{}
	for _, gap := range coverageGaps {
		surfaces[gap.Product] = append(surfaces[gap.Product], gap.Surface)
	}
	var prev string
	for _, entry := range sdkLedger {
		pkg := entry.Package
		if pkg <= prev {
			t.Errorf("%s: ledger is not sorted and unique (after %s)", pkg, prev)
		}
		prev = pkg
		tree, rest, ok := strings.Cut(pkg, "/")
		if !ok || !strings.HasPrefix(rest, "services/") {
			t.Errorf("%s: package is not <product>/services/...", pkg)
			continue
		}
		set := 0
		for _, present := range []bool{len(entry.Resources) > 0, len(entry.Gaps) > 0, entry.Command != "", entry.Excluded != "", entry.OutOfScope != "", entry.Pending != ""} {
			if present {
				set++
			}
		}
		if set != 1 {
			t.Errorf("%s: %d dispositions, want exactly one", pkg, set)
		}
		product, modeled := sdkLedgerProduct[tree]
		if !modeled {
			if entry.Pending == "" && entry.Excluded == "" && entry.OutOfScope == "" {
				t.Errorf("%s: SDK tree %q has no zscalerctl product; only Pending, Excluded or OutOfScope is possible", pkg, tree)
			}
			continue
		}
		if entry.OutOfScope != "" {
			// A modeled product's surface stays visible as a declared gap.
			t.Errorf("%s: OutOfScope is only for SDK trees without a zscalerctl product; use Gaps", pkg)
		}
		for _, surface := range entry.Gaps {
			if !slices.Contains(surfaces[product], surface) {
				t.Errorf("%s: gap %s/%s is not a declared surface", pkg, product, surface)
			}
		}
		for _, ref := range entry.Resources {
			if ref.Product != product {
				t.Errorf("%s: resource %s is outside product %s", pkg, ref, product)
			}
			if _, ok := catalog.FindSpec(ref.Product, ref.Resource); !ok {
				t.Errorf("%s: resource %s is not in the catalog", pkg, ref)
			}
		}
	}
}

// TestSDKLedgerExport writes the ledger as "package<TAB>disposition" lines to
// $ZSCALERCTL_SDK_LEDGER_EXPORT for scripts/verify-sdk-ledger.sh, so the gate
// reads the compiled table instead of scraping Go source.
func TestSDKLedgerExport(t *testing.T) {
	out := os.Getenv("ZSCALERCTL_SDK_LEDGER_EXPORT")
	if out == "" {
		t.Skip("ZSCALERCTL_SDK_LEDGER_EXPORT is not set")
	}
	var b strings.Builder
	for _, entry := range sdkLedger {
		var kinds []string
		for kind, present := range map[string]bool{
			"resources": len(entry.Resources) > 0, "gaps": len(entry.Gaps) > 0, "command": entry.Command != "",
			"excluded": entry.Excluded != "", "out_of_scope": entry.OutOfScope != "", "pending": entry.Pending != "",
		} {
			if present {
				kinds = append(kinds, kind)
			}
		}
		// The gate fails on anything but exactly one disposition.
		kind := "invalid"
		if len(kinds) == 1 {
			kind = kinds[0]
		}
		fmt.Fprintf(&b, "%s\t%s\n", entry.Package, kind)
	}
	if err := os.WriteFile(out, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
}
