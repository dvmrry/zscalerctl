#!/usr/bin/env bash
set -euo pipefail

# windows-ci-scope.sh BASE HEAD prints "true" when the change BASE...HEAD can
# behave differently on Windows, and "false" otherwise; ci.yml runs its Windows
# job only on "true", because Windows minutes bill at twice the Linux rate in a
# private repo. (ci.yml runs the job regardless in a public repo and outside
# pull requests.) A change is in scope when it touches:
#
#   - the dependencies, the CI workflow or this script,
#   - release notes, so a release PR always gets the Windows run,
#   - a package whose behaviour is filesystem- or OS-specific,
#   - a Go file constrained to an OS by its name or a //go:build line, or one
#     that branches on runtime.GOOS.
#
# The reason for "true" goes to stderr.

if (( $# != 2 )); then
	echo "usage: windows-ci-scope.sh BASE HEAD" >&2
	exit 2
fi
base="$1"
head="$2"

for ref in "$base" "$head"; do
	git rev-parse --verify -q "$ref^{commit}" >/dev/null || {
		echo "windows-ci-scope: unknown commit: $ref" >&2
		exit 1
	}
done
merge_base="$(git merge-base "$base" "$head")"

goos='aix|android|darwin|dragonfly|freebsd|hurd|illumos|ios|js|linux|netbsd|openbsd|plan9|solaris|wasip1|windows'

in_scope() {
	echo "windows-ci-scope: $1: $2" >&2
	echo true
	exit 0
}

# os_specific reports whether path at rev, if it exists there, is an OS
# constrained or OS branching Go file.
os_specific() {
	local rev="$1" path="$2" body
	body="$(git show "$rev:$path" 2>/dev/null)" || return 1
	grep -Eq "^//go:build.*(^|[^a-z0-9_])(unix|$goos)([^a-z0-9_]|$)" <<<"$body" ||
		grep -q 'runtime\.GOOS' <<<"$body"
}

while IFS= read -r path; do
	[[ -n "$path" ]] || continue
	case "$path" in
		go.mod | go.sum | vendor/* | .github/workflows/ci.yml | scripts/windows-ci-scope.sh)
			in_scope "$path" "build or CI input" ;;
		docs/releases/*)
			in_scope "$path" "release notes" ;;
		internal/dump/* | internal/effectcommit/* | internal/fileperm/* | internal/keyring/* | internal/secretref/*)
			in_scope "$path" "filesystem- or OS-specific package" ;;
	esac
	[[ "$path" == *.go ]] || continue
	if [[ "$path" =~ _($goos)(_[a-z0-9]+)?(_test)?\.go$ ]]; then
		in_scope "$path" "OS-specific file name"
	fi
	if os_specific "$head" "$path" || os_specific "$merge_base" "$path"; then
		in_scope "$path" "OS build constraint or runtime.GOOS"
	fi
done < <(git diff --no-renames --name-only "$merge_base" "$head")

echo false
