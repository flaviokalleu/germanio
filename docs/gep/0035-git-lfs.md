# GEP 0035: Large files in repositories (Git LFS, no syntax)

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (GitLab stress test, inventory RP-10); decision by the maintainer
- **Level:** 1 (nothing to write: it comes with `tem repositório`)
- **Layer:** core (the Git capability: object store, batch API, transfer links)

## Problem

Repositories with large binary files (images of a book, game assets, datasets, design files) use
Git LFS: the repository keeps a small pointer and the content travels separately, through the
LFS batch API of the server that hosts the repository. Without it, `git lfs push` fails and those
projects cannot be hosted. In frameworks this is a separate service, its own storage, its own
tokens and, too often, its own permission rules that drift from the repository's.

## Evidence

GitLab inventory RP-10 (LFS). Git LFS is an open protocol of Git hosting, spoken by the official
`git lfs` client and every hosting product; any application with `tem repositório` that hosts
real projects needs it.

## Proposal

No new phrase. `X tem repositório` already says "each record has a Git repository served over
HTTP"; LFS is part of serving a Git repository over HTTP. The runtime contract grows:

```text
projetos
    tem
        repositório
```

now also answers `/<chave>.git/info/lfs/objects/batch` and `/<chave>.git/info/lfs/objects/<oid>`.

## Semantics

- **Same people, same rules.** Downloading needs what cloning needs (see the record and, when
  declared, `baixar código`); uploading needs what pushing needs (`enviar código`, a record that is
  not read-only — `arquivado é somente leitura` freezes large files too — and a token scope that
  allows writing). The credentials are the same as git over HTTP: basic authentication with the
  person's access token or password. A running step's token may download from the repository of
  its step, as it may clone it.
- **Batch API** (official specification): `upload` and `download` operations, the `basic`
  transfer, `sha256` objects, at most 1000 objects per request. Upload actions only for objects
  the repository does not have yet; a missing object in a download is a per-object 404; an invalid
  object or one above the limit is a per-object 422; unknown operations, hash algorithms or
  transfers are a 422 for the whole request. Locks are not offered (404); the official client
  goes on without them.
- **Transfer links** carry, in the `Authorization` header the batch gives, a permission signed
  by the server for exactly one repository, object, size and direction, valid for one hour. It is
  signed with a key derived from `GERMANIO_SEGREDO` used for nothing else, so a link is never a
  session or a token, and nothing else is a link. A request without a link is checked like any
  git request.
- **Uploads** are streamed to disk while their SHA-256 is computed; at most the announced size
  plus one byte is read; the object is kept (atomic rename) only when content, oid and size agree,
  otherwise nothing remains (422). Objects never touch the database, so no transaction is held
  open while bytes arrive.
- **Limit:** `GERMANIO_LFS_MAX_MB` (default 100) per object, checked in the batch (422 for that
  object) and in the upload (413).
- **Storage:** under the files root (`GERMANIO_ARQUIVOS`), in `lfs/<repository>/<oid[0:2]>/
  <oid[2:4]>/<oid>`. Each repository has its own objects: having access to one never reveals
  another's content, even with the same oid. Removing the record removes its objects.
- **Links** in the batch answer use `GERMANIO_URL_PUBLICA` when set, otherwise the address of the
  request.

## Errors

Every refusal is a JSON body with `message`, which the `git lfs` client shows: missing
credentials (401, with `LFS-Authenticate`), not visible (404), not allowed or read-only (403),
too large (413/422), content that does not match (422), expired or forged link (401).

## Alternatives studied

- **A phrase (`repositório aceita arquivos grandes`):** a beginner should not need to know that
  large files travel differently; the protocol is part of hosting Git, as smart HTTP already is.
- **Reusing the files of records (GEP 0014):** those belong to one record field; LFS objects
  belong to the repository, are addressed by content and spoken by an external client.
- **Do nothing:** projects with large files cannot be hosted.

## Performance and security

Streaming both ways (`http.ServeContent`, with ranges), constant memory per transfer, one record
lookup per request, no database work for the bytes. Per-repository storage avoids cross-project
leaks through content addressing. Pending: a quota per repository or per person (today only the
per-object limit), and garbage collection of objects no commit points to.

## Tests

`runtime/git/lfs_test.go` (`TestLFSStore`: wrong hash, short and long content and objects above
the limit leave nothing; repositories are separate; invalid repositories and oids refused;
`TestLFSBatch`: the shape of the protocol), `runtime/lfs_espelhos_test.go` (`TestGitLFS`, a
domain without GitLab: anonymous 401 with `LFS-Authenticate`, outsider 404, reader cannot upload,
author uploads, wrong content 422, a link used for another object or direction 401, forged link
401, limit 413/422, reader downloads with the link and with git credentials, read-only record
refuses uploads and keeps downloads, deleting the record deletes the objects) and the GitLab E2E
`examples/gitlab-foss/e2e/lfs_espelhos_test.go`. The official `git lfs` client was not installed
where this was implemented; the tests follow the published batch and basic-transfer specification.
