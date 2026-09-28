package banco

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/flaviokalleu/germanio/compiler/ast"
)

// Changing the declared fields of a data never loses information in
// silence: a column that still holds data but is no longer declared (a
// rename, for example) is reported; a default with an apostrophe works; a
// field that becomes unique later is enforced by the database too.
func TestMigracaoNaoPerdeNadaEmSilencio(t *testing.T) {
	db := filepath.Join(t.TempDir(), "t.db")
	t.Setenv("GERMANIO_SQLITE", db)
	v1 := &ast.Model{Name: "cliente", Fields: []*ast.Field{{Name: "nome", Type: ast.FieldTexto}, {Name: "telefone", Type: ast.FieldTexto}}}
	b, err := Abrir(&ast.DatabaseConfig{Driver: "sqlite"}, "t", []*ast.Model{v1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.CriarMapa("cliente", map[string]any{"nome": "Ana", "telefone": "8199"}); err != nil {
		t.Fatal(err)
	}
	b.Fechar()

	v2 := &ast.Model{Name: "cliente", Fields: []*ast.Field{
		{Name: "nome", Type: ast.FieldTexto},
		{Name: "celular", Type: ast.FieldTexto},
		{Name: "saudacao", Type: ast.FieldTexto, Default: "Olá, d'Ávila"},
		{Name: "codigo", Type: ast.FieldTexto, Unique: true},
	}}
	b, err = Abrir(&ast.DatabaseConfig{Driver: "sqlite"}, "t", []*ast.Model{v2})
	if err != nil {
		t.Fatalf("a migração falhou: %v", err)
	}
	defer b.Fechar()
	if !strings.Contains(strings.Join(b.Avisos, "\n"), "telefone") {
		t.Fatalf("a coluna telefone ainda tem dados e não foi avisada: %v", b.Avisos)
	}
	cols, _ := b.tableColumns("cliente")
	if !cols["saudacao"] || !cols["celular"] {
		t.Fatalf("colunas novas não criadas: %v", cols)
	}
	if _, err := b.CriarMapa("cliente", map[string]any{"nome": "Bia", "codigo": "X1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := b.DB.Exec(`INSERT INTO cliente (nome, codigo) VALUES ('Caio', 'X1')`); err == nil {
		t.Fatal("o banco aceitou um código repetido: único acrescentado depois não virou restrição")
	}
}
