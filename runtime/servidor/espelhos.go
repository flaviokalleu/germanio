package servidor

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/runtime/git"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// Mirrors of a repository (docs/gep/0036-espelhos.md, em teste):
// `espelhos espelham o repositório do projeto`. Each record is an external
// repository. sentido "enviar": after every change of the code the whole
// repository (branches and tags) is sent to its url; sentido "receber":
// every GERMANIO_ESPELHO_MINUTOS the repository becomes a copy of the url.
// The work runs as background tasks after the change is kept — a failing
// mirror never breaks a push — with retries; each attempt is recorded on
// the record (situacao, ultima_atualizacao, ultimo_sucesso, ultimo_erro).
// Credentials written in the url are moved to a hidden field; changing the
// address forgets them, so they never follow to another server.

const defaultMirrorMinutes = 30

func mirrorInterval() time.Duration {
	if n, err := strconv.Atoi(os.Getenv("GERMANIO_ESPELHO_MINUTOS")); err == nil && n > 0 {
		return time.Duration(n) * time.Minute
	}
	return defaultMirrorMinutes * time.Minute
}

// mirrorInput validates the url of a mirror being created (before == nil)
// or edited, takes the credentials out of it and schedules an update.
func (a *intentAPI) mirrorInput(e *ast.Entity, data, before map[string]any) error {
	if e.Mirror == nil {
		return nil
	}
	raw, given := data["url"]
	if given {
		clean, user, pass, err := git.ParseRemoteURL(toStr(raw))
		if err != nil {
			return &interp.RuntimeError{Status: 400, Message: err.Error(), Payload: map[string]any{"url": []any{err.Error()}}}
		}
		data["url"] = clean
		switch {
		case user != "" || pass != "":
			data["credencial"] = url.UserPassword(user, pass).String()
		case before == nil || toStr(before["url"]) != clean:
			data["credencial"] = "" // a new address never inherits the old credentials
		}
		if before != nil && toStr(before["url"]) != clean {
			data["ultimo_erro"] = ""
		}
	}
	on := true
	if v, ok := data["habilitado"]; ok {
		on = truthy(v)
	} else if before != nil {
		on = truthy(before["habilitado"])
	}
	if on {
		data["situacao"] = "agendada"
	}
	return nil
}

// mirrorSaved queues the update of a mirror just created or edited (part
// of the request's transaction: it runs only if the change is kept).
func (a *intentAPI) mirrorSaved(ctx *interp.Context, e *ast.Entity, row map[string]any) {
	if e.Mirror == nil || a.s.Git == nil || toStr(row["situacao"]) != "agendada" {
		return
	}
	a.s.tasks().enqueue(ctx, "espelho", map[string]any{"entidade": e.Singular, "id": row["id"]})
}

// codeChanged queues the push mirrors of a record whose code just changed.
// A mirror already waiting for its turn is left alone: when it runs it
// sends the repository as it is then.
func (a *intentAPI) codeChanged(ctx *interp.Context, owner *ast.Entity, row map[string]any) {
	if a.s.Git == nil {
		return
	}
	for _, n := range a.app.Order {
		m := a.app.Entities[n]
		if m.Mirror == nil || m.Mirror.Owner != owner.Singular {
			continue
		}
		res, err := a.in.Op(ctx, m.Singular, "filtrar", map[string]any{m.Mirror.OwnerField: row["id"], "sentido": "enviar", "habilitado": true}, map[string]any{"limite": 100})
		if err != nil {
			continue
		}
		for _, it := range res.([]any) {
			rec := it.(map[string]any)
			if toStr(rec["situacao"]) == "agendada" {
				continue
			}
			a.in.Op(ctx, m.Singular, "atualizar", rec["id"], map[string]any{"situacao": "agendada"})
			a.s.tasks().enqueue(ctx, "espelho", map[string]any{"entidade": m.Singular, "id": rec["id"]})
		}
	}
}

// runMirror performs one update of a mirror (a background task). An error
// makes the queue try again later; every attempt is recorded.
func (a *intentAPI) runMirror(d map[string]any) error {
	ctx := &interp.Context{}
	m := a.app.Entities[toStr(d["entidade"])]
	if m == nil || m.Mirror == nil || a.s.Git == nil {
		return nil
	}
	res, _ := a.in.Op(ctx, m.Singular, "buscar", d["id"])
	rec, _ := res.(map[string]any)
	if rec == nil {
		return nil // removed meanwhile
	}
	if !truthy(rec["habilitado"]) {
		a.in.Op(ctx, m.Singular, "atualizar", rec["id"], map[string]any{"situacao": "nova"})
		return nil
	}
	oe := a.app.Entities[m.Mirror.Owner]
	ores, _ := a.in.Op(ctx, oe.Singular, "buscar", rec[m.Mirror.OwnerField])
	owner, _ := ores.(map[string]any)
	if owner == nil {
		return nil
	}
	repo := toStr(owner["repositorio"])
	a.in.Op(ctx, m.Singular, "atualizar", rec["id"], map[string]any{"situacao": "atualizando", "ultima_atualizacao": now()})
	remote := git.Remote{URL: toStr(rec["url"])}
	if c := toStr(rec["credencial"]); c != "" {
		if u, err := url.Parse("http://" + c + "@x"); err == nil && u.User != nil {
			remote.User = u.User.Username()
			remote.Password, _ = u.User.Password()
		}
	}
	// a url written by level-3 code may still carry credentials
	if clean, user, pass, err := git.ParseRemoteURL(remote.URL); err == nil && (user != "" || pass != "") {
		remote.URL, remote.User, remote.Password = clean, user, pass
	}
	run, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	var err error
	if toStr(rec["sentido"]) == "receber" {
		var updates []git.RefUpdate
		updates, err = a.s.Git.FetchMirror(run, repo, remote)
		if err == nil && len(updates) > 0 {
			a.codeChanged(ctx, oe, owner) // the new code goes on to the push mirrors
		}
	} else {
		err = a.s.Git.PushMirror(run, repo, remote)
	}
	if err != nil {
		a.in.Op(ctx, m.Singular, "atualizar", rec["id"], map[string]any{"situacao": "falhou", "ultimo_erro": err.Error()})
		return fmt.Errorf("espelho %v: %w", rec["id"], err)
	}
	a.in.Op(ctx, m.Singular, "atualizar", rec["id"], map[string]any{"situacao": "atualizada", "ultimo_sucesso": now(), "ultimo_erro": ""})
	return nil
}

// startMirrors registers the task and, when some data declares mirrors,
// schedules the receiving ones every GERMANIO_ESPELHO_MINUTOS.
func (a *intentAPI) startMirrors() {
	var mirrors []*ast.Entity
	for _, n := range a.app.Order {
		if a.app.Entities[n].Mirror != nil {
			mirrors = append(mirrors, a.app.Entities[n])
		}
	}
	if len(mirrors) == 0 || a.s.Git == nil {
		return
	}
	a.s.tasks().handle("espelho", a.runMirror)
	every := mirrorInterval()
	stop := make(chan struct{})
	a.s.onClose = append(a.s.onClose, func() { close(stop) })
	go func() {
		t := time.NewTicker(min(every, time.Minute))
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case c := <-t.C:
				for _, m := range mirrors {
					a.scheduleReceiving(m, c, every)
				}
			}
		}
	}()
}

// scheduleReceiving queues the receiving mirrors whose last update is older
// than the interval, a page at a time (never the whole table in memory).
func (a *intentAPI) scheduleReceiving(m *ast.Entity, at time.Time, every time.Duration) {
	ctx := &interp.Context{}
	due := at.UTC().Add(-every).Format(time.RFC3339)
	for page := 1; page <= 100; page++ {
		res, err := a.in.Op(ctx, m.Singular, "paginar", map[string]any{"sentido": "receber", "habilitado": true}, map[string]any{"limite": 100, "pagina": page})
		if err != nil {
			return
		}
		items, _ := res.(map[string]any)["itens"].([]any)
		for _, it := range items {
			rec := it.(map[string]any)
			st, last := toStr(rec["situacao"]), toStr(rec["ultima_atualizacao"])
			if st == "agendada" || (st == "atualizando" && last > due) || (last != "" && last > due) {
				continue
			}
			a.in.Op(ctx, m.Singular, "atualizar", rec["id"], map[string]any{"situacao": "agendada"})
			a.s.tasks().enqueue(ctx, "espelho", map[string]any{"entidade": m.Singular, "id": rec["id"]})
		}
		if len(items) < 100 {
			return
		}
	}
}
