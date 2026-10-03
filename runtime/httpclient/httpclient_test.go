package httpclient

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Outgoing requests of an application (chamar, cron) never reach the
// server's own network, whatever spelling the address takes; development
// can allow it explicitly.
func TestChamarNaoAlcancaRedeLocal(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	port := srv.URL[strings.LastIndex(srv.URL, ":")+1:]
	for _, u := range []string{srv.URL, "http://127.1:" + port, "http://2130706433:" + port, "http://[::ffff:127.0.0.1]:" + port, "http://localhost.:" + port} {
		if _, err := Novo().Chamar("GET", u, nil); !errors.Is(err, ErrRedeLocal) {
			t.Errorf("%s: esperado ErrRedeLocal, veio %v", u, err)
		}
	}
	if hits != 0 {
		t.Fatalf("a rede local foi alcançada %d vez(es)", hits)
	}
	t.Setenv("GERMANIO_PERMITIR_REDE_LOCAL", "1")
	if _, err := Novo().Chamar("GET", srv.URL, nil); err != nil || hits != 1 {
		t.Fatalf("com a permissão explícita: %v (hits %d)", err, hits)
	}
}

func TestNomesLocaisSemResolver(t *testing.T) {
	for host, want := range map[string]bool{
		"localhost": true, "LOCALHOST.": true, "api.localhost": true, "127.1": true,
		"2130706433": true, "0x7f.1": true, "0177.0.0.1": true, "10.1": true, "0": true,
		"8.8.8.8": false, "134744072": false, "exemplo.com": false, "1.2.3.4.5": false,
		"256.1.1.1": false, "1.2.65536": false, "localhost.exemplo.com": false,
	} {
		if got := nomeLocal(host); got != want {
			t.Errorf("%s: nomeLocal=%v, esperado %v", host, got, want)
		}
	}
}

func TestEnderecosLocais(t *testing.T) {
	for addr, want := range map[string]bool{
		"127.0.0.1": true, "10.1.2.3": true, "169.254.169.254": true, "100.64.0.1": true,
		"0.1.2.3": true, "::1": true, "fe80::1": true, "fd00::1": true,
		"8.8.8.8": false, "2606:4700:4700::1111": false,
	} {
		if got := local(net.ParseIP(addr)); got != want {
			t.Errorf("%s: local=%v, esperado %v", addr, got, want)
		}
	}
}
