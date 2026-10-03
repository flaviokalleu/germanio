# GEP 0031: E-mail confirmation

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (GitLab stress test, inventory ID-06, rest); decision by the maintainer
- **Level:** 1
- **Layer:** domain (one phrase), core (mechanism)

## Problem

An application that lets anyone sign up trusts the e-mail typed at sign-up: it sends notices
there, recovery links (GEP 0008), maybe invoices. Nothing proves the address belongs to the
person. Someone can sign up with another person's address, spam people from the application,
or keep an account under an address they do not control. Done by hand, confirmation brings the
usual traps: links that never expire, links stored in clear, links that confirm a different
address than the one they were sent to, a resend form that reveals which accounts exist.

## Evidence

GitLab inventory ID-06 ("Recuperação de senha / confirmação de e-mail"): recovery was done in
GEP 0008, confirmation was missing. Devise `:confirmable`, Django-allauth
`ACCOUNT_EMAIL_VERIFICATION = "mandatory"` and Laravel `MustVerifyEmail` all solve the same need.

## Current state

`tenha cadastro` creates the account and signs the person in at once; `tenha recuperação de
senha` already sends e-mail (the mailer is configured outside the source).

## Alternatives studied

- **Sign in allowed, a flag `email_confirmado` false, the product restricts what it wants**
  (Laravel `MustVerifyEmail` on chosen routes, Django-allauth "optional"). Rejected as the
  default: every product would have to remember to restrict, and forgetting is silent; the
  restriction would need new syntax (which actions need a confirmed address) before it is safe.
- **Sign in refused until confirmed** (Devise `:confirmable` without the grace period,
  Django-allauth "mandatory"). Chosen: the safe default needs no other rule. Nobody acts with
  an address they do not own; the only cost is one click before the first sign-in.
- **A grace period** (Devise `allow_unconfirmed_access_for`): more states to explain; it can
  come later as configuration if a product needs it.
- **Signed stateless links**: cannot be used exactly once nor revoked; a table is simpler.
- **Do nothing.**

## Proposal

One phrase next to the other login phrases:

```text
tenha login
tenha cadastro
tenha confirmação de e-mail
login usa email
```

## Semantics

- **Who must confirm:** people created by sign-up (`/cadastro`). People created by an
  administrator, the `administrador inicial`, and people who existed before the phrase was
  declared are confirmed (the system field `email_confirmado` starts `verdadeiro`), so adding
  the phrase never locks anyone out.
- **After sign-up:** the account exists, no session is started, and a link goes to the
  address typed. The answer says so (`201`, with `email_confirmado: false` and a message).
- **Until confirmed:** signing in with the password is refused with an educational message
  (`403`, "Confirme seu e-mail…"), shown only to whoever typed the right password (a wrong
  password still gets the same `401` as an unknown login). OAuth with the password and HTTP
  Basic with the password (Git clients) are refused too.
- **The link:** a random token (32 bytes), stored only as its SHA-256, valid for **24 hours**
  and **once** (claimed in the transaction that confirms, so the same link opened twice at once
  confirms once), tied to the address it was sent to: if the account's e-mail changed in the
  meantime, the link confirms nothing. Confirming deletes every link of the person. The link
  uses the declared public address (`GERMANIO_URL_PUBLICA`), never the request's `Host`.
  Opening it shows a page with a button: a mail scanner fetching the link confirms nothing.
- **Another link:** `/reenviar-confirmacao` answers the same whether or not the account exists
  or is already confirmed; at most one e-mail per account every 2 minutes; requests count in
  the login limit per address, and the e-mail leaves off the answer's path.
- **Changing the e-mail** (an ordinary edit of the person) sets `email_confirmado` to false and
  sends a link to the new address. Sessions and access tokens already issued keep working; the
  next sign-in with the password waits for the confirmation.
- **Nobody sets `email_confirmado` by hand:** it is a system field, ignored on input, shown
  only to the person and to administrators.
- **Without e-mail configured** (`GERMANIO_SMTP_*` or `GERMANIO_CORREIO_PASTA`, plus
  `GERMANIO_URL_PUBLICA`) sign-up stays closed (`503`, with the reason) and the start log says
  why: an application never creates accounts nobody can confirm.

## Errors

The phrase implies the login. Declaring it when the people have no e-mail field is an error
that shows the line to add (`email obrigatório e único`). Another wording
(`tenha confirmação de telefone`) is an error that shows the valid phrases.

## Evaluation

One phrase, no technical concept for the author. Tokens, hashing, expiry, single use under
concurrency, enumeration and the address binding live in the core.

## Impact

Parser: the phrase. AST/resolver: a flag on the login and the system field
`email_confirmado`. Runtime: two answers (`/confirmar-email`, `/reenviar-confirmacao`), two
pages, one internal table, the existing mailer and the effects-after-commit mechanism (an e-mail
change sends its link only if the edit is kept). `ge explain` shows the confirmation.

## Performance and security

One insert and one e-mail per sign-up or request, limited per address and per account. Tokens
are unguessable, stored as hashes, expire, are single-use and bound to an address.

## Compatibility and migration

Additive. The new column starts true for existing rows.

## Tests

`TestConfirmacaoDeEmail` (no session before confirming; `403` with the message for the right
password, the same `401` for a wrong password and an unknown login; HTTP Basic refused; the
same answer to resend for known and unknown accounts and no second e-mail within the gap; the
token not stored in clear; one use; `email_confirmado` cannot be set by the person; a new
address is confirmed again; an expired link and a link of a previous address confirm nothing),
`TestConfirmacaoUsoUnicoConcorrente` (eight simultaneous uses, one success),
`TestConfirmacaoSemCorreio` (sign-up closed, with the reason), `TestFrasesDeIdentidade` (parser).
