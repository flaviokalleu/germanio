package servidor

import (
	"html/template"
	"net/url"
	"sort"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// Data shown by state (docs/gep/0023-por-estado.md, em teste): one column per
// state; moving a card is the transition that leads to its column. Each card
// lists the moves its viewer may make as ordinary buttons (keyboard, screen
// readers, no JavaScript); dragging, in the page script, uses the same
// buttons. The server checks every move as any action.

func byState(pg *ast.PageDecl, e *ast.Entity) bool {
	if pg == nil {
		return false
	}
	for _, bs := range pg.ByState {
		if bs.Data == e.Singular {
			return true
		}
	}
	return false
}

// stateColumns: the initial state, then the transitions' targets in the
// order their fields were declared.
func stateColumns(e *ast.Entity) []string {
	cols := []string{e.Initial}
	seen := map[string]bool{e.Initial: true}
	order := map[string]int{}
	for i, f := range e.Model.Fields {
		order[strings.TrimSuffix(strings.ToLower(f.Name), "_em")] = i
	}
	var targets []string
	for _, tr := range e.Transitions {
		if !seen[tr.Target] {
			seen[tr.Target] = true
			targets = append(targets, tr.Target)
		}
	}
	sort.Slice(targets, func(i, j int) bool {
		oi, iok := order[targets[i]]
		oj, jok := order[targets[j]]
		if iok != jok {
			return iok
		}
		if oi != oj {
			return oi < oj
		}
		return targets[i] < targets[j]
	})
	return append(cols, targets...)
}

type boardMove struct {
	Action, Label, Target string
}

type boardCard struct {
	ID, Href, Title string
	Moves           []boardMove
}

type boardColumn struct {
	State, Label string
	Cards        []boardCard
}

type boardView struct {
	Region, CSRF string
	Columns      []boardColumn
}

// board draws records of e (as the API shows them) in columns by state.
func (ps *pageSite) board(ctx *interp.Context, atual map[string]any, e *ast.Entity, rows []any, base, csrf, region string) template.HTML {
	states := stateColumns(e)
	v := boardView{Region: region, CSRF: csrf}
	index := map[string]int{}
	for i, st := range states {
		index[st] = i
		v.Columns = append(v.Columns, boardColumn{State: st, Label: label(st)})
	}
	var verbs []string
	for verb := range e.Transitions {
		verbs = append(verbs, verb)
	}
	sort.Strings(verbs)
	for _, it := range rows {
		row, _ := it.(map[string]any)
		st := toStr(row[e.StateField])
		col, ok := index[st]
		if !ok {
			continue
		}
		ref := display(row["id"])
		if n, ok := row["numero"]; ok && n != nil {
			ref = display(n)
		}
		href := base + "/" + url.PathEscape(ref)
		card := boardCard{ID: display(row["id"]), Href: href, Title: titleOf(e, row)}
		if atual != nil {
			record := ps.record(ctx, e, row)
			for _, verb := range verbs {
				tr := e.Transitions[verb]
				if tr.Target == st || !ps.available(ctx, atual, e, verb, record) {
					continue
				}
				card.Moves = append(card.Moves, boardMove{Action: href + "/acao/" + verb, Label: label(verb), Target: tr.Target})
			}
		}
		v.Columns[col].Cards = append(v.Columns[col].Cards, card)
	}
	return htmlOf(boardTpl, v)
}

var boardTpl = tpl(`<p class="sr" aria-live="polite" data-quadro-aviso></p><label class="filtro-quadro">Filtrar <input type="search" data-filtro-quadro></label><div class="quadro" data-vivo="{{.Region}}">{{range .Columns}}<section class="coluna" data-estado="{{.State}}" aria-label="{{.Label}}"><h3>{{.Label}} <span class="contagem">{{len .Cards}}</span></h3><ul>{{range .Cards}}<li class="cartao" data-id="{{.ID}}" tabindex="0"{{if .Moves}} draggable="true" aria-keyshortcuts="ArrowLeft ArrowRight"{{end}}><a href="{{.Href}}">{{.Title}}</a>{{if .Moves}}<div class="mover">{{range .Moves}}<form method="post" action="{{.Action}}" data-alvo="{{.Target}}"><input type="hidden" name="_csrf" value="{{$.CSRF}}"><button>{{.Label}}</button></form>{{end}}</div>{{end}}</li>{{end}}</ul></section>{{end}}</div>`)
