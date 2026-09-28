package banco

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flaviokalleu/germanio/compiler/ast"
)

// Migration tests (G93): no destructive change is ever disguised as an
// inferred rename. Each test opens a database with one version of a model,
// then reopens it with the next version, as a real application restarting.

type version struct {
	fields    []*ast.Field
	renames   []ast.FieldRename
	discarded []string
}

func open(t *testing.T, db string, v version) (*Banco, error) {
	t.Helper()
	t.Setenv("GERMANIO_SQLITE", db)
	m := &ast.Model{Name: "cliente", Fields: v.fields, Renames: v.renames, Discarded: v.discarded}
	return Abrir(&ast.DatabaseConfig{Driver: "sqlite"}, "t", []*ast.Model{m})
}

func txt(name string) *ast.Field { return &ast.Field{Name: name, Type: ast.FieldTexto} }

func mustOpen(t *testing.T, db string, v version) *Banco {
	t.Helper()
	b, err := open(t, db, v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func column(t *testing.T, b *Banco, col string) []string {
	t.Helper()
	rows, err := b.DB.Query(fmt.Sprintf("SELECT COALESCE(%s, '') FROM cliente ORDER BY id", q(col)))
	if err != nil {
		t.Fatalf("coluna %s: %v", col, err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		rows.Scan(&s)
		out = append(out, s)
	}
	return out
}

func hasColumn(b *Banco, col string) bool {
	cols, _ := b.tableColumns("cliente")
	return cols[col]
}

func TestRenomearPreservaDados(t *testing.T) {
	db := filepath.Join(t.TempDir(), "t.db")
	b := mustOpen(t, db, version{fields: []*ast.Field{txt("nome")}})
	b.CriarMapa("cliente", map[string]any{"nome": "Ana"})
	b.CriarMapa("cliente", map[string]any{"nome": "Bia"})
	b.Fechar()
	b = mustOpen(t, db, version{fields: []*ast.Field{txt("nome_completo")}, renames: []ast.FieldRename{{From: "nome", To: "nome_completo"}}})
	defer b.Fechar()
	if got := column(t, b, "nome_completo"); strings.Join(got, ",") != "Ana,Bia" {
		t.Fatalf("dados depois do rename: %v", got)
	}
	if hasColumn(b, "nome") {
		t.Fatal("a coluna antiga continua existindo")
	}
}

func TestRenomearMilharesDeRegistros(t *testing.T) {
	db := filepath.Join(t.TempDir(), "t.db")
	b := mustOpen(t, db, version{fields: []*ast.Field{txt("nome")}})
	tx, _ := b.DB.Begin()
	for i := 0; i < 5000; i++ {
		tx.Exec(`INSERT INTO cliente (nome) VALUES (?)`, fmt.Sprint("pessoa ", i))
	}
	tx.Commit()
	b.Fechar()
	b = mustOpen(t, db, version{fields: []*ast.Field{txt("nome_completo")}, renames: []ast.FieldRename{{From: "nome", To: "nome_completo"}}})
	defer b.Fechar()
	var n int
	b.DB.QueryRow(`SELECT COUNT(*) FROM cliente WHERE nome_completo LIKE 'pessoa %'`).Scan(&n)
	if n != 5000 {
		t.Fatalf("registros preservados: %d de 5000", n)
	}
}

func TestRenomearEMudarTipoRecusa(t *testing.T) {
	db := filepath.Join(t.TempDir(), "t.db")
	b := mustOpen(t, db, version{fields: []*ast.Field{txt("codigo")}})
	b.CriarMapa("cliente", map[string]any{"codigo": "A-1"})
	b.Fechar()
	_, err := open(t, db, version{fields: []*ast.Field{{Name: "numero", Type: ast.FieldInteiro}}, renames: []ast.FieldRename{{From: "codigo", To: "numero"}}})
	if err == nil || !strings.Contains(err.Error(), "duas etapas") {
		t.Fatalf("rename com mudança de tipo deveria ser recusado: %v", err)
	}
	b = mustOpen(t, db, version{fields: []*ast.Field{txt("codigo")}})
	defer b.Fechar()
	if got := column(t, b, "codigo"); len(got) != 1 || got[0] != "A-1" {
		t.Fatalf("a recusa alterou os dados: %v", got)
	}
}

func TestRenomearParaCampoExistente(t *testing.T) {
	db := filepath.Join(t.TempDir(), "t.db")
	b := mustOpen(t, db, version{fields: []*ast.Field{txt("nome"), txt("apelido")}})
	b.CriarMapa("cliente", map[string]any{"nome": "Ana", "apelido": "Aninha"})
	b.Fechar()
	// the target has data: refused, nothing changed
	_, err := open(t, db, version{fields: []*ast.Field{txt("apelido")}, renames: []ast.FieldRename{{From: "nome", To: "apelido"}}})
	if err == nil || !strings.Contains(err.Error(), "já existe e tem dados") {
		t.Fatalf("rename sobre campo com dados deveria ser recusado: %v", err)
	}
	b = mustOpen(t, db, version{fields: []*ast.Field{txt("nome"), txt("apelido")}})
	if got := column(t, b, "nome"); got[0] != "Ana" {
		t.Fatalf("dados alterados: %v", got)
	}
	// an empty target (created by an earlier start without the rename) is replaced
	b.DB.Exec(`ALTER TABLE cliente ADD COLUMN nome_completo TEXT`)
	b.Fechar()
	b = mustOpen(t, db, version{fields: []*ast.Field{txt("apelido"), txt("nome_completo")}, renames: []ast.FieldRename{{From: "nome", To: "nome_completo"}}})
	defer b.Fechar()
	if got := column(t, b, "nome_completo"); got[0] != "Ana" {
		t.Fatalf("rename sobre coluna vazia: %v", got)
	}
}

func TestRenomearInexistenteEBancoNovo(t *testing.T) {
	db := filepath.Join(t.TempDir(), "t.db")
	// a new database: nothing to rename, the new field is simply created
	b := mustOpen(t, db, version{fields: []*ast.Field{txt("nome_completo")}, renames: []ast.FieldRename{{From: "nome", To: "nome_completo"}}})
	b.CriarMapa("cliente", map[string]any{"nome_completo": "Ana"})
	b.Fechar()
	// the old rename stays in the program and is harmless on every start
	b = mustOpen(t, db, version{fields: []*ast.Field{txt("nome_completo")}, renames: []ast.FieldRename{{From: "nome", To: "nome_completo"}}})
	defer b.Fechar()
	if got := column(t, b, "nome_completo"); got[0] != "Ana" {
		t.Fatalf("rename antigo mexeu nos dados: %v", got)
	}
}

func TestVariosRenamesEFalhaNaoAplicaNenhum(t *testing.T) {
	db := filepath.Join(t.TempDir(), "t.db")
	b := mustOpen(t, db, version{fields: []*ast.Field{txt("nome"), txt("fone"), txt("cidade"), txt("uf")}})
	b.CriarMapa("cliente", map[string]any{"nome": "Ana", "fone": "81", "cidade": "Recife", "uf": "PE"})
	b.Fechar()
	// the second rename is refused (its target has data): the first is not applied either
	_, err := open(t, db, version{fields: []*ast.Field{txt("nome_completo"), txt("uf"), txt("cidade")}, renames: []ast.FieldRename{{From: "nome", To: "nome_completo"}, {From: "fone", To: "uf"}}})
	if err == nil {
		t.Fatal("rename sobre campo com dados foi aceito")
	}
	b = mustOpen(t, db, version{fields: []*ast.Field{txt("nome"), txt("fone"), txt("cidade"), txt("uf")}})
	if !hasColumn(b, "nome") || hasColumn(b, "nome_completo") {
		t.Fatal("uma falha deixou metade dos renames aplicada")
	}
	b.Fechar()
	// both valid: both applied
	b = mustOpen(t, db, version{fields: []*ast.Field{txt("nome_completo"), txt("telefone"), txt("cidade"), txt("uf")}, renames: []ast.FieldRename{{From: "nome", To: "nome_completo"}, {From: "fone", To: "telefone"}}})
	defer b.Fechar()
	if column(t, b, "nome_completo")[0] != "Ana" || column(t, b, "telefone")[0] != "81" {
		t.Fatal("vários renames não preservaram os dados")
	}
}

func TestRenomeProvavelSemDeclaracaoPara(t *testing.T) {
	db := filepath.Join(t.TempDir(), "t.db")
	b := mustOpen(t, db, version{fields: []*ast.Field{txt("nome")}})
	b.CriarMapa("cliente", map[string]any{"nome": "Ana"})
	b.Fechar()
	_, err := open(t, db, version{fields: []*ast.Field{txt("nome_completo")}})
	if err == nil {
		t.Fatal("um campo com dados sumiu e outro apareceu: a partida deveria parar")
	}
	for _, want := range []string{`parece que "nome" pode ter sido renomeado para "nome_completo"`, "não pode provar", "renomeie nome para nome_completo", "descarte nome"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("mensagem sem %q:\n%v", want, err)
		}
	}
	// nothing was created or dropped
	b = mustOpen(t, db, version{fields: []*ast.Field{txt("nome")}})
	if hasColumn(b, "nome_completo") || column(t, b, "nome")[0] != "Ana" {
		t.Fatal("a recusa mudou a tabela")
	}
	b.Fechar()
	// removed on purpose: descarte lets it start; the old data stays in its column
	b = mustOpen(t, db, version{fields: []*ast.Field{txt("apelido")}, discarded: []string{"nome"}})
	defer b.Fechar()
	if column(t, b, "nome")[0] != "Ana" {
		t.Fatal("descarte apagou dados")
	}
	for _, a := range b.Avisos {
		if strings.Contains(a, `"nome"`) {
			t.Fatalf("o campo descartado continua sendo avisado: %s", a)
		}
	}
}

// A default with an apostrophe works; a field that becomes unique later is
// enforced by the database too.
func TestMigracaoValorPadraoEUnico(t *testing.T) {
	db := filepath.Join(t.TempDir(), "t.db")
	b := mustOpen(t, db, version{fields: []*ast.Field{txt("nome")}})
	b.Fechar()
	b = mustOpen(t, db, version{fields: []*ast.Field{txt("nome"), {Name: "saudacao", Type: ast.FieldTexto, Default: "Olá, d'Ávila"}, {Name: "codigo", Type: ast.FieldTexto, Unique: true}}})
	defer b.Fechar()
	b.CriarMapa("cliente", map[string]any{"nome": "Bia", "codigo": "X1"})
	if _, err := b.DB.Exec(`INSERT INTO cliente (nome, codigo) VALUES ('Caio', 'X1')`); err == nil {
		t.Fatal("o banco aceitou um código repetido")
	}
}
