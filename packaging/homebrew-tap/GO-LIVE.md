# Homebrew tap — go-live checklist

Staged, inert, ready to flip on when `zscalerctl` is declared 1.0. Nothing here
is live: this directory changes no workflow and creates no repo. Decisions
already locked:

- Tap repo: **`dvmrry/homebrew-tap`** → `brew install dvmrry/tap/zscalerctl`
- Integrity: **Homebrew `sha256` pin only** in-formula; cosign/attestation stay
  the documented manual path (this is what sigstore's own tap does).
- Bumps: **auto-open PR, human-merge** (no unattended cross-repo auto-merge).
- Completions: bash/zsh/fish (no pwsh). Prebuilt tarball (no bottles).

The staged files in this directory:

| File | Goes to | At go-live |
| --- | --- | --- |
| `Formula/zscalerctl.rb` | `dvmrry/homebrew-tap` → `Formula/zscalerctl.rb` | seed the tap |
| `ci.yml` | `dvmrry/homebrew-tap` → `.github/workflows/ci.yml` | tap CI |
| `README.md` | `dvmrry/homebrew-tap` → `README.md` | tap landing page |
| `bump-homebrew.yml` | THIS repo → `.github/workflows/bump-homebrew.yml` | auto-bump on release |

The formula currently pins **v0.60.0**. If a newer release exists at go-live,
refresh `version` + the four `sha256` values from that release's `SHA256SUMS`
asset before seeding (or just let the first post-go-live release auto-bump it).

---

## Steps

### 1. Create + seed the tap repo (me or Codex)

```sh
gh repo create dvmrry/homebrew-tap --public \
  --description "Homebrew tap for zscalerctl"
# in a clone of the new repo:
mkdir -p Formula .github/workflows
cp <staging>/Formula/zscalerctl.rb Formula/zscalerctl.rb
cp <staging>/ci.yml               .github/workflows/ci.yml
cp <staging>/README.md            README.md
git add . && git commit -m "Add zscalerctl formula + tap CI" && git push
```

Tap CI (`brew style` + `brew audit --strict` + `brew install`/`brew test`) runs
on push and is the real validation of the formula.

### 2. Mint the cross-repo token (YOU — only you can)

A workflow in `dvmrry/zscalerctl` must write to `dvmrry/homebrew-tap` (a
different repo), so `GITHUB_TOKEN` is not enough. Mint a **fine-grained PAT**:

1. github.com → Settings → Developer settings → Personal access tokens →
   Fine-grained tokens → Generate new token.
2. Resource owner: `dvmrry`. Repository access: **Only `dvmrry/homebrew-tap`**.
3. Permissions: **Contents → Read and write**, **Pull requests → Read and write**.
4. Generate, copy.
5. In `dvmrry/zscalerctl` → Settings → Secrets and variables → Actions → New
   repository secret → name **`HOMEBREW_TAP_PAT`**, paste the token.

### 3. Wire auto-bump in this repo (Codex → review)

```sh
cp packaging/homebrew-tap/bump-homebrew.yml .github/workflows/bump-homebrew.yml
```

**Before merging, SHA-pin the actions** (this repo enforces `verify-actions-pinned.sh`,
which scans `.github/` — the staged file uses floating `@v3`/`@v4`/`@master` and will
fail the gate otherwise). Pin all three to a commit SHA with a trailing Renovate
version comment: `dawidd6/action-homebrew-bump-formula`, `actions/checkout`, and
`Homebrew/actions/setup-homebrew`. Pin at merge time (not now) so the SHAs aren't
stale by go-live. The tap repo's own `ci.yml` isn't subject to this gate, but pin it
too for hygiene.

Open a PR, confirm CI, merge. It triggers on `release: published`.

### 4. Add the Homebrew section to docs (same PR as step 3)

Append to `docs/INSTALL.md` (a new top-level section, e.g. after "Verify Release
Artifacts"):

```markdown
## Install With Homebrew

```sh
brew install dvmrry/tap/zscalerctl
```

Homebrew 6.0+ may prompt you to trust the third-party tap on first use. `brew
install` verifies each download against the `sha256` pinned in the formula; the
cosign bundle and provenance attestations above remain available for independent
verification outside Homebrew.
```

### 5. Smoke + cut 1.0.0

```sh
brew install dvmrry/tap/zscalerctl   # installs v0.60.0 (or current)
zscalerctl version
```

Then cut **v1.0.0** (Actions → release workflow → manual dispatch, `bump=major`;
`ALLOW_MAJOR_ZERO` is already plumbed). The `release: published` event fires the
auto-bump, which opens a formula-update PR in the tap → review → merge → `brew
install dvmrry/tap/zscalerctl` now serves 1.0.0. **The 1.0 cut is the first
end-to-end test of the bump pipeline.**

---

## The one thing to verify on first bump

This formula carries **four** `url`+`sha256` pairs (darwin/linux × amd64/arm64).
`dawidd6/action-homebrew-bump-formula` wraps `brew bump-formula-pr`, which is
rock-solid for single-artifact formulas but can under-update a multi-arch
formula (e.g. patch `version` + one `sha256`, miss the other three). Because the
bump opens a **reviewed PR**, a wrong bump can never auto-ship — but check the
first PR (the 1.0.0 bump) updates **all four** hashes.

If it doesn't, swap the auto-bump job for this custom fallback (fetches
`SHA256SUMS` and rewrites all four hashes deterministically):

```yaml
# .github/workflows/bump-homebrew.yml  (fallback — replaces the dawidd6 job)
name: bump-homebrew-tap
on:
  release:
    types: [published]
jobs:
  bump:
    runs-on: ubuntu-latest
    steps:
      - name: Compute hashes + open formula PR
        env:
          GH_TOKEN: ${{ secrets.HOMEBREW_TAP_PAT }}
          TAG: ${{ github.event.release.tag_name }}
        run: |
          set -euo pipefail
          ver="${TAG#v}"
          gh release download "$TAG" -R dvmrry/zscalerctl -p SHA256SUMS -O /tmp/SUMS
          h() { grep "zscalerctl_${ver}_$1.tar.gz\$" /tmp/SUMS | awk '{print $1}'; }
          da=$(h darwin_arm64); di=$(h darwin_amd64)
          la=$(h linux_arm64);  li=$(h linux_amd64)
          gh repo clone dvmrry/homebrew-tap /tmp/tap
          cd /tmp/tap
          f=Formula/zscalerctl.rb
          sed -i '' -E "s/version \"[^\"]+\"/version \"${ver}\"/" "$f"
          # Replace each sha256 by the comment marker on the line above, or by
          # matching the arch in the url — keep markers in the formula to make
          # this unambiguous. (See note below.)
          git switch -c "bump-${ver}"
          git commit -am "zscalerctl ${ver}"
          git push -u origin "bump-${ver}"
          gh pr create -f --title "zscalerctl ${ver}" --body "Automated bump."
```

> If you adopt the fallback, add `# arch:darwin_arm64` style trailing markers to
> each `sha256` line in the formula so `sed` can target them unambiguously. The
> dawidd6 path is preferred if it handles all four on the first try.

---

## Notes

- Homebrew **6.0.0** (2026-06-11) added mandatory tap-trust; the install docs
  above mention the one-time prompt. `Homebrew/actions/setup-homebrew` handles
  it transparently in CI, so `ci.yml` needs no special env.
- No `Casks/` dir in the tap — avoids the Linux `brew audit --tap=` cask-loader
  bug (Homebrew/brew#22148).
- Research sources are in the session transcript; key refs: docs.brew.sh
  Formula-Cookbook / Taps / Supply-Chain-Security, the 6.0.0 release notes,
  dawidd6/action-homebrew-bump-formula, and the cli/cli #9936 manual-verify
  precedent.
