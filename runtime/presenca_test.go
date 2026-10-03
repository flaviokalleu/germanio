package runtime

import (
	"bufio"
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// Presence (GEP 0021, em teste).
func TestPresenca(t *testing.T) {
	t.Setenv("GERMANIO_PRESENCA_TOLERANCIA", "300ms")
	_, c := loadApp(t, "testdata/presenca/app.ge")
	ana := signIn(t, c.base, "Ana", "ana@x.com")
	bia := signIn(t, c.base, "Bia", "bia@x.com")
	bid := jsonID(bia.expect("GET", "/_ge/eu", nil, 200)["id"])
	online := func() any { return ana.expect("GET", "/_ge/api/usuarios/"+bid, nil, 200)["online"] }

	if online() != false {
		t.Fatalf("Bia sem página aberta não está online: %v", online())
	}
	people, _ := watch(t, ana, "/pessoas")
	expectChange(t, people, true, "Ana entrou (a própria página)")

	// Bia opens a page: online, and Ana's people page is told
	ctx, closeBia := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, "GET", c.base+"/_ge/atualizacoes?p=/pessoas", nil)
	resp, err := bia.http.Do(req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("página de Bia: %v %v", err, resp)
	}
	expectChange(t, people, true, "Bia ficou online")
	if online() != true {
		t.Fatalf("Bia com página aberta está online: %v", online())
	}
	// Bia closes it: still online during the grace period, then offline
	closeBia()
	resp.Body.Close()
	time.Sleep(100 * time.Millisecond)
	if online() != true {
		t.Fatal("dentro da tolerância Bia continua online")
	}
	expectChange(t, people, true, "Bia ficou offline")
	if online() != false {
		t.Fatalf("depois da tolerância Bia está offline: %v", online())
	}
	// back within the grace period, then gone for good: offline in the end
	open := func() func() {
		ctx, cancel := context.WithCancel(context.Background())
		req, _ := http.NewRequestWithContext(ctx, "GET", c.base+"/_ge/atualizacoes?p=/pessoas", nil)
		r, err := bia.http.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(50 * time.Millisecond)
		return func() { cancel(); r.Body.Close() }
	}
	open()()
	time.Sleep(100 * time.Millisecond)
	open()()
	time.Sleep(700 * time.Millisecond)
	if online() != false {
		t.Fatal("voltar dentro da tolerância deixou Bia online para sempre")
	}
}

// Without tenha presença, people have no online field.
func TestSemPresenca(t *testing.T) {
	_, c := loadApp(t, "testdata/vivas/app.ge")
	ana := signIn(t, c.base, "Ana", "ana@x.com")
	if me := ana.expect("GET", "/_ge/eu", nil, 200); me["online"] != nil {
		t.Fatalf("sem presença declarada não há online: %v", me)
	}
}

// "Digitando…" (presence): the viewers of the same page are told who is
// typing; the person typing is not; another page is not; it needs a session
// and the CSRF token.
func TestDigitando(t *testing.T) {
	_, c := loadApp(t, "testdata/presenca/app.ge")
	ana := signIn(t, c.base, "Ana", "ana@x.com")
	bia := signIn(t, c.base, "Bia", "bia@x.com")
	stream := func(who *client, page string) <-chan string {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)
		req, _ := http.NewRequestWithContext(ctx, "GET", c.base+"/_ge/atualizacoes?p="+page, nil)
		resp, err := who.http.Do(req)
		if err != nil || resp.StatusCode != 200 {
			t.Fatalf("assinatura: %v %v", err, resp)
		}
		out := make(chan string, 10)
		go func() {
			defer resp.Body.Close()
			sc := bufio.NewScanner(resp.Body)
			typing := false
			for sc.Scan() {
				line := sc.Text()
				if line == "event: digitando" {
					typing = true
					continue
				}
				if typing && strings.HasPrefix(line, "data: ") {
					out <- strings.TrimPrefix(line, "data: ")
					typing = false
				}
			}
		}()
		time.Sleep(100 * time.Millisecond)
		return out
	}
	biaSees, anaSees, other := stream(bia, "/pessoas"), stream(ana, "/pessoas"), stream(bia, "/inicio")
	post := func(who *client, csrf string) int {
		resp, err := who.http.PostForm(c.base+"/_ge/digitando", url.Values{"p": {"/pessoas"}, "_csrf": {csrf}})
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	if code := post(ana, ana.csrf); code != 204 {
		t.Fatalf("avisar que digita: %d", code)
	}
	select {
	case ev := <-biaSees:
		if !strings.Contains(ev, "Ana") {
			t.Fatalf("aviso: %s", ev)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Bia não soube que Ana digita")
	}
	select {
	case ev := <-anaSees:
		t.Fatalf("Ana recebeu o próprio aviso: %s", ev)
	case ev := <-other:
		t.Fatalf("outra página recebeu o aviso: %s", ev)
	case <-time.After(300 * time.Millisecond):
	}
	if code := post(ana, "errado"); code != 403 {
		t.Fatalf("sem o token CSRF: %d", code)
	}
	_, anon := c.fresh(t)
	if code := post(anon, ""); code != 401 {
		t.Fatalf("sem sessão: %d", code)
	}
}
