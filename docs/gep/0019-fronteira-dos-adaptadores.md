# GEP 0019: The boundary between domain and adapters

- **Status:** Rascunho — AGUARDANDO DECISÃO (nothing implemented)
- **Author:** agent (G114, G105); decision by the maintainer
- **Level:** 1 for the domain (fewer technical words), advanced for adapters
- **Layer:** norm (three layers), compiler (layer rule, named integrations), core (surfaces)

## Problem

The three-layer norm says the domain describes the product and adapters translate external
protocols. In the GitLab example, protocol knowledge still sits in the domain:

| Line (today, in `backend/`) | What it really is |
| --- | --- |
| `integração em "/api/v4"` | protocol: where the external surface lives |
| `mensagens em inglês` | protocol: the language external clients expect |
| `login aceita tokens de acesso no cabeçalho "PRIVATE-TOKEN"` | **mixed**: tokens give access (product) · carried in that header (protocol) |
| `login aceita oauth por 2 horas` | protocol (an OAuth grant) + a duration |
| `disponibilize atividades para integração como "events"` | **mixed**: offered to external clients (product) · called `events` (protocol) |
| `vocabulário da integração` (106 names) | protocol names |
| `integração` › `nome "projects"` inside each data block | protocol name inside the domain |

Facts from the audit: the current layer rule already accepts the prefix, the vocabulary and the
messages line in `integracoes/`; it refuses the `login aceita …` lines, because the login
declaration mixes both kinds. And one global vocabulary per program cannot serve two protocols
(G105 found `acao` wanted as `action_name` by one API and `event_name` by another).

## Proposal

**The adapter translates; the domain decides.** Stated as a rule that can be checked: removing
an adapter never changes what anyone can do through the application's own pages. It only
removes the external surface.

### What stays in the domain (product decisions)

- what exists, who may do what, states, rules, pages (unchanged);
- **that** a data is offered to external clients: `disponibilize projetos para integração`, or
  the section `integração` with no names;
- **that** access tokens are a way in, and their scopes: `login aceita tokens de acesso`;
- durations and limits that are product policy (how long a session lasts).

### What moves to an adapter (`integracoes/`, marked NÍVEL AVANÇADO)

- the surface and its address: a **named integration**, e.g. `integração gitlab em "/api/v4"`;
- external names: that surface's own vocabulary (`projetos é "projects"`,
  `acao é "action_name"`), so two adapters can name the same thing differently;
- how credentials travel: `tokens de acesso chegam no cabeçalho "PRIVATE-TOKEN"`,
  `oauth senha emite tokens de acesso`;
- the language and exact texts external clients depend on: `mensagens em inglês`, and
  protocol-required texts for the core's refusals (see below);
- routes and logic that translate calls into capabilities (as today).

### What an adapter may never do

Declare data, fields, relations, permissions, roles, states, validations, pages or hooks that
change the product; turn a refusal into acceptance; widen what a token or a person may do.

### Protocol messages are not language messages

The core refuses with stable identities (`recusa.branch_protegida`, `recusa.nao_encontrado`) and
Portuguese texts. An adapter may say how its surface renders a refusal, e.g.
`recusa.branch_protegida é "You are not allowed to push code to protected branches on this
project."`, because Git clients match that text. The text belongs to that surface only: pages
and other adapters keep theirs. The English sentence that today lives in the core
(`runtime/servidor/repositorio.go`) moves to the GitLab adapter.

## Generality

The same shape serves:
- a Stripe webhook adapter (signature header, event names mapped to domain actions);
- GitHub or GitLab APIs;
- Slack and WhatsApp (message formats, phone formats);
- bank files (fixed-width layouts);
- legacy APIs.

Each is a named integration with its own vocabulary. None contains the word of another, and
none contains product rules.

## Semantics and errors

- A protocol declaration in `backend/` or `frontend/`: educational error pointing to
  `integracoes/`.
- A product declaration in `integracoes/`: the current error.
- Two named integrations may translate the same name differently; within one integration, the
  G105 rule (one translation per name) holds.
- `ge explain <dado>` lists, per integration, the external name and address.

## Alternatives studied

- **Relax the layer rule to accept `login aceita …` in adapters:** refused by the maintainer.
  It would let product policy (token scopes, durations) live in adapters.
- **Keep everything in the domain, marked as compatibility:** leaks protocol into the domain.
- **One global vocabulary:** already failed (G105).

## Compatibility and migration

The current forms keep working during a transition, with `ge check` warnings that show the
adapter form. GitLab moves to `integracoes/gitlab_api.ge` as the first user.

## Tests to write when accepted

Layer errors in both directions; two integrations translating one name differently; the
removal test (the pages behave the same without the adapter); the protected-branch message
coming from the adapter and not from the core.
