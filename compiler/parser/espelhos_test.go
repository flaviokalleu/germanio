package parser

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

const mirrorOwner = "livros\n    tem\n        titulo\n        caminho único\n    tem repositório\n\n"

// GEP 0036: `espelham o repositório do <dono>` in the data's block is the
// same fact as the flat phrase, and the capability adds the fields every
// mirror needs.
func TestEspelhosEquivaleAFrasePlana(t *testing.T) {
	flat := base + "\n" + mirrorOwner + "copias\n    tem\n        url obrigatório\n    pertence a livro\n\ncopias espelham o repositório do livro\n"
	tree := base + "\n" + mirrorOwner + "copias\n    tem\n        url obrigatório\n    pertence a livro\n    espelham o repositório do livro\n"
	a, b := resolved(t, flat), resolved(t, tree)
	if !reflect.DeepEqual(canonical(t, a), canonical(t, b)) {
		fa, _ := json.MarshalIndent(canonical(t, a), "", " ")
		fb, _ := json.MarshalIndent(canonical(t, b), "", " ")
		t.Fatalf("formas diferentes:\n%s", diffHint(string(fa), string(fb)))
	}
	c := b.Entities["copia"]
	if c.Mirror == nil || c.Mirror.Owner != "livro" || c.Mirror.OwnerField == "" {
		t.Fatalf("espelho não resolvido: %+v", c.Mirror)
	}
	for name, want := range map[string]string{"sentido": "enum", "habilitado": "booleano", "credencial": "texto", "situacao": "enum", "ultimo_erro": "texto", "ultima_atualizacao": "texto", "ultimo_sucesso": "texto"} {
		f := fieldByNameAST(c.Model, name)
		if f == nil || string(f.Type) != want {
			t.Fatalf("campo %s: %+v", name, f)
		}
	}
	if f := fieldByNameAST(c.Model, "credencial"); !f.Hidden || !f.System {
		t.Fatal("a credencial precisa ser oculta e mantida pelo runtime")
	}
	if f := fieldByNameAST(c.Model, "sentido"); f.System || f.DefaultValue != "enviar" {
		t.Fatalf("sentido é escolhido pelas pessoas e começa em enviar: %+v", f)
	}
}

func TestEspelhosErros(t *testing.T) {
	cases := []struct {
		name, src string
		want      []string
	}{
		{"dono sem repositório", "tenha pastas\n\ncada pasta tem\n    nome\n\ncopias\n    tem\n        url\n    pertence a pasta\n    espelham o repositório da pasta\n", []string{"pastas não tem repositório", "tem repositório"}},
		{"sem url", mirrorOwner + "copias\n    tem\n        nome\n    pertence a livro\n    espelham o repositório do livro\n", []string{"falta em copias: url obrigatório"}},
		{"sem dono", mirrorOwner + "copias\n    tem\n        url\n    espelham o repositório do livro\n", []string{"pertence a livro"}},
		{"frase incompleta", mirrorOwner + "copias\n    tem\n        url\n    pertence a livro\n    espelham o livro\n", []string{"espelhos espelham o repositório do projeto", "Por quê"}},
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
