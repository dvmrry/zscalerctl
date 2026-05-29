# Live Smoke

Use this procedure when validating a branch against a real tenant. The smoke is
read-only, but its artifacts are still confidential operational data.

## ZPA Server Groups

Start from a clean checkout of the branch under test:

```sh
git fetch origin
git checkout feature/zpa-server-groups
git pull --ff-only
git rev-parse --short HEAD
```

The PR head for the first ZPA server-groups smoke should be the commit shown on
the pull request. Do not count a smoke that ran from a different branch.

The CLI reads explicit `ZSCALERCTL_*` environment variables. It does not read
raw SDK variables such as `ZPA_CUSTOMER_ID`, and it does not read an agent
`config.toml` unless that harness exports values into the process environment.
In fish, use exported variables:

```fish
set -gx ZSCALERCTL_CLIENT_ID "..."
set -gx ZSCALERCTL_CLIENT_SECRET_FILE "/path/to/owner-only/secret-file"
set -gx ZSCALERCTL_VANITY_DOMAIN "..."
set -gx ZSCALERCTL_CLOUD "PRODUCTION" # optional
set -gx ZSCALERCTL_ZPA_CUSTOMER_ID "..."
set -gx ZSCALERCTL_ZPA_MICROTENANT_ID "..." # optional
```

Before running the smoke, verify the values are visible to child processes
without printing secret values:

```fish
env | string match -r '^ZSCALERCTL_(CLIENT_ID|CLIENT_SECRET_FILE|VANITY_DOMAIN|CLOUD|ZPA_CUSTOMER_ID|ZPA_MICROTENANT_ID)='
```

Run the source checkout directly. This does not require a prebuilt binary:

```fish
rm -rf ./scratch-live-smoke
scripts/live-smoke.sh --require-credentials --resources zpa/server-groups --out ./scratch-live-smoke
```

Only use `--bin` when validating a specific built binary. If `--bin` points at
`./bin/zscalerctl`, build that binary first:

```fish
mkdir -p ./bin
go build -mod=vendor -o ./bin/zscalerctl ./cmd/zscalerctl
test -x ./bin/zscalerctl

rm -rf ./scratch-live-smoke
scripts/live-smoke.sh --bin ./bin/zscalerctl --require-credentials --resources zpa/server-groups --out ./scratch-live-smoke
```

The output must show that the resource argument was honored. Look for these
markers:

```text
[PASS] schema list selected 1 read/list resource(s): zpa/server-groups
[PASS] zpa server-groups list command completed
[PASS] dump command completed for selected resources
[PASS] dump zpa server-groups contains only catalog-allowed top-level fields
[PASS] dump resource files match selected catalog
[PASS] live smoke completed
```

If the script validates ZIA resources, validates every product, or does not show
`zpa/server-groups`, the `--resources` argument did not make it into the command
that ran. Do not count that run.

After a passing smoke, inspect `./scratch-live-smoke` for unexpected empty
projections, over-redaction, unknown fields, and any secret-shaped or
high-entropy rendered string that should have been dropped or redacted. Do not
commit or share the smoke artifacts.
