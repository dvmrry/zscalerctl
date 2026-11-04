package zscaler

import (
	"context"
	"reflect"
	"testing"

	"zscalerctl/internal/resources"
)

// TestZPASegmentKindListsKeepOnlyTheirSegments: browser access, inspection
// and PRA segments are listed from the one /application route, and each list
// keeps only the segments whose own sub-list is populated.
func TestZPASegmentKindListsKeepOnlyTheirSegments(t *testing.T) {
	t.Parallel()

	routes := map[string]string{
		zpaIdentityMgmtV1 + "/application": `{"totalPages":"1","list":[` +
			`{"id":"1","name":"plain-segment"},` +
			`{"id":"2","name":"browser-segment","clientlessApps":[{"id":"21","name":"clientless-app"}]},` +
			`{"id":"3","name":"inspection-segment","inspectionApps":[{"id":"31","name":"inspection-app"}]},` +
			`{"id":"4","name":"pra-segment","praApps":[{"id":"41","name":"pra-app"}]},` +
			`{"id":"5","name":"empty-lists-segment","clientlessApps":[],"inspectionApps":[],"praApps":[]}]}`,
	}
	handlers := newZPAIdentityTestHandlers(t, routes, nil)

	for _, c := range []struct {
		resource string
		want     []string
	}{
		{resourceZPABrowserAccess, []string{"browser-segment"}},
		{resourceZPAInspectionAppSegments, []string{"inspection-segment"}},
		{resourceZPAPRAAppSegments, []string{"pra-segment"}},
	} {
		records, err := handlers[resourceKey{product: resources.ProductZPA, name: c.resource}].List(context.Background())
		if err != nil {
			t.Fatalf("zpa/%s List() error = %v", c.resource, err)
		}
		var names []string
		for _, record := range records {
			name, _ := record.ScalarString("name")
			names = append(names, name)
		}
		if !reflect.DeepEqual(names, c.want) {
			t.Errorf("zpa/%s names = %v, want %v", c.resource, names, c.want)
		}
	}
}
