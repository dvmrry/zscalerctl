#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

cat >"$tmp_dir/report.txt" <<'REPORT'
internal/fileperm/winpath.go:28:14: unreachable func: isUNCPath
internal/livesmoke/run.go:39:6: unreachable func: NewExecRunner
REPORT

cat >"$tmp_dir/allow.txt" <<'ALLOW'
# comment
internal/fileperm/*.go isUNC*  # windows only
internal/livesmoke/run.go NewExecRunner
ALLOW

run() {
  ZSCALERCTL_DEADCODE_OUTPUT="$1" ZSCALERCTL_DEADCODE_ALLOWLIST="$2" \
    bash "$repo_root/scripts/verify-deadcode.sh" 2>"$tmp_dir/stderr"
}

expect_pass() {
  local name="$1"
  if ! run "$2" "$3"; then
    echo "$name: want pass, got failure:" >&2
    cat "$tmp_dir/stderr" >&2
    exit 1
  fi
}

expect_fail() {
  local name="$1" want="$4"
  if run "$2" "$3"; then
    echo "$name: want failure, got pass" >&2
    exit 1
  fi
  if ! grep -qF -- "$want" "$tmp_dir/stderr"; then
    echo "$name: stderr missing '$want':" >&2
    cat "$tmp_dir/stderr" >&2
    exit 1
  fi
}

expect_pass "every hit allowed" "$tmp_dir/report.txt" "$tmp_dir/allow.txt"

: >"$tmp_dir/empty.txt"
expect_pass "nothing reported, nothing allowed" "$tmp_dir/empty.txt" "$tmp_dir/empty.txt"

cp "$tmp_dir/report.txt" "$tmp_dir/new-hit.txt"
echo "internal/runtime/status.go:41:6: unreachable func: NewStatusInspector" >>"$tmp_dir/new-hit.txt"
expect_fail "new dead function" "$tmp_dir/new-hit.txt" "$tmp_dir/allow.txt" \
  "internal/runtime/status.go: NewStatusInspector"

# A glob on the function must not admit a different file.
cp "$tmp_dir/report.txt" "$tmp_dir/other-file.txt"
echo "internal/cli/app.go:9:6: unreachable func: x.serverFrame" >>"$tmp_dir/other-file.txt"
expect_fail "function glob in another file" "$tmp_dir/other-file.txt" "$tmp_dir/allow.txt" \
  "internal/cli/app.go: x.serverFrame"

# An allowlisted file must not admit a different function.
cp "$tmp_dir/report.txt" "$tmp_dir/other-func.txt"
echo "internal/livesmoke/run.go:60:6: unreachable func: Other" >>"$tmp_dir/other-func.txt"
expect_fail "file glob for another function" "$tmp_dir/other-func.txt" "$tmp_dir/allow.txt" \
  "internal/livesmoke/run.go: Other"

# The last line counts without a trailing newline.
cp "$tmp_dir/report.txt" "$tmp_dir/no-newline.txt"
printf 'internal/livesmoke/run.go:60:6: unreachable func: Other' >>"$tmp_dir/no-newline.txt"
expect_fail "final line without newline" "$tmp_dir/no-newline.txt" "$tmp_dir/allow.txt" \
  "internal/livesmoke/run.go: Other"

cp "$tmp_dir/allow.txt" "$tmp_dir/stale.txt"
echo "internal/zscaler/reader.go ValidateReaderConfig  # gone" >>"$tmp_dir/stale.txt"
expect_fail "stale allowlist entry" "$tmp_dir/report.txt" "$tmp_dir/stale.txt" \
  "stale deadcode allowlist entry (nothing matches): internal/zscaler/reader.go ValidateReaderConfig"

cp "$tmp_dir/allow.txt" "$tmp_dir/malformed.txt"
echo "internal/livesmoke/run.go" >>"$tmp_dir/malformed.txt"
expect_fail "entry without a function" "$tmp_dir/report.txt" "$tmp_dir/malformed.txt" \
  "malformed allowlist entry"

cp "$tmp_dir/allow.txt" "$tmp_dir/unmarked-reason.txt"
echo "internal/livesmoke/run.go NewExecRunner make live-smoke" >>"$tmp_dir/unmarked-reason.txt"
expect_fail "reason without #" "$tmp_dir/report.txt" "$tmp_dir/unmarked-reason.txt" \
  "malformed allowlist entry"

cp "$tmp_dir/report.txt" "$tmp_dir/prefixed.txt"
echo "note: internal/cli/app.go:9:6: unreachable func: x" >>"$tmp_dir/prefixed.txt"
expect_fail "prefixed report line" "$tmp_dir/prefixed.txt" "$tmp_dir/allow.txt" \
  "unexpected deadcode output"

cp "$tmp_dir/report.txt" "$tmp_dir/garbled.txt"
echo "go: downloading golang.org/x/tools v0.38.0" >>"$tmp_dir/garbled.txt"
expect_fail "unexpected deadcode output" "$tmp_dir/garbled.txt" "$tmp_dir/allow.txt" \
  "unexpected deadcode output"

expect_fail "missing allowlist" "$tmp_dir/report.txt" "$tmp_dir/nope.txt" \
  "missing deadcode allowlist"

echo "test-verify-deadcode: PASS"
