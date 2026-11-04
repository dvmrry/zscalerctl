# Agent Guide

`zscalerctl` reads Zscaler tenant configuration and never changes it. This
guide is for agents and scripts driving the CLI. The command reference is
[docs/cli/zscalerctl.md](docs/cli/zscalerctl.md); the contract (exit codes,
error vocabulary, schemas) is [docs/VERSIONING.md](docs/VERSIONING.md) and
[docs/cli/machine-contract.md](docs/cli/machine-contract.md).

## Local effects

The tenant is never changed, but some commands have local effects: commands
that load configuration read config and secret files, a configured `cmd:`
secret provider runs a process, `config init` writes a config file, and
`--output` creates or replaces a file, except that `dump --output` writes a
directory (`--force` can replace an earlier dump).
`zscalerctl --format json introspect` lists each command's `effects`; check
them before delegating a command. Prefer stdout, and use `--output` only when
a file is the intended result.

## Discover, don't guess

Resource names, operations and fields come from the binary, not from memory:

```sh
zscalerctl version
zscalerctl --format json machine manifest   # resource reads: operations, shape, get key; config-free
zscalerctl --format json schema list        # catalog: fields, types, classes, modes, references
zscalerctl --format json schema gaps zpa    # surfaces zscalerctl does not read, and why
zscalerctl --format json introspect         # every command and flag, with effects
```

None of these load configuration or contact Zscaler. The manifest and catalog
are large; select one resource instead of printing them. This gives the
operations and get key of `zia/locations`, then the fields `share` renders:

```sh
zscalerctl --format json machine manifest | jq -c '.capabilities[]
  | select(.name == "resources.read" and .input.product == "zia" and .input.resource == "locations")
  | {operations, shape: .meta.shape, get_key: .meta.get_key}'
zscalerctl --format json schema list | jq -c '.[]
  | select(.product == "zia" and .name == "locations")
  | [.fields[] | select(.allowed_modes | index("share")) | .name]'
```

No output means the product or resource name is wrong. Secret fields have no
`allowed_modes`, so a mode filter never selects them.

Each `schema list` field may carry a `type` (`string`, `integer`, `number`,
`boolean`, `object`, or `array<T>`), the JSON type zscalerctl emits. It is
absent where the type is not fixed. Any typed field may also be `null`, so
guard list iteration (`.x // []`). A redacted value renders as a marker string
whatever its type.

ZDX reports are not in the manifest; `zscalerctl zdx --help` lists them.

## Credentials

Agents get credentials from `ZSCALERCTL_*` environment variables, which always
override profiles: `ZSCALERCTL_CLIENT_ID`, `ZSCALERCTL_CLIENT_SECRET` (or
`ZSCALERCTL_CLIENT_SECRET_FILE`), `ZSCALERCTL_VANITY_DOMAIN`,
`ZSCALERCTL_CLOUD`, and `ZSCALERCTL_ZPA_CUSTOMER_ID` for ZPA. The Zscaler SDK's
own variables are never read. Owner-only YAML profiles, selected with
`--profile` and `--config`, are for interactive use; their secret references
(`env:`, `file:`, `keyring:`, `cmd:`) are the operator's to set, and
`ZSCALERCTL_DISALLOW_CMD=true` refuses `cmd:` providers.

`zscalerctl doctor` reports which settings are present or missing without
contacting Zscaler. It exits `0` either way, so read `status`: it is `OK`
only when live reads can run, and
`credentials` (`configured`, `partial`, `invalid settings`, `not configured`)
and `live_api` say why. If something is missing, ask the operator to set it;
do not invent values or search shell configuration.

When a config file has several profiles and none is selected, a successful
live read names the profile it used on stderr. Pass `--profile` when more than
one tenant is configured.

## Reading

Read with separate commands: `list`, `get <id>` for one record, `show` for a
singleton:

```sh
zscalerctl --format json --timeout 30s zia locations list
zscalerctl --format json --timeout 30s zia locations get 12345
zscalerctl --format json --timeout 30s zia advanced-settings show
```

If the task gives an id, use `get` directly instead of listing. `--timeout`
bounds each HTTP request, not a whole list or dump; bound the total run
externally when it matters.

Always pass `--format json` (or `--format ndjson` for one compact record per
line). `--format auto` prints JSON when output is piped, but a PTY harness can
look like a terminal and get the human `pretty` view. Never parse `pretty` or
`table` output.

## Redaction modes

`--redaction` (or `ZSCALERCTL_REDACTION`) decides which fields render.
`standard` is the default and renders every reviewed field, including names,
emails, free text and other identifying values. `share` drops free text and
sensitive identifiers, so rule references (users, groups, departments,
locations) render as ids only. `paranoid` also drops most tenant
configuration, keeping ids, booleans, enums and states; a parent list it does
not render takes its children with it. Secret fields (credentials, keys,
tokens, opaque blobs, unmodeled maps such as `extensions`) never render in any
mode, and `schema list` gives each one's `secret_reason`.

When the mode omits fields, a successful read names them on stderr
(`zscalerctl: share mode omits zia/users fields: name, email, groups.name, ...`).
Omitted fields cannot be recovered except by reading in a mode that renders
them. Use `share` when output will be pasted or shared.

## Narrowing and counting

These run inside the CLI after projection and redaction; they can only narrow.

- `--fields a,b,c` keeps only those fields (`list`, `get`, `show`). A dotted
  path keeps the nested shape: `--fields name,groups.id` gives
  `"groups":[{"id":1},{"id":2}]`.
- `--filter key=value` matches exactly, `--filter key~value` matches a
  case-insensitive substring, and repeated filters must all match. A key may
  be a dotted path to a nested leaf (`groups.name=APP_DEV` matches one group
  exactly); a top-level list matches when any element does.
- `--search term` keeps records where any rendered value contains the term.
- `--count-by a,b` returns counts instead of records:
  `{"count_by":[...],"records":N,"distinct":D,"truncated":false,"counts":[{"values":{"groups.id":10},"count":2},...]}`,
  largest first. A path into a list counts per element (`--count-by groups.id`
  on `zia users` gives members per group), records with no value count under
  `null`, and paths through the same list pair values from the same element.
  `--limit N` keeps the top N counts and sets `truncated`.
- `--limit N --offset K` returns a bounded JSON page of a list:
  `{"records":[...],"pagination":{"offset","limit","matched_count","returned_count","has_more","next_offset","collection_complete"}}`.
  The whole list is still collected first, so paging bounds output, not API
  calls or memory, and separate calls are not a snapshot.

Filters, search, `--count-by` and paging apply to `list` only. Unknown field
names, and filters on a field the mode omits or a secret field, are usage
errors (exit `2`) before any read. No matches is success with `[]`.

```sh
zscalerctl --format json --fields id,name --filter 'name~hq' --limit 20 zia locations list
zscalerctl --format json --count-by groups.id,department.id zia users list
```

What absence means: a list field renders `[]` only when the API returned an
empty list. A missing key means the API did not send it, the mode omits it, or
it is not modeled; it never proves none are set and never means `false`.

## ZDX reports

ZDX is read as time-windowed reports, not configuration:

```sh
zscalerctl --format json zdx apps report --since 2h
zscalerctl --format json zdx alerts report --from 2026-10-01T00:00:00Z --to 2026-10-02T00:00:00Z
zscalerctl --format json zdx apps report 42 --department 7
```

Reports exist for `apps`, `alerts`, `alert-history`, `departments` and
`locations`. The window is `--since` (default `2h`) or `--from`/`--to`
(RFC 3339) and is echoed in the output. Some tenants reject long windows on
some routes. Reports are not part of `dump`, `diff` or `refs`.

## Dumps, diffs and saved reads

```sh
zscalerctl dump --products zia --output ./scratch-dump --log-level info
zscalerctl --format json diff ./scratch-dump-old ./scratch-dump-new
zscalerctl --format json --from-dump ./scratch-dump --fields id,name zia locations list
```

A dump can take minutes; `--log-level info` reports progress on stderr
(metadata only). Exit `6` means a partial dump: read `manifest.json` and
`errors.ndjson`. `diff` compares two dump directories already on disk; lists
compare in order, so a reordered list is drift, and `--fail-on-drift` exits
`7` on any change. Both dumps must use the same redaction mode, and a diff
sees only the fields that mode renders, so take snapshots in `standard` when
every change matters.

`--from-dump` runs the usual resource commands and `refs` against a complete
saved dump, with no configuration, credentials or network. It keeps the dump's
redaction mode and rejects a different `--redaction`, and cannot be combined
with `--profile` or `--config`. Partial or invalid dumps are rejected, and a
missing resource or id is an error, not an empty result. Each call reloads the
directory (up to 256 MiB of resource data).

Dumps and diff reports are sanitized but confidential tenant inventory: keep
them in ignored scratch paths and out of tickets and chats.

## References

`schema list` marks reference fields with their targets. To find every field
that points at a resource, and the `--filter` path to use:

```sh
zscalerctl --format json schema references zia locations
zscalerctl --format json --filter locations.id=12345 zia url-filtering-rules list
```

`refs` does those reads, live or with `--from-dump`:

```sh
zscalerctl --format json refs to zia groups 12345            # records whose fields hold that id
zscalerctl --format json refs check zia url-filtering-rules  # referenced ids not in the lists read
```

Both list under `not_checked` the fields the mode omits, the resources that
could not be read, and fields whose target could not be read. `refs check`
also lists audit fields such as `lastModifiedBy` there, because their ids can
name accounts outside `admin-users`. A reference whose target is not verified
is not listed, so an empty result does not prove nothing points at a resource.

`schema describe zia url-filtering-rules` gives reviewed field meanings for
`zia/locations`, `zia/url-filtering-rules` and `zia/rule-labels`; other
resources report `not_reviewed`.

## Gaps

Before concluding a tenant has none of something, check that zscalerctl reads
it: `zscalerctl --format json schema gaps <product>`. Each gap has a
`surface`, a `status` (`deferred` or `out_of_scope`), a `reason`, a note and,
where one exists, the resources to read `instead`. Asking for a declared gap,
or an alias of one such as `zpa scim-users`, exits `4` with a message naming
the gap. Any other unknown name also exits `4` and names up to three close
catalog resources (`did you mean zpa/access-policy-rules?`); with none named,
find the resource in the manifest.

## Errors

Failures print a JSON envelope on stderr:
`{"error":{"kind":"...","message":"...","product":"...","resource":"..."}}`.
A live failure (exit `5`) adds a `reason`, the HTTP `status` when there was a
response, and `"retryable": true` when backing off and retrying can succeed;
it is absent otherwise.

- `rate_limited`, `server_error`, `timeout`, `transport`: back off and retry
  when `retryable` is present (some `transport` failures are permanent and
  have none), and raise `--timeout` for `timeout`.
- `forbidden`, `unauthorized`, `not_subscribed`: the API role or licence does
  not allow the resource. Report it; do not retry.
- `credentials_rejected`: sign-in was refused. The operator must fix the
  credentials.
- `pagination_incomplete`: the CLI refused a list it could not prove
  complete; the message names the check, in counts and offsets. A tenant
  change during the read can cause it, so run the command once more; the
  same message again means the API's paging is the cause, so report it.
- `decode_failed` (the response did not have the expected shape),
  `client_error` (an HTTP 4xx not listed above), `not_found` (HTTP 404) and
  `unknown`: report the message and status; do not retry.

Exit codes: `0` ok, `1` internal or canceled, `2` usage or unusable setting,
`3` missing credentials, `4` unknown resource, declared gap or nonexistent id,
`5` live API failure, `6` partial dump, `7` drift.

Live reads ignore `HTTPS_PROXY` unless `ZSCALERCTL_PROXY_FROM_ENV=true` or
`ZSCALERCTL_PROXY_URL` is set. When `HTTPS_PROXY` is set but unused, a
`timeout` or `transport` message says so; on a network that needs the proxy,
ask the operator to opt in rather than retrying.
