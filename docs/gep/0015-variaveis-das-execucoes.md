# GEP 0015: Variables of executions

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (GitLab stress test, inventory CI-07); decision by the maintainer
- **Level:** 1
- **Layer:** domain (one phrase), core (step environment, masking)

## Problem

Executions (`projeto executa pipelines …`) need configuration and secrets that must not live in
the repository: a deploy key, an API token, a target address. The people who manage the project
set them once; every step receives them; nobody reads a secret back, and a secret printed by a
script does not appear in the log.

## Evidence

GitLab inventory CI-07 (project CI/CD variables, masked). The same need exists for any
reference application that runs work (deploys, data processing, scheduled jobs).

## Proposal

```text
variaveis
    tem
        chave obrigatório
        valor texto oculto
    pertence a projeto
    acesso
        maintainer
            administrar

pipelines usam as variaveis do projeto
```

Block form, in the block of the executions: `usam variaveis do projeto`.

## Semantics

- Each record of the data that belongs to the execution's owner (the project) becomes a
  variable of every step: its `chave` (or `nome`) is the name, `valor` the value.
- A `valor oculto` is written by whoever may edit it and never returned by the API (the
  existing meaning of `oculto`); it reaches only the executor that runs a step of that owner, and
  is replaced by `[MASKED]` in the step's log (external executors are told to mask it).
- Variables never replace the names the execution already provides (the core's `CI`, the
  adapter's names): those win.
- The data must have a name field (`chave` or `nome`) and a text `valor` (declared
  `valor texto`: by its name alone, `valor` would be money), and belong to the owner of the
  executions; otherwise the error says which part is missing.

## Errors

`pipelines usam as variaveis do grupo` when pipelines belong to projects: the error names the
owner. Missing fields: the error lists them.

## Evaluation

One phrase in the words of the domain; secrets are safe by default (hidden, masked).

## Performance and security

One query per step claimed. The value goes only in the claimed step's payload (to the executor
holding that step's token) and in the local executor's environment.

## Compatibility and migration

Additive.

## Tests

`TestVariaveisDasExecucoes` (a step receives the owner's variables; another project's do not
leak; a hidden value is masked in the log and never returned by the API; the execution's own
names win), the block/flat equivalence, errors.
