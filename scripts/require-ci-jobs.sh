#!/usr/bin/env bash
set -euo pipefail

# require-ci-jobs.sh NEEDS_JSON [OUT_OF_SCOPE_JOB...] passes only when every
# job in NEEDS_JSON, the workflow's toJSON(needs), succeeded, except the jobs
# named after it: the run left those out on purpose, and each must have been
# skipped. Naming them keeps a job that was skipped by mistake from passing.

if (( $# < 1 )); then
	echo "usage: $0 NEEDS_JSON [OUT_OF_SCOPE_JOB...]" >&2
	exit 2
fi
needs_json="$1"
shift

if ! jq -e 'type == "object" and length > 0' <<<"$needs_json" >/dev/null 2>&1; then
	echo "no required CI job results were provided" >&2
	exit 1
fi

for job in "$@"; do
	if ! jq -e --arg job "$job" 'has($job)' <<<"$needs_json" >/dev/null; then
		echo "out-of-scope job is not a required job: $job" >&2
		exit 1
	fi
done

failed=0
while IFS=$'\t' read -r job result; do
	want=success
	for out in "$@"; do
		if [[ "$job" == "$out" ]]; then
			want=skipped
		fi
	done
	echo "$job: $result"
	if [[ "$result" != "$want" ]]; then
		echo "required CI job $job: $result, want $want" >&2
		failed=1
	fi
done < <(jq -r 'to_entries[] | [.key, .value.result] | @tsv' <<<"$needs_json")
exit "$failed"
