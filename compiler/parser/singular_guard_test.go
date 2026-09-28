package parser

import "testing"

// The singular of a data is its storage identity (table, foreign keys,
// stored `recurso` values). Changing the rule renames storage of existing
// programs, so the current outputs — including the known wrong ones of G112
// (noun + adjective) — are fixed here. Changing them needs GEP 0018 (schema
// identity) decided first, not an edit of this table.
func TestSingularEstavel(t *testing.T) {
	cases := map[string]string{
		"clientes":            "cliente",
		"tokens_de_acesso":    "tokem_de_acesso", // `singular token de acesso` corrects it
		"itens_do_pedido":     "item_do_pedido",
		"merge_requests":      "merge_request",
		"contas_bancarias":    "contas_bancaria", // wrong (G112): kept until GEP 0018
		"notas_fiscais":       "notas_fiscal",    // wrong (G112)
		"branches_protegidas": "branches_protegida",
	}
	for plural, want := range cases {
		if got := Singular(plural); got != want {
			t.Errorf("Singular(%q) = %q, era %q: mudar o singular renomeia o armazenamento de programas existentes (G112, GEP 0018)", plural, got, want)
		}
	}
}
