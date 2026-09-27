package servidor

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSafeHTTPClientBlocksLocalNetwork(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	t.Setenv("GERMANIO_PERMITIR_REDE_LOCAL", "")
	_, err := safeHTTPClient().Get(srv.URL)
	if err == nil || !errors.Is(err, errLocalNetwork) {
		t.Fatalf("esperado bloqueio de rede local, obtido %v", err)
	}
	t.Setenv("GERMANIO_PERMITIR_REDE_LOCAL", "1")
	resp, err := safeHTTPClient().Get(srv.URL)
	if err != nil {
		t.Fatalf("rede local permitida explicitamente: %v", err)
	}
	resp.Body.Close()
}
