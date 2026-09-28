# GEP 0009: Pending items

- **Status:** Aceita (2026-09-28, pelo mantenedor). Norma: `docs/INTENCAO.md` › Pendências
- **Author:** agent (GitLab stress test, inventory NT-01); decision by the maintainer
- **Level:** 1
- **Layer:** domain (one phrase), core (the trigger); the data itself is ordinary Germanio

## Problem

When someone is made responsible for something (the assignees of an issue, the reviewers of a
merge request, the attendant of a ticket), they need a list of what is waiting for them. Today
that means a hand-written data, a hook on create and on edit that compares the old and new
people, cleanup on delete — the kind of `quando` logic the norm says to avoid.

## Evidence

GitLab inventory NT-01 (todos). The same need appears in ticketing (Whaticket, Evoticket),
CRMs (tasks of a salesperson) and any workflow with responsibility.

## Current state

No construction. Hooks can do it, at the cost the norm warns about.

## Alternatives studied

GitLab todos (created on assignment and mention, done by the person); GitHub notifications
(an inbox per person, derived from events); Jira "assigned to me" (a query, not a record).
A query would not let the person mark something as done; a record per person does.

## Proposal

```text
issues
    tem
        responsaveis
    pendência para
        responsaveis
```

Flat form: `issue gera pendência para responsaveis`. The fields must hold people (one person
or a list of people).

## Semantics

- The pending items are an ordinary data, `pendencias`, which Germanio writes when the program
  does not declare it (and `ge explain pendencias` shows it, with the origin
  `<pendências (GEP 0009)>`): `motivo`, `recurso`, `recurso_id`, `dono`; starts `aberta`;
  can be `concluida`; each person sees, completes and deletes only their own. An application
  may declare its own `pendencias` with the same fields; its definition wins.
- A person newly placed in a pending field (on create or edit) receives one item; whoever makes
  the change does not get one for themselves.
- A person removed from the field loses their open items for that record; completed ones stay.
- Deleting the record deletes its items.
- Everything happens in the transaction of the change.
- To show them, the application declares a page: `página Pendências` + `mostre pendências`.

## Errors

A field that does not hold people: the error names the field and suggests people fields.

## Evaluation

One phrase, no technical concept; the data is visible and explainable.

## Impact

Parser: the phrase and the `pendência para` section. Resolver: the synthesized data and the
field check. Runtime: the trigger on create, edit and delete.

## Performance and security

One insert per new person; lists stay private to their owner through the ordinary rules.

## Compatibility and migration

Additive.

## Trade-offs and alternatives

Mentions in text (`@ana`) are not covered: they need a notion of mention first. Marking items
done automatically when the record closes is not done either; it would need a decision on which
states "resolve" responsibility.

## Tests

`TestPendencias` (created for the assignee, not for the one assigning, private, completed,
removed with the person, deleted with the record) and the block/flat equivalence.
