package servidor

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flaviokalleu/germanio/compiler/ast"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// Password recovery (docs/gep/0008-recuperacao-de-senha.md): a random
// token, stored only as its SHA-256, valid for one hour and one use; the
// answer never reveals which accounts exist; the link uses the declared
// public address (never the request's Host, which an attacker controls).

const (
	recoveryTable  = "_germanio_recuperacao"
	recoveryExpiry = time.Hour
)

// mailer sends one message. Configured outside the source code:
// GERMANIO_CORREIO_PASTA writes messages as files (development), otherwise
// GERMANIO_SMTP_HOST/PORTA/USUARIO/SENHA/REMETENTE.
type mailer func(to, subject, body string) error

func mailerFromEnv() (mailer, string) {
	if dir := os.Getenv("GERMANIO_CORREIO_PASTA"); dir != "" {
		return func(to, subject, body string) error {
			if err := os.MkdirAll(dir, 0o700); err != nil {
				return err
			}
			name := fmt.Sprintf("%d-%s.txt", time.Now().UnixNano(), strings.NewReplacer("@", "_", "/", "_").Replace(to))
			msg := "Para: " + to + "\nAssunto: " + subject + "\n\n" + body
			return os.WriteFile(filepath.Join(dir, name), []byte(msg), 0o600)
		}, ""
	}
	host := os.Getenv("GERMANIO_SMTP_HOST")
	if host == "" {
		return nil, "defina GERMANIO_SMTP_HOST (e PORTA, USUARIO, SENHA, REMETENTE) ou GERMANIO_CORREIO_PASTA"
	}
	port := os.Getenv("GERMANIO_SMTP_PORTA")
	if port == "" {
		port = "587"
	}
	user, pass, from := os.Getenv("GERMANIO_SMTP_USUARIO"), os.Getenv("GERMANIO_SMTP_SENHA"), os.Getenv("GERMANIO_SMTP_REMETENTE")
	if from == "" {
		from = user
	}
	return func(to, subject, body string) error {
		msg := "From: " + from + "\r\nTo: " + to + "\r\nSubject: " + subject + "\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" + body
		var auth smtp.Auth
		if user != "" {
			auth = smtp.PlainAuth("", user, pass, host)
		}
		return smtp.SendMail(host+":"+port, auth, from, []string{to}, []byte(msg))
	}, ""
}

// recoveryResendGap: at most one recovery e-mail per account in this time,
// however many requests arrive (from however many addresses): repeated
// requests cannot flood someone's mailbox. The answer stays the same.
const recoveryResendGap = 2 * time.Minute

// recentRecovery reports whether a link was sent to the person less than
// recoveryResendGap ago (a token is created with expiry now + 1 hour).
func (a *intentAPI) recentRecovery(pessoa any) bool {
	var n int
	since := time.Now().UTC().Add(recoveryExpiry - recoveryResendGap).Format(time.RFC3339)
	a.s.DB.DB.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE pessoa_id = %s AND expira_em > %s`, recoveryTable, a.s.ph(1), a.s.ph(2)), pessoa, since).Scan(&n)
	return n > 0
}

func hashToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}

// emailField is the e-mail field of the people who log in.
func emailField(le *ast.Entity) string {
	for _, f := range le.Model.Fields {
		if f.Type == ast.FieldEmail {
			return strings.ToLower(f.Name)
		}
	}
	return ""
}

func (a *intentAPI) mountRecovery(mux *routeMux) {
	app := a.app
	le := app.Entities[app.LoginEntity]
	id := "INTEGER PRIMARY KEY AUTOINCREMENT"
	switch a.s.DB.Driver {
	case "postgres", "postgresql":
		id = "SERIAL PRIMARY KEY"
	case "mysql":
		id = "INTEGER PRIMARY KEY AUTO_INCREMENT"
	}
	a.s.DB.DB.Exec(`CREATE TABLE IF NOT EXISTS ` + recoveryTable + ` (id ` + id + `, pessoa_id INTEGER NOT NULL, hash TEXT NOT NULL UNIQUE, expira_em TEXT NOT NULL)`)
	send, why := mailerFromEnv()
	public := strings.TrimSuffix(os.Getenv("GERMANIO_URL_PUBLICA"), "/")
	field := emailField(le)
	switch {
	case why != "":
		fmt.Printf("[germanio] recuperação de senha indisponível: %s\n", why)
	case public == "":
		fmt.Println("[germanio] recuperação de senha indisponível: defina GERMANIO_URL_PUBLICA (o endereço do link no e-mail)")
	case field == "":
		fmt.Printf("[germanio] recuperação de senha indisponível: %s não tem um campo de e-mail\n", le.Plural)
	}
	available := why == "" && public != "" && field != ""
	a.recoveryAvailable = available
	isForm := func(r *http.Request) bool {
		return strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
	}

	mux.HandleFunc("POST /esqueci", func(w http.ResponseWriter, r *http.Request) {
		ctx := &interp.Context{Request: r, Writer: w}
		body, err := readBody(r)
		if err != nil {
			a.failErr(w, r, err)
			return
		}
		if !available {
			a.fail(w, http.StatusServiceUnavailable, "A recuperação de senha ainda não está disponível neste sistema")
			return
		}
		login := first(toStr(body["login"]), toStr(body["email"]))
		if user := a.s.findLogin(ctx, login); user != nil && toStr(user[field]) != "" && !a.recentRecovery(user["id"]) {
			raw := make([]byte, 32)
			if _, err := rand.Read(raw); err != nil {
				a.failErr(w, r, err)
				return
			}
			token := base64.RawURLEncoding.EncodeToString(raw)
			expires := time.Now().UTC().Add(recoveryExpiry).Format(time.RFC3339)
			if _, err := a.s.DB.Executar(fmt.Sprintf(`INSERT INTO %s (pessoa_id, hash, expira_em) VALUES (%s, %s, %s)`, recoveryTable, a.s.ph(1), a.s.ph(2), a.s.ph(3)), user["id"], hashToken(token), expires); err != nil {
				a.failErr(w, r, err)
				return
			}
			text := "Para criar uma senha nova, abra o link abaixo em até 1 hora:\n\n" + public + "/redefinir?token=" + token +
				"\n\nSe não foi você que pediu, ignore esta mensagem: sua senha continua a mesma."
			if err := send(toStr(user[field]), "Recuperação de senha — "+a.s.Program.System.Name, text); err != nil {
				fmt.Printf("[germanio] recuperação de senha: falha ao enviar o e-mail: %v\n", err)
			}
		}
		// the same answer whether or not the account exists
		if isForm(r) {
			http.Redirect(w, r, "/esqueci?enviado=1", http.StatusSeeOther)
			return
		}
		a.json(w, http.StatusAccepted, map[string]any{"message": "Se houver uma conta com esse login, enviamos um link para criar uma senha nova."}, nil)
	})

	mux.HandleFunc("POST /redefinir", func(w http.ResponseWriter, r *http.Request) {
		ctx := &interp.Context{Request: r, Writer: w}
		body, err := readBody(r)
		if err != nil {
			a.failErr(w, r, err)
			return
		}
		token := toStr(body["token"])
		fail := func(msg string) {
			if isForm(r) {
				http.Redirect(w, r, "/redefinir?token="+urlQuery(token)+"&erro="+urlQuery(msg), http.StatusSeeOther)
				return
			}
			a.fail(w, http.StatusBadRequest, msg)
		}
		var pessoa int64
		var expira string
		row := a.s.DB.DB.QueryRow(fmt.Sprintf(`SELECT pessoa_id, expira_em FROM %s WHERE hash = %s`, recoveryTable, a.s.ph(1)), hashToken(token))
		if token == "" || row.Scan(&pessoa, &expira) != nil {
			fail("Este link não vale mais. Peça um novo em Esqueci minha senha.")
			return
		}
		if t, err := time.Parse(time.RFC3339, expira); err != nil || time.Now().After(t) {
			a.s.DB.Executar(fmt.Sprintf(`DELETE FROM %s WHERE hash = %s`, recoveryTable, a.s.ph(1)), hashToken(token))
			fail("Este link expirou. Peça um novo em Esqueci minha senha.")
			return
		}
		pass := first(toStr(body["senha"]), toStr(body["password"]))
		change := map[string]any{}
		for _, f := range le.Model.Fields {
			switch {
			case f.Type == ast.FieldSenha:
				change[strings.ToLower(f.Name)] = pass
			case strings.EqualFold(f.Name, "tentativas_falhas"):
				change["tentativas_falhas"] = 0
			case strings.EqualFold(f.Name, "bloqueado_ate"):
				change["bloqueado_ate"] = nil
			}
		}
		if _, err := a.in.Op(ctx, le.Singular, "atualizar", pessoa, change); err != nil {
			fail(interp.Friendly(err))
			return
		}
		// one use: every recovery link of this person stops working
		a.s.DB.Executar(fmt.Sprintf(`DELETE FROM %s WHERE pessoa_id = %s`, recoveryTable, a.s.ph(1)), pessoa)
		if isForm(r) {
			http.Redirect(w, r, "/entrar?redefinida=1", http.StatusSeeOther)
			return
		}
		a.json(w, http.StatusOK, map[string]any{"message": "Senha nova criada. Entre com ela."}, nil)
	})
}
