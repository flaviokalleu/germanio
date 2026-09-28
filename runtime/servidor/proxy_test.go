package servidor

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/flaviokalleu/germanio/compiler/ast"
)

// The proxy must never reach the server's own network, whatever spelling
// the address takes, and must not be usable by other sites from a browser.
func TestProxyNaoAlcancaRedeLocal(t *testing.T) {
	var hits atomic.Int32
	interno := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Write([]byte(`{"segredo":"metadados"}`))
	}))
	defer interno.Close()
	port := interno.URL[strings.LastIndex(interno.URL, ":")+1:]

	s := Novo(&ast.Program{System: &ast.System{Name: "teste"}}, nil, "0")
	for _, target := range []string{
		"http://127.0.0.1:" + port + "/",
		"http://127.1:" + port + "/",
		"http://2130706433:" + port + "/",
		"http://[::ffff:127.0.0.1]:" + port + "/",
		"http://[::1]:" + port + "/",
		"http://localhost.:" + port + "/",
	} {
		w := httptest.NewRecorder()
		s.handleProxy(w, httptest.NewRequest("POST", "/api/_proxy", strings.NewReader(`{"url":"`+target+`"}`)))
		if w.Code == http.StatusOK {
			t.Errorf("%s: o proxy respondeu 200: %s", target, w.Body.String())
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got == "*" {
			t.Errorf("%s: proxy aberto a qualquer site (CORS *)", target)
		}
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("o proxy alcançou a rede local %d vez(es)", n)
	}
}
