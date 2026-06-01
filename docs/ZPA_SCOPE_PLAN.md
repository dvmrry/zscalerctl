# ZPA Scope Plan

This document is a scout plan, not an enabled resource catalog. It uses the
pinned Go SDK source to decide which ZPA surfaces are plausible future
read-only resources, which need design, and which should stay out of ordinary
inventory work.

The ZIA release path still depends on catalog entries, SDK shape review,
projection canaries, and live smoke. ZPA must use the same gates before any
resource is promoted.

## How To Regenerate

The default SDK inventory scans the committed vendor tree. For ZPA scoping,
scan the full pinned SDK module cache because the current vendor tree may not
contain unimported ZPA service packages:

```sh
SDK_DIR="$(go list -m -f '{{.Dir}}' -mod=mod github.com/zscaler/zscaler-sdk-go/v3)"
make sdk-surface-inventory SDK_DIR="$SDK_DIR" PRODUCT=zpa
make sdk-surface-inventory SDK_DIR="$SDK_DIR" PRODUCT=zpa FORMAT=json
```

The inventory is scout evidence only. It does not prove entitlement, tenant
availability, pagination behavior, response shape, or safe field
classification.

## Current Full-SDK Signal

The pinned SDK version (`github.com/zscaler/zscaler-sdk-go/v3` v3.8.37) exposes
a broad `zscaler/zpa/services/...` tree in the full module cache.

| Category | Count | Meaning |
| --- | ---: | --- |
| `ordinary-list-get` | 15 | Read-like list/get surface without mutating-looking exported helpers. |
| `list-get-with-mutating-neighbors` | 47 | Read-like surface exists, but the package also exposes mutating helpers. Wire only explicit read functions. |
| `read-only-nonstandard` | 8 | Read functions exist, but not the ordinary list/get shape. Needs custom semantics. |
| `mixed-read-write-sdk-package` | 7 | Read and mutating helpers are entangled enough to require design before queueing. |
| `mutating-only` | 3 | Not a read-only inventory candidate. |
| `product-client-config` | 1 | Product client/config package, not a resource. |

## First-Pass Candidate Tiers

### Tier 1: Low-Drama Inventory References

Start here once a separate ZPA smoke-lab branch exists and the smoke harness is
product-aware. These are useful inventory/reference surfaces and mostly avoid
secret-bearing or identity-administration semantics.

| Candidate | SDK package | Category | Why first |
| --- | --- | --- | --- |
| `zpa/server-groups` | `zscaler/zpa/services/servergroup` | `list-get-with-mutating-neighbors` | Core ZPA inventory primitive; proves nested app/server/connector references. |
| `zpa/segment-groups` | `zscaler/zpa/services/segmentgroup` | `list-get-with-mutating-neighbors` | Core application grouping surface; useful without exposing application segment rules first. |
| `zpa/app-connector-groups` | `zscaler/zpa/services/appconnectorgroup` | `list-get-with-mutating-neighbors` | Connector placement metadata; exercises connector-group references. |
| `zpa/app-servers` | `zscaler/zpa/services/appservercontroller` | `list-get-with-mutating-neighbors` | Server inventory surface; likely high value for replacing ad hoc scripts. |
| `zpa/machine-groups` | `zscaler/zpa/services/machinegroup` | `ordinary-list-get` | Low-risk reference surface with normal list/get shape. |
| `zpa/trusted-networks` | `zscaler/zpa/services/trustednetwork` | `ordinary-list-get` | Readable network-reference surface; expect careful identifier treatment. |

### Tier 2: Useful After Tier 1 Shape Is Proven

These look viable, but they carry either richer nested data, cloud connector
semantics, browser-isolation semantics, or posture/IdP metadata that should be
reviewed after the first ZPA smoke pass proves the product path.

| Candidate | SDK package | Category | Reason to wait |
| --- | --- | --- | --- |
| `zpa/application-segments` | `zscaler/zpa/services/applicationsegment` | `list-get-with-mutating-neighbors` | High-value but large nested policy surface. |
| `zpa/browser-access-apps` | `zscaler/zpa/services/applicationsegmentbrowseraccess` | `list-get-with-mutating-neighbors` | Application segment variant with browser-access fields. |
| `zpa/inspection-apps` | `zscaler/zpa/services/applicationsegmentinspection` | `list-get-with-mutating-neighbors` | Inspection rule/app metadata; richer nested controls. |
| `zpa/pra-apps` | `zscaler/zpa/services/applicationsegmentpra` | `list-get-with-mutating-neighbors` | PRA adjacency; keep credentials/portals separate. |
| `zpa/service-edge-groups` | `zscaler/zpa/services/serviceedgegroup` | `list-get-with-mutating-neighbors` | Useful edge inventory, but wait until connector/server refs are proven. |
| `zpa/service-edges` | `zscaler/zpa/services/serviceedgecontroller` | `list-get-with-mutating-neighbors` | Controller-like inventory with update/delete neighbors. |
| `zpa/cloud-connectors` | `zscaler/zpa/services/cloud_connector` | `ordinary-list-get` | Cloud connector product semantics need explicit naming and smoke coverage. |
| `zpa/cloud-connector-groups` | `zscaler/zpa/services/cloud_connector_group` | `ordinary-list-get` | Companion to cloud connectors; likely batch with cloud connectors. |
| `zpa/posture-profiles` | `zscaler/zpa/services/postureprofile` | `ordinary-list-get` | Device posture metadata; likely safe, but privacy semantics need review. |
| `zpa/idps` | `zscaler/zpa/services/idpcontroller` | `ordinary-list-get` | Authentication metadata; useful but closer to identity-plane policy. |

### Tier 3: Design Before Cataloging

These are not "just add a resource" surfaces. They need output semantics,
field-policy choices, or command-shape decisions before scaffolding.

| Surface | SDK package | Why design first |
| --- | --- | --- |
| Policy sets | `policysetcontroller`, `policysetcontrollerv2` | Large policy-rule surface with ordering, conditions, credentials-adjacent structs, and rule-type get semantics. |
| Inspection controls/profiles | `inspectioncontrol/...` | Custom/predefined/profile split and association methods need resource boundaries. |
| LSS configs | `lssconfigcontroller` | Logging receiver/config surface with format/status helpers and sensitive routing data. |
| Browser isolation profiles | `cloudbrowserisolation/...` | Multiple subpackages; certificate/profile/security-control split needs a separate plan. |
| Tags | `tag_controller/...` | Namespaces, keys, values, and status updates should be modeled as a coherent tag system. |
| Client settings/types/platforms | `client_settings`, `clienttypes`, `platforms` | Settings/list-only surfaces need singleton/list-only semantics per endpoint. |
| Extranet and location summaries | `extranet_resource`, `location_controller` | Read-only summary helpers rather than ordinary config objects. |

### Hold Or Exclude From Ordinary Inventory

These are likely important later, but they are not safe breadth candidates
without a sharper policy.

| Surface | SDK package | Hold reason |
| --- | --- | --- |
| API keys | `api_keys` | Secret/key-management plane. |
| Provisioning keys | `provisioningkey` | Credential-bearing by design. |
| PRA credentials and pools | `privilegedremoteaccess/pracredential`, `privilegedremoteaccess/pracredentialpool` | Credential material surface. |
| Certificates and enrollment certs | `bacertificate`, `enrollmentcert`, `cloudbrowserisolation/cbicertificatecontroller` | Certificate/CSR/material boundary. |
| Admins, roles, SCIM, SAML attributes | `administrator_controller`, `role_controller`, `scim_api`, `scimgroup`, `scimattributeheader`, `samlattribute` | Identity/admin/privacy plane. |
| Emergency access | `emergencyaccess` | Break-glass access semantics; not ordinary inventory. |
| Application segment move/share | `applicationsegment_move`, `applicationsegment_share` | Mutating-only. |
| OAuth user verification | `oauth2_user` | Mutating/verification workflow, not inventory. |

## Proposed ZPA Smoke-Lab Order

1. Create `feature/smoke-lab-zpa-surface` from `main`.
2. Make live smoke accept a ZPA manifest and report product-specific resource
   sets without requiring ZIA legacy credentials.
3. Add Tier 1 as a small batch, or split into two batches if the first ZPA path
   needs debugging:
   - `zpa/server-groups`
   - `zpa/segment-groups`
   - `zpa/app-connector-groups`
   - `zpa/app-servers`
   - `zpa/machine-groups`
   - `zpa/trusted-networks`
4. Keep the PR draft until a controlled OneAPI smoke run confirms endpoint
   availability and real response shape.
5. Promote only the resources that pass focused smoke. Park 404/403/shape
   failures in this plan instead of forcing the batch through.

## Open Questions

- Which production OneAPI client profile is authorized for ZPA read-only smoke?
- Should ZPA `get` support name selectors earlier than ZIA because many SDK
  packages expose `GetByName`?
- Do application segment variants share enough shape to use helper mappers, or
  should each variant stay explicit?
- Should policy sets wait until ZIA policy rule output settles further?
