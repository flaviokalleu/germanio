package servidor

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/runtime/banco"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// E-mail confirmation (docs/gep/0031-confirmacao-de-email.md, em teste).
// Whoever signs up receives a link that confirms the address: a random
// token stored only as its SHA-256, valid for 24 hours and one use, tied to
// the address it was sent to. Until then the person cannot sign in with the
// password (a session, OAuth or Git), so nobody acts with an address they do
// not own. Changing the e-mail asks for a new confirmation. Asking for
// another link answers the same whether or not the account exists.

const (
	confirmTable     = "_germanio_confirmacao"
	confirmExpiry    = 24 * time.Hour
	confirmResendGap = 2 * time.Minute
)

const (
	msgConfirmFirst = "Confirme seu e-mail antes de entrar: abra o link que enviamos. Se ele não chegou, peça outro em /reenviar-confirmacao."
	msgConfirmSent  = "Conta criada. Enviamos um link para o seu e-mail: abra-o para confirmar o endereço e depois entre."
	msgConfirmAgain = "Se houver uma conta com esse login esperando confirmação, enviamos um link novo."
	msgConfirmBad   = "Este link de confirmação não vale mais. Peça outro em /reenviar-confirmacao."
)

var errConfirmUsed = errors.New("link de confirmação já usado")

// idColumn is the auto-increment primary key of an internal table.
func (a *intentAPI) idColumn() string {
	switch a.s.DB.Driver {
	case "postgres", "postgresql":
		return "SERIAL PRIMARY KEY"
	case "mysql":
		return "INTEGER PRIMARY KEY AUTO_INCREMENT"
	}
	return "INTEGER PRIMARY KEY AUTOINCREMENT"
}

// newToken is 32 random bytes, URL-safe.
func newToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// issueConfirmation saves a new link for user (in db, the transaction of the
// change when there is one) and sends it after the change is kept.
func (a *intentAPI) issueConfirmation(ctx *interp.Context, db *banco.Banco, user map[string]any) error {
	le := a.app.Entities[a.app.LoginEntity]
	field := emailField(le)
	to := strings.ToLower(strings.TrimSpace(toStr(user[field])))
	if to == "" {
		return nil
	}
	token, err := newToken()
	if err != nil {
		return err
	}
	expires := time.Now().UTC().Add(confirmExpiry).Format(time.RFC3339)
	if _, err := db.Executar(fmt.Sprintf(`INSERT INTO %s (pessoa_id, hash, email, expira_em) VALUES (%s, %s, %s, %s)`, confirmTable, a.s.ph(1), a.s.ph(2), a.s.ph(3), a.s.ph(4)), user["id"], hashToken(token), to, expires); err != nil {
		return err
	}
	send, why := mailerFromEnv()
	public := strings.TrimSuffix(os.Getenv("GERMANIO_URL_PUBLICA"), "/")
	if why != "" || public == "" {
		return nil // said at start; the person can ask again once it is configured
	}
	text := "Para confirmar este e-mail, abra o link abaixo em até 24 horas:\n\n" + public + "/confirmar-email?token=" + token +
		"\n\nSe não foi você que criou a conta, ignore esta mensagem."
	subject := "Confirme seu e-mail — " + a.s.Program.System.Name
	run := func() error { return send(to, subject, text) }
	if ctx != nil && ctx.Effects != nil {
		ctx.Effects.Add(interp.Effect{Kind: "e-mail de confirmação", Run: run})
		return nil
	}
	go func() {
		if err := run(); err != nil {
			fmt.Printf("[germanio] confirmação de e-mail: falha ao enviar o e-mail: %v\n", err)
		}
	}()
	return nil
}

// recentConfirmation: a link was sent to the person less than
// confirmResendGap ago.
func (a *intentAPI) recentConfirmation(pessoa any) bool {
	var n int
	since := time.Now().UTC().Add(confirmExpiry - confirmResendGap).Format(time.RFC3339)
	a.s.DB.DB.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE pessoa_id = %s AND expira_em > %s`, confirmTable, a.s.ph(1), a.s.ph(2)), pessoa, since).Scan(&n)
	return n > 0
}

// emailChange: the edit changes the e-mail of a person of an app that
// confirms e-mails; the new address must be confirmed again.
func (a *intentAPI) emailChange(e *ast.Entity, row, data map[string]any) bool {
	if a.app.Login == nil || !a.app.Login.Confirmation || e.Singular != a.app.LoginEntity {
		return false
	}
	field := emailField(e)
	v, ok := data[field]
	if !ok || field == "" {
		return false
	}
	return strings.ToLower(strings.TrimSpace(toStr(v))) != strings.ToLower(strings.TrimSpace(toStr(row[field])))
}

func (a *intentAPI) mountConfirmation(mux *routeMux) {
	le := a.app.Entities[a.app.LoginEntity]
	a.s.DB.DB.Exec(`CREATE TABLE IF NOT EXISTS ` + confirmTable + ` (id ` + a.idColumn() + `, pessoa_id INTEGER NOT NULL, hash TEXT NOT NULL UNIQUE, email TEXT NOT NULL, expira_em TEXT NOT NULL)`)
	_, why := mailerFromEnv()
	public := os.Getenv("GERMANIO_URL_PUBLICA")
	switch {
	case why != "":
		fmt.Printf("[germanio] confirmação de e-mail indisponível (o cadastro fica fechado): %s\n", why)
	case public == "":
		fmt.Println("[germanio] confirmação de e-mail indisponível (o cadastro fica fechado): defina GERMANIO_URL_PUBLICA (o endereço do link no e-mail)")
	}
	a.confirmAvailable = why == "" && public != ""
	field := emailField(le)
	isForm := func(r *http.Request) bool {
		return strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
	}

	mux.HandleFunc("POST /confirmar-email", func(w http.ResponseWriter, r *http.Request) {
		ctx := &interp.Context{Request: r, Writer: w}
		body, err := readBody(r)
		if err != nil {
			a.failErr(w, r, err)
			return
		}
		token := toStr(body["token"])
		fail := func(msg string) {
			if isForm(r) {
				http.Redirect(w, r, "/confirmar-email?erro="+urlQuery(msg), http.StatusSeeOther)
				return
			}
			a.fail(w, http.StatusBadRequest, msg)
		}
		var pessoa int64
		var email, expira string
		row := a.s.DB.DB.QueryRow(fmt.Sprintf(`SELECT pessoa_id, email, expira_em FROM %s WHERE hash = %s`, confirmTable, a.s.ph(1)), hashToken(token))
		if token == "" || row.Scan(&pessoa, &email, &expira) != nil {
			fail(msgConfirmBad)
			return
		}
		if t, err := time.Parse(time.RFC3339, expira); err != nil || time.Now().After(t) {
			a.s.DB.Executar(fmt.Sprintf(`DELETE FROM %s WHERE hash = %s`, confirmTable, a.s.ph(1)), hashToken(token))
			fail("Este link de confirmação expirou. Peça outro em /reenviar-confirmacao.")
			return
		}
		// one use, even when the same link is opened twice at once: claimed
		// in the transaction that confirms; a link sent to an address the
		// account no longer has confirms nothing
		ctx.Effects = &interp.Effects{}
		err = a.s.DB.EmTransacao(func(tx *banco.Banco) error {
			res, err := tx.Executar(fmt.Sprintf(`DELETE FROM %s WHERE hash = %s AND pessoa_id = %s`, confirmTable, a.s.ph(1), a.s.ph(2)), hashToken(token), pessoa)
			if err != nil {
				return err
			}
			if n, _ := res.RowsAffected(); n != 1 {
				return errConfirmUsed
			}
			ctx.DB = tx
			found, err := a.in.Op(ctx, le.Singular, "buscar", pessoa)
			person, _ := found.(map[string]any)
			if err != nil || person == nil || strings.ToLower(strings.TrimSpace(toStr(person[field]))) != email {
				return errConfirmUsed
			}
			if _, err := a.in.Op(ctx, le.Singular, "atualizar", pessoa, map[string]any{"email_confirmado": true}); err != nil {
				return err
			}
			_, err = tx.Executar(fmt.Sprintf(`DELETE FROM %s WHERE pessoa_id = %s`, confirmTable, a.s.ph(1)), pessoa)
			return err
		})
		if err == errConfirmUsed {
			fail(msgConfirmBad)
			return
		}
		if err != nil {
			fail(interp.Friendly(err))
			return
		}
		runEffects(ctx.Effects.Take())
		if isForm(r) {
			http.Redirect(w, r, "/entrar?confirmado=1", http.StatusSeeOther)
			return
		}
		a.json(w, http.StatusOK, map[string]any{"message": "E-mail confirmado. Agora você já pode entrar."}, nil)
	})

	mux.HandleFunc("POST /reenviar-confirmacao", func(w http.ResponseWriter, r *http.Request) {
		ctx := &interp.Context{Request: r, Writer: w}
		body, err := readBody(r)
		if err != nil {
			a.failErr(w, r, err)
			return
		}
		if !a.confirmAvailable {
			a.fail(w, http.StatusServiceUnavailable, "A confirmação de e-mail ainda não está disponível neste sistema")
			return
		}
		login := first(toStr(body["login"]), toStr(body["email"]))
		if user := a.s.findLogin(ctx, login); user != nil && a.s.unconfirmed(user) && !a.recentConfirmation(user["id"]) {
			// sent off the answer's path: its time does not tell whether the
			// account exists
			if err := a.issueConfirmation(nil, a.s.DB, user); err != nil {
				fmt.Printf("[germanio] confirmação de e-mail: %v\n", err)
			}
		}
		// the same answer whether or not the account exists (or is confirmed)
		if isForm(r) {
			http.Redirect(w, r, "/reenviar-confirmacao?enviado=1", http.StatusSeeOther)
			return
		}
		a.json(w, http.StatusAccepted, map[string]any{"message": msgConfirmAgain}, nil)
	})
}
