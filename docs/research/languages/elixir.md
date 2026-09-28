# Elixir: estudo dirigido para o Germanio

**Data:** 2026-09-28. **Status:** pesquisa, sem força normativa (a norma é `docs/INTENCAO.md`).
**Pergunta principal:** como o Elixir mantém o código expressivo e composicional sem sintaxe
cerimonial, e o que isso custa (em especial as macros) para quem nunca programou?

## Fontes consultadas

Consultadas nesta sessão (WebFetch/WebSearch ou `gh api`):

- Repositório: [elixir_tokenizer.erl](https://github.com/elixir-lang/elixir/blob/main/lib/elixir/src/elixir_tokenizer.erl), [elixir_parser.yrl](https://github.com/elixir-lang/elixir/blob/main/lib/elixir/src/elixir_parser.yrl), [CONTRIBUTING.md](https://github.com/elixir-lang/elixir/blob/main/CONTRIBUTING.md)
- Documentação oficial (hexdocs migrou para subdomínios `*.hexdocs.pm`): [Syntax reference](https://elixir.hexdocs.pm/syntax-reference.html), [Quote and unquote](https://elixir.hexdocs.pm/quote-and-unquote.html), [Macros](https://elixir.hexdocs.pm/macros.html), [Meta-programming anti-patterns](https://elixir.hexdocs.pm/macro-anti-patterns.html), [Code-related anti-patterns](https://elixir.hexdocs.pm/code-anti-patterns.html), [Writing documentation](https://elixir.hexdocs.pm/writing-documentation.html), [Compatibility and deprecations](https://elixir.hexdocs.pm/compatibility-and-deprecations.html), [Code (formatter, diagnostics)](https://elixir.hexdocs.pm/Code.html), [Kernel `|>/2`](https://elixir.hexdocs.pm/Kernel.html), [Gradual set-theoretic types](https://elixir.hexdocs.pm/gradual-set-theoretic-types.html), [mix format](https://mix.hexdocs.pm/Mix.Tasks.Format.html)
- Ecossistema: [Ecto.Schema](https://ecto.hexdocs.pm/Ecto.Schema.html), [Phoenix.Router](https://phoenix.hexdocs.pm/Phoenix.Router.html), [Hex FAQ](https://hex.pm/docs/faq), [Expert LSP](https://github.com/expert-lsp/expert), [anúncio do time oficial de LSP (2024)](https://elixir-lang.org/blog/2024/08/15/welcome-elixir-language-server-team/)
- Blog oficial: [v1.9 (2019)](https://elixir-lang.org/blog/2019/06/24/elixir-v1-9-0-released/), [v1.15 (2023)](https://elixir-lang.org/blog/2023/06/19/elixir-v1-15-0-released/), [v1.16 (2023)](https://elixir-lang.org/blog/2023/12/22/elixir-v1-16-0-released/), [Type system: research → development (2023)](https://elixir-lang.org/blog/2023/06/22/type-system-updates-research-dev/), [Type inference of all constructs and the next 15 months (jan/2026)](https://elixir-lang.org/blog/2026/01/09/type-inference-of-all-and-next-15/), [v1.20: now a gradually typed language (jun/2026)](https://elixir-lang.org/blog/2026/06/03/elixir-v1-20-0-released/)
- Paper: Castagna, Duboc, Valim, [The Design Principles of the Elixir Type System](https://arxiv.org/abs/2306.06391) (2023)

Não verificado nesta sessão: a palestra de José Valim em que o formatter foi anunciado (a
busca achou apenas referências secundárias; os princípios do formatter abaixo vêm da
documentação oficial de `Code.format_string!/2`, que é fonte primária); a página
`ExUnit.DocTest` (retornou HTTP 421; o comportamento dos doctests vem de *Writing
documentation*); o processo formal da mailing list `elixir-lang-core` (o CONTRIBUTING só
menciona que pedidos de feature passam pela mailing list).

---

## Matriz

### Objetivo original
Linguagem funcional e dinâmica sobre a BEAM (VM do Erlang), com a concorrência e a tolerância
a falhas do OTP, mais produtividade, metaprogramação e ferramentas modernas. Em 2019 o time
declarou o conjunto de features "completo": "We don't have any major user-facing feature in
the works nor planned" ([v1.9](https://elixir-lang.org/blog/2019/06/24/elixir-v1-9-0-released/)).
Desde então a evolução é de ferramentas, diagnostics e tipos.

### Filosofia
Um núcleo pequeno e uniforme (quase tudo é chamada de função ou macro sobre a mesma AST), e
extensão por bibliotecas. A documentação oficial é conservadora com o próprio poder: "Macros
should only be used as a last resort", "explicit is better than implicit", "Clear code is
better than concise code" ([Macros](https://elixir.hexdocs.pm/macros.html)).

### Sintaxe
A expressividade sem cerimônia vem de quatro regras, não de muitas palavras-chave:
1. Parênteses opcionais em chamadas com argumentos (`field :nome, :string`); obrigatórios em
   chamada de aridade zero, para não confundir com variável ([Syntax reference](https://elixir.hexdocs.pm/syntax-reference.html)).
2. Keyword list como último argumento dispensa colchetes (`resources "/users", only: [:index]`).
3. `do ... end` é açúcar para um argumento keyword: `if x do a else b end` é
   `if(x, do: a, else: b)` ([Syntax reference](https://elixir.hexdocs.pm/syntax-reference.html)).
4. Poucas palavras reservadas: `true false nil when and or not in fn do end catch rescue after
   else` ([Syntax reference](https://elixir.hexdocs.pm/syntax-reference.html)). `if`, `def`,
   `defmodule`, `schema` são macros comuns, não sintaxe.

Consequência: um bloco `schema "users" do field :name, :string end` de biblioteca tem a mesma
aparência que uma construção nativa. É a raiz da elegância e do risco (ver Macros).

### Gramática
LALR(1) em `yecc`, cerca de 1.350 linhas, com uma tabela de precedência explícita (de `do`,
nível 5, a `.`, nível 310) e 3 conflitos shift/reduce esperados e documentados
([elixir_parser.yrl](https://github.com/elixir-lang/elixir/blob/main/lib/elixir/src/elixir_parser.yrl)).
Para que parênteses opcionais não criem ambiguidade, a gramática separa expressões
`matched`, `unmatched` e `no_parens`, e distingue `no_parens_one`, `no_parens_many` e
`no_parens_one_ambig`; blocos `do` não aninham sem parênteses em certos contextos. Lição: cada
açúcar "sem cerimônia" custou categorias gramaticais inteiras.

### Lexer/tokenizer
Escrito à mão em Erlang ([elixir_tokenizer.erl](https://github.com/elixir-lang/elixir/blob/main/lib/elixir/src/elixir_tokenizer.erl)).
É sensível ao caractere seguinte: o mesmo identificador vira `paren_identifier` (seguido de
`(`), `bracket_identifier` (`[`), `do_identifier` (`do`) ou `kw_identifier` (`nome:`). Emite
tokens `eol` para quebras de linha, rastreia a indentação de heredocs e dá erros com linha,
coluna, delimitador de abertura e o code point do caractere ofensivo. Ou seja: parte da
desambiguação que o parser precisaria foi empurrada para o lexer, que classifica pela
vizinhança.

### Parser
O parser gerado produz a AST diretamente; não há CST separada. Comentários não entram na
AST; o formatter usa `Code.string_to_quoted_with_comments` para reanexá-los e documenta que
"cannot currently preserve them around operators" ([Code](https://elixir.hexdocs.pm/Code.html)).

### AST
A AST é dado comum da linguagem: um nó é uma tupla `{nome, metadados, argumentos}`
("The building block of an Elixir program is a tuple with three elements",
[Quote and unquote](https://elixir.hexdocs.pm/quote-and-unquote.html)). Átomos, números,
listas, strings e tuplas de dois elementos são literais que se representam a si mesmos.
`quote` devolve a AST de um trecho, `unquote` injeta valores, `Macro.to_string` volta a
texto. Metadados carregam linha e coluna, e é deles que saem os diagnostics com posição.

### Representações intermediárias
AST expandida (macros resolvidas) → Erlang Abstract Format → Core Erlang → BEAM bytecode.
O compilador do Erlang faz o resto. (Cadeia conhecida da arquitetura; não reexaminada aqui
arquivo a arquivo.)

### Análise semântica
Expansão de macros com higiene de variáveis ("a variable defined inside a quote won't
conflict with a variable defined in the context where that macro is expanded",
[Macros](https://elixir.hexdocs.pm/macros.html)), resolução de aliases/imports, rastreamento
de dependências de compilação entre módulos. O rastreamento é frágil diante de macros: a
documentação lista "Untracked compile-time dependencies" e "Compile-time dependencies" como
anti-padrões ([Meta-programming anti-patterns](https://elixir.hexdocs.pm/macro-anti-patterns.html)).

### Sistema de tipos
Dinâmico e forte; desde 2023 está virando gradual com tipos de conjunto (união, interseção,
negação) e subtipagem semântica ([Castagna, Duboc, Valim](https://arxiv.org/abs/2306.06391)).
O tipo `dynamic()` é um intervalo de tipos: intersectar com ele torna um tipo gradual
([Gradual set-theoretic types](https://elixir.hexdocs.pm/gradual-set-theoretic-types.html)).
Plano em três marcos: inferência sem anotações (concluído na v1.20,
[jun/2026](https://elixir-lang.org/blog/2026/06/03/elixir-v1-20-0-released/)); structs
tipadas (previsto para v1.21, nov/2026); assinaturas de função, com as typespecs antigas
sendo aposentadas (v1.22, mai/2027, [plano de 15 meses](https://elixir-lang.org/blog/2026/01/09/type-inference-of-all-and-next-15/)).
Os dois últimos marcos ainda são plano, não entrega.

### Type inference
O ponto mais relevante para o Germanio. A inferência da v1.20 extrai tipos de padrões,
guards, `case` e condicionais; `rem(a + b, 8)` basta para inferir que `a` e `b` são inteiros
([jan/2026](https://elixir-lang.org/blog/2026/01/09/type-inference-of-all-and-next-15/)).
O critério de erro é deliberadamente estreito: reporta só "verified bugs", casos em que
"all combinations of a type *will* fail", e "generally avoids emitting false positive type
violations" ([Gradual set-theoretic types](https://elixir.hexdocs.pm/gradual-set-theoretic-types.html)).
É "best-effort": não promete achar todos os erros. O time mediu 12 de 13 categorias do
benchmark "If T" de estreitamento de tipos ([v1.20](https://elixir-lang.org/blog/2026/06/03/elixir-v1-20-0-released/)).
O próprio time admite que ainda precisa "assess its impact on the developer experience".

### Compiler/interpreter
Compilador para bytecode BEAM; IEx como REPL. A v1.20 adicionou `:module_definition`
(`:compiled` ou `:interpreted`) para reduzir tempo de build em projetos grandes
([v1.20](https://elixir-lang.org/blog/2026/06/03/elixir-v1-20-0-released/)).

### Runtime
BEAM/OTP: milhões de processos leves isolados, comunicação só por mensagens, supervisores
que reiniciam processos ("let it crash"), distribuição e troca de código a quente. É a base
do valor do Phoenix para tempo real (LiveView, Channels). Nada disso é sintaxe: é runtime.

### Memory management
Heap por processo com GC por processo (pausas locais, sem stop-the-world global); binários
grandes com contagem de referência compartilhada. Anti-padrão documentado: átomos criados
dinamicamente não são coletados ("Dynamic atom creation",
[Code anti-patterns](https://elixir.hexdocs.pm/code-anti-patterns.html)).

### Standard library
Pequena e coerente (`Enum`, `Stream`, `Map`, `String`, `Task`, `GenServer`, `Registry`),
mais acesso direto a todo o OTP do Erlang.

### Package manager
Mix (build, testes, tarefas, releases) e Hex. O repositório do Hex é imutável: uma versão só
pode ser alterada ou retirada até 60 minutos após a publicação (24 h na primeira versão); depois
disso, a saída é *retire*: a versão continua resolvível, mas quem a usa recebe aviso
([Hex FAQ](https://hex.pm/docs/faq)).

### Formatter
`mix format`, embutido desde a v1.6. Três princípios oficiais ([Code](https://elixir.hexdocs.pm/Code.html)):
nunca mudar a semântica por padrão; quase nenhuma configuração; e **nenhum nome embutido**: o
formatter não trata `def` ou `defmodule` de forma especial. O custo do terceiro princípio
aparece nas DSLs: sem configuração, `field :name, :string` ganharia parênteses. Por isso
existe `locals_without_parens`, e uma biblioteca **exporta** o seu (`export:
[locals_without_parens: ...]` no `.formatter.exs`); o projeto importa com `import_deps`
([mix format](https://mix.hexdocs.pm/Mix.Tasks.Format.html)). Isto é: a gramática não
sabe o que é DSL, e a informação de "estilo de DSL" precisou de um canal lateral entre
pacotes. O formatter também preserva escolhas do usuário (linhas em branco, `do:` versus
`do/end`, coleções multilinha). `--check-formatted` serve para CI; `--migrate` reescreve
construções deprecadas e **muda a AST**, com aviso de risco em projetos com muita
metaprogramação.

### Linter
Fora do núcleo (Credo, da comunidade; não examinado). No núcleo, o compilador emite avisos
e, desde 2023, existe uma referência viva de anti-padrões em quatro categorias (código,
design, processo, metaprogramação), que surgiu de pesquisa acadêmica
([v1.16](https://elixir-lang.org/blog/2023/12/22/elixir-v1-16-0-released/)).

### Language server
Três LSPs da comunidade (ElixirLS, Lexical, Next LS) competiam; em agosto de 2024 formaram um
time oficial ([anúncio](https://elixir-lang.org/blog/2024/08/15/welcome-elixir-language-server-team/)),
e o resultado é o Expert, com 0.1.0 em março de 2026 ([Expert](https://github.com/expert-lsp/expert)).
Lição histórica: dez anos de ferramentas fragmentadas até convergirem.

### IDE tooling
IEx com `h Modulo.funcao` lendo a documentação do bytecode; `Code.Fragment` para autocomplete
(não reexaminado); Livebook.

### Diagnostics
Evolução recente e incremental: a v1.15 passou a reportar vários erros por arquivo em vez de
parar no primeiro, via `Code.with_diagnostics/2`, pensado para editores
([v1.15](https://elixir-lang.org/blog/2023/06/19/elixir-v1-15-0-released/)); a v1.16 trouxe
trechos de código com cursor, delimitador não fechado apontando onde abriu e par de
delimitadores trocados mostrando os dois lados
([v1.16](https://elixir-lang.org/blog/2023/12/22/elixir-v1-16-0-released/)). Os avisos do
type checker (v1.18+) seguem o critério "só bug verificado".

### Testing
ExUnit embutido. Doctests: exemplos `iex>` dentro de `@doc` viram testes quando o módulo de
teste declara `doctest Modulo`, o que mantém a documentação correta
([Writing documentation](https://elixir.hexdocs.pm/writing-documentation.html)).

### Documentation
Cidadã de primeira classe: `@moduledoc`, `@doc`, `@typedoc` são atributos da linguagem, com
metadados (`:since`, `:deprecated`, `:group`); ficam em chunks do bytecode e são lidos em
runtime por `Code.fetch_docs/1` e pelo `h` do IEx. A documentação distingue *documentação*
(contrato para quem usa, sem acesso ao código) de *comentário* (para quem lê o código)
([Writing documentation](https://elixir.hexdocs.pm/writing-documentation.html)). ExDoc gera
HTML, e o hexdocs hospeda a documentação de todo pacote publicado.

### Evolution process
Propostas na mailing list `elixir-lang-core` (o [CONTRIBUTING](https://github.com/elixir-lang/elixir/blob/main/CONTRIBUTING.md)
menciona pedidos de feature pela lista; o rito formal não foi verificado). Uma versão minor a
cada 6 meses; mudanças grandes (tipos) passaram por paper, protótipo e release candidates
dedicados a cada etapa ([jan/2026](https://elixir-lang.org/blog/2026/01/09/type-inference-of-all-and-next-15/)).

### Backward compatibility
Contrato escrito ([Compatibility and deprecations](https://elixir.hexdocs.pm/compatibility-and-deprecations.html)):
comportamento documentado continua funcionando entre versões não-major. Deprecação em três
fases: *soft* (só na documentação), *hard* (aviso emitido) e remoção (só em major). Regra
central: a alternativa proposta "MUST exist for AT LEAST THREE minor versions" antes do aviso
começar. Features experimentais não têm garantia. Complemento mecânico: `mix format --migrate`.

### Principais acertos
- Uma AST uniforme e pequena que é dado da própria linguagem; ferramentas (formatter,
  macros, LSP) trabalham sobre a mesma estrutura.
- Declaração vira dado inspecionável: `Ecto.Schema` gera `__schema__(:fields)`,
  `__schema__(:type, campo)` etc. para introspecção em runtime
  ([Ecto.Schema](https://ecto.hexdocs.pm/Ecto.Schema.html)); `mix phx.routes` lista a tabela
  de rotas compilada ([Phoenix.Router](https://phoenix.hexdocs.pm/Phoenix.Router.html)).
- Composição linear com `|>` ("pipes the value on the left into the first argument",
  [Kernel](https://elixir.hexdocs.pm/Kernel.html)) e casamento de padrões como forma normal
  de decidir ("assertive style", [Code anti-patterns](https://elixir.hexdocs.pm/code-anti-patterns.html)).
- Inferência que só acusa o que é certamente erro.
- Política de deprecação com número (3 minors) e ferramenta de migração.
- Documentação e exemplos executáveis no mesmo lugar.

### Principais problemas
- Macros tornam invisível o que o código faz: `use` injeta código e dependências ocultas;
  "To understand exactly what will happen... it is necessary to have knowledge of the internal
  details" ([anti-patterns](https://elixir.hexdocs.pm/macro-anti-patterns.html)).
- Dependências de compilação criadas por macros causam cascatas de recompilação.
- Parênteses opcionais custaram uma gramática com categorias `no_parens` e conflitos
  aceitos; o pipe tem armadilhas de parênteses e de funções anônimas ([Kernel](https://elixir.hexdocs.pm/Kernel.html)).
- O formatter não conhece DSLs e precisa de configuração exportada por pacote.
- Tipos chegaram 12 anos depois da 1.0; typespecs (Dialyzer) e o novo sistema convivem
  até a v1.22.
- Ferramentas de editor fragmentadas por uma década.

### Complexidade acumulada
Moderada no núcleo (as features pararam em 2019), alta no ecossistema de macros: cada DSL
(Ecto, Phoenix, Absinthe, Ash) é uma mini-linguagem com escopo, regras e erros próprios, e
o erro de uma macro frequentemente aparece no código expandido, não na frase que o autor
escreveu (observação geral; não medida nesta pesquisa).

### O que Germanio pode aprender
- Um núcleo uniforme em que todo bloco declarativo tem a mesma forma de nó.
- Declaração compilada para dado introspectável (`__schema__`, `phx.routes`) equivale ao
  `ge explain`: a mesma estrutura serve ao runtime e à explicação.
- A regra "só erro verificado" na inferência.
- Um contrato de deprecação com prazos e um `--migrate`.
- Exemplos executáveis na documentação.

### O que Germanio NÃO deve copiar
- Macros definidas pelo usuário ou por pacotes (ver abaixo).
- Parênteses opcionais e keyword lists como mecanismo geral de açúcar.
- Formatter configurável por pacote (`locals_without_parens`): no Germanio as seções são
  gramática fechada, então o formatter pode conhecê-las sem canal lateral.
- `|>` como operador: o pipe resolve composição de funções, que o nível 1 não tem.
- A separação entre typespecs e tipos inferidos (duas verdades sobre o mesmo valor).

---

## Resposta à pergunta principal

O Elixir não é pouco cerimonial por ter muitas construções, e sim por ter uma forma só
(chamada com argumentos, bloco `do` como último argumento) e deixar que bibliotecas
preencham o vocabulário. `schema`/`field` (Ecto) e `pipeline`/`pipe_through`/`scope`/
`resources` (Phoenix) são diretamente comparáveis aos blocos do Germanio: seções nomeadas,
hierarquia por bloco, uma frase por linha, e compilação para uma estrutura introspectável.

A diferença é de onde vem a garantia. No Elixir, o significado de `field` é o que a macro
fizer: a gramática não sabe que `field` só vale dentro de `schema`, a mensagem de erro é a
que o autor da macro escreveu (ou uma do código expandido), o formatter precisa ser avisado,
e a análise de dependências pode perder o rastro. O time do Elixir diz isso na própria
documentação e trata macros como último recurso.

**Custo das macros para leigos.** Para quem nunca programou, uma macro de DSL tem quatro
custos que o Germanio já decidiu não pagar:
1. *Promessa falsa de uniformidade:* `field` parece palavra da linguagem, mas cada DSL tem
   regras próprias, e o leigo não sabe onde termina a linguagem e começa a biblioteca (é o
   problema do Inform 7/AppleScript já registrado em `docs/research/sintaxe-hierarquica.md`,
   agora dentro de uma linguagem de programador).
2. *Erros fora do texto escrito:* o erro pode apontar para código gerado.
3. *Dependências escondidas* (Green & Petre): `use` traz funções e comportamento que não
   aparecem no arquivo.
4. *Fim do determinismo verificável:* se o vocabulário é aberto, `ge check`, `ge fmt` e
   `ge explain` não podem garantir nada sem executar código do usuário.

A tese do Germanio (seções fechadas, redução a frases planas, capabilities no core) é a
alternativa às macros: o vocabulário cresce pelo core, com contrato, testes e diagnóstico,
não por código do usuário em tempo de compilação. O Elixir confirma o valor da forma
uniforme e da declaração introspectável e mostra o preço da extensibilidade aberta.

---

## Para o Germanio

| Lição | Classe | Problema concreto que resolve | Arquivo afetado |
|---|---|---|---|
| Declaração vira dado inspecionável pelo runtime **e** pelas ferramentas (`__schema__`, `mix phx.routes`) | ADOTAR (já é a direção) | `ge explain` e o runtime podem divergir se lerem estruturas diferentes; garantir que ambos leiam o mesmo `ast.App` | `compiler/parser/resolver.go`, `tooling/explicar/explicar.go` |
| Uma forma de nó para todo bloco (o `{nome, meta, args}` do Elixir) | ADAPTAR | as seções hoje são tabelas fechadas reduzidas a frases; manter uma forma canônica única (tupla recurso/seção/papel/ação) evita que cada seção nova ganhe estrutura própria | `compiler/parser/hierarquia.go`, `compiler/ast` |
| Inferência que só reporta erro verificado (sem falso positivo) | ADOTAR | o princípio de subtração pede inferir tipos de campo, relações e permissões; um aviso incerto ensina o leigo a ignorar avisos | `compiler/semantic/check.go`, `compiler/diagnostics` |
| Distinguir "certamente errado" de "não sei": no Germanio, "não sei" deve virar pergunta ou erro explícito, nunca `dynamic()` silencioso | ADAPTAR | o Elixir aceita incerteza porque é dinâmico; o Germanio promete determinismo, então a incerteza tem que aparecer (por exemplo como diagnóstico "ambíguo: diga X ou Y") | `compiler/parser/resolver.go`, `tooling/intelligence/questions.go` |
| Contrato de deprecação: soft, aviso, remoção só em major; alternativa existe há N versões antes do aviso | ADOTAR | não há mecanismo de deprecação no Germanio (nenhuma ocorrência em `compiler/`, `tooling/`, `runtime/`); a "refatoração retroativa" de `INTENCAO.md` vai quebrar `.ge` antigos sem aviso | `compiler/diagnostics/diagnostics.go`, `docs/INTENCAO.md` › Evolução |
| `ge fmt --migrar` que reescreve construção antiga na nova e reparseia para provar equivalência | ADAPTAR | o `--migrate` do Elixir muda a AST com risco; no Germanio a reescrita pode ser verificada pelas tuplas canônicas, como o `ge fmt` já faz | `tooling/formatter/intencao.go` |
| Formatter sem nomes embutidos e com `locals_without_parens` exportado por pacote | EVITAR | o vocabulário do Germanio é gramática fechada; o formatter deve conhecer as seções diretamente, sem canal de configuração | `tooling/formatter/` |
| Macros de usuário, `use`, DSLs de biblioteca | EVITAR | quebrariam `ge check`/`ge explain` determinísticos e o teste do leigo; capability nova entra pelo core (sequência de `INTENCAO.md` › Evolução) | `docs/INTENCAO.md`, `skills/germanio-simplicity/SKILL.md` §30 |
| Parênteses opcionais e keyword lists como açúcar geral | EVITAR | custaram ao Elixir categorias `no_parens` e conflitos LALR aceitos; o Germanio já tem "uma linha lógica, um item" | `compiler/parser/hierarquia.go` |
| Exemplos executáveis na documentação (doctest): um exemplo de `docs/INTENCAO.md` é um teste | ADOTAR | a documentação normativa e o parser podem divergir (risco de vários "parsers"); extrair os blocos ```` ```ge ```` de `INTENCAO.md` e rodar `ge check` neles | `docs/INTENCAO.md`, testes em `compiler/parser` |
| Documentação como dado da linguagem (lida por `h` em runtime) | INVESTIGAR | `ge explain` poderia citar a documentação da construção usada ao lado do fato; exige decidir onde vive o texto de cada seção | `tooling/explicar/`, `compiler/diagnostics/diagnostics.go` (`Explanations`) |
| Vários erros por arquivo, com trecho, cursor e "onde abriu" o bloco | ADOTAR | com layout por indentação, um erro de nível mal fechado é mais útil mostrando a linha pai; parar no primeiro erro atrasa o leigo | `compiler/diagnostics/diagnostics.go`, `compiler/parser/hierarquia.go` |
| Um LSP único cedo, para evitar uma década de ferramentas divergentes | INVESTIGAR | risco conhecido de vários "parsers" (regex do VS Code, formatter, parser); o LSP deve reusar o parser Go, não a gramática TextMate | `vscode-germanio/tools/gerar_gramatica.py`, futuro `tooling/lsp` |
| Runtime com processos isolados e supervisores | INVESTIGAR | a fase seguinte prevê tempo real e grande escala; o modelo OTP é mecanismo do core, nunca sintaxe do domínio | `runtime/` |

**Achado lateral no Germanio:** a explicação `GE1002` em `compiler/diagnostics/diagnostics.go`
e as mensagens em `compiler/parser/germanio.go` dizem "dois espaços por nível", enquanto a
norma da camada de intenção (`docs/INTENCAO.md` › Layout) fixa 4 espaços como forma canônica
e aceita outros passos consistentes. É provável que seja o núcleo estrito versus o dialeto de
aplicação, mas o mesmo código de erro com instruções diferentes é exatamente a divergência
entre front-ends que o Elixir levou dez anos para resolver no LSP.
