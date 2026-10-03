# GEP 0023: Showing data by state (boards)

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (FASE 3, the Quadro); decision by the maintainer
- **Level:** 1
- **Layer:** pages (one line under `mostre`), core (columns from the states, moves as
  transitions)

## Problem

Boards (kanban, sales pipelines, support queues, hiring funnels) show records in columns, one per
state, and moving a card is changing its state. In frameworks this is a drag-and-drop library,
column state, optimistic updates and an API call per move, plus checking by hand who may move
what. In Germanio the states, the transitions and who may perform each one are already declared.
What was missing is a way to say "show them by state".

## Proposal

```text
página Quadros
    mostre quadros
        cartoes por estado
```

and for the page's own data: `mostre cartoes por estado`. The line sits where `20 por página`
already sits: it says how the page shows a data, nothing else.

## Semantics

- **Columns are the states:** the initial state first, then the targets of the transitions in
  the order they were declared.
- **Moving a card is the transition** that leads to that column (`concluir` for `concluido`,
  `reabrir` back to the initial state). Only moves the viewer may make, from the card's current
  state, are offered. The server checks again, as for any action; the board adds no way to grant.
- **Works without a mouse:** every card lists its possible moves as ordinary buttons (keyboard,
  screen readers, no JavaScript). Dragging is an enhancement over the same buttons.
- **Keyboard and undo:** a focused card moves to the next or previous column with the arrow
  keys (when that move is one of its buttons); every move is announced to screen readers; Ctrl+Z
  or the "Desfazer" button undoes the last move by making the move back, which the server checks
  like any other (an undo the person may not make is refused and said so).
- **Live:** moves made by others arrive through the live pages (GEP 0020).
- A data without states is an error that says so.

## Alternatives studied

- **A component (`quadro de cartoes`):** a visual concept in the domain; the board is just
  another way to show a list.
- **Columns from any field (`por responsavel`):** useful, but moving would mean editing a field,
  a different rule; left for later.

## Tests

`TestPorEstado` (columns in order; each card offers only the moves its viewer may make from its
state; a move by button performs the transition; someone without the right sees no move, and the
server refuses a forged one), the parser error for data without states.
