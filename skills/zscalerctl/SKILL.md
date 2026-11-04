---
name: zscalerctl
description: Use when asked about Zscaler tenant configuration, inventory, audit or digital-experience telemetry — ZIA/ZPA/ZTW/ZCC/Zidentity locations, rules, policies, app segments, connectors, users and groups, or ZDX app scores and alerts — or to export or diff Zscaler config, when `zscalerctl` is available or should be checked.
---

# zscalerctl

`zscalerctl` reads Zscaler tenant configuration and never changes it. Output is
an allow-listed projection of each resource: a field missing from output is
deliberately unavailable, and a read names any field its redaction mode omits
on stderr. The full agent guide is `AGENTS.md` (two directories up from this
skill in a checkout or release archive, or
<https://github.com/dvmrry/zscalerctl/blob/main/AGENTS.md>).

If `zscalerctl` is not installed, ask the operator to install it. Run commands
with `set -euo pipefail` so a failure stops what follows.

## Before a read

1. Find the resource instead of guessing its name:

   ```sh
   zscalerctl --format json machine manifest |
     jq -c '.capabilities[] | select(.name == "resources.read" and .input.product == "zia")
            | {resource: .input.resource, operations, get_key: (.meta.get_key // null)}'
   ```

   `schema list` has each resource's fields and the modes that render them.
   It is large, and it keys a resource by `.name` where the manifest uses
   `.input.resource`. The fields `share` renders for `zia/locations`:

   ```sh
   zscalerctl --format json schema list |
     jq -c '.[] | select(.product == "zia" and .name == "locations")
            | [.fields[] | select(.allowed_modes | index("share")) | .name]'
   ```

   ZDX reports are not in the manifest; see `zscalerctl zdx --help`.
2. Check that zscalerctl reads what you are looking for:
   `zscalerctl --format json schema gaps <product>`. A declared gap means
   "not read", not "none in the tenant".
3. Check readiness without contacting Zscaler: `zscalerctl doctor`. It exits
   `0` either way; its `status` is `OK` only when live reads can run. If
   credentials are missing, ask the operator to set the `ZSCALERCTL_*`
   variables; never invent values or search shell configuration.

## Reading

```sh
zscalerctl --format json --timeout 30s zia locations list --fields id,name
zscalerctl --format json --timeout 30s zia locations get 12345
zscalerctl --format json --timeout 30s zia advanced-settings show
zscalerctl --format json zia url-filtering-rules list --filter 'name~social'
zscalerctl --format json zia users list --count-by groups.id,department.id
zscalerctl --format json zia locations list --fields id,name --limit 20 --offset 0
zscalerctl --format json zdx apps report --since 2h
```

- Always pass `--format json`; never parse `pretty` or `table` output.
- Use `get` when the task gives an id. `list`, `get` and `show` are separate
  commands, not a pipeline.
- `--fields`, `--filter`, `--search`, `--count-by` and `--limit`/`--offset`
  narrow inside the CLI; they cannot widen output. A dotted path such as
  `groups.id` reaches nested fields.
- Prefer counts and bounded pages to printing large lists into context.
- `[]` means the API returned none. A missing key means not sent, not modeled
  or omitted by the mode; it never means `false` or none.

## Redaction

The default mode is **`standard`**: names, emails, free text and other
identifying values render. Pass `--redaction share` for output that will be
shared or pasted: it drops free text and sensitive identifiers, and rule
references render as ids. `--redaction paranoid` also drops most tenant
configuration. Secret fields never render in any mode. Asking to filter on a
field the mode omits is a usage error naming the mode that renders it.

To see what changed over time, dump now and diff against a later dump:

```sh
zscalerctl dump --products zia --output ./scratch/zia-2026-11-03
zscalerctl --format json diff ./scratch/zia-2026-11-03 ./scratch/zia-2026-11-10
```

Both dumps must use the same redaction mode, and a diff sees only the fields
that mode renders, so snapshot in `standard` when every change matters. Dumps
and diff reports are sanitized but confidential tenant inventory: keep them
in ignored scratch paths, out of tickets and chats. `--from-dump DIR` repeats
reads against a saved dump with no credentials or network.

## Results and errors

Exit codes: `0` ok, `1` internal or canceled, `2` usage or unusable setting,
`3` missing credentials, `4` unknown resource, declared gap or nonexistent
id, `5` live API failure, `6` partial dump, `7` drift (`diff --fail-on-drift`).

Failures print a JSON envelope on stderr. A live failure carries a `reason`:

- `rate_limited`, `server_error`, `transport`, `timeout`: back off and retry
  when the envelope has `"retryable": true`, and raise `--timeout` for
  `timeout`.
- `forbidden`, `unauthorized`, `not_subscribed`: the API role or licence does
  not allow the resource. Report it; do not retry.
- `credentials_rejected`: the operator must fix the credentials.
- `pagination_incomplete`: the CLI refused a list it could not prove complete.
  Run it once more; if the same message returns, report it.
- Any other reason: report the message and status; do not retry.

For references between resources (`schema references`, `refs to`,
`refs check`), dumps, diffs, profiles (`--profile`, `--config`), file output
(`--output`), progress logging (`--log-level`) and proxies, read `AGENTS.md`.
