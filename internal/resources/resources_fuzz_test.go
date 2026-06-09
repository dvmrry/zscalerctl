package resources_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dvmrry/zscalerctl/internal/redact"
	"github.com/dvmrry/zscalerctl/internal/resources"
)

func FuzzProjectRecordSubsetAndCanaryRedaction(f *testing.F) {
	for _, seed := range []struct {
		data   []byte
		prefix string
		suffix string
	}{
		{data: []byte("scalar"), prefix: "operator note", suffix: "rollout"},
		{data: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, prefix: "unicode \u2603", suffix: "ticket CHG-123456"},
		{data: []byte(`{"nested":"shape"}`), prefix: "commit 0123456789abcdef0123456789abcdef01234567", suffix: "done"},
	} {
		f.Add(seed.data, seed.prefix, seed.suffix)
	}

	const canary = "projection-fuzz-canary-value"
	f.Fuzz(func(t *testing.T, data []byte, prefix, suffix string) {
		if len(data)+len(prefix)+len(suffix) > 8192 {
			return
		}
		// The Go fuzzer harvests string constants from this test (including the
		// canary) and injects them into its inputs. The canary is meaningful only
		// in the secret-SHAPED occurrences this harness injects on purpose
		// ("psk="+canary); a bare canary that the fuzzer slips into prefix,
		// suffix, or data lands as an ordinary low-entropy field value, which the
		// redactor correctly does NOT redact (it must not mangle arbitrary
		// strings). Skip those inputs so the assertion stays a real leak test, not
		// a test of the fuzzer feeding itself its own constant.
		if strings.Contains(prefix, canary) || strings.Contains(suffix, canary) || bytes.Contains(data, []byte(canary)) {
			return
		}

		for _, spec := range resources.Catalog() {
			if err := spec.Validate(); err != nil {
				t.Fatalf("ResourceSpec.Validate(%s/%s) error = %v, want nil", spec.Product, spec.Name, err)
			}
			for _, mode := range []redact.Mode{redact.ModeStandard, redact.ModeShare, redact.ModeParanoid} {
				record := resources.NewSourceRecord(fuzzSourceRecord(spec, data, prefix, suffix, canary))
				got, _, err := resources.ProjectRecord(spec, mode, record)
				if err != nil {
					t.Fatalf("ProjectRecord(%s/%s, mode %s) error = %v, want nil", spec.Product, spec.Name, mode, err)
				}

				fields := got.Fields()
				if err := resources.AssertRenderedSubset(spec, mode, fields); err != nil {
					t.Fatalf("AssertRenderedSubset(%s/%s, mode %s, %#v) error = %v, want nil", spec.Product, spec.Name, mode, fields, err)
				}

				body, err := json.Marshal(fields)
				if err != nil {
					t.Fatalf("json.Marshal(ProjectRecord(%s/%s, mode %s)) error = %v, want nil", spec.Product, spec.Name, mode, err)
				}
				if strings.Contains(string(body), canary) {
					t.Fatalf("ProjectRecord(%s/%s, mode %s) JSON = %s, want no canary", spec.Product, spec.Name, mode, string(body))
				}
			}
		}
	})
}

func fuzzSourceRecord(spec resources.ResourceSpec, data []byte, prefix, suffix, canary string) map[string]any {
	record := map[string]any{
		"unknownScalar": "psk=" + canary,
		"unknownNested": map[string]any{
			"description": "psk=" + canary,
			"value":       fuzzValue(data, 0),
		},
		"unknownList": []any{
			"psk=" + canary,
			map[string]any{"token": canary},
		},
	}

	for i, field := range spec.Fields {
		record[field.JSONField()] = fuzzValue(rotateBytes(data, i), 0)
	}
	if len(spec.Fields) > 0 {
		field := spec.Fields[fuzzIndex(data, len(spec.Fields))]
		record[field.JSONField()] = prefix + " psk=" + canary + " " + suffix
	}
	return record
}

func fuzzValue(data []byte, depth int) any {
	if depth >= 3 || len(data) == 0 {
		return string(data)
	}

	switch data[0] % 7 {
	case 0:
		return string(data[1:])
	case 1:
		return []string{string(data[1:]), "ordinary-string"}
	case 2:
		return []any{string(data[1:]), fuzzValue(data[1:], depth+1)}
	case 3:
		return map[string]any{
			"nested": fuzzValue(data[1:], depth+1),
			"extra":  string(data),
		}
	case 4:
		return []map[string]any{
			{
				"nested": fuzzValue(data[1:], depth+1),
			},
		}
	case 5:
		return int(data[0])
	default:
		return data[0]%2 == 0
	}
}

func rotateBytes(data []byte, offset int) []byte {
	if len(data) == 0 {
		return data
	}
	offset %= len(data)
	out := make([]byte, 0, len(data))
	out = append(out, data[offset:]...)
	out = append(out, data[:offset]...)
	return out
}

func fuzzIndex(data []byte, length int) int {
	if length <= 0 || len(data) == 0 {
		return 0
	}
	return int(data[0]) % length
}
