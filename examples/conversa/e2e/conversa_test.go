package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	germanio "github.com/flaviokalleu/germanio/runtime"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// A Conversa como está hoje (FASE 2, passo 1): o domínio funciona com o que
// a linguagem já tem — espaços, canais, mensagens numeradas por canal,
// menções e permissões. O tempo real ainda não existe: é o que a fase prova.

type pessoa struct {
	t    *testing.T
	base string
	http *http.Client
	csrf string
}

func (p *pessoa) do(method, path string, body any, want int) map[string]any {
	p.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, p.base+path, rd)
	req.Header.Set("Content-Type", "application/json")
	if p.csrf != "" {
		req.Header.Set("X-CSRF-Token", p.csrf)
	}
	resp, err := p.http.Do(req)
	if err != nil {
		p.t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != want {
		p.t.Fatalf("%s %s: %d, esperado %d: %s", method, path, resp.StatusCode, want, raw)
	}
	var out map[string]any
	json.Unmarshal(raw, &out)
	return out
}

func entra(t *testing.T, base, nome string) *pessoa {
	jar, _ := cookiejar.New(nil)
	p := &pessoa{t: t, base: base, http: &http.Client{Jar: jar}}
	p.do("POST", "/cadastro", map[string]any{"nome": nome, "username": nome, "email": nome + "@x.com", "senha": "senha-segura-1"}, 201)
	u, _ := url.Parse(base)
	for _, c := range jar.Cookies(u) {
		if c.Name == interp.SessionCookie {
			if sess := interp.VerificarToken(c.Value); sess != nil {
				p.csrf, _ = sess["csrf"].(string)
			}
		}
	}
	return p
}

func TestConversaHoje(t *testing.T) {
	t.Setenv("GERMANIO_SQLITE", filepath.Join(t.TempDir(), "c.db"))
	t.Setenv("GERMANIO_ARQUIVOS", t.TempDir())
	t.Setenv("GERMANIO_BCRYPT_RAPIDO", "1")
	app, err := germanio.Carregar("../app.ge", "0")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(app.Handler)
	defer func() { srv.Close(); app.Fechar() }()
	ana, bia, eve := entra(t, srv.URL, "ana"), entra(t, srv.URL, "bia"), entra(t, srv.URL, "eve")
	esp := ana.do("POST", "/_ge/api/espacos", map[string]any{"nome": "Produto", "visibilidade": "private"}, 201)
	e := jsonID(esp["id"])
	bid := bia.do("GET", "/_ge/eu", nil, 200)["id"]
	ana.do("POST", "/_ge/api/espacos/"+e+"/membros", map[string]any{"pessoa_id": bid, "papel": "membro"}, 201)
	ana.do("POST", "/_ge/api/espacos/"+e+"/canais", map[string]any{"nome": "geral"}, 201)
	m1 := ana.do("POST", "/_ge/api/espacos/"+e+"/canais/1/mensagens", map[string]any{"texto": "oi @bia"}, 201)
	m2 := bia.do("POST", "/_ge/api/espacos/"+e+"/canais/1/mensagens", map[string]any{"texto": "oi!"}, 201)
	if m1["numero"] != float64(1) || m2["numero"] != float64(2) {
		t.Fatalf("ordem por canal: %v %v", m1["numero"], m2["numero"])
	}
	eve.do("GET", "/_ge/api/espacos/"+e+"/canais/1/mensagens", nil, 404)
	eve.do("POST", "/_ge/api/espacos/"+e+"/canais/1/mensagens", map[string]any{"texto": "intrusa"}, 404)
	bia.do("PUT", "/_ge/api/espacos/"+e+"/canais/1/mensagens/1", map[string]any{"texto": "alterada"}, 403)
}

func jsonID(v any) string { b, _ := json.Marshal(v); return string(b) }

// Critério 3 da FASE 2: muitas pessoas escrevendo ao mesmo tempo no mesmo
// canal produzem uma sequência sem buracos nem repetições.
func TestOrdemSobConcorrencia(t *testing.T) {
	t.Setenv("GERMANIO_SQLITE", filepath.Join(t.TempDir(), "c.db"))
	t.Setenv("GERMANIO_BCRYPT_RAPIDO", "1")
	app, err := germanio.Carregar("../app.ge", "0")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(app.Handler)
	defer func() { srv.Close(); app.Fechar() }()
	dono := entra(t, srv.URL, "dono")
	e := jsonID(dono.do("POST", "/_ge/api/espacos", map[string]any{"nome": "Equipe"}, 201)["id"])
	dono.do("POST", "/_ge/api/espacos/"+e+"/canais", map[string]any{"nome": "geral"}, 201)
	var pessoas []*pessoa
	for i := 0; i < 8; i++ {
		p := entra(t, srv.URL, "p"+string(rune('a'+i)))
		pid := p.do("GET", "/_ge/eu", nil, 200)["id"]
		dono.do("POST", "/_ge/api/espacos/"+e+"/membros", map[string]any{"pessoa_id": pid, "papel": "membro"}, 201)
		pessoas = append(pessoas, p)
	}
	const cada = 10
	done := make(chan []float64)
	for _, p := range pessoas {
		go func(p *pessoa) {
			var nums []float64
			for i := 0; i < cada; i++ {
				m := p.do("POST", "/_ge/api/espacos/"+e+"/canais/1/mensagens", map[string]any{"texto": "oi"}, 201)
				nums = append(nums, m["numero"].(float64))
			}
			done <- nums
		}(p)
	}
	seen := map[float64]bool{}
	for range pessoas {
		for _, n := range <-done {
			if seen[n] {
				t.Fatalf("número repetido no canal: %v", n)
			}
			seen[n] = true
		}
	}
	for i := 1; i <= len(pessoas)*cada; i++ {
		if !seen[float64(i)] {
			t.Fatalf("buraco na sequência do canal: falta %d", i)
		}
	}
}
