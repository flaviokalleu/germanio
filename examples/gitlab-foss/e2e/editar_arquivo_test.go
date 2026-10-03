package e2e

import (
	"net/url"
	"strings"
	"testing"
)

// RP-08: editar um arquivo pela web é um commit da pessoa, com as regras de
// enviar código (inclusive a branch padrão protegida).
func TestEditarArquivoPelaWeb(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	bob := signup(t, base, "bob")
	eve := signup(t, base, "eve")
	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Docs", "path": "docs", "initialize_with_readme": true}, 201)
	pid := id(p)
	bobID := bob.must("GET", "/api/v4/user", nil, 200)["id"]
	ada.must("POST", "/api/v4/projects/"+pid+"/members", map[string]any{"user_id": bobID, "access_level": 30}, 201)

	c := ada.must("PUT", "/api/v4/projects/"+pid+"/repository/files/README.md", map[string]any{"branch": "main", "content": "# Docs novos\n", "commit_message": "Atualiza o README"}, 200)
	if c["commit_id"] == nil {
		t.Fatalf("commit: %v", c)
	}
	if f := ada.must("GET", "/api/v4/projects/"+pid+"/repository/files/README.md", nil, 200); !strings.Contains(f["content"].(string), "Docs novos") {
		t.Fatalf("conteúdo depois de editar: %v", f["content"])
	}
	commits := ada.list("/api/v4/projects/" + pid + "/repository/commits")
	if first := commits[0].(map[string]any); first["title"] != "Atualiza o README" || first["author_name"] != "Ada" {
		t.Fatalf("o commit é da pessoa, com a mensagem dela: %v", first)
	}
	// a developer cannot change the protected default branch; an outsider is not told the project exists
	bob.must("PUT", "/api/v4/projects/"+pid+"/repository/files/README.md", map[string]any{"branch": "main", "content": "x"}, 403)
	eve.must("PUT", "/api/v4/projects/"+pid+"/repository/files/README.md", map[string]any{"branch": "main", "content": "x"}, 404)

	// the page: the file shows an edit form to the owner, and saving commits
	b := newBrowser(t, base)
	code, page, _ := b.submit("/entrar", url.Values{"login": {"ada"}, "senha": {"password123"}})
	if code != 200 {
		t.Fatalf("entrar: %d", code)
	}
	_, page, _ = b.get("/projetos/1/codigo?caminho=README.md")
	if !strings.Contains(page, `action="/projetos/1/codigo/salvar?caminho=README.md"`) {
		t.Fatalf("a página do arquivo não tem o formulário de edição:\n%s", page)
	}
	csrf := csrfOf(t, page)
	code, page, _ = b.submit("/projetos/1/codigo/salvar?caminho=README.md", url.Values{"_csrf": {csrf}, "conteudo": {"# Pela página\n"}, "mensagem": {"Edita pela página"}})
	if code != 200 || !strings.Contains(page, "Arquivo salvo") || !strings.Contains(page, "Pela página") {
		t.Fatalf("salvar pela página: %d\n%s", code, page)
	}
}
