# GEP 0034: A record moves to another parent (`pode mudar de projeto`)

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (GitLab stress test, inventory IS-09 "moves"); decision by the maintainer
- **Level:** 1
- **Layer:** domain (one capability under `pode`), core (the move: rules of both places,
  numbering, references)

## Problem

Things change place: an issue goes to another project, a ticket to another queue, an order to
another store, a document to another folder. Written by hand it is an edit of the parent
reference, and that edit is where products leak and break: the person may edit the record but
may not create anything in the destination (or cannot even see it); the number per parent
(`#12` of the project) collides or keeps the old place's number; labels and milestones of the
old place stay attached; a read-only (archived) place receives or lets go of records. GitLab has
a dedicated endpoint (`POST /issues/:iid/move`) for exactly these reasons.

## Evidence

GitLab inventory IS-09 (moving issues between projects). The same need in the reference
applications: tickets between support queues, tasks between boards, documents between folders.

## Current state

No construction. Editing the parent field through the generated surface either is not offered
or bypasses every rule of the destination.

## Alternatives studied

- **GitLab**: moving creates a copy in the destination (new number), closes the original with a
  note "moved to", and keeps labels and milestone only when the destination has ones with the same
  name.
- **Jira** ("Move issue"): a wizard that asks how to map statuses and fields that do not exist in
  the destination.
- **Editing the parent field** (what frameworks do): one `UPDATE`, every rule by hand.
- **Do nothing**: leaves the leak above to each application.

## Proposal

```text
issues
    pode
        fechar
        mudar de projeto
```

Flat form: `issue pode mudar de projeto`. The word after `de` is a data the record belongs to
(`pertence a`, or `projetos › tem › issues`).

## Semantics

- The record keeps its identity and everything that is its own (comments, files, history, links,
  time entries); only its place changes. It is offered as the action `mudar` of the record
  (`POST …/issues/7/mudar` with the destination as the parent field, `projeto_id`; the
  vocabulary of the integration applies).
- **Rules of both places.** Whoever moves must be allowed to change the record where it is
  (`editar`; `mudar` is already a synonym of `editar` in the rules, so `planner pode mudar
  issues` says the same) **and** to create such a record in the destination (the `criar` rules,
  membership and visibility of the destination). A destination the person cannot see does not
  exist for them (404). Moving to the place it already is in is refused.
- **Read-only places** neither let go nor receive: a record inside an archived project cannot
  move out, and nothing moves into one.
- **Numbering.** A number per parent (`numero por projeto`) becomes the destination's next one;
  numbers are never reused (the old place keeps the gap, and a later move into it does not repeat
  a number). The number still never changes by editing.
- **References to the old place do not follow.** A list named by a field (`labels por nome`)
  keeps the names the destination also has; any other reference to something that belongs to the
  old parent (the milestone of the project, a list by id) is emptied. People are kept.
- **One change.** The move, its history (`mudar`, with the new place) and its events happen in
  one transaction; a refusal leaves nothing behind.
- A data may move between parents of one kind only.
- `ge explain` shows "Pode mudar de projeto" with these rules.

## Errors

`pode mudar` without a place; `mudar de X` where X is not a data, or not a data the record belongs
to (the error shows the line to declare it); a transition already named `mudar`; two different
places for the same data. Each says what to write.

## Evaluation

One familiar phrase in the section that already lists what a data can do. No reference field, no
numbering, no check of the destination for the author to know; the dangerous parts (both rule
sets, read-only places, numbers) are derived.

## Impact

Parser and resolver (`movable`, `ast.Entity.MoveField`), runtime (`runtime/servidor/mover.go`,
the action route), `ge explain`.

## Performance and security

A move reads the destination, checks the rules of both places and writes the place and the new
number together (a number is unique only inside its place), then the ordinary update validates
the rest. Security: the destination's create rules are the same as creating there; nothing the
person cannot see is revealed; read-only places are respected.

## Compatibility and migration

Additive. Programs that do not declare `mudar de` behave as before.

## Trade-offs and alternatives

Unlike GitLab, the record is not copied: links to the old address (`/projects/1/issues/7`) stop
working instead of pointing to a closed copy. Keeping a forwarding trace would be a separate
decision. Pages do not offer the move yet (it needs a choice of destination); the action is on the
integration and page APIs.

## Tests

`TestMudarDeLugar` (runtime, a support desk with queues: next number of the destination, labels by
name, history; refused without the right to edit at the origin, to a destination the person cannot
see, to the same place, into or out of an archived queue; the number never changes by editing),
`TestMudarDeLugar` in the parser (block and flat forms; the errors) and the GitLab end-to-end test
`TestMoverIssue` (`POST /issues/:iid/move`).
