package interpreter

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// builtinNames feeds "você quis dizer" suggestions.
var builtinNames = []string{
	"tamanho", "tipo", "texto", "numero", "arredondar", "aleatorio", "agora", "maiusculo", "minusculo",
	"contem", "dividir", "juntar", "abs", "min", "max", "inteiro", "substituir", "cortar", "comeca_com",
	"termina_com", "substring", "adicionar", "remover", "reverter", "chaves", "valores", "json",
	"formato_data", "potencia", "raiz", "chamar", "uuid", "base64_codificar", "base64_decodificar",
	"hash_sha256", "primeiro", "ultimo", "fatia", "ordenar_por", "unicos", "mesclar", "copiar", "vazio",
	"indice_de", "repetir_texto", "codificar_url", "decodificar_url", "agora_iso",
}

var (
	dummyOnce   sync.Once
	dummyHash   []byte
	segredoOnce sync.Once
	segredo     []byte
)

// Segredo returns the key used to sign tokens and sessions: GERMANIO_SEGREDO
// when set (at least 32 bytes), otherwise a random per-process key. The
// fallback is safe but sessions do not survive a restart.
func Segredo() []byte {
	segredoOnce.Do(func() {
		if s := os.Getenv("GERMANIO_SEGREDO"); len(s) >= 32 {
			segredo = []byte(s)
			return
		}
		segredo = make([]byte, 32)
		if _, err := rand.Read(segredo); err != nil {
			panic(err)
		}
		fmt.Println("[germanio] AVISO: GERMANIO_SEGREDO ausente ou curto (< 32 bytes); usando chave aleatória por processo.")
	})
	return segredo
}

func randomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func hmacHex(key, msg []byte) string {
	m := hmac.New(sha256.New, key)
	m.Write(msg)
	return hex.EncodeToString(m.Sum(nil))
}

// AssinarToken produces header.payload.signature (JWT HS256).
func AssinarToken(dados map[string]any, validade time.Duration) (string, error) {
	claims := map[string]any{}
	for k, v := range dados {
		claims[k] = v
	}
	if validade > 0 {
		claims["exp"] = time.Now().Add(validade).Unix()
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	head := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	body := base64.RawURLEncoding.EncodeToString(payload)
	m := hmac.New(sha256.New, Segredo())
	m.Write([]byte(head + "." + body))
	return head + "." + body + "." + base64.RawURLEncoding.EncodeToString(m.Sum(nil)), nil
}

// VerificarToken returns the claims of a valid, unexpired token or nil.
func VerificarToken(tok string) map[string]any {
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		return nil
	}
	m := hmac.New(sha256.New, Segredo())
	m.Write([]byte(parts[0] + "." + parts[1]))
	want := m.Sum(nil)
	got, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(want, got) {
		return nil
	}
	if h, err := base64.RawURLEncoding.DecodeString(parts[0]); err != nil || string(h) != `{"alg":"HS256","typ":"JWT"}` {
		return nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}
	var claims map[string]any
	if json.Unmarshal(payload, &claims) != nil {
		return nil
	}
	if exp, ok := claims["exp"].(float64); ok && time.Now().Unix() > int64(exp) {
		return nil
	}
	return claims
}

// SessionCookie is the cookie holding the signed session.
const SessionCookie = "ge_sessao"

// SessaoDaRequisicao returns the verified session claims of a request.
func SessaoDaRequisicao(r *http.Request) map[string]any {
	if r == nil {
		return nil
	}
	c, err := r.Cookie(SessionCookie)
	if err != nil {
		return nil
	}
	return VerificarToken(c.Value)
}

func registerStdlib(interp *Interpreter) {
	bcryptCost := bcrypt.DefaultCost
	if os.Getenv("GERMANIO_BCRYPT_RAPIDO") == "1" { // test suites only
		bcryptCost = bcrypt.MinCost
	}
	interp.RegisterModule("cripto", map[string]ModuleFunc{
		// hash_senha(senha) → bcrypt hash
		"hash_senha": func(c *Call, args []any) any {
			s := c.Str(args, 0, "senha")
			if len(s) > 72 {
				panic(c.Fail(400, "senha longa demais (máximo 72 bytes)"))
			}
			h, err := bcrypt.GenerateFromPassword([]byte(s), bcryptCost)
			if err != nil {
				panic(c.Fail(0, "hash_senha: %s", err))
			}
			return string(h)
		},
		// verificar_senha(senha, hash) → verdadeiro/falso, tempo constante
		"verificar_senha": func(c *Call, args []any) any {
			s := c.Str(args, 0, "senha")
			h, _ := c.Arg(args, 1, "hash").(string)
			if h == "" {
				// Still spend bcrypt time so absent users are not distinguishable.
				dummyOnce.Do(func() { dummyHash, _ = bcrypt.GenerateFromPassword([]byte("germanio"), bcryptCost) })
				bcrypt.CompareHashAndPassword(dummyHash, []byte(s))
				return false
			}
			return bcrypt.CompareHashAndPassword([]byte(h), []byte(s)) == nil
		},
		// token(bytes=32) → texto aleatório seguro (base64 url)
		"token": func(c *Call, args []any) any {
			n := 32
			if len(args) > 0 {
				n = int(c.Num(args, 0, "bytes"))
			}
			if n < 16 || n > 256 {
				panic(c.Fail(0, "cripto.token: use entre 16 e 256 bytes"))
			}
			return randomToken(n)
		},
		"sha256": func(c *Call, args []any) any {
			sum := sha256.Sum256([]byte(c.Str(args, 0, "texto")))
			return hex.EncodeToString(sum[:])
		},
		"hmac_sha256": func(c *Call, args []any) any {
			return hmacHex([]byte(c.Str(args, 0, "chave")), []byte(c.Str(args, 1, "texto")))
		},
		// iguais(a, b) compares secrets in constant time
		"iguais": func(c *Call, args []any) any {
			return subtle.ConstantTimeCompare([]byte(c.Str(args, 0, "a")), []byte(c.Str(args, 1, "b"))) == 1
		},
	})
	interp.RegisterModule("token", map[string]ModuleFunc{
		// token.assinar(dados, segundos) → texto assinado (HS256)
		"assinar": func(c *Call, args []any) any {
			secs := 0.0
			if len(args) > 1 {
				secs = c.Num(args, 1, "segundos")
			}
			t, err := AssinarToken(c.Map(args, 0, "dados"), time.Duration(secs)*time.Second)
			if err != nil {
				panic(c.Fail(0, "token.assinar: %s", err))
			}
			return t
		},
		// token.verificar(texto) → dados ou nulo
		"verificar": func(c *Call, args []any) any {
			s, _ := c.Arg(args, 0, "token").(string)
			if claims := VerificarToken(s); claims != nil {
				return claims
			}
			return nil
		},
	})
	interp.RegisterModule("sessao", map[string]ModuleFunc{
		// sessao.iniciar(dados, segundos=86400) grava o cookie assinado e
		// devolve os dados com o token CSRF gerado.
		"iniciar": func(c *Call, args []any) any {
			ctx := c.Ctx()
			dados := map[string]any{}
			for k, v := range c.Map(args, 0, "dados") {
				dados[k] = v
			}
			secs := 86400.0
			if len(args) > 1 {
				secs = c.Num(args, 1, "segundos")
			}
			dados["csrf"] = randomToken(24)
			t, err := AssinarToken(dados, time.Duration(secs)*time.Second)
			if err != nil {
				panic(c.Fail(0, "sessao.iniciar: %s", err))
			}
			ctx.SetCookies = append(ctx.SetCookies, &http.Cookie{
				Name: SessionCookie, Value: t, Path: "/", HttpOnly: true,
				SameSite: http.SameSiteLaxMode, MaxAge: int(secs),
				Secure: ctx.Request != nil && ctx.Request.TLS != nil,
			})
			if ctx.Values == nil {
				ctx.Values = map[string]any{}
			}
			ctx.Values["sessao"] = dados
			return dados
		},
		"encerrar": func(c *Call, args []any) any {
			ctx := c.Ctx()
			ctx.SetCookies = append(ctx.SetCookies, &http.Cookie{Name: SessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
			if ctx.Values != nil {
				delete(ctx.Values, "sessao")
			}
			return true
		},
		// sessao.atual() → dados da sessão ou nulo
		"atual": func(c *Call, args []any) any {
			ctx := c.Ctx()
			if ctx.Values != nil {
				if s, ok := ctx.Values["sessao"].(map[string]any); ok {
					return s
				}
			}
			if s := SessaoDaRequisicao(ctx.Request); s != nil {
				return s
			}
			return nil
		},
	})
	interp.RegisterModule("sequencia", map[string]ModuleFunc{
		// sequencia.proxima("issues:42") → 1, 2, 3… atômico por chave
		"proxima": func(c *Call, args []any) any {
			if interp.DB == nil {
				panic(c.Fail(0, "sequencia.proxima exige banco de dados"))
			}
			v, err := interp.DB.Sequencia(c.Str(args, 0, "chave"))
			if err != nil {
				panic(c.Fail(0, "sequencia.proxima: %s", err))
			}
			return float64(v)
		},
	})
	interp.RegisterModule("json", map[string]ModuleFunc{
		"ler": func(c *Call, args []any) any {
			var v any
			if err := json.Unmarshal([]byte(c.Str(args, 0, "texto")), &v); err != nil {
				panic(c.Fail(400, "JSON inválido: %s", err))
			}
			return v
		},
		"escrever": func(c *Call, args []any) any {
			b, err := json.Marshal(c.Arg(args, 0, "valor"))
			if err != nil {
				panic(c.Fail(0, "json.escrever: %s", err))
			}
			return string(b)
		},
	})
}

// extraBuiltins are generic helpers added for larger applications.
func extraBuiltin(name string, args []any) (any, bool) {
	arg := func(i int) any {
		if i < len(args) {
			return args[i]
		}
		return nil
	}
	switch name {
	case "primeiro":
		if l, ok := arg(0).([]any); ok && len(l) > 0 {
			return l[0], true
		}
		return nil, true
	case "ultimo":
		if l, ok := arg(0).([]any); ok && len(l) > 0 {
			return l[len(l)-1], true
		}
		return nil, true
	case "fatia":
		l, ok := arg(0).([]any)
		if !ok {
			return nil, false
		}
		start, end := int(toNumber(arg(1))), len(l)
		if len(args) > 2 {
			end = int(toNumber(arg(2)))
		}
		start = max(0, min(start, len(l)))
		end = max(start, min(end, len(l)))
		out := make([]any, end-start)
		copy(out, l[start:end])
		return out, true
	case "vazio":
		switch v := arg(0).(type) {
		case nil:
			return true, true
		case string:
			return strings.TrimSpace(v) == "", true
		case []any:
			return len(v) == 0, true
		case map[string]any:
			return len(v) == 0, true
		}
		return false, true
	case "mesclar":
		out := map[string]any{}
		for _, a := range args {
			if m, ok := a.(map[string]any); ok {
				for k, v := range m {
					out[k] = v
				}
			}
		}
		return out, true
	case "copiar":
		switch v := arg(0).(type) {
		case map[string]any:
			out := make(map[string]any, len(v))
			for k, x := range v {
				out[k] = x
			}
			return out, true
		case []any:
			out := make([]any, len(v))
			copy(out, v)
			return out, true
		}
		return arg(0), true
	case "unicos":
		l, _ := arg(0).([]any)
		seen := map[string]bool{}
		out := []any{}
		for _, v := range l {
			k := toString(v)
			if !seen[k] {
				seen[k] = true
				out = append(out, v)
			}
		}
		return out, true
	case "indice_de":
		if l, ok := arg(0).([]any); ok {
			for i, v := range l {
				if isEqual(v, arg(1)) {
					return float64(i), true
				}
			}
			return float64(-1), true
		}
		if s, ok := arg(0).(string); ok {
			return float64(strings.Index(s, toString(arg(1)))), true
		}
		return float64(-1), true
	case "ordenar_por":
		// ordenar_por(lista_de_mapas, "campo") — estável; "-campo" decrescente
		l, _ := arg(0).([]any)
		field := toString(arg(1))
		desc := strings.HasPrefix(field, "-")
		field = strings.TrimPrefix(field, "-")
		out := make([]any, len(l))
		copy(out, l)
		key := func(v any) any {
			if m, ok := v.(map[string]any); ok {
				return m[field]
			}
			return v
		}
		sort.SliceStable(out, func(i, j int) bool {
			a, b := key(out[i]), key(out[j])
			var less bool
			if an, ok := tryNumber(a); ok {
				bn, _ := tryNumber(b)
				less = an < bn
			} else {
				less = toString(a) < toString(b)
			}
			if desc {
				return !less && !isEqual(a, b)
			}
			return less
		})
		return out, true
	case "codificar_url":
		return url.PathEscape(toString(arg(0))), true
	case "decodificar_url":
		s, err := url.PathUnescape(toString(arg(0)))
		if err != nil {
			return nil, true
		}
		return s, true
	case "agora_iso":
		return time.Now().UTC().Format(time.RFC3339), true
	case "repetir_texto":
		n := int(toNumber(arg(1)))
		if n < 0 || n > 10000 {
			n = 0
		}
		return strings.Repeat(toString(arg(0)), n), true
	}
	return nil, false
}
