# todo

A to-do list where every person manages only their own tasks.

## What it demonstrates

- Sign up and sign in (`tenha login`, `tenha cadastro`, `login usa email`).
- A task **starts open** and **can be completed or reopened** (`começa aberta`,
  `pode` › `concluir`, `reabrir`). Completing records who did it and when
  (`concluida_em`, `concluida_por_id`); reopening clears them. Completing twice is refused.
- Ownership: `usuario` › `criar seus`, `ver seus`, … — each task belongs to its author;
  another person gets "not found", never someone else's task.
- Search and a filter by state (`permita` › `pesquisar`, `filtrar por estado`).
- One page (`página Tarefas`) with search, create, edit and delete.

## Run

```bash
go build -o ge ./cmd/ge
./ge check examples/todo/app.ge
./ge rodar examples/todo/app.ge 8080
```

Open http://localhost:8080, create an account, then add tasks. The same operations are
available to scripts under `/_ge/api/tarefas` (for example `POST /_ge/api/tarefas/1/concluir`).

Set `GERMANIO_SQLITE=todo.db` to choose the database file and `GERMANIO_SEGREDO` (32+ bytes)
to keep sessions across restarts.

**Capabilities:** data, validation, login, ownership, states and transitions, search,
filters, pages.

**Keywords are Portuguese** (`tem` = has, `começa` = starts, `pode` = can, `acesso` = access,
`permita` = allow, `página` = page). See [the index](../README.md#a-note-on-language).
