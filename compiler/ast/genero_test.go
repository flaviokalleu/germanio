package ast

import "testing"

func TestFeminino(t *testing.T) {
	for _, c := range []struct {
		nome, estado string
		want         bool
	}{
		{"tarefa", "", true}, {"pedido", "", false}, {"cobrança", "", true},
		{"integração", "", true}, {"cidade", "", true}, {"mensagem", "", true},
		{"série", "", true}, {"produto", "", false}, {"sistema", "", false},
		{"dia", "", false}, {"problema", "", false}, {"cliente", "", false},
		{"categoria de despesa", "", true}, {"item de pedido", "", false},
		// the state the person wrote agrees with the name and decides
		{"issue", "aberta", true}, {"issue", "", false}, {"tarefa", "rascunho", true},
		{"pedido", "aberto", false}, {"conta", "ativa", true}, {"chamado", "fechado", false},
	} {
		if got := Feminino(c.nome, c.estado); got != c.want {
			t.Errorf("Feminino(%q, %q) = %v, esperado %v", c.nome, c.estado, got, c.want)
		}
	}
}

func TestNovoRotulo(t *testing.T) {
	if got := (&Entity{Label: "Tarefa", Initial: "aberta"}).NovoRotulo(); got != "Nova tarefa" {
		t.Errorf("tarefa: %q", got)
	}
	if got := (&Entity{Label: "Pedido"}).NovoRotulo(); got != "Novo pedido" {
		t.Errorf("pedido: %q", got)
	}
}
