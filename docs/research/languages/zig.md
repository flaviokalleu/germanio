# Zig: toolchain integrada e simplicidade explícita

- **Data:** 2026-09-28
- **Status:** estudo dirigido concluído. Código lido num clone raso do repositório oficial
  (commit `b6896c4`, 2026-09-28). Os caminhos citados foram vistos nessa árvore.
- **Pergunta principal:** o que o Germanio pode aprender com uma toolchain integrada (um único
  binário que é compiler, formatter, test runner e build)?

## Localização do código (verificada)

O repositório canônico é **https://codeberg.org/ziglang/zig** desde 2025-11-26; o GitHub
(`github.com/ziglang/zig`) ficou somente leitura, não é espelho; issues novas no Codeberg
começam no número 30000 para não colidir com as antigas
([anúncio](https://ziglang.org/news/migrating-from-github-to-codeberg/)). O README do
repositório aponta para Codeberg (`zig-bootstrap`, labels de issue). A interface web do Codeberg
devolve conteúdo embaralhado a scrapers; o código foi lido via `git clone` do mesmo endereço.

## Fontes consultadas

- Repositório: https://codeberg.org/ziglang/zig (arquivos `lib/std/zig/{tokenizer,Parse,Ast,
  AstGen,Zir,Zoir,ZonGen,ErrorBundle,Server}.zig`, `lib/std/zig/Ast/Render.zig`,
  `src/{main,fmt,Sema,Air,Zcu,InternPool,Compilation}.zig`, `lib/compiler/{test_runner,reduce}.zig`,
  `.forgejo/ISSUE_TEMPLATE/`)
- Página e visão geral: https://ziglang.org/ e https://ziglang.org/learn/overview/
- Referência da linguagem: https://ziglang.org/documentation/master/
- Build system: https://ziglang.org/learn/build-system/
- Migração: https://ziglang.org/news/migrating-from-github-to-codeberg/
- Fim do compilador C++ e bootstrap por wasm: https://ziglang.org/news/goodbye-cpp/
- Notas da 0.14.0: https://ziglang.org/download/0.14.0/release-notes.html
- `zig cc` (Andrew Kelley): https://andrewkelley.me/post/zig-cc-powerful-drop-in-replacement-gcc-clang.html
- Código de conduta (política de proposals): https://ziglang.org/code-of-conduct/
- ZLS: https://github.com/zigtools/zls

Zig está em 0.x (0.16.0 é a versão atual, [ziglang.org](https://ziglang.org/)); tudo abaixo
descreve uma linguagem sem promessa de estabilidade. Não tratar como resultado comprovado.

---

## Matriz

### Objetivo original
"A general-purpose programming language and toolchain for maintaining robust, optimal and
reusable software" ([ziglang.org](https://ziglang.org/)). Note que *toolchain* está na
definição da linguagem.

### Filosofia
"Focus on debugging your application rather than debugging your programming language
knowledge" ([overview](https://ziglang.org/learn/overview/)). "No hidden control flow, no
hidden memory allocations, no preprocessor, no macros" ([ziglang.org](https://ziglang.org/)).
O `zig zen` (texto em `src/main.zig`) inclui "Communicate intent precisely", "Favor reading code
over writing code", "Compile errors are better than runtime crashes", "Reduce the amount one
must remember", "Focus on logic, not style".

### Sintaxe
Estilo C, sem sobrecarga de operadores, sem exceções, sem propriedades que escondem chamadas:
lendo `foo(); bar();` sabe-se que só essas duas funções são chamadas, sem conhecer tipos
([overview](https://ziglang.org/learn/overview/)).

### Gramática
PEG publicada na referência, citada como cerca de 580 linhas
([overview](https://ziglang.org/learn/overview/), [langref](https://ziglang.org/documentation/master/)).
O parser declara a gramática LL(k): "predictive, using only constant token lookahead and never
backtracking", para garantir tempo linear no pior caso (`lib/std/zig/Parse.zig`).

### Lexer/tokenizer
`lib/std/zig/tokenizer.zig`, na biblioteca padrão, reutilizável por qualquer ferramenta.

### Parser
Descida recursiva preditiva (`Parse.zig`), com recuperação: o `Ast` resultante tem um campo
`errors` e continua utilizável (`Ast.zig`). Há fuzzing do parser e um oráculo gerado
(`parser_fuzz.zig`, `parser_generated_oracle.zig`).

### AST
Orientada a dados: `tokens` e `nodes` são `MultiArrayList` (estrutura de arrays), cada token
guarda só `tag` e `start: u32`; índices `u32` em vez de ponteiros; a raiz é `nodes[0]`
(`Ast.zig`). Linha e coluna são calculadas sob demanda (`tokenLocation`). **Comentários não
estão na AST**: o renderizador relê o texto-fonte entre um token e o seguinte para
reencontrá-los (`renderComments` em `Ast/Render.zig`). O mesmo `Ast` serve ao ZON, o formato de
dados do Zig (modo `.zon`).

### Representações intermediárias
Três, com fronteiras nítidas:
- **AST** → `AstGen.zig` → **ZIR** ("Zig Intermediate Representation"): sem tipos, um por
  arquivo, gerada sem olhar outros arquivos; depois de pronta, o resto do pipeline não precisa
  mais de AST, tokens nem fonte, exceto para montar erros (`Zir.zig`).
- **ZIR** → `Sema.zig` → **AIR** ("Analyzed Intermediate Representation"): tipada, uma por
  função; `Sema` faz checagem de tipos, execução `comptime` e geração das checagens de
  segurança, "the heart of the Zig compiler" (`Sema.zig`, 35 mil linhas; `Air.zig`).
- **AIR** → `codegen` (backends próprios e LLVM) → `link` (linkers próprios).
- ZON tem o seu atalho: `ZonGen` converte `Ast` em **ZOIR**, uma AST simplificada só para dados
  (`Zoir.zig`).

### Análise semântica
Dividida em duas: tudo que é decidível por arquivo e sem tipos acontece no AstGen (por exemplo
"unused local", "local variable is never mutated", "pointless discard", em `AstGen.zig`); o que
depende de tipos fica no Sema. `zig ast-check` expõe a primeira metade como comando: "Look for
simple compile errors in any set of files" (`src/main.zig`), rápido e paralelizável por arquivo.

### Sistema de tipos
Estático, tipos são valores em `comptime`; genéricos são funções que retornam tipos; opcionais
(`?T`) e uniões de erro (`!T`) no lugar de null e exceções
([overview](https://ziglang.org/learn/overview/)).

### Type inference
Local e por "result location" (o tipo esperado flui para a expressão; `AstRlAnnotate.zig`).
Nada global.

### Compiler/interpreter
Self-hosted (a versão exata em que virou padrão, 0.10, não verificada). Em dezembro de 2022 o compilador C++ (80 mil linhas) foi apagado: o bootstrap usa um
`zig1.wasm` comprimido no repositório, um interpretador WASI de cerca de 4 mil linhas de C e
três estágios até um compilador que se reproduz byte a byte; o pico de memória do build de
debug caiu de 11,3 GB para 3,8 GB ([goodbye-cpp](https://ziglang.org/news/goodbye-cpp/)).
Existe `bootstrap.c` na raiz. Compilação incremental opcional (`-fincremental`) com `--watch`,
de 14 s para 63 ms num projeto grande, ainda sem serializar estado
([0.14.0](https://ziglang.org/download/0.14.0/release-notes.html)).

### Runtime
Mínimo; não há GC nem escalonador. Não se aplica ao modo "runtime grande" do Go.

### Memory management
Manual e explícita: toda função que aloca recebe um `Allocator`; por isso bibliotecas rodam em
qualquer alvo, até freestanding ([overview](https://ziglang.org/learn/overview/)).
`std.testing.allocator` detecta vazamento em testes ([langref](https://ziglang.org/documentation/master/)).

### Standard library
Contém o próprio front-end do compilador (`lib/std/zig/`): tokenizer, parser, AST, renderer,
AstGen, ZIR, ErrorBundle, protocolo cliente/servidor. Ferramentas de terceiros usam o mesmo
código que o compilador.

### Package manager
Embutido no `zig build`: manifesto `build.zig.zon` (escrito em ZON, a mesma sintaxe da
linguagem) com dependências fixadas por hash; `zig fetch` copia para o cache global e imprime o
hash ([build-system](https://ziglang.org/learn/build-system/), `src/main.zig`).

### Formatter
`zig fmt` é subcomando do mesmo binário (`src/fmt.zig`): `--check` (lista arquivos fora do
padrão e sai com erro), `--stdin`, `--ast-check` (roda também os erros de AstGen),
`--exclude`, `--zon`. Sem opções de estilo. `// zig fmt: off/on` desliga trechos
(`Ast/Render.zig`). O renderizador aceita **Fixups** (inserir `_ = x;` para variável não usada,
esvaziar funções, omitir ou substituir nós, renomear identificadores), usados por `zig reduce`,
que minimiza um relato de bug dado um programa "checker" que diz se o caso continua
interessante (`lib/compiler/reduce.zig`). Ou seja: o printer da AST é também o motor de
transformações de código.

### Linter
Não há linter separado: o que seria lint (variável não usada, nunca mutada) é **erro de
compilação** no AstGen. Opinião forte e controversa, mas coerente com "Compile errors are
better than runtime crashes".

### Language server
Não oficial: ZLS, projeto comunitário, que exige pareamento de versão com o compilador e ainda
não tem integração completa com o build system no master
([zls](https://github.com/zigtools/zls)). Ponto fraco real da toolchain "integrada".

### IDE tooling
Via ZLS; o compilador oferece `--watch` e um protocolo de mensagens, mas não LSP.

### Diagnostics
Um `ErrorBundle` junta todos os erros vindos de lugares diferentes (parse, AstGen, Sema,
link) para o consumidor, com codificação vazia quando não há erro, pensado para compilação
incremental (`ErrorBundle.zig`). O mesmo bundle trafega pelo protocolo `std.zig.Server` até o
build runner (`Server.zig`, mensagem `error_bundle`). O repositório tem um formulário de issue
dedicado a "Error message improvement: Compiler produces an unhelpful or misleading error
message" (`.forgejo/ISSUE_TEMPLATE/error_message.yml`): mensagem ruim é bug catalogado.

### Testing
`test "nome" { ... }` é declaração de nível superior no mesmo arquivo do código; só entra em
builds de teste; retorna `anyerror!void`; `error.SkipZigTest` pula; um teste com nome de
identificador é um *doctest* ligado à declaração
([langref](https://ziglang.org/documentation/master/)). `zig test arquivo.zig` compila e roda;
`zig build test` roda como passo do DAG. O test runner padrão (`lib/compiler/test_runner.zig`)
conversa com o build runner pelo mesmo protocolo `std.zig.Server` (mensagens `test_metadata`,
`test_started`, `test_results`, cobertura), o que permite limite de tempo por teste e
resultados estruturados.

### Documentation
`///` documenta a declaração seguinte, `//!` o namespace; doc comment em lugar inesperado é
erro de compilação ([langref](https://ziglang.org/documentation/master/)). `zig std` abre a
documentação da biblioteca padrão no navegador (`src/main.zig`).

### Evolution process
Proposals são issues com labels ("Proposal: Proposed", "Accepted"; README do repositório). Mas
desde a política atual: "Please do not file a proposal to change the language"; quem quiser
propor precisa convencer um membro do core team a abrir e defender a proposta
([code of conduct](https://ziglang.org/code-of-conduct/), referenciado pelo
`.forgejo/ISSUE_TEMPLATE/config.yml`). Na prática a evolução depende de poucos mantenedores (governança formal não verificada). Política estrita "no LLM / no AI" para contribuições (mesma fonte).

### Backward compatibility
Nenhuma promessa antes de 1.0. Cada release quebra linguagem e build system: por exemplo, na
0.14.0, campos de `std.builtin.Type` minúsculos, `@setCold` removido, `@fence` removido,
`addLibrary()` no lugar de `addSharedLibrary`/`addStaticLibrary`
([0.14.0](https://ziglang.org/download/0.14.0/release-notes.html)). O `zig fmt` ajudou em
migrações antigas, mas não há mecanismo tipo GODEBUG. O protocolo do build system tem versão de
ABI própria (`Server.zig`).

### Principais acertos
- **Um binário, muitos papéis**: `build`, `fetch`, `init`, `test`, `run`, `fmt`, `ast-check`,
  `reduce`, `cc`, `c++`, `translate-c`, `ar`, `objcopy`, `env`, `targets`, `std`, `zen`
  (`src/main.zig`). O usuário instala uma coisa.
- **Front-end como biblioteca**: formatter, ast-check, compilador e ferramentas externas usam o
  mesmo tokenizer/parser/AST/renderer (`lib/std/zig/`), então não há "vários parsers".
- **Cross compilation de série**: libc distribuída como fonte e compilada sob demanda com cache;
  `-target aarch64-linux-gnu` basta; distribuição inteira abaixo de 45 MB
  ([zig cc](https://andrewkelley.me/post/zig-cc-powerful-drop-in-replacement-gcc-clang.html),
  [overview](https://ziglang.org/learn/overview/)).
- **Build em Zig**: `build.zig` é código da própria linguagem, modelado como DAG de passos
  concorrentes, com convenções (`standardTargetOptions`, `standardOptimizeOption`, passo `run`)
  ([build-system](https://ziglang.org/learn/build-system/)). O manifesto é ZON, não um terceiro
  formato.
- **Fronteira AST → ZIR sem tipos, por arquivo**, que dá paralelismo, cache e um checador
  rápido.
- **Testes no mesmo arquivo**, excluídos do build normal.

### Principais problemas
- Instabilidade permanente pré-1.0; quebra a cada release.
- LSP fora do projeto, dessincronizado do compilador.
- Build system em linguagem de propósito geral: poderoso, mas é programa, com API que muda.
- Erros de "não usado" como erro atrapalham a exploração (efeito colateral registrado no
  `Fixups.unused_var_decls`, que existe justamente para inserir `_ = x;` mecanicamente).
- Processo de evolução fechado; depende de poucas pessoas.

### Complexidade acumulada
`Sema.zig` com cerca de 35 mil linhas; `comptime` torna a análise semântica um interpretador;
backends e linkers próprios; o `zig cc` embute Clang/LLVM (arquivos `zig_clang_*.cpp`,
`zig_llvm.cpp` em `src/`). A linguagem é pequena; o binário não.

### O que Germanio pode aprender
Ver "Para o Germanio". Resumo: um só binário `ge` com subcomandos; o front-end como pacote
compartilhado por todas as ferramentas; uma fase de checagem por arquivo sem resolver
(`ast-check`); `fmt --check` para CI; um protocolo estruturado de resultados entre compilador,
testes e editor; um coletor único de diagnósticos; testes declarados junto da intenção; o
manifesto escrito na própria linguagem.

### O que Germanio NÃO deve copiar
- Build system como programa (`build.zig`): o público do Germanio não programa; o build tem de
  ser inferido do `.ge`.
- Gerenciamento explícito de memória e de alocadores: é exatamente o que o Germanio subtrai.
- `comptime` e metaprogramação: complexidade de análise que o domínio não pede.
- Variável não usada como erro: para leigo, uma entidade declarada e ainda não usada é passo
  normal do trabalho; no máximo aviso, com `ge explain`.
- Falta de promessa de compatibilidade e processo de evolução fechado.
- Embutir um compilador C: não resolve problema do Germanio.
- Ausência de LSP oficial: no Germanio o LSP deve ser parte do binário, não projeto externo.

---

## Para o Germanio

**ADOTAR: um único binário `ge` como dono de todos os papéis.** Hoje há dois CLIs: o `ge`
(`tooling/gecli/cli.go`, com `check`, `fmt`, `explain`, `testar`, `init`...) e o legado
`germanio` (`cli/cli.go`, com `run`, `build`, `docker`, `new`, `ide`), acessível por
`ge legado`. A lição do `src/main.zig` é que cada papel (rodar, checar, formatar, testar,
empacotar, explicar, servir o editor) é subcomando do mesmo binário, sobre o mesmo front-end.
Mover `build` e `docker` para subcomandos de primeira classe do `ge` e aposentar o `legado`.
Afeta `tooling/gecli/cli.go`, `cli/cli.go`, `main.go`.

**ADOTAR: o front-end como pacote único consumido por todas as ferramentas.** Em Zig,
`zig fmt`, `ast-check`, o compilador e terceiros usam `lib/std/zig/`. No Germanio,
`tooling/formatter/intencao.go` reconstrói sozinho a pilha de indentação que
`compiler/parser/hierarquia.go` (`layoutTree`) já calcula, e o VS Code tem regex próprio. Regra:
nenhuma ferramenta tokeniza ou mede indentação; todas pedem a árvore ao parser. Afeta
`tooling/formatter`, `compiler/parser/hierarquia.go`, `vscode-germanio/tools/gerar_gramatica.py`.

**ADAPTAR: comentários fora da árvore semântica, recuperados por posição.** O `Ast` do Zig não
guarda comentários; o renderizador os reencontra no texto entre tokens, usando só offsets. Para o
Germanio isso casa com a ideia de "reduzir o bloco a frases planas": a árvore semântica fica
limpa, e o formatter usa as posições (offset de início e fim de cada linha lógica) para
recolocar comentários e linhas em branco. Pré-requisito: posições com offset, não só
linha/coluna (ver `go.md`, FileSet). Afeta `compiler/diagnostics/diagnostics.go`,
`tooling/formatter/intencao.go`.

**ADOTAR: `ge fmt --check` para CI e editor.** `zig fmt --check` lista arquivos fora do padrão e
sai com erro (`src/fmt.zig`). Não é preciso reescrever arquivos para validar um PR. Afeta
`tooling/gecli/cli.go` (caso `fmt`).

**ADAPTAR: separar a checagem por arquivo (sem resolver) da checagem do projeto.** A fronteira
AST→ZIR do Zig permite `zig ast-check`, rápido e por arquivo. No Germanio a fronteira natural já
existe: `hierarquia.go` produz `ast.Intent` por arquivo; `resolver.go` funde arquivos em
`ast.App`. Diagnósticos que dependem só do arquivo (seção desconhecida numa tabela fechada,
indentação, frase sem sujeito) devem sair na primeira fase, sem esperar a resolução, o que dá
resposta imediata no editor. Afeta `compiler/parser/hierarquia.go`, `compiler/parser/resolver.go`.

**ADOTAR: um coletor único de diagnósticos.** O `ErrorBundle` junta erros de parse, AstGen, Sema
e link. No Germanio, `semantic.Check` para no primeiro `error`; parser, resolver e
`tooling/intelligence` produzem erros em formatos próprios. Um `diagnostics.Bundle` (lista
ordenada por posição, no formato de quatro partes) permitiria mostrar todos de uma vez e servir
CLI, testes e LSP do mesmo objeto. Afeta `compiler/diagnostics`, `compiler/semantic/check.go`,
`compiler/parser/resolver.go`.

**ADOTAR: mensagem ruim é bug catalogado.** Zig tem formulário de issue só para mensagens de
erro inúteis (`.forgejo/ISSUE_TEMPLATE/error_message.yml`). Para um público leigo isso é mais
importante ainda: um tipo de issue "diagnóstico confuso" com versão, entrada e mensagem
esperada, e cada correção vira teste de diagnóstico. Afeta `compiler/diagnostics/diagnostics_test.go`
e o processo do repositório.

**ADAPTAR: testes junto da intenção.** O `test "nome" {}` do Zig vive no arquivo do código e
some do build normal. O núcleo estrito do Germanio já reconhece `teste`
(`compiler/parser/germanio.go`), e `ge testar` existe. Falta a forma para o dialeto de aplicação,
em linguagem de intenção (por exemplo, um bloco que descreve "quando developer tenta excluir
projeto, é recusado"), excluída do app gerado e executada por `ge testar`. Afeta
`compiler/parser/hierarquia.go`, `tooling/gecli/cli.go`, `docs/INTENCAO.md`.

**INVESTIGAR: um protocolo estruturado entre `ge`, test runner e editor.** Zig usa
`std.zig.Server` com mensagens tipadas (`error_bundle`, `test_metadata`, `test_results`,
`file_system_inputs`) e versão de ABI. Para o Germanio, o análogo é o LSP (padrão existente)
mais uma saída `--json` estável de `ge check`/`ge testar`/`ge explain`. Resolve a ausência de LSP
e o hot reload que hoje re-executa o processo (`runtime/hotreload.go`). Investigar antes de
inventar protocolo próprio: LSP provavelmente basta.

**ADAPTAR: o manifesto na própria linguagem.** `build.zig.zon` é ZON, a sintaxe de dados do
Zig, parseada pelo mesmo `Ast`. Se o Germanio precisar de configuração de projeto (versão da
linguagem, banco, porta, adaptadores), que seja um `.ge` com as mesmas regras de layout, e não
YAML/TOML/JSON. Afeta `cli/modelos`, `compiler/parser`.

**ADAPTAR: cross compilation como padrão.** `zig` cruza para qualquer alvo com `-target`. O
Germanio, em Go com CGO desativado e SQLite puro, já pode gerar executáveis para
Linux/Windows/macOS via `GOOS/GOARCH` em `ge build`; a lição é tornar isso uma palavra
(`ge build --para windows`) e não uma instrução técnica. Afeta `cli/cli.go` (build).

**INVESTIGAR: `ge reduzir`.** `zig reduce` minimiza um caso de bug usando os Fixups do
renderizador. Com um printer sobre a árvore, o Germanio poderia minimizar um `.ge` que
reproduz um erro do runtime, útil para relatos de leigos. Depende do printer sobre a árvore.

**EVITAR:** build como programa; alocação explícita; erro por declaração não usada; LSP fora do
projeto; quebra de compatibilidade sem mecanismo de versão; processo de evolução fechado.
