#!/usr/bin/env bash
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

tmp="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp"
}
trap cleanup EXIT

out="$tmp/scaffold"

bash scripts/scaffold-resource.sh \
  --product zpa \
  --resource fixture-resources \
  --package ./scripts/testdata/catalogdraft/fixture \
  --type Example \
  --out "$out" > "$tmp/stdout"

grep -q "resource scaffold written: $out" "$tmp/stdout"
test -f "$out/.zscalerctl-resource-scaffold"
test -f "$out/catalog-and-shape-review.txt"
test -f "$out/reader-wiring.md"
test -f "$out/docs.md"
test -f "$out/validation.md"
test -x "$out/command.sh"

grep -q "== internal/resources catalog seed ==" "$out/catalog-and-shape-review.txt"
grep -q "== internal/zscaler SDK shape-review seed ==" "$out/catalog-and-shape-review.txt"
grep -q 'Name:           "value"' "$out/catalog-and-shape-review.txt"
grep -q 'Classification: ClassSecret' "$out/catalog-and-shape-review.txt"
grep -q 'newListGetHandler\[fixture.Example\]' "$out/reader-wiring.md"
grep -q 'make live-smoke' "$out/validation.md"
grep -q 'The default live smoke currently validates ZIA resources' "$out/validation.md"
grep -q 'zscalerctl zpa fixture-resources list' "$out/validation.md"
grep -q '## ZPA Fixture Resources' "$out/docs.md"

if bash scripts/scaffold-resource.sh \
  --product zpa \
  --resource fixture-resources \
  --package ./scripts/testdata/catalogdraft/fixture \
  --type Example \
  --out "$out" > "$tmp/overwrite-stdout" 2> "$tmp/overwrite-stderr"; then
  echo "scaffold-resource overwrite unexpectedly succeeded" >&2
  exit 1
fi
grep -q "already exists" "$tmp/overwrite-stderr"

bash "$out/command.sh" > "$tmp/force-stdout"
grep -q "resource scaffold written: $out" "$tmp/force-stdout"

unsafe="$tmp/not-a-scaffold"
mkdir -p "$unsafe"
if bash scripts/scaffold-resource.sh \
  --product zpa \
  --resource fixture-resources \
  --package ./scripts/testdata/catalogdraft/fixture \
  --type Example \
  --out "$unsafe" \
  --force > "$tmp/unsafe-stdout" 2> "$tmp/unsafe-stderr"; then
  echo "scaffold-resource unsafe force unexpectedly succeeded" >&2
  exit 1
fi
grep -q "not a scaffold directory" "$tmp/unsafe-stderr"
