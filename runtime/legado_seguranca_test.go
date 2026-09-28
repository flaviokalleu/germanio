package runtime

import "testing"

// The older dialect (autenticacao + dados) must not leave doors open:
// signing up never grants a role, another person's account cannot be changed
// or deleted, nothing is readable without logging in, and a client cannot
// claim a role with its own headers.
func TestDialetoAnteriorNaoAbrePortas(t *testing.T) {
	t.Setenv("JWT_SECRET", "segredo-de-teste-com-tamanho-suficiente-123")
	_, c := loadApp(t, "testdata/tempo_real/app.ge")

	ana := c.expect("POST", "/api/registro", map[string]any{"nome": "Ana", "email": "ana@x.com", "senha": "senha-forte-123", "role": "admin"}, 201)
	c.expect("POST", "/api/registro", map[string]any{"nome": "Bia", "email": "bia@x.com", "senha": "senha-forte-123"}, 201)
	c.expect("POST", "/api/registro", map[string]any{"nome": "Caio", "email": "caio@x.com", "senha": "senha-forte-123", `nome" ,"role`: "admin"}, 201)
	login := c.expect("POST", "/api/login", map[string]any{"email": "ana@x.com", "senha": "senha-forte-123"}, 200)
	token, _ := login["token"].(string)
	if roleOf(t, c, token) == "admin" {
		t.Fatalf("o cadastro concedeu o papel admin: %v", login)
	}

	// anonymous reads
	c.Header.Del("Authorization")
	for _, path := range []string{"/api/usuario", "/api/nota", "/api/usuario/export/csv", "/api/usuario/1"} {
		if status, _, raw := c.do("GET", path, nil); status == 200 {
			t.Errorf("GET %s sem login respondeu 200: %.120s", path, raw)
		}
	}
	// a client cannot claim a role with its own headers
	c.Header.Set("X-User-Role", "admin")
	if status, _, _ := c.do("GET", "/api/_log", nil); status == 200 {
		t.Error("X-User-Role enviado pelo cliente foi aceito")
	}
	c.Header.Del("X-User-Role")

	// another person's account
	c.Header.Set("Authorization", "Bearer "+token)
	bia := "2"
	if status, _, _ := c.do("PUT", "/api/usuario/"+bia, map[string]any{"nome": "hack", "senha": "trocada-123456"}); status == 200 {
		t.Error("uma pessoa alterou a conta de outra")
	}
	if status, _, _ := c.do("DELETE", "/api/usuario/"+bia, nil); status < 400 {
		t.Error("uma pessoa apagou a conta de outra")
	}
	own := itoa(int(ana["id"].(float64)))
	if status, out, _ := c.do("PUT", "/api/usuario/"+own, map[string]any{"nome": "Ana Maria", "email": "ana@x.com", "senha": "senha-forte-123", "role": "admin"}); status != 200 || out["role"] == "admin" {
		t.Errorf("editar o próprio perfil: status %d, papel %v (o papel não pode subir): %v", status, out["role"], out)
	}
}

// roleOf asks /api/me which role the token carries.
func roleOf(t *testing.T, c *client, token string) string {
	c.Header.Set("Authorization", "Bearer "+token)
	_, me, _ := c.do("GET", "/api/me", nil)
	c.Header.Del("Authorization")
	role, _ := me["role"].(string)
	return role
}
