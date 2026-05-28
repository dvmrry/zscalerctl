package output

import (
	"fmt"
	"strings"
)

type KV struct {
	Key   string
	Value string
	Kind  string
}

func RenderKeyValues(rows []KV, style Style) SafeText {
	width := 0
	for _, row := range rows {
		if len(row.Key) > width {
			width = len(row.Key)
		}
	}

	var body strings.Builder
	for _, row := range rows {
		key := fmt.Sprintf("%-*s", width, row.Key)
		value := row.Value
		if row.Kind == "" {
			row.Kind = row.Key
		}
		fmt.Fprintf(&body, "%s  %s\n", style.Key(key), style.Value(row.Kind, value))
	}
	return NewSafeText(body.String())
}
