package e2e

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func authorizedKey(t *testing.T, pub any, comment string) (string, string) {
	t.Helper()
	k, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(k))) + " " + comment, ssh.FingerprintSHA256(k)
}

// ID-08: chaves SSH das pessoas em /user/keys — conferidas, com impressão
// digital calculada, cada pessoa vê e remove só as suas, e uma chave não
// pode ser cadastrada duas vezes (nem com outro comentário).
func TestChavesSSH(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	eve := signup(t, base, "eve")

	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	key, fp := authorizedKey(t, pub, "ada@notebook")
	k := ada.must("POST", "/api/v4/user/keys", map[string]any{"title": "Notebook", "key": "  " + key + "\n"}, 201)
	if k["title"] != "Notebook" || k["key"] != key || k["fingerprint_sha256"] != fp || k["id"] == nil {
		t.Fatalf("chave criada: %v", k)
	}
	// the same key again (another comment): refused, also for someone else
	other := strings.TrimSuffix(key, "ada@notebook") + "outro"
	ada.must("POST", "/api/v4/user/keys", map[string]any{"title": "De novo", "key": other}, 400)
	eve.must("POST", "/api/v4/user/keys", map[string]any{"title": "Roubada", "key": key}, 400)

	// not a key, a key with options, a weak RSA key, a DSA-like text: refused
	weak, _ := rsa.GenerateKey(rand.Reader, 1024)
	weakKey, _ := authorizedKey(t, &weak.PublicKey, "fraca")
	for _, bad := range []string{"não é uma chave", `command="rm -rf /" ` + key, weakKey, "ssh-dss AAAAB3NzaC1kc3MAAACBAP== x"} {
		code, out, _ := ada.call("POST", "/api/v4/user/keys", map[string]any{"title": "Ruim", "key": bad})
		if code != 400 {
			t.Fatalf("chave inválida aceita (%d): %q → %v", code, bad, out)
		}
	}
	// the fingerprint is the system's, never the person's
	pub2, _, _ := ed25519.GenerateKey(rand.Reader)
	key2, fp2 := authorizedKey(t, pub2, "eve")
	e := eve.must("POST", "/api/v4/user/keys", map[string]any{"title": "Eve", "key": key2, "fingerprint_sha256": "SHA256:forjada"}, 201)
	if e["fingerprint_sha256"] != fp2 {
		t.Fatalf("impressão digital aceita da entrada: %v", e)
	}

	if l := ada.list("/api/v4/user/keys"); len(l) != 1 || l[0].(map[string]any)["title"] != "Notebook" {
		t.Fatalf("chaves de ada: %v", l)
	}
	if l := eve.list("/api/v4/user/keys"); len(l) != 1 || l[0].(map[string]any)["title"] != "Eve" {
		t.Fatalf("chaves de eve: %v", l)
	}
	if code, _, _ := eve.call("GET", "/api/v4/user/keys/"+id(k), nil); code != 404 {
		t.Fatalf("eve vê a chave de ada: %d", code)
	}
	if code, _, _ := eve.call("DELETE", "/api/v4/user/keys/"+id(k), nil); code != 404 {
		t.Fatalf("eve remove a chave de ada: %d", code)
	}
	// a key is not edited: it is removed and another one added
	ada.must("PUT", "/api/v4/user/keys/"+id(k), map[string]any{"key": key2}, 403) // nobody may edit keys here
	anon := &api{t: t, base: base}
	if code, _, _ := anon.call("GET", "/api/v4/user/keys", nil); code != 401 {
		t.Fatalf("visitante lista chaves: %d", code)
	}
	ada.must("DELETE", "/api/v4/user/keys/"+id(k), nil, 204)
	if l := ada.list("/api/v4/user/keys"); len(l) != 0 {
		t.Fatalf("chave removida continua: %v", l)
	}
}
