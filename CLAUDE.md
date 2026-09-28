# CLAUDE.md

## Leitura obrigatória e autoridade

Leia `AGENTS.md`, `docs/INTENCAO.md` e `skills/germanio-simplicity/SKILL.md` antes de trabalhar.
A camada de intenção é normativa; `docs/README.md` distingue guias técnicos e histórico.
Contagens e descrições antigas abaixo não provam o estado atual: confira código e testes.


This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is Germanio

Germanio is an intent-oriented declarative programming language written in Go that generates full-stack web applications from `.ge` files. The default level is the intent layer (`docs/INTENCAO.md`), which is **Portuguese only**. The older technical syntax (`dados`/`telas` blocks, `docs/SPEC.md`) also accepts keywords in 20 languages, normalized to canonical Portuguese tokens.

## Build & Run

```bash
go build -o germanio .

./germanio run demo/plano/inicio.ge [port]
./germanio check demo/plano/inicio.ge
./germanio new <name>          # flat mode (single file)
./germanio init <name>         # organized mode (folders)
./germanio build app.ge -o app # compile to standalone executable
./germanio docker              # generate Dockerfile
```

CGO is disabled — uses pure-Go SQLite (`modernc.org/sqlite`).

## Testing

```bash
go test ./...                 # everything, including the GitLab end-to-end flows
go test -race ./runtime/... ./compiler/... ./tooling/...
scripts/bench.sh              # benchmarks (bench/README.md)
```

Tests live next to each package; `tooling/doctest` checks that every ```ge block in the
public documentation compiles; `runtime/examples_smoke_test.go` starts every example.

## Architecture

Pipeline: `.ge` file → Lexer → Parser/AST → Runtime Engine.

### Compiler (`compiler/`)

- **`lexer/lexer.go`** — Tokenizer with 150+ keywords. For the older technical syntax, maps keywords of 20 languages to canonical Portuguese tokens via `idiomas/idiomas.go` (global map; collisions recorded as G92). The intent layer is Portuguese only.
- **`idiomas/idiomas.go`** — Translation map: foreign word → canonical PT keyword. Supports ES, FR, DE, IT, ZH, JA, KO, AR, HI, BN, RU, ID, TR, VI, PL, NL, TH, SW.
- **`parser/parser.go`** — Recursive descent parser. Handles: `sistema`, `dados`, `telas`, `eventos`, `acoes`, `tema`, `logica`, `banco`, `autenticacao`, `integracoes`, `rotas`, `paginas`, `sidebar`.
- **`ast/ast.go`** — Node definitions including `CustomRoute`, `CustomPage`, `SidebarItem`, theme presets (`ThemePreset()`), color names (`ColorName` map, `ResolveColor()`).

### Runtime (`runtime/`)

- **`engine.go`** — Orchestrator: loads .env, creates DB, sets up auth with JWT from env, wires interpreter with HTTP client, starts hot reload, starts server.
- **`interpreter/interpreter.go`** — Script engine with 30+ built-in functions including async (`paralelo`, `esperar`, `timeout`, `chamar_async`, `consultar_paralelo`), array indexing (`arr[0]`), HTTP calls (`chamar`), JSON parsing.
- **`servidor/servidor.go`** — HTTP server with CRUD endpoints, role-based access control, rate limiting (100 POST/min), SSRF-protected proxy, body size limits, custom routes, custom pages, HTML caching.
- **`servidor/renderizador.go`** — the older SPA renderer: theme CSS variables (the `estilo` field is parsed but not used: G75), Chart.js, FK dropdowns, enum selects, textarea for texto_longo, smart sidebar.
- **`banco/banco.go`** — Database abstraction (SQLite/MySQL/PostgreSQL) with connection pooling, auto-migration, validation rules enforcement, join tables for many-to-many, relationship queries.
- **`auth/auth.go`** — JWT (HMAC-SHA256) + bcrypt with role checking, login rate limiting (5 attempts = 5min lockout).
- **`hotreload.go`** — File watcher that re-execs process on .ge changes.

### CLI (`cli/cli.go`)

Commands: `run`, `check`, `new`, `init`, `build`, `docker`, `version`, `help`.

`germanio build` creates a standalone executable by generating a temp Go project with `go:embed`, compiling the .ge files + runtime into a single binary.

## Key Design Decisions

- **Multilingual (older syntax only)**: 20 languages normalized to canonical PT tokens via `idiomas.go`; the intent layer is Portuguese only (G74, G92).
- **Theme presets**: `tema moderno/simples/elegante/corporativo/claro` — one word for a complete design.
- **Color names**: `cor primaria azul` — the AST resolves names to hex via `ColorName` map.
- **Smart sidebar**: If user defines screens, sidebar shows those; models without custom screens get auto-generated entries.
- **Security**: Auth bypass fixed, SSRF blocked, eval requires admin, XSS escaped, path traversal prevented, uploads whitelisted, body limited, JWT from env, CSV injection protected.
- **Async**: Go goroutines exposed to the scripting engine via `paralelo()`, `timeout()`, etc.
- **Validation rules**: `validar` statements in logic blocks are enforced in `banco.Validar()` on create and update.

## Germanio Simplicity Gate

Antes de criar, modificar ou revisar qualquer arquivo `.ge`, leia obrigatoriamente
`skills/germanio-simplicity/SKILL.md`. Nenhum `.ge` é considerado concluído sem passar pelo
checklist dessa skill. Quando `.ge` não conseguir expressar algo de forma simples: não
implemente a regra da aplicação em Go; identifique a capability genérica faltante,
implemente o mecanismo no Germanio, exponha uma interface simples para `.ge`, teste, volte à
aplicação e refatore o `.ge` obsoleto. A aplicação descreve intenção; Go implementa mecanismos.
Referência da linguagem de intenção: `docs/INTENCAO.md`.

### Três camadas (domínio, core, adaptador)

- **Domínio** (`backend/`, `frontend/` de cada app): intenção, simples para leigos.
- **Core** (`compiler/`, `runtime/`): mecanismos genéricos. Não pode conter nomes, caminhos,
  formatos ou protocolos de sistemas externos (ex.: nada de `/api/v4`, `CI_JOB_ID`, `JOB-TOKEN`).
- **Adaptador** (`integracoes/`): traduz um protocolo externo para capabilities do core; pode
  ser técnico; marcado `NÍVEL AVANÇADO / ADAPTADOR DE COMPATIBILIDADE`; sem regras do produto.

Adaptadores podem conhecer a complexidade do sistema externo. O domínio não.
Detalhes em `skills/germanio-simplicity/SKILL.md` (seção 34b).
