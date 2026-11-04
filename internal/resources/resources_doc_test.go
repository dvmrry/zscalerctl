package resources_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"zscalerctl/internal/redact"
	"zscalerctl/internal/resources"
)

// TestResourcesDocDocumentsEveryCatalogResource guards docs/RESOURCES.md against
// the drift that already bit the README: a resource added to the catalog but
// never written into the reference doc. It is a presence check (every enabled
// resource name must appear as a token in the doc); the field tables are
// checked by TestResourcesDocFieldsAreCurrent.
func TestResourcesDocDocumentsEveryCatalogResource(t *testing.T) {
	t.Parallel()

	body, err := os.ReadFile(filepath.Join("..", "..", "docs", "RESOURCES.md"))
	if err != nil {
		t.Fatalf("read docs/RESOURCES.md: %v", err)
	}
	content := string(body)

	for _, spec := range resources.Catalog() {
		if !docMentionsToken(content, spec.Name) {
			t.Errorf("docs/RESOURCES.md does not document resource %s/%s; add it to the resource reference", spec.Product, spec.Name)
		}
	}
}

// TestResourcesDocFieldsAreCurrent regenerates every field table in
// docs/RESOURCES.md from the catalog in memory and fails on any drift. The
// tables were hand-written once and drifted from the catalog (fields shown as
// secret long after they rendered), so the catalog is now their only source.
// Set RESOURCES_DOC_WRITE=1 (`make resources-doc`) to rewrite the file.
func TestResourcesDocFieldsAreCurrent(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "..", "docs", "RESOURCES.md")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	want, err := rewriteResourceDocFields(string(body), resources.Catalog())
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("RESOURCES_DOC_WRITE") == "1" {
		if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
		t.Logf("wrote %s", path)
		return
	}
	if string(body) != want {
		t.Errorf("%s field tables are out of date; run `make resources-doc`", path)
	}
}

func TestRewriteResourceDocFields(t *testing.T) {
	t.Parallel()

	catalog := resources.ResourceCatalog{{
		Product: resources.ProductZIA,
		Name:    "widgets",
		Fields: []resources.FieldSpec{
			{Name: "id", Classification: resources.ClassOperational, AllowedModes: []redact.Mode{redact.ModeStandard, redact.ModeShare, redact.ModeParanoid}, Type: "integer"},
			{
				Name: "rules", Classification: resources.ClassTenantConfig, AllowedModes: []redact.Mode{redact.ModeStandard, redact.ModeShare}, Type: "array<object>",
				Fields: []resources.FieldSpec{
					{Name: "id", Classification: resources.ClassOperational, AllowedModes: []redact.Mode{redact.ModeStandard, redact.ModeShare, redact.ModeParanoid}},
					{Name: "pac", JSONName: "customPac", Classification: resources.ClassSensitiveIdentifier, AllowedModes: []redact.Mode{redact.ModeStandard}},
				},
			},
			{
				Name: "owner", Classification: resources.ClassSensitiveIdentifier, AllowedModes: []redact.Mode{redact.ModeStandard},
				Fields: []resources.FieldSpec{
					{Name: "id", Classification: resources.ClassOperational, AllowedModes: []redact.Mode{redact.ModeShare, redact.ModeParanoid}},
				},
			},
			{Name: "psk", Classification: resources.ClassSecret, AllowedModes: []redact.Mode{}, SecretReason: resources.SecretCredential},
		},
	}}
	doc := "# Doc\n\n## ZIA Widgets\n\nintro\n\n" + resourceDocBegin("zia/widgets") + "\nstale | row\n" + resourceDocEnd + "\n\nafter\n"
	got, err := rewriteResourceDocFields(doc, catalog)
	if err != nil {
		t.Fatal(err)
	}
	want := "# Doc\n\n## ZIA Widgets\n\nintro\n\n" + resourceDocBegin("zia/widgets") + "\n" +
		"| Field | Type | Classification | Modes |\n" +
		"| --- | --- | --- | --- |\n" +
		"| `id` | `integer` | Operational metadata | `standard`, `share`, `paranoid` |\n" +
		"| `rules` | `array<object>` | Tenant configuration | `standard`, `share` |\n" +
		"| `rules.id` |  | Operational metadata | `standard`, `share` |\n" +
		"| `rules.customPac` |  | Sensitive identifier | `standard` |\n" +
		"| `owner` |  | Sensitive identifier | `standard` |\n" +
		"| `owner.id` |  | Operational metadata | never (outside its parent's modes) |\n" +
		"| `psk` |  | Secret | never (credential) |\n" +
		resourceDocEnd + "\n\nafter\n"
	if got != want {
		t.Fatalf("rewrite =\n%s\nwant\n%s", got, want)
	}

	for name, bad := range map[string]string{
		"missing block":        "# Doc\n",
		"duplicate block":      doc + doc,
		"unknown block":        doc + resourceDocBegin("zia/gadgets") + "\n" + resourceDocEnd + "\n",
		"unclosed block":       "## ZIA Widgets\n\n" + resourceDocBegin("zia/widgets") + "\n",
		"unclosed before next": "## ZIA Widgets\n\n" + resourceDocBegin("zia/widgets") + "\n\n## ZIA Gadgets\n\n" + doc,
		"stray end":            doc + resourceDocEnd + "\n",
		"wrong heading":        "## ZIA Gadgets\n\n" + resourceDocBegin("zia/widgets") + "\n" + resourceDocEnd + "\n",
	} {
		if _, err := rewriteResourceDocFields(bad, catalog); err == nil {
			t.Errorf("%s: rewrite accepted the doc", name)
		}
	}
}

const (
	resourceDocBeginPrefix = "<!-- BEGIN GENERATED FIELDS: "
	resourceDocBeginSuffix = " (make resources-doc) -->"
	resourceDocEnd         = "<!-- END GENERATED FIELDS -->"
)

func resourceDocBegin(key string) string {
	return resourceDocBeginPrefix + key + resourceDocBeginSuffix
}

// rewriteResourceDocFields replaces the body of every generated-fields block
// with the table its catalog resource renders. Every catalog resource needs
// exactly one block, under its own heading, and every block must name a
// catalog resource.
func rewriteResourceDocFields(doc string, catalog resources.ResourceCatalog) (string, error) {
	specs := map[string]resources.ResourceSpec{}
	for _, spec := range catalog {
		specs[string(spec.Product)+"/"+spec.Name] = spec
	}
	seen := map[string]bool{}
	var out strings.Builder
	rest := doc
	for {
		start := strings.Index(rest, resourceDocBeginPrefix)
		if start < 0 {
			if strings.Contains(rest, resourceDocEnd) {
				return "", fmt.Errorf("RESOURCES.md: %q outside a generated-fields block", resourceDocEnd)
			}
			out.WriteString(rest)
			break
		}
		if strings.Contains(rest[:start], resourceDocEnd) {
			return "", fmt.Errorf("RESOURCES.md: %q outside a generated-fields block", resourceDocEnd)
		}
		lineEnd := strings.Index(rest[start:], "\n")
		if lineEnd < 0 {
			return "", fmt.Errorf("RESOURCES.md: unterminated generated-fields marker")
		}
		marker := rest[start : start+lineEnd]
		key, ok := strings.CutPrefix(marker, resourceDocBeginPrefix)
		if ok {
			key, ok = strings.CutSuffix(key, resourceDocBeginSuffix)
		}
		if !ok {
			return "", fmt.Errorf("RESOURCES.md: malformed marker %q", marker)
		}
		spec, known := specs[key]
		if !known {
			return "", fmt.Errorf("RESOURCES.md: generated fields for %s, which is not a catalog resource", key)
		}
		if seen[key] {
			return "", fmt.Errorf("RESOURCES.md: %s has more than one generated-fields block", key)
		}
		seen[key] = true
		heading := "## " + strings.ToUpper(string(spec.Product)) + " " + titleResourceName(spec.Name)
		if lastResourceDocHeading(out.String()+rest[:start]) != heading {
			return "", fmt.Errorf("RESOURCES.md: generated fields for %s are not under %q", key, heading)
		}
		bodyStart := start + lineEnd + 1
		end := strings.Index(rest[bodyStart:], resourceDocEnd)
		if end < 0 || strings.Contains(rest[bodyStart:bodyStart+end], resourceDocBeginPrefix) {
			return "", fmt.Errorf("RESOURCES.md: generated fields for %s are not closed", key)
		}
		out.WriteString(rest[:bodyStart])
		out.WriteString(renderResourceDocFields(spec))
		out.WriteString(resourceDocEnd)
		rest = rest[bodyStart+end+len(resourceDocEnd):]
	}
	var missing []string
	for _, spec := range catalog {
		if key := string(spec.Product) + "/" + spec.Name; !seen[key] {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return "", fmt.Errorf("RESOURCES.md has no generated-fields block for %s", strings.Join(missing, ", "))
	}
	return out.String(), nil
}

// lastResourceDocHeading returns the last "## " heading line in doc.
func lastResourceDocHeading(doc string) string {
	i := strings.LastIndex("\n"+doc, "\n## ")
	if i < 0 {
		return ""
	}
	heading, _, _ := strings.Cut(doc[i:], "\n")
	return heading
}

func renderResourceDocFields(spec resources.ResourceSpec) string {
	var b strings.Builder
	b.WriteString("| Field | Type | Classification | Modes |\n| --- | --- | --- | --- |\n")
	writeResourceDocFieldRows(&b, spec, spec.Fields, "")
	return b.String()
}

func writeResourceDocFieldRows(b *strings.Builder, spec resources.ResourceSpec, fields []resources.FieldSpec, prefix string) {
	for _, field := range fields {
		path := prefix + field.JSONField()
		typ := ""
		if field.Type != "" {
			typ = "`" + field.Type + "`"
		}
		fmt.Fprintf(b, "| `%s` | %s | %s | %s |\n", path, typ, resourceDocClassification(field.Classification), resourceDocModes(spec, field, path))
		writeResourceDocFieldRows(b, spec, field.Fields, path+".")
	}
}

func resourceDocClassification(class resources.FieldClassification) string {
	switch class {
	case resources.ClassPublicProjectData:
		return "Public project data"
	case resources.ClassOperational:
		return "Operational metadata"
	case resources.ClassTenantConfig:
		return "Tenant configuration"
	case resources.ClassSensitiveIdentifier:
		return "Sensitive identifier"
	case resources.ClassFreeText:
		return "Free text"
	case resources.ClassSecret:
		return "Secret"
	default:
		return string(class)
	}
}

// resourceDocModes lists the modes path renders in: its own modes narrowed
// by every ancestor's, since projection drops a parent before its children.
func resourceDocModes(spec resources.ResourceSpec, field resources.FieldSpec, path string) string {
	availability, ok := resources.FieldPathAvailability(spec, path)
	if !ok {
		panic(fmt.Sprintf("%s/%s: %s does not resolve in the catalog", spec.Product, spec.Name, path))
	}
	if len(availability.RenderableIn) == 0 {
		switch {
		case field.SecretReason != "":
			return "never (" + strings.ReplaceAll(string(field.SecretReason), "_", " ") + ")"
		case availability.Secret:
			return "never (under a secret parent)"
		case len(field.AllowedModes) > 0:
			return "never (outside its parent's modes)"
		default:
			return "never"
		}
	}
	modes := make([]string, 0, len(availability.RenderableIn))
	for _, mode := range availability.RenderableIn {
		modes = append(modes, "`"+string(mode)+"`")
	}
	return strings.Join(modes, ", ")
}

// docMentionsToken reports whether token appears in content delimited by
// non-identifier characters on both sides, so a hyphenated resource name like
// "rule-labels" matches the whole name and not a substring of a longer token.
func docMentionsToken(content, token string) bool {
	isBoundary := func(b byte) bool {
		switch {
		case b >= 'A' && b <= 'Z', b >= 'a' && b <= 'z', b >= '0' && b <= '9', b == '_', b == '-':
			return false
		default:
			return true
		}
	}
	for offset := 0; ; {
		idx := strings.Index(content[offset:], token)
		if idx < 0 {
			return false
		}
		start := offset + idx
		end := start + len(token)
		beforeOK := start == 0 || isBoundary(content[start-1])
		afterOK := end == len(content) || isBoundary(content[end])
		if beforeOK && afterOK {
			return true
		}
		offset = start + 1
	}
}
