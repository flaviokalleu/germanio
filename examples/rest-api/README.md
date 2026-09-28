# rest-api

A book catalog made available to other systems. The API is not written by hand: the
`integração` section publishes the data, and the same permissions apply to pages and to
scripts.

## What it demonstrates

- `livros` › `integração` › `nome "books"`: publishes list (with search, filters and
  pagination headers `X-Total` / `X-Total-Pages`), view, create, edit and delete under
  `/api/books`. The fields keep their Portuguese names.
- `usuarios` › `integração`: the signed-in person is available at `/api/usuario`.
- Access keys for scripts: `chaves de acesso` with a generated secret
  (`chave segredo prefixo "bib-"`, stored as a hash and shown only once), an expiry
  (`expira em 90 dias`) and revocation (`revogavel`). `login aceita chaves de acesso` lets a
  script send the key as `Authorization: Bearer …`.
- Everyone may read books; signed-in people create and edit; only administrators delete.

## Run

```bash
go build -o ge ./cmd/ge
./ge check examples/rest-api/app.ge
./ge rodar examples/rest-api/app.ge 8080
```

Create an account at http://localhost:8080/cadastro, then create a key while signed in
(`POST /_ge/api/chaves_de_acesso` with `{"nome": "script"}`) and use it:

```bash
KEY=bib-...   # the "chave" value returned once
curl localhost:8080/api/books
curl -X POST localhost:8080/api/books -H "Authorization: Bearer $KEY" \
     -H 'Content-Type: application/json' \
     -d '{"titulo":"Dom Casmurro","escritor":"Machado de Assis","ano":1899}'
curl 'localhost:8080/api/books?ano=1899'
curl localhost:8080/api/usuario -H "Authorization: Bearer $KEY"
```

External field names (`title` instead of `titulo`) are an advanced compatibility setting
(`vocabulário da integração`, see `examples/gitlab-foss/backend/compatibilidade.ge`); they are
not needed here.

**Capabilities:** integration API, pagination, search, filters, access keys (secret, expiry,
revocation), permissions shared by pages and API.

**Keywords are Portuguese** (`integração` = integration, `segredo` = secret,
`expira em` = expires in). See [the index](../README.md#a-note-on-language).
