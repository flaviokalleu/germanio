package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Copies (GEP 0029) and marks, list items and file addresses (GEP 0030), in
// a domain with no repository, no members and nothing of GitLab: recipes.
func TestCopiasDeReceitas(t *testing.T) {
	t.Setenv("GERMANIO_ARQUIVOS", t.TempDir())
	_, c := loadApp(t, "testdata/copias/app.ge")
	ana := signIn(t, c.base, "Ana", "ana@x.com")
	bia := signIn(t, c.base, "Bia", "bia@x.com")
	orig := ana.expect("POST", "/_ge/api/receitas", map[string]any{"titulo": "Bolo", "preparo": "misture", "etiquetas": []any{"doce", "bolo"}}, 201)
	if code, _, raw := sendRaw(t, ana, "PUT", "/_ge/api/receitas/1/foto?nome=bolo.png", png); code != 200 {
		t.Fatalf("foto: %d %s", code, raw)
	}

	// a copy: by whoever may copy, of everything plain, in their own name
	cp := bia.expect("POST", "/_ge/api/receitas/1/copiar", nil, 201)
	if cp["titulo"] != "Bolo" || cp["preparo"] != "misture" || cp["foto"] != nil || cp["favoritos"] != float64(0) {
		t.Fatalf("cópia: %v", cp)
	}
	if tags, _ := cp["etiquetas"].([]any); len(tags) != 2 {
		t.Fatalf("etiquetas da cópia: %v", cp["etiquetas"])
	}
	if cp["autor_id"] == nil || cp["autor_id"] == orig["autor_id"] {
		t.Fatalf("a cópia é de quem copiou: %v", cp["autor_id"])
	}
	if o, _ := cp["copiado_de"].(map[string]any); o == nil || o["id"] != float64(1) || o["titulo"] != "Bolo" {
		t.Fatalf("a cópia lembra o original: %v", cp["copiado_de"])
	}
	// values may change on the way; the origin cannot be given by hand
	other := bia.expect("POST", "/_ge/api/receitas/1/copiar", map[string]any{"titulo": "Bolo da Bia", "copiado_de_id": 99}, 201)
	if other["titulo"] != "Bolo da Bia" || other["copiado_de_id"] != float64(1) {
		t.Fatalf("cópia com outro título: %v", other)
	}
	bia.expect("PATCH", "/_ge/api/receitas/2", map[string]any{"copiado_de_id": 3}, 200)
	if got := bia.expect("GET", "/_ge/api/receitas/2", nil, 200); got["copiado_de_id"] != float64(1) {
		t.Fatalf("a origem mudou por edição: %v", got["copiado_de_id"])
	}
	anon := &client{t: t, base: c.base, http: c.http, Header: c.Header}
	if code, _, _ := anon.do("POST", "/_ge/api/receitas/1/copiar", nil); code != 404 { // what one cannot see is not found
		t.Fatalf("copiar sem entrar: %d", code)
	}

	// marks: once per person, counted on the record, listed per person
	if got := bia.expect("POST", "/_ge/api/receitas/1/marcar", nil, 200); got["favoritos"] != float64(1) {
		t.Fatalf("favorito: %v", got["favoritos"])
	}
	if code, _, _ := bia.do("POST", "/_ge/api/receitas/1/marcar", nil); code != 304 {
		t.Fatalf("favorito repetido: %d", code)
	}
	ana.expect("POST", "/_ge/api/receitas/2/marcar", nil, 200)
	ids := func(who *client, q string) []string {
		_, _, raw := who.do("GET", "/_ge/api/receitas"+q, nil)
		var list []map[string]any
		json.Unmarshal([]byte(raw), &list)
		var out []string
		for _, it := range list {
			out = append(out, it["titulo"].(string))
		}
		return out
	}
	if got := ids(bia, "?marcados=sim"); len(got) != 1 || got[0] != "Bolo" {
		t.Fatalf("favoritos de Bia: %v", got)
	}
	if got := ids(ana, "?marcados=sim"); len(got) != 1 || got[0] != "Bolo" {
		t.Fatalf("favoritos de Ana: %v", got)
	}
	if code, _, html := bia.do("GET", "/receitas/1", nil); code != 200 || !strings.Contains(html, "/receitas/1/acao/desmarcar") || strings.Contains(html, "/receitas/1/acao/marcar") || !strings.Contains(html, "/receitas/1/acao/copiar") {
		t.Fatalf("botões da receita: %d", code)
	}
	bia.expect("POST", "/_ge/api/receitas/1/desmarcar", nil, 200)
	if code, _, _ := bia.do("POST", "/_ge/api/receitas/1/desmarcar", nil); code != 304 {
		t.Fatalf("desmarcar duas vezes: %d", code)
	}
	bia.expect("PATCH", "/_ge/api/receitas/2", map[string]any{"favoritos": 50}, 200)
	if got := bia.expect("GET", "/_ge/api/receitas/2", nil, 200); got["favoritos"] != float64(1) {
		t.Fatalf("a contagem mudou por edição: %v", got["favoritos"])
	}

	// one item of a list as a filter
	ana.expect("POST", "/_ge/api/receitas", map[string]any{"titulo": "Pão", "etiquetas": []any{"salgado"}}, 201)
	if got := ids(ana, "?etiqueta=doce"); len(got) != 3 {
		t.Fatalf("receitas doces: %v", got)
	}
	if got := ids(ana, "?etiqueta=salgado"); len(got) != 1 || got[0] != "Pão" {
		t.Fatalf("receitas salgadas: %v", got)
	}

	// the record says where its file is
	if got := bia.expect("GET", "/_ge/api/receitas/1", nil, 200); got["foto_endereco"] != "/_ge/api/receitas/1/foto" {
		t.Fatalf("endereço da foto: %v", got["foto_endereco"])
	}

	// deleting the original keeps the copies, without the link, and its marks go
	ana.expect("POST", "/_ge/api/receitas/1/marcar", nil, 200)
	ana.expect("DELETE", "/_ge/api/receitas/1", nil, 204)
	if got := bia.expect("GET", "/_ge/api/receitas/2", nil, 200); got["copiado_de_id"] != nil || got["copiado_de"] != nil {
		t.Fatalf("a cópia ainda aponta para o original excluído: %v", got)
	}
	if got := ids(ana, "?marcados=sim"); len(got) != 1 || got[0] != "Bolo" {
		t.Fatalf("a marca do excluído continua: %v", got) // only receita 2 ("Bolo") remains marked
	}
}

// Marks need people and one kind per data; copiar is free only when the
// program does not define it.
func TestMarcasErros(t *testing.T) {
	cases := map[string]string{
		"sem login":  "crie sistema X\n\nreceitas\n    tem\n        titulo\n    recebe favoritos\n",
		"duas":       "crie sistema X\n\nusuarios\n    tem\n        nome\n        email obrigatório e único\n        senha min 8\n\ntenha login\n\nreceitas\n    tem\n        titulo\n    recebe favoritos\n    recebe curtidas\n",
		"nome usado": "crie sistema X\n\nusuarios\n    tem\n        nome\n        email obrigatório e único\n        senha min 8\n\ntenha login\n\nreceitas\n    tem\n        titulo\n        favoritos\n    recebe favoritos\n",
		"sem plural": "crie sistema X\n\nreceitas\n    tem\n        titulo\n    recebe favorito\n",
	}
	want := map[string]string{"sem login": "tenha login", "duas": "um só", "nome usado": "já é um campo", "sem plural": "recebe"}
	for name, src := range cases {
		dir := t.TempDir()
		file := filepath.Join(dir, "app.ge")
		os.WriteFile(file, []byte(src), 0o644)
		t.Setenv("GERMANIO_SQLITE", filepath.Join(dir, "app.db"))
		app, err := Carregar(file, "0")
		if err == nil {
			app.Fechar()
			t.Fatalf("%s: aceito", name)
		}
		if !strings.Contains(err.Error(), want[name]) {
			t.Fatalf("%s: erro sem explicação: %v", name, err)
		}
	}
}
