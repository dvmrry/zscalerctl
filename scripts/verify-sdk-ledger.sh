#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

# Every zscaler-sdk-go service package that may read (read functions, or
# only unknown-verb ones such as ExportPolicies) is either imported by
# zscalerctl's non-test code or named in internal/resources/sdk_ledger.go, so
# an SDK bump cannot add read surface nobody accounted for. A package whose
# only reader has an unknown verb but that also has mutators (an action such
# as sandbox submission) is not selected. Excluded entries may name only
# packages without read functions.

sdk_module="github.com/zscaler/zscaler-sdk-go/v3"

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

if ! command -v jq >/dev/null 2>&1; then
  echo "verify-sdk-ledger: jq is required" >&2
  exit 1
fi

# The replay variables feed saved inputs (for the self-test).
if [[ -n "${ZSCALERCTL_SDK_LEDGER_INVENTORY:-}" ]]; then
  cp "$ZSCALERCTL_SDK_LEDGER_INVENTORY" "$tmp_dir/inventory.json"
else
  # The vendor tree holds only imported packages, so scan the full module.
  sdk_version="$(go list -mod=mod -m -f '{{.Version}}' "$sdk_module")"
  sdk_dir="$(go mod download -json "$sdk_module@$sdk_version" | jq -r .Dir)"
  go run ./scripts/sdk-surface-inventory.go --sdk-dir "$sdk_dir" \
    --sdk-version "$sdk_version" --format json >"$tmp_dir/inventory.json"
fi
# The ledger comes from the compiled table (package<TAB>disposition), never
# from scraping internal/resources/sdk_ledger.go.
if [[ -n "${ZSCALERCTL_SDK_LEDGER_EXPORT_REPLAY:-}" ]]; then
  cp "$ZSCALERCTL_SDK_LEDGER_EXPORT_REPLAY" "$tmp_dir/ledger.tsv"
else
  ZSCALERCTL_SDK_LEDGER_EXPORT="$tmp_dir/ledger.tsv" \
    go test ./internal/resources/ -run '^TestSDKLedgerExport$' >&2
fi
if [[ ! -s "$tmp_dir/ledger.tsv" ]]; then
  echo "verify-sdk-ledger: the ledger export is empty" >&2
  exit 1
fi
ledger="internal/resources/sdk_ledger.go"
if [[ -n "${ZSCALERCTL_SDK_LEDGER_IMPORTS:-}" ]]; then
  sort -u "$ZSCALERCTL_SDK_LEDGER_IMPORTS" >"$tmp_dir/imported.txt"
else
  go list -f '{{join .Imports "\n"}}' ./cmd/... ./internal/... |
    sed -n "s|^$sdk_module/zscaler/||p" | sort -u >"$tmp_dir/imported.txt"
fi

jq -r '.surfaces[]
  | select(((.read_funcs // []) | length > 0)
      or (((.unknown_funcs // []) | length > 0) and ((.mutate_funcs // []) | length == 0)))
  | .relative_path' \
  "$tmp_dir/inventory.json" |
  sed -n 's|^zscaler/\([^/]*/services/.*\)$|\1|p' | sort -u >"$tmp_dir/read.txt"
jq -r '.surfaces[] | select((.read_funcs // []) | length > 0) | .relative_path' \
  "$tmp_dir/inventory.json" |
  sed -n 's|^zscaler/||p' | sort -u >"$tmp_dir/read-funcs.txt"
awk -F'\t' '$2 == "excluded" {print $1}' "$tmp_dir/ledger.tsv" | sort -u >"$tmp_dir/excluded.txt"
cut -f1 "$tmp_dir/ledger.tsv" | sort >"$tmp_dir/ledger.txt"

if [[ ! -s "$tmp_dir/read.txt" ]]; then
  echo "verify-sdk-ledger: the inventory lists no SDK read packages" >&2
  exit 1
fi

status=0
if dups="$(uniq -d "$tmp_dir/ledger.txt")" && [[ -n "$dups" ]]; then
  echo "verify-sdk-ledger: packages listed twice in $ledger:" >&2
  printf '  %s\n' $dups >&2
  status=1
fi
missing="$(comm -23 "$tmp_dir/read.txt" "$tmp_dir/imported.txt" | comm -23 - <(sort -u "$tmp_dir/ledger.txt"))"
if [[ -n "$missing" ]]; then
  echo "verify-sdk-ledger: SDK read packages neither imported nor in $ledger:" >&2
  printf '  %s\n' $missing >&2
  status=1
fi
imported_too="$(comm -12 <(sort -u "$tmp_dir/ledger.txt") "$tmp_dir/imported.txt")"
if [[ -n "$imported_too" ]]; then
  echo "verify-sdk-ledger: ledger packages that zscalerctl now imports (remove them):" >&2
  printf '  %s\n' $imported_too >&2
  status=1
fi
invalid="$(awk -F'\t' '$2 == "invalid" {print $1}' "$tmp_dir/ledger.tsv")"
if [[ -n "$invalid" ]]; then
  echo "verify-sdk-ledger: ledger entries without exactly one disposition:" >&2
  printf '  %s\n' $invalid >&2
  status=1
fi
excluded_reads="$(comm -12 "$tmp_dir/excluded.txt" "$tmp_dir/read-funcs.txt")"
if [[ -n "$excluded_reads" ]]; then
  echo "verify-sdk-ledger: Excluded packages that have read functions (use Resources, Gaps or Pending, or OutOfScope for an unmodeled product):" >&2
  printf '  %s\n' $excluded_reads >&2
  status=1
fi
gone="$(comm -23 <(sort -u "$tmp_dir/ledger.txt") "$tmp_dir/read.txt")"
if [[ -n "$gone" ]]; then
  echo "verify-sdk-ledger: ledger packages the SDK no longer has as read packages (remove them):" >&2
  printf '  %s\n' $gone >&2
  status=1
fi
exit "$status"
