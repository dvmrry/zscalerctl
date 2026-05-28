package output_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dvmrry/zscalerctl/internal/output"
	"github.com/dvmrry/zscalerctl/internal/redact"
	"github.com/dvmrry/zscalerctl/internal/secret"
)

func TestRendererWriteJSONUsesSecretSafeMarshalAndBackstopRedaction(t *testing.T) {
	t.Parallel()

	const raw = "client-secret-from-response"
	var buf bytes.Buffer
	renderer := output.NewRenderer(redact.New(redact.ModeStandard))
	err := renderer.WriteJSON(&buf, safeJSONFixture{
		AuthHeader: "Authorization: Bearer raw-bearer-token",
		Secret:     secret.New(raw),
	})
	if err != nil {
		t.Fatalf("Renderer.WriteJSON() error = %v, want nil", err)
	}
	got := buf.String()
	for _, forbidden := range []string{raw, "raw-bearer-token"} {
		if strings.Contains(got, forbidden) {
			t.Errorf("Renderer.WriteJSON() = %q, want no %q", got, forbidden)
		}
	}
}

func TestRendererWriteJSONAppliesShareMode(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	renderer := output.NewRenderer(redact.New(redact.ModeShare))
	err := renderer.WriteJSON(&buf, safeIdentifierFixture{
		Email: "alice@example.com",
		IP:    "192.0.2.10",
	})
	if err != nil {
		t.Fatalf("Renderer.WriteJSON() error = %v, want nil", err)
	}
	got := buf.String()
	for _, forbidden := range []string{"alice@example.com", "192.0.2.10"} {
		if strings.Contains(got, forbidden) {
			t.Errorf("Renderer.WriteJSON() = %q, want no %q", got, forbidden)
		}
	}
}

func TestRendererWriteTextRedactsText(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	renderer := output.NewRenderer(redact.New(redact.ModeStandard))
	err := renderer.WriteText(&buf, output.NewSafeText("Authorization: Basic dXNlcjpzZWNyZXQ=\n"))
	if err != nil {
		t.Fatalf("Renderer.WriteText() error = %v, want nil", err)
	}
	if strings.Contains(buf.String(), "dXNlcjpzZWNyZXQ=") {
		t.Errorf("Renderer.WriteText() = %q, want no basic auth payload", buf.String())
	}
}

type safeJSONFixture struct {
	AuthHeader string        `json:"auth_header"`
	Secret     secret.Secret `json:"secret"`
}

func (safeJSONFixture) OutputSafe() {}

type safeIdentifierFixture struct {
	Email string `json:"email"`
	IP    string `json:"ip"`
}

func (safeIdentifierFixture) OutputSafe() {}
