# Germanio architecture

How a `.ge` file becomes a running application, which package does what, and where the
known architectural debts are. The *why* behind the design is in
[`docs/research/`](research/) (non-normative); the *what* of the language is in
[`docs/INTENCAO.md`](INTENCAO.md) (normative) and [`SPEC.md`](../SPEC.md) (the strict core).

## Pipeline

```text
                         ┌──────────────── application dialect (intent layer) ────────────────┐
 .ge files ──► lexer ──► parser ──► ast.Intent ──► resolver ──► ast.App ──► runtime ──► HTTP server
  (+ imports)            layout tree      facts        merge, infer,   resolved     database, auth,
                         + sections      (flat         check, report   model        API, pages, Git,
                         reduced to      phrases)      conflicts                    remote work
                         facts
                         └───────────────── strict core (SPEC.md) ──────────────────┐
 .ge programs ──► strict lexer ──► strict parser ──► typed checker ──► AST evaluator (terminal programs)
```

There are two front-ends today. The **strict core** (`SPEC.md`) is a small typed language
for terminal programs; `ge` tries it first. When a file is not a strict-core program, `ge`
treats it as an **application**: the application dialect, whose default level is the intent
layer.

### Application front-end

1. **Lexer** (`compiler/lexer`) — tokens with positions; indentation becomes structure; a tab
   in indentation is an error. The older technical syntax also maps keywords from several
   languages to the canonical Portuguese ones (`compiler/idiomas`); the intent layer is
   Portuguese only.
2. **Parser** (`compiler/parser`) —
   - `parser.go`: top-level dispatch; the older technical blocks (`sistema`, `dados`, `telas`…);
     any unknown top-level line in an intent program is an error with a suggestion.
   - `hierarquia.go`: the hierarchical syntax. `layoutTree` builds the tree of lines with a
     stack of open levels (the off-side rule); a data block is a name at the left margin whose
     children are *sections* from a closed table (`tem`, `pertence a`, `começa`, `pode`,
     `regras`, `acesso`, `permita`, `integração`…); each section is reduced to the equivalent
     flat phrases.
   - `intencao.go`: the flat phrases (`tenha clientes`, `cada cliente tem …`,
     `usuario pode criar pedidos`…) → `ast.Intent`, the list of declared facts with positions
     and hierarchical context.
3. **Resolver** (`compiler/parser/resolver.go`) — `ast.Intent` → `ast.App`, the resolved
   semantic model: entities with singular/plural, fields and inferred types, relations and
   parents, states and transitions, roles, memberships and inheritance, grants, visibility,
   invariants, integration names. It merges blocks of the same data, reports conflicting values
   with both origins, and applies safe defaults (for example the login lock).
4. **Imports and projects** (`runtime/engine.go`: `Compilar`, `resolveImports`) — `importar`
   pulls whole folders (`backend/`, `frontend/`); `integracoes/` may only hold adapters.

### Runtime

`runtime/engine.go` loads `.env`, opens the database, wires the interpreter and starts the
server. It executes the resolved model directly (there is no code generation step):

| Package | Responsibility |
| --- | --- |
| `runtime/banco` | SQLite (pure Go), MySQL, PostgreSQL; automatic migration; validation; paginated queries; transactions |
| `runtime/servidor` | HTTP server: the intent API (`/_ge/api/...`), server-rendered pages (`paginas.go`, `html/template`), identity (`identidade.go`: login, sign-up, sessions, CSRF, tokens, lockout), authorization, remote work (`trabalho_remoto.go`), transactions per request, SEO files for page sites, the older SPA renderer |
| `runtime/interpreter` | the tree-walking interpreter for hooks (`antes de`, `quando`) and logic, and the generic operations the server calls (`Can`, `Level`, `Transition`…) |
| `runtime/git` | Git repositories: smart HTTP, verified before references are updated |
| `runtime/auth`, `runtime/httpclient`, `runtime/jobs`, `runtime/cron`, `runtime/email`, `runtime/whatsapp` | older-dialect authentication, outgoing requests with SSRF protection, background jobs, schedules, e-mail, messaging |
| `runtime/germanio` | the evaluator of the strict core |

The core contains mechanisms only. Names, paths and formats of external systems live in
adapters (`integracoes/` of an application), never in `compiler/` or `runtime/`.

### Tooling

| Package | Command |
| --- | --- |
| `tooling/gecli` | the `ge` CLI (`cmd/ge`); `ge legado` reaches the older CLI (`cli/`) |
| `tooling/formatter` | `ge fmt`: one canonical form; re-parses and refuses any output whose meaning (the whole program without positions) changed |
| `tooling/explicar` | `ge explain` and the warnings of `ge check`: every fact with its flat phrase and origin |
| `compiler/diagnostics` | located diagnostics with codes (strict core) and the four-part educational format |
| `tooling/intelligence` | Project Intelligence (`ge init`, `graph`, `eject`), experimental |
| `vscode-germanio` | VS Code extension: a TextMate grammar generated from word lists, snippets, themes |

## Tests

- `go test ./...` runs unit tests per package, the normative tests of the intent layer
  (`runtime/*_test.go`), the equivalence tests between block and flat forms
  (`compiler/parser/hierarquia_test.go`), formatter idempotence over every `.ge` in the
  repository, and a smoke test that starts every example.
- `examples/gitlab-foss/e2e`: end-to-end flows of the GitLab stress test, including real
  `git` and, when `GITLAB_RUNNER_BIN` is set, the official `gitlab-runner`.
- `bench/`: benchmarks against a Go-direct baseline ([bench/README.md](../bench/README.md)).
- CI (`.github/workflows/ci.yml`) also runs `-race`, checks and formats every example.

## Known architectural debts

Found by the [language study](research/languages/GERMANIO_LESSONS.md) and recorded in
[`GERMANIO_GAPS.md`](../GERMANIO_GAPS.md):

- **Two front-ends** (strict core and application dialect) with separate lexers and parsers,
  and no shared test of equivalence.
- **Reduction to flat phrases**: blocks are turned into synthesized token sequences and
  re-parsed. It made block/flat equivalence true by construction, but positions have no end,
  errors are sometimes reported on the synthesized phrase, and indented lines the table does
  not expect can be dropped (G67, G69).
- **Several readers of the syntax**: the parser, the formatter's own level stack and the
  VS Code grammar's word lists. The recommended direction is one lossless tree of lines that
  every tool reads, and a language server in the same binary
  ([TOOLING.md](research/languages/TOOLING.md)).
- **Diagnostics as text**: application errors are strings; structured diagnostics with ranges
  and verified suggestions are the next step ([DIAGNOSTICS.md](research/languages/DIAGNOSTICS.md)).
- **Performance** has a benchmark suite but no budgets yet; the audit of the execution path is
  in [`research/performance/`](research/performance/).
