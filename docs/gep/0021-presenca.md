# GEP 0021: Presence

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (FASE 2, obstacle 3 of the Conversa); decision by the maintainer
- **Level:** 1
- **Layer:** domain (one phrase), core (connection tracking, notices)

## Problem

Chat, collaboration and support tools show who is around. Building it by hand means heartbeats,
timeouts, a store of connections and broadcasting changes: infrastructure the author of a `.ge`
should not see.

## Proposal

```text
tenha presença
```

## Semantics

- A person is **online** while they have at least one page of the application open (the live
  subscription of GEP 0020 is the signal; no extra request).
- A person becomes **offline** after their last page closes and a short grace period has passed
  (default 10 s, so a reload or a short drop does not flicker). A connection that dies without
  closing is noticed by the transport's heartbeat.
- The people data shows `online` (true/false). It is never stored: it is state of the running
  application, not of the record.
- Who sees it: whoever may see the person (the people data's own rules).
- A change of presence is a change for GEP 0020: the pages showing people refresh.
- Without `tenha presença`, nothing changes: no `online`, no tracking.

## Alternatives studied

- A stored `ultimo_acesso` field: a write per page view, and still not presence.
- Explicit heartbeat requests from pages: duplicates the live subscription.

## Limits

One process: several servers need shared presence (FASE 4). "Typing…" is not covered (it is
ephemeral per conversation, a later GEP).

## Tests

`TestPresenca` (online while a page is open; not online for an absent person; offline after the
grace period; the people page is told of both changes; without the phrase, no `online`).
