package e2e

import (
	"io"
	"net/http"
	"regexp"
	"testing"
)

// AD-01: o painel conta com as regras de quem vê (GEP 0012). O administrador
// vê o total; quem não é membro não conta o que é privado.
func TestPainel(t *testing.T) {
	base := gitlab(t)
	anon := &api{t: t, base: base}
	tok := anon.must("POST", "/oauth/token", map[string]any{"grant_type": "password", "username": "root", "password": "rootpassword1"}, 200)
	root := &api{t: t, base: base, token: tok["access_token"].(string)}
	ada := signup(t, base, "ada")
	eve := signup(t, base, "eve")
	pub := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Aberto", "path": "aberto", "visibility": "public"}, 201)
	ada.must("POST", "/api/v4/projects/"+id(pub)+"/issues", map[string]any{"title": "pública"}, 201)
	priv := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Fechado", "path": "fechado"}, 201)
	ada.must("POST", "/api/v4/projects/"+id(priv)+"/issues", map[string]any{"title": "privada"}, 201)
	ada.must("POST", "/api/v4/projects/"+id(priv)+"/issues", map[string]any{"title": "privada 2"}, 201)

	painel := func(who *api) map[string]string {
		t.Helper()
		req, _ := http.NewRequest("GET", base+"/painel", nil)
		req.Header.Set("Authorization", "Bearer "+who.token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		out := map[string]string{}
		for _, m := range regexp.MustCompile(`<span class="valor">(\d+)</span><span class="rotulo">([^<]+)</span>`).FindAllStringSubmatch(string(b), -1) {
			out[m[2]] = m[1]
		}
		if len(out) == 0 {
			t.Fatalf("painel sem indicadores (%d):\n%s", resp.StatusCode, b)
		}
		return out
	}
	if got := painel(root); got["Total de issues abertas"] != "3" || got["Total de projetos"] != "2" {
		t.Fatalf("o administrador conta tudo: %v", got)
	}
	if got := painel(ada); got["Total de issues abertas"] != "3" {
		t.Fatalf("a dona conta as suas: %v", got)
	}
	if got := painel(eve); got["Total de issues abertas"] != "1" || got["Total de projetos"] != "1" {
		t.Fatalf("quem não é membro não conta o privado: %v", got)
	}
}
