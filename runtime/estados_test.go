package runtime

import "testing"

// Testes normativos de estados e transições (docs/INTENCAO.md), num domínio
// de chamados: o mecanismo não depende do nome do dado.
func TestEstadosETransicoes(t *testing.T) {
	_, ana := loadApp(t, "testdata/intencao/chamados.ge")
	me := ana.expect("POST", "/cadastro", map[string]any{"nome": "Ana", "email": "ana@x.com", "senha": "senha-da-ana"}, 201)
	ana.csrf = csrfFromCookie(t, ana)
	_, bia := ana.fresh(t)
	bia.expect("POST", "/cadastro", map[string]any{"nome": "Bia", "email": "bia@x.com", "senha": "senha-da-bia"}, 201)
	bia.csrf = csrfFromCookie(t, bia)

	c := ana.expect("POST", "/_ge/api/chamados", map[string]any{"titulo": "Impressora"}, 201)
	path := "/_ge/api/chamados/" + itoa(int(c["id"].(float64)))
	if c["estado"] != "aberto" {
		t.Fatalf("inicia aberto: %v", c)
	}
	// Editar não muda estado nem carimbos.
	if e := ana.expect("PATCH", path, map[string]any{"titulo": "Impressora 2", "estado": "fechado", "fechado_em": "2020-01-01"}, 200); e["estado"] != "aberto" || e["fechado_em"] != nil || e["titulo"] != "Impressora 2" {
		t.Fatalf("editar mudou estado ou carimbo: %v", e)
	}
	// Sem permissão: recusado, nada muda.
	bia.expect("POST", path+"/fechar", nil, 403)
	if got := ana.expect("GET", path, nil, 200); got["estado"] != "aberto" {
		t.Fatalf("fechar sem permissão mudou o estado: %v", got)
	}
	// Fechar registra quem e quando.
	f := ana.expect("POST", path+"/fechar", nil, 200)
	if f["estado"] != "fechado" || f["fechado_em"] == nil || f["fechado_por_id"] != me["id"] {
		t.Fatalf("fechar: %v", f)
	}
	ana.expect("POST", path+"/fechar", nil, 400) // já fechado
	// Reabrir volta ao inicial e limpa os metadados.
	r := ana.expect("POST", path+"/reabrir", nil, 200)
	if r["estado"] != "aberto" || r["fechado_em"] != nil || r["fechado_por_id"] != nil {
		t.Fatalf("reabrir: %v", r)
	}
}

// Busca geral fora do GitLab: o tipo escolhe a coleção; tipo desconhecido é recusado.
func TestBuscaGeralGenerica(t *testing.T) {
	_, ana := loadApp(t, "testdata/intencao/chamados.ge")
	ana.expect("POST", "/cadastro", map[string]any{"nome": "Ana", "email": "ana@x.com", "senha": "senha-da-ana"}, 201)
	ana.csrf = csrfFromCookie(t, ana)
	ana.expect("POST", "/_ge/api/chamados", map[string]any{"titulo": "Impressora sem tinta"}, 201)
	ana.expect("POST", "/_ge/api/chamados", map[string]any{"titulo": "Rede lenta"}, 201)
	_, _, raw := ana.do("GET", "/_ge/api/busca?tipo_busca=chamados&q=impressora", nil)
	if l := decodeList(t, raw); len(l) != 1 || l[0]["titulo"] != "Impressora sem tinta" {
		t.Fatalf("busca: %s", raw)
	}
	ana.expect("GET", "/_ge/api/busca?tipo_busca=pedidos&q=x", nil, 400)
}
