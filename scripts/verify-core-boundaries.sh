#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT
stdlib_imports_file="$tmp_dir/stdlib.imports"
go list std >"$stdlib_imports_file"

check_package() {
  local label="$1"
  local package="$2"
  local deps_file_env="$3"
  local forbidden_re="$4"
  local guidance="$5"
  local mode="${6:-deps}"
  local deps_file="$tmp_dir/${label//[^A-Za-z0-9]/_}.deps"
  local matches

  if [[ -n "${!deps_file_env:-}" ]]; then
    cat "${!deps_file_env}" >"$deps_file"
  else
    case "$mode" in
      deps)
        go list -deps -mod=vendor "$package" >"$deps_file"
        ;;
      imports)
        go list -f '{{range .Imports}}{{.}}{{"\n"}}{{end}}' -mod=vendor "$package" >"$deps_file"
        ;;
      *)
        echo "verify-core-boundaries: unknown check mode $mode" >&2
        exit 1
        ;;
    esac
  fi

  matches="$(grep -E "$forbidden_re" "$deps_file" || true)"
  if [[ -n "$matches" ]]; then
    echo "verify-core-boundaries: $label imports forbidden dependencies:" >&2
    sed 's/^/  /' <<<"$matches" >&2
    echo "$guidance" >&2
    exit 1
  fi
}

cli_rendering_re='github\.com/spf13/cobra|github\.com/charmbracelet/lipgloss|internal/(cli|output)'
raw_runtime_re='zscalerctl/internal/(config|credentials|secret|secretref|zscaler|runtime)'
cli_zscaler_re='^zscalerctl/internal/zscaler$'

check_package \
  "internal/resources" \
  "./internal/resources" \
  "ZSCALERCTL_RESOURCES_DEPS_FILE" \
  "(^|/)(${cli_rendering_re}|${raw_runtime_re})(/|$)" \
  "internal/resources must remain a safe catalog/projection seam: no CLI or rendering packages and no raw runtime, secret, credential, SDK adapter, or runtime facade packages."

check_package \
  "internal/browser" \
  "./internal/browser" \
  "ZSCALERCTL_BROWSER_DEPS_FILE" \
  "(^|/)(${cli_rendering_re}|${raw_runtime_re})(/|$)" \
  "internal/browser must remain an overlay-facing projected-record seam: no CLI or rendering packages and no raw config, secret, credential, or SDK adapter packages."

check_package \
  "internal/machine" \
  "./internal/machine" \
  "ZSCALERCTL_MACHINE_DEPS_FILE" \
  "(^|/)(${cli_rendering_re}|${raw_runtime_re})(/|$)" \
  "internal/machine must remain transport-neutral and projected-record only: no CLI or rendering packages and no raw config, secret, credential, or SDK adapter packages."

check_package \
  "internal/cli" \
  "./internal/cli" \
  "ZSCALERCTL_CLI_DEPS_FILE" \
  "${cli_zscaler_re}" \
  "internal/cli must route live Zscaler access through internal/runtime and must not import internal/zscaler directly." \
  "imports"

echo "verify-core-boundaries: PASS"
