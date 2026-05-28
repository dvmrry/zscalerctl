package output

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type ColorMode string

const (
	ColorAuto   ColorMode = "auto"
	ColorAlways ColorMode = "always"
	ColorNever  ColorMode = "never"
)

func ParseColorMode(value string) (ColorMode, error) {
	switch ColorMode(strings.ToLower(strings.TrimSpace(value))) {
	case "", ColorAuto:
		return ColorAuto, nil
	case ColorAlways:
		return ColorAlways, nil
	case ColorNever:
		return ColorNever, nil
	default:
		return "", fmt.Errorf("unsupported color mode %q", value)
	}
}

func IsTerminal(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func ShouldColor(mode ColorMode, env []string, stdoutTTY bool) bool {
	switch mode {
	case ColorAlways:
		return true
	case ColorNever:
		return false
	default:
		if envValue(env, "NO_COLOR") != "" || envValue(env, "TERM") == "dumb" {
			return false
		}
		return stdoutTTY
	}
}

func Supports256Color(env []string) bool {
	term := strings.ToLower(envValue(env, "TERM"))
	colorTerm := strings.ToLower(envValue(env, "COLORTERM"))
	return strings.Contains(term, "256color") ||
		strings.Contains(colorTerm, "truecolor") ||
		strings.Contains(colorTerm, "24bit")
}

func envValue(env []string, key string) string {
	prefix := key + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			return strings.TrimPrefix(entry, prefix)
		}
	}
	return ""
}
