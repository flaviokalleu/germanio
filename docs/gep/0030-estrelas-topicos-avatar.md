# GEP 0030: Marks by people (`recebe estrelas`), one item of a list as a filter, the address of a file

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (GitLab stress test, inventory PR-06 stars, topics, avatar); decision by the
  maintainer
- **Level:** 1
- **Layer:** domain (one phrase, the sibling of `recebe aprovações`), core (marks, counts,
  filters, file addresses), adapter (external names only)

PR-06 needed three small things. Each is described on its own; they share this GEP because they
came from the same inventory item, not because they depend on each other.

## Part A — marks by people

### Problem

Stars on a repository, favourites in a shop, likes on a post, bookmarks on an article, "follow"
on a ticket: each person marks a record once, can take the mark back, the record shows how many
marks it has, and each person can list what they marked. As ordinary data this needs a pair that is
unique per person and record (Germanio has no "unique per" yet), a count kept in sync (no
aggregates outside pages), and the rule "you may only mark what you see". Written by hand it is a
join table, a counter, races and a leak waiting to happen.

### Proposal

```text
projetos
    recebe estrelas
```

Flat form: `projeto recebe estrelas`. The noun is the name of the marks, in the plural (`recebe
favoritos`, `recebe curtidas`).

### Semantics

- **Who.** Any signed-in person who sees the record marks it (`marcar`) and takes their own mark
  back (`desmarcar`). No access rule is needed: seeing is the rule. Someone who does not see the
  record gets 404, someone not signed in 401.
- **Once.** A person marks a record at most once. Marking again, or taking back a mark that does
  not exist, changes nothing and answers 304 (not modified).
- **Count.** The record shows the number of marks in a field named after them (`estrelas`), kept
  by Germanio in the same transaction as the mark. Nobody sets it; lists never count rows.
- **Mine.** `?marcados=sim` lists only the records the person marked (up to 10 000 of them),
  always inside what the person sees.
- **Private.** Who marked what is never shown to anyone else. Marks are application state, not
  domain data: they go away with the record and with the person (the counts follow).
- **Not a change of the record.** Marking does not edit the record: an archived (read-only) record
  may be marked, and editing rights are not needed.
- **One kind per data.** A data receives one kind of marks; a second `recebe` with another noun is
  an error (the actions would be ambiguous). A program action named `marcar` or `desmarcar` on a
  data with marks is an error too. Marks need `tenha login`.
- **Address.** `POST …/<registro>/marcar` and `…/desmarcar`; the vocabulary may rename them
  (`marcar é "star"`). Pages show "Marcar" or "Desmarcar", whichever applies.

### Amendment: who marked, and the marks of a person

- **Who marked a record is not listed** (GitLab `GET /projects/:id/starrers`). GitLab shows it
  to anyone; here it would reverse the rule above ("who marked what is never shown to anyone
  else"), which is the safe default for likes, favourites and bookmarks. The adapter does not
  offer the route (404). Making marks public would need an explicit opt-in phrase decided by the
  maintainer (a draft idea, not implemented: `recebe estrelas públicas`); nothing changes
  silently.
- **The marks of a person** (GitLab `GET /users/:id/starred_projects`) are answered only to that
  person (`?marcados=sim` through the adapter); asking for someone else's is refused (403), and
  an unknown person is 404.

### Alternatives studied

- **Ordinary data (`estrelas` › `tem dono`, `pertence a projeto`).** Needs uniqueness per pair and
  a count, both missing; would add a public list of who starred; the GitLab endpoints would still
  need an adapter.
- **A list of people on the record (like `recebe aprovações`).** Approvals are few and shown; marks
  can be many thousands, are private, and must be listed per person: a list column cannot answer
  "what did I mark" without reading every record.
- **Do nothing.** Every reference application (code forge, shop, social) has this intent.

## Part B — one item of a list as a filter

`permita filtrar por topico`, where `topicos lista de texto` is a field, filters by one item of the
list (`?topico=go` keeps the records whose `topicos` contain exactly `go`, never `golang`). The
filter is named by the singular of the list (the same singular rules as data names). Before, the
only way was `?topicos=go`, which reads as "the topics are go". As every filter, it narrows what
the person already sees. Several items (`?topico=go,web`) keep the records with all of them
(GEP 0043).

## Part C — the address of a file

A record with a file (`avatar imagem`, GEP 0014) also shows `<campo>_endereco`: where the file is
downloaded on the surface that answered (`/api/v4/projects/1/avatar`, `/_ge/api/projetos/1/avatar`),
prefixed by `GERMANIO_URL_PUBLICA` when it is set. Only when there is a file; downloading still
follows the record's rules. Clients no longer need to build the address themselves (the GitLab
adapter calls it `avatar_url`).

## Performance and security

A mark is one insert or delete on a unique index plus one counter update, in the change's
transaction; concurrent marks are serialized by the database. Lists read the stored count.
`?marcados` reads the person's marks once, bounded. Nothing about who marked leaves the server. The
item filter is the existing parameterized `contem` on the list's stored form. The file address is
computed, never stored, and never reveals the place on disk.

## Tests

`runtime` `TestCopiasDeReceitas` (favourites in a recipes domain: once per person, 304, counted,
listed per person, buttons on the page, count not editable, marks removed with the record; filter by
one tag; `foto_endereco`), `TestMarcasErros` (no login, two kinds, name taken, no plural);
`examples/gitlab-foss/e2e` `TestEstrelas` (star/unstar, `star_count`, 304, private project, not
signed in, `?starred=true`, archived, count not editable, deleted person), `TestTopicos` (`topics`
list, `?topic=`, exact item, visibility kept, editing), `TestAvatar` (upload, `avatar_url`, list,
download, who may change, private project, never set as text, removal).
