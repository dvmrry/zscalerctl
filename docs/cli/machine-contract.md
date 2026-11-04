# Machine Contract

`zscalerctl` is machine-first: agents, scripts and CI read the same projected
and redacted records a person sees, through stable JSON, error envelopes and
exit codes. This page lists what machine consumers can rely on.

## Stable surfaces

- `zscalerctl --format json machine manifest`: the `machine.v1` capability
  manifest of resource reads (list, get, show), with each resource's shape and
  get key. Its schema is
  [machine-manifest.schema.json](../schema/machine-manifest.schema.json).
  ZDX reports are not in it; read them with `zdx <resource> report`.
- `zscalerctl introspect`, version `3`: every command, flag, argument and
  output field, with structured `effects` (local file writes, network,
  configuration-dependent provider processes). Its top-level `read_only`
  means the tenant is never changed.
- JSON and NDJSON output of resource commands, `--count-by` counts and bounded
  `--limit`/`--offset` pages.
- The stderr error envelope and the exit codes in [VERSIONING.md](../VERSIONING.md).
- The JSON Schemas in [schema/](../schema/) for dumps, diffs, errors, pages and
  configuration.

The stderr CLI error envelope and exit-code mapping are supported. The machine
error-kind taxonomy has representative fixtures for the current stable kinds:
`usage`, `unsupported_capability`, `unsupported_operation`, `unknown_resource`,
`not_found`, `invalid_resource_id`, `live_access_failed`, `canceled`,
`deadline_exceeded`, and `internal`. Further taxonomy changes need fixture
coverage and a release note.

## Error vocabulary

Envelope `message` text is diagnostic prose, not a stable surface. Branch on
the error kind, the `reason` of a live failure, and the exit code.

| Scenario | Machine kind | Envelope kind | Exit code | Internal sentinel | Notes |
| --- | --- | --- | --- | --- | --- |
| Usage error | `usage` | `usage` | `2` | `cli.ErrUsage`; `resources.ErrMissingID`; `resources.ErrUnknownField` | Both adapted command errors and bare stable machine usage kinds map to exit 2. |
| Unsupported capability/operation | `unsupported_capability`; `unsupported_operation` | same as machine kind | `4` | stable machine kind | Capability and operation discovery failures share the permanent not-found exit class; malformed request syntax remains usage/2. |
| Unknown/unsupported resource | `unknown_resource`; `unsupported_resource` | `unknown_resource` for catalog machine errors; `unsupported_resource` for the Zscaler or typed URL sentinel; `not_found` for CLI catalog misses | `4` | `resources.ErrUnknownResource`; `zscaler.ErrUnsupportedResource`; `cli.ErrNotFound` | Both stable machine kinds map directly to exit 4; CLI catalog misses unwrap to `cli.ErrNotFound`. |
| Record not found (get of nonexistent id) | `not_found` | `not_found` | `4` | `resources.ErrRecordNotFound`; `zscaler.ErrResourceNotFound`; `cli.ErrNotFound` | |
| Missing credentials | `missing_credentials` | `missing_credentials` | `3` | `zscaler.ErrMissingCredentials` | Only allow-listed missing variable names are reported. Credentials the API rejects are `live_access_failed` with reason `credentials_rejected`, exit `5`. |
| Invalid resource id | `invalid_resource_id` | `invalid_resource_id` | `2` | `resources.ErrInvalidResourceID`; `zscaler.ErrInvalidResourceID` | The id is rejected before any read. |
| Live access failure | `live_access_failed` | `live_access_failed` | `5` | `zscaler.ErrLiveAccessFailed` (non-machine paths); executor default branch | The envelope adds `reason`, `status` and `retryable`; backend messages are never shown. Entitlement failures (`forbidden`, `not_subscribed`) land here. |
| Deadline exceeded | `deadline_exceeded` | `deadline_exceeded` | `5` | `context.DeadlineExceeded` | Caller deadlines and configured HTTP client timeouts preserve deadline identity instead of collapsing into `live_access_failed`. |
| Canceled | `canceled` | `canceled` | `1` | `context.Canceled` | |
| Partial dump | — | `partial_dump` | `6` | `cli.ErrPartialDump` | For non-JSON formats, `run` returns the code without writing the JSON envelope. |
| Drift detected | — | `drift_detected` | `7` | `cli.ErrDriftDetected` | Set by `diff --fail-on-drift`. |
| Invalid config | `invalid_config` | `invalid_config` | `2` | `config.ErrInvalidConfig`; `zscaler.ErrInvalidSettings` | Raw config paths and backend text are discarded. An unusable cloud, vanity domain or auth mode keeps its message: one of a fixed set of reasons that never include the configured value. |
| Invalid proxy config | `invalid_proxy_config` | `invalid_proxy_config` | `2` | `zscaler.ErrInvalidProxyConfig` | Static wording only. |
| Internal | `internal` | `internal` | `1` | `machine.ErrorKindInternal`; default branch | Unmapped errors fall through to internal. |

## Output rules

- Data goes to stdout and diagnostics to stderr.
- JSON and NDJSON are deterministic, carry no ANSI escapes or terminal control
  bytes, and never depend on terminal width, prompts, spinners or help text.
- Every record passes the catalog allow-list projection, redaction and a final
  byte scan before it leaves the process.
- Pass `--format json` or `--format ndjson` explicitly. `--format auto` picks
  JSON when output is piped, but a PTY-based harness can look interactive and
  receive human output.

The `zia admin-audit-logs status` JSON document always includes `status`.
`progress_items_complete` is included only when the upstream response supplies
a non-null, non-negative count; an explicit zero remains `0`. Missing or null `status`, a
top-level `null`, and an empty response body are live-access failures (exit
`5`). The diagnostic performs one status operation; HTTP retries follow the
shared SDK policy. It rejects NDJSON because it returns one document.
