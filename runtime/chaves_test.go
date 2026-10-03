package runtime

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"

	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

func sshKey(t *testing.T, pub any) (string, string) {
	t.Helper()
	k, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(k))), ssh.FingerprintSHA256(k)
}

// A public key (GEP 0032, em teste) is checked by the SSH library, written
// in one canonical form, and gets its fingerprint; weak kinds, options and
// anything else are refused.
func TestChavePublicaValidacao(t *testing.T) {
	ed, _, _ := ed25519.GenerateKey(rand.Reader)
	edKey, edFP := sshKey(t, ed)
	ec, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	ecKey, _ := sshKey(t, &ec.PublicKey)
	strong, _ := rsa.GenerateKey(rand.Reader, 2048)
	strongKey, _ := sshKey(t, &strong.PublicKey)
	weak, _ := rsa.GenerateKey(rand.Reader, 1024)
	weakKey, _ := sshKey(t, &weak.PublicKey)

	canonical, fp, why := interp.PublicKey("  " + edKey + "   ana@notebook \n")
	if why != "" || canonical != edKey+" ana@notebook" || fp != edFP {
		t.Fatalf("ed25519: %q %q %q", canonical, fp, why)
	}
	for _, ok := range []string{ecKey, strongKey, edKey} {
		if _, _, why := interp.PublicKey(ok); why != "" {
			t.Fatalf("chave boa recusada: %s (%s)", ok[:20], why)
		}
	}
	for _, bad := range []string{"", "texto qualquer", weakKey, `no-pty ` + edKey, edKey + "\n" + ecKey, "ssh-ed25519 AAAA", strings.Repeat("a", 9000)} {
		if _, _, why := interp.PublicKey(bad); why == "" {
			t.Fatalf("chave ruim aceita: %.40q", bad)
		}
	}
	if _, _, why := interp.PublicKey(weakKey); !strings.Contains(why, "2048") {
		t.Fatalf("RSA fraca sem explicação: %q", why)
	}
}

// In an app (servers holding people's access keys, no Git, no GitLab): the
// key is stored canonical, the fingerprint is derived and cannot be set,
// the same key with another comment is refused, and the key never changes.
func TestChavePublicaNoApp(t *testing.T) {
	_, c := loadApp(t, "testdata/chaves/app.ge")
	ana := signIn(t, c.base, "Ana", "ana@x.com")
	bia := signIn(t, c.base, "Bia", "bia@x.com")
	ed, _, _ := ed25519.GenerateKey(rand.Reader)
	key, fp := sshKey(t, ed)

	got := ana.expect("POST", "/_ge/api/acessos", map[string]any{"rotulo": "Notebook", "segredo": key + " ana@x", "impressao_digital": "SHA256:forjada"}, 201)
	if got["segredo"] != key+" ana@x" || got["impressao_digital"] != fp {
		t.Fatalf("chave guardada: %v", got)
	}
	out := bia.expect("POST", "/_ge/api/acessos", map[string]any{"rotulo": "Copiada", "segredo": key + " outro-comentario"}, 400)
	if msg, _ := out["message"].(map[string]any); msg == nil || msg["impressao_digital"] == nil {
		t.Fatalf("a mesma chave com outro comentário: %v", out)
	}
	weak, _ := rsa.GenerateKey(rand.Reader, 1024)
	weakKey, _ := sshKey(t, &weak.PublicKey)
	out = ana.expect("POST", "/_ge/api/acessos", map[string]any{"rotulo": "Fraca", "segredo": weakKey}, 400)
	if !strings.Contains(strings.ToLower(asJSON(out)), "fraca") {
		t.Fatalf("mensagem para chave fraca: %v", out)
	}
	ana.expect("POST", "/_ge/api/acessos", map[string]any{"rotulo": "Lixo", "segredo": "isto não é chave"}, 400)
	// editing the label works; the key itself never changes
	ana.expect("PUT", "/_ge/api/acessos/1", map[string]any{"rotulo": "Notebook novo"}, 200)
	ed2, _, _ := ed25519.GenerateKey(rand.Reader)
	key2, _ := sshKey(t, ed2)
	ana.expect("PUT", "/_ge/api/acessos/1", map[string]any{"segredo": key2}, 400)
	if row := ana.expect("GET", "/_ge/api/acessos/1", nil, 200); row["impressao_digital"] != fp || row["rotulo"] != "Notebook novo" {
		t.Fatalf("depois de editar: %v", row)
	}
}

func asJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
