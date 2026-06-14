# Realism Deltas — what the agentic-coverage fixtures deliberately do NOT model

The agentic-coverage eval runs against synthetic, value-free **fixtures**, never a
live Zscaler tenant. That is what makes it deterministic, CI-able, and safe (no
secrets, no tenant data, ever). It also **bounds the claim**: the floor number is a
*self-describability* measurement — "the weakest agent that can drive the tool from
its own surface" — **not a live-tenant guarantee**. This register lists the gaps
between the fixture surface and a real tenant so the floor is never over-read.

Keep this file honest: when a fixture slice starts modeling one of these, move it
out of the register and into the battery.

| Not modeled | What the fixtures do instead | Why it bounds the claim |
|-------------|------------------------------|-------------------------|
| **Pagination at scale** | Small in-memory record sets returned whole. | A real tenant paginates; an agent that stops at page 1 would pass here but undercount live. Until a multi-page fixture exists, count/`set` answers prove parsing, not pagination-handling. |
| **Rate limiting / backoff** | Every call returns immediately. | Real APIs return 429/5xx and require backoff (an `exit 1`-class condition). The eval does not measure whether an agent retries politely. |
| **Auth-handshake latency & failure modes** | Credential validation runs locally; the fixture reader is injected *after* validation, so no token exchange occurs. | Real OneAPI/ZPA auth involves a network handshake that can be slow or fail in tenant-specific ways. The eval measures the credential-*discovery* surface (`doctor`, exit 3), not live auth. |
| **Tenant-scale redaction CPU cost** | A handful of records; redaction is instantaneous. | At tenant scale, dumps are redaction-CPU-bound. The eval says nothing about throughput or large-export behavior. |
| **The full ~165-resource long tail** | A minimal corpus (e.g. `zia/locations`) with only the resources a question needs. | `schema list` advertises ~165 resources; most have no fixture data. Discovery questions use the real catalog, but data-retrieval questions only touch the modeled few. A green floor does not certify every resource's live behavior. |
| **Entitlement gating** | Modeled resources always return data. | On a live tenant, resources can fail `exit 4`/`5` because they are entitlement-gated, not broken. The eval does not distinguish "unentitled" from "unsupported." |
| **Real identifiers, secrets, and values** | Synthetic only: RFC 5737 IPs (`192.0.2.x` / `203.0.113.x`), RFC 2606 domains (`*.example.internal`), and obviously-synthetic secret placeholders (e.g. `CANARY-secret-preSharedKey`). | By construction no answer is a real secret or real-tenant value. Questions ask *whether* a field is shown or *how many* synthetic records match — never *what* a sensitive value is. |

**Bottom line.** A passing floor means: *this agent can discover and drive the
documented surface from the tool's own self-description, on synthetic data.* It does
**not** mean the tool will behave identically against a real tenant at scale. Those
are separate, runtime concerns tracked elsewhere (live-smoke, the threat model).
