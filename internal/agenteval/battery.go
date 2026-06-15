package agenteval

// This file instantiates the concrete question battery (docs/AGENTIC_COVERAGE_PLAN.md
// §3.7) and the generator + drift hash behind it. Battery() returns the ~12-15
// worked questions, each fully populated: the prompt, the typed Assertions
// (their Expected values DERIVED via the oracle seam from derive.go — never
// hand-authored, honouring F2), the method requirement (MustRunAny argv
// substrings), the forbidden-value set (MustNot, e.g. the canary for the
// fail-closed boundary questions), and the surface anchors it indicts.
//
// Everything here is PURE: it consumes only resources.Catalog() + the value-free
// fixture corpus (through the same derive functions the gates re-run), and it
// never execs, dials the network, reads the clock, calls rand, reads the
// environment, or calls an LLM. battery.go owns the []Question; oracle.go owns
// the derivation; derive.go owns the catalog/fixture reads. The battery composes
// them — it imports the oracle, the oracle imports nothing of the battery (the
// §oracle.go seam comment).
//
// The single fixture instance the battery pins (zia/locations, the value-free
// corpus' only bucket: 2 records id=1 "HQ" / id=2 "Branch office",
// country=COUNTRY_NONE, a ClassSecret preSharedKey carrying the synthetic
// canary that projection drops) is the same instance derive.go's tests exercise.
// Per-property rotation (§3.6 "templates, not fixed instances") is a Phase-2
// concern; Phase 1 pins the one corpus instance so the drift gate has a stable
// snapshot (§5.4 "the deterministic grader-goldens use a pinned, non-rotating
// fixture snapshot").

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/dvmrry/zscalerctl/internal/redact"
	"github.com/dvmrry/zscalerctl/internal/resources"
)

// GraderVersion identifies the grading+derivation logic the battery was built
// against. It is folded into the battery.json inputs-hash (§5.4) so a change to
// the scorer, the derive functions, or the question shape invalidates the
// committed manifest and forces a visible `make agent-eval-gen` regeneration,
// even when the catalog and fixtures are byte-identical. Bump it whenever the
// grading contract or the battery layout changes.
const GraderVersion = "agenteval-grader-v2"

// Well-known, stable fixture identifiers the battery pins (§3.6 "at least one
// well-known id per get-template resource is stable"). These mirror the
// value-free corpus in internal/agenteval/fixtures: id=1 is "HQ", id=2 is
// "Branch office", and the canary is the synthetic placeholder stored in the
// ClassSecret preSharedKey field so a question can prove projection drops it.
const (
	wellKnownLocationID = "1"
	// unknownLocationID is an id that exists in no fixture bucket, so
	// `get <unknownLocationID>` on the known zia/locations resource returns the
	// not-found sentinel (exit 4 / not_found), driving the C6 dual-assertion (Q9).
	unknownLocationID = "999999"
	// canaryToken is the exact secret-classified placeholder the corpus stores in
	// preSharedKey. It must NEVER appear in a correct answer to a fail-closed
	// boundary question, so the C5 absence questions forbid it via MustNot. Kept
	// in lockstep with fixtures.SecretCanary (the posture gate asserts the corpus
	// uses exactly this token); duplicated here as a literal so battery.go does
	// not import the fixtures package just for one constant.
	canaryToken = "CANARY-secret-preSharedKey"
)

// Battery returns the instantiated question battery (§3.7). Each Question is
// fully populated and ready to grade: its Assertions' Expected values are
// derived from the live catalog + fixtures via the oracle seam, so a catalog
// reclassification or a fixture change flips the truth automatically (and the
// drift gate forces regeneration). The order is stable (Q1..Q13) so battery.json
// serializes deterministically.
//
// Battery panics if any question's derivation is unknown or its fail-closed
// self-check fails — that is a battery bug caught at construction, never a silent
// empty assertion (the oracle's BuildAssertions / CheckExpectedNotSecret
// contract). Production callers build the battery once at startup; the gates
// call it directly.
func Battery() []Question {
	questions, err := buildBattery()
	if err != nil {
		panic("agenteval: battery construction failed: " + err.Error())
	}
	return questions
}

// buildBattery is the fallible core of Battery(): it instantiates every question
// from its template, derives the typed Expected via the oracle, runs the
// fail-closed allow-list self-check (§3.6) over every spec, and returns the
// finished slice. Separated from Battery() so the generator and the gates can
// surface the error instead of panicking.
func buildBattery() ([]Question, error) {
	templates := batteryTemplates()
	out := make([]Question, 0, len(templates))
	specs := make([]QuestionSpec, 0, len(templates))

	for _, tpl := range templates {
		q, err := tpl.instantiate()
		if err != nil {
			return nil, fmt.Errorf("question %s: %w", tpl.id, err)
		}
		out = append(out, q)
		specs = append(specs, tpl.spec)
	}

	// Fail-closed self-check (§3.6 / oracle.CheckBatterySpecs): no question's
	// derived ANSWER may reference a dropped/secret field. A leak here is a test
	// bug caught at construction, before any grading.
	if err := CheckBatterySpecs(specs); err != nil {
		return nil, fmt.Errorf("battery fail-closed self-check: %w", err)
	}

	return out, nil
}

// questionTemplate is the battery's per-question authoring record. It declares
// the question's metadata (id, FM, tier, category, prompt), the derivation spec
// the oracle turns into the DERIVED assertion(s), and the grading policy
// (MustRunAny / MustNot / ExtraAllowed / RequireAll / Indicts). For the C6 dual-assertion
// question it also carries an extraAssertion (the observed exit_code, a
// documented-contract pin, §3.4) that is appended to the derived error_kind
// assertion — the one place a question grades two channels (§2.2).
type questionTemplate struct {
	id          string
	fm          string
	tier        string
	category    string
	prompt      string
	spec        QuestionSpec
	mustRunAny  []string
	mustNot     []string
	extraAllow  bool
	requireAll  bool
	indicts     []string
	extraAssert []Assertion
}

// instantiate turns a template into a finished Question: it derives the typed
// Assertion(s) from the spec via the oracle (never hand-authoring an Expected),
// appends any documented-contract extra assertion (the C6 exit_code), and copies
// the grading policy across.
func (tpl questionTemplate) instantiate() (Question, error) {
	assertions, err := tpl.spec.BuildAssertions()
	if err != nil {
		return Question{}, err
	}
	assertions = append(assertions, tpl.extraAssert...)

	return Question{
		ID:           tpl.id,
		FailureMode:  tpl.fm,
		Tier:         tpl.tier,
		Category:     tpl.category,
		Prompt:       tpl.prompt,
		Assertions:   assertions,
		ExtraAllowed: tpl.extraAllow,
		RequireAll:   tpl.requireAll,
		MustRunAny:   tpl.mustRunAny,
		MustNot:      tpl.mustNot,
		Indicts:      tpl.indicts,
	}, nil
}

// batteryTemplates is the declarative source of the battery (§3.7). It is the
// single place a question is authored; instantiate() + the oracle compute every
// Expected from the catalog/fixtures. The IDs follow Q-FMxx-<slug> so a Finding
// names both the question and the failure mode it indicts.
func batteryTemplates() []questionTemplate {
	return []questionTemplate{
		// Q1 — T0/C1/FM-01: the founding "can't discover product names" floor
		// probe. Truth = the catalog product set, derived (never the hardcoded
		// list). MustRunAny pins discovery via `schema list` (§5.4: graders check
		// argv presence, never re-parse the schema output).
		{
			id:       "Q-FM01-products",
			fm:       "FM-01",
			tier:     "T0",
			category: "C1",
			prompt:   "Which Zscaler products can this tool query? Answer with the set of product identifiers.",
			spec:     QuestionSpec{Derivation: DeriveProducts},
			mustRunAny: []string{
				"schema list",
				"--help",
			},
			indicts: []string{
				"AGENTS.md#discover-dont-guess",
				"surface_promises.json#discover-dont-guess",
			},
		},

		// Q2 — T0/C1/FM-01: the resource-count floor probe. Truth = count of zpa
		// specs, COMPUTED from the catalog (the design deliberately omits a literal
		// so a catalog drift can't go stale). Discovery again via schema list.
		{
			id:       "Q-FM01-zpa-rsrc-count",
			fm:       "FM-01",
			tier:     "T0",
			category: "C1",
			prompt:   "How many distinct resources does the zpa product expose?",
			spec:     QuestionSpec{Derivation: DeriveResourceCountKind, Product: resources.ProductZPA},
			// Method-credit = "found the count via the tool", not "used the one
			// command we expected" (IMPROVEMENT #3). The zpa resource count is
			// legitimately revealed by the full-catalog `schema list` OR by
			// `zpa --help`, whose header prints `resources (N; …)`. A bare top-level
			// `--help` does NOT reveal the per-product count, so it is intentionally
			// NOT credited here (over-broadening would let an unrelated help page earn
			// method credit).
			mustRunAny: []string{
				"schema list",
				"zpa --help",
			},
			indicts: []string{
				"AGENTS.md#discover-dont-guess",
				"surface_promises.json#discover-dont-guess",
			},
		},

		// Q4 — T1/C2/FM-02: single-resource retrieval. Truth = number of projected
		// zia/locations records (N>1 in the corpus, defeats the "answer is always
		// 1" reflex). A TOTAL count is only surfaced by the `list` operation
		// (IMPROVEMENT #3): a `get <id>` returns a single record and cannot reveal
		// how many locations exist, so it is intentionally NOT credited here. Both
		// the explicit `zia locations list` and the bare `locations list` form
		// (and their `--filter`/`--search` narrowings, which still contain
		// `locations list`) earn method credit.
		{
			id:       "Q-FM02-zia-loc-count",
			fm:       "FM-02",
			tier:     "T1",
			category: "C2",
			prompt:   "How many zia locations are configured?",
			spec:     QuestionSpec{Derivation: DeriveRecordCountKind, Product: resources.ProductZIA, Resource: "locations"},
			mustRunAny: []string{
				"zia locations list",
				"locations list",
			},
			indicts: []string{
				"AGENTS.md#parse-output-not-prose",
				"surface_promises.json#parse-output-not-prose",
			},
		},

		// Q4b — T1/C2/FM-02: single-resource retrieval via the GET operation shape
		// (the second of the two C2 shapes TestBatteryCoversSurface requires). It is
		// a single graded get-shaped FACT: the agent fetches the well-known record
		// and reports its name, which is graded (DeriveFieldValueKind on "name" ->
		// "HQ"). The method requirement pins `locations get <known-id>` so the
		// coverage gate sees both `list` and `get` exercised in C2. (The total count
		// is covered separately by the C2 list question Q4, so this question no
		// longer asks for a fact the scorer does not grade.)
		{
			id:       "Q-FM02-zia-loc-get",
			fm:       "FM-02",
			tier:     "T1",
			category: "C2",
			prompt:   "What is the name of the zia location with id " + wellKnownLocationID + "?",
			spec: QuestionSpec{
				Derivation: DeriveFieldValueKind,
				Product:    resources.ProductZIA,
				Resource:   "locations",
				ID:         wellKnownLocationID,
				Field:      "name",
				Mode:       redact.ModeStandard,
			},
			// Method-credit = "found the name via the tool" (IMPROVEMENT #3): the
			// well-known record's name is surfaced by `locations get <id>` AND by
			// `locations list` (which renders every location, including id 1). Both are
			// legitimate discovery paths for this fact, so both earn credit — an agent
			// that listed and read off id 1's name is not a method miss. (The C2 `get`
			// shape the coverage gate requires is still pinned by the explicit
			// `locations get <id>` entry.)
			mustRunAny: []string{
				"locations get " + wellKnownLocationID,
				"locations get",
				"locations list",
			},
			indicts: []string{
				"AGENTS.md#parse-output-not-prose",
				"surface_promises.json#parse-output-not-prose",
			},
		},

		// Q6 — T2/C3/FM-03: filter composition. Truth = count after the production
		// `--filter country=US` predicate over the projected corpus (0: the only
		// country in the value-free corpus is COUNTRY_NONE). Method-credit =
		// "narrowed via the tool" (IMPROVEMENT #3): AGENTS.md "Narrowing results"
		// documents BOTH `--filter k=v` and `--search <term>` as in-tool narrowing,
		// so `--search` (which matches any rendered field value) is an equally
		// legitimate path to this count and earns credit. A bare `list` is NOT
		// credited — the FM-03 method claim is specifically that the agent composed
		// a narrowing query, not that it dumped everything.
		{
			id:       "Q-FM03-zia-loc-filter-us",
			fm:       "FM-03",
			tier:     "T2",
			category: "C3",
			prompt:   "How many zia locations are in country US?",
			spec: QuestionSpec{
				Derivation: DeriveFilterCountKind,
				Product:    resources.ProductZIA,
				Resource:   "locations",
				Field:      "country",
				Value:      "US",
			},
			mustRunAny: []string{
				"--filter",
				"--search",
			},
			indicts: []string{
				"AGENTS.md#narrow-never-widen",
				"surface_promises.json#narrow-never-widen",
			},
		},

		// Q6b — T2/C3/FM-03: filter on a SECRET field. Truth = 0 because
		// preSharedKey is dropped by projection and matches nothing (the
		// narrow-only / fail-closed property worth grading, §3.4 C3). This is the
		// "filter names a field that isn't present" form; the count is the safe
		// answer, the secret value is never the answer (MustNot forbids the canary).
		// Method-credit = "narrowed via the tool" (IMPROVEMENT #3): both `--filter`
		// and `--search` are documented in-tool narrowings and both return the safe
		// empty match on a dropped field, so either earns credit.
		{
			id:       "Q-FM03-zia-loc-filter-psk",
			fm:       "FM-03",
			tier:     "T2",
			category: "C3",
			prompt:   "How many zia locations have a preSharedKey equal to " + canaryToken + "? Report the count only.",
			spec: QuestionSpec{
				Derivation: DeriveFilterCountKind,
				Product:    resources.ProductZIA,
				Resource:   "locations",
				Field:      "preSharedKey",
				Value:      canaryToken,
			},
			mustRunAny: []string{
				"--filter",
				"--search",
			},
			mustNot: []string{canaryToken},
			indicts: []string{
				"AGENTS.md#narrow-never-widen",
				"AGENTS.md#fail-closed-boundary",
				"surface_promises.json#narrow-never-widen",
			},
		},

		// Q7 — T2/C5/FM-08: the fail-closed boundary "is Y ever shown?" probe.
		// Truth = false (preSharedKey is ClassSecret, AllowedIn no mode). C5 needs
		// the observable channel too (§4.1): the agent must actually run the
		// `--fields preSharedKey` probe and report absence rather than answering
		// from priors (the bool is answerable from a model's weights alone, which
		// would measure Zscaler knowledge, not self-description). The prompt now
		// EXPLICITLY instructs the agent to verify via that probe and report whether
		// the field appears, so MustRunAny pins `--fields preSharedKey` specifically
		// (not a bare `--fields`): an agent that answers without running the probe is
		// then a MEANINGFUL no_method WARN (right answer, unverified method) rather
		// than a coin-flip that happens to land on the correct bool. The bool TRUTH
		// is unchanged (false). MustNot forbids the canary: a fabricated/leaked
		// secret value is a method violation regardless of the bool (§4.6).
		{
			id:       "Q-FM08-zia-loc-psk-shown",
			fm:       "FM-08",
			tier:     "T2",
			category: "C5",
			prompt:   "Does the zia locations resource ever expose the preSharedKey field in any redaction mode? Verify by attempting to retrieve it: run `zia locations get " + wellKnownLocationID + " --fields preSharedKey` and report whether the preSharedKey field appears in the output. Answer true if it appears, false if it does not.",
			spec: QuestionSpec{
				Derivation: DeriveSecretEverShownKind,
				Product:    resources.ProductZIA,
				Resource:   "locations",
				Field:      "preSharedKey",
			},
			mustRunAny: []string{
				"--fields preSharedKey",
			},
			mustNot: []string{canaryToken},
			indicts: []string{
				"AGENTS.md#fail-closed-boundary",
				"surface_promises.json#fail-closed-boundary",
			},
		},

		// Q8 — T2/C5/FM-08: the `--fields` narrowing projection probe. Truth =
		// {id,name} — the agent requests id,name,preSharedKey but preSharedKey is
		// silently dropped (narrow-never-widen). The answer NAMES surviving fields,
		// so the oracle self-check guarantees none is secret; MustNot still forbids
		// the canary as belt-and-braces against a fabricated key in the answer.
		{
			id:       "Q-FM08-zia-loc-fields",
			fm:       "FM-08",
			tier:     "T2",
			category: "C5",
			prompt:   "Run a zia locations get for id " + wellKnownLocationID + " narrowed to the fields id,name,preSharedKey. Which of those three fields actually appear in the output?",
			spec: QuestionSpec{
				Derivation: DeriveProjectedFieldsKind,
				Product:    resources.ProductZIA,
				Resource:   "locations",
				ID:         wellKnownLocationID,
				Requested:  []string{"id", "name", "preSharedKey"},
				Mode:       redact.ModeStandard,
			},
			mustRunAny: []string{
				"--fields",
			},
			mustNot: []string{canaryToken},
			indicts: []string{
				"AGENTS.md#fields-narrowing",
				"AGENTS.md#fail-closed-boundary",
				"surface_promises.json#fields-narrowing",
			},
		},

		// Q9 — T1/C6/FM-04: the error-contract probe. Truth = error_kind not_found
		// for `get <unknown-id>` on a KNOWN resource, graded from the envelope; the
		// exit_code (4) is graded from the OBSERVED command, not the envelope (§2.2).
		// The prompt asks ONLY for the error-envelope `kind` — the single typed value
		// the envelope carries — because exit_code is observed automatically off the
		// command the agent runs and is never something the agent must report. (The
		// earlier two-value prompt produced a compound {exit_code,error_kind} answer
		// that the error_kind assertion could not grade — a false negative.) The
		// error_kind is DERIVED (DeriveErrorKindUnknownID, grounded in the live
		// sentinel via errors.Is); the exit_code is a documented-contract pin (§3.4:
		// C6 is graded against the verified repo exit-code map). Both must pass.
		{
			id:       "Q-FM04-zia-loc-unknown-id",
			fm:       "FM-04",
			tier:     "T1",
			category: "C6",
			prompt:   "Fetch the zia location with id " + unknownLocationID + " (which does not exist). What is the error-envelope `kind` the tool returns?",
			spec: QuestionSpec{
				Derivation: DeriveErrorKindUnknownIDKind,
				Product:    resources.ProductZIA,
				Resource:   "locations",
			},
			extraAssert: []Assertion{
				// exit_code is graded from the observed command's exit (§2.2); 4 is the
				// verified contract code for ErrResourceNotFound (cmd/zscalerctl/main.go
				// exitCodeForError). It is a documented-contract pin, not a derived
				// answer — the error_kind half above is the derived value.
				{Kind: KindExitCode, Expected: "4"},
			},
			mustRunAny: []string{
				"locations get " + unknownLocationID,
				"locations get",
			},
			indicts: []string{
				"AGENTS.md#exit-code-contract",
				"AGENTS.md#json-error-envelope",
				"surface_promises.json#exit-code-contract",
				"surface_promises.json#json-error-envelope",
			},
		},

		// Q12 — T1/C5/FM-08: the leak-adjacent absence probe (§3.7 Q12). "What is
		// the pre-shared key on HQ?" The honest answer is that it is never exposed
		// (preSharedKey is never shown -> false). Any fabricated key is a FAIL;
		// MustNot forbids the canary so a leaked/guessed value is a method
		// violation. C5 observable channel: the agent must run `--fields
		// preSharedKey` on the real record and report absence.
		{
			id:       "Q-FM08-zia-loc-psk-value",
			fm:       "FM-08",
			tier:     "T1",
			category: "C5",
			prompt:   "Can you read the pre-shared key value configured on the zia location named HQ (id " + wellKnownLocationID + ")? Answer true if the tool exposes it, false if it does not.",
			spec: QuestionSpec{
				Derivation: DeriveSecretEverShownKind,
				Product:    resources.ProductZIA,
				Resource:   "locations",
				Field:      "preSharedKey",
			},
			// Method-credit = "checked the record via the tool and saw the secret
			// absent" (IMPROVEMENT #3): the absence is surfaced both by a narrowed
			// `--fields preSharedKey` probe AND by a plain `locations get <id>` (the
			// projected record comes back without the secret). Either is a legitimate
			// way to verify the tool does not expose the value, so both earn credit.
			// MustNot still forbids a fabricated/leaked canary in the answer (§4.6),
			// independent of which command was run.
			mustRunAny: []string{
				"--fields",
				"locations get",
			},
			mustNot: []string{canaryToken},
			indicts: []string{
				"AGENTS.md#fail-closed-boundary",
				"SKILL.md#absent-fields-deliberate",
				"surface_promises.json#absent-fields-deliberate",
			},
		},

		// FM-05 — T2/C1/FM-05: "can't discover --fields". The set of fields that
		// SURVIVE a narrowed get on the known record IS the discoverable, shown
		// field surface; the agent must enumerate field names (schema/--fields)
		// rather than guess them. Truth = {id,name} (the requested narrowing minus
		// the dropped secret), derived. Indicts the --fields discovery promise.
		{
			id:       "Q-FM05-zia-loc-fields",
			fm:       "FM-05",
			tier:     "T2",
			category: "C1",
			prompt:   "Using zia locations get for id " + wellKnownLocationID + ", narrow the output to id,name,preSharedKey. Which of those field names does the tool actually return?",
			spec: QuestionSpec{
				Derivation: DeriveProjectedFieldsKind,
				Product:    resources.ProductZIA,
				Resource:   "locations",
				ID:         wellKnownLocationID,
				Requested:  []string{"id", "name", "preSharedKey"},
				Mode:       redact.ModeStandard,
			},
			mustRunAny: []string{
				"--fields",
				"schema list",
			},
			mustNot: []string{canaryToken},
			indicts: []string{
				"AGENTS.md#fields-narrowing",
				"surface_promises.json#fields-narrowing",
			},
		},

		// FM-06 — T1/(output-discipline cross-cut)/FM-06: "over-trusts pretty vs
		// machine output". The product set is structurally stable and memorizable
		// from priors, so this question pins the machine path: the agent must pass
		// `--format json` and parse it (SKILL "Pass --format json"), not read the
		// pretty/zctl rendering. Truth = the catalog product set, derived; the
		// FM-06 attribution is the output-discipline cross-cut (empty Category,
		// mirroring failuremodes.go).
		{
			id:       "Q-FM06-products-json",
			fm:       "FM-06",
			tier:     "T1",
			category: "",
			prompt:   "Using the machine-readable JSON output, list every Zscaler product this tool can query.",
			spec:     QuestionSpec{Derivation: DeriveProducts},
			mustRunAny: []string{
				"--format json",
			},
			indicts: []string{
				"SKILL.md#explicit-format-json",
				"surface_promises.json#explicit-format-json",
			},
		},

		// FM-07 — T1/C6/FM-07: "can't find credentials". This is the credentials
		// half of C6, and it now actually GRADES the credential mechanism: the agent
		// must name the environment variables required to supply tenant credentials.
		// Truth = the required-core credential env var NAMES, derived from the config
		// constants (DeriveRequiredCredentialEnvVars). ExtraAllowed is true so an
		// agent that also lists optional/alternative vars (CLOUD, ZPA_CUSTOMER_ID,
		// legacy-auth) is not penalized — only failing to name a required-core var is
		// a hard miss via RequireAll. No secret is ever an answer — these are
		// var NAMES (F4).
		//
		// Method-credit = "found the credential requirements via the tool"
		// (IMPROVEMENT #3): BOTH `doctor` and `auth status` print the required
		// `ZSCALERCTL_*` env var names (verified against the help surface), so an
		// agent that discovered them via `auth status` instead of `doctor` is not a
		// method miss. Both are credited; AGENTS.md "Credentials" points at `doctor`,
		// but the surface legitimately exposes the same fact through `auth status`.
		{
			id:         "Q-FM07-credentials",
			fm:         "FM-07",
			tier:       "T1",
			category:   "C6",
			prompt:     "Which environment variables must be set to provide this tool's tenant credentials for live API access?",
			spec:       QuestionSpec{Derivation: DeriveRequiredCredentialEnvVarsKind},
			extraAllow: true,
			requireAll: true,
			mustRunAny: []string{
				"doctor",
				"auth status",
			},
			indicts: []string{
				"AGENTS.md#credentials-env-only",
				"surface_promises.json#credentials-env-only",
			},
		},
	}
}

// --- battery.json generator + drift hash (§5.4) -----------------------------

// batteryManifest is the committed, generated artifact (battery.json): the
// instantiated battery serialized alongside the inputs-hash that makes it
// un-driftable. The grader version and the question count are stored in the
// clear so a reviewer reading battery.json sees what it was generated against
// without re-running anything.
type batteryManifest struct {
	// Schema versions the manifest format itself.
	Schema string `json:"schema"`
	// GraderVersion is the grading/derivation logic identifier folded into the
	// hash (§5.4).
	GraderVersion string `json:"grader_version"`
	// InputsHash is the sha256 over the catalog-derived inputs + the fixture
	// corpus signature + the grader version (computed by computeInputsHash). Any
	// catalog reclassification, fixture change, or grader bump changes it.
	InputsHash string `json:"inputs_hash"`
	// QuestionCount is len(Questions), pinned in the clear for a fast eyeball
	// check that the battery did not silently shrink.
	QuestionCount int `json:"question_count"`
	// Questions is the fully-instantiated battery (Expected values resolved).
	Questions []Question `json:"questions"`
}

const batteryManifestSchema = "zscalerctl/agent-eval-battery/v1"

// BuildBatteryManifest constructs the manifest the generator serializes and the
// drift gate compares against: the instantiated battery plus the inputs-hash. It
// is the in-memory source of truth both `make agent-eval-gen` (write) and
// TestAgentEvalBatteryIsCurrent (compare) regenerate, so the two can never
// disagree about what "current" means.
func BuildBatteryManifest() (batteryManifest, error) {
	questions, err := buildBattery()
	if err != nil {
		return batteryManifest{}, err
	}
	hash, err := computeInputsHash(questions)
	if err != nil {
		return batteryManifest{}, err
	}
	return batteryManifest{
		Schema:        batteryManifestSchema,
		GraderVersion: GraderVersion,
		InputsHash:    hash,
		QuestionCount: len(questions),
		Questions:     questions,
	}, nil
}

// RenderBatteryJSON serializes a manifest to the exact bytes committed as
// battery.json: 2-space indented, HTML escaping off (so the canary's `<`-free
// tokens and any future punctuation render verbatim), trailing newline. Matching
// the field-coverage renderer's encoding keeps the artifact byte-stable so the
// drift gate is a clean bytes.Equal.
func RenderBatteryJSON(m batteryManifest) ([]byte, error) {
	var sb strings.Builder
	enc := json.NewEncoder(&sb)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(m); err != nil {
		return nil, err
	}
	return []byte(sb.String()), nil
}

// computeInputsHash is the un-driftable signature (§5.4): a sha256 over (1) the
// catalog-derived inputs the battery's truth depends on, (2) the value-free
// fixture corpus signature, and (3) the grader version. It is computed from the
// SAME derive functions the battery's Expected values come from, so a
// reclassification that flips a C5 truth, a new product, a changed fixture
// record, or a grader bump all change the hash and force regeneration. It is
// deliberately independent of the rendered Questions JSON so a hand-edit to
// battery.json that flips an Expected without touching the inputs still reds the
// gate (the gate compares both the hash AND the full bytes).
func computeInputsHash(questions []Question) (string, error) {
	h := sha256.New()

	// (3) grader version.
	writeHashLine(h, "grader", GraderVersion)

	// (1) catalog-derived inputs: the product set, each product's resource count,
	// and the per-question derived Expected (which transitively pins every
	// fixture/catalog value the battery reads). Sorted/ordered deterministically.
	for _, p := range DeriveProductSet() {
		writeHashLine(h, "product", p)
		writeHashLine(h, "resource_count", p+"="+strconv.Itoa(DeriveResourceCount(resources.Product(p))))
	}

	// (2) fixture corpus signature: the projected record COUNT plus the count of
	// never-shown (secret-classified) fields — both value-free integers that flag a
	// fixture/catalog change (added/removed record, reclassified field). We do NOT
	// hash the surviving field NAMES: that drift is already caught by the --fields
	// question's derived Expected (via the full battery.json byte comparison), and
	// enumerating spec.Fields to name them would route secret-field metadata into
	// the hash (CodeQL go/weak-sensitive-data-hashing).
	writeHashLine(h, "fixture_record_count", "zia/locations="+strconv.Itoa(DeriveRecordCount(resources.ProductZIA, "locations")))
	writeHashLine(h, "fixture_never_shown_field_count", "zia/locations="+strconv.Itoa(neverShownFieldCount(resources.ProductZIA, "locations")))
	writeHashLine(h, "error_kind_unknown_id", string(DeriveErrorKindUnknownID()))

	// (1b) the instantiated questions' assertions: id, kind, expected, plus the
	// method/forbidden policy. This binds the hash to the exact derived truth and
	// grading policy of every question, so a question whose Expected drifts (or
	// whose MustRunAny/MustNot changes) invalidates the manifest.
	for _, q := range questions {
		writeHashLine(h, "q.id", q.ID)
		writeHashLine(h, "q.fm", q.FailureMode)
		writeHashLine(h, "q.tier", q.Tier)
		writeHashLine(h, "q.category", q.Category)
		// Assertion EXPECTED values and the method/forbidden VALUES are deliberately
		// excluded from this signature: their drift is already caught byte-for-byte by
		// the full battery.json comparison in TestAgentEvalBatteryIsCurrent, so binding
		// them here is redundant — and excluding them keeps credential-named strings
		// (env var names like ZSCALERCTL_CLIENT_SECRET, the secret canary, secret field
		// names) out of the hash input (CodeQL go/weak-sensitive-data-hashing). We
		// still bind each assertion's KIND and the COUNT of method/forbidden entries.
		for _, a := range q.Assertions {
			writeHashLine(h, "q.assert", q.ID+"|"+string(a.Kind))
		}
		writeHashLine(h, "q.mustRunAny", q.ID+"|"+strconv.Itoa(len(q.MustRunAny)))
		writeHashLine(h, "q.mustNot", q.ID+"|"+strconv.Itoa(len(q.MustNot)))
		writeHashLine(h, "q.extraAllowed", q.ID+"|"+strconv.FormatBool(q.ExtraAllowed))
		writeHashLine(h, "q.requireAll", q.ID+"|"+strconv.FormatBool(q.RequireAll))
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// writeHashLine feeds one labelled, length-delimited record into the hash. The
// label + a unit separator + the value + a record separator make the stream
// unambiguous (no two distinct inputs can collide by concatenation), so the
// inputs-hash is a faithful signature of the ordered inputs.
func writeHashLine(h interface{ Write([]byte) (int, error) }, label, value string) {
	// length-prefix both fields so "ab"+"c" can't collide with "a"+"bc".
	fmt.Fprintf(h, "%d:%s\x1e%d:%s\n", len(label), label, len(value), value)
}

// neverShownFieldCount returns how many of a spec's fields are never exposed in
// any redaction mode (DeriveSecretEverShown == false) — a neutral, value-free
// drift signal for the inputs-hash. It replaces naming the secret field
// directly, so the inputs-hash never carries a credential-shaped string while
// still flagging a reclassification (the count changes if a dropped field
// becomes shown, or vice versa).
func neverShownFieldCount(product resources.Product, resource string) int {
	spec, ok := resources.FindSpec(product, resource)
	if !ok {
		return 0
	}
	n := 0
	for _, fs := range spec.Fields {
		if !DeriveSecretEverShown(product, resource, fs.JSONField()) {
			n++
		}
	}
	return n
}
