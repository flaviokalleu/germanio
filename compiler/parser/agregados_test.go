package parser

import (
	"reflect"
	"strings"
	"testing"
)

const aggBase = `crie sistema x

usuarios
    tem
        email obrigatório e único
        senha

tenha login

tenha papeis
    leitor 10

pedidos
    tem
        nome obrigatório
        itens
        notas
        membros com papel
    acesso
        leitor
            ver

itens
    tem
        quantidade inteiro
        preco dinheiro min 0
        titulo
        segredo_interno inteiro oculto
    começa aberto
    pode
        fechar

notas
    tem
        texto
`

// Aggregates (GEP 0047): the block and the flat form say the same; counts,
// sums, states, names and zerar resolve to the record's numbers.
func TestAgregadosFormas(t *testing.T) {
	block := aggBase + `
pedidos
    indicadores
        total de itens
        total de itens fechados
        soma da quantidade dos itens como unidades
        soma do preco dos itens abertos
    pode
        zerar unidades
`
	flat := aggBase + `
pedido mostra total de itens
pedido mostra o total de itens fechados
pedido mostra a soma da quantidade dos itens como unidades
pedido mostra soma do preco dos itens abertos
pedido pode zerar unidades
`
	a, b := resolved(t, block), resolved(t, flat)
	if !reflect.DeepEqual(canonical(t, a), canonical(t, b)) {
		t.Fatalf("bloco e frase plana diferem:\n%v\n%v", a.Entities["pedido"].Aggregates, b.Entities["pedido"].Aggregates)
	}
	ags := a.Entities["pedido"].Aggregates
	if len(ags) != 4 {
		t.Fatalf("quatro números: %+v", ags)
	}
	want := []struct{ name, field, state string }{
		{"total_de_itens", "", ""},
		{"total_de_itens_fechados", "", "fechado"},
		{"unidades", "quantidade", ""},
		{"soma_do_preco_dos_itens_abertos", "preco", "aberto"},
	}
	for i, w := range want {
		ag := ags[i]
		if ag.Name != w.name || ag.Field != w.field || ag.State != w.state || ag.Of != "item" || ag.Via != "pedido_id" {
			t.Fatalf("número %d: %+v (esperado %+v)", i, ag, w)
		}
	}
	if !ags[2].Reset || ags[0].Reset {
		t.Fatalf("só unidades pode ser zerada: %+v", ags)
	}
	if ags[2].Label != "Unidades" || ags[0].Label != "Total de itens" {
		t.Fatalf("rótulos: %q %q", ags[2].Label, ags[0].Label)
	}
}

// Every mistake says what, where, why and how to fix it.
func TestAgregadosErros(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		{"pedidos\n    indicadores\n", "a seção indicadores está vazia"},
		{"pedidos\n    indicadores\n        media do preco dos itens\n", "não é um indicador"},
		{"pedidos\n    indicadores\n        total de planetas\n", `não conheço "planetas"`},
		{"pedidos\n    indicadores\n        total de usuarios\n", "usuarios não pertence a pedido"},
		{"pedidos\n    indicadores\n        total de itens azuis\n", `"azuis" não é um estado de itens`},
		{"pedidos\n    indicadores\n        total de notas lidas\n", "notas não tem estados"},
		{"pedidos\n    indicadores\n        soma do titulo dos itens\n", "só números se somam"},
		{"pedidos\n    indicadores\n        soma do peso dos itens\n", `itens não tem o campo "peso"`},
		{"pedidos\n    indicadores\n        soma do segredo_interno dos itens\n", "a soma revelaria os valores"},
		{"pedidos\n    indicadores\n        total de itens \"Itens\"\n", "dê o nome com como"},
		{"pedidos\n    indicadores\n        total de itens como nome\n", "já tem um campo nome"},
		{"pedidos\n    indicadores\n        total de itens como n\n        soma da quantidade dos itens como n\n", "já mostra um número chamado n"},
		{"pedidos\n    indicadores\n        total de itens\n    pode\n        zerar total de itens\n", "uma contagem não se zera"},
		{"pedidos\n    indicadores\n        soma do preco dos itens\n    pode\n        zerar soma do preco dos itens\n", "tem min 0"},
		{"pedidos\n    pode\n        zerar unidades\n", "não mostra um número chamado unidades"},
		{"pedidos\n    indicadores\n        soma da quantidade dos itens abertos como abertas\n    pode\n        zerar abertas\n", "zere uma soma sem estado"},
		{"mostra total de itens\n", "não entendi a linha"},
	} {
		err := resolveErr(aggBase + "\n" + c.src)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Fatalf("%q: esperado %q, veio %v", c.src, c.want, err)
		}
	}
	// zerar records an entry with only the summed value: other required
	// fields make it impossible, and the error says which
	src := strings.Replace(aggBase, "        titulo\n", "        titulo obrigatório\n", 1) + "\npedidos\n    indicadores\n        soma da quantidade dos itens como unidades\n    pode\n        zerar unidades\n"
	if err := resolveErr(src); err == nil || !strings.Contains(err.Error(), "titulo é obrigatório") {
		t.Fatalf("zerar com outro campo obrigatório: %v", err)
	}
}

// Pairs (GEP 0048): block and flat forms; the record must name the data twice.
func TestParUnicoFormas(t *testing.T) {
	src := aggBase + `
ligacoes
    singular ligacao
    pertence a
        pedido como relacionado
`
	block := src + "\npedidos\n    tem\n        ligacoes\n\nligacoes\n    regras\n        única por par de pedidos\n"
	flat := src + "\npedido tem ligacoes\nligacao é única por par de pedidos\n"
	every := src + "\npedido tem ligacoes\ncada ligacao é única por par de pedidos\n"
	a, b, c := resolved(t, block), resolved(t, flat), resolved(t, every)
	if !reflect.DeepEqual(canonical(t, a), canonical(t, b)) || !reflect.DeepEqual(canonical(t, a), canonical(t, c)) {
		t.Fatal("bloco e frase plana diferem")
	}
	p := a.Entities["ligacao"].Pair
	if p == nil || p.Owner != "pedido_id" || p.Other != "relacionado_id" || p.Of != "pedido" {
		t.Fatalf("par: %+v", p)
	}
	if got := a.Entities["ligacao"].Model.Pairs; len(got) != 1 || got[0] != [2]string{"pedido_id", "relacionado_id"} {
		t.Fatalf("o modelo guarda o par para o banco: %v", got)
	}
	for _, c := range []struct{ src, want string }{
		{"\nitens\n    regras\n        único por par de pedidos\n", "aponta para pedido por 1 campo(s)"},
		{"\nitens\n    regras\n        único por par de planetas\n", `não conheço "planetas"`},
		{"\nitens\n    regras\n        único por par de\n", "frase não reconhecida"},
	} {
		if err := resolveErr(aggBase + c.src); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Fatalf("%q: esperado %q, veio %v", c.src, c.want, err)
		}
	}
}
