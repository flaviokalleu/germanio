# GEP 0020: Pages stay up to date

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (FASE 2, obstacle 2 of the Conversa); decision by the maintainer
- **Gaps:** G66 (the old hub sends everything to everyone and drops in silence)
- **Level:** none: no syntax. It is what a page does
- **Layer:** core (change notices, recipients, transport), pages (partial refresh)

## Problem

A page shows what was true when it was drawn. In a chat, a board or any shared list, someone
else's change stays invisible until the person reloads. Frameworks make the author pick a
transport (WebSocket, SSE, polling), subscribe, decide who receives what and patch the
interface by hand. The mandate says the intent is only "this information must stay up to date",
and for a page that shows data that is always the intent.

## Proposal (no syntax)

```text
página Canais
    mostre canais
```

already means: while someone is looking at it, the page follows the changes of what it shows.

## Semantics

- **What a page depends on:** the data it shows (the chain of its address), the data shown under
  a record (its children), and the data its indicators count.
- **Every change counts:** create, edit, delete, transitions, and the writes the core makes by
  itself (pending items, history). A change is announced **after its commit**; an undone change
  announces nothing.
- **Who is told:** only someone who may see the changed record, before or after the change, by
  the same rules as the list. A record the viewer cannot see changes silently for them.
- **What is sent:** nothing about the record. The notice only says "what you are looking at
  changed". The page then asks for itself again with the viewer's own session, so all data goes
  through the ordinary rules; even a wrong notice cannot leak data.
- **Partial refresh:** in tables (the list, sections of children) the notice carries the row
  already drawn for the viewer, with the ordinary rules, and the page inserts, replaces or removes
  only that row. Other live regions (indicators, details), pages with search or filters, and a
  viewer whose queue of rows is full get a refresh of their live regions. Each table knows the
  parent record it belongs to, so a change elsewhere does not reach it. What the person is typing is not touched: while a form field has focus,
  the refresh waits.
- **Bounded:** many changes in a row become one refresh (notices coalesce). A slow viewer never
  holds memory or slows the others; if the connection drops, the page reconnects and refreshes
  once, so nothing is missed.
- **Transport is the runtime's** (today Server-Sent Events, which browsers reconnect by
  themselves); the program never names it.

## Alternatives studied

- **An explicit phrase per page (`atualizada`):** a choice nobody would turn off, so a word with
  no information. Rejected for level 1; an opt-out can come later if a real case needs it.
- **Sending the changed record:** faster, but every message would need its own authorization and
  projection, a second place for the rules. Refetching reuses the one place.
- **Polling:** simpler, but costs constant requests and adds delay.

## Performance and security

- One check per (change, interested viewer). Large audiences will need the check cached per
  record (FASE 4); the benchmark of the phase measures it.
- Notices carry no data, the subscription requires seeing the page, and the refresh uses the
  viewer's session.

## Tests

`TestPaginasVivas`:
- a viewer of a list is told about a new record;
- a viewer who cannot see the record is not;
- an undone change tells nobody;
- writes made by the core count;
- coalescing;
- the subscription needs access to the page;
- the page carries the live regions and the script.
