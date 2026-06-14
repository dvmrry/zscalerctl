package agenteval_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestShimBinaryBehavior builds the fixture binary
// (internal/agenteval/cmd/zscalerctl-fixture) into t.TempDir() — the pwsh-smoke
// pattern from internal/cli/app_test.go — and asserts the §2.3 contract end to
// end: it never falls through to a live reader without ZSCALERCTL_FIXTURE_DIR,
// it serves the value-free corpus past the real reader seam, it exits 4 on an
// unknown id, and a no-synthetic-creds scenario reaches the genuine exit-3
// missing_credentials path. Without this gate the live runner could silently
// feed agents wrong data.
//
// Every ZSCALERCTL_* value used here is obviously synthetic and value-free: the
// vanity domain is a documentation label and the client id/secret are literal
// "synthetic-..." placeholders that pass validation but never reach an endpoint
// (the fixture reader, not a live client, serves the data).
func TestShimBinaryBehavior(t *testing.T) {
	t.Parallel()

	bin := buildFixtureBinary(t)
	fixtureDir := t.TempDir()

	// Synthetic, value-free OneAPI credentials. These satisfy
	// zscaler.ValidateReaderConfig (ClientID + ClientSecret + VanityDomain) so the
	// fixture reader is injected and normal reads succeed.
	syntheticCreds := []string{
		"ZSCALERCTL_CLIENT_ID=synthetic-client-id",
		"ZSCALERCTL_CLIENT_SECRET=synthetic-client-secret",
		"ZSCALERCTL_VANITY_DOMAIN=example",
	}

	t.Run("no ZSCALERCTL_FIXTURE_DIR exits 1", func(t *testing.T) {
		t.Parallel()
		// FIXTURE_DIR deliberately omitted; synthetic creds present to prove the
		// gate fires before (and independent of) credential validation.
		res := runFixture(t, bin, syntheticCreds, "zia", "locations", "list")
		if res.exit != 1 {
			t.Fatalf("exit = %d, want 1 (must hard-fail without %s)\nstderr: %s", res.exit, "ZSCALERCTL_FIXTURE_DIR", res.stderr)
		}
		if len(res.stdout) != 0 {
			t.Fatalf("stdout = %q, want empty (must not serve data without a fixture dir)", res.stdout)
		}
	})

	t.Run("locations list exits 0 with array length > 1", func(t *testing.T) {
		t.Parallel()
		env := withFixtureDir(fixtureDir, syntheticCreds)
		res := runFixture(t, bin, env, "--format", "json", "zia", "locations", "list")
		if res.exit != 0 {
			t.Fatalf("exit = %d, want 0\nstderr: %s", res.exit, res.stderr)
		}
		var records []map[string]any
		if err := json.Unmarshal(res.stdout, &records); err != nil {
			t.Fatalf("stdout is not a JSON array: %v\nstdout: %s", err, res.stdout)
		}
		if len(records) <= 1 {
			t.Fatalf("locations list length = %d, want > 1 (cardinality mandate, plan §3.5)", len(records))
		}
	})

	t.Run("locations get known id exits 0 with the record", func(t *testing.T) {
		t.Parallel()
		env := withFixtureDir(fixtureDir, syntheticCreds)
		// id 1 is the synthetic "HQ" record in the corpus.
		res := runFixture(t, bin, env, "--format", "json", "zia", "locations", "get", "1")
		if res.exit != 0 {
			t.Fatalf("exit = %d, want 0\nstderr: %s", res.exit, res.stderr)
		}
		var record map[string]any
		if err := json.Unmarshal(res.stdout, &record); err != nil {
			t.Fatalf("stdout is not a JSON object: %v\nstdout: %s", err, res.stdout)
		}
		if got := record["name"]; got != "HQ" {
			t.Fatalf("get 1 name = %v, want \"HQ\"\nstdout: %s", got, res.stdout)
		}
	})

	t.Run("locations get unknown id exits 4", func(t *testing.T) {
		t.Parallel()
		env := withFixtureDir(fixtureDir, syntheticCreds)
		res := runFixture(t, bin, env, "--format", "json", "zia", "locations", "get", "999999")
		if res.exit != 4 {
			t.Fatalf("exit = %d, want 4 (not_found)\nstderr: %s", res.exit, res.stderr)
		}
		if kind := errorKind(t, res.stderr); kind != "not_found" {
			t.Fatalf("error kind = %q, want \"not_found\"\nstderr: %s", kind, res.stderr)
		}
	})

	t.Run("missing credentials exits 3", func(t *testing.T) {
		t.Parallel()
		// FIXTURE_DIR set, but NO synthetic creds: validation must return
		// missing_credentials through the real path and the fixture reader must
		// NOT be injected.
		env := []string{"ZSCALERCTL_FIXTURE_DIR=" + fixtureDir}
		res := runFixture(t, bin, env, "--format", "json", "zia", "locations", "list")
		if res.exit != 3 {
			t.Fatalf("exit = %d, want 3 (missing_credentials)\nstderr: %s", res.exit, res.stderr)
		}
		if kind := errorKind(t, res.stderr); kind != "missing_credentials" {
			t.Fatalf("error kind = %q, want \"missing_credentials\"\nstderr: %s", kind, res.stderr)
		}
	})

	t.Run("schema list exits 0 with full catalog", func(t *testing.T) {
		t.Parallel()
		env := withFixtureDir(fixtureDir, syntheticCreds)
		res := runFixture(t, bin, env, "--format", "json", "schema", "list")
		if res.exit != 0 {
			t.Fatalf("exit = %d, want 0\nstderr: %s", res.exit, res.stderr)
		}
		var catalog []map[string]any
		if err := json.Unmarshal(res.stdout, &catalog); err != nil {
			t.Fatalf("stdout is not a JSON array: %v\nstdout: %s", err, res.stdout)
		}
		// The fixture schema list returns the full catalog (plan §3.5); the corpus
		// has data for only one resource, so a many-entry catalog proves discovery
		// is not limited to fixture-backed resources.
		if len(catalog) <= 1 {
			t.Fatalf("schema list length = %d, want full catalog (> 1)", len(catalog))
		}
	})

	t.Run("observed-command sidecar records argv and exit", func(t *testing.T) {
		t.Parallel()
		// ZSCALERCTL_FIXTURE_LOG is a CONFINED relative filename, resolved against
		// the process cwd (the runner sets cwd to the agent WorkDir). Run the binary
		// with its cwd set to a temp dir and read the log back from there.
		workDir := t.TempDir()
		const logName = "observed.jsonl"
		env := append(withFixtureDir(fixtureDir, syntheticCreds), "ZSCALERCTL_FIXTURE_LOG="+logName)
		res := runFixtureInDir(t, bin, workDir, env, "--format", "json", "zia", "locations", "list")
		if res.exit != 0 {
			t.Fatalf("exit = %d, want 0\nstderr: %s", res.exit, res.stderr)
		}
		data, err := os.ReadFile(filepath.Join(workDir, logName))
		if err != nil {
			t.Fatalf("read sidecar: %v", err)
		}
		var record struct {
			Argv []string `json:"argv"`
			Exit int      `json:"exit"`
		}
		line := bytes.TrimSpace(data)
		if err := json.Unmarshal(line, &record); err != nil {
			t.Fatalf("sidecar line is not valid JSON: %v\nline: %s", err, line)
		}
		if record.Exit != 0 {
			t.Fatalf("sidecar exit = %d, want 0", record.Exit)
		}
		if len(record.Argv) == 0 || record.Argv[len(record.Argv)-1] != "list" {
			t.Fatalf("sidecar argv = %v, want it to end with the command args", record.Argv)
		}
	})
}

// buildFixtureBinary compiles the fixture command into t.TempDir() and returns
// the binary path, mirroring the build-then-exec smoke pattern.
func buildFixtureBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "zscalerctl-fixture")
	cmd := exec.Command("go", "build", "-o", bin, "github.com/dvmrry/zscalerctl/internal/agenteval/cmd/zscalerctl-fixture")
	var buildErr bytes.Buffer
	cmd.Stderr = &buildErr
	if err := cmd.Run(); err != nil {
		t.Fatalf("build fixture binary: %v\n%s", err, buildErr.String())
	}
	return bin
}

type fixtureResult struct {
	stdout []byte
	stderr string
	exit   int
}

// runFixture executes the fixture binary with a clean env (no inherited
// ZSCALERCTL_* values — env hygiene per plan §6.3) plus the caller-supplied
// vars, and reports stdout/stderr/exit. The process cwd is left to the test
// harness default.
func runFixture(t *testing.T, bin string, env []string, args ...string) fixtureResult {
	t.Helper()
	return runFixtureInDir(t, bin, "", env, args...)
}

// runFixtureInDir is runFixture with the process cwd pinned to dir (empty = the
// harness default). The confined sidecar log path (ZSCALERCTL_FIXTURE_LOG, a
// relative filename) resolves against this cwd, so a test exercising the log
// sets dir to the directory it then reads the log back from.
func runFixtureInDir(t *testing.T, bin, dir string, env []string, args ...string) fixtureResult {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	// Start from an explicitly minimal env so the test process's real
	// credentials can never leak into a scenario and flip its outcome.
	cmd.Env = append([]string{"PATH=" + os.Getenv("PATH")}, env...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exit := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exit = exitErr.ExitCode()
		} else {
			t.Fatalf("run fixture binary: %v\nstderr: %s", err, stderr.String())
		}
	}
	return fixtureResult{stdout: stdout.Bytes(), stderr: stderr.String(), exit: exit}
}

func withFixtureDir(dir string, creds []string) []string {
	return append([]string{"ZSCALERCTL_FIXTURE_DIR=" + dir}, creds...)
}

// errorKind decodes the stderr JSON error envelope and returns its kind, so a
// scenario can assert the documented kind alongside the exit code.
func errorKind(t *testing.T, stderr string) string {
	t.Helper()
	var envelope struct {
		Error struct {
			Kind string `json:"kind"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(stderr), &envelope); err != nil {
		t.Fatalf("stderr is not a JSON error envelope: %v\nstderr: %s", err, stderr)
	}
	return envelope.Error.Kind
}
