# crud

A customer register with no login: create, list, edit and delete, with validation,
search, a filter and a paginated page.

## What it demonstrates

- Fields with rules: `nome obrigatório até 120` (required, at most 120 characters),
  `email obrigatório e único` (required, valid e-mail, unique, normalized to lower case).
  Errors come back per field, in Portuguese (`{"message": {"email": ["já está em uso"]}}`).
- Types inferred from the name: `email`, `telefone` (phone), `observações` (long text).
- `permita` › `pesquisar` (search), `filtrar por cidade` (filter), and create / view /
  edit / delete for anyone, since the app has no login.
- A page that shows 10 customers per page (`mostre clientes` › `10 por página`).

## Run

```bash
go build -o ge ./cmd/ge
./ge check examples/crud/app.ge
./ge rodar examples/crud/app.ge 8080
```

Open http://localhost:8080/clientes. From a script:

```bash
curl -X POST localhost:8080/_ge/api/clientes -H 'Content-Type: application/json' \
     -d '{"nome":"Ana","email":"ana@example.com","cidade":"Recife"}'
curl 'localhost:8080/_ge/api/clientes?q=ana'
curl 'localhost:8080/_ge/api/clientes?cidade=Recife'
```

**Capabilities:** data, validation, uniqueness, search, filters, pagination, pages.

**Keywords are Portuguese** (`tem` = has, `obrigatório` = required, `único` = unique,
`permita` = allow). See [the index](../README.md#a-note-on-language).
