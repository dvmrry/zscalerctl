# dvmrry/homebrew-tap

Homebrew tap for [`zscalerctl`](https://github.com/dvmrry/zscalerctl) — a
read-only CLI for querying Zscaler tenant configuration (ZIA, ZPA, ZTW, ZCC,
Zidentity).

## Install

```sh
brew install dvmrry/tap/zscalerctl
```

Homebrew 6.0+ may prompt you to trust this third-party tap on first use; accept
to proceed.

## Verifying release integrity

`brew install` pins each download to the `sha256` recorded in the formula — that
is Homebrew's per-download integrity check.

For independent verification *outside* Homebrew, every `zscalerctl` release also
ships a cosign keyless signature bundle (`SHA256SUMS.bundle`) and GitHub build
provenance attestations. See
[docs/INSTALL.md § Verify Release Artifacts](https://github.com/dvmrry/zscalerctl/blob/main/docs/INSTALL.md#verify-release-artifacts)
for the `cosign verify-blob` and `gh attestation verify` commands.

## Updates

The `zscalerctl` formula is bumped automatically when a new release is published
upstream: a workflow opens a formula-update PR here, which is reviewed and merged.

## License

The `zscalerctl` tool is Apache-2.0. See the
[upstream LICENSE](https://github.com/dvmrry/zscalerctl/blob/main/LICENSE).
