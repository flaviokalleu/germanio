# GEP 0010: Discarding a field on purpose

- **Status:** Em teste (implemented with G93; not normative until the maintainer decides)
- **Author:** agent (G93); decision by the maintainer
- **Gaps:** G93
- **Level:** 1
- **Layer:** domain (one phrase), core (migration planning)

## Problem

G93 made renames explicit (`renomeie nome para nome_completo`, requested by the maintainer).
Once Germanio refuses to guess, a second case needs an answer: a field that held data is
removed **on purpose** while another, unrelated field is added in the same version. Without a
way to say "this one really left", the start stops with the probable-rename error and the
programmer has no way forward except renaming or dropping the column by hand.

## Evidence

`runtime/banco/migracao.go` (`planMigration`): orphan columns with data plus new declared
fields are refused, because the names alone cannot prove a rename. Rails, Django and EF Core
face the same ambiguity and ask the developer (a migration file, an interactive question).

## Proposal (em teste)

```text
clientes
    tem
        nome_completo
    descarte fax
```

Flat form: `descarte fax de clientes`. The application stops using the field; its values stay
in the database, nothing is deleted.

## Alternatives studied

- **No phrase: removing a field always warns and continues.** Silently accepts what may be a
  forgotten rename; contradicts "Germanio nunca infere um rename".
- **Delete the column (`apague fax`).** Destructive; would need its own GEP and a stronger
  confirmation. Not proposed.
- **Other words**: `remova`, `abandone`, `deixe de usar`. `descarte` was chosen because it says
  the intent (it is no longer used) without promising deletion; the choice is the maintainer's.

## Semantics, errors and tests

`descarte X` fails if X is still declared in `tem`. It silences the orphan warning and lets the
start continue. Tests: `TestRenomeProvavelSemDeclaracaoPara` (discard keeps data, no warning),
`TestHierarquiaErros` (discard of a declared field), the block/flat equivalence.

## Compatibility

Additive. Removing or renaming the phrase before acceptance affects only programs written since
G93.
