# GEP 0014: Files of a record

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (GitLab stress test, inventory CI-08 and UP-01); decision by the maintainer
- **Level:** 1 (no new syntax: `arquivo` and `imagem` are existing field types)
- **Layer:** core (storage, upload, download, limits); domain unchanged

## Problem

The norm's type table already says `foto`, `imagem`, `avatar`, `arquivo` and `anexo` are files
or images, but the intent runtime stores them as plain text: there is no way to send a file,
keep it, or give it back only to whoever may see the record. Job artifacts (GitLab CI-08),
attachments (UP-01), product photos, documents of a CRM — every reference application of the
mandate needs it.

## Proposal (no new syntax)

```text
chamados
    tem
        titulo
        anexo arquivo
        foto imagem
```

A field **declared** `arquivo` or `imagem` is a file of the record. A field whose type only
comes from its name (`foto`, `anexo` alone) keeps storing text, as it always has: turning it into
a file would change what existing programs store, so the intent must be explicit (`ge explain`
says so on such a field).

## Semantics

- **Address.** `PUT …/<registro>/<campo>` sends the file (the body is the file; the name comes
  from `?nome=`); `GET` gives it back; `DELETE` removes it. Pages show a link and a form.
- **Who.** Sending or removing needs the right to edit the record; downloading, the right to see
  it — the record's own rules, confidential records included. Anyone else gets 404.
- **The record shows** `{nome, tamanho, tipo}`; the place on disk is never shown and never
  comes from the client (a JSON create or edit cannot set a file field).
- **Limits.** 25 MB per file by default (`GERMANIO_ARQUIVO_MAX_MB` changes it); the file is
  streamed to disk, never held in memory, and never while the database transaction is open (the
  record is updated after the file is complete; the file takes its place after the commit).
- **Safety.** Images (PNG, JPEG, GIF, WebP) are shown inline; every other type is downloaded
  as an attachment, with `nosniff` and a sandbox, so an uploaded HTML or SVG never runs as the
  application. The type is detected from the content, not trusted from the client.
- **Removal.** Replacing, removing or deleting the record deletes the old file after the
  commit; an undone change keeps the old file and discards the new one.
- Storage is a folder (`GERMANIO_ARQUIVOS`); object storage is a future adapter, not part of this
  GEP.

## Errors

A JSON body that sets a file field: 400 with how to send it. A file over the limit: 413 with the
limit. A field that is not a file: 404.

## Evaluation

No new concept for the author: `arquivo` and `imagem` already are field types.

## Performance and security

Streaming with a limit; no transaction held during upload; content-detected type; attachment
disposition; path never from input.

## Tests

`TestArquivos` (upload, download, who may, confidential record, JSON cannot set it, limit, HTML
served as attachment, replacement deletes the old file, deleting the record deletes it, undone
change keeps the old one).
