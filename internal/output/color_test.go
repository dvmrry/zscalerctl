package output_test

import (
	"strings"
	"testing"

	"github.com/dvmrry/zscalerctl/internal/output"
)

func TestShouldColorRespectsPolicy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mode      output.ColorMode
		env       []string
		stdoutTTY bool
		want      bool
	}{
		{name: "always without tty", mode: output.ColorAlways, want: true},
		{name: "never with tty", mode: output.ColorNever, stdoutTTY: true, want: false},
		{name: "auto with tty", mode: output.ColorAuto, stdoutTTY: true, want: true},
		{name: "auto without tty", mode: output.ColorAuto, stdoutTTY: false, want: false},
		{name: "auto with no color", mode: output.ColorAuto, env: []string{"NO_COLOR=1"}, stdoutTTY: true, want: false},
		{name: "auto with dumb term", mode: output.ColorAuto, env: []string{"TERM=dumb"}, stdoutTTY: true, want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := output.ShouldColor(tt.mode, tt.env, tt.stdoutTTY)
			if got != tt.want {
				t.Errorf("ShouldColor(%q, %v, %t) = %t, want %t", tt.mode, tt.env, tt.stdoutTTY, got, tt.want)
			}
		})
	}
}

func TestSupports256Color(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  []string
		want bool
	}{
		{name: "xterm 256", env: []string{"TERM=xterm-256color"}, want: true},
		{name: "truecolor", env: []string{"COLORTERM=truecolor"}, want: true},
		{name: "plain", env: []string{"TERM=xterm"}, want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := output.Supports256Color(tt.env)
			if got != tt.want {
				t.Errorf("Supports256Color(%v) = %t, want %t", tt.env, got, tt.want)
			}
		})
	}
}

func TestRenderKeyValuesUses256ColorWhenEnabled(t *testing.T) {
	t.Parallel()

	got := output.RenderKeyValues([]output.KV{
		{Key: "Status", Value: "OK", Kind: "ok"},
	}, output.NewStyle(true, true))
	if !strings.Contains(got.String(), "\x1b[38;5;") {
		t.Errorf("RenderKeyValues(..., 256 color) = %q, want 256-color escape", got)
	}
}
