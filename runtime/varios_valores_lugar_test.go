package runtime

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flaviokalleu/germanio/runtime/git"
)

// GEP 0043 (several values in a filter), GEP 0044 (the place of a new record named by
// its address) and the copies of one original (GEP 0029), in a domain with
// no repository and nothing of GitLab: model contracts kept in folders.
func TestVariosValoresLugarECopias(t *testing.T) {
	_, c := loadApp(t, "testdata/modelos/app.ge")
	ana := signIn(t, c.base, "ana", "ana@x.com")
	bia := signIn(t, c.base, "bia", "bia@x.com")
	caio := signIn(t, c.base, "caio", "caio@x.com")
	ana.expect("POST", "/_ge/api/contratos", map[string]any{"titulo": "Aluguel", "caminho": "aluguel", "tipo": "locacao", "etiquetas": []any{"imovel", "mensal"}, "visibilidade": "public"}, 201)
	ana.expect("POST", "/_ge/api/contratos", map[string]any{"titulo": "Venda", "caminho": "venda", "tipo": "compra", "etiquetas": []any{"imovel"}, "visibilidade": "public"}, 201)
	ana.expect("POST", "/_ge/api/contratos", map[string]any{"titulo": "Serviço", "caminho": "servico", "tipo": "servico", "etiquetas": []any{"mensal", "imovel"}, "visibilidade": "private"}, 201)

	titles := func(who *client, q string) string {
		t.Helper()
		code, _, raw := who.do("GET", "/_ge/api/contratos"+q, nil)
		if code != 200 {
			t.Fatalf("GET %s: %d %s", q, code, raw)
		}
		var list []map[string]any
		json.Unmarshal([]byte(raw), &list)
		var out []string
		for _, it := range list {
			out = append(out, it["titulo"].(string))
		}
		return strings.Join(out, ",")
	}
	// a filter repeated: any of the values; never more than the person sees
	if got := titles(ana, "?tipo=locacao&tipo=servico"); got != "Serviço,Aluguel" {
		t.Fatalf("tipo locacao ou servico: %s", got)
	}
	if got := titles(bia, "?tipo=locacao&tipo=servico"); got != "Aluguel" {
		t.Fatalf("o contrato privado não aparece para bia: %s", got)
	}
	if got := titles(ana, "?tipo=locacao"); got != "Aluguel" {
		t.Fatalf("um valor continua: %s", got)
	}
	// several items of a list: all of them (commas or repeated)
	if got := titles(ana, "?etiqueta=imovel,mensal"); got != "Serviço,Aluguel" {
		t.Fatalf("imovel e mensal: %s", got)
	}
	if got := titles(ana, "?etiqueta=mensal&etiqueta=imovel"); got != "Serviço,Aluguel" {
		t.Fatalf("etiqueta repetida: %s", got)
	}
	if got := titles(bia, "?etiqueta=imovel,mensal"); got != "Aluguel" {
		t.Fatalf("todas as etiquetas, só o que bia vê: %s", got)
	}

	// the copies of one original: only those the person sees, of an
	// original the person sees
	bia.expect("POST", "/_ge/api/contratos/1/copiar", nil, 201)
	bia.expect("POST", "/_ge/api/contratos/1/copiar", map[string]any{"caminho": "aluguel-2", "visibilidade": "private"}, 201)
	if got := titles(ana, "?copiado_de_id=1"); got != "Aluguel" {
		t.Fatalf("cópias que ana vê: %s", got)
	}
	if got := titles(bia, "?copiado_de_id=1"); got != "Aluguel,Aluguel" {
		t.Fatalf("cópias que bia vê: %s", got)
	}
	ana.expect("POST", "/_ge/api/contratos", map[string]any{"titulo": "Rascunho", "caminho": "rascunho", "visibilidade": "public"}, 201)
	ana.expect("POST", "/_ge/api/contratos/6/copiar", map[string]any{"caminho": "rascunho-2"}, 201)
	ana.expect("PATCH", "/_ge/api/contratos/6", map[string]any{"visibilidade": "private"}, 200)
	// caio sees the copy, not the original: the list does not say where it came from
	if got := titles(caio, "?tipo=&copiado_de_id=6"); got != "" {
		t.Fatalf("cópias de um original que caio não vê: %s", got)
	}
	if got := caio.expect("GET", "/_ge/api/contratos/7", nil, 200); got["copiado_de_id"] != nil || got["copiado_de"] != nil {
		t.Fatalf("a cópia não diz a origem a caio: %v", got)
	}
	if got := titles(caio, "?copiado_de_id=999"); got != "" {
		t.Fatalf("cópias de um original que não existe: %s", got)
	}

	// the place of a new record, named by its address
	bia.expect("POST", "/_ge/api/pastas", map[string]any{"nome": "Escritório", "caminho": "escritorio", "visibilidade": "public"}, 201)
	ana.expect("POST", "/_ge/api/pastas", map[string]any{"nome": "Cofre", "caminho": "cofre", "visibilidade": "private"}, 201)
	in := bia.expect("POST", "/_ge/api/contratos", map[string]any{"titulo": "Sociedade", "caminho": "sociedade", "lugar": "bia/escritorio"}, 201)
	if in["endereco"] != "bia/escritorio/sociedade" || in["pasta_id"] == nil {
		t.Fatalf("criado na pasta pelo endereço: %v", in)
	}
	own := bia.expect("POST", "/_ge/api/contratos", map[string]any{"titulo": "Pessoal", "caminho": "pessoal", "lugar": "bia", "pasta_id": in["pasta_id"]}, 201)
	if own["endereco"] != "bia/pessoal" || own["pasta_id"] != nil {
		t.Fatalf("criado no próprio espaço (o lugar vence a referência): %v", own)
	}
	cp := bia.expect("POST", "/_ge/api/contratos/2/copiar", map[string]any{"lugar": "bia/escritorio"}, 201)
	if cp["endereco"] != "bia/escritorio/venda" {
		t.Fatalf("cópia na pasta pelo endereço: %v", cp)
	}
	for _, tc := range []struct {
		lugar string
		code  int
	}{{"ana", 403}, {"ana/cofre", 404}, {"ninguem", 404}, {"bia/nada", 404}} {
		if code, _, raw := bia.do("POST", "/_ge/api/contratos", map[string]any{"titulo": "X", "caminho": "x", "lugar": tc.lugar}); code != tc.code {
			t.Fatalf("lugar %q: %d %s", tc.lugar, code, raw)
		}
	}
	// a folder one sees but may not create in
	caio.expect("GET", "/_ge/api/pastas/1", nil, 200)
	if code, _, raw := caio.do("POST", "/_ge/api/contratos", map[string]any{"titulo": "X", "caminho": "x", "lugar": "bia/escritorio"}); code != 403 {
		t.Fatalf("lugar onde caio não pode criar: %d %s", code, raw)
	}
}

// GEP 0036, "update now": the built-in action atualizar_agora of a mirror
// (publishing house, no GitLab) runs the same background update, only for
// whoever may edit the mirror.
func TestEspelhoAtualizarAgora(t *testing.T) {
	_, p, _ := editora(t)
	ana := p["ana"]
	remoteDir := t.TempDir()
	remote, err := git.NewStore(remoteDir)
	if err != nil {
		t.Fatal(err)
	}
	remote.Init("destino.git", "main")
	bare := filepath.Join(remoteDir, "destino.git")
	exec.Command("git", "--git-dir", bare, "config", "http.receivepack", "true").Run()
	srv := gitServer(t, remoteDir, "ana", "s3gr3do-do-espelho")
	t.Setenv("GERMANIO_PERMITIR_REDE_LOCAL", "1")
	p["caio"].expect("PUT", "/_ge/api/livros/1/repositorio/arquivos/capitulo1.md", map[string]any{"conteudo": "Era uma vez", "branch": "main"}, 200)
	ana.expect("POST", "/_ge/api/livros/1/copias", map[string]any{"url": strings.Replace(srv, "http://", "http://ana:s3gr3do-do-espelho@", 1) + "/destino.git"}, 201)
	mirror := func() map[string]any { return ana.expect("GET", "/_ge/api/livros/1/copias/1", nil, 200) }
	refs := func() string {
		out, _ := exec.Command("git", "--git-dir", bare, "for-each-ref", "--format=%(refname)").Output()
		return strings.TrimSpace(string(out))
	}
	waitFor(t, "a primeira atualização", func() bool { return mirror()["situacao"] == "atualizada" && refs() == "refs/heads/main" })
	exec.Command("git", "--git-dir", bare, "update-ref", "refs/heads/sobra", "refs/heads/main").Run()
	for _, who := range []string{"bia", "caio", "eve"} { // they do not manage (nor see) the mirrors
		if code, _, _ := p[who].do("POST", "/_ge/api/livros/1/copias/1/atualizar_agora", nil); code != 404 {
			t.Fatalf("%s atualizando o espelho: %d", who, code)
		}
	}
	if refs() == "refs/heads/main" {
		t.Fatal("a ref sobrando não foi criada no outro servidor")
	}
	ana.expect("POST", "/_ge/api/livros/1/copias/1/atualizar_agora", nil, 200)
	waitFor(t, "o espelho voltar a ser cópia exata", func() bool { return mirror()["situacao"] == "atualizada" && refs() == "refs/heads/main" })
	ana.expect("PATCH", "/_ge/api/livros/1/copias/1", map[string]any{"habilitado": false}, 200)
	if code, _, raw := ana.do("POST", "/_ge/api/livros/1/copias/1/atualizar_agora", nil); code != 400 || !strings.Contains(raw, "desligado") {
		t.Fatalf("espelho desligado: %d %s", code, raw)
	}
}
