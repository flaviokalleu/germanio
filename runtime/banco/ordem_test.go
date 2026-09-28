package banco

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/flaviokalleu/germanio/compiler/ast"
)

// The order of a list comes from the request (?ordenar=, ?ordem=). It must
// never reach the SQL as text: the column must exist and the direction is
// ASC or DESC, anything else is refused (it was a blind SQL injection).
func TestListarRecusaOrdemInjetada(t *testing.T) {
	t.Setenv("GERMANIO_SQLITE", filepath.Join(t.TempDir(), "t.db"))
	m := &ast.Model{Name: "nota", Fields: []*ast.Field{{Name: "titulo", Type: ast.FieldTexto}, {Name: "segredo", Type: ast.FieldTexto}}}
	b, err := Abrir(&ast.DatabaseConfig{Driver: "sqlite"}, "t", []*ast.Model{m})
	if err != nil {
		t.Fatal(err)
	}
	defer b.Fechar()
	for _, s := range []string{"b", "a"} {
		if _, err := b.CriarMapa("nota", map[string]any{"titulo": s, "segredo": "x"}); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []ListarParams{
		{Ordenar: "titulo", Ordem: "ASC, (SELECT CASE WHEN substr(segredo,1,1)='x' THEN 1 ELSE 1/0 END FROM nota)"},
		{Ordenar: "titulo", Ordem: "DESC; DROP TABLE nota"},
		{Ordenar: `titulo" DESC, (SELECT 1) --`},
		{Ordenar: "nao_existe"},
	} {
		if _, _, err := b.Listar("nota", &p); err == nil {
			t.Errorf("ordem %q %q foi aceita", p.Ordenar, p.Ordem)
		}
	}
	rows, _, err := b.Listar("nota", &ListarParams{Ordenar: "titulo", Ordem: "asc"})
	if err != nil || len(rows) != 2 || rows[0]["titulo"] != "a" {
		t.Fatalf("ordem válida: %v %v", rows, err)
	}
	if _, _, err := b.Listar("nota", &ListarParams{Limite: 1000000}); err != nil {
		t.Fatal(err)
	}
	if q(`a"b`) != `"a""b"` || strings.Count(q(`x`), `"`) != 2 {
		t.Fatalf("identificador mal escapado: %s", q(`a"b`))
	}
}
