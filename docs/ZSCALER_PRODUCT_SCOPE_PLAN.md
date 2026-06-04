# Zscaler Product Scope Plan

This is a scout plan for SDK-exposed products outside the currently active ZIA
and ZPA resource tracks. It is not an enabled catalog, entitlement check, safety
proof, or live response-shape validation.

The inventory was generated from the full `github.com/zscaler/zscaler-sdk-go/v3`
module cache for SDK `v3.8.37`, not only from the committed vendor tree:

```sh
SDK_DIR="$(go list -m -f '{{.Dir}}' -mod=mod github.com/zscaler/zscaler-sdk-go/v3)"
go run ./scripts/sdk-surface-inventory.go --sdk-dir "$SDK_DIR" --format json
```

Official docs checked during this scout:

- ZDX API docs describe application, device, user, alert, and troubleshooting
  endpoints, with report APIs requiring explicit time ranges:
  <https://help.zscaler.com/zdx/understanding-zdx-api>
- ZIdentity API docs describe users, groups, resource servers, and entitlements
  as identity/admin API resources:
  <https://help.zscaler.com/legacy-apis/understanding-zidentity-apis>
- Workload Group docs describe workload groups as cloud workload policy
  references:
  <https://help.zscaler.com/zia/about-workload-groups>
- Cloud Connector docs describe Cloud Connectors as traffic-forwarding
  infrastructure for ZIA/ZPA and cloud-native workloads:
  <https://help.zscaler.com/zpa/about-cloud-connectors>

The docs support the same split as the SDK: ZTW is closest to config/workload
inventory, ZDX is report/telemetry, ZIdentity is identity-plane data, and ZWA is
audit/incident data.

Preproduction OneAPI access can prove that an endpoint is reachable for that
deployment. It cannot prove production entitlement, production response shape,
or field-level safety for a financial-production tenant. Any product family that
is used only in production remains `scaffolded` or `gates-passed` until a
read-only production OneAPI smoke can run.

## Product Auth Posture

The current OneAPI configuration already wires direct, SDK-cache-free,
SDK-logger-free HTTP clients for ZIA, ZPA, ZTW, ZCC, and ZDX through
`NewOneAPIClient`; Zidentity uses admin/common routing through the same service
boundary. First implementations for new products should use SDK service
functions that accept `*zscaler.Service` and must not instantiate product-local
legacy clients directly.

The full SDK also exposes product-local client/config packages for ZCC, ZDX,
ZTW, and ZWA. Those clients have separate credential, logging, proxy, and cache
behavior. If a future resource requires one of those clients instead of the
OneAPI service, it needs a boundary review before implementation. ZWA especially
needs auth-path verification because the committed OneAPI client does not expose
a dedicated `ZWAHTTPClient` slot.

The CI no-live-credentials gate already blocks ZCC, ZDX, ZTW/ZTC, ZPA, ZIA, and
OneAPI credential-shaped environment names in `.github/**` YAML. That protects
the repository from accidental live-credential workflow additions, but it does
not validate product entitlement.

## Inventory Summary

| Product | SDK packages | Ordinary list/get | List/get with mutating neighbors | Read-only nonstandard | Mixed read/write | Notes |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| ZCC | 25 | 0 | 4 | 8 | 10 | Client Connector admin/config plus device and secret-adjacent surfaces. |
| ZDX | 11 | 3 | 0 | 3 | 2 | Monitoring/reporting data, not ordinary configuration inventory. |
| ZTW | 30 | 2 | 20 | 3 | 2 | Workload/Cloud Connector-style configuration and policy surfaces. |
| Zidentity | 6 | 1 | 2 | 2 | 1 | Identity-plane resources; treat as privacy and authorization sensitive. |
| ZWA | 5 | 0 | 0 | 2 | 1 | Audit and DLP incident/evidence surfaces; not ordinary config inventory. |

Counts are scout signals only. `list-get-with-mutating-neighbors` can still be
safe when zscalerctl wires only read functions, but it requires explicit review.

## Recommended Sequence

1. Keep ZPA resource work separate from this plan until the parked ZPA stack has
   one focused production OneAPI smoke pass.
2. Start ZTW as the first separate non-ZIA/ZPA product family. It has the
   strongest config-like SDK surface and maps most directly to Cloud Connector /
   workload inventory.
3. Add one low-risk ZTW reference resource first, preferably workload groups,
   prove product auth and live smoke, then batch remaining ZTW references.
4. Scout ZCC next for Client Connector configuration references after ZTW's
   product path is understood.
5. Treat ZDX as a separate report/telemetry model rather than forcing it into the
   current config-dump semantics.
6. Leave ZWA as inventory-only unless product ownership and entitlement are
   confirmed. It is audit/incident data, not config inventory.
7. Treat Zidentity last unless there is a specific operator need; users, groups,
   and entitlement reads need a stricter privacy posture than config references.

## ZCC

ZCC is a good separate product track after ZTW. It has configuration-shaped
resources, but several SDK packages sit next to mutating helpers, device data,
or secret-like read functions.

| Candidate | SDK package | Scout category | Queue posture |
| --- | --- | --- | --- |
| `zcc/trusted-networks` | `zscaler/zcc/services/trusted_network_v2` | `list-get-with-mutating-neighbors` | Good first resource if OneAPI service calls work. Network identifiers likely standard-only where rendered. |
| `zcc/notification-templates` | `zscaler/zcc/services/notification_template` | `list-get-with-mutating-neighbors` | Viable early resource. Template body/free-text needs standard-only or share-gated review. |
| `zcc/zia-postures` | `zscaler/zcc/services/zia_posture` | `list-get-with-mutating-neighbors` | Viable early metadata resource. Expect posture/security-condition fields. |
| `zcc/custom-ip-apps` | `zscaler/zcc/services/custom_ip_apps` | `read-only-nonstandard` | Useful, but list semantics are package-specific (`GetCustomIPApps`, `GetByAppID`, `GetByName`). Design as list-only/name-get if needed. |
| `zcc/predefined-ip-apps` | `zscaler/zcc/services/predefined_ip_apps` | `read-only-nonstandard` | Similar to custom IP apps; likely lower sensitivity because predefined. |
| `zcc/process-based-apps` | `zscaler/zcc/services/process_based_apps` | `read-only-nonstandard` | Useful but process/app identifiers may be endpoint-sensitive. |
| `zcc/devices` | `zscaler/zcc/services/devices` | `list-get-with-mutating-neighbors` | High value, but device/user/privacy heavy. Defer until base ZCC auth and smoke are proven. |

Do not queue as ordinary inventory:

| SDK package | Reason |
| --- | --- |
| `zscaler/zcc/services/secrets/getotp` | Secret/OTP surface by name and purpose. |
| `zscaler/zcc/services/secrets/getpasswords` | Password retrieval surface by name and purpose. |
| `zscaler/zcc/services/download_devices` | Export/download semantics, not ordinary JSON inventory. |
| `zscaler/zcc/services/manage_pass` and device-removal packages | Mutating or credential-adjacent admin operations. |

## ZDX

ZDX is primarily monitoring, reporting, user, device, and application experience
data. It should not be treated as plain configuration inventory without an
explicit report model.

| Candidate | SDK package | Scout category | Queue posture |
| --- | --- | --- | --- |
| `zdx/applications` | `zscaler/zdx/services/reports/applications` | `ordinary-list-get` | Best first ZDX candidate. Application-level reporting is less privacy-heavy than users/devices. |
| `zdx/users` | `zscaler/zdx/services/reports/users` | `ordinary-list-get` | Defer. User telemetry is privacy-sensitive and likely needs restricted modes. |
| `zdx/devices` | `zscaler/zdx/services/reports/devices` | `ordinary-list-get` | Defer. Device telemetry and metric fields need separate posture. |
| `zdx/alerts` | `zscaler/zdx/services/alerts` | `read-only-nonstandard` | Defer until alert time-window and affected-device semantics are clear. |
| `zdx/software-inventory` | `zscaler/zdx/services/inventory` | `read-only-nonstandard` | Defer. Software inventory is endpoint-sensitive. |
| `zdx/administration` | `zscaler/zdx/services/administration` | `read-only-nonstandard` | Department/location helpers may be safe, but shape is not ordinary list/get. |

Before enabling ZDX, decide whether report resources belong in `dump` at all or
need a separate time-windowed report command. Defaulting dynamic telemetry into
configuration dumps would weaken the deterministic-config story.

## ZTW

ZTW has workload, cloud account, gateway, DNS, EC group, and policy resources.
It is likely closer to Cloud Connector/Workload than to ZIA inventory.

| Candidate | SDK package | Scout category | Queue posture |
| --- | --- | --- | --- |
| `ztw/workload-groups` | `zscaler/ztw/services/workload_groups` | `ordinary-list-get` | Best first ZTW candidate. Reference-style metadata, likely manageable with existing projection rules. |
| `ztw/public-cloud-accounts` | `zscaler/ztw/services/provisioning/public_cloud_account` | `ordinary-list-get` | Useful but account identifiers require conservative classification. |
| `ztw/forwarding-gateways` | `zscaler/ztw/services/forwarding_gateways` | `list-get-with-mutating-neighbors` | Viable after base ZTW auth is proven; network identifiers standard-only. |
| `ztw/dns-gateways` | `zscaler/ztw/services/dns_gateway` | `list-get-with-mutating-neighbors` | Viable after base ZTW auth is proven; network identifiers standard-only. |
| `ztw/ec-groups` | `zscaler/ztw/services/ecgroup` | `list-get-with-mutating-neighbors` | Viable reference surface after auth proof. |
| `ztw/locations` | `zscaler/ztw/services/location` | `list-get-with-mutating-neighbors` | Useful but may overlap with ZIA location semantics; review separately. |

Do not queue as ordinary inventory:

| SDK package | Reason |
| --- | --- |
| `zscaler/ztw/services/provisioning/api_keys` | API key surface by name. |
| `zscaler/ztw/services/provisioning/provisioning_url` | Provisioning URL surface by name. |
| `zscaler/ztw/services/administration/admin_users` | Admin identity surface. |
| `zscaler/ztw/services/administration/admin_roles` | Admin authorization surface. |
| `zscaler/ztw/services/policy/...` | Policy/control surfaces; valuable, but should follow the simpler ZTW references. |

## ZWA

ZWA appears in the full SDK module cache as audit and DLP incident/evidence
surfaces. It is not present as a comparable high-level product client in the
committed vendor tree, and the committed OneAPI client does not expose a
dedicated `ZWAHTTPClient` field. Treat this as a separate auth-path and data
semantics track only if the product is confirmed in scope.

| Candidate | SDK package | Scout category | Queue posture |
| --- | --- | --- | --- |
| `zwa/customer-audit` | `zscaler/zwa/services/customeraudit` | `read-only-nonstandard` | Defer until auth routing and audit-log retention/output semantics are designed. |
| `zwa/dlp-incidents` | `zscaler/zwa/services/dlp_incidents` | `mixed-read-write-sdk-package` | Defer. Incident, evidence, ticket, and history data are sensitive and sit next to mutating helpers. |
| `zwa/common` | `zscaler/zwa/services/common` | `read-only-nonstandard` | Helper/pagination/types package, not a catalog resource. |

Do not add ZWA to ordinary config dumps without deciding whether audit and DLP
incident records belong in the same output model as static configuration. If it
turns out to be needed, scaffold it only as a draft investigation PR until
product ownership, entitlement, auth routing, and retention expectations are
clear.

## Zidentity

Zidentity is an identity plane. The SDK exposes useful read surfaces, but the
data is privacy and authorization sensitive.

| Candidate | SDK package | Scout category | Queue posture |
| --- | --- | --- | --- |
| `zidentity/resource-servers` | `zscaler/zid/services/resource_servers` | `ordinary-list-get` | Safest first Zidentity resource if this product track is needed. Still review identifiers and OAuth-style fields conservatively. |
| `zidentity/groups` | `zscaler/zid/services/groups` | `list-get-with-mutating-neighbors` | Defer. Group membership and nested user links need privacy posture. |
| `zidentity/users` | `zscaler/zid/services/users` | `list-get-with-mutating-neighbors` | Defer. User identity surface; likely no share mode by default. |
| `zidentity/user-entitlements` | `zscaler/zid/services/user_entitlement` | `read-only-nonstandard` | Defer. Authorization/entitlement surface. |

## Cross-Product Implementation Rules

Apply these before any resource PR from this plan:

1. Use the OneAPI `*zscaler.Service` path when the SDK package supports it.
2. Do not add product-local legacy clients without a boundary review for
   credential discovery, logging, proxy use, cache behavior, and stdout/stderr
   writes.
3. Add product-specific live-smoke manifest support before the first resource in
   a new product family is promoted.
4. Keep resource PRs small: one product family and one coherent API section.
5. Keep reference-only expansion: when a nested object has its own resource or
   could reasonably have one, render only id/name-style references in the parent.
6. For production-only entitlements, accept `gates-passed` without promotion and
   record the missing smoke explicitly.
7. Do not use dev/preprod 404s as proof that a resource is invalid in
   production; record them as entitlement or deployment-shape unknowns.

## First Branch Recommendations

Suggested independent branches, in order:

| Branch | Scope | Expected outcome |
| --- | --- | --- |
| `feature/ztw-scope-plan` | Verify OneAPI SDK call path for ZTW and scaffold `workload_groups`. | Establish Cloud/Workload product semantics without touching provisioning credentials. |
| `feature/zcc-scope-plan` | Verify OneAPI SDK call path for ZCC and scaffold `trusted_network_v2` or `notification_template`. | Establish whether ZCC can use the current service boundary cleanly. |
| `feature/zdx-report-scope-plan` | Decide report command/dump semantics before scaffolding `reports/applications`. | Prevent telemetry from being accidentally treated as deterministic config inventory. |
| `feature/zidentity-scope-plan` | Scope `resource_servers` only. | Keep identity work narrow until privacy posture is explicit. |

ZWA is deliberately not in the first-branch queue. Open a draft
`feature/zwa-scope-plan` only if the product is confirmed in scope.
