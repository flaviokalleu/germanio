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

// tenha confirmação de e-mail (GEP 0031) and tenha autenticação em dois
// fatores (GEP 0032): each implies the login and adds its system field;
// confirmation needs an e-mail field; a different wording is an error.
func TestFrasesDeIdentidade(t *testing.T) {
	base := "crie sistema X\n\nusuarios\n    tem\n        nome\n        email obrigatório e único\n        senha min 8\n\n"
	app := resolved(t, base+"tenha confirmação de e-mail\ntenha autenticação em dois fatores\n")
	if app.Login == nil || !app.Login.Confirmation || !app.Login.TwoFactor {
		t.Fatalf("login: %+v", app.Login)
	}
	m := app.Entities["usuario"].Model
	for _, name := range []string{"email_confirmado", "dois_fatores"} {
		if f := fieldByNameAST(m, name); f == nil || !f.System || !f.Private || f.Type != ast.FieldBooleano {
			t.Fatalf("%s: %+v", name, f)
		}
	}
	if f := fieldByNameAST(m, "email_confirmado"); f.DefaultValue != true {
		t.Fatalf("quem não se cadastrou nasce confirmado: %+v", f)
	}
	if app := resolved(t, base+"tenha login\ntenha confirmacao de email\n"); !app.Login.Confirmation {
		t.Fatal("sem acentos")
	}
	noMail := "crie sistema X\n\nusuarios\n    tem\n        nome\n        senha min 8\n\ntenha confirmação de e-mail\n"
	if err := resolveErr(noMail); err == nil || !strings.Contains(err.Error(), "campo de e-mail") {
		t.Fatalf("sem campo de e-mail: %v", err)
	}
	if err := resolveErr(base + "tenha confirmação de telefone\n"); err == nil || !strings.Contains(err.Error(), "tenha confirmação de e-mail") {
		t.Fatalf("frase diferente: %v", err)
	}
}
