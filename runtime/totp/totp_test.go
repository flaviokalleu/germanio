package totp

import (
	"strings"
	"testing"
	"time"
)

// The SHA-1 test vectors of RFC 6238, Appendix B (8 digits, 30-second steps,
// seed "12345678901234567890").
func TestRFC6238(t *testing.T) {
	seed := []byte("12345678901234567890")
	for _, v := range []struct {
		unix int64
		code string
	}{
		{59, "94287082"},
		{1111111109, "07081804"},
		{1111111111, "14050471"},
		{1234567890, "89005924"},
		{2000000000, "69279037"},
		{20000000000, "65353130"},
	} {
		if got := CodeAt(seed, Step(time.Unix(v.unix, 0)), 8); got != v.code {
			t.Fatalf("T=%d: %s, esperado %s", v.unix, got, v.code)
		}
	}
}

// The HOTP test vectors of RFC 4226, Appendix D (6 digits).
func TestRFC4226(t *testing.T) {
	seed := []byte("12345678901234567890")
	want := []string{"755224", "287082", "359152", "969429", "338314", "254676", "287922", "162583", "399871", "520489"}
	for i, w := range want {
		if got := CodeAt(seed, uint64(i), 6); got != w {
			t.Fatalf("contador %d: %s, esperado %s", i, got, w)
		}
	}
}

// A code is accepted in its step and the neighbours, never again once used,
// and wrong codes or lengths are refused.
func TestCheck(t *testing.T) {
	secret := []byte("segredo-de-teste-20b")
	now := time.Unix(1_700_000_000, 0)
	code := CodeAt(secret, Step(now), Digits)
	step, ok := Check(secret, code, now, 0)
	if !ok || step != Step(now) {
		t.Fatalf("código atual recusado")
	}
	if _, ok := Check(secret, code, now, step); ok {
		t.Fatal("o mesmo código foi aceito duas vezes")
	}
	if _, ok := Check(secret, code[:3]+" "+code[3:], now.Add(Period*time.Second), 0); !ok {
		t.Fatal("código do passo anterior (relógio atrasado) recusado")
	}
	if _, ok := Check(secret, code, now.Add(3*Period*time.Second), 0); ok {
		t.Fatal("código vencido aceito")
	}
	wrong := "000000"
	if wrong == code {
		wrong = "111111"
	}
	for _, bad := range []string{wrong, "", "12345", "1234567", "abcdef"} {
		if _, ok := Check(secret, bad, now, 0); ok {
			t.Fatalf("código errado aceito: %q", bad)
		}
	}
	if u := URI(secret, "Meu App", "ana@x.com"); !strings.HasPrefix(u, "otpauth://totp/Meu%20App:ana@x.com?") || !strings.Contains(u, "secret="+Encode(secret)) {
		t.Fatalf("uri: %s", u)
	}
}
