# GEP 0050: A sum that never goes below zero (`não pode ficar com tempo gasto negativo`)

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (FASE 4, obstacle 7: `add_spent_time` read then wrote); decision by the maintainer
- **Extends:** [GEP 0047](0047-agregados.md) (numbers of a record), which left this rule to the adapter
- **Level:** 1 (one rule under `regras`)
- **Layer:** domain (one rule), core (the check inside the change's transaction, with the record locked)

## Problem

Many sums must never go below zero: the time spent on an issue (GitLab refuses subtracting more
than was spent), the stock of a product (the sum of its movements), the balance of an account
without overdraft, the seats left in a class. Written by hand, the check reads the total and then
writes the entry: two requests at once both read the same total and both pass, and the sum ends
negative. The GitLab adapter did exactly that (`gitlab_issues.ge`, `add_spent_time`).

## Evidence

FASE 4 obstacle 7 ("`add_spent_time` e operações parecidas leem e depois gravam"); GEP 0047 › Trade-offs
("a sum that must never go below zero … is still checked by the adapter, outside the change").
Stock, balances and quotas in any ERP or shop.

## Proposal

```text
issues
    indicadores
        soma da duracao dos tempos gastos como tempo gasto
    regras
        não pode ficar com tempo gasto negativo
```

Flat form: `issue não pode ficar com tempo gasto negativo` (`negativa`, `negativos`, `negativas`
also read). The name is a sum the data already shows under `indicadores` (GEP 0047).

## Semantics

- **When.** Every change of a record counted by the sum — created, edited (also moved to another
  record), deleted — from any path (the generated surface, an adapter through `superficie.pedir`,
  a hook, `.ge` code, the reset `zerar`) is checked.
- **How.** The change runs in a transaction (the request's, or one of its own when there is
  none). Before writing, the record(s) whose sum it touches are locked (SQLite: the immediate
  transaction already serialises writers; PostgreSQL/MySQL: `SELECT … FOR UPDATE`, taken before
  the write so the lock never has to be upgraded from the foreign key's share lock, in a fixed order
  when there are two records). After writing, the whole sum is computed by the database (one
  `SUM`, over **every** record, not only those the person sees: the rule is about the real total)
  and, below zero, the change is refused and undone.
- **Refusal.** 400 on the summed field (`{"message": {"duracao": ["deixaria tempo gasto de issue
  negativo; …"]}}`; English apps: "would make the tempo gasto of the issue negative"). An adapter
  translates it to its protocol (GitLab: `time_spent: Time to subtract exceeds the total time spent`).
- **Concurrency.** Subtractions at once are serialised by the lock: each one sees those committed
  before it; with 5 in stock, eight subtractions of 1 at once give exactly five successes.
- **Deleting the record itself** (an issue with its time entries) is not refused: its records are
  removed one by one with it, and a sum that goes away with the record no longer matters.
- `ge explain <dado>` shows the rule under the number.

## Errors

The name is not a number the data shows (lists the sums, or says to declare one); the number is a
count (a count is never negative); a sum restricted to a state (a record changing state would
change the sum without being written); an incomplete phrase (shows the phrase to write).

## Alternatives studied

- **`min 0` on the field**: says each entry is not negative, not the total; subtractions are
  entries with negative values.
- **A check constraint in the database**: a constraint sees one row, not the sum of a group.
- **A stored counter with a constraint**: would need a column kept in sync and would differ from
  the visibility-aware number of GEP 0047.
- **Keep it in the adapter or a hook**: races (what this GEP removes), and every product repeats it.
- **Do nothing:** negative stock under load.

## Performance and security

One row lock and one aggregate query per change of a counted record, only for data that has the
rule; nothing for other data. The lock is held for the rest of the change's transaction (kept short
by design: external effects run after the commit). A person who may add entries learns whether a
subtraction exceeds the total (as in GitLab); the total itself is never returned to someone who
cannot see every entry.

## Tests

Parser: `TestSomaNuncaNegativaFormas` (block and flat forms are the same application) and
`TestSomaNuncaNegativaErros`. Runtime, a stock domain with no Git and no GitLab
(`runtime/testdata/estoque`): `TestSomaNuncaNegativa` (refused on create, edit and delete; eight
concurrent subtractions with five in stock: five pass; deleting the product with its movements is
not refused). GitLab end-to-end: `TestControleDeTempo` (GitLab's message) and
`TestDescontarTempoAoMesmoTempo` (eight concurrent `add_spent_time?duration=-30m` with one hour
spent: two pass, six are refused, the total ends at zero).
