# Rust: compiler, diagnostics e tooling

- **Data:** 2026-09-28
- **Status:** estudo concluído; foco principal em diagnostics. Nada aqui é normativo para o Germanio; as decisões ficam em `docs/INTENCAO.md`.
- **Pergunta principal:** como Rust transforma erros complexos em diagnostics úteis e educacionais, e o que isso implica para o formato de 4 partes do Germanio (`compiler/diagnostics`) e para o exemplo `projetos / developer / excluir`.

## Fontes consultadas

- rustc dev guide, overview do compiler: https://rustc-dev-guide.rust-lang.org/overview.html
- rustc dev guide, queries: https://rustc-dev-guide.rust-lang.org/query.html
- rustc dev guide, name resolution: https://rustc-dev-guide.rust-lang.org/name-resolution.html
- rustc dev guide, THIR: https://rustc-dev-guide.rust-lang.org/thir.html
- rustc dev guide, diagnostics (estrutura, estilo, Applicability, lint levels, error codes): https://rustc-dev-guide.rust-lang.org/diagnostics.html
- rustc dev guide, diagnostic structs (`#[derive(Diagnostic)]`, `Subdiagnostic`): https://rustc-dev-guide.rust-lang.org/diagnostics/diagnostic-structs.html
- rustc dev guide, error codes: https://rustc-dev-guide.rust-lang.org/diagnostics/error-codes.html
- rustc dev guide, `ErrorGuaranteed`: https://rustc-dev-guide.rust-lang.org/diagnostics/error-guaranteed.html
- rustc dev guide, translation: https://rustc-dev-guide.rust-lang.org/diagnostics/translation.html
- rustc dev guide, LintStore: https://rustc-dev-guide.rust-lang.org/diagnostics/lintstore.html
- rustc dev guide, UI tests: https://rustc-dev-guide.rust-lang.org/tests/ui.html
- API `rustc_errors::Applicability`: https://doc.rust-lang.org/nightly/nightly-rustc/rustc_errors/enum.Applicability.html
- API `rustc_span::edit_distance::find_best_match_for_name`: https://doc.rust-lang.org/nightly/nightly-rustc/rustc_span/edit_distance/fn.find_best_match_for_name.html
- Formato JSON dos diagnostics: https://doc.rust-lang.org/rustc/json.html
- Compiler team MCP #959, "Remove the fluent files" (aceito, jan/2026): https://github.com/rust-lang/compiler-team/issues/959
- `cargo fix`: https://doc.rust-lang.org/cargo/commands/cargo-fix.html
- `rustfix`, enum `Filter` (código-fonte): https://github.com/rust-lang/cargo/blob/master/crates/rustfix/src/lib.rs
- Clippy, índice de lints: https://rust-lang.github.io/rust-clippy/master/index.html
- rustfmt, `Configurations.md` (contagem feita no arquivo bruto): https://github.com/rust-lang/rustfmt/blob/master/Configurations.md
- Style guide, style editions: https://doc.rust-lang.org/nightly/style-guide/editions.html
- RFC 3309 (Style Team): https://rust-lang.github.io/rfcs/3309-style-team.html
- RFC 3338 (style evolution): https://rust-lang.github.io/rfcs/3338-style-evolution.html
- RFC 2052 (epochs/editions): https://rust-lang.github.io/rfcs/2052-epochs.html
- Edition guide: https://doc.rust-lang.org/edition-guide/editions/index.html
- RFC 1105 (API evolution): https://rust-lang.github.io/rfcs/1105-api-evolution.html
- Processo de RFCs (README): https://github.com/rust-lang/rfcs/blob/master/README.md
- rust-analyzer, arquitetura: https://github.com/rust-lang/rust-analyzer/blob/master/docs/book/src/contributing/architecture.md
- Código do Germanio lido: `compiler/diagnostics/diagnostics.go`, `compiler/parser/hierarquia.go` (`teach`, `dataSection`, `access`, `suggest`), `compiler/parser/parser.go` (`errorf`), `compiler/parser/germanio.go`, `docs/INTENCAO.md` (seção "Erros").

`docs/research/sintaxe-hierarquica.md` já registra que o rustc separa o erro do `help` e manda o detalhe para `--explain`. Este documento não repete isso: desce ao mecanismo (estrutura de dados, Applicability, testes de saída, formato de máquina) e o aplica ao caso concreto.

---

## Matriz

### Objetivo original
Linguagem de sistemas com segurança de memória sem coletor de lixo e com abstrações de custo zero. Não se aplica diretamente ao Germanio; o que interessa é a engenharia de compiler e tooling de um projeto grande e de vida longa.

### Filosofia
Estabilidade sem estagnação: mudanças incompatíveis vão para uma edition opt-in, e crates de editions diferentes interoperam ([edition guide](https://doc.rust-lang.org/edition-guide/editions/index.html)). No diagnostic, a filosofia implícita é que a mensagem é parte do produto e é testada como saída (UI tests).

### Sintaxe
Família C, com expressões em todo lugar, macros, atributos, genéricos e lifetimes. Não se aplica ao Germanio (público leigo, sintaxe por indentação).

### Gramática
Não há gramática formal normativa completa; a referência é a Rust Reference e o parser do rustc. O parser aceita um superconjunto da gramática para poder se recuperar e explicar ("parse a superset of Rust's grammar, while also emitting an error type", [overview](https://rustc-dev-guide.rust-lang.org/overview.html)). Lição: aceitar formas erradas conhecidas *de propósito*, para diagnosticá-las com precisão.

### Lexer/tokenizer
Lexer de baixo nível (`rustc_lexer`) que produz tokens sem erro fatal, e uma camada acima que os converte ([overview](https://rustc-dev-guide.rust-lang.org/overview.html)). Todo token carrega um `Span` (intervalo de bytes no arquivo), não só linha e coluna.

### Parser
Descendente recursivo, com recuperação de erro. Constrói AST; a expansão de macros e a resolução de imports acontecem intercaladas ([overview](https://rustc-dev-guide.rust-lang.org/overview.html)).

### AST
Árvore sintática perto do código escrito, com spans. É descartada após o lowering para HIR.

### Representações intermediárias
AST → HIR (desaçucarado: `for`, `async` etc.) → type check sobre HIR → THIR (HIR totalmente tipado; "each body of THIR is only stored temporarily", usado para construir MIR, checar exaustividade e unsafety, [THIR](https://rustc-dev-guide.rust-lang.org/thir.html)) → MIR (grafo de fluxo de controle, onde roda o borrow checker) → LLVM IR ([overview](https://rustc-dev-guide.rust-lang.org/overview.html)). Cada camada existe para uma análise; os spans atravessam todas, e é por isso que um erro do borrow checker em MIR ainda aponta para o texto do usuário.

Paralelo no Germanio: `.ge` → linhas (`layoutTree`) → frases planas → `ast.Intent` → `ast.App` (`compiler/parser/resolver.go`). O equivalente ao "span atravessa as camadas" já existe parcialmente: `ast.Intent` guarda o contexto hierárquico, e `ge explain` mostra arquivo:linha e caminho. Falta o intervalo (início e fim) e falta que o resolver, e não só o parser, emita diagnostics com essa origem.

### Análise semântica
Resolução de nomes em duas fases: durante a expansão (imports e macros) e a "late resolution" sobre a AST expandida, com pilhas de `Rib` por namespace ([name resolution](https://rustc-dev-guide.rust-lang.org/name-resolution.html)). Tipos e valores vivem em namespaces separados. Para nomes não resolvidos, o rustc procura candidatos em outros módulos e crates (`lookup_import_candidates`) e em outros namespaces, o que gera mensagens como "expected value, found struct". Este é o mecanismo mais relevante para o caso `developer`: **o nome não existe no namespace esperado (seções), mas existe em outro (papéis)**, e é exatamente isso que produz a hipótese útil.

O rustc evita cascata de erros com `ErrorGuaranteed`, um tipo que só pode ser construído quando um erro já foi emitido, e com tipos de erro que se propagam em silêncio ([ErrorGuaranteed](https://rustc-dev-guide.rust-lang.org/diagnostics/error-guaranteed.html)).

### Sistema de tipos
Nominal, com traits, genéricos monomorfizados, lifetimes e ownership. Não se aplica: o Germanio não expõe tipos ao leigo além de tipos de campo.

### Type inference
Inferência local dentro de corpos de função (estilo Hindley-Milner estendido com resolução de traits); assinaturas são explícitas. Lição conceitual: inferir dentro, declarar nas fronteiras. No Germanio as "fronteiras" já são declaradas por intenção (campos, papéis), então não há ação.

### Compiler/interpreter
Compiler orientado a queries, não a passes: "the results of the queries are cached on disk so that the compiler can tell which queries' results changed from the last compilation and only redo those" ([overview](https://rustc-dev-guide.rust-lang.org/overview.html)). Cada query registra dinamicamente de quais outras depende, o que torna o incremental (red-green) correto ([queries](https://rustc-dev-guide.rust-lang.org/query.html)). Para o Germanio, o tamanho dos programas não justifica um sistema de queries; justificaria apenas se um LSP precisar recalcular por arquivo (INVESTIGAR, não antes).

### Runtime
Mínimo (sem GC). Não se aplica.

### Memory management
Ownership e borrow checking em MIR. Não se aplica.

### Standard library
`core`/`alloc`/`std` em camadas. Não se aplica.

### Package manager
Cargo: manifesto declarativo, lockfile, resolução de versões semânticas e comandos que unificam build, teste, docs, fmt e fix. RFC 1105 distingue mudança "major" de breaking change tolerável ("all major changes are breaking, but not all breaking changes are major", [RFC 1105](https://rust-lang.github.io/rfcs/1105-api-evolution.html)). Para o Germanio, o relevante é o `cargo fix` (abaixo, em Diagnostics).

### Formatter
rustfmt segue um guia de estilo mantido pelo Style Team (RFC 3309) e evolui por *style editions* (RFC 3338): "changes to the default Rust style only appear in style editions" ([style guide](https://doc.rust-lang.org/nightly/style-guide/editions.html)); `style_edition` pode ser atualizado separadamente da edition da linguagem. Custo acumulado: o `Configurations.md` atual lista 87 opções, das quais 29 marcadas estáveis e 55 instáveis (contagem feita no arquivo bruto do repositório em 2026-09-28). Configurabilidade é dívida: cada opção é um formato a mais para testar. O `ge fmt` sem opções está certo; o que vale copiar é o *versionamento* do estilo, não as opções.

### Linter
Lints nativos do rustc com níveis `allow`, `warn`, `deny`, `forbid` ("forbid: never the default", [diagnostics](https://rustc-dev-guide.rust-lang.org/diagnostics.html)), declarados com `declare_lint!` e registrados num `LintStore` ([LintStore](https://rustc-dev-guide.rust-lang.org/diagnostics/lintstore.html)). Clippy agrupa 834 lints em categorias: `correctness` (deny), `suspicious`, `style`, `complexity`, `perf` (warn), `pedantic`, `restriction`, `nursery`, `cargo` (allow) ([Clippy](https://rust-lang.github.io/rust-clippy/master/index.html)). O modelo útil é o da *categoria com nível padrão*, não a quantidade.

### Language server
rust-analyzer é um front-end separado do rustc, com três camadas: `syntax` (árvore sem perdas sobre `rowan`), `hir` (resolução, expansão, inferência sobre queries salsa) e `ide` (API de tipos simples com "editor terminology") ([arquitetura](https://github.com/rust-lang/rust-analyzer/blob/master/docs/book/src/contributing/architecture.md)). Invariantes documentados: "syntax tree is a value type"; árvores "do not enforce well-formedness"; o parser nunca falha e devolve `(T, Vec<Error>)`; "typing inside a function's body never invalidates global derived data"; cada requisição LSP é protegida por `catch_unwind`.

O custo: rustc e rust-analyzer são dois front-ends que precisam concordar. É o mesmo risco que o Germanio já tem (regex do VS Code, formatter, parser). A lição do rust-analyzer é o *formato* da árvore, não ter dois compilers.

### IDE tooling
Os diagnostics do rustc saem em JSON com spans, labels e `suggested_replacement` + `suggestion_applicability` ([JSON](https://doc.rust-lang.org/rustc/json.html)); editores e o rust-analyzer transformam isso em quick fixes. A correção é dado, não texto.

### Diagnostics
Ver a seção dedicada abaixo.

### Testing
UI tests em `tests/ui`: a saída do compiler é comparada com arquivos `.stderr`/`.stdout`; `--bless` regrava os snapshots, que o autor revisa; anotações `//~ ERROR` no fonte declaram onde cada erro deve sair; caminhos e números de linha são normalizados (`$DIR`, `LL`). A diretiva `run-rustfix` aplica as sugestões, compara com um arquivo `.fixed` e verifica que o resultado compila ([UI tests](https://rustc-dev-guide.rust-lang.org/tests/ui.html)). Consequência: toda mudança de mensagem aparece no diff de revisão.

### Documentation
Cada código de erro tem um Markdown em `rustc_error_codes/src/error_codes/E<código>.md`, cujos exemplos são testados pelo gerador do error index; `rustc --explain E0308` mostra o texto ([error codes](https://rustc-dev-guide.rust-lang.org/diagnostics/error-codes.html)). A explicação diz *por que* o código não é aceito, não só como consertar.

### Evolution process
RFC para "any semantic or syntactic change to the language that is not a bugfix" e para "removing language features"; discussão em PR, Final Comment Period de dez dias com disposição merge/close/postpone; aceitação não implica prioridade de implementação; tracking issue separado ([RFCs](https://github.com/rust-lang/rfcs/blob/master/README.md)). Mudanças internas do compiler usam Major Change Proposals (ex.: MCP #959).

### Backward compatibility
Editions (2015, 2018, 2021, 2024): opt-in por crate, "crates in one edition must seamlessly interoperate with those compiled with other editions", mudanças "skin deep" porque tudo compila para a mesma representação interna; migração por `cargo fix --edition` ([edition guide](https://doc.rust-lang.org/edition-guide/editions/index.html), [RFC 2052](https://rust-lang.github.io/rfcs/2052-epochs.html)). O mecanismo de migração é o mesmo das sugestões de diagnostic: lints de migração com sugestões aplicáveis.

### Principais acertos
1. Diagnostic como dado estruturado (níveis, spans primário e secundários, subdiagnostics, sugestões com Applicability), renderizado depois para terminal ou JSON.
2. Sugestões classificadas por confiança, com uma ferramenta (`cargo fix`/`rustfix`) que só aplica o nível seguro.
3. Saída de erro testada como snapshot, incluindo a aplicação das correções.
4. Explicação longa por código, com exemplos testados.
5. Compatibilidade por editions com migração automática pelas próprias sugestões.

### Principais problemas
1. Dois front-ends (rustc e rust-analyzer) que divergem em casos de borda.
2. Infraestrutura de tradução que não se pagou: o dev guide admite que a infraestrutura Fluent "causes some friction" e está "pending a redesign" ([translation](https://rustc-dev-guide.rust-lang.org/diagnostics/translation.html)); o MCP #959 (aceito em janeiro de 2026) remove os arquivos `.ftl` e volta as mensagens para dentro das structs, porque "it is unclear how translatable the messages ported to Fluent will be in practice" ([MCP #959](https://github.com/rust-lang/compiler-team/issues/959)).
3. Formatter com dezenas de opções instáveis.
4. Mensagens longas em casos de traits e lifetimes, onde a teoria vaza para o usuário.

### Complexidade acumulada
Alta: IRs múltiplas, queries, macros, 834 lints no Clippy, dois front-ends, processo formal com times. Parte disso é inerente a uma linguagem de sistemas; parte (tradução, opções de rustfmt) é custo de generalidade não pedida.

### O que Germanio pode aprender
Ver "Para o Germanio". Resumo: estrutura de dados do diagnostic, Applicability, testes de saída com `.fixed`, busca em outro namespace como fonte de hipótese, `ErrorGuaranteed` para não cascatear, versionamento do estilo junto do formato.

### O que Germanio NÃO deve copiar
- Sistema de queries e incremental em disco: o Germanio compila um app inteiro em milissegundos; o custo não se paga.
- Infraestrutura de tradução separada das mensagens (Fluent): o próprio Rust está recuando. O Germanio é multilíngue nas *palavras-chave*; as mensagens devem ficar ao lado da struct, com tradução como tabela simples só quando houver demanda real.
- Estilo de mensagem do rustc (minúscula, sem ponto, jargão como "expected X, found Y"): é escrito para programadores. O público do Germanio exige frases completas.
- Opções de formatação.
- Um segundo front-end para o editor.
- Clippy como ferramenta separada com centenas de lints: o Germanio deve ter poucos avisos, todos com correção.

---

## Diagnostics em profundidade

### 1. O diagnostic é uma estrutura, não uma string

No rustc um diagnostic tem nível, código opcional, mensagem principal que "stands on its own", um span primário (onde está o problema) e spans secundários (contexto), cada um com label opcional, e subdiagnostics (`note`, `help`, sugestões) ([diagnostics](https://rustc-dev-guide.rust-lang.org/diagnostics.html)). Desde as diagnostic structs, a definição é declarativa e separada da emissão: `#[derive(Diagnostic)]` com `#[primary_span]`, `#[label]`, `#[note]`, `#[help]` e `#[suggestion(code = "...", applicability = "...")]`; campos `Option<T>` tornam uma parte condicional; a emissão é `tcx.dcx().emit_err(Struct { ... })` ([diagnostic structs](https://rustc-dev-guide.rust-lang.org/diagnostics/diagnostic-structs.html)).

O formato JSON mostra o modelo final: `message`, `code {code, explanation}`, `level`, `spans[]` com `byte_start`/`byte_end`, `line_start`/`line_end`, `column_start`/`column_end`, `is_primary`, `label`, `suggested_replacement`, `suggestion_applicability`, e `children[]` ([JSON](https://doc.rust-lang.org/rustc/json.html)).

### 2. Applicability: o contrato de confiança

Texto da API ([Applicability](https://doc.rust-lang.org/nightly/nightly-rustc/rustc_errors/enum.Applicability.html)):

- `MachineApplicable`: "definitely what the user intended, or maintains the exact meaning of the code. This suggestion should be automatically applied."
- `MaybeIncorrect`: "may be what the user intended, but it is uncertain. The suggestion should result in valid Rust code if it is applied."
- `HasPlaceholders`: contém lacunas como `(...)`; "cannot be applied automatically".
- `Unspecified`: desconhecida.

O contrato é aplicado por ferramenta: o `rustfix` tem `Filter::MachineApplicableOnly` e `Filter::Everything` ([rustfix, lib.rs](https://github.com/rust-lang/cargo/blob/master/crates/rustfix/src/lib.rs)); o `cargo fix` aplica as sugestões, recusa rodar com mudanças não commitadas sem `--allow-dirty` e roda `cargo check` depois para mostrar o que sobrou ([cargo fix](https://doc.rust-lang.org/cargo/commands/cargo-fix.html)). Que o `cargo fix` use `MachineApplicableOnly` por padrão é o comportamento esperado a partir desse enum, mas não conferi a linha exata no código do cargo (não verificado).

Duas condições distintas estão embutidas em `MachineApplicable`: (a) certeza de intenção **ou** (b) preservação exata de significado. Para o Germanio, a (b) é a única que pode ser verificada mecanicamente, e é a mesma prova que o `ge fmt` já faz (reparsear e comparar os fatos).

### 3. Estilo das mensagens

O guia pede mensagem factual, identificadores entre crases, "invalid" em vez de "illegal", sugestões que "not be a question" (nada de "did you mean"), sem "the following"/"as shown", porque o span mostra o lugar ([diagnostics](https://rustc-dev-guide.rust-lang.org/diagnostics.html)). Rust usa o span como parte da frase: a mensagem principal diz o quê, o label diz o que aquele pedaço é, o `help` diz o que fazer.

### 4. Hipóteses a partir de outros namespaces e de nomes parecidos

- `find_best_match_for_name` limita a distância de edição a um terço do tamanho da palavra e compara sem caixa ([edit_distance](https://doc.rust-lang.org/nightly/nightly-rustc/rustc_span/edit_distance/fn.find_best_match_for_name.html)). O Gleam copiou esse algoritmo (ver `gleam.md`).
- Nome não resolvido é procurado em outros namespaces e em outros módulos ([name resolution](https://rustc-dev-guide.rust-lang.org/name-resolution.html)). "Achei, mas é outra coisa" produz uma mensagem melhor que "não achei".

O `suggest` do Germanio (`compiler/parser/hierarquia.go`) usa distância fixa `< 3` contra a lista de seções. Para `developer` não há seção parecida, então hoje a mensagem só lista as seções válidas. O que falta é a segunda fonte: o namespace dos papéis e o namespace das ações.

### 5. Não cascatear

`ErrorGuaranteed` prova, no tipo, que um erro já saiu, e o compiler segue com tipos de erro sem emitir diagnostics derivados ([ErrorGuaranteed](https://rustc-dev-guide.rust-lang.org/diagnostics/error-guaranteed.html)). No Germanio isso importa quando houver vários diagnostics por execução: uma linha de seção inválida não deve gerar um segundo erro "a ação excluir não tem sujeito".

### 6. Códigos e explicação longa

Código só quando a explicação "provides value beyond the inline message" ([diagnostics](https://rustc-dev-guide.rust-lang.org/diagnostics.html)); cada código tem Markdown com exemplos testados ([error codes](https://rustc-dev-guide.rust-lang.org/diagnostics/error-codes.html)). O Germanio tem `diagnostics.Explanations`, um mapa de strings de uma linha em Go, e `germanio explicar <GE0000>`. As explicações não têm exemplo testado, e os erros do dialeto de aplicação (`teach`) não têm código nenhum.

### 7. Testar a saída

UI tests com `.stderr` e `.fixed` ([UI tests](https://rustc-dev-guide.rust-lang.org/tests/ui.html)) transformam "a mensagem é boa" em algo revisável. O teste atual do Germanio (`hierarquia_test.go`, caso "recuo sem pai") confere substrings ("Por quê", "4 = acesso"); não compara a saída inteira nem aplica correções.

---

## Comparação com o formato de 4 partes do Germanio

Estado atual verificado no código:

| Aspecto | rustc | Germanio hoje |
|---|---|---|
| Representação | struct com spans, children, sugestões | `diagnostics.Diagnostic` (código, uma posição, textos) no núcleo estrito; no dialeto de aplicação, `teach` monta um texto e `errorf` devolve `fmt.Errorf("%s:%d:%d: %s")` sem código nem struct |
| Onde | intervalo de bytes, primário e secundários com labels | uma posição linha:coluna e um `Context` (caminho hierárquico); o caret aponta um caractere |
| Por quê | `note` | `Reason` / "Por quê:" |
| Como corrigir | `help` e sugestão com texto de substituição | `Fix` / "Como corrigir:" em prosa; nenhuma edição aplicável |
| Confiança | `Applicability` | não existe |
| Vários erros | sim, com supressão de cascata | primeiro erro interrompe (não achei agregação em `compiler/parser`) |
| Máquina | `--error-format=json` | não existe |
| Explicação longa | Markdown por código, exemplos testados | `Explanations` com uma linha por código |
| Teste | snapshot `.stderr` + `.fixed` compilado | `strings.Contains` em partes da mensagem |

O formato de 4 partes (o quê, onde, por quê, como corrigir) corresponde bem ao rustc (mensagem, span, note, help). A diferença está em que o "como corrigir" do Germanio é só prosa. O que o rustc acrescenta é tratar a correção como **edição com confiança declarada**, verificável e aplicável por ferramenta. Uma incoerência pequena: `diagnostics.Explanations["GE1002"]` e `compiler/parser/germanio.go` dizem "dois espaços por nível", enquanto `docs/INTENCAO.md` fixa 4 espaços como forma canônica do dialeto de aplicação. São front-ends diferentes, mas o leigo vê um único Germanio.

---

## O caso `projetos / developer / excluir`

```
projetos
    developer
        excluir
```

Hoje: `dataSection` cai no `case ""` e responde `"developer" não é uma seção de projetos`, lista as seções e tenta `suggest("developer")`, que não acha nada a menos de 3 edições. A mensagem é correta, mas não levanta a hipótese certa.

Mensagem desejada: "Germanio não sabe o que 'developer' representa neste contexto. Talvez você quisesse declarar acesso: projetos / acesso / developer / excluir".

### Mecanismo proposto

**1. Estrutura (em `compiler/diagnostics`).** Estender `Diagnostic` sem quebrar o formato de 4 partes:

- `Span{File; Start, End Pos}` com `Pos{Line, Column, Offset}`; `Primary Span` + `Labels []Label{Span, Text}` (secundários).
- `Suggestions []Suggestion{Message string; Edits []Edit; Applicability; Equivalent []string}` com `Edit{Span, NewText}` (várias edições por sugestão, como o `multipart_suggestion` do rustc) e `Equivalent` com as frases planas que a correção produziria.
- `Applicability`: `Automatica` (equivale a MachineApplicable), `Provavel` (MaybeIncorrect), `ComLacunas` (HasPlaceholders). Sem `Unspecified`: toda sugestão declara a confiança.
- O renderizador de texto continua mostrando o quê / onde / por quê / como corrigir. O "como corrigir" passa a vir de `Suggestion.Message` e do texto já editado; `Equivalent` vira a linha "Equivale a:" que `docs/INTENCAO.md` já exige.
- `teach` passa a construir esse `Diagnostic` e ganha um código (ex.: um código para "linha que não é seção do dado"). O parser devolve `*diagnostics.Diagnostic`, não `fmt.Errorf`.

**2. Span.** Primário: a linha `developer` inteira (coluna 5 até o fim da palavra). Secundário: a linha `projetos` com label "dentro deste dado, o primeiro nível diz de que aspecto se trata". A edição cobre as linhas 2 e 3.

**3. Evidências para a hipótese** (ordem, parando na primeira que der hipótese única):

- (a) Distância de edição contra as seções, com o limite relativo do rustc (um terço da palavra, mínimo 1) em vez de `< 3` fixo. Resolve `acesos`, `tme`.
- (b) Outro namespace: `developer` é um papel. O papel pode estar declarado em outro arquivo, porque "a ordem dos arquivos não muda o resultado" (`docs/INTENCAO.md`). Então esta evidência é do resolver, não do parser. Ou o parser registra a linha como "seção desconhecida pendente" e o resolver, com `ast.Intent.Roles` completo, emite o diagnostic (a "late resolution" do rustc), ou o parser só usa a evidência local de (c).
- (c) Forma do subárvore: todos os filhos de `developer` são ações do vocabulário de `pode` (`excluir`, `ver`, `criar`, `enviar código`). Essa é exatamente a forma de um ator dentro de `acesso`. A evidência é local e determinística.

A hipótese "acesso" só é levantada se (b) ou (c) valer. Com as duas, a mensagem pode afirmar "'developer' é um papel declarado em backend/papeis.ge:3".

**4. A edição.** Inserir a linha `    acesso` antes de `developer` e somar 4 espaços a `developer` e a todo o seu subárvore. Não é preciso mover para um bloco `acesso` que já exista, porque a fusão de blocos (`docs/INTENCAO.md`, "Fusão e conflitos") soma os fatos. A menor edição válida é local.

**5. Verificação especulativa.** Antes de mostrar a sugestão, aplicar as edições em memória, reparsear o arquivo e exigir que: o diagnostic original desapareça; nenhum diagnostic novo apareça naquele subárvore; os fatos produzidos pelo subárvore sejam exatamente `Equivalent` (ex.: `developer pode excluir projetos`). É o `run-rustfix` do rustc feito em tempo de execução, e o `ge fmt` já tem essa mecânica de reparse. Se a verificação falhar, a sugestão é descartada. Uma sugestão que não compila nunca é mostrada.

**6. Applicability: quando é seguro corrigir sozinho.** Pela definição do rustc, `MachineApplicable` exige certeza de intenção ou preservação exata de significado. Este caso não preserva significado: o arquivo não compilava e passaria a **conceder a permissão de excluir**. Regra proposta para o Germanio:

- `Automatica` só para edições que preservam os fatos, verificadas por reparse: tabulação → espaços, recuo inconsistente que tem um único pai possível, sinônimo antigo → forma canônica. É o mesmo critério do `ge fmt`.
- `Provavel` para qualquer edição que cria, remove ou muda fatos. Em especial, **nenhuma edição que cria grant, papel, regra de acesso ou integração é aplicada sem confirmação**, mesmo com distância de edição 1. Aplicar sozinho um `acesos` → `acesso` significa conceder permissões por inferência.
- `ComLacunas` quando a hipótese precisa de um valor que o Germanio não sabe (ex.: "a quem pertence isto?").
- Com mais de uma hipótese sobrevivendo à verificação, mostrar as duas como `Provavel` e não escolher.

Para `developer / excluir`, a sugestão é `Provavel`. Um `ge fix` (ou a ação rápida do editor) aplica só `Automatica` por padrão. A outra opção exige escolha explícita, como o `Filter::Everything` do rustfix.

**7. Saída para o leigo** (renderização em texto do mesmo dado):

```text
backend/projetos.ge:2 — Germanio não sabe o que "developer" representa dentro de projetos.
Por quê: dentro de um dado, cada linha do primeiro nível diz de que aspecto se trata
(tem, pode, acesso, regras...). "developer" é um papel, e abaixo dele há uma ação (excluir).
Como corrigir: talvez você quisesse declarar acesso:
    projetos
        acesso
            developer
                excluir
Equivale a: developer pode excluir projetos
```

O "talvez" é a Applicability `Provavel` dita em português. Quando a sugestão é `Automatica`, a frase é afirmativa ("use 4 espaços; `ge fix` corrige").

**8. Teste.** Um diretório de casos de erro (ex.: `compiler/parser/testdata/erros/*.ge`), cada um com o `.saida` esperado (texto completo) e o `.corrigido` esperado. O teste roda o diagnostic, compara a saída inteira, aplica as sugestões, compara com `.corrigido` e exige que `.corrigido` compile e produza os fatos de `Equivalent`. Uma flag para regravar (o `--bless`) evita manutenção manual.

---

## Para o Germanio

| # | Classe | Lição | Problema do Germanio que resolve | Arquivo afetado |
|---|---|---|---|---|
| 1 | ADOTAR | Diagnostic como struct com span primário/secundários e sugestões com edições ([diagnostic structs](https://rustc-dev-guide.rust-lang.org/diagnostics/diagnostic-structs.html), [JSON](https://doc.rust-lang.org/rustc/json.html)) | dialeto de aplicação devolve `fmt.Errorf` sem código nem estrutura; "como corrigir" é só prosa | `compiler/diagnostics/diagnostics.go`, `compiler/parser/hierarquia.go` (`teach`), `compiler/parser/parser.go` (`errorf`) |
| 2 | ADAPTAR | Applicability, com a regra "automática só preserva fatos; nada que conceda acesso é automático" ([Applicability](https://doc.rust-lang.org/nightly/nightly-rustc/rustc_errors/enum.Applicability.html)) | risco de uma correção automática conceder permissões; editor e `ge fix` precisam saber o que é seguro | `compiler/diagnostics`, futuro `ge fix` em `tooling/gecli/cli.go` |
| 3 | ADOTAR | Verificar a sugestão aplicando-a e reparseando (`run-rustfix`, [UI tests](https://rustc-dev-guide.rust-lang.org/tests/ui.html)) | sugestões em prosa podem estar erradas sem que nada acuse | `compiler/parser/hierarquia.go`, reuso da mecânica de `tooling/formatter` |
| 4 | ADOTAR | Testes de saída completa com arquivo esperado e modo de regravação | testes atuais só conferem substrings; regressões de mensagem passam | `compiler/parser/hierarquia_test.go`, novo `testdata` |
| 5 | ADAPTAR | Hipótese a partir de outro namespace (papéis, ações) e da forma do subárvore ([name resolution](https://rustc-dev-guide.rust-lang.org/name-resolution.html)) | `suggest` só compara com seções; o caso `developer` não recebe hipótese | `compiler/parser/hierarquia.go` (`suggest`, `dataSection`), `compiler/parser/resolver.go` |
| 6 | ADOTAR | Limite de distância relativo (um terço, [edit_distance](https://doc.rust-lang.org/nightly/nightly-rustc/rustc_span/edit_distance/fn.find_best_match_for_name.html)) | limite fixo `< 3` sugere demais em palavras curtas e de menos em longas | `compiler/parser/hierarquia.go` (`suggest`) |
| 7 | ADAPTAR | Vários diagnostics por execução, sem cascata (`ErrorGuaranteed`) | o primeiro erro interrompe; o leigo corrige um por vez | `compiler/parser/hierarquia.go`, `compiler/parser/resolver.go` |
| 8 | ADAPTAR | Explicação longa por código com exemplo testado (`--explain`, [error codes](https://rustc-dev-guide.rust-lang.org/diagnostics/error-codes.html)) | `Explanations` tem uma linha por código; exemplos não são verificados; os erros hierárquicos não têm código | `compiler/diagnostics/diagnostics.go`, `tooling/gecli` (`explicar`) |
| 9 | ADOTAR | Saída de máquina (JSON) dos diagnostics | não há LSP; o VS Code não tem como mostrar quick fix; é o primeiro passo barato antes de um LSP | `tooling/gecli/cli.go`, `vscode-germanio/` |
| 10 | ADAPTAR | Versionar o estilo do formatter junto com a versão do formato, sem opções ([RFC 3338](https://rust-lang.github.io/rfcs/3338-style-evolution.html)) | mudar a saída canônica de `ge fmt` quebraria CI de quem roda `--check` | `tooling/formatter/` |
| 11 | ADAPTAR | Migração por sugestões automáticas quando uma forma for aposentada (`cargo fix --edition`) | o dialeto antigo e o hierárquico coexistem; aposentar formas precisa de migração mecânica | futuro `ge fix`, `compiler/parser` |
| 12 | INVESTIGAR | Árvore sem perdas e parser que nunca falha, como no rust-analyzer ([arquitetura](https://github.com/rust-lang/rust-analyzer/blob/master/docs/book/src/contributing/architecture.md)) | três "parsers" (regex do VS Code, formatter, parser); um LSP futuro precisa de árvore de código quebrado | `compiler/parser/hierarquia.go` (`layoutTree`), `tooling/formatter/` |
| 13 | EVITAR | Infraestrutura de tradução de mensagens separada do código (Fluent; [MCP #959](https://github.com/rust-lang/compiler-team/issues/959)) | tentação de traduzir mensagens para 20 idiomas antes de haver demanda | `compiler/diagnostics` |
| 14 | EVITAR | Estilo de mensagem para programador (minúscula, "expected X, found Y") | público leigo | mensagens em `compiler/parser` |
| 15 | EVITAR | Sistema de queries/incremental e um segundo front-end para o editor | custo sem ganho no tamanho dos apps `.ge` | — |
| 16 | EVITAR | Formatter configurável e linter separado com centenas de regras | cada opção é um formato a mais; o leigo não escolhe lints | `tooling/formatter/` |
| 17 | ADOTAR | (achado local, não lição do Rust) Corrigir a incoerência "dois espaços" vs "4 espaços" nas mensagens de indentação | mensagem contradiz `docs/INTENCAO.md` para quem usa o dialeto de aplicação | `compiler/diagnostics/diagnostics.go` (`GE1002`), `compiler/parser/germanio.go` |
