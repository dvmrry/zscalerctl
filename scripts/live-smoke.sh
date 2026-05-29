#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

denied_exact_keys_json='["preSharedKey","vpnCredentials","createdBy","lastModifiedBy","managedBy","city","primaryDestVip","secondaryDestVip","lastModUser","dynamicLocationGroupCriteria","locations"]'
denied_key_pattern='(?i)(password|secret|token|api[_-]?key|preSharedKey|credential)'
manifest_warning='sanitized dumps remain confidential operational data'

out_dir=""
require_nonempty=0
require_credentials=0
skip_credential_check=0
strict_counts=0
failures=0
resources=()
resource_filters=()
credential_family_name=""

usage() {
  cat <<'EOF'
usage: scripts/live-smoke.sh [--out DIR] [--bin PATH] [--resources LIST] [--require-credentials] [--require-nonempty] [--strict-counts]

Runs a read-only live smoke against the currently configured zscalerctl
credentials and prints PASS/FAIL markers for pre-PR validation.

Options:
  --out DIR            Write validation artifacts under DIR. Defaults to a
                       secure temporary directory that is kept for inspection.
  --bin PATH           zscalerctl binary to run. Defaults to
                       "go run -mod=vendor ./cmd/zscalerctl".
  --resources LIST     Comma-separated resources to validate. Supports bare
                       names and product/name. Defaults to all read/list
                       resources with OneAPI credentials, or current ZIA
                       read/list resources for legacy/back-compatible runs.
  --require-credentials
                       Fail instead of SKIP when no supported live credential
                       family is configured. Use this for release gating.
                       Selected ZPA resources also require
                       ZSCALERCTL_ZPA_CUSTOMER_ID.
  --require-nonempty   Treat a zero-record resource list as a failure.
  --strict-counts      Fail if a list count differs from the dump count.
                       By default this is INFO because live data can change.
  --skip-credential-check
                       Internal test hook for fake CLIs.
  -h, --help           Show this help.

This script does not print credential values or live resource payloads. It
recognizes explicit zscalerctl OneAPI credentials and explicit ZIA legacy
credentials; raw SDK env vars such as ZIA_USERNAME are intentionally ignored.
EOF
}

while (($#)); do
  case "$1" in
    --out)
      if (($# < 2)); then
        echo "--out requires a directory" >&2
        exit 2
      fi
      out_dir="$2"
      shift 2
      ;;
    --bin)
      if (($# < 2)); then
        echo "--bin requires a path" >&2
        exit 2
      fi
      ZSCALERCTL_BIN="$2"
      shift 2
      ;;
    --resources)
      if (($# < 2)); then
        echo "--resources requires a comma-separated list" >&2
        exit 2
      fi
      IFS=',' read -r -a resource_filters <<<"$2"
      shift 2
      ;;
    --require-credentials)
      require_credentials=1
      shift
      ;;
    --require-nonempty)
      require_nonempty=1
      shift
      ;;
    --strict-counts)
      strict_counts=1
      shift
      ;;
    --skip-credential-check)
      skip_credential_check=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

pass() {
  printf '[PASS] %s\n' "$*"
}

info() {
  printf '[INFO] %s\n' "$*"
}

skip() {
  printf '[SKIP] %s\n' "$*"
}

fail() {
  printf '[FAIL] %s\n' "$*" >&2
  failures=$((failures + 1))
}

is_set() {
  [[ -n "${!1:-}" ]]
}

credential_family() {
  local mode="${ZSCALERCTL_AUTH_MODE:-}"
  local oneapi=0
  local legacy=0

  if is_set ZSCALERCTL_CLIENT_ID &&
    (is_set ZSCALERCTL_CLIENT_SECRET || is_set ZSCALERCTL_CLIENT_SECRET_FILE) &&
    is_set ZSCALERCTL_VANITY_DOMAIN; then
    oneapi=1
  fi

  if is_set ZSCALERCTL_ZIA_USERNAME &&
    (is_set ZSCALERCTL_ZIA_PASSWORD || is_set ZSCALERCTL_ZIA_PASSWORD_FILE) &&
    (is_set ZSCALERCTL_ZIA_API_KEY || is_set ZSCALERCTL_ZIA_API_KEY_FILE) &&
    is_set ZSCALERCTL_ZIA_CLOUD; then
    legacy=1
  fi

  case "$mode" in
    zia-legacy)
      if ((legacy)); then
        printf 'ZIA legacy'
        return 0
      fi
      return 1
      ;;
    oneapi|"")
      if ((oneapi)); then
        printf 'OneAPI'
        return 0
      fi
      if [[ -z "$mode" ]] && ((legacy)); then
        printf 'ZIA legacy'
        return 0
      fi
      return 1
      ;;
    *)
      return 1
      ;;
  esac
}

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 2
  fi
}

mode_of() {
  case "$(uname -s)" in
    Darwin|FreeBSD)
      stat -f '%Lp' "$1"
      ;;
    *)
      stat -c '%a' "$1"
      ;;
  esac
}

find_denied_keys() {
  jq -r --argjson exact "$denied_exact_keys_json" --arg pattern "$denied_key_pattern" '
    [.. | objects | keys[] | select((. as $k | $exact | index($k)) or test($pattern))] | unique | .[]
  ' "$1"
}

selected_has_product() {
  local product="$1"
  local resource_key
  for resource_key in "${resources[@]}"; do
    if [[ "${resource_key%%/*}" == "$product" ]]; then
      return 0
    fi
  done
  return 1
}

validate_selected_product_credentials() {
  if ((skip_credential_check)); then
    return 0
  fi
  if selected_has_product zpa && ! is_set ZSCALERCTL_ZPA_CUSTOMER_ID; then
    fail "selected ZPA resources require ZSCALERCTL_ZPA_CUSTOMER_ID"
    return 1
  fi
  return 0
}

validate_json_array() {
  local label="$1"
  local file="$2"
  local count

  if ! jq -e 'type == "array"' "$file" >/dev/null 2>&1; then
    fail "$label is not a JSON array: $file"
    return 1
  fi
  pass "$label returned a JSON array"

  count="$(jq 'length' "$file")"
  if [[ "$count" == "0" ]]; then
    if ((require_nonempty)); then
      fail "$label returned 0 records"
      return 1
    else
      info "$label returned 0 records"
    fi
  else
    pass "$label returned $count records"
  fi
  return 0
}

validate_no_denied_keys() {
  local label="$1"
  local file="$2"
  local denied

  denied="$(find_denied_keys "$file")"
  if [[ -n "$denied" ]]; then
    fail "$label contains denied field key(s): $(tr '\n' ' ' <<<"$denied")"
    return
  fi
  pass "$label contains no denied field keys"
}

find_non_catalog_keys() {
  local product="$1"
  local resource="$2"
  local file="$3"
  local schema="$4"

  jq -r --slurpfile schema "$schema" --arg product "$product" --arg resource "$resource" '
    ($schema[0] | map(select(.product == $product and .name == $resource)) | .[0]) as $spec
    | if $spec == null then
        ["<missing schema resource>"]
      else
        ([
          $spec.fields[]?
          | select(any(.allowed_modes[]?; . == "standard"))
          | (.json_name // .name)
        ]) as $allowed
        | [
          .[]?
          | objects
          | keys[] as $key
          | select(($allowed | index($key)) | not)
          | $key
        ]
        | unique
      end
    | .[]
  ' "$file"
}

validate_catalog_subset() {
  local label="$1"
  local product="$2"
  local resource="$3"
  local file="$4"
  local schema="$5"
  local unexpected

  unexpected="$(find_non_catalog_keys "$product" "$resource" "$file" "$schema")"
  if [[ -n "$unexpected" ]]; then
    fail "$label contains non-catalog field key(s): $(tr '\n' ' ' <<<"$unexpected")"
    return
  fi
  pass "$label contains only catalog-allowed top-level fields"
}

redaction_marker_paths() {
  jq -r '
    def fieldpath:
      reduce .[] as $part ("";
        if ($part | type) == "number" then
          . + "[]"
        elif . == "" then
          . + ($part | tostring)
        else
          . + "." + ($part | tostring)
        end
      );

    [
      paths(strings) as $path
      | select(getpath($path) | contains("<REDACTED:"))
      | $path | fieldpath
    ]
    | unique
    | .[]
  ' "$1"
}

summarize_redaction_markers() {
  local label="$1"
  local file="$2"
  local paths
  local summary

  paths="$(redaction_marker_paths "$file")"
  if [[ -n "$paths" ]]; then
    summary="$(printf '%s\n' "$paths" | paste -sd ' ' -)"
    info "$label redaction markers at: $summary"
    return
  fi
  pass "$label has no redaction markers"
}

load_resources() {
  local schema_file="$1"
  local stderr_file="$2"
  local filter
  local matches
  local resource_key

  if "${cli[@]}" --format json schema list >"$schema_file" 2>"$stderr_file"; then
    pass "schema list command completed"
  else
    fail "schema list command failed; stderr captured at $stderr_file"
    return 1
  fi

  if ! jq -e 'type == "array"' "$schema_file" >/dev/null 2>&1; then
    fail "schema list is not a JSON array: $schema_file"
    return 1
  fi
  pass "schema list returned a JSON array"

  if ((${#resource_filters[@]} == 0)); then
    local default_product="zia"
    if [[ "$credential_family_name" == "OneAPI" ]]; then
      default_product=""
    fi
    while IFS= read -r resource_key; do
      resources+=("$resource_key")
    done < <(jq -r --arg product "$default_product" '
    [
      .[]
      | select($product == "" or .product == $product)
      | select(any(.operations[]?; .name == "list" and .capability == "read"))
      | .product + "/" + .name
    ]
    | sort
    | .[]
  ' "$schema_file")

    if ((${#resources[@]} == 0)); then
      fail "schema list contains no ZIA read/list resources"
      return 1
    fi

    pass "schema list found ${#resources[@]} default read/list resource(s): ${resources[*]}"
    return 0
  fi

  for filter in "${resource_filters[@]}"; do
    filter="$(printf '%s' "$filter" | tr '[:upper:]' '[:lower:]' | xargs)"
    if [[ -z "$filter" ]]; then
      fail "empty resource in --resources"
      return 1
    fi
    if [[ "$filter" == */* ]]; then
      matches="$(jq -r --arg key "$filter" '
        [
          .[]
          | select(any(.operations[]?; .name == "list" and .capability == "read"))
          | .product + "/" + .name
          | select(. == $key)
        ]
        | .[]
      ' "$schema_file")"
    else
      matches="$(jq -r --arg name "$filter" '
        [
          .[]
          | select(.name == $name)
          | select(any(.operations[]?; .name == "list" and .capability == "read"))
          | .product + "/" + .name
        ]
        | sort
        | .[]
      ' "$schema_file")"
    fi
    if [[ -z "$matches" ]]; then
      fail "schema list has no read/list resource matching --resources entry $filter"
      return 1
    fi
    if (($(printf '%s\n' "$matches" | sed '/^$/d' | wc -l | tr -d ' ') > 1)); then
      fail "ambiguous --resources entry $filter; use product/name"
      return 1
    fi
    resources+=("$matches")
  done

  pass "schema list selected ${#resources[@]} read/list resource(s): ${resources[*]}"
  return 0
}

compare_counts() {
  local product="$1"
  local resource="$2"
  local list_count="$3"
  local dump_count="$4"

  if [[ "$list_count" == "$dump_count" ]]; then
    pass "$product $resource list and dump counts match ($list_count records)"
    return
  fi
  if ((strict_counts)); then
    fail "$product $resource list count = $list_count, dump count = $dump_count"
  else
    info "$product $resource list count = $list_count, dump count = $dump_count; live data may have changed between reads"
  fi
}

write_expected_dump_paths() {
  local output="$1"
  local resource_key
  local product
  local resource

  : >"$output"
  for resource_key in "${resources[@]}"; do
    product="${resource_key%%/*}"
    resource="${resource_key#*/}"
    printf 'resources/%s/%s.json\n' "$product" "$resource" >>"$output"
  done
  sort -o "$output" "$output"
}

validate_manifest_resource_set() {
  local manifest="$1"
  local expected="$2"
  local actual="$3"
  local diff_file="$4"

  jq -r '.resources[]? | .path' "$manifest" | sort >"$actual"
  if diff -u "$expected" "$actual" >"$diff_file"; then
    pass "dump manifest resource set matches selected catalog"
    pass "dump manifest contains only selected resources"
  else
    fail "dump manifest resource set differs from selected catalog; diff captured at $diff_file"
  fi
}

validate_dump_file_set() {
  local dump_dir="$1"
  local expected="$2"
  local actual="$3"
  local diff_file="$4"

  if [[ ! -d "$dump_dir/resources" ]]; then
    fail "dump resources directory missing: $dump_dir/resources"
    return
  fi

  find "$dump_dir/resources" -mindepth 2 -maxdepth 2 -type f -name '*.json' -print |
    while IFS= read -r path; do
      printf '%s\n' "${path#"$dump_dir/"}"
    done | sort >"$actual"

  if diff -u "$expected" "$actual" >"$diff_file"; then
    pass "dump resource files match selected catalog"
  else
    fail "dump resource files differ from selected catalog; diff captured at $diff_file"
  fi
}

summarize_redaction_report() {
  local report="$1"
  local found=0
  local product
  local name
  local dropped
  local redacted

  while IFS=$'\t' read -r product name dropped redacted; do
    found=1
    info "redaction report $product $name: dropped fields [$dropped], redacted fields [$redacted]"
  done < <(jq -r '
    .resources[]?
    | select(((.dropped_fields // []) | length) > 0 or ((.redacted_fields // []) | length) > 0)
    | [.product, .name, ((.dropped_fields // []) | join(",")), ((.redacted_fields // []) | join(","))]
    | @tsv
  ' "$report")

  if ((found == 0)); then
    pass "redaction report has no dropped or redacted field entries"
  fi
}

validate_file_mode() {
  local label="$1"
  local path="$2"
  local want="$3"
  local got

  if [[ ! -e "$path" ]]; then
    fail "$label missing: $path"
    return
  fi

  got="$(mode_of "$path")"
  if [[ "$got" != "$want" ]]; then
    fail "$label mode = $got, want $want: $path"
    return
  fi
  pass "$label mode is $want"
}

if [[ -n "${ZSCALERCTL_BIN:-}" ]]; then
  if ! command -v "$ZSCALERCTL_BIN" >/dev/null 2>&1; then
    echo "zscalerctl binary not found or not executable: $ZSCALERCTL_BIN" >&2
    exit 2
  fi
  cli=( "$ZSCALERCTL_BIN" )
else
  cli=( go run -mod=vendor ./cmd/zscalerctl )
fi

need jq

if ((skip_credential_check)); then
  info "credential preflight skipped for fake CLI validation"
else
  if family="$(credential_family)"; then
    credential_family_name="$family"
    pass "live credential preflight found $family credentials"
  else
    message="no supported live credentials configured; set explicit zscalerctl OneAPI or ZIA legacy env vars"
    if ((require_credentials)); then
      fail "$message"
      exit 1
    fi
    skip "$message"
    exit 0
  fi
fi

if [[ -z "$out_dir" ]]; then
  out_dir="$(mktemp -d "${TMPDIR:-/tmp}/zscalerctl-live-smoke.XXXXXX")"
else
  if [[ -e "$out_dir" ]] && [[ -n "$(find "$out_dir" -mindepth 1 -maxdepth 1 -print -quit)" ]]; then
    echo "output directory already exists and is not empty: $out_dir" >&2
    exit 2
  fi
  mkdir -p "$out_dir"
fi
chmod 700 "$out_dir"

lists_dir="$out_dir/lists"
mkdir -p "$lists_dir"
chmod 700 "$lists_dir"

info "artifacts: $out_dir"
info "using CLI: ${cli[*]}"

schema_file="$lists_dir/schema.json"
schema_stderr="$lists_dir/schema.stderr"
if ! load_resources "$schema_file" "$schema_stderr"; then
  fail "live smoke cannot continue without a valid schema resource set"
  exit 1
fi
if ! validate_selected_product_credentials; then
  fail "live smoke cannot continue without product-specific credential metadata"
  exit 1
fi

expected_paths_file="$lists_dir/expected-dump-paths.txt"
write_expected_dump_paths "$expected_paths_file"

dump_products="$(printf '%s\n' "${resources[@]}" | awk -F/ '!seen[$1]++ {print $1}' | paste -sd ',' -)"
dump_resources="$(printf '%s\n' "${resources[@]}" | paste -sd ',' -)"

for resource_key in "${resources[@]}"; do
  product="${resource_key%%/*}"
  resource="${resource_key#*/}"
  stdout_file="$lists_dir/${product}-${resource}.json"
  stderr_file="$lists_dir/${product}-${resource}.stderr"

  if "${cli[@]}" --format json "$product" "$resource" list >"$stdout_file" 2>"$stderr_file"; then
    pass "$product $resource list command completed"
  else
    fail "$product $resource list command failed; stderr captured at $stderr_file"
    continue
  fi

  if validate_json_array "$product $resource list" "$stdout_file"; then
    jq 'length' "$stdout_file" >"$lists_dir/${product}-${resource}.count"
    validate_no_denied_keys "$product $resource list" "$stdout_file"
    validate_catalog_subset "$product $resource list" "$product" "$resource" "$stdout_file" "$schema_file"
    summarize_redaction_markers "$product $resource list" "$stdout_file"
  fi
done

dump_dir="$out_dir/dump"
dump_stdout="$out_dir/dump.stdout"
dump_stderr="$out_dir/dump.stderr"

if "${cli[@]}" dump --products "$dump_products" --resources "$dump_resources" --out "$dump_dir" >"$dump_stdout" 2>"$dump_stderr"; then
  pass "dump command completed for selected resources"
else
  fail "dump command failed for selected resources; stderr captured at $dump_stderr"
fi

if [[ -d "$dump_dir" ]]; then
  validate_file_mode "dump root directory" "$dump_dir" "700"
  validate_file_mode "dump resources directory" "$dump_dir/resources" "700"
  while IFS= read -r product; do
    validate_file_mode "dump $product directory" "$dump_dir/resources/$product" "700"
  done < <(printf '%s\n' "${resources[@]}" | awk -F/ '!seen[$1]++ {print $1}')

  manifest="$dump_dir/manifest.json"
  report="$dump_dir/redaction_report.json"
  if [[ -f "$manifest" ]]; then
    validate_file_mode "dump manifest" "$manifest" "600"
    if jq empty "$manifest" >/dev/null 2>&1; then
      pass "dump manifest is valid JSON"
      if jq -e --arg warning "$manifest_warning" '.warning == $warning' "$manifest" >/dev/null; then
        pass "dump manifest includes confidentiality warning"
      else
        fail "dump manifest missing confidentiality warning"
      fi
      if jq -e '.status == "complete"' "$manifest" >/dev/null; then
        pass "dump manifest status is complete"
      else
        fail "dump manifest status is not complete"
      fi
      if jq -e '(.errors // 0) == 0 and (.errors_path // "") == ""' "$manifest" >/dev/null; then
        pass "dump manifest has no partial-error metadata"
      else
        fail "dump manifest includes unexpected partial-error metadata"
      fi
      validate_manifest_resource_set "$manifest" "$expected_paths_file" "$lists_dir/manifest-dump-paths.txt" "$lists_dir/manifest-dump-paths.diff"
    else
      fail "dump manifest is not valid JSON: $manifest"
    fi
  else
    fail "dump manifest missing: $manifest"
  fi

  if [[ -f "$report" ]]; then
    validate_file_mode "redaction report" "$report" "600"
    if jq empty "$report" >/dev/null 2>&1; then
      pass "redaction report is valid JSON"
      if grep -q '<REDACTED:' "$report"; then
        fail "redaction report contains redaction marker values"
      else
        pass "redaction report is value-free"
      fi
      summarize_redaction_report "$report"
    else
      fail "redaction report is not valid JSON: $report"
    fi
  else
    fail "redaction report missing: $report"
  fi

  if [[ -e "$dump_dir/errors.ndjson" ]]; then
    fail "complete dump unexpectedly wrote errors.ndjson"
  else
    pass "complete dump did not write errors.ndjson"
  fi

  validate_dump_file_set "$dump_dir" "$expected_paths_file" "$lists_dir/actual-dump-paths.txt" "$lists_dir/actual-dump-paths.diff"

  for resource_key in "${resources[@]}"; do
    product="${resource_key%%/*}"
    resource="${resource_key#*/}"
    file="$dump_dir/resources/$product/${resource}.json"
    if [[ ! -f "$file" ]]; then
      fail "dump resource file missing: $file"
      continue
    fi
    validate_file_mode "dump $product $resource file" "$file" "600"
    if validate_json_array "dump $product $resource" "$file"; then
      jq 'length' "$file" >"$lists_dir/dump-${product}-${resource}.count"
      validate_no_denied_keys "dump $product $resource" "$file"
      validate_catalog_subset "dump $product $resource" "$product" "$resource" "$file" "$schema_file"
      summarize_redaction_markers "dump $product $resource" "$file"
      if [[ -f "$lists_dir/${product}-${resource}.count" ]]; then
        compare_counts "$product" "$resource" "$(cat "$lists_dir/${product}-${resource}.count")" "$(cat "$lists_dir/dump-${product}-${resource}.count")"
      fi
    fi
  done

  if [[ -f "$manifest" ]]; then
    while IFS=$'\t' read -r rel_path want_records; do
      file="$dump_dir/$rel_path"
      if [[ ! -f "$file" ]]; then
        fail "manifest references missing resource file: $rel_path"
        continue
      fi
      got_records="$(jq 'length' "$file")"
      if [[ "$got_records" == "$want_records" ]]; then
        pass "manifest count matches $rel_path ($got_records records)"
      else
        fail "manifest count for $rel_path = $want_records, file has $got_records"
      fi
    done < <(jq -r '.resources[]? | [.path, (.records|tostring)] | @tsv' "$manifest" 2>/dev/null || true)
  fi
fi

if ((failures != 0)); then
  fail "live smoke completed with $failures failure(s); artifacts kept at $out_dir"
  exit 1
fi

pass "live smoke completed; artifacts kept at $out_dir"
