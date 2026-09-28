# GEP 0001: The GEP process

- **Status:** Aceita (2026-09-28, requested by the maintainer)
- **Author:** maintainer
- **Level / layer:** not applicable (process)

## Intent

Language changes must be deliberate, discussed and recorded, and not improvised inside a
feature. Until now, decisions were spread over `AGENT_STATE.md` (D1–D11), the research
notes and `GERMANIO_EVOLUTION.md`, without the alternatives or the compatibility impact, and
gap IDs were accidentally reused.

## Proposal

The process described in [README.md](README.md): when a GEP is required and when it is
not, closed statuses, numbers never reused, alternatives required, implementation before
acceptance, and the author never accepting their own proposal.

## Alternatives

- **Do nothing**: keep the Documentation Gate alone. It decides *how* a change is made, but
  not *whether* a change of meaning should happen, and it keeps no record of rejected ideas.
- **Adopt the Rust RFC or Swift Evolution process unchanged**: both assume a team, review
  managers and working groups; they are too heavy for the project's size.

## Compatibility

Earlier decisions stay where they are (`AGENT_STATE.md` › IMPORTANT_DECISIONS,
`docs/research/sintaxe-hierarquica.md` › Parte 3). They are not rewritten as GEPs; new
decisions are.
