package zscaler

import (
	"fmt"
	"log/slog"
	"strings"

	sdklogger "github.com/zscaler/zscaler-sdk-go/v3/logger"
)

// newSDKLogger returns the SDK Logger the reader installs on its SDK
// configurations. When logger is nil it returns the SDK nop logger, so the read
// paths stay completely silent unless an operator opts in with --log-level
// debug. When logger is non-nil, SDK retry/backoff (429 / Retry-After) and
// session/token-renewal activity is surfaced to that diag logger at debug
// level (stderr); stdout is never touched.
func newSDKLogger(logger *slog.Logger) sdklogger.Logger {
	if logger == nil {
		return sdklogger.NewNopLogger()
	}
	return sdkLogAdapter{logger: logger}
}

// sdkLogAdapter bridges the Zscaler SDK's Printf-style Logger interface to the
// CLI's structured diag logger.
//
// It is deliberately fail-closed about secrets. The SDK logs full
// request/response dumps — Authorization headers, request and response bodies —
// through this same Printf interface, so a naive pass-through would leak
// credentials at debug level. Instead the adapter forwards ONLY messages whose
// static SDK format string matches a retry/wait/auth-renewal allow-list and
// drops everything else. The allow-list is keyed on the SDK's compile-time
// format strings (never on interpolated values), so any future SDK log we do
// not recognize is dropped rather than risked.
type sdkLogAdapter struct {
	logger *slog.Logger
}

func (a sdkLogAdapter) Printf(format string, v ...interface{}) {
	if a.logger == nil || !sdkLogForwardable(format) {
		return
	}
	msg := strings.TrimSpace(fmt.Sprintf(format, v...))
	if msg == "" {
		return
	}
	a.logger.Debug(msg, slog.String("source", "zscaler-sdk"))
}

// sdkLogDenyMarkers identify SDK format strings that carry credential- or
// body-bearing content. They are denied unconditionally and take precedence
// over the allow-list, so such messages can never be forwarded even if the
// allow-list is later widened.
//
// The first three are the SDK's full request/response dumps (Authorization
// headers, bodies). The rest cover auth-FAILURE notices whose %v error embeds
// the raw auth-endpoint response body — e.g. oneapiclient.go:286
// ("...got http status: %d, response body: %s"), :359 ("auth error: %v"), and
// zia/v2_client.go:211-217 ("HTTP 401 Unauthorized: %s") propagating into
// oneapiconfig.go:118 ("Failed to renew OAuth2 token: %v") and
// zia/v2_client.go:297 ("Failed to refresh session: %v"). On an auth failure
// that body can carry token material, so it must never reach a log; the failure
// still surfaces to the operator through the normal redacted error path.
var sdkLogDenyMarkers = []string{
	"zscaler sdk request",
	"zscaler sdk response",
	"details:",
	"response body",
	"http status",
	"auth error",
	"failed to renew",
	"failed to refresh",
}

// sdkLogAllow lists case-insensitive substrings of the SDK's retry/backoff and
// session/token-renewal format strings. Only messages whose format matches one
// of these are forwarded; their interpolated values are retry durations,
// Retry-After header values, and renewal/session status — never credentials or
// response bodies. Deliberately excluded: the auth-FAILURE notices ("Failed to
// renew OAuth2 token", "Failed to refresh session"), whose %v error embeds the
// raw auth response body (see sdkLogDenyMarkers). Their safe counterparts —
// renewal success and session-refresh start/state — remain covered by the
// "token successfully renewed", "refreshing session", "session is invalid",
// "session invalidation", and "another goroutine is refreshing" entries, so no
// useful observability is lost.
var sdkLogAllow = []string{
	"retry-after",
	"rate limit",
	"rate limiter",
	"retrying",
	"sleeping for",
	"refreshing session",
	"session is invalid",
	"session invalidation",
	"another goroutine is refreshing",
	"token successfully renewed",
	"backoff",
	"waiting",
}

func sdkLogForwardable(format string) bool {
	lower := strings.ToLower(format)
	for _, marker := range sdkLogDenyMarkers {
		if strings.Contains(lower, marker) {
			return false
		}
	}
	for _, allow := range sdkLogAllow {
		if strings.Contains(lower, allow) {
			return true
		}
	}
	return false
}
