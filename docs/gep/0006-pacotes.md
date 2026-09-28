# GEP 0006: Packages — principles now, a manager later

- **Status:** Rascunho
- **Author:** agent (research consolidation); decision by the maintainer
- **Discussion:** none yet
- **Gaps:** none recorded; related: G90 (pay only for what you use), I7 and I16 in
  `docs/research/languages/GERMANIO_LESSONS.md`
- **Level:** 5 (adapters) and project configuration; nothing at level 1
- **Layer:** core (fetching, verification) and adapter

## Problem

Sooner or later someone will want to reuse something written by someone else: an adapter for a
payment provider, a "comments" block, a login flow. Today the only mechanism is copying files.
The question of this GEP is not "how do we build a package manager" but **"when is one needed,
and which rules must it obey so that it never imports the failures of npm, PyPI or Maven?"**

The honest conclusion of the research is that **it is not time yet**. There is no third-party
`.ge` code, no registry, no user outside the repository asking for one, and the capabilities
that other ecosystems get from packages (HTTP, database, auth, Git, e-mail, jobs) are part of
the Germanio core. Building a manager now would add a concept, a file and an attack surface
with no problem to solve.

What this GEP does propose is small: the **principles** any future manager must follow, and the
**format of the lockfile**, fixed in advance so that the first implementation cannot improvise
them. It also proposes one optional first step that does solve a problem today: recording the
Germanio version a project expects.

## Evidence

- There is no package manager (confirmed at `0c12051`): `importar` only includes local `.ge`
  files and folders (`compiler/parser/parser.go:987-1022`; `docs/INTENCAO.md` › Organização do
  projeto).
- Adapters are `.ge` files in `integracoes/` interpreted by the runtime (for example
  `examples/gitlab-foss/integracoes/gitlab_runner.ge`), not Go code. A downloaded adapter
  would therefore run inside the runtime's capabilities, which are not yet a sandbox: `chamar`
  is available to any hook (lesson P18) and, read in the code at `0c12051`, uses an HTTP client
  without the SSRF guard (lesson A23).
- The only supply chain today is the runtime's own Go modules (`go.mod`/`go.sum`). The
  `germanio build` path generates a temporary Go project and failed in the performance audit
  because the generated `go.mod` had no `go.sum` (`docs/research/performance/AUDITORIA.md` §4).
- The project templates (`cli/modelos/organizado/app.ge`) do not record which Germanio
  version they were written for.
- Research: `docs/research/languages/PACKAGE_MANAGEMENT.md` (npm, pip/uv, Cargo, Go modules,
  Maven/Gradle, NuGet, Composer, Julia Pkg, Deno/JSR); `julia.md` (Project/Manifest, `[compat]`);
  `haskell.md` (Cabal vs. Stack; Stackage); `ruby.md` (the lockfile as "the last time you know
  everything worked"); lessons A33, E21, E22, E28, I16, I22.

## Current state

```text
crie sistema Loja

importar "backend"
importar "frontend"
```

Everything a project uses is either in its own folders or in the Germanio binary. Reuse means
copying a folder.

## Alternatives studied

| Ecosystem | Lesson | For Germanio |
|---|---|---|
| npm (left-pad 2016; the self-replicating worm of 2025 described by CISA, Unit 42 and Wiz) | install scripts, a mutable registry, broad publish tokens and build machines with access to cloud metadata are the attack path | **never** run code on install; immutable sources; no secrets during fetch |
| Go modules | minimal version selection (deterministic), `go.sum` plus a public checksum database, nothing runs on download, proxy and offline cache | **the model** |
| Cargo / crates.io, cargo-vet | immutable registry, lockfile, shareable audits | immutability; audits for adapters |
| uv (Python) | one universal lockfile, hashes, "only versions older than N days" | lockfile always written; optional age limit |
| Deno / JSR | no postinstall by default; explicit permissions | permissions declared per adapter |
| Maven/Gradle | "nearest wins": the result depends on the shape of the tree | **avoid** (E22) |
| Haskell | two official managers split the community | **one** mechanism (E28) |
| Julia | `Project.toml` (declared) + `Manifest.toml` (generated); `[compat]` for the language version | declared vs. generated; the language version is part of the lock |

## Proposal

### 1. No package manager now

No command, no registry, no remote `importar`. This part is a decision to record, not code.
The condition to reopen: a real project needs a `.ge` block or adapter that it cannot copy and
that the core should not absorb (lesson I16).

### 2. Principles any future manager must satisfy (normative once accepted)

1. **Nothing runs when something is fetched.** A block or adapter only runs when the
   application declares it, under the runtime's capabilities.
2. **Sources are immutable.** A published version never changes and is never removed; a
   withdrawn version stays downloadable and is marked.
3. **Every fetched file is verified by hash** (SHA-256 of the normalized tree) against the
   lockfile; a mismatch stops the build.
4. **Resolution is minimal version selection**: for each dependency, the lowest version that
   satisfies every requirement. Never "newest" and never "nearest".
5. **The lockfile is always written and always read**; a build without it fails with an
   educational error.
6. **Adapters declare their permissions** (network destinations, files, processes,
   credentials) and the runtime enforces them; `ge explain` lists them.
7. **Blocks cannot change the language**: no new sections, keywords or checks (E2); they only
   contain the same declarations a project could write.
8. **Fetching has no access to secrets** (no environment, no cloud metadata endpoint).
9. **One mechanism**, in the `ge` binary.

### 3. The lockfile format (reserved)

A text file, `ge.lock`, at the project root, never edited by hand, one entry per line, sorted,
so that its diff is readable and its content deterministic:

```text
# ge.lock — gerado por ge; não edite
germanio 0.7.0
adaptador pagamentos https://example.org/pagamentos.git v1.2.0 commit 3f2a…9c sha256 8d1e…41 rede api.example.org
bloco comentarios https://example.org/blocos.git v0.3.1 commit a71b…02 sha256 c0f4…7e
```

Fields, in order: kind (`bloco` or `adaptador`), name, source, version, commit, content hash,
and for adapters the declared permissions. The first line is the Germanio version the project
last passed `ge check` with.

### 4. Optional first step: the language version (solves a problem today)

Before any dependency exists, `ge.lock` could hold only its first line:

Before (today): nothing records the version.

After (proposed; not implemented):

```text
# ge.lock
germanio 0.7.0
```

`ge new` writes it; `ge check` updates it when the project passes with a newer Germanio;
`ge run` warns when the running Germanio is older than the recorded one. It gives the future
deprecation mechanism (lessons P7, P25, I7) a place to know which meaning the project
expects. This step is **optional** and can be split into its own GEP; the alternative is a line
in `app.ge` (like `go 1.22` in `go.mod`), which is visible to the author and therefore adds a
concept to level 1.

## Semantics

- Principles 1-9 apply to any future mechanism that fetches `.ge` from outside the project.
- `ge.lock` is data for the tool, not for the author: `ge explain` shows the provenance of every
  fact that came from a dependency (source, version, hash), next to the file and line.
- With only the version line, the semantics is: the recorded version is informative plus a
  warning when the binary is older; it never changes the meaning of the program by itself.

## Errors

| Situation | Message | Automatic fix |
|---|---|---|
| (future) hash mismatch | what was expected and found; the file did not come from the recorded source; do not edit `ge.lock` by hand | no |
| (future) dependency without a lock entry | run `ge check` to record it | yes, by `ge check`, only when the source is reachable and the hash is recorded for the first time |
| (future) adapter uses a permission it did not declare | name the permission and the adapter line | no |
| running Germanio older than `ge.lock` | this project was last checked with version X; update Germanio | no |
| `ge.lock` edited by hand into an invalid form | the line and the expected form | regenerate |

## Evaluation

- **Concepts for the level-1 author:** none. `ge.lock` is generated. The only new idea is
  "this project remembers which Germanio it used", which a beginner can ignore.
- **Determinism:** MVS and a sorted lockfile make the build a function of the files.
- **Cost:** reading one small file at startup.

## Impact

- **Parser/AST/resolver:** none now. In the future, provenance on facts (already required by
  `INTENCAO.md` › Inspeção).
- **Runtime:** none now.
- **Tooling:** `ge new`, `ge check`, `ge run` (only for step 4); `ge explain` for provenance
  later.
- **Documentation:** `docs/INTENCAO.md` › Organização do projeto would mention `ge.lock` only if
  step 4 is accepted.

## Performance and security

- Doing nothing now keeps the supply-chain surface of `.ge` at zero.
- The principles close, in advance, the paths used in the npm incidents: install scripts,
  mutable versions, secrets during fetch.
- A prerequisite, not part of this GEP: adapters can only be fetched safely once the runtime
  enforces per-layer access (lesson P18) and all outgoing HTTP goes through the SSRF-safe
  client (lesson A23). Until then, a downloaded adapter would be as dangerous as a local one
  written by a stranger.
- Pay only for what you use (G90) is also a supply-chain issue: the binary today embeds
  WhatsApp and three database drivers for every application.

## Compatibility and migration

Nothing changes for existing programs. Step 4, if accepted, adds a generated file to new
projects; existing projects get it on their next successful `ge check`.

## Trade-offs and alternatives

- **Do nothing and record nothing.** The risk is that the first manager is written in a hurry,
  when the first adapter from a third party appears, and copies npm's defaults. Recording the
  principles costs one document.
- **Build a manager now on Go modules** (as `PACKAGE_MANAGEMENT.md` suggests). Rejected for
  now: adapters are `.ge`, not Go; routing them through `go.mod` would require a Go toolchain on
  the user's machine, which the performance audit already found fragile (missing `go.sum`), and
  there is no demand.
- **Blocks by URL import, Deno style** (`importar "https://…"`). Rejected: it puts a network
  location and a version in the domain file, which is a technical detail at level 1.
- **A curated, audited set of adapters** (Stackage, cargo-vet; lesson I22). The most promising
  first distribution form when adapters exist; left to a future GEP.
- **The version as a line in `app.ge`** instead of `ge.lock`. Visible and simple, but adds a
  line a beginner must understand or ignore; left open for the maintainer.

## Tests

For step 4, if accepted: `ge new` writes `ge.lock` with the running version; `ge check` of a
valid project updates it; `ge run` with an older binary warns and does not change behaviour;
the file is byte-identical across runs (determinism). For the principles: none until a
mechanism exists; each principle becomes a test of that mechanism (a fetch that would run a
script is refused; a changed byte fails the hash; MVS picks the lowest satisfying version in a
diamond).
