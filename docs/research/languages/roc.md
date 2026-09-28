# Roc: estudo dirigido para o Germanio

**Data:** 2026-09-28. **Status:** pesquisa, sem força normativa (a norma é `docs/INTENCAO.md`).
**Pergunta principal:** quais experimentos modernos de design de linguagem do Roc podem ser
relevantes para o Germanio?

**Aviso de maturidade.** O Roc ainda não teve nenhuma versão numerada. O README diz "Roc is
not ready for a 0.1 release yet" ([roc-lang/roc](https://github.com/roc-lang/roc)); o autor
diz que a 0.1.0 do compilador novo deve sair "later this year" ([rtfeldman.com/rust-to-zig](https://rtfeldman.com/rust-to-zig)).
Nada aqui é prática comprovada em produção ampla. Cada ideia vem marcada:
**[EXPERIMENTAL]** para desenho sem evidência de uso em escala, **[DOCUMENTADO]** para
comportamento descrito na documentação e visível no repositório, **[NÃO VERIFICADO]** quando
não consegui confirmar.

## Fontes consultadas

Consultadas nesta sessão (WebFetch/WebSearch ou `gh api` no repositório):

- Site: [roc-lang.org](https://www.roc-lang.org/), [FAQ](https://www.roc-lang.org/faq), [friendly](https://www.roc-lang.org/friendly), [functional](https://www.roc-lang.org/functional), [fast](https://www.roc-lang.org/fast), [SafeMath example](https://www.roc-lang.org/examples/SafeMath/README)
- Repositório (branch `main`): [README](https://github.com/roc-lang/roc), [src/README.md](https://github.com/roc-lang/roc/blob/main/src/README.md) (tabela de status da reescrita), [docs/mini-tutorial-new-compiler.md](https://github.com/roc-lang/roc/blob/main/docs/mini-tutorial-new-compiler.md) (para onde `/tutorial` redireciona), `docs/langref/` ([platforms.md](https://github.com/roc-lang/roc/blob/main/docs/langref/platforms.md), [functions.md](https://github.com/roc-lang/roc/blob/main/docs/langref/functions.md), [compile-time.md](https://github.com/roc-lang/roc/blob/main/docs/langref/compile-time.md), [static-dispatch.md](https://github.com/roc-lang/roc/blob/main/docs/langref/static-dispatch.md), [types.md](https://github.com/roc-lang/roc/blob/main/docs/langref/types.md), [comments-and-docs.md](https://github.com/roc-lang/roc/blob/main/docs/langref/comments-and-docs.md)), [src/fmt/README.md](https://github.com/roc-lang/roc/blob/main/src/fmt/README.md), o snapshot [test/snapshots/reporting/reporting_type_mismatch_multiline.md](https://github.com/roc-lang/roc/blob/main/test/snapshots/reporting/reporting_type_mismatch_multiline.md), a listagem de `src/lsp/` e `test/fx/allow_errors_type_mismatch.roc`
- Reescrita em Zig: gist de Richard Feldman, [Rewriting a Language's Compiler](https://gist.github.com/rtfeldman/77fb430ee57b42f5f2ca973a3992532f) (2025), e o balanço [How Our Rust-to-Zig Rewrite is Going](https://rtfeldman.com/rust-to-zig) (15/07/2026)
- Issues abertas do compilador novo, como amostra de maturidade: [#11532](https://github.com/roc-lang/roc/issues/11532), [#11640](https://github.com/roc-lang/roc/issues/11640), [#11767](https://github.com/roc-lang/roc/issues/11767)

As páginas `roc-lang.org/platforms` e `/docs/main/langref/platforms` renderizam só o menu no
fetch; o conteúdo foi lido no Markdown de origem no repositório.

---

## Matriz

### Objetivo original
"Fast, friendly, functional" ([roc-lang.org](https://www.roc-lang.org/)): compilar para
código de máquina ou WebAssembly, ser amigável em sintaxe, semântica e ferramentas, e ser
funcional de paradigma único. Descende do Elm (o autor veio da comunidade Elm), mas fora do
navegador, via plataformas.

### Filosofia
Decisões por subtração, justificadas por escrito no FAQ: sem currying ("Roc is not one of
those languages" que trocam mensagens de erro por concisão), sem tipos de ordem superior nem
rank-N (para manter inferência principal decidível), sem tipos refinados/dependentes/lineares
(tempo de compilação), sem igualdade de funções (indecidível), sem `null`/`Maybe` (usa tags
com nome do motivo), sem imports curinga (paralelizar a resolução de nomes por módulo), sem
compilar para JS/JVM/BEAM ([FAQ](https://www.roc-lang.org/faq)). **[DOCUMENTADO]**

### Sintaxe
Chamadas com parênteses e vírgulas (`Stdout.line!("Hi")`), lambdas `|x| ...`, blocos `{ }`,
`?` postfixo para propagar erro, métodos por *static dispatch* (`Counter.new().increment()`),
`var` para variável local mutável e `for`/`while`/`break`/`return`
([mini-tutorial](https://github.com/roc-lang/roc/blob/main/docs/mini-tutorial-new-compiler.md),
[functional](https://www.roc-lang.org/functional), [static-dispatch.md](https://github.com/roc-lang/roc/blob/main/docs/langref/static-dispatch.md)).
A sintaxe mudou muito: o gist da reescrita diz que "the grammar has evolved to the point where
a different foundational parsing strategy makes sense" ([gist](https://gist.github.com/rtfeldman/77fb430ee57b42f5f2ca973a3992532f)).
Que a sintaxe anterior usava aplicação por espaço, no estilo Elm, e "abilities" em vez de
static dispatch vem da minha memória, **[NÃO VERIFICADO]** nesta sessão; a documentação atual
diz que "Roc's only ad hoc polymorphism system is static dispatch".

### Gramática
Não há gramática formal publicada que eu tenha encontrado; o `langref` é prosa com exemplos.
**[NÃO VERIFICADO]** se existe uma especificação formal.

### Lexer/tokenizer
Parte de `src/parse/` no compilador em Zig (não examinado arquivo a arquivo). O sufixo `!` no
nome faz parte do identificador e carrega significado (efeito), ver Análise semântica.

### Parser
Reescrito como descendente recursivo "with improved error tolerance": o parser anterior "is not
as error-tolerant as we want it to be" ([gist](https://gist.github.com/rtfeldman/77fb430ee57b42f5f2ca973a3992532f)).
A AST é armazenada em arrays planos (`NodeStore.zig`, `Node.zig` em `src/parse/`), não em
árvore de ponteiros. Na tabela de status da reescrita, o parse aparece como "Polished" só para
algumas áreas ([src/README.md](https://github.com/roc-lang/roc/blob/main/src/README.md)).

### AST
AST plana indexada por IDs. A canonicalização também passou de enums recursivos para "flat
arrays with IDs", para cache em disco ([gist](https://gist.github.com/rtfeldman/77fb430ee57b42f5f2ca973a3992532f)),
e o cache é lido com "zero-parse deserialization", "at roughly the speed of memcpy"
([rust-to-zig](https://rtfeldman.com/rust-to-zig)). **[EXPERIMENTAL]** quanto ao ganho real
para usuários; os números são do autor.

### Representações intermediárias
Parse → canonicalização (CIR) → resolução de imports → checagem de tipos → interpretador (o
backend de desenvolvimento passou de código de máquina para interpretação) → especialização,
lifting, reference counting, IR baixa → bitcode LLVM ([src/README.md](https://github.com/roc-lang/roc/blob/main/src/README.md),
[gist](https://gist.github.com/rtfeldman/77fb430ee57b42f5f2ca973a3992532f)).

### Análise semântica
O experimento mais original é a **inferência de pureza**: "A function is effectful if it calls
another effectful function, and otherwise it's pure. Effectful functions can only be called by
other effectful functions" ([functions.md](https://github.com/roc-lang/roc/blob/main/docs/langref/functions.md)).
O tipo usa `=>` em vez de `->` e o nome termina em `!`. Toda declaração de topo é avaliada em
tempo de compilação, porque não pode ter efeito; um crash em tempo de compilação vira erro de
compilação ([compile-time.md](https://github.com/roc-lang/roc/blob/main/docs/langref/compile-time.md)).
**[DOCUMENTADO; EXPERIMENTAL como design]**

### Sistema de tipos
Estático e "sound" ("you'll never get a runtime type mismatch if everything type-checked",
[friendly](https://www.roc-lang.org/friendly)). Hindley-Milner com restrições deliberadas,
rank-1 "indefinitely" ([FAQ](https://www.roc-lang.org/faq), [types.md](https://github.com/roc-lang/roc/blob/main/docs/langref/types.md)).
Uniões de tags estruturais (`[Loading, Loaded(Artist)]`), registros estruturais, tipos
nominais com métodos. Erros recuperáveis via `Try` com tags nomeadas.

### Type inference
Completa: "it infers the types of everything you write. All type annotations in Roc are
optional" ([mini-tutorial](https://github.com/roc-lang/roc/blob/main/docs/mini-tutorial-new-compiler.md)).
O preço aparece nas mensagens: no snapshot de erro, o tipo inferido de `[1, 2, 3]` é
`List(a) where [a.from_numeral : Numeral -> Try(a, [InvalidNumeral(Str)])]`
([snapshot](https://github.com/roc-lang/roc/blob/main/test/snapshots/reporting/reporting_type_mismatch_multiline.md)).
A inferência completa é real; o tipo inferido, porém, não é linguagem de leigo.

### Compiler/interpreter
Reescrito de Rust para Zig desde o início de 2025. Balanço do autor em 15/07/2026: paridade de
features com o compilador antigo após 487 dias; 354 mil linhas de Rust contra cerca de 464 mil
de Zig; build incremental do compilador em 35 ms (Zig 0.17 nightly) contra 3,4 s (Rust); 21
bugs de corrupção de memória no Rust contra 10 no Zig; e o ganho de build depende de um
`-fincremental` que só funciona no nightly ([rust-to-zig](https://rtfeldman.com/rust-to-zig)).
Motivos declarados: tempo de build do Rust ("20+ seconds to rebuild after a change in our Rust
parser"), alocadores explícitos, struct-of-arrays, e o fato de o roadmap já prever reescrever
quase todas as partes, menos a inferência de tipos ([gist](https://gist.github.com/rtfeldman/77fb430ee57b42f5f2ca973a3992532f)).
Não é self-hosted, e de propósito ([FAQ](https://www.roc-lang.org/faq)). A amostra de issues
abertas mostra panics do próprio compilador em `roc check` e `roc test` (por exemplo
[#11532](https://github.com/roc-lang/roc/issues/11532), [#11767](https://github.com/roc-lang/roc/issues/11767)).

### Runtime
Não há runtime da linguagem no sentido usual: quem roda é a plataforma. "The host implements
`main()`", e a aplicação compila para algo como "a C library which the platform can choose to
call (or not)" ([platforms.md](https://github.com/roc-lang/roc/blob/main/docs/langref/platforms.md)).

### Memory management
Contagem de referências automática (etapa "Reference Counting" no pipeline,
[src/README.md](https://github.com/roc-lang/roc/blob/main/src/README.md)), com alocação e
liberação implementadas pela plataforma. Exemplo citado: um servidor web com arena por
requisição, `free` vazio e arena zerada ao responder ([platforms.md](https://github.com/roc-lang/roc/blob/main/docs/langref/platforms.md)).
**[EXPERIMENTAL]** (o servidor citado, `nea`, é "work-in-progress").

### Standard library
Só funções e estruturas de dados, sem nenhum I/O: "the standard library contains only
functions and data structures; an application gets all of its I/O primitives from its
platform" ([platforms.md](https://github.com/roc-lang/roc/blob/main/docs/langref/platforms.md)).

### Package manager
Não há registro central: plataformas e pacotes são URLs para arquivos `.tar.zst` cujo nome é
um hash de conteúdo, e um pacote só pode usar a plataforma da aplicação se declarar "the
identical platform version and content hash" ([platforms.md](https://github.com/roc-lang/roc/blob/main/docs/langref/platforms.md)).
**[EXPERIMENTAL]**

### Formatter
`roc fmt`, "designed with the time-saving feature of having no configuration options"
([friendly](https://www.roc-lang.org/friendly)). Na reescrita, passa a usar o algoritmo de
Wadler, "A prettier printer", para respeitar largura de linha ([gist](https://gist.github.com/rtfeldman/77fb430ee57b42f5f2ca973a3992532f));
é um módulo reutilizável, pensado para ser integrado a outras ferramentas
([src/fmt/README.md](https://github.com/roc-lang/roc/blob/main/src/fmt/README.md)).

### Linter
Não há linter separado; `roc check` concentra os avisos ([mini-tutorial](https://github.com/roc-lang/roc/blob/main/docs/mini-tutorial-new-compiler.md)).

### Language server
Embutido no compilador (`src/lsp/`: diagnostics, completion, hover, rename, semantic tokens,
code actions), dividido entre testes unitários e testes de integração com o compilador real
([src/README.md](https://github.com/roc-lang/roc/blob/main/src/README.md)). É o oposto do
caminho do Elixir: um LSP só, dentro do compilador, desde o começo. **[EXPERIMENTAL]** quanto à
maturidade.

### IDE tooling
Compilador completo no navegador, via WebAssembly ("You can run the full compiler in the
browser without a backend server!", [roc-lang.org](https://www.roc-lang.org/)); REPL em wasm
(`src/repl_wasm`, `src/playground_wasm`).

### Diagnostics
Estilo Elm: título, trecho do código, explicação em frase, o tipo encontrado e o esperado. O
detalhe arquitetural que interessa é outro: um erro é um **documento estruturado**
(`severity`, `title`, `region`, `headline`, blocos `reflow`/`source-region`/`code-block`)
renderizado para CLI, Markdown e outros alvos, e cada relatório é teste de snapshot com a
saída de cada renderizador ([snapshot](https://github.com/roc-lang/roc/blob/main/test/snapshots/reporting/reporting_type_mismatch_multiline.md)).
Programas com erro de compilação podem rodar: "you can run a program that has compile-time
errors" ([friendly](https://www.roc-lang.org/friendly)); o snapshot marca o erro de tipo com
`severity runtime_error`, e o repositório tem testes como `test/fx/allow_errors_type_mismatch.roc`.
A forma exata da flag de CLI está **[NÃO VERIFICADO]**.

### Testing
`expect` no topo do arquivo (rodado por `roc test`) e `expect` inline; `dbg` para depuração
([mini-tutorial](https://github.com/roc-lang/roc/blob/main/docs/mini-tutorial-new-compiler.md)).
Doc comments com `expect` dentro de bloco de código ([comments-and-docs.md](https://github.com/roc-lang/roc/blob/main/docs/langref/comments-and-docs.md)).

### Documentation
Doc comments `## ` presos à atribuição seguinte; se não houver atribuição logo abaixo, viram
comentário comum. Regra explícita: "Roc's compiler never derives any semantic meaning from
comments" ([comments-and-docs.md](https://github.com/roc-lang/roc/blob/main/docs/langref/comments-and-docs.md)).
Gerador de documentação com resolução de aliases e links automáticos, refeito na reescrita.

### Evolution process
Centralizado no autor e no chat Zulip; decisões registradas no FAQ e em design docs no
repositório (`docs/superpowers/specs/`, por exemplo o design de unificação iterativa de
2026-07-02). Sem RFCs formais **[NÃO VERIFICADO]**.

### Backward compatibility
Nenhuma garantia: pré-0.1, sintaxe e semântica mudaram na reescrita. Não se aplica como
contrato.

### Principais acertos
- Separação aplicação/plataforma com I/O exclusivo da plataforma ("Exclusivity is the point!",
  [FAQ](https://www.roc-lang.org/faq)).
- Inferência de pureza a partir do grafo de chamadas, sem anotação, com um marcador visível
  (`!`) no nome.
- Avaliação em tempo de compilação de tudo o que é puro no topo, e crash em tempo de compilação
  virando erro de compilação.
- FAQ que registra por que cada feature **não** existe.
- Diagnostics como documento estruturado com testes de snapshot por renderizador.
- Formatter e LSP dentro do compilador, sem configuração.

### Principais problemas
- Imaturidade: nenhuma versão, reescrita completa, panics no compilador, sintaxe instável.
- A promessa "sem exceções em runtime" tem limites: tipos não falham em runtime, mas o programa
  pode cair. `crash` existe para o inalcançável e para o que não se trata com elegância, `+`
  cai em overflow ("Integer addition overflowed!"), e a alternativa checada "runs slower"
  ([SafeMath](https://www.roc-lang.org/examples/SafeMath/README)); funções puras "are not
  guaranteed to be total" ([functions.md](https://github.com/roc-lang/roc/blob/main/docs/langref/functions.md)).
- Tipos inferidos vazam para o usuário em forma técnica (`where [a.from_numeral : ...]`).
- O agendamento concorrente pela plataforma é "theoretically possible today", com "missing
  pieces" ([platforms.md](https://github.com/roc-lang/roc/blob/main/docs/langref/platforms.md)).
- Uma aplicação tem uma plataforma só, sem composição; é uma decisão consciente, mas limita
  aplicações que precisam de vários domínios (web mais CLI mais jobs).

### Complexidade acumulada
Baixa na superfície visível, alta na implementação: 464 mil linhas de Zig para uma linguagem
que ainda não lançou a 0.1. A simplicidade para o usuário é comprada com complexidade no
compilador (monomorfização, reference counting, inferência completa), o que é coerente com a
divisão do Germanio ("Go implementa mecanismos").

### O que Germanio pode aprender
Ver "Para o Germanio". Em resumo: a fronteira de efeitos como contrato verificável, a
inferência de propriedades pelo grafo, o FAQ de subtração, os diagnostics estruturados com
snapshot, e "comentário nunca tem semântica".

### O que Germanio NÃO deve copiar
- A superfície funcional (lambdas, `?`, `Try`, métodos, tipos na mensagem): o público do nível
  1 não programa.
- Plataformas escritas pelo usuário em outra linguagem com ABI de C: no Germanio, o core é a
  única "plataforma" e os adaptadores ficam em `.ge` de nível 5.
- Pacotes por URL com hash, sem registro, antes de existir necessidade de pacotes.
- Rodar programa com erro de compilação: contradiz o `ge check` como portão (ver abaixo).
- A reescrita de compilador como estratégia: o custo (487 dias) só se paga com a arquitetura
  do parser como gargalo comprovado.

---

## Resposta à pergunta principal: experimentos relevantes

1. **Aplicação pura, plataforma com efeitos exclusivos.** **[DOCUMENTADO; EXPERIMENTAL em
   escala]** O paralelo com domínio/core/adaptador é direto, mas a direção é outra. No Roc, a
   plataforma é intercambiável e a aplicação escolhe uma; no Germanio, o core é fixo e o
   domínio nem sabe que efeitos existem. O que vale importar é a **garantia**: "There are no
   escape hatches", pois até chamar outra linguagem passa por primitiva da plataforma, que pode
   negá-la ([platforms.md](https://github.com/roc-lang/roc/blob/main/docs/langref/platforms.md)).
   Para o Germanio, isso vira uma regra verificável: um `.ge` de domínio não consegue citar
   primitiva de nível 4/5, e o `ge check` prova, em vez de só recomendar. Os benefícios de
   ecossistema ("these mismatches can be prevented at build time") equivalem a recusar em
   `ge check` o uso de uma capability que o alvo de deploy não oferece.

2. **Inferência de pureza pelo grafo de chamadas.** **[DOCUMENTADO]** A regra "é efetivo se
   chama algo efetivo" é determinística, local e explicável: a explicação é a cadeia de
   chamadas até a primitiva. O Germanio pode aplicar a mesma forma de inferência a propriedades
   do domínio: uma ação "envia e-mail" porque aciona um evento que envia; uma página "exige
   login" porque mostra um dado privado. `ge explain` mostraria a cadeia. É subtração pura: o
   usuário não declara a propriedade, e ela nunca fica errada.

3. **Tudo que é puro e de topo é calculado em tempo de compilação, e crash vira erro de
   compilação.** **[DOCUMENTADO]** No Germanio, configurações de nível 2 (`login bloqueia após
   10 tentativas`) e valores derivados podem ser resolvidos e validados no `ge check`, não no
   primeiro request.

4. **FAQ de subtração ("why no X").** **[DOCUMENTADO]** Cada recusa tem motivo técnico
   registrado (decidibilidade, tempo de compilação, mensagens de erro). O Germanio já tem
   "Decisões e limites" em `INTENCAO.md`; falta a lista pública do que **não** entrará e por
   quê (macros, currying, linguagem natural livre, IA em runtime), para encerrar debates
   repetidos.

5. **Diagnostic como documento estruturado com snapshot por renderizador.** **[DOCUMENTADO]**
   Resolve o risco de vários "parsers" pelo lado da saída: CLI, LSP e VS Code consomem o mesmo
   documento, e o teste fixa o texto de cada um.

6. **Inferência completa sem anotação.** **[DOCUMENTADO]** Confirma o princípio de subtração,
   mas com um alerta: inferir tudo não basta se o resultado for mostrado em linguagem de tipos.
   A explicação precisa voltar em frase do domínio ("`nome` é texto porque aparece em `busca por
   nome`"), não como tipo.

7. **Formatter sem configuração, algoritmo de Wadler, módulo reutilizável; LSP dentro do
   compilador.** **[EXPERIMENTAL quanto à maturidade]** Reforça o que o Germanio já decidiu
   (`ge fmt` canônico) e aponta onde colocar o futuro LSP.

8. **Rodar programas com erro.** **[EXPERIMENTAL]** Útil para programadores que iteram rápido;
   para leigos, é perigoso, porque um erro vira crash em uso. Não copiar.

---

## Para o Germanio

| Lição | Classe | Problema concreto que resolve | Arquivo afetado |
|---|---|---|---|
| Fronteira de efeitos verificável: o domínio não alcança primitivas de nível 4/5; só `integracoes/` alcança; "no escape hatches" | ADOTAR | hoje a verificação é parcial: `checkRole` em `runtime/engine.go` impede `integracoes/` de declarar produto e `backend/` de conter `traduza`, mas não impede `backend/` de declarar rotas, lógica ou primitivas de nível 4, e roda no carregamento do projeto organizado; estender a regra a "domínio não cita primitiva de nível 4/5" e expô-la no `ge check` | `compiler/semantic/check.go`, `compiler/parser/resolver.go`, `docs/INTENCAO.md` › Níveis |
| `ge check` recusa capability que o alvo não oferece (o "mismatch em build time" das plataformas) | INVESTIGAR | `germanio build`/`docker` e o runtime podem descobrir em execução que falta algo (SMTP, storage); depende de haver um manifesto de capabilities por alvo | `tooling/intelligence/capabilities.go`, `tooling/intelligence/manifest.go` |
| Inferir propriedades pelo grafo (pureza no Roc; no Germanio: "exige login", "gera notificação", "altera estado"), com a cadeia mostrada por `ge explain` | ADOTAR | princípio de subtração: o autor não declara o que é consequência; `ge explain` já mostra origem de fatos, e passaria a mostrar a cadeia de derivação | `tooling/explicar/explicar.go`, `tooling/intelligence/graph.go` |
| Avaliar em `ge check` tudo o que é estático (configurações de nível 2, valores iniciais, regras sem dados) e transformar falha em erro de compilação | ADOTAR | erros de configuração aparecem hoje no runtime ou no primeiro uso | `compiler/semantic/check.go`, `runtime/engine.go` |
| Diagnostic como documento estruturado (título, região, blocos) renderizado para CLI/LSP/Markdown, com snapshot por renderizador | ADAPTAR | `Diagnostic.Error()` monta texto direto; um LSP ou o VS Code teriam de reparsear a string ou duplicar a formatação | `compiler/diagnostics/diagnostics.go` e testes |
| Explicar tipos inferidos em frase do domínio, nunca em notação de tipos | ADOTAR | evitar o equivalente a `List(a) where [...]` quando o Germanio inferir tipos de campos a partir do uso | `compiler/diagnostics`, `tooling/explicar/` |
| FAQ "por que o Germanio não tem X" | ADOTAR | debates recorrentes (macros, linguagem natural livre, IA em runtime, sintaxe configurável) sem registro único de recusa | `docs/INTENCAO.md` › Decisões e limites, ou `docs/FAQ.md` |
| "O compilador nunca deriva semântica de comentários" como regra escrita | ADOTAR | com `#` em fim de linha pertencendo à linha (Layout), explicitar que comentário nunca muda fato garante a verificação de equivalência do `ge fmt` | `docs/INTENCAO.md` › Layout, `tooling/formatter/intencao.go` |
| LSP dentro do compilador, reusando parser e diagnostics, com testes de integração no compilador real | ADAPTAR | risco conhecido de vários "parsers" (regex TextMate, formatter, parser) | futuro `tooling/lsp`, `vscode-germanio/tools/gerar_gramatica.py` |
| Formatter com algoritmo de Wadler (largura de linha) | EVITAR por ora | a sintaxe do Germanio é uma linha lógica por item e não quebra linha; o algoritmo resolve um problema que o Germanio não tem | `tooling/formatter/` |
| Rodar programa com erro de compilação | EVITAR | contradiz o `ge check` como portão determinístico e o público leigo | `tooling/gecli/cli.go` |
| Plataformas de terceiros com host em outra linguagem, pacotes por URL/hash | EVITAR | não existe hoje necessidade de várias "plataformas"; o core é único e os adaptadores ficam em `.ge` | — |
| Memória e agendamento especializados por domínio (arena por request, scheduler da plataforma) | INVESTIGAR | a fase seguinte (grande escala, tempo real) pode usar a ideia dentro do core, que conhece o domínio "app web"; no Roc isso ainda está incompleto | `runtime/servidor/`, `runtime/engine.go` |
| Reescrever o compilador para ganhar tolerância a erro no parser | EVITAR como estratégia | o Roc precisou de 487 dias; no Germanio, a tolerância a erro deve crescer no parser atual (`layoutTree` já isola níveis) | `compiler/parser/hierarquia.go` |

**Contradições com a abordagem atual do Germanio:** nenhuma de fundo. Duas tensões. (1) O Roc
põe a garantia de separação de efeitos no compilador; o Germanio a põe sobretudo em norma e
revisão (`SKILL.md` §34b, checklist), com uma verificação parcial por pasta (`checkRole` em
`runtime/engine.go`). A pesquisa sugere que a norma inteira (níveis 4/5 fora do domínio) vire
verificação no compilador, exposta em `ge check`. (2) A inferência completa do Roc mostra que "inferir tudo" desloca a
complexidade para a explicação; o `ge explain` precisa ser tão cuidado quanto a inferência, ou
o princípio de subtração vira dependência escondida.
