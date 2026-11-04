# Script Registry

This registry records the ownership and validation surface for files in
`scripts/`. It is intentionally boring: every top-level script must appear here
so helper code cannot become mystery project infrastructure.

The registry is checked by `scripts/verify-script-registry.sh`, which verifies
that every top-level file under `scripts/` appears in the table and that every
registered path exists.

| Script | Category | Called by | Validation |
| --- | --- | --- | --- |
| `scripts/deadcode-allowlist.txt` | data | `scripts/verify-deadcode.sh` | Stale or malformed entries fail `make verify-deadcode` |
| `scripts/catalog-draft.go` | dev | Manual `go run`; exercised by `scripts/test-catalog-draft.sh` | `make verify-catalog-draft` |
| `scripts/heavy-check.sh` | test | `make heavy-check` | Manual: runs `make race` and `make fuzz-smoke` here and `make test` on a Linux host, with throwaway Go caches |
| `scripts/live-smoke.go` | smoke | `make live-smoke`; manual live tenant validation | `go test ./internal/livesmoke/...` |
| `scripts/scaffold-resource.sh` | dev | `make scaffold-resource`; manual resource scaffolding | `scripts/test-scaffold-resource.sh` |
| `scripts/sdk-surface-inventory.go` | dev | `scripts/verify-sdk-ledger.sh`; manual SDK scouting with `go run` | `scripts/test-sdk-surface-inventory.sh` |
| `scripts/test-catalog-draft.sh` | test | `make verify-catalog-draft` | Self-contained fixture test |
| `scripts/test-gitleaks-allowlist.sh` | test | `make verify-gitleaks-allowlist`; `.github/workflows/ci.yml` | Generates an ephemeral key and proves private-key fixture exceptions cannot suppress it |
| `scripts/test-verify-deadcode.sh` | test | `make verify-deadcode`; `.github/workflows/ci.yml` | Self-contained verifier test (allowlisted, unallowed, stale, malformed and unexpected-output cases) |
| `scripts/test-verify-gitleaks-history-policy.sh` | test | `make verify-gitleaks-allowlist`; `.github/workflows/ci.yml` | Negative fixtures for shallow history and non-exact Gitleaks ignore fingerprints |
| `scripts/test-verify-go-toolchain.sh` | test | `make verify-go-toolchain`; `.github/workflows/ci.yml` | Negative fixtures for stale module and workflow Go security minimums |
| `scripts/test-require-ci-jobs.sh` | test | `make verify-ci-jobs`; `.github/workflows/ci.yml` | Self-contained stable CI aggregation result test |
| `scripts/test-windows-ci-scope.sh` | test | `make verify-ci-jobs`; `.github/workflows/ci.yml` | Self-contained Windows CI scope test |
| `scripts/test-release-version.sh` | test | `make verify-release-automation` | Self-contained release-version test |
| `scripts/test-scaffold-resource.sh` | test | `make verify-resource-scaffold` | Self-contained scaffold test |
| `scripts/test-sdk-surface-inventory.sh` | test | `make verify-sdk-surface-inventory` | Self-contained inventory test |
| `scripts/test-verify-sdk-ledger.sh` | test | `make verify-sdk-ledger`; `.github/workflows/ci.yml` | Self-contained verifier test (complete, empty, missing, unknown-verb, excluded-with-reads, invalid-disposition, imported, no-read and duplicate cases) |
| `scripts/test-verify-actions-pinned.sh` | test | `make verify-actions-pinned`; `.github/workflows/ci.yml` | Self-contained verifier test |
| `scripts/test-verify-ci-no-live-creds.sh` | test | `make verify-ci-no-live-creds`; `.github/workflows/ci.yml` | Self-contained verifier test |
| `scripts/test-verify-core-boundaries.sh` | test | `make verify-core-boundaries`; `.github/workflows/ci.yml` | Self-contained verifier test |
| `scripts/test-verify-machine-contract.sh` | test | `make verify-machine-contract`; `.github/workflows/ci.yml` | Self-contained verifier test |
| `scripts/test-verify-release-artifacts.sh` | test | `make verify-release-artifacts` | Self-contained verifier test |
| `scripts/test-verify-sdk-boundary.sh` | test | `make verify-sdk-boundary`; `.github/workflows/ci.yml` | Self-contained verifier test |
| `scripts/test-verify-script-registry.sh` | test | `make verify-script-registry` | Self-contained verifier test |
| `scripts/test-verify-surface-changes-manifest.sh` | test | `make verify-surface-changes-manifest`; `.github/workflows/ci.yml` | Self-contained verifier test for golden/manifest pairing |
| `scripts/verify-actions-pinned.sh` | verify | `make verify-actions-pinned`; `.github/workflows/ci.yml` | `scripts/test-verify-actions-pinned.sh` |
| `scripts/verify-ci-no-live-creds.sh` | verify | `make verify-ci-no-live-creds`; `.github/workflows/ci.yml` | `scripts/test-verify-ci-no-live-creds.sh` |
| `scripts/gen-cli-docs.go` | dev | `make gen-cli-docs`; manual CLI-reference regeneration | `scripts/verify-cli-docs.sh` |
| `scripts/test-verify-cli-docs.sh` | test | `make docs-cli-check`; `.github/workflows/ci.yml` | Self-contained verifier test (pass + corrupt-doc fail cases) |
| `scripts/verify-cli-docs.sh` | verify | `make docs-cli-check`; `.github/workflows/ci.yml` | `scripts/test-verify-cli-docs.sh` |
| `scripts/verify-core-boundaries.sh` | verify | `make verify-core-boundaries`; `.github/workflows/ci.yml` | `scripts/test-verify-core-boundaries.sh` |
| `scripts/verify-deadcode.sh` | verify | `make verify-deadcode`; `.github/workflows/ci.yml` | Fails on functions unreachable from `main` or tests on linux, darwin or windows (`ZSCALERCTL_DEADCODE_GOOS`) unless allowlisted with a reason; `scripts/test-verify-deadcode.sh` |
| `scripts/verify-docs.sh` | verify | `make docs-check`; `.github/workflows/ci.yml` | No companion test yet; docs secret-pattern gate |
| `scripts/verify-gitleaks-history-policy.sh` | verify | `scripts/test-gitleaks-allowlist.sh`; `make verify-gitleaks-allowlist` | Requires full history and exact commit-bound Gitleaks ignore fingerprints |
| `scripts/verify-go-toolchain.sh` | verify | `make verify-go-toolchain`; `.github/workflows/ci.yml` | Keeps every module and workflow on the root patch-level Go security minimum |
| `scripts/verify-sdk-ledger.sh` | verify | `make verify-sdk-ledger`; `.github/workflows/ci.yml` | Fails when a zscaler-sdk-go service package with read functions is neither imported nor named in `internal/resources/sdk_ledger.go`, or a ledger entry is stale; `scripts/test-verify-sdk-ledger.sh` |
| `scripts/verify-licenses.sh` | verify | `make verify-licenses`; `.github/workflows/ci.yml` | `go-licenses` allow-list check for the shipped binary |
| `scripts/verify-machine-manifest-schema.go` | verify | `scripts/verify-machine-contract.sh` | `scripts/test-verify-machine-contract.sh` |
| `scripts/verify-machine-contract.sh` | verify | `make verify-machine-contract`; `.github/workflows/ci.yml` | `scripts/test-verify-machine-contract.sh` |
| `scripts/verify-release-artifacts.sh` | verify | `make verify-release-artifacts` | `scripts/test-verify-release-artifacts.sh` |
| `scripts/verify-sdk-boundary.sh` | verify | `make verify-sdk-boundary`; `.github/workflows/ci.yml` | `scripts/test-verify-sdk-boundary.sh` |
| `scripts/verify-script-registry.sh` | verify | `make verify-script-registry` | `scripts/test-verify-script-registry.sh` |
| `scripts/verify-surface-changes-manifest.sh` | verify | `make verify-surface-changes-manifest`; `.github/workflows/ci.yml` | Requires changed CLI surface goldens to update `surface_changes.md`; self-tested by `scripts/test-verify-surface-changes-manifest.sh` |
| `scripts/verify-pty-escape-clean.py` | dev | `scripts/verify-pty-escape-clean.sh` | PTY helper for escape-sequence regression check |
| `scripts/verify-pty-escape-clean.sh` | verify | `make verify-pty-escape-clean` | Regression check: normal CLI output is clean in a PTY |
| `scripts/release-version.sh` | release | `.github/workflows/release-notes.yml`; `.github/workflows/release.yml` | `scripts/test-release-version.sh` |
| `scripts/require-ci-jobs.sh` | verify | `.github/workflows/ci.yml`; `scripts/test-require-ci-jobs.sh` | Fails the stable branch-protection check unless every prerequisite job succeeded or was named out of scope and skipped |
| `scripts/windows-ci-scope.sh` | verify | `.github/workflows/ci.yml`; `scripts/test-windows-ci-scope.sh` | Decides whether a private-repo pull request needs the Windows job |
| `scripts/verify-semgrep.sh` | verify | `make semgrep-check`; CI and release workflows | Semgrep rule fixtures under `semgrep/tests`; resolves the owner user-bin install, a PATH install, or pinned `uvx` fallback |
| `scripts/verify-workflow-policies.go` | verify | `scripts/verify-actions-pinned.sh`; `scripts/verify-go-toolchain.sh` | Shared YAML-aware action and setup-go policy traversal |

The testdata directory under `scripts/` holds fixtures for script tests and is
intentionally not a registry entry because the verifier tracks top-level script
files, not fixture directories.
