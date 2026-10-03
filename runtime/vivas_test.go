package runtime

import (
	"bufio"
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Pages stay up to date (GEP 0020, em teste).

// watch opens the live subscription of a page and reports each "mudou".
func watch(t *testing.T, c *client, page string) (<-chan struct{}, int) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	req, _ := http.NewRequestWithContext(ctx, "GET", c.base+"/_ge/atualizacoes?p="+page, nil)
	resp, err := c.http.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		resp.Body.Close()
		return nil, resp.StatusCode
	}
	out := make(chan struct{}, 100)
	ready := make(chan struct{})
	go func() {
		defer resp.Body.Close()
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			line := sc.Text()
			if strings.HasPrefix(line, ": ligado") {
				close(ready)
			}
			if line == "event: mudou" {
				out <- struct{}{}
			}
		}
	}()
	select {
	case <-ready:
	case <-time.After(3 * time.Second):
		t.Fatal("a assinatura não abriu")
	}
	return out, 200
}

func expectChange(t *testing.T, ch <-chan struct{}, want bool, what string) {
	t.Helper()
	wait := 400 * time.Millisecond
	if want {
		wait = 3 * time.Second
	}
	select {
	case <-ch:
		if !want {
			t.Fatalf("%s: a página foi avisada e não devia", what)
		}
	case <-time.After(wait):
		if want {
			t.Fatalf("%s: a página não foi avisada", what)
		}
	}
	for len(ch) > 0 { // drain coalesced extras
		<-ch
	}
}

func TestPaginasVivas(t *testing.T) {
	_, c := loadApp(t, "testdata/vivas/app.ge")
	ana := signIn(t, c.base, "Ana", "ana@x.com")
	bia := signIn(t, c.base, "Bia", "bia@x.com")
	ana.expect("POST", "/_ge/api/projetos", map[string]any{"nome": "Loja"}, 201)

	// the page carries its live regions and the script
	_, _, html := bia.do("GET", "/projetos/1", nil)
	if !strings.Contains(html, `data-vivo="filhos-issues"`) || !strings.Contains(html, `/_ge/atualizar.js`) {
		t.Fatalf("a página não tem regiões vivas:\n%s", html)
	}
	if _, _, js := bia.do("GET", "/_ge/atualizar.js", nil); !strings.Contains(js, "EventSource") {
		t.Fatalf("script: %s", js)
	}

	page, _ := watch(t, bia, "/projetos/1")
	ana.expect("POST", "/_ge/api/projetos/1/issues", map[string]any{"titulo": "Pública"}, 201)
	expectChange(t, page, true, "issue nova no projeto que Bia vê")

	ana.expect("POST", "/_ge/api/projetos/1/issues", map[string]any{"titulo": "Segredo", "confidencial": true}, 201)
	expectChange(t, page, false, "issue confidencial que Bia não vê")

	ana.expect("POST", "/_ge/api/projetos/1/issues", map[string]any{"titulo": "recusada"}, 400)
	expectChange(t, page, false, "mudança desfeita")

	// a write made by the core (a pending item) reaches the page that shows it
	pend, _ := watch(t, bia, "/pendencias")
	bid := bia.expect("GET", "/_ge/eu", nil, 200)["id"]
	ana.expect("POST", "/_ge/api/projetos/1/issues", map[string]any{"titulo": "Para Bia", "responsaveis": []any{bid}}, 201)
	expectChange(t, pend, true, "pendência criada pelo core")

	// many changes in a row: the subscription keeps up and coalesces
	for i := 0; i < 20; i++ {
		ana.expect("POST", "/_ge/api/projetos/1/issues", map[string]any{"titulo": "lote"}, 201)
	}
	expectChange(t, page, true, "lote de mudanças")

	// subscribing needs access to the page
	_, anon := c.fresh(t)
	if _, code := watch(t, anon, "/projetos/1"); code != 404 {
		t.Fatalf("assinatura de quem não pode ver a página: %d", code)
	}
	if _, code := watch(t, bia, "/inexistente"); code != 404 {
		t.Fatalf("assinatura de página inexistente: %d", code)
	}
}
