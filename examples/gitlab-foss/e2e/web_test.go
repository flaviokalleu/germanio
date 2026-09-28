package e2e

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

type browser struct {
	t    *testing.T
	base string
	c    *http.Client
}

func newBrowser(t *testing.T, base string) *browser {
	jar, _ := cookiejar.New(nil)
	return &browser{t: t, base: base, c: &http.Client{Jar: jar}}
}

func (b *browser) get(path string) (int, string, string) {
	b.t.Helper()
	resp, err := b.c.Get(b.base + path)
	if err != nil {
		b.t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body), resp.Request.URL.RequestURI()
}

// submit posts a form (following the redirect) and returns the final page.
func (b *browser) submit(action string, fields url.Values) (int, string, string) {
	b.t.Helper()
	resp, err := b.c.PostForm(b.base+action, fields)
	if err != nil {
		b.t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body), resp.Request.URL.RequestURI()
}

var csrfRe = regexp.MustCompile(`name="_csrf" value="([^"]*)"`)
var formRe = regexp.MustCompile(`<form[^>]*method="post" action="([^"]+)"`)

func csrfOf(t *testing.T, page string) string {
	m := csrfRe.FindStringSubmatch(page)
	if m == nil {
		t.Fatalf("página sem token CSRF:\n%s", page)
	}
	return m[1]
}

func TestInterfaceWeb(t *testing.T) {
	base := gitlab(t)
	b := newBrowser(t, base)
	code, _, at := b.get("/")
	if code != 200 || !strings.HasPrefix(at, "/projetos") {
		t.Fatalf("raiz deveria levar à primeira página: %d %s", code, at)
	}
	_, page, _ := b.get("/cadastro")
	if !strings.Contains(page, `name="username"`) || !strings.Contains(page, `type="password"`) {
		t.Fatalf("formulário de cadastro: %s", page)
	}
	code, page, at = b.submit("/cadastro", url.Values{"username": {"ada"}, "nome": {"Ada"}, "email": {"ada@example.com"}, "senha": {"password123"}})
	if code != 200 || !strings.Contains(page, "Ada") || !strings.Contains(page, "Sair") {
		t.Fatalf("cadastro pelo formulário: %d %s", code, at)
	}
	csrf := csrfOf(t, page)

	// seções da página (GEP 0002): título, rótulo da ação e estado vazio declarados
	for _, want := range []string{"Projetos", "Novo projeto", "Nenhum projeto ainda", "Criar projeto"} {
		if !strings.Contains(page, want) {
			t.Fatalf("página de projetos sem %q:\n%s", want, page)
		}
	}
	// criar projeto pelo formulário
	if !strings.Contains(page, `action="/projetos/novo"`) {
		t.Fatalf("página de projetos sem formulário de criação:\n%s", page)
	}
	code, page, at = b.submit("/projetos/novo", url.Values{"_csrf": {csrf}, "_campos": {"1"}, "nome": {"Web App"}, "caminho": {"web-app"}, "visibilidade": {"private"}})
	if code != 200 || !strings.Contains(page, "Criado com sucesso") || !strings.Contains(at, "/projetos/1") {
		t.Fatalf("criar projeto: %d %s\n%s", code, at, page)
	}
	for _, want := range []string{"ada/web-app", "Issues", "Merge requests", "Pipelines", "repositório ainda está vazio"} {
		if !strings.Contains(page, want) {
			t.Fatalf("página do projeto sem %q:\n%s", want, page)
		}
	}
	// erro aparece para a pessoa, sem perder a página
	code, page, _ = b.submit("/projetos/novo", url.Values{"_csrf": {csrf}, "_campos": {"1"}, "nome": {"Outro"}, "caminho": {"web-app"}})
	if code != 200 || !strings.Contains(page, "aviso erro") {
		t.Fatalf("erro de validação deve aparecer: %d\n%s", code, page)
	}

	// issue com título perigoso: o HTML é escapado
	code, page, at = b.submit("/projetos/1/issues/novo", url.Values{"_csrf": {csrf}, "_campos": {"1"}, "titulo": {"<script>alert(1)</script>"}, "descricao": {"Passos:\n\n1. **abrir** `app`\n\n<img src=x onerror=alert(2)> [x](javascript:alert(3))"}})
	if code != 200 || !strings.Contains(at, "/projetos/1/issues/1") {
		t.Fatalf("criar issue: %d %s\n%s", code, at, page)
	}
	if strings.Contains(page, "<script>alert(1)") || !strings.Contains(page, "&lt;script&gt;") {
		t.Fatalf("título não foi escapado")
	}
	// descrição formatada: Markdown vira HTML, HTML e links perigosos não passam
	// (o texto original continua, escapado, no formulário de edição)
	rendered, _, _ := strings.Cut(page[strings.Index(page, `<div class="texto">`)+1:], "</div>")
	if !strings.Contains(rendered, "<strong>abrir</strong>") || !strings.Contains(rendered, "<code>app</code>") || strings.Contains(rendered, "onerror") || strings.Contains(rendered, "javascript:") || strings.Contains(page, "<img src=x") {
		t.Fatalf("descrição formatada:\n%s", page)
	}
	if !strings.Contains(page, `action="/projetos/1/issues/1/acao/fechar"`) || strings.Contains(page, "acao/reabrir") {
		t.Fatalf("botões de estado da issue aberta:\n%s", page)
	}
	code, page, _ = b.submit("/projetos/1/issues/1/acao/fechar", url.Values{"_csrf": {csrf}})
	if code != 200 || !strings.Contains(page, "fechada") || strings.Contains(page, "acao/fechar\"") || !strings.Contains(page, "acao/reabrir") {
		t.Fatalf("fechar pela interface:\n%s", page)
	}
	// comentário pela interface
	code, page, _ = b.submit("/projetos/1/issues/1/comentarios/novo", url.Values{"_csrf": {csrf}, "_campos": {"1"}, "texto": {"primeiro **comentário**"}})
	if code != 200 {
		t.Fatalf("comentar: %d\n%s", code, page)
	}
	_, page, _ = b.get("/projetos/1/issues/1")
	if !strings.Contains(page, "primeiro <strong>comentário</strong>") {
		t.Fatalf("comentário não aparece na issue")
	}

	// sem token CSRF nada acontece
	resp, _ := b.c.PostForm(base+"/projetos/novo", url.Values{"nome": {"x"}, "caminho": {"x"}})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("formulário sem CSRF deveria ser recusado: %d", resp.StatusCode)
	}
	resp.Body.Close()

	// outra pessoa não vê o projeto privado nem botões
	eve := newBrowser(t, base)
	eve.submit("/cadastro", url.Values{"username": {"eve"}, "nome": {"Eve"}, "email": {"eve@example.com"}, "senha": {"password123"}})
	code, page, _ = eve.get("/projetos/1")
	if code != 404 || strings.Contains(page, "web-app") {
		t.Fatalf("projeto privado visível para outra pessoa: %d", code)
	}
	_, page, _ = eve.get("/projetos")
	if strings.Contains(page, "web-app") {
		t.Fatalf("projeto privado listado para outra pessoa")
	}

	// nenhum botão zumbi: todo formulário das páginas visitadas responde
	visited := map[string]bool{}
	for _, p := range []string{"/projetos", "/projetos/1", "/projetos/1/issues/1", "/grupos"} {
		_, page, _ = b.get(p)
		for _, m := range formRe.FindAllStringSubmatch(page, -1) {
			action := m[1]
			if visited[action] || action == "/sair" || strings.HasSuffix(action, "/excluir") {
				continue
			}
			visited[action] = true
			req, _ := http.NewRequest("POST", base+action, strings.NewReader(url.Values{"_csrf": {csrfOf(t, page)}, "_campos": {"1"}}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			noFollow := &http.Client{Jar: b.c.Jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
			r2, err := noFollow.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			r2.Body.Close()
			if r2.StatusCode != http.StatusSeeOther {
				t.Errorf("botão zumbi: POST %s respondeu %d", action, r2.StatusCode)
			}
		}
	}
	if len(visited) < 5 {
		t.Fatalf("poucos formulários verificados: %v", visited)
	}
}
