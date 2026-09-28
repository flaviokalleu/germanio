package runtime

import (
	"encoding/json"
	"sync"
	"testing"
)

// Testes normativos de `todo time precisa ter pelo menos um dono`
// (docs/INTENCAO.md › Testes normativos), num domínio sem GitLab.

func equipes(t *testing.T) (*client, map[string]*client, map[string]float64) {
	_, admin := loadApp(t, "testdata/intencao/equipes.ge")
	people := map[string]*client{}
	ids := map[string]float64{}
	for _, n := range []string{"ana", "bia", "caio"} {
		_, c := admin.fresh(t)
		me := c.expect("POST", "/cadastro", map[string]any{"nome": n, "email": n + "@x.com", "senha": "senha-" + n + "1"}, 201)
		c.csrf = csrfFromCookie(t, c)
		people[n], ids[n] = c, me["id"].(float64)
	}
	admin.expect("POST", "/entrar", map[string]any{"login": "admin@equipes.test", "senha": "admin12345"}, 200)
	admin.csrf = csrfFromCookie(t, admin)
	return admin, people, ids
}

func memberOf(t *testing.T, c *client, base string, person float64) string {
	t.Helper()
	code, _, raw := c.do("GET", base+"/membros", nil)
	if code != 200 {
		t.Fatalf("membros: %d %s", code, raw)
	}
	for _, m := range decodeList(t, raw) {
		if m["pessoa_id"] == person {
			return itoa(int(m["id"].(float64)))
		}
	}
	t.Fatalf("membro %v não encontrado em %s", person, raw)
	return ""
}

func TestPapelMinimo(t *testing.T) {
	admin, p, ids := equipes(t)
	ana, bia := p["ana"], p["bia"]
	time1 := ana.expect("POST", "/_ge/api/times", map[string]any{"nome": "Núcleo"}, 201)
	base := "/_ge/api/times/" + itoa(int(time1["id"].(float64)))
	anaM := memberOf(t, ana, base, ids["ana"])

	// Um dono: não sai, não é removido, não é rebaixado — nem pelo administrador.
	ana.expect("POST", base+"/sair", nil, 400)
	ana.expect("DELETE", base+"/membros/"+anaM, nil, 400)
	ana.expect("PATCH", base+"/membros/"+anaM, map[string]any{"papel": "colaborador"}, 400)
	admin.expect("DELETE", base+"/membros/"+anaM, nil, 400)
	// Excluir a pessoa que é a última dona também preservaria um time sem dono.
	admin.expect("DELETE", "/_ge/api/usuarios/"+itoa(int(ids["ana"])), nil, 400)

	// Dois donos: um pode sair; adicionar outro permite remover o anterior.
	ana.expect("POST", base+"/membros", map[string]any{"pessoa_id": ids["bia"], "papel": "dono"}, 201)
	ana.expect("PATCH", base+"/membros/"+anaM, map[string]any{"papel": "colaborador"}, 200)
	biaM := memberOf(t, bia, base, ids["bia"])
	bia.expect("DELETE", base+"/membros/"+biaM, nil, 400) // Bia agora é a única dona
	bia.expect("POST", base+"/membros", map[string]any{"pessoa_id": ids["caio"], "papel": "dono"}, 201)
	bia.expect("POST", base+"/sair", nil, 204)

	// Excluir o próprio time não exige manter donos de algo que deixa de existir.
	p["caio"].expect("DELETE", base, nil, 204)
}

func TestPapelMinimoConcorrente(t *testing.T) {
	_, p, ids := equipes(t)
	ana, bia := p["ana"], p["bia"]
	time1 := ana.expect("POST", "/_ge/api/times", map[string]any{"nome": "Dupla"}, 201)
	base := "/_ge/api/times/" + itoa(int(time1["id"].(float64)))
	ana.expect("POST", base+"/membros", map[string]any{"pessoa_id": ids["bia"], "papel": "dono"}, 201)
	// Os dois donos saem ao mesmo tempo: exatamente um consegue.
	for round := 0; round < 1; round++ {
		var wg sync.WaitGroup
		codes := make([]int, 2)
		for i, c := range []*client{ana, bia} {
			wg.Add(1)
			go func(i int, c *client) { defer wg.Done(); codes[i], _, _ = c.do("POST", base+"/sair", nil) }(i, c)
		}
		wg.Wait()
		ok := 0
		for _, code := range codes {
			if code == 204 {
				ok++
			}
		}
		if ok != 1 {
			t.Fatalf("saídas simultâneas: %v (exatamente uma deveria passar)", codes)
		}
	}
}

func TestPapelMinimoComHeranca(t *testing.T) {
	_, p, ids := equipes(t)
	ana, bia := p["ana"], p["bia"]
	time1 := ana.expect("POST", "/_ge/api/times", map[string]any{"nome": "Plataforma"}, 201)
	tid := time1["id"].(float64)
	// Projeto dentro do time: os donos vêm do time; ninguém vira membro direto.
	proj := ana.expect("POST", "/_ge/api/projetos", map[string]any{"nome": "API", "time_id": tid}, 201)
	pbase := "/_ge/api/projetos/" + itoa(int(proj["id"].(float64)))
	if _, _, raw := ana.do("GET", pbase+"/membros", nil); len(decodeList(t, raw)) != 0 {
		t.Fatalf("projeto do time não deveria ter membro direto: %s", raw)
	}
	// Sair do projeto do time: continua com dono herdado.
	ana.expect("POST", pbase+"/membros", map[string]any{"pessoa_id": ids["bia"], "papel": "dono"}, 201)
	bia.expect("POST", pbase+"/sair", nil, 204)
	// Tirar o projeto do time o deixaria sem dono: recusado.
	ana.expect("PATCH", pbase, map[string]any{"time_id": nil}, 400)
	// Projeto pessoal: quem cria é dono direto; não sai sozinho.
	solo := bia.expect("POST", "/_ge/api/projetos", map[string]any{"nome": "Solo"}, 201)
	sbase := "/_ge/api/projetos/" + itoa(int(solo["id"].(float64)))
	bia.expect("POST", sbase+"/sair", nil, 400)
}

func decodeList(t *testing.T, raw string) []map[string]any {
	t.Helper()
	var list []map[string]any
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		t.Fatalf("lista: %v: %s", err, raw)
	}
	return list
}

// Deleting a person who takes part in something removes their memberships
// with them (what belongs to the record goes with it); it used to fail with
// a raw foreign-key error (G76).
func TestExcluirPessoaQueParticipa(t *testing.T) {
	admin, p, ids := equipes(t)
	ana := p["ana"]
	time1 := ana.expect("POST", "/_ge/api/times", map[string]any{"nome": "Núcleo"}, 201)
	base := "/_ge/api/times/" + itoa(int(time1["id"].(float64)))
	ana.expect("POST", base+"/membros", map[string]any{"pessoa_id": ids["bia"], "papel": "colaborador"}, 201)
	code, _, raw := admin.do("DELETE", "/_ge/api/usuarios/"+itoa(int(ids["bia"])), nil)
	if code >= 300 {
		t.Fatalf("excluir uma pessoa que participa de um time: %d %s", code, raw)
	}
	_, _, list := ana.do("GET", base+"/membros", nil)
	for _, m := range decodeList(t, list) {
		if m["pessoa_id"] == ids["bia"] {
			t.Fatalf("a participação da pessoa excluída ficou: %s", list)
		}
	}
}

// Deleting a person: what belongs to them (a required reference to the
// person) goes with them; what only names them (an optional reference:
// author, whoever closed it) stays, without the name.
func TestExcluirPessoaComRegistros(t *testing.T) {
	t.Setenv("GERMANIO_ADMIN_SENHA", "admin-senha-longa-1")
	_, admin := loadApp(t, "testdata/autoria/app.ge")
	_, ana := admin.fresh(t)
	me := ana.expect("POST", "/cadastro", map[string]any{"nome": "Ana", "email": "ana@x.com", "senha": "senha-forte-1"}, 201)
	ana.csrf = csrfFromCookie(t, ana)
	nota := ana.expect("POST", "/_ge/api/notas", map[string]any{"titulo": "n1"}, 201)
	ana.expect("POST", "/_ge/api/tarefas", map[string]any{"titulo": "t1"}, 201)
	admin.expect("POST", "/entrar", map[string]any{"login": "admin@autoria.local", "senha": "admin-senha-longa-1"}, 200)
	admin.csrf = csrfFromCookie(t, admin)
	code, _, raw := admin.do("DELETE", "/_ge/api/usuarios/"+itoa(int(me["id"].(float64))), nil)
	if code >= 300 {
		t.Fatalf("excluir uma pessoa com registros: %d %s", code, raw)
	}
	n := admin.expect("GET", "/_ge/api/notas/"+itoa(int(nota["id"].(float64))), nil, 200)
	if n["autor_id"] != nil {
		t.Fatalf("a nota deveria ficar sem autor: %v", n)
	}
	_, _, list := admin.do("GET", "/_ge/api/tarefas", nil)
	if items := decodeList(t, list); len(items) != 0 {
		t.Fatalf("as tarefas que pertenciam à pessoa ficaram: %s", list)
	}
}
