# Comparação de sintaxe: layout, hierarquia e contexto

**Data:** 2026-09-28. **Status:** pesquisa, sem força normativa. A norma é
`docs/INTENCAO.md` › Sintaxe hierárquica e contextual (linhas 190-380). Fonte principal:
[sintaxe-e-configuracao.md](sintaxe-e-configuracao.md); complementos em
[python.md](python.md), [nim.md](nim.md), [go.md](go.md), [elixir.md](elixir.md) e
[../sintaxe-hierarquica.md](../sintaxe-hierarquica.md).

Pergunta central: como `projetos / acesso / developer / enviar código`, uma palavra por nível,
vira uma construção formal que produz `Entity(projetos)` + `AccessPolicy(resource=projetos,
actor=developer, action=enviar_codigo)` e não `Node/Node/Node`? E quais camadas (CST, AST,
HIR, IR, modelo semântico) fazem sentido?

---

## 1. Off-side rule: três famílias

| Família | Quem | Como decide a estrutura | Consequência |
|---|---|---|---|
| A: recuo **da linha**, resolvido antes do parser | Python, Nim, **Germanio** | pilha de recuos; linha mais funda abre, mais rasa precisa cair num nível aberto | depois do layout a gramática é livre de contexto; o parser nunca olha colunas ([Python, lexical analysis](https://docs.python.org/3/reference/lexical_analysis.html); [Nim manual](https://nim-lang.org/docs/manual.html), pseudo-terminais `IND{>}`, `IND{=}`, `DED`) |
| B: coluna **do primeiro token** depois de certas palavras | Haskell, F# | `let/where/do/of` abrem contexto na coluna do lexema seguinte; a função L insere `{ ; }` | inclui `parse-error(t)`: fecha o bloco quando o próximo token seria erro ([Haskell 2010 §10.3](https://www.haskell.org/onlinereport/haskell2010/haskellch10.html)) |
| C: o recuo **é** a estrutura de dados | YAML | escopo de coleções em bloco, tab proibido | a complexidade vem do que cabe dentro do recuo: 5 estilos de escalar, âncoras, tipos implícitos ([YAML 1.2.2](https://yaml.org/spec/1.2.2/)) |

**Por que o Germanio deve ficar na família A.** A regra cabe numa frase para um leigo ("o que
está abaixo pertence à linha de cima") e não depende de onde uma palavra termina. A família B
tem um custo formal documentado: `parse-error(t)` é um oráculo sobre o comportamento futuro do
parser, então o layout "cannot run as an independent pass", e Adams relata que "even minor
changes to the error propagation of the parser affected whether syntactically correct
programs were accepted" ([Adams, POPL 2013](https://michaeldadams.org/papers/layout_parsing/)).
Numa linguagem para leigos, a aceitação não pode depender da recuperação de erro.

Três invariantes testáveis (propostos em `sintaxe-e-configuracao.md` §1):

1. A estrutura de níveis é função **só das linhas** (recuo de cada linha não vazia).
2. Nenhuma construção abre nível pela coluna de um token no meio da linha.
3. Não existe fechamento de bloco por erro; uma construção de linha única com bloco, se um
   dia existir, terá delimitador explícito.

`layoutTree` (`compiler/parser/hierarquia.go:34-77`) cumpre 1-3 dentro de um bloco. A exceção
é `isDataBlock` (`hierarquia.go:150-198`): decide se a linha de coluna 1 é um bloco de dado
olhando se a **primeira linha filha começa com uma seção** (`hierarquia.go:197`). É lookahead
sobre o conteúdo para decidir a estrutura, e produz o bug de GERMANIO_LESSONS.md › D4 (erro de
digitação na primeira seção desfaz o bloco).

## 2. Layout: tokens virtuais ou árvore de linhas

Há duas formas de entregar o layout ao parser: tokens INDENT/DEDENT (Python, Nim, o scanner
externo do tree-sitter, que os trata como tokens "impossible or inconvenient to describe with
a regular expression", [tree-sitter](https://tree-sitter.github.io/tree-sitter/creating-parsers/4-external-scanners.html))
ou uma **árvore de linhas** construída antes do parser de frases. Para uma linguagem em que
cada linha é uma declaração e nenhuma expressão atravessa linhas, as duas são equivalentes, e
a árvore de linhas é mais simples. O `ABRE`/`FECHA` da EBNF de `INTENCAO.md` (linhas 286-310)
são exatamente as arestas pai-filho da árvore; o código não emite esses símbolos (o lexer
emite `TokenIndent` por linha, `compiler/lexer/lexer.go:454-470`). O estudo do Nim recomenda
declarar na norma qual é a definição real (a pilha de `layoutTree`) e escrever o layout como
pseudo-terminais ([nim.md](nim.md), lição 3). A diferença que importa não é tokens × árvore:
é **com perdas × sem perdas** (seção 8).

## 3. Whitespace significativo: o que o recuo pode e não pode significar

| Regra | Origem | Estado no Germanio |
|---|---|---|
| Só espaços; tab é erro | YAML §6.1, Nim | norma: `INTENCAO.md:273`. O lexer de aplicação recusa tab (`lexer.go:458-462`); o lexer do núcleo estrito o conta como 2 espaços (`lexer.go:463`) e o parser do núcleo o recusa depois (`compiler/parser/germanio.go:50`) |
| Passo livre, estrutura só por comparação | Python | norma: 4 canônico, outros passos consistentes aceitos (`INTENCAO.md:277`) |
| Continuação de linha por recuo | Nim (depois de operador/vírgula) | **evitar**: no `.ge` recuo maior é sempre filho; continuação criaria a ambiguidade filho/continuação ([nim.md](nim.md), lição 4). A norma já proíbe (`INTENCAO.md:281-285`) |
| Recuo dentro da linha (alinhamento) | Nim (espaço em volta de operador muda precedência) | não existe no `.ge`; não criar |
| Linhas vazias e de comentário não contam | Python, YAML | norma (`INTENCAO.md:279`); cumprido |
| Comentário nunca muda fato | Roc (regra escrita), Dhall (hash sobre forma normal) | implícito; vale escrever como regra, porque é o que torna segura a verificação do `ge fmt` ([roc.md](roc.md)) |

## 4. Estruturas declarativas aninhadas: de `Node/Node/Node` a fatos tipados

As linguagens de configuração mostram as duas saídas possíveis de um aninhamento:

- **Árvore genérica** (KDL: todo documento é `nó(nome, argumentos, propriedades, filhos)`,
  [KDL spec](https://kdl.dev/spec/); YAML sem schema). Entrega `Node/Node/Node` e deixa o
  significado para cada consumidor. É o que o Germanio **não** deve ser na camada de domínio.
- **Árvore interpretada por um schema fechado** que dá papel a cada nível. No HCL, "a body
  schema is a description of what is expected within a particular body", e na sintaxe JSON
  "the schema is crucial" para distinguir atributo de bloco
  ([HCL spec](https://github.com/hashicorp/hcl/blob/main/spec.md),
  [HCL JSON](https://github.com/hashicorp/hcl/blob/main/json/spec.md)). No CUE o aninhamento é
  caminho e o significado é o valor no reticulado
  ([CUE spec](https://cuelang.org/docs/reference/spec/)). No Inform 7, a frase vira diagrama
  sintático e depois **proposição de cálculo de predicados**, que pode ser rejeitada por tipo
  e reescrita "without changing their meaning"
  ([Inform 7 calculus](https://ganelson.github.io/inform/calculus-module/index.html)).

A convergência: a árvore sintática é interpretada por um schema fechado, e o produto é um
conjunto de fatos tipados.

### Onde o Germanio está

A norma já diz que cada seção é uma tabela fechada (`INTENCAO.md:320-352`) e que o bloco
"**é**" a frase plana (`INTENCAO.md:235-237`). A implementação realiza isso reduzindo cada
seção a **tokens de frase plana sintetizados** (`synth`, `placed`, `hierarquia.go:213-273`) e
reparseando com o parser de frases (`intentFrom`). Funciona para programas válidos, e
`TestHierarquiaEquivaleAFrasePlana` prova a equivalência. Mas o schema está implícito no
código, e o que ele não prevê é achatado (`flattenLines`, `hierarquia.go:108-115`) ou
ignorado (`access`, `hierarquia.go:401-423`, não visita `a.children`).

### Resposta formal à pergunta central

São necessários três componentes escritos (hoje só implícitos em `hierarquia.go`):

**(a) Um schema de seções como dado.** Para cada seção, o papel de cada nível abaixo dela, a
cardinalidade e o que é proibido:

| Seção | Nível 1 abaixo | Nível 2 abaixo | Fato |
|---|---|---|---|
| `tem` | linha de campo, relação ou pessoa | **proibido** | `Field(entidade, nome, tipo, modificadores)` / `Relation` |
| `pertence a` | alvo `[opcional] [como papel]` | **proibido** | `Relation(entidade, alvo, opcional, papel)` |
| `começa` | (na linha) estado | — | `InitialState(entidade, estado)`, escalar |
| `pode` | capacidade | **proibido** | `Capability(entidade, verbo)` |
| `acesso` | ator `[somente] a {ou b}` | ação `verbo [seu] [alvo]` | `AccessPolicy(resource, actor, action, only, own, target)` |
| `regras` | regra da lista fechada | pessoas (só em `… pode ser vista por`) | `Rule(entidade, forma, args)` |
| `integração` | `nome "x"` | proibido | `IntegrationName(entidade, x)`, escalar |
| `quando` / `antes de` | corpo nível 3 (verbatim) | — | `Hook(entidade, evento, corpo)` |

"Proibido" é o que falta hoje: com o schema, um filho não previsto é erro com a forma
esperada, em vez de linha achatada ou descartada.

**(b) Uma denotação por recursão estrutural**, com um contexto `C = (entidade, seção, ator)`
que cada nível preenche num único slot:

```text
⟦D ▸ secs⟧ ∅                     = {Entity(D)} ∪ ⋃ ⟦sec⟧(D)
⟦acesso ▸ atores⟧(D)             = ⋃ ⟦ator⟧(D, acesso)
⟦[somente] A ▸ ações⟧(D, acesso) = ⋃ { AccessPolicy(D, A, verbo(x), somente, seu(x), alvo(x) ou D) | x ∈ ações }
```

A frase plana tem **a sua própria** denotação no mesmo conjunto de fatos
(`⟦developer pode enviar código para projetos⟧ = {AccessPolicy(projetos, developer,
enviar_codigo, …)}`). A equivalência bloco = frase deixa de ser efeito colateral de reparsear
tokens e vira um teorema sobre duas funções para o mesmo tipo, testável caso a caso.

**(c) Um conjunto fechado de verbos**, reconhecidos por maior casamento contra o léxico que as
capabilities declaram. Hoje `enviar código` é reconhecido por
`if (aw[0] == "enviar" || aw[0] == "baixar") && aw[1] == "codigo"` em `hierarquia.go:404`, e
`branch_padrao` em `hierarquia.go:419` (a mesma regra se repete em
`compiler/parser/intencao.go:257,697` e `resolver.go:1031,1082`). O mecanismo genérico é "um
verbo de várias palavras vem do léxico da capability `repositório`", não um `if` no parser.

Com (a), (b) e (c) a construção é determinística no sentido forte: os fatos são função pura
da árvore de linhas, a árvore é função pura das linhas, e a fusão é comutativa, associativa e
idempotente (seção 6).

### Como fica `projetos / acesso / developer / enviar código`

1. Linhas: `projetos` (recuo 0), `acesso` (4), `developer` (8), `enviar código` (12).
2. Árvore de linhas: `projetos ▸ acesso ▸ developer ▸ enviar código`.
3. Schema: nível 1 de um dado é seção → `acesso`; nível 1 de `acesso` é ator → `developer`
   (conferido depois contra os papéis, que podem estar em outro arquivo); nível 2 é ação →
   `enviar código` casa com o verbo `enviar_codigo` do léxico da capability.
4. Fatos: `Entity(projetos)` e `AccessPolicy(resource=projetos, actor=developer,
   action=enviar_codigo, only=false, own=false, target=projetos)`, com origem = span da linha 4
   + caminho `projetos › acesso › developer` + id do nó.
5. Renderização: `ge explain` mostra "developer pode enviar código para projetos" gerado **a
   partir do fato**, não como entrada do bloco.

Hoje os passos 1, 2 e o resultado final existem; os passos 3-5 são feitos por síntese de
tokens e reparse.

## 5. Propagação de contexto e escopo

- **O que o contexto carrega é fixo pelo schema.** Cada nível preenche um slot (entidade,
  seção, ator). Nada mais é herdado. É o oposto do *late binding* do Pkl, onde propriedades se
  comportam "like spreadsheet cells"
  ([Pkl](https://pkl-lang.org/main/current/language-reference/index.html)), e dos aliases do
  YAML, em que "a node may appear in more than one collection" (§3.2.2.2): nos dois, o
  significado de uma linha depende de algo distante sem marca no ponto de uso.
- **Subtração com limite.** Omitir o sujeito é permitido só quando o caminho o determina, e o
  aspecto nunca é omitido (`INTENCAO.md:242`). Regra formal: um slot omitido na linha deve
  estar preenchido no contexto; preenchido nos dois lugares, só se coincidir.
- **Palavras reservadas por nível** (as *soft keywords* do Python): `pode` é seção no nível 1
  de um dado e palavra comum dentro de `regras` (`confidencial pode ser vista por`). É seguro
  porque `sectionOf` só olha a primeira palavra de uma linha de nível 1
  (`hierarquia.go:130-145`); o schema deve dizer isso, e a gramática do VS Code seguir a
  mesma regra ([python.md](python.md), lição 9).

## 6. Fusão, ordem e conflito

A norma escolheu o modelo do CUE: fatos se somam, repetido não muda nada, escalar com dois
valores é erro com as duas origens, ordem irrelevante (`INTENCAO.md:360-366`). O CUE declara a
unificação "commutative, associative, and idempotent" e reporta "the conflicting locations"
([CUE spec](https://cuelang.org/docs/reference/spec/);
[The Logic of CUE](https://cuelang.org/docs/concept/the-logic-of-cue/)). As alternativas de
sobrescrita (prioridades e `force` do Nickel, [merging](https://nickel-lang.org/user-manual/merging);
amending do Pkl) são dependências escondidas: EVITAR. A implementação ainda não cumpre a
idempotência para campos (GERMANIO_LESSONS.md › D2).

## 7. Ambiguidade

Fontes de ambiguidade que os estudos encontraram e como o Germanio as trata:

| Fonte | Exemplo externo | No Germanio |
|---|---|---|
| Tipagem pela aparência | YAML 1.1 `NO` → `false` ([YAML bool](https://yaml.org/type/bool.html)) | tipo pelo nome do campo é tabela fixa e "declaração explícita sempre vence" (`INTENCAO.md` › O que existe); manter como tabela, nunca heurística aberta |
| Escolha ordenada | PEG do Python ([python.md](python.md), lição 13) | EVITAR: a ordem das alternativas viraria semântica escondida |
| Grafias equivalentes | Nim: `foo_bar` = `fooBar` (RFC 456, [nim.md](nim.md)) | `foldWord` dobra acentos também nos **nomes** (`compiler/parser/intencao.go:35-36`): `secretária` e `secretaria` colidem, `é` vira a conjunção `e` (D7). A lição do Nim: igualdade tolerante exige verificação de consistência e trata colisão como conflito |
| Lista na mesma linha | vírgula como separador e modificador | a norma já resolve: um item por linha (`INTENCAO.md:281-285`) |
| Continuação | Nim, Python dentro de parênteses | proibida pela norma |
| Mesmo nome, dois namespaces | Rust, Gleam (tipo × valor) | `developer` pode ser papel ou (erro de) seção; o diagnóstico deve procurar nos dois ([DIAGNOSTICS.md](DIAGNOSTICS.md)) |

## 8. Recuperação de erros

- **O que os sistemas maduros fazem.** swift-syntax restringe o parser a identificar
  *missing* e *unexpected* e faz o diagnóstico num passo posterior
  ([swift-syntax](https://github.com/swiftlang/swift-syntax), `Parser Recovery.md`). O
  rust-analyzer nunca falha; entrada inválida vira nó ERROR, com conjuntos de recuperação e
  "fuel" contra laços ([Resilient LL](https://matklad.github.io/2023/05/21/resilient-ll-parsing-tutorial.html)).
  O HCL devolve **conteúdo parcial com os diagnósticos**: "The returned body content is valid
  if non-nil, regardless of whether Diagnostics are provided"
  ([hcl/v2](https://pkg.go.dev/github.com/hashicorp/hcl/v2)). O Elixir passou a reportar
  vários erros por arquivo na v1.15 ([v1.15](https://elixir-lang.org/blog/2023/06/19/elixir-v1-15-0-released/));
  o Gleam não interrompe o módulo por um erro numa definição
  ([v1.2](https://gleam.run/news/fault-tolerant-gleam/)).
- **O Germanio hoje** para no primeiro erro: `errorf` devolve `fmt.Errorf`
  (`compiler/parser/parser.go:32-38`), `layoutTree`, `dataSection`, `access` e o resolver
  retornam no primeiro problema. Reproduzido: `enviar códigu` e `excluirr` no mesmo arquivo dão
  um só erro.
- **A vantagem estrutural do Germanio**: a linha é a unidade de declaração (o ponto de
  sincronização é sempre o próximo fim de linha); cabeçalhos de coluna 1 são âncoras fortes
  (`blockLines` já corta o bloco neles); o schema de seções é fechado, então o conjunto de
  recuperação de cada nível é pequeno e conhecido.
- **Política proposta (INVESTIGAR com testes):** parser devolve árvore + fatos +
  `[]Diagnostic`; linha com recuo sem pai vira nó ERROR preso ao nível aberto mais fundo;
  linhas seguintes com o mesmo recuo inválido entram no mesmo nó (sem cascata); nada dentro de
  um nó ERROR vira fato, e o resolver não emite erros derivados de fatos que faltam por causa
  dele; no máximo um diagnóstico por linha.

## 9. Interação com o formatter

- **Uma estrutura, dois leitores.** Hoje há quatro leituras da mesma estrutura: `blockLines`
  (`compiler/parser/declarativo.go:22`), `layoutTree`, a pilha própria do formatter
  (`tooling/formatter/intencao.go:127-145`) e a gramática TextMate
  (`vscode-germanio/tools/gerar_gramatica.py`). O rust-analyzer, o Roslyn e o swift-syntax
  usam árvores sem perdas, em que "all comments and whitespace get preserved"
  ([rust-analyzer syntax](https://github.com/rust-lang/rust-analyzer/blob/master/docs/book/src/contributing/syntax.md)).
  Para o Germanio basta uma árvore sem perdas **de linha** (recuo cru, tokens com span,
  comentário de fim de linha, linhas vazias e comentários como trivia do nó seguinte), não de
  token. O formatter a imprime; o parser a consome ([FORMATTERS.md](FORMATTERS.md)).
- **Forma normal única** para "mesmo significado" (Dhall compara formas normais,
  [Dhall safety](https://docs.dhall-lang.org/discussions/Safety-guarantees.html)): hoje o
  formatter compara o `ast.Program` antes do resolver (`meaning`, `intencao.go:44-60`) e os
  testes comparam o modelo resolvido; são noções diferentes.
- **Formatar com erro**: recusar, nunca escolher um nível pelo usuário. O `ge fmt` já recusa
  quando o reparse falha.
- **Não converter entre forma plana e hierárquica** continua correto; o F# mantém a sintaxe
  verbose "always enabled" ao lado da leve
  ([F# verbose syntax](https://learn.microsoft.com/en-us/dotnet/fsharp/language-reference/verbose-syntax)).

## 10. Camadas: quais fazem sentido para o Germanio

| Camada | Referência | Para o Germanio | Decisão |
|---|---|---|---|
| Tokens com início e fim, comentário como trivia | Roslyn, TypeScript | `lexer.Token` hoje tem só `Line`, `Column` (`lexer.go:214-224`) e o lexer descarta `#` (`lexer.go:495-500`) | ADOTAR |
| CST de linhas sem perdas, única | rust-analyzer (green/red), Roslyn, swift-syntax | `node` de `layoutTree` estendido com recuo cru, spans, comentário, trivia, nó ERROR | ADOTAR |
| AST tipada como vista sobre o CST | rust-analyzer | o `.ge` não tem expressões aninhadas nem dezenas de consumidores de sintaxe | EVITAR |
| Fatos (a "AST semântica" do pipeline) | Inform 7 (proposições), HCL (schema) | `ast.Intent`, gerado **direto** do CST pelo schema, cada fato com origem | ADAPTAR (substitui a frase sintetizada) |
| HIR separado | rustc, rust-analyzer | `ast.Intent` já é "resolvido por forma, não por nome"; outra camada duplicaria | EVITAR |
| Modelo semântico resolvido | Roslyn `SemanticModel`, `hir` | `ast.App` (resolver) + mapa fato → origem | manter; acrescentar o mapa |
| IR de capabilities | — | já existe depois do resolver | manter |
| Red-green, reparse incremental, queries | Roslyn, rustc, rust-analyzer | arquivos `.ge` pequenos; o próprio rust-analyzer registra que "parsing from scratch seems to be fast enough" | EVITAR até haver medição |

São quatro representações (tokens, árvore de linhas, fatos, modelo resolvido), o mesmo número
que o Germanio já tem, com uma troca: a frase plana sintetizada deixa de ser uma
representação intermediária escondida e passa a ser uma renderização. Detalhes e caminho em
[COMPILER_ARCHITECTURE.md](COMPILER_ARCHITECTURE.md).

## 11. Não copiar

Layout pela coluna do primeiro token e `parse-error(t)` (Haskell, F#); múltiplos estilos de
escalar, âncoras e tipos implícitos (YAML); prioridades de merge e amending com late binding
(Nickel, Pkl); modelo genérico de nós na camada de domínio (KDL); duas sintaxes concretas para
o mesmo modelo (HCL nativa + JSON); continuação de linha por recuo (Nim); PEG com escolha
ordenada (Python); parênteses opcionais e keyword lists como açúcar geral, que custaram ao
Elixir categorias `no_parens` e conflitos aceitos na gramática ([elixir.md](elixir.md));
seções registráveis por biblioteca (macros de Nim e Elixir).
