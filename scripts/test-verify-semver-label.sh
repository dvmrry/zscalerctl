#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

run_good() {
	local labels="$1"
	ZSCALERCTL_PR_LABELS="$labels" "$repo_root/scripts/verify-semver-label.sh"
}

run_bad() {
	local labels="$1"
	local want="$2"
	if ZSCALERCTL_PR_LABELS="$labels" "$repo_root/scripts/verify-semver-label.sh" >"$tmp_dir/out" 2>"$tmp_dir/err"; then
		echo "verify-semver-label accepted labels: $labels" >&2
		exit 1
	fi
	if ! grep -q "$want" "$tmp_dir/err"; then
		echo "verify-semver-label failed without expected message: $want" >&2
		cat "$tmp_dir/err" >&2
		exit 1
	fi
}

run_good "dependencies,semver:patch"
run_good "semver:minor"
run_good "semver:none"

run_bad "" "exactly one semver label"
run_bad "semver:patch,semver:minor" "exactly one semver label"
run_bad "semver:major" "reserved for post-1.0"

repo="$(mktemp -d "$tmp_dir/repo.XXXXXX")"
(
	cd "$repo"
	git init -q
	git config user.email "test@example.invalid"
	git config user.name "zscalerctl test"
	git commit --allow-empty -m initial >/dev/null
	git tag v1.0.0
	ZSCALERCTL_PR_LABELS="semver:major" "$repo_root/scripts/verify-semver-label.sh"
)
