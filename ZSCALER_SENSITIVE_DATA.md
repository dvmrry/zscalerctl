# Zscaler Sensitive Data Inventory

This inventory records Zscaler-specific secret, credential, and sensitive data
families that `zscalerctl` must not expose. It is based on an official Zscaler
documentation pass performed on 2026-05-27.

The inventory is intentionally conservative. When a field is ambiguous, resource
catalog entries should classify it as secret until a source-backed exception is
documented.

## Source Map

- [About App Connector Provisioning Keys](https://help.zscaler.com/zpa/about-connector-provisioning-keys)
- [Configuring Provisioning Keys Using API](https://help.zscaler.com/legacy-apis/configuring-provisioning-keys-using-api)
- [About API Key Management](https://help.zscaler.com/zpa/about-api-key-management)
- [Adding API Keys](https://help.zscaler.com/zpa/adding-api-keys)
- [About Cloud Service API Key](https://help.zscaler.com/zia/about-cloud-service-api-key)
- [About Sandbox API Token](https://help.zscaler.com/legacy-zia/about-sandbox-api-token)
- [API Clients](https://help.zscaler.com/legacy-apis/api-clients)
- [About Access Tokens](https://help.zscaler.com/zidentity/about-access-tokens)
- [SCIM API Examples](https://help.zscaler.com/zidentity/scim-api-examples)
- [Getting Started with the ZDX API](https://help.zscaler.com/legacy-apis/getting-started-zdx-api)
- [Getting Started with the Client Connector API](https://help.zscaler.com/legacy-apis/getting-started-client-connector-api)
- [Configuring Privileged Credentials](https://help.zscaler.com/zpa/configuring-privileged-credentials)
- [Configuring Certificates Using API](https://help.zscaler.com/legacy-apis/configuring-certificates-using-api)
- [Enrollment Certificates](https://help.zscaler.com/legacy-apis/enrollment-certificates)
- [Location Management](https://help.zscaler.com/legacy-apis/location-management)
- [Splunk Webhook Configuration Guide](https://help.zscaler.com/zdx/splunk-webhook-configuration-guide)

## Secret Families

### API Authentication Material

Classify as secret:

- Client secrets and secret keys for ZPA, ZIdentity, and Client Connector API
  clients.
- ZDX `key_secret` values.
- Legacy ZIA Cloud Service API keys.
- Sandbox API tokens.
- 3rd-Party App Governance API keys.
- Bearer tokens, OAuth access tokens, refresh tokens, JWTs, and SCIM bearer
  tokens.
- Authorization headers, cookies, session IDs, and credential-bearing URLs.
- Admin passwords used with legacy API-key-plus-admin-credential flows.

Classify as sensitive identifiers:

- API client IDs, ZDX key IDs, API key IDs, tenant IDs, customer IDs, and
  microtenant IDs. These values may be needed for operations, but should be
  minimized in shared output modes.
- Public API-client authentication material such as `keyValue`, `certContent`,
  public certificates, and JWKS URLs. These are not private secrets by
  themselves, but they can identify tenant integrations and domains.

### Provisioning And Enrollment Material

Classify as secret:

- App Connector provisioning keys.
- ZPA Private Service Edge provisioning keys.
- Network Connector provisioning keys.
- Provisioning or enrollment tokens used to enroll infrastructure components.

Known shape:

```text
number|host.name|long-key-material
```

The host component depends on the tenant cloud. The scanner therefore treats any
long pipe-delimited provisioning-key-shaped value as secret, not only values
with `zscaler` in the hostname.

### Privileged Remote Access Credentials

Classify as secret:

- PRA username/password credential passwords.
- VNC passwords.
- SSH private keys.
- SSH private key passphrases.

Usernames and domains associated with privileged credentials are sensitive
identifiers and should be minimized in `share` and `paranoid` modes.

### Certificates And Private Key Material

Classify as secret:

- `certBlob` values when they include uploaded certificate/private-key material.
- Enrollment certificate `privateKey` values.
- `zrsaencryptedprivatekey`.
- `zrsaencryptedsessionkey`.
- Any PEM private key block.

Classify as sensitive identifiers or tenant configuration:

- Public certificates, certificate chains, public keys, serial numbers,
  subject names, SANs, issuers, and CNs. They are not private keys, but they can
  reveal tenant names, internal domains, and application topology.

### Network And Location Secrets

Classify as secret:

- VPN/XAUTH pre-shared keys such as `preSharedKey`.
- Any field named `psk`, `sharedSecret`, or equivalent.

Classify as tenant configuration or sensitive identifiers:

- Gateway addresses, VPN locations, internal domains, source IPs, and comments.

### Webhook And Integration Tokens

Classify as secret:

- Webhook bearer tokens.
- Webhook token-auth values.
- Basic-auth passwords used for webhook delivery.
- Splunk HEC token values or similar integration tokens.

Classify as tenant configuration:

- Webhook URLs and alert names. URLs may contain internal hostnames or embedded
  credentials, so they must still pass through the scanner.

### User Authentication Secrets

Classify as secret:

- Temporary passwords.
- One-time passwords.
- One-time tokens.
- Device tokens when exposed by an API response or configuration export.

Classify usernames, emails, groups, departments, and user IDs as sensitive
identifiers.

## Field Name Denylist

Resource specs must classify these names, and close variants, as `secret` unless
there is a documented source-backed exception:

- `accessToken`
- `apiKey`
- `apiToken`
- `authorization`
- `bearerToken`
- `certBlob`
- `clientSecret`
- `cookie`
- `hecToken`
- `jwt`
- `jwtToken`
- `keySecret`
- `otp`
- `passphrase`
- `password`
- `preSharedKey`
- `privateKey`
- `provisioningKey`
- `refreshToken`
- `sandboxApiToken`
- `secret`
- `secretKey`
- `sessionId`
- `sharedSecret`
- `token`
- `zrsaencryptedprivatekey`
- `zrsaencryptedsessionkey`

The generic field names `value`, `keyValue`, and `certContent` are
context-sensitive. `value` appears as the secret value field in API client
secret responses, but it is also used for SCIM IDs and email addresses. API
client schemas use `keyValue` and `certContent` for public-key and certificate
authentication material. These names are too generic to redact globally without
making legitimate outputs unusable. Resource specs must classify them in
context, and must classify `value` as secret for secret-specific DTOs,
especially API client secret endpoints.

## Scanner Requirements

The redaction scanner must catch:

- Labeled assignments in env, YAML-ish, and JSON forms.
- Authorization headers.
- Credential-bearing URLs.
- JWT-shaped values.
- PEM private key blocks.
- Pipe-delimited Zscaler provisioning-key-shaped values.
- Secret phrases pasted into free-text fields, such as names, descriptions,
  labels, comments, and notes.

The scanner must preserve JSON syntax when redacting JSON string values. A
redaction backstop that leaks less but corrupts machine output will make the CLI
hard to use safely in pipelines.

## Resource Catalog Requirements

The scanner is defense-in-depth. Resource catalog entries are still responsible
for primary safety:

- Unknown fields are dropped by default.
- Secret fields are never renderable.
- Context-sensitive fields such as `value` must be classified per resource.
- Allowed string fields are scanned before rendering.
- Free-text fields require explicit review before they can appear in `standard`
  mode.
