package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/dvmrry/zscalerctl/internal/redact"
)

type Format string

const (
	// FormatAuto resolves at render time: a TTY destination gets pretty,
	// everything else (pipe, file) gets json. It is the default so interactive
	// use is readable while pipelines stay machine-parseable.
	FormatAuto  Format = "auto"
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	// FormatNDJSON is newline-delimited JSON: one compact record per line, for
	// streaming/pipeline consumers (jq -c, SIEM ingest). It only applies to record
	// output (resource list/get/show); other commands reject it like any
	// unsupported format.
	FormatNDJSON Format = "ndjson"
	FormatPretty Format = "pretty"
)

func ParseFormat(value string) (Format, error) {
	switch Format(strings.ToLower(strings.TrimSpace(value))) {
	case "", FormatAuto:
		return FormatAuto, nil
	case FormatTable:
		return FormatTable, nil
	case FormatJSON:
		return FormatJSON, nil
	case FormatNDJSON:
		return FormatNDJSON, nil
	case FormatPretty:
		return FormatPretty, nil
	default:
		return "", fmt.Errorf("unsupported output format %q; supported: auto, table, json, ndjson, pretty", value)
	}
}

type Renderer struct {
	Redactor redact.Redactor
}

type SafeJSON interface {
	OutputSafe()
}

// SafeText marks text assembled from static strings or projected data. It is
// not a sanitizer; WriteText still performs final redaction before output.
type SafeText string

func NewSafeText(value string) SafeText {
	return SafeText(value)
}

func (t SafeText) String() string {
	return string(t)
}

func NewRenderer(redactor redact.Redactor) Renderer {
	return Renderer{Redactor: redactor}
}

func (r Renderer) WriteJSON(w io.Writer, value SafeJSON) error {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	body = append(body, '\n')
	if _, err := w.Write(r.Redactor.Bytes(body)); err != nil {
		return fmt.Errorf("write json: %w", err)
	}
	return nil
}

// WriteNDJSON writes records as newline-delimited JSON: one compact, redacted
// JSON object per line, for streaming/pipeline consumers. Each record is marshaled
// WITHOUT indentation and followed by '\n'; an empty set writes nothing (the empty
// NDJSON stream). Redaction is applied per line — the same fail-closed pass
// WriteJSON uses — so NDJSON can never reveal more than the indented JSON for the
// same projected data.
func (r Renderer) WriteNDJSON(w io.Writer, records []SafeJSON) error {
	for _, record := range records {
		body, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("marshal ndjson: %w", err)
		}
		body = append(body, '\n')
		if _, err := w.Write(r.Redactor.Bytes(body)); err != nil {
			return fmt.Errorf("write ndjson: %w", err)
		}
	}
	return nil
}

func (r Renderer) WriteText(w io.Writer, value SafeText) error {
	if _, err := w.Write(r.Redactor.Bytes([]byte(value.String()))); err != nil {
		return fmt.Errorf("write text: %w", err)
	}
	return nil
}
