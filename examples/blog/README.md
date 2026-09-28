# blog

Authors write posts; readers comment. Drafts are visible only to the person who wrote them.

## What it demonstrates

- `posts` › `tem` › `autor`: the author is whoever creates the post; only the author edits or
  deletes it (`acesso` › `autor` › `editar`, `excluir`).
- `posts` › `tem` › `comentarios`: comments belong to a post and are created under it
  (`usuario` › `comentar`, i.e. `POST /_ge/api/posts/<id>/comentarios`). Each comment has its own
  author, who alone may edit or delete it.
- Drafts: `rascunho começa com verdadeiro` and `regras` › `rascunho pode ser visto por` ›
  `autor`. A draft is "not found" for everyone else, even in lists; the author publishes it by
  turning `rascunho` off.
- Everyone, even without an account, reads published posts and comments.
- Search and a filter by author; a page with 10 posts per page.

## Run

```bash
go build -o ge ./cmd/ge
./ge check examples/blog/app.ge
./ge rodar examples/blog/app.ge 8080
```

Open http://localhost:8080/posts.

Known limitation: a draft is a yes/no condition, not a state with a `publicar` (publish)
action, because restricted visibility only works on conditions today
([G79](../../GERMANIO_GAPS.md)).

**Capabilities:** login, ownership, nested data, restricted visibility, search, filters,
pagination, pages.

**Keywords are Portuguese** (`regras` = rules, `pode ser visto por` = can be seen by,
`começa com verdadeiro` = starts as true). See [the index](../README.md#a-note-on-language).
