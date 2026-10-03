# GEP 0033: An adapter asks the application itself

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (GitLab stress test, inventory IS-09, NT-02, NT-03, AD-01); decision by the
  maintainer
- **Level:** 4/5 (only adapters, `integracoes/`; never the domain)
- **Layer:** adapter (two functions), core (re-entering the generated integration surface as the
  same person)

## Problem

An adapter translates an external protocol into Germanio. Often one external call is simply a
different call of the application's own integration surface: another path
(`POST …/issues/7/time_estimate` is an edit of the issue), other names (`to_project_id`), a lookup
first (an issue named by project and number, then its internal reference), or a number the
protocol wants (how many users, issues, projects). Today an adapter can only answer with the
capabilities made for it (`trabalho_remoto.*`) or read and write the database directly with
`modelo.atualizar(...)`. The second way skips every rule of the product — who may edit, what is
visible, read-only records, history, pending items, events — so an adapter that used it would
become a hole in the application. The norm says an adapter "não pode substituir autorização ou
invariantes".

## Evidence

GitLab inventory IS-09 (time tracking, issue links, moving an issue), NT-02
(`/projects/:id/events`, GitLab's action names), NT-03 (`/notification_settings`), AD-01
(`/application/statistics`). Every one of them is a different shape of something the generated
surface already does with all its guarantees.

## Current state

`integracoes/*.ge` may declare `rotas` and `logica`. The only safe capability adapters had was
`trabalho_remoto.*` (remote executors).

## Alternatives studied

- **Direct database access from the adapter** (what exists): rejected; it bypasses the product.
- **A new phrase per protocol need** in the domain (time estimates, links by number…): puts the
  shape of one external API into the language.
- **Renaming every generated path in the vocabulary**: the vocabulary renames names, not
  shapes; it cannot express "look this up, then edit that".
- **Reverse proxies and API gateways** (Kong, Envoy, Rails `ActionDispatch` re-dispatch): they
  translate a request into another request of the same service, keeping its credentials. That is
  the model here, inside the process.

## Proposal

Two functions, only meaningful inside a `rota` of an adapter:

```text
superficie.pedir(metodo, caminho [, corpo [, consulta]])  → {estado, corpo, cabecalhos}
superficie.contar(dado [, estado])                         → número
```

```text
rotas
    rota POST "/externo/estantes/:id/livros/:n/renomear"
        definir r = superficie.pedir("PUT", ["shelves", requisicao.parametros.id, "books", requisicao.parametros.n], {title: requisicao.corpo.novo})
        responder(r.estado, r.corpo)
```

## Semantics

- `pedir` runs **one operation of the generated integration surface** (the routes Germanio
  derives under `integração em`) **as the person of the current request**: the same credentials
  (token, session), the same token scopes, rules, visibility, read-only records, transaction,
  history, pending items and events as if that person had called it. A browser session's CSRF
  proof is the one the outer route already checked.
- `caminho` is relative to the integration prefix: a text (`"projects/1/issues"`, used as
  written, never leaving the prefix: no `..`, `?` or `#`) or a list of pieces, each escaped (an
  address with `/` is one piece). `consulta` is a map of query parameters; `corpo` is sent as
  JSON.
- The answer is `estado` (the status), `corpo` (decoded JSON, or text) and `cabecalhos` (the
  response headers, as named in the response, e.g. `X-Total`). Nothing is thrown on a refusal:
  the adapter decides what its protocol says.
- `pedir` reaches only what Germanio generates: a request made by `pedir` that lands on a
  declared `rota` is answered 508 with an explanation (an adapter calling itself would loop), and
  a request made by `pedir` cannot call `pedir`.
- `pedir` is refused inside a change (`antes de`, `quando`): each call is a whole change of its
  own; it never nests transactions.
- `contar` counts like the indicators of a page (GEP 0012): only the records of `dado` (plural or
  singular) that the person may see, optionally in one of its states; one `COUNT` for an
  administrator. Someone who may not see the data at all gets 0.
- Neither function grants anything: an adapter using them can do exactly what the person could
  do through the integration surface.

## Errors

`superficie.pedir` outside a route, inside a change, with an unknown method, an empty path or a
path leaving the integration; `superficie.contar` of an unknown data or a word that is not one of
its states (the error lists the states). Each says what to write instead.

## Evaluation

Adapter level only; the domain never sees it. It removes the reason an adapter would touch the
database, and keeps the adapter what the norm says it is: a translator.

## Impact

Runtime only (`runtime/servidor/superficie.go`, the guard in `rotas.go`). No parser change: the
functions are a capability module, like `trabalho_remoto`.

## Performance and security

One in-process call per `pedir` (no network); the answer is kept in memory up to 16 MB. Security
is the point: the person, the scopes and every rule of the generated surface apply, and a
refused change leaves nothing behind (each `pedir` is its own transaction). An adapter that
chains several `pedir` calls gets several changes, not one: a protocol call that must be atomic
needs a capability of its own (as moving a record, GEP 0034).

## Compatibility and migration

Additive.

## Trade-offs and alternatives

Several `pedir` calls are not one transaction (see above). Doing nothing leaves adapters with
direct database access as the only way to reshape a call — the unsafe one.

## Tests

`TestAdaptadorPedeAAplicacao` (runtime, a library with no Git or CI: a write as the person — the
author may, others get 403/404 and nothing changes; reads and counts see only what the person
sees; a path leaving the integration and an unknown state are refused; a declared route is never
reached). GitLab end-to-end: `TestControleDeTempo`, `TestLigacoesEntreIssues`,
`TestMoverIssue`, `TestEventosDoProjeto`, `TestEstatisticasDaAplicacao`,
`TestPreferenciaDeAvisos`.
