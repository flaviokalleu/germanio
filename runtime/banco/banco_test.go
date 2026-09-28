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

func TestDataSemHora(t *testing.T) {
	t.Setenv("GERMANIO_SQLITE", filepath.Join(t.TempDir(), "d.db"))
	m := &ast.Model{Name: "meta", Fields: []*ast.Field{{Name: "prazo", Type: ast.FieldData}, {Name: "quando", Type: ast.FieldDataHora}}}
	b, err := Abrir(&ast.DatabaseConfig{Driver: "sqlite"}, "d", []*ast.Model{m})
	if err != nil {
		t.Fatal(err)
	}
	defer b.Fechar()
	row, err := b.CriarMapa("meta", map[string]any{"prazo": "2026-12-31", "quando": "2026-12-31T10:30:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := b.BuscarRegistro("meta", row["id"].(int64))
	if got["prazo"] != "2026-12-31" || got["quando"] != "2026-12-31T10:30:00Z" {
		t.Fatalf("datas: %v", got)
	}
}

// A log grows by appending chunks; a chunk written for an outdated length
// is refused, so two writers never interleave.
func TestAnexarTexto(t *testing.T) {
	t.Setenv("GERMANIO_SQLITE", filepath.Join(t.TempDir(), "t.db"))
	m := &ast.Model{Name: "job", Fields: []*ast.Field{{Name: "log", Type: ast.FieldTextoLongo}}}
	b, err := Abrir(&ast.DatabaseConfig{Driver: "sqlite"}, "t", []*ast.Model{m})
	if err != nil {
		t.Fatal(err)
	}
	defer b.Fechar()
	row, err := b.CriarMapa("job", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	id := row["id"].(int64)
	for i, chunk := range []string{"linha 1\n", "linha 2\n"} {
		ok, err := b.AnexarTexto("job", id, "log", chunk, i*8)
		if err != nil || !ok {
			t.Fatalf("anexo %d: %v %v", i, ok, err)
		}
	}
	if ok, _ := b.AnexarTexto("job", id, "log", "atrasado\n", 8); ok {
		t.Fatal("um pedaço para um tamanho antigo foi aceito")
	}
	got, _ := b.Buscar("job", id)
	if got["log"] != "linha 1\nlinha 2\n" {
		t.Fatalf("log: %q", got["log"])
	}
}

// Ou: at least one group must hold, besides the other filters.
func TestConsultaOu(t *testing.T) {
	t.Setenv("GERMANIO_SQLITE", filepath.Join(t.TempDir(), "t.db"))
	m := &ast.Model{Name: "item", Fields: []*ast.Field{{Name: "a", Type: ast.FieldTexto}, {Name: "b", Type: ast.FieldTexto}, {Name: "c", Type: ast.FieldTexto}}}
	b, err := Abrir(&ast.DatabaseConfig{Driver: "sqlite"}, "t", []*ast.Model{m})
	if err != nil {
		t.Fatal(err)
	}
	defer b.Fechar()
	for _, r := range []map[string]any{{"a": "1", "b": "x", "c": "k"}, {"a": "2", "b": "y", "c": "k"}, {"a": "3", "b": "x", "c": "z"}, {"a": "4", "b": "w", "c": "k"}} {
		b.CriarMapa("item", r)
	}
	count := func(c Consulta) int64 {
		n, err := b.ContarFiltro("item", c)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	// c = k AND (b = x OR a in [2])
	if n := count(Consulta{Filtros: map[string]any{"c": "k"}, Ou: []map[string]any{{"b": "x"}, {"a__em": []any{"2"}}}}); n != 2 {
		t.Fatalf("ou: %d", n)
	}
	// an empty group always holds
	if n := count(Consulta{Ou: []map[string]any{{"b": "nada"}, {}}}); n != 4 {
		t.Fatalf("grupo vazio: %d", n)
	}
}

// ListarEmLotes reads every row exactly once, newest first, one batch at a
// time.
func TestListarEmLotes(t *testing.T) {
	t.Setenv("GERMANIO_SQLITE", filepath.Join(t.TempDir(), "t.db"))
	m := &ast.Model{Name: "linha", Fields: []*ast.Field{{Name: "n", Type: ast.FieldInteiro}}}
	b, err := Abrir(&ast.DatabaseConfig{Driver: "sqlite"}, "t", []*ast.Model{m})
	if err != nil {
		t.Fatal(err)
	}
	defer b.Fechar()
	tx, _ := b.DB.Begin()
	for i := 1; i <= 2500; i++ {
		tx.Exec(`INSERT INTO linha (n) VALUES (?)`, i)
	}
	tx.Commit()
	var seen []int64
	batches := 0
	err = b.ListarEmLotes("linha", 1000, func(rows []map[string]any) error {
		batches++
		for _, r := range rows {
			seen = append(seen, r["id"].(int64))
		}
		return nil
	})
	if err != nil || batches != 3 || len(seen) != 2500 || seen[0] != 2500 || seen[2499] != 1 {
		t.Fatalf("lotes %d, linhas %d, primeiro %v, último %v, erro %v", batches, len(seen), seen[0], seen[len(seen)-1], err)
	}
	for i := 1; i < len(seen); i++ {
		if seen[i] >= seen[i-1] {
			t.Fatalf("ordem quebrada em %d", i)
		}
	}
}
