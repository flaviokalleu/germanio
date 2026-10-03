package interpreter

import (
	"crypto/rsa"
	"strings"

	"golang.org/x/crypto/ssh"
)

// Public keys (`chave pública`, GEP 0032, em teste). A key is accepted in
// the authorized_keys form ("type base64 [comment]"), checked by the SSH
// library (never parsed by hand), and written back in one canonical form;
// its SHA-256 fingerprint is derived. Weak or obsolete kinds are refused by
// default: DSA, and RSA below 2048 bits.

// maxPublicKey bounds the text read (a 16384-bit RSA key is about 2.8 KB).
const maxPublicKey = 8192

// minRSABits is the smallest RSA key accepted.
const minRSABits = 2048

// PublicKey checks text and returns the canonical key, its fingerprint
// ("SHA256:…") and, when refused, the reason (English, translated like the
// other validation messages).
func PublicKey(text string) (canonical, fingerprint, reason string) {
	text = strings.TrimSpace(text)
	if text == "" || len(text) > maxPublicKey || strings.ContainsAny(text, "\r\n") {
		return "", "", "is invalid"
	}
	pub, comment, options, rest, err := ssh.ParseAuthorizedKey([]byte(text))
	if err != nil || len(options) > 0 || len(strings.TrimSpace(string(rest))) > 0 {
		return "", "", "is invalid"
	}
	switch pub.Type() {
	case ssh.KeyAlgoDSA:
		return "", "", "uses an algorithm that is no longer safe (DSA)"
	case ssh.KeyAlgoRSA:
		if ck, ok := pub.(ssh.CryptoPublicKey); ok {
			if rk, ok := ck.CryptoPublicKey().(*rsa.PublicKey); ok && rk.N.BitLen() < minRSABits {
				return "", "", "is too weak (RSA needs at least 2048 bits)"
			}
		}
	}
	canonical = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pub)))
	if comment = strings.TrimSpace(comment); comment != "" {
		if len(comment) > 255 {
			comment = comment[:255]
		}
		canonical += " " + comment
	}
	return canonical, ssh.FingerprintSHA256(pub), ""
}
