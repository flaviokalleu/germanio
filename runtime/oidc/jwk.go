package oidc

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
)

// jwk is one verification key of the provider (RFC 7517), limited to the
// two kinds OpenID providers sign ID tokens with: RSA (RS256, at least 2048
// bits) and EC P-256 (ES256).
type jwk struct {
	kid string
	alg string // the key's own alg, when it declares one
	rsa *rsa.PublicKey
	ec  *ecdsa.PublicKey
}

func (k jwk) fits(alg string) bool {
	if k.alg != "" && k.alg != alg {
		return false
	}
	switch alg {
	case "RS256":
		return k.rsa != nil
	case "ES256":
		return k.ec != nil
	}
	return false
}

func (k jwk) verify(alg string, signed, sig []byte) bool {
	sum := sha256.Sum256(signed)
	switch {
	case alg == "RS256" && k.rsa != nil:
		return rsa.VerifyPKCS1v15(k.rsa, crypto.SHA256, sum[:], sig) == nil
	case alg == "ES256" && k.ec != nil:
		// JWS (RFC 7518 §3.4): r and s, 32 bytes each, not ASN.1
		if len(sig) != 64 {
			return false
		}
		r, s := new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:])
		return ecdsa.Verify(k.ec, sum[:], r, s)
	}
	return false
}

var errKey = errors.New("chave não aceita")

func parseJWK(raw json.RawMessage) (jwk, error) {
	var j struct {
		Kty string `json:"kty"`
		Kid string `json:"kid"`
		Use string `json:"use"`
		Alg string `json:"alg"`
		N   string `json:"n"`
		E   string `json:"e"`
		Crv string `json:"crv"`
		X   string `json:"x"`
		Y   string `json:"y"`
	}
	if err := json.Unmarshal(raw, &j); err != nil {
		return jwk{}, err
	}
	if j.Use != "" && j.Use != "sig" {
		return jwk{}, errKey
	}
	k := jwk{kid: j.Kid, alg: j.Alg}
	b64 := base64.RawURLEncoding
	switch j.Kty {
	case "RSA":
		n, err1 := b64.DecodeString(j.N)
		e, err2 := b64.DecodeString(j.E)
		if err1 != nil || err2 != nil || len(e) == 0 || len(e) > 4 {
			return jwk{}, errKey
		}
		pub := &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
		if pub.N.BitLen() < 2048 || pub.E < 3 || pub.E%2 == 0 {
			return jwk{}, errKey
		}
		k.rsa = pub
	case "EC":
		if j.Crv != "P-256" {
			return jwk{}, errKey
		}
		x, err1 := b64.DecodeString(j.X)
		y, err2 := b64.DecodeString(j.Y)
		if err1 != nil || err2 != nil || len(x) != 32 || len(y) != 32 {
			return jwk{}, errKey
		}
		// the parser checks that the point is on the curve
		pub, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), append(append([]byte{4}, x...), y...))
		if err != nil {
			return jwk{}, errKey
		}
		k.ec = pub
	default:
		return jwk{}, errKey
	}
	return k, nil
}
