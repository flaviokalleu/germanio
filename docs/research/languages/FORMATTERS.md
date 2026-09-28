# Formatters: forma canônica, idempotência e impressão pela árvore

**Data:** 2026-09-28. **Status:** pesquisa, sem força normativa. Norma: `docs/INTENCAO.md` ›
Layout (linhas 271-285) e › Testes normativos (`ge fmt`: "idempotente; 4 espaços por nível;
preserva comentários; os fatos antes e depois são idênticos", linha 768). Fontes: seções
"Formatter" de cada estudo deste diretório e [../sintaxe-hierarquica.md](../sintaxe-hierarquica.md).

---

## 1. Os formatters estudados

| Formatter | Opções | Como imprime | Garantia de significado | Papel na evolução | Fonte |
|---|---|---|---|---|---|
| **gofmt** | nenhuma | `go/parser` + `go/printer` sobre a AST com comentários (CommentMap); testes `.input`/`.golden` | não reparseia para comparar; confia no printer | `gofmt -r` e o antigo `gofix` só existem porque entrada e saída são canônicas: "the only changes made to the source code are semantic ones"; `go/format` avisa que o estilo muda e manda fixar a versão | [go.md](go.md); [blog gofmt](https://go.dev/blog/gofmt); [splash](https://go.dev/talks/2012/splash.article); [go/format](https://pkg.go.dev/go/format) |
| **rustfmt** | 87 opções no `Configurations.md` (29 estáveis, 55 instáveis, contagem de 2026-09-28) | sobre a AST | — | *style editions* (RFC 3338): "changes to the default Rust style only appear in style editions", separadas da edition da linguagem | [rust.md](rust.md); [style editions](https://doc.rust-lang.org/nightly/style-guide/editions.html); [RFC 3338](https://rust-lang.github.io/rfcs/3338-style-evolution.html) |
| **zig fmt** | nenhuma de estilo; `--check`, `--stdin`, `--ast-check`; `// zig fmt: off/on` | `Ast/Render.zig`; a AST não guarda comentários, o renderizador os reencontra por offset entre tokens | — | o renderizador aceita *Fixups* e é também o motor de `zig reduce` | [zig.md](zig.md) |
| **mix format** (Elixir) | quase nenhuma; `locals_without_parens` exportável por pacote | sobre a AST; preserva escolhas do usuário (linhas vazias, `do:` × `do/end`) | "nunca mudar a semântica por padrão" | `--migrate` reescreve construções deprecadas e **muda a AST**, com aviso de risco | [elixir.md](elixir.md); [Code](https://elixir.hexdocs.pm/Code.html); [mix format](https://mix.hexdocs.pm/Mix.Tasks.Format.html) |
| **gleam format** | nenhuma; recuo constante 2, largura 80 | pretty printer próprio; comentários numa tabela lateral de posições (`extra.rs`) | — | migrou o `assert` antigo (v0.27) | [gleam.md](gleam.md); [format/src/lib.rs](https://github.com/gleam-lang/gleam/blob/main/format/src/lib.rs); [v0.27](https://gleam.run/news/v0.27-hello-panic-goodbye-try/) |
| **nimpretty** / **nph** | nimpretty oficial e fraco; nph de terceiros, "opinionated" | nph compara a AST antes e depois e deixa o arquivo intacto se divergir; idempotente; sem alinhamento vertical | nph: sim | dois formatters = nenhum normativo | [nim.md](nim.md); [tools](https://nim-lang.org/docs/tools.html); [nph](https://arnetheduck.github.io/nph/) |
| **swift-format** | configurável (`.swift-format` JSON) | sobre swift-syntax; modo formatar e modo lint | — | o README admite: "No default Swift code style guidelines have yet been proposed" | [swift.md](swift.md); [swift-format](https://github.com/swiftlang/swift-format) |
| **Black** (Python, externo) | quase nenhuma | sobre a AST | confere que a AST antes e depois é equivalente; estabilidade de estilo entre versões | Python não tem formatter oficial; anos de debate de estilo | [python.md](python.md); [Black style](https://black.readthedocs.io/en/stable/the_black_code_style/index.html) |
| roc fmt (experimental) | nenhuma | na reescrita, algoritmo de Wadler (largura de linha) | — | — | [roc.md](roc.md) |

## 2. O que converge

1. **Forma canônica única, sem opções.** "Gofmt's style is no one's favorite, yet gofmt is
   everyone's favorite" ([Go proverbs](https://go-proverbs.github.io/)). As histórias de
   rustfmt (dezenas de opções), swift-format (sem estilo oficial) e Nim (dois formatters)
   mostram o custo contrário: cada opção é um formato a mais para testar e um debate a mais.
   Para o Germanio, sem opções é ainda mais importante: o leigo não escolhe estilo.
2. **Imprimir a partir da árvore do parser**, não reler texto. gofmt, zig fmt, mix format,
   gleam format e swift-format fazem isso; é o que torna o formatter também uma ferramenta de
   transformação (`gofmt -r`, Fixups do Zig, `--migrate`, `gleam fix`).
3. **Idempotência `fmt(fmt(p)) == fmt(p)`** como teste sobre todo o corpus (nph a declara;
   Black a pratica).
4. **Verificação de significado** (Black, nph): reparsear a saída e exigir o mesmo
   significado; na divergência, **não tocar no arquivo**.
5. **`--check`** para CI e editor, sem reescrever (zig fmt, gleam format, mix format
   `--check-formatted`).
6. **Versionar o estilo** quando ele mudar (style editions do Rust; `go/format` pede fixar o
   binário), para não quebrar o CI de quem roda `--check`.
7. **O formatter como ferramenta de migração** quando uma forma é aposentada (Gleam, Elixir,
   Go).

## 3. A situação atual do `ge fmt`

Conferido em `tooling/formatter/intencao.go` e `formatter.go`:

- **Acertos, alinhados às referências**: sem opções; 4 espaços por nível; sem espaço no fim;
  no máximo uma linha vazia seguida; comentários preservados e colocados no nível do código
  que precedem; texto entre aspas e aspas triplas intactos; tab no início vira espaço
  (`intencao.go:13-20`). Reparseia a saída e recusa se o programa ficar inválido ou se o
  significado mudar, sem alterar o arquivo (`intencao.go:189-197`): é a garantia do Black e
  do nph. `ge fmt --check` existe (ajuda do `ge`). Há teste de idempotência sobre os arquivos
  do repositório (`intencao_test.go:14-45`).
- **O defeito é arquitetural: o formatter recalcula a pilha de níveis.** `formatApplication`
  relê o texto em `fline` e monta os níveis com **uma segunda pilha própria**
  (`intencao.go:127-145`), independente de `layoutTree` (`compiler/parser/hierarquia.go:34-77`).
  As duas pilhas não são iguais: a do formatter aceita um recuo intermediário como novo nível
  (depois de desempilhar, `l.indent > topo` empilha), enquanto `layoutTree` o recusa como
  "linha sem pai". Hoje isso não causa dano porque o formatter parseia o original antes e
  recusa arquivos inválidos (`intencao.go:122-125`), mas é exatamente o risco "vários parsers":
  a verificação de significado pega divergências de **resultado**, não de **estrutura**.
- **Duas noções de "mesmo significado".** O formatter compara `meaning(ast.Program)` (JSON sem
  posições, **antes** do resolver, `intencao.go:44-60`); os testes de equivalência comparam o
  modelo resolvido. Uma forma normal única de fatos resolveria (Dhall,
  [Safety guarantees](https://docs.dhall-lang.org/discussions/Safety-guarantees.html)).
- **Comentários fora da árvore**: o lexer descarta `#` (`compiler/lexer/lexer.go:495-500`), por
  isso o formatter precisa das linhas cruas.

## 4. Recomendação

| Classe | Lição | Efeito no `ge fmt` |
|---|---|---|
| ADOTAR | manter canônico e sem opções | nada a mudar; registrar como decisão para não reabrir |
| ADAPTAR | imprimir a árvore de linhas sem perdas do parser (recuo cru, tokens, comentário de fim de linha, linhas vazias e comentários como trivia do nó seguinte), no lugar da pilha própria | remove a segunda pilha; `print(parse(x))` com recuo normalizado vira teste direto |
| ADOTAR | manter a verificação de significado como rede, mas sobre a **forma normal de fatos** usada também pelos testes e por `ge explain` | uma só definição de "mesmo significado" |
| ADOTAR | testes `.input`/`.golden` (gofmt) além da idempotência sobre o corpus | regressões de estilo ficam visíveis |
| ADOTAR | recusar arquivo com erro de layout em vez de escolher um nível | já é o comportamento; escrever como regra |
| ADAPTAR | versionar o estilo quando mudar, junto com a versão da linguagem declarada no projeto (sem opção) | CI com `--check` não quebra por atualização do `ge` |
| ADAPTAR | `ge fmt` (ou `ge fix`) como migração de formas aposentadas, reparseando e provando que os fatos são os da forma nova | dá ao formatter um papel no processo de evolução ([LANGUAGE_EVOLUTION.md](LANGUAGE_EVOLUTION.md)) |
| EVITAR | formatter configurável; opções por pacote (`locals_without_parens`); dois formatters | — |
| EVITAR por ora | algoritmo de Wadler / largura de linha | o `.ge` é uma linha lógica por item e não quebra linhas |
| EVITAR | converter entre forma plana e hierárquica | a norma já diz que o formatter não converte (`INTENCAO.md:238-239`) |
