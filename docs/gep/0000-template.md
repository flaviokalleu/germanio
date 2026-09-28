# GEP NNNN: Title

- **Status:** Rascunho
- **Author:** name (GitHub handle)
- **Discussion:** link to the issue
- **Gaps:** GNN in `GERMANIO_GAPS.md`, if any
- **Level:** 1 (default) · 2 · 3 · 4 (see `docs/INTENCAO.md` › Níveis)
- **Layer:** domain · core · adapter

## Problem

What a person wants to describe, in plain words, and what goes wrong today.

## Evidence

Measurements, gaps (`GERMANIO_GAPS.md`), research (`docs/research/`), failing examples.

## Current state

How it is written now (the `.ge`) and what the implementation does (files).

## Alternatives studied

What other languages and systems do, and what was learned (with sources).

## Proposal

Syntax before and after. For block syntax, include the equivalent flat phrases.

## Semantics

Exactly what it means: which facts `ge explain` shows, what the runtime guarantees, which
defaults and limits apply, what happens on conflict. Deterministic, with no "usually".

## Errors

The educational errors it introduces (what, where, why, how to fix) and when a correction
can be offered automatically.

## Evaluation

Against `docs/INTENCAO.md` › *Como avaliar uma sintaxe*: technical concepts required,
cognitive load, repetition without loss of context, predictable hierarchy, clarity,
determinism, fast reading. Include the cost for the machine (efficiency norm).

## Impact

Parser, AST, resolver/compiler, runtime, tooling (`ge fmt`, `ge explain`, `ge check`, the
VS Code grammar).

## Performance and security

The cost for the machine (norm: Eficiência) and the security consequences (defaults).

## Compatibility and migration

What existing `.ge` programs do after the change. If something breaks: the deprecation
period and the migration (`ge fmt` rewriting it when that is safe).

## Trade-offs and alternatives

What this costs, and at least "do nothing" with the reason each alternative was not chosen.

## Tests

The normative tests that will prove it (and the equivalence test for a flat form).
