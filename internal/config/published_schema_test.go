package config

// White-box drift guard: profileFile and profileData are unexported, so this
// test lives in package config (no _test suffix). It verifies that every
// YAML-tagged field on those structs appears in docs/schema/config.schema.json
// and vice versa, so the schema cannot silently drift from the Go structs.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/dvmrry/zscalerctl/internal/secretref"
)

// configSchemaNode reads docs/schema/config.schema.json and walks to the node
// at the given path, returning it as a map. Fatal on any failure.
func configSchemaNode(t *testing.T, path []string) map[string]any {
	t.Helper()
	// This file lives in internal/config/, so go up two levels to reach the repo root.
	body, err := os.ReadFile(filepath.Join("..", "..", "docs", "schema", "config.schema.json"))
	if err != nil {
		t.Fatalf("read config.schema.json: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatalf("parse config.schema.json: %v", err)
	}
	var cur any = doc
	for _, seg := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			t.Fatalf("config.schema.json: %q is not an object while walking %v", seg, path)
		}
		cur, ok = m[seg]
		if !ok {
			t.Fatalf("config.schema.json: missing %q while walking %v", seg, path)
		}
	}
	node, ok := cur.(map[string]any)
	if !ok {
		t.Fatalf("config.schema.json: node at %v is not an object", path)
	}
	return node
}

// configSchemaPropertyKeys returns the sorted property keys at the given schema path.
func configSchemaPropertyKeys(t *testing.T, path []string) []string {
	t.Helper()
	props := configSchemaNode(t, path)
	keys := make([]string, 0, len(props))
	for key := range props {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// configYAMLFieldNames returns the sorted yaml tag names for all exported
// (and accessible unexported-via-reflect) fields of the given type. It uses
// the "yaml" struct tag, not "json", because the config file is YAML.
func configYAMLFieldNames(typ reflect.Type) []string {
	var names []string
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("yaml")
		if tag == "" || tag == "-" {
			continue
		}
		// Strip any yaml options (e.g. ",omitempty")
		if idx := strings.IndexByte(tag, ','); idx >= 0 {
			tag = tag[:idx]
		}
		if tag != "" {
			names = append(names, tag)
		}
	}
	sort.Strings(names)
	return names
}

// configSetDiff returns elements in a that are not in b.
func configSetDiff(a, b []string) []string {
	set := make(map[string]struct{}, len(b))
	for _, v := range b {
		set[v] = struct{}{}
	}
	var out []string
	for _, v := range a {
		if _, ok := set[v]; !ok {
			out = append(out, v)
		}
	}
	return out
}

// TestPublishedConfigSchemaFieldTypes guards that the schema TYPE for each
// profileData field matches the Go field type, catching *bool→string drift.
func TestPublishedConfigSchemaFieldTypes(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeOf(profileData{})
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("yaml")
		if tag == "" || tag == "-" {
			continue
		}
		if idx := strings.IndexByte(tag, ','); idx >= 0 {
			tag = tag[:idx]
		}
		if tag == "" {
			continue
		}

		node := configSchemaNode(t, []string{"$defs", "profile", "properties", tag})

		// Dereference pointer to get the base kind.
		ft := field.Type
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}

		t.Run(tag, func(t *testing.T) {
			t.Parallel()
			switch {
			case field.Type == reflect.TypeOf((*secretref.SecretRef)(nil)):
				// *secretref.SecretRef → schema must use $ref
				if _, ok := node["$ref"]; !ok {
					t.Errorf("schema property %q: Go type is *secretref.SecretRef but schema node has no \"$ref\" key (got %v)", tag, node)
				}
			case ft.Kind() == reflect.Bool:
				if got, _ := node["type"].(string); got != "boolean" {
					t.Errorf("schema property %q: Go kind is bool but schema type = %q, want \"boolean\"", tag, got)
				}
			case ft.Kind() == reflect.String:
				if got, _ := node["type"].(string); got != "string" {
					t.Errorf("schema property %q: Go kind is string but schema type = %q, want \"string\"", tag, got)
				}
			}
		})
	}
}

// TestPublishedConfigSchemaMatchesStructs asserts that every YAML-tagged field
// on profileFile and profileData appears in config.schema.json, and vice versa.
// If this test fails, either update the schema or update the struct — they must
// stay in sync.
func TestPublishedConfigSchemaMatchesStructs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		typ       reflect.Type
		propsPath []string
	}{
		{
			name:      "profileFile",
			typ:       reflect.TypeOf(profileFile{}),
			propsPath: []string{"properties"},
		},
		{
			name:      "profileData",
			typ:       reflect.TypeOf(profileData{}),
			propsPath: []string{"$defs", "profile", "properties"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			structFields := configYAMLFieldNames(tc.typ)
			schemaFields := configSchemaPropertyKeys(t, tc.propsPath)
			if !reflect.DeepEqual(structFields, schemaFields) {
				t.Errorf("%s: struct YAML fields and config.schema.json properties differ\n struct: %v\n schema: %v\n missing-from-schema: %v\n extra-in-schema: %v",
					tc.name, structFields, schemaFields,
					configSetDiff(structFields, schemaFields),
					configSetDiff(schemaFields, structFields))
			}
		})
	}
}
