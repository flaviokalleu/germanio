package parser

import (
	"strconv"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/lexer"
)

// pageWord is a page keyword as written, lowercased and without accents, so
// seção, cartão, código and rodapé read like secao, cartao, codigo, rodape.
func pageWord(tok lexer.Token) string { return foldWord(strings.ToLower(tok.Value)) }

// pageBlockWord: the words that open a block of a page; any of them ends the
// block before it.
func pageBlockWord(w string) bool {
	switch w {
	case "pagina", "page", "navbar", "navegacao", "header", "hero", "capa", "destaque_principal",
		"secao", "section", "rodape", "footer":
		return true
	}
	return false
}

// parseCustomPage parses a single custom page declaration.
//
//	pagina "/site"
//	  titulo "Germanio — Portal Oficial"
//	  navbar ...
//	  hero ...
//	  secao ...
//	  rodape ...
func (p *Parser) parseCustomPage() (*ast.CustomPage, error) {
	p.advance() // consume 'pagina' or 'page'
	p.skipWhitespace()

	page := &ast.CustomPage{}

	if !p.isAtEnd() && p.current().Type == lexer.TokenString {
		page.Path = p.advance().Value
	}
	p.skipWhitespace()

	for !p.isAtEnd() && !p.isBlockKeyword() {
		tok := p.current()
		if tok.Type == lexer.TokenNewline || tok.Type == lexer.TokenIndent {
			p.advance()
			continue
		}
		if tok.Type == lexer.TokenEOF {
			break
		}

		val := pageWord(tok)
		if val == "pagina" || val == "page" || val == "rotas" || val == "routes" ||
			val == "paginas" || val == "pages" || val == "tabela" || val == "quando" ||
			val == "app" || val == "banco" || val == "tema" || val == "sidebar" {
			break
		}

		switch val {
		case "titulo", "title":
			p.advance()
			p.skipWhitespace()
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				page.Title = p.advance().Value
			}
		case "html", "conteudo", "content":
			p.advance()
			p.skipWhitespace()
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				page.Content = p.advance().Value
			}
		case "navbar", "navegacao", "header":
			p.advance()
			page.Blocks = append(page.Blocks, p.parsePageNavbar())
		case "hero", "capa", "destaque_principal":
			p.advance()
			page.Blocks = append(page.Blocks, p.parsePageHero())
		case "secao", "section":
			p.advance()
			page.Blocks = append(page.Blocks, p.parsePageSection())
		case "rodape", "footer":
			p.advance()
			page.Blocks = append(page.Blocks, p.parsePageFooter())
		default:
			p.advance()
		}
	}

	return page, nil
}

func (p *Parser) parsePageNavbar() *ast.PageNavbar {
	nav := &ast.PageNavbar{}
	p.skipWhitespace()

	for !p.isAtEnd() && !p.isBlockKeyword() {
		tok := p.current()
		if tok.Type == lexer.TokenNewline || tok.Type == lexer.TokenIndent {
			p.advance()
			continue
		}
		if tok.Type == lexer.TokenEOF {
			break
		}

		val := pageWord(tok)
		if pageBlockWord(val) {
			break
		}

		switch val {
		case "logo", "marca":
			p.advance()
			p.skipWhitespace()
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				nav.Logo = p.advance().Value
			}
			p.skipWhitespace()
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				nav.Brand = p.advance().Value
			}
		case "link":
			p.advance()
			p.skipWhitespace()
			link := &ast.PageNavLink{}
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				link.Label = p.advance().Value
			}
			p.skipWhitespace()
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				link.URL = p.advance().Value
			}
			nav.Links = append(nav.Links, link)
		case "botao", "button":
			p.advance()
			p.skipWhitespace()
			btn := &ast.PageNavButton{Primary: true}
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				btn.Label = p.advance().Value
			}
			p.skipWhitespace()
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				btn.URL = p.advance().Value
			}
			nav.Buttons = append(nav.Buttons, btn)
		default:
			return nav
		}
	}
	return nav
}

func (p *Parser) parsePageHero() *ast.PageHero {
	hero := &ast.PageHero{}
	p.skipWhitespace()

	for !p.isAtEnd() && !p.isBlockKeyword() {
		tok := p.current()
		if tok.Type == lexer.TokenNewline || tok.Type == lexer.TokenIndent {
			p.advance()
			continue
		}
		if tok.Type == lexer.TokenEOF {
			break
		}

		val := pageWord(tok)
		if pageBlockWord(val) {
			break
		}

		switch val {
		case "badge", "selo", "tag", "etiqueta":
			p.advance()
			p.skipWhitespace()
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				hero.Badge = p.advance().Value
			}
		case "titulo", "title":
			p.advance()
			p.skipWhitespace()
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				hero.Title = p.advance().Value
			}
		case "destaque", "highlight":
			p.advance()
			p.skipWhitespace()
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				hero.Highlight = p.advance().Value
			}
		case "descricao", "description", "texto":
			p.advance()
			p.skipWhitespace()
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				hero.Description = p.advance().Value
			}
		case "comando", "terminal":
			p.advance()
			p.skipWhitespace()
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				hero.Command = p.advance().Value
			}
		case "mascote", "imagem":
			p.advance()
			p.skipWhitespace()
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				hero.Mascot = p.advance().Value
			}
		case "botao", "button":
			p.advance()
			p.skipWhitespace()
			isPrimary := true
			if !p.isAtEnd() && (pageWord(p.current()) == "primario" || pageWord(p.current()) == "secundario") {
				if pageWord(p.current()) == "secundario" {
					isPrimary = false
				}
				p.advance()
				p.skipWhitespace()
			}
			btn := &ast.PageHeroButton{Primary: isPrimary}
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				btn.Label = p.advance().Value
			}
			p.skipWhitespace()
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				btn.URL = p.advance().Value
			}
			hero.Buttons = append(hero.Buttons, btn)
		case "fundo", "background":
			p.advance()
			p.skipWhitespace()
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				hero.Background = p.advance().Value
			}
		case "ponto", "point":
			p.advance()
			p.skipWhitespace()
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				hero.Points = append(hero.Points, p.advance().Value)
			}
		case "codigo", "code":
			p.advance()
			p.skipWhitespace()
			preview := &ast.PageCodePreview{}
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				preview.Filename = p.advance().Value
			}
			p.skipWhitespace()
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				preview.Code = p.advance().Value
			}
			hero.CodePreview = preview
		default:
			return hero
		}
	}
	return hero
}

func (p *Parser) parsePageSection() *ast.PageSection {
	sec := &ast.PageSection{Columns: 3}
	p.skipWhitespace()

	if !p.isAtEnd() && p.current().Type == lexer.TokenString {
		sec.Title = p.advance().Value
	}
	p.skipWhitespace()

	for !p.isAtEnd() && !p.isBlockKeyword() {
		tok := p.current()
		if tok.Type == lexer.TokenNewline || tok.Type == lexer.TokenIndent {
			p.advance()
			continue
		}
		if tok.Type == lexer.TokenEOF {
			break
		}

		val := pageWord(tok)
		if pageBlockWord(val) {
			break
		}

		switch val {
		case "subtitulo", "subtitle", "descricao":
			p.advance()
			p.skipWhitespace()
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				sec.Subtitle = p.advance().Value
			}
		case "fundo", "background":
			p.advance()
			p.skipWhitespace()
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				sec.Background = p.advance().Value
			} else if !p.isAtEnd() && (strings.EqualFold(p.current().Value, "claro") || strings.EqualFold(p.current().Value, "light")) {
				p.advance()
				sec.Light = true
			}
		case "imagem", "image":
			p.advance()
			p.skipWhitespace()
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				sec.Image = p.advance().Value
			}
		case "grade", "grid":
			p.advance()
			p.skipWhitespace()
			if !p.isAtEnd() && p.current().Type == lexer.TokenNumber {
				if n, err := strconv.Atoi(p.current().Value); err == nil && n > 0 {
					sec.Columns = n
				}
				p.advance()
			}
			p.skipWhitespace()
			if !p.isAtEnd() && (pageWord(p.current()) == "colunas" || p.current().Value == "cols") {
				p.advance()
			}
		case "card", "cartao":
			cardCol := tok.Column
			p.advance()
			p.skipWhitespace()
			card := &ast.PageCard{}
			for !p.isAtEnd() && !p.isBlockKeyword() {
				cTok := p.current()
				if cTok.Type == lexer.TokenNewline || cTok.Type == lexer.TokenIndent {
					p.advance()
					continue
				}
				if cTok.Type == lexer.TokenEOF {
					break
				}
				cVal := pageWord(cTok)
				// what is not indented under the card belongs to the section
				if cTok.Column <= cardCol || pageBlockWord(cVal) {
					break
				}
				switch cVal {
				case "icone", "icon":
					p.advance()
					p.skipWhitespace()
					if !p.isAtEnd() && p.current().Type == lexer.TokenString {
						card.Icon = p.advance().Value
					}
				case "tag", "badge", "etiqueta", "selo":
					p.advance()
					p.skipWhitespace()
					if !p.isAtEnd() && p.current().Type == lexer.TokenString {
						card.Tag = p.advance().Value
					}
				case "titulo", "title":
					p.advance()
					p.skipWhitespace()
					if !p.isAtEnd() && p.current().Type == lexer.TokenString {
						card.Title = p.advance().Value
					}
				case "texto", "text", "descricao":
					p.advance()
					p.skipWhitespace()
					if !p.isAtEnd() && p.current().Type == lexer.TokenString {
						card.Text = p.advance().Value
					}
				case "link", "url":
					p.advance()
					p.skipWhitespace()
					if !p.isAtEnd() && p.current().Type == lexer.TokenString {
						card.Link = p.advance().Value
					}
				case "botao", "button":
					p.advance()
					p.skipWhitespace()
					if !p.isAtEnd() && (pageWord(p.current()) == "primario" || pageWord(p.current()) == "primary") {
						card.ButtonPrimary = true
						p.advance()
					}
					p.skipWhitespace()
					if !p.isAtEnd() && p.current().Type == lexer.TokenString {
						card.ButtonLabel = p.advance().Value
					}
					p.skipWhitespace()
					if !p.isAtEnd() && p.current().Type == lexer.TokenString {
						card.ButtonURL = p.advance().Value
					}
				case "codigo", "code":
					p.advance()
					p.skipWhitespace()
					if !p.isAtEnd() && p.current().Type == lexer.TokenString {
						card.Code = p.advance().Value
					}
				default:
					goto doneCard
				}
			}
		doneCard:
			sec.Cards = append(sec.Cards, card)
		case "codigo", "code":
			p.advance()
			p.skipWhitespace()
			codeBlock := &ast.PageCodeBlock{}
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				codeBlock.Title = p.advance().Value
			}
			p.skipWhitespace()
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				codeBlock.Code = p.advance().Value
			}
			sec.CodeBlocks = append(sec.CodeBlocks, codeBlock)
		default:
			return sec
		}
	}
	return sec
}

func (p *Parser) parsePageFooter() *ast.PageFooter {
	footer := &ast.PageFooter{}
	p.skipWhitespace()

	for !p.isAtEnd() && !p.isBlockKeyword() {
		tok := p.current()
		if tok.Type == lexer.TokenNewline || tok.Type == lexer.TokenIndent {
			p.advance()
			continue
		}
		if tok.Type == lexer.TokenEOF {
			break
		}

		val := pageWord(tok)
		if pageBlockWord(val) {
			break
		}

		switch val {
		case "copyright", "texto", "text", "direitos":
			p.advance()
			p.skipWhitespace()
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				footer.Copyright = p.advance().Value
			}
		case "link":
			p.advance()
			p.skipWhitespace()
			link := &ast.PageNavLink{}
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				link.Label = p.advance().Value
			}
			p.skipWhitespace()
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				link.URL = p.advance().Value
			}
			footer.Links = append(footer.Links, link)
		default:
			return footer
		}
	}
	return footer
}
