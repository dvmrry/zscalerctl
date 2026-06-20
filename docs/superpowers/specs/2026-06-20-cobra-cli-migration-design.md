# Design: Migrate the zscalerctl CLI dispatch to Cobra

Status: **Design (approved in shape, pending written-spec review)**
Date: 2026-06-20
Owner: Dave (maintainer)
Precedes: a phased implementation plan (writing-plans), then a private `dvmrry/zscalerctl-dev` spike + clean phased PRs to `main`.

## 1. Context & motivation

`zscalerctl` is a read-only, security-first Go CLI (ZIA/ZPA/ZTW/ZCC/Zidentity). Its command dispatch is hand-rolled: `internal/cli/app.go` is **2,364 lines** of nested `case` switches (`doctor`/`auth`/`config`/`schema`/`dump`/`diff`/`list`/`get`/`show`/`version`/`completion`/`product`), plus a **449-line** hand-written 4-shell `completion.go`. Each new local command (`config init`, future `config validate`, `schema show`, `diff summarize`) adds another manual routing branch, another set of hand-written completion cases, and more drift between the README, the (nonexistent) manpages, the completions, and the actual surface.

This is a **pre-1.0** migration with no external consumers yet (0.x is a per-change counter). We have a clean revert point (the adversarial-review fixes just landed), and we want this done **before 1.0** and **well before** any TUI work. A CLI framework dissolves the hand-rolled routing and gives us per-command help, inherited persistent flags, generated completions, generated docs/manpages, standard flag/error hooks, and an introspectable command tree (valuable for agent-facing docs and generated references).

## 2. Framework decision: Cobra

We evaluated the real field for this project's needs (Go; machine-first/no-leak; wants 4-shell completion generation, Markdown/manpage doc-gen, and an introspectable tree):

| Framework | Completion gen | Doc/manpage gen | Introspection | Deps | Notes |
|---|---|---|---|---|---|
| **Cobra** (+pflag) | 4 shells | md/man/rest | yes | moderate | standard (kubectl/gh/docker/helm); imperative; auto-help we must silence |
| Kong | 3rd-party only | none built-in | struct *is* the model | light | declarative, less hidden behavior; but re-hand-rolls the generators |
| urfave/cli v3 | basic | md/man | partial | light–mod | dominated by Cobra for our goals |
| fang (Charm, wraps Cobra) | Cobra's | Cobra's | yes | heavy | styled output; would fight machine-first |

**Decision: Cobra.** The maintainer's top motivations — native multi-shell completion generation, Markdown/manpage doc-gen, the introspectable tree, and the largest/most-audited ecosystem — are precisely Cobra's strengths. The lighter alternatives (Kong) trade away exactly the generators we are trying to stop hand-rolling. Cobra's dependency weight is justified by its audit pedigree, and the project already ships a curated dependency set (Charm `lipgloss`/`termenv`/`x/term`, `zscaler-sdk-go`, `golang.org/x/sys`, `yaml.v3`) — so the posture is *deliberate* deps, not *zero* deps. Cobra (`spf13/cobra` + `spf13/pflag` + `inconshreveable/mousetrap` on Windows) joins that set as a single line-item we accept knowingly.

The one scenario that would have flipped the decision to Kong — prioritizing minimal-deps/declarative-auditability **above** the doc/completion generators — does not hold here, because the generators are half the motivation.

## 3. Scope & non-goals

**In scope** — a surface-preserving refactor *plus* Cobra's "free wins" adopted inline:
- Reimplement the **exact current command/flag surface** on Cobra.
- Adopt Cobra's no-extra-work wins during the move: per-command `--help`, inherited persistent flags, did-you-mean suggestions, standard unknown-flag handling, command aliases, and help `Example` blocks.
- Replace the hand-written completion with Cobra-generated completions + dynamic catalog hooks.
- Add Cobra doc/manpage generation + a CI drift check.

**Non-goals (explicitly deferred):**
- **No command renames / restructuring.** `auth status`, `doctor`, `version` keep natural verbs; `list/get/show` stay resource-only (honoring the locked "don't over-normalize" decision).
- **No braille spinners / animated "loaders."** Deferred to a later, TTY-gated, opt-in polish effort. The migration is structural; a spinner is TTY-only output that must never touch machine/agent/piped streams, and adding it mid-migration would muddy the "nothing regressed" proof. It slots in cleanly afterward, behind the same TTY gate as the pretty renderer.
- **No history reset / "tabula rasa" 1.0.** A post-migration, 1.0-time decision. If wanted, the clean-1.0 *face* is achieved by the README + `v1.0.0` tag/release (the repo "front page" is README + releases, not the commit log); if the maintainer additionally wants main's commit *line* fresh, an in-repo `archive/pre-1.0` tag at the pre-reset HEAD preserves every old commit, release tag, and memory SHA *in the same repo* (reachable, auditable, off the first-parent line). No second repo is needed for the archive. None of this gates the migration.
- **No behavior redesign** beyond Cobra's free wins. Anything that changes *what* a command does (vs how it's routed) is out.

## 4. Architecture: the command tree

Root command `zscalerctl` carries the **persistent (global) flags once**, inherited by all subcommands:
`--format`, `--profile`, `--config`, `--timeout`, `--redaction`, `--fields`, `--filter`, `--search`, `--output`, `--no-cache`, `--color`/`--no-color`, `--log-level`.

Subcommands (surface preserved):
- `list` / `get` / `show <resource>` — catalog-driven; resource names/keys supplied via Cobra `ValidArgsFunc` (dynamic completion).
- `dump`, `diff`
- `config` → `init`, `show` (room for future `validate`)
- `schema` → `show`
- `auth` → `status`
- `doctor`, `version`, `completion`
- `<product> help` topics

`zctl` remains a **thin alias** — one binary, the pretty persona selected by flipping the default output mode (the pretty renderer already lives behind the format switch). No second Cobra root; no duplicated command tree.

The command tree is constructed in small, focused files (one per command group) rather than one monolith, replacing the 2,364-line `app.go` dispatch. Each command group is independently understandable and testable.

## 5. No-leak & exit-code preservation (the load-bearing part)

Cobra's defaults conflict with the no-leak ethos: it prints errors and usage to its own writers and can drive `os.Exit`. We take that control back at a single chokepoint:

- **`SilenceErrors: true` + `SilenceUsage: true`** on the root command. Cobra never prints an error or usage block itself.
- Every command's `RunE` **returns** its error (never prints/exits inline).
- **One top-level handler** (wrapping `rootCmd.Execute()`) catches the returned error and routes it through the **existing redacting error writer** (`writeError` → `ScanRenderedString`). This means the no-leak invariant holds on *every* exit path — including unknown-flag and usage errors, which previously did not flow through the redactor and now do (a strict improvement).
- The **exit-code contract** (`0` ok, `1` internal, `2` usage/invalid-config, `3` credentials/missing-credentials, `4` not-found, `5` live-fail, `6` partial-dump, `7` drift) is mapped from the error type **at that single handler**, not via scattered `os.Exit` calls. Cobra/pflag flag-parse and unknown-command errors map to `2`.
- **`muteProcessOutput`** (stray stdout/stderr/log → `/dev/null` during execution) and the **machine-first stdout** discipline are unchanged. Cobra writes command output through our writers; help renders to stdout (since `--help` is an explicit request), errors via the handler to stderr.
- **Agentic default unchanged:** no color, no spinner, machine-first — color/pretty only when TTY + opted in (the `zctl`/`--format` path).

This handler is the single riskiest unit; Phase 1 proves it end-to-end on a tiny surface (`version` + `doctor`) before any bulk migration.

## 6. Completion

Cobra generates bash/zsh/fish/powershell from the command tree, retiring the 449-line hand-written `completion.go`. The **dynamic** parts — resource names/keys from the catalog — become `ValidArgsFunc` hooks on `list`/`get`/`show` (and any other catalog-aware args). Net: far less hand-written shell, equal-or-better dynamic completion, and no separate per-shell drift to maintain.

## 7. The inline "wins" (bounded)

Adopted during the migration because they are no-extra-work with Cobra and improve the surface without changing command behavior:
- Per-command `--help` from command metadata (replacing custom help branches).
- Inherited persistent/global flags (defined once on root).
- Did-you-mean suggestions for mistyped commands.
- Standard unknown-flag / unknown-command handling (mapped to exit `2`, redacted).
- Command aliases + `Example` blocks in help.

Anything beyond this list (deprecation frameworks, hidden-flag schemes, restructured commands) is a follow-up, not part of the migration.

## 8. Verification gates (both must stay green)

Because we adopt inline wins (so user-visible behavior intentionally shifts), "prove nothing regressed" needs a hard baseline:

1. **Golden CLI-surface snapshot.** Before any migration commit, capture each command's `--help`, representative output on fixed inputs, and exit code as committed `testdata`. A snapshot test then flags *every* delta, so each change is reviewed as **intentional** (Cobra-win) rather than **accidental** (regression). The snapshot is updated deliberately, per phase, with the diff reviewed.
2. **Agentic-coverage eval (DAV-10).** The migration must not lower the measured agent-friendliness floor; the introspectable command tree should *raise* it. The eval runs per phase as a gate.

The existing `internal/cli` test suite (currently `app_test.go` at 3,098 lines) is expected to pass essentially unchanged where it asserts behavior; tests that assert hand-rolled-dispatch internals are migrated alongside the code they cover.

## 9. Workflow

- **Spike sandbox:** a public `dvmrry/zscalerctl-dev` fork holds the messy, throwaway migration churn (proving the no-leak handler + Cobra wiring). It is open-source like the parent (no source secret to protect); the only discipline carried over is the existing one — no real tenant identifier / live-runtime artifact / real credential in committed fixtures, which is trivial for structural routing work. The sandbox is disposable (delete at 1.0).
- **Landing:** the validated approach for each phase lands as a **clean, golden-snapshot-gated phased PR in the main repo** (off current `main`), squash-merged like every other PR. The sandbox's churn never merges into `main`; `main`'s history stays clean. Drift is handled by periodically pulling `main` into the sandbox.

## 10. Phases (~6, each ships a working tool)

1. **Foundation.** Cobra root + all persistent flags + the §5 no-leak/exit handler, proven end-to-end on `version` + `doctor`. The golden-surface snapshot baseline lands here. Gate: snapshot + agentic eval + full suite.
2. **Resources.** `list`/`get`/`show` migrated, with dynamic catalog `ValidArgsFunc`.
3. **dump + diff.**
4. **config + schema + auth + product/help.**
5. **Completion overhaul.** Replace `completion.go` with Cobra-generated completions + the dynamic hooks; retire the hand-written file.
6. **Docs.** Cobra Markdown/manpage generation + a CI drift check reconciling README ⇄ manpage ⇄ completion ⇄ actual surface.

Each phase is an independently reviewable, individually-verifiable PR that leaves the tool fully working — matching the project's established phased pattern (config-profiles was 4 phases).

## 11. Risks & mitigations

- **No-leak/exit regression (highest risk).** Mitigation: the §5 handler is proven on `version`+`doctor` in Phase 1 before any bulk migration; the golden snapshot catches any stdout/stderr/exit-code drift; redaction-invariant tests cover the error paths.
- **Dependency addition.** Mitigation: Cobra is among the most-audited Go deps; vendored + pinned + covered by the existing `govulncheck`/`gitleaks`/license gates.
- **Dynamic completion fidelity.** Mitigation: `ValidArgsFunc` hooks tested against the catalog; completion behavior captured in the golden snapshot where feasible.
- **Doc/manpage drift.** Mitigation: generation is checked in CI (the Phase 6 drift check), so README/manpage/completion can't silently diverge from the surface.
- **Scope creep into redesign.** Mitigation: the §3 non-goals are explicit; the snapshot review surfaces any accidental behavior change for an intentional-or-revert decision.

## 12. Testing

- Golden CLI-surface snapshot (`testdata`) — per-command `--help`, fixed-input output, exit code; updated deliberately per phase.
- Existing `internal/cli` behavior tests pass; dispatch-internal tests migrate with their code.
- Per-command unit tests for the new command files; `RunE` error → exit-code mapping tests at the §5 handler.
- Redaction-invariant tests on the error path (including unknown-flag/usage errors now flowing through the redactor).
- Agentic-coverage eval (DAV-10) as a per-phase gate.
- `make check` (gofmt, vet, staticcheck, govulncheck, semgrep, gitleaks, verify-docs, verify-actions-pinned, sync-agents-skill, verify-release-artifacts) green per phase; `windows-config` CI exercises any platform-tagged paths.
