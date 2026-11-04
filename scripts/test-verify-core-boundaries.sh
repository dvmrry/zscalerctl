#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

cat >"$tmp_dir/cli-good.deps" <<'EOF'
zscalerctl/internal/cli
zscalerctl/internal/config
zscalerctl/internal/output
zscalerctl/internal/runtime
EOF

cat >"$tmp_dir/browser-good.deps" <<'EOF'
zscalerctl/internal/browser
zscalerctl/internal/resources
zscalerctl/internal/redact
EOF

cat >"$tmp_dir/resources-good.deps" <<'EOF'
zscalerctl/internal/resources
zscalerctl/internal/redact
EOF

cat >"$tmp_dir/machine-good.deps" <<'EOF'
zscalerctl/internal/machine
EOF

cat >"$tmp_dir/cli-zscaler-bad.deps" <<'EOF'
zscalerctl/internal/cli
zscalerctl/internal/config
zscalerctl/internal/zscaler
EOF

cat >"$tmp_dir/browser-bad.deps" <<'EOF'
zscalerctl/internal/browser
zscalerctl/internal/cli
github.com/charmbracelet/lipgloss
github.com/spf13/cobra
EOF

cat >"$tmp_dir/browser-raw-bad.deps" <<'EOF'
zscalerctl/internal/browser
zscalerctl/internal/config
zscalerctl/internal/zscaler
EOF

cat >"$tmp_dir/resources-runtime-bad.deps" <<'EOF'
zscalerctl/internal/resources
zscalerctl/internal/runtime
EOF

cat >"$tmp_dir/machine-bad.deps" <<'EOF'
zscalerctl/internal/machine
zscalerctl/internal/output
github.com/charmbracelet/lipgloss
github.com/spf13/cobra
EOF

cat >"$tmp_dir/machine-raw-bad.deps" <<'EOF'
zscalerctl/internal/machine
zscalerctl/internal/credentials
zscalerctl/internal/secretref
zscalerctl/internal/zscaler
EOF

ZSCALERCTL_CLI_DEPS_FILE="$tmp_dir/cli-good.deps" \
ZSCALERCTL_RESOURCES_DEPS_FILE="$tmp_dir/resources-good.deps" \
ZSCALERCTL_BROWSER_DEPS_FILE="$tmp_dir/browser-good.deps" \
ZSCALERCTL_MACHINE_DEPS_FILE="$tmp_dir/machine-good.deps" \
  "$repo_root/scripts/verify-core-boundaries.sh" >/dev/null

if ZSCALERCTL_CLI_DEPS_FILE="$tmp_dir/cli-zscaler-bad.deps" \
  ZSCALERCTL_RESOURCES_DEPS_FILE="$tmp_dir/resources-good.deps" \
  ZSCALERCTL_BROWSER_DEPS_FILE="$tmp_dir/browser-good.deps" \
  ZSCALERCTL_MACHINE_DEPS_FILE="$tmp_dir/machine-good.deps" \
  "$repo_root/scripts/verify-core-boundaries.sh" >"$tmp_dir/cli-zscaler.out" 2>"$tmp_dir/cli-zscaler.err"; then
  echo "verify-core-boundaries accepted internal/cli dependencies on internal/zscaler" >&2
  cat "$tmp_dir/cli-zscaler.out" >&2
  cat "$tmp_dir/cli-zscaler.err" >&2
  exit 1
fi

if ! grep -q "internal/cli imports forbidden dependencies" "$tmp_dir/cli-zscaler.err"; then
  echo "verify-core-boundaries failed without the expected internal/cli zscaler boundary message" >&2
  cat "$tmp_dir/cli-zscaler.err" >&2
  exit 1
fi

if ZSCALERCTL_CLI_DEPS_FILE="$tmp_dir/cli-good.deps" \
  ZSCALERCTL_RESOURCES_DEPS_FILE="$tmp_dir/resources-runtime-bad.deps" \
  ZSCALERCTL_BROWSER_DEPS_FILE="$tmp_dir/browser-good.deps" \
  ZSCALERCTL_MACHINE_DEPS_FILE="$tmp_dir/machine-good.deps" \
  "$repo_root/scripts/verify-core-boundaries.sh" >"$tmp_dir/resources-runtime.out" 2>"$tmp_dir/resources-runtime.err"; then
  echo "verify-core-boundaries accepted resources dependencies on the runtime facade" >&2
  cat "$tmp_dir/resources-runtime.out" >&2
  cat "$tmp_dir/resources-runtime.err" >&2
  exit 1
fi

if ZSCALERCTL_CLI_DEPS_FILE="$tmp_dir/cli-good.deps" \
  ZSCALERCTL_RESOURCES_DEPS_FILE="$tmp_dir/resources-good.deps" \
  ZSCALERCTL_BROWSER_DEPS_FILE="$tmp_dir/browser-bad.deps" \
  ZSCALERCTL_MACHINE_DEPS_FILE="$tmp_dir/machine-good.deps" \
  "$repo_root/scripts/verify-core-boundaries.sh" >"$tmp_dir/browser.out" 2>"$tmp_dir/browser.err"; then
  echo "verify-core-boundaries accepted browser dependencies on CLI or rendering packages" >&2
  cat "$tmp_dir/browser.out" >&2
  cat "$tmp_dir/browser.err" >&2
  exit 1
fi

if ZSCALERCTL_CLI_DEPS_FILE="$tmp_dir/cli-good.deps" \
  ZSCALERCTL_RESOURCES_DEPS_FILE="$tmp_dir/resources-good.deps" \
  ZSCALERCTL_BROWSER_DEPS_FILE="$tmp_dir/browser-raw-bad.deps" \
  ZSCALERCTL_MACHINE_DEPS_FILE="$tmp_dir/machine-good.deps" \
  "$repo_root/scripts/verify-core-boundaries.sh" >"$tmp_dir/browser-raw.out" 2>"$tmp_dir/browser-raw.err"; then
  echo "verify-core-boundaries accepted browser dependencies on raw runtime packages" >&2
  cat "$tmp_dir/browser-raw.out" >&2
  cat "$tmp_dir/browser-raw.err" >&2
  exit 1
fi

if ZSCALERCTL_CLI_DEPS_FILE="$tmp_dir/cli-good.deps" \
  ZSCALERCTL_RESOURCES_DEPS_FILE="$tmp_dir/resources-good.deps" \
  ZSCALERCTL_BROWSER_DEPS_FILE="$tmp_dir/browser-good.deps" \
  ZSCALERCTL_MACHINE_DEPS_FILE="$tmp_dir/machine-bad.deps" \
  "$repo_root/scripts/verify-core-boundaries.sh" >"$tmp_dir/machine.out" 2>"$tmp_dir/machine.err"; then
  echo "verify-core-boundaries accepted machine dependencies on CLI or rendering packages" >&2
  cat "$tmp_dir/machine.out" >&2
  cat "$tmp_dir/machine.err" >&2
  exit 1
fi

if ZSCALERCTL_CLI_DEPS_FILE="$tmp_dir/cli-good.deps" \
  ZSCALERCTL_RESOURCES_DEPS_FILE="$tmp_dir/resources-good.deps" \
  ZSCALERCTL_BROWSER_DEPS_FILE="$tmp_dir/browser-good.deps" \
  ZSCALERCTL_MACHINE_DEPS_FILE="$tmp_dir/machine-raw-bad.deps" \
  "$repo_root/scripts/verify-core-boundaries.sh" >"$tmp_dir/machine-raw.out" 2>"$tmp_dir/machine-raw.err"; then
  echo "verify-core-boundaries accepted machine dependencies on raw runtime packages" >&2
  cat "$tmp_dir/machine-raw.out" >&2
  cat "$tmp_dir/machine-raw.err" >&2
  exit 1
fi

if ! grep -q "internal/resources imports forbidden dependencies" "$tmp_dir/resources-runtime.err"; then
  echo "verify-core-boundaries failed without the expected resources runtime-facade boundary message" >&2
  cat "$tmp_dir/resources-runtime.err" >&2
  exit 1
fi

if ! grep -q "internal/machine imports forbidden dependencies" "$tmp_dir/machine.err"; then
  echo "verify-core-boundaries failed without the expected machine boundary message" >&2
  cat "$tmp_dir/machine.err" >&2
  exit 1
fi

if ! grep -q "internal/browser imports forbidden dependencies" "$tmp_dir/browser.err"; then
  echo "verify-core-boundaries failed without the expected browser boundary message" >&2
  cat "$tmp_dir/browser.err" >&2
  exit 1
fi

if ! grep -q "internal/browser imports forbidden dependencies" "$tmp_dir/browser-raw.err"; then
  echo "verify-core-boundaries failed without the expected browser raw-runtime boundary message" >&2
  cat "$tmp_dir/browser-raw.err" >&2
  exit 1
fi

if ! grep -q "internal/machine imports forbidden dependencies" "$tmp_dir/machine-raw.err"; then
  echo "verify-core-boundaries failed without the expected machine raw-runtime boundary message" >&2
  cat "$tmp_dir/machine-raw.err" >&2
  exit 1
fi

