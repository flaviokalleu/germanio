package runtime

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"testing"

	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

func TestIntentSimpleCRUD(t *testing.T) {
	_, c := loadApp(t, "testdata/intencao/clientes.ge")
	bad := c.expect("POST", "/_ge/api/clientes", map[string]any{"nome": "Ana", "email": "nao-e-email"}, 400)
	if m := bad["message"].(map[string]any); m["email"].([]any)[0] != "é inválido" {
		t.Fatalf("validação em português esperada: %v", bad)
	}
	c.expect("POST", "/_ge/api/clientes", map[string]any{"email": "a@b.co"}, 400)
	ana := c.expect("POST", "/_ge/api/clientes", map[string]any{"nome": "Ana", "email": "ANA@x.com ", "cidade": "Recife"}, 201)
	if ana["email"] != "ana@x.com" || ana["created_at"] == nil {
		t.Fatalf("normalização/serialização: %v", ana)
	}
	dup := c.expect("POST", "/_ge/api/clientes", map[string]any{"nome": "Outra", "email": "ana@x.com"}, 400)
	if m := dup["message"].(map[string]any); m["email"].([]any)[0] != "já está em uso" {
		t.Fatalf("unicidade: %v", dup)
	}
	c.expect("POST", "/_ge/api/clientes", map[string]any{"nome": "Bruno", "email": "b@x.com", "cidade": "Natal"}, 201)
	_, _, raw := c.do("GET", "/_ge/api/clientes?q=bru", nil)
	if !strings.Contains(raw, "Bruno") || strings.Contains(raw, "Ana") {
		t.Fatalf("pesquisa: %s", raw)
	}
	_, _, raw = c.do("GET", "/_ge/api/clientes?cidade=Recife", nil)
	if !strings.Contains(raw, "Ana") || strings.Contains(raw, "Bruno") {
		t.Fatalf("filtro: %s", raw)
	}
	id := int(ana["id"].(float64))
	upd := c.expect("PATCH", "/_ge/api/clientes/"+itoa(id), map[string]any{"telefone": "81999999999"}, 200)
	if upd["telefone"] != "81999999999" || upd["nome"] != "Ana" {
		t.Fatalf("edição parcial: %v", upd)
	}
	c.expect("DELETE", "/_ge/api/clientes/"+itoa(id), nil, 204)
	c.expect("GET", "/_ge/api/clientes/"+itoa(id), nil, 404)
}

func TestIntentLoginOwnershipRoles(t *testing.T) {
	_, c := loadApp(t, "testdata/intencao/loja.ge")
	c.expect("GET", "/_ge/api/pedidos", nil, 401)
	c.expect("POST", "/cadastro", map[string]any{"nome": "Bia", "email": "bia@x.com", "senha": "curta"}, 400)
	bia := c.expect("POST", "/cadastro", map[string]any{"nome": "Bia", "email": "bia@x.com", "senha": "segredo123", "admin": true}, 201)
	if bia["admin"] != false || bia["senha"] != nil {
		t.Fatalf("cadastro não pode virar admin nem expor senha: %v", bia)
	}
	me := c.expect("GET", "/_ge/eu", nil, 200)
	if me["email"] != "bia@x.com" {
		t.Fatalf("sessão: %v", me)
	}
	// Session cookie requires CSRF for writes made by the browser; this test
	// client authenticates like a script, so log out and use a fresh client
	// with the session token header-free flow of the pages later.
	c.expect("POST", "/sair", nil, 204)
	c.expect("POST", "/entrar", map[string]any{"login": "bia@x.com", "senha": "errada"}, 401)
	c.expect("POST", "/entrar", map[string]any{"login": "bia@x.com", "senha": "segredo123"}, 200)
	c.csrf = csrfFromCookie(t, c)
	p1 := c.expect("POST", "/_ge/api/pedidos", map[string]any{"descricao": "livros", "total": 50, "usuario_id": 999}, 201)
	if p1["usuario_id"].(float64) != bia["id"].(float64) {
		t.Fatalf("pedido deve pertencer a quem criou: %v", p1)
	}
	caro := c.expect("POST", "/_ge/api/pedidos", map[string]any{"descricao": "carro", "total": 5000}, 400)
	if caro["message"] != "Pedidos acima de 1000 precisam de aprovação" {
		t.Fatalf("recuse no quando criar: %v", caro)
	}
	_, _, raw := c.do("GET", "/_ge/api/pedidos", nil)
	if strings.Contains(raw, "carro") || !strings.Contains(raw, "livros") {
		t.Fatalf("pedido recusado não pode ficar pela metade: %s", raw)
	}
	c.expect("DELETE", "/_ge/api/pedidos/"+itoa(int(p1["id"].(float64))), nil, 403)

	// Another customer cannot see Bia's orders.
	_, c2 := c.fresh(t)
	c2.expect("POST", "/cadastro", map[string]any{"nome": "Caio", "email": "caio@x.com", "senha": "segredo123"}, 201)
	c2.csrf = csrfFromCookie(t, c2)
	c2.expect("GET", "/_ge/api/pedidos/"+itoa(int(p1["id"].(float64))), nil, 404)
	_, _, raw = c2.do("GET", "/_ge/api/pedidos", nil)
	if strings.Contains(raw, "livros") {
		t.Fatalf("vazou pedido de outra pessoa: %s", raw)
	}
	c2.expect("PATCH", "/_ge/api/usuarios/"+itoa(int(bia["id"].(float64))), map[string]any{"nome": "hack"}, 404)

	// The administrator seeded by `ao iniciar` can delete.
	_, adm := c.fresh(t)
	adm.expect("POST", "/entrar", map[string]any{"login": "admin@loja.test", "senha": "admin12345"}, 200)
	adm.csrf = csrfFromCookie(t, adm)
	adm.expect("DELETE", "/_ge/api/pedidos/"+itoa(int(p1["id"].(float64))), nil, 204)
}

func itoa(n int) string { return strconv.Itoa(n) }

// fresh returns a client with an empty cookie jar against the same server.
func (c *client) fresh(t *testing.T) (*App, *client) {
	jar, _ := cookiejar.New(nil)
	return nil, &client{t: t, base: c.base, http: &http.Client{Jar: jar, CheckRedirect: c.http.CheckRedirect}, Header: http.Header{}}
}

// csrfFromCookie reads the CSRF value a page would embed from the session.
func csrfFromCookie(t *testing.T, c *client) string {
	u, _ := url.Parse(c.base)
	for _, ck := range c.http.Jar.Cookies(u) {
		if ck.Name == interp.SessionCookie {
			if claims := interp.VerificarToken(ck.Value); claims != nil {
				return claims["csrf"].(string)
			}
		}
	}
	t.Fatal("sem cookie de sessão")
	return ""
}
