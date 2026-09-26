# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is Germanio

Germanio is a multilingual (20 languages) declarative programming language written in Go that generates full-stack web applications from `.ge` files. Users describe their app (models, screens, events, logic) and Germanio produces a running application with REST API, database, auth, and UI. Keywords can be written in Portuguese, English, Spanish, French, German, Italian, Chinese, Japanese, Korean, Arabic, Hindi, Bengali, Russian, Indonesian, Turkish, Vietnamese, Polish, Dutch, Thai, or Swahili — all interchangeable in the same file.

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
go test ./compiler/... ./runtime/interpreter/
```

59 tests across lexer, parser, AST, and interpreter. Test files:
- `compiler/lexer/lexer_test.go` (14 tests)
- `compiler/parser/parser_test.go` (16 tests)
- `compiler/ast/ast_test.go` (9 tests)
- `runtime/interpreter/interpreter_test.go` (20 tests)

## Architecture

Pipeline: `.ge` file → Lexer → Parser/AST → Runtime Engine.

### Compiler (`compiler/`)

- **`lexer/lexer.go`** — Tokenizer with 150+ keywords. Normalizes all 20 languages to canonical Portuguese tokens via `idiomas/idiomas.go` translation map.
- **`idiomas/idiomas.go`** — Translation map: foreign word → canonical PT keyword. Supports ES, FR, DE, IT, ZH, JA, KO, AR, HI, BN, RU, ID, TR, VI, PL, NL, TH, SW.
- **`parser/parser.go`** — Recursive descent parser. Handles: `sistema`, `dados`, `telas`, `eventos`, `acoes`, `tema`, `logica`, `banco`, `autenticacao`, `integracoes`, `rotas`, `paginas`, `sidebar`.
- **`ast/ast.go`** — Node definitions including `CustomRoute`, `CustomPage`, `SidebarItem`, theme presets (`ThemePreset()`), color names (`ColorName` map, `ResolveColor()`).

### Runtime (`runtime/`)

- **`engine.go`** — Orchestrator: loads .env, creates DB, sets up auth with JWT from env, wires interpreter with HTTP client, starts hot reload, starts server.
- **`interpreter/interpreter.go`** — Script engine with 30+ built-in functions including async (`paralelo`, `esperar`, `timeout`, `chamar_async`, `consultar_paralelo`), array indexing (`arr[0]`), HTTP calls (`chamar`), JSON parsing.
- **`servidor/servidor.go`** — HTTP server with CRUD endpoints, role-based access control, rate limiting (100 POST/min), SSRF-protected proxy, body size limits, custom routes, custom pages, HTML caching.
- **`servidor/renderizador.go`** — HTML/CSS/JS SPA renderer with 4 style variants (glassmorphism/flat/neumorphism/minimal), theme CSS variables, Chart.js, FK dropdowns, enum selects, textarea for texto_longo, smart sidebar.
- **`banco/banco.go`** — Database abstraction (SQLite/MySQL/PostgreSQL) with connection pooling, auto-migration, validation rules enforcement, join tables for many-to-many, relationship queries.
- **`auth/auth.go`** — JWT (HMAC-SHA256) + bcrypt with role checking, login rate limiting (5 attempts = 5min lockout).
- **`hotreload.go`** — File watcher that re-execs process on .ge changes.

### CLI (`cli/cli.go`)

Commands: `run`, `check`, `new`, `init`, `build`, `docker`, `version`, `help`.

`germanio build` creates a standalone executable by generating a temp Go project with `go:embed`, compiling the .ge files + runtime into a single binary.

## Key Design Decisions

- **Multilingual**: 20 languages normalized to canonical PT tokens via translation map in `idiomas.go`.
- **Theme presets**: `tema moderno/simples/elegante/corporativo/claro` — one word for a complete design.
- **Color names**: `cor primaria azul` — the AST resolves names to hex via `ColorName` map.
- **Smart sidebar**: If user defines screens, sidebar shows those; models without custom screens get auto-generated entries.
- **Security**: Auth bypass fixed, SSRF blocked, eval requires admin, XSS escaped, path traversal prevented, uploads whitelisted, body limited, JWT from env, CSV injection protected.
- **Async**: Go goroutines exposed to the scripting engine via `paralelo()`, `timeout()`, etc.
- **Validation rules**: `validar` statements in logic blocks are enforced in `banco.Validar()` on create and update.
