package runtime

import (
	"testing"

	"zscalerctl/internal/resources"
)

// TestEngineCatalogHasNoReportResources: the engine runtime built from the
// real catalog, which has ZDX report resources, holds none.
func TestEngineCatalogHasNoReportResources(t *testing.T) {
	t.Parallel()

	engine, err := NewEngine(Options{})
	if err != nil {
		t.Fatal(err)
	}
	catalog := engine.options().Catalog
	if len(catalog) == 0 {
		t.Fatal("engine catalog is empty")
	}
	for _, spec := range catalog {
		if spec.EffectiveShape() == resources.ShapeReport || spec.Product == resources.ProductZDX {
			t.Errorf("engine catalog lists %s/%s", spec.Product, spec.Name)
		}
	}
}
