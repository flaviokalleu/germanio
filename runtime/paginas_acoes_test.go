package runtime

import (
	"net/url"
	"regexp"
	"strings"
	"testing"
)

// Pages offer what the API already does, derived from the declarations and
// without any page syntax: merging a proposal (squash, "merge when the
// executions pass", cancel), the approvals still missing, copying with a new
// name, moving to another parent and the children of a record with a
// reference to another record of the same kind. A publisher: books with a
// repository and builds, reviews, tasks with hours and links — no GitLab.

// postForm sends a page form and returns the status and where it leads.
func postForm(t *testing.T, c *client, path string, fields url.Values) (int, string) {
	t.Helper()
	resp, err := c.http.PostForm(c.base+path, fields)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	resp.Body.Close()
	return resp.StatusCode, resp.Header.Get("Location")
}

// page returns the HTML of a page.
func page(t *testing.T, c *client, path string) string {
	t.Helper()
	code, _, html := c.do("GET", path, nil)
	if code != 200 {
		t.Fatalf("GET %s: %d\n%s", path, code, html)
	}
	return html
}

// refused: a page form that the server refuses goes back with an error.
func refused(t *testing.T, code int, at string) {
	t.Helper()
	if code != 303 || !strings.Contains(at, "erro=") {
		t.Fatalf("a página deveria recusar e explicar: %d %s", code, at)
	}
}

func TestPaginasAcoes(t *testing.T) {
	_, anon := loadApp(t, "testdata/paginas_acoes/app.ge")
	ana := signIn(t, anon.base, "Ana", "ana@x.com")
	bia := signIn(t, anon.base, "Bia", "bia@x.com")
	caio := signIn(t, anon.base, "Caio", "caio@x.com")
	dani := signIn(t, anon.base, "Dani", "dani@x.com")
	ana.expect("POST", "/_ge/api/livros", map[string]any{"nome": "atlas"}, 201) // 1
	ana.expect("POST", "/_ge/api/livros", map[string]any{"nome": "mapa"}, 201)  // 2
	for c, role := range map[*client]string{bia: "revisor", caio: "leitor"} {
		pid := c.expect("GET", "/_ge/eu", nil, 200)["id"]
		ana.expect("POST", "/_ge/api/livros/1/membros", map[string]any{"pessoa_id": pid, "papel": role}, 201)
	}
	put := func(branch, path, content string) {
		ana.expect("PUT", "/_ge/api/livros/1/repositorio/arquivos/"+path, map[string]any{"branch": branch, "conteudo": content, "mensagem": "escreve " + path}, 200)
	}
	// builds exist for every push; no executor runs them, so they wait
	put("main", "compilar.yml", "etapas:\n  testar:\n    comandos: [\"true\"]\n")
	ana.expect("POST", "/_ge/api/livros/1/repositorio/branches", map[string]any{"nome": "rev1", "origem": "main"}, 201)
	put("rev1", "cap2.txt", "dois\n")
	put("rev1", "cap3.txt", "três\n")
	bia.expect("POST", "/_ge/api/livros/1/revisoes", map[string]any{"titulo": "Capítulos 2 e 3", "origem": "rev1", "destino": "main"}, 201)
	rev := "/livros/1/revisoes/1"

	// --- approvals still missing, next to where the merge would be ---
	for _, c := range []*client{ana, caio} {
		p := page(t, c, rev)
		if !strings.Contains(p, `data-aprovacoes-faltando="1"`) || !strings.Contains(p, "Falta 1 aprovação para mesclar: tem 0 de 1") || strings.Contains(p, rev+"/acao/mesclar") {
			t.Fatalf("a página diz quantas aprovações faltam e ainda não mescla:\n%s", p)
		}
	}
	// the author's approval does not count; Ana's does (by the page button)
	if code, at := postForm(t, bia, rev+"/acao/aprovar", url.Values{"_csrf": {bia.csrf}}); code != 303 || strings.Contains(at, "erro=") {
		t.Fatalf("aprovar pela página: %d %s", code, at)
	}
	if p := page(t, ana, rev); !strings.Contains(p, "Falta 1 aprovação") {
		t.Fatalf("a aprovação do autor não conta:\n%s", p)
	}
	postForm(t, ana, rev+"/acao/aprovar", url.Values{"_csrf": {ana.csrf}})

	// --- merging: squash, merge when the build passes, cancel ---
	p := page(t, ana, rev)
	if strings.Contains(p, "aprovacoes-faltando") || !strings.Contains(p, `action="`+rev+`/acao/mesclar"`) ||
		!strings.Contains(p, `name="juntar_commits"`) || !strings.Contains(p, `<button name="mesclar_quando_passar" value="true">Mesclar quando passar</button>`) {
		t.Fatalf("formulário de mesclar com juntar commits e mesclar quando passar:\n%s", p)
	}
	if p := page(t, caio, rev); strings.Contains(p, "/acao/mesclar") || strings.Contains(p, `name="mesclar_quando_passar"`) {
		t.Fatalf("o leitor não vê como mesclar:\n%s", p)
	}
	// without CSRF nothing happens; a reader's forged form is refused
	if code, _ := postForm(t, ana, rev+"/acao/mesclar", url.Values{"mesclar_quando_passar": {"true"}}); code != 403 {
		t.Fatalf("formulário sem CSRF: %d", code)
	}
	code, at := postForm(t, caio, rev+"/acao/mesclar", url.Values{"_csrf": {caio.csrf}, "_campos": {"1"}, "mesclar_quando_passar": {"true"}})
	refused(t, code, at)
	// merge when it passes: the build of rev1 is still waiting
	code, at = postForm(t, ana, rev+"/acao/mesclar", url.Values{"_csrf": {ana.csrf}, "_campos": {"1"}, "juntar_commits": {"true"}, "mesclar_quando_passar": {"true"}})
	if code != 303 || !strings.Contains(at, "ok=") || !strings.Contains(at, "agendada") {
		t.Fatalf("mesclar quando passar pela página: %d %s", code, at)
	}
	got := ana.expect("GET", "/_ge/api/livros/1/revisoes/1", nil, 200)
	if got["estado"] != "aberta" || got["mesclar_quando_passar"] != true || got["juntar_commits"] != true {
		t.Fatalf("a revisão espera a compilação, juntando os commits: %v", got)
	}
	p = page(t, ana, rev)
	if !strings.Contains(p, "Mesclagem agendada") || !strings.Contains(p, rev+"/acao/cancelar_mesclagem") || strings.Contains(p, "Mesclar quando passar</button>") {
		t.Fatalf("a página diz que espera e oferece cancelar:\n%s", p)
	}
	if p := page(t, caio, rev); strings.Contains(p, "cancelar_mesclagem") {
		t.Fatalf("o leitor não cancela:\n%s", p)
	}
	code, at = postForm(t, caio, rev+"/acao/cancelar_mesclagem", url.Values{"_csrf": {caio.csrf}})
	refused(t, code, at)
	postForm(t, ana, rev+"/acao/cancelar_mesclagem", url.Values{"_csrf": {ana.csrf}})
	if got := ana.expect("GET", "/_ge/api/livros/1/revisoes/1", nil, 200); got["mesclar_quando_passar"] != false {
		t.Fatalf("cancelar pela página: %v", got)
	}
	// merge now, squashing: one commit on main with the review's title
	code, at = postForm(t, ana, rev+"/acao/mesclar", url.Values{"_csrf": {ana.csrf}, "_campos": {"1"}, "juntar_commits": {"true"}})
	if code != 303 || strings.Contains(at, "erro=") {
		t.Fatalf("mesclar pela página: %d %s", code, at)
	}
	_, _, raw := ana.do("GET", "/_ge/api/livros/1/repositorio/commits?ref_name=main", nil)
	head := decodeList(t, raw)[0]
	if got := ana.expect("GET", "/_ge/api/livros/1/revisoes/1", nil, 200); got["estado"] != "mesclada" || head["title"] != "Capítulos 2 e 3" || len(head["parent_ids"].([]any)) != 1 {
		t.Fatalf("mesclada juntando os commits: %v / %v", got, head)
	}

	// --- children of a task: hours and links to another task ---
	ana.expect("POST", "/_ge/api/livros/1/tarefas", map[string]any{"titulo": "Revisar"}, 201)  // 1
	ana.expect("POST", "/_ge/api/livros/1/tarefas", map[string]any{"titulo": "Ilustrar"}, 201) // 2
	bia.expect("POST", "/_ge/api/livros/1/tarefas", map[string]any{"titulo": "Da Bia"}, 201)   // 3
	t2 := ana.expect("GET", "/_ge/api/livros/1/tarefas/2", nil, 200)
	t1 := ana.expect("GET", "/_ge/api/livros/1/tarefas/1", nil, 200)
	p = page(t, bia, "/livros/1/tarefas/2")
	links := regexp.MustCompile(`(?s)<select name="relacionada_id"[^>]*>(.*?)</select>`).FindStringSubmatch(p)
	if links == nil || !strings.Contains(links[1], `value="`+jsonID(t1["id"])+`"`) || strings.Contains(links[1], `value="`+jsonID(t2["id"])+`"`) {
		t.Fatalf("a ligação escolhe outra tarefa, nunca a própria:\n%s", p)
	}
	if !strings.Contains(p, `action="/livros/1/tarefas/2/horas_gastas/novo"`) || !strings.Contains(p, `name="duracao"`) {
		t.Fatalf("a tarefa mostra as horas gastas com o formulário:\n%s", p)
	}
	if code, at := postForm(t, bia, "/livros/1/tarefas/2/ligacoes/novo", url.Values{"_csrf": {bia.csrf}, "relacionada_id": {jsonID(t1["id"])}}); code != 303 || strings.Contains(at, "erro=") {
		t.Fatalf("ligar pela página: %d %s", code, at)
	}
	if code, at := postForm(t, bia, "/livros/1/tarefas/2/horas_gastas/novo", url.Values{"_csrf": {bia.csrf}, "duracao": {"30"}}); code != 303 || strings.Contains(at, "erro=") {
		t.Fatalf("registrar horas pela página: %d %s", code, at)
	}
	p = page(t, caio, "/livros/1/tarefas/2")
	linked := regexp.MustCompile(`(?s)<div data-vivo="filhos-ligacoes">.*?</div></div>`).FindString(p)
	if !strings.Contains(linked, `">#1 Revisar</a>`) || strings.Contains(p, "horas_gastas/novo") {
		t.Fatalf("o leitor vê a ligação pela outra tarefa e não registra horas:\n%s", p)
	}
	code, at = postForm(t, caio, "/livros/1/tarefas/2/horas_gastas/novo", url.Values{"_csrf": {caio.csrf}, "duracao": {"5"}})
	refused(t, code, at)
	_, _, raw = ana.do("GET", "/_ge/api/livros/1/tarefas/2/horas_gastas", nil)
	if hours := decodeList(t, raw); len(hours) != 1 || hours[0]["duracao"] != float64(30) {
		t.Fatalf("só a hora da revisora: %v", hours)
	}

	// --- moving a task to another book ---
	p = page(t, ana, "/livros/1/tarefas/1")
	move := regexp.MustCompile(`(?s)<form class="caixa" method="post" action="/livros/1/tarefas/1/acao/mudar">.*?</form>`).FindString(p)
	if !strings.Contains(move, "Mudar de livro") || !strings.Contains(move, `<option value="2" >mapa</option>`) || strings.Contains(move, ">atlas<") {
		t.Fatalf("mudar de livro, só para onde a pessoa pode criar:\n%s", p)
	}
	// Bia may edit her task but create in no other book: no form; a forged
	// one is refused
	if p := page(t, bia, "/livros/1/tarefas/3"); strings.Contains(p, "acao/mudar") {
		t.Fatalf("sem destino possível, sem formulário:\n%s", p)
	}
	code, at = postForm(t, bia, "/livros/1/tarefas/3/acao/mudar", url.Values{"_csrf": {bia.csrf}, "livro_id": {"2"}})
	refused(t, code, at)
	if p := page(t, caio, "/livros/1/tarefas/1"); strings.Contains(p, "acao/mudar") {
		t.Fatalf("o leitor não muda tarefas de lugar:\n%s", p)
	}
	code, at = postForm(t, ana, "/livros/1/tarefas/1/acao/mudar", url.Values{"_csrf": {ana.csrf}, "livro_id": {"2"}})
	if code != 303 || !strings.HasPrefix(at, "/livros/2/tarefas/1?ok=") {
		t.Fatalf("mudar pela página abre a tarefa no destino: %d %s", code, at)
	}
	if moved := ana.expect("GET", "/_ge/api/livros/2/tarefas/1", nil, 200); moved["titulo"] != "Revisar" {
		t.Fatalf("a tarefa está no outro livro: %v", moved)
	}

	// --- copying with a new name ---
	p = page(t, ana, "/livros/1")
	cp := regexp.MustCompile(`(?s)<form class="caixa" method="post" action="/livros/1/acao/copiar">.*?</form>`).FindString(p)
	if !strings.Contains(cp, `name="nome" value="" required`) || strings.Contains(p, `<button >Copiar</button>`) {
		t.Fatalf("copiar pergunta o nome (único, então vazio):\n%s", p)
	}
	code, at = postForm(t, ana, "/livros/1/acao/copiar", url.Values{"_csrf": {ana.csrf}, "_campos": {"1"}, "nome": {"atlas"}})
	refused(t, code, at)
	code, at = postForm(t, dani, "/livros/1/acao/copiar", url.Values{"_csrf": {dani.csrf}, "_campos": {"1"}, "nome": {"da-dani"}})
	refused(t, code, at)
	code, at = postForm(t, caio, "/livros/1/acao/copiar", url.Values{"_csrf": {caio.csrf}, "_campos": {"1"}, "nome": {"atlas do caio"}})
	if code != 303 || !strings.HasPrefix(at, "/livros/3?ok=") {
		t.Fatalf("copiar pela página abre a cópia: %d %s", code, at)
	}
	if copy := caio.expect("GET", "/_ge/api/livros/3", nil, 200); copy["nome"] != "atlas do caio" || copy["copiado_de_id"] != float64(1) {
		t.Fatalf("a cópia com o nome escolhido: %v", copy)
	}
	_, _, raw = caio.do("GET", "/_ge/api/livros/3/repositorio/arvore", nil)
	if !strings.Contains(raw, "cap2.txt") {
		t.Fatalf("a cópia tem o repositório: %s", raw)
	}
	_, _, raw = dani.do("GET", "/_ge/api/livros?per_page=100", nil)
	if got := decodeList(t, raw); len(got) != 0 {
		t.Fatalf("nada foi copiado por quem não pode: %v", got)
	}
}
