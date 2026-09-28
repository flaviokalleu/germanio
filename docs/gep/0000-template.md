# GEP NNNN: Title

- **Status:** Rascunho
- **Author:** name (GitHub handle)
- **Discussion:** link to the issue
- **Gaps:** GNN in `GERMANIO_GAPS.md`, if any
- **Level:** 1 (default) · 2 · 3 · 4 (see `docs/INTENCAO.md` › Níveis)
- **Layer:** domain · core · adapter

## Intent

What a person wants to describe, in plain words. No syntax yet.

## Today

How it is written now (the `.ge`), and what is hard, technical or ambiguous about it.

## Proposal

The `.ge` after the change. For block syntax, include the equivalent flat phrases.

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

## Compatibility

What existing `.ge` programs do after the change. If something breaks: the deprecation
period and the migration (`ge fmt` rewriting it when that is safe).

## Alternatives

At least "do nothing", and why each alternative was not chosen.

## Tests

The normative tests that will prove it (and the equivalence test for a flat form).
