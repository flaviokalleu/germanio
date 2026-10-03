package runtime

import (
	"encoding/json"
	"testing"
)

// A record that names two records of the same kind (a citation between two
// documents of any folder): it lives under its own document, goes away with
// either one, can only point at something its author sees, and is seen or
// changed only by whoever sees both — whatever its own rules say.
func TestRegistroComDuasReferencias(t *testing.T) {
	_, anon := loadApp(t, "testdata/citacoes/app.ge")
	ana := signIn(t, anon.base, "Ana", "ana@x.com")
	bia := signIn(t, anon.base, "Bia", "bia@x.com")
	ana.expect("POST", "/_ge/api/pastas", map[string]any{"nome": "Pública", "visibilidade": "public"}, 201)
	ana.expect("POST", "/_ge/api/pastas", map[string]any{"nome": "Privada"}, 201)
	bia.expect("POST", "/_ge/api/pastas", map[string]any{"nome": "Da Bia", "visibilidade": "public"}, 201)
	d1 := ana.expect("POST", "/_ge/api/pastas/1/documentos", map[string]any{"titulo": "Aberto"}, 201)
	d2 := ana.expect("POST", "/_ge/api/pastas/2/documentos", map[string]any{"titulo": "Secreto"}, 201)
	bia.expect("POST", "/_ge/api/pastas/3/documentos", map[string]any{"titulo": "Da Bia"}, 201)
	// nobody creates something inside what they cannot see
	bia.expect("POST", "/_ge/api/documentos", map[string]any{"titulo": "Intruso", "pasta_id": 2}, 404)

	list := func(c *client, path string) []map[string]any {
		t.Helper()
		code, _, raw := c.do("GET", path, nil)
		if code != 200 {
			t.Fatalf("GET %s: %d %s", path, code, raw)
		}
		var out []map[string]any
		json.Unmarshal([]byte(raw), &out)
		return out
	}

	// Bia cites what she sees, never what she does not
	bia.expect("POST", "/_ge/api/pastas/3/documentos/3/citacoes", map[string]any{"citado_id": d1["id"]}, 201)
	bia.expect("POST", "/_ge/api/pastas/3/documentos/3/citacoes", map[string]any{"citado_id": d2["id"]}, 404)
	// the citation lives under its own document, not under the cited one
	if l := list(ana, "/_ge/api/pastas/1/documentos/1/citacoes"); len(l) != 0 {
		t.Fatalf("a citação de Bia é do documento dela: %v", l)
	}
	if l := list(bia, "/_ge/api/pastas/3/documentos/3/citacoes"); len(l) != 1 || l[0]["documento_id"] != float64(3) || l[0]["citado_id"] != d1["id"] {
		t.Fatalf("citação no documento de Bia: %v", l)
	}

	// Ana sees both documents and cites the secret one from the open one
	c := ana.expect("POST", "/_ge/api/pastas/1/documentos/1/citacoes", map[string]any{"citado_id": d2["id"]}, 201)
	if l := list(ana, "/_ge/api/pastas/1/documentos/1/citacoes"); len(l) != 1 {
		t.Fatalf("Ana vê a citação: %v", l)
	}
	// Bia sees the open document but not the secret one: the citation does
	// not exist for her, even though her rules would let her delete citations
	for _, path := range []string{"/_ge/api/pastas/1/documentos/1/citacoes", "/_ge/api/citacoes"} {
		for _, it := range list(bia, path) {
			if it["citado_id"] == d2["id"] {
				t.Fatalf("%s: uma citação de um documento que Bia não vê apareceu: %v", path, it)
			}
		}
	}
	bia.expect("DELETE", "/_ge/api/citacoes/"+jsonID(c["id"]), nil, 404)

	// deleting the cited document takes the citation with it
	ana.expect("DELETE", "/_ge/api/pastas/2/documentos/2", nil, 204)
	if l := list(ana, "/_ge/api/pastas/1/documentos/1/citacoes"); len(l) != 0 {
		t.Fatalf("a citação vai embora com o documento citado: %v", l)
	}
	if l := list(bia, "/_ge/api/citacoes"); len(l) != 1 {
		t.Fatalf("a citação de Bia continua: %v", l)
	}
}
