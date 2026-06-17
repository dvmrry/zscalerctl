# Config Profiles + Pluggable Secret Providers — Design

**Status:** Approved 2026-06-17 (brainstorming). Pre-1.0. Implementation plan to follow (writing-plans).

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

### Secret-specific precedence

```
inline env (e.g. ZSCALERCTL_CLIENT_SECRET)
  > env file (e.g. ZSCALERCTL_CLIENT_SECRET_FILE)
  > profile secret_ref (resolved via provider)
  > unset
```

## Config file

- **Format:** YAML.
- **Location:** `$XDG_CONFIG_HOME/zscalerctl/config.yaml` (default
  `~/.config/zscalerctl/config.yaml`); overridable with `--config PATH` or
  `ZSCALERCTL_CONFIG`.
- **Owner-only, fail-closed:** validated exactly like `*_FILE` secret files (reject
  group/world-readable; same Windows fail-closed posture until ACL support lands).
  Enforced **even though it holds no secrets**, because it holds `cmd:` refs (see
  threat model).
- **Schema** (illustrative; full field set mirrors today's env vars per auth mode):

```yaml
default_profile: prod            # optional; else --profile / ZSCALERCTL_PROFILE required
profiles:
  prod:
    auth_mode: oneapi            # optional, default oneapi
    vanity_domain: example
    cloud: PRODUCTION
    client_id: "..."             # identifier, not a secret
    client_secret_ref: "keyring:zscalerctl/prod/client_secret"
    zpa_customer_id: "..."
    zpa_microtenant_id: "..."    # optional
    # operational defaults (only fields with existing env equivalents) — Decision B
    redaction: standard
    no_cache: false
  preprod:
    auth_mode: zia-legacy
    zia_username: "..."
    zia_password_ref: "file:/path/to/owner-only/pw"
    zia_api_key_ref: "cmd:/usr/local/bin/get-zia-key --profile preprod"
    zia_cloud: "..."
```

A **drift-gated JSON schema** for the config file ships under `docs/schema/`
(matching the existing dump/diff/error schema discipline).

## Secret providers

A secret reference is `scheme:value`, resolved to the existing protected
`secret.Secret` type by a small dispatcher in `internal/secretref`.

| Scheme | Resolution | Notes |
|--------|------------|-------|
| `env:NAME` | read env var `NAME` | unset → error |
| `file:/path` | existing `credentials.ReadOwnerOnlySecretFile` | strict perms; already implemented, just exposed as a scheme |
| `cmd:prog [args]` | exec **directly, no shell**; trimmed stdout is the secret | no shell ⇒ no injection; pipes/SOPS go in a script the ref points at |
| `keyring:service/key` | cgo-free OS keychain | Linux D-Bus, Windows wincred syscall, macOS `security` CLI |

- Unknown scheme → clear error. A bare value with no scheme → error
  ("secret refs require a provider scheme").
- **No automatic fallback:** an unavailable `keyring:`/`cmd:` fails loud
  (`keyring backend unavailable on this host; use env:, file:, or cmd:`).
- `keyring:` sits behind a **mockable interface** so CI never touches a real
  keychain; the keychain is a desktop-interactive convenience on every OS (may
  prompt; locked/headless sessions fail) — which is why `env`/`file`/`cmd` remain
  the headless path.

## `cmd:` threat model (the one real surface expansion)

`cmd:` turns the config file into a code-execution vector. Mitigation chain,
written up as a dedicated `THREAT_MODEL.md` section:

1. **Config is owner-only or we fail closed.** Same gate as secret files.
2. **`cmd:` execs directly, never through a shell.** No shell-injection surface.
3. **Reasoning:** if the config is owner-only, only *you* can add a `cmd:` ref, and
   you could already run anything as yourself — so it grants no new privilege. The
   only escalation ("attacker can write your owner-only config") already implies the
   attacker owns your account (shell rc, PATH, …). This is exactly the AWS
   `credential_process` model.
4. **`ZSCALERCTL_DISALLOW_CMD` kill-switch** (opt-*out*) lets a hardened fleet forbid
   `cmd:` entirely. No separate opt-in is required — the owner-only gate is the gate.
   *(Decision C.)*

## Lazy resolution & safe output

- Secrets resolve **only when a live reader is constructed.** `doctor`,
  `config show`, `schema list`, `version`, `completion` never resolve a secret — no
  keychain prompts, no `cmd:` exec, nothing surfaced.
- `config show` / `doctor` report: **active profile, config source (env-only vs
  config-file path), and each secret's *provider scheme* + set/unset** — never
  values, never the full ref string.

## Module boundaries

- `internal/config` — add config-file + profile loading (`file.go`, `profile.go`);
  still emits `config.Config`. Existing `LoadEnv` stays and is layered under env.
- `internal/secretref` — provider dispatch (`Resolve(ref) (secret.Secret, error)`),
  one small unit per scheme.
- `internal/keyring` — cgo-free backend behind a mockable interface.
- CLI wiring — `--profile` / `--config` flags, precedence resolution, lazy secret
  resolution at reader-build time.

## Error handling

Invalid config, bad permissions, unknown scheme, unavailable provider, or a missing
required secret map to `ErrInvalidConfig` (exit 2 — usage) or the existing
credential/usage codes, each with a clear, actionable message. **Never a silent
fallback.**

## Testing

- Table-driven provider tests (`env`/`file`/`cmd`/`keyring`).
- Owner-only fail-closed enforcement on the config file (GOOS-injection trick for
  the Windows path, mirroring the existing `credentials/files` test).
- Precedence matrix (env overrides profile; flag overrides env).
- No-fallback + unknown-scheme + missing-secret error paths.
- `cmd:` runs no shell (verify a pipe/`$()` in the ref is treated as literal argv).
- `keyring:` behind a mock; real keychain never touched in CI.
- **Backward-compat:** every existing `LoadEnv` test passes unchanged.
- Drift-gated config JSON schema test.

## Docs

- `docs/INSTALL.md` — profiles + providers section; keep the existing env/`*_FILE`
  guidance as the primary/CI path.
- `THREAT_MODEL.md` — the `cmd:` code-execution section + config-file trust + the
  no-plaintext / no-fallback guarantees + provider precedence.
- `config show` / `doctor` doc updates.
- `AGENTS.md` / skill — note that **agents continue to use env**; profiles are
  operator ergonomics (don't push agents onto config files).
- `docs/schema/` — the config-file JSON schema.

## Backward compatibility

No config file ⇒ identical to today. Env always overrides the profile. Existing
env and `*_FILE` semantics are unchanged. The feature can ship dark (nobody who
doesn't create a config file notices anything).

## Rollout

Single coherent pre-1.0 feature. Fast-follows (separately, demand-validated):
keyring **write** helper, native cgo keychain (only if the `security`-CLI hop
proves insufficient), SOPS conveniences.
