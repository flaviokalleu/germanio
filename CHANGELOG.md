# Changelog

All notable changes to Germanio. The format follows [Keep a Changelog](https://keepachangelog.com/),
and versions follow [Semantic Versioning](https://semver.org/). Germanio is **pre-1.0**:
minor versions may still change the language, and changes of meaning go through
[GEPs](docs/gep/README.md).

The project was called **Flang** (files `.fg`) until 2026-09-27. Releases up to v0.6.0 were
published under that name.

## [Unreleased]

### GitLab and identity

- E-mail confirmation (`tenha confirmação de e-mail`, GEP 0031, em teste): no sign-in until the
  address is confirmed; single-use tokens.
- Two-factor authentication (`tenha autenticação em dois fatores`, GEP 0032, em teste): TOTP with
  the secret sealed by AES-GCM under `GERMANIO_SEGREDO`, hashed single-use recovery codes.
- Public keys (`chave pública` field type, GEP 0032) with fingerprint; GitLab `/user/keys`.
- Repository archive download (`repository/archive`), under `baixar_codigo`.
- Copies of a record (`acesso` › papel › `copiar`, GEP 0029, em teste): GitLab fork, with the
  repository cloned, never more visible than the original.
- Marks by people (`recebe estrelas`, GEP 0030, em teste), filtering by one item of a list
  (topics), and `<campo>_endereco` for files (avatar).
- A minimum of approvals before an action (`precisa de 2 aprovações para mesclar`, GEP 0026,
  em teste); the author's approval does not count. The GitLab example does not require approvals,
  like GitLab FOSS.
- Ways to merge (GEP 0027, em teste, no syntax): squash, semi-linear, linear (fast-forward after
  rebase), and merging when the executions of the source pass.
- Git LFS over HTTP (GEP 0035, em teste, no syntax): batch API, sha256 and size checked while
  streaming, `GERMANIO_LFS_MAX_MB`, same rules as clone and push.
- Repository mirrors (`espelhos espelham o repositório do projeto`, GEP 0036, em teste): push and
  pull mirrors over http(s) only, through the SSRF-protected client, credentials hidden.
- Issues: weight, time estimate and spent time, links between issues, moving an issue to another
  project (`pode mudar de projeto`, GEP 0034, em teste).
- Adapters may ask the application itself as the caller (`superficie.pedir`, `superficie.contar`,
  GEP 0033, em teste): GitLab events, `/application/statistics`, time tracking.
- Git over SSH (GEP 0037, em teste, no syntax; `GERMANIO_SSH_ENDERECO`): public-key only, the
  person found by the key's fingerprint, only `git-upload-pack`/`git-receive-pack`, the same
  authorization and after-push path as smart HTTP, connection limits and timeouts.
- Sign-in with an external account (`tenha login com conta externa`, GEP 0039, em teste):
  OpenID Connect code flow with PKCE, state and nonce, ID token verified by JWKS (RS256/ES256)
  with the standard library only; linking needs the e-mail verified on both sides; two factors
  still asked.
- Pages offer what the API already did: merge with squash or when the executions pass, cancel a
  scheduled merge, move a record to another parent, copy with a new name and path, and the
  approvals still missing; a reference to the same kind never offers the record itself.
- GitLab API: `/projects/:id/forks`, `/users/:id/starred_projects` (oneself only), fork into a
  `namespace_path`, events `action` filter, `PUT …/merge`, `GET …/approvals`, mirror `sync`
  (generic `atualizar_agora`), `?topic=a,b`. Several values in one filter (GEP 0043) and a
  place given by its address (GEP 0044), both em teste. `/starrers` stays refused (GEP 0030).
- Totals and sums of a record's children (`indicadores` › `soma do peso das issues`, GEP 0047, em
  teste): one grouped query, only what the viewer may see, live on pages; `zerar <nome>` resets a
  sum atomically.
- Unique pairs (`única por par de issues`, GEP 0048, em teste): no self-link, no repeated pair in
  either order, also enforced by a unique index.
- Per-person choice of notification e-mail (`avisos_por_email`, amendment to GEP 0013).

### Security

- Secrets kept encrypted at rest (GEP 0049, em teste, no syntax: a hidden text field): mirror
  credentials, webhook tokens, CI secret variables and the two-factor secret, with AES-256-GCM,
  keys derived from `GERMANIO_SEGREDO`, rotation through `GERMANIO_SEGREDO_ANTERIOR`, migration of
  old plain rows at startup; production refuses to start without a key.
- A sum that never goes below zero (`não pode ficar com tempo gasto negativo`, GEP 0050, em
  teste), checked in the same transaction with the parent locked.

### Pages

- Presentation pages in Portuguese (GEP 0059, em teste): `página "/"`, `navegação`, `capa`,
  `seção`, `cartão`, `etiqueta`, `rodapé`, accents accepted; `fundo` (image or `claro`), `imagem`,
  `ponto`, 1 and 4 column grids, code panels in sections; only same-application images; no
  sideways scroll on phones. The project's landing page (`examples/landing/`) is written with it.

### Tests


- The examples smoke test runs from a temporary directory, so the legacy WhatsApp example no longer
  leaves `whatsapp.db` inside the repository.

## [0.7.0] — 2026-09-29

The first release as Germanio: the rename, a strict typed core, the intent layer and the
hierarchical syntax.

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
- Repositories list, create and remove tags under the same rules as code.
- Execution graph in the native run file: `precisa` (start when the steps needed finish), branch
  rules (`somente_em`, `exceto_em`, `regras`), `quando: sempre`, artifacts handed to the steps that
  depend on them (local and remote executors) and `artefatos_expiram_em`. GitLab `needs`, `rules`,
  `only/except`, `dependencies` and `expire_in` translate to them (CI-08, CI-09).
- Isolated steps: with `GERMANIO_EXECUTOR=docker`, a step that names an image runs all its commands
  in one container with its variables, no network, no capabilities, no privilege escalation, a
  process limit and the host user (before: one container per line, without the variables).
- Editing a file on the web (API and the file page) commits as the person, under the rules of
  pushing code; executions start as after a push.
- Boards (GEP 0023, in test): `cartoes por estado` under `mostre` shows a data in columns by
  state; moving a card is the transition, offered as buttons and by dragging, only to people who
  may perform it. Section titles use the data's own plural (no more "Cartaos").
- Reading (GEP 0022, in test): `guarda leitura` — opening a container reads it; containers show
  each viewer their unread count; lists and live pages follow it.
- Presence (GEP 0021, in test): `tenha presença` gives people `online` while they have a page
  open, with a grace period; pages showing people follow it.
- Pages stay up to date (GEP 0020, in test; no syntax): an open page follows the changes of what
  it shows, after each commit, only for people who may see the changed record; only its live
  regions are refreshed.
- Mentions (GEP 0017, in test): `pendência para` › `mencionados` gives a pending item to people
  written as `@username` in the record's texts, only if they may see the record.
- Protected branches named by data (GEP 0016, in test): `somente maintainer pode enviar código para
  as branches protegidas dos projetos`, each record a branch or a pattern with `*`. The `singular`
  section now really names the data (G113).
- Variables of executions (GEP 0015, in test): `pipelines usam as variaveis do projeto` gives
  every step the owner's variables; a `valor oculto` is never returned and is masked in logs
  (by whole lines, so a secret written in two pieces never leaks); the execution's own names win.
- Images pasted or dropped into a formatted text are kept with the record of the page, seen by
  whoever sees it, and inserted as Markdown (UP-01).
- Files of a record (GEP 0014, in test): fields declared `arquivo` or `imagem` (`anexo arquivo`,
  `foto imagem`; a type that comes only from the name keeps storing text) are sent to
  `…/<registro>/<campo>` (streamed, outside the transaction, with a size limit), downloaded only by
  whoever may see the record (images inline, anything else as an attachment) and removed with it.
  Remote work can keep a file (`trabalho_remoto.guardar_arquivo`); the native run file gains
  `artefatos`; GitLab job artifacts come from the official runner (CI-08).
- Routes no longer buffer multipart bodies; a multipart form's fields are no longer in
  `requisicao.corpo` (G110). Appending to a log measures bytes, so an accented chunk no longer
  stalls a job's log (G109).
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
- The SSRF guard recognizes `localhost` and the short IPv4 spellings (`127.1`, `2130706433`,
  `0x7f.1`) itself, before resolving, so the answer no longer depends on the system resolver.
- Pages agree in gender with the data ("Nova tarefa", "Tarefa criada", "Novo pedido") and show
  dates as dd/mm/aaaa and moments as dd/mm/aaaa hh:mm; forms keep the browser's format.

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

[Unreleased]: https://github.com/flaviokalleu/germanio/compare/v0.7.0...HEAD
[0.7.0]: https://github.com/flaviokalleu/germanio/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/flaviokalleu/germanio/compare/v0.5.1...v0.6.0
[0.5.1]: https://github.com/flaviokalleu/germanio/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/flaviokalleu/germanio/compare/v0.2.0...v0.5.0
[0.2.0]: https://github.com/flaviokalleu/germanio/releases/tag/v0.2.0
