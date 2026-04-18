// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package ongc

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// TestObjectDocMentionsReadOnly locks in the B5 invariant: the Object type's
// doc comment must warn callers that returned pointers are read-only.
// Without this warning, users who mutate fields silently poison every
// subsequent catalog lookup for the same object.
//
// Guards against accidental deletion of the warning during future refactors.
func TestObjectDocMentionsReadOnly(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "ongc.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parsing ongc.go: %v", err)
	}

	var objectDoc string
	ast.Inspect(file, func(n ast.Node) bool {
		gen, ok := n.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			return true
		}
		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != "Object" {
				continue
			}
			if gen.Doc != nil {
				objectDoc = gen.Doc.Text()
			}
		}
		return true
	})

	if objectDoc == "" {
		t.Fatal("Object type has no doc comment")
	}
	want := []string{"read-only", "mutat"}
	for _, w := range want {
		if !strings.Contains(strings.ToLower(objectDoc), w) {
			t.Errorf("Object doc missing the %q warning. doc:\n%s", w, objectDoc)
		}
	}
}
