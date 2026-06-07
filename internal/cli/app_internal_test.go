package cli

import (
	"testing"

	"github.com/dvmrry/zscalerctl/internal/resources"
)

func TestParseDumpResourcesSupportsCatalogDerivedQualifiedProducts(t *testing.T) {
	t.Parallel()

	product := resources.Product("ztw")
	catalog := resources.ResourceCatalog{
		{
			Product:    product,
			Name:       "workload-groups",
			Operations: resources.ReadOperations(),
		},
	}
	products := map[resources.Product]bool{product: true}

	selected, err := parseDumpResources("ztw/workload-groups", products, catalog)
	if err != nil {
		t.Fatalf("parseDumpResources(ztw/workload-groups) error = %v, want nil", err)
	}
	want := dumpResourceKey{product: product, name: "workload-groups"}
	if len(selected) != 1 || !selected[want] {
		t.Fatalf("parseDumpResources(ztw/workload-groups) = %#v, want only %#v", selected, want)
	}
}
