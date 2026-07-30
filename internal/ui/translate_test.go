package ui

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"softether-tui/internal/i18n"
)

// TestEnCatalogCoversAllSourceStrings statically scans every tr("...") call
// in this package's source (excluding catalog_en.go and tests) and fails if
// enCatalog is missing an entry for it. This guards against typos or
// omissions in the mechanical retrofit that wrapped every hardcoded
// Japanese UI string in tr(...): a miss silently falls back to Japanese in
// English mode rather than failing loudly, so this test is the backstop.
func TestEnCatalogCoversAllSourceStrings(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}

	var missing []string
	fset := token.NewFileSet()
	for _, path := range files {
		if filepath.Base(path) == "catalog_en.go" || strings.HasSuffix(path, "_test.go") {
			continue
		}

		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			ident, ok := call.Fun.(*ast.Ident)
			if !ok || ident.Name != "tr" || len(call.Args) != 1 {
				return true
			}
			lit, ok := call.Args[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(lit.Value)
			if err != nil {
				t.Errorf("%s: could not unquote %s: %v", path, lit.Value, err)
				return true
			}
			if _, ok := enCatalog[value]; !ok {
				missing = append(missing, path+": "+value)
			}
			return true
		})
	}

	if len(missing) > 0 {
		t.Errorf("enCatalog is missing %d translation(s):\n  %s", len(missing), strings.Join(missing, "\n  "))
	}
}

// TestTranslate covers both directions of t(): passthrough under JA, and
// catalog lookup (with safe fallback to the Japanese source for a miss)
// under EN.
func TestTranslate(t *testing.T) {
	defer SetLang(i18n.EN)

	SetLang(i18n.JA)
	if got := tr("読み込み中..."); got != "読み込み中..." {
		t.Errorf("tr() under ja = %q, want unchanged Japanese source", got)
	}

	SetLang(i18n.EN)
	if got := tr("読み込み中..."); got != "Loading..." {
		t.Errorf("tr() under en = %q, want catalog translation", got)
	}
	if got := tr("no such key in catalog"); got != "no such key in catalog" {
		t.Errorf("tr() for an unknown key = %q, want the source string unchanged", got)
	}
}
