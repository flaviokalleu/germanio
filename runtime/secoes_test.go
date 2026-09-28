package runtime

import (
	"io"
	"strings"
	"testing"
)

// Page sections (GEP 0002, em teste): title, text, the create action with
// its label, only the listed filters, only the listed columns in order, and
// the empty state with its own action.
func TestSecoesDePagina(t *testing.T) {
	_, c := loadApp(t, "testdata/secoes/app.ge")
	c.expect("POST", "/cadastro", map[string]any{"nome": "Bia", "email": "bia@x.com", "senha": "senha-forte-1"}, 201)
	c.csrf = csrfFromCookie(t, c)
	page := func() string {
		resp, err := c.http.Get(c.base + "/clientes")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return string(b)
	}
	empty := page()
	for _, want := range []string{"<h1>Nossos clientes</h1>", "Todas as pessoas atendidas.", ">Novo cliente<", "<h2>Nenhum cliente</h2>", "Cadastre seu primeiro cliente.", ">Cadastrar cliente<", `name="cidade"`, `name="q"`} {
		if !strings.Contains(empty, want) {
			t.Errorf("página vazia sem %q", want)
		}
	}
	search := empty[strings.Index(empty, `<form class="busca"`):]
	search = search[:strings.Index(search, "</form>")]
	if strings.Contains(search, `name="status"`) || !strings.Contains(search, `name="cidade"`) {
		t.Errorf("os filtros não seguem a página: %s", search)
	}
	c.expect("POST", "/_ge/api/clientes", map[string]any{"nome": "Ana", "email": "ana@x.com", "telefone": "81999990000", "cidade": "Recife"}, 201)
	full := page()
	if strings.Contains(full, "Nenhum cliente") {
		t.Error("o estado vazio apareceu com dados")
	}
	heads := full[strings.Index(full, "<thead>"):strings.Index(full, "</thead>")]
	if !strings.Contains(heads, "Email") || !strings.Contains(heads, "Cidade") || strings.Contains(heads, "Telefone") {
		t.Errorf("colunas: %s", heads)
	}
	if strings.Index(heads, "Email") > strings.Index(heads, "Cidade") {
		t.Error("a ordem das colunas não segue a página")
	}
}
