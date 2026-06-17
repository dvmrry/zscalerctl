package diff

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/dvmrry/zscalerctl/internal/dump"
	"github.com/dvmrry/zscalerctl/internal/redact"
	"github.com/dvmrry/zscalerctl/internal/resources"
)

const SchemaID = "zscalerctl.diff.v1"

const (
	maxManifestBytes int64 = 1 << 20
	maxResourceBytes int64 = 512 << 20
)

var (
	ErrInvalidDump       = errors.New("invalid dump")
	ErrPartialDumpInput  = errors.New("partial dump input")
	ErrRedactionMismatch = errors.New("redaction mode mismatch")
)

type Options struct {
	Catalog           resources.ResourceCatalog
	Products          map[resources.Product]bool
	Resources         map[ResourceKey]bool
	IgnoreOperational bool
	AllowPartial      bool
}

type ResourceKey struct {
	Product resources.Product
	Name    string
}

type Report struct {
	Schema    string         `json:"schema"`
	Old       DumpRef        `json:"old"`
	New       DumpRef        `json:"new"`
	Summary   Summary        `json:"summary"`
	Resources []ResourceDiff `json:"resources"`
}

func (Report) OutputSafe() {}

func (r Report) HasDrift() bool {
	return r.Summary.RecordsAdded > 0 || r.Summary.RecordsRemoved > 0 || r.Summary.RecordsChanged > 0
}

type DumpRef struct {
	Path           string `json:"path"`
	ManifestSchema string `json:"manifest_schema"`
	Redaction      string `json:"redaction"`
	Status         string `json:"status"`
	Partial        bool   `json:"partial"`
}

type Summary struct {
	ResourcesCompared  int `json:"resources_compared"`
	ResourcesWithDrift int `json:"resources_with_drift"`
	RecordsAdded       int `json:"records_added"`
	RecordsRemoved     int `json:"records_removed"`
	RecordsChanged     int `json:"records_changed"`
}

type ResourceDiff struct {
	Product  string         `json:"product"`
	Resource string         `json:"resource"`
	Identity Identity       `json:"identity"`
	Added    []RecordRef    `json:"added"`
	Removed  []RecordRef    `json:"removed"`
	Changed  []RecordChange `json:"changed"`
	Note     string         `json:"note,omitempty"`
}

func (r ResourceDiff) HasDrift() bool {
	return len(r.Added) > 0 || len(r.Removed) > 0 || len(r.Changed) > 0
}

type Identity struct {
	Mode  string `json:"mode"`
	Field string `json:"field,omitempty"`
}

type RecordRef struct {
	Key    string         `json:"key,omitempty"`
	Hash   string         `json:"hash,omitempty"`
	Record map[string]any `json:"record,omitempty"`
}

type RecordChange struct {
	Key     string        `json:"key"`
	Changes []FieldChange `json:"changes"`
}

type FieldChange struct {
	Field string `json:"field"`
	Old   any    `json:"old"`
	New   any    `json:"new"`
}

type loadedDump struct {
	ref       DumpRef
	manifest  dump.Manifest
	resources map[ResourceKey]loadedResource
}

type loadedResource struct {
	manifest dump.ManifestResource
	records  []map[string]any
}

func Compare(oldDir, newDir string, opts Options) (Report, error) {
	catalog := opts.Catalog
	if len(catalog) == 0 {
		catalog = resources.Catalog()
	}
	oldDump, err := loadDump(oldDir, catalog)
	if err != nil {
		return Report{}, err
	}
	newDump, err := loadDump(newDir, catalog)
	if err != nil {
		return Report{}, err
	}
	if oldDump.manifest.Redaction != newDump.manifest.Redaction {
		return Report{}, fmt.Errorf("%w: old=%s new=%s", ErrRedactionMismatch, oldDump.manifest.Redaction, newDump.manifest.Redaction)
	}
	if !opts.AllowPartial {
		if oldDump.ref.Partial {
			return Report{}, fmt.Errorf("%w: old dump %s is partial", ErrPartialDumpInput, oldDir)
		}
		if newDump.ref.Partial {
			return Report{}, fmt.Errorf("%w: new dump %s is partial", ErrPartialDumpInput, newDir)
		}
	}

	report := Report{
		Schema: SchemaID,
		Old:    oldDump.ref,
		New:    newDump.ref,
	}
	for _, spec := range selectedSpecs(catalog, opts) {
		key := ResourceKey{Product: spec.Product, Name: spec.Name}
		oldRes := oldDump.resources[key]
		newRes := newDump.resources[key]
		if len(oldRes.records) == 0 && len(newRes.records) == 0 {
			continue
		}
		resourceDiff, err := compareResource(spec, oldRes.records, newRes.records, opts.IgnoreOperational)
		if err != nil {
			return Report{}, err
		}
		report.Summary.ResourcesCompared++
		if resourceDiff.HasDrift() {
			report.Summary.ResourcesWithDrift++
		}
		report.Summary.RecordsAdded += len(resourceDiff.Added)
		report.Summary.RecordsRemoved += len(resourceDiff.Removed)
		report.Summary.RecordsChanged += len(resourceDiff.Changed)
		report.Resources = append(report.Resources, resourceDiff)
	}
	return report, nil
}

func selectedSpecs(catalog resources.ResourceCatalog, opts Options) []resources.ResourceSpec {
	specs := make([]resources.ResourceSpec, 0, len(catalog))
	for _, spec := range catalog {
		if len(opts.Products) > 0 && !opts.Products[spec.Product] {
			continue
		}
		if len(opts.Resources) > 0 && !opts.Resources[ResourceKey{Product: spec.Product, Name: spec.Name}] {
			continue
		}
		if !spec.SupportsReadOperation("list") && !spec.SupportsReadOperation("show") {
			continue
		}
		specs = append(specs, spec)
	}
	sort.Slice(specs, func(i, j int) bool {
		if specs[i].Product != specs[j].Product {
			return specs[i].Product < specs[j].Product
		}
		return specs[i].Name < specs[j].Name
	})
	return specs
}

func loadDump(dir string, catalog resources.ResourceCatalog) (loadedDump, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return loadedDump{}, fmt.Errorf("%w: inspect %s: %v", ErrInvalidDump, dir, err)
	}
	if !info.IsDir() {
		return loadedDump{}, fmt.Errorf("%w: %s is not a directory", ErrInvalidDump, dir)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return loadedDump{}, fmt.Errorf("%w: open %s: %v", ErrInvalidDump, dir, err)
	}
	defer root.Close()

	body, err := readRootFile(root, "manifest.json", fmt.Sprintf("manifest for %s", dir), maxManifestBytes)
	if err != nil {
		return loadedDump{}, err
	}
	var manifest dump.Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return loadedDump{}, fmt.Errorf("%w: parse manifest for %s: %v", ErrInvalidDump, dir, err)
	}
	if manifest.Schema != dump.ManifestSchemaID {
		return loadedDump{}, fmt.Errorf("%w: unsupported manifest schema %q", ErrInvalidDump, manifest.Schema)
	}
	if _, err := redact.ParseMode(manifest.Redaction); err != nil {
		return loadedDump{}, fmt.Errorf("%w: invalid redaction mode %q", ErrInvalidDump, manifest.Redaction)
	}
	switch manifest.Status {
	case "complete", "partial":
	default:
		return loadedDump{}, fmt.Errorf("%w: invalid manifest status %q", ErrInvalidDump, manifest.Status)
	}
	loaded := loadedDump{
		ref: DumpRef{
			Path:           dir,
			ManifestSchema: manifest.Schema,
			Redaction:      manifest.Redaction,
			Status:         manifest.Status,
			Partial:        manifest.Status == "partial",
		},
		manifest:  manifest,
		resources: make(map[ResourceKey]loadedResource),
	}
	for _, mr := range manifest.Resources {
		key := ResourceKey{Product: resources.Product(mr.Product), Name: mr.Name}
		_, ok := catalog.FindSpec(key.Product, key.Name)
		if !ok {
			return loadedDump{}, fmt.Errorf("%w: manifest references unknown resource %s/%s", ErrInvalidDump, mr.Product, mr.Name)
		}
		switch mr.Status {
		case "ok":
			if strings.TrimSpace(mr.Path) == "" {
				return loadedDump{}, fmt.Errorf("%w: resource %s/%s has no path", ErrInvalidDump, mr.Product, mr.Name)
			}
			records, err := readResource(root, mr)
			if err != nil {
				return loadedDump{}, err
			}
			if len(records) != mr.Records {
				return loadedDump{}, fmt.Errorf("%w: resource %s/%s manifest records=%d file records=%d", ErrInvalidDump, mr.Product, mr.Name, mr.Records, len(records))
			}
			loaded.resources[key] = loadedResource{manifest: mr, records: records}
		case "error":
			loaded.ref.Partial = true
		default:
			return loadedDump{}, fmt.Errorf("%w: resource %s/%s has invalid status %q", ErrInvalidDump, mr.Product, mr.Name, mr.Status)
		}
	}
	return loaded, nil
}

func readResource(root *os.Root, mr dump.ManifestResource) ([]map[string]any, error) {
	path := filepath.Clean(filepath.FromSlash(mr.Path))
	if !filepath.IsLocal(path) {
		return nil, fmt.Errorf("%w: resource %s/%s has unsafe path %q", ErrInvalidDump, mr.Product, mr.Name, mr.Path)
	}
	body, err := readRootFile(root, path, fmt.Sprintf("resource %s/%s", mr.Product, mr.Name), maxResourceBytes)
	if err != nil {
		return nil, err
	}
	var raw any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("%w: parse resource %s/%s: %v", ErrInvalidDump, mr.Product, mr.Name, err)
	}
	switch value := raw.(type) {
	case []any:
		records := make([]map[string]any, 0, len(value))
		for i, item := range value {
			record, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("%w: resource %s/%s record %d is not an object", ErrInvalidDump, mr.Product, mr.Name, i)
			}
			records = append(records, record)
		}
		return records, nil
	case map[string]any:
		return []map[string]any{value}, nil
	default:
		return nil, fmt.Errorf("%w: resource %s/%s payload is not an object or array", ErrInvalidDump, mr.Product, mr.Name)
	}
}

func readRootFile(root *os.Root, name, label string, maxBytes int64) ([]byte, error) {
	file, err := root.Open(name)
	if err != nil {
		return nil, fmt.Errorf("%w: read %s: %v", ErrInvalidDump, label, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("%w: inspect %s: %v", ErrInvalidDump, label, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: %s is not a regular file", ErrInvalidDump, label)
	}
	if info.Size() > maxBytes {
		return nil, fmt.Errorf("%w: %s is too large", ErrInvalidDump, label)
	}
	body, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%w: read %s: %v", ErrInvalidDump, label, err)
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("%w: %s is too large", ErrInvalidDump, label)
	}
	return body, nil
}

func compareResource(spec resources.ResourceSpec, oldRecords, newRecords []map[string]any, ignoreOperational bool) (ResourceDiff, error) {
	out := ResourceDiff{
		Product:  string(spec.Product),
		Resource: spec.Name,
		Added:    []RecordRef{},
		Removed:  []RecordRef{},
		Changed:  []RecordChange{},
	}
	switch {
	case spec.EffectiveShape() == resources.ShapeSingleton:
		out.Identity = Identity{Mode: "singleton"}
		return compareSingleton(spec, oldRecords, newRecords, ignoreOperational, out)
	case spec.EffectiveGetKey() != "":
		out.Identity = Identity{Mode: "get_key", Field: spec.EffectiveGetKey()}
		return compareKeyed(spec, oldRecords, newRecords, ignoreOperational, out)
	default:
		out.Identity = Identity{Mode: "content_hash"}
		out.Note = "identity unavailable; compared by canonical content hash"
		return compareContentHash(spec, oldRecords, newRecords, out)
	}
}

func compareSingleton(spec resources.ResourceSpec, oldRecords, newRecords []map[string]any, ignoreOperational bool, out ResourceDiff) (ResourceDiff, error) {
	if len(oldRecords) > 1 || len(newRecords) > 1 {
		return out, fmt.Errorf("%w: singleton resource %s/%s has multiple records", ErrInvalidDump, spec.Product, spec.Name)
	}
	switch {
	case len(oldRecords) == 0 && len(newRecords) == 1:
		out.Added = append(out.Added, RecordRef{Key: "singleton", Record: filterRecord(spec, newRecords[0], ignoreOperational)})
	case len(oldRecords) == 1 && len(newRecords) == 0:
		out.Removed = append(out.Removed, RecordRef{Key: "singleton", Record: filterRecord(spec, oldRecords[0], ignoreOperational)})
	case len(oldRecords) == 1 && len(newRecords) == 1:
		changes := diffFields(filterRecord(spec, oldRecords[0], ignoreOperational), filterRecord(spec, newRecords[0], ignoreOperational))
		if len(changes) > 0 {
			out.Changed = append(out.Changed, RecordChange{Key: "singleton", Changes: changes})
		}
	}
	return out, nil
}

func compareKeyed(spec resources.ResourceSpec, oldRecords, newRecords []map[string]any, ignoreOperational bool, out ResourceDiff) (ResourceDiff, error) {
	keyField := spec.EffectiveGetKey()
	oldByKey, err := keyedRecords(spec, keyField, oldRecords, ignoreOperational)
	if err != nil {
		return out, err
	}
	newByKey, err := keyedRecords(spec, keyField, newRecords, ignoreOperational)
	if err != nil {
		return out, err
	}
	for _, key := range sortedKeys(oldByKey, newByKey) {
		oldRecord, oldOK := oldByKey[key]
		newRecord, newOK := newByKey[key]
		switch {
		case !oldOK:
			out.Added = append(out.Added, RecordRef{Key: key, Record: newRecord})
		case !newOK:
			out.Removed = append(out.Removed, RecordRef{Key: key, Record: oldRecord})
		default:
			changes := diffFields(oldRecord, newRecord)
			if len(changes) > 0 {
				out.Changed = append(out.Changed, RecordChange{Key: key, Changes: changes})
			}
		}
	}
	return out, nil
}

func keyedRecords(spec resources.ResourceSpec, keyField string, records []map[string]any, ignoreOperational bool) (map[string]map[string]any, error) {
	out := make(map[string]map[string]any, len(records))
	for i, record := range records {
		raw, ok := record[keyField]
		if !ok || raw == nil {
			return nil, fmt.Errorf("%w: resource %s/%s record %d missing identity field %q", ErrInvalidDump, spec.Product, spec.Name, i, keyField)
		}
		key := identityString(raw)
		if key == "" {
			return nil, fmt.Errorf("%w: resource %s/%s record %d has empty identity field %q", ErrInvalidDump, spec.Product, spec.Name, i, keyField)
		}
		if _, exists := out[key]; exists {
			return nil, fmt.Errorf("%w: resource %s/%s has duplicate identity %q", ErrInvalidDump, spec.Product, spec.Name, key)
		}
		out[key] = filterRecord(spec, record, ignoreOperational)
	}
	return out, nil
}

func identityString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case json.Number:
		return v.String()
	default:
		return fmt.Sprint(v)
	}
}

func compareContentHash(spec resources.ResourceSpec, oldRecords, newRecords []map[string]any, out ResourceDiff) (ResourceDiff, error) {
	oldByHash, err := hashedRecords(spec, oldRecords)
	if err != nil {
		return out, err
	}
	newByHash, err := hashedRecords(spec, newRecords)
	if err != nil {
		return out, err
	}
	for _, hash := range sortedHashKeys(oldByHash, newByHash) {
		oldBucket := oldByHash[hash]
		newBucket := newByHash[hash]
		switch {
		case len(oldBucket) > len(newBucket):
			for _, record := range oldBucket[len(newBucket):] {
				out.Removed = append(out.Removed, RecordRef{Hash: hash, Record: record})
			}
		case len(newBucket) > len(oldBucket):
			for _, record := range newBucket[len(oldBucket):] {
				out.Added = append(out.Added, RecordRef{Hash: hash, Record: record})
			}
		}
	}
	sortRecordRefs(out.Added)
	sortRecordRefs(out.Removed)
	return out, nil
}

func hashedRecords(spec resources.ResourceSpec, records []map[string]any) (map[string][]map[string]any, error) {
	out := make(map[string][]map[string]any, len(records))
	for _, record := range records {
		filtered := filterRecord(spec, record, true)
		hash, err := recordHash(filtered)
		if err != nil {
			return nil, err
		}
		out[hash] = append(out[hash], filtered)
	}
	return out, nil
}

func recordHash(record map[string]any) (string, error) {
	body, err := json.Marshal(canonicalValue(record))
	if err != nil {
		return "", fmt.Errorf("canonicalize record for content hash: %w", err)
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}

func filterRecord(spec resources.ResourceSpec, record map[string]any, ignoreOperational bool) map[string]any {
	out := make(map[string]any, len(record))
	classes := fieldClasses(spec)
	for key, value := range record {
		if ignoreOperational && classes[key] == resources.ClassOperational {
			continue
		}
		out[key] = canonicalValue(value)
	}
	return out
}

func fieldClasses(spec resources.ResourceSpec) map[string]resources.FieldClassification {
	out := make(map[string]resources.FieldClassification, len(spec.Fields))
	for _, field := range spec.Fields {
		out[field.JSONField()] = field.Classification
	}
	return out
}

func canonicalValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[key] = canonicalValue(item)
		}
		return out
	case []any:
		// Arrays are compared in order. Some fields encode precedence or ordered
		// membership, so reordering a list is intentionally reported as drift.
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = canonicalValue(item)
		}
		return out
	case float64:
		return normalizeFloat(v)
	default:
		return v
	}
}

func normalizeFloat(value float64) any {
	if value == math.Trunc(value) && value >= math.MinInt64 && value <= math.MaxInt64 {
		return int64(value)
	}
	return value
}

func diffFields(oldRecord, newRecord map[string]any) []FieldChange {
	var fields []string
	seen := map[string]bool{}
	for key := range oldRecord {
		seen[key] = true
		fields = append(fields, key)
	}
	for key := range newRecord {
		if !seen[key] {
			fields = append(fields, key)
		}
	}
	sort.Strings(fields)
	changes := make([]FieldChange, 0)
	for _, field := range fields {
		oldValue, oldOK := oldRecord[field]
		newValue, newOK := newRecord[field]
		if !oldOK {
			oldValue = nil
		}
		if !newOK {
			newValue = nil
		}
		if !reflect.DeepEqual(oldValue, newValue) {
			changes = append(changes, FieldChange{Field: field, Old: oldValue, New: newValue})
		}
	}
	return changes
}

func sortedKeys(a, b map[string]map[string]any) []string {
	seen := map[string]bool{}
	keys := make([]string, 0, len(a)+len(b))
	for key := range a {
		seen[key] = true
		keys = append(keys, key)
	}
	for key := range b {
		if !seen[key] {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func sortedHashKeys(a, b map[string][]map[string]any) []string {
	seen := map[string]bool{}
	keys := make([]string, 0, len(a)+len(b))
	for key := range a {
		seen[key] = true
		keys = append(keys, key)
	}
	for key := range b {
		if !seen[key] {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func sortRecordRefs(refs []RecordRef) {
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Key != refs[j].Key {
			return refs[i].Key < refs[j].Key
		}
		return refs[i].Hash < refs[j].Hash
	})
}
