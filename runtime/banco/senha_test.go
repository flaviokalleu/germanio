package banco

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"golang.org/x/crypto/bcrypt"
)

// A password is never stored as text, on create or on update, through the
// generic API of the older dialect either.
func TestSenhaNuncaGravadaComoTexto(t *testing.T) {
	t.Setenv("GERMANIO_SQLITE", filepath.Join(t.TempDir(), "t.db"))
	t.Setenv("GERMANIO_BCRYPT_RAPIDO", "1")
	m := &ast.Model{Name: "usuario", Fields: []*ast.Field{{Name: "email", Type: ast.FieldTexto}, {Name: "senha", Type: ast.FieldSenha}}}
	b, err := Abrir(&ast.DatabaseConfig{Driver: "sqlite"}, "t", []*ast.Model{m})
	if err != nil {
		t.Fatal(err)
	}
	defer b.Fechar()
	stored := func(id int64) string {
		var s string
		if err := b.DB.QueryRow(`SELECT senha FROM usuario WHERE id = ?`, id).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	row, err := b.Criar("usuario", json.RawMessage(`{"email":"a@x.com","senha":"primeira-senha"}`))
	if err != nil {
		t.Fatal(err)
	}
	id := int64(row["id"].(int64))
	if s := stored(id); bcrypt.CompareHashAndPassword([]byte(s), []byte("primeira-senha")) != nil {
		t.Fatalf("senha gravada sem hash ao criar: %q", s)
	}
	if _, err := b.Atualizar("usuario", id, json.RawMessage(`{"senha":"segunda-senha"}`)); err != nil {
		t.Fatal(err)
	}
	if s := stored(id); bcrypt.CompareHashAndPassword([]byte(s), []byte("segunda-senha")) != nil {
		t.Fatalf("senha gravada sem hash ao atualizar: %q", s)
	}
}
