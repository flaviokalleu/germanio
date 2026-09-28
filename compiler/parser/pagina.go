package parser

import (
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/lexer"
)

// Page sections (docs/gep/0002-secoes-de-pagina.md, em teste): a closed
// table of regions with domain meaning. Each one replaces only its own
// default; none is required.

var pageSections = map[string]bool{"topo": true, "filtros": true, "colunas": true, "vazio": true}

// pageSection reads the section at body[i]; it returns the index of its last
// line.
func (p *Parser) pageSection(pg *ast.PageDecl, body []dline, i int) (int, error) {
	kind := wordsOf(body[i].toks)[0]
	last := i + len(children(body, i))
	kids := directChildren(body, i)
	if len(kids) == 0 {
		return last, p.teach(body[i].toks[0], "a seção "+kind+" está vazia", pageSectionWhy[kind], pageSectionFix[kind], "página "+pg.Name)
	}
	for _, k := range kids {
		line := body[k]
		w := wordsOf(line.toks)
		switch kind {
		case "topo":
			switch {
			case w[0] == "titulo" || w[0] == "texto":
				text, err := p.pageText(line, w[0])
				if err != nil {
					return last, err
				}
				if w[0] == "titulo" {
					pg.Title = text
				} else {
					pg.Text = text
				}
			case w[0] == "acoes":
				acts := directChildren(body, k)
				if len(acts) == 0 {
					return last, p.teach(line.toks[0], "ações está vazia", "ações lista os verbos que a página oferece no topo", "escreva um verbo por linha, com rótulo opcional: criar \"Novo cliente\"", "página "+pg.Name)
				}
				for _, a := range acts {
					act, err := p.pageAction(body[a])
					if err != nil {
						return last, err
					}
					pg.Actions = append(pg.Actions, act)
				}
			default:
				return last, p.teach(line.toks[0], "\""+lineText(line)+"\" não é parte do topo", "o topo tem título, texto e ações", "use: título \"…\", texto \"…\" ou ações", "página "+pg.Name)
			}
		case "filtros", "colunas":
			if len(directChildren(body, k)) > 0 {
				return last, p.teach(body[k+1].toks[0], "\""+lineText(body[k+1])+"\" está recuada abaixo de \""+lineText(line)+"\"", "em "+kind+", cada linha é um item", "alinhe com os outros itens de "+kind, "página "+pg.Name)
			}
			name, _ := phrase(w)
			if kind == "filtros" {
				pg.Filters = append(pg.Filters, name)
			} else {
				pg.Columns = append(pg.Columns, name)
			}
		case "vazio":
			if pg.Empty == nil {
				pg.Empty = &ast.PageEmpty{Pos: p.at(body[i].toks[0])}
			}
			switch w[0] {
			case "titulo", "texto":
				text, err := p.pageText(line, w[0])
				if err != nil {
					return last, err
				}
				if w[0] == "titulo" {
					pg.Empty.Title = text
				} else {
					pg.Empty.Text = text
				}
			case "acao":
				act, err := p.pageAction(dline{toks: line.toks[1:], indent: line.indent})
				if err != nil {
					return last, err
				}
				pg.Empty.Action = act
			default:
				return last, p.teach(line.toks[0], "\""+lineText(line)+"\" não é parte de vazio", "vazio diz o que a página mostra quando não há nada", "use: título \"…\", texto \"…\" ou ação criar \"…\"", "página "+pg.Name)
			}
		}
	}
	return last, nil
}

var pageSectionWhy = map[string]string{
	"topo":    "o topo tem o título da página, um texto e as ações",
	"filtros": "filtros lista a pesquisa e os campos pelos quais se filtra",
	"colunas": "colunas lista, em ordem, os campos que a tabela mostra",
	"vazio":   "vazio diz o que aparece quando não há nada para mostrar",
}

var pageSectionFix = map[string]string{
	"topo":    "escreva abaixo: título \"Clientes\"",
	"filtros": "escreva abaixo um item por linha: pesquisar, cidade",
	"colunas": "escreva abaixo um campo por linha: nome, email",
	"vazio":   "escreva abaixo: título \"Nenhum cliente\"",
}

// pageText reads `título "X"` / `texto "X"`.
func (p *Parser) pageText(line dline, what string) (string, error) {
	if len(line.toks) != 2 || line.toks[1].Type != lexer.TokenString {
		return "", p.teach(line.toks[0], what+" precisa de um texto entre aspas", "o texto aparece como está escrito", "escreva: "+what+" \"Clientes\"", "")
	}
	return line.toks[1].Value, nil
}

// pageAction reads `verbo ["rótulo"]`. A label alone is refused: a text is
// not an action (GEP 0002: the page must say which action it offers).
func (p *Parser) pageAction(line dline) (*ast.PageAction, error) {
	toks := line.toks
	if len(toks) == 0 {
		return nil, p.errorf(lexer.Token{}, "ação vazia")
	}
	if toks[0].Type == lexer.TokenString {
		return nil, p.teach(toks[0], "\""+toks[0].Value+"\" é só um rótulo: falta dizer qual ação ele faz", "um texto não é uma ação; a página precisa saber o que o botão faz", "escreva o verbo antes do rótulo: criar \""+toks[0].Value+"\"", "")
	}
	act := &ast.PageAction{Pos: p.at(toks[0])}
	var words []string
	for _, t := range toks {
		if t.Type == lexer.TokenString {
			act.Label = t.Value
			break
		}
		words = append(words, wordsOf([]lexer.Token{t})...)
	}
	act.Verb = strings.Join(words, "_")
	return act, nil
}
