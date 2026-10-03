# GEP 0048: A link is unique per pair (`única por par de issues`)

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (GitLab stress test, IS-09 issue links); decision by the maintainer
- **Level:** 1 (a line under `regras`)
- **Layer:** domain (one rule), core (validation, a database index, the answers)

## Problem

Many products link two records of the same data: related issues, friendships between people,
connected accounts, equivalent products. Two mistakes are always possible: linking a record to
itself, and linking the same two records twice — also in the other order, which for a symmetric
relation is the same link. Checked by hand ("count the links between A and B, then insert") it
races, and it lives in every adapter or hook that creates links. The GitLab domain allowed both.

## Evidence

GitLab issue links (`relates_to`): a link appears on both issues, and linking already linked
issues answers 409 `Issue(s) already assigned`. Social networks (one friendship per pair), CRMs
(one relationship between two contacts), catalogs (A equivalent to B is B equivalent to A).

## Current state

`unico` on a field, and `UniqueTogether` (numbering per parent) in the core. Nothing forbids a
reference pointing at the record's own owner, nor a pair in the other order.

## Alternatives studied

- **Two rules** (`relacionada não pode ser a própria issue` and `relacionada é única por issue`):
  two lines for one idea, and the second is ordered.
- **An ordered pair** (`única por issue e relacionada`): right for directed relations (A blocks B),
  wrong for `relates_to`, where B–A after A–B is the same link twice. Left for a later GEP when a
  directed relation needs it.
- **A unique index on the two columns** (what frameworks generate): allows (B, A) after (A, B) and
  says nothing about self-links.
- **Checking in the adapter** (what a protocol translator would do): bypassed by any other route,
  and racy.
- **Do nothing.**

## Proposal

```text
ligacoes
    pertence a
        issue como relacionada
    regras
        única por par de issues
```

Flat: `ligacao é única por par de issues` (also `cada ligacao é…`; `único` for a masculine name).
The record must point at the data through exactly two references (here `issue_id`, from
`issues › tem › ligacoes`, and `relacionada_id`).

## Semantics

- **A pair is two different records, in any order** — the ordinary meaning of "par". The decision
  is symmetric because the link is: it is shown from both records and leaves with either one.
- **Never to itself:** both references holding the same record is a validation error of the
  second reference (400, `{"message": {"relacionada": ["não pode ligar um registro a ele mesmo"]}}`),
  on create and on edit.
- **Once per pair:** a pair already linked, in either order, answers **409** (conflict with an
  existing record) with `{"message": {"relacionada": ["esse par já existe (em qualquer ordem)"]}}`,
  on create and on edit. 409 is also what the database answers when it catches the repetition, so
  the check before writing and the index give the same answer.
- **The database keeps it too:** a unique index on the smaller and the larger of the two
  references (SQLite `MIN/MAX`, PostgreSQL `LEAST/GREATEST` with a partial index, MySQL functional
  key parts), so two requests at once cannot both pass. Existing repeated pairs stop the start with
  an educational error, like a field that became `único`.
- `ge explain <dado>` says it.

## Errors

The data does not point at the other data exactly twice (the error says how to declare the two
references); an unknown data; the phrase without the data.

## Evaluation

One line in the section of rules, with the word `única` people already use for `email … único`.
No index, no query, no status code for the author.

## Impact

Parser and resolver (`compiler/parser/agregados.go`; `ast.Entity.Pair`, `ast.Model.Pairs`),
interpreter (`prepareWrite`: every write, from any route), banco (`pairIndex`), `ge explain`.

## Performance and security

One `COUNT` with two alternatives on indexed columns per write of such a record; the index makes
the guarantee hold under concurrency.

## Compatibility and migration

Additive. A table that already holds a pair twice must be corrected before the index is created
(the start says so).

## Trade-offs and alternatives

Directed relations need the ordered form, not proposed here.

## Tests

Parser: `TestParUnicoFormas` (block, flat and `cada` forms; the two references; the errors).
Runtime, connected accounts in a finance domain: `TestParUnico` (repeated pair and reverse order
409; self-link 400; editing cannot create either; the database refuses the reverse pair written
directly). GitLab end-to-end: `TestLigacoesUnicas` (self-link 400, repeated link in either order
409 `Issue(s) already assigned`).
