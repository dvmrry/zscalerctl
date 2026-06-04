# ZDX Scope Plan

This is a proposal for adding ZDX support without weakening zscalerctl's
current configuration-inventory model. It is not an enabled catalog, live-smoke
result, or entitlement proof.

## Verdict

Treat ZDX as a report/telemetry product, not as ordinary configuration
inventory.

The first implementation should be a narrow `zdx/applications` report surface
with an explicit time window and a dedicated `report` operation. It should not
join default `dump`, and it should not reuse the generic resource `list|get`
output path.

## Why ZDX Is Different

The current resource model assumes deterministic read-only configuration:

- catalog specs describe relatively static tenant objects;
- `list` and `get` return projected records;
- `dump` can collect resources without extra query semantics;
- validation compares output shape to an allow-list.

ZDX report APIs are different:

- SDK comments and filters define a default "last 2 hours" report window.
- Many endpoints accept `from`/`to`, location, department, geolocation, offset,
  and limit filters.
- Returned values are telemetry aggregates, metrics, user/device activity, or
  time-series data.
- A record without its time window is incomplete evidence. The same query run
  later can legitimately return different values.

For that reason, forcing ZDX into the existing config dump model would make the
output look more deterministic than it is.

## SDK Findings

Inspected SDK: `github.com/zscaler/zscaler-sdk-go/v3` `v3.8.37`.

### First Candidate

`zscaler/zdx/services/reports/applications`

- `GetAllApps(ctx, service, filters)` returns `[]Apps`.
- `GetApp(ctx, service, appID, filters)` returns one `Apps`.
- Both accept `common.GetFromToFilters`.
- SDK comments say the endpoint defaults to the last 2 hours when the time
  range is omitted.
- The functions accept the shared `*zscaler.Service`, so they fit the OneAPI
  boundary used by ZIA/ZPA/ZTW/ZCC work. Do not use the product-local
  `zdx.Client` path for the first implementation.

`Apps` fields:

| JSON field | Proposed class | Proposed mode | Notes |
| --- | --- | --- | --- |
| `id` | Operational metadata | `standard`, `share`, `paranoid` | Stable app identifier. |
| `name` | Tenant configuration | `standard`, `share` | Application display/config name. |
| `score` | Operational telemetry | `standard` only | Time-windowed experience score. |
| `total_users` | Operational telemetry | `standard` only | Aggregate user count for the window. |
| `stats.active_users` | Operational telemetry | `standard` only | Aggregate count. |
| `stats.active_devices` | Operational telemetry | `standard` only | Aggregate count. |
| `stats.num_poor` | Operational telemetry | `standard` only | Aggregate health bucket. |
| `stats.num_okay` | Operational telemetry | `standard` only | Aggregate health bucket. |
| `stats.num_good` | Operational telemetry | `standard` only | Aggregate health bucket. |
| `most_impacted_region.*` | Sensitive identifier | `standard` only | Region/city/country/geotype can reveal operational geography. |

No free-text field is present in the application summary type.

### Defer

`GetAppScores` and `GetAppMetrics` return time-series data. They should not ship
with the first ZDX resource unless the report envelope and time-window semantics
are already settled.

`zdx/users` and `zdx/devices` should be deferred. They include user email,
device names, geolocation, hardware, network, software, usernames, hostnames,
serials, MAC/BSSID, and similar endpoint/person data.

`zdx/alerts` should be deferred. Alert details include departments, locations,
geolocations, and affected devices with user IDs, usernames, and user email.

`zdx/software-inventory` should be deferred. It is endpoint-sensitive and has
software-to-user/device drill-down semantics.

`zdx/administration` may be lower risk than users/devices, but it is still a
ZDX-specific helper surface. It can follow after the application report model is
settled.

## Proposed Command Model

Do not add ZDX to default config `dump`.

Preferred first shape:

```sh
zscalerctl zdx applications report --from <unix-seconds> --to <unix-seconds>
zscalerctl zdx applications report --from <unix-seconds> --to <unix-seconds> --id <app-id>
```

Output should be an envelope, not a bare array:

```json
{
  "schema": "zscalerctl.report.v1",
  "product": "zdx",
  "resource": "applications",
  "window": {
    "requested": {
      "from": 1719864000,
      "to": 1719871200
    },
    "effective": {
      "from": 1719864000,
      "to": 1719871200
    }
  },
  "records": []
}
```

Reasons:

- The report window becomes part of the machine-readable contract.
- Future time-series endpoints can use the same envelope.
- Config `dump` remains deterministic inventory.
- The existing projection/redaction machinery can still protect individual
  records before they enter the report envelope.

`window.requested` records the operator's input. `window.effective` records the
window actually queried after any ZDX/API clamping, quantization, or validation.
If the API does not report a normalized range, the first implementation should
set `effective` equal to `requested` and document that limitation in the command
help. Do not silently report only the requested range if the implementation can
determine that the server used a different one.

Empty reports are successful report results:

```json
{
  "schema": "zscalerctl.report.v1",
  "product": "zdx",
  "resource": "applications",
  "window": {
    "requested": {"from": 1719864000, "to": 1719871200},
    "effective": {"from": 1719864000, "to": 1719871200}
  },
  "records": []
}
```

An empty `records` array means no records matched the requested/effective
window. It must not be treated as a query failure unless the API request itself
failed.

Rejected shape:

```sh
zscalerctl zdx applications list --from <unix-seconds> --to <unix-seconds>
zscalerctl zdx applications get <app-id> --from <unix-seconds> --to <unix-seconds>
```

Do not use this shape in v1. `list|get` carry config-resource expectations:
deterministic objects, dump eligibility, and future diff compatibility. ZDX
reports are time-windowed telemetry, so the operation name should make that
boundary structural.

Reports are not diffable like configuration resources. Two report outputs are
comparable only when the report type and effective window are intentionally the
same. Different windows are different evidence, not configuration drift.

Closed historical windows may be cacheable in the future, but report caching is
out of scope for v1. The current global no-cache stance remains unchanged.

## Required Implementation Decisions

Initial implementation decisions:

1. ZDX uses a dedicated `report` operation. Do not overload config `list|get`
   for report-shaped output.
2. `from` and `to` are required in v1. Avoid SDK-default "last 2 hours" output
   because it is implicit and non-reproducible.
3. ZDX report resources may appear in `schema list`, but their operation must be
   shown as `report`, not config `list|get`.
4. Report envelopes are not written by config `dump` in v1. Revisit only after
   report artifact retention and evidence semantics are defined.
5. The report envelope uses schema `zscalerctl.report.v1`. Changes to this
   envelope follow the versioning policy for machine-readable output schemas.

## Safety Requirements

The first implementation should keep the existing safety floor:

- Use the shared OneAPI `*zscaler.Service` path only.
- Do not instantiate `zdx.Client` or product-local ZDX config.
- Keep SDK env/file/cache/logger/proxy suppression unchanged.
- Add `ProductZDX` only where explicit ZDX commands need it.
- Keep ZIA legacy credentials fail-closed for ZDX before service construction.
- Add SDK-shape review entries for every mapped ZDX type.
- Add projection canaries for any deferred nested geography or telemetry fields.
- Add live-smoke manifest support before promotion.
- Treat `200-empty` as availability only, not response-shape proof.

## Proposed First PR

Draft branch: `feature/zdx-report-scope-plan`

Scope:

- This document.
- Optional queue/doc updates only.
- No enabled ZDX catalog resources.
- `semver:none`.

Follow-up implementation branch after review:

- Add minimal ZDX report command support.
- Add `zdx/applications` only.
- Require explicit `--from` and `--to`.
- Output `zscalerctl.report.v1` envelope with requested and effective windows.
- Do not include ZDX in default dump or config dump products.
- Park as draft until a controlled ZDX OneAPI live smoke can run.

## Review Questions

Resolved by review:

1. `zdx/applications` should use a new `report` operation.
2. `id/name` are acceptable in share mode for application summaries; scores,
   counts, and geography remain standard-only.
3. `zdx/administration` waits until one live smoke proves the application report
   envelope.
4. Report outputs remain separate artifacts with explicit window metadata. They
   do not join config `dump` in v1.
