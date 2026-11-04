package machine

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"zscalerctl/internal/resources"
)

// TestReportResourcesStayOutOfMachine: with the real catalog, which has ZDX
// report resources, the machine manifest names none and the executor refuses
// the report operation.
func TestReportResourcesStayOutOfMachine(t *testing.T) {
	t.Parallel()

	catalog := resources.Catalog()
	hasReport := false
	for _, spec := range catalog {
		hasReport = hasReport || spec.EffectiveShape() == resources.ShapeReport
	}
	if !hasReport {
		t.Fatal("catalog has no report resource; this test needs one")
	}
	manifest, err := json.Marshal(ManifestFromCatalog(catalog))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(manifest), `"zdx"`) || strings.Contains(string(manifest), `"report"`) {
		t.Errorf("machine manifest names zdx or report: %s", manifest)
	}
	_, err = Executor{Catalog: catalog}.Execute(context.Background(), Request{
		Capability: CapabilityResourcesRead,
		Operation:  Operation("report"),
		Input:      &Input{Product: "zdx", Resource: "apps"},
	})
	if machineErr, ok := err.(*MachineError); !ok || machineErr.Kind != ErrorKindUnsupportedOperation {
		t.Errorf("Execute(report) error = %v, want unsupported operation", err)
	}
}
