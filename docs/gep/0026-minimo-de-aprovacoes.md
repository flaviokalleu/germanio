# GEP 0026: A minimum of approvals before an action

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (MR-07, approval rules); decision by the maintainer
- **Level:** 1
- **Layer:** data (one line under `regras`), core (the action is refused until the approvals
  are there)

## Problem

Code review, expense reports, purchase orders, contracts and content publishing share one rule:
something happens only after enough other people said yes. `recebe aprovações` already lets
people approve a record (`aprovar`, `desaprovar`, the list of who approved). What was missing is
saying how many approvals an action needs. In frameworks this is a counter checked by hand in
every place that performs the action (API, page, board, background job), which is exactly where
it gets forgotten.

## Proposal

```text
merge requests
    recebe aprovações
    regras
        precisa de 2 aprovações para mesclar
```

and the flat form: `merge requests precisam de 2 aprovações para mesclar` (also `todo merge request
precisa de …`, `precisa de pelo menos 2 aprovações …`, the number written as `2` or `duas`). It
reuses the words of `precisa de pelo menos um owner` and of `recebe aprovações`; the only new
thing is the number and the action after `para`.

## Semantics

- **The action waits:** the action named after `para` (one of the data's actions in `pode`) is
  refused while the record has fewer approvals than the minimum. The refusal says how many it
  has, how many are missing and what to do (405 for clients of the API).
- **Who counts:** distinct people in the record's approvals. The approval of the record's owner
  (the author) never counts: whoever proposes does not approve their own proposal.
- **Everywhere:** the API, the page buttons, the board (GEP 0023) and merges that happen later
  (GEP 0027) go through the same check. A page hides the button and says how many approvals
  there are out of how many.
- **Visible:** the record shows `aprovacoes_necessarias` and `aprovacoes_faltando` (the largest
  minimum, when several actions have one). `ge explain` shows the rule and its origin.
- **Errors:** the data must `recebe aprovações`, the action must exist, the number is between 1
  and 100, and two different numbers for the same action are an error that says so.
- Order for merging: conflicts are told before missing approvals (approving does not fix them).

## Alternatives studied

- **Do nothing:** each application writes `antes de mesclar` + a count; it duplicates a common
  rule in every product and misses the page and the board.
- **A number in the data (`aprovacoes_necessarias` per record or per project):** more flexible,
  but a decision of the product turns into data anyone with edit rights changes; left for a
  later GEP if a product needs it.
- **Approval rules per group of people (`2 aprovações de maintainers`):** useful, larger; the
  phrase leaves room for it (`… de maintainers`) without deciding it now.
- **Counting the author's approval:** simpler, but makes "2 approvals" mean 1 other person.

## Tests

`TestMinimoDeAprovacoes` (parser: block and flat form are the same data, words and digits,
`pelo menos`; errors for a data without approvals, an unknown action, no number, zero, no action,
two numbers), `TestMinimoDeAprovacoes` in `runtime/` (expenses: the author's approval does not
count, approving twice counts once, the refusal says what is missing, another action does not
wait), and the GitLab E2E (`TestAprovacoesEMetodosDeMesclagem`).
