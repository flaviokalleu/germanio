# Gleam: uma linguagem jovem com compiler e tooling num binário só

- **Data:** 2026-09-28
- **Status:** estudo concluído. Nada aqui é normativo para o Germanio; as decisões ficam em `docs/INTENCAO.md`.
- **Pergunta principal:** como uma linguagem jovem organiza compiler e tooling sem décadas de legado, e o que isso ensina ao Germanio, que está na mesma fase.

## Fontes consultadas

- Repositório: https://github.com/gleam-lang/gleam (diretórios de topo e `compiler-core/src` listados pela API do GitHub em 2026-09-28; o crate `gleam-format` declara a versão `1.19.0-rc2`)
- `compiler-core/src/diagnostic.rs`: https://github.com/gleam-lang/gleam/blob/main/compiler-core/src/diagnostic.rs
- `compiler-core/src/error.rs` (5546 linhas; `did_you_mean`, `edit_distance`, `to_diagnostics`): https://github.com/gleam-lang/gleam/blob/main/compiler-core/src/error.rs
- `compiler-core/src/analyse.rs` (ordem por dependência e generalização): https://github.com/gleam-lang/gleam/blob/main/compiler-core/src/analyse.rs
- `compiler-core/src/type_.rs` (unificação, occurs check): https://github.com/gleam-lang/gleam/blob/main/compiler-core/src/type_.rs
- `compiler-core/src/exhaustiveness.rs` (cabeçalho com as referências do algoritmo): https://github.com/gleam-lang/gleam/blob/main/compiler-core/src/exhaustiveness.rs
- `compiler-core/src/parse/extra.rs` (comentários fora da AST): https://github.com/gleam-lang/gleam/blob/main/compiler-core/src/parse/extra.rs
- `format/src/lib.rs`: https://github.com/gleam-lang/gleam/blob/main/format/src/lib.rs
- `compiler-cli/src/lib.rs` (subcomandos, flags de `format`, `fix`): https://github.com/gleam-lang/gleam/blob/main/compiler-cli/src/lib.rs
- `compiler-core/Cargo.toml` (dependências `codespan-reporting`, `pubgrub`, `insta`, `lsp-types`): https://github.com/gleam-lang/gleam/blob/main/compiler-core/Cargo.toml
- Gleam v1: https://gleam.run/news/gleam-version-1/
- v0.25, `use`: https://gleam.run/news/v0.25-introducing-use-expressions/
- v0.27, "Hello panic, goodbye try": https://gleam.run/news/v0.27-hello-panic-goodbye-try/
- v0.28: https://gleam.run/news/v0.28-monorepos-fast-maps-and-more/
- v0.30, externals: https://gleam.run/news/v0.30-local-dependencies-and-enhanced-externals/
- v0.32, "Polishing syntax for stability": https://gleam.run/news/v0.32-polishing-syntax-for-stability/
- v1.2, fault tolerant: https://gleam.run/news/fault-tolerant-gleam/
- v1.6, context aware: https://gleam.run/news/context-aware-compilation/
- v0.21, language server: https://gleam.run/news/v0.21-introducing-the-gleam-language-server/
- Referência do language server: https://gleam.run/documentation/language-server-reference/
- FAQ: https://gleam.run/frequently-asked-questions/
- Tour, `@deprecated`: https://tour.gleam.run/functions/deprecations/
- Código do Germanio lido: `compiler/diagnostics/diagnostics.go`, `compiler/parser/hierarquia.go`, `compiler/lexer/lexer.go` (`Token.Raw`), `tooling/gecli/cli.go`.

Complementa `rust.md` (Applicability, spans, testes de saída). Aqui o foco é a organização de um projeto jovem.

---

## Matriz

### Objetivo original
Linguagem funcional com tipos estáticos para a BEAM (Erlang), depois também para JavaScript. O FAQ conta que protótipos em Erlang sofreram porque "the lack of static types was making refactoring a slow and error prone process", e que a reescrita em Rust eliminou dívida técnica ([FAQ](https://gleam.run/frequently-asked-questions/)).

### Filosofia
Linguagem pequena e uma forma de fazer: "Gleam will always be a small and cohesive language with a minimal feature set" ([FAQ](https://gleam.run/frequently-asked-questions/)); "we always prefer to have fewer ways to do the same thing in Gleam, and fewer things to have to learn" ([v0.27](https://gleam.run/news/v0.27-hello-panic-goodbye-try/)); no v1, "with each new language feature the language as a whole becomes more complex" ([v1](https://gleam.run/news/gleam-version-1/)). Sem type classes, porque tornam fácil escrever código difícil de entender ([FAQ](https://gleam.run/frequently-asked-questions/)).

### Sintaxe
Chaves, expressões, pipes `|>`, `case`, `use`. Não se aplica à forma do Germanio. O que interessa é como a sintaxe foi *podada* antes do v1 (Evolution process).

### Gramática
Sem gramática formal publicada que eu tenha visto (não verificado). O parser em Rust é a referência.

### Lexer/tokenizer
`compiler-core/src/parse/lexer.rs` e `token.rs`. Comentários, linhas vazias, quebras e vírgulas finais não entram na AST: vão para `ModuleExtra`, um conjunto de listas de spans ordenadas (`module_comments`, `doc_comments`, `comments`, `empty_lines`, `new_lines`, `trailing_commas`) consultado por busca binária ([extra.rs](https://github.com/gleam-lang/gleam/blob/main/compiler-core/src/parse/extra.rs)). É uma alternativa mais barata que uma árvore sem perdas: a AST fica limpa e o formatter recupera o que precisa por posição.

### Parser
Descendente recursivo em `parse.rs`, com `parse/error.rs` para os erros. No v1.2 a tolerância a falhas valia para a análise, e a de parsing estava anunciada como trabalho futuro ([v1.2](https://gleam.run/news/fault-tolerant-gleam/)). O estado atual do parser nesse ponto não foi verificado.

### AST
Duas ASTs com os mesmos nós: `ast/untyped.rs` e `ast/typed.rs` (listagem de `compiler-core/src/ast`). A análise transforma a não tipada na tipada. Todo nó carrega `SrcSpan` (intervalo de bytes).

### Representações intermediárias
Poucas: AST não tipada → AST tipada → código gerado (Erlang ou JavaScript), mais árvores de decisão para `case` (`exhaustiveness.rs`). Não há IR de baixo nível própria. O alvo é código-fonte de outra linguagem, então a BEAM e o JS fazem o resto.

### Análise semântica
`analyse.rs` ordena funções e constantes por dependência (grafo de chamadas, `call_graph.rs`), infere cada grupo mutuamente recursivo e generaliza os tipos do grupo ao fim ("Definitions that do not depend on other definitions are inferred first", [analyse.rs](https://github.com/gleam-lang/gleam/blob/main/compiler-core/src/analyse.rs)). Desde o v1.2 um erro numa definição não para o módulo: "it will move on to the next definition in the module, returning all of the errors" ([v1.2](https://gleam.run/news/fault-tolerant-gleam/)).

Exaustividade de `case` segue Jacobs ("How to compile pattern matching") e Maranget, e a primeira implementação foi adaptada do trabalho de Yorick Peterse ([exhaustiveness.rs](https://github.com/gleam-lang/gleam/blob/main/compiler-core/src/exhaustiveness.rs)).

### Sistema de tipos
Estático, sem subtipagem, sem type classes, com tipos customizados (somas de produtos), genéricos e `Result` no lugar de exceções ([FAQ](https://gleam.run/frequently-asked-questions/)).

### Type inference
Estilo Hindley-Milner: variáveis de tipo com `TypeVar::Link` (unificação por ligação), occurs check (`unify_unbound_type`, [type_.rs](https://github.com/gleam-lang/gleam/blob/main/compiler-core/src/type_.rs)) e generalização por grupo de dependência ([analyse.rs](https://github.com/gleam-lang/gleam/blob/main/compiler-core/src/analyse.rs)). Anotações são opcionais. O language server tem uma ação que as insere a partir do tipo inferido ([LS reference](https://gleam.run/documentation/language-server-reference/)). Esse é o ponto que mais se aproxima do princípio de subtração do Germanio: inferir tudo, e mostrar o inferido quando pedirem.

### Compiler/interpreter
Workspace Rust com crates separados por papel: `compiler-core` (parse, analyse, type_, erlang, javascript, exhaustiveness, diagnostic, error, docs, hex, build), `compiler-cli`, `format`, `language-server`, `compiler-wasm` (compiler no navegador), `hexpm` e utilitários (listagem do repositório). O formatter e o language server dependem de `compiler-core`. Há um parser e uma análise só, usados pelo CLI, pelo LSP e pelo WASM.

### Runtime
O da plataforma-alvo: BEAM ou JavaScript. A troca de mensagens tipada fica em bibliotecas, "rather than being part of the core language itself", justamente para rodar nas duas plataformas ([FAQ](https://gleam.run/frequently-asked-questions/)).

### Memory management
O da plataforma-alvo. Não se aplica.

### Standard library
`gleam_stdlib` é um pacote separado, versionado fora do compiler (não verifiquei a política de versões).

### Package manager
Embutido no binário (`gleam add`, `remove`, `deps`, `publish`, `hex`, [cli lib.rs](https://github.com/gleam-lang/gleam/blob/main/compiler-cli/src/lib.rs)). Usa o registro Hex, compartilhado com Erlang e Elixir, e o resolvedor `pubgrub` ([Cargo.toml](https://github.com/gleam-lang/gleam/blob/main/compiler-core/Cargo.toml)). O v0.31 passou a exigir dependências explícitas ([v0.31](https://gleam.run/news/v0.31-keeping-dependencies-explicit/), conteúdo não lido em detalhe).

### Formatter
`gleam format [arquivos] [--stdin] [--check]`, sem nenhuma opção de estilo ([cli lib.rs](https://github.com/gleam-lang/gleam/blob/main/compiler-cli/src/lib.rs)). O recuo é uma constante (`INDENT: isize = 2`), a largura é 80, e a impressão usa um crate de pretty printing próprio (`pretty-arena`) ([format/src/lib.rs](https://github.com/gleam-lang/gleam/blob/main/format/src/lib.rs)). O formatter também foi ferramenta de migração: `gleam format` reescreveu o `assert` antigo ([v0.27](https://gleam.run/news/v0.27-hello-panic-goodbye-try/)).

### Linter
Não há linter separado. Os avisos saem do compiler (`warning.rs`) e o `@deprecated` gera aviso com a mensagem do autor ([tour](https://tour.gleam.run/functions/deprecations/)). Não verifiquei se há como desligar avisos individualmente.

### Language server
`gleam lsp`, no mesmo binário ([v0.21](https://gleam.run/news/v0.21-introducing-the-gleam-language-server/)). O anúncio justifica o LSP pelo problema N×M entre editores e linguagens, mas não justifica a escolha do binário único. O crate `language-server` tem `code_action.rs`, `completer.rs`, `rename.rs`, `reference.rs`, `signature_help.rs` e `engine.rs`. Várias ações nascem de erros do compiler: "Add missing patterns", "Add missing import", "Inexhaustive let to case", "Remove unused imports", "Replace `_` with type" ([LS reference](https://gleam.run/documentation/language-server-reference/)). A tolerância a falhas do v1.2 produziu "a dramatic improvement in the experience of using the Gleam language server" ([v1.2](https://gleam.run/news/fault-tolerant-gleam/)).

### IDE tooling
O LSP é o tooling, e as extensões de editor só configuram `gleam lsp`. O `compiler-wasm` sustenta o tour e o playground no navegador.

### Diagnostics
Ver a seção dedicada abaixo.

### Testing
Snapshots com `insta` em todo o compiler (dependência em `compiler-core/Cargo.toml`). A listagem de `type_/tests/snapshots` mostra 984 arquivos e a de `parse/snapshots`, 264 (a listagem da API pode estar truncada). Os snapshots guardam a mensagem renderizada completa, e o equivalente ao `.stderr` do rustc são esses arquivos `.snap`. Um comentário em `did_you_mean` mostra o efeito: uma regra entra para fazer passar um snapshot concreto ("This seems to solve the `unknown_variable_3` test", [error.rs](https://github.com/gleam-lang/gleam/blob/main/compiler-core/src/error.rs)).

### Documentation
`gleam docs` gera documentação dos pacotes (módulo `docs`). O tour interativo roda o compiler em WASM.

### Evolution process
Centralizado no mantenedor principal e na equipe central, sem processo formal de RFC comparável ao do Rust (não verificado em documento oficial). O que se vê nos anúncios é um padrão repetido: introduzir o recurso mais geral, depreciar o redundante com aviso e migração automática, remover na versão seguinte.
- `use` (v0.25) tornou `try` redundante. `try` foi depreciado no v0.27, com `gleam fix` convertendo para `use`, e removido no v0.28 junto com a sintaxe antiga de `assert` ([v0.25](https://gleam.run/news/v0.25-introducing-use-expressions/), [v0.27](https://gleam.run/news/v0.27-hello-panic-goodbye-try/), [v0.28](https://gleam.run/news/v0.28-monorepos-fast-maps-and-more/)).
- `assert` padrão virou `let assert`, o que liberou a palavra para uma asserção booleana ([v0.27](https://gleam.run/news/v0.27-hello-panic-goodbye-try/)).
- A FFI foi redesenhada com `@external`, que substituiu a compilação condicional por alvo, e "the single function header ensures the various targets are kept in sync" ([v0.30](https://gleam.run/news/v0.30-local-dependencies-and-enhanced-externals/)). Não verifiquei em que versão a sintaxe antiga foi removida.
- O v0.32 ("Polishing syntax for stability") passou a exigir `type` na importação de tipos, porque "a type and a value can have the same name", e renomeou `BitString` para `BitArray`, porque o nome confundia. Tudo com aviso de depreciação e `gleam fix` ([v0.32](https://gleam.run/news/v0.32-polishing-syntax-for-stability/)).

### Backward compatibility
A partir do v1 (4 de março de 2024), a garantia cobre linguagem, compiler, build tool, package manager, formatter, language server e API WASM, com semver e "maintaining backwards compatibility is now a priority". As exceções são segurança e soundness ([v1](https://gleam.run/news/gleam-version-1/)). A poda aconteceu *antes* do v1, justamente para que a garantia cobrisse uma linguagem já enxuta.

### Principais acertos
1. Um front-end só, usado por CLI, LSP, formatter e WASM. Não existe o problema "dois parsers discordam".
2. Remoção ativa de recursos antes da estabilidade, sempre com migração automática.
3. Formatter sem opções que também serve de ferramenta de migração.
4. Diagnostics com hipótese específica (abaixo) e nomes exibidos como o usuário os escreveria.
5. Análise tolerante a falhas por definição, o que melhorou o LSP de uma vez.

### Principais problemas
1. `error.rs` com 5546 linhas: um `match` central que cresce a cada erro. É fácil de achar, mas é um único arquivo gigante.
2. Sem códigos de erro nem explicação longa. `Diagnostic` tem só `title`, `text`, `level`, `location`, `hint` ([diagnostic.rs](https://github.com/gleam-lang/gleam/blob/main/compiler-core/src/diagnostic.rs)). Não há como buscar ou linkar um erro de forma estável.
3. A dica (`hint`) é texto. Não há edição estruturada com confiança no diagnostic, e as correções vivem à parte, como code actions do LSP que reconstroem a situação.
4. Governança concentrada: rápida, mas sem registro formal das decisões (não verificado).

### Complexidade acumulada
Baixa para a idade. A poda pré-v1 e a recusa de type classes e macros mantêm a superfície pequena. Dois alvos de geração são o maior custo estrutural.

### O que Germanio pode aprender
Ver "Para o Germanio".

### O que Germanio NÃO deve copiar
- Hindley-Milner completo: o `.ge` do nível padrão não tem funções polimórficas; inferência de tipo de campo por regras fixas basta.
- Dois alvos de geração de código (Erlang/JS): o Germanio tem um runtime; multiplicar alvos multiplica testes.
- `hint` como texto livre e código de erro ausente: o Germanio já tem códigos e deve evoluir para correções estruturadas (ver `rust.md`).
- Um único `error.rs` gigante: melhor manter o erro perto do mecanismo que o emite, com a renderização centralizada.
- Governança sem registro escrito: o Germanio já tem `docs/INTENCAO.md` como camada normativa.

---

## Diagnostics em profundidade

### Estrutura

`Diagnostic { title, text, level, location: Option<Location>, hint: Option<String> }`, com `Location { src, path, label, extra_labels }` e `Label { text, span }`. O rótulo principal vira `LabelStyle::Primary` e os extras viram `Secondary`, renderizados por `codespan-reporting`. Um rótulo extra pode apontar outro arquivo (`src_info`) ([diagnostic.rs](https://github.com/gleam-lang/gleam/blob/main/compiler-core/src/diagnostic.rs)). Os níveis são só erro e aviso.

### Erros são dados; o texto é derivado

Os erros são enums com os fatos do caso, e um único `to_diagnostics()` converte cada variante em `Diagnostic` ([error.rs](https://github.com/gleam-lang/gleam/blob/main/compiler-core/src/error.rs)). O exemplo mais instrutivo é `TypeError::UnknownVariable`, que carrega, além do nome e do local:
- `discarded_location`: se existe `_nome` descartado no escopo. Nesse caso o diagnostic muda por completo, com rótulo primário "So this is not in scope", rótulo secundário "This value is discarded" e dica "Change `_x` to `x`".
- `type_with_name_in_scope`: se o nome existe como *tipo*, a mensagem vira "`x` is a type, it cannot be used as a value".
- `possible_modules`: módulos importáveis que exportam esse nome, listados como "Did you mean one of these".

É o mesmo mecanismo do rustc (procurar em outro namespace), com uma lição a mais: **a hipótese é um campo da variante do erro, preenchido por quem detecta o erro, e a renderização escolhe o texto pela hipótese presente**. Aplicado ao Germanio, a variante "linha que não é seção" carregaria `PapelDeclaradoEm *Origem` e `FilhosSaoAcoes bool`, e o renderizador escolheria entre "não é uma seção" e "talvez você quisesse declarar acesso".

### Nomes parecidos

`did_you_mean` foi portado do `edit_distance.rs` do rustc (o comentário cita o commit de origem). Com uma opção só, sugere essa opção. Antes da distância, procura igualdade sem caixa, e o limite é um terço do nome, com mínimo 1 ([error.rs](https://github.com/gleam-lang/gleam/blob/main/compiler-core/src/error.rs)). Uma linguagem jovem pegou o algoritmo testado de outra em vez de inventar. Vale para o `suggest` do Germanio (limite fixo `< 3`).

### Nomes como o usuário os escreveria

Desde o v1.6, tipos aparecem nos erros "using the names and syntax that the programmer would use within that specific area of the code": `order.Order`, o alias de import, ou `Order` sem qualificação, e `gleam.Int` quando o prelude está sombreado ([v1.6](https://gleam.run/news/context-aware-compilation/)). No Germanio multilíngue, a transposição é direta. O lexer normaliza palavras-chave para o português canônico, mas guarda a grafia original em `Token.Raw` (`compiler/lexer/lexer.go`). Hoje a lista de seções sugeridas sai sempre em português (`displaySections`). Quem escreveu `access` deveria ver a sugestão com `access`.

### Tolerância a falhas

Um erro numa definição não interrompe a análise do módulo ([v1.2](https://gleam.run/news/fault-tolerant-gleam/)). No Germanio, a unidade natural é o bloco de dado: um erro em `projetos` não deveria esconder erros em `issues`.

### Correção fora do diagnostic

As correções do Gleam são code actions do LSP que refazem a análise da situação, e a dica do compiler é só texto. Isso duplica conhecimento: o compiler sabe que falta um padrão, e o LSP precisa saber de novo como inseri-lo. O modelo do rustc (sugestão com edição e Applicability dentro do diagnostic, consumida por `cargo fix` e pelo editor) evita a duplicação. Para o Germanio, que ainda não tem LSP, o modelo rustc é o mais barato: a mesma sugestão serve ao terminal, ao `ge fix` e, depois, ao editor.

---

## Resposta à pergunta principal

Uma linguagem jovem se organiza sem legado quando faz três coisas enquanto ainda pode:

1. **Um front-end para tudo.** CLI, LSP, formatter e playground usam `compiler-core`. O Germanio tem hoje três leitores da sintaxe (regex do VS Code, formatter, parser). O formatter já reparseia pelo parser, mas a gramática TextMate é gerada à parte.
2. **Podar antes de prometer estabilidade.** Cada remoção seguiu o mesmo roteiro: recurso mais geral, depreciação com aviso, migração automática (`gleam fix` ou `gleam format`), remoção na versão seguinte. O v1 veio depois da poda. O Germanio tem dois dialetos e formas planas e hierárquicas equivalentes, e ainda não prometeu estabilidade. A janela é agora.
3. **Tooling no mesmo binário e sem opções.** `gleam format` sem estilo configurável, `gleam fix` para migrar e `gleam lsp` para o editor. O `ge` já tem `fmt` e `explain`; faltam `fix` e o LSP, e ambos podem nascer como consumidores dos diagnostics estruturados.

---

## Para o Germanio

| # | Classe | Lição | Problema do Germanio que resolve | Arquivo afetado |
|---|---|---|---|---|
| 1 | ADOTAR | Erro como variante com os fatos e as hipóteses (`discarded_location`, `type_with_name_in_scope`), texto derivado num renderizador ([error.rs](https://github.com/gleam-lang/gleam/blob/main/compiler-core/src/error.rs)) | `teach` recebe strings prontas; a hipótese "developer é um papel" não tem onde morar | `compiler/diagnostics/diagnostics.go`, `compiler/parser/hierarquia.go` |
| 2 | ADOTAR | Exibir as palavras na língua em que o usuário escreveu ([v1.6](https://gleam.run/news/context-aware-compilation/)) | sugestões e listas de seções saem sempre em português, mesmo para quem escreve em inglês | `compiler/parser/hierarquia.go` (`displaySections`, `suggest`), `compiler/lexer/lexer.go` (`Token.Raw`) |
| 3 | ADOTAR | Continuar depois do erro, por unidade (bloco de dado) ([v1.2](https://gleam.run/news/fault-tolerant-gleam/)) | o primeiro erro interrompe; o leigo corrige um por vez; um LSP futuro precisa de resultado parcial | `compiler/parser/hierarquia.go`, `compiler/parser/resolver.go` |
| 4 | ADOTAR | Reusar o algoritmo de nome parecido do rustc (terço do nome, sem caixa, opção única) | `suggest` usa limite fixo `< 3` | `compiler/parser/hierarquia.go` (`suggest`, `editDistance`) |
| 5 | ADAPTAR | Roteiro de remoção: forma geral → depreciação com aviso → `ge fix` migra → remoção; podar antes da estabilidade ([v0.27](https://gleam.run/news/v0.27-hello-panic-goodbye-try/), [v0.32](https://gleam.run/news/v0.32-polishing-syntax-for-stability/)) | dois front-ends e formas equivalentes coexistindo; risco de congelar redundância | `compiler/parser/germanio.go`, `compiler/parser/declarativo.go`, `docs/INTENCAO.md`, futuro `ge fix` em `tooling/gecli` |
| 6 | ADOTAR | Um front-end só para CLI, formatter, LSP e editor | três leitores da sintaxe (regex do VS Code, formatter, parser) | `vscode-germanio/tools/gerar_gramatica.py`, `tooling/formatter/` |
| 7 | ADAPTAR | Comentários e linhas vazias numa tabela lateral de posições em vez de árvore sem perdas ([extra.rs](https://github.com/gleam-lang/gleam/blob/main/compiler-core/src/parse/extra.rs)) | `ge fmt` precisa preservar comentários sem poluir `ast.Intent` | `tooling/formatter/`, `compiler/parser/hierarquia.go` |
| 8 | ADOTAR | Snapshot da mensagem renderizada inteira como teste | testes de erro conferem substrings | `compiler/parser/hierarquia_test.go` |
| 9 | ADAPTAR | Code actions a partir de erros, mas vindas da sugestão estruturada do diagnostic (modelo rustc), não reconstruídas no LSP | evita duplicar no LSP futuro o que o compiler já sabe | `compiler/diagnostics`, futuro LSP |
| 10 | ADOTAR | Formatter sem opções que também migra formas antigas | mantém `ge fmt` como está e lhe dá um papel na evolução | `tooling/formatter/` |
| 11 | EVITAR | `hint` como texto livre e ausência de códigos de erro | o Germanio já tem códigos e precisa de correções aplicáveis | `compiler/diagnostics` |
| 12 | EVITAR | Um único arquivo central de erros com milhares de linhas | manter as variantes perto de quem as emite; centralizar só a renderização | `compiler/diagnostics` |
| 13 | EVITAR | Hindley-Milner, múltiplos alvos de geração, type classes | nenhum problema do nível padrão pede isso | — |
| 14 | INVESTIGAR | Se o Germanio deve declarar uma "v1" da sintaxe de intenção, com garantia no estilo do Gleam (linguagem, formatter e ferramentas juntos) só depois da poda | sem marco de estabilidade, usuários não sabem o que vai mudar | `docs/INTENCAO.md` |
