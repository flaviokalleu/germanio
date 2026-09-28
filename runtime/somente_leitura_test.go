package runtime

import "testing"

// `pasta trancada é somente leitura`: a mesma capability do projeto
// arquivado, num domínio sem GitLab.
func TestSomenteLeitura(t *testing.T) {
	_, c := loadApp(t, "testdata/intencao/arquivo.ge")
	c.expect("POST", "/_ge/api/pastas", map[string]any{"nome": "Contratos"}, 201)
	c.expect("POST", "/_ge/api/pastas/1/documentos", map[string]any{"titulo": "A"}, 201)
	c.expect("PATCH", "/_ge/api/pastas/1", map[string]any{"trancada": true}, 200)

	c.expect("PATCH", "/_ge/api/pastas/1", map[string]any{"nome": "Outro"}, 403)
	c.expect("PATCH", "/_ge/api/pastas/1", map[string]any{"nome": "Outro", "trancada": false}, 403) // só destrancar
	c.expect("POST", "/_ge/api/pastas/1/documentos", map[string]any{"titulo": "B"}, 403)
	c.expect("PATCH", "/_ge/api/pastas/1/documentos/1", map[string]any{"titulo": "A2"}, 403)
	c.expect("DELETE", "/_ge/api/pastas/1/documentos/1", nil, 403)
	c.expect("GET", "/_ge/api/pastas/1/documentos/1", nil, 200)

	c.expect("PATCH", "/_ge/api/pastas/1", map[string]any{"trancada": false}, 200)
	c.expect("PATCH", "/_ge/api/pastas/1/documentos/1", map[string]any{"titulo": "A2"}, 200)
	// Excluir a própria pasta trancada continua possível.
	c.expect("PATCH", "/_ge/api/pastas/1", map[string]any{"trancada": true}, 200)
	c.expect("DELETE", "/_ge/api/pastas/1", nil, 204)
}
