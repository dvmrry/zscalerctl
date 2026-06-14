// Command zscalerctl-fixture is the agent-eval fixture binary
// (docs/AGENTIC_COVERAGE_PLAN.md §2.3). It IS the production zscalerctl CLI with
// exactly one thing swapped: the data source. Everything past the reader seam —
// projection, redaction, --fields, --filter, --search, `schema list`, the
// stderr JSON error envelope, and the exit codes — is the real internal/cli
// path. Only the ResourceReader is replaced with the value-free fixture corpus
// (internal/agenteval/fixtures), so the eval can grade self-describability
// without ever contacting a network or touching a live tenant.
//
// It deliberately lives under internal/agenteval/cmd, NOT top-level cmd/, and
// carries NO build tag: isolation is guaranteed structurally (production
// cmd/zscalerctl never imports the fixtures package, and the release workflow
// builds only ./cmd/zscalerctl) rather than by a forgettable -tags guard.
//
// Selection + safety contract:
//
//   - ZSCALERCTL_FIXTURE_DIR unset/empty -> hard exit 1 at startup. The binary
//     can NEVER fall through to a live reader.
//   - The real credential-validation path runs EXPLICITLY at startup via
//     zscaler.ValidateReaderConfig(cfg). Injecting a reader through
//     cli.Options.Reader short-circuits resourceReader() and bypasses
//     NewReader's validation, so the fixture main reproduces it by hand. A
//     scenario with no synthetic creds therefore exits 3 / missing_credentials
//     through the genuine path; the fixture reader is NOT injected in that case.
//   - With synthetic creds present, the App is constructed with the fixture
//     reader and the real catalog, and run normally.
//
// The error-rendering machinery (errorEnvelope/errorDetails/errorKind/
// exitCodeForError/writeError) mirrors cmd/zscalerctl/main.go verbatim, because
// those helpers are unexported in that package. Both the explicit validation
// error and any error App.Run returns funnel through the same mapping, so a
// missing-credentials scenario and a get-<unknown-id> scenario surface exactly
// the exit codes and JSON envelope the real binary would.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dvmrry/zscalerctl/internal/agenteval/fixtures"
	"github.com/dvmrry/zscalerctl/internal/cli"
	"github.com/dvmrry/zscalerctl/internal/config"
	"github.com/dvmrry/zscalerctl/internal/output"
	"github.com/dvmrry/zscalerctl/internal/redact"
	"github.com/dvmrry/zscalerctl/internal/resources"
	"github.com/dvmrry/zscalerctl/internal/zscaler"
)

const (
	exitSuccess           = 0
	exitInternalError     = 1
	exitUsageError        = 2
	exitCredentialError   = 3
	exitNotFound          = 4
	exitLiveAccessFailure = 5
	exitPartialDump       = 6

	// exitNoFixtureDir is the hard-fail code when ZSCALERCTL_FIXTURE_DIR is unset
	// — distinct from the CLI's own exit codes so a misconfigured runner is
	// obviously not a graded CLI outcome.
	exitNoFixtureDir = 1

	envFixtureDir = "ZSCALERCTL_FIXTURE_DIR"
	envFixtureLog = "ZSCALERCTL_FIXTURE_LOG"
)

func main() {
	code := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr, os.Environ())
	logObserved(os.Getenv(envFixtureLog), os.Args, code)
	os.Exit(code)
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer, env []string) int {
	// Gate: refuse to do anything without ZSCALERCTL_FIXTURE_DIR. This is the
	// structural guarantee that the fixture binary can never reach a live reader.
	if dir := getenv(env, envFixtureDir); dir == "" {
		fmt.Fprintf(stderr, "zscalerctl-fixture: %s is unset; this binary serves only the value-free fixture corpus and never contacts a live tenant. Refusing to run.\n", envFixtureDir)
		return exitNoFixtureDir
	}

	// Parse the ZSCALERCTL_* block exactly as the real CLI does, then run the
	// genuine credential-validation path. A config-load error (an unparseable
	// ZSCALERCTL_* setting) is itself a real error-path outcome and is rendered
	// through the same envelope/exit-code mapping below.
	cfg, err := config.LoadEnv(env)
	if err != nil {
		return emitError(args, stdout, stderr, err)
	}
	if err := zscaler.ValidateReaderConfig(readerConfig(cfg)); err != nil {
		// No synthetic creds -> missing_credentials -> exit 3 through the real
		// path. Do NOT inject the fixture reader here: the credential-negative
		// scenario must be reachable without a live tenant precisely because the
		// reader is absent.
		return emitError(args, stdout, stderr, err)
	}

	// Validation passed (synthetic creds present): swap in the fixture reader and
	// the real catalog, then run the genuine CLI. Only the data source differs.
	app := cli.NewWithOptions(stdout, stderr, env, cli.Options{
		Reader:  fixtures.NewReader(),
		Catalog: resources.Catalog(),
	})
	if err := app.Run(ctx, args); err != nil {
		return emitError(args, stdout, stderr, err)
	}
	return exitSuccess
}

// readerConfig maps a parsed config.Config into a zscaler.ReaderConfig using the
// exact same field mapping as (*cli.App).resourceReader (internal/cli/app.go),
// so ValidateReaderConfig sees what the production reader would. Timeout/NoCache
// are not credential-relevant for validation; they are filled from config
// defaults for fidelity.
func readerConfig(cfg config.Config) zscaler.ReaderConfig {
	return zscaler.ReaderConfig{
		ClientID:         cfg.Credentials.ClientID,
		ClientSecret:     cfg.Credentials.ClientSecret,
		VanityDomain:     cfg.VanityDomain,
		Cloud:            cfg.Cloud,
		ZPACustomerID:    cfg.ZPA.CustomerID,
		ZPAMicrotenantID: cfg.ZPA.MicrotenantID,
		AuthMode:         zscaler.AuthMode(cfg.EffectiveAuthMode()),
		ZIALegacy: zscaler.ZIALegacyConfig{
			Username: cfg.ZIALegacy.Username,
			Password: cfg.ZIALegacy.Password,
			APIKey:   cfg.ZIALegacy.APIKey,
			Cloud:    cfg.ZIALegacy.Cloud,
		},
		NoCache: cfg.Defaults.NoCache,
		Proxy: zscaler.ProxyConfig{
			URL:             cfg.Proxy.URL,
			FromEnvironment: cfg.Proxy.FromEnvironment,
		},
	}
}

func getenv(env []string, key string) string {
	prefix := key + "="
	// Last assignment wins, mirroring os.Environ/process semantics.
	value := ""
	for _, kv := range env {
		if len(kv) >= len(prefix) && kv[:len(prefix)] == prefix {
			value = kv[len(prefix):]
		}
	}
	return value
}

// emitError renders a command-boundary error through the exact path
// cmd/zscalerctl/main.go uses: the documented exit code plus, for the JSON
// format (or auto resolving to JSON because stdout is not a terminal), the
// stderr JSON error envelope; otherwise a plain-text line. ErrPartialDump in a
// non-JSON format suppresses the stderr line (the dump command already wrote its
// own notice), matching the real binary.
func emitError(args []string, stdout, stderr io.Writer, err error) int {
	code := exitCodeForError(err)
	format := errorFormat(args, stdout)
	if errors.Is(err, cli.ErrPartialDump) && format != output.FormatJSON {
		return code
	}
	writeError(stderr, format, err)
	return code
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

func (errorEnvelope) OutputSafe() {}

type errorBody struct {
	Kind      string   `json:"kind"`
	Message   string   `json:"message"`
	Missing   []string `json:"missing,omitempty"`
	Operation string   `json:"operation,omitempty"`
	Product   string   `json:"product,omitempty"`
	Resource  string   `json:"resource,omitempty"`
}

func errorFormat(args []string, stdout io.Writer) output.Format {
	switch cli.RequestedFormatRaw(args) {
	case output.FormatJSON:
		return output.FormatJSON
	case output.FormatAuto:
		if !output.IsTerminal(stdout) {
			return output.FormatJSON
		}
		return output.FormatTable
	default:
		return output.FormatTable
	}
}

func writeError(w io.Writer, format output.Format, err error) {
	if format == output.FormatJSON {
		envelope := errorEnvelope{Error: errorDetails(err)}
		_ = output.NewRenderer(redact.New(redact.ModeStandard)).WriteJSON(w, envelope)
		return
	}
	message, _ := redact.New(redact.ModeStandard).ScanRenderedString(err.Error())
	fmt.Fprintf(w, "zscalerctl: %s\n", message)
}

func errorDetails(err error) errorBody {
	message, _ := redact.New(redact.ModeStandard).ScanRenderedString(err.Error())
	body := errorBody{
		Kind:    errorKind(err),
		Message: message,
	}
	var notFound cli.ResourceNotFoundError
	if errors.As(err, &notFound) {
		body.Product = string(notFound.Product)
		body.Resource = notFound.Resource
	}
	var mce *zscaler.MissingCredentialsError
	if errors.As(err, &mce) {
		body.Missing = mce.Missing
	}
	var ctx zscaler.ErrorContexter
	if errors.As(err, &ctx) {
		c := ctx.ErrorContext()
		if c.Product != "" {
			body.Product = c.Product
		}
		if c.Resource != "" {
			body.Resource = c.Resource
		}
		if c.Operation != "" {
			body.Operation = c.Operation
		}
	}
	return body
}

func errorKind(err error) string {
	switch {
	case errors.Is(err, cli.ErrUsage):
		return "usage"
	case errors.Is(err, cli.ErrPartialDump):
		return "partial_dump"
	case errors.Is(err, cli.ErrNotFound):
		return "not_found"
	case errors.Is(err, zscaler.ErrResourceNotFound):
		return "not_found"
	case errors.Is(err, zscaler.ErrMissingCredentials):
		return "missing_credentials"
	case errors.Is(err, zscaler.ErrInvalidResourceID):
		return "invalid_resource_id"
	case errors.Is(err, zscaler.ErrUnsupportedResource):
		return "unsupported_resource"
	case errors.Is(err, zscaler.ErrLiveAccessFailed):
		return "live_access_failed"
	case errors.Is(err, zscaler.ErrInvalidProxyConfig):
		return "invalid_proxy_config"
	case errors.Is(err, config.ErrInvalidConfig):
		return "invalid_config"
	default:
		return "internal"
	}
}

func exitCodeForError(err error) int {
	switch {
	case errors.Is(err, cli.ErrUsage):
		return exitUsageError
	case errors.Is(err, cli.ErrPartialDump):
		return exitPartialDump
	case errors.Is(err, cli.ErrNotFound):
		return exitNotFound
	case errors.Is(err, zscaler.ErrResourceNotFound):
		return exitNotFound
	case errors.Is(err, zscaler.ErrMissingCredentials):
		return exitCredentialError
	case errors.Is(err, zscaler.ErrInvalidResourceID):
		return exitUsageError
	case errors.Is(err, zscaler.ErrUnsupportedResource):
		return exitNotFound
	case errors.Is(err, zscaler.ErrLiveAccessFailed):
		return exitLiveAccessFailure
	case errors.Is(err, zscaler.ErrInvalidProxyConfig):
		return exitUsageError
	case errors.Is(err, config.ErrInvalidConfig):
		return exitUsageError
	default:
		return exitInternalError
	}
}

// observedRecord is one line of the observed-command sidecar (§2.3). argv and
// exit are always present; stdout fields are omitted here because run writes
// directly to os.Stdout (the real CLI mutes/owns fd 1) and a tee would risk
// corrupting that stream. The runner's authoritative stdout capture lives in
// the live half; the sidecar's minimum contract is argv + exit.
type observedRecord struct {
	Argv []string `json:"argv"`
	Exit int      `json:"exit"`
}

// logObserved appends one JSON line per process to the path in
// ZSCALERCTL_FIXTURE_LOG, if set. Best-effort: a logging failure must never
// change the process's exit code or contact a network. The file is opened
// append-only so concurrent fixture invocations each contribute one line.
//
// The log path is CONFINED: ZSCALERCTL_FIXTURE_LOG is treated as a relative
// filename only, resolved against the process cwd (which the runner sets to the
// agent WorkDir). An absolute path or a ".." traversal element (after cleaning)
// is rejected — the env-controlled value can never escape the working directory
// to write an arbitrary file. Rejection is silent (best-effort logging), never
// an exit-code or stream change.
func logObserved(path string, argv []string, exit int) {
	if path == "" {
		return
	}
	clean := filepath.Clean(path)
	// Reject an absolute path or any ".." traversal: the env value must be a
	// confined relative filename under the process cwd.
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return
	}
	record := observedRecord{Argv: append([]string(nil), argv...), Exit: exit}
	line, err := json.Marshal(record)
	if err != nil {
		return
	}
	// #nosec G304 -- relative filename, absolute/.. rejected; the fixture binary is internal eval tooling, never shipped
	f, err := os.OpenFile(clean, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	_, _ = f.Write(append(line, '\n'))
}
