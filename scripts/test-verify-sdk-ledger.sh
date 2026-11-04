#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

cat >"$tmp_dir/inventory.json" <<'JSON'
{"surfaces": [
  {"relative_path": "zscaler/zia/services/imported", "read_funcs": ["GetAll"]},
  {"relative_path": "zscaler/zia/services/ledgered", "read_funcs": ["GetAll"]},
  {"relative_path": "zscaler/zia/services/writeonly", "read_funcs": [], "mutate_funcs": ["Create"], "unknown_funcs": ["New"]},
  {"relative_path": "zscaler/zia/services/unknownonly", "read_funcs": [], "unknown_funcs": ["ExportThings"]},
  {"relative_path": "zscaler/zdx/services/reports", "read_funcs": ["GetAll"]},
  {"relative_path": "zscaler/zia", "read_funcs": ["Get"]}
]}
JSON
printf 'zia/services/imported\n' >"$tmp_dir/imports.txt"

run_case() {
  local name="$1" want="$2" pattern="$3" ledger_rows="$4"
  printf '%s' "$ledger_rows" >"$tmp_dir/ledger.tsv"
  set +e
  ZSCALERCTL_SDK_LEDGER_INVENTORY="$tmp_dir/inventory.json" \
    ZSCALERCTL_SDK_LEDGER_IMPORTS="$tmp_dir/imports.txt" \
    ZSCALERCTL_SDK_LEDGER_EXPORT_REPLAY="$tmp_dir/ledger.tsv" \
    bash "$repo_root/scripts/verify-sdk-ledger.sh" 2>"$tmp_dir/stderr"
  local got="$?"
  set -e
  if [[ "$got" -ne "$want" ]]; then
    echo "test-verify-sdk-ledger: $name: exit $got, want $want" >&2
    cat "$tmp_dir/stderr" >&2
    exit 1
  fi
  if [[ -n "$pattern" ]] && ! grep -q -- "$pattern" "$tmp_dir/stderr"; then
    echo "test-verify-sdk-ledger: $name: stderr lacks '$pattern'" >&2
    cat "$tmp_dir/stderr" >&2
    exit 1
  fi
}

row() { printf '%s\t%s\n' "$1" "$2"; }
complete="$(row zia/services/ledgered pending; row zia/services/unknownonly pending; row zdx/services/reports out_of_scope)
"

run_case complete 0 "" "$complete"
run_case empty-export 1 "export is empty" ""
run_case missing 1 "  zia/services/ledgered" "$(row zia/services/unknownonly pending; row zdx/services/reports out_of_scope)
"
run_case unknown-only-missing 1 "  zia/services/unknownonly" "$(row zia/services/ledgered pending; row zdx/services/reports out_of_scope)
"
run_case excluded-allowed 0 "" "$(row zia/services/ledgered pending; row zia/services/unknownonly excluded; row zdx/services/reports out_of_scope)
"
run_case excluded-with-reads 1 "  zia/services/ledgered" "$(row zia/services/ledgered excluded; row zia/services/unknownonly pending; row zdx/services/reports out_of_scope)
"
run_case imported-too 1 "now imports" "$complete$(row zia/services/imported pending)
"
run_case no-read-funcs 1 "  zia/services/writeonly" "$complete$(row zia/services/writeonly pending)
"
run_case invalid 1 "without exactly one" "$(row zia/services/ledgered invalid; row zia/services/unknownonly pending; row zdx/services/reports out_of_scope)
"
run_case duplicate 1 "listed twice" "$complete$complete"

echo "test-verify-sdk-ledger: PASS"
