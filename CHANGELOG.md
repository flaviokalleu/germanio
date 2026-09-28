# Changelog

All notable changes to Germanio. The format follows [Keep a Changelog](https://keepachangelog.com/),
and versions follow [Semantic Versioning](https://semver.org/). Germanio is **pre-1.0**:
minor versions may still change the language, and changes of meaning go through
[GEPs](docs/gep/README.md).

The project was called **Flang** (files `.fg`) until 2026-09-27. Releases up to v0.6.0 were
published under that name.

## [Unreleased] — 0.7.0-dev

Germanio as it is today: the rename, a strict typed core, the intent layer and the
hierarchical syntax. Nothing below is in a tagged release yet.

### Language

- Renamed Flang to Germanio; source files are `.ge`.
- **Strict core** ([SPEC.md](SPEC.md)): typed lexer, parser and checker with located
  diagnostics, immutable values by default (`variavel` for mutable), optionals with
  flow-sensitive refinement, private functions, generic functions with inference, local
  modules, native `.ge` tests with instruction coverage (`ge testar`).
- **Intent layer** ([docs/INTENCAO.md](docs/INTENCAO.md), normative): an application says
  what exists, who may do what, what happens and what appears — data and relations, states and
  transitions, roles, membership and inheritance, permissions and visibility, validation,
  search and filters, pages, login, integration for API clients, Git repositories, remote
  work for executors (lease, heartbeat, idempotency).
- **Hierarchical, contextual syntax**: what is indented below belongs to the context above;
  every block reduces to equivalent flat phrases, and a test proves the equivalence.
- Merging of blocks of the same data, with conflicts reported with both origins.

- Page sections (`topo`, `ações`, `filtros`, `colunas`, `vazio`; GEP 0002), password recovery
  (`tenha recuperação de senha`; GEP 0008) and pending items (`pendência para`; GEP 0009).
- Explicit migrations (G93): `renomeie nome para nome_completo` renames a column keeping its
  data; `descarte fax` (GEP 0010, in test) records a field removed on purpose. Germanio never infers a rename: a
  field with data that disappears while another appears stops the start with an educational
  error, and `ge check` performs the same verification on the existing database without
  changing it.
- History (GEP 0011, in test): `issue guarda histórico` records who did what to which record,
  in the change's transaction, with the names of changed fields and never their values; each
  activity is seen only by whoever sees the record it describes. A vocabulary that translates
  the same name twice with different values is now an error instead of a silent override (G105).
- Notices by e-mail (GEP 0013, in test): `tenha avisos por e-mail` sends each new pending item to
  its owner after the change is saved. `tenha` now refuses impossible data names instead of
  creating phantom data (G108).
- Indicators (GEP 0012, in test): a page section `indicadores` with `total de issues abertas`;
  each number counts only what the viewer could list, and a page may be only indicators (a
  dashboard).
- Security: a list of data with a restriction such as `confidencial pode ser vista por autor`
  returned restricted records to people who could not see them (the single record was already
  protected); restrictions now make the list check each record (G106).
  A pending item names the record's title only if its owner may see the record (G107).
- External effects run after the commit (G86): inside a change, `chamar` with a writing
  method, webhooks and messages are recorded and run once the change is saved, in order,
  never holding the database's write lock. An undone change has no effects; a failing effect
  does not undo the change; using an effect's answer inside the change is an educational
  error. `ge explain` lists each hook's effects in the order they run.

### Tooling

- `ge` CLI: `new`, `run`, `check`, `fmt`, `explain`, `test`, `explicar`.
- `ge fmt`: one canonical form, idempotent, refusing any output that changes the program's
  meaning; covers application files.
- `ge explain <data>`: every fact the compiler knows, with the equivalent flat phrase and
  its origin (file, line, hierarchical path).
- Educational errors: what, where, why and how to fix.
- Permanent benchmark suite ([bench/](bench/README.md)) comparing Germanio with the same
  application written directly in Go.

### Runtime and security

- Request transactions, hierarchical addresses, lists by name, minimum roles, read-only
  conditions, initial administrator from the environment, reserved addresses.
- Git smart HTTP hosting, verified before references are updated.
- Password recovery: a link sent several times at once now changes the password once (the link
  is claimed in the same transaction that changes the password; before, concurrent requests
  could all succeed), and the e-mail is sent off the answer's path, so the answer time does
  not tell whether the account exists.
- Security fixes: trusted proxies only for forwarded headers; outgoing requests protected
  against SSRF; list references must share the record's parent; `administrar` only governs
  data that belongs to the record; real time (`/ws`) requires a session, the same origin, and
  change notices no longer carry record data.

### Stress test

- A reimplementation of core GitLab concepts in `.ge` ([examples/gitlab-foss](examples/gitlab-foss)),
  including CI jobs run by the official `gitlab-runner` through an adapter.

### Project

- Normative specification, simplicity skill, Documentation and Performance Gates,
  contribution guide, security policy, code of conduct, GEP process.

## [0.6.0] — 2026-04-11 (as Flang)

- Web IDE (Monaco editor, live preview, undo/redo, visual screen builder and flow editor).
- Integration functions (payments, messaging, utilities) and optional AI provider functions
  (OpenAI, Claude, Gemini) callable from logic; project templates; new field types.

## [0.5.1] — 2026-04-10 (as Flang)

- Evoticket example with business logic and integrations; sidebar navigation fix; import
  after block keywords.

## [0.5.0] — 2026-04-10 (as Flang)

- Frontend rewritten with Tailwind CSS; login and registration screens.
- Security, performance and async work; multilingual keywords.
- Blog, CRM, e-commerce and Whaticket examples; installer.

## [0.2.0] — 2026-04-10 (as Flang)

- First tagged release: declarative full-stack applications from `.fg` files (models,
  screens, events), REST API, SQLite/MySQL/PostgreSQL, WebSocket, rate limiting and
  role-based access.

[Unreleased]: https://github.com/flaviokalleu/germanio/compare/v0.6.0...HEAD
[0.6.0]: https://github.com/flaviokalleu/germanio/compare/v0.5.1...v0.6.0
[0.5.1]: https://github.com/flaviokalleu/germanio/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/flaviokalleu/germanio/compare/v0.2.0...v0.5.0
[0.2.0]: https://github.com/flaviokalleu/germanio/releases/tag/v0.2.0
