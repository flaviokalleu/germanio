<p align="center">
  <img src="assets/germanio.png" alt="Germanio mascot: a dark crystal with a terminal face" width="260">
</p>

<h1 align="center">Germanio</h1>

<p align="center">
  <strong>An open-source, intent-oriented programming language built in Go<br>
  for building real applications with simple, readable and deterministic code.</strong>
</p>

<p align="center">
  <a href="https://github.com/flaviokalleu/germanio/actions/workflows/ci.yml"><img src="https://github.com/flaviokalleu/germanio/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <img src="https://img.shields.io/badge/version-0.7.0--dev%20(pre--release)-2fb4ff" alt="version 0.7.0-dev, pre-release">
  <img src="https://img.shields.io/badge/built%20with-Go-2fb4ff" alt="built with Go">
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-2fb4ff" alt="MIT license"></a>
</p>

<p align="center">
  <a href="#show-me">Show me</a> ·
  <a href="#try-it">Try it</a> ·
  <a href="examples/">Examples</a> ·
  <a href="#documentation">Documentation</a> ·
  <a href="#project-status">Status</a> ·
  <a href="CONTRIBUTING.md">Contributing</a> ·
  <a href="README.pt-BR.md">Português</a>
</p>

You describe **what exists, who may do what, what happens and what appears**. Germanio turns
that description into a running web application: database, validation, login, permissions,
state changes, a REST API and server-rendered pages.

**No AI inside.** The compiler and the runtime are deterministic: the same `.ge` file always
means the same program, and `ge explain` shows every fact it inferred and where it came from.
AI tools can help you write or review `.ge` code, but Germanio never needs them to understand
or run it.

## Show me

This is a complete application. Keywords are in Portuguese (see [Project status](#project-status)):
*tem* = has, *pertence a* = belongs to, *começa* = starts, *pode* = can,
*acesso* = access, *seus* = their own, *página* = page, *mostre* = show, *permita* = allow.

```ge
crie sistema Tarefas

usuarios
    tem
        nome obrigatório
        email obrigatório e único
        senha min 8

tenha login
tenha cadastro
login usa email

tarefas
    tem
        titulo obrigatório até 120
        descrição
        prazo data
    pertence a usuario
    começa aberta
    pode
        concluir
        reabrir
    acesso
        usuario
            criar seus
            ver seus
            editar seus
            concluir seus
            reabrir seus
            excluir seus

página Tarefas
    mostre tarefas
    permita
        pesquisar
        cadastrar
        editar
        excluir
```

What is below belongs to what is above: `tarefas › acesso › usuario › ver seus` means *a user
may see their own tasks*. `ge run app.ge` starts the application: sign-up and login pages
with sessions and CSRF protection, a task list with search, forms with validation, the
*concluir* (complete) and *reabrir* (reopen) transitions, a REST API, and a database created
and migrated automatically. Nobody can see or change someone else's tasks.

`ge explain tarefas app.ge` shows what Germanio understood, including the fields it adds by
itself (excerpt):

```text
Campos:
  titulo                 texto, obrigatório, até 120
  prazo                  data
  usuario_id             referência a usuario, obrigatório
  estado                 texto, começa com aberta, mantido pelo sistema
  concluida_em           texto, mantido pelo sistema
Estados: começa aberta
  concluir → concluida (registra concluida_em e concluida_por_id)
  reabrir → aberta
Quem pode:
  ver            o dono/autor
  administrador  tudo
De onde vem cada fato (frase plana equivalente — origem):
  tarefa tem titulo obrigatório até 120 — app.ge:15 (tarefas › tem)
  tarefa pode concluir — app.ge:21 (tarefas › pode)
```

## Why Germanio exists

Building an ordinary web application still requires knowing HTTP, SQL, an ORM, migrations,
sessions, password hashing, CSRF, authorization checks, templates and a frontend framework,
before writing a single rule of the actual product. Germanio moves all of that into the
language implementation, where it is written once and tested, with safety as the default
rather than an add-on ([what is covered and what is still open](SECURITY.md)), and keeps
the program at the level of the product: data, people, permissions, states and pages.

The guiding rule: **Go builds mechanisms; Germanio builds products.** When something cannot be
said simply in `.ge`, the answer is a new generic capability in the core, never application
code in Go. To test that idea against something large, the repository contains a
reimplementation of core [GitLab](examples/gitlab-foss) concepts in `.ge` (groups, projects,
Git hosting, issues, merge requests, CI jobs run by the official `gitlab-runner`).

## Who it is for

- People who are not programmers and want to describe a real application precisely.
- Developers who want full-stack applications without framework boilerplate, and who can
  inspect everything the compiler inferred.
- Anyone interested in language design: an intent-oriented, indentation-based,
  deterministic language with a normative specification and a documented evolution process.

## Try it

Requirements: [Go](https://go.dev/dl/) (the version in [`go.mod`](go.mod)) and Git. No C
compiler and no database server: SQLite is embedded and pure Go.

```bash
git clone https://github.com/flaviokalleu/germanio.git
cd germanio
go build -o ge ./cmd/ge

./ge new my_app              # app.ge, backend/, frontend/
./ge run my_app/app.ge       # http://localhost:8080
```

Or install with the script (clones and builds into `~/.local/bin`):
`curl -fsSL https://raw.githubusercontent.com/flaviokalleu/germanio/master/install.sh | bash`.
To run an application in Docker, see the [Dockerfile](Dockerfile).
Prebuilt binaries will be attached to the next [release](https://github.com/flaviokalleu/germanio/releases)
(the releases v0.2–v0.6 predate the rename and are published as *Flang*).

| Command | What it does |
| --- | --- |
| `ge new <name>` | Creates an application: `app.ge`, `backend/` (what exists, who may do what) and `frontend/` (pages) |
| `ge run [app.ge] [port]` | Checks and runs (port 8080 by default) |
| `ge check [app.ge]` | Checks without running; errors say what, where, why and how to fix |
| `ge explain <data> [app.ge]` | Everything Germanio knows about a piece of data, and where each fact comes from |
| `ge fmt [path] [--check]` | Rewrites files in the one canonical form; refuses any change of meaning |
| `ge test [path]` | Runs `.ge` tests of core programs |
| `ge --version` | Prints the version |

`ge help` lists everything, including the core commands (`novo`, `explicar`) and the
experimental Project Intelligence commands.

## Examples

| Example | What it shows |
| --- | --- |
| [examples/](examples/) | The index of runnable examples, each with a short README |
| [examples/gitlab-foss](examples/gitlab-foss) | The large stress test: GitLab concepts in `.ge` |
| [examples/germanio](examples/germanio) | Programs in the strict typed core (no web application) |

## Backend and frontend

An application created by `ge new` separates *what exists and who may do what*
(`backend/`) from *what appears* (`frontend/`); `app.ge` imports both. Pages use the data and
permissions already declared: a page never repeats the schema and never checks roles by hand.

## Project status

Germanio is **pre-1.0 and under active development**. What works today, with tests:

- the intent layer for web applications (data, relations, validation, states, login,
  sign-up, roles, membership and inheritance, permissions and visibility, search, filters,
  pages, REST API for integrations, Git hosting, remote job executors);
- the hierarchical syntax and its equivalent flat form;
- `ge check`, `ge explain`, `ge fmt` and educational error messages;
- the strict typed core for terminal programs, with tests and coverage ([SPEC.md](SPEC.md)).

Known limits, recorded in [GERMANIO_GAPS.md](GERMANIO_GAPS.md):

- **The intent layer is in Portuguese only.** Programs written with English keywords are
  not understood yet (an older technical syntax accepted several languages; it is kept for
  compatibility but is not the default level).
- Pages do not yet have indicators, charts or real-time updates.
- No language server, package manager or published VS Code extension yet (the extension in
  [vscode-germanio](vscode-germanio) can be installed from source).
- The language may still change before 1.0; changes of meaning go through
  [GEPs](docs/gep/README.md).

## Documentation

For people learning Germanio:

- [Getting started](docs/getting-started.md) and the [language tour](docs/language-tour.md)
- [Examples](examples/)
- [Comparisons with Python, Go and JavaScript/TypeScript](docs/comparisons.md)

For developers of Germanio:

- [Normative specification of the intent layer](docs/INTENCAO.md) (Portuguese), including
  the grammar of the hierarchical syntax
- [Specification of the strict core](SPEC.md)
- [Architecture](docs/ARCHITECTURE.md): lexer, parser, resolved model, runtime
- [Tests](.github/workflows/ci.yml) (`go test ./...`) and [benchmarks](bench/README.md)
- [GEPs](docs/gep/README.md), [roadmap](docs/ROADMAP.md), [changelog](CHANGELOG.md),
  [known gaps](GERMANIO_GAPS.md)
- [Research notes](docs/research/) behind the design decisions

## Benchmarks

The permanent suite in [bench/](bench/README.md) compares Germanio with the same application
written directly in Go, and records hardware, OS, Go version, commit and command with every
result. Germanio makes no speed claims without such a reproducible comparison.

## Contributing

Issues, ideas and pull requests are welcome, in English or Portuguese. Start with
[CONTRIBUTING.md](CONTRIBUTING.md). Security problems: see [SECURITY.md](SECURITY.md).
Everyone is expected to follow the [Code of Conduct](CODE_OF_CONDUCT.md).

## License

[MIT](LICENSE) © 2026 Flavio Kalleu.
