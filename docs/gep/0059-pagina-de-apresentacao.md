# GEP 0059: Presentation pages in Portuguese (`página "/"`)

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (landing page of the project); decision by the maintainer
- **Level:** technical page level (`página "/endereço"`), not the intent page (`página Clientes`)
- **Layer:** pages (vocabulary and renderer)

## Problem

A product needs pages that show no data: a landing page, an "about" page, a page that explains
something. The intent page (`página Clientes` + `mostre clientes`) always shows a data, so these
pages are written at the technical page level (`pagina "/"` with `hero`, `navbar`, `card`…).
That level had English words (`hero`, `navbar`, `badge`, `card`, `tag`, `copyright`), did not
accept accents (`seção`, `cartão`, `código`, `rodapé` were not understood), and could not put an
image under the page's opening, make a light band, show a wide image or a worked example: a
designer's landing page could not be written in `.ge`.

## Proposal

```text
crie sistema Germanio

página "/"
    título "Germanio"

    navegação
        marca "/assets/germanio.png" "Germanio"
        link "Documentação" "/docs"
        botão "Baixar" "/baixar"

    capa
        fundo "/assets/montanhas.png"
        selo "UMA LINGUAGEM. APLICAÇÕES COMPLETAS."
        título "Desenvolva mais com"
        destaque "menos complexidade."
        descrição "…"
        botão primário "Começar agora" "#comecar"
        botão secundário "Ver documentação" "/docs"
        ponto "Sintaxe em português"
        código "app.ge" "crie sistema Tarefas"

    seção "Por que Germanio"
        fundo claro
        subtítulo "…"
        imagem "/assets/editor.png"
        grade 4 colunas
            cartão
                ícone "📄"
                etiqueta "NOVO"
                título "Simples de aprender"
                texto "…"
                código "ge run app.ge"
        código "app.ge" "…"

    rodapé
        direitos "Germanio © 2026"
        link "GitHub" "https://github.com/…"
```

## Semantics

- **Words:** `navegação` (or `navbar`), `marca` (or `logo`), `capa` (or `hero`), `selo` (or
  `badge`), `cartão` (or `card`), `etiqueta` (or `tag`), `rodapé` (or `footer`), `direitos` (or
  `copyright`). Every page word is read without accents: `seção` = `secao`, `código` = `codigo`,
  `botão primário` = `botao primario`. The older words keep working.
- **`página "/endereço"`** with an address is a presentation page; `página Clientes` without an
  address stays the intent page.
- **`fundo "/imagem"`** under `capa` or a `seção` puts that image behind it, darkened so the text
  stays readable; **`fundo claro`** makes a section a light band. **`imagem "/imagem"`** shows a
  wide picture after the section's title. **`ponto "…"`** lists short highlights under the
  buttons of the `capa`. **`grade 1|2|3|4 colunas`**.
- **`código` in a section** shows a code panel; a section with only code shows its title beside
  the code (a worked example). Inside a `cartão`, `código` belongs to the card; what is not
  indented under the card belongs to the section.
- **Safety:** an image must be a file of the application itself (an address starting with `/`,
  made of letters, digits, `/`, `.`, `_`, `-`); anything else is ignored, so an address can never
  leave the page's style or attributes. Every text is escaped. Grids never make a phone scroll
  sideways.

## Alternatives studied

- **Free HTML/CSS in the page (`html "…"`, `tema css "…"`):** exists at the primitive level and
  stays there; a landing page should not need it.
- **Making the intent page show text without data:** a different contract (what a page shows);
  left for a GEP of its own if a real case needs presentation blocks inside data pages.

## Limits

No tabs, no images from other sites, no animation choices. The look is the runtime's single
style; there is no per-page theme beyond `tema` colours.

## Tests

`TestPaginaEmPortugues` (Portuguese words with accents, indentation ending a card, every new
line), `TestPaginaDeIntencaoNaoViraEndereco`, `TestPaginaImagemSegura` (unsafe addresses
ignored), `TestPaginaFundosPontosEFaixas` (rendering, escaping).
