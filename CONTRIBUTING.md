# Contributing to Germanio

Thank you for helping. Germanio is a small project with a strong idea: people describe
intent, Germanio implements mechanisms. Contributions in English or Portuguese are equally
welcome (the language keywords and most internal docs are in Portuguese).

## Before you start

Read these three documents. They are short enough, and they decide most review questions:

1. [`docs/INTENCAO.md`](docs/INTENCAO.md): the normative specification of the language
   (the intent layer, the hierarchical syntax, efficiency).
2. [`skills/germanio-simplicity/SKILL.md`](skills/germanio-simplicity/SKILL.md): the
   simplicity checklist every `.ge` file and every new construction must pass.
3. [`AGENTS.md`](AGENTS.md): the Documentation Gate and the Performance Gate. They apply to
   humans too.

The one rule that surprises most contributors: **when `.ge` cannot express something
simply, do not implement the application rule in Go.** Find the generic capability that is
missing, implement it in the core (`compiler/`, `runtime/`), expose it simply to `.ge`, test
it, and then simplify the application. Product-specific protocols belong to adapters
(`integracoes/`), never to the core.

## Development setup

Requirements: Go (the version in [`go.mod`](go.mod)). No C compiler: SQLite is pure Go.

```bash
git clone https://github.com/flaviokalleu/germanio.git
cd germanio
go build -o ge ./cmd/ge      # the Germanio tool: new, run, check, fmt, explain, test
go build -o germanio .       # the older CLI (build, docker, ide), also reachable as `ge legado`
go test ./...
```

## Making a change

1. **Small units.** One logical change per commit, with a message in the style of the log
   (`feat(parser): …`, `fix(security): …`, `docs(normative): …`).
2. **Green before commit.** `gofmt -l .` prints nothing, `go vet ./...` and `go test ./...`
   pass; run `go test -race` on the packages you touched. Never commit something you know is
   broken.
3. **Tests prove behavior.** A bug fix comes with a test that fails without the fix. A new
   capability comes with a generalization test (it must serve another domain, not only the
   example that motivated it).
4. **Documentation in the same change.** If the meaning of the language changes, the
   specification changes in the same unit of work. If code and specification disagree, do
   not silently pick one: record it in [`GERMANIO_GAPS.md`](GERMANIO_GAPS.md).
5. **Performance.** If you touch a hot path (parser, resolver, server, database), run the
   relevant benchmarks in [`bench/`](bench/README.md) before and after.

## Language changes: GEPs

Changes to syntax, semantics, the runtime contract, the standard library or a capability
go through a short **Germanio Evolution Proposal** ([`docs/gep/`](docs/gep/README.md)).
Bug fixes, typos and performance work that does not change meaning do not need one.

## Reporting

- Bugs and ideas: [open an issue](https://github.com/flaviokalleu/germanio/issues/new/choose).
- Security problems: **do not open a public issue**; see [`SECURITY.md`](SECURITY.md).
- A confusing error message is a bug. Report it with the `.ge` that produced it.

## Code of conduct

Participation is governed by the [Code of Conduct](CODE_OF_CONDUCT.md).
