package diff

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestPublishedDiffSchemaMatchesStructs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		typ       reflect.Type
		propsPath []string
	}{
		{"Report", reflect.TypeOf(Report{}), []string{"properties"}},
		{"DumpRef", reflect.TypeOf(DumpRef{}), []string{"$defs", "dumpRef", "properties"}},
		{"Summary", reflect.TypeOf(Summary{}), []string{"$defs", "summary", "properties"}},
		{"ResourceDiff", reflect.TypeOf(ResourceDiff{}), []string{"$defs", "resourceDiff", "properties"}},
		{"Identity", reflect.TypeOf(Identity{}), []string{"$defs", "identity", "properties"}},
		{"RecordRef", reflect.TypeOf(RecordRef{}), []string{"$defs", "recordRef", "properties"}},
		{"RecordChange", reflect.TypeOf(RecordChange{}), []string{"$defs", "recordChange", "properties"}},
		{"FieldChange", reflect.TypeOf(FieldChange{}), []string{"$defs", "fieldChange", "properties"}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			structFields := diffJSONFieldNames(tc.typ)
			schemaFields := diffSchemaPropertyKeys(t, tc.propsPath)
			if !reflect.DeepEqual(structFields, schemaFields) {
				t.Errorf("%s: struct JSON fields and diff.schema.json properties differ\n struct: %v\n schema: %v\n missing-from-schema: %v\n extra-in-schema: %v",
					tc.name, structFields, schemaFields,
					diffSetDiff(structFields, schemaFields), diffSetDiff(schemaFields, structFields))
			}
		})
	}
}

func TestDiffSchemaIDMatchesPublishedConst(t *testing.T) {
	t.Parallel()

	node := diffSchemaNode(t, []string{"properties", "schema"})
	published, ok := node["const"].(string)
	if !ok {
		t.Fatalf("diff.schema.json: properties.schema.const = %v, want string", node["const"])
	}
	if SchemaID != published {
		t.Errorf("SchemaID = %q, want published schema const %q", SchemaID, published)
	}
}

func TestDiffIdentityModesMatchSchemaEnum(t *testing.T) {
	t.Parallel()

	node := diffSchemaNode(t, []string{"$defs", "identity", "properties", "mode"})
	raw, ok := node["enum"].([]any)
	if !ok {
		t.Fatalf("diff.schema.json identity.mode enum = %v, want array", node["enum"])
	}
	got := make([]string, 0, len(raw))
	for _, value := range raw {
		mode, ok := value.(string)
		if !ok {
			t.Fatalf("identity.mode enum value = %v, want string", value)
		}
		got = append(got, mode)
	}
	sort.Strings(got)
	want := []string{"content_hash", "get_key", "singleton"}
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("identity.mode enum = %v, want %v", got, want)
	}
}

func diffSchemaNode(t *testing.T, path []string) map[string]any {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "..", "docs", "schema", "diff.schema.json"))
	if err != nil {
		t.Fatalf("read diff.schema.json: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatalf("parse diff.schema.json: %v", err)
	}
	var cur any = doc
	for _, seg := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			t.Fatalf("diff.schema.json: %q is not an object while walking %v", seg, path)
		}
		cur, ok = m[seg]
		if !ok {
			t.Fatalf("diff.schema.json: missing %q while walking %v", seg, path)
		}
	}
	node, ok := cur.(map[string]any)
	if !ok {
		t.Fatalf("diff.schema.json: node at %v is not an object", path)
	}
	return node
}

func diffSchemaPropertyKeys(t *testing.T, path []string) []string {
	t.Helper()
	props := diffSchemaNode(t, path)
	keys := make([]string, 0, len(props))
	for key := range props {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func diffJSONFieldNames(typ reflect.Type) []string {
	var names []string
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" {
			continue
		}
		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
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

func diffSetDiff(a, b []string) []string {
	set := make(map[string]struct{}, len(b))
	for _, value := range b {
		set[value] = struct{}{}
	}
	var out []string
	for _, value := range a {
		if _, ok := set[value]; !ok {
			out = append(out, value)
		}
	}
	return out
}
