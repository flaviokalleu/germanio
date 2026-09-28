# Pesquisa: sintaxe por indentação, linguagens de configuração e camadas de representação

**Data:** 2026-09-28. **Status:** pesquisa, sem força normativa (a norma é `docs/INTENCAO.md`
› Sintaxe hierárquica e contextual). Complementa `docs/research/sintaxe-hierarquica.md` sem
repetir o que ela já cobre (pilha do Python, CUE/TOML como caminho, Hedy, Green & Petre,
Elm/rustc, gofmt/Black). O formato de matriz por linguagem não se aplica: o texto está
organizado por tema.

**Pergunta central.** Como fazer `projetos / acesso / developer / enviar código`, cada palavra
num nível, ser uma construção formal, previsível e determinística, que produza
`Entity(projetos) + AccessGrant(resource=projetos, actor=developer, action=enviar_codigo)` e
não apenas `Node/Node/Node`? E quais camadas (CST, AST, HIR, IR, modelo semântico) fazem
sentido para o Germanio?

## Fontes consultadas

Especificações e documentação oficial:

- Haskell 2010 Report, cap. 10.3 (Layout): https://www.haskell.org/onlinereport/haskell2010/haskellch10.html
- Python, Lexical analysis: https://docs.python.org/3/reference/lexical_analysis.html
- Nim Manual (Indentation): https://nim-lang.org/docs/manual.html
- F#, Verbose Syntax: https://learn.microsoft.com/en-us/dotnet/fsharp/language-reference/verbose-syntax
- F# 4.1 Language Specification (offside rule): https://fsharp.org/specs/language-spec/4.1/FSharpSpec-4.1-latest.pdf (localizada pela busca; o conteúdo usado aqui veio do resumo da busca, então as regras do F# estão marcadas como não verificadas)
- YAML 1.2.2: https://yaml.org/spec/1.2.2/ ; tipo bool do YAML 1.1: https://yaml.org/type/bool.html
- yaml-test-suite: https://github.com/yaml/yaml-test-suite ; matriz: https://matrix.yaml.info/
- CUE spec: https://cuelang.org/docs/reference/spec/ ; The Logic of CUE: https://cuelang.org/docs/concept/the-logic-of-cue/
- HCL spec: https://github.com/hashicorp/hcl/blob/main/spec.md ; sintaxe JSON: https://github.com/hashicorp/hcl/blob/main/json/spec.md ; API Go (`hcl.Diagnostic`): https://pkg.go.dev/github.com/hashicorp/hcl/v2
- Dhall, Safety guarantees: https://docs.dhall-lang.org/discussions/Safety-guarantees.html ; Language Tour: https://docs.dhall-lang.org/tutorials/Language-Tour.html ; padrão: https://github.com/dhall-lang/dhall-lang/blob/master/standard/README.md
- Nickel: RATIONALE https://github.com/tweag/nickel/blob/master/RATIONALE.md ; README https://github.com/nickel-lang/nickel/blob/master/README.md ; merging https://nickel-lang.org/user-manual/merging ; contracts https://nickel-lang.org/user-manual/contracts ; anúncio https://www.tweag.io/blog/2020-10-22-nickel-open-sourcing/
- KDL spec: https://kdl.dev/spec/
- Pkl language reference: https://pkl-lang.org/main/current/language-reference/index.html
- Inform 7: módulo linguistics https://ganelson.github.io/inform/linguistics-module/index.html ; módulo calculus https://ganelson.github.io/inform/calculus-module/index.html

Árvores de sintaxe e recuperação de erros:

- rust-analyzer, syntax: https://github.com/rust-lang/rust-analyzer/blob/master/docs/book/src/contributing/syntax.md ; architecture: https://rust-analyzer.github.io/book/contributing/architecture.html
- Roslyn Overview: https://github.com/dotnet/roslyn/blob/main/docs/wiki/Roslyn-Overview.md ; Lippert, Red-green trees: https://ericlippert.com/2012/06/08/red-green-trees/
- swift-syntax: README https://github.com/swiftlang/swift-syntax ; `Contributor Documentation/Parser Recovery.md` e `Parser Design.md` no mesmo repositório
- tree-sitter, external scanners: https://tree-sitter.github.io/tree-sitter/creating-parsers/4-external-scanners.html ; scanner do Python: https://github.com/tree-sitter/tree-sitter-python/blob/master/src/scanner.c
- parso: https://parso.readthedocs.io/en/latest/docs/usage.html
- Kladov (mantenedor do rust-analyzer), Resilient LL Parsing Tutorial: https://matklad.github.io/2023/05/21/resilient-ll-parsing-tutorial.html
- Adams, *Principled Parsing for Indentation-Sensitive Languages: Revisiting Landin's Offside Rule*, POPL 2013: https://michaeldadams.org/papers/layout_parsing/ (texto lido do PDF)

Código do Germanio lido e executado nesta pesquisa: `compiler/parser/hierarquia.go`,
`compiler/parser/declarativo.go` (`dline`, `blockLines`), `compiler/parser/intencao.go`
(`intentFrom`, `blockLinesWithHeader`), `compiler/parser/parser.go` (`errorf`, `at`),
`compiler/ast/intencao.go` (`Intent`, `Grant`), `compiler/diagnostics/diagnostics.go`
(`Position`), `compiler/lexer/lexer.go` (`TokenIndent`), `tooling/formatter/intencao.go`,
`compiler/parser/hierarquia_test.go`. As sondagens com `ge check`/`ge explain` estão na seção 9.

---

## 1. A off-side rule: três famílias, e onde o Germanio está

Landin chamou de off-side rule a ideia de que a coluna delimita a estrutura. As
implementações se dividem em três famílias, e a diferença entre elas decide se a regra é
formalizável e se os erros são bons.

**Família A: indentação por linha, resolvida no lexer (Python, Nim).** O Python mantém uma
pilha que começa em `[0]`; linha mais recuada empilha e emite INDENT; linha menos recuada
precisa coincidir com um valor da pilha, desempilha e emite um DEDENT por nível; no fim do
arquivo emite os DEDENT restantes. Uma volta para uma coluna fora da pilha é erro do próprio
lexer ("inconsistent dedent"). Linhas só com espaço e comentário não geram NEWLINE. Dentro de
`() [] {}` as linhas se juntam e a indentação da continuação "não importa"
([Python](https://docs.python.org/3/reference/lexical_analysis.html)). O Nim faz o mesmo com
pseudo-terminais `IND{>}`, `IND{=}` e `DED` sobre uma pilha de níveis, e só aceita espaços
([Nim Manual](https://nim-lang.org/docs/manual.html)). A consequência central: depois do
layout, **a gramática é livre de contexto** e o parser nunca olha colunas.

**Família B: indentação pela coluna do primeiro token de uma construção (Haskell, F#).** No
Haskell, depois de `let`, `where`, `do` e `of` o lexer marca `{n}` com a coluna do próximo
lexema, e cada lexema que começa uma linha recebe `<n>`. A função L traduz isso para `{ ; }`
explícitos usando uma pilha de contextos
([Haskell 2010 §10.3](https://www.haskell.org/onlinereport/haskell2010/haskellch10.html)). O
F# chama essas colunas de *offside lines* e as abre depois de `let`, `if`, `then`, `->`,
`with`, `(`, `begin` etc., na coluna do primeiro token seguinte (não verificado no texto da
especificação; ver a nota nas fontes). O F# mantém as duas sintaxes: a *verbose*, menos
sensível à indentação, "is always enabled" mesmo com a *lightweight* como padrão
([F# Verbose Syntax](https://learn.microsoft.com/en-us/dotnet/fsharp/language-reference/verbose-syntax)).

**Família C: a indentação define a estrutura de dados diretamente (YAML).** O YAML proíbe
tab na indentação ("tab characters must not be used in indentation", §6.1) e usa o recuo
para escopo de coleções em bloco ([YAML 1.2.2](https://yaml.org/spec/1.2.2/)), mas combina
isso com cinco estilos de escalar, indicadores de *chomping*, estilos *flow* e chaves
complexas. A complexidade não vem da indentação; vem do que se permite dentro dela (seção 4).

**O Germanio está na família A**, e deve continuar lá. `layoutTree` usa só o recuo da linha
(nunca a coluna de um token no meio da linha) e uma pilha de níveis, e recusa a volta para um
recuo fechado. Isso é a decisão certa para leigos: a regra cabe numa frase ("o que está
abaixo pertence à linha de cima") e não depende de onde uma palavra termina.

### O problema do `parse-error(t)` e por que ele não pode existir no Germanio

A regra mais citada da função L é:

```
L (t : ts) (m : ms) = } : (L (t : ts) ms)   if m ≠ 0 and parse-error(t)
```

Ou seja: feche um bloco implícito **quando o próximo token seria um erro de parsing e um
`}` não seria**. É o que permite `let x = e; y = x in e'` numa linha só: o `in` fecha o
contexto do `let` porque, sem o fechamento, o `in` é ilegal
([Haskell 2010 §10.3](https://www.haskell.org/onlinereport/haskell2010/haskellch10.html)).
Adams mostra o custo: `parse-error(t)` é um oráculo sobre o comportamento *futuro* do parser,
então L "cannot run as an independent pass"; GHC e Hugs não usam um passo separado, lexer e
parser compartilham a pilha, e o protocolo depende de "some mildly complicated interactions
between the lexer and parser". Ao construir o próprio parser, Adams notou que "even minor
changes to the error propagation of the parser affected whether syntactically correct
programs were accepted" ([Adams, POPL 2013](https://michaeldadams.org/papers/layout_parsing/)).
A proposta dele é anotar a gramática com relações de indentação (`=`, `>`, `≥` e `⊛`, "sem
restrição") para que a regra deixe de ser operacional e caiba numa extensão de CFG com
algoritmos LR(k)/GLR (mesma fonte).

**Lição para o Germanio.** Três propriedades precisam continuar verdadeiras, e vale
escrevê-las como invariantes testáveis:

1. A estrutura de níveis é função **só das linhas** (recuo de cada linha não vazia). Nenhuma
   decisão de layout consulta o que o parser espera. Hoje é verdade em `layoutTree`.
2. Nenhuma construção abre nível pela coluna de um token no meio da linha (família B).
3. Não existe fechamento de bloco por erro. Se o Germanio um dia quiser uma construção de
   linha única com bloco (`se x então y`), ela deve ter delimitador explícito, e não um
   `parse-error(t)`.

Há uma exceção atual que viola o espírito da propriedade 1: `isDataBlock` decide se um bloco
é um bloco de dado **olhando se a primeira linha filha começa com uma seção**. É lookahead
sobre o conteúdo para decidir a estrutura. A sondagem 9.4 mostra a consequência: se a
primeira seção tiver um erro de digitação (`tme`), o bloco deixa de ser reconhecido e o erro
recai sobre a linha `projetos` ("não entendi a linha"), sem a sugestão "você quis dizer tem?"
que a mesma palavra recebe quando não é a primeira seção. A classificação deveria ser
estrutural: "linha de coluna 1 feita só de palavras, com filhos" é um cabeçalho de bloco; o
tipo do bloco se decide depois, e uma primeira seção desconhecida vira o erro de seção de
sempre.

## 2. Tokens virtuais, pilha INDENT/DEDENT e árvore sem perdas

Há duas formas de entregar o layout ao parser:

- **Tokens virtuais** INDENT/DEDENT (Python, Nim, o scanner externo do tree-sitter). O
  tree-sitter trata INDENT/DEDENT como tokens "impossible or inconvenient to describe with a
  regular expression", emitidos por uma função C que consulta `valid_symbols`
  ([tree-sitter](https://tree-sitter.github.io/tree-sitter/creating-parsers/4-external-scanners.html)).
  O scanner do Python serializa a pilha de recuos em 2 bytes por nível para que o parser
  incremental possa retomar
  ([scanner.c](https://github.com/tree-sitter/tree-sitter-python/blob/master/src/scanner.c)).
- **Árvore de linhas** construída antes do parser de frases (o que o Germanio faz em
  `layoutTree`). Para uma linguagem em que *cada linha é uma declaração* e não há expressões
  que atravessam linhas, a árvore de linhas é mais simples que os tokens virtuais e equivale a
  eles: `ABRE`/`FECHA` da gramática do `INTENCAO.md` são exatamente as arestas pai-filho.

A diferença importante não é entre essas duas formas, e sim entre uma estrutura **com perdas**
e uma **sem perdas**. O rust-analyzer, o Roslyn e o swift-syntax adotam árvores *lossless*:
"All comments and whitespace get preserved", e mesmo com entrada inválida "the tree produced
by the parser represents it exactly"
([rust-analyzer syntax](https://github.com/rust-lang/rust-analyzer/blob/master/docs/book/src/contributing/syntax.md));
a árvore do Roslyn é "completely round-trippable back to the text it was parsed from", com
espaço, comentários e diretivas guardados como *trivia* presa aos tokens
([Roslyn Overview](https://github.com/dotnet/roslyn/blob/main/docs/wiki/Roslyn-Overview.md));
o swift-syntax é "a source-accurate tree representation"
([swift-syntax](https://github.com/swiftlang/swift-syntax)).

**Estado do Germanio.** O `node` de `layoutTree` é uma árvore de linhas **com perdas**: as
linhas vazias e os comentários não entram nela (o lexer não emite comentários), e só o
`indent` numérico sobrevive. Por isso o formatter (`tooling/formatter/intencao.go`) não usa a
árvore do parser: relê o texto em `fline`, recalcula os níveis com **uma segunda pilha
própria** e depois compara o `meaning` (o `ast.Program` em JSON sem posições) para detectar
divergência. Contando `blockLines` (que corta o bloco na próxima linha de coluna 1),
`layoutTree`, a pilha do formatter e a gramática TextMate do VS Code, há quatro leituras da
mesma estrutura. A verificação de significado do formatter pega divergências de *resultado*,
não de *estrutura*: se as duas pilhas discordarem de um jeito que o parser aceite, o erro só
aparece quando mudar o significado.

**Recomendação (ADAPTAR, não copiar).** Uma árvore de linhas sem perdas **de linha**, não de
token: cada nó guarda o recuo cru, os tokens com *span* (início e fim), o comentário de fim de
linha, e as linhas vazias e de comentário como trivia presa ao nó seguinte (a mesma política
que o formatter já aplica: "a comment takes the level of the code it precedes"). O formatter
imprime essa árvore; o parser a consome. Isso elimina uma das pilhas e torna a propriedade
`print(parse(x))` com recuo normalizado um teste direto. Não é preciso trivia por token nem
round-trip byte a byte: o Germanio não tem expressões multilinha, e o formatter já normaliza
espaços dentro da linha.

## 3. Da hierarquia visual à semântica: `Node/Node/Node` versus fatos tipados

### O que as linguagens de configuração fazem

- **HCL** separa o *modelo de informação* (corpos, atributos, blocos com tipo e rótulos) das
  sintaxes concretas; nativa e JSON mapeiam para o mesmo modelo
  ([HCL spec](https://github.com/hashicorp/hcl/blob/main/spec.md)). A peça decisiva é o
  **schema fornecido pela aplicação**: "A body schema is a description of what is expected
  within a particular body, which can then be used to extract the body content". Na sintaxe
  JSON a dependência é total: "the schema is crucial to allow differentiation of attribute
  definitions and block definitions"
  ([HCL JSON](https://github.com/hashicorp/hcl/blob/main/json/spec.md)). Um bloco rotulado
  (`resource "tipo" "nome" { … }`) é interpretado pelo schema de quem o consome, não pela
  forma.
- **CUE** trata o aninhamento como caminho e resolve campos repetidos por unificação: "the
  result of which is the unification of all those fields", e `{a: 1} & {a: 2}` dá `_|_`
  ([CUE spec](https://cuelang.org/docs/reference/spec/)). A árvore visual não tem significado
  próprio; o significado é o valor no reticulado.
- **KDL** é o contraexemplo útil: todo documento é `nó(nome, argumentos, propriedades,
  filhos)`, e o espaço em branco é insignificante ([KDL](https://kdl.dev/spec/)). É um formato
  de *dados* genérico: `Node/Node/Node` é exatamente o que ele entrega, e a interpretação fica
  inteira com a aplicação. É o que o Germanio **não** deve ser na camada de domínio.
- **Inform 7**, a referência mais próxima em linguagem natural controlada, não para na árvore
  sintática: o módulo *linguistics* monta "sentence diagrams" com subárvores de sintagma
  nominal e verbal
  ([linguistics](https://ganelson.github.io/inform/linguistics-module/index.html)), e o
  módulo *calculus* converte as frases em **proposições de cálculo de predicados**, onde elas
  podem ser rejeitadas por tipo e reescritas "without changing their meaning"
  ([calculus](https://ganelson.github.io/inform/calculus-module/index.html)). A frase é
  superfície; a proposição é o significado.

A convergência é clara: **a árvore sintática é interpretada por um schema fechado que
atribui papel a cada nível, e o produto é um conjunto de fatos tipados**. A árvore genérica
(KDL, ou YAML sem schema) só adia o problema para quem consome.

### Resposta formal à pergunta central

Para `projetos / acesso / developer / enviar código` ser uma construção formal, três
componentes precisam estar escritos (e hoje estão só implícitos no código de `hierarquia.go`):

**(a) Um schema de seções como dado.** Para cada seção, o que cada nível abaixo dela
significa, a cardinalidade e a profundidade máxima. Esboço:

| Seção | Nível 1 abaixo | Nível 2 abaixo | Fato produzido |
|---|---|---|---|
| `tem` | linha de campo | proibido | `Field(entity, nome, tipo, modificadores)` |
| `pertence a` | alvo | proibido | `Relation(entity, alvo, opcional, papel)` |
| `começa` | (na linha) estado | — | `InitialState(entity, estado)` escalar |
| `pode` | capacidade | proibido | `Capability(entity, verbo)` |
| `acesso` | ator (`[somente] a {ou b}`) | ação `verbo [seu] [alvo]` | `AccessGrant(resource, actor, action, only, own, target)` |
| `regras` | regra | lista de pessoas (só em `… pode ser vista por`) | `Rule(entity, forma, args)` |
| `integração` | `nome "x"` | proibido | `IntegrationName(entity, x)` escalar |
| `quando`/`antes de` | corpo verbatim (nível 3) | — | `Hook(entity, evento, corpo)` |

"Proibido" é decisivo: hoje uma linha em nível 2 abaixo de `tem` é achatada como irmã e uma
linha em nível 3 abaixo de `acesso` é descartada (sondagens 9.2 e 9.3). Com o schema escrito,
filho não previsto é erro com a forma esperada.

**(b) Uma função de denotação por recursão estrutural.** Com um contexto `C = (entidade,
seção, ator, …)` que cada nível preenche num único slot:

```
⟦D ▸ secs⟧ ∅                      = {Entity(D)} ∪ ⋃ ⟦sec⟧(D)
⟦acesso ▸ atores⟧(D)              = ⋃ ⟦ator⟧(D, acesso)
⟦[somente] A ▸ ações⟧(D, acesso)  = ⋃ { AccessGrant(D, A, verbo(x), somente, seu(x), alvo(x) ou D) | x ∈ ações }
```

Cada regra é total sobre o schema (toda forma tem denotação ou erro) e não depende de ordem
nem de estado externo. A **frase plana tem a sua própria denotação no mesmo conjunto de
fatos** (`⟦developer pode enviar código para projetos⟧ = {AccessGrant(projetos, developer,
enviar_codigo, …)}`), e a equivalência passa a ser um teorema sobre dois mapeamentos para o
mesmo tipo, não um efeito colateral de reparsear tokens.

**(c) Um conjunto fechado de verbos de ação.** `enviar código` precisa ser reconhecido como
um verbo de duas palavras por uma regra geral (maior casamento contra o léxico de verbos que
as capabilities declaram), não por um `if aw[0] == "enviar" || aw[0] == "baixar"` no parser.
Hoje `access()` tem `enviar codigo`, `baixar codigo` e `branch_padrao` escritos no código do
core. Pela regra das três camadas do `CLAUDE.md`, vocabulário de um produto (Git hospedado)
não deveria estar em `compiler/`; o mecanismo genérico é "verbos de várias palavras vêm do
léxico das capabilities".

Com (a), (b) e (c), a construção é determinística no sentido forte: o conjunto de fatos é uma
função pura da árvore de linhas, a árvore é uma função pura das linhas, e a fusão (seção 5) é
comutativa, associativa e idempotente.

## 4. YAML: o que acontece quando a sintaxe tem muitas formas e tipagem implícita

- **Tipagem implícita pela aparência.** No YAML 1.1, o tipo bool aceita `y, Y, yes, Yes, YES,
  true, …, on, On, ON` e os negativos `n, N, no, No, NO, …, off`
  ([YAML 1.1 bool](https://yaml.org/type/bool.html)); daí o *Norway problem* (`NO` vira
  `false`). O YAML 1.2 removeu yes/no do schema JSON e do schema core (§10.2, §10.3), e ainda
  define um *failsafe schema* sem resolução implícita (§10.1)
  ([YAML 1.2.2](https://yaml.org/spec/1.2.2/)). Mas a matriz de conformidade registra que
  "some processors implement 1.1 or 1.0 only" ([matrix.yaml.info](https://matrix.yaml.info/)):
  a correção na especificação não chegou a todos os usuários, porque a especificação não
  controla as implementações.
- **Muitas formas de escrever a mesma coisa.** Cinco estilos de escalar (plain, aspas
  simples, aspas duplas, literal `|`, dobrado `>`) com três modos de *chomping*, mais estilos
  flow e block para coleções ([YAML 1.2.2](https://yaml.org/spec/1.2.2/) §7.3, §8.1). Cada
  forma extra multiplica os casos que parser, formatter e realçador precisam concordar.
- **Âncoras e aliases** transformam a árvore num *grafo de representação*: "a node may appear
  in more than one collection" (§3.2.2.2). Isso é uma dependência escondida no sentido de
  Green & Petre: o significado de uma linha depende de outra, distante, sem marca no ponto de
  uso.
- **Implementar um parser correto é difícil.** O yaml-test-suite existe como dado
  independente de linguagem (entrada, eventos esperados, JSON equivalente, casos de erro) e é
  usado por mais de 20 bibliotecas ([yaml-test-suite](https://github.com/yaml/yaml-test-suite)).
  A matriz tem 402 casos (308 válidos, 94 inválidos); os melhores parsers passam cerca de 300
  dos válidos, e recursos como chaves vazias, chaves complexas e tags locais faltam em várias
  bibliotecas, inclusive PyYAML ([matrix.yaml.info](https://matrix.yaml.info/)). Depois de
  duas décadas, nenhuma implementação popular concorda com a especificação em todos os
  casos.

**Para o Germanio.** Os três antídotos já estão na norma (uma forma por construção, nada de
tipo inferido da aparência, fusão explícita). O que falta é a lição do yaml-test-suite:
**uma suíte de conformidade como dados**, com pares `.ge` → fatos esperados e `.ge` →
diagnóstico esperado, que todos os leitores da estrutura tenham de passar (parser, formatter,
`ge explain` e a gramática do VS Code gerada por `vscode-germanio/tools/gerar_gramatica.py`).
Hoje `TestHierarquiaEquivaleAFrasePlana` tem oito pares escritos em Go; a mesma informação em
arquivos (`casos/NNN.ge`, `casos/NNN.fatos`, `casos/NNN.erro`) serve a todos os consumidores e
impede que o risco "vários parsers" (o do contexto deste estudo) apareça como divergência
silenciosa.

## 5. Fusão, ordem e conflito: CUE, Nickel, Pkl

- **CUE.** A unificação é "commutative, associative, and idempotent", e a ordem de avaliação
  é irrelevante ([CUE spec](https://cuelang.org/docs/reference/spec/)). "There is no need to
  specify these in a particular order or even in one location", e um valor conflitante é
  reportado "and the conflicting locations"
  ([The Logic of CUE](https://cuelang.org/docs/concept/the-logic-of-cue/)). Definições (`#X`)
  são fechadas recursivamente: campo não previsto é erro (spec).
- **Nickel.** O merge `&` é simétrico e comutativo; valores não-registro só se fundem se forem
  iguais; registros se fundem recursivamente. Para desempatar, há **prioridades**
  (`default`, numéricas, `force`), e o valor de prioridade maior substitui o menor sem fusão
  recursiva ([Nickel merging](https://nickel-lang.org/user-manual/merging)).
- **Pkl.** Objetos são imutáveis, mas *amending* cria um objeto novo que "only differs in
  selected properties", com *late binding*: as propriedades se comportam "like spreadsheet
  cells", e mudar a base muda os derivados
  ([Pkl](https://pkl-lang.org/main/current/language-reference/index.html)).

**Para o Germanio.** O `INTENCAO.md` escolheu o modelo do CUE (união de conjuntos, escalar
conflitante é erro com as duas origens, ordem irrelevante). A escolha está certa para leigos:
prioridades (Nickel) e herança com *late binding* (Pkl) são mecanismos de *sobrescrever*, e
sobrescrever é exatamente a dependência escondida que o `ge explain` tenta combater. Mas a
sondagem 9.5 mostra que a implementação **não cumpre a idempotência** para campos: o mesmo
`nome obrigatório` em dois blocos (ou em bloco e em frase plana) dá "projeto já tem o campo
nome". E o conflito real (`nome obrigatório` contra `nome opcional`) dá a mesma mensagem, sem
as duas origens. `TestHierarquiaFusao` cobre estado inicial, `pode` e `acesso`, mas não
campos. As três propriedades do CUE (comutativa, associativa, idempotente) deveriam ser um
teste de propriedade sobre o resolver: permutar blocos e arquivos, duplicar blocos, e o
modelo canônico não muda.

## 6. Normalização e igualdade semântica: Dhall

O Dhall é total ("if an expression type-checks then evaluating that expression always
succeeds in a finite amount of time") e sem efeitos além de três formas de import; dois
programas são equivalentes quando as **formas normais** coincidem, e o *hash semântico* é
feito sobre a forma normal, então refatorar ou comentar não muda o hash
([Dhall safety](https://docs.dhall-lang.org/discussions/Safety-guarantees.html);
[Language Tour](https://docs.dhall-lang.org/tutorials/Language-Tour.html)). O padrão
especifica gramática em ABNF e julgamentos separados de normalização alfa e beta,
equivalência e tipagem
([Dhall standard](https://github.com/dhall-lang/dhall-lang/blob/master/standard/README.md)).

O Germanio já tem, sem esse nome, uma forma normal: `meaning()` no formatter (programa em
JSON sem posições) e `canonical()` nos testes de equivalência. Falta torná-la **uma só e
definida**: um conjunto ordenado de fatos, sem posições, que seja o que o formatter compara, o
que os testes de equivalência comparam e o que `ge explain` lista. Hoje o formatter compara o
`ast.Program` (antes do resolver) e os testes comparam o modelo resolvido, que são noções
diferentes de "mesmo significado". O *hash* da forma normal (`ge explain --hash` ou similar)
seria útil em CI para provar que uma refatoração de `.ge` não mudou a aplicação; é barato
depois que a forma normal existir. O Dhall também mostra o limite: a igualdade por forma
normal exige uma linguagem sem efeitos. No Germanio ela vale para o nível 1 e 2 (declarações);
os corpos de nível 3 (`quando`, `antes de`) entram na forma normal como texto verbatim, sem
pretensão de igualdade semântica.

## 7. Contratos e culpa: Nickel

Os contratos do Nickel são verificações em runtime com **blame**: "Nickel tries to associate
each contract violation with the location of the code that triggered it"
([Nickel contracts](https://nickel-lang.org/user-manual/contracts)). O README descreve o
Dhall como exigindo anotar "all of their code with types" e o CUE como exigindo abandonar
"functions and Turing-completeness"
([README](https://github.com/nickel-lang/nickel/blob/master/README.md)); o RATIONALE diz que
"Nix lacks any native typing and validation capabilities"
([RATIONALE](https://github.com/tweag/nickel/blob/master/RATIONALE.md)). Sobre o YAML, não
encontrei no RATIONALE, no README, no manual nem no anúncio de 2020 uma crítica documentada
além de "unlike YAML, … it anticipates large configurations by being programmable"
([Tweag](https://www.tweag.io/blog/2020-10-22-nickel-open-sourcing/)); a crítica específica
ao YAML atribuída ao design do Nickel fica **(não verificado)**.

Para o Germanio, o conceito útil é o *blame*, não o contrato: todo fato derivado precisa
carregar a origem que o justificou (arquivo, span, caminho), e todo erro do resolver precisa
apontar para a linha que o causou **e** para a outra quando houver conflito. Isso já é a
intenção de `diagnostics.Position.Context`; falta *span* e falta a segunda origem.

## 8. Diagnósticos e recuperação de erros em linguagens de layout

### O que os sistemas maduros fazem

- **HCL.** `hcl.Diagnostic` tem `Severity`, `Summary` (terso), `Detail` (longo), `Subject`
  ("a tight range referring to exactly the construct that is problematic"), `Context` (faixa
  mais larga para mostrar em volta) e `Extra` legível por máquina; `hcl.Range` tem início e
  fim. As funções de decodificação devolvem **conteúdo parcial junto com os diagnósticos**:
  "The returned body content is valid if non-nil, regardless of whether Diagnostics are
  provided" ([hcl/v2](https://pkg.go.dev/github.com/hashicorp/hcl/v2)).
- **swift-syntax.** O parser só identifica duas classes de erro, *unexpected* e *missing*
  syntax; o diagnóstico é um passo posterior. "By restricting the parser to just the
  identification of missing and unexpected syntax, the implementation becomes far easier to
  reason about". Tokens ausentes entram na árvore sem texto; tokens inesperados são guardados
  num nó próprio; a recuperação olha para frente por um modelo de precedência de tokens
  (palavras que abrem declarações valem mais) (`Contributor Documentation/Parser
  Recovery.md` e `Parser Design.md` em https://github.com/swiftlang/swift-syntax).
- **Roslyn.** Insere token ausente com span vazio ou pula tokens até poder continuar, e
  guarda os pulados como trivia
  ([Roslyn Overview](https://github.com/dotnet/roslyn/blob/main/docs/wiki/Roslyn-Overview.md)).
- **rust-analyzer / Kladov.** O parser nunca falha; entrada inválida vira nó ERROR. A
  recuperação usa conjuntos FIRST/FOLLOW e "recovery sets" para decidir entre pular um token e
  sair de um laço, e um "fuel" contra laços infinitos; o erro fica localizado porque "valid
  prefixes are always recognized"
  ([Resilient LL](https://matklad.github.io/2023/05/21/resilient-ll-parsing-tutorial.html)).
- **Layout especificamente.** O scanner do tree-sitter para Python é chamado "with all tokens
  marked as valid" durante a recuperação e precisa detectar esse modo
  ([tree-sitter](https://tree-sitter.github.io/tree-sitter/creating-parsers/4-external-scanners.html));
  o do Python o detecta quando `STRING_CONTENT` e `INDENT` são válidos ao mesmo tempo
  ([scanner.c](https://github.com/tree-sitter/tree-sitter-python/blob/master/src/scanner.c)).
  O parso, com recuperação ligada, devolve código inválido "as an error node" e lista os
  problemas, inclusive de indentação, por `iter_errors`
  ([parso](https://parso.readthedocs.io/en/latest/docs/usage.html)). O CPython, por outro
  lado, para no primeiro erro de indentação (não verificado nesta pesquisa; é o comportamento
  observado do interpretador).

### O Germanio hoje

`errorf` devolve um `error`; `layoutTree`, `dataSection`, `access` e o resolver retornam no
primeiro problema. A sondagem 9.1 mostra só um erro por execução mesmo com dois
problemas independentes no arquivo (`enviar códigu` e `excluirr`). `diagnostics.Position`
tem linha e coluna, sem fim. As mensagens do parser de hierarquia seguem as quatro partes;
as do resolver para fatos vindos de bloco muitas vezes não (sondagem 9.1: "a.ge:8: não conheço
"codigu". Declare com: tenha codigu", sem "Onde", e com uma correção errada, porque o erro foi
interpretado na frase plana sintetizada, onde `codigu` parece um dado).

### A vantagem estrutural do Germanio

A recuperação em linguagens de layout é difícil porque um DEDENT errado muda onde *todas* as
linhas seguintes pertencem. No Germanio, três fatos tornam a recuperação quase trivial:

1. **A linha é a unidade de declaração.** Não há expressão que atravesse linhas (a norma
   proíbe continuação). O ponto de sincronização é sempre o próximo fim de linha.
2. **Cabeçalhos de coluna 1 são âncoras fortes**, como as palavras de declaração de alta
   precedência do swift-syntax. `blockLines` já corta o bloco na próxima linha de coluna 1.
3. **O schema de seções é fechado**, então o "recovery set" de cada nível é uma lista
   pequena e conhecida.

### Proposta de política (INVESTIGAR com testes antes de adotar)

- O parser devolve `(árvore, fatos, []Diagnostic)` e nunca aborta. Um erro de layout numa
  linha não impede a leitura do resto do bloco nem dos outros blocos.
- **Recuo que não corresponde a nível aberto** (6 entre 4 e 8): a linha vira um nó ERROR
  preso ao nível aberto mais profundo com recuo menor que o dela, e o diagnóstico diz as duas
  leituras possíveis (é o que a mensagem atual já faz). As linhas seguintes com **o mesmo
  recuo inválido** entram no mesmo nó ERROR em vez de gerar um erro cada, para não haver
  cascata.
- **Nada produzido dentro de um nó ERROR vira fato**, e o resolver não emite erros derivados
  de fatos que faltam por causa de um erro de sintaxe no mesmo bloco (supressão de cascata).
  Isso evita o pior modo de falha para um leigo: um erro de espaço gerar cinco mensagens de
  "não conheço".
- **Seção desconhecida** (`tme`): nó ERROR com a sugestão, e os filhos dela ficam fora dos
  fatos; as outras seções continuam.
- **Limite**: no máximo N diagnósticos por arquivo e um por linha, como proteção equivalente
  ao "fuel".

Isso é o que um LSP vai exigir de qualquer forma (código incompleto enquanto a pessoa digita),
e é útil já no `ge check`, que hoje obriga o leigo a um ciclo corrigir-um-rodar-de-novo.

## 9. Sondagens no código atual

Executadas com o binário construído a partir de `master` (`go build`), sobre arquivos
temporários no scratchpad. Todos os casos passam no `ge check` ou falham como descrito.

**9.1 Um erro por vez e correção errada.** Bloco `acesso` com `developer › enviar códigu` e
`maintainer › excluirr`. Saída única: `a.ge:8: não conheço "codigu". Declare com: tenha
codigu (dados declarados: maintainer, projetos)`. O segundo erro não aparece; a mensagem não
tem "Onde" nem a frase plana equivalente; a correção sugerida (declarar um dado `codigu`) é
consequência de o erro ser detectado na frase sintetizada `developer pode enviar codigu
projetos`. (A lista "dados declarados" incluindo `maintainer` e não `developer` merece uma
verificação à parte.)

**9.2 Nível a mais abaixo de uma ação é descartado em silêncio.**

```ge
projetos
    tem
        nome obrigatório
    acesso
        developer
          ver
            criar
```

`ge check` passa; `ge explain projetos` lista só `developer pode ver projetos`. O `criar`
sumiu: `access()` percorre `actor.children` e ignora os filhos de cada ação. É uma violação
direta de "Linhas que nenhuma construção reconhece são erro, nunca ignoradas".

**9.3 Nível a mais abaixo de `tem` e de `pertence a`.** `descricao` recuado abaixo de
`nome obrigatório` virou um campo irmão (`flattenLines` achata a árvore antes de entregar ao
parser antigo de corpo), e `clientes` recuado abaixo de `grupos opcional` foi descartado. Na
mesma sondagem, o fato `projeto pertence a grupos opcional` aparece com origem `app.ge:6`
(a linha da seção `pertence a`), não `app.ge:7` (a linha do item), porque o sujeito
sintetizado é posicionado no token da seção.

**9.4 Erro de digitação na primeira seção.** `projetos` › `tme` › `nome obrigatório` dá
`não entendi a linha "projetos"`, sem sugestão; a mesma palavra como terceira seção dá
`"tme" não é uma seção de projetos … (você quis dizer "tem"?)`. Causa: `isDataBlock` decide
pelo conteúdo da primeira linha filha (seção 1).

**9.5 Fusão não idempotente para campos.** `nome obrigatório` repetido em dois blocos
`tem`, em dois blocos `projetos` ou em bloco e frase plana: `projeto já tem o campo nome`.
`nome obrigatório` contra `nome opcional` dá a mesma mensagem, sem a outra origem. Contradiz
`INTENCAO.md` › Fusão e conflitos ("O mesmo fato repetido não muda nada") e o comentário do
próprio `TestHierarquiaFusao`.

**9.6 Tab.** Tab numa linha em branco no fim do arquivo dá `erro léxico … linha 5: tab na
indentação`. Coerente com a norma; vale confirmar se uma linha *só de espaço em branco* deve
ser erro (a norma diz que linhas vazias não contam para a indentação).

## 10. CST, AST, HIR, IR, modelo semântico: quais camadas o Germanio precisa

### Como os projetos de referência separam

- **rust-analyzer**: a árvore *green* (imutável, sem perdas, compartilhada) e a *red*
  (`SyntaxNode`, com pai e identidade) formam o CST; uma camada de AST tipada é só uma *vista*
  sobre ela, com todos os campos opcionais "to accommodate incomplete and/or erroneous source
  code" ([syntax](https://github.com/rust-lang/rust-analyzer/blob/master/docs/book/src/contributing/syntax.md)).
  O crate `syntax` "doesn't store semantic info"; o `hir` "provides a static, fully resolved
  view of the code"; a ponte de volta é o `Semantics`/`source_to_def`
  ([architecture](https://rust-analyzer.github.io/book/contributing/architecture.html)).
- **Roslyn**: árvore green com larguras e sem posições absolutas; red construída sob demanda
  com pai e posição, jogada fora a cada edição; reúso incremental em O(log n) nós
  ([Lippert](https://ericlippert.com/2012/06/08/red-green-trees/)). O `SemanticModel` é
  separado da sintaxe
  ([Roslyn Overview](https://github.com/dotnet/roslyn/blob/main/docs/wiki/Roslyn-Overview.md)).
- **swift-syntax**: árvore *source-accurate*, imutável, com nós *missing* e *unexpected*, e
  diagnóstico como passo separado (seção 8).
- **tree-sitter**: CST concreto e incremental para editores, sem camada semântica.

As motivações dessas camadas são: edição incremental em arquivos de milhares de linhas,
dezenas de consumidores (IDE, refatoração, macros), e linguagens com resolução de nomes e
tipos complexas. O próprio documento do rust-analyzer registra que "parsing from scratch
seems to be fast enough" apesar do suporte incremental.

### O pipeline do Germanio hoje

```
texto → lexer (tokens com linha/coluna, TokenIndent por linha; sem comentários)
      → blockLines / blockLinesWithHeader (linhas do bloco até a próxima coluna 1)
      → layoutTree (árvore de linhas `node`, com perdas)
      → dataSection / access / rule (monta tokens sintetizados de frase plana)
      → intentFrom (parser de frase plana) → ast.Intent (com Pos e Context)
      → resolver → ast.App (entidades, grants, estados, capabilities)
formatter: texto → fline + pilha própria → texto; confere meaning(ast.Program)
```

### Proposta de camadas, só as que resolvem um problema concreto

| Camada | Germanio | Problema que resolve | Decisão |
|---|---|---|---|
| Tokens com span e trivia de comentário | `lexer.Token` + fim | posições exatas (9.3), destaque no LSP, comentários no formatter | ADOTAR |
| CST de linhas sem perdas (única) | `node` estendido: recuo cru, tokens, comentário, trivia, nó ERROR | quatro leituras da estrutura; formatter imprime a árvore; recuperação (seção 8) | ADOTAR |
| AST tipada como vista sobre o CST (rust-analyzer) | não | o Germanio não tem expressões aninhadas nem muitos consumidores de sintaxe | EVITAR |
| Fatos (a "AST semântica" do pipeline desejado) | `ast.Intent`, gerado **direto** do CST pelo schema de seções, cada fato com origem (span + caminho + id do nó) | equivalência por denotação, não por reparse; blame | ADAPTAR (substitui o reparse sintetizado) |
| HIR separado | não | o `ast.Intent` já é o nível "resolvido por forma, não por nome"; outra camada duplicaria | EVITAR |
| Modelo semântico | `ast.App` (resolver) + mapa fato → origem | `ge explain`, conflitos com duas origens, futuro "ir para definição" | manter; acrescentar o mapa de origem |
| IR de capabilities | já existe depois do resolver | não | manter |
| Red-green, reparse incremental | não | arquivos `.ge` pequenos; reparse completo é suficiente | EVITAR até haver medição que diga o contrário |

O resultado são **quatro representações** (tokens, CST de linhas, fatos, modelo semântico),
o mesmo número que o Germanio já tem, com uma troca: a "frase plana sintetizada" deixa de ser
uma representação intermediária escondida.

## 11. A "redução a frases planas" é uma boa decisão ou uma dívida?

**Foi uma boa decisão de arranque, e é uma dívida para o que vem depois.**

O que ela acertou:

- Garantiu a equivalência com custo mínimo: todas as validações, mensagens e regras
  derivadas das frases planas passaram a valer para os blocos no mesmo commit.
- Deu uma definição executável e testável ("o bloco **é** a frase") que a norma pode citar,
  e `TestHierarquiaEquivaleAFrasePlana` a verifica.
- Deixou a frase plana como a forma que o `ge explain` mostra, o que ajuda o leigo a ligar as
  duas formas.

Onde ela vira dívida, com evidência:

1. **Posições.** Tokens sintetizados recebem a linha e a coluna de um token âncora
   (`synth`, `placed`); não existe span. O fato de `pertence a` aponta para a linha da seção,
   não para o item (9.3). Um LSP precisa de span exato para sublinhar, e um "ir para origem"
   precisa do nó.
2. **Diagnósticos na forma errada.** O erro é detectado na frase que o usuário **não**
   escreveu, e a correção é sugerida para ela (9.1). Consertar isso mensagem por mensagem é
   exatamente o "mais um `if`" que a auditoria da Parte 1 de `sintaxe-hierarquica.md` quis
   eliminar.
3. **A forma da árvore é jogada fora.** `flattenLines` entrega ao parser antigo uma lista
   plana de linhas; filhos não previstos são achatados ou ignorados (9.2, 9.3). A árvore
   existe, mas o significado não é definido sobre ela.
4. **Cirurgia de tokens por seção.** `access()` procura índices `k`, pula possessivos e
   *fillers*, trata `enviar codigo`, `baixar codigo` e `branch_padrao` como casos especiais,
   e `rule()` reconhece `quem cria vira` e `precisa de pelo menos` por posição de palavra.
   Cada nova seção aumenta essa superfície, e vocabulário de produto entra no core.
5. **Sem recuperação.** Como a reparse devolve `error`, o primeiro problema aborta tudo. Uma
   recuperação decente exige saber em que nó do CST o erro está, e a frase sintetizada não
   tem nó.
6. **Formatter e LSP não conseguem ligar fato a nó.** O formatter precisa da sua própria
   pilha, e a verificação de significado compara o programa inteiro sem dizer *onde* mudou.

**Caminho de saída incremental (sem reescrita):**

1. Escrever o schema de seções como tabela de dados em `compiler/parser/` (seção 3a) e fazer
   `layoutTree` validar filhos proibidos. Isso sozinho corrige 9.2 e 9.3 e não muda nenhum
   fato válido.
2. Acrescentar span aos tokens e à `diagnostics.Position`; guardar o id do nó do CST no fato.
3. Para cada seção, trocar "sintetiza frase e chama `intentFrom`" por "constrói o fato
   direto" (`&ast.Grant{Role, Verb, Target, Only, Own, Context, Pos}`), começando por
   `acesso`, que é a mais cirúrgica. O teste de equivalência continua o mesmo e prova que
   nada mudou.
4. Gerar a frase plana **a partir do fato** (uma função `fato → texto`) para `ge explain` e
   para o "Equivale a:" dos diagnósticos. A direção se inverte: a frase plana deixa de ser a
   entrada do bloco e passa a ser uma renderização do fato.
5. Mover o reconhecimento de verbos de várias palavras para o léxico das capabilities.
6. Trocar `error` por `[]Diagnostic` no parser e no resolver, com a política da seção 8.

## 12. Interação com o formatter

- **Uma estrutura, dois leitores** (CST único): o formatter é `print(CST)`, com recuo
  normalizado a 4 por nível. Hoje ele reconstrói a estrutura com uma pilha própria.
- **Forma normal única para a verificação** (Dhall): o formatter recusa quando os fatos
  normalizados mudam, a mesma forma normal que os testes e o `ge explain` usam.
- **Formatar com erro.** Com o CST tendo nós ERROR, o formatter deve recusar (ou reimprimir o
  nó ERROR verbatim) em vez de adivinhar o nível de uma linha com recuo inválido. O gofmt
  recusa arquivos com erro de sintaxe (não verificado nesta pesquisa). Um formatter que
  "conserta" o recuo escolheria um significado pelo usuário, e a norma diz que o recuo
  carrega significado.
- **Não converter entre as formas plana e hierárquica** continua correto (compromisso
  prematuro, Green & Petre). O F# mantém a sintaxe verbose "always enabled" ao lado da
  lightweight (fonte na seção 1), o mesmo arranjo das duas formas do Germanio.

## 13. Propagação de contexto e escopo

- **O que o contexto carrega é fixo pelo schema.** Cada nível preenche um slot
  (`entidade`, `seção`, `ator`). Nada mais é herdado: nenhum nível enxerga "a seção de cima da
  de cima" nem valores declarados em outro bloco. É o contrário do *late binding* do Pkl e dos
  aliases do YAML, que tornam o significado de uma linha dependente de algo distante.
- **Subtração com limite.** Omitir o sujeito é permitido só quando o caminho o determina
  (`INTENCAO.md`: "o aspecto nunca é omitido"). A regra formal: um slot omitido na linha deve
  estar preenchido no contexto; um slot preenchido nos dois lugares é permitido só se
  coincidir, senão é erro. Hoje o alvo explícito de uma ação é aceito "se o alvo for o dado ou
  algo que pertence a ele"; isso deveria virar uma regra do schema, verificada no fato
  (`Grant.Context`), e não uma checagem espalhada.
- **Palavras reservadas por nível, não globais.** `pode` é seção no nível 1 de um dado e
  palavra comum dentro de `regras` (`confidencial pode ser vista por`). Isso só é seguro
  porque as seções só são reconhecidas no nível 1; o schema deve dizer isso explicitamente
  (e a gramática do VS Code deve seguir a mesma regra, o que a suíte de conformidade da seção
  4 verificaria).

## 14. Não copiar

- **Layout por coluna do primeiro token e `parse-error(t)`** (Haskell, offside lines do F#):
  poderoso, informal, e o próprio Adams relata que mudanças pequenas no tratamento de erro
  mudam o que é aceito.
- **Múltiplos estilos de escalar, âncoras, aliases e tipos implícitos** (YAML).
- **Prioridades de merge e `force`** (Nickel) e **amending com late binding** (Pkl): mecanismos
  de sobrescrita, contrários a "conflito é erro". Se um dia houver padrões sobrescrevíveis,
  que fiquem no nível de configuração, com palavra explícita.
- **Modelo genérico de nós** (KDL) como camada de domínio: entrega `Node/Node/Node` e deixa o
  significado para cada consumidor.
- **Duas sintaxes concretas para o mesmo modelo com schema obrigatório** (HCL nativa + JSON):
  o Germanio não precisa de uma forma para máquinas escreverem; o `ge fmt` já dá a forma
  canônica.
- **Red-green trees, reparse incremental, AST tipada gerada por cima do CST e HIR separado**
  (Roslyn, rust-analyzer): resolvem escala e número de consumidores que o Germanio não tem.
- **Contratos com checagem preguiçosa em runtime** (Nickel): o Germanio verifica tudo no
  `ge check`, antes de rodar.

---

## Para o Germanio

| # | Lição | Classe | Problema concreto que resolve | Arquivos afetados |
|---|---|---|---|---|
| 1 | Layout é função só das linhas; nada de `parse-error(t)` nem coluna de token; classificar o bloco pela estrutura, não pelo conteúdo da primeira seção ([Haskell §10.3](https://www.haskell.org/onlinereport/haskell2010/haskellch10.html), [Adams](https://michaeldadams.org/papers/layout_parsing/)) | ADOTAR | 9.4: typo na primeira seção vira "não entendi a linha projetos" | `compiler/parser/hierarquia.go` (`isDataBlock`) |
| 2 | Schema de seções como dado (papel de cada nível, cardinalidade, filhos proibidos), à la body schema do HCL ([HCL spec](https://github.com/hashicorp/hcl/blob/main/spec.md)) | ADOTAR | 9.2 e 9.3: linhas descartadas ou achatadas em silêncio; base para formatter, LSP e gramática VS Code | `compiler/parser/hierarquia.go`, `vscode-germanio/tools/gerar_gramatica.py` |
| 3 | Fatos tipados gerados direto do CST por denotação; frase plana vira renderização do fato (Inform 7 → proposições, [calculus](https://ganelson.github.io/inform/calculus-module/index.html)) | ADAPTAR | dívida do reparse sintetizado: posições, diagnóstico na frase errada (9.1), cirurgia de tokens | `compiler/parser/hierarquia.go`, `compiler/parser/intencao.go`, `compiler/ast/intencao.go`, `tooling/explicar/explicar.go` |
| 4 | Verbos de várias palavras reconhecidos pelo léxico das capabilities, não por `if` no parser | ADOTAR | `enviar codigo`, `baixar codigo`, `branch_padrao` escritos no core | `compiler/parser/hierarquia.go` (`access`) |
| 5 | CST de linhas sem perdas e único (comentários e linhas vazias como trivia), consumido pelo parser e impresso pelo formatter ([rust-analyzer](https://github.com/rust-lang/rust-analyzer/blob/master/docs/book/src/contributing/syntax.md), [Roslyn](https://github.com/dotnet/roslyn/blob/main/docs/wiki/Roslyn-Overview.md)) | ADAPTAR | quatro leituras independentes da mesma estrutura | `compiler/parser/declarativo.go`, `compiler/parser/hierarquia.go`, `tooling/formatter/intencao.go` |
| 6 | Span (início e fim) em tokens e posições; diagnóstico com `Subject` e `Context` e segunda origem em conflitos ([hcl.Diagnostic](https://pkg.go.dev/github.com/hashicorp/hcl/v2)) | ADOTAR | 9.3 (origem na linha errada), 9.5 (conflito sem a outra origem), LSP futuro | `compiler/lexer/lexer.go`, `compiler/diagnostics/diagnostics.go`, `compiler/parser/resolver.go` |
| 7 | Parser e resolver devolvem resultado parcial + lista de diagnósticos; nó ERROR, sincronização por linha, supressão de cascata ([swift-syntax Parser Recovery](https://github.com/swiftlang/swift-syntax), [Resilient LL](https://matklad.github.io/2023/05/21/resilient-ll-parsing-tutorial.html)) | INVESTIGAR (política da seção 8, validar com exemplos reais antes) | 9.1: um erro por execução | `compiler/parser/parser.go`, `compiler/parser/hierarquia.go`, `compiler/parser/resolver.go` |
| 8 | Fusão comutativa, associativa e **idempotente**, testada por propriedade (permutar e duplicar blocos) ([CUE spec](https://cuelang.org/docs/reference/spec/)) | ADOTAR | 9.5: fato idêntico repetido é erro, contra a norma | `compiler/parser/resolver.go`, `compiler/parser/hierarquia_test.go` |
| 9 | Uma forma normal única de fatos, usada pelo formatter, pelos testes e pelo `ge explain`; hash opcional para CI ([Dhall](https://docs.dhall-lang.org/discussions/Safety-guarantees.html)) | ADAPTAR | `meaning()` (programa antes do resolver) e `canonical()` (modelo resolvido) são noções diferentes de "mesmo significado" | `tooling/formatter/intencao.go`, `compiler/parser/hierarquia_test.go`, `tooling/explicar/explicar.go` |
| 10 | Suíte de conformidade como dados (`.ge` → fatos, `.ge` → diagnóstico) para todos os leitores ([yaml-test-suite](https://github.com/yaml/yaml-test-suite), [matriz](https://matrix.yaml.info/)) | ADOTAR | risco de parsers divergentes (parser, formatter, VS Code) | nova pasta de casos ao lado de `compiler/parser/`; `vscode-germanio/tools/gerar_gramatica.py` |
| 11 | Blame: todo fato e todo erro derivado carregam a origem que o justifica ([Nickel contracts](https://nickel-lang.org/user-manual/contracts)) | ADAPTAR | mensagens do resolver sem "Onde" para fatos de bloco (9.1) | `compiler/parser/resolver.go`, `compiler/diagnostics/diagnostics.go` |
| 12 | Formatter recusa arquivo com erro de layout em vez de escolher um nível | ADOTAR | recuo carrega significado; "consertar" seria decidir pelo usuário | `tooling/formatter/intencao.go` |
| 13 | Prioridades de merge (Nickel), amending/late binding (Pkl), âncoras (YAML) | EVITAR | dependências escondidas; contraria "conflito é erro" | — |
| 14 | Red-green, reparse incremental, HIR separado, AST tipada gerada | EVITAR (reavaliar só com medição de LSP) | nenhum problema atual; custo alto | — |
| 15 | Relações de indentação na gramática (Adams) para formalizar a norma | INVESTIGAR | a EBNF do `INTENCAO.md` usa `ABRE`/`FECHA` informais; a notação de Adams poderia tornar o layout parte da especificação | `docs/INTENCAO.md` |

**Contradições com a abordagem atual encontradas nesta pesquisa:** (1) a norma diz que linha
não reconhecida nunca é ignorada, e há três casos em que linhas somem ou mudam de nível em silêncio dentro de
blocos (9.2, 9.3); (2) a norma diz que o fato repetido não muda nada, e para campos ele é
erro (9.5); (3) a equivalência "por construção" vale para programas válidos, mas não para
diagnósticos, que são emitidos sobre a frase sintetizada (9.1); (4) vocabulário de um produto
específico está no parser do core (`access`).
