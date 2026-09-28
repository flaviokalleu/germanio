# Kotlin: estudo dirigido para o Germanio

**Data:** 2026-09-28. **Status:** pesquisa, sem força normativa (a norma é `docs/INTENCAO.md`).
**Pergunta principal:** o que o Kotlin ensina sobre tornar a ausência de valor, a
concorrência e as DSLs hierárquicas seguras sem aumentar os conceitos que uma pessoa precisa
aprender?

## Fontes consultadas

Consultadas nesta sessão (WebFetch):

- [Null safety](https://kotlinlang.org/docs/null-safety.html)
- [Coroutines basics / structured concurrency](https://kotlinlang.org/docs/coroutines-basics.html)
- [Type-safe builders e `@DslMarker`](https://kotlinlang.org/docs/type-safe-builders.html)
- [Kotlin evolution principles](https://kotlinlang.org/docs/kotlin-evolution-principles.html)
- [KEEP (repositório)](https://github.com/Kotlin/KEEP)
- [Gradle: Kotlin DSL primer, seção Limitations](https://docs.gradle.org/current/userguide/kotlin_dsl.html)
- [Ktor: routing](https://ktor.io/docs/server-routing.html)

Do conhecimento prévio, **não verificado nesta sessão**: detalhes do compilador K2 (FIR
como IR de frontend), o formato exato de `data class` (`equals/hashCode/toString/copy/
componentN` gerados) e o histórico do Kotlin LSP oficial de 2025.

Código do Germanio conferido: `compiler/ast/ast.go:331-334`, `runtime/servidor/intencao.go:378-418`,
`runtime/interpreter/interpreter.go:1128-1260`, `runtime/banco/banco.go:262-277`.

---

## Matriz (compacta)

| Item | Kotlin | Relevância para o Germanio |
|---|---|---|
| Objetivo original | Linguagem pragmática para a JVM, interoperável com Java, mais concisa e segura (JetBrains, 2011) | Mostra uma linguagem que venceu por reduzir cerimônia sem romper com o ecossistema |
| Filosofia | "Pragmatismo": features testadas com usuários antes de estabilizar ([evolution principles](https://kotlinlang.org/docs/kotlin-evolution-principles.html)) | Coincide com "testar a capability em outro domínio" (`docs/INTENCAO.md`, Evolução) |
| Sintaxe | Ver seção "Sintaxe" | — |
| Gramática | Gramática publicada (ANTLR) na spec; parênteses de chamada omissíveis com lambda final (trailing lambda) (não verificado nesta sessão) | O trailing lambda é o que permite DSLs de aparência declarativa |
| Lexer / Parser / AST | PSI (árvore da IDE) compartilhada entre compilador e IntelliJ (não verificado) | Reforça A6 (um front-end para todas as ferramentas) |
| IR | FIR (frontend K2) e IR de backend comum a JVM/JS/Native/Wasm (não verificado) | não se aplica |
| Análise semântica | Smart casts: depois de `if (x != null)` o tipo é refinado ([null safety](https://kotlinlang.org/docs/null-safety.html)) | Refinamento por fluxo é o equivalente técnico de "enquanto a condição vale" |
| Tipos / inferência | Estático, inferência local; nulabilidade dentro do tipo (`String` vs `String?`) | Germanio infere tipo pelo nome (`INTENCAO.md`, "Tipo pelo nome") |
| Compiler / runtime | JVM, JS, Native, Wasm; runtime de coroutines é biblioteca (`kotlinx.coroutines`) | não se aplica |
| Memória | GC da plataforma | não se aplica |
| Stdlib | Pequena, extensões sobre coleções Java | não se aplica |
| Package manager | Nenhum próprio; Gradle/Maven | Ver Gradle Kotlin DSL abaixo |
| Formatter / linter | Sem formatter oficial único (ktlint, ktfmt de terceiros) (não verificado) | Contraexemplo: o Germanio já tem `ge fmt` canônico |
| LSP / IDE | Historicamente só IntelliJ; Gradle exige importar o projeto na IDE para ter assistência ([Gradle](https://docs.gradle.org/current/userguide/kotlin_dsl.html)) | Linguagem cuja legibilidade depende da IDE; o Germanio não pode depender disso |
| Diagnostics | Avisos de deprecação com migração automática (`ReplaceWith`) (não verificado) | Ver P7 em `GERMANIO_LESSONS.md` |
| Testing / docs | kotlin.test; KDoc | não se aplica |
| Evolution process | KEEP: proposta numerada, discussão pública, status no cabeçalho (Accepted, Experimental, Stable, Revoked…) ([KEEP](https://github.com/Kotlin/KEEP)); features em preview exigem opt-in explícito; Language Committee decide mudanças incompatíveis | Modelo aproveitável (ver "Para o Germanio") |
| Backward compatibility | Ciclo de deprecação anunciado → aviso → migração por ferramenta → mudança; flags `-language-version`, `-api-version`, `-progressive` ([principles](https://kotlinlang.org/docs/kotlin-evolution-principles.html)) | `-progressive` é "adotar correções já" por opt-in |
| Principais acertos | Nulabilidade no tipo; structured concurrency; data classes; evolução com opt-in | — |
| Principais problemas | NPE continua possível na fronteira com Java (platform types), com `!!` e com inicialização parcial ([null safety](https://kotlinlang.org/docs/null-safety.html)); DSLs com receptores implícitos exigem `@DslMarker` para não aceitar aninhamento errado; Kotlin DSL do Gradle mais lento na primeira execução e com accessors que só existem para plugins do bloco `plugins {}` ([Gradle](https://docs.gradle.org/current/userguide/kotlin_dsl.html)) | — |
| Complexidade acumulada | Coroutines (suspend, scopes, dispatchers, Flow, Channel), sealed/inline/value classes, context receivers → context parameters | A cor `suspend` é mais um conceito por função |
| Aprender | Ver "Para o Germanio" | — |
| Não copiar | Símbolos `?.`, `?:`, `!!`; lambdas com receptor como mecanismo exposto ao autor | — |

---

## Sintaxe

| Pergunta | Kotlin | Germanio hoje (IMPLEMENTADO, com arquivo) | Comparação de custo cognitivo |
|---|---|---|---|
| Dados | `data class Cliente(val nome: String, val email: String?)` gera igualdade, cópia e texto | `cada cliente tem` + linhas; tipo pelo nome (`INTENCAO.md`, "O que existe"); persistência, API e tela derivadas | O Kotlin tira boilerplate de *objeto*; o Germanio tira o de *sistema* (tabela, rota, formulário). O Germanio exige menos conceitos, mas o valor não tem identidade de valor nem cópia: não precisa, porque o registro vive no banco |
| Funções | `fun f(x: Int): Int`, lambdas, extensões, `suspend fun` | Sem funções no nível 1/2; hooks e funções só no nível 3 (`INTENCAO.md`, "Lógica específica") | O Germanio remove "função" do nível padrão; correto para o público |
| Módulos / imports | pacotes + `import`; visibilidade `internal` por módulo Gradle | `importar "backend"` (pasta inteira, ordem alfabética); `importar produtos e pedidos do backend` no frontend (`INTENCAO.md`, "Organização do projeto") | Import explícito no frontend é mais legível que o import implícito por pacote; o import de pasta inteira é convenção (ver ruby.md, autoload) |
| Relações | Não há: referências entre objetos; relações vêm do ORM (Exposed, JPA) | `cliente tem pedidos`, `pedido pertence a cliente` (`compiler/parser/resolver.go`) | O Germanio é mais direto; nada a copiar |
| Estado | `val`/`var`; `StateFlow` na UI | `começa ativo`, `pode bloquear`; estados só por ação; campo `estado` com `System: true` (`resolver.go:408`) | O Germanio é declarativo e fechado; o Kotlin é imperativo |
| Fluxo | `if`/`when` como expressão; smart cast | Condições declarativas (`X arquivado é somente leitura`); `se` só no nível 3 | — |
| Erros | Exceções não checadas; `Result<T>` e `runCatching` na stdlib | Erros de validação reunidos por campo; transação desfeita (`INTENCAO.md`, "Garantias automáticas") | O Germanio é melhor para o público: não há exceção a tratar |
| Concorrência | Coroutines com structured concurrency: filho só em scope; pai espera filhos; falha/cancelamento propagam pela árvore ([coroutines](https://kotlinlang.org/docs/coroutines-basics.html)) | Nível 3: `paralelo`, `timeout`, `chamar_async`, `consultar_paralelo` (`runtime/interpreter/interpreter.go:1128-1260`) | **Germanio errado aqui**, ver abaixo |
| Tipos | Estáticos, nominais | Tipos de campo inferidos pelo nome ou explícitos | — |
| Null | `T` nunca é nulo; `T?` pode ser; o compilador obriga a tratar | Campo é opcional por padrão; `obrigatório` é opt-in (`INTENCAO.md`, "Modificadores") | Padrão invertido em relação ao Kotlin (ver INVESTIGAR) |
| Organização | Gradle multi-módulo; convenção `src/main/kotlin` | `backend/`, `frontend/`, `integracoes/` com papéis fechados | O Germanio impõe mais e explica mais |
| Boilerplate | data class, propriedades, argumentos nomeados com default, extensões, DSLs | Inferência de tipo, rotas, telas, migrations | — |
| Legibilidade em projetos grandes | Boa em código comum; DSLs dependem da IDE para saber *qual receptor* está em escopo ([type-safe builders](https://kotlinlang.org/docs/type-safe-builders.html)) | A hierarquia do Germanio se reduz a frases planas e `ge explain` mostra a origem de cada fato | O ponto mais relevante: ver abaixo |

### Lambdas com receptor versus a sintaxe hierárquica do Germanio

A DSL Kotlin (Gradle, Ktor `routing { route("/order") { get { call.respond… } } }`
([Ktor](https://ktor.io/docs/server-routing.html))) tem a *aparência* da sintaxe hierárquica
do Germanio, mas o significado de cada nome depende de uma pilha de receptores implícitos
resolvida pelo sistema de tipos. Consequências documentadas:

1. Sem `@DslMarker` o aninhamento errado (`head { head { } }`) compila, porque o nome é
   encontrado num receptor externo ([type-safe builders](https://kotlinlang.org/docs/type-safe-builders.html)).
   A correção é uma anotação que o autor da DSL precisa lembrar de pôr.
2. O que está disponível num bloco depende de *quando* o plugin foi aplicado: accessors
   tipados só existem para plugins do bloco `plugins {}`; o resto "will not work with
   type-safe model accessors" ([Gradle](https://docs.gradle.org/current/userguide/kotlin_dsl.html)).
   É uma regra implícita de ordem.
3. A assistência exige importar o projeto na IDE; a primeira execução é mais lenta porque o
   script é compilado ([Gradle](https://docs.gradle.org/current/userguide/kotlin_dsl.html)).

O Germanio já faz o oposto do item 1: cada seção é uma **tabela fechada**, e o que não está
na tabela é erro (`compiler/parser/hierarquia.go`, `dataSection`; lição A2 em
`GERMANIO_LESSONS.md`). Isso é tecnicamente superior para leigos: o escopo é dado pela
estrutura, não por resolução de nomes, e se reduz a frases planas testadas. O risco que o
Kotlin mostra é outro: se o Germanio um dia permitir que *bibliotecas* ou `integracoes/`
acrescentem seções, a tabela deixa de ser fechada e volta o problema do receptor implícito.

### A concorrência do nível 3 do Germanio viola structured concurrency

Conferido em `runtime/interpreter/interpreter.go:1182-1219`: `timeout(f, ms)` lança uma
goroutine e, ao estourar o prazo, devolve `nil` e só escreve um aviso no log. A goroutine
**continua executando** (não há contexto nem cancelamento), podendo gravar depois que a
requisição terminou, fora da transação que `INTENCAO.md` promete para hooks. Função
inexistente também devolve `nil` silenciosamente. No modelo do Kotlin, o filho pertence ao
escopo do pai e é cancelado com ele ([coroutines](https://kotlinlang.org/docs/coroutines-basics.html)).
Não foi verificado se o interpretador é seguro para chamadas concorrentes a
`interp.callFunction` a partir de goroutines (INVESTIGAR com `go test -race`).

---

## Para o Germanio

**ADOTAR — structured concurrency como regra do runtime, invisível ao autor.**
Problema: `timeout`/`paralelo`/`chamar_async` criam goroutines sem escopo; o prazo vencido
não cancela o trabalho; a falha vira `nil`. Arquivo: `runtime/interpreter/interpreter.go:1128-1260`.
Proposta: toda execução de hook recebe o `context.Context` da requisição; toda goroutine
derivada é filha dele; prazo vencido cancela e vira erro educativo, não `nil`; a transação
só fecha depois que todos os filhos terminam. Remove da cabeça: "e se isso continuar
rodando depois?". Não acrescenta sintaxe.

**ADAPTAR — processo de evolução no estilo KEEP, com opt-in de preview.**
Problema: mudanças de linguagem hoje entram direto na norma. Arquivo: `docs/INTENCAO.md`
("Evolução"), `GERMANIO_EVOLUTION.md`. Proposta: proposta numerada com status no cabeçalho
(Exploração, Preview, Estável, Revogada) e construções em preview só ativas com uma linha
explícita no `app.ge` (não verificado se já existe mecanismo equivalente). Complementa P7
(deprecação com prazo) de `GERMANIO_LESSONS.md`. Remove da cabeça: "essa construção vai
mudar?".

**ADAPTAR — ausência de valor explícita na fronteira externa (platform types).**
Problema: o NPE residual do Kotlin nasce na fronteira com código que não declara
nulabilidade ([null safety](https://kotlinlang.org/docs/null-safety.html)). No Germanio a
fronteira é `chamar` (JSON) e `integracoes/`. Proposta: o adaptador declara a forma do que
recebe, e campo ausente vira erro educativo na borda, não `nil` que se propaga. Arquivo:
`runtime/interpreter/interpreter.go` (`chamar`), `integracoes/`. Remove da cabeça: checar
nulo em cada uso.

**EVITAR — expor lambdas com receptor, `?.`, `?:`, `!!` ou qualquer símbolo de nulabilidade.**
O Germanio já resolve escopo por tabela fechada; símbolos adicionariam conceitos. Manter a
tabela fechada também para extensões futuras (A2).

**EVITAR — que a legibilidade dependa da IDE.** O caso Gradle mostra uma linguagem cujo
significado só é visível com o projeto importado. `ge explain` (`tooling/explicar/`) deve
continuar sendo a resposta de linha de comando.

**INVESTIGAR — padrão de opcionalidade dos campos.**
Kotlin torna "pode faltar" explícito; o Germanio torna "não pode faltar" explícito
(`obrigatório`). Para formulários de leigos o padrão atual reduz atrito, mas deixa passar
registros incompletos em silêncio. Não há evidência para inverter. Mínimo proposto:
`ge explain` mostrar "opcional (padrão)" em cada campo sem `obrigatório`, e o detector de
complexidade apontar campos de relação ou de estado nunca preenchidos. Arquivo:
`tooling/explicar/`, `compiler/ast/ast.go` (`Field`).
