package runtime

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/flaviokalleu/germanio/runtime/banco"
	"github.com/flaviokalleu/germanio/runtime/cofre"
	"github.com/flaviokalleu/germanio/tooling/explicar"
)

// Secrets at rest (GEP 0049, em teste) in a domain with no Git and no
// GitLab: a restaurant tells its delivery partner about each order, with the
// key the partner gave (`token oculto`). The key is sealed in the database,
// still reaches the partner, never comes back through the API, pages,
// history, events or explain, cannot be filtered or searched, rows written
// in clear are sealed at start, the key rotates with
// GERMANIO_SEGREDO_ANTERIOR, and production refuses to start without a key.

const (
	segredoA = "chave-A-do-restaurante-com-mais-de-32-caracteres"
	segredoB = "chave-B-do-restaurante-com-mais-de-32-caracteres"
	segredoC = "chave-C-do-restaurante-com-mais-de-32-caracteres"
)

type partner struct {
	mu   sync.Mutex
	keys []string
	body []string
	srv  *httptest.Server
}

func newPartner(t *testing.T) *partner {
	p := &partner{}
	p.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		p.mu.Lock()
		p.keys = append(p.keys, r.Header.Get("X-Germanio-Token"))
		p.body = append(p.body, string(b))
		p.mu.Unlock()
	}))
	t.Cleanup(p.srv.Close)
	return p
}

// wait returns the n-th delivery (key and body), waiting for the queue.
func (p *partner) wait(t *testing.T, n int) (string, string) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		p.mu.Lock()
		if len(p.keys) >= n {
			k, b := p.keys[n-1], p.body[n-1]
			p.mu.Unlock()
			return k, b
		}
		p.mu.Unlock()
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("o parceiro não recebeu a entrega %d", n)
	return "", ""
}

// restaurant starts the app on the database file db; stop closes it (the
// file stays, for the next start).
func restaurant(t *testing.T, db string) (app *App, base string, stop func()) {
	t.Helper()
	t.Setenv("GERMANIO_SQLITE", db)
	t.Setenv("GERMANIO_BCRYPT_RAPIDO", "1")
	t.Setenv("GERMANIO_PERMITIR_REDE_LOCAL", "1") // the partner runs on localhost
	app, err := Carregar("testdata/segredos/app.ge", "0")
	if err != nil {
		t.Fatalf("Carregar: %v", err)
	}
	srv := httptest.NewServer(app.Handler)
	return app, srv.URL, func() { srv.Close(); app.Fechar() }
}

// rawFile is everything SQLite keeps on disk for db (the file and its
// write-ahead log).
func rawFile(t *testing.T, db string) []byte {
	t.Helper()
	var all []byte
	for _, f := range []string{db, db + "-wal"} {
		if b, err := os.ReadFile(f); err == nil {
			all = append(all, b...)
		}
	}
	if len(all) == 0 {
		t.Fatal("o arquivo do banco não existe")
	}
	return all
}

func storedKey(t *testing.T, app *App, id int) string {
	t.Helper()
	var v any
	if err := app.DB.DB.QueryRow(`SELECT token FROM aviso WHERE id = ?`, id).Scan(&v); err != nil {
		t.Fatal(err)
	}
	s, _ := v.(string)
	return s
}

func session(t *testing.T, base, email string) *client {
	t.Helper()
	jar, _ := cookiejar.New(nil)
	c := &client{t: t, base: base, http: &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, Header: http.Header{}}
	c.expect("POST", "/entrar", map[string]any{"login": email, "senha": "senha-segura-1"}, 200)
	c.csrf = csrfFromCookie(t, c)
	return c
}

func TestSegredosGuardados(t *testing.T) {
	t.Setenv("GERMANIO_SEGREDO", segredoA)
	db := filepath.Join(t.TempDir(), "restaurante.db")
	app, base, stop := restaurant(t, db)
	defer func() { stop() }()
	parceiro := newPartner(t)
	const key = "chave-do-parceiro-XYZ-987"

	ana := signIn(t, base, "Ana", "ana@x.com")
	ana.expect("POST", "/_ge/api/restaurantes", map[string]any{"nome": "Cantina"}, 201)
	aviso := ana.expect("POST", "/_ge/api/restaurantes/1/avisos", map[string]any{"url": parceiro.srv.URL, "token": key, "nota": "entregas"}, 201)
	if _, ok := aviso["token"]; ok {
		t.Fatalf("a chave volta na resposta: %v", aviso)
	}

	// the database holds only ciphertext, bound to its column
	stored := storedKey(t, app, 1)
	if !strings.HasPrefix(stored, cofre.Prefixo) || strings.Contains(stored, key) {
		t.Fatalf("a chave está em texto puro no banco: %q", stored)
	}
	// still works: the partner receives the key with each order
	ana.expect("POST", "/_ge/api/restaurantes/1/pedidos", map[string]any{"prato": "lasanha"}, 201)
	got, body := parceiro.wait(t, 1)
	if got != key {
		t.Fatalf("o parceiro recebeu a chave %q", got)
	}
	if strings.Contains(body, key) || strings.Contains(body, stored) {
		t.Fatalf("a chave aparece no corpo do evento: %s", body)
	}

	// never returned: list, record, page, history
	for _, path := range []string{"/_ge/api/restaurantes/1/avisos", "/_ge/api/restaurantes/1/avisos/1", "/avisos", "/_ge/api/atividades"} {
		if _, _, raw := ana.do("GET", path, nil); strings.Contains(raw, key) || strings.Contains(raw, stored) {
			t.Fatalf("%s mostra a chave: %s", path, raw)
		}
	}
	// editing it keeps it sealed, and the history does not record it
	ana.expect("PATCH", "/_ge/api/restaurantes/1/avisos/1", map[string]any{"token": "nova-chave-ABC-123"}, 200)
	if s := storedKey(t, app, 1); !strings.HasPrefix(s, cofre.Prefixo) || strings.Contains(s, "nova-chave") {
		t.Fatalf("depois de editar: %q", s)
	}
	if _, _, raw := ana.do("GET", "/_ge/api/atividades", nil); strings.Contains(raw, "nova-chave") {
		t.Fatalf("o histórico guarda a chave: %s", raw)
	}
	ana.expect("POST", "/_ge/api/restaurantes/1/pedidos", map[string]any{"prato": "risoto"}, 201)
	if got, _ := parceiro.wait(t, 2); got != "nova-chave-ABC-123" {
		t.Fatalf("depois de editar, o parceiro recebeu %q", got)
	}
	// explain describes the field, never a value
	if out, err := explicar.Entidade(app.Program, "aviso"); err != nil || !strings.Contains(out, "cifrado no banco") || strings.Contains(out, "nova-chave") {
		t.Fatalf("explain: %v %s", err, out)
	}

	// a sealed field is never compared: filter, search, order are refused
	if _, _, err := app.DB.Filtrar("aviso", banco.Consulta{Filtros: map[string]any{"token": "nova-chave-ABC-123"}}); !isSecretErr(err) {
		t.Fatalf("filtrar pela chave: %v", err)
	}
	if _, _, err := app.DB.Filtrar("aviso", banco.Consulta{Busca: "nova", BuscaCampos: []string{"token"}}); !isSecretErr(err) {
		t.Fatalf("pesquisar na chave: %v", err)
	}
	if _, _, err := app.DB.Filtrar("aviso", banco.Consulta{Ordenar: "token"}); !isSecretErr(err) {
		t.Fatalf("ordenar pela chave: %v", err)
	}
	if rows, _, err := app.DB.Filtrar("aviso", banco.Consulta{Busca: "entregas"}); err != nil || len(rows) != 1 {
		t.Fatalf("a pesquisa nos outros campos continua: %v %v", rows, err)
	}
	if code, _, raw := ana.do("GET", "/_ge/api/restaurantes/1/avisos?ordenar=token", nil); code < 400 && strings.Contains(raw, "nova-chave") {
		t.Fatalf("ordenar pela chave na API: %d %s", code, raw)
	}

	// the file on disk never holds the key in clear
	stop()
	stop = func() {}
	raw := rawFile(t, db)
	for _, k := range []string{key, "nova-chave-ABC-123"} {
		if bytes.Contains(raw, []byte(k)) {
			t.Fatalf("o arquivo do banco contém %q em texto puro", k)
		}
	}
}

func isSecretErr(err error) bool {
	var es *banco.ErrSegredo
	return errors.As(err, &es)
}

// Rows written in clear (before the key existed) are sealed at the next
// start; the key rotates with GERMANIO_SEGREDO_ANTERIOR; a key that is gone
// stops the start with the reason.
func TestSegredosMigracaoERotacao(t *testing.T) {
	db := filepath.Join(t.TempDir(), "restaurante.db")
	parceiro := newPartner(t)
	const key = "legado-em-texto-puro-555"

	// 1. development without a key: kept in clear, with a warning
	t.Setenv("GERMANIO_SEGREDO", "")
	t.Setenv("GERMANIO_SEGREDO_ANTERIOR", "")
	app, base, stop := restaurant(t, db)
	ana := signIn(t, base, "Ana", "ana@x.com")
	ana.expect("POST", "/_ge/api/restaurantes", map[string]any{"nome": "Cantina"}, 201)
	ana.expect("POST", "/_ge/api/restaurantes/1/avisos", map[string]any{"url": parceiro.srv.URL, "token": key}, 201)
	if s := storedKey(t, app, 1); s != key {
		t.Fatalf("sem chave, em claro: %q", s)
	}
	stop()

	// 2. production without a key refuses to start, with the reason
	t.Setenv("GERMANIO_PRODUCAO", "1")
	t.Setenv("GERMANIO_SQLITE", db)
	if _, err := Carregar("testdata/segredos/app.ge", "0"); err == nil || !strings.Contains(err.Error(), "GERMANIO_SEGREDO") || !strings.Contains(err.Error(), "aviso.token") {
		t.Fatalf("produção sem chave: %v", err)
	}

	// 3. with key A: the legacy row is sealed at start, and still works
	t.Setenv("GERMANIO_SEGREDO", segredoA)
	app, base, stop = restaurant(t, db)
	sealedA := storedKey(t, app, 1)
	if !strings.HasPrefix(sealedA, cofre.Prefixo) || strings.Contains(sealedA, key) {
		t.Fatalf("o legado não foi cifrado na partida: %q", sealedA)
	}
	ana = session(t, base, "ana@x.com")
	ana.expect("POST", "/_ge/api/restaurantes/1/pedidos", map[string]any{"prato": "lasanha"}, 201)
	if got, _ := parceiro.wait(t, 1); got != key {
		t.Fatalf("depois da migração, o parceiro recebeu %q", got)
	}
	stop()
	if bytes.Contains(rawFile(t, db), []byte(key)) {
		t.Fatal("o texto puro antigo continua no arquivo do banco depois da migração")
	}
	// starting again changes nothing (idempotent)
	app, _, stop = restaurant(t, db)
	if s := storedKey(t, app, 1); s != sealedA {
		t.Fatalf("a segunda partida cifrou de novo sem necessidade: %q", s)
	}
	stop()

	// 4. rotation: B is the new key, A the previous one
	t.Setenv("GERMANIO_SEGREDO", segredoB)
	t.Setenv("GERMANIO_SEGREDO_ANTERIOR", segredoA)
	app, base, stop = restaurant(t, db)
	sealedB := storedKey(t, app, 1)
	vb, _ := cofre.Novo(segredoB)
	if !vb.Atual(sealedB) || sealedB == sealedA {
		t.Fatalf("a rotação não cifrou com a chave nova: %q", sealedB)
	}
	ana = session(t, base, "ana@x.com")
	ana.expect("POST", "/_ge/api/restaurantes/1/pedidos", map[string]any{"prato": "risoto"}, 201)
	if got, _ := parceiro.wait(t, 2); got != key {
		t.Fatalf("depois da rotação, o parceiro recebeu %q", got)
	}
	stop()

	// 5. the previous key can go; a key that is gone stops the start
	t.Setenv("GERMANIO_SEGREDO_ANTERIOR", "")
	_, _, stop = restaurant(t, db)
	stop()
	t.Setenv("GERMANIO_SEGREDO", segredoC)
	if _, err := Carregar("testdata/segredos/app.ge", "0"); err == nil || !strings.Contains(err.Error(), "GERMANIO_SEGREDO_ANTERIOR") {
		t.Fatalf("chave desconhecida: %v", err)
	}
	// and without any key, sealed values stop a development start too
	t.Setenv("GERMANIO_PRODUCAO", "")
	t.Setenv("GERMANIO_SEGREDO", "")
	if _, err := Carregar("testdata/segredos/app.ge", "0"); err == nil || !strings.Contains(err.Error(), "não tem GERMANIO_SEGREDO") {
		t.Fatalf("sem chave com valores cifrados: %v", err)
	}
}

// The second factor's secrets (GEP 0032) follow the same keys: the first
// format ("v1:") and secrets sealed with a previous key are sealed again
// with the current key at start, and the codes keep working.
func TestDoisFatoresRotacao(t *testing.T) {
	db := filepath.Join(t.TempDir(), "cofre.db")
	start := func() (*App, string, func()) {
		t.Helper()
		t.Setenv("GERMANIO_SQLITE", db)
		t.Setenv("GERMANIO_BCRYPT_RAPIDO", "1")
		app, err := Carregar("testdata/dois_fatores/app.ge", "0")
		if err != nil {
			t.Fatalf("Carregar: %v", err)
		}
		srv := httptest.NewServer(app.Handler)
		return app, srv.URL, func() { srv.Close(); app.Fechar() }
	}
	stored := func(app *App) (int64, string) {
		var id int64
		var s string
		if err := app.DB.DB.QueryRow(`SELECT pessoa_id, segredo FROM _germanio_dois_fatores`).Scan(&id, &s); err != nil {
			t.Fatal(err)
		}
		return id, s
	}
	signInWithCode := func(app *App, base string, secret []byte, offset int) {
		t.Helper()
		// a code is never accepted twice: forget the last one accepted
		app.DB.DB.Exec(`UPDATE _germanio_dois_fatores SET ultimo_passo = 0`)
		c := newClient(t, base)
		ch := c.expect("POST", "/entrar", map[string]any{"login": "ana@x.com", "senha": "senha-segura-1"}, 202)
		c.expect("POST", "/entrar/codigo", map[string]any{"desafio": toStrT(ch["desafio"]), "codigo": codeAt(secret, offset)}, 200)
	}

	t.Setenv("GERMANIO_SEGREDO", segredoA)
	t.Setenv("GERMANIO_SEGREDO_ANTERIOR", "")
	app, base, stop := start()
	ana := signIn(t, base, "Ana", "ana@x.com")
	secret, _ := enableFactor(t, ana, "senha-segura-1")
	person, _ := stored(app)
	// the row as the first format wrote it
	key, _ := hkdf.Key(sha256.New, []byte(segredoA), nil, "germanio/dois-fatores/v1", 32)
	block, _ := aes.NewCipher(key)
	aead, _ := cipher.NewGCM(block)
	nonce := make([]byte, aead.NonceSize())
	rand.Read(nonce)
	legacy := "v1:" + base64.RawStdEncoding.EncodeToString(aead.Seal(nonce, nonce, secret, []byte(strconv.FormatInt(person, 10))))
	if _, err := app.DB.DB.Exec(`UPDATE _germanio_dois_fatores SET segredo = ?`, legacy); err != nil {
		t.Fatal(err)
	}
	stop()

	// rotation to B with A as the previous key: sealed again, codes still work
	t.Setenv("GERMANIO_SEGREDO", segredoB)
	t.Setenv("GERMANIO_SEGREDO_ANTERIOR", segredoA)
	app, base, stop = start()
	vb, _ := cofre.Novo(segredoB)
	if _, s := stored(app); !vb.Atual(s) {
		t.Fatalf("o segredo antigo não foi cifrado com a chave nova: %q", s)
	}
	signInWithCode(app, base, secret, 0)
	stop()

	// the previous key can go now
	t.Setenv("GERMANIO_SEGREDO_ANTERIOR", "")
	app, base, stop = start()
	signInWithCode(app, base, secret, 0)
	stop()
}
