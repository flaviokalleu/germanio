package runtime

import (
	"bufio"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// wsHandshake opens a raw WebSocket handshake and returns the status line and
// the connection (open only when the upgrade succeeded).
func wsHandshake(t *testing.T, base, path string, header http.Header) (int, net.Conn, *bufio.Reader) {
	t.Helper()
	conn, err := net.Dial("tcp", strings.TrimPrefix(base, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest("GET", base+path, nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	for k, v := range header {
		req.Header[k] = v
	}
	req.Write(conn)
	rd := bufio.NewReader(conn)
	resp, err := http.ReadResponse(rd, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		conn.Close()
		return resp.StatusCode, nil, nil
	}
	t.Cleanup(func() { conn.Close() })
	return resp.StatusCode, conn, rd
}

// readFrame reads one unmasked server text frame.
func readFrame(t *testing.T, conn net.Conn, rd *bufio.Reader) string {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	for {
		h := make([]byte, 2)
		if _, err := rd.Read(h[:1]); err != nil {
			t.Fatalf("nenhuma mensagem: %v", err)
		}
		rd.Read(h[1:])
		n := int(h[1] & 0x7f)
		if n == 126 {
			b := make([]byte, 2)
			rd.Read(b)
			n = int(b[0])<<8 | int(b[1])
		}
		body := make([]byte, n)
		for got := 0; got < n; {
			m, err := rd.Read(body[got:])
			if err != nil {
				t.Fatal(err)
			}
			got += m
		}
		if !strings.Contains(string(body), "presenca_socket") {
			return string(body)
		}
	}
}

// Real-time updates must not become a side door around authentication and
// permissions: no socket without a session, no socket for another site, and
// change notifications carry no record data (the page reloads it through the
// API, which applies the permissions of whoever is looking).
func TestTempoRealNaoVazaDados(t *testing.T) {
	t.Setenv("JWT_SECRET", "segredo-de-teste-com-tamanho-suficiente-123")
	_, c := loadApp(t, "testdata/tempo_real/app.ge")

	if status, _, _ := wsHandshake(t, c.base, "/ws", nil); status != http.StatusUnauthorized {
		t.Fatalf("/ws sem sessão: status %d, esperado 401", status)
	}

	c.expect("POST", "/api/registro", map[string]any{"nome": "Ana", "email": "ana@x.com", "senha": "senha-forte-123"}, 201)
	login := c.expect("POST", "/api/login", map[string]any{"email": "ana@x.com", "senha": "senha-forte-123"}, 200)
	token, _ := login["token"].(string)
	if token == "" {
		t.Fatalf("login sem token: %v", login)
	}

	foreign := http.Header{"Origin": {"https://atacante.example"}}
	if status, _, _ := wsHandshake(t, c.base, "/ws?token="+token, foreign); status != http.StatusForbidden {
		t.Fatalf("/ws de outro site: status %d, esperado 403", status)
	}

	status, conn, rd := wsHandshake(t, c.base, "/ws?token="+token, http.Header{"Origin": {c.base}})
	if status != http.StatusSwitchingProtocols {
		t.Fatalf("/ws com sessão: status %d", status)
	}
	c.Header.Set("Authorization", "Bearer "+token)
	c.expect("POST", "/api/nota", map[string]any{"titulo": "Plano", "segredo": "senha do cofre"}, 201)
	msg := readFrame(t, conn, rd)
	if !strings.Contains(msg, `"model":"nota"`) {
		t.Fatalf("aviso de mudança esperado: %s", msg)
	}
	if strings.Contains(msg, "senha do cofre") || strings.Contains(msg, "Plano") {
		t.Fatalf("o aviso carrega os dados do registro: %s", msg)
	}
}
