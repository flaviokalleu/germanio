# GEP 0008: Password recovery

- **Status:** Aceita (2026-09-28, pelo mantenedor). Norma: `docs/INTENCAO.md` › Login
- **Author:** agent (GitLab stress test, inventory ID-06); decision by the maintainer
- **Level:** 1
- **Layer:** domain (one phrase), core (mechanism)

## Problem

A person who forgets their password has no way back into an application: an administrator
must intervene. Every real application with login needs recovery, and doing it by hand means
tokens, expiry, e-mail, and several classic mistakes (revealing which e-mails exist, reusable
links, tokens stored in clear, links that never expire).

## Evidence

GitLab inventory ID-06 (NOT_STARTED). No capability in `docs/INTENCAO.md` › Login.
`docs/research/languages/SECURITY_DEFAULTS.md` lists account recovery among the flows
frameworks get wrong most often.

## Current state

`tenha login` and `tenha cadastro` exist; recovery does not. The runtime can send e-mail
(`runtime/email`), configured today by an `email` block whose SMTP password sits in the `.ge`
source.

## Alternatives studied

- Rails `has_secure_password` + `generates_token_for` (signed, expiring tokens tied to the
  password digest); Django's `PasswordResetView` (token derived from the password hash and last
  login, so it dies once used); Laravel's password broker (a hashed token table with expiry).
  All converge on: a random or signed token, short expiry, single use, the same answer whether
  or not the account exists.
- Magic links (log in by e-mail) — a different feature; out of scope.

## Proposal

One phrase next to the other login phrases:

```text
tenha login
tenha cadastro
tenha recuperação de senha
login usa email
```

## Semantics

- `/esqueci`: a form asking for the login. Whatever is typed, the answer is the same ("if an
  account exists, we sent a link"): the page never reveals which accounts exist.
- When the account exists, a random token (32 bytes from `crypto/rand`) is created; only its
  SHA-256 is stored, with an expiry of 1 hour. The e-mail carries the link
  `GERMANIO_URL_PUBLICA/redefinir?token=…`.
- `/redefinir`: a form for the new password, validated by the password field's own rules.
  A token works once; using it deletes every recovery token of that person and clears the
  login lock. The page then sends the person to `/entrar`.
- Requests are limited like logins (per address), and at most one e-mail goes to an account
  every 2 minutes, however many requests arrive.
- E-mail is configured outside the source: `GERMANIO_SMTP_HOST`, `GERMANIO_SMTP_PORTA`,
  `GERMANIO_SMTP_USUARIO`, `GERMANIO_SMTP_SENHA`, `GERMANIO_SMTP_REMETENTE`, plus
  `GERMANIO_URL_PUBLICA` for the link. Without them the application starts, and `/esqueci`
  says recovery is not available yet, logging why (never a silent failure).

## Errors

Like `tenha cadastro`, `tenha recuperação de senha` implies the login (it declares it when
absent). Anything else on the same line is an error (as for `tenha login`).

## Evaluation

One phrase, no technical concept for the author. The dangerous details (tokens, hashing,
expiry, enumeration) live in the core.

## Impact

Parser: the phrase. AST/resolver: a flag on the login. Runtime: two pages, one internal table,
the mail sender. `ge explain` shows the recovery and its expiry.

## Performance and security

One insert and one e-mail per request, limited per address. Tokens are unguessable, stored as
hashes, expire, and are single-use; the answer does not reveal accounts.

## Compatibility and migration

Additive.

## Trade-offs and alternatives

"Do nothing" leaves every application without recovery or with a hand-made one. Signed
stateless tokens avoid the table but cannot be revoked individually; a table is simpler to
reason about.

## Tests

The same answer for existing and unknown accounts; the e-mail carries a link whose token works
once and expires; the new password obeys the field rules; the lock is cleared; without SMTP the
page says so.
