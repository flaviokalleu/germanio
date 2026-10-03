package parser

import (
	"strings"
	"testing"

	"github.com/flaviokalleu/germanio/compiler/ast"
)

const pessoas = "crie sistema X\n\nusuarios\n    tem\n        nome\n        email obrigatório e único\n        senha min 8\n\ntenha login\n"

// chave pública (GEP 0032, em teste): one type, a derived fingerprint that
// carries the uniqueness, a key that never changes; two keys are an error.
func TestChavePublicaTipo(t *testing.T) {
	app := resolved(t, pessoas+"\nmaquinas\n    tem\n        chave chave pública obrigatório e único\n")
	m := app.Entities["maquina"].Model
	key, fp := fieldByNameAST(m, "chave"), fieldByNameAST(m, "impressao_digital")
	if key == nil || key.Type != ast.FieldChavePublica || key.Unique || !key.Immutable || !key.Required {
		t.Fatalf("campo da chave: %+v", key)
	}
	if fp == nil || !fp.System || !fp.Unique {
		t.Fatalf("impressão digital: %+v", fp)
	}
	err := resolveErr(pessoas + "\nmaquinas\n    tem\n        a chave pública\n        b chave pública\n")
	if err == nil || !strings.Contains(err.Error(), "duas chaves públicas") {
		t.Fatalf("duas chaves: %v", err)
	}
}
