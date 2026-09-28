package parser

import (
	"strings"
	"testing"
)

func TestEnderecoResolvido(t *testing.T) {
	prog := parse(t, `crie sistema x

tenha usuarios, grupos, projetos

usuario tem
    username obrigatório e único
    senha

tenha login
login usa username

grupo tem
    caminho obrigatório
    subgrupos
    endereço dentro do grupo pai

projeto tem
    caminho obrigatório
    criador
    repositório
    endereço dentro do grupo ou do criador

projeto pertence a grupo opcional
`)
	if err := ResolveIntent(prog); err != nil {
		t.Fatal(err)
	}
	g, p := prog.App.Entities["grupo"], prog.App.Entities["projeto"]
	if g.Address == nil || len(g.Address.Within) != 1 || g.Address.Within[0].Field != "pai_id" {
		t.Fatalf("grupo: %+v", g.Address)
	}
	w := p.Address.Within
	if len(w) != 2 || w[0].Field != "grupo_id" || w[0].Entity != "grupo" || w[1].Field != "criador_id" || w[1].Entity != "usuario" {
		t.Fatalf("projeto: %+v", w)
	}
	if p.RepoKey != "endereco" {
		t.Fatalf("repositório deveria ser endereçado pelo endereço: %s", p.RepoKey)
	}
}

func TestEnderecoErros(t *testing.T) {
	for _, c := range []struct{ body, want string }{
		{"grupo tem\n    nome\n    endereço dentro do grupo pai\n", "declare também o campo caminho"},
		{"grupo tem\n    caminho\n    endereço dentro do grupo pai\n", `precisa de "grupo tem subgrupos"`},
		{"grupo tem\n    caminho\n    endereço dentro do projeto\n", `precisa de "grupo pertence a projeto"`},
		{"grupo tem\n    caminho\n    endereço dentro da empresa\n", "não conheço esse dado"},
	} {
		prog := parse(t, "crie sistema x\n\ntenha grupos, projetos\n\n"+c.body)
		err := ResolveIntent(prog)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Fatalf("%q: esperado %q, veio %v", c.body, c.want, err)
		}
	}
}
