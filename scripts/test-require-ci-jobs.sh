#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
checker="$repo_root/scripts/require-ci-jobs.sh"

# needs builds a toJSON(needs) object from job=result pairs.
needs() {
	local pair json='{}'
	for pair in "$@"; do
		json="$(jq -c --arg job "${pair%%=*}" --arg result "${pair#*=}" '.[$job] = {result: $result, outputs: {}}' <<<"$json")"
	done
	printf '%s' "$json"
}

accepts() {
	if ! "$checker" "$@" >/dev/null 2>&1; then
		echo "require-ci-jobs rejected: $*" >&2
		exit 1
	fi
}

rejects() {
	if "$checker" "$@" >/dev/null 2>&1; then
		echo "require-ci-jobs accepted: $*" >&2
		exit 1
	fi
}

accepts "$(needs unit=success race=success)"
accepts "$(needs unit=success windows=skipped)" windows

rejects "$(needs unit=success race=failure)"
rejects "$(needs unit=cancelled)"
# A skip passes only for a job the run named as out of scope.
rejects "$(needs unit=success windows=skipped)"
rejects "$(needs unit=skipped windows=skipped)" windows
# An out-of-scope job that ran anyway means the scope and the job disagree.
rejects "$(needs unit=success windows=success)" windows
rejects "$(needs unit=success)" windows
rejects '{}'
rejects ''
rejects 'not json'
rejects

echo "test-require-ci-jobs: PASS"
