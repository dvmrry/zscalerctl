#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

mkdir -p "$tmp_dir/good" "$tmp_dir/bad"

cat >"$tmp_dir/good/reader.go" <<'GO'
package zscaler

func service() {
	_, _ = zsdk.NewOneAPIClient(nil)
}
GO

cat >"$tmp_dir/bad/reader.go" <<'GO'
package zscaler

func service() {
	_, _ = zsdk.NewConfiguration()
	_, _ = zsdk.NewOneAPIClient(nil)
}
GO

ZSCALERCTL_ADAPTER_DIR="$tmp_dir/good" \
ZSCALERCTL_SKIP_GO_TEST=1 \
"$repo_root/scripts/verify-sdk-boundary.sh"

if ZSCALERCTL_ADAPTER_DIR="$tmp_dir/bad" \
	ZSCALERCTL_SKIP_GO_TEST=1 \
	"$repo_root/scripts/verify-sdk-boundary.sh" >"$tmp_dir/out" 2>"$tmp_dir/err"; then
	echo "verify-sdk-boundary accepted an adapter that calls NewConfiguration" >&2
	cat "$tmp_dir/out" >&2
	cat "$tmp_dir/err" >&2
	exit 1
fi

if ! grep -q "must not call SDK NewConfiguration" "$tmp_dir/err"; then
	echo "verify-sdk-boundary failed without the expected NewConfiguration message" >&2
	cat "$tmp_dir/err" >&2
	exit 1
fi
