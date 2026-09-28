# Concorrência por intenção: potência sem ensinar threads, goroutines, mutexes ou event loops

Data: 2026-09-28
Status: pesquisa transversal — quase tudo aqui é **PROPOSTO**. O que existe no Germanio hoje
está marcado IMPLEMENTADO com arquivo:linha.
Escopo: confronta modelos de concorrência de linguagens maduras com o que o Germanio já tem
(`paralelo` no interpretador, trabalho remoto com lease/heartbeat, filas `jobs`/`tarefas`) e
propõe uma representação por INTENÇÃO com semântica formal. Leia junto
`docs/research/performance/RUNTIMES.md` (seção "Structured concurrency", "Backpressure",
"Limites de recursos padrão") e a `AUDITORIA.md` (riscos 3.2, 3.4).

Fontes:
- Go: `context` (https://go.dev/blog/context), `errgroup`
  (https://pkg.go.dev/golang.org/x/sync/errgroup), `semaphore`
  (https://pkg.go.dev/golang.org/x/sync/semaphore), `sync.WaitGroup.Go`
  (https://pkg.go.dev/sync#WaitGroup.Go).
- Structured concurrency: N. J. Smith, "Notes on structured concurrency, or: Go statement
  considered harmful" (2018) — https://vorpus.org/blog/notes-on-structured-concurrency-or-go-statement-considered-harmful/
  ; Trio nurseries; Wikipedia https://en.wikipedia.org/wiki/Structured_concurrency
- Java: JEP 444 Virtual Threads (https://openjdk.org/jeps/444), JEP 505 Structured Concurrency
  (https://openjdk.org/jeps/505), `StructuredTaskScope`/`Joiner` (all-or-fail, first-success).
- Kotlin: R. Elizarov, "Structured concurrency"
  (https://elizarov.medium.com/structured-concurrency-722d765aa952); `coroutineScope`
  (https://kotlinlang.org/docs/coroutines-basics.html).
- Swift: SE-0306 Actors (https://github.com/apple/swift-evolution/blob/main/proposals/0306-actors.md);
  Sendable e data-race safety no Swift 6.
- Erlang/Elixir/OTP: supervisão e "let it crash"
  (https://www.erlang.org/doc/design_principles/sup_princ.html).
- Rust async / coloração: B. Nystrom, "What Color is Your Function?"
  (https://journal.stuffwithstuff.com/2015/02/01/what-color-is-your-function/); cancelamento por
  drop de Future (cancel safety).
- Backpressure: Tokio channels (https://tokio.rs/tokio/tutorial/channels), .NET Pipelines
  (https://learn.microsoft.com/en-us/dotnet/standard/io/pipelines).

---

## O problema

Toda a potência de concorrência das linguagens maduras vem embrulhada em conceitos que o
público-alvo do Germanio (quem nunca programou) não deve aprender: thread, goroutine, mutex,
canal, `Future`, event loop, `async`/`await`, ator, `Sendable`. A pergunta do estudo:
**como oferecer concorrência forte expondo só a intenção?** ("processe pedidos em paralelo",
"até 10 ao mesmo tempo", "se demorar mais de 30 segundos, desista", "tente de novo 3 vezes").

O `INTENCAO.md` (seção Eficiência) já é normativo sobre isso: "O autor declara intenção; o
runtime decide workers, limites, cancelamento, timeouts, propagação de erro e limpeza. Nenhuma
construção cria trabalho ilimitado." E: "goroutine, channel, mutex, ponteiro, buffer pool,
connection pool e allocator são mecanismos do core; não aparecem no nível padrão." A frase de
carga "só vira sintaxe depois que a semântica, os limites padrão e os testes estiverem
definidos". Este documento é o esboço dessa semântica.

---

## O que cada modelo resolve — e o que custa em conceitos

### Go: goroutines + channels + `context` + `errgroup`
- **Acerto:** goroutines são baratas; `context` propaga cancelamento e prazo por toda a árvore
  de chamada ("quando um request é cancelado, todas as goroutines saem rápido" — go.dev/blog/context).
  `errgroup.WithContext` cancela no primeiro erro e `Wait` devolve o primeiro erro;
  `errgroup.SetLimit` põe teto; `semaphore.Weighted.Acquire(ctx, n)` respeita cancelamento.
- **Custo cognitivo:** `go f()` sem dono é "goto concorrente" (Smith 2018): quebra pilha de
  chamada para erros e limpeza. Sem disciplina, vazam goroutines e há data race. É exatamente o
  erro do `paralelo` atual (ver abaixo).
- **Para o Germanio:** o **substrato de implementação** ideal — `context`+`errgroup`+`semaphore`
  dão limite, cancelamento e propagação de erro sem inventar nada. Mas nunca na superfície.

### Erlang/Elixir/OTP: atores isolados + árvores de supervisão + "let it crash"
- **Acerto:** processos não compartilham memória (sem mutex); falha isola-se ao processo; o
  supervisor reinicia segundo uma estratégia declarada (one-for-one, rest-for-one…). A
  resiliência é **declarativa** (a árvore de supervisão), não imperativa.
- **Custo:** o modelo de mailbox e as estratégias de supervisão são muitos conceitos para um
  leigo, mas a **ideia** — "descreva o que reiniciar quando algo falha" — é declarativa e casa
  com o Germanio.
- **Para o Germanio:** ADAPTAR a supervisão declarativa para jobs/tarefas: o autor diz "tente 3
  vezes, depois desista"; o runtime é o supervisor. Já existe embrião em trabalho remoto (retry,
  lease, cancelamento).

### Java: virtual threads (JEP 444) + structured concurrency (JEP 505)
- **Acerto:** virtual threads são leves (milhões); pode-se dedicar uma por subtarefa e escrever
  código bloqueante simples. `StructuredTaskScope` amarra as subtarefas a um escopo com
  **política de junção** explícita: `open()` = "todas ou falha" (all-or-fail); há
  "primeira que der certo" (first-success). O escopo fecha e junta antes de sair (o `join` roda
  na thread dona).
- **Custo:** a API de scope/joiner ainda é técnica, mas o **vocabulário** ("espere todas", "a
  primeira", "falhe se qualquer uma falhar") é exatamente o que uma frase de intenção precisa.
- **Para o Germanio:** ADOTAR o vocabulário de política de junção como semântica das frases.

### Kotlin: coroutines + `coroutineScope`
- **Acerto:** concorrência estruturada imposta pelo escopo — "o pai sempre espera os filhos; o
  pai nunca perde uma coroutine" (Elizarov). Cancelar o escopo cancela a árvore.
- **Custo:** `suspend` colore funções (ver Rust); mas a regra do escopo é ensinável.
- **Para o Germanio:** a **regra do escopo** (nada sobrevive ao bloco que o criou) deve ser
  invariante do runtime, não algo que o autor gerencia.

### Swift: actors + `Sendable` (data-race safety no Swift 6)
- **Acerto:** um ator garante acesso serial ao seu estado sem locks; `Sendable` é verificado em
  tempo de compilação — o que atravessa domínios de concorrência é seguro por prova do compilador.
- **Custo:** "What Color is Your Function?" — `async` propaga para cima; Swift adiciona mais
  cores (isolado ao ator, `nonisolated`, main actor, `Sendable`/não). Muitos contextos: alto
  custo cognitivo. Explicitamente o que o Germanio deve **evitar** expor.
- **Para o Germanio:** ADOTAR a **ideia** (dados que cruzam limites de concorrência precisam ser
  seguros) como garantia do runtime; EVITAR a coloração na superfície.

### Rust async: potência com coloração e cancelamento por drop
- **Acerto:** zero-cost, sem data race por construção (o borrow checker).
- **Custo:** coloração `async`/`await` (Nystrom); cancel safety é sutil — soltar um `Future`
  cancela e pode perder estado. Conceitos demais para leigos.
- **Para o Germanio:** EVITAR na superfície; o cancelamento deve ser automático e seguro, sem o
  autor pensar em "o que perco se cancelar".

**Conclusão transversal:** todas as linguagens convergiram para **concorrência estruturada** —
trabalho concorrente tem um escopo que o contém, espera, cancela e propaga erro. É o modelo
certo para o Germanio implementar por baixo. A superfície deve ser só a **intenção** e a
**política de junção**, sem cor de função nem primitivas.

---

## O que existe no Germanio hoje (IMPLEMENTADO) e onde erra

- **`paralelo` e amigos** (`runtime/interpreter/interpreter.go:1139` `paralelo`, `:1221`
  `consultar_paralelo`, `:1266` `chamar_async`, `:1182` `timeout`): uma goroutine por item,
  **sem teto**; escopos compartilham um `map` **sem mutex** (data race — `interpreter.go:41-48`);
  o pânico vira a string `"erro: ..."` e **não cancela as irmãs**; `timeout` **abandona** a
  goroutine, que continua rodando. Viola as três propriedades da concorrência estruturada
  (limite, cancelamento, propagação de erro). É o G89 (AUDITORIA 3.4).
- **Fila `jobs`** (`runtime/servidor/servidor.go:75`, `jobs/jobs.go`): 4 workers, 256 posições,
  **sempre criada**; quando cheia, **descarta em silêncio** (`jobs/jobs.go:56-66`). Backpressure
  errado (deve conter o produtor ou rejeitar na borda, nunca descartar).
- **Fila `tarefas`** (`runtime/servidor/tarefas.go`): polling do banco 4x/s, tabela sem índice,
  sem limpeza (AUDITORIA 3.3).
- **Trabalho remoto** (`runtime/servidor/trabalho_remoto.go`, G54): **o embrião certo** — claim
  atômico, reserva com lease/renovação/expiração, heartbeat, cancelamento visível, retry, log
  incremental por offset, token temporário com escopo mínimo. É concorrência estruturada
  distribuída, genérica (`X executam Y`), testada sem GitLab. Falta trazer a mesma disciplina
  para a concorrência **local** (`paralelo`).
- **Escopo por requisição** (G22, DONE): handler concorrente com escopo isolado filho do global
  — base necessária para paralelismo seguro dentro de um request.
- **Escritor único do banco** (`banco.go`, AUDITORIA 3.2): trava global presa o handler inteiro,
  sem fila, sem backpressure, sem `context` chegando ao banco. p99 de segundos com 16 clientes.

---

## PROPOSTA: concorrência por intenção com semântica formal

Nenhuma frase abaixo é sintaxe aceita hoje (o `INTENCAO.md` exige semântica+limites+testes
antes). São candidatas a design.

### Frases de intenção (candidatas)
- "processe pedidos em paralelo" → paralelismo com teto padrão.
- "até 10 ao mesmo tempo" → limite explícito de concorrência (semáforo).
- "se demorar mais de 30 segundos, desista" → prazo (deadline) com cancelamento.
- "tente de novo 3 vezes" → retry com política (backoff) declarada.
- "assim que a primeira responder" → política de junção first-success.
- "em segundo plano" → fila durável (jobs/tarefas), não bloqueia a resposta.

### Semântica que cada frase precisa fixar (o contrato)
1. **Limites padrão.** Sem "até N", o teto é `GOMAXPROCS` para CPU e um semáforo por recurso
   externo para I/O (RUNTIMES.md, tabela de limites). Nunca uma goroutine por item.
2. **Backpressure.** Toda fila tem teto; cheia → o produtor espera (dentro do processo) ou é
   rejeitado cedo (na borda HTTP). **Nunca descartar em silêncio** (corrige `jobs`).
3. **Cancelamento.** Um `context` flui do request até o trabalho e até o banco; desistência por
   prazo, erro ou desconexão do cliente cancela toda a árvore e **para** as goroutines (corrige
   `timeout` que abandona).
4. **Propagação de erro.** Política de junção explícita, no vocabulário do JEP 505: "espere
   todas" (falha se qualquer uma falhar, cancela as irmãs), "a primeira que der certo",
   "espere todas e colete resultados+erros". O erro é um valor estruturado com origem
   (arquivo:linha), não a string `"erro: ..."`.
5. **Ordem dos resultados.** Definir e documentar: resultados na ordem de entrada (determinismo)
   mesmo com execução fora de ordem — importante para `ge explain` e reprodutibilidade.
6. **Transações.** Trabalho paralelo que escreve no banco respeita o escritor único e a
   transação por operação; paralelismo é para I/O/CPU independentes, não para burlar a
   serialização de escrita. Documentar o que é seguro paralelizar.
7. **Segurança de dados compartilhados.** Como o Swift/`Sendable`: o runtime garante que o que
   as tarefas compartilham é seguro (cópia ou isolamento), sem expor a regra ao autor. Corrige o
   data race do `paralelo`.

### Implementação proposta (core, invisível)
- Substrato: `context` + `errgroup.WithContext` + `errgroup.SetLimit`/`semaphore.Weighted` —
  já apontado em RUNTIMES.md. `sync.WaitGroup.Go` (Go 1.25) reduz boilerplate.
- Supervisão de jobs/tarefas ao estilo OTP: política de retry/desistência declarada; o runtime
  é o supervisor. Reusar o mecanismo de `trabalho_remoto.go` (lease/heartbeat/retry) para o
  trabalho local e em segundo plano — uma só capability de "trabalho", local ou remoto.
- `ge explain` mostra os limites em vigor (teto de concorrência, timeout, política de retry, se
  o trabalho é em segundo plano), como manda o `INTENCAO.md`.

---

## Para o Germanio

- **ADOTAR** concorrência estruturada como invariante do runtime (Smith 2018; Kotlin
  `coroutineScope`; JEP 505). Problema resolvido: G89 / AUDITORIA 3.4 (`paralelo` sem teto, sem
  cancelamento, data race, erro como string). Arquivo: `runtime/interpreter/interpreter.go`.
  Remove da cabeça do programador: limite, cancelamento, coleta de erros, limpeza.
- **ADOTAR** o vocabulário de política de junção do JEP 505 como semântica das frases de
  intenção ("espere todas", "a primeira", "colete tudo"). Arquivo: `docs/INTENCAO.md`
  (pendência de design), depois lexer/parser/resolver. Remove: pensar em canais e `select`.
- **ADOTAR** backpressure real (RUNTIMES.md; Tokio/Pipelines): fila com teto que contém o
  produtor ou rejeita cedo, **nunca** descarta. Problema: `jobs/jobs.go:56-66` descarta em
  silêncio. Arquivo: `runtime/jobs`, `runtime/servidor/servidor.go`.
- **ADOTAR** propagação de `context` do request ao banco, com deadline e cancelamento por
  desconexão. Problema: AUDITORIA 3.2 (nenhum `context` chega ao banco). Arquivo:
  `runtime/banco`, `runtime/servidor/transacao.go`.
- **ADAPTAR** a supervisão declarativa do OTP para retry/desistência de jobs; reusar
  `trabalho_remoto.go` como capability única de "trabalho" (local, em segundo plano, remoto).
  Arquivo: `runtime/servidor/trabalho_remoto.go`, `runtime/servidor/tarefas.go`,
  `runtime/servidor/jobs`.
- **EVITAR** expor coloração de função (Rust/Swift `async`, `Sendable`, `suspend`) e primitivas
  (goroutine, canal, mutex, ator) no nível padrão — é o alto custo cognitivo que o `INTENCAO.md`
  proíbe. A garantia de segurança de dados fica no runtime, provada por isolamento/cópia.
- **INVESTIGAR** determinismo da ordem dos resultados e quais operações de banco podem ser
  paralelizadas com segurança sob o escritor único (interação com G85/G86). Marcar como pendência
  de design em `docs/INTENCAO.md` até haver semântica e testes (suíte STRESS).
