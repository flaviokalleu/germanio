# GEP 0017: Mentions create pending items

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (GitLab stress test, inventory NT-01); decision by the maintainer
- **Level:** 1
- **Layer:** domain (one word in an existing section), core (finding mentions)

## Problem

GEP 0009 gives a pending item to people placed in a field (the assignees). People are also
called into a conversation by name: `@ana, pode revisar?`. GEP 0009 left mentions out because
they needed a notion of mention first.

## Evidence

GitLab inventory NT-01 (todos on mention). Chat, ticketing and document tools do the same.

## Proposal

```text
issues
    pendência para
        responsaveis
        mencionados

comentarios
    pendência para
        mencionados
```

Flat form: `issue gera pendência para responsaveis e mencionados`.

## Semantics

- `mencionados` means the people written as `@<nome de usuário>` in the record's texts (long
  text fields). The name of a person is the login data's unique `username` (or `usuario`,
  `apelido`, `login`); without one, the error says so.
- A person newly mentioned (on create, or added on edit) receives one pending item, the reason
  saying it was a mention. Whoever writes does not get one for themselves.
- **Only people who may see the record are notified.** Mentioning someone outside a private
  project or a confidential issue creates nothing (no pending item, no e-mail): a mention never
  reveals a record to someone who could not read it.
- Removing a mention does not remove the pending item (it happened).
- A data field named `mencionados` keeps its own meaning (a people field), as before.

## Errors

`mencionados` in a data without long text, or an application without a username field: the
error says what is missing.

## Evaluation

One word, the one people already use.

## Performance and security

One lookup per distinct mentioned name, only on create and edit; names are matched exactly
(no search). A mention of someone who cannot see the record is dropped.

## Tests

`TestMencoes` (a mention creates one pending item; the author does not get one; a repeated
mention on edit does not create another; a person outside a confidential record gets nothing),
the block/flat equivalence, errors.
