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

// A page written with the Portuguese words (accents included) reads the same
// as the one written with the older words, and a card ends where its
// indentation ends: a code block after the cards belongs to the section.
func TestPaginaEmPortugues(t *testing.T) {
	input := `crie sistema Site

página "/"
    título "Início"

    navegação
        marca "/assets/logo.png" "Site"
        link "Docs" "/docs"
        botão "Baixar" "/baixar"

    capa
        fundo "/assets/montanhas.png"
        selo "NOVO"
        título "Diga o que quer"
        destaque "e pronto."
        descrição "Uma linguagem."
        botão primário "Começar" "/docs"
        botão secundário "Ver" "/ver"
        ponto "Em português"
        ponto "Seguro"
        código "app.ge" "crie sistema X"

    seção "Recursos"
        fundo claro
        subtítulo "Tudo"
        imagem "/assets/editor.png"
        grade 4 colunas
            cartão
                ícone "⚡"
                etiqueta "NOVO"
                título "Simples"
                texto "Sintaxe natural"
                código "ge run app.ge"
            cartão
                título "Dois"
        código "app.ge" "crie sistema Y"

    rodapé
        direitos "Site © 2026"
        link "GitHub" "https://github.com"
`
	toks, err := lexer.New(input).Tokenize()
	if err != nil {
		t.Fatal(err)
	}
	prog, err := New(toks).Parse()
	if err != nil {
		t.Fatal(err)
	}
	if len(prog.Pages) != 1 || prog.Pages[0].Path != "/" || prog.Pages[0].Title != "Início" {
		t.Fatalf("página: %+v", prog.Pages)
	}
	blocks := prog.Pages[0].Blocks
	if len(blocks) != 4 {
		t.Fatalf("blocos: %d", len(blocks))
	}
	nav := blocks[0].(*ast.PageNavbar)
	if nav.Logo != "/assets/logo.png" || nav.Brand != "Site" || len(nav.Links) != 1 || len(nav.Buttons) != 1 {
		t.Fatalf("navegação: %+v", nav)
	}
	hero := blocks[1].(*ast.PageHero)
	if hero.Background != "/assets/montanhas.png" || hero.Badge != "NOVO" || hero.Title != "Diga o que quer" ||
		hero.Description != "Uma linguagem." || len(hero.Buttons) != 2 || !hero.Buttons[0].Primary || hero.Buttons[1].Primary ||
		len(hero.Points) != 2 || hero.CodePreview == nil || hero.CodePreview.Code != "crie sistema X" {
		t.Fatalf("capa: %+v", hero)
	}
	sec := blocks[2].(*ast.PageSection)
	if !sec.Light || sec.Subtitle != "Tudo" || sec.Image != "/assets/editor.png" || sec.Columns != 4 || len(sec.Cards) != 2 {
		t.Fatalf("seção: %+v", sec)
	}
	if c := sec.Cards[0]; c.Icon != "⚡" || c.Tag != "NOVO" || c.Code != "ge run app.ge" {
		t.Fatalf("cartão: %+v", c)
	}
	if len(sec.CodeBlocks) != 1 || sec.CodeBlocks[0].Code != "crie sistema Y" {
		t.Fatalf("o código depois dos cartões é da seção: %+v", sec.CodeBlocks)
	}
	if foot := blocks[3].(*ast.PageFooter); foot.Copyright != "Site © 2026" || len(foot.Links) != 1 {
		t.Fatalf("rodapé: %+v", foot)
	}
}

// página Clientes (without an address) stays the intent page.
func TestPaginaDeIntencaoNaoViraEndereco(t *testing.T) {
	input := "crie sistema X\n\nclientes\n    tem\n        nome\n\npágina Clientes\n    mostre clientes\n"
	toks, err := lexer.New(input).Tokenize()
	if err != nil {
		t.Fatal(err)
	}
	prog, err := New(toks).Parse()
	if err != nil {
		t.Fatal(err)
	}
	if len(prog.Pages) != 0 {
		t.Fatalf("página de intenção lida como endereço: %+v", prog.Pages)
	}
}
