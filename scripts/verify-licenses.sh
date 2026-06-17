#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

allowed="${ZSCALERCTL_ALLOWED_LICENSES:-Apache-2.0,BSD-2-Clause,BSD-3-Clause,MIT,ISC,MPL-2.0}"
target="${ZSCALERCTL_LICENSE_TARGET:-./cmd/zscalerctl}"
# go-licenses/v2 currently fails to classify go-jmespath even though the module
# includes an Apache-2.0 LICENSE file. Ignore only that package; its
# dependencies are still checked.
ignored="${ZSCALERCTL_LICENSE_IGNORE:-github.com/jmespath/go-jmespath}"

# go-licenses is pinned in tools/go.mod for Renovate visibility. Resolve the
# version from that module so the verifier and Renovate pin cannot drift.
tool_module="github.com/google/go-licenses/v2"
tool_version="$(cd tools && go list -m -f '{{.Version}}' "$tool_module")"
if [[ -z "$tool_version" || "$tool_version" == "<nil>" ]]; then
  echo "go-licenses version not found in tools/go.mod" >&2
  exit 1
fi

# GOFLAGS=-mod=mod lets this verifier execute the pinned tool without
# interacting with vendor/.
ignore_args=()
IFS=',' read -r -a ignored_packages <<< "$ignored"
for package in "${ignored_packages[@]}"; do
  if [[ -n "$package" ]]; then
    ignore_args+=(--ignore "$package")
  fi
done

GOFLAGS=-mod=mod go run "${tool_module}@${tool_version}" check "$target" --allowed_licenses="$allowed" "${ignore_args[@]}"
