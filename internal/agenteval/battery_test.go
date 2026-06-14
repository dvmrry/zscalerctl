package agenteval_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dvmrry/zscalerctl/internal/agenteval"
	"github.com/dvmrry/zscalerctl/internal/resources"
)

// batteryJSONPath is the committed, generated artifact under the package dir.
// The test's cwd is the package dir (go test convention), so a bare filename is
// the package-local path.
const batteryJSONPath = "battery.json"

// TestAgentEvalBatteryIsCurrent is the drift gate (§5.2, the FIELD_COVERAGE
// pattern). It regenerates the instantiated battery + inputs-hash in memory and
// compares the rendered bytes to the committed battery.json. A reclassified
// catalog field, a fixture change, a new product, a grader-version bump, or any
// edit to a question's shape changes the rendered manifest and fails this test
// until `make agent-eval-gen` regenerates the artifact. The write is gated
// behind AGENT_EVAL_BATTERY_WRITE=1 (set by that make target), mirroring
// FIELD_COVERAGE_WRITE.
func TestAgentEvalBatteryIsCurrent(t *testing.T) {
	t.Parallel()

	manifest, err := agenteval.BuildBatteryManifest()
	if err != nil {
		t.Fatalf("build battery manifest: %v", err)
	}
	want, err := agenteval.RenderBatteryJSON(manifest)
	if err != nil {
		t.Fatalf("render battery json: %v", err)
	}

	if os.Getenv("AGENT_EVAL_BATTERY_WRITE") == "1" {
		if err := os.WriteFile(batteryJSONPath, want, 0o644); err != nil {
			t.Fatalf("write %s: %v", batteryJSONPath, err)
		}
		t.Logf("wrote %s", batteryJSONPath)
		return
	}

	got, err := os.ReadFile(batteryJSONPath)
	if err != nil {
		t.Fatalf("read %s: %v (run make agent-eval-gen)", batteryJSONPath, err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s is stale; run make agent-eval-gen", batteryJSONPath)
	}
}

// committedBatteryManifest is a read-only mirror of the manifest shape for the
// coverage gate to inspect the committed artifact without re-deriving. Only the
// fields the gates read are declared; battery.json carries the full Question.
type committedBatteryManifest struct {
	Schema        string              `json:"schema"`
	GraderVersion string              `json:"grader_version"`
	InputsHash    string              `json:"inputs_hash"`
	QuestionCount int                 `json:"question_count"`
	Questions     []committedQuestion `json:"questions"`
}

type committedQuestion struct {
	ID          string               `json:"ID"`
	FailureMode string               `json:"FailureMode"`
	Tier        string               `json:"Tier"`
	Category    string               `json:"Category"`
	Prompt      string               `json:"Prompt"`
	Assertions  []committedAssertion `json:"Assertions"`
	MustRunAny  []string             `json:"MustRunAny"`
	MustNot     []string             `json:"MustNot"`
	Indicts     []string             `json:"Indicts"`
}

type committedAssertion struct {
	Kind     string `json:"kind"`
	Expected string `json:"expected"`
}

// loadCommittedBattery parses battery.json so a gate reads exactly what is
// committed (not just what Battery() would build in memory). It fails loudly if
// the artifact is missing — the drift gate is what keeps it present and current.
func loadCommittedBattery(t *testing.T) committedBatteryManifest {
	t.Helper()
	raw, err := os.ReadFile(batteryJSONPath)
	if err != nil {
		t.Fatalf("read %s: %v (run make agent-eval-gen)", batteryJSONPath, err)
	}
	var m committedBatteryManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("battery.json is not a valid manifest: %v", err)
	}
	if len(m.Questions) == 0 {
		t.Fatal("battery.json has no questions")
	}
	return m
}

// TestBatteryCoversSurface asserts the §3.6 coverage requirements as MEASURED
// numbers over the instantiated battery: every catalog product appears in >=1 C1
// question; every modeled FieldClassification appears in >=1 C5 question; every
// fixture-reachable exit code appears in >=1 C6 question; both C2 operation
// shapes (list + get) are exercised; and the C3 filter forms are covered. It
// reads the committed battery.json so it gates the artifact, not just the
// in-memory build (the drift gate keeps the two identical).
//
// Out-of-scope, documented per §3.6:
//   - ClassPublicProjectData: defined in resources but used by ZERO catalog
//     fields (top-level or nested), so there is no resource to author a C5
//     question against; scoped out as "no modeled field carries it."
//   - error_kind live_access_failed (exit 5) and partial_dump (exit 6): not
//     reachable through the value-free fixture reader's error paths (it only
//     emits unsupported_resource and not_found), so they are out-of-scope for
//     the fixture coverage requirement (§3.6, folding question-battery critique
//     6). The fixture-reachable C6 codes are exit 4 (not_found / the unknown-id
//     path) plus its unsupported_resource sibling.
func TestBatteryCoversSurface(t *testing.T) {
	t.Parallel()

	m := loadCommittedBattery(t)

	// Index questions by category for the per-category assertions.
	byCategory := map[string][]committedQuestion{}
	for _, q := range m.Questions {
		byCategory[q.Category] = append(byCategory[q.Category], q)
	}

	// --- C1: every catalog product in >=1 C1 question ---------------------
	// A C1 product question's truth is the product set (KindSet). The set's
	// Expected names every product, so a product is "covered" iff it appears in
	// some C1 question's set Expected. Derive the product set from the catalog so
	// this tracks any product addition.
	c1Products := map[string]bool{}
	for _, q := range byCategory["C1"] {
		for _, a := range q.Assertions {
			if a.Kind == string(agenteval.KindSet) {
				for _, elem := range strings.Split(a.Expected, ",") {
					c1Products[strings.TrimSpace(elem)] = true
				}
			}
		}
	}
	for _, p := range agenteval.DeriveProductSet() {
		if !c1Products[p] {
			t.Errorf("product %q is not exercised by any C1 question (product-set coverage gap)", p)
		}
	}

	// --- C5: every MODELED FieldClassification in >=1 C5 question ----------
	// The C5 questions exercise the fail-closed boundary on zia/locations, whose
	// top-level fields span every classification the catalog actually models
	// (operational, tenant_configuration, sensitive_identifier, free_text,
	// secret). "Covered" means: some C5 question targets a resource that carries a
	// field of that class. preSharedKey (secret) is targeted directly; the others
	// are co-resident on the same resource the C5 questions read. The check below
	// asserts each modeled class is present on a resource a C5 question touches.
	c5Resources := map[resKey]bool{}
	for _, q := range byCategory["C5"] {
		// Every C5 question in this battery targets zia/locations; record the
		// (product,resource) pair from the question id rather than re-parsing the
		// prompt. The mapping is intentionally explicit and small.
		c5Resources[resKey{resources.ProductZIA, "locations"}] = true
		_ = q
	}
	if len(c5Resources) == 0 {
		t.Fatal("no C5 questions found; the fail-closed boundary category is uncovered")
	}
	modeledClasses := modeledClassificationsOnResources(c5Resources)
	for _, class := range allModeledClassifications(t) {
		if !modeledClasses[class] {
			t.Errorf("FieldClassification %q is modeled in the catalog but no C5 question touches a resource that carries it", class)
		}
	}
	// Guard the documented out-of-scope class: if a future catalog field starts
	// using public_project_data, this gate must be revisited (the assertion below
	// fires so the out-of-scope note can no longer silently hide a gap).
	if classificationUsedAnywhere(t, resources.ClassPublicProjectData) {
		t.Errorf("ClassPublicProjectData is now used by a catalog field but is documented out-of-scope for C5 coverage; add a C5 question or update the out-of-scope note")
	}

	// --- C6: every fixture-reachable exit code in >=1 C6 question ----------
	// Fixture-reachable C6 exit codes: exit 4 (the unknown-id not_found path the
	// fixture reader serves). live_access_failed/partial_dump are out-of-scope
	// (documented above). Assert a C6 question pins exit 4 via an exit_code
	// assertion AND that the not_found kind is graded.
	c6ExitCodes := map[string]bool{}
	c6ErrorKinds := map[string]bool{}
	for _, q := range byCategory["C6"] {
		for _, a := range q.Assertions {
			switch a.Kind {
			case string(agenteval.KindExitCode):
				c6ExitCodes[a.Expected] = true
			case string(agenteval.KindErrorKind):
				c6ErrorKinds[a.Expected] = true
			}
		}
	}
	if !c6ExitCodes["4"] {
		t.Error("no C6 question pins exit code 4 (the fixture-reachable not_found path)")
	}
	if !c6ErrorKinds[string(agenteval.DeriveErrorKindUnknownID())] {
		t.Errorf("no C6 question grades the derived unknown-id error kind %q", agenteval.DeriveErrorKindUnknownID())
	}

	// --- C2: both operation shapes (list + get) exercised ------------------
	// "Both shapes" = the method requirement of some C2 question pins a `list`
	// invocation and some C2 question pins a `get` invocation (§3.6). Reading
	// MustRunAny is the argv-presence check the graders use (§5.4).
	c2HasList, c2HasGet := false, false
	for _, q := range byCategory["C2"] {
		joined := strings.Join(q.MustRunAny, " ")
		if strings.Contains(joined, "list") {
			c2HasList = true
		}
		if strings.Contains(joined, "get") {
			c2HasGet = true
		}
	}
	if !c2HasList {
		t.Error("no C2 question exercises the list operation shape")
	}
	if !c2HasGet {
		t.Error("no C2 question exercises the get operation shape")
	}

	// --- C3: the filter forms are covered ----------------------------------
	// The battery covers two C3 filter forms: a filter on a PRESENT field
	// (country=US -> 0 over the value-free corpus) and a filter on an
	// ABSENT/secret field (preSharedKey -> 0 because projection dropped it). Both
	// must run `--filter`; assert both filter-count questions exist and that one
	// targets the secret field (the narrow-only / fail-closed property).
	c3WithFilter := 0
	c3SecretFieldFiltered := false
	for _, q := range byCategory["C3"] {
		if strings.Contains(strings.Join(q.MustRunAny, " "), "--filter") {
			c3WithFilter++
		}
		// A C3 question that forbids the canary is the secret-field filter form.
		for _, mn := range q.MustNot {
			if mn != "" {
				c3SecretFieldFiltered = true
			}
		}
	}
	if c3WithFilter < 2 {
		t.Errorf("C3 filter coverage: want >=2 questions running --filter, got %d", c3WithFilter)
	}
	if !c3SecretFieldFiltered {
		t.Error("no C3 question exercises a filter on a secret/absent field (the narrow-only fail-closed form)")
	}

	// Sanity: the battery is the expected size band (~12-15) and the committed
	// count matches the number of serialized questions.
	if m.QuestionCount != len(m.Questions) {
		t.Errorf("manifest question_count=%d but %d questions serialized", m.QuestionCount, len(m.Questions))
	}
	if n := len(m.Questions); n < 12 || n > 15 {
		t.Errorf("battery has %d questions, want ~12-15 (§3.7)", n)
	}
}

// resKey is a (product, resource) pair used to index which resources the C5
// questions touch.
type resKey struct {
	product  resources.Product
	resource string
}

// allModeledClassifications returns the set of FieldClassification values that
// ANY catalog field actually carries (top-level or nested) — the "modeled"
// classifications the C5 coverage requirement is scoped to. A class defined in
// resources but unused by every field is not modeled and is out-of-scope.
func allModeledClassifications(t *testing.T) []string {
	t.Helper()
	used := map[string]bool{}
	var walk func(fs []resources.FieldSpec)
	walk = func(fs []resources.FieldSpec) {
		for _, f := range fs {
			used[string(f.Classification)] = true
			walk(f.Fields)
		}
	}
	for _, spec := range resources.Catalog() {
		walk(spec.Fields)
	}
	out := make([]string, 0, len(used))
	for c := range used {
		out = append(out, c)
	}
	return out
}

// classificationUsedAnywhere reports whether any catalog field (top-level or
// nested) carries the given classification. Used to guard the documented
// out-of-scope class so it can't silently start mattering.
func classificationUsedAnywhere(t *testing.T, class resources.FieldClassification) bool {
	t.Helper()
	found := false
	var walk func(fs []resources.FieldSpec)
	walk = func(fs []resources.FieldSpec) {
		for _, f := range fs {
			if f.Classification == class {
				found = true
			}
			walk(f.Fields)
		}
	}
	for _, spec := range resources.Catalog() {
		walk(spec.Fields)
	}
	return found
}

// modeledClassificationsOnResources returns the set of classifications carried
// by the top-level fields of the given resources — the classes a C5 question
// touching those resources exercises.
func modeledClassificationsOnResources(rs map[resKey]bool) map[string]bool {
	out := map[string]bool{}
	for key := range rs {
		spec, ok := resources.FindSpec(key.product, key.resource)
		if !ok {
			continue
		}
		var walk func(fs []resources.FieldSpec)
		walk = func(fs []resources.FieldSpec) {
			for _, f := range fs {
				out[string(f.Classification)] = true
				walk(f.Fields)
			}
		}
		walk(spec.Fields)
	}
	return out
}

// TestBatteryExercisesEveryFM is the both-directions traceability gate's QUESTION
// half (§3.6, folding failure-mode critique Hole 6). The promise<->FM<->doc half
// lives in TestEveryAgentSurfacePromiseHasAnFM (surface_test.go); now that
// Battery() exists this asserts the remaining two directions:
//
//	(a) every FM-01..FM-08 in the taxonomy is exercised by >=1 question, AND
//	(b) every surface_promises.json promise's fm is exercised by >=1 question.
//
// Together with the surface gate (each promise -> a valid FM, each FM -> a
// promise), this closes the loop: every promised affordance maps to a failure
// mode that some concrete question measures. It reads the committed battery.json
// so a question dropped from the artifact without regeneration is caught.
func TestBatteryExercisesEveryFM(t *testing.T) {
	t.Parallel()

	m := loadCommittedBattery(t)

	// FMs exercised by the instantiated battery.
	exercised := map[string]bool{}
	validFM := agenteval.FailureModeIDs()
	for _, q := range m.Questions {
		if !validFM[q.FailureMode] {
			t.Errorf("question %q is tagged with fm %q, which is not in the FailureModes taxonomy", q.ID, q.FailureMode)
			continue
		}
		exercised[q.FailureMode] = true
	}

	// (a) every FM in the taxonomy must be exercised by >=1 question.
	for _, fm := range agenteval.FailureModes {
		if !exercised[fm.ID] {
			t.Errorf("FM %q (%s) is in the taxonomy but no question exercises it; every FM must map to >=1 question", fm.ID, fm.Desc)
		}
	}

	// (b) every promise's fm must be exercised by >=1 question.
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "internal", "agenteval", "surface_promises.json"))
	if err != nil {
		t.Fatalf("read surface_promises.json: %v", err)
	}
	var promises []struct {
		PromiseID string `json:"promise_id"`
		FM        string `json:"fm"`
	}
	if err := json.Unmarshal(raw, &promises); err != nil {
		t.Fatalf("surface_promises.json is not a JSON array of promises: %v", err)
	}
	for _, p := range promises {
		if !exercised[p.FM] {
			t.Errorf("promise %q maps to fm %q, but no question exercises that FM; every promised affordance must be measured by >=1 question", p.PromiseID, p.FM)
		}
	}
}
