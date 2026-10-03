package servidor

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/flaviokalleu/germanio/compiler/ast"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// Pages stay up to date (docs/gep/0020-paginas-vivas.md, em teste). Every
// record written through the interpreter is announced after its commit to the
// viewers whose page shows that data and who may see the record. A notice
// carries no data: the page asks for itself again with the viewer's session,
// so everything still goes through the ordinary rules. Notices coalesce in a
// one-slot channel: a slow viewer never holds memory nor slows the others.

type change struct {
	model         string
	before, after map[string]any
}

type watcher struct {
	person  any             // the viewer's id (nil: not signed in)
	depends map[string]bool // models the page shows
	signal  chan struct{}   // one slot: many changes, one refresh
}

type liveHub struct {
	a        *intentAPI
	mu       sync.RWMutex
	watchers map[*watcher]bool
}

func newLiveHub(a *intentAPI) *liveHub {
	h := &liveHub{a: a, watchers: map[*watcher]bool{}}
	a.in.OnChange = h.onChange
	return h
}

// onChange records a write; it is announced once the change is kept.
func (h *liveHub) onChange(ctx *interp.Context, model string, before, after map[string]any) {
	c := change{model: strings.ToLower(model), before: before, after: after}
	if ctx != nil {
		if st := txOf(ctx.Request); st != nil {
			st.onCommit = append(st.onCommit, func() { h.publish(c) })
			return
		}
	}
	h.publish(c) // outside a request transaction the write is already kept
}

func (h *liveHub) publish(c change) {
	h.mu.RLock()
	var interested []*watcher
	for w := range h.watchers {
		if w.depends[c.model] {
			interested = append(interested, w)
		}
	}
	h.mu.RUnlock()
	if len(interested) == 0 {
		return
	}
	e := h.a.app.Entities[c.model]
	ctx := &interp.Context{}
	for _, w := range interested {
		if e != nil && !h.sees(ctx, w, e, c) {
			continue // a record this viewer cannot see changes silently for them
		}
		select {
		case w.signal <- struct{}{}:
		default: // a refresh is already pending
		}
	}
}

// sees: the viewer may see the record before or after the change, by the
// rules as they are now.
func (h *liveHub) sees(ctx *interp.Context, w *watcher, e *ast.Entity, c change) bool {
	var person map[string]any
	if w.person != nil {
		res, _ := h.a.in.Op(ctx, h.a.app.LoginEntity, "buscar", w.person)
		person, _ = res.(map[string]any)
		if person == nil {
			return false // the person no longer exists
		}
	}
	for _, row := range []map[string]any{c.after, c.before} {
		if row != nil && h.a.in.Can(ctx, person, e, "ver", row) {
			return true
		}
	}
	return false
}

func (h *liveHub) add(w *watcher) {
	h.mu.Lock()
	h.watchers[w] = true
	h.mu.Unlock()
}

func (h *liveHub) remove(w *watcher) {
	h.mu.Lock()
	delete(h.watchers, w)
	h.mu.Unlock()
}

// dependencies: the models a page at path shows — its chain, the children
// shown under its record, the data its indicators count — or nil when path
// is not one of the pages or the viewer may not load it.
func (ps *pageSite) dependencies(r *http.Request, path string) map[string]bool {
	req := r.Clone(r.Context())
	req.URL.Path = path
	pg, parts := ps.pageOf(req)
	if pg == nil {
		return nil
	}
	deps := map[string]bool{}
	for _, ind := range pg.Indicators {
		deps[ind.Entity] = true
	}
	if pg.Show == "" {
		return deps // a dashboard
	}
	chain, api, _, _ := ps.resolve(pg, parts)
	if code, _, _ := ps.call(r, "GET", api, nil); code != http.StatusOK {
		return nil // the viewer may not see this page
	}
	for _, st := range chain {
		deps[st.e.Singular] = true
	}
	if last := chain[len(chain)-1]; last.ref != "" {
		for _, c := range ps.a.childrenOf(last.e) {
			deps[c.Singular] = true
		}
	}
	return deps
}

// serveLive is the subscription of an open page: Server-Sent Events, one
// "mudou" per (coalesced) change, a comment every 25 s to keep it open.
func (ps *pageSite) serveLive(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("p")
	if !strings.HasPrefix(path, "/") {
		http.Error(w, "página inválida", http.StatusBadRequest)
		return
	}
	deps := ps.dependencies(r, path)
	if deps == nil {
		http.NotFound(w, r)
		return
	}
	ctx := &interp.Context{Request: r}
	atual, _ := ps.a.s.identify(ctx, r)
	wt := &watcher{depends: deps, signal: make(chan struct{}, 1)}
	if atual != nil {
		wt.person = atual["id"]
	}
	hub := ps.a.live
	hub.add(wt)
	defer hub.remove(wt)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	rc := http.NewResponseController(w)
	write := func(s string) bool {
		rc.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if _, err := fmt.Fprint(w, s); err != nil {
			return false
		}
		return rc.Flush() == nil
	}
	if !write("retry: 2000\n: ligado\n\n") {
		return
	}
	beat := time.NewTicker(25 * time.Second)
	defer beat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-beat.C:
			if !write(": batimento\n\n") {
				return
			}
		case <-wt.signal:
			if !write("event: mudou\ndata: {}\n\n") {
				return
			}
		}
	}
}

// liveScript refreshes the live regions of a page when it changed, and once
// after a reconnection (something may have been missed meanwhile). It waits
// while the person is typing in a form.
const liveScript = `(function () {
  if (!window.EventSource) return;
  var pending = false, busy = false;
  function typing() {
    var el = document.activeElement;
    return el && el.form && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.tagName === 'SELECT');
  }
  function refresh() {
    if (busy || typing()) { pending = true; return; }
    busy = true; pending = false;
    fetch(location.href, {credentials: 'same-origin', headers: {'X-Germanio-Vivo': '1'}})
      .then(function (r) { return r.ok ? r.text() : null; })
      .then(function (html) {
        if (!html) return;
        var doc = new DOMParser().parseFromString(html, 'text/html');
        document.querySelectorAll('[data-vivo]').forEach(function (el) {
          var fresh = doc.querySelector('[data-vivo="' + el.getAttribute('data-vivo') + '"]');
          if (fresh) el.replaceWith(document.importNode(fresh, true));
        });
      })
      .finally(function () { busy = false; if (pending) refresh(); });
  }
  document.addEventListener('focusout', function () { setTimeout(function () { if (pending) refresh(); }, 0); });
  var opened = false;
  var es = new EventSource('/_ge/atualizacoes?p=' + encodeURIComponent(location.pathname));
  es.addEventListener('mudou', refresh);
  es.onopen = function () { if (opened) refresh(); opened = true; };
})();
`

func serveLiveScript(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	fmt.Fprint(w, liveScript)
}
