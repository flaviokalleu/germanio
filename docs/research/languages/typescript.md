# TypeScript: um compiler que também é o language server

- **Data:** 2026-09-28
- **Status:** pesquisa concluída; sem força normativa (ver `docs/README.md`). A arquitetura
  recomendada no fim é proposta, não implementação.
- **Pergunta principal:** como compiler e language server compartilham o conhecimento da
  linguagem? E como aplicar isso a `ge check`, `ge explain`, `ge fmt`, `ge graph`, um futuro
  LSP e o VS Code?

## Fontes consultadas

- Repositório: https://github.com/microsoft/TypeScript. Em 2026-09-28 o `main` contém o
  código nativo em Go em `tsc/` (listado pela API do GitHub: `tsc/internal/{scanner, parser,
  ast, binder, checker, compiler, transformers, printer, diagnostics, ls, lsp, project, format,
  fourslash, api, ...}`, `tsc/testdata/tests/cases/{compiler,conformance,transpile}`,
  `tsc/testdata/baselines/reference/`), e há ramos `release-6.0` e `release-7.0`.
  O layout clássico em JavaScript (`src/compiler`, `src/services`, `src/server`) pertence ao
  6.x e anteriores; não o reverifiquei arquivo por arquivo nesta pesquisa.
- CONTRIBUTING (testes e baselines): https://github.com/microsoft/TypeScript/blob/main/CONTRIBUTING.md
- Parser nativo: https://github.com/microsoft/TypeScript/blob/main/tsc/internal/parser/parser.go
- Diagnósticos nativos: https://github.com/microsoft/TypeScript/tree/main/tsc/internal/diagnostics
  (`diagnosticMessages.json` com 2213 entradas `code` em 2026-09-28, e `diagnostics_generated.go`)
- `Diagnostic` com `relatedInformation`: https://github.com/microsoft/TypeScript/blob/main/tsc/internal/ast/diagnostic.go
- Snapshots e pool de checkers do LSP: https://github.com/microsoft/TypeScript/blob/main/tsc/internal/project/snapshot.go ,
  https://github.com/microsoft/TypeScript/blob/main/tsc/internal/project/checkerpool.go
- Conversão de posições UTF-16: https://github.com/microsoft/TypeScript/blob/main/tsc/internal/ls/lsconv/converters.go
- Fourslash (testes do language service): https://github.com/microsoft/TypeScript/tree/main/tsc/internal/fourslash
- TypeScript Compiler Notes (mantenedores): https://github.com/microsoft/TypeScript-Compiler-Notes
  (README, GLOSSARY, `codebase/src/compiler/parser.md`, `systems/codefixes.md`)
- Wiki: Architectural Overview https://github.com/microsoft/TypeScript/wiki/Architectural-Overview ;
  Standalone Server https://github.com/microsoft/TypeScript/wiki/Standalone-Server-(tsserver) ;
  Design Goals https://github.com/microsoft/TypeScript/wiki/TypeScript-Design-Goals
- Project references: https://www.typescriptlang.org/docs/handbook/project-references.html ;
  `incremental`: https://www.typescriptlang.org/tsconfig/#incremental
- Port nativo: https://devblogs.microsoft.com/typescript/typescript-native-port/ (mar/2025);
  https://devblogs.microsoft.com/typescript/announcing-typescript-native-previews/ ;
  https://devblogs.microsoft.com/typescript/progress-on-typescript-7-december-2025/ ;
  https://devblogs.microsoft.com/typescript/announcing-typescript-7-0/ (8/jul/2026);
  "Why Go?": https://github.com/microsoft/typescript-go/discussions/411 ;
  README do typescript-go: https://github.com/microsoft/typescript-go

Arquivos do Germanio lidos: `tooling/gecli/cli.go`, `tooling/intelligence/analyzer.go`,
`tooling/formatter/formatter.go`, `tooling/formatter/intencao.go`, `compiler/lexer/lexer.go`,
`compiler/parser/parser.go`, `compiler/parser/hierarquia.go`, `compiler/parser/germanio.go`,
`compiler/diagnostics/diagnostics.go`, `runtime/engine.go` (`Compilar`),
`vscode-germanio/tools/gerar_gramatica.py`, `vscode-germanio/package.json`.

---

## Matriz

### Objetivo original
JavaScript com tipos opcionais para aplicações grandes, compilando para JavaScript legível. Os
Design Goals dizem "Statically identify constructs that are likely to be errors" e "Provide a
structuring mechanism for larger pieces of code". Desde o início o **editor** é um cliente de
primeira classe: o language service foi publicado junto com o compiler.

### Filosofia
Design Goals e Non-goals explícitos (wiki). Três relevantes ao Germanio: "Avoid adding
expression-level syntax" (goal 8: sintaxe nova é cara); "Do not cause substantial breaking
changes from TypeScript 1.0" (goal 11); e o non-goal "Apply a sound or 'provably correct'
type system" (utilidade acima de pureza). Também o non-goal 7: "Introduce behaviour that is
likely to surprise users".

### Sintaxe
Superconjunto de JavaScript; a sintaxe de expressão é do TC39, não do TypeScript. Não se
aplica ao Germanio.

### Gramática
Não há gramática formal publicada como fonte única; a especificação antiga foi abandonada e o
parser escrito à mão é a referência de fato (não verificado nesta pesquisa além da ausência de
spec atual no repositório). Contraste: o Germanio tem `INTENCAO.md` normativo, o que é
melhor; o risco é o código divergir dele.

### Lexer/tokenizer
`scanner`: produz tokens sob demanda para o parser e é reutilizado por outras partes
(formatação, classificação). Conceitos de "full start" (inclui trivia anterior) e "token
start" (Compiler Notes, Architectural Overview) permitem que a mesma árvore sirva ao
compiler e a ferramentas que precisam de comentários e espaços sem que a árvore armazene
trivia como nós.

### Parser
Recursivo descendente escrito à mão; o `parser.go` nativo tem 6854 linhas. Propriedades
decisivas para o editor, verificadas no código Go:

- `ParseSourceFile(...) *ast.SourceFile` **não devolve erro**: sempre há uma árvore; os erros
  de sintaxe ficam como diagnósticos do arquivo.
- Recuperação por **contextos de lista** (`ParsingContext`, `parseList`,
  `abortParsingListOrMoveToNextToken`, `isInSomeParsingContext`): diante de um token
  inesperado, o parser pergunta se ele inicia um elemento ou fecha **alguma** lista aberta;
  se sim, encerra a lista atual; se não, pula o token e registra o erro.
- Nós ausentes (`createMissingIdentifier`, `createMissingList`) e a flag
  `NodeFlagsThisNodeHasError` marcam a árvore parcial.
- Reparse incremental reutiliza nós da versão anterior (Compiler Notes, "IncrementalParser",
  "syntax cursor"; não verifiquei se o port nativo manteve isso).

### AST
`Node` com `kind`, `pos`/`end` (intervalo, não só início) e pai; `SourceFile` guarda texto,
mapa de linhas e diagnósticos de parse (GLOSSARY). A AST é tratada como imutável depois de
criada; informação semântica fica fora dela (símbolos, tipos, *links* do checker), o que
permite reutilizar a mesma `SourceFile` entre programas.

### Representações intermediárias
Não há IR de otimização; há transformações AST → AST (`transformers`) e o `printer`/`emitter`.
Não se aplica ao Germanio.

### Análise semântica
Separada em duas etapas com responsabilidades nítidas:

- **Binder**: percorre a árvore, cria um `Symbol` por declaração nomeada e as tabelas de
  escopo; também monta o grafo de fluxo de controle (GLOSSARY: "Linking declarations
  contributing to the same structure using a Symbol").
- **Checker**: une símbolos entre arquivos, resolve tipos **sob demanda** e produz
  diagnósticos semânticos (README do Compiler Notes: "lazily computes type relationships and
  semantic diagnostics").

O `Program` ("a collection of SourceFiles and a set of compilation options", GLOSSARY) é a
unidade semântica. **O language service não tem outro modelo**: ele pede ao mesmo `Program`
e ao mesmo checker o tipo em uma posição, os símbolos, as referências.

### Sistema de tipos
Estrutural, apagável, deliberadamente não sólido. Não se aplica.

### Type inference
Local e contextual, com análise de fluxo de controle para estreitamento. Não se aplica
diretamente; a lição transferível é "sob demanda": o checker não calcula tudo, calcula o que
alguém pergunta, e isso serve tanto a `tsc` quanto a um hover.

### Compiler/interpreter
Pipeline: pré-processamento (resolução de arquivos) → parse → bind → `Program` → check →
emit (README do Compiler Notes). No 7.0 o check roda em paralelo com `--checkers` (padrão 4)
e os projetos referenciados com `--builders` (anúncio do 7.0).

### Runtime
Não há runtime próprio (JavaScript). Não se aplica.

### Memory management
No 6.x, GC do Node; no 7.0, GC do Go. O anúncio do port cita memória "approximately half"
(mar/2025); o do 7.0 mede redução de 6% a 26% nos projetos testados. A discussão "Why Go?"
cita "excellent control of memory layout and allocation without requiring that the entire
codebase continually concern itself with memory management".

### Standard library
Arquivos `lib.*.d.ts` com as declarações do ambiente. Não se aplica.

### Package manager
Usa npm; resolução de módulos é parte do compiler. Não se aplica.

### Formatter
O formatter é **parte do language service** (`tsc/internal/format`, `ls/format.go`), baseado
em regras sobre pares de tokens e na mesma árvore. Não é canônico (tem opções). O ecossistema
adotou formatadores externos, fora do escopo aqui.

### Linter
Não faz parte (ESLint é externo). Os diagnósticos "suggestion" e `reportsUnnecessary` /
`reportsDeprecated` (campos do `Diagnostic` nativo) cobrem parte do espaço de lint dentro do
próprio compiler, e o editor os mostra como texto esmaecido ou riscado.

### Language server
Duas gerações:

- **tsserver (até 6.x)**: executável Node que embrulha compiler e language service e fala
  um protocolo JSON próprio por stdin/stdout ("tsserver listens on stdin and writes messages
  back to stdout"); gerencia projetos *configured* (tsconfig), *external* e *inferred*
  (arquivo solto) (wiki Standalone Server).
- **LSP nativo (7.0)**: "a new foundation based on the Language Server Protocol (LSP)" com
  multithreading; o protocolo próprio foi abandonado e "some behavior specific to the
  TypeScript VS Code Extension may have changed" (Progress on TypeScript 7). O código mostra a
  estrutura: `lsp/server.go` (JSON-RPC), `project/session.go` e `project/snapshot.go`
  (**snapshots imutáveis** do workspace: "Immutable state, cloned between snapshots", com
  `overlayfs` para o texto não salvo do editor), `project/checkerpool.go` (um checker
  dedicado a diagnósticos "providing consistent walk order", checkers temporários para
  consultas e um para a API) e `ls/` (hover, completions, definition, references, rename,
  semantic tokens, inlay hints, code actions, format).

Divisão de trabalho declarada: o **checker foi portado** quase sem mudar comportamento; o
**language service foi fortemente reescrito** ("much of the code that powers completions,
hover tooltips, navigation, and more, has been heavily rewritten"). Ou seja: o conhecimento
da linguagem mora no compiler; o LS é uma camada fina de consultas e conversões.

### IDE tooling
A extensão do VS Code é cliente do servidor. Colorização: gramática TextMate
(mantida em repositório próprio, não verificado aqui) para cor imediata, e **semantic
tokens** do servidor (`ls/semantictokens.go`) para cor correta. `lsconv/converters.go`
converte offsets internos para posições UTF-16 do LSP (`unicode/utf16`).

### Diagnostics
- Catálogo único `diagnosticMessages.json`: chave = texto da mensagem com `{0}`, valor =
  `category` e `code` (ex.: "Unterminated string literal." → Error 1002). Um gerador produz
  `diagnostics_generated.go` e a localização (`loc_generated.go`). O código é estável e é o
  que as pessoas procuram (`TS2322`).
- Categorias: Error, Warning, Suggestion, Message.
- `relatedInformation []*Diagnostic`: um erro aponta para outros locais (a outra declaração,
  a origem do tipo esperado). É exatamente a forma de "valor único com dois valores é erro
  com as duas origens" (`INTENCAO.md` › Fusão).
- **Code fixes** são indexados pelo código do erro: "they bubble through TypeScript's
  internals initially from a compiler error", o que permite pulá-los "cheaply" quando o código
  não está presente; `fixId` + `getCombinedCodeFix` aplica a mesma correção em todo o projeto
  (Compiler Notes, `systems/codefixes.md`). Refactors são separados e não dependem de erro.

### Testing
- `tests/cases/{compiler,conformance,...}`: um arquivo de entrada por caso; o harness grava
  **baselines** (diagnósticos, tipos, símbolos, JS emitido) em `baselines/local/`, comparadas
  com `baselines/reference/`; aceitar a mudança é um ato explícito (`hereby baseline-accept`)
  que aparece no diff do pull request (CONTRIBUTING).
- **Fourslash**: arquivos com marcadores `/*nome*/` e chamadas ao language service naquele
  ponto; o resultado também vira baseline.
- A suíte foi o instrumento do port: "around 20,000 compiler test cases, of which about 6,000
  produce at least one error in TypeScript 6.0. In all but 74 cases, TypeScript 7 also
  produces at least one error" (Progress on TypeScript 7).

### Documentation
Handbook para usuários, wiki e Compiler Notes para contribuidores ("not official
documentation"), release notes por versão. Sem especificação normativa atual.

### Evolution process
Sem RFC formal: issues com a etiqueta de sugestão, notas públicas de *design meetings*,
*iteration plans* e release notes (não verificado em detalhe nesta pesquisa). O filtro real
são os Design Goals, e a sintaxe de expressão é delegada ao TC39. Para o processo formal, ver
`swift.md`.

### Backward compatibility
Goal 11 ("no substantial breaking changes from 1.0"), com exceções encenadas: o 6.0 depreciou
opções; o 7.0 "provides hard errors in the face of any flags and constructs deprecated in
TypeScript 6.0" (anúncio do 7.0). O 7.0 saiu **sem API programática** ("TypeScript 7.1 to ship
with a new (and different) API"): a API pública é tratada como contrato separado do
compiler, que pode quebrar numa reescrita.

### Principais acertos
- **Um só front-end** (scanner, parser, binder, checker) para compilar e para o editor.
- Parser que sempre devolve árvore; diagnósticos como dados.
- Checker sob demanda: a mesma pergunta serve a `tsc` e a um hover.
- Catálogo de diagnósticos com códigos estáveis, gerado; code fixes presos ao código;
  related information.
- Baselines: toda mudança de comportamento observável aparece como diff revisável.
- Port nativo conduzido como *port*, medido por baselines, com resultado de ~10x.

### Principais problemas
- O tsserver com protocolo próprio envelheceu e precisou ser substituído pelo LSP,
  quebrando comportamento do cliente principal.
- Monolito de um único thread (6.x): desempenho limitou editor e build em projetos grandes,
  motivando o port inteiro.
- Checker gigantesco (`tsc/internal/checker/checker.go` tem cerca de 1,4 MB de fonte em
  2026-09-28, pela API do GitHub) e sem especificação: o comportamento é
  definido pelas baselines.
- Duas implementações simultâneas durante a transição (6.x JS e 7.x Go), com 74 casos de
  divergência de erro documentados.

### Complexidade acumulada
Alta e conhecida: o sistema de tipos de JavaScript real é o motivo. Para o Germanio, o dado
útil é arquitetural: mesmo com essa complexidade, o **número de front-ends** permaneceu um.

### O que Germanio pode aprender
Ver "Arquitetura recomendada" e "Para o Germanio".

### O que Germanio NÃO deve copiar
- Protocolo próprio de editor (tsserver): começar direto em LSP, como o 7.0 terminou fazendo.
- Formatter configurável dentro do LS.
- Sistema de tipos não sólido e sem especificação: o Germanio tem norma; as baselines devem
  **confirmar** `INTENCAO.md`, não substituí-la.
- Project references, `composite`, `.tsbuildinfo` e `--builders`: resolvem monorepos de
  milhões de linhas; um app `.ge` do tamanho do GitLab é resolvido em uma passada. Registrar
  como INVESTIGAR só se medições mostrarem necessidade.
- Pool de checkers e paralelismo: o mesmo argumento; primeiro medir.

---

## O port para Go: por que e como (relevante porque o Germanio já é Go)

- **Por quê**: escala. Editor e build em projetos grandes; números do anúncio (VS Code 1,5 M
  linhas: 77,8 s → 7,5 s; carregamento no editor 9,6 s → 1,2 s) e a necessidade de
  "refactoring, navigation, and AI-powered features that require extensive semantic
  information with tight latency constraints" (typescript-native-port).
- **Por que Go**: "Idiomatic Go strongly resembles the existing coding patterns of the
  TypeScript codebase"; percursos polimórficos de árvore em várias direções são ergonômicos;
  GC sem abrir mão do controle de layout; Rust exigiria reestruturar por causa do modelo de
  posse, incompatível com um port (Why Go?, discussão #411). Infraestrutura auxiliar em Rust
  (`libsyncrpc`) e o *watcher* portado de C++ com "minimal assembly shims".
- **Como**: port, não reescrita, "maintaining the structure and logic of the original codebase
  to keep results consistent and compatible" (anúncio do 7.0); exceções deliberadas
  (checagem de JavaScript e o language service reescritos; *declaration emit* "differs
  greatly, intentionally", com `CHANGES.md`); validação pelas baselines.
- **O que o Go permitiu que o Node não permitia**: paralelismo de memória compartilhada
  (checkers e builders), snapshots imutáveis compartilhados entre requisições do LSP.
- **Para o Germanio**: o Germanio não precisa portar nada; já tem o que o TypeScript foi
  buscar. O LSP do Germanio pode ser um subcomando do mesmo binário (`ge lsp`) importando os
  mesmos pacotes de `compiler/` e `tooling/`, sem IPC nem segunda linguagem. O único
  componente fora de Go é a extensão do VS Code (cliente fino) e o gerador Python da gramática
  TextMate, que deve deixar de conter conhecimento próprio (ver abaixo).

---

## Diagnóstico do Germanio hoje: quantos "conhecimentos da linguagem" existem

Verificado no código em 2026-09-28:

| Consumidor | O que lê o `.ge` | Evidência |
|---|---|---|
| `ge check` | tenta `semantic.Load` (núcleo estrito); se falhar, `germanioRuntime.Compilar` (lexer + parser de aplicação + imports + `ResolveIntent`); se falhar de novo, cai no CLI legado | `tooling/gecli/cli.go`, caso `"rodar", "check"` |
| `ge explain <dado>` | `Compilar` → `ast.App` | `tooling/gecli/cli.go`, `tooling/explicar` |
| `ge explain pagina` e `ge graph` | `intelligence.ProjectAnalyzer`: varredura por linhas e regex que reconhece `tabela `/`modelo `, `pagina "…"`, `botao …`, `quando receber …` | `tooling/intelligence/analyzer.go` (`extractTablesFromSource`, `extractPagesFromSource`, `extractAPIsFromSource`) |
| `ge fmt` | parser do núcleo estrito; se não for núcleo, `formatApplication`: lexer + parser de aplicação para conferir o significado, mais um re-leitor de linhas próprio para preservar comentários | `tooling/formatter/formatter.go`, `tooling/formatter/intencao.go` |
| VS Code | gramática TextMate gerada por `gerar_gramatica.py`, com listas de palavras escritas à mão em Python (tipos, modificadores, controle) | `vscode-germanio/tools/gerar_gramatica.py` |

Consequências concretas:

1. `ge graph` e `ge explain pagina` não conhecem a sintaxe de intenção nem a hierárquica: o
   analisador procura `tabela`/`modelo`, que não aparecem em `projetos` › `tem`. É o G21
   ("tooling lê o modelo de intelligence") ainda OPEN em `GERMANIO_GAPS.md`.
2. As palavras do VS Code são uma cópia manual do léxico; uma palavra nova no lexer ou em
   `idiomas.go` não colore até alguém editar o Python.
3. O parser de aplicação para no **primeiro** erro e o devolve como texto (`errorf` →
   `fmt.Errorf("%s:%d:%d: %s")`; `teach` concatena "Onde/Por quê/Como corrigir" numa string).
   Só o núcleo estrito usa `diagnostics.Diagnostic` com código. Um LSP não consegue extrair
   código, intervalo nem "como corrigir" de forma estruturada.
4. Posições só têm início (`Line`, `Column`; `Token` sem fim), e o lexer descarta comentários
   (`#`): não há intervalo para sublinhar, nem árvore sem perda para o formatter, que por isso
   relê as linhas cruas.
5. Dois front-ends com convenções diferentes (núcleo: "dois espaços por nível" em GE1002;
   aplicação: 4 canônicos).

## Arquitetura recomendada

Princípio (do TypeScript): **um front-end, um modelo semântico, várias consultas finas**.
Nenhuma ferramenta lê `.ge` por conta própria.

```text
texto .ge ──► sintaxe ──► árvore de layout (sem perda) ──► frases (Intent) ──► resolver ──► ast.App + proveniência
                 │                     │                         │                  │
                 └────────── diagnósticos estruturados (código, intervalo, 4 partes, origens relacionadas) ──┘
                                             │
                          workspace (snapshot imutável: arquivos abertos + disco)
                                             │
   consultas: check · explain · fmt · graph · hover · definição · referências · completar · semantic tokens · code actions
                                             │
                     CLI (ge check / explain / fmt / graph)        ge lsp (JSON-RPC stdio)
                                                                          │
                                                          VS Code (cliente fino + TextMate mínima gerada)
```

1. **Sintaxe tolerante e sem perda** (evoluir `compiler/lexer` + `compiler/parser/hierarquia.go`):
   - token com `início` e `fim` (offset em bytes, linha, coluna em runas) e comentários como
     trivia anexada à linha (o formatter deixa de reler texto cru);
   - `layoutTree` sempre devolve a árvore; linha sem pai, tab e seção desconhecida viram nós
     marcados com erro e diagnósticos, e o parse continua na próxima linha de mesmo nível
     (a "lista aberta" do TypeScript é, no Germanio, a pilha de níveis: uma linha que cabe em
     algum nível aberto encerra os mais profundos);
   - a função pública passa a ser `Parse(arquivo, texto) (*Arvore, []Diagnostico)`, nunca
     `error`; `ge check` e `ge run` decidem recusar se houver erro.
2. **Modelo semântico único**: `ResolveIntent` → `ast.App`, que já existe, com a proveniência
   de cada fato (arquivo, intervalo, caminho hierárquico, frase plana), como `ge explain`
   exige. O resolver também devolve `[]Diagnostico` em vez de parar no primeiro conflito.
3. **Catálogo único de diagnósticos** (evoluir `compiler/diagnostics`): cada código com
   categoria (erro, aviso, sugestão), texto com parâmetros, "por quê", "como corrigir", link
   para a explicação educativa, e `Relacionados []Posicao` (as duas origens de um conflito de
   fusão). `teach`/`errorf` passam a construir `Diagnostico`, e a string atual vira só a
   renderização da CLI. Correções automáticas são registradas pelo código, e só quando forem
   a correção única e óbvia (critério do Swift, `swift.md`).
4. **Workspace/snapshot**: pacote novo (por exemplo `tooling/workspace`) que guarda, por
   caminho, o texto (do disco ou do editor), a árvore e os diagnósticos de sintaxe (cache por
   conteúdo) e resolve o `ast.App` do conjunto de imports. Snapshot imutável por versão, como
   `project/snapshot.go`; sem incremental semântico até haver medida que o justifique.
5. **Consultas**: `check`, `explain`, `fmt` e `graph` viram funções sobre o snapshot. `ge graph`
   e `ge explain pagina` passam a ler `ast.App`, e `tooling/intelligence/analyzer.go` sai do
   caminho de leitura (fecha G21). Hover = `explain` do fato sob o cursor; definição = origem
   do fato; completar = seções válidas no contexto (a tabela fechada de seções já é o que um
   completador precisa); semantic tokens = classe de cada token segundo o parser (dado,
   seção, papel, ação, modificador).
6. **`ge lsp`**: subcomando do mesmo binário, JSON-RPC por stdio, só traduzindo LSP ↔ consultas
   (incluindo a conversão de coluna em runas para UTF-16, como `lsconv`). Formatação do LSP =
   `ge fmt`; diagnósticos do LSP = os mesmos de `ge check`.
7. **VS Code**: a extensão vira cliente do `ge lsp`. A gramática TextMate continua, mínima
   (comentários, textos entre aspas, palavras de seção), para cor instantânea, mas é **gerada
   a partir das tabelas Go** (lexer, `idiomas.go`, tabela de seções) por um comando do próprio
   `ge`, e não mais por listas no Python; a cor correta vem dos semantic tokens.
8. **Testes por baseline** (padrão `tests/cases` + `baselines/reference`): um diretório de
   casos `.ge` e, para cada caso, os arquivos de saída de `check` (diagnósticos), `explain`
   (fatos com origem), `fmt` e `graph`, comparados literalmente, com atualização explícita
   (por exemplo `go test ./... -atualizar`). Marcadores de cursor no estilo fourslash para as
   consultas do LSP. Os "Testes normativos" de `INTENCAO.md` passam a ter casos nessa pasta.

Ordem sugerida (cada passo útil sozinho): 3 (diagnóstico estruturado) → 1 (tolerância e
intervalos) → 5 para `graph`/`explain pagina` (fecha G21) → 8 (baselines) → 4 e 6 (`ge lsp`) →
7 (VS Code).

---

## Para o Germanio

| Classificação | Lição | Problema concreto do Germanio | Arquivo afetado |
|---|---|---|---|
| ADOTAR | Um front-end para compiler e editor; nenhuma ferramenta com leitor próprio (TypeScript: scanner/parser/checker comuns a `tsc` e ao LS) | `ge graph` e `ge explain pagina` usam varredura por regex; o VS Code usa listas próprias | `tooling/intelligence/analyzer.go`, `tooling/gecli/cli.go`, `vscode-germanio/tools/gerar_gramatica.py` |
| ADOTAR | Parser que sempre devolve árvore e lista de diagnósticos (`ParseSourceFile` sem `error`; contextos de lista; nós ausentes) | parser de aplicação para no primeiro erro, em texto | `compiler/parser/parser.go` (`errorf`), `compiler/parser/hierarquia.go` (`layoutTree`, `teach`) |
| ADOTAR | Catálogo único de diagnósticos com código estável, categoria e gerado a partir de um arquivo de dados | erros do dialeto de aplicação não têm código; `Explanations` cobre só o núcleo | `compiler/diagnostics/diagnostics.go` |
| ADOTAR | `relatedInformation` | conflito de fusão precisa mostrar as duas origens (`INTENCAO.md` › Fusão) | `compiler/diagnostics`, `compiler/parser/resolver.go` |
| ADOTAR | Intervalos (início e fim) em todo token e nó, com conversão para UTF-16 só na borda do LSP | `Position` só tem início; nomes com acento tornam a conversão obrigatória | `compiler/lexer/lexer.go`, `compiler/diagnostics/diagnostics.go` |
| ADOTAR | Baselines revisáveis para diagnósticos, `explain`, `fmt` e `graph`; marcadores de cursor para o LSP | mudanças de mensagem e de inferência hoje só aparecem se algum teste pontual as verificar | novo diretório de casos; `tooling/*_test.go` |
| ADOTAR | LSP padrão desde o início, no mesmo binário Go (lição do tsserver e do 7.0) | não há LSP; evitar inventar protocolo | novo `ge lsp` em `tooling/gecli` |
| ADAPTAR | Semantic tokens do servidor + TextMate mínima gerada das tabelas Go | TextMate duplica o léxico à mão | `vscode-germanio/`, `compiler/lexer`, `compiler/idiomas` |
| ADAPTAR | Code fixes indexados por código de diagnóstico, com "corrigir todos" | "Como corrigir" é texto; tab → espaços e seção com erro de digitação (`suggest`/`editDistance` já existem) são correções únicas e óbvias | `compiler/parser/hierarquia.go`, futuro `ge lsp` |
| ADAPTAR | Snapshot imutável do workspace com texto não salvo (overlay) | `Compilar` lê do disco; um editor precisa do buffer | novo `tooling/workspace`; `runtime/engine.go` (`Compilar`) |
| ADAPTAR | Trivia (comentários) anexada aos tokens em vez de nós, permitindo árvore sem perda | lexer descarta `#`; o formatter relê linhas cruas | `compiler/lexer/lexer.go`, `tooling/formatter/intencao.go` |
| EVITAR | Protocolo próprio de editor (tsserver) | — | — |
| EVITAR | Formatter com opções dentro do LS | `ge fmt` é canônico | `tooling/formatter/` |
| EVITAR | Comportamento definido só por baselines, sem norma | o Germanio tem `INTENCAO.md`; baseline confirma a norma | `docs/INTENCAO.md` › Testes normativos |
| EVITAR (por ora) | Project references, `.tsbuildinfo`, pool de checkers, paralelismo | escala do Germanio não pede; só com medição | — |
| INVESTIGAR | Resolução semântica sob demanda (checker preguiçoso) para hover rápido em apps grandes | medir o tempo de `Compilar` no GitLab inteiro antes de decidir | `compiler/parser/resolver.go` |
| INVESTIGAR | Unificar os dois front-ends (núcleo estrito e aplicação) numa só camada de sintaxe e layout | convenções divergentes (GE1002 "dois espaços" × 4 canônicos) e `ge check` com três caminhos de fallback | `compiler/parser/germanio.go`, `compiler/semantic`, `tooling/gecli/cli.go` |
