#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
script="$repo_root/scripts/windows-ci-scope.sh"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

cd "$tmp_dir"
git init -q
git config user.email "test@example.invalid"
git config user.name "zscalerctl test"
mkdir -p internal/cli internal/fileperm docs/releases
printf 'package cli\n' >internal/cli/plain.go
printf '//go:build windows\n\npackage cli\n' >internal/cli/console.go
printf 'package fileperm\n' >internal/fileperm/perm.go
printf '# zscalerctl\n' >README.md
git add .
git commit -q -m base
git tag base

# change PATH CONTENT commits PATH with CONTENT on a branch from base; an
# empty CONTENT deletes it.
change() {
	git checkout -q --detach base
	if [[ -z "$2" ]]; then
		git rm -q "$1"
	else
		mkdir -p "$(dirname "$1")"
		printf '%b' "$2" >"$1"
		git add "$1"
	fi
	git commit -q -m change
}

assert_scope() {
	local want="$1" label="$2" got
	got="$("$script" base HEAD 2>/dev/null)"
	if [[ "$got" != "$want" ]]; then
		echo "windows-ci-scope.sh = $got for $label, want $want" >&2
		exit 1
	fi
}

change README.md '# changed\n'
assert_scope false "a docs change"
change internal/cli/plain.go 'package cli\n\nfunc f() {}\n'
assert_scope false "a portable Go change"
change internal/cli/race_on_test.go '//go:build race\n\npackage cli\n'
assert_scope false "a race-only build constraint"
change internal/cli/plain.go 'package cli\n\nimport "runtime"\n\nvar onWindows = runtime.GOOS == "windows"\n'
assert_scope true "a runtime.GOOS branch"
change internal/cli/term.go '//go:build !windows\n\npackage cli\n'
assert_scope true "a !windows build constraint"
change internal/cli/term.go '//go:build darwin || linux\n\npackage cli\n'
assert_scope true "an OS list build constraint"
change internal/cli/term_unix.go '//go:build unix\n\npackage cli\n'
assert_scope true "a unix build constraint"
change internal/cli/acl_windows.go 'package cli\n'
assert_scope true "a _windows.go file"
change internal/cli/sys_linux_amd64_test.go 'package cli\n'
assert_scope true "an OS and arch test file"
change internal/cli/console.go ''
assert_scope true "deleting a windows-only file"
change internal/fileperm/perm.go 'package fileperm\n\nfunc g() {}\n'
assert_scope true "an OS-specific package"
change go.sum 'example.com/m v1.0.0 h1:x\n'
assert_scope true "a dependency change"
change docs/releases/v1.0.0.md 'Notes.\n'
assert_scope true "release notes"
change .github/workflows/ci.yml 'name: ci\n'
assert_scope true "the CI workflow"

if "$script" base >/dev/null 2>&1; then
	echo "windows-ci-scope.sh accepted one argument" >&2
	exit 1
fi
if "$script" base no-such-ref >/dev/null 2>&1; then
	echo "windows-ci-scope.sh accepted an unknown commit" >&2
	exit 1
fi

echo "test-windows-ci-scope: PASS"
