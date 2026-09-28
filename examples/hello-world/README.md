# hello-world

The smallest Germanio program: one line that prints a greeting.

```ge
mostre "Olá, mundo"
```

This file uses the **strict core** (the general-purpose language described in
[`SPEC.md`](../../SPEC.md)), not the application layer: there is no `crie sistema`, so
`ge rodar` runs it as a program and exits. It does not start a server.

## Run

```bash
go build -o ge ./cmd/ge
./ge check examples/hello-world/app.ge
./ge rodar examples/hello-world/app.ge     # prints: Olá, mundo
```

For more strict-core programs (functions, generics, tests, imports) see
[`examples/germanio`](../germanio).

**Keywords are Portuguese** (`mostre` = show). The intent layer and the examples in this
directory are written in Portuguese; see [the index](../README.md#a-note-on-language).
