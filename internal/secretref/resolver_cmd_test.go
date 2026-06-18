package secretref

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResolverCmdResolvesStdoutAndTrimsNewlines(t *testing.T) {
	t.Parallel()

	got, err := NewResolver(ResolverOpts{AllowCmd: true}).Resolve(context.Background(), SecretRef{
		Scheme: "cmd",
		Argv:   helperCommand("print", "resolved-secret\r\n\n"),
	})
	if err != nil {
		t.Fatalf("Resolve(cmd print) error = %v, want nil", err)
	}
	if got.Reveal() != "resolved-secret" {
		t.Fatalf("Resolve(cmd print) = %q, want trimmed secret", got.Reveal())
	}
}

func TestResolverCmdExecutesArgvLiterallyWithoutShell(t *testing.T) {
	t.Parallel()

	sentinel := filepath.Join(t.TempDir(), "shell-expanded")
	payload := "$(touch " + sentinel + ")"
	got, err := NewResolver(ResolverOpts{AllowCmd: true}).Resolve(context.Background(), SecretRef{
		Scheme: "cmd",
		Argv:   helperCommand("echo-args", payload, "literal;rm", "literal|pipe"),
	})
	if err != nil {
		t.Fatalf("Resolve(cmd echo-args) error = %v, want nil", err)
	}
	if !strings.Contains(got.Reveal(), payload) {
		t.Fatalf("Resolve(cmd echo-args) = %q, want literal shell-looking argument", got.Reveal())
	}
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Fatalf("shell-looking argument created %q; stat err = %v", sentinel, err)
	}
}

func TestResolverCmdTimeoutKillsProvider(t *testing.T) {
	t.Parallel()

	start := time.Now()
	_, err := NewResolver(ResolverOpts{AllowCmd: true}).Resolve(context.Background(), SecretRef{
		Scheme:  "cmd",
		Argv:    helperCommand("sleep", "2s"),
		Timeout: 10 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("Resolve(cmd sleep) error = nil, want timeout error")
	}
	if time.Since(start) > time.Second {
		t.Fatalf("Resolve(cmd sleep) took %s, want timeout to kill provider promptly", time.Since(start))
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("Resolve(cmd sleep) error = %v, want timeout text", err)
	}
}

func TestResolverCmdErrorIsValueFree(t *testing.T) {
	t.Parallel()

	const stderrSecret = "stderr-secret-token-1234567890"
	_, err := NewResolver(ResolverOpts{AllowCmd: true}).Resolve(context.Background(), SecretRef{
		Scheme: "cmd",
		Argv:   helperCommand("fail-stderr", stderrSecret),
	})
	if err == nil {
		t.Fatal("Resolve(cmd fail-stderr) error = nil, want error")
	}
	if strings.Contains(err.Error(), stderrSecret) {
		t.Fatalf("Resolve(cmd fail-stderr) error = %q, want value-free stderr summary", err.Error())
	}
	if !strings.Contains(err.Error(), "stderr omitted") {
		t.Fatalf("Resolve(cmd fail-stderr) error = %q, want stderr omission summary", err.Error())
	}
}

func TestResolverCmdDisabledDoesNotExec(t *testing.T) {
	t.Parallel()

	sentinel := filepath.Join(t.TempDir(), "provider-ran")
	_, err := NewResolver(ResolverOpts{AllowCmd: false}).Resolve(context.Background(), SecretRef{
		Scheme: "cmd",
		Argv:   helperCommand("touch", sentinel),
	})
	if err == nil {
		t.Fatal("Resolve(disabled cmd) error = nil, want disabled error")
	}
	if _, statErr := os.Stat(sentinel); !os.IsNotExist(statErr) {
		t.Fatalf("disabled cmd provider created %q; stat err = %v", sentinel, statErr)
	}
}

func TestResolverCmdWaitDelayBoundsForkingChild(t *testing.T) {
	t.Parallel()

	start := time.Now()
	_, err := NewResolver(ResolverOpts{AllowCmd: true}).Resolve(context.Background(), SecretRef{
		Scheme:  "cmd",
		Argv:    helperCommand("fork-sleep", "5s"),
		Timeout: 10 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("Resolve(cmd fork-sleep) error = nil, want timeout error")
	}
	if time.Since(start) > 3*time.Second {
		t.Fatalf("Resolve(cmd fork-sleep) took %s, want timeout+WaitDelay bounds (approx 2s)", time.Since(start))
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("Resolve(cmd fork-sleep) error = %v, want timeout text", err)
	}
}

func TestResolverCmdStripsZscalerctlEnv(t *testing.T) {
	t.Setenv("ZSCALERCTL_TEST_LEAK", "secret-value")

	got, err := NewResolver(ResolverOpts{AllowCmd: true}).Resolve(context.Background(), SecretRef{
		Scheme: "cmd",
		Argv:   helperCommand("print-env"),
	})
	if err != nil {
		t.Fatalf("Resolve(cmd print-env) error = %v, want nil", err)
	}
	if strings.Contains(got.Reveal(), "ZSCALERCTL_TEST_LEAK") {
		t.Errorf("Resolve(cmd print-env) output contains ZSCALERCTL_TEST_LEAK. Output: %q", got.Reveal())
	}
}

func TestResolverCmdBoundsOutput(t *testing.T) {
	t.Parallel()

	_, err := NewResolver(ResolverOpts{AllowCmd: true}).Resolve(context.Background(), SecretRef{
		Scheme: "cmd",
		Argv:   helperCommand("write-large", "100000"), // 100k
	})
	if err == nil {
		t.Fatal("Resolve(cmd write-large) error = nil, want error")
	}
	if !strings.Contains(err.Error(), "output too large") {
		t.Fatalf("Resolve(cmd write-large) error = %v, want 'output too large'", err)
	}
}

func TestResolverCmdEmptyOutput(t *testing.T) {
	t.Parallel()

	_, err := NewResolver(ResolverOpts{AllowCmd: true}).Resolve(context.Background(), SecretRef{
		Scheme: "cmd",
		Argv:   helperCommand("print", "\r\n"),
	})
	if err == nil {
		t.Fatal("Resolve(cmd empty) error = nil, want error")
	}
	if !strings.Contains(err.Error(), "produced no output") {
		t.Fatalf("Resolve(cmd empty) error = %v, want 'produced no output'", err)
	}
}

func helperCommand(args ...string) []string {
	cmd := []string{os.Args[0], "-test.run=^TestResolverCmdHelperProcess$", "--"}
	return append(cmd, args...)
}

func TestResolverCmdHelperProcess(t *testing.T) {
	index := -1
	for i, arg := range os.Args {
		if arg == "--" {
			index = i
			break
		}
	}
	if index < 0 {
		return
	}
	args := os.Args[index+1:]
	if len(args) == 0 {
		os.Exit(2)
	}
	switch args[0] {
	case "print":
		fmt.Print(strings.Join(args[1:], " "))
	case "print-env":
		for _, e := range os.Environ() {
			fmt.Println(e)
		}
	case "write-large":
		n := 100000
		if len(args) > 1 {
			fmt.Sscanf(args[1], "%d", &n)
		}
		buf := make([]byte, n)
		fmt.Print(string(buf))
	case "fork-sleep":
		if len(args) > 1 {
			cmd := exec.Command(os.Args[0], "-test.run=^TestResolverCmdHelperProcess$", "--", "sleep", args[1])
			cmd.Stdout = os.Stdout
			cmd.Start()
		}
	case "echo-args":
		fmt.Print(strings.Join(args[1:], "\n"))
	case "sleep":
		d, err := time.ParseDuration(args[1])
		if err != nil {
			os.Exit(2)
		}
		time.Sleep(d)
	case "fail-stderr":
		fmt.Fprint(os.Stderr, strings.Join(args[1:], " "))
		os.Exit(3)
	case "touch":
		if len(args) < 2 {
			os.Exit(2)
		}
		if err := os.WriteFile(args[1], []byte("ran"), 0o600); err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
	default:
		os.Exit(2)
	}
	os.Exit(0)
}
