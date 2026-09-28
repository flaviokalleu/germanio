# Matriz comparativa do ecossistema

**Data:** 2026-09-28. **Status:** pesquisa, **sem força normativa** (ver [README.md](README.md)).
Código do Germanio conferido na revisão `0c12051` (master).

**Não é um ranking.** A matriz resume, numa linha por linguagem, o que os estudos deste
diretório registraram; a coluna "Germanio Lesson" aponta a lição correspondente em
[GERMANIO_LESSONS.md](GERMANIO_LESSONS.md) (ids `A`, `P`, `E`, `I`). A última linha é o próprio
Germanio, preenchida com o estado **atual** e as lacunas de
[`GERMANIO_GAPS.md`](../../../GERMANIO_GAPS.md), não com o que a norma promete.

Convenções: "—" = não se aplica ou o estudo não trata; "(nv)" = não verificado no estudo;
células resumem o estudo citado na primeira coluna, que traz as fontes (URLs). As linhas das
descobertas (Wasp, Hedy etc.) vêm de [DISCOVERIES.md](DISCOVERIES.md) e
[ECOSYSTEM_TRIAGE.md](ECOSYSTEM_TRIAGE.md); nelas muitas colunas técnicas não se aplicam,
porque o que ensinam é a história do projeto.

A tabela está dividida em duas por largura. A primeira trata da linguagem e da execução; a
segunda, do ecossistema e da lição.

## Parte 1 — Linguagem e execução

| Language | Syntax | Typing | Memory | Runtime | Compilation | Concurrency | Errors |
|---|---|---|---|---|---|---|---|
| [Python](python.md) | off-side (indentação), expressões | dinâmica; hints graduais (mypy, terceiros) | contagem de referências + GC de ciclos | CPython VM; GIL (free-threaded opcional) | bytecode; gramática PEG (PEP 617) | threads com GIL, `asyncio` (coloração) | exceções; mensagens especializadas desde 3.10 |
| [Go](go.md) | chaves; `gofmt` único | estática nominal; generics; inferência local | GC concorrente, poucos botões | scheduler M:N no binário | AOT rápido; binário estático | goroutines, canais, `context`, `errgroup` | valor `error`; `panic` raro |
| [Rust](rust.md) | chaves, expressões, macros | estática, ownership, traits | ownership/borrow, sem GC | sem runtime; executor async externo | AOT (LLVM), lento | `async`/`await` colorido, `Send`/`Sync` | `Result`/`Option` + `?`; diagnósticos estruturados |
| [Swift](swift.md) | chaves, rótulos de argumento | estática, inferência bidirecional, protocolos | ARC | runtime Swift/ObjC | AOT (LLVM); type-check lento em expressões | actors, `async`, `Sendable` (Swift 6) | `throws`, `Optional`; fix-its |
| [Nim](nim.md) | off-side; UFCS | estática, generics, macros | ARC/ORC | gera C/C++/JS | AOT via C | `async` por macro, threads | exceções |
| [TypeScript](typescript.md) | chaves (JS + anotações) | estrutural gradual; `any` | GC do motor JS | Node/Deno/Bun/navegador | transpila (`tsc`, port para Go) | event loop, `async` colorido | exceções; catálogo de diagnósticos |
| [Elixir](elixir.md) | `do`/`end`, pipes, macros | dinâmica; checagem gradual (1.18+) | GC por processo (BEAM) | BEAM VM | bytecode BEAM | processos isolados, supervisão OTP | `{:ok, _}`/`{:error, _}`, "let it crash" |
| [Gleam](gleam.md) | chaves, sem macros | estática HM, sem `null` | GC da BEAM ou do JS | BEAM ou JS | compila para Erlang/JS | processos OTP | `Result`; sem exceções; compilador tolerante a falhas |
| [Roc](roc.md) (experimento) | ML, indentação | estática HM, efeitos inferidos | contagem de referências, mutação in-place | a plataforma (host) dá os efeitos | AOT; compilador em reescrita | pela plataforma | `Result`, tags |
| [Zig](zig.md) | chaves, `comptime` | estática, generics por `comptime` | manual, allocator explícito | nenhum | AOT, incremental | threads; `async` removido | error unions + `try`; `ErrorBundle` |
| [Kotlin](kotlin.md) | chaves, lambdas com receptor | estática; nulidade no tipo | GC (JVM/Native) | JVM/Native/JS/WASM | JIT ou AOT | coroutines, concorrência estruturada, `suspend` | exceções; *platform types* na fronteira |
| [Java](java.md) | chaves, verboso | estática nominal; generics por erasure | GC (G1/ZGC) | JVM | bytecode + JIT; AOT (GraalVM) | virtual threads (JEP 444), `StructuredTaskScope` (JEP 505) | exceções checked/unchecked |
| [C#](csharp.md) | chaves, LINQ | estática; nulidade gradual (NRT) | GC .NET | CLR | IL + JIT; Native AOT | `async`/`Task` | exceções; NRT como avisos |
| [JavaScript](javascript.md) | chaves, ASI | dinâmica com coerção | GC | motor + event loop | JIT | event loop, microtasks, `async` | exceções; rejeições não tratadas |
| [Ruby](ruby.md) | blocos `do`/`end`, DSLs | dinâmica (duck); RBS opcional | GC | MRI/YJIT, GVL | interpretado + JIT | threads com GVL, Ractors, fibers | exceções |
| [Dart](dart.md) | chaves | estática sólida, null safety sólida | GC | VM + AOT, JS/WASM | JIT em dev, AOT em produção | isolates (sem memória compartilhada) | exceções |
| [Scala](scala.md) | chaves ou indentação (Scala 3) | estática avançada; implicits → `given` | GC (JVM) | JVM/JS/Native | JIT | bibliotecas (Futures, efeitos) | exceções, `Either`/`Try` |
| [C](c.md) | chaves, pré-processador | estática fraca | manual | libc | AOT | threads, atomics (C11) | códigos de retorno; comportamento indefinido |
| [C++](cpp.md) | chaves, templates | estática; templates → concepts | manual + RAII | mínimo | AOT lento | threads, coroutines (C++20) | exceções e `expected` |
| [V](v.md) | parecida com Go | estática | autofree prometido, GC (nv atual) | gera C | AOT rápido | `spawn`, canais | `!`/`?`, `or {}` |
| [Crystal](crystal.md) | parecida com Ruby | estática com inferência **global** | GC | nativo (LLVM) | AOT do programa inteiro, lento | fibers + canais | exceções; `Nil` em uniões |
| [Julia](julia.md) | blocos `end` | dinâmica, dispatch múltiplo | GC | JIT (LLVM) | JIT; *package images* (1.9) | tasks, threads | exceções; ambiguidade de método é erro |
| [Lua / Luau](lua.md) | blocos `end`, tabelas | dinâmica; Luau gradual | GC incremental | VM pequena e embutível | bytecode | corrotinas | `error`/`pcall`; interrupção pelo host (Luau) |
| [Haskell](haskell.md) | layout (off-side), puro | estática HM + type classes | GC; preguiça (space leaks) | RTS do GHC | AOT | green threads, STM | `Maybe`/`Either`; exceções em `IO` |
| CUE / HCL / Dhall ([sintaxe-e-configuracao](sintaxe-e-configuracao.md)) | blocos declarativos | CUE: valores como tipos; Dhall: total | — | avaliador | avaliação/normalização | — | conflito de unificação é erro (CUE) |
| Wasp ([DISCOVERIES](DISCOVERIES.md) D1) | DSL `main.wasp` → spec em TypeScript (2026) | herda TS | — | Node + React + Prisma gerados | gera código | jobs | herda JS |
| IntentLang, Human, Amana (D2) | inglês controlado / DSL | — | — | geram Node/React/Express | geram código | — | ambíguo falha explicitamente (IntentLang) |
| Hedy (D3) | graduada por níveis; 54 idiomas | dinâmica | — | Python | transpila para Python | — | mensagens por nível |
| Portugol / Égua / Potigol (D4) | imperativa em português | — | — | IDE no navegador | interpretada | — | didáticas |
| Kip (D4) | turco; casos gramaticais decidem o papel | por casos | — | — | — | — | experimental |
| Eve (D5) | relacional, literária | — | — | plataforma própria | — | reativa por padrões | encerrada em 2018 |
| Ur/Web, Opa, Links (D6) | ML, uma linguagem para as três camadas | estática forte | Ur/Web sem GC | servidor + cliente gerados | compilação do programa inteiro | — | mensagens confusas (Ur/Web) |
| Darklang (D7) | editor estruturado (removido) | estática | — | plataforma "deployless" | — | — | traces de requisições |
| Elm / Lamdera, Catala, Wing (D8) | puras/declarativas | estática | — | Lamdera: cliente + servidor; Wing: preflight/inflight | AOT | Elm: sem concorrência exposta | Elm: erros amigáveis |
| **Germanio (hoje)** | frases em português + off-side de 4 espaços; **dois front-ends** (intenção e núcleo estrito) mais o dialeto técnico anterior | tipo pelo nome, explícito vence; núcleo estrito tipado | GC do Go; `map[string]any` por linha do banco à resposta (AUDITORIA 1.3) | interpretador de `ast.App` em Go, binário único; RSS em repouso 26,9 MB contra 12,4 MB do Go direto (G90) | parse + resolve a cada partida (8–10 ms no GitLab); sem IR; `germanio build` embute os fontes | goroutine por request (`net/http`); escritor único do SQLite preso ao handler (G86); `paralelo` sem teto e `timeout` que abandona (G89, G96); trabalho remoto com lease e retry | diagnósticos em 4 partes; um erro por execução, avaliado na frase sintetizada (G69); linhas ignoradas (G67) |

## Parte 2 — Ecossistema e lição

| Language | Package Manager | Tooling | Frontend | Learning Curve | Strength | Weakness | Germanio Lesson |
|---|---|---|---|---|---|---|---|
| [Python](python.md) | pip/PyPI; `uv.lock` universal | ferramentas de terceiros (Black, Ruff, Pyright) | — (server-side) | baixa no início | legibilidade; gramática executável | migração 2→3 de uma década; empacotamento fragmentado | P8, P7, A10 |
| [Go](go.md) | Go modules, MVS, `go.sum` + checksum DB, nada roda na instalação | `gofmt`, `vet`, `gopls`, `go test` num toolchain | — (`html/template`, WASM) | baixa | compatibilidade Go 1; toolchain única | `go f()` sem dono; `nil` | A21, A33, P9, I7 |
| [Rust](rust.md) | Cargo, crates.io imutável, `Cargo.lock`, cargo-vet | rustfmt, clippy, rust-analyzer | WASM (terceiros) | alta | segurança de memória sem GC; diagnósticos | curva; compilação lenta; *cancel safety* | A1, A7, P2, E20 |
| [Swift](swift.md) | SwiftPM | swift-format, sourcekit-lsp | SwiftUI (nativo, declarativo) | média | Swift Evolution; fix-it "único e óbvio" | dois parsers; muitas "cores" de concorrência | A10, P2, E20 |
| [Nim](nim.md) | nimble | nimpretty/nph, nimsuggest | backend JS | média | sintaxe limpa | igualdade de grafias confusa; macros | A14, E2, E10 |
| [TypeScript](typescript.md) | npm | um front-end para compilador e editor | todos os frameworks JS | média | catálogo de diagnósticos; LSP | tipos não sólidos; npm | A6, A11, A17 |
| [Elixir](elixir.md) | Hex, `mix.lock` | `mix format`, doctest, ExUnit | Phoenix LiveView (servidor) | média | tolerância a falhas; deprecação com contrato | macros (`use`) escondem comportamento | P15, P7, A15, E2 |
| [Gleam](gleam.md) | Hex | um binário (compilador, fmt, LSP) | Lustre | baixa a média | simplicidade; poda antes da v1 | ecossistema jovem | A8, A13, I6 |
| [Roc](roc.md) | pacotes por URL com hash | formatter, LSP (em reescrita) | pela plataforma | média | fronteira de efeitos pela plataforma | experimental, pré-0.1 | P11, P12, E14 |
| [Zig](zig.md) | `build.zig.zon` com hash | um binário; `zig fmt --check`; `ast-check` | WASM | média a alta | nenhum fluxo escondido | pré-1.0; allocator exposto | A8, A16, P10, E13 |
| [Kotlin](kotlin.md) | Gradle/Maven | IntelliJ, K2 | Compose Multiplatform | média | null safety; concorrência estruturada | Gradle legível só com IDE | A21, P25, P27 |
| [Java](java.md) | Maven/Gradle ("nearest wins") | IDEs | — | média | décadas sem quebrar; *preview* (JEP 12) | configuração implícita por anotações | A21, P25, E22 |
| [C#](csharp.md) | NuGet | Roslyn (compilador como plataforma) | Blazor (Server/WASM) | média | migração gradual de NRT; EF migrations | muitos modos visíveis | P16, P25, E27 |
| [JavaScript](javascript.md) | npm (install scripts; worm de 2025) | fragmentado; Deno/Bun unificam | nativo do navegador | baixa no início, alta depois | roda em todo lugar | não pode remover nada | P18, E21, E30 |
| [Ruby](ruby.md) | RubyGems + Bundler (`Gemfile.lock`) | RuboCop, ruby-lsp | Rails (Hotwire) | baixa | convenção sobre configuração | mass assignment (GitHub 2012); monkey patching | A25, A30, E2 |
| [Dart](dart.md) | pub, `pubspec.lock` | `dart format` sem opções | Flutter (declarativo, hot reload) | baixa a média | recarga com estado; migração para null safety | migração presa ao ecossistema inteiro | A29, P28 |
| [Scala](scala.md) | sbt, Coursier | Metals, scalafmt | Scala.js | alta | expressividade | complexidade acumulada; duas sintaxes | P24, E27 |
| [C](c.md) | nenhum oficial | Clang (ranges, fix-its), sanitizers | — | começa fácil, difícil acertar | especificação ISO | comportamento indefinido | A26, A1 |
| [C++](cpp.md) | nenhum oficial (vcpkg, Conan) | clangd, clang-format | — | muito alta | zero overhead | acúmulo; módulos só após ~20 anos | A28, E27 |
| [V](v.md) | vpm | `v fmt` | V UI | baixa | compilação rápida | promessas à frente da implementação (revisões datadas) | A27, E31 |
| [Crystal](crystal.md) | shards | formatter; LSP difícil | — | baixa a média | Ruby com tipos | inferência global: compilação e LSP caros | A34, I7 |
| [Julia](julia.md) | Pkg (`Project.toml` + `Manifest.toml`) | Revise, LanguageServer.jl | — | média | composição por dispatch | latência de primeira execução | P21, A31 |
| [Lua / Luau](lua.md) | LuaRocks | mínimo; Luau com LSP | — | baixa | pequena, embutível, sandbox | ecossistema disperso | P18, P19, E29 |
| [Haskell](haskell.md) | Cabal **e** Stack | HLS, ormolu | — | alta | pureza; `deriving` | extensões por arquivo; preguiça | P17, E27, E28, E33 |
| CUE / HCL / Dhall | — | CUE e HCL com formatters | — | média | schema fechado; fusão comutativa | CUE pouco conhecido | A2, A4, P5 |
| Wasp | npm | LSP e extensão próprios a ~80%, trocados por TS | React gerado | baixa para quem já usa JS | especificação do app inteiro | custo da linguagem própria | A17, E25 |
| IntentLang, Human, Amana | npm / cargo | Amana: formatter, LSP, JSON diagnostics | geram React etc. | baixa (declarado) | rejeitam o ambíguo; impressão digital semântica | < 10 estrelas; geram código a manter | I8, E25, E34 |
| Hedy | — | ambiente web | ambiente web | muito baixa | níveis graduais; 54 idiomas | localizar a gramática não foi resolvido | A32, E23 |
| Portugol / Égua / Potigol | — | IDE no navegador | — | muito baixa | zero instalação | nunca viraram produção | I12 |
| Kip | — | — | — | — | a língua como estrutura, não vocabulário | experimental | E23 |
| Eve | — | editor próprio | própria | — | modelo relacional único | nada real construído a tempo | E24 |
| Ur/Web, Opa, Links | — | — | gerado do mesmo programa | alta | garantias por construção | garantias pedem entender o sistema de tipos | E26 |
| Darklang | — | editor estruturado (removido em 2023) | — | — | traces de requisições reais | editor próprio foi a parte mais cara | E24, I18 |
| Elm / Lamdera, Catala, Wing | — | Lamdera: `live`; Wing: simulador local | Elm/Lamdera | média | fio cliente-servidor derivado; revisão por especialistas | nichos | P17, I18 |
| **Germanio (hoje)** | **nenhum**; `importar` só inclui arquivos locais; Go modules só para o runtime | `ge check`, `ge explain`, `ge fmt` com conferência por reparse, `ge test`, doctest dos documentos públicos; **sem LSP**; gramática TextMate gerada; cinco leitores da sintaxe (G21) | HTML do servidor, zero JS nas páginas de intenção; contraste, labels e formulário abaixo de WCAG (G75); SPA anterior com CDN | nível 1 baixo **para quem lê português**; a camada de intenção só existe em português, e um programa em inglês passa no `check` com 0 dados (G74) | modelo semântico do app inteiro; o runtime é o único autor de SQL, HTML e rotas; `ge explain` com origem | visibilidade O(tabela) (G85); "pague só pelo que usar" não vale (G90); mapa global de 20 idiomas com colisões (G92); linhas ignoradas e fusão não idempotente (G67, G68); migração que perde dados em silêncio (G93) | [RESPOSTAS.md](RESPOSTAS.md) |

## Leitura

1. Nenhuma coluna é, sozinha, argumento. As colunas "Concurrency" e "Errors" mostram a
   convergência mais clara do ecossistema: concorrência estruturada (Go `errgroup`, Kotlin,
   Java JEP 505) e erro como valor com origem (Rust, Gleam, Zig). O Germanio ainda não
   implementa nenhuma das duas por completo ([GEP 0005](../../gep/0005-concorrencia-por-intencao.md)).
2. Na coluna "Package Manager" o Germanio está numa posição rara: não há registro, logo não há
   supply chain de `.ge`. A pesquisa recomenda não criar um antes de haver necessidade
   ([GEP 0006](../../gep/0006-pacotes.md)).
3. Na coluna "Learning Curve", a linha do Germanio depende da língua do leitor. É o ponto em
   que a promessa pública mais se afastou do código
   ([GEP 0007](../../gep/0007-idiomas-da-intencao.md)).
4. As linhas das descobertas mostram que os projetos parecidos morreram pela periferia
   (instalação, editor, mensagens de erro), não pela ideia. Na linha do Germanio, essa
   periferia também está incompleta: não há LSP nem ambiente sem instalação.
