package servidor

import (
	"bytes"
	"context"
	"net/http"

	"github.com/flaviokalleu/germanio/runtime/banco"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// Every request that changes data through the intent surface runs in one
// database transaction: the record, its hooks (antes de/quando), memberships,
// numbering and queued events are kept together or not at all. The response
// is held until the commit, so a client never sees success for a change that
// was undone. Work outside the database (repositories on disk) registers an
// undo step that runs if the transaction is rolled back.

type txState struct {
	db         *banco.Banco
	onRollback []func()
	onCommit   []func()
}

type txKey struct{}

func txOf(r *http.Request) *txState {
	if r == nil {
		return nil
	}
	st, _ := r.Context().Value(txKey{}).(*txState)
	return st
}

// newContext builds the execution context of a request, joined to its
// transaction when there is one.
func newContext(w http.ResponseWriter, r *http.Request) *interp.Context {
	ctx := &interp.Context{Request: r, Writer: w}
	if st := txOf(r); st != nil {
		ctx.DB = st.db
	}
	return ctx
}

// afterCommit runs fn once the request's change is kept (immediately when
// there is no transaction). Irreversible work — deleting files — waits here.
func afterCommit(ctx *interp.Context, fn func()) {
	if ctx != nil {
		if st := txOf(ctx.Request); st != nil {
			st.onCommit = append(st.onCommit, fn)
			return
		}
	}
	fn()
}

// undoOnRollback registers fn to run if the request's transaction is undone.
func undoOnRollback(ctx *interp.Context, fn func()) {
	if ctx == nil {
		return
	}
	if st := txOf(ctx.Request); st != nil {
		st.onRollback = append(st.onRollback, fn)
	}
}

type heldResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (h *heldResponse) Header() http.Header { return h.header }
func (h *heldResponse) WriteHeader(code int) {
	if h.status == 0 {
		h.status = code
	}
}
func (h *heldResponse) Write(b []byte) (int, error) {
	if h.status == 0 {
		h.status = http.StatusOK
	}
	return h.body.Write(b)
}

// transactional runs serve inside a transaction and releases the response
// after the commit. Status ≥ 400 or a panic undoes everything.
func (s *Servidor) transactional(w http.ResponseWriter, r *http.Request, serve func(http.ResponseWriter, *http.Request)) {
	if s.DB == nil || txOf(r) != nil {
		serve(w, r)
		return
	}
	held := &heldResponse{header: http.Header{}}
	st := &txState{}
	errUndo := errRollback{}
	err := s.DB.EmTransacao(func(tx *banco.Banco) error {
		st.db = tx
		serve(held, r.WithContext(context.WithValue(r.Context(), txKey{}, st)))
		if held.status >= 400 {
			return errUndo
		}
		return nil
	})
	if err != nil {
		for i := len(st.onRollback) - 1; i >= 0; i-- {
			st.onRollback[i]()
		}
		if err != errUndo {
			http.Error(w, `{"message":"não foi possível salvar"}`, http.StatusInternalServerError)
			return
		}
	} else {
		for _, fn := range st.onCommit {
			fn()
		}
	}
	for k, v := range held.header {
		w.Header()[k] = v
	}
	if held.status == 0 {
		held.status = http.StatusOK
	}
	w.WriteHeader(held.status)
	w.Write(held.body.Bytes())
}

type errRollback struct{}

func (errRollback) Error() string { return "rollback" }
