package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/dvmrry/zscalerctl/internal/resources"
)

func TestParseDumpResourcesRejectsAmbiguousUnqualifiedName(t *testing.T) {
	products := map[resources.Product]bool{
		resources.ProductZIA: true,
		resources.ProductZPA: true,
	}
	catalog := resources.ResourceCatalog{
		dumpListSpec(resources.ProductZIA, "locations"),
		dumpListSpec(resources.ProductZPA, "locations"),
	}

	_, err := parseDumpResources("locations", products, catalog)
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("parseDumpResources(locations) error = %v, want ErrUsage", err)
	}
	if !strings.Contains(err.Error(), "ambiguous") || !strings.Contains(err.Error(), "product/name") {
		t.Errorf("parseDumpResources(locations) error = %q, want ambiguous product/name guidance", err.Error())
	}
}

func TestParseDumpResourcesRejectsMalformedResourceNames(t *testing.T) {
	products := map[resources.Product]bool{
		resources.ProductZIA: true,
		resources.ProductZPA: true,
	}
	catalog := resources.ResourceCatalog{
		dumpListSpec(resources.ProductZIA, "locations"),
	}
	tests := []string{
		"zia/locations/extra",
		"/locations",
		"zia/",
		"locations,",
		"locations,,rule-labels",
	}

	for _, value := range tests {
		value := value
		t.Run(value, func(t *testing.T) {
			_, err := parseDumpResources(value, products, catalog)
			if !errors.Is(err, ErrUsage) {
				t.Fatalf("parseDumpResources(%q) error = %v, want ErrUsage", value, err)
			}
		})
	}
}

func dumpListSpec(product resources.Product, name string) resources.ResourceSpec {
	return resources.ResourceSpec{
		Product: product,
		Name:    name,
		Operations: []resources.Operation{{
			Name:       "list",
			Capability: resources.CapabilityRead,
		}},
	}
}
