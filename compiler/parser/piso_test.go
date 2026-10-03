package parser

import (
	"reflect"
	"strings"
	"testing"
)

// GEP 0050: `regras › não pode ficar com <soma> negativo` is the flat
// `<dado> não pode ficar com <soma> negativo`, and marks the sum.
func TestSomaNuncaNegativaFormas(t *testing.T) {
	block := aggBase + `
pedidos
    indicadores
        soma da quantidade dos itens como unidades
    regras
        não pode ficar com unidades negativas
`
	flat := aggBase + `
pedido mostra soma da quantidade dos itens como unidades
pedido nao pode ficar com unidades negativo
`
	a, b := resolved(t, block), resolved(t, flat)
	if !reflect.DeepEqual(canonical(t, a), canonical(t, b)) {
		t.Fatalf("bloco e frase plana diferem:\n%+v\n%+v", a.Entities["pedido"].Aggregates[0], b.Entities["pedido"].Aggregates[0])
	}
	if ag := a.Entities["pedido"].Aggregates[0]; !ag.NonNegative || ag.Name != "unidades" {
		t.Fatalf("a soma não ficou marcada: %+v", ag)
	}
}

func TestSomaNuncaNegativaErros(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		{"pedidos\n    regras\n        não pode ficar com saldo negativo\n", "não mostra um número chamado saldo"},
		{"pedidos\n    indicadores\n        soma da quantidade dos itens como unidades\n    regras\n        não pode ficar com saldo negativo\n", "use uma das somas de pedidos: unidades"},
		{"pedidos\n    indicadores\n        total de itens\n    regras\n        não pode ficar com total de itens negativo\n", "uma contagem nunca é negativa"},
		{"pedidos\n    indicadores\n        soma da quantidade dos itens abertos como abertas\n    regras\n        não pode ficar com abertas negativas\n", "use a regra numa soma sem estado"},
		{"pedidos\n    regras\n        não pode ficar com unidades\n", "não pode ficar com tempo gasto negativo"},
	} {
		err := resolveErr(aggBase + "\n" + c.src)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Fatalf("%q: esperado %q, veio %v", c.src, c.want, err)
		}
	}
}
