# GEP 0027: How a proposal is merged, and merging when the executions pass

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (MR-07, merge strategies); decision by the maintainer
- **Level:** 1 (no new syntax), 2 (choices are data of the records)
- **Layer:** core (code review: records with `origem` and `destino` branches; Git without a
  working tree), adapter vocabulary (external names only)

## Problem

A record that proposes changes between two branches (`origem: branch`, `destino: branch`) is
merged with a merge commit. Teams want three more things, in every code review product: a
linear history (no merge commits), one commit per proposal (squash), and "merge it when the
checks pass" so nobody has to wait in front of the screen. In frameworks this is a set of Git
commands, a job queue that watches the pipeline and a second permission check at the moment the
job runs, which is where it usually goes wrong.

## Proposal

No new phrase. The review capability already exists; it gains three choices, which are data of
the records and are inspected by `ge explain`:

| Field | Of | Meaning |
| --- | --- | --- |
| `forma_de_mesclar` | the record with the repository (`projeto`) | `mesclagem` (default): a merge commit; `semi_linear`: a merge commit over an `origem` brought up to date first; `linear`: no merge commit, `destino` only advances to an `origem` brought up to date |
| `juntar_commits` | the proposal | every change becomes one commit on top of `destino`, with the proposal's title |
| `mesclar_quando_passar` | the proposal (only when the record with the repository `executa` something) | the merge waits for the latest execution of `origem` |

`mesclar` also accepts `juntar_commits` and `mesclar_quando_passar` for one merge, and
`cancelar_mesclagem` stops waiting. External names come from the vocabulary of the adapter
(`squash`, `merge_method` with `merge`/`rebase_merge`/`ff`, `merge_when_pipeline_succeeds`,
`cancel_merge_when_pipeline_succeeds`); the core knows none of them.

## Semantics

- **Bringing `origem` up to date** replays its commits on `destino` (authors and messages kept,
  merge commits dropped, at most 1000 commits) and rewrites `origem`, so it follows the rules of
  sending code to `origem` (protected branches). A commit that does not apply refuses the merge
  with the conflicting files; nothing changes. Every branch moves by compare-and-swap.
- **Juntar commits** wins over the form: one commit, one parent (`destino`).
- **The record keeps** `commit_mesclagem` and, internally, where `destino` was, so the changes and
  commits of a merged proposal are still shown correctly after a squash or an advance.
- **Mesclar quando passar:** everything a merge needs (open, not a draft, no conflicts, the
  approvals of GEP 0026, the right to merge into `destino`) is checked when it is asked. If the
  latest execution of the current version of `origem` is still running, the proposal waits and
  remembers who asked; if it already passed, or nothing runs for that version, it merges now; if
  it failed or was canceled, it is refused (the person asked to merge only if it passes).
- **When the execution ends:** only the latest execution of the current version of `origem`
  counts (a new push makes the proposal wait for the new one). If it passed, the merge happens as
  the person who asked, with every rule checked again; if anything refuses, or the execution
  failed or was canceled, the proposal stops waiting and stays open. Closing the proposal stops
  the waiting too. The merge happens after the execution's lock is released.
- **Pages:** the project form offers the three forms, the proposal form the checkbox; a waiting
  proposal says so and offers "Cancelar mesclagem" to whoever may merge.

## Alternatives studied

- **Do nothing:** each product writes `quando` hooks that call Git and poll executions; the
  second permission check is forgotten.
- **Phrases in the program (`merge requests mesclam em histórico linear`):** a fixed choice per
  program, while the products studied let each project choose; the choice is data.
- **Refusing instead of bringing `origem` up to date** (what GitLab does, with a separate rebase
  button): one more action for the person; replaying is safe because conflicts refuse and the
  rules of sending code apply.
- **Keeping the schedule after a failed execution:** a later push would merge something the
  person did not see pass again; stopping is the safe default.

## Tests

`TestSquash`, `TestRebaseFastForward`, `TestRebaseConflict` (`runtime/git`);
`TestFormasDeMesclar` and `TestMesclarQuandoPassar` (`runtime/`, a book with chapters proposed in
branches and builds as executions: squash, linear, semi-linear, a value outside the list, merge
after a passing build, a failing build stops the waiting and refuses a new request, cancel,
merge now when the build already passed); `TestAprovacoesEMetodosDeMesclagem` (GitLab E2E:
`merge_method: ff`, `squash: true`, `merge_when_pipeline_succeeds`).
