# GEP 0012: Indicators (counts on a page)

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (GitLab stress test, inventory AD-01); decision by the maintainer
- **Gaps:** G62 (indicators were left out of GEP 0002)
- **Level:** 1
- **Layer:** domain (a page section), core (counting under the viewer's visibility)

## Problem

Dashboards start with numbers: how many users, open issues, orders today. GEP 0002 left
`indicadores` out because they need a semantics for aggregates. Without it, a dashboard means
the technical page level or a hand-written route counting rows — which, done by hand, easily
counts records the viewer may not see.

## Evidence

GitLab AD-01 (the admin dashboard: counts of users, projects, groups, issues, merge requests);
the dashboard reference application of the mandate (phase 5); `docs/INTENCAO.md` › Pendências da
sintaxe (`total de clientes` as direction).

## Proposal

```text
página Painel
    indicadores
        total de usuarios
        total de projetos
        total de issues abertas
        total de merge requests mesclados "Mesclados"
```

A section of the page, like `topo` or `vazio`. Each line is `total de <dado>`, optionally
followed by one of the data's states (written as it reads: `abertas` for the state `aberta`)
and an optional label. A page may have only indicators (no `mostre`): a dashboard.

## Semantics

- **A number counts only what the viewer can see.** The count follows exactly the rules of the
  list (the same record-by-record visibility, confidential records, private projects): an
  administrator sees every record, anyone else the number of records they could list. A viewer
  who cannot see the data at all does not see the indicator.
- The state must be a state of the data (the initial one or the target of a transition).
- The label defaults to the line itself ("Total de issues abertas").
- Only counts. Sums, averages, groupings and time windows (`vendas do mês`) are not in this GEP:
  each needs a decision (which field, which period, which time zone) and gets its own GEP.

## Errors

Unknown data: the usual error. A word after the data that is not one of its states: the error
lists the states. A line not starting with `total de`: the error shows the form.

## Evaluation

One familiar phrase per number; no query, no aggregate function, no visibility to write.

## Performance and security

An administrator's count, or a count of data whose visibility does not depend on each record,
is one `COUNT`. Otherwise the count reads in batches and checks each record, like the list;
large volumes will need counts maintained incrementally (phase 4). The number never includes a
record the viewer could not list.

## Compatibility and migration

Additive.

## Tests

`TestIndicadores` (administrator counts all; a person counts only what they see; state filter;
hidden indicator for data the viewer cannot see; dashboard page without `mostre`), parser
errors, and `ge explain pagina`.
