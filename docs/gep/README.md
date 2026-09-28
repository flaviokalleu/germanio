# Germanio Evolution Proposals (GEPs)

A GEP is a short, numbered document that proposes a deliberate change to the language and
records the decision. The process borrows the useful parts of Python's PEPs, Rust's RFCs and
Swift Evolution (a template, closed statuses, alternatives required, implementation before
acceptance) and drops what a small project does not need. The study behind it is in
[`docs/research/languages/LANGUAGE_EVOLUTION.md`](../research/languages/LANGUAGE_EVOLUTION.md).

## When a GEP is required

A GEP is required when a change alters, for the same `.ge` program:

- the **syntax** (a new construction, a new section, a new keyword, a removed form);
- the **meaning**, meaning `ge explain` would show a different fact;
- the **runtime contract** (what a declared capability guarantees: limits, security,
  transactions, what an HTTP client of a generated app observes);
- the **standard library** or a **capability** exposed to `.ge`.

A GEP is **not** required for bug fixes (the implementation disagrees with
[`docs/INTENCAO.md`](../INTENCAO.md)), clearer error messages, typos, performance work that
keeps the meaning, or internal refactoring. When in doubt, open an issue and ask.

## Lifecycle

| Status | Meaning |
| --- | --- |
| Rascunho (draft) | written, under discussion; anything may change |
| Em teste (trial) | implemented behind the proposal, tested, not yet normative |
| Aceita (accepted) | decided; the specification is updated in the same change as the implementation |
| Implementada (implemented) | in a release |
| Rejeitada (rejected) | decided against, with the reason; listed below so it is not re-proposed without new arguments |
| Retirada (withdrawn) | the author gave up |
| Substituída (superseded) | replaced by a later GEP |

Rules:

- **Numbers are never reused.** A new GEP takes the next free number.
- **Alternatives are required**, including "do nothing".
- **Implementation before acceptance.** A syntax change is accepted only with a working
  implementation, normative tests and the equivalence test when a flat form exists.
- **The author does not accept their own proposal.** The maintainer decides; an automated
  agent may write and implement a GEP but never marks it accepted.
- **The specification stays normative.** Until a GEP is accepted, `docs/INTENCAO.md`
  governs; a proposal never changes it silently.

## How to propose

1. Open an issue with the *Language proposal* template to discuss the intent.
2. Copy [`0000-template.md`](0000-template.md) to `NNNN-short-name.md` and open a pull request.
3. Evaluate the proposal with the criteria of `docs/INTENCAO.md` › *Como avaliar uma
   sintaxe* and the checklist in the simplicity skill.

## Index

| GEP | Title | Status |
| --- | --- | --- |
| [0001](0001-processo-gep.md) | The GEP process | Aceita |

Pending decisions that will become GEPs are listed in
[`docs/INTENCAO.md` › Pendências da sintaxe hierárquica](../INTENCAO.md#pendências-da-sintaxe-hierárquica)
and in [`GERMANIO_GAPS.md`](../../GERMANIO_GAPS.md).

## Rejected ideas

None recorded yet. Rejected GEPs are listed here with a one-line reason.
