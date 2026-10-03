package parser

import (
	"strings"
	"testing"
)

// GEP 0049: which fields are kept encrypted at rest, without new syntax.
func TestSegredosQuaisCampos(t *testing.T) {
	src := base + "\n" + mirrorOwner + `avisos
    tem
        url obrigatório
        token oculto
        endereco url oculto
        tentativas inteiro oculto
        nota
        codigo segredo
    pertence a livro

copias
    tem
        url obrigatório
    pertence a livro
    espelham o repositório do livro
`
	app := resolved(t, src)
	a := app.Entities["aviso"]
	for name, want := range map[string]bool{"token": true, "endereco": true, "tentativas": false, "nota": false, "codigo": false} {
		if f := fieldByNameAST(a.Model, name); f == nil || f.Sealed != want {
			t.Fatalf("%s cifrado = %v, esperado %v", name, f != nil && f.Sealed, want)
		}
	}
	if f := fieldByNameAST(app.Entities["copia"].Model, "credencial"); !f.Sealed {
		t.Fatal("a credencial do espelho fica cifrada")
	}
	// fields the system keeps for itself (paths, logs) are not secrets
	for _, e := range app.Entities {
		for _, f := range e.Model.Fields {
			if f.System && f.Sealed && f.Name != "credencial" {
				t.Fatalf("%s.%s cifrado sem ser segredo", e.Singular, f.Name)
			}
		}
	}
}

func TestSegredosErros(t *testing.T) {
	cases := []struct {
		name, src string
		want      []string
	}{
		{"oculto e único", "avisos\n    tem\n        token oculto e único\n", []string{"oculto e único", "Por quê", "segredo"}},
		{"oculto com índice", "avisos\n    tem\n        token oculto índice\n", []string{"índice"}},
		{"filtrar por oculto", "avisos\n    tem\n        token oculto\n    permita\n        filtrar por token\n", []string{"permita filtrar avisos por token", "cifrado", "Como corrigir"}},
		{"página filtra por oculto", "avisos\n    tem\n        nome\n        token oculto\n\npágina Avisos\n    mostre avisos\n    filtros\n        token\n", []string{"filtra por token", "cifrado"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := resolveErr(base + "\n" + c.src)
			if err == nil {
				t.Fatalf("esperado erro para:\n%s", c.src)
			}
			for _, w := range c.want {
				if !strings.Contains(err.Error(), w) {
					t.Fatalf("mensagem sem %q:\n%v", w, err)
				}
			}
		})
	}
}
