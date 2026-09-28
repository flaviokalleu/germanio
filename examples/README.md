# Germanio examples

Every example in the table below is written in the **intent layer**, the default level of the
language ([`docs/INTENCAO.md`](../docs/INTENCAO.md)): data described in blocks (`tem`, `pode`,
`acesso`, `permita`, `regras`), short phrases for standalone facts, and pages. None of them
contains HTTP, SQL, IDs or hand-written CRUD. CI runs `ge check` and `ge fmt --check` on each
one, and the test suite starts every application example and requests its pages
(`runtime/examples_smoke_test.go`).

```bash
go build -o ge ./cmd/ge
./ge check examples/todo/app.ge
./ge rodar examples/todo/app.ge 8080     # then open http://localhost:8080
```

| Example | What it demonstrates | Capabilities |
| --- | --- | --- |
| [hello-world](hello-world) | the smallest program, `mostre "Olá, mundo"` (strict core, no server) | output |
| [todo](todo) | personal tasks that start open and can be completed or reopened | login, ownership, states and transitions, search, filters, page |
| [crud](crud) | a customer register with no login | validation, uniqueness, search, filters, pagination, page |
| [authentication](authentication) | sign-in, sign-up, teams with roles, documents per role | login, lockout, initial administrator, roles, membership, minimum owner, permissions |
| [rest-api](rest-api) | a book catalog published to other systems, with access keys | integration API, access keys (secret, expiry, revocation), shared permissions |
| [blog](blog) | posts, comments, authors, drafts visible only to their author | ownership, nested data, restricted visibility, pagination |
| [crm](crm) | companies, contacts, opportunities that can be won or lost | relations, cascading delete, final states, ownership, filters |
| [ecommerce](ecommerce) | public catalog, administrator-managed products, customers' own orders | administrator-only actions, relations, states, validation |
| [fullstack](fullstack) | pointer to [gitlab-foss](gitlab-foss), the complete application | everything above, Git repositories, remote work, adapters |

Other directories:

- [`germanio/`](germanio) — programs for the strict core (functions, generics, tests,
  imports), run by CI with `ge rodar` and `ge testar`.
- [`gitlab-foss/`](gitlab-foss) — GitLab FOSS reimplemented in Germanio, with end-to-end tests.
- [`site-germanio/`](site-germanio) — the project website, built with the technical page level.

## Not yet supported

These kinds of application have no example because the intent layer cannot express them
simply today. They are open gaps, not missing files:

| Kind | Why | Gap |
| --- | --- | --- |
| Chat, live updates, WebSocket | real time has no rooms or recipients derived from permissions, so it is not offered to intent pages | [G66](../GERMANIO_GAPS.md) |
| Dashboard, indicators, charts | pages only have `mostre`, `permita` and `N por página`; `total de clientes`, charts and lists are direction, not contract | [G62](../GERMANIO_GAPS.md) |
| Stock that decreases with each order | no declarative rule for "cannot order more than the stock" | [G61](../GERMANIO_GAPS.md) |
| SaaS with plans and billing | no intent phrases for plans, subscriptions or payments; teams and roles are shown in [authentication](authentication) | not registered |

Known limitations found while writing these examples are registered as G76–G80 in
[`GERMANIO_GAPS.md`](../GERMANIO_GAPS.md) and mentioned in each README.

## A note on language

Germanio's keywords are **Portuguese**. `tem` = has, `começa` = starts, `pode` = can,
`acesso` = access, `permita` = allow, `regras` = rules, `somente` = only, `página` = page,
`mostre` = show. Accents are optional (`página` = `pagina`). The earlier technical syntax
accepts keywords in 20 languages, but the intent layer is Portuguese-only for now
([G74](../GERMANIO_GAPS.md)).

## Legacy

[`legacy/`](legacy) holds the examples written in the earlier technical syntax (`sistema` /
`dados` / `telas` with `campo: tipo`, or `app` / `tabela` / `quando receber`). It is still
supported and tested, but it is not the default level: do not copy it into new code. See
[`legacy/README.md`](legacy/README.md).
