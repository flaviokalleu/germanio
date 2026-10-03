package servidor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

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
	// rows: tables of the page that take a single row (by model); a change
	// of their data sends the row drawn for this viewer instead of a refresh
	rows  map[string]*rowRegion
	queue chan string // ready row events; when full, a refresh is asked
	// only: records of the page's address (model → id): changes of other
	// records of those data are not on this page
	only   map[string]string
	path   string      // the page (people typing on it are announced to its viewers)
	typing chan string // who is typing (one slot: ephemeral, never queued)
}

// rowRegion is a table of a page: its live region, the address its rows
// link to, its columns, and the parent record it belongs to (scope).
type rowRegion struct {
	name, base string
	cols       []string
	scope      map[string]any
}

func (r *rowRegion) holds(row map[string]any) bool {
	if row == nil {
		return false
	}
	for k, v := range r.scope {
		if fmt.Sprint(row[k]) != fmt.Sprint(v) {
			return false
		}
	}
	return true
}

type liveHub struct {
	a        *intentAPI
	mu       sync.RWMutex
	watchers map[*watcher]bool
	// presence (GEP 0021): open pages per person, and when each person
	// went offline is still pending (grace period)
	pages   map[string]int
	leaving map[string]*time.Timer
}

func newLiveHub(a *intentAPI) *liveHub {
	h := &liveHub{a: a, watchers: map[*watcher]bool{}, pages: map[string]int{}, leaving: map[string]*time.Timer{}}
	a.in.OnChange = h.onChange
	if a.app.Presence {
		a.in.Online = h.online
	}
	return h
}

// presenceGrace: how long after the last page closes a person stays online
// (a reload or a short drop does not flicker).
func presenceGrace() time.Duration {
	if d, err := time.ParseDuration(os.Getenv("GERMANIO_PRESENCA_TOLERANCIA")); err == nil && d >= 0 {
		return d
	}
	return 10 * time.Second
}

func (h *liveHub) online(id any) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.pages[fmt.Sprint(id)] > 0
}

// arrive and leave count the open pages of a person; a change of presence
// is a change of the person (pages showing people refresh).
func (h *liveHub) arrive(id any) {
	if !h.a.app.Presence || id == nil {
		return
	}
	k := fmt.Sprint(id)
	h.mu.Lock()
	if t := h.leaving[k]; t != nil {
		// back within the grace period: the page that was leaving is this one
		t.Stop()
		delete(h.leaving, k)
		h.mu.Unlock()
		return
	}
	h.pages[k]++
	first := h.pages[k] == 1
	h.mu.Unlock()
	if first {
		h.presenceChanged(id)
	}
}

func (h *liveHub) leave(id any) {
	if !h.a.app.Presence || id == nil {
		return
	}
	k := fmt.Sprint(id)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.pages[k] > 1 {
		h.pages[k]--
		return
	}
	if h.leaving[k] != nil {
		return
	}
	h.leaving[k] = time.AfterFunc(presenceGrace(), func() {
		h.mu.Lock()
		gone := h.leaving[k] != nil
		delete(h.leaving, k)
		if gone {
			h.pages[k]--
			if h.pages[k] <= 0 {
				delete(h.pages, k)
			}
		}
		h.mu.Unlock()
		if gone && !h.online(id) {
			h.presenceChanged(id)
		}
	})
}

func (h *liveHub) presenceChanged(id any) {
	le := h.a.app.LoginEntity
	res, _ := h.a.in.Op(&interp.Context{}, le, "buscar", id)
	if row, _ := res.(map[string]any); row != nil {
		h.publish(change{model: le, before: row, after: row})
	}
}

// onChange records a write; it is announced once the change is kept.
func (h *liveHub) onChange(ctx *interp.Context, model string, before, after map[string]any) {
	c := change{model: strings.ToLower(model), before: before, after: after}
	announce := func() {
		h.publish(c)
		// a record read per person changes the unread count of its container
		if e := h.a.app.Entities[c.model]; e != nil && e.ReadParent != "" {
			row := c.after
			if row == nil {
				row = c.before
			}
			if res, _ := h.a.in.Op(&interp.Context{}, e.ReadParent, "buscar", row[e.ReadField]); res != nil {
				p := res.(map[string]any)
				h.publish(change{model: e.ReadParent, before: p, after: p})
			}
		}
		// the numbers a record shows about its records change with them (GEP 0047)
		for _, pc := range h.a.aggregateParents(c.model, c.before, c.after) {
			h.publish(pc)
		}
	}
	if ctx != nil {
		if st := txOf(ctx.Request); st != nil {
			st.onCommit = append(st.onCommit, announce)
			return
		}
	}
	announce() // outside a request transaction the write is already kept
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
	// what a viewer receives depends only on who they are and the table:
	// drawn once per (person, table) in this announcement
	drawn := map[string]string{}
	for _, w := range interested {
		if e == nil {
			w.refresh()
			continue
		}
		if id, ok := w.only[c.model]; ok && !sameID(c.after, id) && !sameID(c.before, id) {
			continue // another record of a data in the page's address
		}
		person, ok := h.viewer(ctx, w)
		if !ok {
			continue
		}
		region := w.rows[c.model]
		if region != nil && !region.holds(c.after) && !region.holds(c.before) {
			continue // a record of another parent: not on this page
		}
		seesAfter := c.after != nil && h.a.in.Can(ctx, person, e, "ver", c.after)
		seesBefore := c.before != nil && h.a.in.Can(ctx, person, e, "ver", c.before)
		if !seesAfter && !seesBefore {
			continue // a record this viewer cannot see changes silently for them
		}
		if region == nil {
			w.refresh()
			continue
		}
		key := fmt.Sprint(w.person, "\x00", region.name, "\x00", region.base, "\x00", seesAfter)
		msg, ok := drawn[key]
		if !ok {
			ev := map[string]any{"regiao": region.name}
			switch {
			case seesAfter && region.holds(c.after):
				out := serializeFor(ctx, h.a.in, person, e, c.after, false)
				ev["id"], ev["html"] = fmt.Sprint(c.after["id"]), rowHTML(e, out, region.base, region.cols, h.a.namedBy(ctx, person, e, out, region.scope))
				ev["acao"] = "criar"
				if c.before != nil {
					ev["acao"] = "editar"
				}
			default:
				ev["id"], ev["acao"] = fmt.Sprint(c.before["id"]), "excluir"
			}
			b, _ := json.Marshal(ev)
			msg = string(b)
			drawn[key] = msg
		}
		select {
		case w.queue <- msg:
		default:
			w.refresh() // too many pending rows: one refresh instead
		}
	}
}

func sameID(row map[string]any, id string) bool {
	return row != nil && fmt.Sprint(row["id"]) == id
}

func (w *watcher) refresh() {
	select {
	case w.signal <- struct{}{}:
	default: // a refresh is already pending
	}
}

// viewer: the person looking, as they are now (nil when not signed in);
// ok=false when the person no longer exists.
func (h *liveHub) viewer(ctx *interp.Context, w *watcher) (map[string]any, bool) {
	if w.person == nil {
		return nil, true
	}
	res, _ := h.a.in.Op(ctx, h.a.app.LoginEntity, "buscar", w.person)
	person, _ := res.(map[string]any)
	return person, person != nil
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
func (ps *pageSite) dependencies(r *http.Request, path string) (map[string]bool, map[string]*rowRegion, map[string]string) {
	req := r.Clone(r.Context())
	req.URL.Path = path
	pg, parts := ps.pageOf(req)
	if pg == nil {
		return nil, nil, nil
	}
	deps := map[string]bool{}
	rows := map[string]*rowRegion{}
	only := map[string]string{}
	for _, ind := range pg.Indicators {
		deps[ind.Entity] = true
	}
	if pg.Show == "" {
		return deps, rows, only // a dashboard
	}
	chain, api, _, _ := ps.resolve(pg, parts)
	if code, _, _ := ps.call(r, "GET", api, nil); code != http.StatusOK {
		return nil, nil, nil // the viewer may not see this page
	}
	for _, st := range chain {
		deps[st.e.Singular] = true
	}
	// the parent record of each level, as the page resolves it
	ctx := &interp.Context{Request: r}
	scope := map[string]any{}
	var parentRow map[string]any
	for i, st := range chain {
		if st.ref == "" {
			break
		}
		row := ps.a.find(ctx, st.e, st.ref, scope)
		if row == nil {
			return deps, rows, only
		}
		parentRow = row
		only[st.e.Singular] = fmt.Sprint(row["id"])
		if i+1 < len(chain) {
			scope = ps.a.parentScope(st.e, row, chain[i+1].e)
		}
	}
	last := chain[len(chain)-1]
	if last.ref == "" {
		var cols []string
		if len(chain) == 1 {
			cols = pg.Columns
		}
		if !byState(pg, last.e) { // a board refreshes as a whole
			rows[last.e.Singular] = &rowRegion{name: "lista", base: strings.TrimSuffix(path, "/"), cols: cols, scope: scope}
		}
		return deps, rows, only
	}
	for _, c := range ps.a.childrenOf(last.e) {
		deps[c.Singular] = true
		if byState(pg, c) {
			continue // a board refreshes as a whole
		}
		rows[c.Singular] = &rowRegion{name: "filhos-" + c.Plural, base: strings.TrimSuffix(path, "/") + "/" + c.Plural, scope: ps.a.parentScope(last.e, parentRow, c)}
	}
	return deps, rows, only
}

// serveLive is the subscription of an open page: Server-Sent Events, one
// "mudou" per (coalesced) change, a comment every 25 s to keep it open.
func (ps *pageSite) serveLive(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("p")
	if !strings.HasPrefix(path, "/") {
		http.Error(w, "página inválida", http.StatusBadRequest)
		return
	}
	deps, rows, only := ps.dependencies(r, path)
	if deps == nil {
		http.NotFound(w, r)
		return
	}
	ctx := &interp.Context{Request: r}
	atual, _ := ps.a.s.identify(ctx, r)
	wt := &watcher{depends: deps, signal: make(chan struct{}, 1), rows: rows, queue: make(chan string, 32), only: only, path: path, typing: make(chan string, 1)}
	if atual != nil {
		wt.person = atual["id"]
	}
	hub := ps.a.live
	hub.add(wt)
	defer hub.remove(wt)
	hub.arrive(wt.person)
	defer hub.leave(wt.person)

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
		case ev := <-wt.queue:
			if !write("event: linha\ndata: " + ev + "\n\n") {
				return
			}
		case ev := <-wt.typing:
			if !write("event: digitando\ndata: " + ev + "\n\n") {
				return
			}
		}
	}
}

// liveScript refreshes the live regions of a page when it changed, and once
// after a reconnection (something may have been missed meanwhile). It waits
// while the person is typing in a form.
const liveScript = `(function () {
  // boards (GEP 0023): dropping a card on a column submits that card's own
  // move button for the column; a card without that move is not accepted
  var dragged = null;
  document.addEventListener('dragstart', function (e) {
    var card = e.target.closest && e.target.closest('.cartao[draggable=true]');
    if (!card) return;
    dragged = card;
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('text/plain', card.getAttribute('data-id'));
  });
  function moveFor(col) {
    return dragged && col && dragged.querySelector('form[data-alvo="' + col.getAttribute('data-estado') + '"]');
  }
  document.addEventListener('dragover', function (e) {
    var col = e.target.closest && e.target.closest('.coluna');
    if (moveFor(col)) { e.preventDefault(); col.classList.add('alvo'); }
  });
  document.addEventListener('dragleave', function (e) {
    var col = e.target.closest && e.target.closest('.coluna');
    if (col) col.classList.remove('alvo');
  });
  // one move = one transition, through the card's own button; remembered so
  // it can be undone by the move back (which the server checks again)
  var done = [];
  function say(text) {
    var live = document.querySelector('[data-quadro-aviso]');
    if (live) live.textContent = text;
  }
  function columnName(state) {
    var col = document.querySelector('.coluna[data-estado="' + state + '"]');
    return col ? col.getAttribute('aria-label') : state;
  }
  // the cards a move applies to: the selected ones, when the moved card is
  // selected; each goes only if that move is one of its own buttons
  function group(card) {
    if (!card.classList.contains('selecionado')) return [card];
    return Array.prototype.slice.call(document.querySelectorAll('.cartao.selecionado'));
  }
  function move(card, form, remember) {
    var to = form.getAttribute('data-alvo');
    var col = document.querySelector('.coluna[data-estado="' + to + '"]');
    var moved = [];
    group(card).forEach(function (c) {
      var f = c === card ? form : c.querySelector('form[data-alvo="' + to + '"]');
      if (!f) return;
      moved.push({card: c, form: f, from: c.closest('.coluna').getAttribute('data-estado')});
      if (col) col.querySelector('ul').appendChild(c);
    });
    card.focus();
    Promise.all(moved.map(function (m) {
      return fetch(m.form.action, {method: 'POST', body: new URLSearchParams(new FormData(m.form)), credentials: 'same-origin'});
    })).then(function (rs) {
      if (rs.some(function (r) { return !r.ok; })) { say('Não foi possível mover.'); location.reload(); return; }
      if (remember) {
        done.push(moved.map(function (m) { return {id: m.card.getAttribute('data-id'), state: m.from}; }));
        showUndo();
      }
      say(moved.length === 1 ? (card.querySelector('a') || card).textContent + ' movido para ' + columnName(to) + '.'
                             : moved.length + ' cartões movidos para ' + columnName(to) + '.');
    });
  }
  function undo() {
    var last = done.pop();
    showUndo();
    if (!last) return;
    var back = [];
    last.forEach(function (m) {
      var card = document.querySelector('.cartao[data-id="' + m.id + '"]');
      var form = card && card.querySelector('form[data-alvo="' + m.state + '"]');
      if (form) back.push(fetch(form.action, {method: 'POST', body: new URLSearchParams(new FormData(form)), credentials: 'same-origin'}));
    });
    if (back.length < last.length) say('Não é possível desfazer todo o movimento.');
    else say('Movimento desfeito.');
    Promise.all(back).then(refresh);
  }
  function toggle(card) {
    card.classList.toggle('selecionado');
    say(document.querySelectorAll('.cartao.selecionado').length + ' cartões selecionados.');
  }
  // local filter: hides the cards that do not match, without asking the server
  document.addEventListener('input', function (e) {
    if (!e.target.matches('[data-filtro-quadro]')) return;
    var q = e.target.value.trim().toLowerCase();
    document.querySelectorAll('.quadro .cartao').forEach(function (c) {
      c.hidden = q !== '' && c.textContent.toLowerCase().indexOf(q) < 0;
    });
  });
  function showUndo() {
    var board = document.querySelector('.quadro');
    if (!board) return;
    var btn = document.querySelector('[data-desfazer]');
    if (!btn) {
      btn = document.createElement('button');
      btn.type = 'button';
      btn.textContent = 'Desfazer';
      btn.setAttribute('data-desfazer', '');
      btn.setAttribute('aria-keyshortcuts', 'Control+Z');
      btn.addEventListener('click', undo);
      board.parentNode.insertBefore(btn, board);
    }
    btn.hidden = done.length === 0;
  }
  document.addEventListener('drop', function (e) {
    var col = e.target.closest && e.target.closest('.coluna');
    var form = moveFor(col);
    if (!form) return;
    e.preventDefault();
    col.classList.remove('alvo');
    move(dragged, form, true);
  });
  document.addEventListener('keydown', function (e) {
    if ((e.ctrlKey || e.metaKey) && e.key === 'z' && done.length && !(e.target.form)) { e.preventDefault(); undo(); return; }
    var card = e.target.closest && e.target.closest('.cartao');
    if (card && e.key === ' ' && e.target === card) { e.preventDefault(); toggle(card); return; }
    if (!card || (e.key !== 'ArrowRight' && e.key !== 'ArrowLeft')) return;
    var col = card.closest('.coluna');
    var next = e.key === 'ArrowRight' ? col.nextElementSibling : col.previousElementSibling;
    var form = next && card.querySelector('form[data-alvo="' + next.getAttribute('data-estado') + '"]');
    if (!form) { say('Este cartão não pode ir para ' + (next ? next.getAttribute('aria-label') : 'lá') + '.'); return; }
    e.preventDefault();
    move(card, form, true);
  });
  document.addEventListener('dragend', function () { dragged = null; });

  // an image pasted or dropped into a formatted text is sent to the record
  // of the page and its Markdown inserted where the cursor is
  function sendImage(area, file) {
    var dest = document.querySelector('meta[name=ge-imagens]');
    var csrf = area.form && area.form.querySelector('input[name=_csrf]');
    if (!dest || !csrf || !/^image\//.test(file.type)) return false;
    fetch(dest.content + '?nome=' + encodeURIComponent(file.name || 'imagem'), {method: 'POST', body: file,
      credentials: 'same-origin', headers: {'X-CSRF-Token': csrf.value}})
      .then(function (r) { return r.ok ? r.json() : r.json().then(function (e) { throw new Error(e.message); }); })
      .then(function (out) {
        var at = area.selectionStart || area.value.length;
        area.value = area.value.slice(0, at) + out.markdown + area.value.slice(at);
      })
      .catch(function (e) { alert('Não foi possível enviar a imagem: ' + e.message); });
    return true;
  }
  document.addEventListener('paste', function (e) {
    if (!e.target.matches || !e.target.matches('textarea[data-formatado]')) return;
    var files = Array.prototype.filter.call(e.clipboardData.files || [], function (f) { return /^image\//.test(f.type); });
    if (files.length) { e.preventDefault(); files.forEach(function (f) { sendImage(e.target, f); }); }
  });
  document.addEventListener('drop', function (e) {
    if (!e.target.matches || !e.target.matches('textarea[data-formatado]') || !e.dataTransfer.files.length) return;
    e.preventDefault();
    Array.prototype.forEach.call(e.dataTransfer.files, function (f) { sendImage(e.target, f); });
  });

  // a card opens in a panel over the board (its link still works on its
  // own); forms inside the panel submit without leaving the board
  var panel = null;
  function openPanel(href) {
    fetch(href, {credentials: 'same-origin'}).then(function (r) { return r.text(); }).then(function (html) {
      var doc = new DOMParser().parseFromString(html, 'text/html');
      var main = doc.querySelector('main');
      if (!main) { location.href = href; return; }
      if (!panel) {
        panel = document.createElement('dialog');
        panel.className = 'painel';
        panel.setAttribute('aria-label', 'Detalhes');
        document.body.appendChild(panel);
        panel.addEventListener('submit', function (e) {
          var form = e.target;
          if ((form.method || '').toLowerCase() !== 'post') return;
          e.preventDefault();
          fetch(form.action, {method: 'POST', body: new URLSearchParams(new FormData(form)), credentials: 'same-origin'})
            .then(function () {
              if (/\/excluir$/.test(form.action)) panel.close(); else openPanel(panel.getAttribute('data-href'));
              refresh();
            });
        });
        panel.addEventListener('click', function (e) { if (e.target === panel) panel.close(); });
      }
      panel.setAttribute('data-href', href);
      panel.innerHTML = '<button type="button" class="fechar" aria-label="Fechar">×</button>';
      panel.querySelector('.fechar').addEventListener('click', function () { panel.close(); });
      Array.prototype.forEach.call(main.children, function (el) {
        if (el.matches('nav.migalhas, .aviso')) return;
        panel.appendChild(document.importNode(el, true));
      });
      if (!panel.open) panel.showModal();
    });
  }
  document.addEventListener('click', function (e) {
    var picked = e.target.closest && e.target.closest('.cartao');
    if (picked && (e.ctrlKey || e.metaKey) && !e.target.closest('form')) { e.preventDefault(); toggle(picked); return; }
    var a = e.target.closest && e.target.closest('.cartao a');
    if (!a || e.ctrlKey || e.metaKey || e.shiftKey || e.button !== 0) return;
    e.preventDefault();
    openPanel(a.href);
  });

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
      .then(function (r) { if (!r.ok) throw new Error(r.status); return r.text(); })
      .then(function (html) {
        var doc = new DOMParser().parseFromString(html, 'text/html');
        document.querySelectorAll('[data-vivo]').forEach(function (el) {
          var fresh = doc.querySelector('[data-vivo="' + el.getAttribute('data-vivo') + '"]');
          if (fresh) el.replaceWith(document.importNode(fresh, true));
        });
        var filter = document.querySelector('[data-filtro-quadro]');
        if (filter && filter.value) filter.dispatchEvent(new Event('input', {bubbles: true}));
      })
      .then(function () { busy = false; if (pending) refresh(); },
            function () { busy = false; pending = true; setTimeout(refresh, 2000); }); // offline: try again
  }
  window.addEventListener('online', refresh);
  document.addEventListener('focusout', function () { setTimeout(function () { if (pending) refresh(); }, 0); });
  // a row of a table changed: insert, replace or remove only that row (on
  // the first page without search or filters; otherwise refresh)
  function row(e) {
    var ev = JSON.parse(e.data);
    var region = document.querySelector('[data-vivo="' + ev.regiao + '"]');
    var body = region && region.querySelector('tbody');
    if (!body || location.search) { refresh(); return; }
    var old = body.querySelector('tr[data-id="' + ev.id + '"]');
    if (ev.acao === 'excluir') { if (old) old.remove(); return; }
    var t = document.createElement('tbody');
    t.innerHTML = ev.html;
    var fresh = t.firstElementChild;
    if (!fresh) { refresh(); return; }
    if (old) old.replaceWith(fresh); else body.insertBefore(fresh, body.firstChild);
  }
  var opened = false;
  var es = new EventSource('/_ge/atualizacoes?p=' + encodeURIComponent(location.pathname));
  es.addEventListener('mudou', refresh);
  es.addEventListener('linha', row);
  // someone else typing on this page (presence): shown for a few seconds
  var typingTimer = null;
  es.addEventListener('digitando', function (e) {
    var who = JSON.parse(e.data).nome;
    var box = document.querySelector('[data-digitando]');
    if (!box) {
      box = document.createElement('p');
      box.className = 'digitando';
      box.setAttribute('data-digitando', '');
      box.setAttribute('aria-live', 'polite');
      var main = document.querySelector('main');
      if (main) main.insertBefore(box, main.firstChild);
    }
    box.textContent = who + ' está digitando…';
    clearTimeout(typingTimer);
    typingTimer = setTimeout(function () { box.textContent = ''; }, 5000);
  });
  // tell the others while typing in a form of this page (at most every 3 s)
  var lastTyped = 0;
  document.addEventListener('input', function (e) {
    var form = e.target.form;
    if (!form || e.target.matches('[data-filtro-quadro]') || Date.now() - lastTyped < 3000) return;
    var csrf = form.querySelector('input[name=_csrf]');
    if (!csrf) return;
    lastTyped = Date.now();
    fetch('/_ge/digitando', {method: 'POST', credentials: 'same-origin',
      body: new URLSearchParams({p: location.pathname, _csrf: csrf.value})}).catch(function () {});
  });
  es.onopen = function () { if (opened) refresh(); opened = true; };
})();
`

func serveLiveScript(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	fmt.Fprint(w, liveScript)
}

// serveTyping: someone signed in is typing on a page (presence, GEP 0021).
// The viewers of the same page are told, once, by name; nothing is stored.
func (ps *pageSite) serveTyping(w http.ResponseWriter, r *http.Request) {
	if !ps.a.app.Presence {
		http.NotFound(w, r)
		return
	}
	ctx := &interp.Context{Request: r}
	atual, _ := ps.a.s.identify(ctx, r)
	sess := interp.SessaoDaRequisicao(r)
	if atual == nil || sess == nil {
		http.Error(w, "entre para escrever", http.StatusUnauthorized)
		return
	}
	if r.FormValue("_csrf") != fmt.Sprint(sess["csrf"]) {
		http.Error(w, "token CSRF ausente ou inválido", http.StatusForbidden)
		return
	}
	path := r.FormValue("p")
	if deps, _, _ := ps.dependencies(r, path); deps == nil || !strings.HasPrefix(path, "/") {
		http.NotFound(w, r)
		return
	}
	name := first(toStr(atual["nome"]), toStr(atual["name"]), toStr(atual["username"]), "alguém")
	b, _ := json.Marshal(map[string]string{"nome": name})
	ev := string(b)
	hub := ps.a.live
	hub.mu.RLock()
	for wt := range hub.watchers {
		if wt.path == path && fmt.Sprint(wt.person) != fmt.Sprint(atual["id"]) {
			select {
			case wt.typing <- ev:
			default: // already told
			}
		}
	}
	hub.mu.RUnlock()
	w.WriteHeader(http.StatusNoContent)
}
