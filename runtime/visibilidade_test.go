package runtime

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"reflect"
	"testing"

	"github.com/flaviokalleu/germanio/runtime/servidor"
)

// Lists narrow what they read by what a person may reach (G85). The answer
// must be exactly the one of the full scan, for every person: random
// projects (private, internal, public), roles, authors, assignees and
// confidential issues, compared with the narrowing on and off.
func TestPreFiltroDeVisibilidadeNaoMudaResposta(t *testing.T) {
	for seed := int64(1); seed <= 6; seed++ {
		t.Run(fmt.Sprint("semente ", seed), func(t *testing.T) { preFiltroComparado(t, seed) })
	}
}

func preFiltroComparado(t *testing.T, seed int64) {
	app, c := loadApp(t, "testdata/visibilidade/app.ge")
	rng := rand.New(rand.NewSource(seed))

	newClient := func() *client {
		jar, _ := cookiejar.New(nil)
		return &client{t: t, base: c.base, http: &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, Header: http.Header{}}
	}
	var people []*client
	var ids []int
	for i := 0; i < 6; i++ {
		p := newClient()
		out := p.expect("POST", "/cadastro", map[string]any{"nome": fmt.Sprint("P", i), "email": fmt.Sprintf("p%d@x.com", i), "senha": "senha-forte-123"}, 201)
		people = append(people, p)
		ids = append(ids, int(out["id"].(float64)))
	}
	db := app.DB.DB
	vis := []string{"private", "internal", "public"}
	roles := []string{"guest", "reporter", "developer"}
	for pr := 1; pr <= 8; pr++ {
		if _, err := db.Exec(`INSERT INTO projeto (id, nome, visibilidade) VALUES (?, ?, ?)`, pr, fmt.Sprint("projeto", pr), vis[rng.Intn(3)]); err != nil {
			t.Fatal(err)
		}
		for _, uid := range ids {
			if rng.Intn(3) == 0 {
				db.Exec(`INSERT INTO membro (recurso, recurso_id, pessoa_id, papel) VALUES ('projeto', ?, ?, ?)`, pr, uid, roles[rng.Intn(3)])
			}
		}
	}
	for i := 0; i < 200; i++ {
		var autor any
		if rng.Intn(2) == 0 {
			autor = ids[rng.Intn(len(ids))]
		}
		resp := "[]"
		if rng.Intn(3) == 0 {
			resp = fmt.Sprintf("[%d]", ids[rng.Intn(len(ids))])
		}
		if _, err := db.Exec(`INSERT INTO issue (titulo, projeto_id, autor_id, responsaveis, confidencial) VALUES (?, ?, ?, ?, ?)`,
			fmt.Sprint("issue", i), 1+rng.Intn(8), autor, resp, rng.Intn(4) == 0); err != nil {
			t.Fatal(err)
		}
	}

	viewers := append([]*client{newClient()}, people...) // anonymous first
	list := func(v *client, path string) (int, string) {
		resp, err := v.http.Get(v.base + path)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, resp.Header.Get("X-Total") + " " + string(raw)
	}
	distinct := map[string]bool{}
	defer func() { servidor.PreFiltroVisibilidade = true }()
	defer func() {
		// the comparison only proves something if people see different things
		if len(distinct) < 4 {
			t.Fatalf("só %d respostas diferentes entre as pessoas: o teste não exercita a visibilidade", len(distinct))
		}
	}()
	for vi, v := range viewers {
		for _, path := range []string{"/_ge/api/issues?per_page=100", "/_ge/api/issues?per_page=100&page=2", "/_ge/api/projetos?per_page=100"} {
			servidor.PreFiltroVisibilidade = true
			s1, on := list(v, path)
			servidor.PreFiltroVisibilidade = false
			s2, off := list(v, path)
			if s1 != s2 || !reflect.DeepEqual(on, off) {
				t.Fatalf("pessoa %d, %s: com o pré-filtro (%d) %s\nsem (%d) %s", vi, path, s1, on, s2, off)
			}
			if s1 == 200 && path == "/_ge/api/issues?per_page=100" {
				distinct[on] = true
			}
		}
	}
}
