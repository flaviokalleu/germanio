package servidor

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/flaviokalleu/germanio/runtime/banco"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
	"github.com/flaviokalleu/germanio/runtime/totp"
)

// Two-factor authentication (docs/gep/0032-dois-fatores-e-chaves.md, em
// teste). Opt-in per person: a TOTP secret (RFC 6238) shared with an
// authenticator app. The secret is stored encrypted (AES-256-GCM, key
// derived with HKDF-SHA256 from GERMANIO_SEGREDO, bound to the person) and
// shown only while it is being set up; recovery codes are random, shown
// once, stored as SHA-256 and work once. With the second factor on, the
// password alone opens nothing: signing in asks for a code (a one-use
// challenge kept on the server, never a signed cookie that could pass for a
// session), wrong codes count toward the lock of the login, and a code is
// never accepted twice. Git and OAuth with the password are refused; access
// tokens keep working, as their own credential.

const (
	factorTable     = "_germanio_dois_fatores"
	factorCodes     = "_germanio_codigos_recuperacao"
	factorPending   = "_germanio_desafios"
	challengeTTL    = 5 * time.Minute
	recoveryCount   = 10
	challengeCookie = "ge_desafio"
)

const (
	msgWrongCode     = "Código incorreto."
	msgChallengeGone = "O pedido do código venceu. Entre de novo com a senha."
	msgNeedSecretKey = "Os dois fatores precisam de GERMANIO_SEGREDO (pelo menos 32 caracteres) no servidor, para guardar o segredo cifrado."
)

var errFactorRefused = errors.New("código recusado")

func pid(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case float64:
		return int64(x)
	case string:
		n, _ := strconv.ParseInt(x, 10, 64)
		return n
	}
	return 0
}

// factorKeyReady: the server has a lasting key; without it a secret saved
// now would be unreadable after a restart, so turning the factor on waits.
func factorKeyReady() bool { return len(os.Getenv("GERMANIO_SEGREDO")) >= 32 }

func factorAEAD() (cipher.AEAD, error) {
	key, err := hkdf.Key(sha256.New, interp.Segredo(), nil, "germanio/dois-fatores/v1", 32)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// sealSecret encrypts secret for person (the person's id is authenticated
// data: a row copied to someone else does not open).
func sealSecret(person int64, secret []byte) (string, error) {
	aead, err := factorAEAD()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	out := aead.Seal(nonce, nonce, secret, []byte(strconv.FormatInt(person, 10)))
	return "v1:" + base64.RawStdEncoding.EncodeToString(out), nil
}

func openSecret(person int64, sealed string) ([]byte, error) {
	raw, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(sealed, "v1:"))
	if err != nil || !strings.HasPrefix(sealed, "v1:") {
		return nil, errors.New("segredo ilegível")
	}
	aead, err := factorAEAD()
	if err != nil {
		return nil, err
	}
	if len(raw) < aead.NonceSize() {
		return nil, errors.New("segredo ilegível")
	}
	return aead.Open(nil, raw[:aead.NonceSize()], raw[aead.NonceSize():], []byte(strconv.FormatInt(person, 10)))
}

// newRecoveryCodes: recoveryCount codes of 80 random bits each, written
// xxxx-xxxx-xxxx-xxxx.
func newRecoveryCodes() ([]string, error) {
	enc := base32.StdEncoding.WithPadding(base32.NoPadding)
	var out []string
	for i := 0; i < recoveryCount; i++ {
		raw := make([]byte, 10)
		if _, err := rand.Read(raw); err != nil {
			return nil, err
		}
		s := strings.ToLower(enc.EncodeToString(raw))
		out = append(out, s[0:4]+"-"+s[4:8]+"-"+s[8:12]+"-"+s[12:16])
	}
	return out, nil
}

func normalizeRecovery(code string) string {
	return strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' {
			return -1
		}
		if r >= 'A' && r <= 'Z' {
			return r + ('a' - 'A')
		}
		return r
	}, code)
}

func isTOTPShape(code string) bool {
	code = strings.NewReplacer(" ", "", "-", "").Replace(code)
	if len(code) != totp.Digits {
		return false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (a *intentAPI) mountSecondFactor(mux *routeMux) {
	a.s.DB.DB.Exec(`CREATE TABLE IF NOT EXISTS ` + factorTable + ` (pessoa_id INTEGER PRIMARY KEY, segredo TEXT NOT NULL, ativo INTEGER NOT NULL DEFAULT 0, ultimo_passo INTEGER NOT NULL DEFAULT 0)`)
	a.s.DB.DB.Exec(`CREATE TABLE IF NOT EXISTS ` + factorCodes + ` (id ` + a.idColumn() + `, pessoa_id INTEGER NOT NULL, hash TEXT NOT NULL UNIQUE)`)
	a.s.DB.DB.Exec(`CREATE TABLE IF NOT EXISTS ` + factorPending + ` (id ` + a.idColumn() + `, pessoa_id INTEGER NOT NULL, hash TEXT NOT NULL UNIQUE, expira_em TEXT NOT NULL)`)
	if !factorKeyReady() {
		fmt.Println("[germanio] dois fatores: defina GERMANIO_SEGREDO (≥ 32 caracteres) para que as pessoas possam ligá-los; sem ela, ligar fica recusado")
	}
}

// askSecondFactor answers a right password of someone with the second
// factor on: a one-use challenge, kept on the server for 5 minutes.
func (a *intentAPI) askSecondFactor(ctx *interp.Context, w http.ResponseWriter, r *http.Request, user map[string]any) {
	token, err := newToken()
	if err != nil {
		a.failErr(w, r, err)
		return
	}
	now := time.Now().UTC()
	a.s.DB.Executar(fmt.Sprintf(`DELETE FROM %s WHERE expira_em < %s`, factorPending, a.s.ph(1)), now.Format(time.RFC3339))
	if _, err := a.s.DB.Executar(fmt.Sprintf(`INSERT INTO %s (pessoa_id, hash, expira_em) VALUES (%s, %s, %s)`, factorPending, a.s.ph(1), a.s.ph(2), a.s.ph(3)), user["id"], hashToken(token), now.Add(challengeTTL).Format(time.RFC3339)); err != nil {
		a.failErr(w, r, err)
		return
	}
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		http.SetCookie(w, &http.Cookie{Name: challengeCookie, Value: token, Path: "/entrar", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: int(challengeTTL.Seconds()), Secure: r.TLS != nil})
		http.Redirect(w, r, "/entrar/codigo", http.StatusSeeOther)
		return
	}
	a.json(w, http.StatusAccepted, map[string]any{"dois_fatores": true, "desafio": token,
		"message": "Senha certa. Agora envie o código do aplicativo autenticador (ou um código de recuperação) com este desafio para /entrar/codigo."}, nil)
}

// checkFactor tells whether code (an app code or a recovery code) is right
// for person, using it up: an app code moves the last accepted step (a code
// never works twice, even sent twice at once), a recovery code is deleted.
func (a *intentAPI) checkFactor(db *banco.Banco, person int64, code string) (bool, error) {
	var sealed string
	var active, last int64
	row := db.Linha(fmt.Sprintf(`SELECT segredo, ativo, ultimo_passo FROM %s WHERE pessoa_id = %s`, factorTable, a.s.ph(1)), person)
	if row.Scan(&sealed, &active, &last) != nil || active != 1 {
		return false, nil
	}
	if isTOTPShape(code) {
		secret, err := openSecret(person, sealed)
		if err != nil {
			fmt.Printf("[germanio] dois fatores: o segredo de uma pessoa não abre (GERMANIO_SEGREDO mudou?); ela entra com um código de recuperação\n")
			return false, nil
		}
		step, ok := totp.Check(secret, code, time.Now(), uint64(last))
		if !ok {
			return false, nil
		}
		res, err := db.Executar(fmt.Sprintf(`UPDATE %s SET ultimo_passo = %s WHERE pessoa_id = %s AND ativo = 1 AND ultimo_passo < %s`, factorTable, a.s.ph(1), a.s.ph(2), a.s.ph(3)), int64(step), person, int64(step))
		if err != nil {
			return false, err
		}
		n, _ := res.RowsAffected()
		return n == 1, nil
	}
	norm := normalizeRecovery(code)
	if len(norm) != 16 {
		return false, nil
	}
	res, err := db.Executar(fmt.Sprintf(`DELETE FROM %s WHERE pessoa_id = %s AND hash = %s`, factorCodes, a.s.ph(1), a.s.ph(2)), person, hashToken(norm))
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

// signInWithCode finishes signing in: the challenge and the code. It
// returns the person, or the status and message of the refusal.
func (a *intentAPI) signInWithCode(ctx *interp.Context, challenge, code string) (map[string]any, int, string) {
	var person int64
	var expira string
	row := a.s.DB.DB.QueryRow(fmt.Sprintf(`SELECT pessoa_id, expira_em FROM %s WHERE hash = %s`, factorPending, a.s.ph(1)), hashToken(challenge))
	if challenge == "" || row.Scan(&person, &expira) != nil {
		return nil, http.StatusUnauthorized, msgChallengeGone
	}
	if t, err := time.Parse(time.RFC3339, expira); err != nil || time.Now().After(t) {
		a.s.DB.Executar(fmt.Sprintf(`DELETE FROM %s WHERE hash = %s`, factorPending, a.s.ph(1)), hashToken(challenge))
		return nil, http.StatusUnauthorized, msgChallengeGone
	}
	user := a.s.loadPerson(ctx, person)
	if user == nil || !a.s.active(user) || a.s.locked(user) {
		return nil, http.StatusUnauthorized, msgWrongCode
	}
	err := a.s.DB.EmTransacao(func(tx *banco.Banco) error {
		ok, err := a.checkFactor(tx, person, code)
		if err != nil {
			return err
		}
		if !ok {
			return errFactorRefused
		}
		res, err := tx.Executar(fmt.Sprintf(`DELETE FROM %s WHERE hash = %s AND pessoa_id = %s`, factorPending, a.s.ph(1), a.s.ph(2)), hashToken(challenge), person)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n != 1 {
			return errFactorRefused // the same challenge used twice at once
		}
		return nil
	})
	if err != nil {
		if err == errFactorRefused {
			a.s.failedAttempt(ctx, user)
			return nil, http.StatusUnauthorized, msgWrongCode
		}
		return nil, http.StatusInternalServerError, interp.Friendly(err)
	}
	a.s.clearAttempts(ctx, user)
	return user, 0, ""
}

// passwordOK checks the password of a signed-in person again (to change
// the second factor); a wrong one counts toward the lock.
func (a *intentAPI) passwordOK(ctx *interp.Context, user map[string]any, password string) bool {
	if a.s.locked(user) {
		return false
	}
	ok, _ := a.in.Op(ctx, a.app.LoginEntity, "verificar_senha", user, password)
	if ok != true {
		a.s.failedAttempt(ctx, user)
		return false
	}
	return true
}

// factorState: "" (off), "pendente" (being set up) or "ligado".
func (a *intentAPI) factorState(person int64) (string, string) {
	var sealed string
	var active int64
	if a.s.DB.DB.QueryRow(fmt.Sprintf(`SELECT segredo, ativo FROM %s WHERE pessoa_id = %s`, factorTable, a.s.ph(1)), person).Scan(&sealed, &active) != nil {
		return "", ""
	}
	if active == 1 {
		return "ligado", ""
	}
	return "pendente", sealed
}

// factorEnable starts setting up: a new secret, shown now (and on the setup
// page) until the first right code turns the factor on.
func (a *intentAPI) factorEnable(ctx *interp.Context, user map[string]any, password string) (map[string]any, int, string) {
	if !factorKeyReady() {
		return nil, http.StatusServiceUnavailable, msgNeedSecretKey
	}
	person := pid(user["id"])
	if st, _ := a.factorState(person); st == "ligado" {
		return nil, http.StatusConflict, "Os dois fatores já estão ligados."
	}
	if !a.passwordOK(ctx, user, password) {
		return nil, http.StatusBadRequest, "Senha incorreta."
	}
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		return nil, http.StatusInternalServerError, err.Error()
	}
	sealed, err := sealSecret(person, secret)
	if err != nil {
		return nil, http.StatusInternalServerError, err.Error()
	}
	err = a.s.DB.EmTransacao(func(tx *banco.Banco) error {
		if _, err := tx.Executar(fmt.Sprintf(`DELETE FROM %s WHERE pessoa_id = %s`, factorTable, a.s.ph(1)), person); err != nil {
			return err
		}
		_, err := tx.Executar(fmt.Sprintf(`INSERT INTO %s (pessoa_id, segredo, ativo, ultimo_passo) VALUES (%s, %s, 0, 0)`, factorTable, a.s.ph(1), a.s.ph(2)), person, sealed)
		return err
	})
	if err != nil {
		return nil, http.StatusInternalServerError, interp.Friendly(err)
	}
	return a.setupInfo(user, secret), 0, ""
}

func (a *intentAPI) setupInfo(user map[string]any, secret []byte) map[string]any {
	account := ""
	for _, f := range a.app.Login.Fields {
		if account = toStr(user[f]); account != "" {
			break
		}
	}
	return map[string]any{"segredo": totp.Encode(secret), "uri": totp.URI(secret, a.s.Program.System.Name, account),
		"message": "Cadastre o segredo no aplicativo autenticador e envie o primeiro código para ligar os dois fatores."}
}

// pendingSetup is the secret being set up (never one already on).
func (a *intentAPI) pendingSetup(user map[string]any) map[string]any {
	person := pid(user["id"])
	st, sealed := a.factorState(person)
	if st != "pendente" {
		return nil
	}
	secret, err := openSecret(person, sealed)
	if err != nil {
		return nil
	}
	return a.setupInfo(user, secret)
}

// replaceCodes writes new recovery codes (only their hashes) and returns them.
func (a *intentAPI) replaceCodes(tx *banco.Banco, person int64) ([]string, error) {
	codes, err := newRecoveryCodes()
	if err != nil {
		return nil, err
	}
	if _, err := tx.Executar(fmt.Sprintf(`DELETE FROM %s WHERE pessoa_id = %s`, factorCodes, a.s.ph(1)), person); err != nil {
		return nil, err
	}
	for _, c := range codes {
		if _, err := tx.Executar(fmt.Sprintf(`INSERT INTO %s (pessoa_id, hash) VALUES (%s, %s)`, factorCodes, a.s.ph(1), a.s.ph(2)), person, hashToken(normalizeRecovery(c))); err != nil {
			return nil, err
		}
	}
	return codes, nil
}

// factorConfirm turns the factor on with the first right code and returns
// the recovery codes (shown this once).
func (a *intentAPI) factorConfirm(ctx *interp.Context, user map[string]any, code string) ([]string, int, string) {
	person := pid(user["id"])
	st, sealed := a.factorState(person)
	switch st {
	case "ligado":
		return nil, http.StatusConflict, "Os dois fatores já estão ligados."
	case "":
		return nil, http.StatusBadRequest, "Comece ligando os dois fatores: peça um segredo novo."
	}
	secret, err := openSecret(person, sealed)
	if err != nil {
		return nil, http.StatusBadRequest, "O segredo em preparo não vale mais. Peça um segredo novo."
	}
	step, ok := totp.Check(secret, code, time.Now(), 0)
	if !ok {
		return nil, http.StatusBadRequest, msgWrongCode + " Confira o relógio do aparelho e digite o código que aparece agora."
	}
	var codes []string
	le := a.app.Entities[a.app.LoginEntity]
	err = a.s.DB.EmTransacao(func(tx *banco.Banco) error {
		res, err := tx.Executar(fmt.Sprintf(`UPDATE %s SET ativo = 1, ultimo_passo = %s WHERE pessoa_id = %s AND ativo = 0 AND segredo = %s`, factorTable, a.s.ph(1), a.s.ph(2), a.s.ph(3)), int64(step), person, sealed)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n != 1 {
			return errFactorRefused
		}
		if codes, err = a.replaceCodes(tx, person); err != nil {
			return err
		}
		c := &interp.Context{Request: ctx.Request, DB: tx}
		_, err = a.in.Op(c, le.Singular, "atualizar", person, map[string]any{"dois_fatores": true})
		return err
	})
	if err == errFactorRefused {
		return nil, http.StatusConflict, "O segredo mudou enquanto isso. Abra a página dos dois fatores de novo."
	}
	if err != nil {
		return nil, http.StatusInternalServerError, interp.Friendly(err)
	}
	return codes, 0, ""
}

// factorDisable turns the factor off: the password and a code (or a
// recovery code) are both needed.
func (a *intentAPI) factorDisable(ctx *interp.Context, user map[string]any, password, code string) (int, string) {
	person := pid(user["id"])
	if st, _ := a.factorState(person); st != "ligado" {
		return http.StatusBadRequest, "Os dois fatores não estão ligados."
	}
	if !a.passwordOK(ctx, user, password) {
		return http.StatusBadRequest, "Senha incorreta."
	}
	le := a.app.Entities[a.app.LoginEntity]
	err := a.s.DB.EmTransacao(func(tx *banco.Banco) error {
		ok, err := a.checkFactor(tx, person, code)
		if err != nil {
			return err
		}
		if !ok {
			return errFactorRefused
		}
		for _, t := range []string{factorTable, factorCodes, factorPending} {
			if _, err := tx.Executar(fmt.Sprintf(`DELETE FROM %s WHERE pessoa_id = %s`, t, a.s.ph(1)), person); err != nil {
				return err
			}
		}
		c := &interp.Context{Request: ctx.Request, DB: tx}
		_, err = a.in.Op(c, le.Singular, "atualizar", person, map[string]any{"dois_fatores": false})
		return err
	})
	if err == errFactorRefused {
		a.s.failedAttempt(ctx, user)
		return http.StatusBadRequest, msgWrongCode
	}
	if err != nil {
		return http.StatusInternalServerError, interp.Friendly(err)
	}
	return 0, ""
}

// factorNewCodes replaces the recovery codes (password and an app code).
func (a *intentAPI) factorNewCodes(ctx *interp.Context, user map[string]any, password, code string) ([]string, int, string) {
	person := pid(user["id"])
	if st, _ := a.factorState(person); st != "ligado" {
		return nil, http.StatusBadRequest, "Os dois fatores não estão ligados."
	}
	if !a.passwordOK(ctx, user, password) {
		return nil, http.StatusBadRequest, "Senha incorreta."
	}
	if !isTOTPShape(code) {
		return nil, http.StatusBadRequest, "Use um código do aplicativo autenticador para gerar códigos de recuperação novos."
	}
	var codes []string
	err := a.s.DB.EmTransacao(func(tx *banco.Banco) error {
		ok, err := a.checkFactor(tx, person, code)
		if err != nil {
			return err
		}
		if !ok {
			return errFactorRefused
		}
		codes, err = a.replaceCodes(tx, person)
		return err
	})
	if err == errFactorRefused {
		a.s.failedAttempt(ctx, user)
		return nil, http.StatusBadRequest, msgWrongCode
	}
	if err != nil {
		return nil, http.StatusInternalServerError, interp.Friendly(err)
	}
	return codes, 0, ""
}

// ---------- pages and answers ----------

// sessionPerson is the person signed in on this browser. Changing the
// second factor needs a browser session (and its CSRF proof on writes):
// an access token, OAuth or the password of a git client cannot do it.
func (ps *pageSite) sessionPerson(w http.ResponseWriter, r *http.Request, write bool) (*interp.Context, map[string]any, bool) {
	ctx := &interp.Context{Request: r, Writer: w}
	form := strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
	sess := interp.SessaoDaRequisicao(r)
	header := r.Header.Get("Authorization") != ""
	if t := ps.a.app.Login.TokenHeader; t != "" && r.Header.Get(t) != "" {
		header = true
	}
	user, err := ps.a.s.identify(ctx, r)
	if err != nil || user == nil || sess == nil || header {
		if !write && !header {
			http.Redirect(w, r, "/entrar", http.StatusSeeOther)
		} else {
			ps.a.fail(w, http.StatusUnauthorized, "Entre pela página de login para mudar os dois fatores.")
		}
		return nil, nil, false
	}
	if write {
		got := r.Header.Get("X-CSRF-Token")
		if got == "" && form {
			got = r.FormValue("_csrf")
		}
		if want, _ := sess["csrf"].(string); want == "" || got != want {
			ps.a.fail(w, http.StatusForbidden, "token CSRF ausente ou inválido")
			return nil, nil, false
		}
	}
	return ctx, user, true
}

func (ps *pageSite) secondFactorPages(mux *routeMux) {
	a := ps.a
	isForm := func(r *http.Request) bool {
		return strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
	}
	field := func(body map[string]any, names ...string) string {
		for _, n := range names {
			if v := toStr(body[n]); v != "" {
				return v
			}
		}
		return ""
	}
	page := func(w http.ResponseWriter, r *http.Request, user map[string]any, flash, problem string, codes []string) {
		v := ps.base(r, "Dois fatores")
		v.Flash, v.Error = flash, problem
		st, _ := a.factorState(pid(user["id"]))
		v.Body = htmlOf(factorTpl, map[string]any{"State": st, "Setup": a.pendingSetup(user), "Codes": codes, "CSRF": v.CSRF, "KeyReady": factorKeyReady()})
		status := http.StatusOK
		if problem != "" {
			status = http.StatusBadRequest
		}
		ps.render(w, v, status)
	}
	answer := func(w http.ResponseWriter, r *http.Request, user map[string]any, status int, msg string, ok map[string]any, codes []string, flash string) {
		if isForm(r) {
			if status != 0 {
				page(w, r, user, "", msg, nil)
				return
			}
			page(w, r, user, flash, "", codes)
			return
		}
		if status != 0 {
			a.fail(w, status, msg)
			return
		}
		a.json(w, http.StatusOK, ok, nil)
	}

	mux.HandleFunc("GET /dois-fatores", func(w http.ResponseWriter, r *http.Request) {
		_, user, ok := ps.sessionPerson(w, r, false)
		if !ok {
			return
		}
		page(w, r, user, "", "", nil)
	})
	mux.HandleFunc("POST /dois-fatores/ativar", func(w http.ResponseWriter, r *http.Request) {
		ctx, user, ok := ps.sessionPerson(w, r, true)
		if !ok {
			return
		}
		body, _ := readBody(r)
		info, status, msg := a.factorEnable(ctx, user, field(body, "senha", "password"))
		answer(w, r, user, status, msg, info, nil, "")
	})
	mux.HandleFunc("POST /dois-fatores/confirmar", func(w http.ResponseWriter, r *http.Request) {
		ctx, user, ok := ps.sessionPerson(w, r, true)
		if !ok {
			return
		}
		body, _ := readBody(r)
		codes, status, msg := a.factorConfirm(ctx, user, field(body, "codigo", "code"))
		answer(w, r, user, status, msg, map[string]any{"dois_fatores": true, "codigos_de_recuperacao": codes,
			"message": "Dois fatores ligados. Guarde os códigos de recuperação: cada um vale uma vez e eles não serão mostrados de novo."}, codes,
			"Dois fatores ligados. Guarde os códigos de recuperação abaixo: cada um vale uma vez e eles não serão mostrados de novo.")
	})
	mux.HandleFunc("POST /dois-fatores/desativar", func(w http.ResponseWriter, r *http.Request) {
		ctx, user, ok := ps.sessionPerson(w, r, true)
		if !ok {
			return
		}
		body, _ := readBody(r)
		status, msg := a.factorDisable(ctx, user, field(body, "senha", "password"), field(body, "codigo", "code"))
		answer(w, r, user, status, msg, map[string]any{"dois_fatores": false, "message": "Dois fatores desligados."}, nil, "Dois fatores desligados.")
	})
	mux.HandleFunc("POST /dois-fatores/codigos", func(w http.ResponseWriter, r *http.Request) {
		ctx, user, ok := ps.sessionPerson(w, r, true)
		if !ok {
			return
		}
		body, _ := readBody(r)
		codes, status, msg := a.factorNewCodes(ctx, user, field(body, "senha", "password"), field(body, "codigo", "code"))
		answer(w, r, user, status, msg, map[string]any{"codigos_de_recuperacao": codes, "message": "Códigos de recuperação novos; os anteriores não valem mais."}, codes,
			"Códigos de recuperação novos; os anteriores não valem mais. Guarde-os: não serão mostrados de novo.")
	})

	mux.HandleFunc("GET /entrar/codigo", func(w http.ResponseWriter, r *http.Request) {
		v := ps.base(r, "Código de verificação")
		if c, err := r.Cookie(challengeCookie); err != nil || c.Value == "" {
			http.Redirect(w, r, "/entrar", http.StatusSeeOther)
			return
		}
		v.Body = htmlOf(codeLoginTpl, nil)
		ps.render(w, v, http.StatusOK)
	})
	mux.HandleFunc("POST /entrar/codigo", func(w http.ResponseWriter, r *http.Request) {
		ctx := &interp.Context{Request: r, Writer: w}
		body, err := readBody(r)
		if err != nil {
			a.failErr(w, r, err)
			return
		}
		challenge := field(body, "desafio")
		if c, err := r.Cookie(challengeCookie); challenge == "" && err == nil {
			challenge = c.Value
		}
		user, status, msg := a.signInWithCode(ctx, challenge, field(body, "codigo", "code"))
		if user == nil {
			if isForm(r) {
				if msg == msgChallengeGone {
					http.SetCookie(w, &http.Cookie{Name: challengeCookie, Value: "", Path: "/entrar", MaxAge: -1, HttpOnly: true})
					http.Redirect(w, r, "/entrar?erro=1", http.StatusSeeOther)
					return
				}
				v := ps.base(r, "Código de verificação")
				v.Error = msg
				v.Body = htmlOf(codeLoginTpl, nil)
				// the status keeps counting toward the limit of this address
				ps.render(w, v, status)
				return
			}
			a.fail(w, status, msg)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: challengeCookie, Value: "", Path: "/entrar", MaxAge: -1, HttpOnly: true})
		startSession(ctx, w, user)
		if isForm(r) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		a.json(w, http.StatusOK, serialize(a.app.Entities[a.app.LoginEntity], user), nil)
	})
}

var codeLoginTpl = tpl(`<form class="caixa" method="post" action="/entrar/codigo"><h3>Código de verificação</h3><p>Digite o código do aplicativo autenticador, ou um código de recuperação.</p><label>Código<input name="codigo" required autofocus autocomplete="one-time-code" inputmode="text"></label><button>Entrar</button><a href="/entrar">Voltar</a></form>`)

var factorTpl = template.Must(template.New("dois-fatores").Parse(`<div class="caixa"><h3>Dois fatores</h3>
{{if .Codes}}<p>Códigos de recuperação (cada um vale uma vez):</p><pre>{{range .Codes}}{{.}}
{{end}}</pre>{{end}}
{{if eq .State "ligado"}}<p>Os dois fatores estão <strong>ligados</strong>: entrar pede um código do aplicativo depois da senha.</p>
<form method="post" action="/dois-fatores/codigos"><input type="hidden" name="_csrf" value="{{.CSRF}}"><h4>Códigos de recuperação novos</h4><label>Senha<input type="password" name="senha" required autocomplete="current-password"></label><label>Código do aplicativo<input name="codigo" required autocomplete="one-time-code"></label><button>Gerar códigos novos</button></form>
<form method="post" action="/dois-fatores/desativar"><input type="hidden" name="_csrf" value="{{.CSRF}}"><h4>Desligar</h4><label>Senha<input type="password" name="senha" required autocomplete="current-password"></label><label>Código (do aplicativo ou de recuperação)<input name="codigo" required autocomplete="one-time-code"></label><button class="perigo">Desligar os dois fatores</button></form>
{{else if eq .State "pendente"}}{{with .Setup}}<p>Cadastre este segredo no aplicativo autenticador (ou abra o endereço abaixo no celular):</p><p><code>{{.segredo}}</code></p><p><code>{{.uri}}</code></p>{{end}}
<form method="post" action="/dois-fatores/confirmar"><input type="hidden" name="_csrf" value="{{.CSRF}}"><label>Código que o aplicativo mostra agora<input name="codigo" required autofocus autocomplete="one-time-code" inputmode="numeric"></label><button>Ligar</button></form>
{{else}}<p>Com os dois fatores, entrar pede, além da senha, um código de um aplicativo autenticador no celular.</p>
{{if .KeyReady}}<form method="post" action="/dois-fatores/ativar"><input type="hidden" name="_csrf" value="{{.CSRF}}"><label>Senha<input type="password" name="senha" required autocomplete="current-password"></label><button>Começar</button></form>{{else}}<p>Este servidor ainda não guarda segredos cifrados (falta GERMANIO_SEGREDO).</p>{{end}}{{end}}
</div>`))
