# Resource Reference

**Note on Output Fields:** Output field names mirror the upstream Zscaler SDK/API JSON tags verbatim, including known upstream oddities (such as the doubled-scope `adminScopescopeGroupMemberEntities`). This ensures output cross-references cleanly against the API. Do not rename SDK fields in output.

This document describes the currently enabled resource catalog. The catalog is
the output allow-list: reader adapters map SDK response shapes into source
records, and projection decides which fields can render for each redaction mode.

Each resource's field table is generated from the catalog (`make resources-doc`)
and a test fails when it drifts, so the table is what the binary renders: every
field path, nested fields included, with its type, classification and the modes
that render it. A field marked `never` names its reason (credential, token, key
material, opaque blob or unstructured). Field notes and prose below a table are
hand-written. `zscalerctl schema` prints the same catalog.

Booleans render as the SDK decodes them: a `false` that the SDK's `omitempty` tags would drop is kept, so an API that omits a boolean reads as `false`. A few resources read through pointer-boolean types and render an omitted boolean as `null` instead (noted per resource).

## Redaction Modes

`standard` is the default; `share` and `paranoid` are chosen with
`--redaction` or `ZSCALERCTL_REDACTION`. Each field's table row lists the modes
that render it.

- `standard`: local operational use. Renders every reviewed field, including
  names, emails, free text and other identifying values, with secret scanning
  and rendered-string high-entropy token scanning still applied. Structured
  display-name fields skip only the high-entropy heuristic in this mode.
- `share`: output for tickets, reviews and chat. Drops free text and sensitive
  identifiers; rule references such as users, groups and locations render as
  ids only.
- `paranoid`: also drops most tenant configuration, keeping ids, booleans,
  enums, states and a reviewed set of structural settings. A parent list that
  paranoid does not render takes its children with it.

`share` and `paranoid` scrub values the same way; they differ only in which
fields render.

All fields, including allowed strings, pass through the final redaction backstop
before stdout or dump files. Rendered string values usually receive a
conservative high-entropy token scan for bare unlabeled secret material.
Structured display-name fields such as `name`, `configuredName`, and
`displayName` skip only the high-entropy heuristic in `standard` mode so long
cloud-style identifiers remain readable during local operation; an unlabeled
key pasted into such a name is not detected, by design. `share` and
`paranoid` redact high-entropy display-name values. Values after generic credential labels such as `key:`, `token`, or
`Bearer` redact in every field and mode when they are credential-shaped. Self-describing secrets such
as `psk=...`, credential URLs, JWTs, and private keys still redact in display
names in every mode. Canonical UUIDs and contextual git commit SHAs are
preserved. In `standard` mode, structured rendered strings also preserve
compact UUIDs and 40/64-character hex fingerprints; `share` and `paranoid`
redact those fingerprint-shaped values. Free-text prose may redact bare hashes
without context.

## Selective Dumps

By default, `zscalerctl dump` collects every catalog resource in the selected
products. Use `--resources` to limit a dump to specific resource names:

```sh
zscalerctl dump --products zia --resources locations,static-ips --output ./scratch-live-dump
zscalerctl dump --resources zia/locations --output ./scratch-live-dump-locations
```

Unqualified names are matched within the selected products. Product-qualified
names use `product/name` and are the safer form once multiple products expose
similarly named resources. Unknown resources fail before live API access.

Dump commands fail closed by default. If a selected resource fails, no files are
written. Use `--continue-on-error` only when a partial dump is acceptable; the
manifest is marked `partial`, successful resources remain in `resources/`, and
value-free per-resource failures are written to `errors.ndjson`.

Dump builds and validates a private sibling staging directory before publishing
the complete directory. It refuses to overwrite files by default. An existing
directory tree containing only empty directories remains usable only when the
filesystem supports atomic directory exchange; otherwise the destination must
not already exist. On POSIX, the output's immediate parent must be owned by the
current user and must not be group- or world-writable; a writable ancestor is
accepted only when sticky-directory rules protect the next user-owned path
component. macOS rejects permit or unknown extended-ACL entries anywhere in
that resolved ancestry; deny-only ACLs remain acceptable. Use `--force` only to
replace a complete zscalerctl dump directory from a previous run. Replacement
requires atomic directory exchange, an exact supported manifest schema,
reconciled manifest, redaction-report, resource-shape/count and error metadata,
and no foreign artifact files. Windows requires an operator-restricted parent
DACL, can publish a new dump directory atomically, and fails closed when
replacement of an existing directory would require a non-atomic backup
sequence.

Failed cleanup or successful forced replacement can leave an owner-only
`.zscalerctl-discard-*` or `.zscalerctl-cleanup-*` directory beside the output,
normally containing only an empty `root/` directory. These skeletons contain no
dump data. Neither public pathname is unlinked: deleting through a public
ancestor after an identity check would reintroduce a same-name substitution
race.

If an extraordinary concurrent mutation or I/O failure prevents both cleanup
and whole-root restoration, a discard directory may instead retain `root/`.
Treat any retained quarantine containing `root/` as confidential dump data;
normal and all preflight-detectable failure paths either restore the staging
root or leave only the empty quarantine skeleton.

## Dump Diffs

`zscalerctl diff` compares two existing dump directories locally; it does not
schedule collection or contact Zscaler. Use the same `--products` and
`--resources` narrowing grammar as `dump`:

```sh
zscalerctl --format json diff ./old-dump ./new-dump --resources zia/locations
zscalerctl diff ./old-dump ./new-dump --fail-on-drift
```

Diff rejects redaction-mode mismatches, different collection scopes among the
resources selected for comparison, and partial dumps by default
(`--allow-partial` is explicit opt-in). A resource absent from a manifest is
“not selected,” not an empty successful collection. Under
`--allow-partial`, failed resources are reported as not compared and never
converted into fabricated additions or removals. Keyed resources report added,
removed, and changed records; singleton resources report changes to their single
record; list-only resources without a stable key use a canonical
non-operational content hash and report added/removed records only.

The diff flags are:

- `--detail`: add record-level rows (the individual added/removed/changed
  records) to the `table` and `pretty` reports. It is a no-op under
  `--format json`, whose report already carries the full record-level detail.
- `--ignore-operational`: ignore fields classified as operational metadata
  (such as identifiers and SDK timestamps) on keyed and singleton resources, so
  only configuration changes are reported.
- `--allow-partial`: compare dumps written as partial (some resources failed)
  instead of rejecting them.
- `--fail-on-drift`: exit `7` when added, removed, or changed records are
  detected; without it, drift is reported with exit `0`.

## ZIA Locations

Commands:

```sh
zscalerctl zia locations list
zscalerctl zia locations get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/locations (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `ipAddresses` | `array<string>` | Sensitive identifier | `standard` |
| `description` | `string` | Free text | `standard` |
| `preSharedKey` |  | Secret | never (credential) |
| `vpnCredentials` |  | Secret | never (credential) |
| `parentId` | `integer` | Operational metadata | `standard`, `share` |
| `country` | `string` | Sensitive identifier | `standard` |
| `state` | `string` | Sensitive identifier | `standard` |
| `tz` | `string` | Tenant configuration | `standard`, `share` |
| `language` | `string` | Operational metadata | `standard`, `share` |
| `profile` | `string` | Tenant configuration | `standard`, `share` |
| `upBandwidth` | `integer` | Tenant configuration | `standard`, `share` |
| `dnBandwidth` | `integer` | Tenant configuration | `standard`, `share` |
| `ports` | `array<integer>` | Sensitive identifier | `standard` |
| `authRequired` | `boolean` | Tenant configuration | `standard`, `share` |
| `basicAuthEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `digestAuthEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `kerberosAuth` | `boolean` | Tenant configuration | `standard`, `share` |
| `sslScanEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `zappSSLScanEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `xffForwardEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `surrogateIP` | `boolean` | Tenant configuration | `standard`, `share` |
| `surrogateIPEnforcedForKnownBrowsers` | `boolean` | Tenant configuration | `standard`, `share` |
| `idleTimeInMinutes` | `integer` | Operational metadata | `standard`, `share` |
| `surrogateRefreshTimeInMinutes` | `integer` | Operational metadata | `standard`, `share` |
| `ofwEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `ipsControl` | `boolean` | Tenant configuration | `standard`, `share` |
| `aupEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `cautionEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `aupBlockInternetUntilAccepted` | `boolean` | Tenant configuration | `standard`, `share` |
| `aupForceSslInspection` | `boolean` | Tenant configuration | `standard`, `share` |
| `aupTimeoutInDays` | `integer` | Operational metadata | `standard`, `share` |
| `iotDiscoveryEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `iotEnforcePolicySet` | `boolean` | Tenant configuration | `standard`, `share` |
| `cookiesAndProxy` | `boolean` | Tenant configuration | `standard`, `share` |
| `geoOverride` | `boolean` | Tenant configuration | `standard`, `share` |
| `ipv6Enabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `defaultExtranetTsPool` | `boolean` | Tenant configuration | `standard`, `share` |
| `defaultExtranetDns` | `boolean` | Tenant configuration | `standard`, `share` |
| `ipv6Dns64Prefix` | `boolean` | Tenant configuration | `standard`, `share` |
| `extranet` | `object` | Tenant configuration | `standard`, `share` |
| `extranet.id` | `integer` | Operational metadata | `standard`, `share` |
| `extranet.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetIpPool` | `object` | Tenant configuration | `standard`, `share` |
| `extranetIpPool.id` | `integer` | Operational metadata | `standard`, `share` |
| `extranetIpPool.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDns` | `object` | Tenant configuration | `standard`, `share` |
| `extranetDns.id` | `integer` | Operational metadata | `standard`, `share` |
| `extranetDns.name` | `string` | Tenant configuration | `standard`, `share` |
| `subLocScopeEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `subLocScope` | `string` | Tenant configuration | `standard`, `share` |
| `subLocScopeValues` | `array<string>` | Sensitive identifier | `standard` |
| `subLocAccIds` | `array<string>` | Sensitive identifier | `standard` |
| `otherSubLocation` | `boolean` | Operational metadata | `standard`, `share` |
| `other6SubLocation` | `boolean` | Operational metadata | `standard`, `share` |
| `ecLocation` | `boolean` | Operational metadata | `standard`, `share` |
| `dynamiclocationGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `dynamiclocationGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `dynamiclocationGroups.name` | `string` | Tenant configuration | `standard` |
| `dynamiclocationGroups.extensions` |  | Secret | never (unstructured) |
| `staticLocationGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `staticLocationGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `staticLocationGroups.name` | `string` | Tenant configuration | `standard` |
| `staticLocationGroups.extensions` |  | Secret | never (unstructured) |
| `virtualZenClusters` | `array<object>` | Tenant configuration | `standard`, `share` |
| `virtualZenClusters.id` | `integer` | Operational metadata | `standard`, `share` |
| `virtualZenClusters.name` | `string` | Tenant configuration | `standard` |
| `virtualZenClusters.extensions` |  | Secret | never (unstructured) |
| `virtualZens` | `array<object>` | Tenant configuration | `standard`, `share` |
| `virtualZens.id` | `integer` | Operational metadata | `standard`, `share` |
| `virtualZens.name` | `string` | Tenant configuration | `standard` |
| `virtualZens.extensions` |  | Secret | never (unstructured) |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: ZIA location identifier.
- `vpnCredentials`: SDK nested credentials are mapped into source records and dropped by projection.

## ZIA Location Groups

Commands:

```sh
zscalerctl zia location-groups list
zscalerctl zia location-groups get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/location-groups (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `deleted` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `groupType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `dynamicLocationGroupCriteria` | `object` | Tenant configuration | `standard`, `share` |
| `dynamicLocationGroupCriteria.name` | `object` | Tenant configuration | `standard`, `share` |
| `dynamicLocationGroupCriteria.name.matchString` | `string` | Sensitive identifier | `standard` |
| `dynamicLocationGroupCriteria.name.matchType` | `string` | Operational metadata | `standard`, `share` |
| `dynamicLocationGroupCriteria.countries` | `array<string>` | Tenant configuration | `standard`, `share` |
| `dynamicLocationGroupCriteria.city` | `object` | Tenant configuration | `standard`, `share` |
| `dynamicLocationGroupCriteria.city.matchString` | `string` | Sensitive identifier | `standard` |
| `dynamicLocationGroupCriteria.city.matchType` | `string` | Operational metadata | `standard`, `share` |
| `dynamicLocationGroupCriteria.managedBy` | `array<object>` | Tenant configuration | `standard`, `share` |
| `dynamicLocationGroupCriteria.managedBy.id` | `integer` | Operational metadata | `standard`, `share` |
| `dynamicLocationGroupCriteria.managedBy.name` | `string` | Tenant configuration | `standard` |
| `dynamicLocationGroupCriteria.managedBy.extensions` |  | Secret | never (unstructured) |
| `dynamicLocationGroupCriteria.enforceAuthentication` | `boolean` | Tenant configuration | `standard`, `share` |
| `dynamicLocationGroupCriteria.enforceAup` | `boolean` | Tenant configuration | `standard`, `share` |
| `dynamicLocationGroupCriteria.enforceFirewallControl` | `boolean` | Tenant configuration | `standard`, `share` |
| `dynamicLocationGroupCriteria.enableXffForwarding` | `boolean` | Tenant configuration | `standard`, `share` |
| `dynamicLocationGroupCriteria.enableCaution` | `boolean` | Tenant configuration | `standard`, `share` |
| `dynamicLocationGroupCriteria.enableBandwidthControl` | `boolean` | Tenant configuration | `standard`, `share` |
| `dynamicLocationGroupCriteria.profiles` | `array<string>` | Tenant configuration | `standard`, `share` |
| `comments` | `string` | Free text | `standard` |
| `locations` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locations.id` | `integer` | Operational metadata | `standard`, `share` |
| `locations.name` | `string` | Tenant configuration | `standard` |
| `locations.extensions` |  | Secret | never (unstructured) |
| `lastModTime` | `integer` | Operational metadata | `standard`, `share` |
| `predefined` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModUser` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModUser.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModUser.name` | `string` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Location group identifier.
- `deleted`: Whether the group is marked deleted.
- `groupType`: Static or dynamic location group type.
- `dynamicLocationGroupCriteria`: Reviewed dynamic-membership criteria. Name and city match strings and `managedBy` references are standard-only; nested extension maps never render.
- `predefined`: Whether the group is predefined by Zscaler.

List and dump collection request `fetchLocations=true`; this is required for
the API to populate associated location references. Zscaler location groups
can contain parent locations and sublocations independently, so the returned
array is intentionally preserved as a flat set of id/name members rather than
having parent-child relationships inferred locally.

## ZIA Rule Labels

Commands:

```sh
zscalerctl zia rule-labels list
zscalerctl zia rule-labels get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/rule-labels (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share` |
| `referencedRuleCount` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `createdBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `createdBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `createdBy.name` | `string` | Sensitive identifier | `standard` |
| `lastModifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: ZIA rule-label identifier.
- `referencedRuleCount`: Number of referencing rules.

## ZIA Auth Settings

Commands:

```sh
zscalerctl zia auth-settings list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/auth-settings (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `orgAuthType` | `string` | Operational metadata | `standard`, `share` |
| `oneTimeAuth` | `string` | Operational metadata | `standard`, `share` |
| `samlEnabled` | `boolean` | Operational metadata | `standard`, `share` |
| `kerberosEnabled` | `boolean` | Operational metadata | `standard`, `share` |
| `kerberosPwd` |  | Secret | never (credential) |
| `authFrequency` | `string` | Operational metadata | `standard`, `share` |
| `authCustomFrequency` | `integer` | Operational metadata | `standard`, `share` |
| `passwordStrength` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `passwordExpiry` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastSyncStartTime` | `integer` | Operational metadata | `standard`, `share` |
| `lastSyncEndTime` | `integer` | Operational metadata | `standard`, `share` |
| `mobileAdminSamlIdpEnabled` | `boolean` | Operational metadata | `standard`, `share` |
| `autoProvision` | `boolean` | Operational metadata | `standard`, `share` |
| `directorySyncMigrateToScimEnabled` | `boolean` | Operational metadata | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `orgAuthType`, `oneTimeAuth`: Organization authentication mode metadata.
- `samlEnabled`, `kerberosEnabled`, `mobileAdminSamlIdpEnabled`: Enabled-state flags for authentication integrations.
- `authFrequency`, `authCustomFrequency`: Authentication frequency policy metadata.
- `lastSyncStartTime`, `lastSyncEndTime`: Directory sync timestamps from the SDK.
- `autoProvision`, `directorySyncMigrateToScimEnabled`: Provisioning and migration flags.

This is a singleton settings object. The CLI exposes it through `list` so dumps
and live smoke can treat singleton resources as one-record arrays.

## ZIA Static IPs

Commands:

```sh
zscalerctl zia static-ips list
zscalerctl zia static-ips get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/static-ips (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `ipAddress` | `string` | Sensitive identifier | `standard` |
| `routableIP` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `geoOverride` | `boolean` | Operational metadata | `standard`, `share` |
| `latitude` | `number` | Sensitive identifier | `standard` |
| `longitude` | `number` | Sensitive identifier | `standard` |
| `comment` | `string` | Free text | `standard` |
| `lastModificationTime` | `integer` | Operational metadata | `standard`, `share` |
| `city` | `object` | Tenant configuration | `standard` |
| `city.id` | `integer` | Operational metadata | `standard` |
| `city.name` | `string` | Tenant configuration | `standard` |
| `managedBy` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `managedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `managedBy.name` | `string` | Tenant configuration | `standard`, `share` |
| `lastModifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Static IP identifier.
- `ipAddress`: Static IP address; dropped from `share` and `paranoid`.
- `routableIP`: Whether the address is publicly routable.
- `geoOverride`: Whether geographic coordinates are explicitly configured.
- `latitude`: Geographic coordinate; dropped from `share` and `paranoid`.
- `longitude`: Geographic coordinate; dropped from `share` and `paranoid`.

## ZIA GRE Tunnels

Commands:

```sh
zscalerctl zia gre-tunnels list
zscalerctl zia gre-tunnels get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/gre-tunnels (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `sourceIp` | `string` | Sensitive identifier | `standard` |
| `internalIpRange` | `string` | Sensitive identifier | `standard` |
| `lastModificationTime` | `integer` | Operational metadata | `standard`, `share` |
| `withinCountry` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `comment` | `string` | Free text | `standard` |
| `ipUnnumbered` | `boolean` | Operational metadata | `standard`, `share` |
| `subcloud` | `string` | Tenant configuration | `standard`, `share` |
| `primaryDestVip` | `object` | Tenant configuration | `standard`, `share` |
| `primaryDestVip.id` | `integer` | Operational metadata | `standard`, `share` |
| `primaryDestVip.virtualIp` | `string` | Sensitive identifier | `standard` |
| `primaryDestVip.privateServiceEdge` | `boolean` | Operational metadata | `standard`, `share` |
| `primaryDestVip.datacenter` | `string` | Tenant configuration | `standard`, `share` |
| `primaryDestVip.latitude` | `number` | Sensitive identifier | `standard` |
| `primaryDestVip.longitude` | `number` | Sensitive identifier | `standard` |
| `primaryDestVip.city` | `string` | Sensitive identifier | `standard` |
| `primaryDestVip.countryCode` | `string` | Sensitive identifier | `standard` |
| `primaryDestVip.region` | `string` | Sensitive identifier | `standard` |
| `secondaryDestVip` | `object` | Tenant configuration | `standard`, `share` |
| `secondaryDestVip.id` | `integer` | Operational metadata | `standard`, `share` |
| `secondaryDestVip.virtualIp` | `string` | Sensitive identifier | `standard` |
| `secondaryDestVip.privateServiceEdge` | `boolean` | Operational metadata | `standard`, `share` |
| `secondaryDestVip.datacenter` | `string` | Tenant configuration | `standard`, `share` |
| `secondaryDestVip.latitude` | `number` | Sensitive identifier | `standard` |
| `secondaryDestVip.longitude` | `number` | Sensitive identifier | `standard` |
| `secondaryDestVip.city` | `string` | Sensitive identifier | `standard` |
| `secondaryDestVip.countryCode` | `string` | Sensitive identifier | `standard` |
| `secondaryDestVip.region` | `string` | Sensitive identifier | `standard` |
| `managedBy` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `managedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `managedBy.name` | `string` | Tenant configuration | `standard`, `share` |
| `lastModifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: GRE tunnel identifier.
- `sourceIp`: Source IP address; dropped from `share` and `paranoid`.
- `internalIpRange`: Internal tunnel range; dropped from `share` and `paranoid`.
- `withinCountry`: Whether destination VIPs are restricted to the source-IP country.
- `ipUnnumbered`: Whether the tunnel uses IP unnumbered mode.
- `subcloud`: Configured subcloud restriction.

## ZIA Locations Lite

Commands:

```sh
zscalerctl zia locations-lite list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/locations-lite (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `parentId` | `integer` | Operational metadata | `standard`, `share` |
| `ecLocation` | `boolean` | Operational metadata | `standard`, `share` |
| `otherSubLocation` | `boolean` | Operational metadata | `standard`, `share` |
| `aupBlockInternetUntilAccepted` | `boolean` | Tenant configuration | `standard`, `share` |
| `aupEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `aupForceSslInspection` | `boolean` | Tenant configuration | `standard`, `share` |
| `cautionEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `digestAuthEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `iotDiscoveryEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `ipsControl` | `boolean` | Tenant configuration | `standard`, `share` |
| `kerberosAuth` | `boolean` | Tenant configuration | `standard`, `share` |
| `ofwEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `surrogateIP` | `boolean` | Tenant configuration | `standard`, `share` |
| `surrogateIPEnforcedForKnownBrowsers` | `boolean` | Tenant configuration | `standard`, `share` |
| `tz` | `string` | Tenant configuration | `standard`, `share` |
| `xffForwardEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `zappSslScanEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Location identifier; predefined locations such as Road Warrior (`-3`) appear here and not in `zia locations`.
- `parentId`: Parent location identifier.
- `ecLocation`, `otherSubLocation`: Location kind flags.
- `tz`: Time zone.
- `aupEnabled`, `aupBlockInternetUntilAccepted`, `aupForceSslInspection`, `cautionEnabled`, `digestAuthEnabled`, `kerberosAuth`, `surrogateIP`, `surrogateIPEnforcedForKnownBrowsers`, `xffForwardEnabled`, `ofwEnabled`, `ipsControl`, `iotDiscoveryEnabled`, `zappSslScanEnabled`: Location settings, classified as on `zia locations`.

`GET /locations/lite` is a short view, not a superset of `zia locations`: on a
test tenant it returned 3 records, one of them a regular location. Read without
filters, as zscalerctl reads it, the API leaves out sublocations and every
parent location that has sublocations unless `includeSubLocations` or
`includeParentLocations` is set (per the SDK docs; zscalerctl sends neither), so
a location missing here may still be configured; read `zia locations` and `zia
sublocations` for those. The fields are the keys a live read returned; keys the
SDK model lists but no live read returned (`ipv6Enabled`, `other6SubLocation`,
`subLocScope*`, `subLocAccIds`) are not read. Rule `locations` references list
it as a target beside `zia locations` and `zia sublocations`, so `refs` resolves
predefined location ids. Because a missing id could be in any target, a `refs`
run that cannot read `locations-lite` (no role access, or a `--from-dump` saved
before the resource existed) reports every rule `locations` field as
`target_not_checked`; dump again to restore those checks.

## ZIA Sublocations

Commands:

```sh
zscalerctl zia sublocations list
zscalerctl zia sublocations get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/sublocations (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `parentId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `ipAddresses` | `array<string>` | Sensitive identifier | `standard` |
| `ports` | `array<integer>` | Sensitive identifier | `standard` |
| `description` | `string` | Free text | `standard` |
| `profile` | `string` | Tenant configuration | `standard`, `share` |
| `country` | `string` | Sensitive identifier | `standard` |
| `state` | `string` | Sensitive identifier | `standard` |
| `tz` | `string` | Operational metadata | `standard`, `share` |
| `authRequired` | `boolean` | Tenant configuration | `standard`, `share` |
| `sslScanEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `ofwEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `ipsControl` | `boolean` | Tenant configuration | `standard`, `share` |
| `language` | `string` | Operational metadata | `standard`, `share` |
| `upBandwidth` | `integer` | Tenant configuration | `standard`, `share` |
| `dnBandwidth` | `integer` | Tenant configuration | `standard`, `share` |
| `basicAuthEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `digestAuthEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `kerberosAuth` | `boolean` | Tenant configuration | `standard`, `share` |
| `zappSSLScanEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `xffForwardEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `surrogateIP` | `boolean` | Tenant configuration | `standard`, `share` |
| `surrogateIPEnforcedForKnownBrowsers` | `boolean` | Tenant configuration | `standard`, `share` |
| `idleTimeInMinutes` | `integer` | Operational metadata | `standard`, `share` |
| `surrogateRefreshTimeInMinutes` | `integer` | Operational metadata | `standard`, `share` |
| `aupEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `cautionEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `aupBlockInternetUntilAccepted` | `boolean` | Tenant configuration | `standard`, `share` |
| `aupForceSslInspection` | `boolean` | Tenant configuration | `standard`, `share` |
| `aupTimeoutInDays` | `integer` | Operational metadata | `standard`, `share` |
| `iotDiscoveryEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `iotEnforcePolicySet` | `boolean` | Tenant configuration | `standard`, `share` |
| `cookiesAndProxy` | `boolean` | Tenant configuration | `standard`, `share` |
| `otherSubLocation` | `boolean` | Operational metadata | `standard`, `share` |
| `other6SubLocation` | `boolean` | Operational metadata | `standard`, `share` |
| `geoOverride` | `boolean` | Tenant configuration | `standard`, `share` |
| `ipv6Enabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `defaultExtranetTsPool` | `boolean` | Tenant configuration | `standard`, `share` |
| `defaultExtranetDns` | `boolean` | Tenant configuration | `standard`, `share` |
| `ipv6Dns64Prefix` | `boolean` | Tenant configuration | `standard`, `share` |
| `extranet` | `object` | Tenant configuration | `standard`, `share` |
| `extranet.id` | `integer` | Operational metadata | `standard`, `share` |
| `extranet.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetIpPool` | `object` | Tenant configuration | `standard`, `share` |
| `extranetIpPool.id` | `integer` | Operational metadata | `standard`, `share` |
| `extranetIpPool.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDns` | `object` | Tenant configuration | `standard`, `share` |
| `extranetDns.id` | `integer` | Operational metadata | `standard`, `share` |
| `extranetDns.name` | `string` | Tenant configuration | `standard`, `share` |
| `subLocScopeEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `subLocScope` | `string` | Tenant configuration | `standard`, `share` |
| `subLocScopeValues` | `array<string>` | Sensitive identifier | `standard` |
| `subLocAccIds` | `array<string>` | Sensitive identifier | `standard` |
| `ecLocation` | `boolean` | Operational metadata | `standard`, `share` |
| `dynamiclocationGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `dynamiclocationGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `dynamiclocationGroups.name` | `string` | Tenant configuration | `standard` |
| `dynamiclocationGroups.extensions` |  | Secret | never (unstructured) |
| `staticLocationGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `staticLocationGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `staticLocationGroups.name` | `string` | Tenant configuration | `standard` |
| `staticLocationGroups.extensions` |  | Secret | never (unstructured) |
| `virtualZenClusters` | `array<object>` | Tenant configuration | `standard`, `share` |
| `virtualZenClusters.id` | `integer` | Operational metadata | `standard`, `share` |
| `virtualZenClusters.name` | `string` | Tenant configuration | `standard` |
| `virtualZenClusters.extensions` |  | Secret | never (unstructured) |
| `virtualZens` | `array<object>` | Tenant configuration | `standard`, `share` |
| `virtualZens.id` | `integer` | Operational metadata | `standard`, `share` |
| `virtualZens.name` | `string` | Tenant configuration | `standard` |
| `virtualZens.extensions` |  | Secret | never (unstructured) |
| `vpnCredentials` |  | Secret | never (credential) |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Sublocation identifier.
- `parentId`: Parent location identifier.
- `ipAddresses`: Sublocation egress/internal/GRE tunnel IP entries; dropped from `share` and `paranoid`.
- `ports`: Associated ports; dropped from `share` and `paranoid`.
- `profile`: Location traffic profile tag.
- `country`, `state`: Geographic placement; dropped from `share` and `paranoid`.
- `tz`: Time zone.
- `authRequired`, `sslScanEnabled`, `ofwEnabled`, `ipsControl`: Selected inherited/enforced policy controls.
- `vpnCredentials`: SDK nested credentials are mapped into source records and dropped by projection.

## ZIA Ssl Inspection Rules

Commands:

```sh
zscalerctl zia ssl-inspection-rules list
zscalerctl zia ssl-inspection-rules get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/ssl-inspection-rules (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `action` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `action.type` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `action.showEUN` | `boolean` | Tenant configuration | `standard`, `share` |
| `action.showEUNATP` | `boolean` | Tenant configuration | `standard`, `share` |
| `action.overrideDefaultCertificate` | `boolean` | Tenant configuration | `standard`, `share` |
| `action.sslInterceptionCert` | `object` | Tenant configuration | `standard`, `share` |
| `action.sslInterceptionCert.id` | `integer` | Operational metadata | `standard`, `share` |
| `action.sslInterceptionCert.name` | `string` | Tenant configuration | `standard`, `share` |
| `action.sslInterceptionCert.defaultCertificate` | `boolean` | Tenant configuration | `standard`, `share` |
| `action.decryptSubActions` | `object` | Tenant configuration | `standard`, `share` |
| `action.decryptSubActions.serverCertificates` | `string` | Tenant configuration | `standard`, `share` |
| `action.decryptSubActions.ocspCheck` | `boolean` | Tenant configuration | `standard`, `share` |
| `action.decryptSubActions.blockSslTrafficWithNoSniEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `action.decryptSubActions.minClientTLSVersion` | `string` | Tenant configuration | `standard`, `share` |
| `action.decryptSubActions.minServerTLSVersion` | `string` | Tenant configuration | `standard`, `share` |
| `action.decryptSubActions.blockUndecrypt` | `boolean` | Tenant configuration | `standard`, `share` |
| `action.decryptSubActions.http2Enabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `action.doNotDecryptSubActions` | `object` | Tenant configuration | `standard`, `share` |
| `action.doNotDecryptSubActions.bypassOtherPolicies` | `boolean` | Tenant configuration | `standard`, `share` |
| `action.doNotDecryptSubActions.serverCertificates` | `string` | Tenant configuration | `standard`, `share` |
| `action.doNotDecryptSubActions.ocspCheck` | `boolean` | Tenant configuration | `standard`, `share` |
| `action.doNotDecryptSubActions.blockSslTrafficWithNoSniEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `action.doNotDecryptSubActions.minTLSVersion` | `string` | Tenant configuration | `standard`, `share` |
| `state` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `accessControl` | `string` | Operational metadata | `standard`, `share` |
| `order` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `rank` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `urlCategories` | `array<string>` | Tenant configuration | `standard`, `share` |
| `platforms` | `array<string>` | Tenant configuration | `standard`, `share` |
| `cloudApplications` | `array<string>` | Tenant configuration | `standard`, `share` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share` |
| `defaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `predefined` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `roadWarriorForKerberos` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `userAgentTypes` | `array<string>` | Tenant configuration | `standard`, `share` |
| `deviceTrustLevels` | `array<string>` | Tenant configuration | `standard`, `share` |
| `lastModifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
| `labels` | `array<object>` | Tenant configuration | `standard`, `share` |
| `labels.id` | `integer` | Operational metadata | `standard`, `share` |
| `labels.name` | `string` | Tenant configuration | `standard`, `share` |
| `labels.extensions` |  | Secret | never (unstructured) |
| `timeWindows` | `array<object>` | Tenant configuration | `standard`, `share` |
| `timeWindows.id` | `integer` | Operational metadata | `standard`, `share` |
| `timeWindows.name` | `string` | Tenant configuration | `standard`, `share` |
| `timeWindows.extensions` |  | Secret | never (unstructured) |
| `locations` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locations.id` | `integer` | Operational metadata | `standard`, `share` |
| `locations.name` | `string` | Tenant configuration | `standard` |
| `locations.extensions` |  | Secret | never (unstructured) |
| `locationGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locationGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `locationGroups.name` | `string` | Tenant configuration | `standard` |
| `locationGroups.extensions` |  | Secret | never (unstructured) |
| `departments` | `array<object>` | Tenant configuration | `standard`, `share` |
| `departments.id` | `integer` | Operational metadata | `standard`, `share` |
| `departments.name` | `string` | Tenant configuration | `standard` |
| `departments.extensions` |  | Secret | never (unstructured) |
| `groups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `groups.id` | `integer` | Operational metadata | `standard`, `share` |
| `groups.name` | `string` | Tenant configuration | `standard` |
| `groups.extensions` |  | Secret | never (unstructured) |
| `users` | `array<object>` | Tenant configuration | `standard`, `share` |
| `users.id` | `integer` | Operational metadata | `standard`, `share` |
| `users.name` | `string` | Tenant configuration | `standard` |
| `users.extensions` |  | Secret | never (unstructured) |
| `deviceGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `deviceGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `deviceGroups.name` | `string` | Tenant configuration | `standard` |
| `deviceGroups.extensions` |  | Secret | never (unstructured) |
| `devices` | `array<object>` | Tenant configuration | `standard`, `share` |
| `devices.id` | `integer` | Operational metadata | `standard`, `share` |
| `devices.name` | `string` | Tenant configuration | `standard` |
| `devices.extensions` |  | Secret | never (unstructured) |
| `destIpGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `destIpGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `destIpGroups.name` | `string` | Tenant configuration | `standard` |
| `destIpGroups.extensions` |  | Secret | never (unstructured) |
| `sourceIpGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `sourceIpGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `sourceIpGroups.name` | `string` | Tenant configuration | `standard` |
| `sourceIpGroups.extensions` |  | Secret | never (unstructured) |
| `proxyGateways` | `array<object>` | Tenant configuration | `standard`, `share` |
| `proxyGateways.id` | `integer` | Operational metadata | `standard`, `share` |
| `proxyGateways.name` | `string` | Tenant configuration | `standard` |
| `proxyGateways.extensions` |  | Secret | never (unstructured) |
| `zpaAppSegments` | `array<object>` | Tenant configuration | `standard` |
| `zpaAppSegments.id` | `integer` | Operational metadata | `standard` |
| `zpaAppSegments.name` | `string` | Tenant configuration | `standard` |
| `zpaAppSegments.externalId` | `string` | Operational metadata | `standard` |
| `zpaAppSegments.extensions` |  | Secret | never (unstructured) |
| `workloadGroups` | `array<object>` | Tenant configuration | `standard` |
| `workloadGroups.id` | `integer` | Operational metadata | `standard` |
| `workloadGroups.name` | `string` | Tenant configuration | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: SSL inspection rule identifier.
- `action.type`: Rule action such as decrypt or do-not-decrypt.
- `state`: Rule state.
- `order`, `rank`: Rule ordering metadata.
- `urlCategories`: URL category identifiers referenced by the rule.
- `platforms`, `cloudApplications`: Selected non-principal criteria lists.
- `defaultRule`, `predefined`: Whether the rule is a default or predefined rule.

## ZIA Url Categories

Commands:

```sh
zscalerctl zia url-categories list
zscalerctl zia url-categories get <id>
```

The list walks the API's pages with a bounded paginator and includes ordinary
`URL_CATEGORY` records and `TLD_CATEGORY` records. The paginator accounts for
the service clamping a requested page size to 20 instead of treating the first
short response as complete, and for tenants whose service ignores pagination
and returns every category on each request (when that is fewer than the
requested page size). List responses request the API's count-only
payload; use `get <id>` when the category's members are needed. TLD members use
the existing `urls` field.

Fields:

<!-- BEGIN GENERATED FIELDS: zia/url-categories (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `configuredName` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `type` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `customCategory` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `editable` | `boolean` | Operational metadata | `standard`, `share` |
| `customUrlsCount` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `urlsRetainingParentCategoryCount` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `customIpRangesCount` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `ipRangesRetainingParentCategoryCount` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `categoryGroup` | `string` | Tenant configuration | `standard`, `share` |
| `superCategory` | `string` | Tenant configuration | `standard`, `share` |
| `urlType` | `string` | Operational metadata | `standard`, `share` |
| `urlKeywordCounts` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `urlKeywordCounts.totalUrlCount` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `urlKeywordCounts.retainParentUrlCount` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `urlKeywordCounts.totalKeywordCount` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `urlKeywordCounts.retainParentKeywordCount` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `keywords` | `array<string>` | Sensitive identifier | `standard` |
| `keywordsRetainingParentCategory` | `array<string>` | Sensitive identifier | `standard` |
| `urls` | `array<string>` | Sensitive identifier | `standard` |
| `dbCategorizedUrls` | `array<string>` | Sensitive identifier | `standard` |
| `ipRanges` | `array<string>` | Sensitive identifier | `standard` |
| `ipRangesRetainingParentCategory` | `array<string>` | Sensitive identifier | `standard` |
| `regexPatterns` | `array<string>` | Sensitive identifier | `standard` |
| `regexPatternsRetainingParentCategory` | `array<string>` | Sensitive identifier | `standard` |
| `scopes` | `array<object>` | Tenant configuration | `standard`, `share` |
| `scopes.Type` | `string` | Operational metadata | `standard`, `share` |
| `scopes.ScopeEntities` | `array<object>` | Tenant configuration | `standard`, `share` |
| `scopes.ScopeEntities.id` | `integer` | Operational metadata | `standard`, `share` |
| `scopes.ScopeEntities.name` | `string` | Tenant configuration | `standard` |
| `scopes.ScopeEntities.extensions` |  | Secret | never (unstructured) |
| `scopes.scopeGroupMemberEntities` | `array<object>` | Tenant configuration | `standard`, `share` |
| `scopes.scopeGroupMemberEntities.id` | `integer` | Operational metadata | `standard`, `share` |
| `scopes.scopeGroupMemberEntities.name` | `string` | Tenant configuration | `standard` |
| `scopes.scopeGroupMemberEntities.extensions` |  | Secret | never (unstructured) |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: URL category identifier.
- `configuredName`: Custom category display name when present.
- `type`: Category type.
- `customCategory`: Whether the category is custom.
- `editable`: Whether the caller can edit the category.
- `customUrlsCount`, `customIpRangesCount`, `urlsRetainingParentCategoryCount`, `ipRangesRetainingParentCategoryCount`: Category member counts.
- `categoryGroup`, `superCategory`: Category grouping metadata.
- `urlType`: Match type such as exact or regex.
- `urls`, `dbCategorizedUrls`, `ipRanges`, `ipRangesRetainingParentCategory`, `keywords`, `keywordsRetainingParentCategory`, `regexPatterns`, `regexPatternsRetainingParentCategory`: Raw category members. Values are scanned before output.

## ZIA Url Filtering Rules

Commands:

```sh
zscalerctl zia url-filtering-rules list
zscalerctl zia url-filtering-rules get <id>
```

`list` and dump collection explicitly request 100-record pages until the API
returns a short page. The walk fails closed after 1,000 full pages or on any
page error, so a partial ruleset is never presented as complete.

Fields:

<!-- BEGIN GENERATED FIELDS: zia/url-filtering-rules (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `state` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `order` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `rank` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `action` | `string` | Tenant configuration | `standard`, `share` |
| `protocols` | `array<string>` | Tenant configuration | `standard`, `share` |
| `requestMethods` | `array<string>` | Tenant configuration | `standard`, `share` |
| `urlCategories` | `array<string>` | Tenant configuration | `standard`, `share` |
| `urlCategories2` | `array<string>` | Tenant configuration | `standard`, `share` |
| `userRiskScoreLevels` | `array<string>` | Tenant configuration | `standard`, `share` |
| `userAgentTypes` | `array<string>` | Tenant configuration | `standard`, `share` |
| `deviceTrustLevels` | `array<string>` | Tenant configuration | `standard`, `share` |
| `lastModifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
| `sourceCountries` | `array<string>` | Operational metadata | `standard`, `share` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share` |
| `enforceTimeValidity` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `validityStartTime` | `integer` | Operational metadata | `standard`, `share` |
| `validityEndTime` | `integer` | Operational metadata | `standard`, `share` |
| `validityTimeZoneId` | `string` | Operational metadata | `standard`, `share` |
| `blockOverride` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `timeQuota` | `integer` | Operational metadata | `standard`, `share` |
| `sizeQuota` | `integer` | Operational metadata | `standard`, `share` |
| `ciparule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `browserEunTemplateId` | `integer` | Sensitive identifier | `standard` |
| `endUserNotificationUrl` | `string` | Sensitive identifier | `standard` |
| `cbiProfileId` | `integer` | Sensitive identifier | `standard` |
| `cbiProfile` | `object` | Tenant configuration | `standard` |
| `cbiProfile.id` | `string` | Operational metadata | `standard` |
| `cbiProfile.name` | `string` | Tenant configuration | `standard` |
| `cbiProfile.url` | `string` | Sensitive identifier | `standard` |
| `cbiProfile.profileSeq` | `integer` | Operational metadata | `standard` |
| `labels` | `array<object>` | Tenant configuration | `standard`, `share` |
| `labels.id` | `integer` | Operational metadata | `standard`, `share` |
| `labels.name` | `string` | Tenant configuration | `standard`, `share` |
| `labels.extensions` |  | Secret | never (unstructured) |
| `timeWindows` | `array<object>` | Tenant configuration | `standard`, `share` |
| `timeWindows.id` | `integer` | Operational metadata | `standard`, `share` |
| `timeWindows.name` | `string` | Tenant configuration | `standard`, `share` |
| `timeWindows.extensions` |  | Secret | never (unstructured) |
| `locations` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locations.id` | `integer` | Operational metadata | `standard`, `share` |
| `locations.name` | `string` | Tenant configuration | `standard` |
| `locations.extensions` |  | Secret | never (unstructured) |
| `locationGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locationGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `locationGroups.name` | `string` | Tenant configuration | `standard` |
| `locationGroups.extensions` |  | Secret | never (unstructured) |
| `sourceIpGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `sourceIpGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `sourceIpGroups.name` | `string` | Tenant configuration | `standard` |
| `sourceIpGroups.extensions` |  | Secret | never (unstructured) |
| `departments` | `array<object>` | Tenant configuration | `standard`, `share` |
| `departments.id` | `integer` | Operational metadata | `standard`, `share` |
| `departments.name` | `string` | Tenant configuration | `standard` |
| `departments.extensions` |  | Secret | never (unstructured) |
| `groups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `groups.id` | `integer` | Operational metadata | `standard`, `share` |
| `groups.name` | `string` | Tenant configuration | `standard` |
| `groups.extensions` |  | Secret | never (unstructured) |
| `users` | `array<object>` | Tenant configuration | `standard`, `share` |
| `users.id` | `integer` | Operational metadata | `standard`, `share` |
| `users.name` | `string` | Tenant configuration | `standard` |
| `users.extensions` |  | Secret | never (unstructured) |
| `deviceGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `deviceGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `deviceGroups.name` | `string` | Tenant configuration | `standard` |
| `deviceGroups.extensions` |  | Secret | never (unstructured) |
| `devices` | `array<object>` | Tenant configuration | `standard`, `share` |
| `devices.id` | `integer` | Operational metadata | `standard`, `share` |
| `devices.name` | `string` | Tenant configuration | `standard` |
| `devices.extensions` |  | Secret | never (unstructured) |
| `overrideUsers` | `array<object>` | Tenant configuration | `standard`, `share` |
| `overrideUsers.id` | `integer` | Operational metadata | `standard`, `share` |
| `overrideUsers.name` | `string` | Tenant configuration | `standard` |
| `overrideUsers.extensions` |  | Secret | never (unstructured) |
| `overrideGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `overrideGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `overrideGroups.name` | `string` | Tenant configuration | `standard` |
| `overrideGroups.extensions` |  | Secret | never (unstructured) |
| `httpHeaderProfiles` | `array<object>` | Tenant configuration | `standard`, `share` |
| `httpHeaderProfiles.id` | `integer` | Operational metadata | `standard`, `share` |
| `httpHeaderProfiles.name` | `string` | Tenant configuration | `standard` |
| `httpHeaderProfiles.extensions` |  | Secret | never (unstructured) |
| `httpHeaderActionProfiles` | `array<object>` | Tenant configuration | `standard`, `share` |
| `httpHeaderActionProfiles.id` | `integer` | Operational metadata | `standard`, `share` |
| `httpHeaderActionProfiles.name` | `string` | Tenant configuration | `standard` |
| `httpHeaderActionProfiles.extensions` |  | Secret | never (unstructured) |
| `workloadGroups` | `array<object>` | Tenant configuration | `standard` |
| `workloadGroups.id` | `integer` | Operational metadata | `standard` |
| `workloadGroups.name` | `string` | Tenant configuration | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: URL filtering rule identifier.
- `state`, `order`, `rank`: Rule state and ordering metadata.
- `action`, `protocols`, `requestMethods`, `urlCategories`, `urlCategories2`, `userRiskScoreLevels`, `userAgentTypes`: Primary rule criteria and action fields.
- `endUserNotificationUrl`, `cbiProfileId`: Local-only URL/browser isolation references.
- `labels`, `timeWindows`: Nested references render reviewed `id`/`name` fields only.

## ZIA Firewall Filtering Rules

Commands:

```sh
zscalerctl zia firewall-filtering-rules list
zscalerctl zia firewall-filtering-rules get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/firewall-filtering-rules (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `state` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `order` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `rank` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `action` | `string` | Tenant configuration | `standard`, `share` |
| `accessControl` | `string` | Operational metadata | `standard`, `share` |
| `enableFullLogging` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `defaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `predefined` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `excludeContextShieldEndPoint` | `boolean` | Tenant configuration | `standard`, `share` |
| `isEunEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `eunTemplateId` | `integer` | Tenant configuration | `standard`, `share` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share` |
| `sourceCountries` | `array<string>` | Operational metadata | `standard`, `share` |
| `destCountries` | `array<string>` | Operational metadata | `standard`, `share` |
| `excludeSrcCountries` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `nwApplications` | `array<string>` | Tenant configuration | `standard`, `share` |
| `srcIps` | `array<string>` | Sensitive identifier | `standard` |
| `destAddresses` | `array<string>` | Sensitive identifier | `standard` |
| `destIpCategories` | `array<string>` | Sensitive identifier | `standard` |
| `deviceTrustLevels` | `array<string>` | Tenant configuration | `standard`, `share` |
| `lastModifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
| `labels` | `array<object>` | Tenant configuration | `standard`, `share` |
| `labels.id` | `integer` | Operational metadata | `standard`, `share` |
| `labels.name` | `string` | Tenant configuration | `standard`, `share` |
| `labels.extensions` |  | Secret | never (unstructured) |
| `timeWindows` | `array<object>` | Tenant configuration | `standard`, `share` |
| `timeWindows.id` | `integer` | Operational metadata | `standard`, `share` |
| `timeWindows.name` | `string` | Tenant configuration | `standard`, `share` |
| `timeWindows.extensions` |  | Secret | never (unstructured) |
| `locations` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locations.id` | `integer` | Operational metadata | `standard`, `share` |
| `locations.name` | `string` | Tenant configuration | `standard` |
| `locations.extensions` |  | Secret | never (unstructured) |
| `locationGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locationGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `locationGroups.name` | `string` | Tenant configuration | `standard` |
| `locationGroups.extensions` |  | Secret | never (unstructured) |
| `departments` | `array<object>` | Tenant configuration | `standard`, `share` |
| `departments.id` | `integer` | Operational metadata | `standard`, `share` |
| `departments.name` | `string` | Tenant configuration | `standard` |
| `departments.extensions` |  | Secret | never (unstructured) |
| `groups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `groups.id` | `integer` | Operational metadata | `standard`, `share` |
| `groups.name` | `string` | Tenant configuration | `standard` |
| `groups.extensions` |  | Secret | never (unstructured) |
| `users` | `array<object>` | Tenant configuration | `standard`, `share` |
| `users.id` | `integer` | Operational metadata | `standard`, `share` |
| `users.name` | `string` | Tenant configuration | `standard` |
| `users.extensions` |  | Secret | never (unstructured) |
| `srcIpGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `srcIpGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `srcIpGroups.name` | `string` | Tenant configuration | `standard` |
| `srcIpGroups.extensions` |  | Secret | never (unstructured) |
| `destIpGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `destIpGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `destIpGroups.name` | `string` | Tenant configuration | `standard` |
| `destIpGroups.extensions` |  | Secret | never (unstructured) |
| `nwServices` | `array<object>` | Tenant configuration | `standard`, `share` |
| `nwServices.id` | `integer` | Operational metadata | `standard`, `share` |
| `nwServices.name` | `string` | Tenant configuration | `standard` |
| `nwServices.extensions` |  | Secret | never (unstructured) |
| `nwServiceGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `nwServiceGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `nwServiceGroups.name` | `string` | Tenant configuration | `standard` |
| `nwServiceGroups.extensions` |  | Secret | never (unstructured) |
| `nwApplicationGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `nwApplicationGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `nwApplicationGroups.name` | `string` | Tenant configuration | `standard` |
| `nwApplicationGroups.extensions` |  | Secret | never (unstructured) |
| `appServices` | `array<object>` | Tenant configuration | `standard`, `share` |
| `appServices.id` | `integer` | Operational metadata | `standard`, `share` |
| `appServices.name` | `string` | Tenant configuration | `standard` |
| `appServices.extensions` |  | Secret | never (unstructured) |
| `appServiceGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `appServiceGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `appServiceGroups.name` | `string` | Tenant configuration | `standard` |
| `appServiceGroups.extensions` |  | Secret | never (unstructured) |
| `deviceGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `deviceGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `deviceGroups.name` | `string` | Tenant configuration | `standard` |
| `deviceGroups.extensions` |  | Secret | never (unstructured) |
| `devices` | `array<object>` | Tenant configuration | `standard`, `share` |
| `devices.id` | `integer` | Operational metadata | `standard`, `share` |
| `devices.name` | `string` | Tenant configuration | `standard` |
| `devices.extensions` |  | Secret | never (unstructured) |
| `zpaAppSegments` | `array<object>` | Tenant configuration | `standard` |
| `zpaAppSegments.id` | `integer` | Operational metadata | `standard` |
| `zpaAppSegments.name` | `string` | Tenant configuration | `standard` |
| `zpaAppSegments.externalId` | `string` | Operational metadata | `standard` |
| `zpaAppSegments.extensions` |  | Secret | never (unstructured) |
| `workloadGroups` | `array<object>` | Tenant configuration | `standard` |
| `workloadGroups.id` | `integer` | Operational metadata | `standard` |
| `workloadGroups.name` | `string` | Tenant configuration | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Firewall filtering rule identifier.
- `state`, `order`, `rank`, `enableFullLogging`, `defaultRule`, `predefined`, `excludeSrcCountries`: Rule state, logging, ordering, and default/predefined flags.
- `sourceCountries`, `destCountries`, `lastModifiedTime`: Non-principal rule metadata.
- `srcIps`, `destAddresses`, `destIpCategories`: Local-only IP and destination category criteria.
- `labels`, `timeWindows`: Nested references render reviewed `id`/`name` fields only.

## ZIA Forwarding Rules

Commands:

```sh
zscalerctl zia forwarding-rules list
zscalerctl zia forwarding-rules get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/forwarding-rules (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `type` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `state` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `order` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `rank` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardMethod` | `string` | Tenant configuration | `standard`, `share` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share` |
| `zpaBrokerRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `destCountries` | `array<string>` | Operational metadata | `standard`, `share` |
| `srcIps` | `array<string>` | Sensitive identifier | `standard` |
| `destAddresses` | `array<string>` | Sensitive identifier | `standard` |
| `destIpCategories` | `array<string>` | Sensitive identifier | `standard` |
| `resCategories` | `array<string>` | Sensitive identifier | `standard` |
| `labels` | `array<object>` | Tenant configuration | `standard`, `share` |
| `labels.id` | `integer` | Operational metadata | `standard`, `share` |
| `labels.name` | `string` | Tenant configuration | `standard`, `share` |
| `labels.extensions` |  | Secret | never (unstructured) |
| `locations` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locations.id` | `integer` | Operational metadata | `standard`, `share` |
| `locations.name` | `string` | Tenant configuration | `standard` |
| `locations.extensions` |  | Secret | never (unstructured) |
| `locationGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locationGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `locationGroups.name` | `string` | Tenant configuration | `standard` |
| `locationGroups.extensions` |  | Secret | never (unstructured) |
| `ecGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `ecGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `ecGroups.name` | `string` | Tenant configuration | `standard` |
| `ecGroups.extensions` |  | Secret | never (unstructured) |
| `departments` | `array<object>` | Tenant configuration | `standard`, `share` |
| `departments.id` | `integer` | Operational metadata | `standard`, `share` |
| `departments.name` | `string` | Tenant configuration | `standard` |
| `departments.extensions` |  | Secret | never (unstructured) |
| `groups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `groups.id` | `integer` | Operational metadata | `standard`, `share` |
| `groups.name` | `string` | Tenant configuration | `standard` |
| `groups.extensions` |  | Secret | never (unstructured) |
| `users` | `array<object>` | Tenant configuration | `standard`, `share` |
| `users.id` | `integer` | Operational metadata | `standard`, `share` |
| `users.name` | `string` | Tenant configuration | `standard` |
| `users.extensions` |  | Secret | never (unstructured) |
| `deviceGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `deviceGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `deviceGroups.name` | `string` | Tenant configuration | `standard` |
| `deviceGroups.extensions` |  | Secret | never (unstructured) |
| `lastModifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
| `srcIpGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `srcIpGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `srcIpGroups.name` | `string` | Tenant configuration | `standard` |
| `srcIpGroups.extensions` |  | Secret | never (unstructured) |
| `srcIpv6Groups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `srcIpv6Groups.id` | `integer` | Operational metadata | `standard`, `share` |
| `srcIpv6Groups.name` | `string` | Tenant configuration | `standard` |
| `srcIpv6Groups.extensions` |  | Secret | never (unstructured) |
| `destIpGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `destIpGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `destIpGroups.name` | `string` | Tenant configuration | `standard` |
| `destIpGroups.extensions` |  | Secret | never (unstructured) |
| `destIpv6Groups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `destIpv6Groups.id` | `integer` | Operational metadata | `standard`, `share` |
| `destIpv6Groups.name` | `string` | Tenant configuration | `standard` |
| `destIpv6Groups.extensions` |  | Secret | never (unstructured) |
| `nwServices` | `array<object>` | Tenant configuration | `standard`, `share` |
| `nwServices.id` | `integer` | Operational metadata | `standard`, `share` |
| `nwServices.name` | `string` | Tenant configuration | `standard` |
| `nwServices.extensions` |  | Secret | never (unstructured) |
| `nwServiceGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `nwServiceGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `nwServiceGroups.name` | `string` | Tenant configuration | `standard` |
| `nwServiceGroups.extensions` |  | Secret | never (unstructured) |
| `nwApplicationGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `nwApplicationGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `nwApplicationGroups.name` | `string` | Tenant configuration | `standard` |
| `nwApplicationGroups.extensions` |  | Secret | never (unstructured) |
| `appServiceGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `appServiceGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `appServiceGroups.name` | `string` | Tenant configuration | `standard` |
| `appServiceGroups.extensions` |  | Secret | never (unstructured) |
| `proxyGateway` | `object` | Tenant configuration | `standard` |
| `proxyGateway.id` | `integer` | Operational metadata | `standard` |
| `proxyGateway.name` | `string` | Tenant configuration | `standard` |
| `dedicatedIPGateway` | `object` | Tenant configuration | `standard` |
| `dedicatedIPGateway.id` | `integer` | Operational metadata | `standard` |
| `dedicatedIPGateway.name` | `string` | Tenant configuration | `standard` |
| `zpaGateway` | `object` | Tenant configuration | `standard` |
| `zpaGateway.id` | `integer` | Operational metadata | `standard` |
| `zpaGateway.name` | `string` | Tenant configuration | `standard` |
| `zpaAppSegments` | `array<object>` | Tenant configuration | `standard` |
| `zpaAppSegments.id` | `integer` | Operational metadata | `standard` |
| `zpaAppSegments.name` | `string` | Tenant configuration | `standard` |
| `zpaAppSegments.externalId` | `string` | Operational metadata | `standard` |
| `zpaAppSegments.extensions` |  | Secret | never (unstructured) |
| `zpaApplicationSegments` | `array<object>` | Tenant configuration | `standard` |
| `zpaApplicationSegments.id` | `integer` | Operational metadata | `standard` |
| `zpaApplicationSegments.name` | `string` | Tenant configuration | `standard` |
| `zpaApplicationSegments.description` | `string` | Free text | `standard` |
| `zpaApplicationSegments.zpaId` | `integer` | Sensitive identifier | `standard` |
| `zpaApplicationSegments.deleted` | `boolean` | Operational metadata | `standard` |
| `zpaApplicationSegmentGroups` | `array<object>` | Tenant configuration | `standard` |
| `zpaApplicationSegmentGroups.id` | `integer` | Operational metadata | `standard` |
| `zpaApplicationSegmentGroups.name` | `string` | Tenant configuration | `standard` |
| `zpaApplicationSegmentGroups.zpaId` | `integer` | Sensitive identifier | `standard` |
| `zpaApplicationSegmentGroups.deleted` | `boolean` | Operational metadata | `standard` |
| `zpaApplicationSegmentGroups.zpaAppSegmentsCount` | `integer` | Operational metadata | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Forwarding rule identifier.
- `forwardMethod`: Forwarding method selected for the rule.
- `srcIps`, `destAddresses`, `destIpCategories`, `resCategories`: Local-only source and destination criteria.
- `labels`: Nested references render reviewed `id`/`name` fields only.

## ZIA IPs Signature Rules

Commands:

```sh
zscalerctl zia ips-signature-rules list
zscalerctl zia ips-signature-rules get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/ips-signature-rules (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `category` | `object` | Tenant configuration | `standard` |
| `category.id` | `integer` | Operational metadata | `standard` |
| `category.name` | `string` | Tenant configuration | `standard` |
| `category.isNameL10nTag` | `boolean` | Operational metadata | `standard` |
| `enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `deleted` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `promoteTime` | `integer` | Operational metadata | `standard`, `share` |
| `ruleTextModTime` | `integer` | Operational metadata | `standard`, `share` |
| `dynamicValidationSubmitted` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `dynamicValidationRejected` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `dynamicValidationSucceeded` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `disabledFromZSCM` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `dynamicValRejectCode` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: IPS signature rule identifier.
- `enabled`, `deleted`, `dynamicValidationSubmitted`, `dynamicValidationRejected`, `dynamicValidationSucceeded`, `disabledFromZSCM`, `dynamicValRejectCode`: Rule state and dynamic-validation status flags.
- `promoteTime`, `ruleTextModTime`: Non-principal rule timestamps.
- `category`: Threat category reference: `id` and `isNameL10nTag` in every mode the parent renders, `name` in `standard`.

The SDK also returns the signature rule text. It embeds tenant network
indicators, so the reader never maps it and it is not in the catalog.

## ZIA IPs Policies

Commands:

```sh
zscalerctl zia ips-policies list
zscalerctl zia ips-policies get <id>
zscalerctl dump --products zia --resources zia/ips-policies --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/ips-policies (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `order` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `rank` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `accessControl` | `string` | Tenant configuration | `standard`, `share` |
| `enableFullLogging` | `boolean` | Tenant configuration | `standard`, `share` |
| `action` | `string` | Tenant configuration | `standard`, `share` |
| `state` | `string` | Tenant configuration | `standard`, `share` |
| `defaultRule` | `boolean` | Tenant configuration | `standard`, `share` |
| `capturePCAP` | `boolean` | Tenant configuration | `standard`, `share` |
| `predefined` | `boolean` | Tenant configuration | `standard`, `share` |
| `isEunEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `eunTemplateId` | `integer` | Tenant configuration | `standard`, `share` |
| `sourceCountries` | `array<string>` | Operational metadata | `standard`, `share` |
| `destCountries` | `array<string>` | Operational metadata | `standard`, `share` |
| `srcIps` | `array<string>` | Sensitive identifier | `standard` |
| `destAddresses` | `array<string>` | Sensitive identifier | `standard` |
| `destIpCategories` | `array<string>` | Sensitive identifier | `standard` |
| `resCategories` | `array<string>` | Sensitive identifier | `standard` |
| `lastModifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
| `labels` | `array<object>` | Tenant configuration | `standard`, `share` |
| `labels.id` | `integer` | Operational metadata | `standard`, `share` |
| `labels.name` | `string` | Tenant configuration | `standard`, `share` |
| `labels.extensions` |  | Secret | never (unstructured) |
| `timeWindows` | `array<object>` | Tenant configuration | `standard`, `share` |
| `timeWindows.id` | `integer` | Operational metadata | `standard`, `share` |
| `timeWindows.name` | `string` | Tenant configuration | `standard`, `share` |
| `timeWindows.extensions` |  | Secret | never (unstructured) |
| `locations` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locations.id` | `integer` | Operational metadata | `standard`, `share` |
| `locations.name` | `string` | Tenant configuration | `standard` |
| `locations.extensions` |  | Secret | never (unstructured) |
| `locationGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locationGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `locationGroups.name` | `string` | Tenant configuration | `standard` |
| `locationGroups.extensions` |  | Secret | never (unstructured) |
| `departments` | `array<object>` | Tenant configuration | `standard`, `share` |
| `departments.id` | `integer` | Operational metadata | `standard`, `share` |
| `departments.name` | `string` | Tenant configuration | `standard` |
| `departments.extensions` |  | Secret | never (unstructured) |
| `groups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `groups.id` | `integer` | Operational metadata | `standard`, `share` |
| `groups.name` | `string` | Tenant configuration | `standard` |
| `groups.extensions` |  | Secret | never (unstructured) |
| `users` | `array<object>` | Tenant configuration | `standard`, `share` |
| `users.id` | `integer` | Operational metadata | `standard`, `share` |
| `users.name` | `string` | Tenant configuration | `standard` |
| `users.extensions` |  | Secret | never (unstructured) |
| `destIpGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `destIpGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `destIpGroups.name` | `string` | Tenant configuration | `standard` |
| `destIpGroups.extensions` |  | Secret | never (unstructured) |
| `destIpv6Groups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `destIpv6Groups.id` | `integer` | Operational metadata | `standard`, `share` |
| `destIpv6Groups.name` | `string` | Tenant configuration | `standard` |
| `destIpv6Groups.extensions` |  | Secret | never (unstructured) |
| `nwServices` | `array<object>` | Tenant configuration | `standard`, `share` |
| `nwServices.id` | `integer` | Operational metadata | `standard`, `share` |
| `nwServices.name` | `string` | Tenant configuration | `standard` |
| `nwServices.extensions` |  | Secret | never (unstructured) |
| `nwServiceGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `nwServiceGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `nwServiceGroups.name` | `string` | Tenant configuration | `standard` |
| `nwServiceGroups.extensions` |  | Secret | never (unstructured) |
| `srcIpGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `srcIpGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `srcIpGroups.name` | `string` | Tenant configuration | `standard` |
| `srcIpGroups.extensions` |  | Secret | never (unstructured) |
| `srcIpv6Groups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `srcIpv6Groups.id` | `integer` | Operational metadata | `standard`, `share` |
| `srcIpv6Groups.name` | `string` | Tenant configuration | `standard` |
| `srcIpv6Groups.extensions` |  | Secret | never (unstructured) |
| `deviceGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `deviceGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `deviceGroups.name` | `string` | Tenant configuration | `standard` |
| `deviceGroups.extensions` |  | Secret | never (unstructured) |
| `devices` | `array<object>` | Tenant configuration | `standard`, `share` |
| `devices.id` | `integer` | Operational metadata | `standard`, `share` |
| `devices.name` | `string` | Tenant configuration | `standard` |
| `devices.extensions` |  | Secret | never (unstructured) |
| `threatCategories` | `array<object>` | Tenant configuration | `standard`, `share` |
| `threatCategories.id` | `integer` | Operational metadata | `standard`, `share` |
| `threatCategories.name` | `string` | Tenant configuration | `standard` |
| `threatCategories.extensions` |  | Secret | never (unstructured) |
| `zpaAppSegments` | `array<object>` | Tenant configuration | `standard` |
| `zpaAppSegments.id` | `integer` | Operational metadata | `standard` |
| `zpaAppSegments.name` | `string` | Tenant configuration | `standard` |
| `zpaAppSegments.externalId` | `string` | Operational metadata | `standard` |
| `zpaAppSegments.extensions` |  | Secret | never (unstructured) |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `order`, `rank`, `lastModifiedTime`: IPS policy identifiers, ordering, and timestamp metadata.
- `name`, `accessControl`, `enableFullLogging`, `action`, `state`, `defaultRule`, `capturePCAP`, `predefined`, `isEunEnabled`, `eunTemplateId`: Policy controls and enabled-state metadata.
- `sourceCountries`, `destCountries`: Country selector metadata.
- `srcIps`, `destAddresses`, `destIpCategories`, `resCategories`: Local-only IP and category selectors.
- `labels`, `timeWindows`: Nested references render reviewed `id`/`name` fields only.

## ZIA IP Source Groups

Commands:

```sh
zscalerctl zia ip-source-groups list
zscalerctl zia ip-source-groups get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/ip-source-groups (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `ipAddresses` | `array<string>` | Sensitive identifier | `standard` |
| `isNonEditable` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Source IP group identifier.
- `ipAddresses`: Local-only source IP addresses/ranges.
- `isNonEditable`: Whether the group is predefined/non-editable.

## ZIA IP Destination Groups

Commands:

```sh
zscalerctl zia ip-destination-groups list
zscalerctl zia ip-destination-groups get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/ip-destination-groups (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `type` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `addresses` | `array<string>` | Sensitive identifier | `standard` |
| `ipCategories` | `array<string>` | Sensitive identifier | `standard` |
| `countries` | `array<string>` | Operational metadata | `standard`, `share` |
| `isNonEditable` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Destination IP group identifier.
- `type`: Destination group type.
- `addresses`, `ipCategories`: Local-only destination addresses, FQDNs, wildcard FQDNs, and URL category references.
- `countries`: Destination country criteria.
- `isNonEditable`: Whether the group is predefined/non-editable.

## ZIA Network Services

Commands:

```sh
zscalerctl zia network-services list
zscalerctl zia network-services get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/network-services (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `tag` | `string` | Tenant configuration | `standard`, `share` |
| `type` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `protocol` | `string` | Tenant configuration | `standard`, `share` |
| `isNameL10nTag` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `srcTcpPorts` | `array<object>` | Tenant configuration | `standard`, `share` |
| `srcTcpPorts.start` | `integer` | Operational metadata | `standard`, `share` |
| `srcTcpPorts.end` | `integer` | Operational metadata | `standard`, `share` |
| `destTcpPorts` | `array<object>` | Tenant configuration | `standard`, `share` |
| `destTcpPorts.start` | `integer` | Operational metadata | `standard`, `share` |
| `destTcpPorts.end` | `integer` | Operational metadata | `standard`, `share` |
| `srcUdpPorts` | `array<object>` | Tenant configuration | `standard`, `share` |
| `srcUdpPorts.start` | `integer` | Operational metadata | `standard`, `share` |
| `srcUdpPorts.end` | `integer` | Operational metadata | `standard`, `share` |
| `destUdpPorts` | `array<object>` | Tenant configuration | `standard`, `share` |
| `destUdpPorts.start` | `integer` | Operational metadata | `standard`, `share` |
| `destUdpPorts.end` | `integer` | Operational metadata | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Network service identifier.
- `tag`, `protocol`: Service tag and protocol metadata.
- `type`, `isNameL10nTag`: Service type and localization flag.
- `srcTcpPorts`, `destTcpPorts`, `srcUdpPorts`, `destUdpPorts`: Port ranges render reviewed `start`/`end` values only.

## ZIA Network Service Groups

Commands:

```sh
zscalerctl zia network-service-groups list
zscalerctl zia network-service-groups get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/network-service-groups (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `services` | `array<object>` | Tenant configuration | `standard` |
| `services.id` | `integer` | Operational metadata | `standard` |
| `services.name` | `string` | Tenant configuration | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Network service group identifier.
- `services`: Local-only service references render reviewed `id`/`name` fields only. Query `zia/network-services` for service details.

## ZIA Network Applications

Commands:

```sh
zscalerctl zia network-applications list
zscalerctl zia network-applications get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/network-applications (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `parentCategory` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `deprecated` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `deprecated`: Network application identifier and deprecated flag.
- `parentCategory`: ZIA network application parent category.

## ZIA Application Services

Commands:

```sh
zscalerctl zia application-services list
zscalerctl zia application-services get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/application-services (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `nameL10nTag` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Application service identifier.
- `nameL10nTag`: Whether the name is a localization tag.

## ZIA Application Service Groups

Commands:

```sh
zscalerctl zia application-service-groups list
zscalerctl zia application-service-groups get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/application-service-groups (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `nameL10nTag` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Application service group identifier.
- `nameL10nTag`: Whether the name is a localization tag.

The ZIA SDK does not expose a direct get-by-id helper for application services
or application service groups, so `get` uses the list endpoint and returns the
matching identifier.

## ZIA Network Application Groups

Commands:

```sh
zscalerctl zia network-application-groups list
zscalerctl zia network-application-groups get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/network-application-groups (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `networkApplications` | `array<string>` | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Network application group identifier.
- `networkApplications`: Network application identifiers included in the group.

## ZIA Time Windows

Commands:

```sh
zscalerctl zia time-windows list
zscalerctl zia time-windows get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/time-windows (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `startTime` | `integer` | Tenant configuration | `standard`, `share` |
| `endTime` | `integer` | Tenant configuration | `standard`, `share` |
| `dayOfWeek` | `array<string>` | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Time-window identifier.
- `startTime`, `endTime`, `dayOfWeek`: Scheduled matching criteria.

The ZIA SDK does not expose a direct get-by-id helper for time windows, so
`get` uses the list endpoint and returns the matching identifier.

## ZIA Proxies

Commands:

```sh
zscalerctl zia proxies list
zscalerctl zia proxies get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/proxies (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `type` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `address` | `string` | Sensitive identifier | `standard` |
| `port` | `integer` | Operational metadata | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `insertXauHeader` | `boolean` | Tenant configuration | `standard`, `share` |
| `base64EncodeXauHeader` | `boolean` | Tenant configuration | `standard`, `share` |
| `cert` | `object` | Tenant configuration | `standard`, `share` |
| `cert.id` | `integer` | Operational metadata | `standard`, `share` |
| `cert.name` | `string` | Tenant configuration | `standard`, `share` |
| `cert.externalId` | `string` | Operational metadata | `standard`, `share` |
| `cert.extensions` |  | Secret | never (unstructured) |
| `lastModifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
| `lastModifiedBy.externalId` | `string` | Sensitive identifier | `standard` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Proxy identifier.
- `type`: Proxy type.
- `address`: Proxy address or FQDN.
- `port`: Proxy listener port.
- `insertXauHeader`, `base64EncodeXauHeader`: X-Authenticated-User header controls.
- `cert`: Certificate reference; `externalId` and `extensions` are dropped.

## ZIA Proxy Gateways

Commands:

```sh
zscalerctl zia proxy-gateways list
zscalerctl zia proxy-gateways get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/proxy-gateways (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `failClosed` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `type` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `primaryProxy` | `object` | Tenant configuration | `standard`, `share` |
| `primaryProxy.id` | `integer` | Operational metadata | `standard`, `share` |
| `primaryProxy.name` | `string` | Tenant configuration | `standard`, `share` |
| `primaryProxy.externalId` | `string` | Operational metadata | `standard`, `share` |
| `primaryProxy.extensions` |  | Secret | never (unstructured) |
| `secondaryProxy` | `object` | Tenant configuration | `standard`, `share` |
| `secondaryProxy.id` | `integer` | Operational metadata | `standard`, `share` |
| `secondaryProxy.name` | `string` | Tenant configuration | `standard`, `share` |
| `secondaryProxy.externalId` | `string` | Operational metadata | `standard`, `share` |
| `secondaryProxy.extensions` |  | Secret | never (unstructured) |
| `lastModifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Proxy gateway identifier.
- `failClosed`, `type`: Gateway behavior and type.
- `primaryProxy`, `secondaryProxy`: Proxy references; `externalId` and `extensions` are dropped.

The ZIA SDK does not expose a direct get-by-id helper for proxy gateways, so
`get` uses the list endpoint and returns the matching identifier.

## ZIA Dedicated IP Gateways

Commands:

```sh
zscalerctl zia dedicated-ip-gateways list
zscalerctl zia dedicated-ip-gateways get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/dedicated-ip-gateways (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `primaryDataCenter` | `object` | Tenant configuration | `standard`, `share` |
| `primaryDataCenter.id` | `integer` | Operational metadata | `standard`, `share` |
| `primaryDataCenter.name` | `string` | Tenant configuration | `standard`, `share` |
| `primaryDataCenter.extensions` |  | Secret | never (unstructured) |
| `secondaryDataCenter` | `object` | Tenant configuration | `standard`, `share` |
| `secondaryDataCenter.id` | `integer` | Operational metadata | `standard`, `share` |
| `secondaryDataCenter.name` | `string` | Tenant configuration | `standard`, `share` |
| `secondaryDataCenter.extensions` |  | Secret | never (unstructured) |
| `createTime` | `integer` | Operational metadata | `standard`, `share` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share` |
| `lastModifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
| `default` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Dedicated IP gateway identifier.
- `primaryDataCenter`, `secondaryDataCenter`: Data-center references; `extensions` is dropped.
- `default`: Whether this is the default gateway.

Dedicated IP gateways use the SDK lite endpoint and list-derived `get`.

## ZIA Time Intervals

Commands:

```sh
zscalerctl zia time-intervals list
zscalerctl zia time-intervals get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/time-intervals (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `startTime` | `integer` | Tenant configuration | `standard`, `share` |
| `endTime` | `integer` | Tenant configuration | `standard`, `share` |
| `daysOfWeek` | `array<string>` | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Time interval identifier.
- `startTime`, `endTime`, `daysOfWeek`: Scheduled matching criteria.

## ZIA Bandwidth Classes

Commands:

```sh
zscalerctl zia bandwidth-classes list
zscalerctl zia bandwidth-classes get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/bandwidth-classes (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `isNameL10nTag` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `getfileSize` | `string` | Tenant configuration | `standard`, `share` |
| `fileSize` | `string` | Tenant configuration | `standard`, `share` |
| `type` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `webApplications` | `array<string>` | Tenant configuration | `standard`, `share` |
| `urls` | `array<string>` | Sensitive identifier | `standard` |
| `applicationServiceGroups` | `array<string>` | Tenant configuration | `standard`, `share` |
| `networkApplications` | `array<string>` | Tenant configuration | `standard`, `share` |
| `networkServices` | `array<string>` | Tenant configuration | `standard`, `share` |
| `urlCategories` | `array<string>` | Tenant configuration | `standard`, `share` |
| `applications` | `array<string>` | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Bandwidth class identifier.
- `isNameL10nTag`, `type`: Localization flag and class type.
- `name`, `getfileSize`, `fileSize`, `webApplications`, `applicationServiceGroups`, `networkApplications`, `networkServices`, `urlCategories`, `applications`: Reviewed class matching and sizing metadata.
- `urls`: Local-only URL criteria; values are scanned before output.

## ZIA Bandwidth Control Rules

Commands:

```sh
zscalerctl zia bandwidth-control-rules list
zscalerctl zia bandwidth-control-rules get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/bandwidth-control-rules (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `order` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `state` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `rank` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `maxBandwidth` | `integer` | Tenant configuration | `standard`, `share` |
| `minBandwidth` | `integer` | Tenant configuration | `standard`, `share` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share` |
| `accessControl` | `string` | Operational metadata | `standard`, `share` |
| `defaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `protocols` | `array<string>` | Tenant configuration | `standard`, `share` |
| `deviceTrustLevels` | `array<string>` | Tenant configuration | `standard`, `share` |
| `lastModifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
| `bandwidthClasses` | `array<object>` | Tenant configuration | `standard`, `share` |
| `bandwidthClasses.id` | `integer` | Operational metadata | `standard`, `share` |
| `bandwidthClasses.name` | `string` | Tenant configuration | `standard`, `share` |
| `bandwidthClasses.extensions` |  | Secret | never (unstructured) |
| `labels` | `array<object>` | Tenant configuration | `standard`, `share` |
| `labels.id` | `integer` | Operational metadata | `standard`, `share` |
| `labels.name` | `string` | Tenant configuration | `standard`, `share` |
| `labels.extensions` |  | Secret | never (unstructured) |
| `timeWindows` | `array<object>` | Tenant configuration | `standard`, `share` |
| `timeWindows.id` | `integer` | Operational metadata | `standard`, `share` |
| `timeWindows.name` | `string` | Tenant configuration | `standard`, `share` |
| `timeWindows.extensions` |  | Secret | never (unstructured) |
| `locations` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locations.id` | `integer` | Operational metadata | `standard`, `share` |
| `locations.name` | `string` | Tenant configuration | `standard` |
| `locations.extensions` |  | Secret | never (unstructured) |
| `locationGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locationGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `locationGroups.name` | `string` | Tenant configuration | `standard` |
| `locationGroups.extensions` |  | Secret | never (unstructured) |
| `devices` | `array<object>` | Tenant configuration | `standard`, `share` |
| `devices.id` | `integer` | Operational metadata | `standard`, `share` |
| `devices.name` | `string` | Tenant configuration | `standard` |
| `devices.extensions` |  | Secret | never (unstructured) |
| `deviceGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `deviceGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `deviceGroups.name` | `string` | Tenant configuration | `standard` |
| `deviceGroups.extensions` |  | Secret | never (unstructured) |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Bandwidth rule identifier.
- `order`, `state`, `rank`, `defaultRule`: Rule ordering, state, and default metadata.
- `lastModifiedTime`, `accessControl`: SDK timestamp and access metadata.
- `maxBandwidth`, `minBandwidth`, `protocols`, `deviceTrustLevels`, `bandwidthClasses`, `labels`, `timeWindows`: Reviewed bandwidth and rule criteria.

## ZIA Dns Gateways

Commands:

```sh
zscalerctl zia dns-gateways list
zscalerctl zia dns-gateways get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/dns-gateways (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `dnsGatewayType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `primaryIpOrFqdn` | `string` | Sensitive identifier | `standard` |
| `primaryPorts` | `array<integer>` | Tenant configuration | `standard`, `share` |
| `secondaryIpOrFqdn` | `string` | Sensitive identifier | `standard` |
| `secondaryPorts` | `array<integer>` | Tenant configuration | `standard`, `share` |
| `protocols` | `array<string>` | Tenant configuration | `standard`, `share` |
| `failureBehavior` | `string` | Tenant configuration | `standard`, `share` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share` |
| `lastModifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
| `autoCreated` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `natZtrGateway` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `dnsGatewayProtocols` | `array<string>` | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: DNS gateway identifier.
- `dnsGatewayType`, `autoCreated`, `natZtrGateway`: Gateway type and flags.
- `primaryIpOrFqdn`, `secondaryIpOrFqdn`: Local-only gateway endpoints.
- `primaryPorts`, `secondaryPorts`, `protocols`, `failureBehavior`, `dnsGatewayProtocols`: Reviewed DNS gateway connectivity settings.

## ZIA Nat Control Rules

Commands:

```sh
zscalerctl zia nat-control-rules list
zscalerctl zia nat-control-rules get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/nat-control-rules (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `order` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `rank` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `state` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `accessControl` | `string` | Operational metadata | `standard`, `share` |
| `redirectFqdn` | `string` | Sensitive identifier | `standard` |
| `redirectIp` | `string` | Sensitive identifier | `standard` |
| `redirectPort` | `integer` | Tenant configuration | `standard`, `share` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share` |
| `lastModifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
| `trustedResolverRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableFullLogging` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `predefined` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `defaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `destAddresses` | `array<string>` | Sensitive identifier | `standard` |
| `srcIps` | `array<string>` | Sensitive identifier | `standard` |
| `destCountries` | `array<string>` | Operational metadata | `standard`, `share` |
| `destIpCategories` | `array<string>` | Sensitive identifier | `standard` |
| `resCategories` | `array<string>` | Sensitive identifier | `standard` |
| `labels` | `array<object>` | Tenant configuration | `standard`, `share` |
| `labels.id` | `integer` | Operational metadata | `standard`, `share` |
| `labels.name` | `string` | Tenant configuration | `standard`, `share` |
| `labels.extensions` |  | Secret | never (unstructured) |
| `timeWindows` | `array<object>` | Tenant configuration | `standard`, `share` |
| `timeWindows.id` | `integer` | Operational metadata | `standard`, `share` |
| `timeWindows.name` | `string` | Tenant configuration | `standard`, `share` |
| `timeWindows.extensions` |  | Secret | never (unstructured) |
| `locations` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locations.id` | `integer` | Operational metadata | `standard`, `share` |
| `locations.name` | `string` | Tenant configuration | `standard` |
| `locations.extensions` |  | Secret | never (unstructured) |
| `locationGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locationGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `locationGroups.name` | `string` | Tenant configuration | `standard` |
| `locationGroups.extensions` |  | Secret | never (unstructured) |
| `srcIpGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `srcIpGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `srcIpGroups.name` | `string` | Tenant configuration | `standard` |
| `srcIpGroups.extensions` |  | Secret | never (unstructured) |
| `srcIpv6Groups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `srcIpv6Groups.id` | `integer` | Operational metadata | `standard`, `share` |
| `srcIpv6Groups.name` | `string` | Tenant configuration | `standard` |
| `srcIpv6Groups.extensions` |  | Secret | never (unstructured) |
| `destIpGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `destIpGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `destIpGroups.name` | `string` | Tenant configuration | `standard` |
| `destIpGroups.extensions` |  | Secret | never (unstructured) |
| `destIpv6Groups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `destIpv6Groups.id` | `integer` | Operational metadata | `standard`, `share` |
| `destIpv6Groups.name` | `string` | Tenant configuration | `standard` |
| `destIpv6Groups.extensions` |  | Secret | never (unstructured) |
| `nwServices` | `array<object>` | Tenant configuration | `standard`, `share` |
| `nwServices.id` | `integer` | Operational metadata | `standard`, `share` |
| `nwServices.name` | `string` | Tenant configuration | `standard` |
| `nwServices.extensions` |  | Secret | never (unstructured) |
| `nwServiceGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `nwServiceGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `nwServiceGroups.name` | `string` | Tenant configuration | `standard` |
| `nwServiceGroups.extensions` |  | Secret | never (unstructured) |
| `groups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `groups.id` | `integer` | Operational metadata | `standard`, `share` |
| `groups.name` | `string` | Tenant configuration | `standard` |
| `groups.extensions` |  | Secret | never (unstructured) |
| `departments` | `array<object>` | Tenant configuration | `standard`, `share` |
| `departments.id` | `integer` | Operational metadata | `standard`, `share` |
| `departments.name` | `string` | Tenant configuration | `standard` |
| `departments.extensions` |  | Secret | never (unstructured) |
| `users` | `array<object>` | Tenant configuration | `standard`, `share` |
| `users.id` | `integer` | Operational metadata | `standard`, `share` |
| `users.name` | `string` | Tenant configuration | `standard` |
| `users.extensions` |  | Secret | never (unstructured) |
| `devices` | `array<object>` | Tenant configuration | `standard`, `share` |
| `devices.id` | `integer` | Operational metadata | `standard`, `share` |
| `devices.name` | `string` | Tenant configuration | `standard` |
| `devices.extensions` |  | Secret | never (unstructured) |
| `deviceGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `deviceGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `deviceGroups.name` | `string` | Tenant configuration | `standard` |
| `deviceGroups.extensions` |  | Secret | never (unstructured) |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: NAT control rule identifier.
- `order`, `rank`, `state`, `trustedResolverRule`, `enableFullLogging`, `predefined`, `defaultRule`: Rule state, order, logging, and default/predefined metadata.
- `redirectFqdn`, `redirectIp`, `destAddresses`, `srcIps`, `destIpCategories`, `resCategories`: Local-only NAT endpoints and criteria.

## ZIA Groups

Commands:

```sh
zscalerctl zia groups list
zscalerctl zia groups get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/groups (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `idpId` | `integer` | Tenant configuration | `standard`, `share` |
| `comments` | `string` | Free text | `standard` |
| `isSystemDefined` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Group identifier.
- `idpId`: Identity provider identifier.
- `isSystemDefined`: Whether the group is system-defined.

## ZIA Departments

Commands:

```sh
zscalerctl zia departments list
zscalerctl zia departments get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/departments (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `idpId` | `integer` | Tenant configuration | `standard`, `share` |
| `comments` | `string` | Free text | `standard` |
| `deleted` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `deleted`: Department identifier and deleted flag.
- `name`, `idpId`: Identity-provider department metadata.

## ZIA Users

Commands:

```sh
zscalerctl zia users list
zscalerctl zia users get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/users (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Sensitive identifier | `standard` |
| `email` | `string` | Sensitive identifier | `standard` |
| `groups` | `array<object>` | Tenant configuration | `standard` |
| `groups.id` | `integer` | Operational metadata | `standard` |
| `groups.name` | `string` | Tenant configuration | `standard` |
| `groups.comments` | `string` | Free text | `standard` |
| `department` | `object` | Tenant configuration | `standard` |
| `department.id` | `integer` | Operational metadata | `standard` |
| `department.name` | `string` | Tenant configuration | `standard` |
| `department.comments` | `string` | Free text | `standard` |
| `comments` | `string` | Free text | `standard` |
| `tempAuthEmail` | `string` | Sensitive identifier | `standard` |
| `authMethods` | `array<string>` | Tenant configuration | `standard` |
| `password` |  | Secret | never (credential) |
| `adminUser` | `boolean` | Operational metadata | `standard`, `share` |
| `type` | `string` | Tenant configuration | `standard`, `share` |
| `deleted` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `deleted`: User identifier and deleted flag.
- `adminUser`: Administrator flag.
- `type`: User type/classification returned by ZIA.
- `name`, `email`, `tempAuthEmail`: Employee identity/contact fields; visible to the local administrator, dropped from `share` and `paranoid`.
- `groups`, `department`: Rendered as id/name references only. Query `zia/groups` or `zia/departments` for authoritative metadata.
- `authMethods`: Authentication methods are local-only because they describe employee auth posture.

## ZIA Device Groups

Commands:

```sh
zscalerctl zia device-groups list
zscalerctl zia device-groups get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/device-groups (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `groupType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `description` | `string` | Free text | `standard` |
| `osType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `predefined` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `deviceNames` | `string` | Sensitive identifier | `standard` |
| `deviceCount` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Device group identifier.
- `groupType`, `osType`, `predefined`, `deviceCount`: Group type, OS, predefined flag, and member count.
- `deviceNames`: Local-only device-name list when returned by the SDK.

Device groups use a list-derived `get`.

## ZIA Devices

Commands:

```sh
zscalerctl zia devices list
zscalerctl zia devices get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/devices (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Sensitive identifier | `standard` |
| `deviceGroupType` | `string` | Tenant configuration | `standard`, `share` |
| `deviceModel` | `string` | Sensitive identifier | `standard` |
| `osType` | `string` | Tenant configuration | `standard`, `share` |
| `osVersion` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `ownerUserId` | `integer` | Sensitive identifier | `standard` |
| `ownerName` | `string` | Sensitive identifier | `standard` |
| `hostName` | `string` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Device identifier.
- `deviceGroupType`, `osType`, `osVersion`: Device classification and OS metadata.
- `name`, `deviceModel`, `ownerUserId`, `ownerName`, `hostName`: Employee/device locating fields; visible to the local administrator, dropped from `share` and `paranoid`.

## ZIA Workload Groups

Commands:

```sh
zscalerctl zia workload-groups list
zscalerctl zia workload-groups get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/workload-groups (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `expression` | `string` | Sensitive identifier | `standard` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share` |
| `lastModifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Workload group identifier.
- `expression`: Local-only workload tag expression.

## ZIA Alert Subscriptions

Commands:

```sh
zscalerctl zia alert-subscriptions list
zscalerctl zia alert-subscriptions get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/alert-subscriptions (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `description` | `string` | Free text | `standard` |
| `email` | `string` | Sensitive identifier | `standard` |
| `deleted` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `pt0Severities` | `array<string>` | Tenant configuration | `standard`, `share` |
| `secureSeverities` | `array<string>` | Tenant configuration | `standard`, `share` |
| `manageSeverities` | `array<string>` | Tenant configuration | `standard`, `share` |
| `complySeverities` | `array<string>` | Tenant configuration | `standard`, `share` |
| `systemSeverities` | `array<string>` | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `deleted`: Subscription identifier and deletion state.
- `email`: Local-only notification recipient address.
- `pt0Severities`, `secureSeverities`, `manageSeverities`, `complySeverities`, `systemSeverities`: Reviewed alert severity selections.

## ZIA Activation Status

Commands:

```sh
zscalerctl zia activation-status show
zscalerctl dump --products zia --resources zia/activation-status --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/activation-status (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `status` | `string` | Operational metadata | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `status`: Activation status metadata.

## ZIA Eusa Status

Commands:

```sh
zscalerctl zia eusa-status show
zscalerctl dump --products zia --resources zia/eusa-status --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/eusa-status (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `version` | `object` | Tenant configuration | `standard`, `share` |
| `version.id` | `integer` | Operational metadata | `standard`, `share` |
| `version.name` | `string` | Tenant configuration | `standard`, `share` |
| `version.extensions` |  | Secret | never (unstructured) |
| `acceptedStatus` | `boolean` | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: EUSA status identifier.
- `version`: Rendered as an id/name reference; SDK extension data is dropped.
- `acceptedStatus`: Current acceptance status.

## ZIA Auth Exempted Urls

Commands:

```sh
zscalerctl zia auth-exempted-urls show
zscalerctl dump --products zia --resources zia/auth-exempted-urls --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/auth-exempted-urls (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `urls` | `array<string>` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `urls`: Authentication-exempt URL list; dropped from `share` and `paranoid`.

## ZIA Intermediate Ca Certificates

Commands:

```sh
zscalerctl zia intermediate-ca-certificates list
zscalerctl zia intermediate-ca-certificates get <id>
zscalerctl dump --products zia --resources zia/intermediate-ca-certificates --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/intermediate-ca-certificates (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `type` | `string` | Tenant configuration | `standard`, `share` |
| `region` | `string` | Sensitive identifier | `standard` |
| `status` | `string` | Tenant configuration | `standard`, `share` |
| `defaultCertificate` | `boolean` | Tenant configuration | `standard`, `share` |
| `certStartDate` | `integer` | Operational metadata | `standard`, `share` |
| `certExpDate` | `integer` | Operational metadata | `standard`, `share` |
| `currentState` | `string` | Tenant configuration | `standard`, `share` |
| `publicKey` | `string` | Operational metadata | `standard` |
| `keyGenerationTime` | `integer` | Operational metadata | `standard`, `share` |
| `hsmAttestationVerifiedTime` | `integer` | Operational metadata | `standard`, `share` |
| `csrFileName` | `string` | Sensitive identifier | `standard` |
| `csrGenerationTime` | `integer` | Operational metadata | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Certificate metadata identifier.
- `name`, `type`, `status`, `defaultCertificate`, `currentState`: Certificate metadata and workflow state.
- `region`, `csrFileName`: Location and file identifiers are visible locally and dropped from `share` and `paranoid`.
- `certStartDate`, `certExpDate`, `keyGenerationTime`, `hsmAttestationVerifiedTime`, `csrGenerationTime`: Certificate lifecycle timestamps.

## ZIA Cloud App Instances

Commands:

```sh
zscalerctl zia cloud-app-instances list
zscalerctl zia cloud-app-instances get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/cloud-app-instances (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `instanceId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `instanceType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `instanceName` | `string` | Tenant configuration | `standard`, `share` |
| `modifiedAt` | `integer` | Operational metadata | `standard`, `share` |
| `modifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy.name` | `string` | Sensitive identifier | `standard` |
| `instanceIdentifiers` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `instanceIdentifiers.instanceId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `instanceIdentifiers.instanceIdentifier` | `string` | Sensitive identifier | `standard` |
| `instanceIdentifiers.instanceIdentifierName` | `string` | Sensitive identifier | `standard` |
| `instanceIdentifiers.identifierType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `instanceIdentifiers.modifiedAt` | `integer` | Operational metadata | `standard`, `share` |
| `instanceIdentifiers.modifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `instanceIdentifiers.modifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `instanceIdentifiers.modifiedBy.name` | `string` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

## ZIA Tenancy Restriction Profiles

Commands:

```sh
zscalerctl zia tenancy-restriction-profiles list
zscalerctl zia tenancy-restriction-profiles get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/tenancy-restriction-profiles (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `appType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `description` | `string` | Free text | `standard` |
| `itemTypePrimary` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `itemTypeSecondary` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `restrictPersonalO365Domains` | `boolean` | Tenant configuration | `standard`, `share` |
| `allowGoogleConsumers` | `boolean` | Tenant configuration | `standard`, `share` |
| `msLoginServicesTrV2` | `boolean` | Tenant configuration | `standard`, `share` |
| `allowGoogleVisitors` | `boolean` | Tenant configuration | `standard`, `share` |
| `allowGcpCloudStorageRead` | `boolean` | Tenant configuration | `standard`, `share` |
| `itemDataPrimary` | `array<string>` | Sensitive identifier | `standard` |
| `itemDataSecondary` | `array<string>` | Sensitive identifier | `standard` |
| `itemValue` | `array<string>` | Sensitive identifier | `standard` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share` |
| `lastModifiedUserId` | `integer` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `restrictPersonalO365Domains`, `allowGoogleConsumers`, `msLoginServicesTrV2`, `allowGoogleVisitors`, `allowGcpCloudStorageRead`: Reviewed tenancy restriction flags.
- `itemDataPrimary`, `itemDataSecondary`, `itemValue`, `lastModifiedUserId`: Local-only tenant data values and admin identifier.

## ZIA Vzen Clusters

Commands:

```sh
zscalerctl zia vzen-clusters list
zscalerctl zia vzen-clusters get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/vzen-clusters (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `status` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `ipAddress` | `string` | Sensitive identifier | `standard` |
| `subnetMask` | `string` | Sensitive identifier | `standard` |
| `defaultGateway` | `string` | Sensitive identifier | `standard` |
| `type` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `ipSecEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `virtualZenNodes` | `array<object>` | Tenant configuration | `standard` |
| `virtualZenNodes.id` | `integer` | Operational metadata | `standard` |
| `virtualZenNodes.name` | `string` | Tenant configuration | `standard` |
| `virtualZenNodes.externalId` | `string` | Operational metadata | `standard` |
| `virtualZenNodes.extensions` |  | Secret | never (unstructured) |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `status`, `type`, `ipSecEnabled`: Cluster identifier, state, type, and IPSec flag.
- `ipAddress`, `subnetMask`, `defaultGateway`: Local-only network addressing.
- `virtualZenNodes`: Local-only node references render reviewed `id`/`name` fields only; external IDs and extensions drop.

## ZIA Vzen Nodes

Commands:

```sh
zscalerctl zia vzen-nodes list
zscalerctl zia vzen-nodes get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/vzen-nodes (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `zgatewayId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `status` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `inProduction` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `ipAddress` | `string` | Sensitive identifier | `standard` |
| `subnetMask` | `string` | Sensitive identifier | `standard` |
| `defaultGateway` | `string` | Sensitive identifier | `standard` |
| `type` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `ipSecEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `onDemandSupportTunnelEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `establishSupportTunnelEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `loadBalancerIpAddress` | `string` | Sensitive identifier | `standard` |
| `deploymentMode` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `clusterName` | `string` | Tenant configuration | `standard`, `share` |
| `vzenSkuType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `zgatewayId`, `status`, `inProduction`, `type`, `ipSecEnabled`, `onDemandSupportTunnelEnabled`, `establishSupportTunnelEnabled`, `deploymentMode`, `vzenSkuType`: Node identity, status, deployment, support-tunnel, and SKU metadata.
- `ipAddress`, `subnetMask`, `defaultGateway`, `loadBalancerIpAddress`: Local-only network addressing.

## ZIA Dlp Engines

Commands:

```sh
zscalerctl zia dlp-engines list
zscalerctl zia dlp-engines get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/dlp-engines (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `predefinedEngineName` | `string` | Tenant configuration | `standard`, `share` |
| `engineExpression` | `string` | Sensitive identifier | `standard` |
| `customDlpEngine` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `customDlpEngine`: Engine identity and custom/predefined flag.
- `name`, `predefinedEngineName`: Engine names; scanned for pasted secret-shaped values.
- `engineExpression`: Local-only dictionary expression; dropped from `share` and `paranoid`.

## ZIA Dlp Dictionaries

Commands:

```sh
zscalerctl zia dlp-dictionaries list
zscalerctl zia dlp-dictionaries get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/dlp-dictionaries (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `confidenceThreshold` | `string` | Tenant configuration | `standard`, `share` |
| `customPhraseMatchType` | `string` | Tenant configuration | `standard`, `share` |
| `nameL10nTag` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `custom` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `thresholdType` | `string` | Tenant configuration | `standard`, `share` |
| `dictionaryType` | `string` | Tenant configuration | `standard`, `share` |
| `proximity` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `phrases` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `phrases.action` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `phrases.phrase` | `string` | Free text | `standard` |
| `patterns` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `patterns.action` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `patterns.pattern` | `string` | Free text | `standard` |
| `exactDataMatchDetails` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `exactDataMatchDetails.dictionaryEdmMappingId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `exactDataMatchDetails.schemaId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `exactDataMatchDetails.primaryFields` | `array<integer>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `exactDataMatchDetails.secondaryFields` | `array<integer>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `exactDataMatchDetails.secondaryFieldMatchOn` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `idmProfileMatchAccuracyDetails` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `idmProfileMatchAccuracyDetails.adpIdmProfile` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `idmProfileMatchAccuracyDetails.adpIdmProfile.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `idmProfileMatchAccuracyDetails.adpIdmProfile.name` | `string` | Sensitive identifier | `standard` |
| `idmProfileMatchAccuracyDetails.adpIdmProfile.extensions` |  | Secret | never (unstructured) |
| `idmProfileMatchAccuracyDetails.matchAccuracy` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `ignoreExactMatchIdmDict` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `includeBinNumbers` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `binNumbers` | `array<integer>` | Free text | `standard` |
| `dictTemplateId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `predefinedClone` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `predefinedCountActionType` | `string` | Tenant configuration | `standard`, `share` |
| `proximityLengthEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `proximityEnabledForCustomDictionary` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `dictionaryCloningEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `customPhraseSupported` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `hierarchicalDictionary` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `hierarchicalIdentifiers` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `predefinedPhrases` | `array<string>` | Free text | `standard` |
| `thresholdAllowed` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `confidenceLevelForPredefinedDict` | `string` | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `nameL10nTag`, `custom`, `proximity`, `ignoreExactMatchIdmDict`, `includeBinNumbers`, `dictTemplateId`, `predefinedClone`, `proximityLengthEnabled`, `proximityEnabledForCustomDictionary`, `dictionaryCloningEnabled`, `customPhraseSupported`, `hierarchicalDictionary`, `thresholdAllowed`: Dictionary identity, broad type/status flags, and numeric thresholds.
- `name`, `confidenceThreshold`, `customPhraseMatchType`, `thresholdType`, `dictionaryType`, `predefinedCountActionType`, `confidenceLevelForPredefinedDict`: Detection-mode metadata; scanned for pasted secret-shaped values.

## ZIA Dlp Edm Schemas

Commands:

```sh
zscalerctl zia dlp-edm-schemas list
zscalerctl zia dlp-edm-schemas get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/dlp-edm-schemas (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `schemaId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `edmClient` | `object` | Tenant configuration | `standard`, `share` |
| `edmClient.id` | `integer` | Operational metadata | `standard`, `share` |
| `edmClient.name` | `string` | Tenant configuration | `standard` |
| `edmClient.extensions` |  | Secret | never (unstructured) |
| `projectName` | `string` | Sensitive identifier | `standard` |
| `revision` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `filename` | `string` | Sensitive identifier | `standard` |
| `originalFileName` | `string` | Sensitive identifier | `standard` |
| `fileUploadStatus` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `schemaStatus` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `origColCount` | `integer` | Operational metadata | `standard` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy.name` | `string` | Sensitive identifier | `standard` |
| `createdBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `createdBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `createdBy.name` | `string` | Sensitive identifier | `standard` |
| `cellsUsed` | `integer` | Operational metadata | `standard` |
| `schemaActive` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `schedulePresent` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `tokenList` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `tokenList.name` | `string` | Sensitive identifier | `standard` |
| `tokenList.type` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `tokenList.primaryKey` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `tokenList.originalColumn` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `tokenList.hashfileColumnOrder` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `tokenList.colLengthBitmap` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `schedule` | `object` | Tenant configuration | `standard` |
| `schedule.scheduleType` | `string` | Tenant configuration | `standard` |
| `schedule.scheduleDayOfMonth` | `array<string>` | Tenant configuration | `standard` |
| `schedule.scheduleDayOfWeek` | `array<string>` | Tenant configuration | `standard` |
| `schedule.scheduleTime` | `integer` | Operational metadata | `standard` |
| `schedule.scheduleDisabled` | `boolean` | Operational metadata | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `schemaId`, `revision`, `fileUploadStatus`, `schemaStatus`, `lastModifiedTime`, `schemaActive`, `schedulePresent`: Schema identity, revision, status, and broad scheduling/activity flags.
- `origColCount`, `cellsUsed`: Local-only EDM scale metadata.
- `projectName`, `filename`, `originalFileName`: Local-only schema and upload identifiers.

## ZIA Dlp Idm Profile Lite

Commands:

```sh
zscalerctl zia dlp-idm-profile-lite list
zscalerctl zia dlp-idm-profile-lite get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/dlp-idm-profile-lite (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `profileId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `templateName` | `string` | Sensitive identifier | `standard` |
| `clientVm` | `object` | Tenant configuration | `standard`, `share` |
| `clientVm.id` | `integer` | Operational metadata | `standard`, `share` |
| `clientVm.name` | `string` | Tenant configuration | `standard` |
| `clientVm.extensions` |  | Secret | never (unstructured) |
| `numDocuments` | `integer` | Operational metadata | `standard` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy.name` | `string` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `profileId`, `lastModifiedTime`: IDM profile identifier and timestamp.
- `numDocuments`: Local-only indexed-document volume.
- `templateName`: Local-only IDM template name.

## ZIA Dlp Idm Profiles

Commands:

```sh
zscalerctl zia dlp-idm-profiles list
zscalerctl zia dlp-idm-profiles get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/dlp-idm-profiles (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `profileId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `profileName` | `string` | Sensitive identifier | `standard` |
| `profileDesc` | `string` | Free text | `standard` |
| `profileType` | `string` | Tenant configuration | `standard`, `share` |
| `host` | `string` | Sensitive identifier | `standard` |
| `port` | `integer` | Sensitive identifier | `standard` |
| `profileDirPath` | `string` | Sensitive identifier | `standard` |
| `scheduleType` | `string` | Tenant configuration | `standard` |
| `scheduleDay` | `integer` | Operational metadata | `standard` |
| `scheduleDayOfMonth` | `array<string>` | Tenant configuration | `standard` |
| `scheduleDayOfWeek` | `array<string>` | Tenant configuration | `standard` |
| `scheduleTime` | `integer` | Operational metadata | `standard` |
| `scheduleDisabled` | `boolean` | Operational metadata | `standard` |
| `uploadStatus` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `userName` | `string` | Sensitive identifier | `standard` |
| `version` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `idmClient` | `object` | Tenant configuration | `standard`, `share` |
| `idmClient.id` | `integer` | Operational metadata | `standard`, `share` |
| `idmClient.name` | `string` | Tenant configuration | `standard` |
| `idmClient.extensions` |  | Secret | never (unstructured) |
| `volumeOfDocuments` | `integer` | Operational metadata | `standard` |
| `numDocuments` | `integer` | Operational metadata | `standard` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy.name` | `string` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `profileId`, `version`, `lastModifiedTime`, `uploadStatus`: IDM profile identity, version, upload status, and timestamp.
- `scheduleDay`, `scheduleTime`, `scheduleDisabled`, `volumeOfDocuments`, `numDocuments`: Local-only schedule and indexed-document volume.
- `profileType`: IDM profile type.
- `profileName`, `host`, `port`, `profileDirPath`, `userName`: Local-only host, path, and account identifiers.

## ZIA Dlp Web Rules

Commands:

```sh
zscalerctl zia dlp-web-rules list
zscalerctl zia dlp-web-rules get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/dlp-web-rules (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `order` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `accessControl` | `string` | Operational metadata | `standard`, `share` |
| `protocols` | `array<string>` | Tenant configuration | `standard`, `share` |
| `rank` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `fileTypes` | `array<string>` | Tenant configuration | `standard`, `share` |
| `cloudApplications` | `array<string>` | Tenant configuration | `standard`, `share` |
| `minSize` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `action` | `string` | Tenant configuration | `standard`, `share` |
| `state` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `matchOnly` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `withoutContentInspection` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `ocrEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `dlpDownloadScanEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `zccNotificationsEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `zscalerIncidentReceiver` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `eunTemplateId` | `integer` | Sensitive identifier | `standard` |
| `externalAuditorEmail` | `string` | Sensitive identifier | `standard` |
| `auditor` | `object` | Tenant configuration | `standard` |
| `auditor.id` | `integer` | Operational metadata | `standard` |
| `auditor.name` | `string` | Tenant configuration | `standard` |
| `lastModifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
| `notificationTemplate` | `object` | Tenant configuration | `standard` |
| `notificationTemplate.id` | `integer` | Operational metadata | `standard` |
| `notificationTemplate.name` | `string` | Tenant configuration | `standard` |
| `icapServer` | `object` | Tenant configuration | `standard` |
| `icapServer.id` | `integer` | Operational metadata | `standard` |
| `icapServer.name` | `string` | Tenant configuration | `standard` |
| `receiver` | `object` | Tenant configuration | `standard` |
| `receiver.id` | `integer` | Operational metadata | `standard` |
| `receiver.name` | `string` | Tenant configuration | `standard` |
| `receiver.type` | `string` | Operational metadata | `standard` |
| `receiver.tenant` | `object` | Tenant configuration | `standard` |
| `receiver.tenant.id` | `integer` | Operational metadata | `standard` |
| `receiver.tenant.name` | `string` | Tenant configuration | `standard` |
| `receiver.tenant.extensions` |  | Secret | never (unstructured) |
| `locations` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locations.id` | `integer` | Operational metadata | `standard`, `share` |
| `locations.name` | `string` | Tenant configuration | `standard` |
| `locations.extensions` |  | Secret | never (unstructured) |
| `locationGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locationGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `locationGroups.name` | `string` | Tenant configuration | `standard` |
| `locationGroups.extensions` |  | Secret | never (unstructured) |
| `groups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `groups.id` | `integer` | Operational metadata | `standard`, `share` |
| `groups.name` | `string` | Tenant configuration | `standard` |
| `groups.extensions` |  | Secret | never (unstructured) |
| `departments` | `array<object>` | Tenant configuration | `standard`, `share` |
| `departments.id` | `integer` | Operational metadata | `standard`, `share` |
| `departments.name` | `string` | Tenant configuration | `standard` |
| `departments.extensions` |  | Secret | never (unstructured) |
| `users` | `array<object>` | Tenant configuration | `standard`, `share` |
| `users.id` | `integer` | Operational metadata | `standard`, `share` |
| `users.name` | `string` | Tenant configuration | `standard` |
| `users.extensions` |  | Secret | never (unstructured) |
| `urlCategories` | `array<object>` | Tenant configuration | `standard`, `share` |
| `urlCategories.id` | `integer` | Operational metadata | `standard`, `share` |
| `urlCategories.name` | `string` | Tenant configuration | `standard` |
| `urlCategories.extensions` |  | Secret | never (unstructured) |
| `dlpEngines` | `array<object>` | Tenant configuration | `standard`, `share` |
| `dlpEngines.id` | `integer` | Operational metadata | `standard`, `share` |
| `dlpEngines.name` | `string` | Tenant configuration | `standard` |
| `dlpEngines.extensions` |  | Secret | never (unstructured) |
| `timeWindows` | `array<object>` | Tenant configuration | `standard`, `share` |
| `timeWindows.id` | `integer` | Operational metadata | `standard`, `share` |
| `timeWindows.name` | `string` | Tenant configuration | `standard` |
| `timeWindows.extensions` |  | Secret | never (unstructured) |
| `labels` | `array<object>` | Tenant configuration | `standard`, `share` |
| `labels.id` | `integer` | Operational metadata | `standard`, `share` |
| `labels.name` | `string` | Tenant configuration | `standard` |
| `labels.extensions` |  | Secret | never (unstructured) |
| `excludedGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `excludedGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `excludedGroups.name` | `string` | Tenant configuration | `standard` |
| `excludedGroups.extensions` |  | Secret | never (unstructured) |
| `excludedDepartments` | `array<object>` | Tenant configuration | `standard`, `share` |
| `excludedDepartments.id` | `integer` | Operational metadata | `standard`, `share` |
| `excludedDepartments.name` | `string` | Tenant configuration | `standard` |
| `excludedDepartments.extensions` |  | Secret | never (unstructured) |
| `excludedUsers` | `array<object>` | Tenant configuration | `standard`, `share` |
| `excludedUsers.id` | `integer` | Operational metadata | `standard`, `share` |
| `excludedUsers.name` | `string` | Tenant configuration | `standard` |
| `excludedUsers.extensions` |  | Secret | never (unstructured) |
| `includedDomainProfiles` | `array<object>` | Tenant configuration | `standard`, `share` |
| `includedDomainProfiles.id` | `integer` | Operational metadata | `standard`, `share` |
| `includedDomainProfiles.name` | `string` | Tenant configuration | `standard` |
| `includedDomainProfiles.extensions` |  | Secret | never (unstructured) |
| `excludedDomainProfiles` | `array<object>` | Tenant configuration | `standard`, `share` |
| `excludedDomainProfiles.id` | `integer` | Operational metadata | `standard`, `share` |
| `excludedDomainProfiles.name` | `string` | Tenant configuration | `standard` |
| `excludedDomainProfiles.extensions` |  | Secret | never (unstructured) |
| `sourceIpGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `sourceIpGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `sourceIpGroups.name` | `string` | Tenant configuration | `standard` |
| `sourceIpGroups.extensions` |  | Secret | never (unstructured) |
| `workloadGroups` | `array<object>` | Tenant configuration | `standard` |
| `workloadGroups.id` | `integer` | Operational metadata | `standard` |
| `workloadGroups.name` | `string` | Tenant configuration | `standard` |
| `fileTypeCategories` | `array<object>` | Tenant configuration | `standard` |
| `fileTypeCategories.id` | `integer` | Operational metadata | `standard` |
| `fileTypeCategories.name` | `string` | Tenant configuration | `standard` |
| `severity` | `string` | Tenant configuration | `standard`, `share` |
| `parentRule` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `subRules` |  | Secret | never (unstructured) |
| `userRiskScoreLevels` | `array<string>` | Tenant configuration | `standard` |
| `dlpContentLocationsScopes` | `array<string>` | Tenant configuration | `standard`, `share` |
| `inspectHttpGetEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `order`, `rank`, `minSize`, `state`, `matchOnly`, `lastModifiedTime`, `withoutContentInspection`, `ocrEnabled`, `dlpDownloadScanEnabled`, `zccNotificationsEnabled`, `zscalerIncidentReceiver`, `parentRule`, `inspectHttpGetEnabled`: Rule identity, ordering, state, and broad scan/notification flags.
- `eunTemplateId`, `externalAuditorEmail`: Local-only template and auditor identifiers.

## ZIA Dlp Icap Servers

Commands:

```sh
zscalerctl zia dlp-icap-servers list
zscalerctl zia dlp-icap-servers get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/dlp-icap-servers (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `url` | `string` | Sensitive identifier | `standard` |
| `status` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `status`: ICAP server identity and status.
- `url`: Local-only ICAP server endpoint.

## ZIA Dlp Incident Receiver Servers

Commands:

```sh
zscalerctl zia dlp-incident-receiver-servers list
zscalerctl zia dlp-incident-receiver-servers get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/dlp-incident-receiver-servers (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `url` | `string` | Sensitive identifier | `standard` |
| `status` | `string` | Tenant configuration | `standard`, `share` |
| `flags` | `integer` | Operational metadata | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `url`: Local-only incident receiver endpoint.

## ZIA Dlp Notification Templates

Commands:

```sh
zscalerctl zia dlp-notification-templates list
zscalerctl zia dlp-notification-templates get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/dlp-notification-templates (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `subject` | `string` | Free text | `standard` |
| `attachContent` | `boolean` | Tenant configuration | `standard` |
| `plainTextMessage` | `string` | Free text | `standard` |
| `htmlMessage` | `string` | Free text | `standard` |
| `tlsEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: DLP notification template identifier.
- `name`, `tlsEnabled`: Template name and TLS flag.
- `subject`: Notification subject; scanned before output, including bare high-entropy tokens.
- `attachContent`: Local-only attachment behavior because it can indicate whether violating content is emailed.

## ZIA C2c Incident Receivers

Commands:

```sh
zscalerctl zia c2c-incident-receivers list
zscalerctl zia c2c-incident-receivers get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/c2c-incident-receivers (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `status` | `array<string>` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastTenantValidationTime` | `integer` | Operational metadata | `standard` |
| `lastValidationMsg` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastValidationMsg.errorMsg` | `string` | Free text | `standard` |
| `lastValidationMsg.errorCode` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
| `onboardableEntity` | `object` | Tenant configuration | `standard` |
| `onboardableEntity.id` | `integer` | Operational metadata | `standard` |
| `onboardableEntity.name` | `string` | Tenant configuration | `standard` |
| `onboardableEntity.type` | `string` | Operational metadata | `standard` |
| `onboardableEntity.enterpriseTenantId` | `string` | Sensitive identifier | `standard` |
| `onboardableEntity.application` | `string` | Tenant configuration | `standard` |
| `onboardableEntity.lastValidationMsg` | `object` | Operational metadata | `standard` |
| `onboardableEntity.lastValidationMsg.errorMsg` | `string` | Free text | `standard` |
| `onboardableEntity.lastValidationMsg.errorCode` | `string` | Operational metadata | `standard` |
| `onboardableEntity.tenantAuthorizationInfo` |  | Secret | never (credential) |
| `onboardableEntity.zscalerAppTenantId` | `object` | Tenant configuration | `standard` |
| `onboardableEntity.zscalerAppTenantId.id` | `integer` | Operational metadata | `standard` |
| `onboardableEntity.zscalerAppTenantId.name` | `string` | Tenant configuration | `standard` |
| `onboardableEntity.zscalerAppTenantId.extensions` |  | Secret | never (unstructured) |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `status`, `modifiedTime`: Receiver identity, status, and modification timestamp.
- `lastTenantValidationTime`: Local-only validation timestamp.
- `name`: Receiver name; scanned for pasted secret-shaped values.
- `onboardableEntity`: Local-only cloud/onboarded entity reference; authorization details inside the nested SDK shape are dropped.

## ZIA Risk Profiles

Commands:

```sh
zscalerctl zia risk-profiles list
zscalerctl zia risk-profiles get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/risk-profiles (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `profileName` | `string` | Tenant configuration | `standard`, `share` |
| `profileType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `status` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `createTime` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModTime` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `customTags` | `array<object>` | Tenant configuration | `standard` |
| `customTags.id` | `integer` | Operational metadata | `standard` |
| `customTags.name` | `string` | Tenant configuration | `standard` |
| `customTags.externalId` | `string` | Operational metadata | `standard` |
| `customTags.extensions` |  | Secret | never (unstructured) |
| `modifiedBy` | `object` | Tenant configuration | `standard`, `share` |
| `modifiedBy.id` | `integer` | Operational metadata | `standard`, `share` |
| `modifiedBy.name` | `string` | Tenant configuration | `standard` |
| `modifiedBy.extensions` |  | Secret | never (unstructured) |
| `sourceIpRestrictions` | `string` | Sensitive identifier | `standard` |
| `adminAuditLogs` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `certifications` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `dataBreach` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `dataEncryptionInTransit` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `dnsCaaPolicy` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `domainBasedMessageAuth` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `domainKeysIdentifiedMail` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `evasive` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `excludeCertificates` | `integer` | Tenant configuration | `standard`, `share`, `paranoid` |
| `fileSharing` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `httpSecurityHeaders` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `malwareScanningForContent` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `mfaSupport` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `passwordStrength` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `poorItemsOfService` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `remoteScreenSharing` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `riskIndex` | `array<integer>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `senderPolicyFramework` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `sslCertKeySize` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `sslCertValidity` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `sslPinned` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `supportForWaf` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `vulnerability` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `vulnerabilityDisclosure` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `vulnerableToHeartBleed` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `vulnerableToLogJam` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `vulnerableToPoodle` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `weakCipherSupport` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `profileType`, `status`, `createTime`, `lastModTime`: Risk-profile identity, type, state, and timestamp metadata.

## ZIA Nss Servers

Commands:

```sh
zscalerctl zia nss-servers list
zscalerctl zia nss-servers get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/nss-servers (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `status` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `state` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `type` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `icapSvrId` | `integer` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `status`, `state`, `type`: NSS server identity, state, and type metadata.
- `icapSvrId`: Local-only ICAP server reference.

## ZIA Nss Feeds

Commands:

```sh
zscalerctl zia nss-feeds list
zscalerctl zia nss-feeds get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/nss-feeds (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `feedStatus` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `nssLogType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `nssFeedType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `feedOutputFormat` | `string` | Tenant configuration | `standard`, `share` |
| `userObfuscation` | `string` | Tenant configuration | `standard`, `share` |
| `timeZone` | `string` | Tenant configuration | `standard`, `share` |
| `epsRateLimit` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `jsonArrayToggle` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `siemType` | `string` | Tenant configuration | `standard`, `share` |
| `maxBatchSize` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `connectionURL` | `string` | Sensitive identifier | `standard` |
| `authenticationToken` |  | Secret | never (token) |
| `connectionHeaders` |  | Secret | never (opaque blob) |
| `lastSuccessFullTest` | `integer` | Operational metadata | `standard`, `share` |
| `testConnectivityCode` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `base64EncodedCertificate` | `string` | Operational metadata | `standard` |
| `nssType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `clientId` | `string` | Sensitive identifier | `standard` |
| `clientSecret` |  | Secret | never (credential) |
| `authenticationUrl` | `string` | Sensitive identifier | `standard` |
| `grantType` | `string` | Tenant configuration | `standard`, `share` |
| `scope` | `string` | Sensitive identifier | `standard` |
| `cloudNss` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `oauthAuthentication` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `serverIps` | `array<string>` | Sensitive identifier | `standard` |
| `clientIps` | `array<string>` | Sensitive identifier | `standard` |
| `domains` | `array<string>` | Sensitive identifier | `standard` |
| `dnsRequestTypes` | `array<string>` | Tenant configuration | `standard`, `share` |
| `dnsResponseTypes` | `array<string>` | Tenant configuration | `standard`, `share` |
| `dnsResponses` | `array<string>` | Tenant configuration | `standard`, `share` |
| `durations` | `array<string>` | Tenant configuration | `standard`, `share` |
| `dnsActions` | `array<string>` | Tenant configuration | `standard`, `share` |
| `firewallLoggingMode` | `string` | Tenant configuration | `standard`, `share` |
| `clientSourceIps` | `array<string>` | Sensitive identifier | `standard` |
| `firewallActions` | `array<string>` | Tenant configuration | `standard`, `share` |
| `countries` | `array<string>` | Operational metadata | `standard`, `share` |
| `serverSourcePorts` | `array<string>` | Sensitive identifier | `standard` |
| `clientSourcePorts` | `array<string>` | Sensitive identifier | `standard` |
| `actionFilter` | `string` | Tenant configuration | `standard`, `share` |
| `emailDlpPolicyAction` | `string` | Tenant configuration | `standard`, `share` |
| `direction` | `string` | Tenant configuration | `standard`, `share` |
| `event` | `string` | Tenant configuration | `standard`, `share` |
| `policyReasons` | `array<string>` | Tenant configuration | `standard`, `share` |
| `protocolTypes` | `array<string>` | Tenant configuration | `standard`, `share` |
| `userAgents` | `array<string>` | Sensitive identifier | `standard` |
| `requestMethods` | `array<string>` | Tenant configuration | `standard`, `share` |
| `casbPolicyTypes` | `array<string>` | Tenant configuration | `standard`, `share` |
| `urlSuperCategories` | `array<string>` | Tenant configuration | `standard`, `share` |
| `responseCodes` | `array<string>` | Tenant configuration | `standard`, `share` |
| `nwApplications` | `array<string>` | Tenant configuration | `standard`, `share` |
| `natActions` | `array<string>` | Tenant configuration | `standard`, `share` |
| `tunnelTypes` | `array<string>` | Tenant configuration | `standard`, `share` |
| `auditLogType` | `array<string>` | Tenant configuration | `standard`, `share` |
| `projectName` | `array<string>` | Sensitive identifier | `standard` |
| `repoName` | `array<string>` | Sensitive identifier | `standard` |
| `objectName` | `array<string>` | Sensitive identifier | `standard` |
| `channelName` | `array<string>` | Sensitive identifier | `standard` |
| `fileSource` | `array<string>` | Sensitive identifier | `standard` |
| `fileName` | `array<string>` | Sensitive identifier | `standard` |
| `sessionCounts` | `array<string>` | Tenant configuration | `standard`, `share` |
| `refererUrls` | `array<string>` | Sensitive identifier | `standard` |
| `hostNames` | `array<string>` | Sensitive identifier | `standard` |
| `fullUrls` | `array<string>` | Sensitive identifier | `standard` |
| `threatNames` | `array<string>` | Tenant configuration | `standard`, `share` |
| `pageRiskIndexes` | `array<string>` | Tenant configuration | `standard`, `share` |
| `clientDestinationPorts` | `array<string>` | Sensitive identifier | `standard` |
| `tunnelSourcePort` | `array<string>` | Sensitive identifier | `standard` |
| `customEscapedCharacter` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `casbSeverity` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `casbApplications` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `casbAction` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `webApplications` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `webApplicationClasses` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `malwareNames` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `urlClasses` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `malwareClasses` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `advancedThreats` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `trafficForwards` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `webTrafficForwards` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `alerts` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `objectType` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `activity` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `objectType1` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `objectType2` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `endPointDLPLogType` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `emailDLPLogType` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `fileTypeSuperCategories` | `array<string>` | Tenant configuration | `standard`, `share` |
| `fileTypeCategories` | `array<string>` | Tenant configuration | `standard`, `share` |
| `casbFileType` | `array<string>` | Tenant configuration | `standard`, `share` |
| `casbFileTypeSuperCategories` | `array<string>` | Tenant configuration | `standard`, `share` |
| `messageSize` | `array<string>` | Tenant configuration | `standard`, `share` |
| `fileSizes` | `array<string>` | Tenant configuration | `standard`, `share` |
| `requestSizes` | `array<string>` | Tenant configuration | `standard`, `share` |
| `responseSizes` | `array<string>` | Tenant configuration | `standard`, `share` |
| `transactionSizes` | `array<string>` | Tenant configuration | `standard`, `share` |
| `inBoundBytes` | `array<string>` | Tenant configuration | `standard`, `share` |
| `outBoundBytes` | `array<string>` | Tenant configuration | `standard`, `share` |
| `downloadTime` | `array<string>` | Tenant configuration | `standard`, `share` |
| `scanTime` | `array<string>` | Tenant configuration | `standard`, `share` |
| `serverSourceIps` | `array<string>` | Sensitive identifier | `standard` |
| `serverDestinationIps` | `array<string>` | Sensitive identifier | `standard` |
| `tunnelIps` | `array<string>` | Sensitive identifier | `standard` |
| `internalIps` | `array<string>` | Sensitive identifier | `standard` |
| `tunnelSourceIps` | `array<string>` | Sensitive identifier | `standard` |
| `tunnelDestIps` | `array<string>` | Sensitive identifier | `standard` |
| `clientDestinationIps` | `array<string>` | Sensitive identifier | `standard` |
| `advUserAgents` | `array<string>` | Sensitive identifier | `standard` |
| `externalOwners` | `array<object>` | Tenant configuration | `standard`, `share` |
| `externalOwners.id` | `integer` | Operational metadata | `standard`, `share` |
| `externalOwners.name` | `string` | Tenant configuration | `standard` |
| `externalOwners.extensions` |  | Secret | never (unstructured) |
| `externalCollaborators` | `array<object>` | Tenant configuration | `standard`, `share` |
| `externalCollaborators.id` | `integer` | Operational metadata | `standard`, `share` |
| `externalCollaborators.name` | `string` | Tenant configuration | `standard` |
| `externalCollaborators.extensions` |  | Secret | never (unstructured) |
| `internalCollaborators` | `array<object>` | Tenant configuration | `standard`, `share` |
| `internalCollaborators.id` | `integer` | Operational metadata | `standard`, `share` |
| `internalCollaborators.name` | `string` | Tenant configuration | `standard` |
| `internalCollaborators.extensions` |  | Secret | never (unstructured) |
| `itsmObjectType` | `array<object>` | Tenant configuration | `standard`, `share` |
| `itsmObjectType.id` | `integer` | Operational metadata | `standard`, `share` |
| `itsmObjectType.name` | `string` | Tenant configuration | `standard` |
| `itsmObjectType.extensions` |  | Secret | never (unstructured) |
| `urlCategories` | `array<object>` | Tenant configuration | `standard`, `share` |
| `urlCategories.id` | `integer` | Operational metadata | `standard`, `share` |
| `urlCategories.name` | `string` | Tenant configuration | `standard` |
| `urlCategories.extensions` |  | Secret | never (unstructured) |
| `dlpEngines` | `array<object>` | Tenant configuration | `standard`, `share` |
| `dlpEngines.id` | `integer` | Operational metadata | `standard`, `share` |
| `dlpEngines.name` | `string` | Tenant configuration | `standard` |
| `dlpEngines.extensions` |  | Secret | never (unstructured) |
| `dlpDictionaries` | `array<object>` | Tenant configuration | `standard`, `share` |
| `dlpDictionaries.id` | `integer` | Operational metadata | `standard`, `share` |
| `dlpDictionaries.name` | `string` | Tenant configuration | `standard` |
| `dlpDictionaries.extensions` |  | Secret | never (unstructured) |
| `rules` | `array<object>` | Tenant configuration | `standard`, `share` |
| `rules.id` | `integer` | Operational metadata | `standard`, `share` |
| `rules.name` | `string` | Tenant configuration | `standard` |
| `rules.extensions` |  | Secret | never (unstructured) |
| `nwServices` | `array<object>` | Tenant configuration | `standard`, `share` |
| `nwServices.id` | `integer` | Operational metadata | `standard`, `share` |
| `nwServices.name` | `string` | Tenant configuration | `standard` |
| `nwServices.extensions` |  | Secret | never (unstructured) |
| `casbTenant` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `casbTenant.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `casbTenant.pid` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `casbTenant.name` | `string` | Tenant configuration | `standard`, `share` |
| `casbTenant.description` | `string` | Free text | `standard` |
| `casbTenant.deleted` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `casbTenant.getlId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `locations` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `locations.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `locations.pid` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `locations.name` | `string` | Tenant configuration | `standard`, `share` |
| `locations.description` | `string` | Free text | `standard` |
| `locations.deleted` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `locations.getlId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `locationGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `locationGroups.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `locationGroups.pid` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `locationGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `locationGroups.description` | `string` | Free text | `standard` |
| `locationGroups.deleted` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `locationGroups.getlId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `users` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `users.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `users.pid` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `users.name` | `string` | Sensitive identifier | `standard` |
| `users.description` | `string` | Free text | `standard` |
| `users.deleted` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `users.getlId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `departments` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `departments.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `departments.pid` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `departments.name` | `string` | Tenant configuration | `standard`, `share` |
| `departments.description` | `string` | Free text | `standard` |
| `departments.deleted` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `departments.getlId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `senderName` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `senderName.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `senderName.pid` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `senderName.name` | `string` | Sensitive identifier | `standard` |
| `senderName.description` | `string` | Free text | `standard` |
| `senderName.deleted` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `senderName.getlId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `buckets` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `buckets.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `buckets.pid` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `buckets.name` | `string` | Sensitive identifier | `standard` |
| `buckets.description` | `string` | Free text | `standard` |
| `buckets.deleted` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `buckets.getlId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `vpnCredentials` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `vpnCredentials.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `vpnCredentials.pid` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `vpnCredentials.name` | `string` | Sensitive identifier | `standard` |
| `vpnCredentials.description` | `string` | Free text | `standard` |
| `vpnCredentials.deleted` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `vpnCredentials.getlId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

## ZIA File Type Rules

Commands:

```sh
zscalerctl zia file-type-rules list
zscalerctl zia file-type-rules get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/file-type-rules (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `state` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `order` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `rank` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `filteringAction` | `string` | Tenant configuration | `standard`, `share` |
| `timeQuota` | `integer` | Operational metadata | `standard`, `share` |
| `sizeQuota` | `integer` | Operational metadata | `standard`, `share` |
| `accessControl` | `string` | Operational metadata | `standard`, `share` |
| `capturePCAP` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `passwordProtected` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `operation` | `string` | Tenant configuration | `standard`, `share` |
| `activeContent` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `unscannable` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `browserEunTemplateId` | `integer` | Sensitive identifier | `standard` |
| `cloudApplications` | `array<string>` | Tenant configuration | `standard`, `share` |
| `fileTypes` | `array<string>` | Tenant configuration | `standard`, `share` |
| `minSize` | `integer` | Operational metadata | `standard`, `share` |
| `maxSize` | `integer` | Operational metadata | `standard`, `share` |
| `protocols` | `array<string>` | Tenant configuration | `standard`, `share` |
| `urlCategories` | `array<string>` | Tenant configuration | `standard`, `share` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share` |
| `lastModifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
| `locations` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locations.id` | `integer` | Operational metadata | `standard`, `share` |
| `locations.name` | `string` | Tenant configuration | `standard` |
| `locations.extensions` |  | Secret | never (unstructured) |
| `locationGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locationGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `locationGroups.name` | `string` | Tenant configuration | `standard` |
| `locationGroups.extensions` |  | Secret | never (unstructured) |
| `groups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `groups.id` | `integer` | Operational metadata | `standard`, `share` |
| `groups.name` | `string` | Tenant configuration | `standard` |
| `groups.extensions` |  | Secret | never (unstructured) |
| `departments` | `array<object>` | Tenant configuration | `standard`, `share` |
| `departments.id` | `integer` | Operational metadata | `standard`, `share` |
| `departments.name` | `string` | Tenant configuration | `standard` |
| `departments.extensions` |  | Secret | never (unstructured) |
| `users` | `array<object>` | Tenant configuration | `standard`, `share` |
| `users.id` | `integer` | Operational metadata | `standard`, `share` |
| `users.name` | `string` | Tenant configuration | `standard` |
| `users.extensions` |  | Secret | never (unstructured) |
| `timeWindows` | `array<object>` | Tenant configuration | `standard`, `share` |
| `timeWindows.id` | `integer` | Operational metadata | `standard`, `share` |
| `timeWindows.name` | `string` | Tenant configuration | `standard`, `share` |
| `timeWindows.extensions` |  | Secret | never (unstructured) |
| `labels` | `array<object>` | Tenant configuration | `standard`, `share` |
| `labels.id` | `integer` | Operational metadata | `standard`, `share` |
| `labels.name` | `string` | Tenant configuration | `standard`, `share` |
| `labels.extensions` |  | Secret | never (unstructured) |
| `deviceGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `deviceGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `deviceGroups.name` | `string` | Tenant configuration | `standard` |
| `deviceGroups.extensions` |  | Secret | never (unstructured) |
| `devices` | `array<object>` | Tenant configuration | `standard`, `share` |
| `devices.id` | `integer` | Operational metadata | `standard`, `share` |
| `devices.name` | `string` | Tenant configuration | `standard` |
| `devices.extensions` |  | Secret | never (unstructured) |
| `deviceTrustLevels` | `array<string>` | Tenant configuration | `standard`, `share` |
| `zpaAppSegments` | `array<object>` | Tenant configuration | `standard` |
| `zpaAppSegments.id` | `integer` | Operational metadata | `standard` |
| `zpaAppSegments.name` | `string` | Tenant configuration | `standard` |
| `zpaAppSegments.externalId` | `string` | Operational metadata | `standard` |
| `zpaAppSegments.extensions` |  | Secret | never (unstructured) |
<!-- END GENERATED FIELDS -->

## ZIA Sandbox Rules

Commands:

```sh
zscalerctl zia sandbox-rules list
zscalerctl zia sandbox-rules get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/sandbox-rules (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `state` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `order` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `rank` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `baRuleAction` | `string` | Tenant configuration | `standard`, `share` |
| `firstTimeEnable` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `firstTimeOperation` | `string` | Tenant configuration | `standard`, `share` |
| `mlActionEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `byThreatScore` | `integer` | Operational metadata | `standard`, `share` |
| `accessControl` | `string` | Operational metadata | `standard`, `share` |
| `protocols` | `array<string>` | Tenant configuration | `standard`, `share` |
| `baPolicyCategories` | `array<string>` | Tenant configuration | `standard`, `share` |
| `fileTypes` | `array<string>` | Tenant configuration | `standard`, `share` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share` |
| `lastModifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
| `locations` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locations.id` | `integer` | Operational metadata | `standard`, `share` |
| `locations.name` | `string` | Tenant configuration | `standard` |
| `locations.extensions` |  | Secret | never (unstructured) |
| `locationGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locationGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `locationGroups.name` | `string` | Tenant configuration | `standard` |
| `locationGroups.extensions` |  | Secret | never (unstructured) |
| `groups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `groups.id` | `integer` | Operational metadata | `standard`, `share` |
| `groups.name` | `string` | Tenant configuration | `standard` |
| `groups.extensions` |  | Secret | never (unstructured) |
| `departments` | `array<object>` | Tenant configuration | `standard`, `share` |
| `departments.id` | `integer` | Operational metadata | `standard`, `share` |
| `departments.name` | `string` | Tenant configuration | `standard` |
| `departments.extensions` |  | Secret | never (unstructured) |
| `users` | `array<object>` | Tenant configuration | `standard`, `share` |
| `users.id` | `integer` | Operational metadata | `standard`, `share` |
| `users.name` | `string` | Tenant configuration | `standard` |
| `users.extensions` |  | Secret | never (unstructured) |
| `timeWindows` | `array<object>` | Tenant configuration | `standard`, `share` |
| `timeWindows.id` | `integer` | Operational metadata | `standard`, `share` |
| `timeWindows.name` | `string` | Tenant configuration | `standard`, `share` |
| `timeWindows.extensions` |  | Secret | never (unstructured) |
| `labels` | `array<object>` | Tenant configuration | `standard`, `share` |
| `labels.id` | `integer` | Operational metadata | `standard`, `share` |
| `labels.name` | `string` | Tenant configuration | `standard`, `share` |
| `labels.extensions` |  | Secret | never (unstructured) |
| `deviceGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `deviceGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `deviceGroups.name` | `string` | Tenant configuration | `standard` |
| `deviceGroups.extensions` |  | Secret | never (unstructured) |
| `devices` | `array<object>` | Tenant configuration | `standard`, `share` |
| `devices.id` | `integer` | Operational metadata | `standard`, `share` |
| `devices.name` | `string` | Tenant configuration | `standard` |
| `devices.extensions` |  | Secret | never (unstructured) |
| `urlCategories` | `array<string>` | Tenant configuration | `standard`, `share` |
| `zpaAppSegments` | `array<object>` | Tenant configuration | `standard` |
| `zpaAppSegments.id` | `integer` | Operational metadata | `standard` |
| `zpaAppSegments.name` | `string` | Tenant configuration | `standard` |
| `zpaAppSegments.externalId` | `string` | Operational metadata | `standard` |
| `zpaAppSegments.extensions` |  | Secret | never (unstructured) |
| `defaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

## ZIA Firewall Dns Rules

Commands:

```sh
zscalerctl zia firewall-dns-rules list
zscalerctl zia firewall-dns-rules get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/firewall-dns-rules (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `order` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `rank` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `accessControl` | `string` | Operational metadata | `standard`, `share` |
| `action` | `string` | Tenant configuration | `standard`, `share` |
| `state` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `description` | `string` | Free text | `standard` |
| `redirectIp` | `string` | Sensitive identifier | `standard` |
| `blockResponseCode` | `string` | Tenant configuration | `standard`, `share` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share` |
| `lastModifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
| `srcIps` | `array<string>` | Sensitive identifier | `standard` |
| `destAddresses` | `array<string>` | Sensitive identifier | `standard` |
| `destIpCategories` | `array<string>` | Sensitive identifier | `standard` |
| `destCountries` | `array<string>` | Operational metadata | `standard`, `share` |
| `sourceCountries` | `array<string>` | Operational metadata | `standard`, `share` |
| `resCategories` | `array<string>` | Sensitive identifier | `standard` |
| `applications` | `array<string>` | Tenant configuration | `standard`, `share` |
| `dnsRuleRequestTypes` | `array<string>` | Tenant configuration | `standard`, `share` |
| `protocols` | `array<string>` | Tenant configuration | `standard`, `share` |
| `defaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `capturePCAP` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `predefined` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `isWebEunEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `isWebEUNEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `defaultDnsRuleNameUsed` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `excludeContextShieldEndPoint` | `boolean` | Tenant configuration | `standard`, `share` |
| `isEunEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `eunTemplateId` | `integer` | Tenant configuration | `standard`, `share` |
| `applicationGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `applicationGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `applicationGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `applicationGroups.extensions` |  | Secret | never (unstructured) |
| `dnsGateway` | `object` | Tenant configuration | `standard` |
| `dnsGateway.id` | `integer` | Operational metadata | `standard` |
| `dnsGateway.name` | `string` | Tenant configuration | `standard` |
| `zpaIpGroup` | `object` | Tenant configuration | `standard` |
| `zpaIpGroup.id` | `integer` | Operational metadata | `standard` |
| `zpaIpGroup.name` | `string` | Tenant configuration | `standard` |
| `ednsEcsObject` | `object` | Tenant configuration | `standard` |
| `ednsEcsObject.id` | `integer` | Operational metadata | `standard` |
| `ednsEcsObject.name` | `string` | Tenant configuration | `standard` |
| `locations` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locations.id` | `integer` | Operational metadata | `standard`, `share` |
| `locations.name` | `string` | Tenant configuration | `standard` |
| `locations.extensions` |  | Secret | never (unstructured) |
| `locationGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locationGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `locationGroups.name` | `string` | Tenant configuration | `standard` |
| `locationGroups.extensions` |  | Secret | never (unstructured) |
| `departments` | `array<object>` | Tenant configuration | `standard`, `share` |
| `departments.id` | `integer` | Operational metadata | `standard`, `share` |
| `departments.name` | `string` | Tenant configuration | `standard` |
| `departments.extensions` |  | Secret | never (unstructured) |
| `groups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `groups.id` | `integer` | Operational metadata | `standard`, `share` |
| `groups.name` | `string` | Tenant configuration | `standard` |
| `groups.extensions` |  | Secret | never (unstructured) |
| `users` | `array<object>` | Tenant configuration | `standard`, `share` |
| `users.id` | `integer` | Operational metadata | `standard`, `share` |
| `users.name` | `string` | Tenant configuration | `standard` |
| `users.extensions` |  | Secret | never (unstructured) |
| `timeWindows` | `array<object>` | Tenant configuration | `standard`, `share` |
| `timeWindows.id` | `integer` | Operational metadata | `standard`, `share` |
| `timeWindows.name` | `string` | Tenant configuration | `standard`, `share` |
| `timeWindows.extensions` |  | Secret | never (unstructured) |
| `labels` | `array<object>` | Tenant configuration | `standard`, `share` |
| `labels.id` | `integer` | Operational metadata | `standard`, `share` |
| `labels.name` | `string` | Tenant configuration | `standard`, `share` |
| `labels.extensions` |  | Secret | never (unstructured) |
| `destIpGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `destIpGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `destIpGroups.name` | `string` | Tenant configuration | `standard` |
| `destIpGroups.extensions` |  | Secret | never (unstructured) |
| `destIpv6Groups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `destIpv6Groups.id` | `integer` | Operational metadata | `standard`, `share` |
| `destIpv6Groups.name` | `string` | Tenant configuration | `standard` |
| `destIpv6Groups.extensions` |  | Secret | never (unstructured) |
| `srcIpGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `srcIpGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `srcIpGroups.name` | `string` | Tenant configuration | `standard` |
| `srcIpGroups.extensions` |  | Secret | never (unstructured) |
| `srcIpv6Groups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `srcIpv6Groups.id` | `integer` | Operational metadata | `standard`, `share` |
| `srcIpv6Groups.name` | `string` | Tenant configuration | `standard` |
| `srcIpv6Groups.extensions` |  | Secret | never (unstructured) |
| `deviceGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `deviceGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `deviceGroups.name` | `string` | Tenant configuration | `standard` |
| `deviceGroups.extensions` |  | Secret | never (unstructured) |
| `devices` | `array<object>` | Tenant configuration | `standard`, `share` |
| `devices.id` | `integer` | Operational metadata | `standard`, `share` |
| `devices.name` | `string` | Tenant configuration | `standard` |
| `devices.extensions` |  | Secret | never (unstructured) |
<!-- END GENERATED FIELDS -->

Field notes:

- `redirectIp`, `srcIps`, `destAddresses`, `destIpCategories`, `resCategories`: Local-only DNS and network targeting data.

## ZIA Custom File Types

Commands:

```sh
zscalerctl zia custom-file-types list
zscalerctl zia custom-file-types get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/custom-file-types (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `extension` | `string` | Tenant configuration | `standard`, `share` |
| `fileTypeId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `fileTypeId`: Custom file type identifiers.

## ZIA Zpa Gateways

Commands:

```sh
zscalerctl zia zpa-gateways list
zscalerctl zia zpa-gateways get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/zpa-gateways (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `zpaServerGroup` | `object` | Tenant configuration | `standard` |
| `zpaServerGroup.id` | `integer` | Operational metadata | `standard` |
| `zpaServerGroup.name` | `string` | Tenant configuration | `standard` |
| `zpaServerGroup.externalId` | `string` | Operational metadata | `standard` |
| `zpaServerGroup.extensions` |  | Secret | never (unstructured) |
| `zpaAppSegments` | `array<object>` | Tenant configuration | `standard` |
| `zpaAppSegments.id` | `integer` | Operational metadata | `standard` |
| `zpaAppSegments.name` | `string` | Tenant configuration | `standard` |
| `zpaAppSegments.externalId` | `string` | Operational metadata | `standard` |
| `zpaAppSegments.extensions` |  | Secret | never (unstructured) |
| `zpaTenantId` | `integer` | Sensitive identifier | `standard` |
| `lastModifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share` |
| `type` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

## ZIA Browser Isolation Profiles

Commands:

```sh
zscalerctl zia browser-isolation-profiles list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/browser-isolation-profiles (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Sensitive identifier | `standard` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `url` | `string` | Sensitive identifier | `standard` |
| `defaultProfile` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `url`: Local-only browser-isolation profile identifier and URL.
- `name`: Profile name; scanned for pasted secret-shaped values.
- `defaultProfile`: Whether Zscaler marks the profile as default.

This is a list-only resource because the SDK exposes list and name lookup, but
no integer ID `get` path.

## ZIA Dlp Edm Schemas Lite

Commands:

```sh
zscalerctl zia dlp-edm-schemas-lite list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/dlp-edm-schemas-lite (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `schema` | `object` | Tenant configuration | `standard` |
| `schema.id` | `integer` | Operational metadata | `standard` |
| `schema.name` | `string` | Sensitive identifier | `standard` |
| `schema.externalId` | `string` | Sensitive identifier | `standard` |
| `tokenList` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `tokenList.name` | `string` | Sensitive identifier | `standard` |
| `tokenList.type` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `tokenList.primaryKey` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `tokenList.originalColumn` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `tokenList.hashfileColumnOrder` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `tokenList.colLengthBitmap` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `schema`: EDM schema reference; `id`, `name`, and `externalId` render only in standard mode.

This is a list-only lite view of EDM schemas. Full EDM schema metadata remains
available through `zia/dlp-edm-schemas`.

## ZIA Dc Exclusions

Commands:

```sh
zscalerctl zia dc-exclusions list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/dc-exclusions (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `dcid` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `expired` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `startTime` | `integer` | Operational metadata | `standard`, `share` |
| `endTime` | `integer` | Operational metadata | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `dcName` | `object` | Tenant configuration | `standard`, `share` |
| `dcName.id` | `integer` | Operational metadata | `standard`, `share` |
| `dcName.name` | `string` | Tenant configuration | `standard` |
| `dcName.extensions` |  | Secret | never (unstructured) |
<!-- END GENERATED FIELDS -->

Field notes:

- `dcid`, `expired`: Data-center exclusion identity and expiry state.
- `startTime`, `endTime`: Exclusion window timestamps.

This is a list-only resource because the SDK exposes list and name lookup, but
no integer ID `get` path.

## ZIA Sub Clouds

Commands:

```sh
zscalerctl zia sub-clouds list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/sub-clouds (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `dcs` | `array<object>` | Tenant configuration | `standard` |
| `dcs.id` | `integer` | Operational metadata | `standard` |
| `dcs.name` | `string` | Tenant configuration | `standard` |
| `dcs.country` | `string` | Sensitive identifier | `standard` |
| `exclusions` | `array<object>` | Tenant configuration | `standard` |
| `exclusions.datacenter` | `object` | Tenant configuration | `standard` |
| `exclusions.datacenter.id` | `integer` | Operational metadata | `standard` |
| `exclusions.datacenter.name` | `string` | Tenant configuration | `standard` |
| `exclusions.datacenter.extensions` |  | Secret | never (unstructured) |
| `exclusions.lastModifiedUser` | `object` | Operational metadata | `standard` |
| `exclusions.lastModifiedUser.id` | `integer` | Operational metadata | `standard` |
| `exclusions.lastModifiedUser.name` | `string` | Sensitive identifier | `standard` |
| `exclusions.country` | `string` | Sensitive identifier | `standard` |
| `exclusions.expired` | `boolean` | Operational metadata | `standard` |
| `exclusions.disabledByOps` | `boolean` | Operational metadata | `standard` |
| `exclusions.createTime` | `integer` | Operational metadata | `standard` |
| `exclusions.startTime` | `integer` | Operational metadata | `standard` |
| `exclusions.endTime` | `integer` | Operational metadata | `standard` |
| `exclusions.lastModifiedTime` | `integer` | Operational metadata | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Sub-cloud identifier.
- `name`: Sub-cloud name; scanned for pasted secret-shaped values.
- `dcs`: Data-center topology references render only in standard mode.
- `exclusions`: Exclusion metadata renders only in standard mode; nested admin identity is dropped.

This is list-only because the SDK's integer `Get` path answers a different
"last DC in country" question rather than returning the listed sub-cloud record.

## ZIA Ipv6 Config

Commands:

```sh
zscalerctl zia ipv6-config show
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/ipv6-config (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `ipV6Enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `natPrefixes` | `array<object>` | Tenant configuration | `standard` |
| `natPrefixes.id` | `integer` | Operational metadata | `standard` |
| `natPrefixes.name` | `string` | Tenant configuration | `standard` |
| `natPrefixes.description` | `string` | Free text | `standard` |
| `natPrefixes.prefixMask` | `string` | Sensitive identifier | `standard` |
| `natPrefixes.dnsPrefix` | `boolean` | Operational metadata | `standard` |
| `natPrefixes.nonEditable` | `boolean` | Operational metadata | `standard` |
| `dnsPrefix` | `string` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `ipV6Enabled`: IPv6 enabled-state flag.
- `natPrefixes`: NAT64 prefix references render only in standard mode.
- `dnsPrefix`: DNS64 prefix value; dropped from `share` and `paranoid`.

This is a singleton settings object exposed through `show`.

## ZIA Ipv6 Dns64 Prefixes

Commands:

```sh
zscalerctl zia ipv6-dns64-prefixes list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/ipv6-dns64-prefixes (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `prefixMask` | `string` | Sensitive identifier | `standard` |
| `dnsPrefix` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `nonEditable` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Prefix identifier.
- `name`: Prefix name; scanned for pasted secret-shaped values.
- `prefixMask`: Prefix value; dropped from `share` and `paranoid`.
- `dnsPrefix`, `nonEditable`: Prefix flags returned by the SDK.

This is a list-only prefix catalog.

## ZIA Ipv6 Nat64 Prefixes

Commands:

```sh
zscalerctl zia ipv6-nat64-prefixes list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/ipv6-nat64-prefixes (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `prefixMask` | `string` | Sensitive identifier | `standard` |
| `dnsPrefix` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `nonEditable` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Prefix identifier.
- `name`: Prefix name; scanned for pasted secret-shaped values.
- `prefixMask`: Prefix value; dropped from `share` and `paranoid`.
- `dnsPrefix`, `nonEditable`: Prefix flags returned by the SDK.

This is a list-only prefix catalog.

## ZIA Pac Files

Commands:

```sh
zscalerctl zia pac-files list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/pac-files (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `domain` | `string` | Sensitive identifier | `standard` |
| `pacUrl` | `string` | Sensitive identifier | `standard` |
| `pacContent` | `string` | Free text | `standard` |
| `editable` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `pacSubURL` | `string` | Sensitive identifier | `standard` |
| `pacUrlObfuscated` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `pacVerificationStatus` | `string` | Operational metadata | `standard`, `share` |
| `pacVersionStatus` | `string` | Tenant configuration | `standard`, `share` |
| `pacVersion` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `pacCommitMessage` | `string` | Free text | `standard` |
| `totalHits` | `integer` | Operational metadata | `standard`, `share` |
| `lastModificationTime` | `integer` | Operational metadata | `standard`, `share` |
| `lastModifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
| `createTime` | `integer` | Operational metadata | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `editable`, `pacUrlObfuscated`, `pacVersion`: PAC identity, editability, and version metadata.
- `totalHits`, `lastModificationTime`, `createTime`: PAC usage and lifecycle metadata.
- `description`, `pacContent`, `pacCommitMessage`: Admin-controlled text and PAC script content; scanned before output and dropped from shared modes.
- `domain`, `pacUrl`, `pacSubURL`: Tenant domain and PAC URL values.

This is list-only because the SDK exposes list and name lookup for PAC files,
but no base integer ID `get` path.

## ZIA Cloud App Control

Commands:

```sh
zscalerctl zia cloud-app-control list
zscalerctl dump --products zia --resources zia/cloud-app-control --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/cloud-app-control (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `actions` | `array<string>` | Tenant configuration | `standard`, `share` |
| `state` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `rank` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `type` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `order` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `timeQuota` | `integer` | Tenant configuration | `standard`, `share` |
| `sizeQuota` | `integer` | Tenant configuration | `standard`, `share` |
| `cascadingEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `accessControl` | `string` | Operational metadata | `standard`, `share` |
| `applications` | `array<string>` | Tenant configuration | `standard`, `share` |
| `numberOfApplications` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `promptCaptureEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `eunEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `eunTemplateId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `browserEunTemplateId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `predefined` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `validityStartTime` | `integer` | Operational metadata | `standard`, `share` |
| `validityEndTime` | `integer` | Operational metadata | `standard`, `share` |
| `validityTimeZoneId` | `string` | Operational metadata | `standard`, `share` |
| `userAgentTypes` | `array<string>` | Tenant configuration | `standard`, `share` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share` |
| `enforceTimeValidity` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `deviceTrustLevels` | `array<string>` | Tenant configuration | `standard`, `share` |
| `userRiskScoreLevels` | `array<string>` | Tenant configuration | `standard`, `share` |
| `labels` | `array<object>` | Tenant configuration | `standard`, `share` |
| `labels.id` | `integer` | Operational metadata | `standard`, `share` |
| `labels.name` | `string` | Tenant configuration | `standard`, `share` |
| `labels.extensions` |  | Secret | never (unstructured) |
| `timeWindows` | `array<object>` | Tenant configuration | `standard`, `share` |
| `timeWindows.id` | `integer` | Operational metadata | `standard`, `share` |
| `timeWindows.name` | `string` | Tenant configuration | `standard`, `share` |
| `timeWindows.extensions` |  | Secret | never (unstructured) |
| `locations` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locations.id` | `integer` | Operational metadata | `standard`, `share` |
| `locations.name` | `string` | Tenant configuration | `standard` |
| `locations.extensions` |  | Secret | never (unstructured) |
| `locationGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locationGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `locationGroups.name` | `string` | Tenant configuration | `standard` |
| `locationGroups.extensions` |  | Secret | never (unstructured) |
| `tenancyProfileIds` | `array<object>` | Tenant configuration | `standard`, `share` |
| `tenancyProfileIds.id` | `integer` | Operational metadata | `standard`, `share` |
| `tenancyProfileIds.name` | `string` | Tenant configuration | `standard` |
| `tenancyProfileIds.extensions` |  | Secret | never (unstructured) |
| `departments` | `array<object>` | Tenant configuration | `standard`, `share` |
| `departments.id` | `integer` | Operational metadata | `standard`, `share` |
| `departments.name` | `string` | Tenant configuration | `standard` |
| `departments.extensions` |  | Secret | never (unstructured) |
| `groups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `groups.id` | `integer` | Operational metadata | `standard`, `share` |
| `groups.name` | `string` | Tenant configuration | `standard` |
| `groups.extensions` |  | Secret | never (unstructured) |
| `users` | `array<object>` | Tenant configuration | `standard`, `share` |
| `users.id` | `integer` | Operational metadata | `standard`, `share` |
| `users.name` | `string` | Tenant configuration | `standard` |
| `users.extensions` |  | Secret | never (unstructured) |
| `deviceGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `deviceGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `deviceGroups.name` | `string` | Tenant configuration | `standard` |
| `deviceGroups.extensions` |  | Secret | never (unstructured) |
| `devices` | `array<object>` | Tenant configuration | `standard`, `share` |
| `devices.id` | `integer` | Operational metadata | `standard`, `share` |
| `devices.name` | `string` | Tenant configuration | `standard` |
| `devices.extensions` |  | Secret | never (unstructured) |
| `cloudAppRiskProfile` | `object` | Tenant configuration | `standard` |
| `cloudAppRiskProfile.id` | `integer` | Operational metadata | `standard` |
| `cloudAppRiskProfile.name` | `string` | Tenant configuration | `standard` |
| `cloudAppInstances` | `array<object>` | Tenant configuration | `standard` |
| `cloudAppInstances.id` | `integer` | Operational metadata | `standard` |
| `cloudAppInstances.name` | `string` | Tenant configuration | `standard` |
| `cloudAppInstances.type` | `string` | Operational metadata | `standard` |
| `cbiProfile` | `object` | Tenant configuration | `standard` |
| `cbiProfile.id` | `string` | Operational metadata | `standard` |
| `cbiProfile.name` | `string` | Tenant configuration | `standard` |
| `cbiProfile.url` | `string` | Sensitive identifier | `standard` |
| `cbiProfile.profileSeq` | `integer` | Operational metadata | `standard` |
| `cbiProfile.defaultProfile` | `boolean` | Operational metadata | `standard` |
| `cbiProfile.sandboxMode` | `boolean` | Tenant configuration | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Cloud App Control rule identifier.
- `state`, `rank`, `type`, `order`, `cascadingEnabled`, `numberOfApplications`, `eunEnabled`, `eunTemplateId`, `browserEunTemplateId`, `predefined`, `enforceTimeValidity`: Rule state, type, ordering, and feature flags.
- `accessControl`, `validityStartTime`, `validityEndTime`, `validityTimeZoneId`, `lastModifiedTime`: Non-principal rule metadata.
- `actions`, `applications`, `userAgentTypes`, `deviceTrustLevels`, `userRiskScoreLevels`, `timeQuota`, `sizeQuota`: Reviewed rule action, application, and quota criteria.
- `labels`, `timeWindows`: Nested references render reviewed `id`/`name` fields only.

Cloud App Control has no flat list endpoint; the reader enumerates the rule
types and concatenates each type's rules. Only `list` is supported (per-rule
`get` requires a rule-type and id pair).

## ZIA Cloud Application Policy

Commands:

```sh
zscalerctl zia cloud-application-policy list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/cloud-application-policy (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `app` | `string` | Tenant configuration | `standard`, `share` |
| `appName` | `string` | Tenant configuration | `standard`, `share` |
| `parent` | `string` | Tenant configuration | `standard`, `share` |
| `parentName` | `string` | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `app`, `appName`, `parent`, `parentName`: Cloud application and category identifiers returned by the policy catalog.

This is a list-only catalog resource.

## ZIA Cloud Application Ssl Policy

Commands:

```sh
zscalerctl zia cloud-application-ssl-policy list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/cloud-application-ssl-policy (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `app` | `string` | Tenant configuration | `standard`, `share` |
| `appName` | `string` | Tenant configuration | `standard`, `share` |
| `parent` | `string` | Tenant configuration | `standard`, `share` |
| `parentName` | `string` | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `app`, `appName`, `parent`, `parentName`: Cloud application and category identifiers returned by the SSL policy catalog.

This is a list-only catalog resource.

## ZIA Domain Profiles

Commands:

```sh
zscalerctl zia domain-profiles list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/domain-profiles (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `profileId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `profileName` | `string` | Tenant configuration | `standard`, `share` |
| `includeCompanyDomains` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `includeSubdomains` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `description` | `string` | Free text | `standard` |
| `customDomains` | `array<string>` | Sensitive identifier | `standard` |
| `predefinedEmailDomains` | `array<string>` | Tenant configuration | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `profileId`, `includeCompanyDomains`, `includeSubdomains`: Domain-profile identity and flags.
- `profileName`: Domain-profile name; scanned for pasted secret-shaped values.
- `customDomains`: Tenant-defined domains.
- `predefinedEmailDomains`: Provider domain catalog references.

This is a list-only SaaS Security API resource.

## ZIA Casb Tombstone Templates

Commands:

```sh
zscalerctl zia casb-tombstone-templates list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/casb-tombstone-templates (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Template identifier.
- `name`: Template name; scanned for pasted secret-shaped values.
- `description`: Tombstone text; scanned before output and dropped from shared modes.

This is a list-only SaaS Security API resource.

## ZIA Casb Email Labels

Commands:

```sh
zscalerctl zia casb-email-labels list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/casb-email-labels (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `labelDesc` | `string` | Free text | `standard` |
| `labelColor` | `string` | Tenant configuration | `standard`, `share` |
| `labelDeleted` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `labelDeleted`: Label identity and deletion flag.
- `name`, `labelColor`: Label name and color.
- `labelDesc`: Label description; scanned before output.

This is a list-only SaaS Security API resource.

## ZIA Casb Tenants

Commands:

```sh
zscalerctl zia casb-tenants list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/casb-tenants (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `tenantId` | `integer` | Sensitive identifier | `standard` |
| `modifiedTime` | `integer` | Operational metadata | `standard`, `share` |
| `lastTenantValidationTime` | `integer` | Operational metadata | `standard` |
| `tenantDeleted` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `tenantWebhookEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `reAuth` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `featuresSupported` | `array<string>` | Tenant configuration | `standard`, `share` |
| `status` | `array<string>` | Tenant configuration | `standard`, `share` |
| `enterpriseTenantId` | `string` | Sensitive identifier | `standard` |
| `tenantName` | `string` | Sensitive identifier | `standard` |
| `saasApplication` | `string` | Tenant configuration | `standard`, `share` |
| `zscalerAppTenantId` | `object` | Tenant configuration | `standard` |
| `zscalerAppTenantId.id` | `integer` | Operational metadata | `standard` |
| `zscalerAppTenantId.name` | `string` | Tenant configuration | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `modifiedTime`: Tenant lifecycle metadata.
- `featuresSupported`, `status`, `saasApplication`: SaaS tenant feature and status metadata.
- `enterpriseTenantId`, `tenantName`: External tenant identifiers and names.

This is a list-only SaaS Security API resource.

## ZIA Casb Dlp Rules

Commands:

```sh
zscalerctl zia casb-dlp-rules list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/casb-dlp-rules (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `order` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `rank` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `type` | `string` | Tenant configuration | `standard`, `share` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `state` | `string` | Tenant configuration | `standard`, `share` |
| `action` | `string` | Tenant configuration | `standard`, `share` |
| `severity` | `string` | Tenant configuration | `standard`, `share` |
| `accessControl` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `bucketOwner` | `string` | Sensitive identifier | `standard` |
| `externalAuditorEmail` | `string` | Sensitive identifier | `standard` |
| `contentLocation` | `string` | Sensitive identifier | `standard` |
| `recipient` | `string` | Sensitive identifier | `standard` |
| `quarantineLocation` | `string` | Sensitive identifier | `standard` |
| `domains` | `array<string>` | Sensitive identifier | `standard` |
| `numberOfInternalCollaborators` | `string` | Tenant configuration | `standard`, `share` |
| `numberOfExternalCollaborators` | `string` | Tenant configuration | `standard`, `share` |
| `watermarkDeleteOldVersion` | `boolean` | Tenant configuration | `standard`, `share` |
| `includeCriteriaDomainProfile` | `boolean` | Tenant configuration | `standard`, `share` |
| `includeEmailRecipientProfile` | `boolean` | Tenant configuration | `standard`, `share` |
| `withoutContentInspection` | `boolean` | Tenant configuration | `standard`, `share` |
| `includeEntityGroups` | `boolean` | Tenant configuration | `standard`, `share` |
| `fileTypes` | `array<string>` | Tenant configuration | `standard`, `share` |
| `collaborationScope` | `array<string>` | Tenant configuration | `standard`, `share` |
| `components` | `array<string>` | Tenant configuration | `standard`, `share` |
| `deviceTrustLevels` | `array<string>` | Tenant configuration | `standard`, `share` |
| `objectTypes` | `array<object>` | Tenant configuration | `standard`, `share` |
| `objectTypes.id` | `integer` | Operational metadata | `standard`, `share` |
| `objectTypes.name` | `string` | Tenant configuration | `standard` |
| `objectTypes.extensions` |  | Secret | never (unstructured) |
| `buckets` | `array<object>` | Tenant configuration | `standard`, `share` |
| `buckets.id` | `integer` | Operational metadata | `standard`, `share` |
| `buckets.name` | `string` | Tenant configuration | `standard` |
| `buckets.extensions` |  | Secret | never (unstructured) |
| `labels` | `array<object>` | Tenant configuration | `standard`, `share` |
| `labels.id` | `integer` | Operational metadata | `standard`, `share` |
| `labels.name` | `string` | Tenant configuration | `standard`, `share` |
| `labels.extensions` |  | Secret | never (unstructured) |
| `includedDomainProfiles` | `array<object>` | Tenant configuration | `standard`, `share` |
| `includedDomainProfiles.id` | `integer` | Operational metadata | `standard`, `share` |
| `includedDomainProfiles.name` | `string` | Tenant configuration | `standard` |
| `includedDomainProfiles.extensions` |  | Secret | never (unstructured) |
| `excludedDomainProfiles` | `array<object>` | Tenant configuration | `standard`, `share` |
| `excludedDomainProfiles.id` | `integer` | Operational metadata | `standard`, `share` |
| `excludedDomainProfiles.name` | `string` | Tenant configuration | `standard` |
| `excludedDomainProfiles.extensions` |  | Secret | never (unstructured) |
| `criteriaDomainProfiles` | `array<object>` | Tenant configuration | `standard`, `share` |
| `criteriaDomainProfiles.id` | `integer` | Operational metadata | `standard`, `share` |
| `criteriaDomainProfiles.name` | `string` | Tenant configuration | `standard` |
| `criteriaDomainProfiles.extensions` |  | Secret | never (unstructured) |
| `emailRecipientProfiles` | `array<object>` | Tenant configuration | `standard`, `share` |
| `emailRecipientProfiles.id` | `integer` | Operational metadata | `standard`, `share` |
| `emailRecipientProfiles.name` | `string` | Tenant configuration | `standard` |
| `emailRecipientProfiles.extensions` |  | Secret | never (unstructured) |
| `devices` | `array<object>` | Tenant configuration | `standard`, `share` |
| `devices.id` | `integer` | Operational metadata | `standard`, `share` |
| `devices.name` | `string` | Tenant configuration | `standard` |
| `devices.extensions` |  | Secret | never (unstructured) |
| `deviceGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `deviceGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `deviceGroups.name` | `string` | Tenant configuration | `standard` |
| `deviceGroups.extensions` |  | Secret | never (unstructured) |
| `entityGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `entityGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `entityGroups.name` | `string` | Tenant configuration | `standard` |
| `entityGroups.extensions` |  | Secret | never (unstructured) |
| `cloudAppTenants` | `array<object>` | Tenant configuration | `standard`, `share` |
| `cloudAppTenants.id` | `integer` | Operational metadata | `standard`, `share` |
| `cloudAppTenants.name` | `string` | Tenant configuration | `standard` |
| `cloudAppTenants.extensions` |  | Secret | never (unstructured) |
| `users` | `array<object>` | Tenant configuration | `standard`, `share` |
| `users.id` | `integer` | Operational metadata | `standard`, `share` |
| `users.name` | `string` | Tenant configuration | `standard` |
| `users.extensions` |  | Secret | never (unstructured) |
| `groups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `groups.id` | `integer` | Operational metadata | `standard`, `share` |
| `groups.name` | `string` | Tenant configuration | `standard` |
| `groups.extensions` |  | Secret | never (unstructured) |
| `departments` | `array<object>` | Tenant configuration | `standard`, `share` |
| `departments.id` | `integer` | Operational metadata | `standard`, `share` |
| `departments.name` | `string` | Tenant configuration | `standard` |
| `departments.extensions` |  | Secret | never (unstructured) |
| `dlpEngines` | `array<object>` | Tenant configuration | `standard`, `share` |
| `dlpEngines.id` | `integer` | Operational metadata | `standard`, `share` |
| `dlpEngines.name` | `string` | Tenant configuration | `standard` |
| `dlpEngines.extensions` |  | Secret | never (unstructured) |
| `lastModifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
| `auditor` | `object` | Tenant configuration | `standard` |
| `auditor.id` | `integer` | Operational metadata | `standard` |
| `auditor.name` | `string` | Tenant configuration | `standard` |
| `zscalerIncidentReceiver` | `object` | Tenant configuration | `standard` |
| `zscalerIncidentReceiver.id` | `integer` | Operational metadata | `standard` |
| `zscalerIncidentReceiver.name` | `string` | Tenant configuration | `standard` |
| `auditorNotification` | `object` | Tenant configuration | `standard` |
| `auditorNotification.id` | `integer` | Operational metadata | `standard` |
| `auditorNotification.name` | `string` | Tenant configuration | `standard` |
| `tag` | `object` | Tenant configuration | `standard` |
| `tag.id` | `integer` | Operational metadata | `standard` |
| `tag.name` | `string` | Tenant configuration | `standard` |
| `watermarkProfile` | `object` | Tenant configuration | `standard` |
| `watermarkProfile.id` | `integer` | Operational metadata | `standard` |
| `watermarkProfile.name` | `string` | Tenant configuration | `standard` |
| `redactionProfile` | `object` | Tenant configuration | `standard` |
| `redactionProfile.id` | `integer` | Operational metadata | `standard` |
| `redactionProfile.name` | `string` | Tenant configuration | `standard` |
| `casbEmailLabel` | `object` | Tenant configuration | `standard` |
| `casbEmailLabel.id` | `integer` | Operational metadata | `standard` |
| `casbEmailLabel.name` | `string` | Tenant configuration | `standard` |
| `casbTombstoneTemplate` | `object` | Tenant configuration | `standard` |
| `casbTombstoneTemplate.id` | `integer` | Operational metadata | `standard` |
| `casbTombstoneTemplate.name` | `string` | Tenant configuration | `standard` |
| `receiver` | `object` | Tenant configuration | `standard` |
| `receiver.id` | `integer` | Operational metadata | `standard` |
| `receiver.name` | `string` | Tenant configuration | `standard` |
| `receiver.type` | `string` | Tenant configuration | `standard` |
| `receiver.tenant` | `object` | Tenant configuration | `standard` |
| `receiver.tenant.id` | `integer` | Operational metadata | `standard` |
| `receiver.tenant.name` | `string` | Tenant configuration | `standard` |
| `receiver.tenant.extensions` |  | Secret | never (unstructured) |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `order`, `rank`, `lastModifiedTime`: Rule identity, order, rank, and lifecycle metadata.
- `type`, `name`, `state`, `action`, `severity`, `accessControl`, `fileTypes`, `collaborationScope`, `components`, `deviceTrustLevels`, `watermarkDeleteOldVersion`, `includeCriteriaDomainProfile`, `includeEmailRecipientProfile`, `withoutContentInspection`, `includeEntityGroups`, `numberOfInternalCollaborators`, `numberOfExternalCollaborators`: Rule configuration and matching flags.
- `description`: Admin-controlled rule text; scanned before output.
- `bucketOwner`, `externalAuditorEmail`, `contentLocation`, `recipient`, `quarantineLocation`, `domains`: Tenant, recipient, domain, and storage-location identifiers.
- `labels`: Rule labels.

This is list-only because the SDK's ID `get` path requires a rule type in
addition to the rule ID.

## ZIA Casb Malware Rules

Commands:

```sh
zscalerctl zia casb-malware-rules list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/casb-malware-rules (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `order` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `type` | `string` | Tenant configuration | `standard`, `share` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `state` | `string` | Tenant configuration | `standard`, `share` |
| `action` | `string` | Tenant configuration | `standard`, `share` |
| `accessControl` | `string` | Tenant configuration | `standard`, `share` |
| `quarantineLocation` | `string` | Sensitive identifier | `standard` |
| `scanInboundEmailLink` | `string` | Sensitive identifier | `standard` |
| `lastModifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
| `casbEmailLabel` | `object` | Tenant configuration | `standard` |
| `casbEmailLabel.id` | `integer` | Operational metadata | `standard` |
| `casbEmailLabel.name` | `string` | Tenant configuration | `standard` |
| `casbTombstoneTemplate` | `object` | Tenant configuration | `standard` |
| `casbTombstoneTemplate.id` | `integer` | Operational metadata | `standard` |
| `casbTombstoneTemplate.name` | `string` | Tenant configuration | `standard` |
| `buckets` | `array<object>` | Tenant configuration | `standard`, `share` |
| `buckets.id` | `integer` | Operational metadata | `standard`, `share` |
| `buckets.name` | `string` | Tenant configuration | `standard` |
| `buckets.extensions` |  | Secret | never (unstructured) |
| `labels` | `array<object>` | Tenant configuration | `standard`, `share` |
| `labels.id` | `integer` | Operational metadata | `standard`, `share` |
| `labels.name` | `string` | Tenant configuration | `standard`, `share` |
| `labels.extensions` |  | Secret | never (unstructured) |
| `cloudAppTenantIds` | `array<object>` | Tenant configuration | `standard`, `share` |
| `cloudAppTenantIds.id` | `integer` | Operational metadata | `standard`, `share` |
| `cloudAppTenantIds.name` | `string` | Tenant configuration | `standard` |
| `cloudAppTenantIds.extensions` |  | Secret | never (unstructured) |
| `cloudAppTenants` | `array<object>` | Tenant configuration | `standard`, `share` |
| `cloudAppTenants.id` | `integer` | Operational metadata | `standard`, `share` |
| `cloudAppTenants.name` | `string` | Tenant configuration | `standard` |
| `cloudAppTenants.extensions` |  | Secret | never (unstructured) |
| `cloudApplicationTenant` | `array<object>` | Tenant configuration | `standard` |
| `cloudApplicationTenant.tenantId` | `integer` | Sensitive identifier | `standard` |
| `cloudApplicationTenant.modifiedTime` | `integer` | Operational metadata | `standard` |
| `cloudApplicationTenant.lastTenantValidationTime` | `integer` | Operational metadata | `standard` |
| `cloudApplicationTenant.tenantDeleted` | `boolean` | Operational metadata | `standard` |
| `cloudApplicationTenant.tenantWebhookEnabled` | `boolean` | Operational metadata | `standard` |
| `cloudApplicationTenant.reAuth` | `boolean` | Operational metadata | `standard` |
| `cloudApplicationTenant.featuresSupported` | `array<string>` | Tenant configuration | `standard` |
| `cloudApplicationTenant.status` | `array<string>` | Tenant configuration | `standard` |
| `cloudApplicationTenant.enterpriseTenantId` | `string` | Sensitive identifier | `standard` |
| `cloudApplicationTenant.tenantName` | `string` | Sensitive identifier | `standard` |
| `cloudApplicationTenant.saasApplication` | `string` | Tenant configuration | `standard` |
| `cloudApplicationTenant.zscalerAppTenantId` | `object` | Tenant configuration | `standard` |
| `cloudApplicationTenant.zscalerAppTenantId.id` | `integer` | Operational metadata | `standard` |
| `cloudApplicationTenant.zscalerAppTenantId.name` | `string` | Tenant configuration | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `order`: Rule identity and order.
- `type`, `name`, `state`, `action`, `accessControl`: Rule configuration and state.
- `quarantineLocation`, `scanInboundEmailLink`: Tenant storage and email-link identifiers.
- `labels`: Rule labels.

This is list-only because the SDK's ID `get` path requires a rule type in
addition to the rule ID.

## ZIA Browser Control Settings

Commands:

```sh
zscalerctl zia browser-control-settings show
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/browser-control-settings (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `pluginCheckFrequency` | `string` | Tenant configuration | `standard`, `share` |
| `bypassAllBrowsers` | `boolean` | Tenant configuration | `standard`, `share` |
| `allowAllBrowsers` | `boolean` | Tenant configuration | `standard`, `share` |
| `enableWarnings` | `boolean` | Tenant configuration | `standard`, `share` |
| `enableSmartBrowserIsolation` | `boolean` | Tenant configuration | `standard`, `share` |
| `bypassPlugins` | `array<string>` | Sensitive identifier | `standard` |
| `bypassApplications` | `array<string>` | Sensitive identifier | `standard` |
| `blockedInternetExplorerVersions` | `array<string>` | Sensitive identifier | `standard` |
| `blockedChromeVersions` | `array<string>` | Sensitive identifier | `standard` |
| `blockedFirefoxVersions` | `array<string>` | Sensitive identifier | `standard` |
| `blockedSafariVersions` | `array<string>` | Sensitive identifier | `standard` |
| `blockedOperaVersions` | `array<string>` | Sensitive identifier | `standard` |
| `smartIsolationProfileId` | `integer` | Operational metadata | `standard` |
| `smartIsolationUsers` | `array<object>` | Tenant configuration | `standard`, `share` |
| `smartIsolationUsers.id` | `integer` | Operational metadata | `standard`, `share` |
| `smartIsolationUsers.name` | `string` | Tenant configuration | `standard` |
| `smartIsolationUsers.extensions` |  | Secret | never (unstructured) |
| `smartIsolationGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `smartIsolationGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `smartIsolationGroups.name` | `string` | Tenant configuration | `standard` |
| `smartIsolationGroups.extensions` |  | Secret | never (unstructured) |
| `smartIsolationProfile` | `object` | Tenant configuration | `standard` |
| `smartIsolationProfile.id` | `string` | Operational metadata | `standard` |
| `smartIsolationProfile.name` | `string` | Tenant configuration | `standard` |
| `smartIsolationProfile.url` | `string` | Sensitive identifier | `standard` |
| `smartIsolationProfile.defaultProfile` | `boolean` | Operational metadata | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `pluginCheckFrequency`, `bypassAllBrowsers`, `allowAllBrowsers`, `enableWarnings`, `enableSmartBrowserIsolation`: Browser warning and Smart Isolation controls.
- `bypassPlugins`, `bypassApplications`, `blockedInternetExplorerVersions`, `blockedChromeVersions`, `blockedFirefoxVersions`, `blockedSafariVersions`, `blockedOperaVersions`: Application/plugin and browser-version targeting.

This is a singleton settings resource.

## ZIA Supported Browser Versions

Commands:

```sh
zscalerctl zia supported-browser-versions list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/supported-browser-versions (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `browserType` | `string` | Tenant configuration | `standard`, `share` |
| `versions` | `array<string>` | Tenant configuration | `standard`, `share` |
| `olderVersions` | `array<string>` | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `browserType`, `versions`, `olderVersions`: Browser support catalog values returned by the Browser Control API.

This is a list-only browser-support catalog.

## ZIA Ftp Control Policy

Commands:

```sh
zscalerctl zia ftp-control-policy show
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/ftp-control-policy (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `ftpOverHttpEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `ftpEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `urlCategories` | `array<string>` | Tenant configuration | `standard`, `share` |
| `urls` | `array<string>` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `ftpOverHttpEnabled`, `ftpEnabled`, `urlCategories`: FTP enablement and category controls.
- `urls`: Tenant-defined FTP URL entries.

This is a singleton settings resource.

## ZIA Remote Assistance

Commands:

```sh
zscalerctl zia remote-assistance show
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/remote-assistance (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `viewOnlyUntil` | `integer` | Operational metadata | `standard`, `share` |
| `fullAccessUntil` | `integer` | Operational metadata | `standard`, `share` |
| `usernameObfuscated` | `boolean` | Tenant configuration | `standard`, `share` |
| `deviceInfoObfuscate` | `boolean` | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `viewOnlyUntil`, `fullAccessUntil`: Remote-assistance access timestamps.
- `usernameObfuscated`, `deviceInfoObfuscate`: Dashboard obfuscation controls.

This is a singleton settings resource.

## ZIA Admin Users

Commands:

```sh
zscalerctl zia admin-users list
zscalerctl zia admin-users get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/admin-users (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `loginName` | `string` | Sensitive identifier | `standard` |
| `userName` | `string` | Sensitive identifier | `standard` |
| `email` | `string` | Sensitive identifier | `standard` |
| `comments` | `string` | Free text | `standard` |
| `disabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `password` |  | Secret | never (credential) |
| `pwdLastModifiedTime` | `integer` | Operational metadata | `standard` |
| `isNonEditable` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `isPasswordLoginAllowed` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `isPasswordExpired` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `isAuditor` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `isSecurityReportCommEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `isServiceUpdateCommEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `isProductUpdateCommEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `isExecMobileAppEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `adminScopescopeGroupMemberEntities` | `array<object>` | Tenant configuration | `standard`, `share` |
| `adminScopescopeGroupMemberEntities.id` | `integer` | Operational metadata | `standard`, `share` |
| `adminScopescopeGroupMemberEntities.name` | `string` | Tenant configuration | `standard` |
| `adminScopescopeGroupMemberEntities.extensions` |  | Secret | never (unstructured) |
| `adminScopeScopeEntities` | `array<object>` | Tenant configuration | `standard`, `share` |
| `adminScopeScopeEntities.id` | `integer` | Operational metadata | `standard`, `share` |
| `adminScopeScopeEntities.name` | `string` | Tenant configuration | `standard` |
| `adminScopeScopeEntities.extensions` |  | Secret | never (unstructured) |
| `adminScopeType` | `string` | Tenant configuration | `standard`, `share` |
| `role` | `object` | Tenant configuration | `standard` |
| `role.id` | `integer` | Operational metadata | `standard` |
| `role.name` | `string` | Tenant configuration | `standard` |
| `role.isNameL10nTag` | `boolean` | Operational metadata | `standard` |
| `role.extensions` |  | Secret | never (unstructured) |
| `execMobileAppTokens` |  | Secret | never (token) |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `disabled`, `isNonEditable`, `isAuditor`: Admin identity and state flags.
- `pwdLastModifiedTime`: Password lifecycle timestamp.
- `loginName`, `userName`, `email`: Admin person identifiers.
- `comments`: Admin-controlled notes; scanned before output.
- `isSecurityReportCommEnabled`, `isServiceUpdateCommEnabled`, `isProductUpdateCommEnabled`, `isExecMobileAppEnabled`, `adminScopeType`: Admin communication, app-access, and scope-type controls.

## ZIA Admin Roles

Commands:

```sh
zscalerctl zia admin-roles list
zscalerctl zia admin-roles get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/admin-roles (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `rank` | `integer` | Operational metadata | `standard` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `policyAccess` | `string` | Tenant configuration | `standard` |
| `alertingAccess` | `string` | Tenant configuration | `standard` |
| `dashboardAccess` | `string` | Tenant configuration | `standard` |
| `reportAccess` | `string` | Tenant configuration | `standard` |
| `analysisAccess` | `string` | Tenant configuration | `standard` |
| `usernameAccess` | `string` | Tenant configuration | `standard` |
| `adminAcctAccess` | `string` | Tenant configuration | `standard` |
| `deviceInfoAccess` | `string` | Tenant configuration | `standard` |
| `isAuditor` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `permissions` | `array<string>` | Tenant configuration | `standard` |
| `isNonEditable` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `logsLimit` | `string` | Tenant configuration | `standard` |
| `roleType` | `string` | Tenant configuration | `standard`, `share` |
| `reportTimeDuration` | `integer` | Tenant configuration | `standard` |
| `featurePermissions` |  | Secret | never (unstructured) |
| `extFeaturePermissions` |  | Secret | never (unstructured) |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `isAuditor`, `isNonEditable`: Role identity and state flags.
- `rank`: Admin-rank metadata.
- `name`, `roleType`: Role name and type.
- `policyAccess`, `alertingAccess`, `dashboardAccess`, `reportAccess`, `analysisAccess`, `usernameAccess`, `adminAcctAccess`, `deviceInfoAccess`, `permissions`, `logsLimit`, `reportTimeDuration`: Detailed authorization and log-access inventory.
- `featurePermissions`, `extFeaturePermissions`: Arbitrary permission maps are dropped until modeled with a stable shape.

## ZIA Email Profiles

Commands:

```sh
zscalerctl zia email-profiles list
zscalerctl zia email-profiles get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/email-profiles (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `emails` | `array<string>` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Email recipient profile identifier.
- `emails`: Local-only recipient addresses.

## ZIA Advanced Settings

Commands:

```sh
zscalerctl zia advanced-settings show
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/advanced-settings (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `apiSessionTimeout` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `blockConnectHostSniMismatch` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `blockDomainFrontingOnHostHeader` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `blockHttpTunnelOnNonHttpPorts` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `blockNonCompliantHttpRequestOnHttpPorts` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `blockNonHttpOnHttpPortEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `cascadeUrlFiltering` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `dynamicUserRiskEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `ecsForAllEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableAdminRankAccess` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableDnsResolutionOnTransparentProxy` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableEvaluatePolicyOnGlobalSSLBypass` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableIPv6DnsOptimizationOnAllTransparentProxy` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableIPv6DnsResolutionOnTransparentProxy` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableOffice365` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `enablePolicyForUnauthenticatedTraffic` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `enforceSurrogateIpForWindowsApp` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `http2NonbrowserTrafficEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `logInternalIp` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `preferSniOverConnHost` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `sipaXffHeaderEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `trackHttpTunnelOnHttpPorts` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `uiSessionTimeout` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `zscalerClientConnector1AndPacRoadWarriorInFirewall` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `authBypassApps` | `array<string>` | Sensitive identifier | `standard` |
| `authBypassUrlCategories` | `array<string>` | Sensitive identifier | `standard` |
| `authBypassUrls` | `array<string>` | Sensitive identifier | `standard` |
| `basicBypassApps` | `array<string>` | Sensitive identifier | `standard` |
| `basicBypassUrlCategories` | `array<string>` | Sensitive identifier | `standard` |
| `blockDomainFrontingApps` | `array<string>` | Sensitive identifier | `standard` |
| `digestAuthBypassApps` | `array<string>` | Sensitive identifier | `standard` |
| `digestAuthBypassUrlCategories` | `array<string>` | Sensitive identifier | `standard` |
| `digestAuthBypassUrls` | `array<string>` | Sensitive identifier | `standard` |
| `dnsResolutionOnTransparentProxyApps` | `array<string>` | Sensitive identifier | `standard` |
| `dnsResolutionOnTransparentProxyExemptApps` | `array<string>` | Sensitive identifier | `standard` |
| `dnsResolutionOnTransparentProxyExemptUrlCategories` | `array<string>` | Sensitive identifier | `standard` |
| `dnsResolutionOnTransparentProxyExemptUrls` | `array<string>` | Sensitive identifier | `standard` |
| `dnsResolutionOnTransparentProxyIPv6Apps` | `array<string>` | Sensitive identifier | `standard` |
| `dnsResolutionOnTransparentProxyIPv6ExemptApps` | `array<string>` | Sensitive identifier | `standard` |
| `dnsResolutionOnTransparentProxyIPv6ExemptUrlCategories` | `array<string>` | Sensitive identifier | `standard` |
| `dnsResolutionOnTransparentProxyIPv6UrlCategories` | `array<string>` | Sensitive identifier | `standard` |
| `dnsResolutionOnTransparentProxyUrlCategories` | `array<string>` | Sensitive identifier | `standard` |
| `dnsResolutionOnTransparentProxyUrls` | `array<string>` | Sensitive identifier | `standard` |
| `domainFrontingBypassUrlCategories` | `array<string>` | Sensitive identifier | `standard` |
| `httpRangeHeaderRemoveUrlCategories` | `array<string>` | Sensitive identifier | `standard` |
| `kerberosBypassApps` | `array<string>` | Sensitive identifier | `standard` |
| `kerberosBypassUrlCategories` | `array<string>` | Sensitive identifier | `standard` |
| `kerberosBypassUrls` | `array<string>` | Sensitive identifier | `standard` |
| `preferSniOverConnHostApps` | `array<string>` | Sensitive identifier | `standard` |
| `sniDnsOptimizationBypassUrlCategories` | `array<string>` | Sensitive identifier | `standard` |
| `ecsObject` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `ecsObject.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `ecsObject.name` | `string` | Tenant configuration | `standard`, `share` |
| `ecsObject.externalId` | `string` | Sensitive identifier | `standard` |
| `ecsObject.extensions` |  | Secret | never (unstructured) |
<!-- END GENERATED FIELDS -->

This singleton settings page renders reviewed global ZIA advanced-setting
toggles; bypass app, URL and category lists are standard only.

## ZIA Advanced Threat Settings

Commands:

```sh
zscalerctl zia advanced-threat-settings show
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/advanced-threat-settings (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `activeXBlocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `activeXCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `adSpywareSitesBlocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `adSpywareSitesCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `alertForUnknownOrSuspiciousC2Traffic` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `anonymizerBlocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `anonymizerCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `bitTorrentBlocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `bitTorrentCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `blockCountriesCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `browserExploitsBlocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `browserExploitsCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `cmdCtlServerBlocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `cmdCtlServerCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `cmdCtlTrafficBlocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `cmdCtlTrafficCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `cryptoMiningBlocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `cryptoMiningCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `dgaDomainsBlocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `dgaDomainsCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `fileFormatVunerabilitesBlocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `fileFormatVunerabilitesCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `googleTalkBlocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `googleTalkCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `ircTunnellingBlocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `ircTunnellingCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `knownPhishingSitesBlocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `knownPhishingSitesCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `maliciousUrlsCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `malwareSitesBlocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `malwareSitesCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `potentialMaliciousRequestsBlocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `potentialMaliciousRequestsCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `riskTolerance` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `riskToleranceCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `sshTunnellingBlocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `sshTunnellingCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `suspectAdwareSpywareSitesBlocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `suspectAdwareSpywareSitesCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `suspectedPhishingSitesBlocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `suspectedPhishingSitesCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `torBlocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `torCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `webspamBlocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `webspamCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `cookieStealingBlocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `cookieStealingPCAPEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `blockedCountries` | `array<string>` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

This singleton settings page renders reviewed advanced-threat policy toggles.
Country blocks render only in `standard`; secret-looking boolean control names
carry explicit catalog reasons.

## ZIA Mobile Threat Settings

Commands:

```sh
zscalerctl zia mobile-threat-settings show
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/mobile-threat-settings (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `blockAppsCommunicatingWithAdWebsites` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `blockAppsCommunicatingWithRemoteUnknownServers` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `blockAppsSendingDeviceIdentifier` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `blockAppsSendingLocationInfo` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `blockAppsSendingPersonallyIdentifiableInfo` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `blockAppsWithKnownVulnerabilities` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `blockAppsWithMaliciousActivity` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `blockAppsSendingUnencryptedUserCredentials` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

This singleton settings page renders reviewed mobile-threat protection toggles,
including a credential-exfiltration control modeled as boolean metadata rather
than credential material.

## ZIA Sandbox Settings

Commands:

```sh
zscalerctl zia sandbox-settings show
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/sandbox-settings (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `fileHashesToBeBlocked` | `array<string>` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

This singleton settings page renders reviewed sandbox settings. File hashes to
block are sensitive identifiers and render only in `standard`.

## ZIA End User Notification Settings

Commands:

```sh
zscalerctl zia end-user-notification-settings show
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/end-user-notification-settings (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `aupCustomFrequency` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `aupDayOffset` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `aupFrequency` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `cautionAgainAfter` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `cautionPerDomain` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `displayCompLogo` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `displayReason` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `notificationType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `securityReviewEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `securityReviewSubmitToSecurityCloud` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `urlCatReviewEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `urlCatReviewSubmitToSecurityCloud` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `webDlpReviewEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `webDlpReviewSubmitToSecurityCloud` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `displayCompName` | `boolean` | Tenant configuration | `standard`, `share` |
| `redirectUrl` | `string` | Sensitive identifier | `standard` |
| `securityReviewCustomLocation` | `string` | Sensitive identifier | `standard` |
| `supportEmail` | `string` | Sensitive identifier | `standard` |
| `supportPhone` | `string` | Sensitive identifier | `standard` |
| `urlCatReviewCustomLocation` | `string` | Sensitive identifier | `standard` |
| `webDlpReviewCustomLocation` | `string` | Sensitive identifier | `standard` |
| `orgPolicyLink` | `string` | Sensitive identifier | `standard` |
| `aupMessage` | `string` | Free text | `standard` |
| `cautionCustomText` | `string` | Free text | `standard` |
| `customText` | `string` | Free text | `standard` |
| `idpProxyNotificationText` | `string` | Free text | `standard` |
| `quarantineCustomNotificationText` | `string` | Free text | `standard` |
| `securityReviewText` | `string` | Free text | `standard` |
| `urlCatReviewText` | `string` | Free text | `standard` |
| `webDlpReviewText` | `string` | Free text | `standard` |
<!-- END GENERATED FIELDS -->

This singleton settings page renders reviewed notification toggles and public
display metadata. Custom message bodies and policy-review text are dropped as
free-form tenant content.

## ZIA Org Information

Commands:

```sh
zscalerctl zia org-information show
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/org-information (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `alertTimer` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `customerContactInherit` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `employeeCount` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `externalEmailPortal` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `internalCompany` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `language` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `legacyInsightsReportWasEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `logoMimeType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `timezone` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `cloudName` | `string` | Tenant configuration | `standard`, `share` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `addrLine1` | `string` | Sensitive identifier | `standard` |
| `addrLine2` | `string` | Sensitive identifier | `standard` |
| `city` | `string` | Sensitive identifier | `standard` |
| `country` | `string` | Sensitive identifier | `standard` |
| `domains` | `array<string>` | Sensitive identifier | `standard` |
| `geoLocation` | `string` | Sensitive identifier | `standard` |
| `hqLocation` | `string` | Sensitive identifier | `standard` |
| `orgId` | `integer` | Sensitive identifier | `standard` |
| `pdomain` | `string` | Sensitive identifier | `standard` |
| `state` | `string` | Sensitive identifier | `standard` |
| `zipcode` | `string` | Sensitive identifier | `standard` |
| `zpaTenantCloud` | `string` | Sensitive identifier | `standard` |
| `zpaTenantId` | `integer` | Sensitive identifier | `standard` |
| `primaryBillingContactcontactType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `primaryBusinessContactcontactType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `primaryTechnicalContactcontactType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `secondaryBillingContactcontactType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `secondaryBusinessContactcontactType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `secondaryTechnicalContactcontactType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `industryVertical` | `string` | Tenant configuration | `standard`, `share` |
| `execInsightsHref` | `string` | Sensitive identifier | `standard` |
| `primaryBillingContactAltPhone` | `string` | Sensitive identifier | `standard` |
| `primaryBillingContactEmail` | `string` | Sensitive identifier | `standard` |
| `primaryBillingContactInsightsHref` | `string` | Sensitive identifier | `standard` |
| `primaryBillingContactName` | `string` | Sensitive identifier | `standard` |
| `primaryBillingContactPhone` | `string` | Sensitive identifier | `standard` |
| `primaryBillingContactTitle` | `string` | Sensitive identifier | `standard` |
| `primaryBusinessContactAltPhone` | `string` | Sensitive identifier | `standard` |
| `primaryBusinessContactEmail` | `string` | Sensitive identifier | `standard` |
| `primaryBusinessContactInsightsHref` | `string` | Sensitive identifier | `standard` |
| `primaryBusinessContactName` | `string` | Sensitive identifier | `standard` |
| `primaryBusinessContactPhone` | `string` | Sensitive identifier | `standard` |
| `primaryBusinessContactTitle` | `string` | Sensitive identifier | `standard` |
| `primaryTechnicalContactAltPhone` | `string` | Sensitive identifier | `standard` |
| `primaryTechnicalContactEmail` | `string` | Sensitive identifier | `standard` |
| `primaryTechnicalContactInsightsHref` | `string` | Sensitive identifier | `standard` |
| `primaryTechnicalContactName` | `string` | Sensitive identifier | `standard` |
| `primaryTechnicalContactPhone` | `string` | Sensitive identifier | `standard` |
| `primaryTechnicalContactTitle` | `string` | Sensitive identifier | `standard` |
| `secondaryBillingContactAltPhone` | `string` | Sensitive identifier | `standard` |
| `secondaryBillingContactEmail` | `string` | Sensitive identifier | `standard` |
| `secondaryBillingContactInsightsHref` | `string` | Sensitive identifier | `standard` |
| `secondaryBillingContactName` | `string` | Sensitive identifier | `standard` |
| `secondaryBillingContactPhone` | `string` | Sensitive identifier | `standard` |
| `secondaryBillingContactTitle` | `string` | Sensitive identifier | `standard` |
| `secondaryBusinessContactAltPhone` | `string` | Sensitive identifier | `standard` |
| `secondaryBusinessContactEmail` | `string` | Sensitive identifier | `standard` |
| `secondaryBusinessContactInsightsHref` | `string` | Sensitive identifier | `standard` |
| `secondaryBusinessContactName` | `string` | Sensitive identifier | `standard` |
| `secondaryBusinessContactPhone` | `string` | Sensitive identifier | `standard` |
| `secondaryBusinessContactTitle` | `string` | Sensitive identifier | `standard` |
| `secondaryTechnicalContactAltPhone` | `string` | Sensitive identifier | `standard` |
| `secondaryTechnicalContactEmail` | `string` | Sensitive identifier | `standard` |
| `secondaryTechnicalContactInsightsHref` | `string` | Sensitive identifier | `standard` |
| `secondaryTechnicalContactName` | `string` | Sensitive identifier | `standard` |
| `secondaryTechnicalContactPhone` | `string` | Sensitive identifier | `standard` |
| `secondaryTechnicalContactTitle` | `string` | Sensitive identifier | `standard` |
| `logoBase64Data` |  | Secret | never (opaque blob) |
<!-- END GENERATED FIELDS -->

This singleton settings page renders narrow organization metadata. Addresses,
domains, and tenant identifiers render only in `standard`; contact details and
logo payloads are dropped.

## ZIA Atp Malware Policy

Commands:

```sh
zscalerctl zia atp-malware-policy show
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/atp-malware-policy (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `blockPasswordProtectedArchiveFiles` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `blockUnscannableFiles` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

This singleton settings page renders reviewed ATP malware policy toggles. The
password-protected archive control is modeled as boolean metadata, not password
material.

## ZIA Atp Malware Settings

Commands:

```sh
zscalerctl zia atp-malware-settings show
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/atp-malware-settings (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `adwareBlocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `adwareCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `ransomwareBlocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `ransomwareCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `remoteAccessToolBlocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `remoteAccessToolCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `spywareBlocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `spywareCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `trojanBlocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `trojanCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `unwantedApplicationsBlocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `unwantedApplicationsCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `virusBlocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `virusCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `wormBlocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `wormCapture` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

This singleton settings page renders reviewed ATP malware family block/capture
toggles.

## ZIA Atp Malware Inspection

Commands:

```sh
zscalerctl zia atp-malware-inspection show
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/atp-malware-inspection (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `inspectInbound` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `inspectOutbound` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

This singleton settings page renders reviewed inbound and outbound malware
inspection toggles.

## ZIA Atp Malware Protocols

Commands:

```sh
zscalerctl zia atp-malware-protocols show
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/atp-malware-protocols (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `inspectFtp` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `inspectFtpOverHttp` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `inspectHttp` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

This singleton settings page renders reviewed protocol inspection toggles.

## ZIA Malicious Urls

Commands:

```sh
zscalerctl zia malicious-urls show
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/malicious-urls (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `maliciousUrls` | `array<string>` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

This singleton settings page renders the tenant malicious-URL list only in
`standard`.

## ZIA Security Exceptions

Commands:

```sh
zscalerctl zia security-exceptions show
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/security-exceptions (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `bypassUrls` | `array<string>` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

This singleton settings page renders the tenant security-exception URL list
only in `standard`.

## ZIA Url Allow List

Commands:

```sh
zscalerctl zia url-allow-list show
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/url-allow-list (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `whitelistUrls` | `array<string>` | Sensitive identifier | `standard` |
| `blacklistUrls` | `array<string>` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

This singleton settings page renders the global URL allow list only in
`standard`. The deny-list field is explicitly modeled and dropped.

## ZIA Url Deny List

Commands:

```sh
zscalerctl zia url-deny-list show
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/url-deny-list (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `blacklistUrls` | `array<string>` | Sensitive identifier | `standard` |
| `whitelistUrls` | `array<string>` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

This singleton settings page renders the global URL deny list only in
`standard`. The allow-list field is explicitly modeled and dropped.

## ZIA Dns Application Groups

Commands:

```sh
zscalerctl zia dns-application-groups list
zscalerctl zia dns-application-groups get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/dns-application-groups (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `dnsApplications` | `array<string>` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Group identity.
- `name`, `dnsApplications`: Group name and the DNS applications it collects.

## ZIA Endpoint Dlp Rules

Commands:

```sh
zscalerctl zia endpoint-dlp-rules list
zscalerctl zia endpoint-dlp-rules get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/endpoint-dlp-rules (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `state` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `order` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `rank` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `parentRule` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
| `action` | `string` | Tenant configuration | `standard`, `share` |
| `severity` | `string` | Tenant configuration | `standard`, `share` |
| `fileTypes` | `array<string>` | Tenant configuration | `standard`, `share` |
| `dataTransferMethod` | `string` | Tenant configuration | `standard`, `share` |
| `deviceTrustLevels` | `array<string>` | Tenant configuration | `standard`, `share` |
| `networkType` | `string` | Tenant configuration | `standard`, `share` |
| `minSize` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `withoutContentInspection` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `eunEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `eunTemplateId` | `integer` | Sensitive identifier | `standard` |
| `ucTemplateId` | `integer` | Sensitive identifier | `standard` |
| `externalAuditorEmail` | `string` | Sensitive identifier | `standard` |
| `userRiskScoreLevels` | `array<string>` | Tenant configuration | `standard` |
| `auditor` | `object` | Tenant configuration | `standard` |
| `auditor.id` | `integer` | Operational metadata | `standard` |
| `auditor.name` | `string` | Tenant configuration | `standard` |
| `notificationTemplate` | `object` | Tenant configuration | `standard` |
| `notificationTemplate.id` | `integer` | Operational metadata | `standard` |
| `notificationTemplate.name` | `string` | Tenant configuration | `standard` |
| `receiver` | `object` | Tenant configuration | `standard` |
| `receiver.id` | `integer` | Operational metadata | `standard` |
| `receiver.name` | `string` | Tenant configuration | `standard` |
| `receiver.type` | `string` | Operational metadata | `standard` |
| `receiver.tenant` | `object` | Tenant configuration | `standard` |
| `receiver.tenant.id` | `integer` | Operational metadata | `standard` |
| `receiver.tenant.name` | `string` | Tenant configuration | `standard` |
| `receiver.tenant.extensions` |  | Secret | never (unstructured) |
| `resources` | `array<object>` | Tenant configuration | `standard`, `share` |
| `resources.id` | `integer` | Operational metadata | `standard`, `share` |
| `resources.name` | `string` | Tenant configuration | `standard` |
| `resources.extensions` |  | Secret | never (unstructured) |
| `resourceGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `resourceGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `resourceGroups.name` | `string` | Tenant configuration | `standard` |
| `resourceGroups.extensions` |  | Secret | never (unstructured) |
| `labels` | `array<object>` | Tenant configuration | `standard`, `share` |
| `labels.id` | `integer` | Operational metadata | `standard`, `share` |
| `labels.name` | `string` | Tenant configuration | `standard` |
| `labels.extensions` |  | Secret | never (unstructured) |
| `dlpEngines` | `array<object>` | Tenant configuration | `standard`, `share` |
| `dlpEngines.id` | `integer` | Operational metadata | `standard`, `share` |
| `dlpEngines.name` | `string` | Tenant configuration | `standard` |
| `dlpEngines.extensions` |  | Secret | never (unstructured) |
| `users` | `array<object>` | Tenant configuration | `standard`, `share` |
| `users.id` | `integer` | Operational metadata | `standard`, `share` |
| `users.name` | `string` | Tenant configuration | `standard` |
| `users.extensions` |  | Secret | never (unstructured) |
| `groups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `groups.id` | `integer` | Operational metadata | `standard`, `share` |
| `groups.name` | `string` | Tenant configuration | `standard` |
| `groups.extensions` |  | Secret | never (unstructured) |
| `departments` | `array<object>` | Tenant configuration | `standard`, `share` |
| `departments.id` | `integer` | Operational metadata | `standard`, `share` |
| `departments.name` | `string` | Tenant configuration | `standard` |
| `departments.extensions` |  | Secret | never (unstructured) |
| `devices` | `array<object>` | Tenant configuration | `standard`, `share` |
| `devices.id` | `integer` | Operational metadata | `standard`, `share` |
| `devices.name` | `string` | Tenant configuration | `standard` |
| `devices.extensions` |  | Secret | never (unstructured) |
| `deviceGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `deviceGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `deviceGroups.name` | `string` | Tenant configuration | `standard` |
| `deviceGroups.extensions` |  | Secret | never (unstructured) |
| `endPointApplications` | `array<object>` | Tenant configuration | `standard`, `share` |
| `endPointApplications.resourceId` | `integer` | Operational metadata | `standard`, `share` |
| `endPointApplications.applicationName` | `string` | Tenant configuration | `standard`, `share` |
| `endPointApplications.description` | `string` | Free text | `standard` |
| `endPointApplications.osType` | `string` | Operational metadata | `standard`, `share` |
| `endPointApplications.applicationType` | `string` | Operational metadata | `standard`, `share` |
| `endPointApplications.bundleID` | `string` | Sensitive identifier | `standard` |
| `endPointApplications.filename` | `string` | Sensitive identifier | `standard` |
| `endPointApplications.originalFileName` | `string` | Sensitive identifier | `standard` |
| `endPointApplications.digitallySigned` | `boolean` | Operational metadata | `standard`, `share` |
| `endPointApplications.modUId` | `integer` | Operational metadata | `standard`, `share` |
| `endPointApplications.lastModifiedTime` | `integer` | Operational metadata | `standard`, `share` |
| `endPointApplications.zappId` | `string` | Operational metadata | `standard`, `share` |
| `endPointApplications.deleted` | `boolean` | Operational metadata | `standard`, `share` |
| `endPointApplications.version` | `object` | Operational metadata | `standard`, `share` |
| `endPointApplications.version.version` | `string` | Operational metadata | `standard`, `share` |
| `endPointApplications.version.z_ver_id_md32` | `integer` | Operational metadata | `standard`, `share` |
| `endPointApplications.version.threat_type` | `integer` | Operational metadata | `standard`, `share` |
| `endPointApplications.version.threat_level` | `string` | Operational metadata | `standard`, `share` |
| `endPointApplications.version.code_signing_certificate_status` | `integer` | Operational metadata | `standard`, `share` |
| `endPointApplications.version.threatLevelUpdated` | `boolean` | Operational metadata | `standard`, `share` |
| `endPointApplications.version.bundleID` | `string` | Sensitive identifier | `standard` |
| `endPointApplications.versions` | `object` | Operational metadata | `standard`, `share` |
| `endPointApplications.versions.version` | `string` | Operational metadata | `standard`, `share` |
| `endPointApplications.versions.z_ver_id_md32` | `integer` | Operational metadata | `standard`, `share` |
| `endPointApplications.versions.threat_type` | `integer` | Operational metadata | `standard`, `share` |
| `endPointApplications.versions.threat_level` | `string` | Operational metadata | `standard`, `share` |
| `endPointApplications.versions.code_signing_certificate_status` | `integer` | Operational metadata | `standard`, `share` |
| `endPointApplications.versions.threatLevelUpdated` | `boolean` | Operational metadata | `standard`, `share` |
| `endPointApplications.versions.bundleID` | `string` | Sensitive identifier | `standard` |
| `endPointApplicationGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `endPointApplicationGroups.groupId` | `integer` | Operational metadata | `standard`, `share` |
| `endPointApplicationGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `endPointApplicationGroups.description` | `string` | Free text | `standard` |
| `endPointApplicationGroups.modUId` | `integer` | Operational metadata | `standard`, `share` |
| `endPointApplicationGroups.lastModifiedTime` | `integer` | Operational metadata | `standard`, `share` |
| `endPointApplicationGroups.endPointApplications` | `array<object>` | Tenant configuration | `standard`, `share` |
| `endPointApplicationGroups.endPointApplications.resourceId` | `integer` | Operational metadata | `standard`, `share` |
| `endPointApplicationGroups.endPointApplications.applicationName` | `string` | Tenant configuration | `standard`, `share` |
| `endPointApplicationGroups.endPointApplications.description` | `string` | Free text | `standard` |
| `endPointApplicationGroups.endPointApplications.osType` | `string` | Operational metadata | `standard`, `share` |
| `endPointApplicationGroups.endPointApplications.applicationType` | `string` | Operational metadata | `standard`, `share` |
| `endPointApplicationGroups.endPointApplications.bundleID` | `string` | Sensitive identifier | `standard` |
| `endPointApplicationGroups.endPointApplications.filename` | `string` | Sensitive identifier | `standard` |
| `endPointApplicationGroups.endPointApplications.originalFileName` | `string` | Sensitive identifier | `standard` |
| `endPointApplicationGroups.endPointApplications.digitallySigned` | `boolean` | Operational metadata | `standard`, `share` |
| `endPointApplicationGroups.endPointApplications.modUId` | `integer` | Operational metadata | `standard`, `share` |
| `endPointApplicationGroups.endPointApplications.lastModifiedTime` | `integer` | Operational metadata | `standard`, `share` |
| `endPointApplicationGroups.endPointApplications.zappId` | `string` | Operational metadata | `standard`, `share` |
| `endPointApplicationGroups.endPointApplications.deleted` | `boolean` | Operational metadata | `standard`, `share` |
| `endPointApplicationGroups.endPointApplications.version` | `object` | Operational metadata | `standard`, `share` |
| `endPointApplicationGroups.endPointApplications.version.version` | `string` | Operational metadata | `standard`, `share` |
| `endPointApplicationGroups.endPointApplications.version.z_ver_id_md32` | `integer` | Operational metadata | `standard`, `share` |
| `endPointApplicationGroups.endPointApplications.version.threat_type` | `integer` | Operational metadata | `standard`, `share` |
| `endPointApplicationGroups.endPointApplications.version.threat_level` | `string` | Operational metadata | `standard`, `share` |
| `endPointApplicationGroups.endPointApplications.version.code_signing_certificate_status` | `integer` | Operational metadata | `standard`, `share` |
| `endPointApplicationGroups.endPointApplications.version.threatLevelUpdated` | `boolean` | Operational metadata | `standard`, `share` |
| `endPointApplicationGroups.endPointApplications.version.bundleID` | `string` | Sensitive identifier | `standard` |
| `endPointApplicationGroups.endPointApplications.versions` | `object` | Operational metadata | `standard`, `share` |
| `endPointApplicationGroups.endPointApplications.versions.version` | `string` | Operational metadata | `standard`, `share` |
| `endPointApplicationGroups.endPointApplications.versions.z_ver_id_md32` | `integer` | Operational metadata | `standard`, `share` |
| `endPointApplicationGroups.endPointApplications.versions.threat_type` | `integer` | Operational metadata | `standard`, `share` |
| `endPointApplicationGroups.endPointApplications.versions.threat_level` | `string` | Operational metadata | `standard`, `share` |
| `endPointApplicationGroups.endPointApplications.versions.code_signing_certificate_status` | `integer` | Operational metadata | `standard`, `share` |
| `endPointApplicationGroups.endPointApplications.versions.threatLevelUpdated` | `boolean` | Operational metadata | `standard`, `share` |
| `endPointApplicationGroups.endPointApplications.versions.bundleID` | `string` | Sensitive identifier | `standard` |
| `subRules` |  | Secret | never (unstructured) |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `state`, `order`, `rank`, `parentRule`, `lastModifiedTime`, `minSize`, `withoutContentInspection`, `eunEnabled`: Rule identity, ordering, state and scan flags.
- `lastModifiedBy`: Admin reference: id in every mode, name standard-only.
- `name`, `action`, `severity`, `fileTypes`, `dataTransferMethod`, `deviceTrustLevels`, `networkType`: Rule metadata and match criteria.
- `endPointApplications`, `endPointApplicationGroups`: Endpoint applications the rule matches: application and group names, platform and version records render in `share`; `bundleID`, `filename`, `originalFileName` and descriptions are standard only (in-house binaries carry the company's domain and tool names).
- `eunTemplateId`, `ucTemplateId`, `externalAuditorEmail`: Local-only template and auditor identifiers.
- `auditor`, `notificationTemplate`, `receiver`, `userRiskScoreLevels`: Local-only references.
- `resources`, `resourceGroups`, `labels`, `dlpEngines`, `users`, `groups`, `departments`, `devices`, `deviceGroups`: Scope references (`users`, `groups`, `dlpEngines`, ...) render their ids in `share`; their names render in `standard` only.

## ZIA Email Dlp Rules

Commands:

```sh
zscalerctl zia email-dlp-rules list
zscalerctl zia email-dlp-rules get <id>
```

Outbound email DLP rules (`/emailDlpRules`).

Fields:

<!-- BEGIN GENERATED FIELDS: zia/email-dlp-rules (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `state` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `order` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `parentRule` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy` | `object` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
| `action` | `string` | Tenant configuration | `standard`, `share` |
| `severity` | `string` | Tenant configuration | `standard`, `share` |
| `fileTypes` | `array<string>` | Tenant configuration | `standard`, `share` |
| `contentLocations` | `array<string>` | Tenant configuration | `standard`, `share` |
| `minSize` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `withoutContentInspection` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `externalAuditorEmail` | `string` | Sensitive identifier | `standard` |
| `customHeader` | `string` | Free text | `standard` |
| `userRiskScoreLevels` | `array<string>` | Tenant configuration | `standard` |
| `auditor` | `object` | Tenant configuration | `standard` |
| `auditor.id` | `integer` | Operational metadata | `standard` |
| `auditor.name` | `string` | Tenant configuration | `standard` |
| `notificationTemplate` | `object` | Tenant configuration | `standard` |
| `notificationTemplate.id` | `integer` | Operational metadata | `standard` |
| `notificationTemplate.name` | `string` | Tenant configuration | `standard` |
| `receiver` | `object` | Tenant configuration | `standard` |
| `receiver.id` | `integer` | Operational metadata | `standard` |
| `receiver.name` | `string` | Tenant configuration | `standard` |
| `receiver.type` | `string` | Operational metadata | `standard` |
| `receiver.tenant` | `object` | Tenant configuration | `standard` |
| `receiver.tenant.id` | `integer` | Operational metadata | `standard` |
| `receiver.tenant.name` | `string` | Tenant configuration | `standard` |
| `receiver.tenant.extensions` |  | Secret | never (unstructured) |
| `labels` | `array<object>` | Tenant configuration | `standard`, `share` |
| `labels.id` | `integer` | Operational metadata | `standard`, `share` |
| `labels.name` | `string` | Tenant configuration | `standard` |
| `labels.extensions` |  | Secret | never (unstructured) |
| `dlpEngines` | `array<object>` | Tenant configuration | `standard`, `share` |
| `dlpEngines.id` | `integer` | Operational metadata | `standard`, `share` |
| `dlpEngines.name` | `string` | Tenant configuration | `standard` |
| `dlpEngines.extensions` |  | Secret | never (unstructured) |
| `timeWindows` | `array<object>` | Tenant configuration | `standard`, `share` |
| `timeWindows.id` | `integer` | Operational metadata | `standard`, `share` |
| `timeWindows.name` | `string` | Tenant configuration | `standard` |
| `timeWindows.extensions` |  | Secret | never (unstructured) |
| `users` | `array<object>` | Tenant configuration | `standard`, `share` |
| `users.id` | `integer` | Operational metadata | `standard`, `share` |
| `users.name` | `string` | Tenant configuration | `standard` |
| `users.extensions` |  | Secret | never (unstructured) |
| `groups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `groups.id` | `integer` | Operational metadata | `standard`, `share` |
| `groups.name` | `string` | Tenant configuration | `standard` |
| `groups.extensions` |  | Secret | never (unstructured) |
| `departments` | `array<object>` | Tenant configuration | `standard`, `share` |
| `departments.id` | `integer` | Operational metadata | `standard`, `share` |
| `departments.name` | `string` | Tenant configuration | `standard` |
| `departments.extensions` |  | Secret | never (unstructured) |
| `excludedUsers` | `array<object>` | Tenant configuration | `standard`, `share` |
| `excludedUsers.id` | `integer` | Operational metadata | `standard`, `share` |
| `excludedUsers.name` | `string` | Tenant configuration | `standard` |
| `excludedUsers.extensions` |  | Secret | never (unstructured) |
| `excludedGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `excludedGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `excludedGroups.name` | `string` | Tenant configuration | `standard` |
| `excludedGroups.extensions` |  | Secret | never (unstructured) |
| `excludedDepartments` | `array<object>` | Tenant configuration | `standard`, `share` |
| `excludedDepartments.id` | `integer` | Operational metadata | `standard`, `share` |
| `excludedDepartments.name` | `string` | Tenant configuration | `standard` |
| `excludedDepartments.extensions` |  | Secret | never (unstructured) |
| `includedDomainProfiles` | `array<object>` | Tenant configuration | `standard`, `share` |
| `includedDomainProfiles.id` | `integer` | Operational metadata | `standard`, `share` |
| `includedDomainProfiles.name` | `string` | Tenant configuration | `standard` |
| `includedDomainProfiles.extensions` |  | Secret | never (unstructured) |
| `emailRecipientProfiles` | `array<object>` | Tenant configuration | `standard`, `share` |
| `emailRecipientProfiles.id` | `integer` | Operational metadata | `standard`, `share` |
| `emailRecipientProfiles.name` | `string` | Tenant configuration | `standard` |
| `emailRecipientProfiles.extensions` |  | Secret | never (unstructured) |
| `emailTenants` | `array<object>` | Tenant configuration | `standard`, `share` |
| `emailTenants.id` | `integer` | Operational metadata | `standard`, `share` |
| `emailTenants.name` | `string` | Tenant configuration | `standard` |
| `emailTenants.extensions` |  | Secret | never (unstructured) |
| `subRules` |  | Secret | never (unstructured) |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `state`, `order`, `parentRule`, `lastModifiedTime`, `minSize`, `withoutContentInspection`: Rule identity, ordering, state and scan flags.
- `lastModifiedBy`: Admin reference: id in every mode, name standard-only.
- `name`, `action`, `severity`, `fileTypes`, `contentLocations`: Rule metadata and match criteria.
- `externalAuditorEmail`: Local-only auditor address.
- `auditor`, `notificationTemplate`, `receiver`, `userRiskScoreLevels`: Local-only references.
- `labels`, `dlpEngines`, `timeWindows`, `users`, `groups`, `departments`, `excludedUsers`, `excludedGroups`, `excludedDepartments`, `includedDomainProfiles`, `emailRecipientProfiles`, `emailTenants`: Scope references (`users`, `groups`, `dlpEngines`, ...) render their ids in `share`; their names render in `standard` only.

## ZIA Alert Definitions

Commands:

```sh
zscalerctl zia alert-definitions list
zscalerctl zia alert-definitions get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/alert-definitions (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `alertName` | `string` | Tenant configuration | `standard`, `share` |
| `status` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `occurrence` | `string` | Tenant configuration | `standard`, `share` |
| `trafficChangePercent` | `integer` | Tenant configuration | `standard`, `share` |
| `interval` | `string` | Tenant configuration | `standard`, `share` |
| `scope` | `string` | Tenant configuration | `standard`, `share` |
| `severity` | `string` | Tenant configuration | `standard`, `share` |
| `entity` | `object` | Tenant configuration | `standard`, `share` |
| `entity.id` | `integer` | Operational metadata | `standard`, `share` |
| `entity.name` | `string` | Tenant configuration | `standard` |
| `entity.extensions` |  | Secret | never (unstructured) |
| `comments` | `string` | Free text | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `status`: Definition identity and state.
- `alertName`, `occurrence`, `trafficChangePercent`, `interval`, `scope`, `severity`: What the alert watches and when it fires.
- `entity`: The watched entity: id in `share`, name standard-only (it can be a user).

## ZIA Http Header Profiles

Commands:

```sh
zscalerctl zia http-header-profiles list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/http-header-profiles (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `slotId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `deleted` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `profileReadyForUse` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `httpHeaderProfileCriteria` | `array<object>` | Tenant configuration | `standard`, `share` |
| `httpHeaderProfileCriteria.id` | `integer` | Operational metadata | `standard`, `share` |
| `httpHeaderProfileCriteria.header` | `string` | Tenant configuration | `standard`, `share` |
| `httpHeaderProfileCriteria.operator` | `string` | Operational metadata | `standard`, `share` |
| `httpHeaderProfileCriteria.categoryBitmap` | `array<string>` | Tenant configuration | `standard`, `share` |
| `httpHeaderProfileCriteria.cloudAppBitmap` | `array<string>` | Tenant configuration | `standard`, `share` |
| `httpHeaderProfileCriteria.userAgent` | `string` | Tenant configuration | `standard`, `share` |
| `httpHeaderProfileCriteria.userAgentBitmap` | `string` | Tenant configuration | `standard`, `share` |
| `httpHeaderProfileCriteria.userAgentVersion` | `string` | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `slotId`, `deleted`, `profileReadyForUse`: Profile identity and state.
- `name`, `httpHeaderProfileCriteria`: Profile name and its header match criteria.

## ZIA Http Header Action Profiles

Commands:

```sh
zscalerctl zia http-header-action-profiles list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/http-header-action-profiles (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `slotId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `deleted` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `profileReadyForUse` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `httpHeaderActionProfileKeys` | `array<object>` | Tenant configuration | `standard`, `share` |
| `httpHeaderActionProfileKeys.id` | `integer` | Operational metadata | `standard`, `share` |
| `httpHeaderActionProfileKeys.key` | `string` | Tenant configuration | `standard`, `share` |
| `httpHeaderActionProfileKeys.value` |  | Secret | never (credential) |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `slotId`, `deleted`, `profileReadyForUse`: Profile identity and state.
- `name`, `httpHeaderActionProfileKeys`: Profile name and the header names it sets.

## ZIA Adaptive Access Profiles

Commands:

```sh
zscalerctl zia adaptive-access-profiles list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/adaptive-access-profiles (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `type` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `aapIndex` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `iamAapId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `deleted` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `type`, `aapIndex`, `iamAapId`, `deleted`: Profile identity; `iamAapId` is the id access policies use.
- `name`: Profile name.

## ZIA Dlp Global Options

Commands:

```sh
zscalerctl zia dlp-global-options show
```

This singleton settings page renders the web DLP global options.

Fields:

<!-- BEGIN GENERATED FIELDS: zia/dlp-global-options (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `applications` | `array<string>` | Tenant configuration | `standard`, `share` |
| `urls` | `array<string>` | Sensitive identifier | `standard` |
| `urlCategories` | `array<object>` | Tenant configuration | `standard`, `share` |
| `urlCategories.id` | `integer` | Operational metadata | `standard`, `share` |
| `urlCategories.name` | `string` | Tenant configuration | `standard` |
| `urlCategories.extensions` |  | Secret | never (unstructured) |
| `httpGetCustomUrlCategories` | `array<string>` | Tenant configuration | `standard`, `share` |
| `exemptUrlEncodedData` | `boolean` | Tenant configuration | `standard`, `share` |
| `enableNpkEdmTemplates` | `boolean` | Tenant configuration | `standard`, `share` |
| `enableNpkEdmTemplatesForOrg` | `boolean` | Tenant configuration | `standard`, `share` |
| `enableInlineDlpOcr` | `boolean` | Tenant configuration | `standard`, `share` |
| `enableCasbOcr` | `boolean` | Tenant configuration | `standard`, `share` |
| `enableEmailDlpOcr` | `boolean` | Tenant configuration | `standard`, `share` |
| `enableEvaluateAllDlpRules` | `boolean` | Tenant configuration | `standard`, `share` |
| `enableEdmPopularFormat` | `boolean` | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `applications`, `httpGetCustomUrlCategories`, `exemptUrlEncodedData`, `enableNpkEdmTemplates`, `enableNpkEdmTemplatesForOrg`, `enableInlineDlpOcr`, `enableCasbOcr`, `enableEmailDlpOcr`, `enableEvaluateAllDlpRules`, `enableEdmPopularFormat`: Web DLP exemptions and global switches.
- `urls`: Local-only exempted URLs.
- `urlCategories`: Exempted categories: ids in `share`, names in `standard` only, like `dlp-web-rules`.

The paginated ZPA list resources use a shared fail-closed reader, and the
ZIA, ZTW and ZCC list readers are bounded the same way.

Paginated readers fail closed when pages repeat or a record identifier appears
twice. The one exception is an endpoint that ignores pagination: when page 2
repeats the records of a short page 1, in any order, and a smaller page size still
returns every record, page 1 is accepted as the list. A tenant change during a long list walk can shift records across page
boundaries and trigger that check; the command then exits with an error and
returns no partial list. Retrying the command is the remedy.

## ZIA Registered Devices

Commands:

```sh
zscalerctl zia registered-devices list
zscalerctl zia registered-devices get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/registered-devices (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Sensitive identifier | `standard` |
| `active` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `deleted` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `rooted` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `os` | `string` | Tenant configuration | `standard`, `share` |
| `version` | `string` | Tenant configuration | `standard`, `share` |
| `vendor` | `string` | Tenant configuration | `standard`, `share` |
| `model` | `string` | Sensitive identifier | `standard` |
| `locale` | `string` | Tenant configuration | `standard`, `share` |
| `hostname` | `string` | Sensitive identifier | `standard` |
| `udid` | `string` | Sensitive identifier | `standard` |
| `hardwareId` | `string` | Sensitive identifier | `standard` |
| `macAddress` | `string` | Sensitive identifier | `standard` |
| `user` | `object` | Sensitive identifier | `standard` |
| `user.id` | `integer` | Operational metadata | `standard` |
| `user.name` | `string` | Sensitive identifier | `standard` |
| `user.extensions` |  | Secret | never (unstructured) |
| `firstRegistrationTimestamp` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastRegistrationTimestamp` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `unRegistrationTimestamp` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `active`, `deleted`, `rooted`, `firstRegistrationTimestamp`, `lastRegistrationTimestamp`, `unRegistrationTimestamp`: Registration state and timestamps; booleans render `false` when false, and a timestamp of `0` means unset (e.g. a device never unregistered).
- `os`, `version`, `vendor`, `locale`: Platform details.
- `name`, `model`, `hostname`, `udid`, `hardwareId`, `macAddress`: Device and hardware identifiers.
- `user`: The owning user (`id` references `zia/users`); standard only, like `zia/devices` `ownerUserId`.

The `/devices` view of the devices `zia/devices` reads from `/deviceGroups/devices`, with registration state and hardware identifiers. It sends no `includeCbiDevices` or `valid` filter, so the server's defaults decide whether Cloud Browser Isolation or invalid devices are included, as with the SDK's `GetAll`.

## ZIA Integration Partners

Commands:

```sh
zscalerctl zia integration-partners list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/integration-partners (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `type` | `string` | Tenant configuration | `standard`, `share` |
| `disabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `disabled`: Partner integration id and state.
- `name`, `type`: Partner integration name and type.

The route answers an array, though the SDK's `GetPartnerIntegrations` decodes one object; it is one unpaged read. CrowdStrike device and URL lookups are the gap `zia/partner-device-lookups`.

## ZIA Smpc Instances

Commands:

```sh
zscalerctl zia smpc-instances list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zia/smpc-instances (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `datacenterName` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `city` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `countryCode` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zscmIsolationEnabled` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `zscmIsolationEnabledApiFlag` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `connectionStatus` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `dcIsolationStatus` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `nodes` | `array<object>` | Operational metadata | `standard`, `share`, `paranoid` |
| `nodes.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `nodes.nodeName` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `nodes.instanceId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `nodes.adminIsolationEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `nodes.connectionState` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `nodes.lastTimestamp` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `datacenterName`, `city`, `countryCode`, `zscmIsolationEnabled`, `zscmIsolationEnabledApiFlag`, `connectionStatus`, `dcIsolationStatus`, `nodes`: Zscaler data-center instances and their nodes' isolation and connection state; integers render `0` when unset. A response without a `data` list fails the read.

## ZPA Version Profiles

Commands:

```sh
zscalerctl zpa version-profiles list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/version-profiles (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `customerId` | `string` | Sensitive identifier | `standard` |
| `customScopeCustomerIds` | `array<object>` | Sensitive identifier | `standard` |
| `customScopeCustomerIds.name` | `string` | Sensitive identifier | `standard` |
| `customScopeCustomerIds.customerId` | `string` | Sensitive identifier | `standard` |
| `customScopeCustomerIds.excludeConstellation` | `boolean` | Operational metadata | `standard` |
| `customScopeCustomerIds.isPartner` | `boolean` | Operational metadata | `standard` |
| `customScopeRequestCustomerIds` | `object` | Sensitive identifier | `standard` |
| `customScopeRequestCustomerIds.addCustomerIds` | `string` | Sensitive identifier | `standard` |
| `customScopeRequestCustomerIds.deleteCustomerIds` | `string` | Sensitive identifier | `standard` |
| `versions` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `versions.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `versions.customerId` | `string` | Sensitive identifier | `standard` |
| `versions.creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `versions.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `versions.modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `versions.platform` | `string` | Tenant configuration | `standard`, `share` |
| `versions.restartAfterUptimeInDays` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `versions.role` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `versions.version` | `string` | Tenant configuration | `standard`, `share` |
| `versions.version_profile_gid` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `visibilityScope` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `upgradePriority` | `string` | Tenant configuration | `standard`, `share` |
| `numberOfAssistants` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `numberOfCustomers` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `numberOfPrivateBrokers` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `numberOfSiteControllers` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `numberOfUpdatedAssistants` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `numberOfUpdatedPrivateBrokers` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `numberOfUpdatedSiteControllers` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Version profile identifier.
- `creationTime`, `modifiedBy`, `modifiedTime`, `visibilityScope`, `numberOfAssistants`, `numberOfCustomers`, `numberOfPrivateBrokers`, `numberOfSiteControllers`, `numberOfUpdatedAssistants`, `numberOfUpdatedPrivateBrokers`, `numberOfUpdatedSiteControllers`: Profile metadata and broker/assistant counts.
- `upgradePriority`: Upgrade priority setting.

## ZPA Client Types

Commands:

```sh
zscalerctl zpa client-types list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/client-types (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `zpn_client_type_exporter` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpn_client_type_exporter_noauth` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpn_client_type_browser_isolation` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpn_client_type_machine_tunnel` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpn_client_type_ip_anchoring` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpn_client_type_edge_connector` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpn_client_type_zapp` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpn_client_type_slogger` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpn_client_type_branch_connector` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpn_client_type_zapp_partner` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpn_client_type_vdi` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpn_client_type_zia_inspection` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `zpn_client_type_exporter`, `zpn_client_type_exporter_noauth`, `zpn_client_type_browser_isolation`, `zpn_client_type_machine_tunnel`, `zpn_client_type_ip_anchoring`, `zpn_client_type_edge_connector`, `zpn_client_type_zapp`, `zpn_client_type_slogger`, `zpn_client_type_branch_connector`, `zpn_client_type_zapp_partner`, `zpn_client_type_vdi`, `zpn_client_type_zia_inspection`: Client-type reference identifiers.

## ZPA Platforms

Commands:

```sh
zscalerctl zpa platforms list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/platforms (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `linux` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `android` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `windows` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `ios` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `mac` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `linux`, `android`, `windows`, `ios`, `mac`: Supported client platform reference values.

## ZPA Microtenants

Commands:

```sh
zscalerctl zpa microtenants list
zscalerctl zpa microtenants get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/microtenants (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `criteriaAttribute` | `string` | Tenant configuration | `standard`, `share` |
| `criteriaAttributeValues` | `array<string>` | Sensitive identifier | `standard` |
| `privilegedApprovalsEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `operator` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `priority` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `roles` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `roles.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `roles.name` | `string` | Tenant configuration | `standard`, `share` |
| `roles.customRole` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `user` | `object` | Sensitive identifier | `standard` |
| `user.id` | `string` | Operational metadata | `standard` |
| `user.name` | `string` | Sensitive identifier | `standard` |
| `user.username` | `string` | Sensitive identifier | `standard` |
| `user.displayName` | `string` | Sensitive identifier | `standard` |
| `user.email` | `string` | Sensitive identifier | `standard` |
| `user.phoneNumber` | `string` | Sensitive identifier | `standard` |
| `user.customerId` | `string` | Sensitive identifier | `standard` |
| `user.enabled` | `boolean` | Operational metadata | `standard` |
| `user.isEnabled` | `boolean` | Operational metadata | `standard` |
| `user.isLocked` | `boolean` | Operational metadata | `standard` |
| `user.localLoginDisabled` | `boolean` | Operational metadata | `standard` |
| `user.oneIdentityUser` | `boolean` | Operational metadata | `standard` |
| `user.forcePwdChange` | `boolean` | Operational metadata | `standard` |
| `user.twoFactorAuthEnabled` | `boolean` | Operational metadata | `standard` |
| `user.twoFactorAuthType` | `string` | Operational metadata | `standard` |
| `user.roleId` | `string` | Operational metadata | `standard` |
| `user.microtenantId` | `string` | Operational metadata | `standard` |
| `user.microtenantName` | `string` | Tenant configuration | `standard` |
| `user.languageCode` | `string` | Operational metadata | `standard` |
| `user.timezone` | `string` | Operational metadata | `standard` |
| `user.creationTime` | `string` | Operational metadata | `standard` |
| `user.modifiedBy` | `string` | Operational metadata | `standard` |
| `user.modifiedTime` | `string` | Operational metadata | `standard` |
| `user.password` |  | Secret | never (credential) |
| `user.tmpPassword` |  | Secret | never (credential) |
| `user.tokenId` |  | Secret | never (token) |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Microtenant identifier.
- `enabled`, `privilegedApprovalsEnabled`, `operator`, `priority`, `creationTime`, `modifiedBy`, `modifiedTime`: Microtenant state and metadata.
- `criteriaAttribute`: Scoping criteria attribute.
- `criteriaAttributeValues`: Local-only criteria values.

## ZPA Isolation Profiles

Commands:

```sh
zscalerctl zpa isolation-profiles list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/isolation-profiles (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `isolationProfileId` | `string` | Sensitive identifier | `standard` |
| `isolationTenantId` | `string` | Sensitive identifier | `standard` |
| `isolationUrl` | `string` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `enabled`, `creationTime`, `modifiedBy`, `modifiedTime`: Profile state and metadata.
- `isolationProfileId`, `isolationUrl`: Tenant-scoped isolation profile reference and URL.

## ZPA Branch Connectors

Commands:

```sh
zscalerctl zpa branch-connectors list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/branch-connectors (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `branchConnectorGroupId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `branchConnectorGroupName` | `string` | Tenant configuration | `standard`, `share` |
| `edgeConnectorGroupId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `edgeConnectorGroupName` | `string` | Tenant configuration | `standard`, `share` |
| `enrollmentCert` |  | Secret | never (unstructured) |
| `fingerprint` | `string` | Sensitive identifier | `standard` |
| `ipAcl` | `array<string>` | Sensitive identifier | `standard` |
| `issuedCertId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `enabled`, `creationTime`, `modifiedBy`, `modifiedTime`: Connector state and metadata.
- `name`, `branchConnectorGroupName`, `edgeConnectorGroupName`: Connector and group names.
- `ipAcl`: Connector IP ACL entries.

The SDK also returns enrollment material (`enrollmentCert`); it never renders.

## ZPA User Portals

Commands:

```sh
zscalerctl zpa user-portals list
zscalerctl zpa user-portals get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/user-portals (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `certificateId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `certificateName` | `string` | Tenant configuration | `standard`, `share` |
| `domain` | `string` | Sensitive identifier | `standard` |
| `extDomain` | `string` | Sensitive identifier | `standard` |
| `extDomainName` | `string` | Sensitive identifier | `standard` |
| `extDomainTranslation` | `string` | Sensitive identifier | `standard` |
| `extLabel` | `string` | Tenant configuration | `standard`, `share` |
| `getcName` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `imageData` |  | Secret | never (opaque blob) |
| `managedByZs` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `userNotification` | `string` | Tenant configuration | `standard`, `share` |
| `userNotificationEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `enabled`, `creationTime`, `modifiedBy`, `modifiedTime`, `getcName`, `managedByZs`, `userNotificationEnabled`: Portal state and metadata.
- `name`, `certificateName`, `extLabel`, `microtenantName`, `userNotification`: Portal naming and notification text.
- `domain`, `extDomain`, `extDomainName`, `extDomainTranslation`: Portal hostnames and domains.

The SDK also returns the portal image blob (`imageData`); it is not rendered.

## ZPA User Portal Aups

Commands:

```sh
zscalerctl zpa user-portal-aups list
zscalerctl zpa user-portal-aups get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/user-portal-aups (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `aup` | `string` | Free text | `standard` |
| `email` | `string` | Sensitive identifier | `standard` |
| `microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `phoneNum` | `string` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `enabled`, `creationTime`, `modifiedBy`, `modifiedTime`: AUP state and metadata.
- `name`, `microtenantName`: AUP name and microtenant name.
- `email`, `phoneNum`: AUP contact email and phone number.

## ZPA User Portal Links

Commands:

```sh
zscalerctl zpa user-portal-links list
zscalerctl zpa user-portal-links get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/user-portal-links (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `applicationId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `iconText` | `string` | Tenant configuration | `standard`, `share` |
| `link` | `string` | Sensitive identifier | `standard` |
| `linkPath` | `string` | Sensitive identifier | `standard` |
| `microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `nameWithoutTrim` | `string` | Tenant configuration | `standard`, `share` |
| `protocol` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `userPortalId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `userPortals` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `userPortals.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `userPortals.name` | `string` | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `enabled`, `creationTime`, `modifiedBy`, `modifiedTime`, `protocol`: Link state and metadata.
- `name`, `iconText`, `microtenantName`, `nameWithoutTrim`: Link naming and microtenant name.
- `link`, `linkPath`: Link target URL and path.

## ZPA Browser Access

Commands:

```sh
zscalerctl zpa browser-access list
zscalerctl zpa browser-access get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/browser-access (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `segmentGroupName` | `string` | Tenant configuration | `standard`, `share` |
| `bypassType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `bypassOnReauth` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `passiveHealthEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `fqdnDnsCheck` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `apiProtectionEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `selectConnectorCloseToApp` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `doubleEncrypt` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `healthCheckType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `isCnameEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `ipAnchored` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `tcpKeepAlive` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `useInDrMode` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `inspectTrafficWithZia` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `healthReporting` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `icmpAccessType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `readOnly` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `restrictionType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `domainNames` | `array<string>` | Sensitive identifier | `standard` |
| `tcpPortRanges` | `array<string>` | Sensitive identifier | `standard` |
| `udpPortRanges` | `array<string>` | Sensitive identifier | `standard` |
| `segmentGroupId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `appRecommendationId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `matchStyle` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `configSpace` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `isIncompleteDRConfig` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zscalerManaged` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `weightedLoadBalancing` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `hbrEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `stickyEntity` | `string` | Tenant configuration | `standard`, `share` |
| `stickyGroup` | `string` | Tenant configuration | `standard`, `share` |
| `tcpPortRange` | `array<object>` | Tenant configuration | `standard` |
| `tcpPortRange.from` | `string` | Operational metadata | `standard` |
| `tcpPortRange.to` | `string` | Operational metadata | `standard` |
| `udpPortRange` | `array<object>` | Tenant configuration | `standard` |
| `udpPortRange.from` | `string` | Operational metadata | `standard` |
| `udpPortRange.to` | `string` | Operational metadata | `standard` |
| `clientlessApps` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `clientlessApps.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `clientlessApps.name` | `string` | Tenant configuration | `standard`, `share` |
| `clientlessApps.description` | `string` | Free text | `standard` |
| `clientlessApps.domain` | `string` | Sensitive identifier | `standard` |
| `clientlessApps.localDomain` | `string` | Sensitive identifier | `standard` |
| `clientlessApps.certificateId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `clientlessApps.certificateName` | `string` | Tenant configuration | `standard`, `share` |
| `clientlessApps.applicationProtocol` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `clientlessApps.applicationPort` | `string` | Sensitive identifier | `standard` |
| `clientlessApps.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `clientlessApps.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `clientlessApps.microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `serverGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `serverGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `serverGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `sharedMicrotenantDetails` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `sharedMicrotenantDetails.sharedFromMicrotenant` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `sharedMicrotenantDetails.sharedFromMicrotenant.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `sharedMicrotenantDetails.sharedFromMicrotenant.name` | `string` | Tenant configuration | `standard`, `share` |
| `sharedMicrotenantDetails.sharedToMicrotenants` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `sharedMicrotenantDetails.sharedToMicrotenants.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `sharedMicrotenantDetails.sharedToMicrotenants.name` | `string` | Tenant configuration | `standard`, `share` |
| `policyStyle` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnErId` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `zpnErId.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnErId.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `zpnErId.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnErId.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `zpnErId.ziaCloud` | `string` | Sensitive identifier | `standard` |
| `zpnErId.ziaErId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnErId.ziaErName` | `string` | Tenant configuration | `standard`, `share` |
| `zpnErId.ziaModifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `zpnErId.ziaOrgId` | `string` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `bypassType`, `bypassOnReauth`, `enabled`, `passiveHealthEnabled`, `fqdnDnsCheck`, `apiProtectionEnabled`, `selectConnectorCloseToApp`, `doubleEncrypt`, `healthCheckType`, `isCnameEnabled`, `ipAnchored`, `tcpKeepAlive`, `useInDrMode`, `inspectTrafficWithZia`, `healthReporting`, `icmpAccessType`, `creationTime`, `modifiedBy`, `modifiedTime`, `readOnly`, `restrictionType`: Segment state and metadata.
- `name`, `segmentGroupName`, `microtenantName`: Segment, group, and microtenant names.
- `domainNames`, `tcpPortRanges`, `udpPortRanges`: Application domains and port ranges.

## ZPA Inspection App Segments

Commands:

```sh
zscalerctl zpa inspection-app-segments list
zscalerctl zpa inspection-app-segments get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/inspection-app-segments (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `segmentGroupName` | `string` | Tenant configuration | `standard`, `share` |
| `bypassType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `bypassOnReauth` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `adpEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `autoAppProtectEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `passiveHealthEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `fqdnDnsCheck` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `apiProtectionEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `selectConnectorCloseToApp` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `doubleEncrypt` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `healthCheckType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `isCnameEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `ipAnchored` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `tcpKeepAlive` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `useInDrMode` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `healthReporting` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `icmpAccessType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `readOnly` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `restrictionType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `tcpProtocols` | `array<string>` | Operational metadata | `standard`, `share`, `paranoid` |
| `udpProtocols` | `array<string>` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `domainNames` | `array<string>` | Sensitive identifier | `standard` |
| `tcpPortRanges` | `array<string>` | Sensitive identifier | `standard` |
| `udpPortRanges` | `array<string>` | Sensitive identifier | `standard` |
| `segmentGroupId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `appRecommendationId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `matchStyle` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `configSpace` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `isIncompleteDRConfig` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zscalerManaged` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `weightedLoadBalancing` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `hbrEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `stickyEntity` | `string` | Tenant configuration | `standard`, `share` |
| `stickyGroup` | `string` | Tenant configuration | `standard`, `share` |
| `tcpPortRange` | `array<object>` | Tenant configuration | `standard` |
| `tcpPortRange.from` | `string` | Operational metadata | `standard` |
| `tcpPortRange.to` | `string` | Operational metadata | `standard` |
| `udpPortRange` | `array<object>` | Tenant configuration | `standard` |
| `udpPortRange.from` | `string` | Operational metadata | `standard` |
| `udpPortRange.to` | `string` | Operational metadata | `standard` |
| `inspectionApps` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `inspectionApps.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `inspectionApps.appId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `inspectionApps.name` | `string` | Tenant configuration | `standard`, `share` |
| `inspectionApps.description` | `string` | Free text | `standard` |
| `inspectionApps.enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `inspectionApps.domain` | `string` | Sensitive identifier | `standard` |
| `inspectionApps.certificateId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `inspectionApps.certificateName` | `string` | Tenant configuration | `standard`, `share` |
| `inspectionApps.applicationProtocol` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `inspectionApps.applicationPort` | `string` | Sensitive identifier | `standard` |
| `inspectionApps.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `inspectionApps.microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `inspectionApps.trustUntrustedCert` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `inspectionApps.protocols` | `array<string>` | Operational metadata | `standard`, `share`, `paranoid` |
| `commonAppsDto` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `commonAppsDto.appsConfig` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `commonAppsDto.appsConfig.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `commonAppsDto.appsConfig.appId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `commonAppsDto.appsConfig.inspectAppId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `commonAppsDto.appsConfig.name` | `string` | Tenant configuration | `standard`, `share` |
| `commonAppsDto.appsConfig.description` | `string` | Free text | `standard` |
| `commonAppsDto.appsConfig.enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `commonAppsDto.appsConfig.appTypes` | `array<string>` | Operational metadata | `standard`, `share`, `paranoid` |
| `commonAppsDto.appsConfig.applicationPort` | `string` | Sensitive identifier | `standard` |
| `commonAppsDto.appsConfig.applicationProtocol` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `commonAppsDto.appsConfig.cname` | `string` | Sensitive identifier | `standard` |
| `commonAppsDto.appsConfig.domain` | `string` | Sensitive identifier | `standard` |
| `commonAppsDto.appsConfig.localDomain` | `string` | Sensitive identifier | `standard` |
| `commonAppsDto.appsConfig.hidden` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `commonAppsDto.appsConfig.portal` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `commonAppsDto.appsConfig.adpEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `commonAppsDto.appsConfig.allowOptions` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `commonAppsDto.appsConfig.trustUntrustedCert` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `commonAppsDto.appsConfig.protocols` | `array<string>` | Operational metadata | `standard`, `share`, `paranoid` |
| `commonAppsDto.appsConfig.certificateId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `commonAppsDto.appsConfig.certificateName` | `string` | Tenant configuration | `standard`, `share` |
| `commonAppsDto.deletedInspectApps` | `array<string>` | Operational metadata | `standard`, `share`, `paranoid` |
| `serverGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `serverGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `serverGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `sharedMicrotenantDetails` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `sharedMicrotenantDetails.sharedFromMicrotenant` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `sharedMicrotenantDetails.sharedFromMicrotenant.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `sharedMicrotenantDetails.sharedFromMicrotenant.name` | `string` | Tenant configuration | `standard`, `share` |
| `sharedMicrotenantDetails.sharedToMicrotenants` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `sharedMicrotenantDetails.sharedToMicrotenants.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `sharedMicrotenantDetails.sharedToMicrotenants.name` | `string` | Tenant configuration | `standard`, `share` |
| `policyStyle` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `bypassType`, `bypassOnReauth`, `enabled`, `adpEnabled`, `autoAppProtectEnabled`, `passiveHealthEnabled`, `fqdnDnsCheck`, `apiProtectionEnabled`, `selectConnectorCloseToApp`, `doubleEncrypt`, `healthCheckType`, `isCnameEnabled`, `ipAnchored`, `tcpKeepAlive`, `useInDrMode`, `healthReporting`, `icmpAccessType`, `creationTime`, `modifiedBy`, `modifiedTime`, `readOnly`, `restrictionType`, `tcpProtocols`, `udpProtocols`: Segment state and metadata.
- `name`, `segmentGroupName`, `microtenantName`: Segment, group, and microtenant names.
- `domainNames`, `tcpPortRanges`, `udpPortRanges`: Application domains and port ranges.

## ZPA Pra App Segments

Commands:

```sh
zscalerctl zpa pra-app-segments list
zscalerctl zpa pra-app-segments get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/pra-app-segments (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `segmentGroupName` | `string` | Tenant configuration | `standard`, `share` |
| `bypassType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `bypassOnReauth` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `passiveHealthEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `fqdnDnsCheck` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `apiProtectionEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `selectConnectorCloseToApp` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `doubleEncrypt` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `healthCheckType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `isCnameEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `ipAnchored` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `tcpKeepAlive` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `useInDrMode` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `healthReporting` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `icmpAccessType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `readOnly` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `restrictionType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `domainNames` | `array<string>` | Sensitive identifier | `standard` |
| `tcpPortRanges` | `array<string>` | Sensitive identifier | `standard` |
| `udpPortRanges` | `array<string>` | Sensitive identifier | `standard` |
| `segmentGroupId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `appRecommendationId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `matchStyle` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `configSpace` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `applications` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `isIncompleteDRConfig` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zscalerManaged` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `weightedLoadBalancing` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `hbrEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `stickyEntity` | `string` | Tenant configuration | `standard`, `share` |
| `stickyGroup` | `string` | Tenant configuration | `standard`, `share` |
| `tcpPortRange` | `array<object>` | Tenant configuration | `standard` |
| `tcpPortRange.from` | `string` | Operational metadata | `standard` |
| `tcpPortRange.to` | `string` | Operational metadata | `standard` |
| `udpPortRange` | `array<object>` | Tenant configuration | `standard` |
| `udpPortRange.from` | `string` | Operational metadata | `standard` |
| `udpPortRange.to` | `string` | Operational metadata | `standard` |
| `defaultIdleTimeout` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `defaultMaxAge` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `praApps` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `praApps.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `praApps.appId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `praApps.name` | `string` | Tenant configuration | `standard`, `share` |
| `praApps.description` | `string` | Free text | `standard` |
| `praApps.enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `praApps.domain` | `string` | Sensitive identifier | `standard` |
| `praApps.certificateId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `praApps.certificateName` | `string` | Tenant configuration | `standard`, `share` |
| `praApps.applicationProtocol` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `praApps.applicationPort` | `string` | Sensitive identifier | `standard` |
| `praApps.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `praApps.microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `praApps.connectionSecurity` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `praApps.hidden` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `praApps.portal` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `commonAppsDto` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `commonAppsDto.appsConfig` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `commonAppsDto.appsConfig.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `commonAppsDto.appsConfig.appId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `commonAppsDto.appsConfig.praAppId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `commonAppsDto.appsConfig.name` | `string` | Tenant configuration | `standard`, `share` |
| `commonAppsDto.appsConfig.description` | `string` | Free text | `standard` |
| `commonAppsDto.appsConfig.enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `commonAppsDto.appsConfig.appTypes` | `array<string>` | Operational metadata | `standard`, `share`, `paranoid` |
| `commonAppsDto.appsConfig.applicationPort` | `string` | Sensitive identifier | `standard` |
| `commonAppsDto.appsConfig.applicationProtocol` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `commonAppsDto.appsConfig.cname` | `string` | Sensitive identifier | `standard` |
| `commonAppsDto.appsConfig.domain` | `string` | Sensitive identifier | `standard` |
| `commonAppsDto.appsConfig.localDomain` | `string` | Sensitive identifier | `standard` |
| `commonAppsDto.appsConfig.hidden` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `commonAppsDto.appsConfig.portal` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `commonAppsDto.appsConfig.connectionSecurity` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `commonAppsDto.deletedPraApps` | `array<string>` | Operational metadata | `standard`, `share`, `paranoid` |
| `serverGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `serverGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `serverGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `sharedMicrotenantDetails` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `sharedMicrotenantDetails.sharedFromMicrotenant` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `sharedMicrotenantDetails.sharedFromMicrotenant.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `sharedMicrotenantDetails.sharedFromMicrotenant.name` | `string` | Tenant configuration | `standard`, `share` |
| `sharedMicrotenantDetails.sharedToMicrotenants` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `sharedMicrotenantDetails.sharedToMicrotenants.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `sharedMicrotenantDetails.sharedToMicrotenants.name` | `string` | Tenant configuration | `standard`, `share` |
| `policyStyle` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnErId` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `zpnErId.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnErId.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `zpnErId.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnErId.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `zpnErId.ziaCloud` | `string` | Sensitive identifier | `standard` |
| `zpnErId.ziaErId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnErId.ziaErName` | `string` | Tenant configuration | `standard`, `share` |
| `zpnErId.ziaModifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `zpnErId.ziaOrgId` | `string` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `bypassType`, `bypassOnReauth`, `enabled`, `passiveHealthEnabled`, `fqdnDnsCheck`, `apiProtectionEnabled`, `selectConnectorCloseToApp`, `doubleEncrypt`, `healthCheckType`, `isCnameEnabled`, `ipAnchored`, `tcpKeepAlive`, `useInDrMode`, `healthReporting`, `icmpAccessType`, `creationTime`, `modifiedBy`, `modifiedTime`, `readOnly`, `restrictionType`: Segment state and metadata.
- `name`, `segmentGroupName`, `microtenantName`: Segment, group, and microtenant names.
- `domainNames`, `tcpPortRanges`, `udpPortRanges`: Application domains and port ranges.

## ZPA Server Groups

Commands:

```sh
zscalerctl zpa server-groups list
zscalerctl zpa server-groups get <id>
zscalerctl dump --products zpa --resources zpa/server-groups --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/server-groups (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `ipAnchored` | `boolean` | Sensitive identifier | `standard` |
| `configSpace` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `dynamicDiscovery` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `readOnly` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `restrictionType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zscalerManaged` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `appConnectorGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `appConnectorGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appConnectorGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `servers` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `servers.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `servers.name` | `string` | Tenant configuration | `standard`, `share` |
| `servers.enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `servers.address` | `string` | Sensitive identifier | `standard` |
| `applications` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `applications.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `applications.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO.ziaLocations` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.ziaLocations.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.ziaLocations.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO.ziaLocations.enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.ziaErName` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.zpnErId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

## ZPA Segment Groups

Commands:

```sh
zscalerctl zpa segment-groups list
zscalerctl zpa segment-groups get <id>
zscalerctl dump --products zpa --resources zpa/segment-groups --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/segment-groups (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `configSpace` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyMigrated` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `tcpKeepAliveEnabled` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `addedApps` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `deletedApps` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `applications` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `applications.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `applications.name` | `string` | Tenant configuration | `standard`, `share` |
| `applicationNames` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `applicationNames.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `applicationNames.name` | `string` | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `enabled`, `creationTime`, `modifiedTime`, `policyMigrated`, `tcpKeepAliveEnabled`: Segment group identity, state, lifecycle, and operational flags.

## ZPA Application Segments

Commands:

```sh
zscalerctl zpa application-segments list
zscalerctl zpa application-segments get <id>
zscalerctl dump --products zpa --resources zpa/application-segments --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/application-segments (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `bypassType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `healthCheckType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `icmpAccessType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `healthReporting` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `passiveHealthEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `ipAnchored` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `fqdnDnsCheck` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `tcpKeepAlive` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `isCnameEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `selectConnectorCloseToApp` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `restrictionType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `useInDrMode` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `creationTime` | `string` | Operational metadata | `standard`, `share` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `apiProtectionEnabled` | `boolean` | Operational metadata | `standard` |
| `inspectTrafficWithZia` | `boolean` | Operational metadata | `standard` |
| `doubleEncrypt` | `boolean` | Operational metadata | `standard` |
| `adpEnabled` | `boolean` | Operational metadata | `standard` |
| `autoAppProtectEnabled` | `boolean` | Operational metadata | `standard` |
| `bypassOnReauth` | `boolean` | Operational metadata | `standard` |
| `hbrEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `stickyEntity` | `string` | Tenant configuration | `standard`, `share` |
| `stickyGroup` | `string` | Tenant configuration | `standard`, `share` |
| `segmentGroupName` | `string` | Tenant configuration | `standard`, `share` |
| `microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `domainNames` | `array<string>` | Sensitive identifier | `standard` |
| `tcpPortRanges` | `array<string>` | Sensitive identifier | `standard` |
| `udpPortRanges` | `array<string>` | Sensitive identifier | `standard` |
| `serverGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `serverGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `serverGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `segmentGroupId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appRecommendationId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `applications` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `configSpace` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `matchStyle` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyStyle` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `isIncompleteDRConfig` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `defaultMaxAge` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `defaultIdleTimeout` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `readOnly` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `zscalerManaged` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `clientlessApps` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `clientlessApps.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `clientlessApps.name` | `string` | Tenant configuration | `standard`, `share` |
| `clientlessApps.description` | `string` | Free text | `standard` |
| `clientlessApps.domain` | `string` | Sensitive identifier | `standard` |
| `clientlessApps.localDomain` | `string` | Sensitive identifier | `standard` |
| `clientlessApps.certificateId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `clientlessApps.certificateName` | `string` | Tenant configuration | `standard`, `share` |
| `clientlessApps.applicationProtocol` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `clientlessApps.applicationPort` | `string` | Sensitive identifier | `standard` |
| `clientlessApps.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `clientlessApps.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `clientlessApps.microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `sharedMicrotenantDetails` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `sharedMicrotenantDetails.sharedFromMicrotenant` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `sharedMicrotenantDetails.sharedFromMicrotenant.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `sharedMicrotenantDetails.sharedFromMicrotenant.name` | `string` | Tenant configuration | `standard`, `share` |
| `sharedMicrotenantDetails.sharedToMicrotenants` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `sharedMicrotenantDetails.sharedToMicrotenants.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `sharedMicrotenantDetails.sharedToMicrotenants.name` | `string` | Tenant configuration | `standard`, `share` |
| `shareToMicrotenants` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `tags` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `tags.namespace` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `tags.namespace.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `tags.namespace.name` | `string` | Tenant configuration | `standard`, `share` |
| `tags.namespace.enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `tags.tagKey` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `tags.tagKey.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `tags.tagKey.name` | `string` | Tenant configuration | `standard`, `share` |
| `tags.tagKey.enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `tags.tagValue` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `tags.tagValue.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `tags.tagValue.name` | `string` | Tenant configuration | `standard`, `share` |
| `tags.origin` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnErId` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `zpnErId.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnErId.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `zpnErId.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnErId.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `zpnErId.ziaCloud` | `string` | Sensitive identifier | `standard` |
| `zpnErId.ziaErId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnErId.ziaErName` | `string` | Tenant configuration | `standard`, `share` |
| `zpnErId.ziaModifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `zpnErId.ziaOrgId` | `string` | Sensitive identifier | `standard` |
| `tcpPortRange` | `array<object>` | Tenant configuration | `standard` |
| `tcpPortRange.from` | `string` | Operational metadata | `standard` |
| `tcpPortRange.to` | `string` | Operational metadata | `standard` |
| `udpPortRange` | `array<object>` | Tenant configuration | `standard` |
| `udpPortRange.from` | `string` | Operational metadata | `standard` |
| `udpPortRange.to` | `string` | Operational metadata | `standard` |
| `weightedLoadBalancing` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `enabled`, `bypassType`, `healthCheckType`, `icmpAccessType`, `healthReporting`, `passiveHealthEnabled`, `ipAnchored`, `fqdnDnsCheck`, `tcpKeepAlive`, `isCnameEnabled`, `selectConnectorCloseToApp`, `restrictionType`, `useInDrMode`: Application segment identity and reviewed behavior flags.
- `creationTime`, `modifiedTime`: Lifecycle timestamps.
- `apiProtectionEnabled`, `inspectTrafficWithZia`, `doubleEncrypt`, `adpEnabled`, `autoAppProtectEnabled`, `bypassOnReauth`: Local-only posture flags.
- `domainNames`, `tcpPortRanges`, `udpPortRanges`: Local-only application matching surface and traffic ports.

## ZPA App Connector Groups

Commands:

```sh
zscalerctl zpa app-connector-groups list
zscalerctl zpa app-connector-groups get <id>
zscalerctl dump --products zpa --resources zpa/app-connector-groups --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/app-connector-groups (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `cityCountry` | `string` | Sensitive identifier | `standard` |
| `countryCode` | `string` | Sensitive identifier | `standard` |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `dnsQueryType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `connectorGroupType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `geoLocationId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `latitude` | `string` | Sensitive identifier | `standard` |
| `location` | `string` | Sensitive identifier | `standard` |
| `longitude` | `string` | Sensitive identifier | `standard` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `overrideVersionProfile` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `praEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `wafDisabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `upgradeDay` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `upgradeTimeInSecs` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `versionProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `versionProfileName` | `string` | Tenant configuration | `standard`, `share` |
| `versionProfileVisibilityScope` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `tcpQuickAckApp` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `tcpQuickAckAssistant` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `useInDrMode` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `tcpQuickAckReadAssistant` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `lssAppConnectorGroup` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `siteId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `siteName` | `string` | Tenant configuration | `standard`, `share` |
| `readOnly` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `restrictionType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zscalerManaged` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `dcHostingInfo` | `string` | Sensitive identifier | `standard` |
| `nameWithoutTrim` | `string` | Tenant configuration | `standard`, `share` |
| `serverGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `serverGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `serverGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `connectors` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `connectors.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `connectors.name` | `string` | Tenant configuration | `standard`, `share` |
| `npAssistantGroup` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `npAssistantGroup.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `npAssistantGroup.appConnectorGroupId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `npAssistantGroup.creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `npAssistantGroup.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `npAssistantGroup.modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `npAssistantGroup.mtu` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `npAssistantGroup.lanSubnets` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `npAssistantGroup.lanSubnets.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `npAssistantGroup.lanSubnets.name` | `string` | Tenant configuration | `standard`, `share` |
| `npAssistantGroup.lanSubnets.description` | `string` | Free text | `standard` |
| `npAssistantGroup.lanSubnets.appConnectorGroupId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `npAssistantGroup.lanSubnets.creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `npAssistantGroup.lanSubnets.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `npAssistantGroup.lanSubnets.modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `npAssistantGroup.lanSubnets.subnet` | `string` | Sensitive identifier | `standard` |
| `npAssistantGroup.lanSubnets.npserverips` | `array<string>` | Sensitive identifier | `standard` |
| `npAssistantGroup.lanSubnets.fqdns` | `array<string>` | Sensitive identifier | `standard` |
| `npAssistantGroup.lanSubnets.npDnsNsRecord` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `npAssistantGroup.lanSubnets.npDnsNsRecord.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `npAssistantGroup.lanSubnets.npDnsNsRecord.name` | `string` | Tenant configuration | `standard`, `share` |
| `npAssistantGroup.lanSubnets.npDnsNsRecord.creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `npAssistantGroup.lanSubnets.npDnsNsRecord.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `npAssistantGroup.lanSubnets.npDnsNsRecord.modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `npAssistantGroup.lanSubnets.npDnsNsRecord.fqdn` | `array<string>` | Sensitive identifier | `standard` |
| `npAssistantGroup.lanSubnets.npDnsNsRecord.nameserverIps` | `array<string>` | Sensitive identifier | `standard` |
| `enrollmentCertId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

## ZPA App Servers

Commands:

```sh
zscalerctl zpa app-servers list
zscalerctl zpa app-servers get <id>
zscalerctl dump --products zpa --resources zpa/app-servers --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/app-servers (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `address` | `string` | Sensitive identifier | `standard` |
| `appServerGroupIds` | `array<string>` | Operational metadata | `standard`, `share`, `paranoid` |
| `configSpace` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `description` | `string` | Free text | `standard` |
| `enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `enabled`, `creationTime`, `modifiedTime`: Application server identity, state, and lifecycle metadata.
- `address`: Local-only server address.

## ZPA Machine Groups

Commands:

```sh
zscalerctl zpa machine-groups list
zscalerctl zpa machine-groups get <id>
zscalerctl dump --products zpa --resources zpa/machine-groups --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/machine-groups (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `machines` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `machines.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `machines.name` | `string` | Tenant configuration | `standard`, `share` |
| `machines.description` | `string` | Free text | `standard` |
| `machines.creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `machines.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `machines.modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `machines.fingerprint` | `string` | Sensitive identifier | `standard` |
| `machines.issuedCertId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `machines.machineGroupId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `machines.machineGroupName` | `string` | Tenant configuration | `standard`, `share` |
| `machines.machineTokenId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `machines.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `machines.microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `machines.signingCert` |  | Secret | never (unstructured) |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantName` | `string` | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `creationTime`, `modifiedTime`: Machine group identity and lifecycle metadata.

## ZPA Trusted Networks

Commands:

```sh
zscalerctl zpa trusted-networks list
zscalerctl zpa trusted-networks get <id>
zscalerctl dump --products zpa --resources zpa/trusted-networks --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/trusted-networks (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `domain` | `string` | Sensitive identifier | `standard` |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `masterCustomerId` | `string` | Sensitive identifier | `standard` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `networkId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zscalerCloud` | `string` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `creationTime`, `modifiedTime`: Trusted network identity and lifecycle metadata.

## ZPA Service Edge Groups

Commands:

```sh
zscalerctl zpa service-edge-groups list
zscalerctl zpa service-edge-groups get <id>
zscalerctl dump --products zpa --resources zpa/service-edge-groups --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/service-edge-groups (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `altCloud` | `string` | Sensitive identifier | `standard` |
| `city` | `string` | Sensitive identifier | `standard` |
| `cityCountry` | `string` | Sensitive identifier | `standard` |
| `countryCode` | `string` | Sensitive identifier | `standard` |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `description` | `string` | Free text | `standard` |
| `enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `enrollmentCertId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `exclusiveForBusinessContinuity` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `geoLocationId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `graceDistanceEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `graceDistanceValue` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `graceDistanceValueUnit` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `isPublic` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `latitude` | `string` | Sensitive identifier | `standard` |
| `location` | `string` | Sensitive identifier | `standard` |
| `longitude` | `string` | Sensitive identifier | `standard` |
| `microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `nameWithoutTrim` | `string` | Tenant configuration | `standard`, `share` |
| `objectType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `overrideVersionProfile` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `readOnly` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `restrictedEntity` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `restrictionType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `scopeName` | `string` | Tenant configuration | `standard`, `share` |
| `serviceEdges` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `serviceEdges.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `serviceEdges.name` | `string` | Tenant configuration | `standard`, `share` |
| `siteId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `siteName` | `string` | Tenant configuration | `standard`, `share` |
| `trustedNetworks` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `trustedNetworks.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `trustedNetworks.name` | `string` | Tenant configuration | `standard`, `share` |
| `upgradeDay` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `upgradeTimeInSecs` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `useInDrMode` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `versionProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `versionProfileName` | `string` | Tenant configuration | `standard`, `share` |
| `versionProfileVisibilityScope` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zscalerManaged` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `enabled`, `creationTime`, `modifiedBy`, `modifiedTime`, `isPublic`, `objectType`, `overrideVersionProfile`, `readOnly`, `restrictedEntity`, `restrictionType`, `graceDistanceEnabled`, `exclusiveForBusinessContinuity`, `upgradeDay`, `upgradeTimeInSecs`, `useInDrMode`: Service edge group identity, state, lifecycle, and operational flags.
- `cityCountry`, `countryCode`, `latitude`, `location`, `longitude`: Local-only placement metadata.

## ZPA Service Edges

Commands:

```sh
zscalerctl zpa service-edges list
zscalerctl zpa service-edges get <id>
zscalerctl dump --products zpa --resources zpa/service-edges --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/service-edges (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `applicationStartTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `controlChannelStatus` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `ctrlBrokerName` | `string` | Tenant configuration | `standard`, `share` |
| `currentVersion` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `enrollmentCert` |  | Secret | never (unstructured) |
| `expectedUpgradeTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `expectedVersion` | `string` | Tenant configuration | `standard`, `share` |
| `fingerprint` | `string` | Sensitive identifier | `standard` |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `ipAcl` | `string` | Sensitive identifier | `standard` |
| `issuedCertId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastBrokerConnectTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastBrokerConnectTimeDuration` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastBrokerDisconnectTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastBrokerDisconnectTimeDuration` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastUpgradeTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `latitude` | `string` | Sensitive identifier | `standard` |
| `listenIps` | `array<string>` | Sensitive identifier | `standard` |
| `location` | `string` | Sensitive identifier | `standard` |
| `longitude` | `string` | Sensitive identifier | `standard` |
| `microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `platform` | `string` | Tenant configuration | `standard`, `share` |
| `platformDetail` | `string` | Tenant configuration | `standard`, `share` |
| `previousVersion` | `string` | Tenant configuration | `standard`, `share` |
| `privateBrokerVersion` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privateBrokerVersion.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privateBrokerVersion.applicationStartTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privateBrokerVersion.brokerId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privateBrokerVersion.creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privateBrokerVersion.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privateBrokerVersion.modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privateBrokerVersion.ctrlChannelStatus` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privateBrokerVersion.currentVersion` | `string` | Tenant configuration | `standard`, `share` |
| `privateBrokerVersion.previousVersion` | `string` | Tenant configuration | `standard`, `share` |
| `privateBrokerVersion.sargeVersion` | `string` | Tenant configuration | `standard`, `share` |
| `privateBrokerVersion.disableAutoUpdate` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `privateBrokerVersion.lastUpgradedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privateBrokerVersion.loneWarrior` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `privateBrokerVersion.platform` | `string` | Tenant configuration | `standard`, `share` |
| `privateBrokerVersion.platformDetail` | `string` | Tenant configuration | `standard`, `share` |
| `privateBrokerVersion.privateIp` | `string` | Sensitive identifier | `standard` |
| `privateBrokerVersion.publicIp` | `string` | Sensitive identifier | `standard` |
| `privateBrokerVersion.restartTimeInSec` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privateBrokerVersion.runtimeOS` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privateBrokerVersion.systemStartTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privateBrokerVersion.upgradeAttempt` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privateBrokerVersion.upgradeStatus` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privateBrokerVersion.upgradeNowOnce` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `privateBrokerVersion.serviceEdgeGroupId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privateBrokerVersion.lastConnectTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privateBrokerVersion.lastDisconnectTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privateBrokerVersion.zpnSubModuleUpgradeList` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privateBrokerVersion.zpnSubModuleUpgradeList.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privateBrokerVersion.zpnSubModuleUpgradeList.creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privateBrokerVersion.zpnSubModuleUpgradeList.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privateBrokerVersion.zpnSubModuleUpgradeList.modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privateBrokerVersion.zpnSubModuleUpgradeList.currentVersion` | `string` | Tenant configuration | `standard`, `share` |
| `privateBrokerVersion.zpnSubModuleUpgradeList.expectedVersion` | `string` | Tenant configuration | `standard`, `share` |
| `privateBrokerVersion.zpnSubModuleUpgradeList.previousVersion` | `string` | Tenant configuration | `standard`, `share` |
| `privateBrokerVersion.zpnSubModuleUpgradeList.entityGid` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privateBrokerVersion.zpnSubModuleUpgradeList.entityType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privateBrokerVersion.zpnSubModuleUpgradeList.role` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privateBrokerVersion.zpnSubModuleUpgradeList.upgradeStatus` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privateBrokerVersion.zpnSubModuleUpgradeList.upgradeTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privateIp` | `string` | Sensitive identifier | `standard` |
| `provisioningKeyId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `provisioningKeyName` | `string` | Tenant configuration | `standard`, `share` |
| `publicIp` | `string` | Sensitive identifier | `standard` |
| `publishIps` | `array<string>` | Sensitive identifier | `standard` |
| `publishIpv6` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `runtimeOS` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `sargeVersion` | `string` | Tenant configuration | `standard`, `share` |
| `serviceEdgeGroupId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `serviceEdgeGroupName` | `string` | Tenant configuration | `standard`, `share` |
| `upgradeAttempt` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `upgradeStatus` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `enabled`, `applicationStartTime`, `controlChannelStatus`, `creationTime`, `expectedUpgradeTime`, `lastBrokerConnectTime`, `lastBrokerConnectTimeDuration`, `lastBrokerDisconnectTime`, `lastBrokerDisconnectTimeDuration`, `lastUpgradeTime`, `modifiedBy`, `modifiedTime`, `publishIpv6`, `runtimeOS`, `upgradeStatus`: Service edge identity, state, lifecycle, connection, runtime, and upgrade metadata.
- `ipAcl`, `latitude`, `listenIps`, `location`, `longitude`, `privateIp`, `publicIp`, `publishIps`: Local-only placement and network metadata.

## ZPA Cloud Connector Groups

Commands:

```sh
zscalerctl zpa cloud-connector-groups list
zscalerctl zpa cloud-connector-groups get <id>
zscalerctl dump --products zpa --resources zpa/cloud-connector-groups --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/cloud-connector-groups (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `cloudConnectors` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `cloudConnectors.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `cloudConnectors.name` | `string` | Tenant configuration | `standard`, `share` |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `description` | `string` | Free text | `standard` |
| `enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `geoLocationId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `ziaCloud` | `string` | Sensitive identifier | `standard` |
| `ziaOrgId` | `string` | Sensitive identifier | `standard` |
| `znfGroupType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `enabled`, `creationTime`, `modifiedBy`, `modifiedTime`, `znfGroupType`: Cloud connector group identity, state, lifecycle, and operational type metadata.

## ZPA Cloud Connectors

Commands:

```sh
zscalerctl zpa cloud-connectors list
zscalerctl dump --products zpa --resources zpa/cloud-connectors --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/cloud-connectors (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `description` | `string` | Free text | `standard` |
| `edgeConnectorGroupId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `edgeConnectorGroupName` | `string` | Tenant configuration | `standard`, `share` |
| `enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `enrollmentCert` |  | Secret | never (unstructured) |
| `fingerprint` | `string` | Sensitive identifier | `standard` |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `ipAcl` | `array<string>` | Sensitive identifier | `standard` |
| `issuedCertId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `enabled`, `creationTime`, `modifiedBy`, `modifiedTime`: Cloud connector identity, state, and lifecycle metadata.
- `ipAcl`: Local-only cloud connector ACL metadata.

## ZPA Posture Profiles

Commands:

```sh
zscalerctl zpa posture-profiles list
zscalerctl zpa posture-profiles get <id>
zscalerctl dump --products zpa --resources zpa/posture-profiles --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/posture-profiles (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `applyToMachineTunnelEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `crlCheckEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `domain` | `string` | Sensitive identifier | `standard` |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `masterCustomerId` | `string` | Sensitive identifier | `standard` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `nonExportablePrivateKeyEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `platform` | `string` | Tenant configuration | `standard`, `share` |
| `postureType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `postureUdid` | `string` | Sensitive identifier | `standard` |
| `rootCert` | `string` | Tenant configuration | `standard` |
| `zscalerCloud` | `string` | Sensitive identifier | `standard` |
| `zscalerCustomerId` | `string` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `applyToMachineTunnelEnabled`, `creationTime`, `crlCheckEnabled`, `modifiedBy`, `modifiedTime`, `postureType`: Posture profile identity, lifecycle, posture type, and certificate-check flags.
- `domain`: Local-only posture domain metadata.

## ZPA Cbi Zpa Profiles

Commands:

```sh
zscalerctl zpa cbi-zpa-profiles list
zscalerctl zpa cbi-zpa-profiles get <id>
zscalerctl dump --products zpa --resources zpa/cbi-zpa-profiles --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/cbi-zpa-profiles (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `cbiProfileId` | `string` | Sensitive identifier | `standard` |
| `cbiTenantId` | `string` | Sensitive identifier | `standard` |
| `cbiUrl` | `string` | Sensitive identifier | `standard` |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `description` | `string` | Free text | `standard` |
| `enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `enabled`, `creationTime`, `modifiedBy`, `modifiedTime`: CBI ZPA profile identity, state, and lifecycle metadata.
- `cbiProfileId`, `cbiUrl`: Local-only CBI profile and tenant URL metadata.

## ZPA App Connectors

Commands:

```sh
zscalerctl zpa app-connectors list
zscalerctl zpa app-connectors get <id>
zscalerctl dump --products zpa --resources zpa/app-connectors --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/app-connectors (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `appConnectorGroupId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appConnectorGroupName` | `string` | Tenant configuration | `standard`, `share` |
| `applicationStartTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `assistantVersion` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `assistantVersion.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `assistantVersion.applicationStartTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `assistantVersion.brokerId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `assistantVersion.creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `assistantVersion.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `assistantVersion.modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `assistantVersion.ctrlChannelStatus` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `assistantVersion.currentVersion` | `string` | Tenant configuration | `standard`, `share` |
| `assistantVersion.previousVersion` | `string` | Tenant configuration | `standard`, `share` |
| `assistantVersion.sargeVersion` | `string` | Tenant configuration | `standard`, `share` |
| `assistantVersion.disableAutoUpdate` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `assistantVersion.lastUpgradedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `assistantVersion.loneWarrior` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `assistantVersion.platform` | `string` | Tenant configuration | `standard`, `share` |
| `assistantVersion.platformDetail` | `string` | Tenant configuration | `standard`, `share` |
| `assistantVersion.privateIp` | `string` | Sensitive identifier | `standard` |
| `assistantVersion.publicIp` | `string` | Sensitive identifier | `standard` |
| `assistantVersion.restartTimeInSec` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `assistantVersion.runtimeOS` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `assistantVersion.systemStartTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `assistantVersion.upgradeAttempt` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `assistantVersion.upgradeStatus` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `assistantVersion.upgradeNowOnce` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `assistantVersion.appConnectorGroupId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `assistantVersion.expectedVersion` | `string` | Tenant configuration | `standard`, `share` |
| `assistantVersion.lastBrokerConnectTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `assistantVersion.lastBrokerDisconnectTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `assistantVersion.latitude` | `string` | Sensitive identifier | `standard` |
| `assistantVersion.longitude` | `string` | Sensitive identifier | `standard` |
| `controlChannelStatus` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `ctrlBrokerName` | `string` | Tenant configuration | `standard`, `share` |
| `currentVersion` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `enrollmentCert` |  | Secret | never (unstructured) |
| `expectedUpgradeTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `expectedVersion` | `string` | Tenant configuration | `standard`, `share` |
| `fingerprint` | `string` | Sensitive identifier | `standard` |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `ipAcl` | `string` | Sensitive identifier | `standard` |
| `issuedCertId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastBrokerConnectTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastBrokerConnectTimeDuration` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastBrokerDisconnectTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastBrokerDisconnectTimeDuration` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastUpgradeTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `latitude` | `string` | Sensitive identifier | `standard` |
| `location` | `string` | Sensitive identifier | `standard` |
| `longitude` | `string` | Sensitive identifier | `standard` |
| `microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `platform` | `string` | Tenant configuration | `standard`, `share` |
| `platformDetail` | `string` | Tenant configuration | `standard`, `share` |
| `previousVersion` | `string` | Tenant configuration | `standard`, `share` |
| `privateIp` | `string` | Sensitive identifier | `standard` |
| `provisioningKeyId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `provisioningKeyName` | `string` | Tenant configuration | `standard`, `share` |
| `publicIp` | `string` | Sensitive identifier | `standard` |
| `readOnly` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `restrictionType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `runtimeOS` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `sargeVersion` | `string` | Tenant configuration | `standard`, `share` |
| `upgradeAttempt` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `upgradeStatus` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnSubModuleUpgradeList` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `zpnSubModuleUpgradeList.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnSubModuleUpgradeList.creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnSubModuleUpgradeList.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnSubModuleUpgradeList.modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnSubModuleUpgradeList.currentVersion` | `string` | Tenant configuration | `standard`, `share` |
| `zpnSubModuleUpgradeList.expectedVersion` | `string` | Tenant configuration | `standard`, `share` |
| `zpnSubModuleUpgradeList.previousVersion` | `string` | Tenant configuration | `standard`, `share` |
| `zpnSubModuleUpgradeList.entityGid` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnSubModuleUpgradeList.entityType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnSubModuleUpgradeList.role` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnSubModuleUpgradeList.upgradeStatus` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnSubModuleUpgradeList.upgradeTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zscalerManaged` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `enabled`, `applicationStartTime`, `controlChannelStatus`, `creationTime`, `expectedUpgradeTime`, `lastBrokerConnectTime`, `lastBrokerConnectTimeDuration`, `lastBrokerDisconnectTime`, `lastBrokerDisconnectTimeDuration`, `lastUpgradeTime`, `modifiedBy`, `modifiedTime`, `readOnly`, `restrictionType`, `runtimeOS`, `upgradeStatus`: Connector identity, lifecycle, control-channel, runtime, and upgrade metadata.
- `ipAcl`, `latitude`, `location`, `longitude`, `privateIp`, `publicIp`: Local-only connector placement and network metadata.

## ZPA C2c IP Ranges

Commands:

```sh
zscalerctl zpa c2c-ip-ranges list
zscalerctl zpa c2c-ip-ranges get <id>
zscalerctl dump --products zpa --resources zpa/c2c-ip-ranges --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/c2c-ip-ranges (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `availableIps` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `countryCode` | `string` | Sensitive identifier | `standard` |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `customerId` | `string` | Sensitive identifier | `standard` |
| `description` | `string` | Free text | `standard` |
| `enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `ipRangeBegin` | `string` | Sensitive identifier | `standard` |
| `ipRangeEnd` | `string` | Sensitive identifier | `standard` |
| `isDeleted` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `latitudeInDb` | `string` | Sensitive identifier | `standard` |
| `location` | `string` | Sensitive identifier | `standard` |
| `locationHint` | `string` | Sensitive identifier | `standard` |
| `longitudeInDb` | `string` | Sensitive identifier | `standard` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `sccmFlag` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `subnetCidr` | `string` | Sensitive identifier | `standard` |
| `totalIps` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `usedIps` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `enabled`, `creationTime`, `modifiedBy`, `modifiedTime`, `availableIps`, `totalIps`, `usedIps`, `isDeleted`, `sccmFlag`: C2C IP range identity, lifecycle, state, and utilization metadata.
- `countryCode`, `ipRangeBegin`, `ipRangeEnd`, `latitudeInDb`, `location`, `locationHint`, `longitudeInDb`, `subnetCidr`: Local-only network range and placement metadata.

## ZPA Config Overrides

Commands:

```sh
zscalerctl zpa config-overrides list
zscalerctl dump --products zpa --resources zpa/config-overrides --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/config-overrides (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `brokerName` | `string` | Tenant configuration | `standard`, `share` |
| `configKey` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `configValue` | `string` | Free text | `standard` |
| `configValueInt` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `customerId` | `string` | Sensitive identifier | `standard` |
| `customerName` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `targetGid` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `targetName` | `string` | Tenant configuration | `standard`, `share` |
| `targetType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `targetType`: Override target type.

## ZPA Access Policy Rules

Commands:

```sh
zscalerctl zpa access-policy-rules list
```

Each ZPA policy type is its own list resource with this one rule shape, read
from `/policySet/rules/policyType/<type>` (the endpoint behind the SDK's
`policysetcontroller.GetAllByType`). No policy-set create, update, reorder or
delete helper is bound. ZPA returns `policyType` as a numeric code (`1` for
access, `2` for timeout, `4` for client forwarding on the validation tenant),
not the type name.

In share mode a rule keeps its logic (operand `objectType`, `lhs`, condition
operators, order, actions) but not its operand values: `rhs` and `name` are
one field whether they hold an application id or a SCIM user attribute, so
they are standard only, and the CLI names them as omitted.

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/access-policy-rules (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `disabled` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `action` | `string` | Tenant configuration | `standard`, `share` |
| `actionId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policySetId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `ruleOrder` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `priority` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `operator` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `version` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `groupId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `defaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `defaultRuleName` | `string` | Tenant configuration | `standard`, `share` |
| `bypassDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `reauthDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `reauthIdleTimeout` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `reauthTimeout` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `lssDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `customMsg` | `string` | Free text | `standard` |
| `auditMessage` | `string` | Free text | `standard` |
| `extranetEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `devicePostureFailureNotificationEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnCbiProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnIsolationProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnInspectionProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnInspectionProfileName` | `string` | Tenant configuration | `standard`, `share` |
| `browserPostureProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `browserPostureName` | `string` | Tenant configuration | `standard`, `share` |
| `microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `readOnly` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `restrictionType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zscalerManaged` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `creationTime` | `string` | Operational metadata | `standard`, `share` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `conditions.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operator` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.negated` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `conditions.operands.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.objectType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.lhs` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.rhs` | `string` | Sensitive identifier | `standard` |
| `conditions.operands.name` | `string` | Sensitive identifier | `standard` |
| `conditions.operands.idpId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.operands.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.operands.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appServerGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `appServerGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appServerGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `appConnectorGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `appConnectorGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appConnectorGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `serviceEdgeGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `serviceEdgeGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `serviceEdgeGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `credential` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `credential.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `credential.name` | `string` | Tenant configuration | `standard`, `share` |
| `privilegedCapabilities` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.capabilities` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `privilegedCapabilities.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `privilegedCapabilities.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities.capabilities` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO.ziaLocations` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.ziaLocations.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.ziaLocations.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO.ziaLocations.enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.ziaErName` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.zpnErId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `postActions` |  | Secret | never (unstructured) |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `disabled`, `actionId`, `policySetId`, `policyType`, `ruleOrder`, `priority`, `operator`, `version`, `groupId`, `defaultRule`, `bypassDefaultRule`, `reauthDefaultRule`, `lssDefaultRule`, `extranetEnabled`, `devicePostureFailureNotificationEnabled`, `readOnly`, `restrictionType`, `zscalerManaged`, `modifiedBy`, `microtenantId`, `zpnCbiProfileId`, `zpnIsolationProfileId`, `zpnInspectionProfileId`, `browserPostureProfileId`: Rule state, order and link ids.
- `creationTime`, `modifiedTime`: Timestamps.
- `reauthTimeout`, `reauthIdleTimeout`: Timeout policy durations.
- `conditions.operator`, `conditions.negated`: How a rule's criteria combine.
- `conditions.operands.objectType`, `conditions.operands.lhs`, `conditions.operands.idpId`: What kind of criterion each operand is.
- `conditions.operands.rhs`, `conditions.operands.name`: The criterion's value: an object id or name, or a user, group, SAML or SCIM attribute value. Standard only, like the users and groups on ZIA rules.

The SDK also returns `desktopPolicyMappings` (PRA desktop images embedding
whole application segments); it is not rendered.

## ZPA Timeout Policy Rules

Commands:

```sh
zscalerctl zpa timeout-policy-rules list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/timeout-policy-rules (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `disabled` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `action` | `string` | Tenant configuration | `standard`, `share` |
| `actionId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policySetId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `ruleOrder` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `priority` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `operator` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `version` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `groupId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `defaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `defaultRuleName` | `string` | Tenant configuration | `standard`, `share` |
| `bypassDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `reauthDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `reauthIdleTimeout` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `reauthTimeout` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `lssDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `customMsg` | `string` | Free text | `standard` |
| `auditMessage` | `string` | Free text | `standard` |
| `extranetEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `devicePostureFailureNotificationEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnCbiProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnIsolationProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnInspectionProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnInspectionProfileName` | `string` | Tenant configuration | `standard`, `share` |
| `browserPostureProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `browserPostureName` | `string` | Tenant configuration | `standard`, `share` |
| `microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `readOnly` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `restrictionType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zscalerManaged` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `creationTime` | `string` | Operational metadata | `standard`, `share` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `conditions.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operator` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.negated` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `conditions.operands.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.objectType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.lhs` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.rhs` | `string` | Sensitive identifier | `standard` |
| `conditions.operands.name` | `string` | Sensitive identifier | `standard` |
| `conditions.operands.idpId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.operands.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.operands.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appServerGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `appServerGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appServerGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `appConnectorGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `appConnectorGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appConnectorGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `serviceEdgeGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `serviceEdgeGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `serviceEdgeGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `credential` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `credential.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `credential.name` | `string` | Tenant configuration | `standard`, `share` |
| `privilegedCapabilities` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.capabilities` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `privilegedCapabilities.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `privilegedCapabilities.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities.capabilities` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO.ziaLocations` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.ziaLocations.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.ziaLocations.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO.ziaLocations.enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.ziaErName` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.zpnErId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `postActions` |  | Secret | never (unstructured) |
<!-- END GENERATED FIELDS -->

The `TIMEOUT_POLICY` rules, with the fields of ZPA Access Policy Rules.

## ZPA Client Forwarding Policy Rules

Commands:

```sh
zscalerctl zpa client-forwarding-policy-rules list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/client-forwarding-policy-rules (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `disabled` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `action` | `string` | Tenant configuration | `standard`, `share` |
| `actionId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policySetId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `ruleOrder` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `priority` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `operator` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `version` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `groupId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `defaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `defaultRuleName` | `string` | Tenant configuration | `standard`, `share` |
| `bypassDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `reauthDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `reauthIdleTimeout` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `reauthTimeout` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `lssDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `customMsg` | `string` | Free text | `standard` |
| `auditMessage` | `string` | Free text | `standard` |
| `extranetEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `devicePostureFailureNotificationEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnCbiProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnIsolationProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnInspectionProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnInspectionProfileName` | `string` | Tenant configuration | `standard`, `share` |
| `browserPostureProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `browserPostureName` | `string` | Tenant configuration | `standard`, `share` |
| `microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `readOnly` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `restrictionType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zscalerManaged` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `creationTime` | `string` | Operational metadata | `standard`, `share` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `conditions.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operator` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.negated` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `conditions.operands.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.objectType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.lhs` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.rhs` | `string` | Sensitive identifier | `standard` |
| `conditions.operands.name` | `string` | Sensitive identifier | `standard` |
| `conditions.operands.idpId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.operands.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.operands.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appServerGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `appServerGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appServerGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `appConnectorGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `appConnectorGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appConnectorGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `serviceEdgeGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `serviceEdgeGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `serviceEdgeGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `credential` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `credential.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `credential.name` | `string` | Tenant configuration | `standard`, `share` |
| `privilegedCapabilities` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.capabilities` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `privilegedCapabilities.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `privilegedCapabilities.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities.capabilities` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO.ziaLocations` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.ziaLocations.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.ziaLocations.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO.ziaLocations.enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.ziaErName` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.zpnErId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `postActions` |  | Secret | never (unstructured) |
<!-- END GENERATED FIELDS -->

The `CLIENT_FORWARDING_POLICY` rules, with the fields of ZPA Access Policy Rules.

## ZPA Inspection Policy Rules

Commands:

```sh
zscalerctl zpa inspection-policy-rules list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/inspection-policy-rules (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `disabled` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `action` | `string` | Tenant configuration | `standard`, `share` |
| `actionId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policySetId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `ruleOrder` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `priority` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `operator` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `version` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `groupId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `defaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `defaultRuleName` | `string` | Tenant configuration | `standard`, `share` |
| `bypassDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `reauthDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `reauthIdleTimeout` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `reauthTimeout` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `lssDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `customMsg` | `string` | Free text | `standard` |
| `auditMessage` | `string` | Free text | `standard` |
| `extranetEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `devicePostureFailureNotificationEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnCbiProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnIsolationProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnInspectionProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnInspectionProfileName` | `string` | Tenant configuration | `standard`, `share` |
| `browserPostureProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `browserPostureName` | `string` | Tenant configuration | `standard`, `share` |
| `microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `readOnly` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `restrictionType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zscalerManaged` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `creationTime` | `string` | Operational metadata | `standard`, `share` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `conditions.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operator` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.negated` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `conditions.operands.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.objectType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.lhs` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.rhs` | `string` | Sensitive identifier | `standard` |
| `conditions.operands.name` | `string` | Sensitive identifier | `standard` |
| `conditions.operands.idpId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.operands.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.operands.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appServerGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `appServerGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appServerGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `appConnectorGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `appConnectorGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appConnectorGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `serviceEdgeGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `serviceEdgeGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `serviceEdgeGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `credential` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `credential.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `credential.name` | `string` | Tenant configuration | `standard`, `share` |
| `privilegedCapabilities` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.capabilities` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `privilegedCapabilities.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `privilegedCapabilities.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities.capabilities` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO.ziaLocations` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.ziaLocations.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.ziaLocations.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO.ziaLocations.enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.ziaErName` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.zpnErId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `postActions` |  | Secret | never (unstructured) |
<!-- END GENERATED FIELDS -->

The `INSPECTION_POLICY` rules, with the fields of ZPA Access Policy Rules.

## ZPA Isolation Policy Rules

Commands:

```sh
zscalerctl zpa isolation-policy-rules list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/isolation-policy-rules (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `disabled` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `action` | `string` | Tenant configuration | `standard`, `share` |
| `actionId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policySetId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `ruleOrder` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `priority` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `operator` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `version` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `groupId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `defaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `defaultRuleName` | `string` | Tenant configuration | `standard`, `share` |
| `bypassDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `reauthDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `reauthIdleTimeout` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `reauthTimeout` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `lssDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `customMsg` | `string` | Free text | `standard` |
| `auditMessage` | `string` | Free text | `standard` |
| `extranetEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `devicePostureFailureNotificationEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnCbiProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnIsolationProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnInspectionProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnInspectionProfileName` | `string` | Tenant configuration | `standard`, `share` |
| `browserPostureProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `browserPostureName` | `string` | Tenant configuration | `standard`, `share` |
| `microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `readOnly` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `restrictionType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zscalerManaged` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `creationTime` | `string` | Operational metadata | `standard`, `share` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `conditions.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operator` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.negated` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `conditions.operands.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.objectType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.lhs` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.rhs` | `string` | Sensitive identifier | `standard` |
| `conditions.operands.name` | `string` | Sensitive identifier | `standard` |
| `conditions.operands.idpId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.operands.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.operands.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appServerGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `appServerGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appServerGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `appConnectorGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `appConnectorGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appConnectorGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `serviceEdgeGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `serviceEdgeGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `serviceEdgeGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `credential` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `credential.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `credential.name` | `string` | Tenant configuration | `standard`, `share` |
| `privilegedCapabilities` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.capabilities` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `privilegedCapabilities.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `privilegedCapabilities.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities.capabilities` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO.ziaLocations` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.ziaLocations.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.ziaLocations.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO.ziaLocations.enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.ziaErName` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.zpnErId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `postActions` |  | Secret | never (unstructured) |
<!-- END GENERATED FIELDS -->

The `ISOLATION_POLICY` rules, with the fields of ZPA Access Policy Rules.

## ZPA Redirection Policy Rules

Commands:

```sh
zscalerctl zpa redirection-policy-rules list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/redirection-policy-rules (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `disabled` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `action` | `string` | Tenant configuration | `standard`, `share` |
| `actionId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policySetId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `ruleOrder` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `priority` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `operator` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `version` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `groupId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `defaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `defaultRuleName` | `string` | Tenant configuration | `standard`, `share` |
| `bypassDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `reauthDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `reauthIdleTimeout` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `reauthTimeout` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `lssDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `customMsg` | `string` | Free text | `standard` |
| `auditMessage` | `string` | Free text | `standard` |
| `extranetEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `devicePostureFailureNotificationEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnCbiProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnIsolationProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnInspectionProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnInspectionProfileName` | `string` | Tenant configuration | `standard`, `share` |
| `browserPostureProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `browserPostureName` | `string` | Tenant configuration | `standard`, `share` |
| `microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `readOnly` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `restrictionType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zscalerManaged` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `creationTime` | `string` | Operational metadata | `standard`, `share` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `conditions.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operator` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.negated` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `conditions.operands.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.objectType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.lhs` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.rhs` | `string` | Sensitive identifier | `standard` |
| `conditions.operands.name` | `string` | Sensitive identifier | `standard` |
| `conditions.operands.idpId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.operands.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.operands.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appServerGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `appServerGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appServerGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `appConnectorGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `appConnectorGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appConnectorGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `serviceEdgeGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `serviceEdgeGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `serviceEdgeGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `credential` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `credential.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `credential.name` | `string` | Tenant configuration | `standard`, `share` |
| `privilegedCapabilities` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.capabilities` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `privilegedCapabilities.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `privilegedCapabilities.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities.capabilities` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO.ziaLocations` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.ziaLocations.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.ziaLocations.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO.ziaLocations.enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.ziaErName` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.zpnErId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `postActions` |  | Secret | never (unstructured) |
<!-- END GENERATED FIELDS -->

The `REDIRECTION_POLICY` rules, with the fields of ZPA Access Policy Rules.

## ZPA Capabilities Policy Rules

Commands:

```sh
zscalerctl zpa capabilities-policy-rules list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/capabilities-policy-rules (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `disabled` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `action` | `string` | Tenant configuration | `standard`, `share` |
| `actionId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policySetId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `ruleOrder` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `priority` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `operator` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `version` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `groupId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `defaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `defaultRuleName` | `string` | Tenant configuration | `standard`, `share` |
| `bypassDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `reauthDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `reauthIdleTimeout` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `reauthTimeout` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `lssDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `customMsg` | `string` | Free text | `standard` |
| `auditMessage` | `string` | Free text | `standard` |
| `extranetEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `devicePostureFailureNotificationEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnCbiProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnIsolationProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnInspectionProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnInspectionProfileName` | `string` | Tenant configuration | `standard`, `share` |
| `browserPostureProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `browserPostureName` | `string` | Tenant configuration | `standard`, `share` |
| `microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `readOnly` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `restrictionType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zscalerManaged` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `creationTime` | `string` | Operational metadata | `standard`, `share` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `conditions.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operator` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.negated` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `conditions.operands.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.objectType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.lhs` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.rhs` | `string` | Sensitive identifier | `standard` |
| `conditions.operands.name` | `string` | Sensitive identifier | `standard` |
| `conditions.operands.idpId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.operands.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.operands.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appServerGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `appServerGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appServerGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `appConnectorGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `appConnectorGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appConnectorGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `serviceEdgeGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `serviceEdgeGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `serviceEdgeGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `credential` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `credential.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `credential.name` | `string` | Tenant configuration | `standard`, `share` |
| `privilegedCapabilities` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.capabilities` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `privilegedCapabilities.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `privilegedCapabilities.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities.capabilities` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO.ziaLocations` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.ziaLocations.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.ziaLocations.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO.ziaLocations.enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.ziaErName` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.zpnErId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `postActions` |  | Secret | never (unstructured) |
<!-- END GENERATED FIELDS -->

The `CAPABILITIES_POLICY` rules, with the fields of ZPA Access Policy Rules.

## ZPA Credential Policy Rules

Commands:

```sh
zscalerctl zpa credential-policy-rules list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/credential-policy-rules (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `disabled` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `action` | `string` | Tenant configuration | `standard`, `share` |
| `actionId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policySetId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `ruleOrder` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `priority` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `operator` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `version` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `groupId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `defaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `defaultRuleName` | `string` | Tenant configuration | `standard`, `share` |
| `bypassDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `reauthDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `reauthIdleTimeout` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `reauthTimeout` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `lssDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `customMsg` | `string` | Free text | `standard` |
| `auditMessage` | `string` | Free text | `standard` |
| `extranetEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `devicePostureFailureNotificationEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnCbiProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnIsolationProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnInspectionProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnInspectionProfileName` | `string` | Tenant configuration | `standard`, `share` |
| `browserPostureProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `browserPostureName` | `string` | Tenant configuration | `standard`, `share` |
| `microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `readOnly` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `restrictionType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zscalerManaged` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `creationTime` | `string` | Operational metadata | `standard`, `share` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `conditions.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operator` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.negated` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `conditions.operands.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.objectType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.lhs` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.rhs` | `string` | Sensitive identifier | `standard` |
| `conditions.operands.name` | `string` | Sensitive identifier | `standard` |
| `conditions.operands.idpId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.operands.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.operands.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appServerGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `appServerGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appServerGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `appConnectorGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `appConnectorGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appConnectorGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `serviceEdgeGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `serviceEdgeGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `serviceEdgeGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `credential` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `credential.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `credential.name` | `string` | Tenant configuration | `standard`, `share` |
| `privilegedCapabilities` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.capabilities` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `privilegedCapabilities.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `privilegedCapabilities.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities.capabilities` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO.ziaLocations` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.ziaLocations.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.ziaLocations.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO.ziaLocations.enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.ziaErName` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.zpnErId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `postActions` |  | Secret | never (unstructured) |
<!-- END GENERATED FIELDS -->

The `CREDENTIAL_POLICY` rules, with the fields of ZPA Access Policy Rules.

## ZPA Clientless Session Protection Policy Rules

Commands:

```sh
zscalerctl zpa clientless-session-protection-policy-rules list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/clientless-session-protection-policy-rules (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `disabled` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `action` | `string` | Tenant configuration | `standard`, `share` |
| `actionId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policySetId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `ruleOrder` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `priority` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `operator` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `version` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `groupId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `defaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `defaultRuleName` | `string` | Tenant configuration | `standard`, `share` |
| `bypassDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `reauthDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `reauthIdleTimeout` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `reauthTimeout` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `lssDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `customMsg` | `string` | Free text | `standard` |
| `auditMessage` | `string` | Free text | `standard` |
| `extranetEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `devicePostureFailureNotificationEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnCbiProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnIsolationProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnInspectionProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnInspectionProfileName` | `string` | Tenant configuration | `standard`, `share` |
| `browserPostureProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `browserPostureName` | `string` | Tenant configuration | `standard`, `share` |
| `microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `readOnly` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `restrictionType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zscalerManaged` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `creationTime` | `string` | Operational metadata | `standard`, `share` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `conditions.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operator` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.negated` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `conditions.operands.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.objectType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.lhs` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.rhs` | `string` | Sensitive identifier | `standard` |
| `conditions.operands.name` | `string` | Sensitive identifier | `standard` |
| `conditions.operands.idpId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.operands.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.operands.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appServerGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `appServerGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appServerGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `appConnectorGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `appConnectorGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appConnectorGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `serviceEdgeGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `serviceEdgeGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `serviceEdgeGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `credential` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `credential.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `credential.name` | `string` | Tenant configuration | `standard`, `share` |
| `privilegedCapabilities` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.capabilities` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `privilegedCapabilities.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `privilegedCapabilities.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities.capabilities` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO.ziaLocations` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.ziaLocations.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.ziaLocations.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO.ziaLocations.enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.ziaErName` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.zpnErId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `postActions` |  | Secret | never (unstructured) |
<!-- END GENERATED FIELDS -->

The `CLIENTLESS_SESSION_PROTECTION_POLICY` rules, with the fields of ZPA Access Policy Rules.

## ZPA SIEM Policy Rules

Commands:

```sh
zscalerctl zpa siem-policy-rules list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/siem-policy-rules (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `disabled` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `action` | `string` | Tenant configuration | `standard`, `share` |
| `actionId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policySetId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `ruleOrder` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `priority` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `operator` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `version` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `groupId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `defaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `defaultRuleName` | `string` | Tenant configuration | `standard`, `share` |
| `bypassDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `reauthDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `reauthIdleTimeout` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `reauthTimeout` | `string` | Tenant configuration | `standard`, `share`, `paranoid` |
| `lssDefaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `customMsg` | `string` | Free text | `standard` |
| `auditMessage` | `string` | Free text | `standard` |
| `extranetEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `devicePostureFailureNotificationEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnCbiProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnIsolationProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnInspectionProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnInspectionProfileName` | `string` | Tenant configuration | `standard`, `share` |
| `browserPostureProfileId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `browserPostureName` | `string` | Tenant configuration | `standard`, `share` |
| `microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `readOnly` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `restrictionType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zscalerManaged` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `creationTime` | `string` | Operational metadata | `standard`, `share` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `conditions.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operator` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.negated` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `conditions.operands.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.objectType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.lhs` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.rhs` | `string` | Sensitive identifier | `standard` |
| `conditions.operands.name` | `string` | Sensitive identifier | `standard` |
| `conditions.operands.idpId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditions.operands.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.operands.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `conditions.operands.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appServerGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `appServerGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appServerGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `appConnectorGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `appConnectorGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appConnectorGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `serviceEdgeGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `serviceEdgeGroups.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `serviceEdgeGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `credential` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `credential.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `credential.name` | `string` | Tenant configuration | `standard`, `share` |
| `privilegedCapabilities` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.capabilities` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedCapabilities.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `privilegedCapabilities.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `privilegedCapabilities.modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities.capabilities` | `array<string>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `privilegedPortalCapabilities.microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationDTO.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO.ziaLocations` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.ziaLocations.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.locationGroupDTO.ziaLocations.name` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.locationGroupDTO.ziaLocations.enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetDTO.ziaErName` | `string` | Tenant configuration | `standard`, `share` |
| `extranetDTO.zpnErId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `postActions` |  | Secret | never (unstructured) |
<!-- END GENERATED FIELDS -->

The `SIEM_POLICY` rules, with the fields of ZPA Access Policy Rules.

## ZPA Admins

Commands:

```sh
zscalerctl zpa admins list
zscalerctl zpa admins get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/admins (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `username` | `string` | Sensitive identifier | `standard` |
| `displayName` | `string` | Sensitive identifier | `standard` |
| `email` | `string` | Sensitive identifier | `standard` |
| `phoneNumber` | `string` | Sensitive identifier | `standard` |
| `comments` | `string` | Free text | `standard` |
| `isEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `isLocked` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `localLoginDisabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `forcePwdChange` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `pinSession` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `timezone` | `string` | Operational metadata | `standard` |
| `languageCode` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `eula` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `roleId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `role` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `role.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `groupIds` | `array<string>` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `syncVersion` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `deliveryTag` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `operationType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `password` |  | Secret | never (credential) |
| `tmpPassword` |  | Secret | never (credential) |
| `tokenId` |  | Secret | never (token) |
<!-- END GENERATED FIELDS -->

Field notes:

- `username`, `displayName`, `email`, `phoneNumber`: The administrator's identity and contact details; standard only.
- `roleId`, `role.id`: The admin role (`zpa/roles`).
- `password`, `tmpPassword`, `tokenId`: Credentials and the two-factor token; never rendered. The live list route did not return them.
- `timezone`: Standard only; a per-person location hint.

Two-factor state is not read: the live route omits `twoFactorAuthEnabled` and `twoFactorAuthType`, and a missing boolean would render as `false`, claiming 2FA is off. `get` fails unless the response is the requested admin, so a `null` or `{}` body never renders as a disabled account.

The administrators route answers one bare JSON array rather than the paged envelope the SDK's `GetAll` expects, so the list is a single unpaged read. It also returns `creationTime`, `modifiedTime`, `modifiedBy`, `customerId` and `isDeleted`, which the SDK struct does not model; they are not read. Because `isDeleted` is not read, a soft-deleted admin the route returns is not marked as one.

## ZPA Roles

Commands:

```sh
zscalerctl zpa roles list
zscalerctl zpa roles get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/roles (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `customRole` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `systemRole` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `restrictedRole` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `bypassRemoteAssistanceCheck` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `users` | `string` | Sensitive identifier | `standard` |
| `apiKeys` | `string` | Sensitive identifier | `standard` |
| `classPermissionGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `classPermissionGroups.id` | `string` | Operational metadata | `standard`, `share` |
| `classPermissionGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `classPermissionGroups.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `classPermissionGroups.modifiedBy` | `string` | Operational metadata | `standard`, `share` |
| `classPermissionGroups.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `classPermissionGroups.hidden` | `boolean` | Operational metadata | `standard`, `share` |
| `classPermissionGroups.internal` | `boolean` | Operational metadata | `standard`, `share` |
| `classPermissionGroups.localScopePermissionGroup` | `boolean` | Operational metadata | `standard`, `share` |
| `classPermissionGroups.classPermissions` | `array<object>` | Tenant configuration | `standard`, `share` |
| `classPermissionGroups.classPermissions.id` | `string` | Operational metadata | `standard`, `share` |
| `classPermissionGroups.classPermissions.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `classPermissionGroups.classPermissions.modifiedBy` | `string` | Operational metadata | `standard`, `share` |
| `classPermissionGroups.classPermissions.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `classPermissionGroups.classPermissions.permission` | `object` | Tenant configuration | `standard`, `share` |
| `classPermissionGroups.classPermissions.permission.type` | `string` | Tenant configuration | `standard`, `share` |
| `classPermissionGroups.classPermissions.permission.mask` | `string` | Operational metadata | `standard`, `share` |
| `classPermissionGroups.classPermissions.permission.maxMask` | `string` | Operational metadata | `standard`, `share` |
| `classPermissionGroups.classPermissions.classType` | `object` | Tenant configuration | `standard`, `share` |
| `classPermissionGroups.classPermissions.classType.id` | `string` | Operational metadata | `standard`, `share` |
| `classPermissionGroups.classPermissions.classType.aclClass` | `string` | Tenant configuration | `standard`, `share` |
| `classPermissionGroups.classPermissions.classType.friendlyName` | `string` | Tenant configuration | `standard`, `share` |
| `classPermissionGroups.classPermissions.classType.localScopeMask` | `string` | Operational metadata | `standard`, `share` |
| `classPermissionGroups.classPermissions.classType.customerId` | `string` | Sensitive identifier | `standard` |
| `classPermissionGroups.classPermissions.classType.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `classPermissionGroups.classPermissions.classType.modifiedBy` | `string` | Operational metadata | `standard`, `share` |
| `classPermissionGroups.classPermissions.classType.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `permissions` | `array<object>` | Tenant configuration | `standard`, `share` |
| `permissions.id` | `string` | Operational metadata | `standard`, `share` |
| `permissions.permissionMask` | `string` | Operational metadata | `standard`, `share` |
| `permissions.role` | `string` | Operational metadata | `standard`, `share` |
| `permissions.customerId` | `string` | Sensitive identifier | `standard` |
| `permissions.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `permissions.modifiedBy` | `string` | Operational metadata | `standard`, `share` |
| `permissions.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `permissions.classType` | `object` | Tenant configuration | `standard`, `share` |
| `permissions.classType.id` | `string` | Operational metadata | `standard`, `share` |
| `permissions.classType.aclClass` | `string` | Tenant configuration | `standard`, `share` |
| `permissions.classType.friendlyName` | `string` | Tenant configuration | `standard`, `share` |
| `permissions.classType.localScopeMask` | `string` | Operational metadata | `standard`, `share` |
| `permissions.classType.customerId` | `string` | Sensitive identifier | `standard` |
| `permissions.classType.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `permissions.classType.modifiedBy` | `string` | Operational metadata | `standard`, `share` |
| `permissions.classType.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `creationTime`, `modifiedBy`, `modifiedTime`, `microtenantId`, `customRole`, `systemRole`, `restrictedRole`, `bypassRemoteAssistanceCheck`: Role identity and type flags.
- `users`, `apiKeys`: Summaries the API returns as strings; they may name admins or API keys, never key material.
- `classPermissionGroups`, `permissions`: Zscaler-defined permission groups and classes with access masks and type (`FULL`, `VIEW_ONLY`); nested customer IDs are standard only.

The roles route answers one bare JSON array rather than the paged envelope the SDK's `GetAll` expects, so the list is a single unpaged read.

## ZPA Idps

Commands:

```sh
zscalerctl zpa idps list
zscalerctl zpa idps get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/idps (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `ssoType` | `array<string>` | Tenant configuration | `standard`, `share` |
| `scimEnabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableScimBasedPolicy` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `disableSamlBasedPolicy` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `forceAuth` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `reauthOnUserUpdate` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `redirectBinding` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `loginHint` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `useCustomSPMetadata` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `autoProvision` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableArbitraryAuthDomains` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `signSamlRequest` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `loginNameAttribute` | `string` | Tenant configuration | `standard`, `share` |
| `scimSharedSecretExists` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `adminSpSigningCertId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `userSpSigningCertId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `domainList` | `array<string>` | Sensitive identifier | `standard` |
| `idpEntityId` | `string` | Sensitive identifier | `standard` |
| `loginUrl` | `string` | Sensitive identifier | `standard` |
| `scimServiceProviderEndpoint` | `string` | Sensitive identifier | `standard` |
| `adminMetadata` | `object` | Sensitive identifier | `standard` |
| `adminMetadata.certificateUrl` | `string` | Sensitive identifier | `standard` |
| `adminMetadata.spBaseUrl` | `string` | Sensitive identifier | `standard` |
| `adminMetadata.spEntityId` | `string` | Sensitive identifier | `standard` |
| `adminMetadata.spMetadataUrl` | `string` | Sensitive identifier | `standard` |
| `adminMetadata.spPostUrl` | `string` | Sensitive identifier | `standard` |
| `userMetadata` | `object` | Sensitive identifier | `standard` |
| `userMetadata.certificateUrl` | `string` | Sensitive identifier | `standard` |
| `userMetadata.spBaseUrl` | `string` | Sensitive identifier | `standard` |
| `userMetadata.spEntityId` | `string` | Sensitive identifier | `standard` |
| `userMetadata.spMetadataUrl` | `string` | Sensitive identifier | `standard` |
| `userMetadata.spPostUrl` | `string` | Sensitive identifier | `standard` |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `enabled`, `scimEnabled`, `enableScimBasedPolicy`, `disableSamlBasedPolicy`, `forceAuth`, `reauthOnUserUpdate`, `redirectBinding`, `loginHint`, `useCustomSPMetadata`, `autoProvision`, `enableArbitraryAuthDomains`, `signSamlRequest`, `adminSpSigningCertId`, `userSpSigningCertId`, `creationTime`, `modifiedBy`, `modifiedTime`: IdP flags and signing-certificate IDs.
- `scimSharedSecretExists`: Whether a SCIM shared secret is set, never the secret.
- `name`, `ssoType`, `loginNameAttribute`: IdP name, user/admin SSO type and login attribute.
- `domainList`, `idpEntityId`, `loginUrl`, `scimServiceProviderEndpoint`, `adminMetadata`, `userMetadata`: Tenant domains, IdP entity ID and URLs, and ZPA service-provider metadata URLs that embed the customer ID.

## ZPA Saml Attributes

Commands:

```sh
zscalerctl zpa saml-attributes list
zscalerctl zpa saml-attributes get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/saml-attributes (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `samlName` | `string` | Sensitive identifier | `standard` |
| `idpId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `idpName` | `string` | Tenant configuration | `standard`, `share` |
| `userAttribute` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `delta` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `idpId`, `userAttribute`, `delta`, `creationTime`, `modifiedBy`, `modifiedTime`: `idpId` references `zpa/idps`.
- `name`, `idpName`: Attribute names policy rules match on.
- `samlName`: The SAML claim name; custom claims can embed an IdP org or tenant domain.

## ZPA Scim Groups

Commands:

```sh
zscalerctl zpa scim-groups list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/scim-groups (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `idpId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `idpName` | `string` | Tenant configuration | `standard`, `share` |
| `idpGroupId` | `string` | Sensitive identifier | `standard` |
| `internalId` | `string` | Sensitive identifier | `standard` |
| `iamIdpId` | `string` | Sensitive identifier | `standard` |
| `iamIdpName` | `string` | Tenant configuration | `standard`, `share` |
| `creationTime` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `idpId`, `creationTime`, `modifiedTime`: `idpId` references `zpa/idps`.
- `name`, `idpName`, `iamIdpName`: Group names, rendered in share like ZIA and Zidentity groups.
- `idpGroupId`, `internalId`, `iamIdpId`: External IdP and internal identifiers.

SCIM groups are listed per IdP; the list reads every IdP's groups in IdP order, and any IdP's failure fails the whole list.

## ZPA Scim Attributes

Commands:

```sh
zscalerctl zpa scim-attributes list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/scim-attributes (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `idpId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `dataType` | `string` | Tenant configuration | `standard`, `share` |
| `canonicalValues` | `array<string>` | Sensitive identifier | `standard` |
| `caseSensitive` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `multivalued` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `required` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `uniqueness` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `mutability` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `returned` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `schemaURI` | `string` | Sensitive identifier | `standard` |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `idpId`, `caseSensitive`, `multivalued`, `required`, `uniqueness`, `mutability`, `returned`, `creationTime`, `modifiedBy`, `modifiedTime`: SCIM schema properties; `idpId` references `zpa/idps`.
- `name`, `dataType`: Attribute name and type.
- `canonicalValues`, `schemaURI`: Allowed values the IdP declares (they can be tenant department or group names) and the SCIM schema URN (custom ones can embed an org name).

SCIM attributes are listed per IdP; the list reads every IdP's attributes in IdP order, and any IdP's failure fails the whole list.

## ZPA Branch Connector Groups

Commands:

```sh
zscalerctl zpa branch-connector-groups list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/branch-connector-groups (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `enabled`: Summary identifier and state.
- `name`: Branch connector group name.

An id/name summary view (`/branchConnectorGroup/summary`). `enabled` renders `false` when false and `null` when the API omits it.

## ZPA Extranet Partners

Commands:

```sh
zscalerctl zpa extranet-partners list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/extranet-partners (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Sensitive identifier | `standard` |
| `enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `enabled`: Summary identifier and state.
- `name`: The partner organization's name; not the tenant's own object.

## ZPA Location Summaries

Commands:

```sh
zscalerctl zpa location-summaries list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/location-summaries (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `enabled`: Summary identifier and state.
- `name`: Location name.

ZIA locations as ZPA summarizes them for extranet use (`/location/summary`).

## ZPA Workload Tag Groups

Commands:

```sh
zscalerctl zpa workload-tag-groups list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/workload-tag-groups (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `enabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `enabled`: Summary identifier and state.
- `name`: Workload tag group name.

## ZPA Dr Tool Versions

Commands:

```sh
zscalerctl zpa dr-tool-versions list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/dr-tool-versions (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `platform` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `version` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `latest` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `customerId` | `string` | Sensitive identifier | `standard` |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `platform`, `version`, `latest`, `creationTime`, `modifiedBy`, `modifiedTime`: Disaster-recovery tool builds available to the tenant.
- `name`: Build name.
- `customerId`: The tenant's customer ID.

## ZPA Managed Browser Profiles

Commands:

```sh
zscalerctl zpa managed-browser-profiles list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/managed-browser-profiles (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `browserType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `customerId` | `string` | Sensitive identifier | `standard` |
| `microtenantId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `microtenantName` | `string` | Tenant configuration | `standard`, `share` |
| `chromePostureProfile` | `object` | Tenant configuration | `standard`, `share` |
| `chromePostureProfile.id` | `string` | Operational metadata | `standard`, `share` |
| `chromePostureProfile.browserType` | `string` | Operational metadata | `standard`, `share` |
| `chromePostureProfile.crowdStrikeAgent` | `boolean` | Operational metadata | `standard`, `share` |
| `chromePostureProfile.creationTime` | `string` | Operational metadata | `standard`, `share` |
| `chromePostureProfile.modifiedBy` | `string` | Operational metadata | `standard`, `share` |
| `chromePostureProfile.modifiedTime` | `string` | Operational metadata | `standard`, `share` |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `browserType`, `microtenantId`, `creationTime`, `modifiedBy`, `modifiedTime`: Profile identity and browser type.
- `chromePostureProfile`: Chrome posture settings (browser type, CrowdStrike agent check).
- `customerId`: The tenant's customer ID.

## ZPA Client Settings

Commands:

```sh
zscalerctl zpa client-settings list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/client-settings (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `clientCertificateType` | `string` | Tenant configuration | `standard`, `share` |
| `enrollmentCertId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `enrollmentCertName` | `string` | Tenant configuration | `standard`, `share` |
| `singningCertExpiryInEpochSec` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `creationTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `modifiedBy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `enrollmentCertId`, `singningCertExpiryInEpochSec`, `creationTime`, `modifiedBy`: Client certificate settings; the enrollment certificate is referenced, never read.
- `name`, `clientCertificateType`, `enrollmentCertName`: Setting and certificate names.

The route answers an array, though the SDK's `GetAllClientSettings` decodes one object; it is read as one unpaged list.

## ZPA Admin Sso Options

Commands:

```sh
zscalerctl zpa admin-sso-options list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/admin-sso-options (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `ssologinonly` | `boolean` | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `ssologinonly`: Whether administrators must sign in with SSO; a response without it fails the read.

## ZPA Session Termination Settings

Commands:

```sh
zscalerctl zpa session-termination-settings list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/session-termination-settings (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `sessionTerminationOnReauth` | `boolean` | Tenant configuration | `standard`, `share` |
| `allowDisableSessionTerminationOnReauth` | `boolean` | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `sessionTerminationOnReauth`, `allowDisableSessionTerminationOnReauth`: Session termination on reauthentication; a response without the first fails the read.

The ZIA cloud configuration beside it holds ZIA credentials and is never requested (gap `zpa/zia-cloud-config`).

## ZPA Auth Domains

Commands:

```sh
zscalerctl zpa auth-domains list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/auth-domains (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `authDomains` | `array<string>` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `authDomains`: The tenant's authentication domains.

## ZPA Iam Idp Mapping

Commands:

```sh
zscalerctl zpa iam-idp-mapping list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/iam-idp-mapping (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `sequence` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `iamIdpIdToIamIdMapping` | `object` | Tenant configuration | `standard`, `share` |
| `iamIdpIdToIamIdMapping.deliveryTag` | `integer` | Operational metadata | `standard`, `share` |
| `iamIdpIdToIamIdMapping.orgId` | `string` | Sensitive identifier | `standard` |
| `iamIdpIdToIamIdMapping.mappings` | `array<object>` | Tenant configuration | `standard`, `share` |
| `iamIdpIdToIamIdMapping.mappings.iamIdpId` | `string` | Sensitive identifier | `standard` |
| `iamIdpIdToIamIdMapping.mappings.idpId` | `string` | Operational metadata | `standard`, `share` |
| `iamIdpIdToIamIdMapping.mappings.idpName` | `string` | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `sequence`: Mapping identity.
- `iamIdpIdToIamIdMapping`: ZPA IdP to Zidentity IdP mappings: `idpId` (references `zpa/idps`) and `idpName` render in share; `orgId` and `iamIdpId` are standard only.

## ZPA Step Up Auth Levels

Commands:

```sh
zscalerctl zpa step-up-auth-levels list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zpa/step-up-auth-levels (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `levels` | `array<string>` | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `levels`: Step-up authentication level names.

The route answers a bare string array; zscalerctl wraps it as one `levels` record.

## ZTW Activation Status

Commands:

```sh
zscalerctl ztw activation-status show
zscalerctl dump --products ztw --resources ztw/activation-status --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: ztw/activation-status (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `orgEditStatus` | `string` | Operational metadata | `standard`, `share` |
| `orgLastActivateStatus` | `string` | Operational metadata | `standard`, `share` |
| `adminStatusMap` |  | Secret | never (unstructured) |
| `adminActivateStatus` | `string` | Operational metadata | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `orgEditStatus`, `orgLastActivateStatus`, `adminActivateStatus`: EC admin activation status metadata.

The SDK also returns a per-admin status map (`adminStatusMap`); the catalog keeps it out of rendered output as unmodeled tenant-identifying detail.

## ZTW Workload Groups

Commands:

```sh
zscalerctl ztw workload-groups list
zscalerctl ztw workload-groups get <id>
zscalerctl dump --products ztw --resources ztw/workload-groups --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: ztw/workload-groups (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `expression` | `string` | Sensitive identifier | `standard` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share` |
| `lastModifiedBy` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
| `expressionJson` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `expressionJson.expressionContainers` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `expressionJson.expressionContainers.tagType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `expressionJson.expressionContainers.operator` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `expressionJson.expressionContainers.tagContainer` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `expressionJson.expressionContainers.tagContainer.operator` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `expressionJson.expressionContainers.tagContainer.tags` | `array<object>` | Tenant configuration | `standard` |
| `expressionJson.expressionContainers.tagContainer.tags.key` | `string` | Sensitive identifier | `standard` |
| `expressionJson.expressionContainers.tagContainer.tags.value` | `string` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: ZTW workload group identifier.

## ZTW Public Cloud Accounts

Commands:

```sh
zscalerctl ztw public-cloud-accounts list
zscalerctl ztw public-cloud-accounts get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: ztw/public-cloud-accounts (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `accountId` | `string` | Sensitive identifier | `standard` |
| `platformId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `platformId`: Internal record ID and cloud platform.
- `accountId`: Cloud account/subscription identifier; local-only.

## ZTW Dns Gateways

Commands:

```sh
zscalerctl ztw dns-gateways list
zscalerctl ztw dns-gateways get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: ztw/dns-gateways (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `dnsGatewayType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `ecDnsGatewayOptionsPrimary` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `ecDnsGatewayOptionsSecondary` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `failureBehavior` | `string` | Tenant configuration | `standard`, `share` |
| `primaryIp` | `string` | Sensitive identifier | `standard` |
| `secondaryIp` | `string` | Sensitive identifier | `standard` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share` |
| `lastModifiedBy` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
| `lastModifiedBy.externalId` | `string` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `dnsGatewayType`: Gateway identifier and type.
- `primaryIp`, `secondaryIp`: Local-only gateway addresses.

## ZTW Forwarding Gateways

Commands:

```sh
zscalerctl ztw forwarding-gateways list
zscalerctl ztw forwarding-gateways get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: ztw/forwarding-gateways (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `failClosed` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `manualPrimary` | `string` | Sensitive identifier | `standard` |
| `manualSecondary` | `string` | Sensitive identifier | `standard` |
| `subcloudPrimary` | `object` | Tenant configuration | `standard` |
| `subcloudPrimary.id` | `integer` | Operational metadata | `standard` |
| `subcloudPrimary.name` | `string` | Tenant configuration | `standard` |
| `subcloudPrimary.externalId` | `string` | Operational metadata | `standard` |
| `subcloudPrimary.extensions` |  | Secret | never (unstructured) |
| `subcloudSecondary` | `object` | Tenant configuration | `standard` |
| `subcloudSecondary.id` | `integer` | Operational metadata | `standard` |
| `subcloudSecondary.name` | `string` | Tenant configuration | `standard` |
| `subcloudSecondary.externalId` | `string` | Operational metadata | `standard` |
| `subcloudSecondary.extensions` |  | Secret | never (unstructured) |
| `primaryType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `secondaryType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `type` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `failureBehavior` | `string` | Tenant configuration | `standard`, `share` |
| `dnsGatewayType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `primaryIp` | `string` | Sensitive identifier | `standard` |
| `secondaryIp` | `string` | Sensitive identifier | `standard` |
| `ecDnsGatewayOptionsPrimary` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `ecDnsGatewayOptionsSecondary` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `failClosed`, `primaryType`, `secondaryType`, `type`, `dnsGatewayType`: Gateway identity and behavior flags.
- `manualPrimary`, `manualSecondary`, `primaryIp`, `secondaryIp`: Local-only gateway endpoints.
- `subcloudPrimary`, `subcloudSecondary`: Rendered as id/name references only; `externalId` and extensions are dropped.

## ZTW Ec Groups

Commands:

```sh
zscalerctl ztw ec-groups list
zscalerctl ztw ec-groups get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: ztw/ec-groups (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `desc` | `string` | Free text | `standard` |
| `deployType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `status` | `array<string>` | Operational metadata | `standard`, `share`, `paranoid` |
| `platform` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `awsAvailabilityZone` | `string` | Sensitive identifier | `standard` |
| `azureAvailabilityZone` | `string` | Sensitive identifier | `standard` |
| `maxEcCount` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `tunnelMode` | `string` | Tenant configuration | `standard`, `share` |
| `location` | `object` | Tenant configuration | `standard` |
| `location.id` | `integer` | Operational metadata | `standard` |
| `location.name` | `string` | Tenant configuration | `standard` |
| `location.externalId` | `string` | Operational metadata | `standard` |
| `location.extensions` |  | Secret | never (unstructured) |
| `provTemplate` | `object` | Tenant configuration | `standard` |
| `provTemplate.id` | `integer` | Operational metadata | `standard` |
| `provTemplate.name` | `string` | Tenant configuration | `standard` |
| `provTemplate.externalId` | `string` | Operational metadata | `standard` |
| `provTemplate.extensions` |  | Secret | never (unstructured) |
| `ecVMs` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `deployType`, `status`, `platform`, `maxEcCount`: EC group identity and state.
- `awsAvailabilityZone`, `azureAvailabilityZone`: Local-only cloud placement metadata.
- `location`, `provTemplate`: Rendered as id/name references only; `externalId` and extensions are dropped.

## ZTW IP Source Groups

Commands:

```sh
zscalerctl ztw ip-source-groups list
zscalerctl ztw ip-source-groups get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: ztw/ip-source-groups (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `ipAddresses` | `array<string>` | Sensitive identifier | `standard` |
| `creatorContext` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `isNonEditable` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `creatorContext`, `isNonEditable`: Group identity and immutable/predefined state.
- `ipAddresses`: Local-only source addresses.

## ZTW IP Destination Groups

Commands:

```sh
zscalerctl ztw ip-destination-groups list
zscalerctl ztw ip-destination-groups get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: ztw/ip-destination-groups (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `type` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `addresses` | `array<string>` | Sensitive identifier | `standard` |
| `ipCategories` | `array<string>` | Sensitive identifier | `standard` |
| `countries` | `array<string>` | Operational metadata | `standard`, `share` |
| `isNonEditable` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `addresses`, `ipCategories`: Local-only destination addresses and category references.

## ZTW IP Groups

Commands:

```sh
zscalerctl ztw ip-groups list
zscalerctl ztw ip-groups get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: ztw/ip-groups (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `ipAddresses` | `array<string>` | Sensitive identifier | `standard` |
| `creatorContext` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `isNonEditable` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `extranetIpPool` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `isPredefined` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `creatorContext`, `isNonEditable`, `extranetIpPool`, `isPredefined`: Group identity and immutable/predefined state.
- `ipAddresses`: Local-only addresses.

## ZTW Network Services

Commands:

```sh
zscalerctl ztw network-services list
zscalerctl ztw network-services get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: ztw/network-services (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `tag` | `string` | Tenant configuration | `standard`, `share` |
| `srcTcpPorts` | `array<object>` | Tenant configuration | `standard`, `share` |
| `srcTcpPorts.start` | `integer` | Operational metadata | `standard`, `share` |
| `srcTcpPorts.end` | `integer` | Operational metadata | `standard`, `share` |
| `destTcpPorts` | `array<object>` | Tenant configuration | `standard`, `share` |
| `destTcpPorts.start` | `integer` | Operational metadata | `standard`, `share` |
| `destTcpPorts.end` | `integer` | Operational metadata | `standard`, `share` |
| `srcUdpPorts` | `array<object>` | Tenant configuration | `standard`, `share` |
| `srcUdpPorts.start` | `integer` | Operational metadata | `standard`, `share` |
| `srcUdpPorts.end` | `integer` | Operational metadata | `standard`, `share` |
| `destUdpPorts` | `array<object>` | Tenant configuration | `standard`, `share` |
| `destUdpPorts.start` | `integer` | Operational metadata | `standard`, `share` |
| `destUdpPorts.end` | `integer` | Operational metadata | `standard`, `share` |
| `type` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `isNameL10nTag` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `creatorContext` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `type`, `isNameL10nTag`, `creatorContext`: Service identity and metadata.
- `srcTcpPorts`, `destTcpPorts`, `srcUdpPorts`, `destUdpPorts`: Port ranges are rendered as `start`/`end` pairs.

## ZTW Network Service Groups

Commands:

```sh
zscalerctl ztw network-service-groups list
zscalerctl ztw network-service-groups get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: ztw/network-service-groups (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `services` | `array<object>` | Tenant configuration | `standard` |
| `services.id` | `integer` | Operational metadata | `standard` |
| `services.name` | `string` | Tenant configuration | `standard` |
| `description` | `string` | Free text | `standard` |
| `creatorContext` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `creatorContext`: Group identity and source context.
- `services`: Rendered as id/name references only; service ports/details are owned by `ztw/network-services`.

## ZTW Admin Users

Commands:

```sh
zscalerctl ztw admin-users list
zscalerctl ztw admin-users get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: ztw/admin-users (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `loginName` | `string` | Sensitive identifier | `standard` |
| `userName` | `string` | Sensitive identifier | `standard` |
| `email` | `string` | Sensitive identifier | `standard` |
| `comments` | `string` | Free text | `standard` |
| `disabled` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `password` |  | Secret | never (credential) |
| `pwdLastModifiedTime` | `integer` | Operational metadata | `standard` |
| `isNonEditable` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `isPasswordLoginAllowed` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `isPasswordExpired` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `isAuditor` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `isSecurityReportCommEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `isServiceUpdateCommEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `isProductUpdateCommEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `isExecMobileAppEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `adminScopescopeGroupMemberEntities` | `array<object>` | Tenant configuration | `standard`, `share` |
| `adminScopescopeGroupMemberEntities.id` | `integer` | Operational metadata | `standard`, `share` |
| `adminScopescopeGroupMemberEntities.name` | `string` | Tenant configuration | `standard` |
| `adminScopescopeGroupMemberEntities.extensions` |  | Secret | never (unstructured) |
| `adminScopeScopeEntities` | `array<object>` | Tenant configuration | `standard`, `share` |
| `adminScopeScopeEntities.id` | `integer` | Operational metadata | `standard`, `share` |
| `adminScopeScopeEntities.name` | `string` | Tenant configuration | `standard` |
| `adminScopeScopeEntities.extensions` |  | Secret | never (unstructured) |
| `adminScopeType` | `string` | Tenant configuration | `standard`, `share` |
| `role` | `object` | Tenant configuration | `standard` |
| `role.id` | `integer` | Operational metadata | `standard` |
| `role.name` | `string` | Tenant configuration | `standard` |
| `role.isNameL10nTag` | `boolean` | Operational metadata | `standard` |
| `role.extensions` |  | Secret | never (unstructured) |
| `execMobileAppTokens` |  | Secret | never (token) |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `disabled`, `isNonEditable`, `isAuditor`: Admin account identity and broad state.
- `loginName`, `userName`, `email`: Person-identifying admin account fields. Visible to the local administrator, dropped from `share` and `paranoid` output.
- `pwdLastModifiedTime`: Password-age metadata is visible only in the local admin view.
- `isSecurityReportCommEnabled`, `isServiceUpdateCommEnabled`, `isProductUpdateCommEnabled`, `isExecMobileAppEnabled`, `adminScopeType`: Admin communication, app access, and scope-type settings.
- `role`: Role reference renders as id/name plus localization flag; extensions are dropped.

## ZTW Admin Roles

Commands:

```sh
zscalerctl ztw admin-roles list
zscalerctl ztw admin-roles get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: ztw/admin-roles (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `rank` | `integer` | Operational metadata | `standard` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `policyAccess` | `string` | Tenant configuration | `standard` |
| `alertingAccess` | `string` | Tenant configuration | `standard` |
| `dashboardAccess` | `string` | Tenant configuration | `standard` |
| `reportAccess` | `string` | Tenant configuration | `standard` |
| `analysisAccess` | `string` | Tenant configuration | `standard` |
| `usernameAccess` | `string` | Tenant configuration | `standard` |
| `adminAcctAccess` | `string` | Tenant configuration | `standard` |
| `deviceInfoAccess` | `string` | Tenant configuration | `standard` |
| `isAuditor` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `permissions` | `array<string>` | Tenant configuration | `standard` |
| `isNonEditable` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `logsLimit` | `string` | Tenant configuration | `standard` |
| `roleType` | `string` | Tenant configuration | `standard`, `share` |
| `featurePermissions` |  | Secret | never (unstructured) |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `isAuditor`, `isNonEditable`: Role identity and broad state.
- `name`, `roleType`: Share-safe role identity metadata.

## ZTW Locations

Commands:

```sh
zscalerctl ztw locations list
zscalerctl ztw locations get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: ztw/locations (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `parentId` | `integer` | Operational metadata | `standard` |
| `enforceBandwidthControl` | `boolean` | Tenant configuration | `standard` |
| `upBandwidth` | `integer` | Operational metadata | `standard` |
| `dnBandwidth` | `integer` | Operational metadata | `standard` |
| `overrideUpBandwidth` | `integer` | Operational metadata | `standard` |
| `overrideDnBandwidth` | `integer` | Operational metadata | `standard` |
| `sharedUpBandwidth` | `integer` | Operational metadata | `standard` |
| `sharedDownBandwidth` | `integer` | Operational metadata | `standard` |
| `unusedUpBandwidth` | `integer` | Operational metadata | `standard` |
| `country` | `string` | Tenant configuration | `standard` |
| `state` | `string` | Tenant configuration | `standard` |
| `language` | `string` | Tenant configuration | `standard` |
| `tz` | `string` | Tenant configuration | `standard` |
| `ipAddresses` | `array<string>` | Sensitive identifier | `standard` |
| `ports` | `array<integer>` | Tenant configuration | `standard` |
| `authRequired` | `boolean` | Tenant configuration | `standard` |
| `sslScanEnabled` | `boolean` | Tenant configuration | `standard` |
| `zappSSLScanEnabled` | `boolean` | Tenant configuration | `standard` |
| `xffForwardEnabled` | `boolean` | Tenant configuration | `standard` |
| `otherSubLocation` | `boolean` | Operational metadata | `standard` |
| `other6SubLocation` | `boolean` | Operational metadata | `standard` |
| `ecLocation` | `boolean` | Tenant configuration | `standard` |
| `surrogateIP` | `boolean` | Tenant configuration | `standard` |
| `idleTimeInMinutes` | `integer` | Operational metadata | `standard` |
| `displayTimeUnit` | `string` | Operational metadata | `standard` |
| `surrogateIPEnforcedForKnownBrowsers` | `boolean` | Tenant configuration | `standard` |
| `surrogateRefreshTimeInMinutes` | `integer` | Operational metadata | `standard` |
| `surrogateRefreshTimeUnit` | `string` | Operational metadata | `standard` |
| `ofwEnabled` | `boolean` | Tenant configuration | `standard` |
| `ipsControl` | `boolean` | Tenant configuration | `standard` |
| `aupEnabled` | `boolean` | Tenant configuration | `standard` |
| `cautionEnabled` | `boolean` | Tenant configuration | `standard` |
| `aupBlockInternetUntilAccepted` | `boolean` | Tenant configuration | `standard` |
| `aupForceSslInspection` | `boolean` | Tenant configuration | `standard` |
| `aupTimeoutInDays` | `integer` | Operational metadata | `standard` |
| `profile` | `string` | Tenant configuration | `standard` |
| `description` | `string` | Free text | `standard` |
| `ipv6Enabled` | `boolean` | Tenant configuration | `standard` |
| `ipv6Dns64Prefix` | `boolean` | Tenant configuration | `standard` |
| `kerberosAuth` | `boolean` | Tenant configuration | `standard` |
| `digestAuthEnabled` | `boolean` | Tenant configuration | `standard` |
| `childCount` | `integer` | Operational metadata | `standard` |
| `matchInChild` | `boolean` | Tenant configuration | `standard` |
| `excludeFromDynamicGroups` | `boolean` | Tenant configuration | `standard` |
| `excludeFromManualGroups` | `boolean` | Tenant configuration | `standard` |
| `vpnCredentials` |  | Secret | never (credential) |
| `virtualZens` | `array<object>` | Tenant configuration | `standard` |
| `virtualZens.id` | `integer` | Operational metadata | `standard` |
| `virtualZens.name` | `string` | Tenant configuration | `standard` |
| `virtualZens.externalId` | `string` | Operational metadata | `standard` |
| `virtualZens.extensions` |  | Secret | never (unstructured) |
| `virtualZenClusters` | `array<object>` | Tenant configuration | `standard` |
| `virtualZenClusters.id` | `integer` | Operational metadata | `standard` |
| `virtualZenClusters.name` | `string` | Tenant configuration | `standard` |
| `virtualZenClusters.externalId` | `string` | Operational metadata | `standard` |
| `virtualZenClusters.extensions` |  | Secret | never (unstructured) |
| `staticLocationGroups` | `array<object>` | Tenant configuration | `standard` |
| `staticLocationGroups.id` | `integer` | Operational metadata | `standard` |
| `staticLocationGroups.name` | `string` | Tenant configuration | `standard` |
| `staticLocationGroups.externalId` | `string` | Operational metadata | `standard` |
| `staticLocationGroups.extensions` |  | Secret | never (unstructured) |
| `dynamiclocationGroups` | `array<object>` | Tenant configuration | `standard` |
| `dynamiclocationGroups.id` | `integer` | Operational metadata | `standard` |
| `dynamiclocationGroups.name` | `string` | Tenant configuration | `standard` |
| `dynamiclocationGroups.externalId` | `string` | Operational metadata | `standard` |
| `dynamiclocationGroups.extensions` |  | Secret | never (unstructured) |
| `publicCloudAccountId` | `object` | Tenant configuration | `standard` |
| `publicCloudAccountId.id` | `integer` | Operational metadata | `standard` |
| `publicCloudAccountId.name` | `string` | Tenant configuration | `standard` |
| `vpcInfo` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `vpcInfo.cloudProvider` | `string` | Tenant configuration | `standard`, `share` |
| `vpcInfo.cloudMeta` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `vpcInfo.cloudMeta.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `vpcInfo.cloudMeta.name` | `string` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: ZTW location identifier.
- `parentId`, `upBandwidth`, `dnBandwidth`, `overrideUpBandwidth`, `overrideDnBandwidth`, `sharedUpBandwidth`, `sharedDownBandwidth`, `unusedUpBandwidth`, `otherSubLocation`, `other6SubLocation`, `idleTimeInMinutes`, `displayTimeUnit`, `surrogateRefreshTimeInMinutes`, `surrogateRefreshTimeUnit`, `aupTimeoutInDays`, `childCount`: Location hierarchy, counters, and timing/bandwidth values are local-admin only.
- `country`, `state`, `language`, `tz`, `ports`, `authRequired`, `sslScanEnabled`, `zappSSLScanEnabled`, `xffForwardEnabled`, `ecLocation`, `surrogateIP`, `surrogateIPEnforcedForKnownBrowsers`, `ofwEnabled`, `ipsControl`, `aupEnabled`, `cautionEnabled`, `aupBlockInternetUntilAccepted`, `aupForceSslInspection`, `profile`, `ipv6Enabled`, `ipv6Dns64Prefix`, `kerberosAuth`, `digestAuthEnabled`, `matchInChild`, `excludeFromDynamicGroups`, `excludeFromManualGroups`: Detailed location controls are visible to the local administrator and dropped from shared output.
- `ipAddresses`: Local-only location addresses.
- `virtualZens`, `virtualZenClusters`, `staticLocationGroups`, `dynamiclocationGroups`: Rendered as id/name/external-ID references only.
- `publicCloudAccountId`: Rendered as an id/name reference only.

## ZTW Location Templates

Commands:

```sh
zscalerctl ztw location-templates list
zscalerctl ztw location-templates get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: ztw/location-templates (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `desc` | `string` | Free text | `standard` |
| `template` | `object` | Tenant configuration | `standard` |
| `template.templatePrefix` | `string` | Tenant configuration | `standard` |
| `template.xffForwardEnabled` | `boolean` | Tenant configuration | `standard` |
| `template.authRequired` | `boolean` | Tenant configuration | `standard` |
| `template.cautionEnabled` | `boolean` | Tenant configuration | `standard` |
| `template.aupEnabled` | `boolean` | Tenant configuration | `standard` |
| `template.aupTimeoutInDays` | `integer` | Operational metadata | `standard` |
| `template.ofwEnabled` | `boolean` | Tenant configuration | `standard` |
| `template.ipsControl` | `boolean` | Tenant configuration | `standard` |
| `template.enforceBandwidthControl` | `boolean` | Tenant configuration | `standard` |
| `template.upBandwidth` | `integer` | Operational metadata | `standard` |
| `template.dnBandwidth` | `integer` | Operational metadata | `standard` |
| `template.displayTimeUnit` | `string` | Operational metadata | `standard` |
| `template.idleTimeInMinutes` | `integer` | Operational metadata | `standard` |
| `template.surrogateIPEnforcedForKnownBrowsers` | `boolean` | Tenant configuration | `standard` |
| `template.surrogateRefreshTimeUnit` | `string` | Operational metadata | `standard` |
| `template.surrogateRefreshTimeInMinutes` | `integer` | Operational metadata | `standard` |
| `template.surrogateIP` | `boolean` | Tenant configuration | `standard` |
| `template.editable` | `boolean` | Operational metadata | `standard` |
| `template.lastModUid` | `object` | Tenant configuration | `standard` |
| `template.lastModUid.id` | `integer` | Operational metadata | `standard` |
| `template.lastModUid.name` | `string` | Sensitive identifier | `standard` |
| `template.lastModUid.externalId` | `string` | Sensitive identifier | `standard` |
| `editable` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModTime` | `integer` | Operational metadata | `standard`, `share` |
| `lastModUid` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `lastModUid.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModUid.name` | `string` | Sensitive identifier | `standard` |
| `lastModUid.externalId` | `string` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `editable`: Template identity and editability flag.
- `template`: Nested template controls render only in local-admin output.

## ZTW Account Groups

Commands:

```sh
zscalerctl ztw account-groups list
zscalerctl ztw account-groups get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: ztw/account-groups (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `cloudType` | `string` | Tenant configuration | `standard`, `share` |
| `publicCloudAccounts` | `array<object>` | Tenant configuration | `standard`, `share` |
| `publicCloudAccounts.id` | `integer` | Operational metadata | `standard`, `share` |
| `publicCloudAccounts.name` | `string` | Tenant configuration | `standard` |
| `publicCloudAccounts.extensions` |  | Secret | never (unstructured) |
| `cloudConnectorGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `cloudConnectorGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `cloudConnectorGroups.name` | `string` | Tenant configuration | `standard` |
| `cloudConnectorGroups.extensions` |  | Secret | never (unstructured) |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Account group identifier.

## ZTW Public Cloud Info

Commands:

```sh
zscalerctl ztw public-cloud-info list
zscalerctl ztw public-cloud-info get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: ztw/public-cloud-info (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Sensitive identifier | `standard` |
| `cloudType` | `string` | Tenant configuration | `standard`, `share` |
| `externalId` | `string` | Sensitive identifier | `standard` |
| `lastModTime` | `integer` | Operational metadata | `standard`, `share` |
| `lastSyncTime` | `integer` | Operational metadata | `standard`, `share` |
| `accountGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `accountGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `accountGroups.name` | `string` | Tenant configuration | `standard` |
| `accountGroups.extensions` |  | Secret | never (unstructured) |
| `lastModUser` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `lastModUser.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModUser.name` | `string` | Sensitive identifier | `standard` |
| `lastModUser.externalId` | `string` | Sensitive identifier | `standard` |
| `regionStatus` | `array<object>` | Tenant configuration | `standard` |
| `regionStatus.id` | `integer` | Operational metadata | `standard` |
| `regionStatus.name` | `string` | Tenant configuration | `standard` |
| `regionStatus.cloudType` | `string` | Tenant configuration | `standard` |
| `regionStatus.status` | `boolean` | Operational metadata | `standard` |
| `supportedRegions` | `array<object>` | Tenant configuration | `standard` |
| `supportedRegions.id` | `integer` | Operational metadata | `standard` |
| `supportedRegions.name` | `string` | Tenant configuration | `standard` |
| `supportedRegions.cloudType` | `string` | Tenant configuration | `standard` |
| `accountDetails` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `accountDetails.name` | `string` | Sensitive identifier | `standard` |
| `accountDetails.awsAccountId` | `string` | Sensitive identifier | `standard` |
| `accountDetails.awsRoleName` | `string` | Sensitive identifier | `standard` |
| `accountDetails.cloudWatchGroupArn` | `string` | Sensitive identifier | `standard` |
| `accountDetails.eventBusName` | `string` | Sensitive identifier | `standard` |
| `accountDetails.externalId` | `string` | Sensitive identifier | `standard` |
| `accountDetails.logInfoType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `accountDetails.troubleShootingLogging` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `accountDetails.trustedAccountId` | `string` | Sensitive identifier | `standard` |
| `accountDetails.trustedRole` | `string` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Public cloud info identifier.
- `name`: Cloud account/display identifier; local-only.

## ZTW Zpa Application Segments

Commands:

```sh
zscalerctl ztw zpa-application-segments list
```

Fields:

<!-- BEGIN GENERATED FIELDS: ztw/zpa-application-segments (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `zpaId` | `integer` | Sensitive identifier | `standard` |
| `deleted` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `deleted`: Segment identity and deletion state.
- `zpaId`: Local-only ZPA tenant/application reference.

## ZTW Forwarding Rules

Commands:

```sh
zscalerctl ztw forwarding-rules list
zscalerctl ztw forwarding-rules get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: ztw/forwarding-rules (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `accessControl` | `string` | Operational metadata | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `type` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `order` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `rank` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardMethod` | `string` | Tenant configuration | `standard`, `share` |
| `defaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `wanSelection` | `string` | Tenant configuration | `standard`, `share` |
| `state` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `blockResponseCode` | `string` | Tenant configuration | `standard`, `share` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share` |
| `srcIps` | `array<string>` | Sensitive identifier | `standard` |
| `destAddresses` | `array<string>` | Sensitive identifier | `standard` |
| `destIpCategories` | `array<string>` | Sensitive identifier | `standard` |
| `resCategories` | `array<string>` | Sensitive identifier | `standard` |
| `destCountries` | `array<string>` | Operational metadata | `standard`, `share` |
| `sourceCountries` | `array<string>` | Operational metadata | `standard`, `share` |
| `nwApplications` | `array<string>` | Tenant configuration | `standard`, `share` |
| `sourceIpGroupExclusion` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpaBrokerRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `locations` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locations.id` | `integer` | Operational metadata | `standard`, `share` |
| `locations.name` | `string` | Tenant configuration | `standard` |
| `locations.extensions` |  | Secret | never (unstructured) |
| `locationGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locationGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `locationGroups.name` | `string` | Tenant configuration | `standard` |
| `locationGroups.extensions` |  | Secret | never (unstructured) |
| `ecGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `ecGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `ecGroups.name` | `string` | Tenant configuration | `standard` |
| `ecGroups.extensions` |  | Secret | never (unstructured) |
| `departments` | `array<object>` | Tenant configuration | `standard`, `share` |
| `departments.id` | `integer` | Operational metadata | `standard`, `share` |
| `departments.name` | `string` | Tenant configuration | `standard` |
| `departments.extensions` |  | Secret | never (unstructured) |
| `groups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `groups.id` | `integer` | Operational metadata | `standard`, `share` |
| `groups.name` | `string` | Tenant configuration | `standard` |
| `groups.extensions` |  | Secret | never (unstructured) |
| `users` | `array<object>` | Tenant configuration | `standard`, `share` |
| `users.id` | `integer` | Operational metadata | `standard`, `share` |
| `users.name` | `string` | Tenant configuration | `standard` |
| `users.extensions` |  | Secret | never (unstructured) |
| `lastModifiedBy` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
| `srcIpGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `srcIpGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `srcIpGroups.name` | `string` | Tenant configuration | `standard` |
| `srcIpGroups.extensions` |  | Secret | never (unstructured) |
| `srcIpv6Groups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `srcIpv6Groups.id` | `integer` | Operational metadata | `standard`, `share` |
| `srcIpv6Groups.name` | `string` | Tenant configuration | `standard` |
| `srcIpv6Groups.extensions` |  | Secret | never (unstructured) |
| `destIpGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `destIpGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `destIpGroups.name` | `string` | Tenant configuration | `standard` |
| `destIpGroups.extensions` |  | Secret | never (unstructured) |
| `destIpv6Groups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `destIpv6Groups.id` | `integer` | Operational metadata | `standard`, `share` |
| `destIpv6Groups.name` | `string` | Tenant configuration | `standard` |
| `destIpv6Groups.extensions` |  | Secret | never (unstructured) |
| `nwServices` | `array<object>` | Tenant configuration | `standard`, `share` |
| `nwServices.id` | `integer` | Operational metadata | `standard`, `share` |
| `nwServices.name` | `string` | Tenant configuration | `standard` |
| `nwServices.extensions` |  | Secret | never (unstructured) |
| `nwServiceGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `nwServiceGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `nwServiceGroups.name` | `string` | Tenant configuration | `standard` |
| `nwServiceGroups.extensions` |  | Secret | never (unstructured) |
| `labels` | `array<object>` | Tenant configuration | `standard`, `share` |
| `labels.id` | `integer` | Operational metadata | `standard`, `share` |
| `labels.name` | `string` | Tenant configuration | `standard`, `share` |
| `labels.extensions` |  | Secret | never (unstructured) |
| `nwApplicationGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `nwApplicationGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `nwApplicationGroups.name` | `string` | Tenant configuration | `standard` |
| `nwApplicationGroups.extensions` |  | Secret | never (unstructured) |
| `appServiceGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `appServiceGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `appServiceGroups.name` | `string` | Tenant configuration | `standard` |
| `appServiceGroups.extensions` |  | Secret | never (unstructured) |
| `srcWorkloadGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `srcWorkloadGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `srcWorkloadGroups.name` | `string` | Tenant configuration | `standard` |
| `srcWorkloadGroups.extensions` |  | Secret | never (unstructured) |
| `proxyGateway` | `object` | Tenant configuration | `standard` |
| `proxyGateway.id` | `integer` | Operational metadata | `standard` |
| `proxyGateway.name` | `string` | Tenant configuration | `standard` |
| `zpaApplicationSegments` | `array<object>` | Tenant configuration | `standard` |
| `zpaApplicationSegments.id` | `integer` | Operational metadata | `standard` |
| `zpaApplicationSegments.name` | `string` | Tenant configuration | `standard` |
| `zpaApplicationSegments.description` | `string` | Free text | `standard` |
| `zpaApplicationSegments.zpaId` | `integer` | Sensitive identifier | `standard` |
| `zpaApplicationSegments.deleted` | `boolean` | Operational metadata | `standard` |
| `zpaApplicationSegmentGroups` | `array<object>` | Tenant configuration | `standard` |
| `zpaApplicationSegmentGroups.id` | `integer` | Operational metadata | `standard` |
| `zpaApplicationSegmentGroups.name` | `string` | Tenant configuration | `standard` |
| `zpaApplicationSegmentGroups.zpaId` | `integer` | Sensitive identifier | `standard` |
| `zpaApplicationSegmentGroups.deleted` | `boolean` | Operational metadata | `standard` |
| `zpaApplicationSegmentGroups.zpaAppSegmentsCount` | `integer` | Operational metadata | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `srcIps`, `destAddresses`, `destIpCategories`, `resCategories`: Local-only network and resource criteria.

## ZTW Traffic Dns Rules

Commands:

```sh
zscalerctl ztw traffic-dns-rules list
zscalerctl ztw traffic-dns-rules get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: ztw/traffic-dns-rules (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `type` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `action` | `string` | Tenant configuration | `standard`, `share` |
| `order` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `rank` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `state` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `predefined` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `defaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `srcIps` | `array<string>` | Sensitive identifier | `standard` |
| `destAddresses` | `array<string>` | Sensitive identifier | `standard` |
| `locations` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locations.id` | `integer` | Operational metadata | `standard`, `share` |
| `locations.name` | `string` | Tenant configuration | `standard` |
| `locations.extensions` |  | Secret | never (unstructured) |
| `locationGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locationGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `locationGroups.name` | `string` | Tenant configuration | `standard` |
| `locationGroups.extensions` |  | Secret | never (unstructured) |
| `ecGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `ecGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `ecGroups.name` | `string` | Tenant configuration | `standard` |
| `ecGroups.extensions` |  | Secret | never (unstructured) |
| `srcIpGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `srcIpGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `srcIpGroups.name` | `string` | Tenant configuration | `standard` |
| `srcIpGroups.extensions` |  | Secret | never (unstructured) |
| `destIpGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `destIpGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `destIpGroups.name` | `string` | Tenant configuration | `standard` |
| `destIpGroups.extensions` |  | Secret | never (unstructured) |
| `dnsGateway` | `object` | Tenant configuration | `standard` |
| `dnsGateway.id` | `integer` | Operational metadata | `standard` |
| `dnsGateway.name` | `string` | Tenant configuration | `standard` |
| `zpaIpGroup` | `object` | Tenant configuration | `standard` |
| `zpaIpGroup.id` | `integer` | Operational metadata | `standard` |
| `zpaIpGroup.name` | `string` | Tenant configuration | `standard` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share` |
| `lastModifiedBy` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `srcIps`, `destAddresses`: Local-only network criteria.

## ZTW Traffic Log Rules

Commands:

```sh
zscalerctl ztw traffic-log-rules list
zscalerctl ztw traffic-log-rules get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: ztw/traffic-log-rules (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `order` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `rank` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `state` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `type` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardMethod` | `string` | Tenant configuration | `standard`, `share` |
| `defaultRule` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `locations` | `array<object>` | Tenant configuration | `standard`, `share` |
| `locations.id` | `integer` | Operational metadata | `standard`, `share` |
| `locations.name` | `string` | Tenant configuration | `standard` |
| `locations.extensions` |  | Secret | never (unstructured) |
| `proxyGateway` | `object` | Tenant configuration | `standard` |
| `proxyGateway.id` | `integer` | Operational metadata | `standard` |
| `proxyGateway.name` | `string` | Tenant configuration | `standard` |
| `lastModifiedTime` | `integer` | Operational metadata | `standard`, `share` |
| `lastModifiedBy` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `lastModifiedBy.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `lastModifiedBy.name` | `string` | Sensitive identifier | `standard` |
| `ecGroups` | `array<object>` | Tenant configuration | `standard`, `share` |
| `ecGroups.id` | `integer` | Operational metadata | `standard`, `share` |
| `ecGroups.name` | `string` | Tenant configuration | `standard` |
| `ecGroups.extensions` |  | Secret | never (unstructured) |
<!-- END GENERATED FIELDS -->

## ZCC Fail Open Policy

Commands:

```sh
zscalerctl zcc fail-open-policy list
zscalerctl dump --products zcc --resources zcc/fail-open-policy --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zcc/fail-open-policy (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `active` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableFailOpen` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableCaptivePortalDetection` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `captivePortalWebSecDisableMinutes` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableStrictEnforcementPrompt` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `strictEnforcementPromptDelayMinutes` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `strictEnforcementPromptMessage` | `string` | Free text | `standard` |
| `enableWebSecOnProxyUnreachable` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableWebSecOnTunnelFailure` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `tunnelFailureRetryCount` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `createdBy` | `string` | Sensitive identifier | `standard` |
| `editedBy` | `string` | Sensitive identifier | `standard` |
| `companyId` | `string` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `active`, `enableFailOpen`, `enableCaptivePortalDetection`, `captivePortalWebSecDisableMinutes`, `enableStrictEnforcementPrompt`, `strictEnforcementPromptDelayMinutes`, `enableWebSecOnProxyUnreachable`, `enableWebSecOnTunnelFailure`, `tunnelFailureRetryCount`: Fail-open policy state and settings.

This is a ZCC OneAPI reachability probe (`/zcc/papi/public/v1/webFailOpenPolicy`). ZCC coverage expands only after a live-smoke pass confirms the `/zcc/papi/` route is reachable on the tenant.


## ZCC Forwarding Profiles

Commands:

```sh
zscalerctl zcc forwarding-profiles list
zscalerctl dump --products zcc --resources zcc/forwarding-profiles --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zcc/forwarding-profiles (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `active` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditionType` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableLWFDriver` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `predefinedTnAll` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableUnifiedTunnel` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableAllDefaultAdaptersTN` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableSplitVpnTN` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `evaluateTrustedNetwork` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `skipTrustedCriteriaMatch` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `dnsServers` | `string` | Sensitive identifier | `standard` |
| `dnsSearchDomains` | `string` | Sensitive identifier | `standard` |
| `hostname` | `string` | Sensitive identifier | `standard` |
| `resolvedIpsForHostname` | `string` | Sensitive identifier | `standard` |
| `trustedSubnets` | `string` | Sensitive identifier | `standard` |
| `trustedGateways` | `string` | Sensitive identifier | `standard` |
| `trustedDhcpServers` | `string` | Sensitive identifier | `standard` |
| `trustedEgressIps` | `string` | Sensitive identifier | `standard` |
| `predefinedTrustedNetworks` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `trustedNetworkIds` | `array<integer>` | Operational metadata | `standard`, `share`, `paranoid` |
| `trustedNetworks` | `array<string>` | Tenant configuration | `standard`, `share` |
| `trustedNetworkIdsSelected` | `array<integer>` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.networkType` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.actionType` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.systemProxy` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.enablePacketTunnel` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.primaryTransport` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.DTLSTimeout` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.UDPTimeout` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.TLSTimeout` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.mtuForZadapter` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.blockUnreachableDomainsTraffic` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.allowTLSFallback` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.tunnel2FallbackType` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.sendAllDNSToTrustedServer` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.dropIpv6Traffic` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.redirectWebTraffic` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.dropIpv6IncludeTrafficInT2` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.useTunnel2ForProxiedWebTraffic` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.useTunnel2ForUnencryptedWebTraffic` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.pathMtuDiscovery` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.latencyBasedZenEnablement` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.zenProbeInterval` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.zenProbeSampleSize` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.zenThresholdLimit` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.dropIpv6TrafficInIpv6Network` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.optimiseForUnstableConnections` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.latencyBasedServerEnablement` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.lbsProbeInterval` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.lbsProbeSampleSize` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.lbsThresholdLimit` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.latencyBasedServerMTEnablement` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.isSameAsOnTrustedNetwork` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.customPac` | `string` | Sensitive identifier | `standard` |
| `forwardingProfileActions.systemProxyData` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.systemProxyData.proxyAction` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.systemProxyData.enableAutoDetect` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.systemProxyData.enablePAC` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.systemProxyData.enableProxyServer` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.systemProxyData.bypassProxyForPrivateIP` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.systemProxyData.performGPUpdate` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileActions.systemProxyData.pacURL` | `string` | Sensitive identifier | `standard` |
| `forwardingProfileActions.systemProxyData.proxyServerAddress` | `string` | Sensitive identifier | `standard` |
| `forwardingProfileActions.systemProxyData.proxyServerPort` | `string` | Sensitive identifier | `standard` |
| `forwardingProfileActions.systemProxyData.pacDataPath` | `string` | Sensitive identifier | `standard` |
| `forwardingProfileZpaActions` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `forwardingProfileZpaActions.networkType` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileZpaActions.actionType` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileZpaActions.primaryTransport` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileZpaActions.DTLSTimeout` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileZpaActions.TLSTimeout` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileZpaActions.mtuForZadapter` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileZpaActions.sendTrustedNetworkResultToZpa` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileZpaActions.latencyBasedServerEnablement` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileZpaActions.lbsProbeInterval` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileZpaActions.lbsProbeSampleSize` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileZpaActions.lbsThresholdLimit` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileZpaActions.latencyBasedServerMTEnablement` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileZpaActions.isSameAsOnTrustedNetwork` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileZpaActions.partnerInfo` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `forwardingProfileZpaActions.partnerInfo.primaryTransport` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileZpaActions.partnerInfo.allowTlsFallback` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileZpaActions.partnerInfo.mtuForZadapter` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `unifiedTunnel` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `unifiedTunnel.networkType` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `unifiedTunnel.actionTypeZIA` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `unifiedTunnel.actionTypeZPA` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `unifiedTunnel.primaryTransport` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `unifiedTunnel.DTLSTimeout` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `unifiedTunnel.TLSTimeout` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `unifiedTunnel.mtuForZadapter` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `unifiedTunnel.allowTLSFallback` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `unifiedTunnel.pathMtuDiscovery` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `unifiedTunnel.optimiseForUnstableConnections` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `unifiedTunnel.tunnel2FallbackType` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `unifiedTunnel.redirectWebTraffic` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `unifiedTunnel.dropIpv6Traffic` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `unifiedTunnel.dropIpv6TrafficInIpv6Network` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `unifiedTunnel.blockUnreachableDomainsTraffic` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `unifiedTunnel.dropIpv6IncludeTrafficInT2` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `unifiedTunnel.sendAllDNSToTrustedServer` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `unifiedTunnel.sameAsOnTrusted` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `unifiedTunnel.systemProxyData` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `unifiedTunnel.systemProxyData.proxyAction` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `unifiedTunnel.systemProxyData.enableAutoDetect` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `unifiedTunnel.systemProxyData.enablePAC` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `unifiedTunnel.systemProxyData.enableProxyServer` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `unifiedTunnel.systemProxyData.bypassProxyForPrivateIP` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `unifiedTunnel.systemProxyData.performGPUpdate` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `unifiedTunnel.systemProxyData.pacURL` | `string` | Sensitive identifier | `standard` |
| `unifiedTunnel.systemProxyData.proxyServerAddress` | `string` | Sensitive identifier | `standard` |
| `unifiedTunnel.systemProxyData.proxyServerPort` | `string` | Sensitive identifier | `standard` |
| `unifiedTunnel.systemProxyData.pacDataPath` | `string` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

## ZCC Trusted Networks

Commands:

```sh
zscalerctl zcc trusted-networks list
zscalerctl dump --products zcc --resources zcc/trusted-networks --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zcc/trusted-networks (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `active` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `conditionType` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `networkName` | `string` | Tenant configuration | `standard`, `share` |
| `dnsSearchDomains` | `string` | Sensitive identifier | `standard` |
| `dnsServers` | `string` | Sensitive identifier | `standard` |
| `hostnames` | `string` | Sensitive identifier | `standard` |
| `resolvedIpsForHostname` | `string` | Sensitive identifier | `standard` |
| `ssids` | `string` | Sensitive identifier | `standard` |
| `trustedDhcpServers` | `string` | Sensitive identifier | `standard` |
| `trustedEgressIps` | `string` | Sensitive identifier | `standard` |
| `trustedGateways` | `string` | Sensitive identifier | `standard` |
| `trustedSubnets` | `string` | Sensitive identifier | `standard` |
| `guid` | `string` | Sensitive identifier | `standard` |
| `companyId` | `string` | Sensitive identifier | `standard` |
| `createdBy` | `string` | Sensitive identifier | `standard` |
| `editedBy` | `string` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

## ZCC Web App Services

Commands:

```sh
zscalerctl zcc web-app-services list
zscalerctl dump --products zcc --resources zcc/web-app-services --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zcc/web-app-services (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `appName` | `string` | Tenant configuration | `standard`, `share` |
| `active` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `appVersion` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `appSvcId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `version` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `editedTimestamp` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `uid` | `string` | Sensitive identifier | `standard` |
| `appDataBlob` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `appDataBlob.proto` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appDataBlob.port` | `string` | Sensitive identifier | `standard` |
| `appDataBlob.ipaddr` | `string` | Sensitive identifier | `standard` |
| `appDataBlob.fqdn` | `string` | Sensitive identifier | `standard` |
| `appDataBlobV6` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `appDataBlobV6.proto` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appDataBlobV6.port` | `string` | Sensitive identifier | `standard` |
| `appDataBlobV6.ipaddr` | `string` | Sensitive identifier | `standard` |
| `appDataBlobV6.fqdn` | `string` | Sensitive identifier | `standard` |
| `zappDataBlob` |  | Secret | never (opaque blob) |
| `zappDataBlobV6` |  | Secret | never (opaque blob) |
| `createdBy` | `string` | Sensitive identifier | `standard` |
| `editedBy` | `string` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

## ZCC Application Profiles

Commands:

```sh
zscalerctl zcc application-profiles list
zscalerctl dump --products zcc --resources zcc/application-profiles --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zcc/application-profiles (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `deviceType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `pac_url` | `string` | Sensitive identifier | `standard` |
| `active` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `ruleOrder` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `logMode` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `logLevel` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `logFileSize` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `reauth_period` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `reactivateWebSecurityMinutes` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `highlightActiveControl` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `sendDisableServiceReason` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `refreshKerberosToken` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableDeviceGroups` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `onNetPolicy` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `notificationTemplateId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfileId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `ziaPostureConfigId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `tunnelZappTraffic` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `groupAll` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `showVPNTunNotification` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `useTunnelSDK4_3` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `ipv6Mode` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `passcode` |  | Secret | never (credential) |
| `logout_password` |  | Secret | never (credential) |
| `disable_password` |  | Secret | never (credential) |
| `uninstall_password` |  | Secret | never (credential) |
| `policyToken` |  | Secret | never (token) |
| `groupIds` | `array<string>` | Operational metadata | `standard`, `share`, `paranoid` |
| `deviceGroupIds` | `array<string>` | Operational metadata | `standard`, `share`, `paranoid` |
| `userIds` | `array<string>` | Operational metadata | `standard`, `share`, `paranoid` |
| `bypassAppIds` | `array<string>` | Operational metadata | `standard`, `share`, `paranoid` |
| `appServiceIds` | `array<string>` | Operational metadata | `standard`, `share`, `paranoid` |
| `bypassCustomAppIds` | `array<string>` | Operational metadata | `standard`, `share`, `paranoid` |
| `groups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `groups.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `groups.name` | `string` | Tenant configuration | `standard`, `share` |
| `groups.authType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `groups.active` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `groups.lastModification` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `deviceGroups` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `deviceGroups.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `deviceGroups.name` | `string` | Tenant configuration | `standard`, `share` |
| `deviceGroups.authType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `deviceGroups.active` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `deviceGroups.lastModification` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `users` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `users.id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `users.loginName` | `string` | Sensitive identifier | `standard` |
| `users.active` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `users.lastModification` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `users.companyId` | `string` | Sensitive identifier | `standard` |
| `appServices` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `appServices.active` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `appServices.appDataBlob` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `appServices.appDataBlob.port` | `string` | Sensitive identifier | `standard` |
| `appServices.appDataBlob.ipaddr` | `string` | Sensitive identifier | `standard` |
| `appServices.appDataBlob.fqdn` | `string` | Sensitive identifier | `standard` |
| `policyExtension` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `policyExtension.useV8JsEngine` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.followRoutingTable` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.useWsaPollForZpa` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.useDefaultAdapterForDNS` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.useZscalerNotificationFramework` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.switchFocusToNotification` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.fallbackToGatewayDomain` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.enableZCCRevert` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.zpaAuthExpOnSleep` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.zpaAuthExpOnSysRestart` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.zpaAuthExpOnNetIpChange` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.instantForceZPAReauthStateUpdate` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.zpaAuthExpOnWinLogonSession` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.zpaAuthExpOnWinSessionLock` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.zpaAuthExpSessionLockStateMinTimeInSecond` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.enableSetProxyOnVPNAdapters` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.disableDNSRouteExclusion` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.advanceZpaReauth` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.useProxyPortForT1` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.useProxyPortForT2` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.allowPacExclusionsOnly` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.interceptZIATrafficAllAdapters` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.enableAntiTampering` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.overrideATCmdByPolicy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.reactivateAntiTamperingTime` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.enforceSplitDNS` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.dropQuicTraffic` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.enableZdpService` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.updateDnsSearchOrder` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.truncateLargeUDPDNSResponse` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.prioritizeDnsExclusions` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.purgeKerberosPreferredDCCache` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.deleteDHCPOption121Routes` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.enableLocationPolicyOverride` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.enableCustomTheme` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.zccFailCloseSettingsLockdownOnTunnelProcessExit` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.zccFailCloseSettingsLockdownOnFirewallError` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.zccFailCloseSettingsLockdownOnDriverError` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.zccAppFailOpenPolicy` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.zccTunnelFailPolicy` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.followGlobalForPartnerLogin` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.userAllowedToAddPartner` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.allowClientCertCachingForWebView2` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.showConfirmationDialogForCachedCert` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.enableFlowBasedTunnel` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.enableNetworkTrafficProcessMapping` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.enableLocalPacketCapture` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.oneIdMTDeviceAuthEnabled` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.enableCustomProxyDetection` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.preventAutoReauthDuringDeviceLock` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.useEndPointLocationForDCSelection` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.enableCrashReporting` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.recacheSystemProxy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.enableAutomaticPacketCapture` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.enableAPCforCriticalSections` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.enableAPCforOtherSections` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.enablePCAdditionalSpace` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.pcAdditionalSpace` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.clientConnectorUiLanguage` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.blockPrivateRelay` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.bypassDNSTrafficUsingUDPProxy` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.reconnectTunOnWakeup` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.browserAuthType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.useDefaultBrowser` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.exitPassword` |  | Secret | never (credential) |
| `policyExtension.zdxDisablePassword` |  | Secret | never (credential) |
| `policyExtension.zdDisablePassword` |  | Secret | never (credential) |
| `policyExtension.zpaDisablePassword` |  | Secret | never (credential) |
| `policyExtension.zdpDisablePassword` |  | Secret | never (credential) |
| `policyExtension.zccRevertPassword` |  | Secret | never (credential) |
| `policyExtension.zccFailCloseSettingsExitUninstallPassword` |  | Secret | never (credential) |
| `policyExtension.sourcePortBasedBypasses` | `string` | Sensitive identifier | `standard` |
| `policyExtension.packetTunnelExcludeList` | `string` | Sensitive identifier | `standard` |
| `policyExtension.packetTunnelIncludeList` | `string` | Sensitive identifier | `standard` |
| `policyExtension.packetTunnelExcludeListForIPv6` | `string` | Sensitive identifier | `standard` |
| `policyExtension.packetTunnelIncludeListForIPv6` | `string` | Sensitive identifier | `standard` |
| `policyExtension.customDNS` | `string` | Sensitive identifier | `standard` |
| `policyExtension.zccFailCloseSettingsIpBypasses` | `string` | Sensitive identifier | `standard` |
| `policyExtension.zccFailCloseSettingsThumbPrint` | `string` | Operational metadata | `standard` |
| `policyExtension.zdxLiteConfigObj` |  | Secret | never (opaque blob) |
| `policyExtension.ddilConfig` |  | Secret | never (opaque blob) |
| `policyExtension.locationRulesetPolicies` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `policyExtension.locationRulesetPolicies.offTrusted` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `policyExtension.locationRulesetPolicies.offTrusted.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.locationRulesetPolicies.offTrusted.name` | `string` | Tenant configuration | `standard`, `share` |
| `policyExtension.locationRulesetPolicies.trusted` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `policyExtension.locationRulesetPolicies.trusted.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.locationRulesetPolicies.trusted.name` | `string` | Tenant configuration | `standard`, `share` |
| `policyExtension.locationRulesetPolicies.vpnTrusted` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `policyExtension.locationRulesetPolicies.vpnTrusted.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.locationRulesetPolicies.vpnTrusted.name` | `string` | Tenant configuration | `standard`, `share` |
| `policyExtension.locationRulesetPolicies.splitVpnTrusted` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `policyExtension.locationRulesetPolicies.splitVpnTrusted.id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.locationRulesetPolicies.splitVpnTrusted.name` | `string` | Tenant configuration | `standard`, `share` |
| `policyExtension.generateCliPasswordContract` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `policyExtension.generateCliPasswordContract.policyId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.generateCliPasswordContract.enableCli` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.generateCliPasswordContract.allowZpaDisableWithoutPassword` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.generateCliPasswordContract.allowZiaDisableWithoutPassword` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyExtension.generateCliPasswordContract.allowZdxDisableWithoutPassword` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `disasterRecovery` | `object` | Tenant configuration | `standard`, `share`, `paranoid` |
| `disasterRecovery.policyId` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `disasterRecovery.enableZiaDR` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `disasterRecovery.enableZpaDR` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `disasterRecovery.ziaDRMethod` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `disasterRecovery.useZiaGlobalDb` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `disasterRecovery.allowZiaTest` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `disasterRecovery.allowZpaTest` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `disasterRecovery.ziaCustomDbUrl` | `string` | Sensitive identifier | `standard` |
| `disasterRecovery.ziaGlobalDbUrl` | `string` | Sensitive identifier | `standard` |
| `disasterRecovery.ziaGlobalDbUrlv2` | `string` | Sensitive identifier | `standard` |
| `disasterRecovery.ziaDomainName` | `string` | Sensitive identifier | `standard` |
| `disasterRecovery.zpaDomainName` | `string` | Sensitive identifier | `standard` |
| `disasterRecovery.ziaRSAPubKeyName` | `string` | Tenant configuration | `standard`, `share` |
| `disasterRecovery.zpaRSAPubKeyName` | `string` | Tenant configuration | `standard`, `share` |
| `disasterRecovery.ziaRSAPubKey` | `string` | Operational metadata | `standard` |
| `disasterRecovery.zpaRSAPubKey` | `string` | Operational metadata | `standard` |
| `notificationTemplateContract` |  | Secret | never (unstructured) |
| `ziaPostureConfig` |  | Secret | never (unstructured) |
| `bypassApps` |  | Secret | never (unstructured) |
| `bypassCustomApps` |  | Secret | never (unstructured) |
<!-- END GENERATED FIELDS -->

Field notes:

- `refreshKerberosToken`: Operational control flag.

## ZCC Custom IP Apps

Commands:

```sh
zscalerctl zcc custom-ip-apps list
zscalerctl dump --products zcc --resources zcc/custom-ip-apps --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zcc/custom-ip-apps (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `appName` | `string` | Tenant configuration | `standard`, `share` |
| `active` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `uid` | `string` | Sensitive identifier | `standard` |
| `appDataBlob` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `appDataBlob.proto` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appDataBlob.port` | `string` | Sensitive identifier | `standard` |
| `appDataBlob.ipaddr` | `string` | Sensitive identifier | `standard` |
| `appDataBlob.fqdn` | `string` | Sensitive identifier | `standard` |
| `appDataBlobV6` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `appDataBlobV6.proto` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appDataBlobV6.port` | `string` | Sensitive identifier | `standard` |
| `appDataBlobV6.ipaddr` | `string` | Sensitive identifier | `standard` |
| `appDataBlobV6.fqdn` | `string` | Sensitive identifier | `standard` |
| `zappDataBlob` |  | Secret | never (opaque blob) |
| `zappDataBlobV6` |  | Secret | never (opaque blob) |
| `createdBy` | `string` | Sensitive identifier | `standard` |
| `editedBy` | `string` | Sensitive identifier | `standard` |
| `editedTimestamp` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

## ZCC Predefined IP Apps

Commands:

```sh
zscalerctl zcc predefined-ip-apps list
zscalerctl dump --products zcc --resources zcc/predefined-ip-apps --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zcc/predefined-ip-apps (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `appName` | `string` | Tenant configuration | `standard`, `share` |
| `active` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `appVersion` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `appSvcId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `uid` | `string` | Sensitive identifier | `standard` |
| `appDataBlob` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `appDataBlob.proto` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appDataBlob.port` | `string` | Sensitive identifier | `standard` |
| `appDataBlob.ipaddr` | `string` | Sensitive identifier | `standard` |
| `appDataBlob.fqdn` | `string` | Sensitive identifier | `standard` |
| `appDataBlobV6` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `appDataBlobV6.proto` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appDataBlobV6.port` | `string` | Sensitive identifier | `standard` |
| `appDataBlobV6.ipaddr` | `string` | Sensitive identifier | `standard` |
| `appDataBlobV6.fqdn` | `string` | Sensitive identifier | `standard` |
| `zappDataBlob` |  | Secret | never (opaque blob) |
| `zappDataBlobV6` |  | Secret | never (opaque blob) |
| `createdBy` | `string` | Sensitive identifier | `standard` |
| `editedBy` | `string` | Sensitive identifier | `standard` |
| `editedTimestamp` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

## ZCC Process Based Apps

Commands:

```sh
zscalerctl zcc process-based-apps list
zscalerctl dump --products zcc --resources zcc/process-based-apps --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zcc/process-based-apps (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `appName` | `string` | Tenant configuration | `standard`, `share` |
| `matchingCriteria` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `fileNames` | `array<string>` | Sensitive identifier | `standard` |
| `filePaths` | `array<string>` | Sensitive identifier | `standard` |
| `signaturePayload` | `string` | Operational metadata | `standard` |
| `certificatePayload` | `string` | Operational metadata | `standard` |
| `createdBy` | `string` | Sensitive identifier | `standard` |
| `editedBy` | `string` | Sensitive identifier | `standard` |
| `editedTimestamp` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

## ZCC Devices

Commands:

```sh
zscalerctl zcc devices list
zscalerctl dump --products zcc --resources zcc/devices --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zcc/devices (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `agentVersion` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `config_download_time` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `deregistrationTimestamp` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `download_count` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `keepAliveTime` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `last_seen_time` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `osVersion` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyName` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `registrationState` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `registration_time` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `state` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `tunnelVersion` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `type` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `upmVersion` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `vpnState` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `zappArch` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `manufacturer` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `companyName` | `string` | Sensitive identifier | `standard` |
| `machineHostname` | `string` | Sensitive identifier | `standard` |
| `macAddress` | `string` | Sensitive identifier | `standard` |
| `owner` | `string` | Sensitive identifier | `standard` |
| `user` | `string` | Sensitive identifier | `standard` |
| `udid` | `string` | Sensitive identifier | `standard` |
| `hardwareFingerprint` | `string` | Sensitive identifier | `standard` |
| `detail` | `string` | Free text | `standard` |
<!-- END GENERATED FIELDS -->

## ZCC Admin Roles

Commands:

```sh
zscalerctl zcc admin-roles list
zscalerctl dump --products zcc --resources zcc/admin-roles --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zcc/admin-roles (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `roleName` | `string` | Tenant configuration | `standard`, `share` |
| `isEditable` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `adminManagement` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `administratorGroup` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `androidProfile` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appBypass` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `appProfileGroup` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `auditLogs` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `authSetting` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `clientConnectorAppStore` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `clientConnectorIdp` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `clientConnectorNotifications` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `clientConnectorSupport` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `dashboard` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `ddilConfiguration` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `dedicatedProxyPorts` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `deviceGroups` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `deviceOverview` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `devicePosture` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `enrolledDevicesGroup` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `forwardingProfile` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `iosProfile` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `linuxProfile` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `macProfile` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `machineTunnel` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `obfuscateData` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `partnerDeviceOverview` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `publicApi` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `trustedNetwork` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `userAgent` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `windowsProfile` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpaPartnerLogin` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zscalerDeception` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zscalerEntitlement` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `companyId` | `string` | Sensitive identifier | `standard` |
| `createdBy` | `string` | Sensitive identifier | `standard` |
| `updatedBy` | `string` | Sensitive identifier | `standard` |
<!-- END GENERATED FIELDS -->

## ZCC Company Info

Commands:

```sh
zscalerctl zcc company-info show
zscalerctl dump --products zcc --resources zcc/company-info --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zcc/company-info (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `orgId` | `string` | Sensitive identifier | `standard` |
| `masterCustomerId` | `string` | Sensitive identifier | `standard` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `businessName` | `string` | Tenant configuration | `standard`, `share` |
| `businessContactNumber` | `string` | Sensitive identifier | `standard` |
| `activationRecipient` | `string` | Sensitive identifier | `standard` |
| `activationCopy` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `mdmStatus` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `sendEmail` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `proxyEnabled` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpnEnabled` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `upmEnabled` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zadEnabled` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableDeceptionForAll` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `dlpEnabled` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `tunnelProtocolType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `secureAgentBasic` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `secureAgentAdvanced` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `supportAdminEmail` | `string` | Sensitive identifier | `standard` |
| `supportEnabled` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `fetchLogsForAdminsEnabled` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableRectifyUtils` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `supportTicketEnabled` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `disableLoggingControls` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `defaultAuthType` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `version` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `policyActivationRequired` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableAutofillUsername` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `autoFillUsingLoginHint` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `dcServiceReadOnly` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableTunnelZappTrafficToggle` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `machineIdpAuth` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `linuxVisibility` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `registryPathForPac` | `string` | Sensitive identifier | `standard` |
| `usePollsetForSocketReactor` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableDtlsForZpa` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `useV8JsEngine` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `disableParallelIpv4AndIPv6` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `send64BitBuild` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `useAddIfscopeRoute` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `useClearArpCache` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `useDnsPriorityOrdering` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableBrowserAuth` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `enablePublicAPI` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `disableReasonVisibility` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `followRoutingTable` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `useDefaultAdapterForDNS` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableMinimumDeviceCleanupAsOne` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `dnsPriorityOrderingForTrustedDnsCriteria` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `machineTunnelPosture` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpaPartnerLogin` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `proxyPort` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `dnsCacheTtlWindows` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `dnsCacheTtlMac` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `dnsCacheTtlAndroid` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `dnsCacheTtlIos` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `dnsCacheTtlLinux` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpaClientCertExpInDays` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableFlowLogger` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `flowLoggingBufferLimit` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `flowLoggingTimeInterval` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `postureBasedService` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `enablePostureBasedProfile` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `disasterRecovery` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `ziaGlobalDbUrlForDR` | `string` | Sensitive identifier | `standard` |
| `enableReactUI` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `launchReactUIbyDefault` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `dlpNotification` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `vpnGatewayCharLimit` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `deviceGroupsCount` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `vpnBypassRefreshInterval` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `destIncludeExcludeCharLimit` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `ipV6SupportForTunnel2` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `destIncludeExcludeCharLimitForIpv6` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableSetProxyOnVPNAdapters` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `disableDNSRouteExclusion` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `showVPNTunNotification` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `addAppBypassToVPNGateway` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableZscalerFirewall` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `persistentZscalerFirewall` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `clearMupCache` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `executeGpoUpdate` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `enablePortBasedZPAFilter` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableAntiTampering` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpaReauthEnabled` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `zpaAutoReauthTimeout` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableZpaAuthUserName` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableGlobalZCCTelemetry` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `configureTunnel2fallbackForZia` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableInstallWebView2` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableCustomProxyPorts` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `interceptZIATrafficAllAdapters` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `swaggerLink` | `string` | Sensitive identifier | `standard` |
| `enableOneIdAdmin` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableOneIdUser` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `restrictAdminAccess` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableZiaUserDepartmentSync` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `enableUDPTransportSelection` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `computeDeviceGroupsForZIA` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `computeDeviceGroupsForZPA` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `computeDeviceGroupsForZDX` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `computeDeviceGroupsForZAD` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `useTunnel2SmeForTunnel1` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `maCloudName` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `ziaCloudName` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `zt2HealthProbeInterval` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `devicePostureFrequency` | `array<object>` | Tenant configuration | `standard`, `share`, `paranoid` |
| `devicePostureFrequency.postureId` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `devicePostureFrequency.iosValue` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `devicePostureFrequency.androidValue` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `devicePostureFrequency.windowsValue` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `devicePostureFrequency.macValue` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `devicePostureFrequency.linuxValue` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `devicePostureFrequency.defaultValue` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `devicePostureFrequency.postureName` | `string` | Tenant configuration | `standard`, `share` |
| `zdxManualRollout` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `winZdxLiteEnabled` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `telemetryDefault` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `devicePostureFrequency`: Per-posture check frequency: `postureId` and the per-OS values (`iosValue`, `androidValue`, `windowsValue`, `macValue`, `linuxValue`, `defaultValue`) in every mode; `postureName` in `standard` and `share`.

The SDK also returns `webAppConfig`, about 185 admin-portal feature toggles.
It is not tenant policy and is not in the catalog.

## ZCC Web Privacy

Commands:

```sh
zscalerctl zcc web-privacy list
```

Fields:

<!-- BEGIN GENERATED FIELDS: zcc/web-privacy (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `active` | `string` | Tenant configuration | `standard`, `share` |
| `collectUserInfo` | `string` | Tenant configuration | `standard`, `share` |
| `collectMachineHostname` | `string` | Tenant configuration | `standard`, `share` |
| `collectZdxLocation` | `string` | Tenant configuration | `standard`, `share` |
| `enablePacketCapture` | `string` | Tenant configuration | `standard`, `share` |
| `disableCrashlytics` | `string` | Tenant configuration | `standard`, `share` |
| `overrideT2ProtocolSetting` | `string` | Tenant configuration | `standard`, `share` |
| `restrictRemotePacketCapture` | `string` | Tenant configuration | `standard`, `share` |
| `grantAccessToZscalerLogFolder` | `string` | Tenant configuration | `standard`, `share` |
| `exportLogsForNonAdmin` | `string` | Tenant configuration | `standard`, `share` |
| `enableAutoLogSnippet` | `string` | Tenant configuration | `standard`, `share` |
| `enforceSecurePacUrls` | `string` | Tenant configuration | `standard`, `share` |
| `enableFQDNMatchForVpnBypasses` | `string` | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Setting identifier.
- `active`, `collectUserInfo`, `collectMachineHostname`, `collectZdxLocation`, `enablePacketCapture`, `disableCrashlytics`, `overrideT2ProtocolSetting`, `restrictRemotePacketCapture`, `grantAccessToZscalerLogFolder`, `exportLogsForNonAdmin`, `enableAutoLogSnippet`, `enforceSecurePacUrls`, `enableFQDNMatchForVpnBypasses`: Client Connector privacy and diagnostics switches, as the API's string flags.

## ZIDENTITY Groups

Commands:

```sh
zscalerctl zidentity groups list
zscalerctl zidentity groups get <id>
zscalerctl dump --products zidentity --resources zidentity/groups --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zidentity/groups (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `source` | `string` | Tenant configuration | `standard`, `share` |
| `isDynamicGroup` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `dynamicGroup` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `adminEntitlementEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `serviceEntitlementEnabled` | `boolean` | Tenant configuration | `standard`, `share` |
| `idp` | `object` | Tenant configuration | `standard`, `share` |
| `idp.id` | `string` | Operational metadata | `standard`, `share` |
| `idp.name` | `string` | Tenant configuration | `standard`, `share` |
| `idp.displayName` | `string` | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `isDynamicGroup`, `dynamicGroup`: Group identity and dynamic-group flags.
- `name`, `source`, `adminEntitlementEnabled`, `serviceEntitlementEnabled`, `idp`: Workforce directory and entitlement-toggle metadata. `idp` renders id/name/displayName only.

## ZIDENTITY Users

Commands:

```sh
zscalerctl zidentity users list
zscalerctl zidentity users get <id>
zscalerctl dump --products zidentity --resources zidentity/users --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zidentity/users (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `source` | `string` | Tenant configuration | `standard`, `share` |
| `loginName` | `string` | Sensitive identifier | `standard` |
| `displayName` | `string` | Sensitive identifier | `standard` |
| `firstName` | `string` | Sensitive identifier | `standard` |
| `lastName` | `string` | Sensitive identifier | `standard` |
| `primaryEmail` | `string` | Sensitive identifier | `standard` |
| `secondaryEmail` | `string` | Sensitive identifier | `standard` |
| `status` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `department` | `object` | Tenant configuration | `standard`, `share` |
| `department.id` | `string` | Operational metadata | `standard`, `share` |
| `department.name` | `string` | Tenant configuration | `standard`, `share` |
| `department.displayName` | `string` | Tenant configuration | `standard`, `share` |
| `idp` | `object` | Tenant configuration | `standard`, `share` |
| `idp.id` | `string` | Operational metadata | `standard`, `share` |
| `idp.name` | `string` | Tenant configuration | `standard`, `share` |
| `idp.displayName` | `string` | Tenant configuration | `standard`, `share` |
| `customAttrsInfo` |  | Secret | never (unstructured) |
| `title` | `string` | Sensitive identifier | `standard` |
| `division` | `string` | Sensitive identifier | `standard` |
| `costCenter` | `string` | Sensitive identifier | `standard` |
| `organization` | `string` | Sensitive identifier | `standard` |
| `skipMfaUntil` | `integer` | Tenant configuration | `standard` |
| `guest` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `belongInternalDomain` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `hostedIdp` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `entityType` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `groups` | `array<object>` | Tenant configuration | `standard` |
| `groups.id` | `string` | Operational metadata | `standard` |
| `groups.name` | `string` | Tenant configuration | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `status`: User identifier and enabled/status flag.
- `source`, `department`, `idp`: Workforce directory context visible to Zidentity administrators. `department` and `idp` render id/name/displayName only.
- `loginName`, `displayName`, `firstName`, `lastName`, `primaryEmail`, `secondaryEmail`: Person-identifying workforce directory fields. Visible to the local administrator, dropped from `share` and `paranoid` output.

## ZIDENTITY Resource Servers

Commands:

```sh
zscalerctl zidentity resource-servers list
zscalerctl zidentity resource-servers get <id>
zscalerctl dump --products zidentity --resources zidentity/resource-servers --output ./scratch-live-smoke
```

Fields:

<!-- BEGIN GENERATED FIELDS: zidentity/resource-servers (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `displayName` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `primaryAud` | `string` | Sensitive identifier | `standard` |
| `defaultApi` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `serviceScopes` | `array<object>` | Tenant configuration | `standard` |
| `serviceScopes.service` | `object` | Tenant configuration | `standard` |
| `serviceScopes.service.id` | `string` | Operational metadata | `standard` |
| `serviceScopes.service.name` | `string` | Tenant configuration | `standard` |
| `serviceScopes.service.displayName` | `string` | Tenant configuration | `standard` |
| `serviceScopes.service.externalName` | `string` | Tenant configuration | `standard` |
| `serviceScopes.service.cloudName` | `string` | Sensitive identifier | `standard` |
| `serviceScopes.service.orgName` | `string` | Sensitive identifier | `standard` |
| `serviceScopes.scopes` | `array<object>` | Tenant configuration | `standard` |
| `serviceScopes.scopes.id` | `string` | Operational metadata | `standard` |
| `serviceScopes.scopes.name` | `string` | Tenant configuration | `standard` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `defaultApi`: Resource server identifier and default API flag.
- `primaryAud`: OAuth audience value; dropped from `share` and `paranoid`.
- `serviceScopes`: Rendered as service and scope id/name references only. Service `cloudName` and `orgName` are dropped pending product-specific review.

## ZIDENTITY Api Clients

Commands:

```sh
zscalerctl zidentity api-clients list
zscalerctl zidentity api-clients get <id>
```

Fields:

<!-- BEGIN GENERATED FIELDS: zidentity/api-clients (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` | `string` | Operational metadata | `standard`, `share`, `paranoid` |
| `name` | `string` | Tenant configuration | `standard`, `share` |
| `description` | `string` | Free text | `standard` |
| `status` | `boolean` | Operational metadata | `standard`, `share`, `paranoid` |
| `accessTokenLifeTime` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |
| `clientAuthentication` | `object` | Tenant configuration | `standard`, `share` |
| `clientAuthentication.authType` | `string` | Tenant configuration | `standard`, `share` |
| `clientAuthentication.clientJWKsUrl` | `string` | Sensitive identifier | `standard` |
| `clientAuthentication.publicKeys` | `array<object>` | Tenant configuration | `standard` |
| `clientAuthentication.publicKeys.keyName` | `string` | Sensitive identifier | `standard` |
| `clientAuthentication.publicKeys.keyValue` |  | Secret | never (opaque blob) |
| `clientResources` | `array<object>` | Tenant configuration | `standard`, `share` |
| `clientResources.id` | `string` | Operational metadata | `standard`, `share` |
| `clientResources.name` | `string` | Tenant configuration | `standard`, `share` |
| `clientResources.defaultApi` | `boolean` | Operational metadata | `standard`, `share` |
| `clientResources.selectedScopes` | `array<object>` | Tenant configuration | `standard`, `share` |
| `clientResources.selectedScopes.id` | `string` | Operational metadata | `standard`, `share` |
| `clientResources.selectedScopes.name` | `string` | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `status`, `accessTokenLifeTime`: Client state and access-token lifetime (seconds).
- `name`: Client name.
- `clientAuthentication`: `authType` renders in share; `clientJWKsUrl` and public-key names are standard only; public-key values and client certificates (not cataloged) never render.
- `clientResources`: Resource servers and scopes the client may call.

Client secrets (`GetSecrets`) are never requested.

## Adding A Resource

Before enabling another resource:

- Start with `scripts/scaffold-resource.sh --product <zia|zpa|ztw|zidentity> --resource
  <name> --package <sdk-package> --type <sdk-type>` to create a review bundle
  under `scratch/resource-drafts/`. The bundle wraps `catalog-draft.go`, adds
  reader notes, and does not mutate production files.
- Treat the generated scaffold as a fail-closed starting point: only approved
  global field names render by default, while ambiguous names such as `value`,
  `key`, `data`, `content`, and `metadata` stay `secret` unless modeled with
  resource-specific context.
- Map the SDK response shape into source records without using the reader as a
  second safety allow-list.
- Classify every candidate output field in the catalog.
- Mark known secret-bearing fields as `secret`, even when they are expected to
  be dropped.
- Treat free-text fields as standard-only exceptions. Each emitted free-text
  field must carry a catalog `standard_free_text_reason`, must not be allowed in
  `share` or `paranoid`, and must retain scanner backstop coverage.
- Add canary tests proving secret-looking names or descriptions are redacted,
  including bare high-entropy tokens for emitted string fields.
- Add nested drop tests for any SDK object that contains user, admin, key,
  token, credential, or free-text data.
- Confirm `AssertRenderedSubset` runs before rendering and dump writing.
- Update this reference and the shell completion tests.

## ZDX Apps

Commands:

```sh
zscalerctl zdx apps report --since 2h
zscalerctl zdx apps report <id> --from 2026-10-26T10:00:00Z --to 2026-10-26T12:00:00Z
```

Fields:

<!-- BEGIN GENERATED FIELDS: zdx/apps (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `name` |  | Tenant configuration | `standard`, `share` |
| `score` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `total_users` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `most_impacted_region` |  | Tenant configuration | `standard`, `share` |
| `most_impacted_region.id` |  | Sensitive identifier | `standard` |
| `most_impacted_region.city` |  | Sensitive identifier | `standard` |
| `most_impacted_region.region` |  | Tenant configuration | `standard`, `share` |
| `most_impacted_region.country` |  | Tenant configuration | `standard`, `share` |
| `most_impacted_region.geo_type` |  | Operational metadata | `standard`, `share` |
| `stats` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `stats.active_users` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `stats.active_devices` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `stats.num_poor` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `stats.num_okay` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `stats.num_good` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `score_trend` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `score_trend.metric` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `score_trend.unit` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `score_trend.datapoints` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `score_trend.datapoints.timestamp` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `score_trend.datapoints.value` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `metrics` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `metrics.metric` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `metrics.unit` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `metrics.datapoints` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `metrics.datapoints.timestamp` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `metrics.datapoints.value` |  | Operational metadata | `standard`, `share`, `paranoid` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `score`, `total_users`, `stats`: ZDX score and user/device counts by experience (poor, okay, good) over the window.
- `name`: Application name.
- `most_impacted_region`: `region`, `country` and `geo_type` render in share; `id` and `city` are standard only.
- `score_trend`, `metrics`: One app only (`report <id>`): score and metric series as `{metric, unit, datapoints: [{timestamp, value}]}`.

Report resources read a time window (`--since`, default `2h`, or `--from`/`--to`), which the output echoes; they are not part of `dump` or `diff`. `--department` and `--location` narrow the report. On the validation tenant the apps and departments routes rejected windows longer than 2 hours (alerts accepted 7 days); a score of `-1` in a series means no data for that interval.

## ZDX Alerts

Commands:

```sh
zscalerctl zdx alerts report --since 2h
```

Fields:

<!-- BEGIN GENERATED FIELDS: zdx/alerts (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `rule_name` |  | Tenant configuration | `standard`, `share` |
| `severity` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `alert_type` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `alert_status` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `num_geolocations` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `num_devices` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `started_on` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `ended_on` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `application` |  | Tenant configuration | `standard`, `share` |
| `application.id` |  | Operational metadata | `standard`, `share` |
| `application.name` |  | Tenant configuration | `standard`, `share` |
| `departments` |  | Tenant configuration | `standard`, `share` |
| `departments.id` |  | Operational metadata | `standard`, `share` |
| `departments.name` |  | Tenant configuration | `standard`, `share` |
| `departments.num_devices` |  | Operational metadata | `standard`, `share` |
| `locations` |  | Tenant configuration | `standard`, `share` |
| `locations.id` |  | Operational metadata | `standard`, `share` |
| `locations.name` |  | Tenant configuration | `standard`, `share` |
| `locations.num_devices` |  | Operational metadata | `standard`, `share` |
| `locations.groups` |  | Tenant configuration | `standard`, `share` |
| `locations.groups.id` |  | Operational metadata | `standard`, `share` |
| `locations.groups.name` |  | Tenant configuration | `standard`, `share` |
| `geolocations` |  | Tenant configuration | `standard`, `share` |
| `geolocations.id` |  | Sensitive identifier | `standard` |
| `geolocations.name` |  | Sensitive identifier | `standard` |
| `geolocations.num_devices` |  | Operational metadata | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `severity`, `alert_type`, `alert_status`, `num_geolocations`, `num_devices`, `started_on`, `ended_on`: Alert state, counts and times (Unix seconds; 0 means unset).
- `rule_name`: The alert rule's name.
- `application`, `departments`, `locations`: ZDX application, department and location (with location groups) the alert concerns, with device counts.
- `geolocations`: Device counts per geographic area; area ids and names are standard only.

Ongoing alerts. Report resources read a time window (`--since`, default `2h`, or `--from`/`--to`), which the output echoes; they are not part of `dump` or `diff`. Affected users and devices are not read.

## ZDX Alert History

Commands:

```sh
zscalerctl zdx alert-history report --since 7d
```

Fields:

<!-- BEGIN GENERATED FIELDS: zdx/alert-history (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `rule_name` |  | Tenant configuration | `standard`, `share` |
| `severity` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `alert_type` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `alert_status` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `num_geolocations` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `num_devices` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `started_on` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `ended_on` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `application` |  | Tenant configuration | `standard`, `share` |
| `application.id` |  | Operational metadata | `standard`, `share` |
| `application.name` |  | Tenant configuration | `standard`, `share` |
| `departments` |  | Tenant configuration | `standard`, `share` |
| `departments.id` |  | Operational metadata | `standard`, `share` |
| `departments.name` |  | Tenant configuration | `standard`, `share` |
| `departments.num_devices` |  | Operational metadata | `standard`, `share` |
| `locations` |  | Tenant configuration | `standard`, `share` |
| `locations.id` |  | Operational metadata | `standard`, `share` |
| `locations.name` |  | Tenant configuration | `standard`, `share` |
| `locations.num_devices` |  | Operational metadata | `standard`, `share` |
| `locations.groups` |  | Tenant configuration | `standard`, `share` |
| `locations.groups.id` |  | Operational metadata | `standard`, `share` |
| `locations.groups.name` |  | Tenant configuration | `standard`, `share` |
| `geolocations` |  | Tenant configuration | `standard`, `share` |
| `geolocations.id` |  | Sensitive identifier | `standard` |
| `geolocations.name` |  | Sensitive identifier | `standard` |
| `geolocations.num_devices` |  | Operational metadata | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`, `severity`, `alert_type`, `alert_status`, `num_geolocations`, `num_devices`, `started_on`, `ended_on`: Alert state, counts and times (Unix seconds; 0 means unset).
- `rule_name`: The alert rule's name.
- `application`, `departments`, `locations`: ZDX application, department and location (with location groups) the alert concerns, with device counts.
- `geolocations`: Device counts per geographic area; area ids and names are standard only.

Alerts that ended in the window. Report resources read a time window (`--since`, default `2h`, or `--from`/`--to`), which the output echoes; they are not part of `dump` or `diff`.

## ZDX Departments

Commands:

```sh
zscalerctl zdx departments report
```

Fields:

<!-- BEGIN GENERATED FIELDS: zdx/departments (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `name` |  | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Department id for `--department`.
- `name`: Department name.

ZDX's report-filter dimension. Report resources read a time window (`--since`, default `2h`, or `--from`/`--to`), which the output echoes; they are not part of `dump` or `diff`.

## ZDX Locations

Commands:

```sh
zscalerctl zdx locations report
```

Fields:

<!-- BEGIN GENERATED FIELDS: zdx/locations (make resources-doc) -->
| Field | Type | Classification | Modes |
| --- | --- | --- | --- |
| `id` |  | Operational metadata | `standard`, `share`, `paranoid` |
| `name` |  | Tenant configuration | `standard`, `share` |
<!-- END GENERATED FIELDS -->

Field notes:

- `id`: Location id for `--location`.
- `name`: Location name.

ZDX's report-filter dimension. Report resources read a time window (`--since`, default `2h`, or `--from`/`--to`), which the output echoes; they are not part of `dump` or `diff`.
