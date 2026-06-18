# Config Profiles + Pluggable Secret Providers — Design

**Status:** Approved-in-concept 2026-06-17 (brainstorming); revised 2026-06-17 per
implementation review. Pre-1.0. Implementation plan to follow (writing-plans).

**Revision (review pass):** introduced an explicit `SecretSource` type to resolve
the lazy-resolution vs. "same `Config`" contradiction; promoted Windows DACL
validation of the config file to a **phase-1 requirement**; made `cmd:` a
**structured** provider (`argv` array, no shell-like string) with a fetch timeout;
tightened `keyring:` segment rules and `ZSCALERCTL_DISALLOW_CMD` parsing.

## Problem

`zscalerctl` is configured strictly through `ZSCALERCTL_*` environment variables.
That is excellent for CI and isolated agent containers but cumbersome for local
operators and multi-tenant work: switching tenants means swapping the entire env
block, and there is no ergonomic way to delegate secrets to an OS keychain or an
external provider. This resolves the deferred multi-tenant/keychain finding and
the `THREAT_MODEL.md` "future keychain" item — without putting plaintext secrets
on disk and without regressing the env-only security posture.

## Goals

- Named **profiles** selectable with `--profile NAME`.
- Profiles hold **non-secrets plus secret *references* only** — a secret value is
  never written into a config file.
- **Pluggable secret providers:** `env:`, `file:`, `cmd:`, `keyring:` (cgo-free).
- Clean story for **both worlds:** headless/CI uses `env:`/`file:`/`cmd:`;
  desktop/userland can additionally use `keyring:`.
- **No automatic plaintext fallback** — fail closed and loud.
- **100% backward compatible:** env-only behavior is byte-for-byte unchanged, and
  env has the highest precedence.
- **Cross-platform**, Windows included (config-file DACL validation in phase 1).

## Non-goals (v1 — explicit scope boundaries)

- **No keyring *write* helper** (`zscalerctl keyring set`). Fast-follow. v1 *reads*
  from providers; the operator provisions secrets with their own OS tools. *(Decision A.)*
- **No native cgo macOS Keychain.** cgo-free only (`security` CLI on macOS).
- **No built-in SOPS/age.** Reachable through `cmd:` pointed at a script.
- **No plaintext fallback**, no `plain:` provider (secrets are never inline).
- **No config merging / includes / multiple config files.** One file, profiles within.
- **No new tenant-write capability.** Still read-only with respect to tenant state;
  `keyring:`/`cmd:` only *read* a local secret.

## Architecture

Extend the existing seam; do not rewrite. Today `config.LoadEnv(environ) → Config`
is a pure function. Add a config-file/profile layer that produces the **same**
`config.Config`, so the env-only path is unchanged and the new path is purely
additive.

### Precedence (highest wins)

```
flag  >  env (ZSCALERCTL_*)  >  active profile  >  built-in default
```

- Env always overrides the profile, so **CI/automation/agents keep working exactly
  as today**; the profile is purely a local convenience layer.
- Profile selection: `--profile NAME` > `ZSCALERCTL_PROFILE` > config `default_profile`.
- **No config file present ⇒ identical to today.**

### Config shape — `SecretSource` (resolves the lazy-vs-eager contradiction)

The credential secret fields on `Config` change from concrete `secret.Secret` to a
**`SecretSource`** — a small type that carries *provenance metadata* and can resolve
to a `secret.Secret` on demand:

```
type SecretSource interface {
    Scheme() string                 // "env" | "file" | "cmd" | "keyring" | "" (unset)
    IsConfigured() bool             // a source exists (not whether it resolves)
    Resolve(ctx context.Context) (secret.Secret, error)
}
```

- **Env-inline and env-file resolve eagerly at load** (preserves today's behavior,
  keeps fail-fast on a bad secret-file permission). Their `SecretSource` simply
  wraps the already-resolved `secret.Secret`; `Resolve` returns it.
- **Profile `*_ref` sources stay unresolved** — `Resolve` runs the provider
  (`cmd:`/`keyring:`/`file:`/`env:`) only when called.
- **`Resolve` is called exactly once, when `resourceReader` is constructed** (i.e.
  a live API call). `doctor` / `config show` / `schema list` / `version` /
  `completion` never call `Resolve`.

This is the core implementation shape: `LoadEnv`/`LoadConfig` return a `Config`
whose secret fields are `SecretSource`, not a mix of eager and lazy that pretends
to be the same concrete type.

### Secret-specific precedence (which source wins)

```
inline env (e.g. ZSCALERCTL_CLIENT_SECRET)
  > env file (e.g. ZSCALERCTL_CLIENT_SECRET_FILE)
  > profile *_ref (deferred SecretSource)
  > unset
```

The winning source becomes the field's `SecretSource`; only the winner is ever
resolved.

## Config file

- **Format:** YAML.
- **Location:** `$XDG_CONFIG_HOME/zscalerctl/config.yaml` (default
  `~/.config/zscalerctl/config.yaml`); overridable with `--config PATH` or
  `ZSCALERCTL_CONFIG`.
- **Permission validation, fail-closed, on every platform — phase 1:**
  - **POSIX (macOS/Linux):** reject group/world readable or writable (owner-only),
    same primitive as `*_FILE` secret files.
  - **Windows:** **DACL validation is required in phase 1** (not deferred). Accept
    the **current user, the file owner, `SYSTEM`, and `Administrators`**; **reject**
    broad principals such as `Everyone`, `Users`, `Authenticated Users`,
    `Domain Users`, or **any inherited grant that gives a non-owner interactive user
    read/write access**. Administrative access (`SYSTEM`/`Administrators`) is
    acceptable but **not sufficient on its own** — the file is rejected if any broad
    interactive principal is granted, regardless of the admin entries (so "standard
    inherited ACLs" are *not* automatically fine). It need not handle every exotic
    ACL; it must accept the normal owner-only case and reject the broad-grant and
    broad-inherited cases.
    Implemented via `golang.org/x/sys/windows` security-descriptor APIs (cgo-free).
  - This validation is enforced **even though the file holds no secrets**, because
    it holds `cmd:` argv (see threat model). The Windows DACL validator is written as
    a reusable primitive and is shared by the config loader and `*_FILE` secret files
    in phase 1.
- **Schema** (illustrative; full field set mirrors today's env vars per auth mode):

```yaml
default_profile: prod            # optional; else --profile / ZSCALERCTL_PROFILE required
profiles:
  prod:
    auth_mode: oneapi            # optional, default oneapi
    vanity_domain: example
    cloud: PRODUCTION
    client_id: "..."             # identifier, not a secret
    client_secret_ref: "keyring:zscalerctl/prod-client-secret"
    zpa_customer_id: "..."
    zpa_microtenant_id: "..."    # optional
    # operational defaults (only fields with existing env equivalents) — Decision B
    redaction: standard
    no_cache: false
  preprod:
    auth_mode: zia-legacy
    zia_username: "..."
    zia_password_ref: "file:/path/to/owner-only/pw"
    zia_api_key_ref:             # cmd is STRUCTURED, never a shell string
      cmd:
        argv: ["/usr/local/bin/get-zia-key", "--profile", "preprod"]
        timeout: 10s             # optional; overrides the 10s default; bounds the fetch
    zia_cloud: "..."
```

A **drift-gated JSON schema** for the config file ships under `docs/schema/`
(matching the existing dump/diff/error schema discipline).

## Secret references

A secret reference is **either a string** (simple schemes) **or a structured map**
(`cmd`). A custom `SecretRef` YAML unmarshaller accepts both forms and produces a
`SecretSource`.

| Form | Provider | Resolution | Notes |
|------|----------|------------|-------|
| `"env:NAME"` | env | read env var `NAME` | unset → error |
| `"file:/path"` | file | existing `ReadOwnerOnlySecretFile` (+ Windows DACL primitive) | strict permissions on every supported platform |
| `"keyring:service/key"` | keyring | cgo-free OS keychain | Linux D-Bus, Windows wincred syscall, macOS `security` CLI |
| `{cmd: {argv: [...], timeout: D}}` | cmd | exec `argv[0]` with `argv[1:]`, **no shell**; trimmed stdout is the secret | deterministic argv across OSes; no quoting/splitting; pipes/SOPS go in a script that `argv` points at |

Rules:
- **Unknown scheme / bare value with no scheme → clear error**
  ("secret refs require a provider scheme").
- **`keyring:` format:** exactly `keyring:<service>/<key>`. Split on the **first**
  `/`. Both segments are required and **must not be empty or contain `/`** (reject
  otherwise) — no escaping, no ambiguity.
- **`cmd:` is structured-only.** No `"cmd:prog args"` string form — that would
  require platform-dependent argv parsing. `argv` is a non-empty list; `argv[0]` is
  the executable; nothing is shell-interpreted.
- **`cmd:` timeout:** every `cmd:` resolution is bounded by a context timeout
  (per-ref `timeout`, else a built-in **default of 10s — final**). A hung provider
  can never hang a live read. The timeout is independent of `--timeout` (which bounds
  HTTP requests). 10s is enough for `op`/`pass`/`sops`/`az`/a local wrapper and short
  enough to avoid a "why is my list hung?" spiral.
- **No automatic fallback:** an unavailable `keyring:`/`cmd:` fails loud
  (`keyring backend unavailable on this host; use env:, file:, or cmd:`).
- `keyring:` sits behind a **mockable interface** so CI never touches a real
  keychain. The keychain is a desktop-interactive convenience on every OS (may
  prompt; locked/headless sessions fail) — which is why `env`/`file`/`cmd` remain
  the headless path.

## `cmd:` threat model (the one real surface expansion)

`cmd:` turns the config file into a code-execution vector. Mitigation chain,
written up as a dedicated `THREAT_MODEL.md` section:

1. **Config permissions are validated or we fail closed** — POSIX owner-only and
   Windows DACL (phase 1). Same gate, both platforms.
2. **`cmd:` execs a structured `argv` directly, never through a shell.** No
   shell-injection surface, no quoting ambiguity.
3. **Bounded by a fetch timeout** so a hung/forked provider can't wedge a live read.
4. **Reasoning:** if the config is writable only by you, only *you* can add a `cmd:`
   ref, and you could already run anything as yourself — so it grants no new
   privilege. The only escalation ("attacker can write your protected config")
   already implies the attacker owns your account. This is the AWS
   `credential_process` model.
5. **`ZSCALERCTL_DISALLOW_CMD` kill-switch** (opt-*out*) lets a hardened fleet forbid
   `cmd:` entirely. Parsed with the **existing bool-env parser** (consistent
   truthy/falsey handling); an invalid value fails **value-free** (`ErrInvalidConfig`
   without echoing the value, per the leak-safe error posture). No separate opt-in is
   required — the permission gate is the gate. *(Decision C.)*

## Lazy resolution & safe output

- Secrets resolve **only when a live reader is constructed**, via
  `SecretSource.Resolve`. `doctor`, `config show`, `schema list`, `version`,
  `completion` never resolve a secret — no keychain prompts, no `cmd:` exec, nothing
  surfaced.
- `config show` / `doctor` render from **`SecretSource` metadata**
  (`Scheme()` + `IsConfigured()`), never from a resolved value: e.g.
  `client secret source: keyring (configured)`. They also report the **active profile** and
  **config source** (env-only vs. the config-file path). Never values, never the
  full ref, never `cmd` argv.

## Module boundaries

- `internal/config` — add config-file + profile loading (`file.go`, `profile.go`)
  and the `SecretSource` type; `LoadEnv` stays and is layered under env. Returns a
  `Config` whose secret fields are `SecretSource`.
- `internal/secretref` — `SecretRef` parsing (string|structured) and the provider
  implementations (`env`/`file`/`cmd`/`keyring`), one small unit per scheme, each
  producing a `SecretSource`.
- `internal/keyring` — cgo-free backend behind a mockable interface.
- `internal/credentials` (or a small `internal/fileperm`) — the POSIX owner-only and
  **Windows DACL** validators, as a reusable primitive shared by the config loader
  and the `file:` provider.
- CLI wiring — `--profile` / `--config` flags, precedence resolution, and the single
  `Resolve` call at reader-build time.

## Error handling

Invalid config, bad permissions/DACL, unknown scheme, unavailable provider, a hung
provider (timeout), or a missing required secret map to `ErrInvalidConfig` (exit 2 —
usage) or the existing credential codes, each with a clear, actionable message.
Sensitive inputs (the `DISALLOW_CMD` value, secret values) are **never echoed**.
**Never a silent fallback.**

## Testing

- Table-driven provider tests (`env`/`file`/`cmd`/`keyring`) producing `SecretSource`.
- **`SecretSource` semantics:** env sources resolve eagerly and `Resolve` returns the
  captured value; profile sources defer and `Resolve` runs the provider once; safe
  output reads metadata without resolving.
- **Permission enforcement, fail-closed:** POSIX group/world reject (GOOS-injection
  trick where needed); **Windows DACL accept (owner/SYSTEM/Administrators) and reject
  (`Everyone`/`Users`/`Authenticated Users`)** — the phase-1 Windows requirement,
  testable via constructed security descriptors.
- Precedence matrix (env overrides profile; flag overrides env; which source wins).
- No-fallback + unknown-scheme + missing-secret error paths.
- **`cmd:` structured argv:** a `$()`/pipe inside an `argv` element is treated as a
  literal argument (no shell); **timeout** fires on a hung provider.
- **`keyring:` segment validation** (reject empty/`/`-containing segments).
- `keyring:` behind a mock; real keychain never touched in CI.
- **`ZSCALERCTL_DISALLOW_CMD`** truthy/falsey parsing + value-free error on garbage.
- **Backward-compat:** every existing `LoadEnv` test passes unchanged.
- Drift-gated config JSON schema test.

## Docs

- `docs/INSTALL.md` — profiles + providers section; keep the existing env/`*_FILE`
  guidance as the primary/CI path.
- `THREAT_MODEL.md` — the `cmd:` code-execution section + config-file trust (POSIX +
  Windows DACL) + the no-plaintext / no-fallback guarantees + provider precedence.
- `config show` / `doctor` doc updates.
- `AGENTS.md` / skill — note that **agents continue to use env**; profiles are
  operator ergonomics (don't push agents onto config files).
- `docs/schema/` — the config-file JSON schema.

## Backward compatibility

No config file ⇒ identical to today. Env always overrides the profile. Existing env
and `*_FILE` semantics are unchanged (now expressed through `SecretSource`, with the
same resolved-eagerly behavior). The feature can ship dark — nobody who doesn't
create a config file notices anything.

## Rollout

Single coherent pre-1.0 feature. Likely phasing for the implementation plan:
**(1)** `SecretSource` + config loader + POSIX & Windows DACL validation +
`env`/`file` providers + precedence + backward-compat **+ a minimal `SafeConfig`
metadata view** (active profile, config source, per-secret `Scheme()`/`IsConfigured()`
— no resolution). This view lands in phase 1 on purpose: it is how phase 1 *proves*
the no-resolution property ("safe output reads metadata without resolving"), so it is
a test surface, not just docs polish. **(2)** `cmd:` (structured, timeout,
kill-switch) + threat-model docs; **(3)** `keyring:` cgo-free backend (read);
**(4)** fuller `config show`/`doctor` surfacing + config JSON schema + remaining docs. Fast-follows
(separately, demand-validated): keyring **write** helper, native cgo keychain,
SOPS conveniences.
