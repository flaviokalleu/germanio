package e2e

import (
	"net/url"
	"regexp"
	"strings"
	"testing"
)

// The web pages offer what the API already did, derived from the GitLab
// domain without any page syntax: merge when the pipeline passes, squash and
// cancel (GEP 0027), no approvals missing (GitLab FOSS has no minimum), moving an issue
// to another project (GEP 0034), time spent and links on the issue page, and
// forking one's own project with a new path (GEP 0029).

// login opens a browser session with the sign-in form.
func login(t *testing.T, base, username string) (*browser, string) {
	t.Helper()
	b := newBrowser(t, base)
	code, page, _ := b.submit("/entrar", url.Values{"login": {username}, "senha": {"password123"}})
	if code != 200 || !strings.Contains(page, "Sair") {
		t.Fatalf("entrar como %s: %d\n%s", username, code, page)
	}
	return b, csrfOf(t, page)
}

// open returns a page that must exist.
func open(t *testing.T, b *browser, path string) string {
	t.Helper()
	code, page, _ := b.get(path)
	if code != 200 {
		t.Fatalf("GET %s = %d\n%s", path, code, page)
	}
	return page
}

// fine: the form was accepted (no error on the page it led to).
func fine(t *testing.T, code int, page, at string) {
	t.Helper()
	if code != 200 || strings.Contains(at, "erro=") {
		t.Fatalf("formulário recusado: %d %s\n%s", code, at, page)
	}
}

// denied: the form was refused and the page says why.
func denied(t *testing.T, code int, page, at string) {
	t.Helper()
	if !strings.Contains(at, "erro=") || !strings.Contains(page, `class="aviso erro"`) {
		t.Fatalf("formulário deveria ser recusado: %d %s", code, at)
	}
}

func TestPaginasRestantes(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada") // owner of both projects
	bob := signup(t, base, "bob") // developer of Livro
	carol := signup(t, base, "carol")
	signup(t, base, "dan") // sees nothing private
	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Livro", "path": "livro", "initialize_with_readme": true}, 201)
	pid := id(p)
	outro := id(ada.must("POST", "/api/v4/projects", map[string]any{"name": "Outro", "path": "outro"}, 201))
	for who, level := range map[*api]int{bob: 30, carol: 20} {
		uid := who.must("GET", "/api/v4/user", nil, 200)["id"]
		ada.must("POST", "/api/v4/projects/"+pid+"/members", map[string]any{"user_id": uid, "access_level": level}, 201)
	}
	files := "/api/v4/projects/" + pid + "/repository/files/"
	// a pipeline per push; no executor runs it here, so it keeps running
	ada.must("PUT", files+".gitlab-ci.yml", map[string]any{"branch": "main", "content": "testar:\n  script: [\"true\"]\n", "commit_message": "ci"}, 200)
	ada.must("POST", "/api/v4/projects/"+pid+"/repository/branches", map[string]any{"branch": "cap", "ref": "main"}, 201)
	ada.must("PUT", files+"um.txt", map[string]any{"branch": "cap", "content": "um\n", "commit_message": "um"}, 200)
	ada.must("PUT", files+"dois.txt", map[string]any{"branch": "cap", "content": "dois\n", "commit_message": "dois"}, 200)
	mr := "/api/v4/projects/" + pid + "/merge_requests/1"
	bob.must("POST", "/api/v4/projects/"+pid+"/merge_requests", map[string]any{"title": "Capítulo", "source_branch": "cap", "target_branch": "main"}, 201)

	wa, ca := login(t, base, "ada")
	wb, cb := login(t, base, "bob")
	wc, cc := login(t, base, "carol")
	wd, cd := login(t, base, "dan")
	mrPage := "/projetos/" + pid + "/merge_requests/1"

	// --- no approval minimum in GitLab FOSS: nothing says approvals are missing ---
	for _, w := range []*browser{wa, wc} {
		if page := open(t, w, mrPage); strings.Contains(page, "aprovacoes-faltando") || strings.Contains(page, "aprovação para") {
			t.Fatalf("o GitLab FOSS não exige aprovações:\n%s", page)
		}
	}

	// --- merge when the pipeline passes, with squash; cancel; merge now ---
	page := open(t, wa, mrPage)
	if strings.Contains(page, "aprovacoes-faltando") || !strings.Contains(page, `name="juntar_commits"`) || !strings.Contains(page, `name="mesclar_quando_passar" value="true"`) {
		t.Fatalf("mesclar com juntar commits e mesclar quando passar:\n%s", page)
	}
	if page := open(t, wc, mrPage); strings.Contains(page, "/acao/mesclar") {
		t.Fatalf("o reporter não mescla:\n%s", page)
	}
	code, page, at := wc.submit(mrPage+"/acao/mesclar", url.Values{"_csrf": {cc}, "_campos": {"1"}, "mesclar_quando_passar": {"true"}})
	denied(t, code, page, at)
	code, page, at = wa.submit(mrPage+"/acao/mesclar", url.Values{"_csrf": {ca}, "_campos": {"1"}, "juntar_commits": {"true"}, "mesclar_quando_passar": {"true"}})
	fine(t, code, page, at)
	if got := ada.must("GET", mr, nil, 200); got["state"] != "opened" || got["merge_when_pipeline_succeeds"] != true || got["squash"] != true {
		t.Fatalf("o merge request espera o pipeline, juntando os commits: %v", got)
	}
	if !strings.Contains(page, "Mesclagem agendada") || !strings.Contains(page, mrPage+"/acao/cancelar_mesclagem") {
		t.Fatalf("a página diz que espera e oferece cancelar:\n%s", page)
	}
	code, page, at = wc.submit(mrPage+"/acao/cancelar_mesclagem", url.Values{"_csrf": {cc}})
	denied(t, code, page, at)
	code, page, at = wa.submit(mrPage+"/acao/cancelar_mesclagem", url.Values{"_csrf": {ca}})
	fine(t, code, page, at)
	if got := ada.must("GET", mr, nil, 200); got["merge_when_pipeline_succeeds"] != false {
		t.Fatalf("cancelar pela página: %v", got)
	}
	code, page, at = wa.submit(mrPage+"/acao/mesclar", url.Values{"_csrf": {ca}, "_campos": {"1"}, "juntar_commits": {"true"}})
	fine(t, code, page, at)
	head := ada.list("/api/v4/projects/" + pid + "/repository/commits?ref_name=main")[0].(map[string]any)
	if got := ada.must("GET", mr, nil, 200); got["state"] != "merged" || head["title"] != "Capítulo" || len(head["parent_ids"].([]any)) != 1 {
		t.Fatalf("mesclado pela página juntando os commits: %v / %v", got, head)
	}

	// --- time spent and links on the issue page ---
	issues := "/api/v4/projects/" + pid + "/issues"
	ada.must("POST", issues, map[string]any{"title": "Um"}, 201)
	ada.must("POST", issues, map[string]any{"title": "Dois"}, 201)
	issuePage := "/projetos/" + pid + "/issues/2"
	page = open(t, wa, issuePage)
	links := regexp.MustCompile(`(?s)<select name="relacionada_id"[^>]*>(.*?)</select>`).FindStringSubmatch(page)
	if links == nil || !strings.Contains(links[1], "#1 Um") || strings.Contains(links[1], "#2 Dois") {
		t.Fatalf("a ligação escolhe outra issue, nunca a própria:\n%s", page)
	}
	if !strings.Contains(page, issuePage+"/tempos_gastos/novo") || strings.Contains(page, `name="projeto_id"  >`) {
		t.Fatalf("tempo gasto na página da issue, e o projeto não se edita (muda-se):\n%s", page)
	}
	one := ada.must("GET", issues+"/1", nil, 200)
	code, page, at = wa.submit(issuePage+"/ligacoes/novo", url.Values{"_csrf": {ca}, "relacionada_id": {id(one)}})
	fine(t, code, page, at)
	code, page, at = wa.submit(issuePage+"/tempos_gastos/novo", url.Values{"_csrf": {ca}, "duracao": {"3600"}})
	fine(t, code, page, at)
	if l := ada.list(issues + "/2/links"); len(l) != 1 || l[0].(map[string]any)["title"] != "Um" {
		t.Fatalf("ligação criada pela página: %v", l)
	}
	if s := ada.must("GET", issues+"/2/time_stats", nil, 200); s["total_time_spent"] != float64(3600) {
		t.Fatalf("tempo gasto pela página: %v", s)
	}
	linked := regexp.MustCompile(`(?s)<div data-vivo="filhos-ligacoes">.*?</div></div>`).FindString(open(t, wc, issuePage))
	if !strings.Contains(linked, ">#1 Um</a>") {
		t.Fatalf("a ligação aparece pela outra issue:\n%s", linked)
	}
	// someone who cannot see the project changes nothing through a page form
	code, page, at = wd.submit(issuePage+"/tempos_gastos/novo", url.Values{"_csrf": {cd}, "duracao": {"60"}})
	denied(t, code, page, at)

	// --- moving an issue to another project ---
	page = open(t, wa, "/projetos/"+pid+"/issues/1")
	move := regexp.MustCompile(`(?s)action="/projetos/` + pid + `/issues/1/acao/mudar">.*?</form>`).FindString(page)
	if !strings.Contains(move, "Mudar de projeto") || !strings.Contains(move, `value="`+outro+`" >Outro`) || strings.Contains(move, ">Livro<") {
		t.Fatalf("mudar de projeto, só para onde a pessoa pode criar:\n%s", page)
	}
	// Bob may edit the issue but create nothing in Outro: no form, and a
	// forged one is refused
	if page := open(t, wb, "/projetos/"+pid+"/issues/1"); strings.Contains(page, "acao/mudar") {
		t.Fatalf("sem destino possível, sem formulário:\n%s", page)
	}
	code, page, at = wb.submit("/projetos/"+pid+"/issues/1/acao/mudar", url.Values{"_csrf": {cb}, "projeto_id": {outro}})
	denied(t, code, page, at)
	code, page, at = wa.submit("/projetos/"+pid+"/issues/1/acao/mudar", url.Values{"_csrf": {ca}, "projeto_id": {outro}})
	fine(t, code, page, at)
	if !strings.HasPrefix(at, "/projetos/"+outro+"/issues/1") || !strings.Contains(page, "<h1>#1 Um</h1>") {
		t.Fatalf("a página abre a issue no outro projeto: %s\n%s", at, page)
	}
	if got := ada.must("GET", "/api/v4/projects/"+outro+"/issues/1", nil, 200); got["title"] != "Um" {
		t.Fatalf("a issue mudou de projeto: %v", got)
	}

	// --- forking one's own project with a new path ---
	page = open(t, wa, "/projetos/"+pid)
	fork := regexp.MustCompile(`(?s)action="/projetos/` + pid + `/acao/copiar">.*?</form>`).FindString(page)
	if !strings.Contains(fork, `name="nome" value="Livro" required`) || !strings.Contains(fork, `name="caminho" value="livro" required`) {
		t.Fatalf("copiar pergunta nome e caminho:\n%s", page)
	}
	code, page, at = wa.submit("/projetos/"+pid+"/acao/copiar", url.Values{"_csrf": {ca}, "_campos": {"1"}, "nome": {"Livro"}, "caminho": {"livro"}})
	denied(t, code, page, at) // the same path next to the original
	code, page, at = wd.submit("/projetos/"+pid+"/acao/copiar", url.Values{"_csrf": {cd}, "_campos": {"1"}, "nome": {"Livro"}, "caminho": {"livro"}})
	denied(t, code, page, at) // a private project dan cannot see
	code, page, at = wa.submit("/projetos/"+pid+"/acao/copiar", url.Values{"_csrf": {ca}, "_campos": {"1"}, "nome": {"Livro 2"}, "caminho": {"livro-2"}})
	fine(t, code, page, at)
	if !strings.Contains(page, "<h1>Livro 2</h1>") {
		t.Fatalf("a página abre a cópia: %s\n%s", at, page)
	}
	forks := ada.list("/api/v4/projects?search=livro-2")
	if len(forks) != 1 {
		t.Fatalf("a cópia: %v", forks)
	}
	f := forks[0].(map[string]any)
	if f["full_path"] != "ada/livro-2" || f["forked_from_project"] == nil || id(f["forked_from_project"].(map[string]any)) != pid {
		t.Fatalf("cópia com o caminho escolhido e a origem: %v", f)
	}
}
