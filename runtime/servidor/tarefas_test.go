package servidor

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/runtime/banco"
)

func TestSafeHTTPClientBlocksLocalNetwork(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	t.Setenv("GERMANIO_PERMITIR_REDE_LOCAL", "")
	_, err := safeHTTPClient().Get(srv.URL)
	if err == nil || !errors.Is(err, errLocalNetwork) {
		t.Fatalf("esperado bloqueio de rede local, obtido %v", err)
	}
	t.Setenv("GERMANIO_PERMITIR_REDE_LOCAL", "1")
	resp, err := safeHTTPClient().Get(srv.URL)
	if err != nil {
		t.Fatalf("rede local permitida explicitamente: %v", err)
	}
	resp.Body.Close()
}

// The task queue is found through an index and does not grow forever:
// finished tasks are removed after a while (failed ones kept longer).
func TestFilaDeTarefasIndexadaELimpa(t *testing.T) {
	t.Setenv("GERMANIO_SQLITE", filepath.Join(t.TempDir(), "t.db"))
	db, err := banco.Abrir(&ast.DatabaseConfig{Driver: "sqlite"}, "t", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Fechar()
	s := Novo(&ast.Program{System: &ast.System{Name: "teste"}}, db, "0")
	q := s.tasks()
	defer close(q.stop)
	var idx int
	db.DB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_germanio_tarefas_devidas'`).Scan(&idx)
	if idx != 1 {
		t.Fatal("a fila de tarefas não tem índice para as tarefas devidas")
	}
	now := time.Now()
	old := now.Add(-40 * 24 * time.Hour).UTC().Format(time.RFC3339Nano)
	recent := now.Add(-2 * 24 * time.Hour).UTC().Format(time.RFC3339Nano)
	for _, r := range []struct{ estado, criado string }{
		{"concluida", old}, {"concluida", recent}, {"morta", old}, {"morta", recent}, {"pendente", old},
	} {
		db.DB.Exec(`INSERT INTO `+tasksTable+` (tipo, dados, estado, tentativas, proxima_em, criado_em) VALUES ('x', '{}', ?, 0, '9999', ?)`, r.estado, r.criado)
	}
	q.cleanup(now)
	var n int
	db.DB.QueryRow(`SELECT COUNT(*) FROM ` + tasksTable).Scan(&n)
	if n != 3 {
		t.Fatalf("ficaram %d tarefas; esperado 3 (concluída recente, morta recente, pendente)", n)
	}
}
