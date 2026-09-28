# GEP 0011: History (activity)

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (GitLab stress test, inventory NT-02); decision by the maintainer
- **Level:** 1
- **Layer:** domain (one phrase), core (recording and visibility); the data itself is ordinary
  Germanio

## Problem

People need to know what happened: who opened, edited, closed or deleted something, and when.
GitLab calls it the activity feed (events); ticketing, CRMs and ERPs call it history or audit
trail. Today it means a hand-written data plus a `quando` hook on every verb of every data —
the kind of repeated logic the norm says to avoid, and easy to get wrong in the part that
matters most: an activity must never reveal a record the reader cannot see.

## Evidence

GitLab inventory NT-02 (`/events`, `/projects/:id/events`). The same need appears in every
reference application of the mandate (SaaS/ERP audit, chat history, editors' undo history is a
different concept and is not covered here).

## Current state

No construction. `a.emit` (`runtime/servidor/eventos.go`) is already the single point every
change of the intent surface passes through (create, edit, delete, transitions, code pushes),
and it runs inside the change's transaction.

## Alternatives studied

- **GitLab events**: one row per action with author, action, target type/id and project.
- **Rails `paper_trail` / Django `simple_history`**: a copy of the record on every change. Keeps
  values, which leaks secrets and grows fast; rejected for level 1.
- **Event sourcing**: the history *is* the state. Changes the whole model; out of scope.
- **A hook per data**: what exists today; rejected (repetition, easy leaks).

## Proposal

```text
issues
    tem
        titulo
    guarda histórico
```

Flat form: `issue guarda histórico`.

## Semantics

- Germanio writes an ordinary data, `atividades`, when the program does not declare it
  (`ge explain atividades` shows it, with the origin `<histórico (GEP 0011)>`): `acao` (criar,
  editar, excluir, the transition verbs, enviar_codigo), `recurso`, `recurso_id`, `resumo`
  (the record's title), `campos` (for an edit: the **names** of the fields that changed, never
  their values), `dentro`, `dentro_id` (the record's parent, when it belongs to one) and `autor`.
- One activity per change, written in the change's transaction: an undone change leaves no
  activity.
- **Who sees an activity is who sees the record it describes, now.** A confidential issue's
  activity is visible only to those who see the issue; if the record was deleted, to those who
  see its parent; with neither, only to its author. Nobody edits or deletes activities.
- Deleting a record empties the `resumo` of all its activities (and its `excluir` activity has
  none): once the record is gone, its activities are seen by whoever sees its parent, and a
  confidential title must not reach people who could never see it.
- Filtering by `recurso`, `recurso_id`, `dentro`, `dentro_id` and `autor` lists the history of
  one record, of what belongs to a record (a project's activity) or of one person.
- To show it, the application declares a page: `página Atividade` + `mostre atividades`.

## Errors

`guarda histórico` in a data that is not declared: the usual unknown-data error. Declaring
`atividades` without the fields above: the error lists the missing ones.

## Evaluation

One phrase, no technical concept; the history is visible and explainable; the dangerous part
(visibility) is derived, never written by the author.

## Performance and security

One insert per change. Listing checks each activity against the record it describes (the
record-dependent path of the list: batches, per-row check, correct totals); the read cache
avoids reading the same record twice. Values are never stored, so a secret field's content
cannot reach the history.

## Compatibility and migration

Additive.

## Trade-offs

Values are not kept, so "show me the old title" is not answered (a later `guarda versões` could
be its own GEP). A project's activity lists what belongs *directly* to it; deeper levels
(a group's activity over its projects' issues) would need the ancestor chain.

## Tests

`TestHistorico` (created/edited/transition/deleted activities; field names without values;
undone change leaves none; confidential record's activity hidden from others; deleted record
keeps no title) and the block/flat equivalence.
