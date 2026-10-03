package parser

import (
	"reflect"
	"strings"
	"testing"
)

// GEP 0026: o mínimo de aprovações para uma ação, num domínio sem código
// (despesas pagas depois de aprovadas).
const despesas = base + `
despesas
    tem
        descricao obrigatório
        autor
    começa aberta
    pode
        pagar
    recebe aprovações
`

func TestMinimoDeAprovacoes(t *testing.T) {
	block := resolved(t, despesas+"    regras\n        precisa de duas aprovações para pagar\n")
	if n := block.Entities["despesa"].ApprovalsNeeded["pagar"]; n != 2 {
		t.Fatalf("mínimo no bloco: %d", n)
	}
	flatForm := resolved(t, despesas+"\ndespesas precisam de 2 aprovações para pagar\n")
	if !reflect.DeepEqual(canonical(t, block).(map[string]any)["Entities"], canonical(t, flatForm).(map[string]any)["Entities"]) {
		t.Fatal("o bloco e a frase plana deveriam produzir o mesmo dado")
	}
	atLeast := resolved(t, despesas+"    regras\n        precisa de pelo menos 3 aprovações para pagar\n")
	if n := atLeast.Entities["despesa"].ApprovalsNeeded["pagar"]; n != 3 {
		t.Fatalf("pelo menos: %d", n)
	}
	one := resolved(t, despesas+"\ntoda despesa precisa de uma aprovação para pagar\n")
	if n := one.Entities["despesa"].ApprovalsNeeded["pagar"]; n != 1 {
		t.Fatalf("uma aprovação: %d", n)
	}
}

func TestMinimoDeAprovacoesErros(t *testing.T) {
	noApprovals := strings.Replace(despesas, "    recebe aprovações\n", "", 1)
	cases := []struct{ name, src, want string }{
		{"sem receber aprovações", noApprovals + "\ndespesas precisam de 2 aprovações para pagar\n", "Declare também: despesa recebe aprovações"},
		{"ação desconhecida", despesas + "\ndespesas precisam de 2 aprovações para voar\n", "voar não é uma ação de despesas (ações: pagar)"},
		{"sem número", despesas + "\ndespesas precisam de aprovações para pagar\n", "não é um número de aprovações"},
		{"zero", despesas + "\ndespesas precisam de 0 aprovações para pagar\n", "entre 1 e 100"},
		{"sem ação", despesas + "\ndespesas precisam de 2 aprovações\n", "não diz para quê"},
		{"dois números", despesas + "\ndespesas precisam de 2 aprovações para pagar\ndespesas precisam de 3 aprovações para pagar\n", "declare um só número"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := resolveErr(c.src)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("esperado %q, veio: %v", c.want, err)
			}
		})
	}
}
