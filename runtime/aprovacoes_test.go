package runtime

import (
	"strings"
	"testing"
)

// GEP 0026 num domínio sem código: uma despesa só é paga depois de duas
// aprovações de pessoas que não são o autor; as outras ações não esperam.
func TestMinimoDeAprovacoes(t *testing.T) {
	_, first := loadApp(t, "testdata/intencao/despesas.ge")
	people := map[string]*client{}
	for _, n := range []string{"ana", "bia", "caio"} {
		_, c := first.fresh(t)
		c.expect("POST", "/cadastro", map[string]any{"nome": n, "email": n + "@x.com", "senha": "senha-" + n + "1"}, 201)
		c.csrf = csrfFromCookie(t, c)
		people[n] = c
	}
	ana, bia, caio := people["ana"], people["bia"], people["caio"]
	d := ana.expect("POST", "/_ge/api/despesas", map[string]any{"descricao": "Hotel"}, 201)
	base := "/_ge/api/despesas/" + itoa(int(d["id"].(float64)))

	if got := ana.expect("GET", base, nil, 200); got["aprovacoes_necessarias"] != float64(2) || got["aprovacoes_faltando"] != float64(2) {
		t.Fatalf("aprovações no registro: %v", got)
	}
	// a aprovação do autor não conta
	ana.expect("POST", base+"/aprovar", nil, 200)
	code, out, _ := bia.do("POST", base+"/pagar", nil)
	if msg, _ := out["message"].(string); code != 405 || !strings.Contains(msg, "precisa de 2 aprovações para pagar e tem 0") || !strings.Contains(msg, "Faltam 2") {
		t.Fatalf("recusa sem aprovações: %d %v", code, out)
	}
	bia.expect("POST", base+"/aprovar", nil, 200)
	bia.expect("POST", base+"/aprovar", nil, 200) // aprovar de novo não conta duas vezes
	bia.expect("POST", base+"/pagar", nil, 405)
	if got := bia.expect("GET", base, nil, 200); got["aprovacoes_faltando"] != float64(1) || got["estado"] != "aberta" {
		t.Fatalf("depois de uma aprovação: %v", got)
	}
	caio.expect("POST", base+"/aprovar", nil, 200)
	paid := caio.expect("POST", base+"/pagar", nil, 200)
	if paid["estado"] != "paga" {
		t.Fatalf("pagar com duas aprovações: %v", paid)
	}

	// outra ação do mesmo dado não espera aprovação
	d2 := ana.expect("POST", "/_ge/api/despesas", map[string]any{"descricao": "Táxi"}, 201)
	bia.expect("POST", "/_ge/api/despesas/"+itoa(int(d2["id"].(float64)))+"/arquivar", nil, 200)
}
