#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

fake_bin="$tmp_dir/zscalerctl"
without_live_creds=(
  env
  -u ZSCALERCTL_AUTH_MODE
  -u ZSCALERCTL_CLIENT_ID
  -u ZSCALERCTL_CLIENT_SECRET
  -u ZSCALERCTL_CLIENT_SECRET_FILE
  -u ZSCALERCTL_VANITY_DOMAIN
  -u ZSCALERCTL_ZIA_USERNAME
  -u ZSCALERCTL_ZIA_PASSWORD
  -u ZSCALERCTL_ZIA_PASSWORD_FILE
  -u ZSCALERCTL_ZIA_API_KEY
  -u ZSCALERCTL_ZIA_API_KEY_FILE
  -u ZSCALERCTL_ZIA_CLOUD
)

cat >"$fake_bin" <<'SH'
#!/usr/bin/env bash
set -euo pipefail

mode="${ZSCALERCTL_FAKE_MODE:-good}"
resources=(gre-tunnels locations rule-labels static-ips)

write_schema() {
  local resource

  printf '[\n'
  for resource in "${resources[@]}"; do
    if [[ "$resource" != "${resources[0]}" ]]; then
      printf ',\n'
    fi
    printf '  {"product":"zia","name":"%s","operations":[{"name":"list","capability":"read"},{"name":"get","capability":"read"}],"fields":[]}' "$resource"
  done
  printf '\n]\n'
}

write_resource() {
  local resource="$1"
  case "$mode:$resource" in
    leaky:locations)
      printf '[{"id":1,"name":"HQ","preSharedKey":"plain-secret"}]\n'
      ;;
    invalid-json:gre-tunnels)
      printf '{"broken":'
      ;;
    *:locations)
      printf '[{"id":1,"name":"HQ","description":"<REDACTED:SECRET>","ipAddresses":["192.0.2.10"]}]\n'
      ;;
    *:rule-labels)
      printf '[{"id":2,"name":"Production","description":"","lastModifiedTime":1632411150,"referencedRuleCount":4}]\n'
      ;;
    *:static-ips)
      printf '[{"id":3,"ipAddress":"198.51.100.10","routableIP":true,"comment":""}]\n'
      ;;
    *:gre-tunnels)
      printf '[{"id":4,"sourceIp":"203.0.113.10","internalIpRange":"10.0.0.0/24","comment":"","withinCountry":true}]\n'
      ;;
    *)
      echo "unexpected resource: $resource" >&2
      exit 2
      ;;
  esac
}

write_dump() {
  local out=""
  shift
  while (($#)); do
    case "$1" in
      --products)
        shift 2
        ;;
      --out)
        out="$2"
        shift 2
        ;;
      *)
        echo "unexpected dump arg: $1" >&2
        exit 2
        ;;
    esac
  done
  if [[ -z "$out" ]]; then
    echo "missing --out" >&2
    exit 2
  fi

  mkdir -p "$out/resources/zia"
  chmod 700 "$out" "$out/resources" "$out/resources/zia"

  write_resource locations >"$out/resources/zia/locations.json"
  write_resource rule-labels >"$out/resources/zia/rule-labels.json"
  write_resource static-ips >"$out/resources/zia/static-ips.json"
  write_resource gre-tunnels >"$out/resources/zia/gre-tunnels.json"

  if [[ "$mode" == "missing-manifest-resource" ]]; then
    cat >"$out/manifest.json" <<'JSON'
{
  "schema": "zscalerctl.dump.manifest.v1",
  "redaction": "standard",
  "warning": "sanitized dumps remain confidential operational data",
  "resources": [
    {"product": "zia", "name": "locations", "path": "resources/zia/locations.json", "records": 1},
    {"product": "zia", "name": "rule-labels", "path": "resources/zia/rule-labels.json", "records": 1},
    {"product": "zia", "name": "static-ips", "path": "resources/zia/static-ips.json", "records": 1}
  ]
}
JSON
  else
    cat >"$out/manifest.json" <<'JSON'
{
  "schema": "zscalerctl.dump.manifest.v1",
  "redaction": "standard",
  "warning": "sanitized dumps remain confidential operational data",
  "resources": [
    {"product": "zia", "name": "locations", "path": "resources/zia/locations.json", "records": 1},
    {"product": "zia", "name": "rule-labels", "path": "resources/zia/rule-labels.json", "records": 1},
    {"product": "zia", "name": "static-ips", "path": "resources/zia/static-ips.json", "records": 1},
    {"product": "zia", "name": "gre-tunnels", "path": "resources/zia/gre-tunnels.json", "records": 1}
  ]
}
JSON
  fi

  cat >"$out/redaction_report.json" <<'JSON'
{
  "schema": "zscalerctl.redaction.report.v1",
  "redaction": "standard",
  "resources": [
    {
      "product": "zia",
      "name": "locations",
      "path": "resources/zia/locations.json",
      "records": 1,
      "included_fields": ["description", "id", "ipAddresses", "name"],
      "dropped_fields": ["vpnCredentials"],
      "redacted_fields": ["description"]
    }
  ]
}
JSON

  chmod 600 "$out"/manifest.json "$out"/redaction_report.json "$out"/resources/zia/*.json
  echo "dump written: $out"
}

if [[ "${1:-}" == "--format" && "${2:-}" == "json" && "${3:-}" == "schema" && "${4:-}" == "list" ]]; then
  write_schema
  exit 0
fi

if [[ "${1:-}" == "--format" ]]; then
  if [[ "${2:-}" != "json" || "${3:-}" != "zia" || "${5:-}" != "list" ]]; then
    echo "unexpected list args: $*" >&2
    exit 2
  fi
  write_resource "$4"
  exit 0
fi

if [[ "${1:-}" == "dump" ]]; then
  write_dump "$@"
  exit 0
fi

echo "unexpected args: $*" >&2
exit 2
SH
chmod +x "$fake_bin"

run_smoke() {
  local mode="$1"
  local out="$tmp_dir/out-$mode"
  local stdout="$tmp_dir/stdout-$mode"
  local stderr="$tmp_dir/stderr-$mode"

  if ZSCALERCTL_BIN="$fake_bin" ZSCALERCTL_FAKE_MODE="$mode" \
    "$repo_root/scripts/live-smoke.sh" --skip-credential-check --out "$out" >"$stdout" 2>"$stderr"; then
    return 0
  fi
  return 1
}

if ! "${without_live_creds[@]}" "$repo_root/scripts/live-smoke.sh" --out "$tmp_dir/out-skip" >"$tmp_dir/stdout-skip" 2>"$tmp_dir/stderr-skip"; then
  echo "live-smoke without credentials did not skip cleanly" >&2
  cat "$tmp_dir/stdout-skip" >&2
  cat "$tmp_dir/stderr-skip" >&2
  exit 1
fi

if ! grep -q '\[SKIP\] no supported live credentials configured' "$tmp_dir/stdout-skip"; then
  echo "live-smoke without credentials did not print SKIP marker" >&2
  cat "$tmp_dir/stdout-skip" >&2
  exit 1
fi

if "${without_live_creds[@]}" "$repo_root/scripts/live-smoke.sh" --require-credentials --out "$tmp_dir/out-require-creds" >"$tmp_dir/stdout-require-creds" 2>"$tmp_dir/stderr-require-creds"; then
  echo "live-smoke --require-credentials accepted missing credentials" >&2
  cat "$tmp_dir/stdout-require-creds" >&2
  cat "$tmp_dir/stderr-require-creds" >&2
  exit 1
fi

if ! grep -q '\[FAIL\] no supported live credentials configured' "$tmp_dir/stderr-require-creds"; then
  echo "live-smoke --require-credentials did not print missing-credentials failure" >&2
  cat "$tmp_dir/stderr-require-creds" >&2
  exit 1
fi

if "$repo_root/scripts/live-smoke.sh" --skip-credential-check --bin "$tmp_dir/missing-zscalerctl" --out "$tmp_dir/out-missing-bin" >"$tmp_dir/stdout-missing-bin" 2>"$tmp_dir/stderr-missing-bin"; then
  echo "live-smoke accepted a missing --bin path" >&2
  cat "$tmp_dir/stdout-missing-bin" >&2
  cat "$tmp_dir/stderr-missing-bin" >&2
  exit 1
fi

if ! grep -q 'zscalerctl binary not found or not executable' "$tmp_dir/stderr-missing-bin"; then
  echo "live-smoke missing-bin failure did not mention the binary path problem" >&2
  cat "$tmp_dir/stderr-missing-bin" >&2
  exit 1
fi

if ! run_smoke good; then
  echo "live-smoke rejected the good fixture" >&2
  cat "$tmp_dir/stdout-good" >&2
  cat "$tmp_dir/stderr-good" >&2
  exit 1
fi

if ! grep -q '\[PASS\] live smoke completed' "$tmp_dir/stdout-good"; then
  echo "live-smoke good fixture did not print final PASS marker" >&2
  cat "$tmp_dir/stdout-good" >&2
  exit 1
fi

if ! grep -q '\[PASS\] manifest count matches resources/zia/locations.json (1 records)' "$tmp_dir/stdout-good"; then
  echo "live-smoke good fixture did not validate manifest counts" >&2
  cat "$tmp_dir/stdout-good" >&2
  exit 1
fi

if ! grep -q '\[INFO\] redaction report zia locations: dropped fields \[vpnCredentials\]' "$tmp_dir/stdout-good"; then
  echo "live-smoke good fixture did not summarize dropped fields without record values" >&2
  cat "$tmp_dir/stdout-good" >&2
  exit 1
fi

if ! grep -q '\[PASS\] zia locations list and dump counts match (1 records)' "$tmp_dir/stdout-good"; then
  echo "live-smoke good fixture did not compare list and dump counts" >&2
  cat "$tmp_dir/stdout-good" >&2
  exit 1
fi

if ! grep -F -q '[INFO] zia locations list redaction markers at: [].description' "$tmp_dir/stdout-good"; then
  echo "live-smoke good fixture did not summarize list redaction marker paths" >&2
  cat "$tmp_dir/stdout-good" >&2
  exit 1
fi

if ! grep -F -q '[INFO] dump zia locations redaction markers at: [].description' "$tmp_dir/stdout-good"; then
  echo "live-smoke good fixture did not summarize dump redaction marker paths" >&2
  cat "$tmp_dir/stdout-good" >&2
  exit 1
fi

if ! grep -q '\[PASS\] dump manifest resource set matches ZIA catalog' "$tmp_dir/stdout-good"; then
  echo "live-smoke good fixture did not validate manifest resource set" >&2
  cat "$tmp_dir/stdout-good" >&2
  exit 1
fi

if ! grep -q '\[PASS\] dump resource files match ZIA catalog' "$tmp_dir/stdout-good"; then
  echo "live-smoke good fixture did not validate dump file set" >&2
  cat "$tmp_dir/stdout-good" >&2
  exit 1
fi

if run_smoke leaky; then
  echo "live-smoke accepted a fixture with a denied secret key" >&2
  cat "$tmp_dir/stdout-leaky" >&2
  cat "$tmp_dir/stderr-leaky" >&2
  exit 1
fi

if ! grep -q 'preSharedKey' "$tmp_dir/stderr-leaky"; then
  echo "live-smoke denied-key failure did not mention preSharedKey" >&2
  cat "$tmp_dir/stderr-leaky" >&2
  exit 1
fi

if run_smoke invalid-json; then
  echo "live-smoke accepted invalid JSON" >&2
  cat "$tmp_dir/stdout-invalid-json" >&2
  cat "$tmp_dir/stderr-invalid-json" >&2
  exit 1
fi

if ! grep -q 'is not a JSON array' "$tmp_dir/stderr-invalid-json"; then
  echo "live-smoke invalid-JSON failure did not mention JSON array validation" >&2
  cat "$tmp_dir/stderr-invalid-json" >&2
  exit 1
fi

if run_smoke missing-manifest-resource; then
  echo "live-smoke accepted a manifest missing a catalog resource" >&2
  cat "$tmp_dir/stdout-missing-manifest-resource" >&2
  cat "$tmp_dir/stderr-missing-manifest-resource" >&2
  exit 1
fi

if ! grep -q 'dump manifest resource set differs from ZIA catalog' "$tmp_dir/stderr-missing-manifest-resource"; then
  echo "live-smoke missing-manifest-resource failure did not mention resource-set drift" >&2
  cat "$tmp_dir/stderr-missing-manifest-resource" >&2
  exit 1
fi
