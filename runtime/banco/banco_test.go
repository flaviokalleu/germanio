package banco

import (
	"path/filepath"
	"testing"

	"github.com/flaviokalleu/germanio/compiler/ast"
)

func TestFiltrosDeLista(t *testing.T) {
	t.Setenv("GERMANIO_SQLITE", filepath.Join(t.TempDir(), "t.db"))
	m := &ast.Model{Name: "issue", Fields: []*ast.Field{{Name: "titulo", Type: ast.FieldTexto}, {Name: "labels", Type: ast.FieldTexto}}}
	b, err := Abrir(&ast.DatabaseConfig{Driver: "sqlite"}, "t", []*ast.Model{m})
	if err != nil {
		t.Fatal(err)
	}
	defer b.Fechar()
	for _, r := range []map[string]any{{"titulo": "a", "labels": `["1","2"]`}, {"titulo": "b", "labels": `["10"]`}, {"titulo": "c", "labels": `[]`}} {
		if _, err := b.CriarMapa("issue", r); err != nil {
			t.Fatal(err)
		}
	}
	count := func(f map[string]any) int64 {
		n, err := b.ContarFiltro("issue", Consulta{Filtros: f})
		if err != nil {
			t.Fatalf("%v: %v", f, err)
		}
		return n
	}
	if n := count(map[string]any{"id__em": []any{float64(1), float64(3)}}); n != 2 {
		t.Fatalf("__em: %d", n)
	}
	if n := count(map[string]any{"id__nao_em": []any{float64(1)}}); n != 2 {
		t.Fatalf("__nao_em: %d", n)
	}
	if n := count(map[string]any{"labels__contem_algum": []any{`"1"`, `"10"`}}); n != 2 {
		t.Fatalf("__contem_algum: %d", n)
	}
	if n := count(map[string]any{"labels__contem_algum": []any{`"2"`}}); n != 1 {
		t.Fatalf("__contem_algum sem confundir 2 com 20: %d", n)
	}
	if n := count(map[string]any{"labels__contem_algum": []any{}}); n != 0 {
		t.Fatalf("__contem_algum vazio: %d", n)
	}
}
