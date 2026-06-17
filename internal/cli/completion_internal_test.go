package cli

import (
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestCompletionScriptsExposeSameGeneratedSurface(t *testing.T) {
	t.Parallel()

	wantTokens := completionSurfaceTokensForTest()
	for _, shell := range completionShells {
		shell := shell
		t.Run(shell, func(t *testing.T) {
			t.Parallel()

			script, err := completionScript(shell)
			if err != nil {
				t.Fatalf("completionScript(%q) error = %v, want nil", shell, err)
			}

			var missing []string
			for _, token := range wantTokens {
				if !completionScriptContainsToken(script, token) {
					missing = append(missing, token)
				}
			}
			if len(missing) > 0 {
				t.Errorf("completionScript(%q) missing %d source-of-truth token(s): %s", shell, len(missing), strings.Join(missing, ", "))
			}
		})
	}
}

// TestCompletionScriptsIncludeURLLookup guards the zia-only url-lookup
// diagnostic verb. It is dispatched directly in app.go (not a catalog
// resource), so resourceNames omits it; every shell's completion block must
// still offer it after "zia".
func TestCompletionScriptsIncludeURLLookup(t *testing.T) {
	t.Parallel()

	for _, shell := range completionShells {
		shell := shell
		t.Run(shell, func(t *testing.T) {
			t.Parallel()

			script, err := completionScript(shell)
			if err != nil {
				t.Fatalf("completionScript(%q) error = %v, want nil", shell, err)
			}
			if !completionScriptContainsToken(script, urlLookupCommandName) {
				t.Errorf("completionScript(%q) missing %q", shell, urlLookupCommandName)
			}
		})
	}
}

// TestCompletionScriptsOfferLogLevelValues asserts every shell completes the
// --log-level flag's values (off/error/warn/info/debug), not just the flag name.
func TestCompletionScriptsOfferLogLevelValues(t *testing.T) {
	t.Parallel()

	for _, shell := range completionShells {
		shell := shell
		t.Run(shell, func(t *testing.T) {
			t.Parallel()

			script, err := completionScript(shell)
			if err != nil {
				t.Fatalf("completionScript(%q) error = %v, want nil", shell, err)
			}
			want := words(completionLogLevels)
			if shell == "powershell" {
				want = powershellArray(completionLogLevels)
			}
			if !strings.Contains(script, want) {
				t.Errorf("completionScript(%q) missing log-level value list %q", shell, want)
			}
			// The values must be wired to the --log-level flag specifically.
			if !strings.Contains(script, "log-level") {
				t.Errorf("completionScript(%q) does not reference log-level", shell)
			}
		})
	}
}

func completionSurfaceTokensForTest() []string {
	seen := map[string]struct{}{}
	add := func(values ...string) {
		for _, value := range values {
			if strings.TrimSpace(value) == "" {
				continue
			}
			seen[value] = struct{}{}
		}
	}

	add(completionFlags...)
	add(completionDumpFlags...)
	add(completionFormats...)
	add(completionRedaction...)
	add(completionColors...)
	add(completionShells...)
	add(completionProductValues()...)
	add(completionCommandNames()...)
	add(operationNames()...)
	add(allResourceNames()...)
	add(dumpResourceNames()...)

	out := make([]string, 0, len(seen))
	for value := range seen {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func completionScriptContainsToken(script, token string) bool {
	if tokenBoundaryRE(token).MatchString(script) {
		return true
	}
	if strings.HasPrefix(token, "--") {
		// fish spells long flags as "-l name" instead of "--name".
		name := regexp.QuoteMeta(strings.TrimPrefix(token, "--"))
		return regexp.MustCompile(`(^|\s)-l\s+` + name + `(\s|$)`).MatchString(script)
	}
	return false
}

func tokenBoundaryRE(token string) *regexp.Regexp {
	// Treat resource-name characters as part of tokens so "locations" does not
	// pass only because "zia/locations" is present.
	boundary := `A-Za-z0-9_/\-,`
	return regexp.MustCompile(`(^|[^` + boundary + `])` + regexp.QuoteMeta(token) + `($|[^` + boundary + `])`)
}
