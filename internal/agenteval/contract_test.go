package agenteval_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strconv"
	"testing"

	"github.com/dvmrry/zscalerctl/internal/agenteval"
)

// TestErrorKindEnumMatchesBinary asserts the contract's ErrorKind enum is the
// EXACT set of strings the binary's errorKind() returns (§5.2, folding review
// finding 1). The grader compares against the literal envelope "kind" string,
// so if the binary gains, drops, or renames a kind without this enum being
// updated in lockstep, the eval would grade against a vocabulary the binary no
// longer emits. This gate reds the build instead.
//
// It does NOT import package main (a `main` package can't be imported). Instead
// it parses cmd/zscalerctl/main.go with go/parser + go/ast, finds func
// errorKind, and collects every string literal it returns.
func TestErrorKindEnumMatchesBinary(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	mainPath := filepath.Join(root, "cmd", "zscalerctl", "main.go")

	binaryKinds := errorKindReturnsFromBinary(t, mainPath)
	contractKinds := contractErrorKinds()

	missingFromContract := difference(binaryKinds, contractKinds)
	extraInContract := difference(contractKinds, binaryKinds)

	for _, k := range missingFromContract {
		t.Errorf("errorKind() in %s returns %q, but contract.go's ErrorKind enum has no such value; add it (the binary gained/renamed a kind without the eval being updated)", mainPath, k)
	}
	for _, k := range extraInContract {
		t.Errorf("contract.go declares ErrorKind %q, but errorKind() in %s never returns it; remove or fix it (the eval would grade against a kind the binary doesn't emit)", k, mainPath)
	}
}

// errorKindReturnsFromBinary parses main.go and returns the set of distinct
// string-literal values returned by func errorKind. It fails loudly if the
// function is missing, has no string-literal returns, or returns a non-literal
// (e.g. a computed string), since any of those would mean the static set this
// gate compares against is incomplete.
func errorKindReturnsFromBinary(t *testing.T, mainPath string) []string {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, mainPath, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", mainPath, err)
	}

	var fn *ast.FuncDecl
	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		// Top-level func errorKind (no receiver).
		if fd.Recv == nil && fd.Name.Name == "errorKind" {
			fn = fd
			break
		}
	}
	if fn == nil {
		t.Fatalf("func errorKind not found in %s; the contract gate can't locate the binary's kind vocabulary", mainPath)
	}

	kinds := map[string]bool{}
	var inspectErr error
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		ret, ok := n.(*ast.ReturnStmt)
		if !ok {
			return true
		}
		for _, res := range ret.Results {
			lit, ok := res.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				inspectErr = &nonLiteralReturnError{pos: fset.Position(res.Pos()).String()}
				continue
			}
			val, err := strconv.Unquote(lit.Value)
			if err != nil {
				t.Fatalf("unquote return literal %q in errorKind: %v", lit.Value, err)
			}
			kinds[val] = true
		}
		return true
	})
	if inspectErr != nil {
		t.Fatalf("errorKind returns a non-string-literal value (%v); this gate only handles literal kinds and would otherwise compare an incomplete set", inspectErr)
	}
	if len(kinds) == 0 {
		t.Fatalf("errorKind in %s returned no string literals; the parse found the function but no kind values", mainPath)
	}

	return setToSortedSlice(kinds)
}

// nonLiteralReturnError flags an errorKind return whose value is not a string
// literal, so the failure message can point at the source position.
type nonLiteralReturnError struct{ pos string }

func (e *nonLiteralReturnError) Error() string { return "non-string-literal return at " + e.pos }

// contractErrorKinds returns the set of declared ErrorKind constant values from
// the contract. These are the exported constants in contract.go; listing them
// here keeps this gate a pure two-set comparison. If contract.go adds a new
// ErrorKind constant, it must be added here too — and a constant added here
// without a matching errorKind() return reds the build via extraInContract.
func contractErrorKinds() []string {
	all := []agenteval.ErrorKind{
		agenteval.ErrorKindUsage,
		agenteval.ErrorKindPartialDump,
		agenteval.ErrorKindNotFound,
		agenteval.ErrorKindMissingCredentials,
		agenteval.ErrorKindInvalidResourceID,
		agenteval.ErrorKindUnsupportedResource,
		agenteval.ErrorKindLiveAccessFailed,
		agenteval.ErrorKindInvalidProxyConfig,
		agenteval.ErrorKindInvalidConfig,
		agenteval.ErrorKindInternal,
	}
	set := map[string]bool{}
	for _, k := range all {
		set[string(k)] = true
	}
	return setToSortedSlice(set)
}

func setToSortedSlice(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// difference returns the elements of a not present in b.
func difference(a, b []string) []string {
	inB := map[string]bool{}
	for _, k := range b {
		inB[k] = true
	}
	var out []string
	for _, k := range a {
		if !inB[k] {
			out = append(out, k)
		}
	}
	return out
}
