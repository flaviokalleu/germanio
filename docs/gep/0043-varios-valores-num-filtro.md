# GEP 0043: Several values in a filter (no syntax)

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (GitLab stress test: `?topic=a,b`, the `action` filter of the events); decision
  by the maintainer
- **Level:** 1 (no new phrase; `permita filtrar por …` gains a meaning for several values)
- **Layer:** core (the generated listing and `superficie.pedir`), adapter (external names only)

## Problem

`permita filtrar projetos por tópico` (GEP 0030, part B) filters by one item of a list, and
`permita filtrar atividades por acao` by one value. People often want more than one: the
projects about *go* **and** *web*; the activity that was a closing **or** a reopening; the
orders that are *paid* or *sent*. Today the second value of a filter is ignored, so the only way
is several requests merged by hand — which breaks pagination and counts.

## Evidence

GitLab `GET /projects?topic=a,b` ("projects with all the given topics") and the `action` filter
of `/events`, whose `updated` covers two verbs of the history (`editar`, `mudar`). The same need
exists in a shop (products in two categories), a help desk (tickets new or waiting) and a CRM.

## Current state

`runtime/servidor/intencao.go` (`list`) read `q.Get(filter)`: the first value only. A list item
filter was always one item.

## Proposal

No new phrase. The filters already declared accept several values:

```text
?estado=aberta&estado=fechada     any of the values (one field has one value)
?topico=go,web                     all the items (a list may have them all)
?topico=go&topico=web              the same
```

## Semantics

- **A field with one value** (text, number, state, reference) repeated in the query keeps the
  records whose value is **any** of the values given. Asking a single value for all of them would
  always be empty; "any" is the only useful reading.
- **One item of a list** (`filtrar por topico`, with `topicos lista de texto`) given several
  items — separated by commas or repeated — keeps the records whose list has **all** of them,
  each item matched exactly as in GEP 0030 (never `golang` for `go`). Each item narrows further,
  like adding a filter. Spaces around items and repeated items are ignored.
- A list read by name (`labels por nome`) keeps its current rule (one name per query).
- As every filter, values only narrow what the person already sees; per-record visibility is
  checked after the query, exactly as before. Counts (`X-Total`) count the filtered records the
  person sees.
- `superficie.pedir` (GEP 0033): a list in the query map becomes a repeated parameter, so an
  adapter asks for "any of" without building a query string by hand
  (`superficie.pedir("GET", "historico", nulo, {action_name: ["editar", "mudar"]})`).

## Errors

None new: a value that matches nothing gives an empty list, as one value does today.

## Evaluation

No concept is added for the author of the domain. For clients it follows the common convention
of HTTP query strings (repeated parameter = several values) and of the products studied (commas
for tags).

## Impact

Runtime only: `runtime/servidor/intencao.go` (`list`, `listItemsOf`), `runtime/banco/consulta.go`
(the `contem_todos` operator: one condition per item, all required), `runtime/servidor/superficie.go`.

## Performance and security

"Any of" is one `IN (…)` with parameters; "all" is one parameterized `LIKE` per item on the stored
form of the list, the same condition as one item today. Nothing is concatenated into SQL. The
number of values is bounded by the size of the request.

## Compatibility and migration

A single value behaves exactly as before. A value with a comma given to a list item filter is now
several items; a single item containing a comma can no longer be filtered (GitLab topics, the
reference use, never contain commas).

## Trade-offs and alternatives

- **Commas for every filter:** a text value may contain commas (`Silva, João`); only list items,
  which are short tags, use them.
- **"All" for repeated scalar values:** always empty; useless.
- **A new phrase (`permita filtrar por vários tópicos`):** the filter is already declared; the
  number of values is a choice of who asks, not of the program.
- **Do nothing:** adapters would page through results by hand, with wrong totals.

## Tests

`runtime` `TestVariosValoresLugarECopias` (contracts in folders, no GitLab: a type repeated keeps
any of them, never a private record of someone else; tags separated by commas or repeated keep all
of them); `examples/gitlab-foss/e2e` `TestVariosTopicos` (`?topic=go,web`, exact items, spaces and
repetition, visibility kept) and `TestFiltroDeAcaoDosEventos` (`action=updated` through a list in
`superficie.pedir`).
