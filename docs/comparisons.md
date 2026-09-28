# Germanio compared with Python, Go and JavaScript/TypeScript

These comparisons explain where Germanio sits, not which language is better. Python, Go and
JavaScript/TypeScript are mature general-purpose languages with large ecosystems; Germanio is
a young, pre-1.0 language with a narrow purpose. They are complementary more often than they
are alternatives: Germanio itself is written in Go.

## The short version

| | Germanio | Python | Go | JavaScript / TypeScript |
| --- | --- | --- | --- | --- |
| Kind | intent-oriented, declarative | general purpose | general purpose, systems and services | general purpose, the language of the browser |
| You write | what exists, who may do what, what happens, what appears | instructions, using libraries and frameworks | instructions, with explicit types and error handling | instructions, using libraries and frameworks |
| A web app needs | the language only | a framework (Django, Flask, FastAPI…) plus a database layer and templates | the standard library or a framework, plus SQL or an ORM | a server framework, a database layer and a frontend framework |
| Types | inferred from names and declarations; shown by `ge explain` | dynamic, optional hints | static | dynamic (JS) or static, structural (TS) |
| Maturity | pre-1.0, one main application tested (the GitLab stress test) | decades, huge ecosystem | over a decade, large ecosystem | decades, the largest ecosystem |
| Keywords | Portuguese (intent layer) | English | English | English |

## Germanio and Python

**Philosophy.** Python optimizes for readable instructions ("readability counts") and lets
you build anything by combining libraries. Germanio optimizes for readable *descriptions*:
the program says what the application is, and the language implementation decides how to
store, validate, authorize and render it.

**Syntax.** Both use indentation to show structure. In Python, indentation delimits blocks of
statements; in Germanio it delimits *context* (`tarefas › acesso › usuario › ver seus`), and
each level has a fixed role taken from a closed table, so there is exactly one reading.

**Abstraction.** A Python web application states its data model, its routes, its permission
checks and its templates separately, and a framework connects them. In Germanio these are
one description; permissions are never checked by hand in a page.

**Use cases and trade-offs.** Python covers data science, scripting, automation, machine
learning and web back ends. Germanio covers data-centric web applications (registers,
workflows, teams and roles, portals, internal tools). Anything outside that — numerical
work, arbitrary algorithms, a custom protocol — is Python's ground, not Germanio's.

## Germanio and Go

**Philosophy.** Go keeps the language small and puts power in the tooling and the standard
library (`gofmt`, `go vet`, `go test`). Germanio borrows that attitude — one canonical format
(`ge fmt`), one binary for every tool — and goes further in the other direction: the program
contains no mechanisms at all.

**Abstraction.** In Go you write the HTTP handlers, the SQL, the session handling and the
authorization checks, with full control and full responsibility. Germanio's runtime is Go
code that implements those mechanisms once, generically; a `.ge` program never sees them.
The cost of that abstraction is measured, not assumed: the benchmark suite compares a
Germanio application with the same application written directly in Go
([bench/](../bench/README.md)).

**Use cases and trade-offs.** Go is the right tool for services, infrastructure, CLIs and
anything where control over concurrency, memory and protocols matters. Germanio is for when
those concerns should be someone else's problem; when an application needs a mechanism
Germanio lacks, the answer is a new generic capability in Germanio's Go core.

## Germanio and JavaScript/TypeScript

**Philosophy.** JavaScript and TypeScript are the languages of the web platform; modern
applications combine a component framework (React, Vue, Svelte…) in the browser with a
server framework and an API between them. Germanio describes the application once and
renders pages on the server; there is no client framework to learn, no build step and no
hydration.

**Abstraction.** In a JavaScript stack, the schema usually appears several times (database,
API types, form validation, UI). In Germanio a page shows data that is already declared, and
forms, validation messages and the actions a person may take come from that declaration.

**Use cases and trade-offs.** JavaScript/TypeScript can build any interface, including
highly interactive ones (editors, real-time collaboration, rich visualizations). Germanio's
pages are server-rendered and currently have no indicators, charts or real-time updates
(see [project status](../README.md#project-status)); an interactive frontend is JavaScript's
ground today.

## Maturity, stated plainly

Germanio is pre-1.0: the language may still change (through [GEPs](gep/README.md)), there is
no package manager or language server yet, and the intent layer only understands Portuguese
keywords. Its strongest evidence is the test suite and the GitLab stress test in
[examples/gitlab-foss](../examples/gitlab-foss), not years of production use.
