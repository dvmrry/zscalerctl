#!/usr/bin/env bash
set -euo pipefail

bump="${1:-${ZSCALERCTL_BUMP:-}}"
allow_major_zero="${ZSCALERCTL_ALLOW_MAJOR_ZERO:-false}"

case "$bump" in
	patch | minor | major | none) ;;
	*)
		echo "usage: next-version.sh patch|minor|major|none" >&2
		exit 2
		;;
esac

if [[ "$bump" == "none" ]]; then
	exit 0
fi

latest_tag="$(git tag -l 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname | sed -n '1p')"
major=0
minor=0
patch=0
if [[ -n "$latest_tag" ]]; then
	if [[ ! "$latest_tag" =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
		echo "latest semver tag has invalid format: $latest_tag" >&2
		exit 1
	fi
	major="${BASH_REMATCH[1]}"
	minor="${BASH_REMATCH[2]}"
	patch="${BASH_REMATCH[3]}"
fi

case "$bump" in
	patch)
		patch=$((patch + 1))
		;;
	minor)
		minor=$((minor + 1))
		patch=0
		;;
	major)
		if [[ "$major" == "0" && "$allow_major_zero" != "true" ]]; then
			echo "major bump is reserved for post-1.0 releases; use minor for breaking 0.x changes" >&2
			exit 1
		fi
		major=$((major + 1))
		minor=0
		patch=0
		;;
esac

printf 'v%d.%d.%d\n' "$major" "$minor" "$patch"
