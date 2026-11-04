#!/usr/bin/env bash
set -euo pipefail

# heavy-check.sh [--post] runs the tests a private repo's pull requests leave
# out of CI to save Actions minutes: every package under the race detector
# (make race), a short pass of each fuzz target (make fuzz-smoke, FUZZTIME per
# target, default 1m) and, on a Linux host, the plain test suite (make test).
#
# Go builds into a throwaway cache deleted on exit, and runs the toolchain
# go.mod names, downloading it into that throwaway space when the installed Go
# is older, so a run leaves no build or module cache behind. It starts from the
# repo's fuzz seeds each time; the corpus a run grows is not kept.
#
# ZSCALERCTL_LINUX_HOST names the Linux host as an SSH destination, such as an
# ~/.ssh/config alias (never commit one). The working tree goes there as a git
# archive, into a temporary directory that also holds that run's Go toolchain
# and caches and is deleted afterwards. The host needs Nix, which supplies Go
# and make. Without the variable, the Linux run is skipped.
#
# With --post, the result for a clean, pushed HEAD is posted to GitHub as the
# local/heavy-check commit status, so the pull request shows it beside CI.
# --post requires the Linux run.

post=false
case "${1:-}" in
	"") ;;
	--post) post=true ;;
	*)
		echo "usage: heavy-check.sh [--post]" >&2
		exit 2
		;;
esac

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
sha="$(git rev-parse HEAD)"
context="local/heavy-check"
go_version="$(sed -n 's/^go //p' go.mod)"
linux_host="${ZSCALERCTL_LINUX_HOST:-}"

if [[ "$post" == true ]]; then
	if [[ -z "$linux_host" ]]; then
		echo "heavy-check: --post needs ZSCALERCTL_LINUX_HOST, since private-repo pull requests skip CI's Linux tests" >&2
		exit 1
	fi
	if [[ -n "$(git status --porcelain --untracked-files=no)" ]]; then
		echo "heavy-check: --post needs a clean tree, so the result is HEAD's" >&2
		exit 1
	fi
	repo="$(gh repo view --json nameWithOwner --jq .nameWithOwner)"
	if ! gh api "repos/$repo/commits/$sha" --silent 2>/dev/null; then
		echo "heavy-check: --post needs HEAD ($sha) pushed to $repo" >&2
		exit 1
	fi
fi

work="$(mktemp -d "${TMPDIR:-/tmp}/zscalerctl-heavy.XXXXXX")"
remote_dir=""
linux_pid=""
remote() {
	ssh -o BatchMode=yes "$linux_host" "$@"
}
# shellcheck disable=SC2329 # run by the EXIT trap
cleanup() {
	if [[ -n "$linux_pid" ]]; then
		kill "$linux_pid" 2>/dev/null || true
	fi
	if [[ -n "$remote_dir" ]]; then
		# Go makes its module cache read-only; make it writable to delete it.
		remote "chmod -R u+w '$remote_dir' && rm -rf '$remote_dir'" ||
			echo "heavy-check: could not remove $remote_dir on the Linux host" >&2
	fi
	# Go makes its module cache read-only; make it writable to delete it.
	chmod -R u+w "$work" 2>/dev/null || true
	rm -rf "$work"
}
trap cleanup EXIT
trap 'exit 130' INT TERM

# linux_test runs make test on the Linux host with the toolchain go.mod names.
linux_test() {
	local tree
	tree="$(git stash create)"
	git archive "${tree:-HEAD}" | remote "mkdir '$remote_dir/src' && tar -x -C '$remote_dir/src'"
	remote "cd '$remote_dir/src' && env GOTOOLCHAIN='go$go_version' GOCACHE='$remote_dir/cache' GOMODCACHE='$remote_dir/mod' GOPATH='$remote_dir/gopath' XDG_CONFIG_HOME='$remote_dir/config' nix shell nixpkgs#go nixpkgs#gnumake -c make test"
}

post_status() {
	local state="$1" description="$2"
	[[ "$post" == true ]] || return 0
	gh api "repos/$repo/statuses/$sha" --silent \
		-f state="$state" -f context="$context" -f description="$description"
	echo "heavy-check: posted $context $state for $sha"
}

started="$(date +%s)"
if [[ -n "$linux_host" ]]; then
	# shellcheck disable=SC2016 # TMPDIR expands on the Linux host
	remote_dir="$(remote 'mktemp -d "${TMPDIR:-/tmp}/zscalerctl-heavy.XXXXXX"')"
	linux_test >"$work/linux.log" 2>&1 &
	linux_pid=$!
else
	echo "heavy-check: ZSCALERCTL_LINUX_HOST is unset; skipping the Linux run" >&2
fi

status=0
export GOTOOLCHAIN="go$go_version" GOCACHE="$work/gocache" GOMODCACHE="$work/mod"
make race && make fuzz-smoke FUZZTIME="${FUZZTIME:-1m}" || status=$?

legs="race and fuzz ($(go env GOOS)/$(go env GOARCH))"
if [[ -n "$linux_pid" ]]; then
	linux_status=0
	wait "$linux_pid" || linux_status=$?
	linux_pid=""
	echo "== make test on the Linux host"
	cat "$work/linux.log"
	if (( linux_status != 0 )); then
		echo "heavy-check: the Linux run failed (exit $linux_status)" >&2
		status=1
	fi
	legs="$legs and Linux tests"
fi
minutes=$(( ($(date +%s) - started + 59) / 60 ))

if (( status == 0 )); then
	echo "heavy-check: PASS in ${minutes}m"
	post_status success "$legs passed in ${minutes}m"
else
	echo "heavy-check: FAIL in ${minutes}m" >&2
	post_status failure "$legs: a run failed; rerun make heavy-check"
fi
exit "$status"
