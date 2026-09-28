package servidor

import (
	"encoding/json"
	"fmt"
	"github.com/flaviokalleu/germanio/runtime/httpclient"
	"net/http"
	"strings"
	"sync"
	"time"

	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// Persistent background tasks: work that must happen reliably after a
// request (webhook deliveries, application functions queued with
// tarefas.enfileirar). Tasks survive restarts, retry with backoff and are
// marked dead after the last attempt, keeping the error for inspection.

const tasksTable = "_germanio_tarefas"

var backoff = []time.Duration{time.Second, 5 * time.Second, 30 * time.Second, 2 * time.Minute, 10 * time.Minute}

type taskHandler func(dados map[string]any) error

type taskQueue struct {
	s        *Servidor
	mu       sync.Mutex
	handlers map[string]taskHandler
	stop     chan struct{}
	wake     chan struct{}
}

func (s *Servidor) tasks() *taskQueue {
	if s.queue != nil {
		return s.queue
	}
	q := &taskQueue{s: s, handlers: map[string]taskHandler{}, stop: make(chan struct{}), wake: make(chan struct{}, 1)}
	id := "INTEGER PRIMARY KEY AUTOINCREMENT"
	switch s.DB.Driver {
	case "postgres", "postgresql":
		id = "SERIAL PRIMARY KEY"
	case "mysql":
		id = "INTEGER PRIMARY KEY AUTO_INCREMENT"
	}
	_, err := s.DB.DB.Exec(`CREATE TABLE IF NOT EXISTS ` + tasksTable + ` (id ` + id + `, tipo TEXT NOT NULL, dados TEXT NOT NULL,
		estado TEXT NOT NULL, tentativas INTEGER NOT NULL, proxima_em TEXT NOT NULL, erro TEXT, criado_em TEXT NOT NULL)`)
	if err != nil {
		fmt.Printf("[germanio] fila de tarefas indisponível: %v\n", err)
	}
	s.queue = q
	s.onClose = append(s.onClose, func() { close(q.stop) })
	go q.loop()
	return q
}

func (q *taskQueue) handle(tipo string, h taskHandler) {
	q.mu.Lock()
	q.handlers[tipo] = h
	q.mu.Unlock()
}

func (q *taskQueue) ph(n int) string {
	if q.s.DB.Driver == "postgres" || q.s.DB.Driver == "postgresql" {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

// enqueue stores a task to run as soon as possible.
// enqueue stores a task. Inside a transaction the task is part of it: it
// runs only if the change that produced it is kept.
func (q *taskQueue) enqueue(ctx *interp.Context, tipo string, dados map[string]any) error {
	b, err := json.Marshal(dados)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	db := q.s.DB
	if ctx != nil && ctx.DB != nil {
		db = ctx.DB
	}
	_, err = db.Executar(fmt.Sprintf(`INSERT INTO %s (tipo, dados, estado, tentativas, proxima_em, criado_em) VALUES (%s, %s, 'pendente', 0, %s, %s)`,
		tasksTable, q.ph(1), q.ph(2), q.ph(3), q.ph(4)), tipo, string(b), now, now)
	select {
	case q.wake <- struct{}{}:
	default:
	}
	return err
}

func (q *taskQueue) loop() {
	tick := time.NewTicker(250 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-q.stop:
			return
		case <-tick.C:
		case <-q.wake:
		}
		for q.runOne() {
		}
	}
}

// runOne claims the next due task (atomic update) and runs it.
func (q *taskQueue) runOne() bool {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	var id int64
	var tipo, dados string
	var tries int
	row := q.s.DB.DB.QueryRow(fmt.Sprintf(`SELECT id, tipo, dados, tentativas FROM %s WHERE estado = 'pendente' AND proxima_em <= %s ORDER BY id LIMIT 1`, tasksTable, q.ph(1)), now)
	if err := row.Scan(&id, &tipo, &dados, &tries); err != nil {
		return false
	}
	res, err := q.s.DB.DB.Exec(fmt.Sprintf(`UPDATE %s SET estado = 'executando' WHERE id = %s AND estado = 'pendente'`, tasksTable, q.ph(1)), id)
	if err != nil {
		return false
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return true
	}
	q.mu.Lock()
	h := q.handlers[tipo]
	q.mu.Unlock()
	var payload map[string]any
	json.Unmarshal([]byte(dados), &payload)
	if h == nil {
		err = fmt.Errorf("tipo de tarefa desconhecido: %s", tipo)
	} else {
		err = safeRun(h, payload)
	}
	if err == nil {
		q.s.DB.DB.Exec(fmt.Sprintf(`UPDATE %s SET estado = 'concluida', tentativas = %s, erro = NULL WHERE id = %s`, tasksTable, q.ph(1), q.ph(2)), tries+1, id)
		return true
	}
	tries++
	if tries > len(backoff) {
		q.s.DB.DB.Exec(fmt.Sprintf(`UPDATE %s SET estado = 'morta', tentativas = %s, erro = %s WHERE id = %s`, tasksTable, q.ph(1), q.ph(2), q.ph(3)), tries, err.Error(), id)
		fmt.Printf("[germanio] tarefa %d (%s) desistiu após %d tentativas: %v\n", id, tipo, tries, err)
		return true
	}
	next := time.Now().UTC().Add(backoff[tries-1]).Format(time.RFC3339Nano)
	q.s.DB.DB.Exec(fmt.Sprintf(`UPDATE %s SET estado = 'pendente', tentativas = %s, proxima_em = %s, erro = %s WHERE id = %s`, tasksTable, q.ph(1), q.ph(2), q.ph(3), q.ph(4)), tries, next, err.Error(), id)
	return true
}

func safeRun(h taskHandler, payload map[string]any) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("falha: %v", r)
		}
	}()
	return h(payload)
}

// registerTaskModule exposes `tarefas.enfileirar("funcao", dados)` to .ge:
// the function runs in the background, with retries, outside the request.
func (s *Servidor) registerTaskModule() {
	q := s.tasks()
	q.handle("funcao", func(d map[string]any) error {
		name, _ := d["funcao"].(string)
		_, err := s.Interpreter.RunFunction(name, []any{d["dados"]}, &interp.Context{})
		return err
	})
	s.Interpreter.RegisterModule("tarefas", map[string]interp.ModuleFunc{
		"enfileirar": func(c *interp.Call, args []any) any {
			name := c.Str(args, 0, "função")
			if _, ok := s.Interpreter.Functions[name]; !ok {
				panic(c.Fail(0, "tarefas.enfileirar: a função %q não existe", name))
			}
			var dados any
			if len(args) > 1 {
				dados = args[1]
			}
			if err := q.enqueue(c.Ctx(), "funcao", map[string]any{"funcao": name, "dados": dados}); err != nil {
				panic(c.Fail(0, "tarefas.enfileirar: %v", err))
			}
			return true
		},
	})
}

// ---------- safe outbound HTTP ----------

var errLocalNetwork = httpclient.ErrRedeLocal

// safeHTTPClient refuses the server's own networks (httpclient.Transport)
// and does not follow redirects.
func safeHTTPClient() *http.Client {
	return &http.Client{Transport: httpclient.Transport(), Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func isLocalURL(u string) bool {
	return strings.HasPrefix(u, "http://localhost") || strings.HasPrefix(u, "http://127.")
}
