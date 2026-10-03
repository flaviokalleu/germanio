// Package cofre seals values the application must read back to talk to other
// systems (credentials of outgoing connections, tokens of webhooks, values
// handed to executions) so they are never stored in clear (GEP 0049).
//
// The mechanism is deliberately small and standard: AES-256-GCM from the
// standard library with a random 96-bit nonce per value, the key derived with
// HKDF-SHA256 from the server's secret (GERMANIO_SEGREDO) under a label of
// its own, and authenticated data that binds each value to its purpose (the
// table and column of a field): a value copied to another column does not
// open. Nothing here is invented: no custom cipher, no custom MAC.
//
// Format (versioned): "ge1:<key id>:<base64url(nonce || ciphertext)>". The
// key id is derived from the secret with HKDF under another label, so it
// names the key without revealing it, and lets a server holding several keys
// (the current one and previous ones during a rotation) pick the right one.
// A value without the prefix is legacy plain text: it is returned as it is,
// and rewritten sealed by the caller.
package cofre

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
)

// Prefixo starts every sealed value; the version is part of it.
const Prefixo = "ge1:"

// TamanhoMinimo is the shortest secret accepted (the same rule as sessions).
const TamanhoMinimo = 32

const (
	labelKey = "germanio/segredos-em-repouso/v1"
	labelID  = "germanio/segredos-em-repouso/id"
)

var (
	// ErrIlegivel: the value is not a sealed value this format can read
	// (truncated, altered, or moved from another purpose).
	ErrIlegivel = errors.New("segredo ilegível")
	// ErrChaveDesconhecida: the value was sealed with a key this server does
	// not have.
	ErrChaveDesconhecida = errors.New("segredo cifrado com uma chave que este servidor não tem")
)

type chave struct {
	id      string
	aead    cipher.AEAD
	segredo []byte
}

// Cofre seals with the current key and opens with the current or a previous
// one. It is safe for concurrent use.
type Cofre struct {
	atual chave
	todas []chave // current first, then the previous ones
}

func derivar(segredo string) (chave, error) {
	if len(segredo) < TamanhoMinimo {
		return chave{}, fmt.Errorf("a chave tem %d caracteres; são precisos pelo menos %d", len(segredo), TamanhoMinimo)
	}
	key, err := hkdf.Key(sha256.New, []byte(segredo), nil, labelKey, 32)
	if err != nil {
		return chave{}, err
	}
	id, err := hkdf.Key(sha256.New, []byte(segredo), nil, labelID, 6)
	if err != nil {
		return chave{}, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return chave{}, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return chave{}, err
	}
	return chave{id: hex.EncodeToString(id), aead: aead, segredo: []byte(segredo)}, nil
}

// Novo builds a Cofre that seals with atual and also opens what was sealed
// with any of anteriores (a key rotation in progress).
func Novo(atual string, anteriores ...string) (*Cofre, error) {
	cur, err := derivar(atual)
	if err != nil {
		return nil, err
	}
	c := &Cofre{atual: cur, todas: []chave{cur}}
	for _, s := range anteriores {
		k, err := derivar(s)
		if err != nil {
			return nil, fmt.Errorf("GERMANIO_SEGREDO_ANTERIOR: %w", err)
		}
		if k.id != cur.id {
			c.todas = append(c.todas, k)
		}
	}
	return c, nil
}

// DoAmbiente builds the Cofre of this server from GERMANIO_SEGREDO (the
// current key) and GERMANIO_SEGREDO_ANTERIOR (previous keys, separated by
// spaces). Without GERMANIO_SEGREDO it returns nil: there is no lasting key.
// A key that is set but too short is an error, never silently ignored.
func DoAmbiente() (*Cofre, error) {
	atual := os.Getenv("GERMANIO_SEGREDO")
	anteriores := strings.Fields(os.Getenv("GERMANIO_SEGREDO_ANTERIOR"))
	if atual == "" {
		if len(anteriores) > 0 {
			return nil, errors.New("GERMANIO_SEGREDO_ANTERIOR está definida, mas falta GERMANIO_SEGREDO.\nPor quê: a chave anterior só serve para ler o que foi cifrado com ela e cifrar de novo com a chave atual.\nComo corrigir: defina GERMANIO_SEGREDO com a chave nova (pelo menos 32 caracteres)")
		}
		return nil, nil
	}
	if len(atual) < TamanhoMinimo {
		return nil, fmt.Errorf("GERMANIO_SEGREDO tem %d caracteres; são precisos pelo menos %d.\nPor quê: ela cifra os segredos guardados no banco e assina as sessões; uma chave curta é fácil de adivinhar.\nComo corrigir: gere uma com, por exemplo, openssl rand -base64 48", len(atual), TamanhoMinimo)
	}
	return Novo(atual, anteriores...)
}

func aad(proposito string) []byte { return []byte(Prefixo + proposito) }

// Selar seals valor for proposito (for a field: "tabela.coluna").
func (c *Cofre) Selar(proposito, valor string) (string, error) {
	nonce := make([]byte, c.atual.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	out := c.atual.aead.Seal(nonce, nonce, []byte(valor), aad(proposito))
	return Prefixo + c.atual.id + ":" + base64.RawURLEncoding.EncodeToString(out), nil
}

// Abrir returns the clear value of v sealed for proposito. A value without
// the prefix is legacy plain text and is returned unchanged.
func (c *Cofre) Abrir(proposito, v string) (string, error) {
	if !Selado(v) {
		return v, nil
	}
	id, body, ok := partes(v)
	if !ok {
		return "", ErrIlegivel
	}
	for _, k := range c.todas {
		if k.id != id {
			continue
		}
		raw, err := base64.RawURLEncoding.DecodeString(body)
		if err != nil || len(raw) < k.aead.NonceSize() {
			return "", ErrIlegivel
		}
		n := k.aead.NonceSize()
		plain, err := k.aead.Open(nil, raw[:n], raw[n:], aad(proposito))
		if err != nil {
			return "", ErrIlegivel
		}
		return string(plain), nil
	}
	return "", ErrChaveDesconhecida
}

// Selado reports whether v is in the sealed format (whatever the key).
func Selado(v string) bool { return strings.HasPrefix(v, Prefixo) }

func partes(v string) (id, body string, ok bool) {
	rest := strings.TrimPrefix(v, Prefixo)
	i := strings.IndexByte(rest, ':')
	if i <= 0 {
		return "", "", false
	}
	return rest[:i], rest[i+1:], true
}

// Atual reports whether v is sealed with the current key (nothing to do).
func (c *Cofre) Atual(v string) bool {
	id, _, ok := partes(v)
	return Selado(v) && ok && id == c.atual.id
}

// Conhece reports whether v is sealed with a key this Cofre holds.
func (c *Cofre) Conhece(v string) bool {
	id, _, ok := partes(v)
	if !Selado(v) || !ok {
		return false
	}
	for _, k := range c.todas {
		if k.id == id {
			return true
		}
	}
	return false
}

// PrefixoAtual is what every value sealed with the current key starts with
// (to find, in the database, the values that still need sealing).
func (c *Cofre) PrefixoAtual() string { return Prefixo + c.atual.id + ":" }

// ChavesLegadas derives, for each key held (current first), the 32-byte key
// of an older format that used HKDF-SHA256 under rotulo directly. It lets
// such values be opened once and sealed again in this format.
func (c *Cofre) ChavesLegadas(rotulo string) [][]byte {
	var out [][]byte
	for _, k := range c.todas {
		if key, err := hkdf.Key(sha256.New, k.segredo, nil, rotulo, 32); err == nil {
			out = append(out, key)
		}
	}
	return out
}
