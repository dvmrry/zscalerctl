package zscaler

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"sort"
)

// pageFingerprint is a stable, content-based marker for one page of records.
// It deliberately fingerprints the complete page rather than individual
// records, so it does not assume that every resource has an identity field.
type pageFingerprint [sha256.Size]byte

func fingerprintPage[T any](records []T) (pageFingerprint, error) {
	payload, err := json.Marshal(records)
	if err != nil {
		return pageFingerprint{}, err
	}
	return pageFingerprint(sha256.Sum256(payload)), nil
}

// paginationRecordIdentity returns the JSON id of a record when it has one.
// Records without an id, or with a zero-value id (an absent field decoded into
// a plain int or string), rely on page fingerprints to detect repeated content.
func paginationRecordIdentity(record any) (string, bool, error) {
	payload, err := json.Marshal(record)
	if err != nil {
		return "", false, err
	}
	if len(payload) == 0 || payload[0] != '{' {
		return "", false, nil
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		return "", false, err
	}
	identity, ok := fields["id"]
	if !ok || string(identity) == "null" || string(identity) == "0" {
		return "", false, nil
	}
	var stringIdentity string
	if err := json.Unmarshal(identity, &stringIdentity); err == nil && stringIdentity == "" {
		return "", false, nil
	}
	return string(identity), true, nil
}

// pageSummary is what the paginator remembers about a page to recognize an
// endpoint that ignores the page parameter: its size and a fingerprint of its
// records as a set, each rendered with list elements sorted. A reordered page,
// or a record whose lists come back in another order, matches, but a changed
// value does not (networkServices was observed returning its port lists in a
// different order on every request). Matching needs no record ids, so a page
// with a record that has none still matches its reordered repeat.
type pageSummary struct {
	records  int
	contents pageFingerprint
}

func summarizePage[T any](records []T) (pageSummary, error) {
	canonical := make([]string, len(records))
	for i, record := range records {
		payload, err := canonicalRecord(record)
		if err != nil {
			return pageSummary{}, err
		}
		canonical[i] = string(payload)
	}
	sort.Strings(canonical)
	contents, err := fingerprintPage(canonical)
	if err != nil {
		return pageSummary{}, err
	}
	return pageSummary{records: len(records), contents: contents}, nil
}

// sameRecords reports whether two nonempty pages hold the same records in any
// order. A record whose fields changed between the two requests never matches,
// so a tenant change cannot pass for an endpoint that ignores paging.
func (s pageSummary) sameRecords(other pageSummary) bool {
	return s.records > 0 && s.records == other.records && s.contents == other.contents
}

// canonicalRecord renders a record as JSON with every list's elements sorted,
// so two responses that differ only in list order compare equal. It is used
// only to recognize repeated pages; returned records keep their order.
func canonicalRecord(record any) (json.RawMessage, error) {
	payload, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}
	// UseNumber keeps each number's exact literal; decoding into float64 would
	// make large integers that differ compare equal.
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	canonical, err := canonicalValue(value)
	if err != nil {
		return nil, err
	}
	return json.Marshal(canonical)
}

func canonicalValue(value any) (any, error) {
	switch v := value.(type) {
	case []any:
		elements := make([]string, len(v))
		for i, element := range v {
			canonical, err := canonicalValue(element)
			if err != nil {
				return nil, err
			}
			encoded, err := json.Marshal(canonical)
			if err != nil {
				return nil, err
			}
			elements[i] = string(encoded)
		}
		sort.Strings(elements)
		out := make([]json.RawMessage, len(elements))
		for i, element := range elements {
			out[i] = json.RawMessage(element)
		}
		return out, nil
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, element := range v {
			canonical, err := canonicalValue(element)
			if err != nil {
				return nil, err
			}
			out[key] = canonical
		}
		return out, nil
	}
	return value, nil
}
