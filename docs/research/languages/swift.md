# Swift: compiler, ferramentas e evolução formal da linguagem

- **Data:** 2026-09-28
- **Status:** pesquisa concluída; sem força normativa (ver `docs/README.md`). Nada aqui foi
  implementado no Germanio.
- **Pergunta principal:** como uma linguagem grande evolui formalmente sem transformar mudanças
  de sintaxe em decisões improvisadas?
- **Não repete:** `docs/research/sintaxe-hierarquica.md` (layout, Elm/rustc, gofmt/Black).

## Fontes consultadas

Repositórios e documentação oficial (lidos em 2026-09-28):

- Processo: https://github.com/swiftlang/swift-evolution/blob/main/process.md
- Template de proposal: https://github.com/swiftlang/swift-evolution/blob/main/proposal-templates/0000-swift-template.md
- Mudanças comumente rejeitadas: https://github.com/swiftlang/swift-evolution/blob/main/commonly_proposed.md
- Vision documents: https://github.com/swiftlang/swift-evolution/tree/main/visions
- SE-0362 (upcoming features): https://github.com/swiftlang/swift-evolution/blob/main/proposals/0362-piecemeal-future-features.md
- SE-0443 (diagnostic groups, controle de warnings): https://github.com/swiftlang/swift-evolution/blob/main/proposals/0443-warning-control-flags.md
- Language Steering Group: https://www.swift.org/language-steering-group/
- Arquitetura do compiler: https://www.swift.org/documentation/swift-compiler/
- Type checker: https://github.com/swiftlang/swift/blob/main/docs/TypeChecker.md
- Request evaluator: https://github.com/swiftlang/swift/blob/main/docs/RequestEvaluator.md
- Diagnostics: https://github.com/swiftlang/swift/blob/main/docs/Diagnostics.md
- Educational notes / grupos: https://github.com/swiftlang/swift/tree/main/userdocs/diagnostics
- SIL: https://github.com/swiftlang/swift/blob/main/docs/SIL/SIL.md
- Lexicon: https://github.com/swiftlang/swift/blob/main/docs/Lexicon.md
- ASTGen (ponte swift-syntax → AST C++): https://github.com/swiftlang/swift/tree/main/lib/ASTGen
- swift-syntax: https://github.com/swiftlang/swift-syntax ;
  SwiftParser: https://github.com/swiftlang/swift-syntax/blob/main/Sources/SwiftParser/SwiftParser.docc/SwiftParser.md
- Anúncio do novo parser (Doug Gregor, 2022): https://forums.swift.org/t/a-new-swift-parser-for-swiftsyntax/59813
- Estado do ParserASTGen (2026): https://forums.swift.org/t/how-to-completely-bypass-legacy-c-parser-validation-when-testing-parserastgen/84210
- swift-format: https://github.com/swiftlang/swift-format
- SourceKit-LSP: https://github.com/swiftlang/sourcekit-lsp
- Library evolution / ABI: https://www.swift.org/blog/library-evolution/
- Problema de desempenho do solver (relato de usuários, sintoma): https://forums.swift.org/t/the-compiler-is-unable-to-type-check-this-expression-in-reasonable-time/79726

Arquivos do Germanio lidos para a comparação: `AGENTS.md` (Documentation Gate),
`GERMANIO_EVOLUTION.md`, `GERMANIO_GAPS.md`, `AGENT_STATE.md` (IMPORTANT_DECISIONS),
`docs/INTENCAO.md` (Autoridade, Como avaliar uma sintaxe, Evolução, Testes normativos,
Pendências), `docs/README.md`, `docs/research/sintaxe-hierarquica.md` (Parte 3),
`compiler/diagnostics/diagnostics.go`, `compiler/parser/parser.go`, `compiler/parser/hierarquia.go`.

---

## Matriz

### Objetivo original
Substituto seguro e moderno de Objective-C para as plataformas Apple, com interoperabilidade
direta com C/Objective-C (o Clang importer é uma fase do compiler:
https://www.swift.org/documentation/swift-compiler/). Hoje também servidor, Linux, Windows,
embarcado e WebAssembly (vision documents em `visions/`).

### Filosofia
"Seguro por padrão, rápido, expressivo". Na prática, a filosofia mais relevante ao Germanio é a
de governança: a linguagem é tratada como um produto cujas mudanças de design passam por um
processo público, com registro escrito de motivação e alternativas.

### Sintaxe
Família C, chaves, sem ponto e vírgula obrigatório. O `commonly_proposed.md` registra que o
Swift **não** trocará chaves por indentação, nem `&&` por `and`, e justifica esta última com
ferramentas: as gramáticas de operador e de identificador são "intentionally partitioned", o
que permite analisar o código sem olhar os imports. É um exemplo de sintaxe decidida por
propriedade de ferramenta, não por gosto.

### Gramática
Documentada no livro *The Swift Programming Language* (seção de referência). O template exige
que uma proposta de sintaxe "show the additions and changes to the Swift grammar"
(template, seção Detailed design). A gramática não é gerada a partir de uma fonte única para
todos os parsers; por isso existem dois parsers (ver Parser).

### Lexer/tokenizer
No compiler C++, "a simple, recursive-descent parser with an integrated, hand-coded lexer"
(https://www.swift.org/documentation/swift-compiler/). No swift-syntax, o lexer produz tokens
com *trivia* (espaços, comentários) anexada, para que a árvore volte ao texto byte a byte.

### Parser
Há dois parsers convivendo há anos:

1. `lib/Parse` (C++), o padrão do compiler.
2. `SwiftParser` (em Swift, no swift-syntax), anunciado em 2022 com o objetivo explícito de
   substituir o C++ primeiro no swift-syntax e depois no compiler
   (https://forums.swift.org/t/a-new-swift-parser-for-swiftsyntax/59813).

A ponte é `lib/ASTGen`, que converte a árvore do swift-syntax na AST C++. Em 2026 o caminho
ASTGen ainda é experimental (`-enable-experimental-feature ParserASTGen`), e o parser C++
continua validando antes (https://forums.swift.org/t/how-to-completely-bypass-legacy-c-parser-validation-when-testing-parserastgen/84210).
Lição central: **uma linguagem madura e bem financiada levou mais de três anos sem conseguir
eliminar o segundo parser**; macros, swift-format e ferramentas usam um, o compiler usa outro.
É o risco que o Germanio já registrou ("vários parsers"), em escala maior.

Recuperação de erro do SwiftParser: o parser **não falha**; tokens exigidos e ausentes viram
tokens *missing* e texto que não casa com a gramática vai para nós *unexpected*; "all errors
are described in the syntax tree itself, and can be diagnosed by a separate pass"
(SwiftParser.md). O anúncio diz que ele produz "no errors regardless of how ill-formed the
input source text is".

### AST
Duas árvores: a árvore sintática *lossless* do swift-syntax (fiel ao texto, com trivia,
missing e unexpected) e a AST semântica do compiler (C++), que o Lexicon chama de "more of a
directed graph". A separação árvore concreta (para ferramentas) × AST (para semântica) é
deliberada.

### Representações intermediárias
SIL, IR de alto nível em SSA entre a AST e o LLVM IR, "used by the Swift compiler for
flow-sensitive diagnostics, optimization, and LLVM IR generation" (Lexicon). SIL *raw* sai do
SILGen; passes obrigatórios produzem SIL *canônico* e emitem diagnósticos de fluxo que afetam
a correção (atribuição definida, retorno, exclusividade), rodados "regardless of the selected
optimization mode" (SIL.md). Ou seja: parte dos erros de usuário só existe numa IR posterior
à AST.

### Análise semântica
`Sema` faz type checking, validação e reescrita de expressões antes do SILGen (Lexicon). O
**request evaluator** transformou perguntas sobre a AST ("qual a superclasse desta classe?")
em requisições preguiçosas e memorizadas, com detecção de ciclos (`-debug-cycles`) e grafo
de dependências que serve à compilação incremental e à IDE
(https://github.com/swiftlang/swift/blob/main/docs/RequestEvaluator.md). A motivação
declarada foi o estado mutável da AST e a dificuldade de saber o que afetou cada decisão.

### Sistema de tipos
Nominal com protocolos, genéricos, tipos de valor e referência, *existentials* (`any`),
concorrência verificada (actors, `Sendable`). Não se aplica diretamente ao Germanio, cujo
usuário não escreve tipos na maioria dos casos.

### Type inference
Por restrições (constraints), em três fases: geração, resolução e aplicação da solução
(TypeChecker.md). Sobrecargas viram *disjunções* e cada termo "is visited separately";
*locators* ligam cada restrição ao nó de origem para diagnóstico. O próprio documento admite:
"Solving the constraint systems generated by the Swift language can, in the worst case,
require exponential time." O sintoma para o usuário é a mensagem "the compiler is unable to
type-check this expression in reasonable time; try breaking up the expression", ainda
relatada em 2025 (fórum citado). Causa estrutural: sobrecarga pesada de operadores e
literais polimórficos combinadas com inferência bidirecional.

### Compiler/interpreter
Fases: Parse → Sema → Clang importer → SILGen → SIL obrigatório → SIL otimizado → IRGen → LLVM
(https://www.swift.org/documentation/swift-compiler/). Há REPL e modo script, secundários.

### Runtime
Runtime com metadados de tipo, contagem de referências e, desde Swift 5.0, embutido nos
sistemas da Apple graças à estabilidade de ABI (ver Backward compatibility).

### Memory management
ARC (contagem automática de referências), com otimizações no SIL; tipos não copiáveis mais
recentes. Não se aplica ao Germanio (Go tem GC e o usuário `.ge` nunca vê memória).

### Standard library
Evolui pelo mesmo processo que a linguagem (process.md: o LSG governa "language and standard
library"). Relevante: a stdlib não é "detalhe de implementação"; adicionar API pública é
decisão de design.

### Package manager
SwiftPM, governado pelo Ecosystem Steering Group com o mesmo mecanismo de proposals
(process.md).

### Formatter
swift-format, construído sobre swift-syntax, com modo formatação e modo lint; incluído na
toolchain desde Swift 6. O README admite: "No default Swift code style guidelines have yet
been proposed. The style that is currently applied by swift-format is just one possibility."
Configurável por `.swift-format` (JSON). Contraste: `ge fmt` já é canônico e não
configurável, o que é melhor para o público do Germanio.

### Linter
O modo `lint` do swift-format; diagnósticos de estilo separados dos do compiler.

### Language server
SourceKit-LSP, "built on top of sourcekitd and clangd" (README). `sourcekitd` é o próprio
compiler exposto como serviço: a IDE pergunta ao mesmo Sema que compila. A formatação no LSP
vem do swift-format (README do swift-format). Resultado: a semântica é compartilhada com o
compiler; a sintaxe para formatação e macros vem da outra árvore.

### IDE tooling
Xcode e SourceKit-LSP consomem o mesmo `sourcekitd`. Macros expandem sobre swift-syntax:
"the macro expansion nodes are represented as SwiftSyntax nodes and a macro generates a
SwiftSyntax tree to be inserted into the source file" (README do swift-syntax).

### Diagnostics
- Declarados em arquivos `.def` por categoria (erro, warning, nota); notas ficam "attached" ao
  erro anterior (Diagnostics.md).
- Estilo normativo: frase curta sem ponto final; formular como regra ("'super.init' cannot be
  called outside of an initializer"); citar nomes; "Omit words that make the diagnostic longer
  without adding information".
- Fix-it: "it must be the single, obvious, and very likely correct way to fix the issue"; deve
  ficar no mesmo arquivo e, idealmente, permitir que o compiler se recupere como se aplicado.
- Warning só quando "the intent of the code is clear *and* it won't immediately result in a crash".
- **Diagnostic groups** (SE-0443, Swift 6.1): identificador estável acima dos IDs internos;
  `-Werror <grupo>`, `-Wwarning <grupo>`; o nome aparece entre colchetes na saída. Cada grupo
  tem uma página em `userdocs/diagnostics/` (por exemplo `exclusivity-violation.md`,
  `existential-any.md`, `error-in-future-swift-version.md`): texto curto, um conceito,
  "beginner friendly", pensado como "teachable moment".

### Testing
O compiler usa testes `lit` com verificação de diagnósticos no próprio arquivo de teste
(anotações `expected-error`) (não verificado nesta pesquisa além do conhecimento geral; não
cito caminho). Há uma suíte de compatibilidade de fonte com projetos reais (não verificado
nesta pesquisa).

### Documentation
Três camadas separadas: o livro (normativo para usuários), `docs/` no repositório do compiler
(para quem mantém) e `userdocs/diagnostics/` (explicação ligada ao erro). As proposals
aceitas servem de registro histórico das decisões e dos motivos.

### Evolution process
Resumo verificado em process.md e no template:

1. **Pitch** no fórum: "an informal, open-ended discussion about the problem you're trying
   to solve".
2. **Proposal** no template, por pull request no swift-evolution.
3. **Implementação prévia obrigatória** para linguagem e stdlib: "a prototype implementation
   before they can be reviewed. This implementation doesn't have to be fully professional
   [...] but it at least has to be a viable proof of concept."
4. **Review** pública conduzida por um *review manager*, por "at least ten days, covering at
   minimum two consecutive weekends". O review manager conduz, não decide.
5. **Decisão** do grupo de trabalho (Language Steering Group para linguagem/stdlib; Ecosystem
   Steering Group para SwiftPM; Testing Workgroup), com o Core Team podendo sobrepor e a
   autoridade final no Project Lead (https://www.swift.org/language-steering-group/).
6. **Status** fechados: Awaiting review, Scheduled for review, Active review, Returned for
   revision, Withdrawn, Rejected, Accepted, Accepted with revisions, Previewing,
   Implemented (com a versão).

Template (seções): Summary of changes (até 50 palavras), Motivation, Proposed solution,
Detailed design (com a mudança de gramática), **Source compatibility**, **ABI
compatibility**, **Implications on adoption**, **Future directions** (sem "we will"),
**Alternatives considered** ("an important part of most proposal documents"),
Acknowledgments; cabeçalho com número, autores, review manager, status, implementação e
*feature flag*.

Escopo: "applies only to the design of features. Neither the implementation of the feature
nor the user documentation for it require evolution review." Fora do escopo: correções,
otimizações, recursos experimentais, design de IDE. Regra de fronteira: mudanças em recurso
**já lançado** exigem aprovação; "Bugs in features that have not yet been officially released
can be freely fixed."

Instrumentos complementares: **vision documents** (direção de uma área antes das proposals:
macros, memory safety, approachable concurrency…), **commonly_proposed.md** (memória das
rejeições, para não reabrir a mesma discussão) e **feature flags** (experimental e upcoming).

### Backward compatibility
- **Fonte:** "Swift code that worked in previous releases of the tools should work in new
  releases" (template). Mudanças que quebram são encenadas: SE-0362 criou
  `-enable-upcoming-feature X`; na próxima versão da linguagem "X will be implied by that
  language version and the compiler flag will be rejected" (design que se autolimpa); o
  código testa com `#if hasFeature(X)`. Distingue *versão das ferramentas* de *modo da
  linguagem* (Swift 5 mode, Swift 6 mode).
- **ABI:** estável desde Swift 5.0; *module stability* e *library evolution* desde 5.1
  (`.swiftinterface`, `@frozen`, mudanças "resilient")
  (https://www.swift.org/blog/library-evolution/). Não se aplica ao Germanio hoje (não há
  binários `.ge` distribuídos separadamente), mas o análogo existe: o **contrato externo** que
  uma aplicação publica (API gerada, esquema do banco, `disponibilize … como "projects"`).

### Principais acertos
- Implementação antes da review: a decisão é tomada sobre algo que roda.
- "Alternatives considered" e "Source compatibility" obrigatórios: toda sintaxe nova precisa
  dizer o que custou e o que quebra.
- Separação de papéis: autor, review manager, grupo que decide.
- Memória institucional explícita: status fechados, número por proposta, `commonly_proposed.md`.
- Diagnósticos com estilo normativo, grupos estáveis e documentação por grupo.
- Parser tolerante com erros descritos na árvore; diagnóstico como passo separado.
- Request evaluator: semântica preguiçosa, memorizada e com ciclos detectados.

### Principais problemas
- **Dois parsers** por anos (C++ e swift-syntax) e duas árvores; ASTGen ainda experimental.
- **Solver exponencial** no pior caso, com mensagem que transfere o problema ao usuário.
- Processo pesado para mudanças pequenas; proposals longas; diversas revisões
  ("Returned for revision", "Accepted with revisions") mostram custo de coordenação.
- swift-format sem estilo oficial: formatação não é fonte única.
- Parte dos diagnósticos só aparece no SIL, longe da AST que a IDE mostra.

### Complexidade acumulada
Linguagem grande (genéricos, protocolos com tipos associados, result builders, macros,
concorrência, ownership), vários modos de linguagem e flags, duas árvores. O processo não
impediu o acúmulo; impediu que o acúmulo fosse **improvisado e sem registro**.

### O que Germanio pode aprender
Ver "Para o Germanio". Em uma frase: o valor do Swift Evolution não é a cerimônia, é obrigar
cada mudança de sintaxe a ter número, motivação, alternativas, impacto de compatibilidade,
implementação provando o design e um status que não se perde.

### O que Germanio NÃO deve copiar
- Inferência por restrições com sobrecarga: o Germanio decide por tabelas fechadas e
  determinísticas; "tempo razoável" não é uma garantia aceitável para `ge check`.
- Review de dez dias, steering groups, review manager separado: não há comunidade para isso;
  copiar a cerimônia criaria processo vazio.
- Estilo de formatação configurável (swift-format): `ge fmt` deve continuar sem opções.
- Duas árvores mantidas por implementações separadas.
- Modos de linguagem múltiplos convivendo (Swift 5/6 mode) antes de haver usuários externos:
  hoje o Germanio pode migrar todos os `.ge` do repositório na mesma unidade de trabalho,
  como fez com a sintaxe hierárquica. Modos só se justificam quando existir código `.ge` de
  terceiros que o projeto não controla.

---

## Comparação: Swift Evolution × processo atual do Germanio

O Germanio já tem as peças, mas espalhadas e sem ciclo de vida:

| Função | Swift | Germanio hoje | Lacuna |
|---|---|---|---|
| Norma | livro + proposals aceitas | `docs/INTENCAO.md` (normativo) | ok |
| Obrigação de consultar a norma | implícita no review | Documentation Gate em `AGENTS.md` (passos 1–7) | ok para agentes; não diz **quando** uma mudança é de design |
| Lacunas conhecidas | issues + pitches | `GERMANIO_GAPS.md` (ID, classe, severidade, status) | IDs duplicados: G57, G58 e G59 aparecem duas vezes cada, com significados diferentes; mistura bug, capability e sintaxe |
| Registro de decisões | proposal aceita | `AGENT_STATE.md` D1–D11 (uma linha cada), `sintaxe-hierarquica.md` Parte 3 (tabela) | decisões sem motivação, alternativas e compatibilidade num lugar só; D-números e G-números não se referenciam |
| Histórico | status "Implemented (versão)" | `GERMANIO_EVOLUTION.md` (linha do tempo) | narrativa depois do fato, não decisão antes |
| Ideias pendentes | pitch / vision | "Pendências da sintaxe hierárquica" em `INTENCAO.md` + G62/G63 repetindo o mesmo | a mesma pendência mora em dois documentos |
| Critério de julgamento | seções do template | "Como avaliar uma sintaxe" (8 critérios ordenados) + checklist de 15 itens | ótimo, mas não está ligado a um artefato que registre a resposta |
| Implementação antes da aceitação | obrigatória | Gate exige documentação e testes na mesma unidade | equivalente, mais forte (mesma unidade de trabalho) |
| Compatibilidade | seção obrigatória | não existe seção; a sintaxe hierárquica tratou a migração caso a caso (Parte 3, decisão 8) | falta a pergunta fixa "o que acontece com os `.ge` existentes e com o contrato externo da aplicação?" |
| Rejeições | `commonly_proposed.md` | "O que Germanio NÃO deve copiar" nas pesquisas; nenhuma lista de rejeitados | risco de reabrir discussões |
| Escopo | só design; bugs livres | Gate vale para "qualquer mudança de comportamento" | não distingue correção de bug de mudança de linguagem, então ou tudo vira processo ou nada vira |

Conclusão: o Germanio não precisa de mais regras; precisa de **um artefato por decisão de
linguagem** que concentre o que hoje está distribuído, e de uma regra de escopo que diga
quando ele é obrigatório.

## Proposta: formato leve de proposta para o Germanio

Nome sugerido: **Proposta de linguagem (PL)**. Local sugerido: `docs/propostas/NNNN-nome.md`
(não criado nesta pesquisa). Numeração sequencial, nunca reutilizada.

### Quando é obrigatória (escopo)

Obrigatória quando a mudança altera o que um `.ge` significa ou aceita:

- nova palavra, seção, frase ou modificador; nova inferência; mudança de padrão
  (default) ou de segurança padrão; mudança na tabela de seções ou no layout;
- remoção ou mudança de significado de algo que algum `.ge` do repositório usa;
- mudança no contrato externo gerado (nomes de rota, formato de erro, esquema).

Dispensada (como no Swift): correção de bug de algo que contradiz `INTENCAO.md` (vai para
`GERMANIO_GAPS.md` como BUG), otimização, mensagens de diagnóstico que não mudam aceitação,
tooling que não muda significado. Na dúvida: se `ge explain` mostraria um fato diferente para
o mesmo `.ge`, é proposta.

### Cabeçalho

```text
PL-0007 — verbos que ligam e desligam uma condição
Status: rascunho | em teste | aceita | rejeitada | retirada | implementada (commit)
Lacunas: G63   Decisões substituídas: —   Nível: 1 (padrão)
```

### Seções (curtas; a proposta inteira deve caber em uma ou duas telas)

1. **Intenção humana** (até 50 palavras, como o "Summary" do Swift): o que a pessoa quer
   dizer e hoje não consegue, com a frase que ela escreveria.
2. **Hoje**: como se expressa agora e o custo (conceitos técnicos, repetição, lógica nível 3).
3. **Proposta**: exemplo `.ge` hierárquico e a frase plana equivalente.
4. **Semântica**: os fatos que a construção produz no `ast.App`, o que `ge explain` mostra
   (declarado/inferido, origem) e o que `ge check` recusa.
5. **Avaliação** pelos 8 critérios de "Como avaliar uma sintaxe", na ordem, uma linha cada;
   e o checklist de design de novas construções, marcando só o que não é óbvio.
6. **Camada**: domínio, core ou adaptador; que capability genérica é criada; um segundo
   domínio onde ela serve (teste de generalização).
7. **Compatibilidade**: `.ge` do repositório afetados (`grep` com o número), contrato externo
   afetado, e o plano (migração na mesma unidade, ou coexistência com prazo).
8. **Alternativas consideradas**: pelo menos uma, e por que perdeu. Obrigatória.
9. **Testes normativos**: linhas novas para a tabela de `INTENCAO.md` › Testes normativos.
10. **Fora do escopo / direções futuras**: sem "faremos".

### Ciclo de vida

- *rascunho* → *em teste* quando existe implementação em branch com testes (equivale à
  exigência de protótipo do Swift) → *aceita* pela pessoa responsável pelo design
  (hoje o mantenedor; um agente nunca aceita a própria proposta) → *implementada* quando o
  commit que atualiza `INTENCAO.md`, testes e exemplos entra.
- Quando aceita, a norma passa para `INTENCAO.md`; a proposta fica como registro do **porquê**
  e não é normativa (como proposals antigas no Swift).
- Rejeitadas ficam com o motivo; um índice curto das rejeitadas funciona como
  `commonly_proposed.md`.

### Encaixe nos documentos existentes

- `GERMANIO_GAPS.md` continua sendo o registro de lacunas; uma lacuna de linguagem aponta
  para a PL que a resolve. Corrigir os IDs duplicados (G57–G59) e adotar a regra "ID nunca
  reutilizado".
- "Pendências" de `INTENCAO.md` passa a listar apenas títulos com o número da PL ou da
  lacuna, para que a pendência exista em um único lugar.
- `AGENT_STATE.md` › IMPORTANT_DECISIONS cita a PL quando a decisão é de linguagem.
- `GERMANIO_EVOLUTION.md` continua como narrativa medida; cada etapa cita a PL.
- Documentation Gate, passo 1: acrescentar "se a mudança está no escopo de proposta, a PL
  existe e está *em teste* ou *aceita*".

---

## Para o Germanio

| Classificação | Lição | Problema concreto do Germanio | Arquivo afetado |
|---|---|---|---|
| ADOTAR | Um artefato numerado por decisão de linguagem com seções fixas: intenção, alternativas, compatibilidade, testes (template do Swift Evolution) | decisões espalhadas entre `AGENT_STATE.md` D1–D11, `sintaxe-hierarquica.md` Parte 3 e `GERMANIO_EVOLUTION.md`, sem alternativas nem impacto registrados | novo `docs/propostas/`; `AGENTS.md` (Gate, passo 1) |
| ADOTAR | Regra de escopo: só mudanças de significado passam por proposta; bugs contra a norma são livres (process.md) | o Gate vale para "qualquer mudança de comportamento" e não distingue bug de design | `AGENTS.md` |
| ADOTAR | Implementação com testes antes da aceitação ("viable proof of concept") | já praticado; formalizar o status *em teste* evita aceitar sintaxe só no papel | `docs/INTENCAO.md` › Evolução |
| ADOTAR | Status fechados e IDs nunca reutilizados | `GERMANIO_GAPS.md` tem G57, G58 e G59 duplicados | `GERMANIO_GAPS.md` |
| ADOTAR | Lista de rejeitados com motivo (`commonly_proposed.md`) | discussões de sintaxe podem reabrir sem memória; as seções "não copiar" das pesquisas não são consultáveis como lista | `docs/propostas/rejeitadas.md` (sugerido) |
| ADAPTAR | Seção "Source compatibility" vira "Compatibilidade": `.ge` do repositório + contrato externo gerado (o análogo Germanio da ABI) | a migração da sintaxe hierárquica foi decidida caso a caso; mudanças no resolver podem alterar rotas e esquema sem pergunta fixa | template da PL; `docs/INTENCAO.md` |
| ADAPTAR | Diagnostic groups com página educativa curta por grupo (SE-0443, `userdocs/diagnostics`) | `diagnostics.Explanations` só cobre códigos do núcleo estrito; os erros do dialeto de aplicação não têm código. Além disso GE1002 (emitido pelo parser do núcleo estrito, `compiler/parser/germanio.go`) manda usar "dois espaços por nível", enquanto a forma canônica do dialeto de aplicação é 4 espaços (`INTENCAO.md` › Layout): dois front-ends, duas convenções de layout | `compiler/diagnostics/diagnostics.go`, `compiler/parser/hierarquia.go` (`teach`) |
| ADAPTAR | Regras de redação de diagnóstico e de fix-it ("single, obvious, and very likely correct") | "Como corrigir" existe, mas não há critério de quando oferecer uma correção automática (futuro LSP) | `compiler/diagnostics/`; `docs/INTENCAO.md` › Erros educativos |
| ADAPTAR | Vision document curto antes de uma família de propostas | as seções de página (`topo`, `vazio`, `gráfico`) são "direção" dentro da norma; uma visão separada evitaria misturar direção e contrato | `docs/INTENCAO.md` › Pendências |
| ADOTAR | Parser que nunca falha: missing/unexpected na árvore, diagnóstico como passo separado (SwiftParser) | o parser de aplicação para no primeiro erro e devolve `fmt.Errorf` em texto (`parser.go` `errorf`); um LSP precisa de árvore parcial e de todos os erros | `compiler/parser/parser.go`, `compiler/parser/hierarquia.go` |
| EVITAR | Dois parsers para a mesma linguagem (C++ e swift-syntax, ASTGen experimental anos depois) | já há o núcleo estrito, o dialeto de aplicação, a gramática TextMate e o analisador por linhas de `tooling/intelligence` | ver `typescript.md` › Arquitetura recomendada |
| EVITAR | Inferência por restrições com busca e sobrecarga (solver exponencial) | as inferências do Germanio (tipo pelo nome, pessoas, singular) devem continuar tabelas determinísticas explicáveis por `ge explain` | `compiler/parser/resolver.go` |
| EVITAR | Formatter configurável sem estilo oficial | `ge fmt` é canônico; manter sem opções | `tooling/formatter/` |
| EVITAR | Modos de linguagem e flags de recurso enquanto todo `.ge` relevante está no repositório | migrar na mesma unidade é mais simples e já funcionou | — |
| INVESTIGAR | Request evaluator (perguntas semânticas preguiçosas, memorizadas, com ciclos detectados) como forma do resolver servir `ge explain` e um LSP sem recalcular o app inteiro | hoje `Compilar` resolve tudo de uma vez; aceitável até o tamanho do GitLab, a medir antes de mudar | `compiler/parser/resolver.go`, `runtime/engine.go` |
| INVESTIGAR | Upcoming feature flags autolimpantes (SE-0362) para o dia em que existir `.ge` de terceiros | ainda não há usuários externos; registrar o gatilho, não implementar | — |
