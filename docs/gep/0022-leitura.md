# GEP 0022: Reading (unread counts)

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (FASE 2, obstacle 4 of the Conversa); decision by the maintainer
- **Level:** 1
- **Layer:** domain (one phrase, the sibling of `guarda histórico`), core (marks, counts)

## Problem

A chat shows, per channel, how many messages each person has not read yet; an inbox, a ticket
queue or a forum does the same. By hand: a table of last-read marks, a write when the person
looks, a count per row and keeping it fresh — per person, per container.

## Proposal

```text
mensagens
    pertence a canal
    guarda leitura
```

Flat form: `mensagem guarda leitura`.

## Semantics

- The records belong to a container (here, the channel): the data's parent that is not the
  people data.
- **Reading is looking:** when a person opens the container's page, every record in it up to
  that moment is read for that person.
- The container shows, for whoever looks at it, `nao_lidas`: how many of its records arrived
  after the person's last reading, not counting the ones the person wrote. Lists show it next to
  the title.
- New records change the count of their container, so pages listing containers follow it
  (GEP 0020).
- The marks are state of the application per person, not domain data: they are never shown to
  anyone else and are removed with the person.

## Alternatives studied

- A `lida` flag per record and person: a row per message per member.
- Reading by explicit action only: real people expect "I looked, so I read it".

## Limits

Only pages mark reading; an API client has no action for it yet. Counting is one query per
container shown.

## Tests

`TestLeitura` (new records count for others, not for the author; opening the container clears
it; a listing page is told when the count changes; marks are per person).
