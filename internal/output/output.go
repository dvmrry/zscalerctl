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
	FormatTable  Format = "table"
	FormatJSON   Format = "json"
	FormatYAML   Format = "yaml"
	FormatNDJSON Format = "ndjson"
)

func ParseFormat(value string) (Format, error) {
	switch Format(strings.ToLower(strings.TrimSpace(value))) {
	case FormatTable:
		return FormatTable, nil
	case FormatJSON:
		return FormatJSON, nil
	case FormatYAML:
		return FormatYAML, nil
	case FormatNDJSON:
		return FormatNDJSON, nil
	default:
		return "", fmt.Errorf("unsupported output format %q", value)
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

func (r Renderer) WriteText(w io.Writer, value SafeText) error {
	if _, err := w.Write(r.Redactor.Bytes([]byte(value.String()))); err != nil {
		return fmt.Errorf("write text: %w", err)
	}
	return nil
}
