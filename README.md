# zscalerctl

[![CI](https://github.com/dvmrry/zscalerctl/actions/workflows/ci.yml/badge.svg)](https://github.com/dvmrry/zscalerctl/actions/workflows/ci.yml)
[![CodeQL](https://github.com/dvmrry/zscalerctl/actions/workflows/codeql.yml/badge.svg)](https://github.com/dvmrry/zscalerctl/actions/workflows/codeql.yml)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/dvmrry/zscalerctl/badge)](https://scorecard.dev/viewer/?uri=github.com/dvmrry/zscalerctl)
[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/13176/badge)](https://www.bestpractices.dev/projects/13176)
[![Release](https://img.shields.io/github/v/release/dvmrry/zscalerctl)](https://github.com/dvmrry/zscalerctl/releases)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue)](LICENSE)

> Unofficial, **read-only** CLI for authorized Zscaler administrators and
> their automation: configuration queries, inventory, sanitized dumps and
> drift reports.

Not affiliated with, endorsed by, or sponsored by Zscaler.

## What it does

- **Reads, never writes.** There are no commands that change a tenant and no
  raw API executor.
- **Renders only reviewed fields.** Every resource is read through a catalog
  that lists the fields allowed to render and the redaction modes that render
  them. Anything else, including every credential, is dropped. Redaction and
  secret scanning run on top as defense in depth.
- **Machine-first output.** Piped output is deterministic JSON, failures emit a
  JSON error envelope on stderr, and exit codes are a stable contract.
- **Explicit configuration.** Only `ZSCALERCTL_*` variables and optional
  owner-only profiles are read; never the Zscaler SDK's own variables or files.

Coverage: configuration for **ZIA, ZPA, ZTW, ZCC and Zidentity**, and
time-windowed **ZDX** reports. The catalog is the source of truth:

```sh
zscalerctl --format json schema list     # every resource, operation and field
zscalerctl --format json schema gaps     # what is deliberately not read, and why
```

The [resource reference](https://github.com/dvmrry/zscalerctl/blob/main/docs/RESOURCES.md)
lists every field and the modes that render it.

## Install

Download a release archive for macOS, Linux (amd64, arm64) or Windows (amd64)
and verify it, or build from a clone with Go 1.26.10 or newer:

```sh
git clone https://github.com/dvmrry/zscalerctl.git
cd zscalerctl
go install ./cmd/zscalerctl
```

[docs/INSTALL.md](docs/INSTALL.md) covers verification, credentials, profiles,
proxies, completions and platform notes.

## Quick start

```sh
zscalerctl doctor                     # is configuration ready? (no network)
zscalerctl zia locations list
zscalerctl zpa server-groups list --fields id,name
zscalerctl zia users list --count-by groups.id        # members per group
zscalerctl zdx apps report --since 2h
zscalerctl dump --products zia --output ./scratch-dump
zscalerctl diff ./scratch-dump-old ./scratch-dump-new --fail-on-drift
```

A terminal gets a readable `pretty` view; a pipe or redirect gets JSON. Scripts
and agents should pass `--format json` explicitly. Dumps and diff reports are
sanitized but still confidential tenant inventory: keep them out of tickets
and chats.

## Authentication

OneAPI is the default:

```sh
export ZSCALERCTL_CLIENT_ID=<client-id>
export ZSCALERCTL_CLIENT_SECRET_FILE=/path/to/owner-only/secret-file
export ZSCALERCTL_VANITY_DOMAIN=<vanity-domain>
export ZSCALERCTL_CLOUD=PRODUCTION          # the OneAPI gateway, not the product cloud
export ZSCALERCTL_ZPA_CUSTOMER_ID=<id>      # ZPA resources only
```

ZIA legacy credentials, profiles, secret providers (`env:`, `file:`,
`keyring:`, `cmd:`) and proxies are described in [docs/INSTALL.md](docs/INSTALL.md).

## Redaction modes

`--redaction` (or `ZSCALERCTL_REDACTION`) chooses which fields render:

| Mode | Renders |
| --- | --- |
| `standard` (default) | Every reviewed field, including names, emails, free text and other identifying values. |
| `share` | Drops free text and sensitive identifiers; rule references such as users, groups and locations render as ids. For output that will be shared. |
| `paranoid` | Also drops most tenant configuration, keeping ids, booleans, enums and states. |

Secret fields never render in any mode, and no mode can widen what the catalog
allows. When a mode omits fields, the read names them on stderr.

## Automation

Exit codes: `0` success, `1` internal or canceled, `2` usage or unusable
setting, `3` missing credentials, `4` unknown resource, declared gap or
nonexistent id, `5` live API failure (including rejected credentials and
entitlement errors, with a `reason`), `6` partial dump, `7` drift. See
[docs/VERSIONING.md](docs/VERSIONING.md) for the full contract and
[docs/cli/machine-contract.md](docs/cli/machine-contract.md) for the error
vocabulary.

Agents start with [AGENTS.md](AGENTS.md) or the installable skill in
[skills/zscalerctl/](skills/zscalerctl/SKILL.md). The full command reference is
[docs/cli/zscalerctl.md](docs/cli/zscalerctl.md), and `zscalerctl introspect`
prints the same map as JSON, including each command's local effects.

## Security

The read-only guarantee covers the tenant. Locally, `config init` and
`--output` write files (a directory for `dump`), commands that load configuration read
config and secret files, and a configured `cmd:` secret provider runs a
process; `introspect` lists these effects per command.

Report vulnerabilities privately as described in
[SECURITY.md](https://github.com/dvmrry/zscalerctl/blob/main/SECURITY.md).

## Development

```sh
make check        # tests, vet, vulnerability scan, staticcheck, semgrep, secret scan, doc and registry checks
make heavy-check  # race and fuzz here, make test on a Linux host; POST=1 reports it on the PR
```

In a private repo, pull requests skip CI's unit and race jobs to save Actions
minutes, so run `ZSCALERCTL_LINUX_HOST=<ssh host> make heavy-check POST=1` on
the pushed branch before merging. The Linux host needs Nix.

A pull request either adds `docs/releases/vX.Y.Z.md` (the next version's
release notes; merging publishes that release) or carries the `no-release`
label. New behaviour needs tests.

License: [Apache-2.0](LICENSE).
