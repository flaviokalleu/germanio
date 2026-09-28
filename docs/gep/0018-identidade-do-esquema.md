# GEP 0018: Schema identity — human, canonical and physical names

- **Status:** Rascunho — AGUARDANDO DECISÃO (nothing implemented)
- **Author:** agent (G112); decision by the maintainer
- **Gaps:** G112 (and the table-level part of G93)
- **Level:** none for the author in the common case (no new syntax); an explicit command for
  migrations
- **Layer:** compiler (canonical names), core (storage identity), tooling (`ge check`,
  `ge explain`, the migration command)

## Problem

Today one string is three things at once: the singular the resolver computes from the name
the author wrote (`contas bancarias` → `contas_bancaria`) is also the name of the table, the
prefix of foreign-key columns in other tables (`contas_bancaria_id`), and a value written into
rows (`recurso = 'contas_bancaria'` in memberships, pending items and history). The singular
rule is wrong for noun + adjective (`contas_bancaria`, `notas_fiscal`), but fixing it would
silently point every existing program at new, empty tables and columns. And, worse, it would
orphan memberships: people would lose their roles in the renamed data, a security problem as
much as a data one.

## Evidence

Where the singular becomes stored identity (audit of 2026-09-28):

| Place | Example | Code |
| --- | --- | --- |
| table name | `CREATE TABLE contas_bancaria` | `runtime/banco` (model name = singular) |
| foreign keys in children | `contas_bancaria_id` | `compiler/parser/resolver.go` (`to.Singular + "_id"`) |
| unique indexes | `uq_contas_bancaria_numero` | `runtime/banco` |
| values in rows | `recurso = 'contas_bancaria'` (memberships, pending items, history) | `runtime/servidor/intencao.go`, `pendencias.go`, `historico.go` |
| task payloads | `entidade: 'contas_bancaria'` (webhook deliveries in the queue) | `runtime/servidor/eventos.go` |
| file storage | `arquivos/contas_bancaria/…` | `runtime/servidor/arquivos_registro.go` |

## Proposal

Three names with separate lives:

| Name | What it is | Decided by | May change? |
| --- | --- | --- | --- |
| **human** | what the author writes and people read: `contas bancárias`, label "Conta bancária" | the source | freely |
| **canonical** | the language's identity for the data and its fields: `conta_bancaria` | the resolver: a pure function of the source (the singular rule, or `singular`) | with the language (a new rule version) |
| **physical** | the storage identity: table, columns, indexes, stored `recurso` values, file folders | recorded once in the database, never recomputed | only by an explicit migration |

- **Schema record.** The database keeps a small table (`_germanio_esquema`) mapping each
  canonical data and field to its physical name, with the rule version that produced it.
- **Resolution at start** (deterministic given the database):
  1. the canonical name is in the record → its recorded physical name is used;
  2. not in the record, but a table exists with the name an **earlier rule** would have given
     → that table is adopted and recorded (existing projects keep their data, with no action);
  3. neither → the physical name is the canonical one, and it is recorded (new projects get the
     correct singular).
- **Stored identity uses the physical name** everywhere in the table above. The canonical and
  human names are what the language, `ge explain`, pages and APIs show.
- **Migration is explicit.** `ge migrar nomes` (a tooling command, not syntax) shows what would
  change and, when asked, renames tables, columns, indexes, stored `recurso` values and folders
  to the canonical names in one transaction, then updates the record. `ge check` reports every
  divergence and the command to align it.
- **The singular rule gets a version.** Version 2 inflects noun and adjectives when there is no
  preposition (`contas bancarias` → `conta_bancaria`, `notas fiscais` → `nota_fiscal`,
  `tokens de acesso` unchanged). It is generic, with no word-specific cases; `singular` still
  overrides it.
- `ge explain contas_bancarias` shows the three names and where each came from, for example:
  `canônico conta_bancaria (regra de singular v2) · físico contas_bancaria (adotado da regra v1
  em 2026-10-02; alinhar com: ge migrar nomes)`.

## Semantics and errors

- Two data whose physical names would collide: error with both origins.
- A canonical name that changed between versions of the program (a rename) is found by the
  record, not guessed: the record says which physical table held it. This is also where G93's
  `renomeie` becomes cheaper: renaming a field can update the record instead of the column.
- Without a database (`ge check` on a fresh project), the physical names are the canonical ones.

## Alternatives studied

- **Fix the rule and accept the break:** refused by the maintainer.
- **Keep the old rule forever:** keeps wrong names in every new project.
- **Physical name always equal to the plural as written:** removes inflection from storage, but
  still ties storage to spelling (renaming the data renames the table).
- **Migrations as files (Rails, Django):** powerful, but a new artefact type for the common case.
  The record plus an explicit command keeps the common case invisible.

## Evaluation

The author learns nothing new in the common case. The explicit step exists only when a stored
name must change, which is exactly when intent must be explicit.

## Performance and security

One small read at start. Memberships and every stored `recurso` keep pointing at the same
physical identity, so a language change can never strip people of their roles.

## Compatibility and migration

Existing databases are adopted by rule 2 with no action. Programs keep their meaning. The
change of the singular rule affects only canonical names (what `ge explain` shows) until the
project runs the explicit migration.

## Tests to write when accepted

A database created with rule v1 opened by a program under rule v2 keeps its data and
memberships; a new database gets v2 names; `ge migrar nomes` renames everything or nothing; the
colliding case; `ge explain` output.
