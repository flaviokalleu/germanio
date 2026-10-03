package runtime

import (
	"context"
	"net/http"
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
