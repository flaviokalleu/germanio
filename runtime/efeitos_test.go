package runtime

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// External effects (G86) run after the commit, never holding the write lock.
// The destination below is the outside world: it records every call and, at
// that moment, whether the change that asked for it is already visible.

type outside struct {
	mu     sync.Mutex
	calls  []string // path + body, in arrival order
	seen   []bool   // the pedido was visible (committed) when the effect arrived
	slow   time.Duration
	app    string // the application's address (to look and to write back)
	during struct {
		status int
		took   time.Duration
	}
}

func (o *outside) handler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	who := strings.Trim(string(body), `"`)
	visible := false
	if resp, err := http.Get(o.app + "/_ge/api/pedidos"); err == nil {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		visible = bytes.Contains(b, []byte(`"`+who+`"`))
	}
	o.mu.Lock()
	o.calls = append(o.calls, r.URL.Path+" "+who)
	o.seen = append(o.seen, visible)
	o.mu.Unlock()
	switch {
	case r.URL.Path == "/lento":
		// a slow effect that writes back: it only finishes quickly if the
		// change that asked for it no longer holds the write lock
		start := time.Now()
		resp, err := http.Post(o.app+"/_ge/api/pedidos", "application/json", strings.NewReader(`{"cliente":"durante"}`))
		o.mu.Lock()
		o.during.took = time.Since(start)
		if err == nil {
			o.during.status = resp.StatusCode
			resp.Body.Close()
		}
		o.mu.Unlock()
		time.Sleep(o.slow)
	case who == "falha" && r.URL.Path == "/um":
		http.Error(w, "fora do ar", http.StatusBadGateway)
		return
	}
	w.Write([]byte(`{"ok":true}`))
}

func (o *outside) snapshot() ([]string, []bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]string(nil), o.calls...), append([]bool(nil), o.seen...)
}

func loadEffects(t *testing.T) (*App, *client, *outside) {
	t.Helper()
	o := &outside{slow: 300 * time.Millisecond}
	dest := httptest.NewServer(http.HandlerFunc(o.handler))
	t.Cleanup(dest.Close)
	t.Setenv("DESTINO", dest.URL)
	t.Setenv("GERMANIO_PERMITIR_REDE_LOCAL", "1")
	app, c := loadApp(t, "testdata/efeitos/app.ge")
	o.app = c.base
	return app, c, o
}

func countRows(t *testing.T, app *App, table string) int {
	t.Helper()
	var n int
	if err := app.DB.DB.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// commit ok + effects ok: both run, in order, after the change is saved.
func TestEfeitosDepoisDoCommit(t *testing.T) {
	_, c, o := loadEffects(t)
	c.expect("POST", "/_ge/api/pedidos", map[string]any{"cliente": "Ana"}, 201)
	calls, seen := o.snapshot()
	if strings.Join(calls, ",") != "/um Ana,/dois Ana" {
		t.Fatalf("efeitos (ordem): %v", calls)
	}
	for i, v := range seen {
		if !v {
			t.Fatalf("o efeito %s chegou antes de a mudança estar salva", calls[i])
		}
	}
}

// commit ok + a failing effect: the change stays, the next effect still runs.
func TestEfeitoQueFalhaNaoDesfazAMudanca(t *testing.T) {
	app, c, o := loadEffects(t)
	c.expect("POST", "/_ge/api/pedidos", map[string]any{"cliente": "falha"}, 201)
	if n := countRows(t, app, "pedido"); n != 1 {
		t.Fatalf("a falha do efeito desfez a mudança: %d pedidos", n)
	}
	if calls, _ := o.snapshot(); strings.Join(calls, ",") != "/um falha,/dois falha" {
		t.Fatalf("depois de um efeito que falha, os outros continuam: %v", calls)
	}
}

// a change undone before the commit: no effect happens.
func TestMudancaDesfeitaNaoTemEfeitos(t *testing.T) {
	app, c, o := loadEffects(t)
	c.expect("POST", "/_ge/api/pedidos", map[string]any{"cliente": "recusado"}, 400)
	if calls, _ := o.snapshot(); len(calls) != 0 {
		t.Fatalf("uma mudança desfeita teve efeitos: %v", calls)
	}
	if n := countRows(t, app, "pedido"); n != 0 {
		t.Fatalf("pedidos: %d", n)
	}
}

// a slow effect does not hold the write lock: while it runs, another change
// (made by the effect itself) is saved at once.
func TestEfeitoLentoNaoSeguraATrava(t *testing.T) {
	app, c, o := loadEffects(t)
	start := time.Now()
	c.expect("POST", "/_ge/api/lentos", map[string]any{"nome": "x"}, 201)
	o.mu.Lock()
	status, took := o.during.status, o.during.took
	o.mu.Unlock()
	if status != 201 {
		t.Fatalf("a escrita feita durante o efeito lento falhou: %d", status)
	}
	if took > 2*time.Second {
		t.Fatalf("a escrita esperou %v: o efeito segurou a trava", took)
	}
	if time.Since(start) < o.slow {
		t.Fatal("o efeito lento não rodou")
	}
	if n := countRows(t, app, "pedido"); n != 1 {
		t.Fatalf("pedidos: %d", n)
	}
}

// many changes at once, each with its effects: every effect of every kept
// change runs exactly once, none of an undone one.
func TestEfeitosConcorrentes(t *testing.T) {
	_, c, o := loadEffects(t)
	var wg sync.WaitGroup
	const n = 16
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			who, want := "c"+itoa(i), 201
			if i%4 == 0 {
				who, want = "recusado", 400
			}
			b, _ := json.Marshal(map[string]any{"cliente": who})
			resp, err := http.Post(c.base+"/_ge/api/pedidos", "application/json", bytes.NewReader(b))
			if err != nil {
				t.Error(err)
				return
			}
			resp.Body.Close()
			if resp.StatusCode != want {
				t.Errorf("%s: %d", who, resp.StatusCode)
			}
		}(i)
	}
	wg.Wait()
	calls, seen := o.snapshot()
	count := map[string]int{}
	for i, call := range calls {
		count[call]++
		if !seen[i] {
			t.Errorf("%s chegou antes do commit", call)
		}
	}
	for i := 0; i < n; i++ {
		if i%4 == 0 {
			continue
		}
		who := "c" + itoa(i)
		if count["/um "+who] != 1 || count["/dois "+who] != 1 {
			t.Errorf("%s: efeitos %d/%d", who, count["/um "+who], count["/dois "+who])
		}
	}
	if count["/um recusado"]+count["/dois recusado"] != 0 {
		t.Errorf("mudanças desfeitas tiveram efeitos: %v", count)
	}
}

// using the answer of an effect inside a change is refused: the answer does
// not exist before the commit. In a hook the compiler says so
// (TestHierarquiaErros); through a function, the runtime refuses and undoes.
func TestRespostaDeEfeitoDentroDaMudancaRecusada(t *testing.T) {
	app, c, o := loadEffects(t)
	c.expect("POST", "/_ge/api/respostas", map[string]any{"nome": "x"}, 500)
	if calls, _ := o.snapshot(); len(calls) != 0 {
		t.Fatalf("o efeito aconteceu: %v", calls)
	}
	if n := countRows(t, app, "resposta"); n != 0 {
		t.Fatalf("respostas: %d", n)
	}
}
