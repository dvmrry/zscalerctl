package zscaler

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	sdklogger "github.com/zscaler/zscaler-sdk-go/v3/logger"
)

// debugSlogLogger builds a debug-level text slog logger writing to buf, matching
// how the CLI wires --log-level debug to stderr.
func debugSlogLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestNewSDKLoggerNilReturnsNop(t *testing.T) {
	t.Parallel()

	got := newSDKLogger(nil)
	if got == nil {
		t.Fatal("newSDKLogger(nil) = nil, want non-nil nop logger")
	}
	// The nop logger must never panic and must produce no output.
	got.Printf("[INFO] got Retry-After from header:%s\n", "5")
	if _, ok := got.(sdkLogAdapter); ok {
		t.Fatalf("newSDKLogger(nil) returned %T, want the SDK nop logger", got)
	}
}

func TestSDKLogAdapterForwardsRetryAndAuthEvents(t *testing.T) {
	t.Parallel()

	// These mirror the SDK's real retry/backoff and session/token-renewal
	// format strings; each must be surfaced at debug.
	cases := []struct {
		name   string
		format string
		args   []interface{}
		want   string
	}{
		{"retry_after_header", "[INFO] got Retry-After from header:%s\n", []interface{}{"5"}, "Retry-After"},
		{"rate_limiter", "[DEBUG] Rate limiter triggered. Sleeping for %v", []interface{}{"2s"}, "Sleeping for 2s"},
		{"session_refresh", "[INFO] Session is invalid or expired. Refreshing session...", nil, "Refreshing session"},
		{"token_renewed", "[INFO] OAuth2 token successfully renewed", nil, "successfully renewed"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			adapter := newSDKLogger(debugSlogLogger(&buf))
			adapter.Printf(tc.format, tc.args...)
			out := buf.String()
			if !strings.Contains(out, tc.want) {
				t.Errorf("Printf(%q) logged %q, want it to contain %q", tc.format, out, tc.want)
			}
			if !strings.Contains(out, "source=zscaler-sdk") {
				t.Errorf("Printf(%q) logged %q, want it tagged source=zscaler-sdk", tc.format, out)
			}
			if !strings.Contains(out, "level=DEBUG") {
				t.Errorf("Printf(%q) logged %q, want it emitted at DEBUG level", tc.format, out)
			}
		})
	}
}

// TestSDKLogAdapterDropsRequestResponseDumps is the security guard: the SDK logs
// full request/response dumps (Authorization headers, bodies) through the same
// Printf interface, and the adapter must never forward them.
func TestSDKLogAdapterDropsRequestResponseDumps(t *testing.T) {
	t.Parallel()

	const secret = "Bearer super-secret-token"
	dumps := []string{
		`[DEBUG] Request "%s %s" details:
---[ ZSCALER SDK REQUEST | ID:%s ]-------------------------------
%s
---------------------------------------------------------`,
		`[DEBUG] Response "%s %s" details:
---[ ZSCALER SDK RESPONSE | ID:%s | Duration:%s ]--------------------------------
%s
-------------------------------------------------------`,
	}
	for _, format := range dumps {
		var buf bytes.Buffer
		adapter := newSDKLogger(debugSlogLogger(&buf))
		adapter.Printf(format, "GET", "https://x/api?token="+secret, "id", "Authorization: "+secret)
		if buf.Len() != 0 {
			t.Fatalf("Printf(dump) logged %q, want empty (dumps must never be forwarded)", buf.String())
		}
		if strings.Contains(buf.String(), secret) {
			t.Fatalf("Printf(dump) leaked secret %q", secret)
		}
	}
}

func TestSDKLogAdapterDropsUnknownMessages(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	adapter := newSDKLogger(debugSlogLogger(&buf))
	// A message that is neither a known retry/auth event nor a dump is dropped
	// (fail-closed allow-list).
	adapter.Printf("[DEBUG] Retrieved URL Filter and Cloud App Settings: %+v", map[string]string{"k": "v"})
	if buf.Len() != 0 {
		t.Fatalf("Printf(unknown) logged %q, want empty", buf.String())
	}
}

// TestSDKLogAdapterSilentBelowDebug confirms that at info/off levels no
// SDK-origin lines appear, because the adapter logs at DEBUG.
func TestSDKLogAdapterSilentBelowDebug(t *testing.T) {
	t.Parallel()

	levels := map[string]slog.Level{
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}
	for name, lvl := range levels {
		name, lvl := name, lvl
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: lvl}))
			adapter := newSDKLogger(logger)
			adapter.Printf("[INFO] got Retry-After from header:%s\n", "5")
			if buf.Len() != 0 {
				t.Errorf("at level %s Printf logged %q, want empty", name, buf.String())
			}
		})
	}
}

// Compile-time assertion that the adapter satisfies the SDK Logger interface.
var _ sdklogger.Logger = sdkLogAdapter{}
