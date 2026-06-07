# zscalerctl

`zscalerctl` is an unofficial, security-first Go CLI for authorized Zscaler
administrators. It is read-only by design and focuses on safe configuration
query, inventory, and controlled sanitized exports.

This project is not affiliated with, endorsed by, or sponsored by Zscaler.

The canonical binary is `zscalerctl`. If you prefer a short command locally,
use a shell alias:

```sh
alias zctl=zscalerctl
```

## What It Is For

The primary use case is CLI and agentic automation: one reviewed command that
can replace duplicated Python snippets across pipelines and workflows. Human
tables should be readable, but machine output should stay explicit,
deterministic, and script-friendly.

`zscalerctl` currently exposes reviewed read/list/show coverage across ZIA, ZPA,
and ZTW. The current catalog is the source of truth:

```sh
zscalerctl --format json schema list
```

For the human-readable resource reference, see
[docs/RESOURCES.md](docs/RESOURCES.md). For queued, deferred, and future
resource work, see [docs/RESOURCE_QUEUE.md](docs/RESOURCE_QUEUE.md).

ZCC was scouted and smoke-tested, but the first conservative ZCC PAPI v2 batch
returned 404 for every staged list endpoint under production OneAPI. ZCC remains
deferred until endpoint, auth, or entitlement behavior is understood.

## Install

Release archives are published for macOS, Linux, and Windows. Releases include
checksums, per-target CycloneDX SBOMs, and GitHub provenance attestations. See
[docs/INSTALL.md](docs/INSTALL.md) for installation, artifact verification,
credential, proxy, shell-completion, and platform notes.

From a checkout:

```sh
go install ./cmd/zscalerctl
zscalerctl version
```

## Quick Start

Check local configuration without contacting Zscaler:

```sh
zscalerctl doctor
zscalerctl auth status
zscalerctl config show
```

Inspect the reviewed resource catalog:

```sh
zscalerctl schema list
zscalerctl --format json schema list
```

Read resources:

```sh
zscalerctl zia locations list
zscalerctl zia locations get <id>
zscalerctl zpa server-groups list
zscalerctl ztw workload-groups list
```

Write a sanitized dump:

```sh
zscalerctl dump --products zia --out ./dump
zscalerctl dump --resources zia/locations,zpa/server-groups --out ./dump-subset
zscalerctl dump --continue-on-error --out ./partial-dump
```

Use `--output <path>` to write command output to a single restricted file. Use
`dump --out <dir>` for dump directories; `--output` and `dump` are intentionally
not combined.

## Authentication

The CLI reads only explicit `ZSCALERCTL_*` configuration. It does not read the
Zscaler SDK's own environment variables or SDK config files. SDK response
caching is disabled, SDK logging is muted, and ambient proxy variables are
ignored unless opted in.

OneAPI is the default auth mode:

```sh
export ZSCALERCTL_CLIENT_ID=<client-id>
export ZSCALERCTL_CLIENT_SECRET_FILE=/path/to/owner-only/secret-file
export ZSCALERCTL_VANITY_DOMAIN=<vanity-domain>
export ZSCALERCTL_CLOUD=PRODUCTION
export ZSCALERCTL_ZPA_CUSTOMER_ID=<zpa-customer-id> # required for ZPA resources
```

ZIA legacy credentials are still supported for read-only ZIA resources. Detailed
legacy, proxy, config-file, Windows, and secret-file behavior lives in
[docs/INSTALL.md](docs/INSTALL.md).

Corporate proxy settings are opt-in. To use standard `HTTPS_PROXY`,
`HTTP_PROXY`, and `NO_PROXY` values, set:

```sh
export ZSCALERCTL_PROXY_FROM_ENV=true
```

## Security Posture

This is defensive administration software. It is not an exploitation,
credential discovery, bypass, traffic interception, or attack-path tool.

The primary leak-prevention model is allow-list projection into safe view
records. Output redaction and secret scanning are defense-in-depth, not an
excuse to render raw API responses.

Administrator-controlled free-text fields are standard-only catalog exceptions:
each one must be justified in the schema, scanner-backed, and excluded from
`share` and `paranoid` output.

Version 1 must not include write commands or a generic raw API executor.

Table output is best-effort for quick human inspection. JSON and dump output are
the primary automation surfaces.

## Dumps

Dump commands fail closed by default: if a selected resource fails, no dump is
written. `--continue-on-error` is opt-in and writes a clearly marked partial
dump with `manifest.json` status `partial` and value-free `errors.ndjson`. A
partial dump exits with code `6`, not success.

`make live-smoke` validates the current branch's live-smoke resource manifest
or explicit `LIVE_SMOKE_RESOURCES` selection and writes artifacts to a secure
temporary directory. Set `LIVE_SMOKE_OUT=./scratch-live-smoke` only when you
want a predictable artifact path.

## Automation Contract

Exit codes are stable for automation:

| Code | Meaning |
| --- | --- |
| `0` | Complete success. |
| `1` | Internal or unclassified failure. |
| `2` | Usage or argument error. |
| `3` | Missing or invalid credentials. |
| `4` | Product/resource not found. |
| `5` | Live Zscaler API access failure. |
| `6` | Partial dump written; inspect `manifest.json` and `errors.ndjson`. |

When `--format json` is requested and a command fails, diagnostics are emitted
as a redacted JSON envelope on stderr:

```json
{
  "error": {
    "kind": "missing_credentials",
    "message": "missing zscaler API credentials: ZSCALERCTL_CLIENT_ID is required"
  }
}
```

## Documentation

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- [docs/THREAT_MODEL.md](docs/THREAT_MODEL.md)
- [docs/DATA_CLASSIFICATION.md](docs/DATA_CLASSIFICATION.md)
- [docs/ZSCALER_SENSITIVE_DATA.md](docs/ZSCALER_SENSITIVE_DATA.md)
- [docs/INSTALL.md](docs/INSTALL.md)
- [docs/RESOURCES.md](docs/RESOURCES.md)
- [docs/RESOURCE_QUEUE.md](docs/RESOURCE_QUEUE.md)
- [docs/SDK_SURFACE_INVENTORY.md](docs/SDK_SURFACE_INVENTORY.md)
- [docs/ZSCALER_PRODUCT_SCOPE_PLAN.md](docs/ZSCALER_PRODUCT_SCOPE_PLAN.md)
- [docs/VERSIONING.md](docs/VERSIONING.md)
- [docs/DEPENDENCY_POLICY.md](docs/DEPENDENCY_POLICY.md)
- [docs/RELEASE_CHECKLIST.md](docs/RELEASE_CHECKLIST.md)

## Development

```sh
make check
make live-smoke
make sdk-surface-inventory
```

Optional local checks once installed:

```sh
gitleaks dir .
gitleaks git .
```
