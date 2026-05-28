# Resource Reference

This document describes the currently enabled resource catalog. The catalog is
the output allow-list: reader adapters map SDK response shapes into source
records, and projection decides which fields can render for each redaction mode.

## Redaction Modes

- `standard`: local operational use. Allows explicitly reviewed tenant
  configuration and free-text fields, with secret scanning and free-text
  high-entropy token scanning still applied.
- `share`: lower-detail output for tickets, reviews, and chat. Drops free text
  and sensitive identifiers.
- `paranoid`: minimal identifiers and counts only.

All fields, including allowed strings, pass through the final redaction backstop
before stdout or dump files. Free-text fields also receive a conservative
high-entropy token scan for bare unlabeled secret material. Canonical UUIDs and
contextual git commit SHAs are preserved; other long hashes or thumbprints may
be redacted in free text.

## ZIA Locations

Commands:

```sh
zscalerctl zia locations list
zscalerctl zia locations get <id>
```

Fields:

| Field | Classification | Modes | Notes |
| --- | --- | --- | --- |
| `id` | Operational metadata | `standard`, `share`, `paranoid` | ZIA location identifier. |
| `name` | Tenant configuration | `standard`, `share` | Scanned for pasted secret-shaped values. |
| `ipAddresses` | Sensitive identifier | `standard` | Dropped from `share` and `paranoid`. |
| `description` | Free text | `standard` | High-risk admin-controlled text; scanned before output, including bare high-entropy tokens. |
| `preSharedKey` | Secret | never | Explicitly modeled so it cannot render. |
| `vpnCredentials` | Secret | never | SDK nested credentials are mapped into source records and dropped by projection. |

## ZIA Rule Labels

Commands:

```sh
zscalerctl zia rule-labels list
zscalerctl zia rule-labels get <id>
```

Fields:

| Field | Classification | Modes | Notes |
| --- | --- | --- | --- |
| `id` | Operational metadata | `standard`, `share`, `paranoid` | ZIA rule-label identifier. |
| `name` | Tenant configuration | `standard`, `share` | Scanned for pasted secret-shaped values. |
| `description` | Free text | `standard` | High-risk admin-controlled text; scanned before output, including bare high-entropy tokens. |
| `lastModifiedTime` | Operational metadata | `standard`, `share` | SDK timestamp value. |
| `referencedRuleCount` | Operational metadata | `standard`, `share`, `paranoid` | Number of referencing rules. |

The SDK also returns admin references such as `createdBy` and `lastModifiedBy`.
The reader maps those nested objects into source records, but the catalog does
not allow them to render, so projection drops them.

## Adding A Resource

Before enabling another resource:

- Map the SDK response shape into source records without using the reader as a
  second safety allow-list.
- Classify every candidate output field in the catalog.
- Mark known secret-bearing fields as `secret`, even when they are expected to
  be dropped.
- Add canary tests proving secret-looking names or descriptions are redacted,
  including bare high-entropy tokens for any emitted free-text field.
- Add nested drop tests for any SDK object that contains user, admin, key,
  token, credential, or free-text data.
- Confirm `AssertRenderedSubset` runs before rendering and dump writing.
- Update this reference and the shell completion tests.
