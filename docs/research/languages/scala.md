# Scala: estudo dirigido para o Germanio

**Data:** 2026-09-28. **Status:** pesquisa, sem força normativa (a norma é `docs/INTENCAO.md`).
**Pergunta principal:** o que a complexidade acumulada do Scala, o redesenho dos implicits
e a polêmica da sintaxe opcional por indentação no Scala 3 ensinam à sintaxe hierárquica do
Germanio?

Complementa [sintaxe-e-configuracao.md](sintaxe-e-configuracao.md) (layout do Haskell,
YAML, HCL, CUE…) e `docs/research/sintaxe-hierarquica.md`; o layout em geral não é repetido
aqui, só o caso Scala 3, que é o único em que uma linguagem **grande e já adotada** passou
a aceitar duas sintaxes para o mesmo programa.

## Fontes consultadas

Consultadas nesta sessão (WebFetch):

- [Scala 3 Reference: Optional Braces](https://docs.scala-lang.org/scala3/reference/other-new-features/indentation.html)
- [Scala 3 Reference: Contextual Abstractions (crítica aos implicits e novo desenho)](https://docs.scala-lang.org/scala3/reference/contextual/index.html)
- [Scala Contributors: "Feedback sought: optional braces" (2020)](https://contributors.scala-lang.org/t/feedback-sought-optional-braces/4702)

Do conhecimento prévio, **não verificado nesta sessão**: TASTy como formato intermediário
e de compatibilidade entre Scala 3.x, o compilador Dotty (base teórica DOT), o processo SIP,
o suporte do scalafmt às duas sintaxes, e a quebra de compatibilidade binária entre 2.11,
2.12 e 2.13 que dividiu o ecossistema de bibliotecas.

Código do Germanio conferido: `docs/INTENCAO.md` ("Sintaxe hierárquica e contextual",
"Layout"), `compiler/lexer/lexer.go` (tabulação é erro), `compiler/parser/hierarquia.go`,
`tooling/formatter/`.

---

## Matriz (compacta)

| Item | Scala | Relevância para o Germanio |
|---|---|---|
| Objetivo original | Unir orientação a objetos e programação funcional na JVM (Odersky, EPFL, 2004) | não se aplica |
| Filosofia | Poucos mecanismos muito gerais que se combinam; o poder fica com o programador | O oposto do Germanio: muitos conceitos de domínio fechados, pouco poder genérico |
| Sintaxe | Chaves ou indentação (Scala 3); ver Sintaxe | Caso central deste estudo |
| Gramática | Off-side opcional: regiões de indentação começam depois de `:` no fim da linha em corpos de classe/objeto, depois de palavras como `=`, `=>`, `if`, `then`, `else`, `match`, `try`, `do`, `yield`; `end` opcional com especificador que precisa casar ([Optional Braces](https://docs.scala-lang.org/scala3/reference/other-new-features/indentation.html)) | O Germanio abre região sem marcador (`INTENCAO.md`, "Layout") |
| Lexer | Tabs e espaços permitidos; é erro quando larguras são "incomparable"; recomenda não misturar ([Optional Braces](https://docs.scala-lang.org/scala3/reference/other-new-features/indentation.html)) | O Germanio proíbe tabulação na indentação (`compiler/lexer/lexer.go`): regra mais simples, sem "comparabilidade" |
| Parser / AST | Recursivo descendente; árvores tipadas (não verificado) | — |
| IR | TASTy (não verificado) | não se aplica |
| Tipos / inferência | Sistema muito expressivo (tipos de ordem superior, tipos dependentes de caminho, união/interseção, match types); inferência local | Custo cognitivo máximo; não se aplica ao público do Germanio |
| Contextual abstractions | Scala 2: um modificador `implicit` para conversões, parâmetros e instâncias; Scala 3: `given`, `using`, extensões, `Conversion` explícita, import separado de givens ([Contextual](https://docs.scala-lang.org/scala3/reference/contextual/index.html)) | Ver Sintaxe |
| Compiler / runtime | JVM, Scala.js, Scala Native | não se aplica |
| Package manager / build | sbt (DSL de build própria), Mill, scala-cli (não verificado) | Outro caso de build como programa (ver `kotlin.md`, Gradle) |
| Formatter | scalafmt (terceiro, configurável) (não verificado) | Contraexemplo (P13) |
| LSP / IDE | Metals (LSP) e IntelliJ, com o plugin da IDE usando outro front-end (não verificado) | Risco de dois front-ends (A6) |
| Diagnostics | Crítica oficial: busca implícita recursiva falhava com mensagens pouco úteis ([Contextual](https://docs.scala-lang.org/scala3/reference/contextual/index.html)) | Diagnóstico ruim como consequência de inferência poderosa |
| Evolution process | SIPs (não verificado); a sintaxe opcional foi empurrada pelo criador, que admitiu usar sua autoridade de "BDFL informal" ([debate](https://contributors.scala-lang.org/t/feedback-sought-optional-braces/4702)) | Ver EVITAR |
| Backward compatibility | Flags `-no-indent`, `-old-syntax`, `-source 3.0-migration`; reescrita automática nos dois sentidos com `-rewrite -indent` e `-rewrite -no-indent` ([Optional Braces](https://docs.scala-lang.org/scala3/reference/other-new-features/indentation.html)) | Ver ADOTAR |
| Principais acertos | Redesenho explícito dos implicits por **intenção** e não por mecanismo; conversão automática entre as duas sintaxes | — |
| Principais problemas | Complexidade acumulada; duas sintaxes; divisão da comunidade | — |
| Complexidade acumulada | A maior entre as linguagens estudadas neste diretório | — |

---

## Sintaxe

| Pergunta | Scala | Germanio hoje (IMPLEMENTADO) | Comparação |
|---|---|---|---|
| Dados | `case class`, `enum` (Scala 3) | `cada X tem` | — |
| Funções | `def`, lambdas, currying, parâmetros `using` | Nível 3 | — |
| Módulos / imports | `package`, `import`; givens exigem `import x.given` separado ([Contextual](https://docs.scala-lang.org/scala3/reference/contextual/index.html)) | `importar` | O Scala 3 separou o import do que muda o significado sem aparecer no código; o Germanio precisa da mesma visibilidade (ver abaixo) |
| Relações | Não há | `tem` / `pertence a` | — |
| Estado | `val`/`var`; imutabilidade preferida | `começa`, `pode` | — |
| Fluxo | `if … then … else`, `match` como expressão | Declarativo | — |
| Erros | Exceções, `Try`, `Either` | Por campo, transação | Três mecanismos de erro são o padrão Scala; o Germanio tem um |
| Concorrência | `Future`, e ecossistemas concorrentes (Akka/Pekko, Cats Effect, ZIO) | Goroutines | A escolha de ecossistema de efeitos dividiu a comunidade (não verificado nesta sessão) |
| Tipos | Os mais expressivos da JVM | Pelo nome | — |
| Null | `null` herdado da JVM; `Option`; `-Yexplicit-nulls` experimental (não verificado) | `obrigatório` | — |
| Organização | sbt multi-projeto | `backend/`, `frontend/` | — |
| Boilerplate | Muito baixo: case classes, inferência, implicits/givens, macros | Inferência | O Scala reduziu boilerplate aumentando conceitos; o Germanio não pode fazer essa troca |
| Legibilidade em projetos grandes | Varia por "dialeto" da equipe (Scala como Java melhor, Scala funcional puro…) | Uma forma normal de fatos (P5) | — |

### Implicits: o caso de "poder demais"

A própria referência do Scala 3 lista por que os implicits foram redesenhados: poder
demais, com conversões implícitas "often dubious"; implicits que "can hide anywhere in a
long list of imports" e erros que "disappeared mysteriously with the right import"; uma
única palavra (`implicit`) para vários mecanismos, o que descreve "implementation details
rather than purpose"; e ferramentas com dificuldade para completar e diagnosticar
([Contextual](https://docs.scala-lang.org/scala3/reference/contextual/index.html)). A
correção foi dar a cada **intenção** sua própria palavra (`given`, `using`, `extension`,
`Conversion`) e um import próprio.

Paralelos no Germanio (onde ele pode estar errado):

1. **Inferência de tipo pelo nome** (`INTENCAO.md`, "Tipo pelo nome") é um "implícito": o
   significado de `valor` depende de uma tabela que o autor não vê. A norma já exige que
   `ge explain` mostre o tipo inferido, a origem e o motivo, e que ambiguidade vire erro.
   É exatamente a correção do Scala 3 (tornar visível o que foi inferido). Manter.
2. **`importar "backend"` carrega a pasta inteira** em ordem alfabética: um arquivo novo
   pode acrescentar fatos a um dado declarado em outro arquivo (fusão de blocos,
   `compiler/parser/resolver.go`). É o "implicit escondido num import". O mitigador é a
   origem em cada fato (P6) e a fusão que acusa conflito. Não se sabe se `ge explain` de um
   dado mostra **todos** os arquivos que contribuíram para ele (INVESTIGAR).
3. **Sinônimos** (várias línguas e verbos sinônimos, `compiler/idiomas/idiomas.go`,
   `INTENCAO.md`, "Ações sinônimas") não são conversões com efeito; não são o mesmo risco.

### A sintaxe opcional por indentação do Scala 3 e a sintaxe hierárquica do Germanio

Fatos do caso Scala:

- A motivação foi produtividade e estética, com experiência positiva em ensino (cursos da
  EPFL) e conversões de céticos; Odersky chamou de "the single most important productivity
  boost" ([debate](https://contributors.scala-lang.org/t/feedback-sought-optional-braces/4702)).
- As críticas, de quem usou por mais tempo: blocos só por indentação "tended to bleed
  together"; refatorar exige gerenciar indentação em vez de deixar o formatter resolver;
  lambdas de várias linhas geram estilo misto; indentação de 2 espaços difícil de ler
  rapidamente; `:` como abridor de bloco ambíguo com anotação de tipo; copiar e colar e
  refatoração com os mesmos problemas do Python; medo de divisão do ecossistema; e um
  processo percebido como imposto pela autoridade do criador (mesma fonte).
- A documentação recomenda marcadores `end` quando o bloco tem linhas em branco, passa de
  15-20 linhas ou termina muito indentado ([Optional Braces](https://docs.scala-lang.org/scala3/reference/other-new-features/indentation.html)).

Onde o Germanio está melhor (demonstrável):

- **Sem marcador de abertura ambíguo**: o bloco nasce da estrutura das linhas, não de `:`,
  e cada seção é uma tabela fechada; um filho no lugar errado é erro, não outro significado
  (A2, A3 em `GERMANIO_LESSONS.md`).
- **Tabulação proibida** em vez de "larguras incomparáveis".
- **Uma indentação canônica (4 espaços)** imposta pelo `ge fmt`, que reparseia e recusa
  saída que muda significado (`tooling/formatter/`); isso ataca "blocks bleed together" e
  a queixa dos 2 espaços.
- **Equivalência provada**: o bloco se reduz a frases planas e um teste prova a
  equivalência; o Scala não tem essa garantia semântica, só a reescrita sintática.
- **O conteúdo é declarativo e raso**: não há lambdas de várias linhas dentro de argumentos,
  que foi o ponto não resolvido do Scala.

Onde o Germanio repete o risco do Scala (INVESTIGAR/ADOTAR):

- **Duas formas válidas para o mesmo fato** (frase plana e bloco), e "o formatter não
  converte uma na outra" (`INTENCAO.md`, "Sintaxe hierárquica e contextual"). É a mesma
  situação que gerou o medo de divisão no Scala. O Scala ao menos oferece reescrita nos
  dois sentidos (`-rewrite -indent` / `-rewrite -no-indent`); o Germanio não tem comando
  que converta um `.ge` plano em hierárquico (não verificado se `ge explain` imprime a forma
  plana de um bloco inteiro de modo reaproveitável como arquivo).
- **Linhas em branco dentro de blocos** são permitidas (o exemplo normativo de `projetos`
  usa linhas em branco entre seções) e não há marcador de fim. Pelo critério do próprio
  Scala, é o caso em que o leitor perde o fim do bloco. Para `.ge` de dados curtos isso é
  pouco provável; para páginas longas, é o risco.
- **Copiar e colar entre níveis**: colar um bloco num nível diferente muda o sujeito sem
  erro quando o bloco colado também é válido no novo contexto (mesmo problema do Python e
  do Scala). A tabela fechada reduz, mas não elimina (ex.: colar `developer / enviar código`
  sob outro dado).

---

## Para o Germanio

**ADOTAR — reescrita automática entre forma plana e forma hierárquica, nos dois sentidos,
com equivalência provada.** Problema: duas formas válidas sem conversão (`INTENCAO.md`;
`tooling/formatter/`). Proposta: um comando explícito (não o `ge fmt` padrão) que reescreve
um arquivo ou um dado de uma forma para a outra, verificando que o `ast.App` resultante é
idêntico (a mesma verificação que o formatter já faz). Remove da cabeça: "qual das duas
formas uso, e como passo de uma para a outra?". Evidência: [Optional Braces](https://docs.scala-lang.org/scala3/reference/other-new-features/indentation.html).

**ADOTAR — dar a cada intenção sua palavra, nunca uma palavra para vários mecanismos.** É a
lição central do redesenho dos implicits ([Contextual](https://docs.scala-lang.org/scala3/reference/contextual/index.html))
e já é a política do Germanio (`tem`, `pertence a`, `pode`, `acesso`, `regras` como seções
distintas). Aplicar ao revisar `pode` (capacidade sem objeto versus permissão com objeto,
`INTENCAO.md`, "Estados, condições e pessoas"): uma palavra com dois significados decididos
pela presença do objeto é o padrão que o Scala abandonou. Classificar como decisão
deliberada; não mudar sem evidência de confusão real (teste do leigo).

**ADAPTAR — tornar visível tudo o que um import acrescenta.** Proposta: `ge explain <dado>`
lista os arquivos que contribuíram com fatos para o dado, e `ge check` avisa quando um dado
recebe fatos de mais de um arquivo do mesmo lado sem conflito (informativo). Arquivo:
`tooling/explicar/`, `compiler/parser/resolver.go`. Remove da cabeça: "de onde veio esta
regra?".

**EVITAR — decidir sintaxe por autoridade ou por gosto de especialistas.** O debate do
Scala mostra conversão de céticos *e* desistência de usuários de longo prazo. O Germanio já
tem critérios escritos (`INTENCAO.md`, "Como avaliar uma sintaxe", "Teste do leigo e teste
do profissional"); toda mudança de sintaxe passa por eles e pelo diff sobre o corpus de
exemplos (`dart.md`).

**EVITAR — reduzir boilerplate acrescentando mecanismos gerais** (implicits, macros,
tipos de ordem superior). O Scala é a prova de que cada mecanismo geral multiplica as
combinações que o leitor precisa entender.

**INVESTIGAR — marcador de fim ou guia visual para blocos longos.** Medir, nos `.ge` de
`demo/` e `examples/`, quantos blocos passam de 15-20 linhas ou têm linhas em branco
internas (limiar do Scala). Se forem raros, uma regra do detector de complexidade
(`INTENCAO.md`, "Detector de complexidade acidental") que sugira dividir o bloco é
preferível a uma palavra `fim`; guias de indentação no editor (`vscode-germanio/`) são a
alternativa sem sintaxe nova.

**INVESTIGAR — colar bloco no nível errado.** Um teste de propriedade: mover cada bloco de
seção para cada outro nível possível dos exemplos e contar quantos continuam válidos com
outro significado. Se o número não for zero, é o caso para um diagnóstico de "bloco aceito,
mas o sujeito mudou" no `ge explain`.
