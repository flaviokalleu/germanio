package servidor

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/runtime/git"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// Git over SSH, by configuration (no .ge phrase): when the host sets
// GERMANIO_SSH_ENDERECO and the program keeps people's public keys (a data
// with a `chave pública` field that belongs to the people who sign in),
// people clone and push with their key. The rules are the ones of smart
// HTTP, through the same functions: baixar/enviar código, read-only
// records, protected branches, `antes de enviar código`, and after a push
// the event, the history, the executions and `quando enviar código`.

// IniciarSSH starts the SSH server when GERMANIO_SSH_ENDERECO is set and
// returns the address it listens on ("" when SSH is not configured).
func (s *Servidor) IniciarSSH() (string, error) {
	addr := strings.TrimSpace(os.Getenv("GERMANIO_SSH_ENDERECO"))
	if addr == "" {
		return "", nil
	}
	a := s.intent
	if a == nil || s.Git == nil || len(a.repositoryEntities()) == 0 {
		return "", fmt.Errorf("GERMANIO_SSH_ENDERECO está definido, mas este programa não tem repositórios.\n" +
			"Por quê: o SSH só serve para clonar e enviar código de dados com `tem repositório`.\n" +
			"Como corrigir: declare um repositório (por exemplo `projetos` › `tem` › `repositório`) ou remova GERMANIO_SSH_ENDERECO")
	}
	keys, owner := a.publicKeyData()
	if keys == nil {
		return "", fmt.Errorf("GERMANIO_SSH_ENDERECO está definido, mas este programa não guarda chaves públicas das pessoas.\n" +
			"Por quê: no SSH cada pessoa é reconhecida pela chave que cadastrou.\n" +
			"Como corrigir: declare um dado com um campo `chave pública` que pertença às pessoas, por exemplo\n" +
			"    chaves\n        tem\n            titulo obrigatório\n            conteudo chave pública obrigatório e único\n        pertence a\n            usuario")
	}
	keyPath := strings.TrimSpace(os.Getenv("GERMANIO_SSH_CHAVE_HOST"))
	if keyPath == "" {
		keyPath = filepath.Join(s.Git.Root, ".ssh", "chave_host_ed25519")
	}
	hostKey, err := git.LoadOrCreateHostKey(keyPath)
	if err != nil {
		return "", err
	}
	maxConns := 64
	if v, err := strconv.Atoi(os.Getenv("GERMANIO_SSH_CONEXOES")); err == nil && v > 0 {
		maxConns = v
	}
	entities := a.repositoryEntities()
	srv := &git.SSHServer{
		Store:    s.Git,
		MaxConns: maxConns,
		Identify: func(key ssh.PublicKey) string { return a.keyOwner(keys, owner, key) },
		Authorize: func(ctx context.Context, who, remote, service, address string) (*git.SSHAccess, error) {
			return a.authorizeSSH(ctx, entities, who, remote, service, address)
		},
		Greet: func(who string) string {
			ictx := &interp.Context{Values: map[string]any{}}
			if p := s.loadPerson(ictx, who); p != nil {
				return "Olá, " + first(toStr(p["nome"]), toStr(p["username"]), toStr(p["email"])) + "! Sua chave foi reconhecida."
			}
			return "Sua chave foi reconhecida."
		},
	}
	bound, err := srv.Listen(addr, hostKey)
	if err != nil {
		return "", fmt.Errorf("não foi possível abrir o SSH em %s: %w", addr, err)
	}
	s.onClose = append(s.onClose, func() { srv.Close() })
	fmt.Printf("[germanio] Git por SSH em %s\n", bound)
	return bound.String(), nil
}

// publicKeyData finds the data holding people's public keys: a `chave
// pública` field and a relation to the people who sign in.
func (a *intentAPI) publicKeyData() (*ast.Entity, string) {
	if a.app.LoginEntity == "" {
		return nil, ""
	}
	for _, n := range a.app.Order {
		e := a.app.Entities[n]
		if e.Model == nil {
			continue
		}
		hasKey := false
		for _, f := range e.Model.Fields {
			hasKey = hasKey || f.Type == ast.FieldChavePublica
		}
		if !hasKey {
			continue
		}
		for field, target := range e.Parents {
			if target == a.app.LoginEntity {
				return e, field
			}
		}
	}
	return nil, ""
}

// keyOwner returns the id of the person who registered key, "" when nobody
// did. The key is found by its fingerprint and compared whole.
func (a *intentAPI) keyOwner(keys *ast.Entity, owner string, key ssh.PublicKey) string {
	ctx := &interp.Context{Values: map[string]any{}}
	res, err := a.in.Op(ctx, keys.Singular, "encontrar", map[string]any{"impressao_digital": ssh.FingerprintSHA256(key)})
	row, ok := res.(map[string]any)
	if err != nil || !ok {
		return ""
	}
	for _, f := range keys.Model.Fields {
		if f.Type != ast.FieldChavePublica {
			continue
		}
		stored, _, _, _, err := ssh.ParseAuthorizedKey([]byte(toStr(row[strings.ToLower(f.Name)])))
		if err != nil || !bytes.Equal(stored.Marshal(), key.Marshal()) {
			return ""
		}
		if row[owner] == nil {
			return ""
		}
		return toStr(row[owner])
	}
	return ""
}

// authorizeSSH applies to one git command over SSH the rules smart HTTP
// applies, and says why in Portuguese when it refuses.
func (a *intentAPI) authorizeSSH(ctx context.Context, entities []*ast.Entity, who, remote, service, address string) (*git.SSHAccess, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "/"+address+".git/git-"+service, nil)
	if err != nil {
		return nil, err
	}
	r.RemoteAddr = remote
	ictx := &interp.Context{Request: r, Writer: discardResponse{}, Values: map[string]any{}}
	atual := a.s.loadPerson(ictx, who)
	switch {
	case atual == nil:
		return nil, fmt.Errorf("esta chave não pertence mais a ninguém; cadastre-a de novo no seu perfil.")
	case !a.s.active(atual):
		return nil, fmt.Errorf("sua conta não está ativa (por exemplo, foi bloqueada), por isso o acesso pelo git foi recusado. Fale com quem administra o sistema.")
	case a.s.unconfirmed(atual):
		return nil, fmt.Errorf("confirme o seu e-mail antes de usar o git; o link de confirmação foi enviado quando você se cadastrou.")
	}
	// the person is already known by the key: identify returns them
	ictx.Values["atual"] = atual
	ictx.Values["token"] = "ssh"
	var refusal error
	grant, ok := a.codeAccess(ictx, r, entities, address, service == "receive-pack", false, func(status int, msg string) {
		switch {
		case status == http.StatusUnauthorized || status == http.StatusNotFound:
			refusal = fmt.Errorf("o repositório %q não foi encontrado, ou você não tem acesso a ele. Confira o endereço (por exemplo ssh://servidor/grupo/projeto.git).", address)
		case msg == codeDenied("enviar_codigo"):
			refusal = fmt.Errorf("você pode ver o repositório %q, mas não pode enviar código para ele. Peça a quem administra um papel que permita enviar código.", address)
		case msg == codeDenied("baixar_codigo"):
			refusal = fmt.Errorf("você pode ver o repositório %q, mas não pode baixar o código dele. Peça a quem administra um papel que permita baixar código.", address)
		default: // a read-only record: the rule's own friendly message
			refusal = fmt.Errorf("%s", msg)
		}
	})
	if !ok {
		return nil, refusal
	}
	e, row := grant.e, grant.row
	repo, _ := row["repositorio"].(string)
	acc := &git.SSHAccess{Repo: repo}
	if service == "receive-pack" {
		acc.Check = a.pushRules(ictx, atual, e, row)
		acc.After = func(applied []git.RefUpdate) { a.afterPush(ictx, atual, e, row, applied) }
	}
	return acc, nil
}

// discardResponse stands for the HTTP answer of hooks run for an SSH
// command: what they would write goes nowhere.
type discardResponse struct{}

func (discardResponse) Header() http.Header         { return http.Header{} }
func (discardResponse) Write(b []byte) (int, error) { return len(b), nil }
func (discardResponse) WriteHeader(int)             {}
