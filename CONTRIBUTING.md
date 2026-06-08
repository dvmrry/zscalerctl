# Contributing to zscalerctl

## Output Field Naming Convention

Output field names mirror SDK JSON tags verbatim; the drift harness enforces this. Don't rename SDK fields in output — surface as-is and document if confusing. This ensures output cleanly cross-references against the upstream Zscaler API.

Known oddities (e.g. `adminScopescopeGroupMemberEntities`) should be kept as they are returned by the upstream API/SDK.
