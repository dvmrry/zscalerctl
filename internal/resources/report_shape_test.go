package resources

import (
	"errors"
	"testing"
)

// TestReportShapeAndOperationGoTogetherAlone: a report spec has exactly the
// report operation and only a report spec has it, so dump, diff and refs,
// which select by list/show, can never pick a report up.
func TestReportShapeAndOperationGoTogetherAlone(t *testing.T) {
	t.Parallel()

	fields := []FieldSpec{operationalField("id", allModes())}
	report := Operation{Name: "report", Capability: CapabilityRead}
	list := Operation{Name: "list", Capability: CapabilityRead}
	for _, c := range []struct {
		name  string
		shape ResourceShape
		ops   []Operation
		valid bool
	}{
		{"report alone", ShapeReport, []Operation{report}, true},
		{"report shape with list too", ShapeReport, []Operation{report, list}, false},
		{"report shape with list only", ShapeReport, []Operation{list}, false},
		{"list shape with report", ShapeList, []Operation{list, report}, false},
		{"default shape with report", "", []Operation{report}, false},
	} {
		err := ResourceSpec{Product: ProductZDX, Name: "probe", Shape: c.shape, Operations: c.ops, Fields: fields}.Validate()
		if got := err == nil; got != c.valid {
			t.Errorf("%s: Validate() error = %v, want valid %v", c.name, err, c.valid)
		}
		if err != nil && !errors.Is(err, ErrInvalidResourceSpec) {
			t.Errorf("%s: Validate() error = %v, want ErrInvalidResourceSpec", c.name, err)
		}
	}
}
