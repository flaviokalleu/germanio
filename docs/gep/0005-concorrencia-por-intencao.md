# GEP 0005: Concurrency by intent

- **Status:** Rascunho
- **Author:** agent (research consolidation); decision by the maintainer
- **Discussion:** none yet
- **Gaps:** G89, G96 (primary); G86, G87 (related); also the `jobs` queue that drops work
- **Level:** 1 and 2 (hook modifiers), 3 (`para cada …, em paralelo`); the runtime contract of
  the level-4 built-ins changes
- **Layer:** core (mechanism); domain (the words)

## Problem

A person wants to say "send the e-mails in the background", "try again three times", "give up
after ten minutes", "at most eight at a time", "do these in parallel". Today the only way is
level-3/4 built-ins whose behaviour is unsafe:

- `paralelo([...])` starts one goroutine per item with no limit; a panic becomes the string
  `"erro: …"` and does not cancel the other items;
- `timeout(f, ms)` returns `nil` when the deadline passes and **leaves the goroutine running**,
  possibly outside the request's transaction;
- `consultar_paralelo` and `chamar_async` start one goroutine per item with no limit;
- the in-memory `jobs` queue (4 workers, 256 slots) refuses work when full, and the three
  callers ignore the refusal, so notifications disappear without a trace.

`docs/INTENCAO.md` › Eficiência is normative on the goal ("the author declares intent; the
runtime decides workers, limits, cancellation, timeouts, error propagation and cleanup. No
construction creates unbounded work") and says that load and concurrency intent "only becomes
syntax once the semantics, the default limits and the tests are defined". This GEP is that
definition.

## Evidence

- `runtime/interpreter/interpreter.go:1128-1280` (`paralelo`, `esperar`, `timeout`,
  `consultar_paralelo`, `chamar_async`), read at `0c12051`.
- `runtime/jobs/jobs.go:56-67` (`Submit` returns `false` when full);
  `runtime/servidor/servidor.go:976, 989, 1076` (return value ignored).
- `runtime/servidor/tarefas.go`: a durable task table with retries (backoff 1 s, 5 s, 30 s,
  2 min, 10 min, then "morta"), exposed only as `tarefas.enfileirar("funcao", dados)` (level 4);
  polled 4 times per second without an index and never cleaned (G87: 51% of a core idle with
  one million finished tasks, `docs/research/performance/AUDITORIA.md` §2.6).
- The single SQLite writer holds the lock for the whole handler, with no queue and no
  `context` reaching the database (G86: p99 of 2.8 s with 16 clients, AUDITORIA §2.4).
- `runtime/servidor/trabalho_remoto.go` already has the right discipline for remote work
  (atomic claim, lease, heartbeat, retry, visible cancellation), tested without GitLab
  (`runtime/trabalho_remoto_test.go`).
- No `.ge` file in the repository calls `paralelo`, `timeout`, `chamar_async` or
  `consultar_paralelo` (searched at `0c12051`).
- Research: `docs/research/languages/CONCURRENCY.md`; `docs/research/performance/ARQUITETURA.md`
  §7; lessons A21, A22, P15, P19, E20, E32 in `docs/research/languages/GERMANIO_LESSONS.md`.

## Current state

```text
quando faturar pedido
    resultados = paralelo(["enviar_nota", "avisar_cliente"])
    r = timeout("consultar_banco_externo", 5000)
```

What the runtime does: two unbounded goroutines, errors as strings, and a timeout that
abandons its goroutine. Background work exists only through `tarefas.enfileirar` (level 4),
which a level-1 author never sees.

## Alternatives studied

All from `docs/research/languages/CONCURRENCY.md`, with sources there.

| Model | What it solves | What it costs in concepts | Taken |
|---|---|---|---|
| Go goroutines + `context` + `errgroup` + `semaphore` | limit, cancellation, first error | `go f()` without an owner; channels | **as the implementation substrate only** |
| Structured concurrency (Smith 2018, Trio, Kotlin `coroutineScope`) | nothing outlives the block that started it | a scope object | **as the runtime invariant**, with no visible scope |
| Java virtual threads + `StructuredTaskScope` (JEP 444, JEP 505) | join policies: all-or-fail, first success | a scope API | **the vocabulary of join policies** |
| Erlang/Elixir OTP supervision | declarative restart policy | mailboxes, strategies | **the idea**: the author says "try again N times"; the runtime is the supervisor |
| Swift actors + `Sendable`; Rust `async` | data-race freedom proven by the compiler | function colouring, isolation domains | **the guarantee, not the surface**: the runtime isolates; nothing is coloured |
| Luau interrupts | the host bounds execution | none for the author | **a per-request budget tied to the `context`** |
| Tokio channels, .NET Pipelines | bounded queues and backpressure | channel capacities | **bounded queues that hold or refuse, never drop** |

## Proposal

Three parts, in order of delivery. Part 1 has no syntax.

### Part 1 — the runtime contract (no syntax)

Every concurrent operation the runtime starts, including the existing level-4 built-ins, runs
under one structured scope bound to the request's or the task's `context`:

- a default limit (below), never one goroutine per item;
- a deadline that **cancels** the work, never abandons it;
- the first error cancels the siblings and reaches the caller as a structured error with
  file and line, never as a string;
- the scope does not end, and the transaction does not commit, until every child has ended;
- bounded queues: a full queue makes an internal producer wait (with its deadline) and makes
  an HTTP origin receive "try later" (503 with `Retry-After`); nothing is dropped.

### Part 2 — hook modifiers (levels 1 and 2)

The modifiers go on the `quando` line, after a comma, like field modifiers
(`email obrigatório, único e privado`). They are a closed table.

Before (level 4, today):

```text
pedidos
    pode
        faturar
    quando faturar
        tarefas.enfileirar("enviar_nota", pedido)
```

After (proposed; does not compile today):

```text
pedidos
    pode
        faturar
    quando faturar, em segundo plano
        enviar nota fiscal do pedido
    quando exportar, em segundo plano, até 2 ao mesmo tempo, termina em até 10 minutos e tenta de novo 3 vezes
        ...
```

Equivalent flat phrases:

```text
quando faturar pedido, em segundo plano
    enviar nota fiscal do pedido
quando exportar pedido, em segundo plano, até 2 ao mesmo tempo, termina em até 10 minutos e tenta de novo 3 vezes
    ...
```

| Modifier | Allowed on | Meaning |
|---|---|---|
| `em segundo plano` | `quando` | the hook body becomes a durable task, enqueued in the same transaction as the change; the person gets the answer as soon as the change commits |
| `até N ao mesmo tempo` | `quando` | at most N executions of this hook at once, across the application |
| `termina em até N segundos/minutos/horas` | `quando` | deadline of one execution; reaching it cancels the execution |
| `tenta de novo N vezes` | `quando … em segundo plano` | N retries after a failure, with the task queue's backoff |

(`enviar nota fiscal do pedido` above is illustrative of a level-3 body; it is not proposed
syntax.)

### Part 3 — bounded iteration (level 3)

Before:

```text
resultados = paralelo(["enviar_nota", "avisar_cliente"])
```

After (proposed):

```text
para cada pedido em pedidos, em paralelo
    ...
para cada arquivo em arquivos, em paralelo, até 4 ao mesmo tempo
    ...
```

`paralelo`, `timeout`, `consultar_paralelo` and `chamar_async` remain at level 4 with the Part
1 contract, and `ge check` suggests the level-3 form.

Out of scope, deliberately: server-wide load intent ("aceite muitas conexões"); its defaults
come from the STRESS suite in `bench/` first (ARQUITETURA §7.2).

## Semantics

**Default limits** (proposed; each must be confirmed by measurement before acceptance):

| What | Default | Source of the number |
|---|---|---|
| `para cada …, em paralelo` without `até` | number of CPUs for CPU work; 16 when the body does I/O (HTTP, database, file) | ARQUITETURA §7.2 |
| background workers | number of CPUs, at least 2 | idem |
| background queue | durable (the task table), indexed, finished tasks removed after 7 days | G87 |
| in-memory queue (`jobs`) | removed; its callers move to the durable queue | A22 |
| deadline of a synchronous hook | the request's deadline | the server |
| deadline of a background execution | 1 hour | ARQUITETURA §7.2 |
| retries of a background hook | the task queue's current policy (1 s, 5 s, 30 s, 2 min, 10 min, then dead) unless `tenta de novo N vezes` | `tarefas.go:25` |
| shutdown | wait up to 10 s, then cancel | — |

**Order.** Results of `para cada …, em paralelo` are delivered in input order, whatever the
execution order. `ge explain` shows no execution order because none is promised.

**Errors.** In `para cada …, em paralelo`, the first error cancels the remaining items and
is raised at the `para cada` line with the item's position. A background execution that
exhausts its retries is marked dead with its error, visible to administrators; it never
disappears.

**Transactions.** A synchronous hook stays inside the change's transaction; parallel items
inside it that write to the database are serialized by the single writer (parallelism is for
independent I/O and CPU, not for bypassing write serialization). A background hook runs
after the commit, in its own transaction per execution; if the change rolls back, the task
was never enqueued (`INTENCAO.md` › Garantias automáticas: "nem eventos na fila").

**Shared state.** Each iteration of `em paralelo` has its own scope. Assigning to a variable
declared outside the block is an error found by `ge check`. The runtime copies values passed
to background tasks (they are serialized), so nothing is shared by reference.

**Inspection.** `ge explain pedido` shows, for each hook: synchronous or background, the
limit, the deadline, the retries, and where each value came from (default, the `.ge` line,
or an environment variable).

**Conflicts.** The same hook declared twice with different values for one modifier is an
error showing both origins (`INTENCAO.md` › Fusão e conflitos).

## Errors

All in the four-part format (what, where, why, how to fix):

| Situation | Message (what / how to fix) | Automatic fix |
|---|---|---|
| `antes de faturar, em segundo plano` | "antes de" must finish before the change is saved; use `quando faturar, em segundo plano` | no (changes meaning) |
| `tenta de novo 3 vezes` without `em segundo plano` | a retry would hold the person's request; add `em segundo plano` | no |
| `até 0 ao mesmo tempo`, negative or non-numeric N | N must be a whole number from 1 | no |
| `termina em até 3 semanas` | the maximum is 24 horas; split the work | no |
| unknown modifier after the comma | lists the four modifiers, with the nearest by edit distance | only a spelling fix, never a new modifier |
| assignment to an outer variable inside `em paralelo` | each item runs on its own; collect results with the value of the block | no |
| `timeout(...)` or `paralelo(...)` at level 3 | suggests `termina em até …` or `para cada …, em paralelo` | warning only |

## Evaluation

Against `INTENCAO.md` › *Como avaliar uma sintaxe*:

1. **Technical concepts required:** none new at levels 1-2. "Em segundo plano", "ao mesmo
   tempo", "tente de novo" and "termina em até" are everyday phrases; goroutine, channel,
   mutex, queue, backoff and context do not appear (E20).
2. **Cognitive load:** four modifiers in a closed table, on the line they affect.
3. **Repetition:** none; the modifiers are written once per hook.
4. **Hierarchy:** unchanged; modifiers follow the field-modifier pattern (after a comma).
5. **Clarity:** each modifier has one meaning. Risk: a long `quando` line with four
   modifiers; the formatter keeps it on one line, as the norm requires.
6. **Determinism:** defaults are fixed and shown by `ge explain`; result order is fixed.
7. **Cost for the machine:** bounded by construction (see below).

## Impact

- **Lexer:** none (the words exist).
- **Parser:** `compiler/parser/hierarquia.go` and `intencao.go`: modifiers after the comma on
  `quando`/`antes de` lines; `, em paralelo` and `, até N ao mesmo tempo` on `para cada`.
- **AST:** `ast.Hook` gains `Background bool`, `Limit int`, `Deadline time.Duration`,
  `Retries int` with origins; the level-3 `for` node gains `Parallel` and `Limit`.
- **Resolver:** checks the table of the Errors section; fusion rule for conflicting values.
- **Runtime:** `runtime/interpreter` (a structured scope with `errgroup.WithContext` and
  `SetLimit`; `context` into every built-in and into the database);
  `runtime/servidor/tarefas.go` (index on `estado, proxima_em`, cleanup, notification instead
  of polling where possible); removal of `runtime/jobs` in favour of the durable queue;
  `runtime/servidor/transacao.go` (`context` until the database).
- **Tooling:** `ge explain` (limits and origins), `ge check` (errors above), `ge fmt`
  (canonical modifier order: `em segundo plano`, `até`, `termina em até`, `tenta de novo`),
  the VS Code grammar generator.

## Performance and security

- Bounded work everywhere: no construction can create unbounded goroutines or queue entries.
- The deadline releases resources (connections, the write lock) instead of leaking them.
- A full queue answers 503 with `Retry-After` at the HTTP edge instead of accumulating memory.
- Indexing and cleaning the task table removes the idle cost measured in G87.
- Security: a cancelled task stops using its credentials; background tasks run with the
  identity recorded at enqueue time and are re-authorized on execution (a person whose role
  was removed does not keep acting through a queued task). Race conditions from the shared
  `map` in `paralelo` disappear, because each item has its own scope.
- Cost to measure before acceptance: overhead of the scope per request (expected: one
  `context` and one `errgroup`), throughput of the durable queue under the single writer.

## Compatibility and migration

- Part 1 changes the observable behaviour of `timeout` (it cancels and raises an error instead
  of returning `nil` while the work continues) and of `paralelo` (a limit; errors raised, not
  returned as strings). No `.ge` in the repository uses these built-ins; external programs
  are not known. Proposed period: one release in which the old behaviour remains and
  `ge check` warns, then the new contract.
- The `jobs` queue is internal (WhatsApp and e-mail notifiers); moving it to the durable queue
  has no syntax impact.
- Parts 2 and 3 only add forms; existing programs keep their meaning.
- `ge fmt` does not rewrite `paralelo(...)` into `para cada …, em paralelo` (the meaning is
  not identical: the new form cancels on first error). `ge check` suggests it.

## Trade-offs and alternatives

- **Do nothing.** Keeps G89 and G96: unbounded goroutines, abandoned work, silent loss of
  notifications. Rejected: it violates the normative efficiency rules already in force.
- **Part 1 only, no syntax.** Fixes the defects and leaves level 1 without a way to say
  "in the background". Acceptable as a first step; it is the recommended delivery order.
- **A `trabalho` section per data block** (listing actions with their limits). Rejected for
  now: a new section for four modifiers adds a concept; the modifiers belong where the effect
  is written.
- **Expose `async`/`await` or a scope object.** Rejected (E20): colouring and scopes are the
  concepts the norm forbids at level 1.
- **Automatic retries for every background hook.** Kept as the default because it is the
  current task-queue behaviour, but it is a real risk for effects that are not idempotent
  (charging a card twice). Open question for the maintainer: default to no retries and require
  `tenta de novo N vezes`?
- **Unified local and remote work** (one capability for `em segundo plano` and `X executam Y`,
  lesson P15). Desirable; deferred until Part 1 is measured.

## Tests

- `paralelo` with 10 000 items never has more than the limit running (counter in the test).
- `timeout` that expires: the child observes cancellation within 100 ms and does not write
  after the deadline; the transaction rolls back.
- `para cada …, em paralelo` where item 3 fails: items not started do not start, running items
  are cancelled, the error names item 3 and the line; results of a successful run come in
  input order.
- `-race` on the interpreter tests passes with parallel blocks (today `ci.yml` runs `-race` on
  `./compiler/...`, `./runtime/germanio` and `./tooling/...`, not on `runtime/interpreter`).
- `quando faturar, em segundo plano`: the change commits and the response returns before the
  body runs; a rolled-back change leaves no task; a failing body is retried with the declared
  count and then marked dead with its error.
- Full queue at the HTTP edge: 503 with `Retry-After`; nothing is dropped (the number of
  accepted equals the number executed).
- `ge explain` shows each limit with its origin; `ge check` reports every row of the Errors
  table.
- Equivalence test: each hierarchical form produces the same facts as its flat phrase.
- Generalization test in a second domain (e.g. file conversions), without the GitLab example.
