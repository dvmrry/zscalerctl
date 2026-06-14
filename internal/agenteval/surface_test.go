package agenteval_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dvmrry/zscalerctl/internal/agenteval"
)

// surfacePromise mirrors one entry of surface_promises.json: a promised
// affordance, the verbatim doc anchor that proves it, the FM it maps to, and
// which doc carries it.
type surfacePromise struct {
	PromiseID string `json:"promise_id"`
	Anchor    string `json:"anchor"`
	FM        string `json:"fm"`
	Doc       string `json:"doc"`
}

// TestEveryAgentSurfacePromiseHasAnFM is the traceability gate (§3.6). It never
// parses or interprets prose: it only checks that each declared anchor exists
// verbatim in its named doc, that each promise points at a valid FM, and that
// every FM in the taxonomy is claimed by at least one promise.
//
// This covers the promise<->FM<->doc half of §3.6. The "every FM -> >=1
// QUESTION" half lands once battery.go exists and the instantiated questions
// can be enumerated; failuremodes.go is already the shared FM source both
// halves read from.
func TestEveryAgentSurfacePromiseHasAnFM(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)

	raw, err := os.ReadFile(filepath.Join(root, "internal", "agenteval", "surface_promises.json"))
	if err != nil {
		t.Fatalf("read surface_promises.json: %v", err)
	}
	var promises []surfacePromise
	if err := json.Unmarshal(raw, &promises); err != nil {
		t.Fatalf("surface_promises.json is not a JSON array of promises: %v", err)
	}
	if len(promises) == 0 {
		t.Fatal("surface_promises.json is empty; the traceability gate has nothing to check")
	}

	// docPath resolves the registry's symbolic doc name to a concrete file under
	// the repo root. Any other doc value is rejected so a typo can't silently
	// skip the anchor check.
	docPath := func(doc string) (string, bool) {
		switch doc {
		case "AGENTS.md":
			return filepath.Join(root, "AGENTS.md"), true
		case "SKILL.md":
			return filepath.Join(root, "skills", "zscalerctl", "SKILL.md"), true
		default:
			return "", false
		}
	}

	// Cache doc contents so we read each file once.
	docCache := map[string]string{}
	loadDoc := func(t *testing.T, doc string) string {
		t.Helper()
		if body, ok := docCache[doc]; ok {
			return body
		}
		path, ok := docPath(doc)
		if !ok {
			t.Fatalf("promise references unknown doc %q; only \"AGENTS.md\" and \"SKILL.md\" are allowed", doc)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read doc %q at %s: %v", doc, path, err)
		}
		docCache[doc] = string(body)
		return docCache[doc]
	}

	validFM := agenteval.FailureModeIDs()
	seenFM := map[string]bool{}
	seenPromiseID := map[string]bool{}

	for _, p := range promises {
		if p.PromiseID == "" {
			t.Errorf("promise with empty promise_id (anchor %q); every promise needs a stable id", p.Anchor)
			continue
		}
		if seenPromiseID[p.PromiseID] {
			t.Errorf("duplicate promise_id %q in surface_promises.json", p.PromiseID)
		}
		seenPromiseID[p.PromiseID] = true

		// (b) the promise's FM must be a real taxonomy ID.
		if !validFM[p.FM] {
			t.Errorf("promise %q maps to fm %q, which is not in the FailureModes taxonomy (failuremodes.go)", p.PromiseID, p.FM)
		} else {
			seenFM[p.FM] = true
		}

		// (a) the anchor must occur verbatim in the named doc.
		if p.Anchor == "" {
			t.Errorf("promise %q has an empty anchor; the gate needs a verbatim marker to find in the doc", p.PromiseID)
			continue
		}
		body := loadDoc(t, p.Doc)
		if !strings.Contains(body, p.Anchor) {
			t.Errorf("promise %q anchor not found verbatim in %s:\n  anchor: %q\n  (renaming or dropping a promised affordance must update the doc or the registry)", p.PromiseID, p.Doc, p.Anchor)
		}
	}

	// (c) every FM in the taxonomy must be claimed by >=1 promise, so no failure
	// mode is left unmapped.
	for _, fm := range agenteval.FailureModes {
		if !seenFM[fm.ID] {
			t.Errorf("FM %q (%s) appears in the taxonomy but no promise in surface_promises.json maps to it; every FM must be claimed by >=1 promise", fm.ID, fm.Desc)
		}
	}
}
