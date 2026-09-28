# GEP 0003: The form contract

- **Status:** Rascunho
- **Author:** agent (research consolidation); decision by the maintainer
- **Gaps:** G75 (form part)
- **Level:** 1 (no syntax change; runtime contract)
- **Layer:** core

## Problem

When a form fails validation, the page redirects with `?erro=…`: what the person typed is
lost, the message is one sentence for all fields and not attached to any field, and the text in
the URL can be forged by anyone who builds the link. `required` is marked only when creating.

## Evidence

`docs/research/frontend/GERMANIO_FRONTEND.md` §4 and "Divergências" (`paginas.go`: the
redirect, `message` joining field errors, `required` only on create, `?ok=` in the query
string). WCAG 2.2 criteria 3.3.1 (error identification), 3.3.7 (redundant entry) and 4.1.3
(status messages). The API already returns errors per field
(`{"message": {"email": ["já está em uso"]}}`); only the page layer flattens them.

## Current state

`runtime/servidor/paginas.go` posts to the API internally, and on failure redirects with the
joined message.

## Alternatives studied

SvelteKit form actions (`fail(422, data)` re-renders the form with the values); React's
`useActionState`; Rails' re-render of `new` with the invalid record. All converge on: answer
422 and render the same form again with the values and the errors per field.

## Proposal

No syntax changes. The runtime contract becomes:

- On a validation error, the server answers **422** and renders the same form with the typed
  values (never the password), each field's messages next to it (`aria-describedby`,
  `aria-invalid`), and a summary at the top that receives focus.
- On success, redirect to the record (post/redirect/get, as today), with the notice stored in
  the session for one use — never in the URL — and announced with `role="status"`.
- `required`, lengths and ranges are rendered from the field on create **and** edit; the
  server remains the authority.
- The field → control table (`telefone` → `tel`, `data` → `date`, `hora` → `time`, …) is one
  table, tested, and shown by `ge explain`.

## Semantics

A form is a projection of the entity's fields (without hidden, system, immutable or secret
fields); its states (empty, error per field, success) are derived, never declared.

## Impact

Runtime only (`paginas.go` and its templates). `ge explain` gains the controls of each field.

## Performance and security

One render instead of a redirect plus a render. Security improves: no forged notices, no
password echoed back, CSRF unchanged.

## Compatibility and migration

Pages behave better with the same `.ge`. Clients that relied on `?erro=` in the URL (none in
the repository) would change.

## Trade-offs and alternatives

"Do nothing" keeps losing input and failing WCAG 3.3.x. Client-side validation only is
rejected: the server is the authority and pages must work without JavaScript.

## Tests

Per example: a failing create keeps the typed values and shows each message on its field
with status 422; a forged `?erro=` shows nothing; the success notice appears once; `required`
is present on the edit form.
