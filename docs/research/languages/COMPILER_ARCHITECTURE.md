# Arquitetura do compilador: comparação e recomendação

**Data:** 2026-09-28. **Status:** pesquisa, sem força normativa. Fontes: os estudos por
linguagem deste diretório, em especial [sintaxe-e-configuracao.md](sintaxe-e-configuracao.md)
§10-11, [typescript.md](typescript.md) › Arquitetura recomendada, [go.md](go.md),
[zig.md](zig.md), [rust.md](rust.md), [swift.md](swift.md) e [gleam.md](gleam.md). Código do
Germanio citado pela revisão `fc31daf`.

---

## 1. O pipeline do Germanio hoje

```text
texto .ge
  → lexer (compiler/lexer/lexer.go): Token{Type, Value, Line, Column, Indent, Raw}; TokenIndent por linha;
    comentários descartados (lexer.go:495-500); sem posição de fim (lexer.go:214-224)
  → blockLines / blockLinesWithHeader (declarativo.go:22, intencao.go:454): linhas até a próxima coluna 1
  → layoutTree (hierarquia.go:34-77): árvore de linhas `node`, com perdas
  → dataSection / access / rule (hierarquia.go:275-432): sintetiza tokens de frase plana
  → intentFrom (intencao.go): parser de frase plana → ast.Intent (Pos + Context)
  → ResolveIntent (resolver.go; chamado em runtime/engine.go:551) → ast.App
       (entidades, grants, estados, capabilities); funde blocos e acusa conflitos
formatter (tooling/formatter/intencao.go): texto → fline + pilha própria → texto; confere meaning(ast.Program)
ge graph / ge explain pagina: tooling/intelligence/analyzer.go, varredura por regex (analyzer.go:180-181)
VS Code: gramática TextMate gerada por vscode-germanio/tools/gerar_gramatica.py com listas próprias
núcleo estrito (SPEC.md): outro front-end, compiler/parser/germanio.go + compiler/semantic
```

Pontos fortes que a pesquisa confirma: a fronteira por arquivo (`ast.Intent`) × por projeto
(`ast.App`) já existe; o modelo resolvido é único para runtime e `ge explain`; o formatter
verifica significado por reparse, o que o gofmt não faz e o Black faz.

Pontos fracos: posições sem fim; árvore de linhas com perdas; frase sintetizada como IR
escondida; `error` em vez de lista de diagnósticos; vários leitores da sintaxe; dois
front-ends.

## 2. Arquiteturas comparadas

| Projeto | Representações | Um front-end para tudo? | Tolerância a erro | Lição para o Germanio |
|---|---|---|---|---|
| Go | `go/token` (Pos inteiro + FileSet), `go/ast` com comentários, `go/types`; `types2` no compilador gerado de `go/types` com teste de igualdade | sim: gofmt, vet, gopls usam `go/*` | parcial | posições compactas com fim; quando houver duas implementações, gere uma da outra e teste ([go.md](go.md); [go/token](https://pkg.go.dev/go/token)) |
| Zig | tokens → `Ast` (sem comentários; renderizador os reencontra por offset) → ZIR (por arquivo) → Sema → AIR | sim: `lib/std/zig/` serve `zig fmt`, `ast-check`, compilador | `ErrorBundle` acumula todos os erros | checagem por arquivo antes da do projeto; coletor único de diagnósticos ([zig.md](zig.md)) |
| Rust (rustc) | tokens → AST → HIR → THIR → MIR; queries incrementais | não: rust-analyzer é um segundo front-end | `ErrorGuaranteed` evita cascata | diagnóstico estruturado; **não** copiar HIR/MIR/queries ([rust.md](rust.md); [overview](https://rustc-dev-guide.rust-lang.org/overview.html)) |
| rust-analyzer | green/red tree sem perdas, AST tipada como vista, `hir` resolvido | é o front-end do editor | parser nunca falha, nó ERROR | árvore sem perdas; o resto é escala que o Germanio não tem ([syntax](https://github.com/rust-lang/rust-analyzer/blob/master/docs/book/src/contributing/syntax.md)) |
| Swift | parser C++ legado + swift-syntax (Swift), ASTGen fazendo a ponte anos depois; request evaluator | não, por anos | swift-syntax: *missing*/*unexpected* na árvore | o custo de dois parsers para a mesma linguagem ([swift.md](swift.md)) |
| TypeScript | scanner → parser → binder → checker, compartilhados por `tsc` e language service; 7.0 portado para Go | sim | `ParseSourceFile` sem `error`, nós ausentes | um front-end, um modelo semântico, consultas finas; o port em Go valida-se por baselines ([typescript.md](typescript.md)) |
| Gleam | `compiler-core` para CLI, LSP, formatter, playground; erros como enums com fatos | sim | por definição ([v1.2](https://gleam.run/news/fault-tolerant-gleam/)) | um binário, um front-end, erros como dados ([gleam.md](gleam.md)) |
| Python | `python.gram` (PEG) gera o parser; ASDL descreve a AST; 3.9 trocou o parser sem mudar a linguagem | sim para CPython; lib2to3 (cópia da gramática) morreu | segunda passada com regras `invalid_*` | fonte única da gramática; trocar front-end provando AST igual ([python.md](python.md); [PEP 617](https://peps.python.org/pep-0617/)) |
| Nim | `grammar.txt` gerado de `parser.nim`; NIF com proveniência | parcial | — | proveniência dentro da IR ([nim.md](nim.md)) |
| HCL | modelo de informação + sintaxes (nativa, JSON) + body schema da aplicação | sim | conteúdo parcial + diagnósticos | schema de seções como dado ([sintaxe-e-configuracao.md](sintaxe-e-configuracao.md) §3) |

O padrão comum às ferramentas que envelheceram bem: **um front-end, um modelo semântico,
várias consultas finas**; nenhuma ferramenta lê o texto por conta própria. O padrão comum às
que sofreram: cópias da gramática (lib2to3, dois parsers do Swift, TextMate à mão).

## 3. A arquitetura recomendada: quatro representações

```text
texto .ge
  │
  ▼ (1) TOKENS com início e fim
  │     Token{Type, Value, Raw, Start, End}  (Start/End = offset em bytes; linha e coluna calculadas por um FileSet)
  │     comentário de fim de linha preservado como trivia
  ▼ (2) ÁRVORE DE LINHAS SEM PERDAS (uma só, por arquivo)
  │     Linha{recuo cru, tokens, comentário, trivia antes (linhas vazias e de comentário), filhos, erro?}
  │     construída por layoutTree; consumida pelo parser, impressa pelo formatter, lida pelo LSP
  ▼ (3) FATOS (ast.Intent), gerados direto da árvore pelo schema de seções
  │     Field, Relation, InitialState, Capability, AccessPolicy (hoje Grant), Rule, IntegrationName, Hook
  │     cada fato com Origem{arquivo, span, caminho hierárquico, id do nó}
  ▼ (4) MODELO RESOLVIDO (ast.App) + MAPA DE ORIGEM
        fusão comutativa/associativa/idempotente dos fatos de todos os arquivos;
        cada elemento resolvido aponta para os fatos (e portanto os spans) que o justificam
  │
  └─ diagnósticos estruturados em todas as etapas: []Diagnostic (código, span primário, labels, sugestões)
```

Por que exatamente estas quatro:

1. **Tokens com span**: sem fim não há o que sublinhar, substituir (sugestão, LSP) nem
   recortar (formatter). `go/token` mostra que um `Pos` inteiro + `FileSet` é barato e pode até
   ser reutilizado da biblioteca padrão ([go.md](go.md)). A conversão para UTF-16 só na borda
   do LSP, como `lsconv` do TypeScript ([typescript.md](typescript.md)), porque nomes com
   acento a tornam obrigatória.
2. **Árvore de linhas sem perdas, única**: elimina as quatro leituras da estrutura (seção 1
   de [SYNTAX_COMPARISON.md](SYNTAX_COMPARISON.md) §9). Sem perdas **de linha**, não de token:
   o `.ge` não tem expressões multilinha, e o formatter já normaliza espaços dentro da linha.
3. **Fatos**: é a "AST semântica" do pipeline desejado. Gerar direto da árvore, por
   denotação, remove a frase sintetizada como IR escondida e com ela os bugs de posição e de
   diagnóstico na frase errada. A frase plana passa a ser **renderização** do fato.
4. **Modelo resolvido + mapa de origem**: já existe (`ast.App`); falta que cada elemento
   resolvido aponte para as origens, o que dá conflito com duas origens, `ge explain` completo
   e "ir para definição". É o *blame* do Nickel
   ([contracts](https://nickel-lang.org/user-manual/contracts)) e a proveniência "for free"
   da NIF do Nim ([nim.md](nim.md), lição 10).

O que **não** entra, e por quê (cada item pede um problema que o Germanio não tem hoje):
AST tipada sobre o CST, HIR separado, MIR/IR de otimização, red-green trees, reparse
incremental, queries/request evaluator, project references. Reavaliar só com medição do
`Compilar` no maior app (GitLab) e com o LSP funcionando.

## 4. Os dois front-ends

O núcleo estrito (`SPEC.md`, `compiler/parser/germanio.go`, `compiler/semantic`) e o dialeto
de aplicação (`parser.go` + `hierarquia.go` + `resolver.go`) têm lexer em modo diferente
(`lexer.go:458-465`), convenções de layout diferentes (GE1002 "dois espaços" contra 4
canônicos) e formatos de erro diferentes (`diagnostics.Diagnostic` com código contra
`fmt.Errorf`). O `ge check` tenta um, cai no outro e depois no CLI legado
(`tooling/gecli/cli.go:201-236`).

Roteiro que o estudo do Python sugere (PEP 617): **trocar o front-end sem mudar a
linguagem**, provando AST igual em todo o corpus e com chave de escape por uma versão; só
depois remover o antigo ([python.md](python.md), lição 8). O Swift mostra o custo de não
fazer isso: dois parsers por anos ([swift.md](swift.md)). Unificar os dois front-ends é
INVESTIGAR (exige decisão sobre o núcleo estrito, que é normativo por `SPEC.md`); o mínimo
imediato é compartilhar a camada (1)-(2) e um só formato de diagnóstico.

## 5. Caminho incremental a partir do código atual

Cada passo é útil sozinho, não muda nenhum fato válido e é verificado pelos testes existentes
(`TestHierarquiaEquivaleAFrasePlana`, `TestHierarquiaFusao`, os testes do formatter e os
testes de execução do GitLab).

| # | Passo | Arquivos | Corrige |
|---|---|---|---|
| 1 | Escrever o schema de seções como tabela de dados (papel de cada nível, cardinalidade, filhos proibidos) e fazer a árvore validar filhos proibidos | `compiler/parser/hierarquia.go` | D1 (linhas ignoradas ou achatadas) |
| 2 | Classificar o bloco pela estrutura (coluna 1, só palavras, com filhos), não pela primeira seção; seção desconhecida vira o erro de seção de sempre | `hierarquia.go:150-198` | D4 |
| 3 | `Diagnostic` estruturado com span, labels, sugestões e `Related`; `teach`, `errorf` e `errAt` passam a construí-lo; renderização de texto igual à atual | `compiler/diagnostics/diagnostics.go`, `parser.go:32`, `hierarquia.go:93`, `resolver.go:88` | base para D2, D3 |
| 4 | `Start`/`End` nos tokens; `Position` com fim; id do nó da árvore guardado no fato | `compiler/lexer/lexer.go`, `compiler/diagnostics`, `compiler/ast/intencao.go` | origem na linha errada (D10) |
| 5 | Fusão de campos idempotente; conflito escalar com as duas origens; teste de propriedade (permutar e duplicar blocos e arquivos) | `compiler/parser/resolver.go:250-264` | D2 |
| 6 | Por seção, trocar "sintetiza frase e chama `intentFrom`" por "constrói o fato" (`&ast.Grant{Role, Verb, Target, Only, Own, Context, Pos}`), começando por `acesso`; o teste de equivalência prova que nada mudou | `hierarquia.go:392-432` e seguintes | D3 |
| 7 | Frase plana gerada **a partir do fato** (função fato → texto) para `ge explain` e para "Equivale a:" | `tooling/explicar`, `compiler/ast` | "Equivale a" em todo erro de bloco |
| 8 | Verbos de várias palavras do léxico da capability, não de `if` no parser | `hierarquia.go:404,419`, `intencao.go:257,697`, `resolver.go:1031,1082` | D5 |
| 9 | Comentários e linhas vazias como trivia na árvore; o formatter imprime a árvore e mantém a checagem de significado como rede | `lexer.go`, `hierarquia.go`, `tooling/formatter/intencao.go` | pilha duplicada do formatter |
| 10 | `[]Diagnostic` em vez de `error` no parser e no resolver, com a política de recuperação de [SYNTAX_COMPARISON.md](SYNTAX_COMPARISON.md) §8 | `parser.go`, `hierarquia.go`, `resolver.go` | um erro por execução |
| 11 | `ge graph` e `ge explain pagina` lendo `ast.App`; `analyzer.go` sai do caminho de leitura | `tooling/intelligence/analyzer.go`, `tooling/gecli/cli.go` | G21 |
| 12 | Workspace (snapshot imutável por versão, texto do editor ou do disco) e `ge lsp` | novo pacote em `tooling/`; `runtime/engine.go` (`Compilar` lê do disco) | ausência de LSP |

A ordem 1-2-3 vem primeiro porque corrige violações da norma com o menor risco; 6-7 são o
coração da mudança (a frase sintetizada sai do caminho); 10-12 são pré-requisito do editor.

## 6. Critério de desempenho

Pelo Performance Gate de `AGENTS.md`, os passos 4, 6 e 10 tocam o caminho quente do
`Compilar`. Medir antes e depois com `bench/` no GitLab inteiro. A pesquisa não encontrou
nenhuma razão para esperar regressão (menos reparse, não mais), mas isso é hipótese, não
medição.
