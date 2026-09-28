package e2e

import (
	"net/url"
	"testing"
)

// Bloquear/desbloquear: um usuário bloqueado não entra, e nenhuma credencial
// já emitida (sessão OAuth, token de acesso, git) continua valendo.
func TestBloquearUsuario(t *testing.T) {
	base := gitlab(t)
	anon := &api{t: t, base: base}
	tok := anon.must("POST", "/oauth/token", map[string]any{"grant_type": "password", "username": "root", "password": "rootpassword1"}, 200)
	root := &api{t: t, base: base, token: tok["access_token"].(string)}
	ada := signup(t, base, "ada")
	bob := signup(t, base, "bob")
	me := ada.must("GET", "/api/v4/user", nil, 200)
	if me["state"] != "active" {
		t.Fatalf("estado inicial: %v", me["state"])
	}
	ada.must("POST", "/api/v4/projects", map[string]any{"name": "Lib", "path": "lib", "initialize_with_readme": true}, 201)
	pat := ada.must("POST", "/api/v4/personal_access_tokens", map[string]any{"name": "ci"}, 201)
	adaPAT := &api{t: t, base: base, token: pat["token"].(string), pat: true}
	adaID := id(me)

	// Só administrador bloqueia
	bob.must("POST", "/api/v4/users/"+adaID+"/block", nil, 403)
	ada.must("POST", "/api/v4/users/"+id(bob.must("GET", "/api/v4/user", nil, 200))+"/block", nil, 403)
	u := root.must("POST", "/api/v4/users/"+adaID+"/block", nil, 200)
	if u["state"] != "blocked" {
		t.Fatalf("depois de bloquear: %v", u)
	}
	root.must("POST", "/api/v4/users/"+adaID+"/block", nil, 400) // já bloqueado

	// Nenhuma credencial vale enquanto bloqueado
	anon.must("POST", "/oauth/token", map[string]any{"grant_type": "password", "username": "ada", "password": "password123"}, 400)
	ada.must("GET", "/api/v4/user", nil, 401)
	adaPAT.must("GET", "/api/v4/user", nil, 401)
	gu, _ := url.Parse(base)
	gu.User = url.UserPassword("ada", pat["token"].(string))
	runFails(t, t.TempDir(), "git", "clone", "--quiet", gu.String()+"/ada/lib.git", "x")

	// Desbloquear devolve o acesso
	u = root.must("POST", "/api/v4/users/"+adaID+"/unblock", nil, 200)
	if u["state"] != "active" {
		t.Fatalf("depois de desbloquear: %v", u)
	}
	adaPAT.must("GET", "/api/v4/user", nil, 200)
	anon.must("POST", "/oauth/token", map[string]any{"grant_type": "password", "username": "ada", "password": "password123"}, 200)
	run(t, t.TempDir(), "git", "clone", "--quiet", gu.String()+"/ada/lib.git", "x")
}
