package zscaler

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	zpamachinegroup "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/machinegroup"

	"zscalerctl/internal/redact"
	"zscalerctl/internal/resources"
)

type boolsEmbedded struct {
	EmbeddedFlag bool `json:"embeddedFlag,omitempty"`
}

type boolsChild struct {
	ChildFlag bool `json:"childFlag,omitempty"`
}

type boolsRecord struct {
	boolsEmbedded
	ID          string       `json:"id,omitempty"`
	Enabled     bool         `json:"enabled,omitempty"`
	Set         bool         `json:"set,omitempty"`
	Pointer     *bool        `json:"pointer,omitempty"`
	AsString    bool         `json:"asString,omitempty,string"`
	Hidden      bool         `json:"-"`
	Child       boolsChild   `json:"child"`
	ChildPtr    *boolsChild  `json:"childPtr,omitempty"`
	Children    []boolsChild `json:"children,omitempty"`
	NotOmitted  bool         `json:"notOmitted"`
	unexported  bool
	EmptyAbsent []boolsChild `json:"emptyAbsent,omitempty"`
}

// TestJSONSourceRecordKeepsFalseBooleans: a false the SDK's omitempty tag
// dropped comes back in nested structs and slices, while a nil pointer stays
// absent, ",string" and "-" fields are left alone, and embedded structs are
// not walked (json's shadowing rules decide their names).
func TestJSONSourceRecordKeepsFalseBooleans(t *testing.T) {
	t.Parallel()

	item := boolsRecord{
		ID:         "1",
		Set:        true,
		ChildPtr:   &boolsChild{},
		Children:   []boolsChild{{ChildFlag: true}, {}},
		unexported: true,
	}
	// The decode jsonSourceRecord makes, then the restore it applies.
	body, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	fields := map[string]any{}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&fields); err != nil {
		t.Fatal(err)
	}
	restoreOmittedFalse(reflect.ValueOf(item), fields)
	raw, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"child":{"childFlag":false},"childPtr":{"childFlag":false},"children":[{"childFlag":true},{"childFlag":false}],` +
		`"enabled":false,"id":"1","notOmitted":false,"set":true}`
	if string(raw) != want {
		t.Errorf("jsonSourceRecord() = %s\nwant %s", raw, want)
	}
}

// TestZPAMachineGroupDisabledRendersFalse: an existing ZPA resource whose SDK
// bool is omitempty renders a disabled group's false instead of no field.
func TestZPAMachineGroupDisabledRendersFalse(t *testing.T) {
	t.Parallel()

	spec, ok := resources.FindSpec(resources.ProductZPA, resourceZPAMachineGroups)
	if !ok {
		t.Fatal("FindSpec(zpa, machine-groups) ok = false")
	}
	record := jsonSourceRecord(zpamachinegroup.MachineGroup{ID: "1", Name: "group", Enabled: false})
	projected, _, err := resources.ProjectRecords(spec, redact.ModeParanoid, []resources.SourceRecord{record})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(projected.Records()[0].Fields())
	if got := string(raw); !strings.Contains(got, `"enabled":false`) {
		t.Errorf("machine-groups paranoid = %s, want \"enabled\":false", got)
	}
}
