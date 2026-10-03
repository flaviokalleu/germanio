# GEP 0049: Secrets kept encrypted at rest (no syntax)

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (FASE 4, obstacle 4 / criterion 5); decision by the maintainer
- **Level:** 1 (no new phrase: `oculto` already says it)
- **Layer:** core (the vault, the database layer, the start), no domain change

## Problem

Some values must be **read back** by the application to talk to another system: the password
of a remote mirror, the secret token a webhook sends with each delivery, the value of a CI
variable handed to an execution, the second-factor secret of a person. Unlike passwords
(kept as hashes, never read back), these cannot be hashed. Until now they were stored in clear
(the mirror credential, the webhook token, the CI variables) or with a mechanism of their own
(the TOTP secret, GEP 0032): whoever got a copy of the database file — a backup, a misplaced
dump, a shared volume — got every credential the application uses to reach other systems.
Doing it by hand is a trap: a cipher picked from a blog, a fixed nonce, a key in the source, a
key that can never change, and rows written before the encryption that stay in clear forever.

## Evidence

FASE 4 obstacle 4 ("credenciais de saída guardadas sem cifra; não há mecanismo de segredo
guardado") and criterion 5 ("credenciais de saída cifradas em repouso, com rotação da chave;
nada em texto puro no banco"). In the GitLab reference app: `espelhos.credencial` (GEP 0036),
`webhooks › token oculto`, `variaveis › valor texto oculto` (GEP 0015). Any product that calls
partners has the same need: the API key of a payment provider per shop, the token of a delivery
partner per restaurant, the SMTP password of each tenant.

## Proposal

No new phrase. A field people write that **never appears** (`oculto`) exists only to be used by
the system itself, so it is a secret at rest:

```text
webhooks
    tem
        url obrigatório
        token oculto
```

The capability fields that hold credentials are marked by the capability (the mirror's
`credencial`). `ge explain` shows "cifrado no banco" on each such field.

## Semantics

- **Which fields.** A field with `oculto`, written by people (not maintained by the system),
  of a text type (`texto`, `texto longo`, `url`, `link`, or inferred text), plus capability
  credentials (`credencial` of mirrors). Not sealed: passwords and generated secrets (`senha`,
  `segredo`, already hashes, never read back), hidden fields of other types (a hidden number
  is not a credential), and fields the system keeps for itself (paths, logs, digests).
- **Transparent.** The database layer seals the value on every write (create, edit, update by
  filter) and opens it on every read that returns whole records to the runtime (`filtrar`,
  `buscar`, and everything built on them: the surface, hooks, adapters, deliveries, mirrors,
  executions). The domain and adapters see the clear value exactly as before. The legacy
  readers (old `/api/<modelo>` routes, exports) never return sealed columns at all.
- **Never shown.** As before for `oculto`: never in the API, pages, history, events, webhook
  payloads, copies or `ge explain`; now never in the database file either.
- **Never compared.** Each sealed value is different even for the same text, and comparing it
  would let whoever filters guess it one value at a time. Filtering, searching, ordering,
  grouping or summing a sealed field is refused: at compile time (`permita filtrar … por token`,
  a page filter, `oculto e único`, `oculto` with `índice`, with what to write instead) and at run
  time (the database layer answers with an educational error; filtering by "no value" is still
  allowed).
- **Cipher.** AES-256-GCM from the Go standard library, a random 96-bit nonce per value, the key
  derived with HKDF-SHA256 from `GERMANIO_SEGREDO` under the label
  `germanio/segredos-em-repouso/v1`; the authenticated data is the purpose (`tabela.coluna`), so
  a value copied to another column does not open. No custom cryptography.
- **Format (versioned).** `ge1:<key id>:<base64url(nonce‖ciphertext)>`. The key id (12 hex
  characters) is derived from the secret with HKDF under another label: it names the key without
  revealing it. A value without the prefix is legacy clear text: it is read as it is and sealed by
  the next start.
- **Start: migration without downtime.** Every start brings each sealed column to the current
  key: values in clear are sealed, values sealed with a previous key are sealed again, in batches
  of 500, each row rewritten only if it still holds what was read (compare and swap), so another
  server writing at the same time loses nothing; readers accept both forms meanwhile. It is
  idempotent: a second start changes nothing. In SQLite, after sealing values that were in clear,
  the file is rebuilt once (`VACUUM`, WAL checkpoint) so the old clear text does not survive in
  free pages.
- **Rotation.** `GERMANIO_SEGREDO` is the current key; `GERMANIO_SEGREDO_ANTERIOR` lists previous
  keys (separated by spaces). Restarting with the new key and the old one as previous re-seals
  everything with the new key at start; afterwards the old one can be removed. The same start
  re-seals the second-factor secrets (GEP 0032), which now use the same vault (with the person as
  authenticated data); their first format (`v1:`) is still read and re-sealed.
- **Refusals at start.** A value sealed with a key the server does not have stops the start, with
  what to set. Sealed values with no `GERMANIO_SEGREDO` at all stop the start too (they could not
  be read). A program with sealed fields started in production (`GERMANIO_PRODUCAO=1`) without
  `GERMANIO_SEGREDO` does not start, and the message names the fields and how to create a key.
  In development without a key the values stay in clear with a warning, and the start after a key
  is set seals them.
- **Unreadable value at run time** (a row altered by hand): treated as empty and reported once in
  the log, without its content; never a crash of the listing.

## Errors

Compile time: `oculto e único` (use `segredo` when the value identifies the record), `oculto` with
`índice`, `permita filtrar … por` a sealed field, a page `filtros` with a sealed field. Start: no
key in production; sealed values without a key; values sealed with an unknown key; a key shorter
than 32 characters; `GERMANIO_SEGREDO_ANTERIOR` without `GERMANIO_SEGREDO`. Run time: filtering,
searching, ordering or grouping by a sealed field (`ErrSegredo`, 400 with the reason).

## Alternatives studied

- **A new modifier (`cifrado`, `segredo guardado`)**: one more word a beginner must remember to
  write, and forgetting it leaves credentials in clear. `oculto` already says "only the system
  uses this"; encryption is the safe default behind it.
- **Encrypting the whole database file (SQLCipher, disk encryption)**: protects the file, not
  dumps, logical backups, replicas or PostgreSQL rows; it is an operation choice, complementary.
- **Database-side encryption (`pgcrypto`)**: the key travels to the database server and into its
  logs; not portable across SQLite/PostgreSQL/MySQL.
- **A KMS / Vault**: the right next step for large deployments (envelope encryption with a data
  key per column); the format's key id leaves room for it. Too heavy as the default.
- **Deterministic encryption (to keep filters)**: reveals equal values and allows guessing; the
  use cases (credentials) never need to be filtered.
- **Do nothing:** every credential in clear in every backup.

## Performance and security

One AES-GCM operation per sealed value written or read (microseconds); tables without sealed
fields pay nothing (a map lookup per row read). The start scans only rows not yet under the
current key (`NOT LIKE 'ge1:<id>:%'`), in batches. Limits: the key lives in the server's
environment (whoever has the environment and the database has everything); rotating
`GERMANIO_SEGREDO` also ends the sessions signed with it; in PostgreSQL, old row versions keep
the previous form until the database's own vacuum reuses them; values are bound to their column,
not to their row (a row's value copied to another row of the same column by someone with write
access to the database opens there).

## Tests

`runtime/cofre/cofre_test.go` (round trip, random nonces, purpose bound, altered and truncated
values refused, legacy clear text passes, rotation, unknown key, environment rules).
`compiler/parser/segredos_test.go` (which fields are sealed; `oculto e único`, index, filters
refused). `runtime/segredos_test.go`, a restaurant with no Git and no GitLab:
`TestSegredosGuardados` (ciphertext in the column, the partner still receives the key, nothing
in API/page/history/event/explain, editing keeps it sealed, filter/search/order refused, the raw
database file has no clear text), `TestSegredosMigracaoERotacao` (clear rows from development
sealed at start and gone from the file, idempotent second start, production refused without a
key, rotation with `GERMANIO_SEGREDO_ANTERIOR`, unknown key and missing key refused),
`TestDoisFatoresRotacao` (a first-format TOTP secret is re-sealed under the new key and codes keep
working after the previous key is removed). GitLab end-to-end
`examples/gitlab-foss/e2e/segredos_test.go` (`TestSegredosNoBanco`: mirror credential, webhook
token and CI variable sealed in the database and absent from the file; the mirror still
authenticates to another Git server and the webhook still sends its token).
