# GEP 0047: Numbers of a record (`indicadores` › `soma do peso das issues`)

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (GitLab stress test: milestone total weight, issue time spent, IS-09
  `time_stats` and `reset_spent_time`); decision by the maintainer
- **Extends:** [GEP 0012](0012-indicadores.md) (indicators of a page), which left sums for a GEP of
  their own
- **Level:** 1 (`indicadores`, `pode zerar`)
- **Layer:** domain (a section of the data block and one capability), core (one aggregate query
  per number, visibility, the atomic reset)

## Problem

Records show numbers about what belongs to them: a milestone shows the total weight of its
issues, an issue the time spent on it, an order its total, an account its balance, a project how
many open issues it has. Without a construction, the number is computed by hand: an adapter or a
hook reads the children page by page and adds them up (`gitlab_issues.ge` read the time entries
100 by 100), which loads every row, counts records the viewer may not see unless the author
remembers to check, and goes stale on an open page. Resetting a sum is worse: "read the total,
then write the entry that subtracts it" is two changes, and two resets at once both subtract.

## Evidence

GitLab: milestone total weight; `time_stats` (`total_time_spent`) and `reset_spent_time`
(IS-09). The same need everywhere: order totals, account balances, stock levels (the sum of the
movements), points, open items per project. `docs/INTENCAO.md` › Pendências da sintaxe listed
aggregates as a direction ("dependentes de uma GEP de agregados").

## Current state

GEP 0012 counts on a page (`indicadores` › `total de issues abertas`), under the viewer's
visibility. Nothing gives a record a number about its children, nothing sums, and nothing resets
a sum.

## Alternatives studied

- **A computed field in `tem`** (`peso total soma do peso das issues`): mixes a value people write
  with a value the system derives in the same list, and invents a second family next to the
  indicators of GEP 0012.
- **A stored counter kept up to date** (Rails `counter_cache`, the marks of GEP 0030): one column
  read for free, but one number for everybody — it would count confidential records for people
  who cannot see them. A number that respects visibility depends on the viewer, so it is computed
  when shown.
- **SQL views / ORM annotations** (Django `annotate(Sum(...))`, Ecto `aggregate`): the mechanism
  this GEP uses inside the core; at the domain level they are the technical vocabulary Germanio
  hides.
- **Doing it in the adapter or in hooks** (what existed): loads rows, forgets visibility, races.
- **Reset as delete-all-entries**: loses history; GitLab itself records a negative entry.
- **Do nothing**: every product writes the loop again.

## Proposal

```text
milestones
    indicadores
        soma do peso das issues como peso total
        total de issues abertas

issues
    indicadores
        soma da duracao dos tempos gastos como tempo gasto
    pode
        zerar tempo gasto
```

`indicadores` is the section name pages already use (GEP 0012); in a data block it lists the
numbers each record shows. Each line is:

- `total de <dado> [estado]` — how many records of the data point at the record;
- `soma do <campo> dos <dado> [estado]` — the sum of a numeric field (`de/do/da/dos/das` as reads
  naturally);
- optionally `como <nome>` — the name of the number in the record; without it, the line itself
  (`total de issues abertas` → `total_de_issues_abertas`).

Flat form: `milestone mostra soma do peso das issues como peso total` (`mostra a soma…` and
`mostra o total…` also read).

`pode` › `zerar <nome>` (flat: `issue pode zerar tempo gasto`) brings a sum back to zero.

## Semantics

- **What counts.** Records of `<dado>` that point at the record through their reference to it
  (the owner reference, `issue_id`, when there are several), only in the state when one is
  written (`abertas` for the state `aberta`, as in GEP 0012).
- **Only what the viewer sees.** A number counts exactly the records the viewer could list, with
  the list's rules (confidential records, private projects, ownership); an administrator counts
  everything. A sum of a private, hidden or secret field is refused: it would reveal the values.
- **Computed, never stored, never written.** The value appears in the record (API, integration
  with the vocabulary, the record's page), is `0` when nothing counts, and is ignored when sent.
  Its name may not repeat a field.
- **Live.** A change of a counted record (created, edited, moved to another parent, deleted)
  announces the record(s) it counts in, so open pages follow (GEP 0020).
- **Zerar.** `zerar_<nome>` (an action of the record) creates, as the person and with every rule,
  hook, history entry and event of creating it, one record of `<dado>` whose field subtracts the
  whole sum; records are never deleted. Who may: whoever may create those records in that record.
  It is refused (403) when the person does not see every record counted (the subtraction would
  reveal their sum). It runs in the request's transaction after locking the record, so two resets
  at once never both subtract: the second finds zero and records nothing.
- `ge explain <dado>` lists each number, what it counts and through which reference, and the reset.

## Errors

An empty `indicadores`; a line that is neither `total de` nor `soma`; an unknown data; data that
does not belong to the record (the error shows the line to declare it); a word that is not a state
(lists the states) or a state of data without states; a field that does not exist (lists the
numeric ones) or is not a number; a private/hidden/secret field; a quoted label (use `como`); a
name already used by a field or another number. `zerar` of nothing, of an unknown name, of a count,
of a sum with a state, of a field with `min 0` (the entry is negative), or when the counted data
has other required fields (the entry would be incomplete). Each says what to write.

## Evaluation

One familiar line per number, in the section that already means "numbers"; no query, no
aggregate function, no visibility filter, no paging, no lock for the author.

## Impact

Parser and resolver (`compiler/parser/agregados.go`; `ast.Entity.Aggregates`), banco
(`Banco.Agregar`: one grouped `COUNT`/`SUM` query), runtime (`runtime/servidor/agregados.go`:
batch per list page, visibility per group, the reset, live announcements), `ge explain`, the
record page.

## Performance and security

A number is **one aggregate query** for a whole page of records (`GROUP BY` the reference):
records are never loaded. For an administrator that is all. Otherwise the query also groups by
the columns visibility depends on (the record's parents, owners, visibility field and the flags
and lists of its restrictions) and the rules are applied **once per distinct combination**, not
once per record (a milestone's issues usually make one or two groups). Data whose visibility
depends on the record's own identity (members of its own, a hierarchy, `antes de ver`) is read in
batches of 500 and checked record by record — the same cost as listing it. The reset holds the
record's lock (SQLite: the immediate transaction; PostgreSQL/MySQL: `SELECT … FOR UPDATE`) only
for one sum and one insert. A number never includes a record the viewer could not list.

## Compatibility and migration

Additive: no column is created.

## Trade-offs and alternatives

Computing on read costs one query per number per page; a stored counter would be free to read but
wrong for viewers with less visibility. Pages show the numbers in the record's details; showing
them as table columns (`colunas`) and sums on dashboards (`indicadores` of a page) are left for
later. A sum that must never go below zero (GitLab refuses subtracting more time than was spent)
is still checked by the adapter, outside the change.

## Tests

Parser: `TestAgregadosFormas` (block and flat forms are the same application; counts, states,
names, labels, zerar) and `TestAgregadosErros`. Runtime, a finance domain with no Git or GitLab
(`runtime/testdata/agregados`): `TestAgregados` (zeros; the author, a reader and an administrator
see different numbers; the list fills every record; values sent are ignored; the record page; live
list page; reset refused to a reader and to a treasurer who does not see every entry; eight resets
at once subtract once). GitLab end-to-end (`examples/gitlab-foss/e2e/agregados_test.go`):
`TestPesoTotalDaMilestone`, `TestTempoGastoTotal` (eight concurrent `reset_spent_time`).
