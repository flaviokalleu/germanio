# GEP 0016: Protected branches named by data

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (GitLab stress test, inventory RP-06); decision by the maintainer
- **Level:** 1
- **Layer:** domain (the target of an existing phrase), core (push check)

## Problem

`somente maintainer pode enviar código para a branch padrão dos projetos` protects one branch.
Real projects protect several, chosen by the people who manage each project and changing over
time (`release/*`, `stable`) — so the list is data, not source code.

## Evidence

GitLab inventory RP-06 (protected branches with wildcards). Any application hosting code needs
the same.

## Proposal

```text
branches protegidas
    tem
        nome obrigatório
    pertence a projeto
    acesso
        maintainer
            administrar

projetos
    acesso
        somente maintainer
            enviar código para as branches protegidas
```

Flat form: `somente maintainer pode enviar código para as branches protegidas dos projetos`.

## Semantics

- The target of `enviar código para` may be a data whose records belong to the record with the
  repository and have a `nome`. Each `nome` is a branch or a pattern where `*` stands for any
  text (`release/*`).
- Pushing, creating or deleting a branch that matches a record of that project needs the role
  (or a higher one); anyone else gets the same refusal as for the default branch. Administrators
  are not refused.
- Adding a protected branch is ordinary data: who may do it is decided by its own access rules.

## Errors

A target that is neither `a branch padrão` nor a data belonging to the data with the repository:
the error says which data it must belong to. A data without `nome`: the error says so.

## Evaluation

The same phrase the norm already has; the new part is a data the domain already understands.

## Performance and security

One query per push (the project's records), matched in memory. Merges are not covered yet: a
merge request merged into a protected branch follows the `mesclar` permission (a later GEP).

## Tests

`TestBranchesProtegidas` (a developer cannot push to a protected pattern, a maintainer can,
unprotected branches stay open, removing the record lifts the protection), resolver errors.
