#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

allowlist="${ZSCALERCTL_DEADCODE_ALLOWLIST:-scripts/deadcode-allowlist.txt}"
deadcode_version="${DEADCODE_VERSION:-v0.38.0}"

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT
report="$tmp_dir/deadcode.txt"

# Build-tagged files make code reachable on one platform and dead on
# another, so the report is the union over every target platform.
target_goos="${ZSCALERCTL_DEADCODE_GOOS:-linux darwin windows}"

# ZSCALERCTL_DEADCODE_OUTPUT replays a saved report (for the self-test).
if [[ -n "${ZSCALERCTL_DEADCODE_OUTPUT:-}" ]]; then
  cat "$ZSCALERCTL_DEADCODE_OUTPUT" >"$report"
else
  if [[ -z "${target_goos// /}" ]]; then
    echo "verify-deadcode: ZSCALERCTL_DEADCODE_GOOS names no platform" >&2
    exit 1
  fi
  # Install the tool for the host once, whatever GOOS the caller set; GOOS
  # then selects only the analyzed build configuration.
  GOOS="$(go env GOHOSTOS)" GOARCH="$(go env GOHOSTARCH)" GOBIN="$tmp_dir" \
    go install "golang.org/x/tools/cmd/deadcode@$deadcode_version"
  : >"$tmp_dir/raw.txt"
  for goos in $target_goos; do
    GOOS="$goos" "$tmp_dir/deadcode" -test ./... >>"$tmp_dir/raw.txt"
  done
  sort -u "$tmp_dir/raw.txt" >"$report"
fi

if [[ ! -f "$allowlist" ]]; then
  echo "missing deadcode allowlist: $allowlist" >&2
  exit 1
fi

# "path:line:col: unreachable func: Name" -> "path Name"
hits=()
while IFS= read -r line || [[ -n "$line" ]]; do
  [[ "$line" =~ ^([^:]+):[0-9]+:[0-9]+:\ unreachable\ func:\ (.+)$ ]] || {
    echo "verify-deadcode: unexpected deadcode output: $line" >&2
    exit 1
  }
  hits+=("${BASH_REMATCH[1]} ${BASH_REMATCH[2]}")
done <"$report"

entries=()
while IFS= read -r line || [[ -n "$line" ]]; do
  line="${line%%#*}"
  read -r file_glob func_glob extra <<<"$line" || true
  [[ -z "${file_glob:-}" ]] && continue
  if [[ -z "${func_glob:-}" || -n "${extra:-}" ]]; then
    echo "verify-deadcode: malformed allowlist entry: $line" >&2
    exit 1
  fi
  entries+=("$file_glob $func_glob")
done <"$allowlist"

fail=0
used=()
for entry in "${entries[@]+"${entries[@]}"}"; do
  used+=(0)
done
for hit in "${hits[@]+"${hits[@]}"}"; do
  file="${hit%% *}"
  func="${hit#* }"
  allowed=0
  for i in "${!entries[@]}"; do
    entry="${entries[$i]}"
    # shellcheck disable=SC2053 # the allowlist holds glob patterns
    if [[ "$file" == ${entry%% *} && "$func" == ${entry#* } ]]; then
      allowed=1
      used[i]=1
    fi
  done
  if (( allowed == 0 )); then
    if (( fail == 0 )); then
      echo "unreachable code (delete it, or add it to $allowlist with a reason):" >&2
    fi
    echo "  $file: $func" >&2
    fail=1
  fi
done

for i in "${!entries[@]}"; do
  if (( used[i] == 0 )); then
    echo "stale deadcode allowlist entry (nothing matches): ${entries[$i]}" >&2
    fail=1
  fi
done

exit "$fail"
