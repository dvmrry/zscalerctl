package cli_test

// cobra_dump_test.go - Tests for the Cobra dump command.
//
// Tests here verify:
//  1. dump --output <dir> with a fake reader writes files; stdout empty; exit 0.
//  2. Local flags survive splitGlobalArgs: dump --output /tmp/x --products zia --force
//     parses correctly (key risk: value-taking --products stays intact through the global-strip pass).
//  3. dump (no --output) → UsageError (dumpUsage); dump extra-positional --output x → NArg!=0 UsageError.
//  4. Partial dump → exit 6 (ErrPartialDump): fake reader errors on one resource with
//     --continue-on-error; without that flag the collect fails appropriately.
//  5. --format ndjson dump --output x → exit-2/UsageError (rejectUnsupportedFormat).
//  6. --output <dir> dump → the global flag before the command names the dump directory.
//  7. dump --help → exit 0, Cobra-formatted help containing the dump flags (--output/--products/etc.).

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"zscalerctl/internal/cli"
	"zscalerctl/internal/config"
	"zscalerctl/internal/resources"
)

// newDumpApp returns an App wired to in-memory buffers with an optional fake reader.
func newDumpApp(t *testing.T, reader cli.ResourceReader, env []string) (*cli.App, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	var out, errBuf bytes.Buffer
	var a *cli.App
	if reader != nil {
		a = cli.NewWithOptions(&out, &errBuf, env, cli.Options{Reader: reader})
	} else {
		a = cli.New(&out, &errBuf, env)
	}
	return a, &out, &errBuf
}

// TestCobraDump_BasicWritesFiles confirms that dump --output <dir> with a fake reader
// writes dump files, emits a "dump written" status to stderr, and leaves stdout empty.
func TestCobraDump_BasicWritesFiles(t *testing.T) {
	t.Parallel()

	reader := dumpFakeReader{
		list: []resources.SourceRecord{resources.NewSourceRecord(map[string]any{
			"id":   "1",
			"name": "test",
		})},
	}
	outDir := filepath.Join(t.TempDir(), "dump-out")
	app, out, errBuf := newDumpApp(t, reader, nil)

	err := app.Run(context.Background(), []string{"dump", "--output", outDir})
	if err != nil {
		t.Fatalf("App.Run(dump --output) error = %v, want nil", err)
	}
	if out.Len() != 0 {
		t.Errorf("App.Run(dump --output) stdout = %q, want empty", out.String())
	}
	if !strings.Contains(errBuf.String(), "dump written: "+outDir) {
		t.Errorf("App.Run(dump --output) stderr = %q, want 'dump written: %s'", errBuf.String(), outDir)
	}
	// manifest.json must exist (dump.Write creates it).
	if _, err := os.Stat(filepath.Join(outDir, "manifest.json")); err != nil {
		t.Errorf("os.Stat(manifest.json) = %v, want nil", err)
	}
}

// TestCobraDump_LocalFlagsSurviveSplitGlobalArgs is the KEY RISK test: it verifies
// that local dump flags (especially value-taking --products and --resources, which
// sound global but are NOT) are passed through splitGlobalArgs correctly and parsed
// by Cobra. If splitGlobalArgs stripped --products as a global flag, the value would
// be lost and the command would fail with a usage error.
func TestCobraDump_LocalFlagsSurviveSplitGlobalArgs(t *testing.T) {
	t.Parallel()

	reader := dumpFakeReader{
		list: []resources.SourceRecord{resources.NewSourceRecord(map[string]any{
			"id":   "1",
			"name": "test",
		})},
	}
	outDir := filepath.Join(t.TempDir(), "split-global-dump")
	app, out, errBuf := newDumpApp(t, reader, nil)

	// This invocation mixes global flags (--format, --output) with local dump
	// flags (--products, --force). splitGlobalArgs must take --output and its
	// value and leave --products (non-global, despite sounding global-ish).
	err := app.Run(context.Background(), []string{
		"--format", "table",
		"dump",
		"--output", outDir,
		"--products", "zia",
		"--force",
	})
	if err != nil {
		t.Fatalf("App.Run(dump --output --products --force through splitGlobalArgs) error = %v, want nil", err)
	}
	if out.Len() != 0 {
		t.Errorf("App.Run(dump local flags through splitGlobalArgs) stdout = %q, want empty", out.String())
	}
	if !strings.Contains(errBuf.String(), "dump written: "+outDir) {
		t.Errorf("App.Run(dump local flags through splitGlobalArgs) stderr = %q, want 'dump written: %s'", errBuf.String(), outDir)
	}
}

// TestCobraDump_NoOutputFlag confirms that dump without --output returns a
// UsageError with the dumpUsage string.
func TestCobraDump_NoOutputFlag(t *testing.T) {
	t.Parallel()

	app, _, _ := newDumpApp(t, nil, nil)
	err := app.Run(context.Background(), []string{"dump"})
	if err == nil {
		t.Fatal("App.Run(dump, no --output) error = nil, want UsageError")
	}
	if !errors.Is(err, cli.ErrUsage) {
		t.Errorf("App.Run(dump, no --output) error = %v, want ErrUsage (exit 2)", err)
	}
	if !strings.Contains(err.Error(), "usage: zscalerctl dump") {
		t.Errorf("App.Run(dump, no --output) error = %q, want dumpUsage message", err.Error())
	}
}

// TestCobraDump_ExtraPositionalArg confirms that dump with a positional arg (NArg != 0)
// returns a UsageError with the dumpUsage string.
func TestCobraDump_ExtraPositionalArg(t *testing.T) {
	t.Parallel()

	outDir := filepath.Join(t.TempDir(), "dump-unused")
	app, _, _ := newDumpApp(t, nil, nil)
	err := app.Run(context.Background(), []string{"dump", "extra-positional", "--output", outDir})
	if err == nil {
		t.Fatal("App.Run(dump extra-positional --output x) error = nil, want UsageError")
	}
	if !errors.Is(err, cli.ErrUsage) {
		t.Errorf("App.Run(dump extra-positional --output x) error = %v, want ErrUsage (exit 2)", err)
	}
}

// TestCobraDump_PartialDumpExitsWithErrPartialDump confirms that when a fake reader
// errors on one resource with --continue-on-error, the command returns ErrPartialDump
// (errors.Is == cli.ErrPartialDump; exit 6).
func TestCobraDump_PartialDumpExitsWithErrPartialDump(t *testing.T) {
	t.Parallel()

	reader := dumpSelectiveErrorReader{
		list: []resources.SourceRecord{resources.NewSourceRecord(map[string]any{
			"id":   "1",
			"name": "test",
		})},
		failures: map[string]error{
			"zia/rule-labels": errors.New("simulated-failure"),
		},
	}
	outDir := filepath.Join(t.TempDir(), "partial-dump")
	app, out, errBuf := newDumpApp(t, reader, nil)

	err := app.Run(context.Background(), []string{
		"dump",
		"--products", "zia",
		"--resources", "locations,rule-labels",
		"--continue-on-error",
		"--output", outDir,
	})
	if !errors.Is(err, cli.ErrPartialDump) {
		t.Fatalf("App.Run(dump --continue-on-error) error = %v, want ErrPartialDump (exit 6)", err)
	}
	if !strings.Contains(errBuf.String(), "partial dump written: "+outDir) {
		t.Errorf("App.Run(dump --continue-on-error) stderr = %q, want partial dump written line", errBuf.String())
	}
	if out.Len() != 0 {
		t.Errorf("App.Run(dump --continue-on-error) stdout = %q, want empty", out.String())
	}
}

// TestCobraDump_WithoutContinueOnErrorFailsFast confirms that without --continue-on-error,
// a reader error causes the collect phase to fail immediately (not ErrPartialDump).
func TestCobraDump_WithoutContinueOnErrorFailsFast(t *testing.T) {
	t.Parallel()

	reader := dumpSelectiveErrorReader{
		list: []resources.SourceRecord{resources.NewSourceRecord(map[string]any{
			"id":   "1",
			"name": "test",
		})},
		failures: map[string]error{
			"zia/rule-labels": errors.New("simulated-failure"),
		},
	}
	outDir := filepath.Join(t.TempDir(), "fail-fast-dump")
	app, _, _ := newDumpApp(t, reader, nil)

	err := app.Run(context.Background(), []string{
		"dump",
		"--products", "zia",
		"--resources", "locations,rule-labels",
		"--output", outDir,
	})
	if err == nil {
		t.Fatal("App.Run(dump without --continue-on-error, reader fails) error = nil, want error")
	}
	// Must NOT be ErrPartialDump — that only fires when --continue-on-error is set
	// and the partial result is written. Without it, the collect fails immediately.
	if errors.Is(err, cli.ErrPartialDump) {
		t.Errorf("App.Run(dump without --continue-on-error) error = ErrPartialDump, want plain error from reader failure")
	}
}

// TestCobraDump_FormatNDJSON_Rejected confirms that --format ndjson dump --output x
// returns a UsageError (rejectUnsupportedFormat → exit 2), before any config load.
func TestCobraDump_FormatNDJSON_Rejected(t *testing.T) {
	t.Parallel()

	outDir := filepath.Join(t.TempDir(), "ndjson-dump-unused")
	app, _, _ := newDumpApp(t, nil, nil)
	err := app.Run(context.Background(), []string{"--format", "ndjson", "dump", "--output", outDir})
	if err == nil {
		t.Fatal("App.Run(--format ndjson dump --output x) error = nil, want UsageError")
	}
	if !errors.Is(err, cli.ErrUsage) {
		t.Errorf("App.Run(--format ndjson dump --output x) error = %v, want ErrUsage (exit 2)", err)
	}
	if !strings.Contains(err.Error(), "ndjson") {
		t.Errorf("App.Run(--format ndjson dump) error = %q, want ndjson mentioned", err.Error())
	}
}

// TestCobraDump_OutputBeforeCommand confirms that the global --output may come
// before the command and names the dump directory; nothing is written to stdout
// or to a file wrapper.
func TestCobraDump_OutputBeforeCommand(t *testing.T) {
	t.Parallel()

	reader := dumpFakeReader{
		list: []resources.SourceRecord{resources.NewSourceRecord(map[string]any{
			"id":   "1",
			"name": "test",
		})},
	}
	outDir := filepath.Join(t.TempDir(), "dump-before")
	app, out, errBuf := newDumpApp(t, reader, nil)

	err := app.Run(context.Background(), []string{"--output", outDir, "dump", "--products", "zia"})
	if err != nil {
		t.Fatalf("App.Run(--output <dir> dump) error = %v, want nil", err)
	}
	if out.Len() != 0 {
		t.Errorf("App.Run(--output <dir> dump) stdout = %q, want empty", out.String())
	}
	if !strings.Contains(errBuf.String(), "dump written: "+outDir) {
		t.Errorf("App.Run(--output <dir> dump) stderr = %q, want 'dump written: %s'", errBuf.String(), outDir)
	}
	if _, err := os.Stat(filepath.Join(outDir, "manifest.json")); err != nil {
		t.Errorf("os.Stat(%s/manifest.json) = %v, want nil", outDir, err)
	}
}

// TestCobraDump_HelpCobraFormatted confirms that dump --help exits 0 and renders
// Cobra-formatted help that includes the dump flags (--output, --products, --resources,
// --continue-on-error, --force) and does not leak credentials.
func TestCobraDump_HelpCobraFormatted(t *testing.T) {
	t.Parallel()

	app, out, errBuf := newDumpApp(t, nil, []string{
		config.EnvClientSecretFile + "=/path/that/must/not-be-read",
	})
	err := app.Run(context.Background(), []string{"dump", "--help"})
	if err != nil {
		t.Fatalf("App.Run(dump --help) error = %v, want nil", err)
	}
	got := out.String()
	if got == "" {
		t.Fatal("App.Run(dump --help) stdout is empty; Cobra help should have been rendered")
	}
	// Cobra help must include "zscalerctl dump" in the Usage line.
	if !strings.Contains(got, "zscalerctl dump") {
		t.Errorf("App.Run(dump --help) stdout = %q, want 'zscalerctl dump'", got)
	}
	// All dump flags must appear.
	for _, flag := range []string{"--output", "--products", "--resources", "--continue-on-error", "--force"} {
		if !strings.Contains(got, flag) {
			t.Errorf("App.Run(dump --help) stdout = %q, want flag %q", got, flag)
		}
	}
	// Credential-like strings must not appear.
	if strings.Contains(got, "/path/that/must/not-be-read") {
		t.Errorf("App.Run(dump --help) stdout = %q, leaked secret path", got)
	}
	if errBuf.Len() != 0 {
		t.Errorf("App.Run(dump --help) stderr = %q, want empty", errBuf.String())
	}
}

// TestCobraDump_ConfigError confirms that dump surfaces a config load error
// (ErrInvalidConfig -> exit 2) when --config points to a nonexistent path.
func TestCobraDump_ConfigError(t *testing.T) {
	t.Parallel()

	outDir := filepath.Join(t.TempDir(), "dump-config-err")
	app, _, _ := newDumpApp(t, nil, nil)
	err := app.Run(context.Background(), []string{
		"--config", "/nonexistent/path/zscalerctl.yaml",
		"dump", "--output", outDir,
	})
	if err == nil {
		t.Fatal("App.Run(dump --config /nonexistent) error = nil, want config error")
	}
	if !errors.Is(err, config.ErrInvalidConfig) {
		t.Errorf("App.Run(dump --config /nonexistent) error = %v, want ErrInvalidConfig", err)
	}
}

// TestCobraDump_DashPrefixedOutputValue confirms that "dump --output -weird-path"
// takes -weird-path as the directory: the global flag consumes its next token
// verbatim, so a dash-prefixed path needs no "--".
func TestCobraDump_DashPrefixedOutputValue(t *testing.T) {
	t.Parallel()

	reader := dumpFakeReader{
		list: []resources.SourceRecord{resources.NewSourceRecord(map[string]any{
			"id":   "1",
			"name": "test",
		})},
	}
	app, out, errBuf := newDumpApp(t, reader, nil)

	// A relative -weird-path would need the working directory changed, which a
	// parallel test cannot do; the absolute path keeps the dash-prefixed name as
	// its last element.
	outDir := filepath.Join(t.TempDir(), "-weird-path")

	err := app.Run(context.Background(), []string{"dump", "--output", outDir})
	if err != nil {
		t.Fatalf("App.Run(dump --output -weird-path) error = %v, want nil", err)
	}
	if out.Len() != 0 {
		t.Errorf("App.Run(dump --output -weird-path) stdout = %q, want empty", out.String())
	}
	if !strings.Contains(errBuf.String(), "dump written: "+outDir) {
		t.Errorf("App.Run(dump --output -weird-path) stderr = %q, want 'dump written: %s'", errBuf.String(), outDir)
	}
	if _, err := os.Stat(filepath.Join(outDir, "manifest.json")); err != nil {
		t.Errorf("os.Stat(%s/manifest.json) = %v, want nil", outDir, err)
	}
}

// TestCobraDump_DashPrefixedLocalValue_Terminator confirms that
// "dump --resources -- -weird" gives -weird to --resources rather than failing
// on an unknown flag: the terminator after a value-taking local flag is not
// reinserted.
func TestCobraDump_DashPrefixedLocalValue_Terminator(t *testing.T) {
	t.Parallel()

	outDir := filepath.Join(t.TempDir(), "dump-unused")
	app, _, _ := newDumpApp(t, nil, nil)

	err := app.Run(context.Background(), []string{"dump", "--output", outDir, "--resources", "--", "-weird"})
	if !errors.Is(err, cli.ErrUsage) {
		t.Fatalf("App.Run(dump --resources -- -weird) error = %v, want ErrUsage", err)
	}
	if !strings.Contains(err.Error(), "unsupported dump resource") {
		t.Errorf("App.Run(dump --resources -- -weird) error = %q, want the value read as a resource", err.Error())
	}
}

// TestCobraDump_BoolFlag_TerminatorStillProtectsPositionals confirms that when
// the token before "--" is a boolean local flag (e.g. --force), the terminator
// is still reinserted so the following dash-prefixed token is treated as a
// positional, not a flag. The command should fail with an extra-positional usage
// error, not an unknown flag error.
func TestCobraDump_BoolFlag_TerminatorStillProtectsPositionals(t *testing.T) {
	t.Parallel()

	app, _, _ := newDumpApp(t, nil, nil)
	err := app.Run(context.Background(), []string{"dump", "--force", "--", "-extra"})
	if err == nil {
		t.Fatal("App.Run(dump --force -- -extra) error = nil, want UsageError")
	}
	if !errors.Is(err, cli.ErrUsage) {
		t.Errorf("App.Run(dump --force -- -extra) error = %v, want ErrUsage (extra positional)", err)
	}
	if strings.Contains(err.Error(), "unknown flag") || strings.Contains(err.Error(), "unknown shorthand") {
		t.Errorf("App.Run(dump --force -- -extra) error = %q, want not a flag error", err.Error())
	}
}

// ── local fake readers for cobra_dump_test.go ────────────────────────────────

// dumpFakeReader implements cli.ResourceReader for dump tests.
// It always succeeds, returning the same list for every List call.
type dumpFakeReader struct {
	list []resources.SourceRecord
}

func (r dumpFakeReader) List(_ context.Context, _ resources.Product, _ string) ([]resources.SourceRecord, error) {
	return r.list, nil
}

func (r dumpFakeReader) Get(_ context.Context, _ resources.Product, _ string, _ string) (resources.SourceRecord, error) {
	return resources.SourceRecord{}, nil
}

func (r dumpFakeReader) Show(_ context.Context, _ resources.Product, _ string) (resources.SourceRecord, error) {
	return resources.SourceRecord{}, nil
}

// dumpSelectiveErrorReader returns an error for specific "product/resource" keys.
type dumpSelectiveErrorReader struct {
	list     []resources.SourceRecord
	failures map[string]error
}

func (r dumpSelectiveErrorReader) List(_ context.Context, product resources.Product, name string) ([]resources.SourceRecord, error) {
	if err := r.failures[string(product)+"/"+name]; err != nil {
		return nil, err
	}
	return r.list, nil
}

func (r dumpSelectiveErrorReader) Get(_ context.Context, _ resources.Product, _ string, _ string) (resources.SourceRecord, error) {
	return resources.SourceRecord{}, errors.New("get must not be called")
}

func (r dumpSelectiveErrorReader) Show(_ context.Context, _ resources.Product, _ string) (resources.SourceRecord, error) {
	return resources.SourceRecord{}, errors.New("show must not be called")
}
