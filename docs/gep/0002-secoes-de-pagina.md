# GEP 0002: Page sections

- **Status:** Rascunho
- **Author:** agent (research consolidation); decision by the maintainer
- **Gaps:** G62 (and G80: `N por página` only under `mostre`)
- **Level:** 1 (default)
- **Layer:** domain (syntax), core (page model and renderer)

## Problem

A page can only say what it shows (`mostre`), which actions it offers (`permita`) and how
many rows per page. A person cannot say "the page title is *Clientes*", "the create button is
called *Novo cliente*", "filter by status and city", "show these columns" or "when there is
nothing, say *Nenhum cliente* and offer to register one". Today those are fixed by the
renderer, so a real product ends up wanting the technical page level (`pagina "/…"`, with
`hero`, `navbar`…), which is closer to HTML.

## Evidence

- `docs/INTENCAO.md` › Pendências: `topo`, `vazio`, `gráfico`, `lista` and indicators are
  "direction, not contract"; the grammar names `secao_pagina` without defining it.
- `docs/research/frontend/GERMANIO_FRONTEND.md` §3 and `COMPONENTS.md` §4: the proposal and
  its sources (design systems name the same regions: header, actions, filters, table, empty
  state).
- The examples written for the launch could not give their pages titles, labels or empty
  states (`examples/README.md` › Not yet supported).

## Current state

`ast.PageDecl{Name, Manage, Show, Permits, PerPage, Pos}` (`compiler/ast/intencao.go`);
`runtime/servidor/paginas.go` derives the title from the name, the columns from the entity,
the actions from `permita` filtered by the viewer's grants (`available`), and a fixed empty
message.

## Alternatives studied

- **Component trees (React, Vue, Svelte)**: the author composes components; powerful, but
  requires thinking in components and props. Rejected for level 1.
- **Page builders (blocks with free layout)**: visual and flexible, but the result is not a
  description of intent and cannot be validated against the domain.
- **Named regions with domain meaning (design systems' page anatomy)**: a small closed set of
  regions, each with a default derived from the domain. Chosen.

## Proposal

A closed table of page sections, like the table of data sections. Every section is optional;
`página Clientes` + `mostre clientes` alone keeps producing the whole page from defaults.

Before (today, still valid after):

```text
página Clientes
    mostre clientes
    permita
        pesquisar
        criar
        editar
        excluir
```

After (proposed; `text` because it does not compile yet):

```text
página Clientes
    mostre clientes
    topo
        título "Clientes"
        ações
            criar "Novo cliente"
    filtros
        pesquisar
        status
        cidade
    colunas
        nome
        email
    vazio
        título "Nenhum cliente"
        texto "Cadastre seu primeiro cliente."
        ação criar "Cadastrar cliente"
    permita
        pesquisar
        criar
        editar
        excluir
```

| Section | Parent | Content | Checked against the domain | Default when absent |
| --- | --- | --- | --- | --- |
| `topo` | page | `título`, `texto`, `ações` | — | title = page name |
| `título "X"` | topo, vazio | one text | — | — |
| `texto "X"` | topo, vazio | one text | — | — |
| `ações` | topo | one verb per line, optional label | the verb is in the page's `permita` (see Semantics) | the collection actions of `permita` |
| `filtros` | page | `pesquisar` or one field per line | the field exists in the shown data; `pesquisar` needs `permita pesquisar` on the data | from the data's `permita pesquisar/filtrar` |
| `colunas` | page | one field per line, in order | the field exists and is not `privado`/`oculto` | the entity's visible fields |
| `vazio` | page | `título`, `texto`, `ação` verb ["label"] | the action is in `permita` | a message derived from the data's name, plus `criar` if allowed |

`indicadores` and `gráfico` are **not** in this GEP: they need a semantics for aggregates
(`total de clientes`, `vendas do mês`) and get their own GEP.

## Semantics

- **The page asks; the domain decides.** `ações › criar` gives a label and a position to a verb
  the page already allows; the button is rendered only for people who have the grant, exactly
  as `permita` works today. A verb in `ações` or `vazio` that is not in the page's `permita` is
  an error (option *a* of GERMANIO_FRONTEND §3.2; options *b*, "ações replaces permita", and
  *c*, "ações implies permita", were rejected because they create a second place that grants
  or hides actions).
- **A label is text, an action is a verb.** `criar "Novo cliente"` is deterministic; a bare
  `"Novo cliente"` is an error ("say which action: criar, …").
- Each section replaces only its own slot; merging two pages of the same name follows the
  merge rule (same fact twice is harmless, two different titles are a conflict with both
  origins).
- `ge explain pagina Clientes` shows every node with its origin: the line and path
  (`Clientes › topo › ações`) or "default, from `permita criar`".
- `N por página` becomes a section of the page as well (G80), keeping `mostre` › `20 por página`
  valid.

## Errors

Unknown section: lists the valid ones and the closest ("você quis dizer `filtros`?"). Field
not in the data: names the data and its fields. Verb not allowed: points at the `permita` it
should be added to — as a *probable* suggestion only, because adding a verb to `permita`
changes permissions (DIAGNOSTICS.md: a fix that grants is never automatic).

## Evaluation

Concepts added: five section names with domain meaning (top, actions, filters, columns,
empty), no visual concept (no colors, sizes, grids). Depth: at most four levels
(`página › topo › ações › verbo`), the limit the norm recommends. Repetition: none (the data
comes from `mostre`; `tabela clientes` from the earlier sketch was dropped for repeating it).

## Impact

- Parser: `hierarquia.go` gains the page section table (the same mechanism as data sections).
- AST: `PageDecl` gains `Header{Title, Text, Actions[{Verb, Label}]}`, `Filters`, `Columns`,
  `Empty{Title, Text, Action}`, each with `Pos` and path.
- Resolver: validation against the entity and against `permita`.
- Runtime: `paginas.go` reads the page model instead of fixed defaults; the analysis moves out
  of the per-request `serve` (G85-related work).
- Tooling: `ge explain pagina`, `ge fmt`, the VS Code grammar (generated).

## Performance and security

The page model is computed once per program load; rendering cost does not grow. No new way to
grant: security is unchanged by construction (the verb must already be allowed).

## Compatibility and migration

Additive: every existing page keeps its meaning. No migration.

## Trade-offs and alternatives

The closed table limits what a page can express at level 1; richer layouts stay in the
technical level or in future blocks (a copied `.ge` file, never a keyword). "Do nothing"
keeps pushing real products to the technical page level.

## Tests

Equivalence between block and flat forms for every section; the page tree per role
(anonymous, logged in, administrator) asserted on the model, not on HTML; errors for unknown
sections, missing fields and verbs not allowed; the examples updated to use `topo`, `vazio`.
