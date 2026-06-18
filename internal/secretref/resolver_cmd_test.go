package secretref

import (
	"context"
	"fmt"
	"os"
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
