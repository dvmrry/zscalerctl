package runtime

import (
	"errors"
	"reflect"
	"testing"

	"zscalerctl/internal/config"
	"zscalerctl/internal/redact"
	"zscalerctl/internal/resources"
)

func TestNewEngineDefensivelyCopiesHostOptions(t *testing.T) {
	t.Parallel()

	env := []string{"RUNTIME_HOST_OPTION=original"}
	catalog := resources.ResourceCatalog{{
		Product:    resources.ProductZIA,
		Name:       "locations",
		Operations: resources.ListOperations(),
		Fields: []resources.FieldSpec{{
			Name:           "id",
			Classification: resources.ClassOperational,
			AllowedModes:   []redact.Mode{redact.ModeStandard},
		}},
	}}
	loadedConfig, err := config.LoadConfig(nil, config.LoadOptions{})
	if err != nil {
		t.Fatalf("config.LoadConfig(engine fixture) error = %v, want nil", err)
	}
	var loaderInputs [][]string
	engine, err := NewEngine(Options{
		Env:     env,
		Catalog: catalog,
		loadConfig: func(got []string, _ config.LoadOptions) (config.Config, error) {
			loaderInputs = append(loaderInputs, append([]string(nil), got...))
			got[0] = "loader-mutated"
			return loadedConfig, nil
		},
	})
	if err != nil {
		t.Fatalf("NewEngine() error = %v, want nil", err)
	}
	env[0] = "caller-mutated"
	catalog[0].Name = "caller-mutated"
	catalog[0].Operations[0].Name = "caller-mutated"
	catalog[0].Fields[0].AllowedModes[0] = redact.ModeParanoid

	for range 2 {
		opts := engine.options()
		if len(opts.Catalog) != 1 || opts.Catalog[0].Name != "locations" ||
			opts.Catalog[0].Operations[0].Name != "list" ||
			opts.Catalog[0].Fields[0].AllowedModes[0] != redact.ModeStandard {
			t.Fatalf("Engine.options().Catalog = %#v, want original options snapshot", opts.Catalog)
		}
		if _, err := opts.loadConfig(opts.Env, config.LoadOptions{}); err != nil {
			t.Fatalf("loadConfig() error = %v, want nil", err)
		}
		opts.Catalog[0].Name = "options-mutated"
	}
	wantInputs := [][]string{
		{"RUNTIME_HOST_OPTION=original"},
		{"RUNTIME_HOST_OPTION=original"},
	}
	if !reflect.DeepEqual(loaderInputs, wantInputs) {
		t.Fatalf("Engine loader inputs = %#v, want fresh copies %#v", loaderInputs, wantInputs)
	}
}

func TestNewEngineRejectsMutatingCatalog(t *testing.T) {
	t.Parallel()

	engine, err := NewEngine(Options{Catalog: resources.ResourceCatalog{{
		Product: resources.ProductZIA,
		Name:    "mutating",
		Operations: []resources.Operation{{
			Name: "update", Capability: resources.CapabilityWrite,
		}},
	}}})
	if engine != nil {
		t.Fatalf("NewEngine(mutating catalog) = %#v, want nil", engine)
	}
	if !errors.Is(err, resources.ErrMutatingOperation) {
		t.Fatalf("NewEngine(mutating catalog) error = %v, want ErrMutatingOperation", err)
	}
}
