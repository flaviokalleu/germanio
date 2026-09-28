# Tooling: um front-end, um binário, um editor

**Data:** 2026-09-28. **Status:** pesquisa, sem força normativa. Fontes:
[typescript.md](typescript.md) (um front-end para compiler e editor; port para Go),
[go.md](go.md) (`go/analysis`, gopls), [zig.md](zig.md) (binário único), [gleam.md](gleam.md)
(`compiler-core` para CLI, LSP e formatter), [nim.md](nim.md) (nimsuggest),
[elixir.md](elixir.md) e [roc.md](roc.md) (LSP tardio), [python.md](python.md) (lib2to3),
[swift.md](swift.md) (dois parsers).

---

## 1. O problema: quantos "conhecimentos da linguagem" existem

Conferido no código da revisão `fc31daf`:

| Consumidor | O que lê o `.ge` | Evidência |
|---|---|---|
| `ge check` | tenta o núcleo estrito (`semantic.Load`); se falhar, `Compilar` (parser de aplicação + resolver); se falhar, o CLI legado | `tooling/gecli/cli.go:201-236` |
| `ge explain <dado>` | `Compilar` → `ast.App` | `cli.go:85-99`, `tooling/explicar` |
| `ge explain pagina`, `ge graph` | varredura por regex (`pagina\s+"…"`, `titulo\s+"…"`) do `intelligence.ProjectAnalyzer` | `tooling/intelligence/analyzer.go:180-181`; lacuna G21 ainda OPEN |
| `ge fmt` | parser do núcleo ou `formatApplication`, com releitura de linhas e pilha própria | `tooling/formatter/intencao.go:127-145` |
| VS Code | gramática TextMate gerada por Python com listas de palavras e regex próprias (`tem`, `pertence a`, lista de começos de linha) | `vscode-germanio/tools/gerar_gramatica.py:39-160` |
| EBNF normativa | cópia em `docs/INTENCAO.md` | `INTENCAO.md:286-310` |
| Tabela de seções | três cópias no próprio parser: `sections`, o `switch` de `sectionOf` e `displaySections` | `compiler/parser/hierarquia.go:127-128, 135, 434-436` |

São pelo menos cinco leituras independentes (parser, formatter, regex de intelligence,
TextMate, EBNF), além das três cópias internas da tabela de seções. O Python mostra o destino
de cópias da gramática: `lib2to3` foi deprecado porque "Python 3.10 may include new language
syntax that is not parsable by lib2to3's LL(1) parser"
([What's New 3.9](https://docs.python.org/3/whatsnew/3.9.html)). O Swift mostra o custo de
dois parsers para a mesma linguagem, mantidos por anos até o ASTGen
([swift.md](swift.md)). O Elixir levou uma década até um LSP oficial
([elixir.md](elixir.md)).

## 2. Princípio: um front-end, um modelo semântico, consultas finas

O TypeScript compartilha scanner, parser, binder e checker entre `tsc` e o language service;
o Gleam usa `compiler-core` para CLI, LSP, formatter e playground; o Zig usa `lib/std/zig/`
para `zig fmt`, `ast-check` e o compilador. Regra para o Germanio: **nenhuma ferramenta
tokeniza, mede indentação ou reconhece seções por conta própria; todas pedem a árvore e os
fatos ao front-end** ([zig.md](zig.md), [typescript.md](typescript.md)).

```text
front-end (compiler/): tokens com span → árvore de linhas sem perdas → fatos → ast.App + origens → []Diagnostic
                                       │
                      workspace: snapshot imutável (arquivos do editor + disco)
                                       │
  consultas: check · explain · fmt · graph · hover · definição · completar · semantic tokens · code actions
                                       │
          ge check / explain / fmt / graph / fix            ge lsp (JSON-RPC por stdio)
                                                                 │
                                                  VS Code: cliente fino + TextMate mínima gerada
```

- `ge check`: os diagnósticos do front-end mais os **analisadores**; `--json` estável.
- `ge explain`: o fato sob consulta com a origem; hover do LSP = o mesmo texto.
- `ge fmt`: imprime a árvore; formatação do LSP = `ge fmt`.
- `ge graph` e `ge explain pagina`: leem `ast.App`; `analyzer.go` sai do caminho de leitura
  (fecha G21).
- completar = seções válidas no contexto (a tabela fechada já é o que um completador precisa).

## 3. `ge check` como driver de analisadores (modelo `go/analysis`)

Em Go, cada regra é um `Analyzer{Name, Doc, Requires, Run(pass)}` que lê o programa já
analisado e reporta diagnósticos com correção sugerida opcional; um driver os ordena por
dependência e os mesmos analisadores servem `go vet`, gopls e ferramentas de terceiros
([analysis](https://pkg.go.dev/golang.org/x/tools/go/analysis), [go.md](go.md)). No Germanio
as regras de contradição, ambiguidade e segurança estão espalhadas entre `compiler/semantic`,
`tooling/intelligence/validator.go` (regex de segredos) e o resolver, e `semantic.Check`
retorna no primeiro erro. Proposta: cada regra vira um analisador sobre `ast.App` com teste
próprio; a pendência de `INTENCAO.md` "cobertura dos diagnósticos de contradição e
ambiguidade em `ge check`" vira "um analisador por regra". Critério do Elixir e do Roc para os
avisos: **só bug verificado**; o que o Germanio "não sabe" vira pergunta ou erro explícito,
nunca aviso incerto ([elixir.md](elixir.md)). EVITAR: linter separado com centenas de regras
configuráveis (Clippy): o leigo não escolhe lints.

## 4. `ge lsp` no mesmo binário

- **LSP padrão, não protocolo próprio.** O `tsserver` precedeu o LSP e ficou com um protocolo
  próprio ([typescript.md](typescript.md)); o Zig considerou protocolo próprio para o build
  (`std.zig.Server`), mas para o editor o LSP basta ([zig.md](zig.md)).
- **No mesmo binário, importando os mesmos pacotes.** O TypeScript 7 foi portado para Go
  justamente para ter paralelismo de memória compartilhada e snapshots imutáveis entre
  requisições; o Germanio já é Go e não precisa de IPC nem de segunda linguagem
  ([typescript.md](typescript.md) › O port para Go). O nimsuggest e o LSP do Roc são camadas
  sobre o compilador, não segundos parsers ([nim.md](nim.md), [roc.md](roc.md)).
- **Pré-requisitos**, nesta ordem: diagnóstico estruturado com span; parser que devolve árvore
  parcial + lista de diagnósticos; `Compilar` aceitando texto do editor (hoje lê do disco,
  `runtime/engine.go`); conversão de coluna em runas para UTF-16 só na borda (como `lsconv`).
- **Code actions vêm das sugestões do diagnóstico**, com a confiança declarada
  ([DIAGNOSTICS.md](DIAGNOSTICS.md) §5), e não são reconstruídas no LSP (o defeito que o estudo
  do Gleam aponta nas code actions dele).
- **Testes**: os mesmos casos de baseline do `ge check`/`explain`/`fmt`, mais marcadores de
  cursor no estilo fourslash para hover, definição e completar
  ([fourslash](https://github.com/microsoft/TypeScript/tree/main/tsc/internal/fourslash)).
- EVITAR por ora: pool de checkers, project references, incremental semântico. Medir o
  `Compilar` do GitLab inteiro antes.

## 5. A gramática do VS Code gerada das tabelas

A TextMate continua útil para cor instantânea, mas deve ser **mínima** (comentários, textos
entre aspas, palavras de seção) e **gerada a partir das tabelas Go** (lexer, `idiomas.go`,
tabela de seções) por um subcomando do próprio `ge`, não por listas em Python. A cor correta
vem dos *semantic tokens* do `ge lsp`, que classificam cada token segundo o parser (dado,
seção, papel, ação, modificador). O Python e o Nim geram a gramática documental a partir da
gramática executável ([python.md](python.md), lição 6; [nim.md](nim.md), lição 2); o Go gera
`go/types` de `types2` e testa a igualdade ([go.md](go.md)). Mínimo imediato, mesmo antes do
LSP: um teste que falhe quando a tabela de seções do parser e a gramática gerada divergirem,
alimentado pela suíte de conformidade (`.ge` → fatos, `.ge` → diagnóstico) de
[sintaxe-e-configuracao.md](sintaxe-e-configuracao.md) §4, no espírito do
[yaml-test-suite](https://github.com/yaml/yaml-test-suite).

## 6. Um binário único

Hoje há dois CLIs:

- `ge` (`cmd/ge/main.go` → `tooling/gecli/cli.go`): `check`, `rodar`, `fmt`, `explain`,
  `explicar`, `graph`, `testar`, `init`, `novo`, `eject`, `legado`.
- `germanio` (`main.go` → `cli/cli.go`): `run`, `check`, `new`, `init`, `build`, `docker`,
  `ide`, `version`. O `main.go:14-16` desvia para o legado quando o executável se chama
  `germanio`; `ge legado <comando>` também chega a ele.

O Zig, o Go e o Gleam têm um binário que é dono de todos os papéis (rodar, checar, formatar,
testar, empacotar, explicar, servir o editor) sobre o mesmo front-end ([zig.md](zig.md),
[gleam.md](gleam.md)). Proposta: `build` e `docker` como subcomandos de primeira classe do
`ge`, o legado aposentado com um prazo de deprecação ([LANGUAGE_EVOLUTION.md](LANGUAGE_EVOLUTION.md)),
e `ge lsp` e `ge fix` nascendo no mesmo binário.

Achado lateral de nomenclatura (conferido): `ge explain <dado>` explica os fatos de um dado
(`cli.go:85`) e `ge explicar <GE0000>` explica um código de diagnóstico (`cli.go:252`). Numa
linguagem cujas palavras são intercambiáveis entre 20 idiomas, dois comandos cujo nome só
difere pelo idioma e cujo significado é diferente contrariam a expectativa criada pela própria
linguagem. Exige decisão (qual nome fica com qual papel), não correção silenciosa.

Achado lateral de robustez (reproduzido): `ge check` num arquivo de intenção válido **sem**
`crie sistema` termina em `panic: nil pointer dereference` em `tooling/gecli/cli.go:225`
(`prog.System.Name` com `prog.System == nil`); com `crie sistema Teste` o mesmo arquivo passa.

## 7. Resumo das lições de tooling

| Classe | Lição | Arquivo afetado |
|---|---|---|
| ADOTAR | um front-end; nenhuma ferramenta lê `.ge` sozinha | `tooling/formatter`, `tooling/intelligence/analyzer.go`, `vscode-germanio/tools/gerar_gramatica.py` |
| ADOTAR | `ge check` como driver de analisadores sobre `ast.App`, todos os diagnósticos de uma vez, `--json` | `compiler/semantic`, `tooling/intelligence/validator.go`, `tooling/gecli/cli.go` |
| ADOTAR | `ge lsp` padrão, no mesmo binário | novo subcomando em `tooling/gecli` |
| ADAPTAR | TextMate mínima gerada das tabelas Go + semantic tokens | `vscode-germanio/`, `compiler/lexer`, `compiler/idiomas` |
| ADOTAR | um binário `ge`; legado com prazo | `main.go`, `cli/cli.go`, `tooling/gecli/cli.go` |
| ADOTAR | baselines para `check`, `explain`, `fmt`, `graph`, e marcadores de cursor para o LSP | novo diretório de casos |
| ADAPTAR | workspace com snapshot imutável e texto não salvo | novo pacote em `tooling/`; `runtime/engine.go` |
| EVITAR | protocolo próprio de editor; formatter com opções dentro do LSP; LSP fora do projeto (Zig) | — |
| INVESTIGAR | `ge reduzir` (minimizar um `.ge` que reproduz um erro, como `zig reduce`), depois do printer sobre a árvore | `tooling/formatter` |
