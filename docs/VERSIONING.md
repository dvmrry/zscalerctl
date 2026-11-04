# Versioning

`zscalerctl` follows semantic versioning from `v1.0.0`. Release tags have the
form `vMAJOR.MINOR.PATCH`, and release builds stamp the tag into
`zscalerctl version`. The v0.x tags are an earlier line and are not supported.

## What the version covers

The supported surface is what the CLI does and prints:

- commands, flags, arguments and environment variables, as listed in
  [cli/zscalerctl.md](cli/zscalerctl.md) and `zscalerctl introspect`;
- the JSON and NDJSON output of resource reads, `--count-by` counts and
  bounded list pages;
- the stderr error envelope and the exit codes below;
- dump directories (manifest, resource files, redaction report,
  `errors.ndjson`) and diff reports;
- `machine manifest` (`machine.v1`), `introspect` and `schema` output;
- the resource catalog: which resources are read and which fields render in
  each redaction mode.

The JSON Schemas in [schema/](schema/) describe these artifacts. A schema URL
never changes meaning once a release has emitted it: an incompatible shape
gets a new file and URL. Go packages under `internal/` are not a supported API.

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Success. |
| `1` | Internal error or canceled. |
| `2` | Usage error, invalid resource id, or an unusable configuration or proxy setting. |
| `3` | Credentials missing: unset, incomplete, or a secret that could not be resolved. |
| `4` | Unknown or unsupported resource or operation, a declared coverage gap, or a `get` whose id does not exist. |
| `5` | Live API failure, including deadline exceeded and credentials the API rejects. The envelope's `reason` says which (`credentials_rejected`, `forbidden`, `not_subscribed`, `rate_limited`, `timeout`, ...). |
| `6` | Partial dump: some resources failed. |
| `7` | Drift detected by `diff --fail-on-drift`. |

Changing what a code means is a breaking change.

## Which number changes

- **Major:** a breaking change to the supported surface, such as removing or
  renaming a command, flag, environment variable, resource or field, an
  incompatible output or schema change, a changed exit code, a change to
  configuration precedence, or a weaker read-only or redaction guarantee.
- **Minor:** a backward-compatible addition: a resource, a field, a command,
  a flag, an output mode or an auth path.
- **Patch:** fixes, dependency updates, documentation and internal changes
  that add nothing to the surface.

Redaction changes are named in the release notes whatever their size,
because automation and people reading output can see them. Rendering a field
in a mode that did not render it before is at least a minor release; hiding
more to fix a leak can be a patch.

## Releasing

A pull request either adds one notes file, `docs/releases/vX.Y.Z.md`, or
carries the `no-release` label; the `release-notes` check requires exactly
one of the two. The file name is the version and must be the next patch,
minor or major after the latest tag; the file's contents are the release
notes. Released notes are never edited; a change may delete old ones, since
the published release keeps its notes.

Merging a pull request that adds a notes file releases that version: the
release workflow runs the release gates, builds and signs the archives,
attests their provenance, tags the merge and publishes the GitHub release
with those notes. Tags are made only by that workflow.
