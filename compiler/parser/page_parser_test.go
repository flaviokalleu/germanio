package parser

import (
	"testing"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/lexer"
)

func TestParseDeclarativePage(t *testing.T) {
	input := `
pagina "/"
  titulo "Germanio — Diga o que quer construir"

  navbar
    logo "/assets/germanio.png" "Germanio"
    link "Início" "/"
    link "Docs" "/docs"
    botao "Instalar" "/instalar"

  hero
    badge "GERMANIO"
    titulo "Diga o que quer construir."
    destaque "Germanio transforma em software."
    descricao "Uma linguagem full-stack."
    botao primario "Começar →" "/docs"
    botao secundario "Playground" "/playground"
    comando "ge instalar"

  secao "Recursos"
    subtitulo "Tudo o que você precisa"
    grade 3 colunas
      card
        icone "⚡"
        tag "NOVO"
        titulo "Simples"
        texto "Sintaxe natural"
        link "/docs"
        botao primario "Abrir" "/abrir"

  rodape
    copyright "Germanio © 2026"
    link "GitHub" "https://github.com/flaviokalleu/germanio"
`

	l := lexer.New(input)
	toks, err := l.Tokenize()
	if err != nil {
		t.Fatalf("tokenize error: %v", err)
	}

	p := New(toks)
	prog, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(prog.Pages) != 1 {
		t.Fatalf("got %d pages, want 1", len(prog.Pages))
	}

	page := prog.Pages[0]
	if page.Path != "/" {
		t.Errorf("got path %q, want '/'", page.Path)
	}
	if page.Title != "Germanio — Diga o que quer construir" {
		t.Errorf("got title %q", page.Title)
	}
	if len(page.Blocks) != 4 {
		t.Fatalf("got %d blocks, want 4 (navbar, hero, secao, rodape)", len(page.Blocks))
	}

	nav, ok := page.Blocks[0].(*ast.PageNavbar)
	if !ok || nav.Brand != "Germanio" || len(nav.Links) != 2 || len(nav.Buttons) != 1 {
		t.Errorf("invalid navbar: %+v", nav)
	}

	hero, ok := page.Blocks[1].(*ast.PageHero)
	if !ok || hero.Title != "Diga o que quer construir." || hero.Badge != "GERMANIO" || len(hero.Buttons) != 2 {
		t.Errorf("invalid hero: %+v", hero)
	}

	sec, ok := page.Blocks[2].(*ast.PageSection)
	if !ok || sec.Title != "Recursos" || sec.Columns != 3 || len(sec.Cards) != 1 {
		t.Errorf("invalid section: %+v", sec)
	}

	card := sec.Cards[0]
	if card.ButtonLabel != "Abrir" || card.ButtonURL != "/abrir" || !card.ButtonPrimary {
		t.Errorf("botão do card foi perdido no parse: %+v", card)
	}

	foot, ok := page.Blocks[3].(*ast.PageFooter)
	if !ok || foot.Copyright != "Germanio © 2026" || len(foot.Links) != 1 {
		t.Errorf("invalid footer: %+v", foot)
	}
}
