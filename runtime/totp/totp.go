// Package totp implements time-based one-time passwords (RFC 6238 over the
// HOTP of RFC 4226) with HMAC-SHA1, the algorithm authenticator apps use.
// It is the standard algorithm built from the standard library (crypto/hmac,
// crypto/sha1); nothing here is a new cryptographic construction.
package totp

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Period is the time step of a code; Digits is its length (the values every
// authenticator app uses by default).
const (
	Period = 30
	Digits = 6
)

// Step is the time step of t.
func Step(t time.Time) uint64 { return uint64(t.Unix()) / Period }

// CodeAt is the HOTP value of secret at counter, with digits digits.
func CodeAt(secret []byte, counter uint64, digits int) string {
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)
	m := hmac.New(sha1.New, secret)
	m.Write(msg[:])
	sum := m.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	bin := (uint32(sum[off])&0x7f)<<24 | uint32(sum[off+1])<<16 | uint32(sum[off+2])<<8 | uint32(sum[off+3])
	mod := uint32(1)
	for i := 0; i < digits; i++ {
		mod *= 10
	}
	return fmt.Sprintf("%0*d", digits, bin%mod)
}

// Check compares code with the codes of the steps around now (one step of
// clock drift each way) in constant time, and returns the step that matched.
// A step at or before last (the last accepted step) never matches again: a
// code works once.
func Check(secret []byte, code string, now time.Time, last uint64) (uint64, bool) {
	code = strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' {
			return -1
		}
		return r
	}, code)
	if len(code) != Digits {
		return 0, false
	}
	cur := Step(now)
	var found uint64
	ok := 0
	for _, s := range []uint64{cur - 1, cur, cur + 1} {
		eq := subtle.ConstantTimeCompare([]byte(CodeAt(secret, s, Digits)), []byte(code))
		if eq == 1 && s > last && found == 0 {
			found = s
		}
		ok |= eq
	}
	return found, ok == 1 && found != 0
}

// Encode writes a secret the way authenticator apps read it (base32, no padding).
func Encode(secret []byte) string {
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret)
}

// URI is the otpauth:// address that apps read (usually as a QR code).
func URI(secret []byte, issuer, account string) string {
	label := url.PathEscape(issuer + ":" + account)
	q := url.Values{"secret": {Encode(secret)}, "issuer": {issuer}, "algorithm": {"SHA1"}, "digits": {fmt.Sprint(Digits)}, "period": {fmt.Sprint(Period)}}
	return "otpauth://totp/" + label + "?" + q.Encode()
}
