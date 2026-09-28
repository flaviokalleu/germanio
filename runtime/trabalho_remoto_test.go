package runtime

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync"
	"testing"
	"time"

	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// A fila genérica de trabalho remoto, sem git, CI, pipeline ou repositório:
// conversões de imagem feitas por trabalhadores com um protocolo próprio.

type worker struct {
	t          *testing.T
	base, cred string
}

func (w *worker) post(path string, body map[string]any) (int, map[string]any) {
	w.t.Helper()
	b, _ := json.Marshal(body)
	resp, err := http.Post(w.base+path, "application/json", bytes.NewReader(b))
	if err != nil {
		w.t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func (w *worker) take(key string) (int, map[string]any) {
	return w.post("/worker/pegar", map[string]any{"credencial": w.cred, "chave": key})
}

func imagens(t *testing.T) (*App, *client, []*worker) {
	app, c := loadApp(t, "testdata/intencao/imagens/app.ge")
	var ws []*worker
	for _, n := range []string{"A", "B", "C"} {
		res, err := app.Interpreter.Op(&interp.Context{}, "trabalhador", "criar", map[string]any{"nome": n})
		if err != nil {
			t.Fatal(err)
		}
		ws = append(ws, &worker{t: t, base: c.base, cred: res.(map[string]any)["token"].(string)})
	}
	c.expect("POST", "/cadastro", map[string]any{"nome": "Ana", "email": "ana@x.com", "senha": "senha-da-ana"}, 201)
	c.expect("POST", "/entrar", map[string]any{"email": "ana@x.com", "senha": "senha-da-ana"}, 200)
	c.csrf = csrfFromCookie(t, c)
	return app, c, ws
}

func TestTrabalhoRemotoGenerico(t *testing.T) {
	_, c, ws := imagens(t)
	a, b := ws[0], ws[1]
	(&worker{t: t, base: c.base, cred: "falsa"}).post("/worker/pegar", map[string]any{"credencial": "falsa"})
	if code, _ := (&worker{t: t, base: c.base, cred: "falsa"}).take(""); code != 403 {
		t.Fatalf("credencial falsa: %d", code)
	}
	conv := c.expect("POST", "/_ge/api/conversoes", map[string]any{"arquivo": "foto.png", "formato": "webp"}, 201)
	if conv["estado"] != "pendente" {
		t.Fatalf("conversão nova: %v", conv)
	}

	// Dois trabalhadores pedem ao mesmo tempo: só um recebe.
	var wg sync.WaitGroup
	codes := make([]int, 2)
	jobs := make([]map[string]any, 2)
	for i, w := range []*worker{a, b} {
		wg.Add(1)
		go func(i int, w *worker) { defer wg.Done(); codes[i], jobs[i] = w.take("") }(i, w)
	}
	wg.Wait()
	if !(codes[0] == 200 && codes[1] == 204 || codes[0] == 204 && codes[1] == 200) {
		t.Fatalf("o mesmo trabalho para dois trabalhadores: %v", codes)
	}
	job, owner := jobs[0], a
	if codes[1] == 200 {
		job, owner = jobs[1], b
	}
	if job["arquivo"] != "foto.png" {
		t.Fatalf("trabalho: %v", job)
	}
	tok := job["token"].(string)

	// Log por deslocamento, com reenvio idempotente.
	if _, s := owner.post("/worker/log", map[string]any{"token": tok, "texto": "10%\n", "inicio": 0}); s["aceito"] != true || s["tamanho"] != float64(4) {
		t.Fatalf("log: %v", s)
	}
	if _, s := owner.post("/worker/log", map[string]any{"token": tok, "texto": "10%\n", "inicio": 0}); s["aceito"] != true || s["tamanho"] != float64(4) {
		t.Fatalf("reenvio do mesmo trecho: %v", s)
	}
	if _, s := owner.post("/worker/log", map[string]any{"token": tok, "texto": "x", "inicio": 99}); s["aceito"] != false || s["tamanho"] != float64(4) {
		t.Fatalf("deslocamento errado: %v", s)
	}
	if code, _ := owner.post("/worker/renovar", map[string]any{"token": "outro-token"}); code != 403 {
		t.Fatalf("token de outro trabalho: %d", code)
	}
	if _, s := owner.post("/worker/renovar", map[string]any{"token": tok}); s["estado"] != "executando" || s["cancelado"] != false {
		t.Fatalf("renovar: %v", s)
	}

	// Cancelamento pedido pela pessoa; o trabalhador percebe e encerra.
	c.expect("POST", "/_ge/api/conversoes/"+itoa(int(conv["id"].(float64)))+"/cancelar", nil, 200)
	if _, s := owner.post("/worker/renovar", map[string]any{"token": tok}); s["cancelado"] != true {
		t.Fatalf("o trabalhador deveria ver o cancelamento: %v", s)
	}
	if _, s := owner.post("/worker/log", map[string]any{"token": tok, "texto": "20%\n", "inicio": 4}); s["aceito"] != false {
		t.Fatalf("log depois de cancelado: %v", s)
	}
	if _, s := owner.post("/worker/concluir", map[string]any{"token": tok, "resultado": "sucesso"}); s["estado"] != "cancelado" {
		t.Fatalf("sucesso não desfaz o cancelamento: %v", s)
	}
	if _, s := owner.post("/worker/concluir", map[string]any{"token": tok, "resultado": "cancelado"}); s["estado"] != "cancelado" {
		t.Fatalf("confirmar cancelamento: %v", s)
	}

	// Concluir é idempotente.
	c.expect("POST", "/_ge/api/conversoes", map[string]any{"arquivo": "b.png"}, 201)
	_, j2 := owner.take("")
	t2 := j2["token"].(string)
	for i := 0; i < 2; i++ {
		if _, s := owner.post("/worker/concluir", map[string]any{"token": t2, "resultado": "sucesso"}); s["estado"] != "sucesso" {
			t.Fatalf("concluir #%d: %v", i+1, s)
		}
	}
	if _, s := owner.post("/worker/concluir", map[string]any{"token": t2, "resultado": "falhou"}); s["estado"] != "sucesso" {
		t.Fatalf("resultado não muda depois de concluído: %v", s)
	}
	if got := c.expect("GET", "/_ge/api/conversoes/"+itoa(int(j2["id"].(float64))), nil, 200); got["estado"] != "sucesso" {
		t.Fatalf("conversão concluída: %v", got)
	}

	// Pedir de novo com a mesma chave devolve o mesmo trabalho (e só o token novo vale).
	c.expect("POST", "/_ge/api/conversoes", map[string]any{"arquivo": "c.png"}, 201)
	_, k1 := a.take("chave-1")
	_, k2 := a.take("chave-1")
	if k1["id"] != k2["id"] || k1["token"] == k2["token"] {
		t.Fatalf("pedido repetido: %v %v", k1, k2)
	}
	if code, _ := a.post("/worker/renovar", map[string]any{"token": k1["token"]}); code != 403 {
		t.Fatalf("token antigo depois de repetir o pedido: %d", code)
	}
}

// Vários trabalhos e vários trabalhadores: cada trabalho é entregue uma vez.
func TestTrabalhoRemotoConcorrencia(t *testing.T) {
	_, c, ws := imagens(t)
	const n = 12
	for i := 0; i < n; i++ {
		c.expect("POST", "/_ge/api/conversoes", map[string]any{"arquivo": "f" + itoa(i) + ".png"}, 201)
	}
	var mu sync.Mutex
	seen := map[float64]int{}
	var wg sync.WaitGroup
	for _, w := range ws {
		wg.Add(1)
		go func(w *worker) {
			defer wg.Done()
			for {
				code, job := w.take("")
				if code == 204 {
					return
				}
				if code != 200 {
					t.Errorf("pegar: %d", code)
					return
				}
				mu.Lock()
				seen[job["id"].(float64)]++
				mu.Unlock()
				w.post("/worker/concluir", map[string]any{"token": job["token"], "resultado": "sucesso"})
			}
		}(w)
	}
	wg.Wait()
	if len(seen) != n {
		t.Fatalf("entregues %d de %d", len(seen), n)
	}
	for id, k := range seen {
		if k != 1 {
			t.Fatalf("trabalho %v entregue %d vezes", id, k)
		}
	}
}

// Reserva que expira: o trabalho volta para a fila; após 3 tentativas, falha.
func TestTrabalhoRemotoReservaExpira(t *testing.T) {
	t.Setenv("GERMANIO_RESERVA", "400ms")
	_, c, ws := imagens(t)
	conv := c.expect("POST", "/_ge/api/conversoes", map[string]any{"arquivo": "lenta.png"}, 201)
	path := "/_ge/api/conversoes/" + itoa(int(conv["id"].(float64)))
	for attempt := 1; attempt <= 3; attempt++ {
		code, job := ws[attempt%2].take("")
		if code != 200 {
			t.Fatalf("tentativa %d: %d", attempt, code)
		}
		time.Sleep(900 * time.Millisecond) // silêncio: a reserva expira
		if code, _ := ws[attempt%2].post("/worker/renovar", map[string]any{"token": job["token"]}); code != 403 {
			t.Fatalf("token de reserva expirada: %d", code)
		}
		got := c.expect("GET", path, nil, 200)
		want := "pendente"
		if attempt == 3 {
			want = "falhou"
		}
		if got["estado"] != want {
			t.Fatalf("depois da tentativa %d: %v", attempt, got["estado"])
		}
	}
	// Renovar mantém a reserva viva.
	c.expect("POST", "/_ge/api/conversoes", map[string]any{"arquivo": "viva.png"}, 201)
	_, job := ws[0].take("")
	for i := 0; i < 4; i++ {
		time.Sleep(200 * time.Millisecond)
		if _, s := ws[0].post("/worker/renovar", map[string]any{"token": job["token"]}); s["estado"] != "executando" {
			t.Fatalf("renovação %d: %v", i, s)
		}
	}
}
