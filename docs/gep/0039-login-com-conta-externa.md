# GEP 0039: Sign-in with an external account

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (GitLab stress test, inventory ID-07, OAuth part); decision by the maintainer
- **Level:** 1 (the phrase); the provider is server configuration, outside the program
- **Layer:** domain (one phrase), core (OpenID Connect relying party), configuration (environment)

## Problem

Companies and communities already have an account system (Google Workspace, Microsoft Entra,
Keycloak, Authentik, a university's login). People expect "Entrar com …" instead of one more
password. Done by hand, external sign-in is one of the most error-prone flows on the web: a
missing or unchecked `state` (login CSRF: the victim ends up signed in as the attacker), no
PKCE (a stolen code is enough), an ID token believed without checking its signature, audience,
issuer, expiry or nonce, `alg: none`, a redirect address built from the request's `Host`, an
account matched by an e-mail the provider never verified (account takeover), the client secret
in the source, the second factor silently skipped.

## Evidence

GitLab inventory ID-07 ("2FA / OAuth"): GitLab signs people in through OmniAuth providers
(`omniauth_providers`, `/users/auth/:provider/callback`, `identities` of a user). Every product
that is sold to organizations has the same need: CRMs, help desks, internal tools, LMSs.

## Proposal

One phrase next to the other login phrases:

```text
tenha login
tenha cadastro
tenha login com conta externa
login usa email
```

Which provider is **configuration of whoever hosts the application**, never part of the program:

| Variable | Meaning |
| --- | --- |
| `GERMANIO_OIDC_EMISSOR` | the provider's issuer identifier, exactly as its discovery document announces it (`https://accounts.google.com`, `https://sso.example.com/realms/main`) |
| `GERMANIO_OIDC_CLIENTE` | the client id registered at the provider |
| `GERMANIO_OIDC_SEGREDO` | the client secret (optional: a public client relies on PKCE alone); read only from the environment, never logged, never in `.ge` |
| `GERMANIO_OIDC_NOME` | the name on the button, "Entrar com <nome>" (default: the issuer's host) |
| `GERMANIO_URL_PUBLICA` | the application's public address; the return address registered at the provider is `GERMANIO_URL_PUBLICA/entrar/externo/retorno` |

The phrase was chosen over `tenha login com provedor externo` (the word *provedor* is technical
for many readers; *conta externa* is what the person has) and over naming a provider
(`tenha login com google`): the program would change when the organization changes providers,
and provider knowledge would leak into the domain. `tenha login com google`, `tenha login com
provedor externo` and similar wordings are errors that show the phrase and the variables.

## Semantics

### The protocol (core, generic)

- **OpenID Connect Core 1.0, Authorization Code flow**, with **PKCE S256** (RFC 7636), `state`
  and `nonce`, scope `openid email profile`. Only providers that speak OpenID Connect (discovery
  document, ID tokens, JWKS) are supported; plain OAuth 2.0 sites without ID tokens (GitHub's
  sign-in, for example) are not: each one would need its own protocol knowledge, an adapter.
- **Discovery:** `GERMANIO_OIDC_EMISSOR/.well-known/openid-configuration`, fetched on first use
  and kept for an hour. Its `issuer` must be *exactly* the configured one (Discovery §4.3); its
  endpoints must be `https` (plain `http` only with `GERMANIO_PERMITIR_REDE_LOCAL=1`, for a local
  development provider); a provider announcing PKCE methods without `S256` is refused.
- **Start** (`POST /entrar/externo`, the button): three random values of 32 bytes (state, nonce,
  PKCE verifier) are kept on the server for **10 minutes** (the state only as its SHA-256); the
  state also goes to a cookie `HttpOnly`, `SameSite=Lax`, limited to `/entrar/externo`
  (`Secure` when the public address is https). The browser goes to the provider.
- **Return** (`GET /entrar/externo/retorno`): the `state` of the address must equal the cookie
  (constant-time compare) — a return started in another browser is refused (login CSRF) — and
  the pending sign-in is **used up** (deleted; the same return twice, even at once, works once)
  and must not be expired. A provider error (`access_denied`) is said in plain words.
- **Code exchange:** at the token endpoint, with the PKCE verifier and the same return address;
  client authentication `client_secret_basic` (the default of Core §9) or `client_secret_post`
  when the provider only announces that; none without a secret.
- **ID token**, all checked before anything is believed: compact JWS signed with **RS256 or
  ES256** only (never `none`, never HMAC), by a key of the provider's JWKS (`kid`; RSA keys under
  2048 bits and EC curves other than P-256 are ignored; an unknown `kid` fetches the JWKS again at
  most every 30 seconds, for key rotation); `iss` equal to the issuer; `aud` containing the client
  id, and `azp` equal to it when present or when there are several audiences; `exp` in the future
  and `iat` not in the future (60 seconds of clock skew); `nbf` respected; `nonce` equal to the
  one sent (constant time); `sub` present. Verification uses only the Go standard library
  (`crypto/rsa`, `crypto/ecdsa`, `crypto/sha256`); no dependency was added.
- **Userinfo:** only when the ID token has no e-mail; its `sub` must be the token's.
- **Outgoing requests** go through the SSRF-protected transport (`runtime/httpclient`): no
  request reaches the server's own network unless `GERMANIO_PERMITIR_REDE_LOCAL=1`; redirects are
  not followed (a redirected token request could carry the secret elsewhere); answers are limited
  to 1 MB and 10 seconds.

### Who the external account is (the decision about accounts)

- An external account is the pair **(issuer, subject)**, stored in an internal table
  (`_germanio_contas_externas`, one external account per person per provider). The e-mail is not
  an identity: providers let people change it, and some let them type any address.
- **Already linked:** the person of that pair signs in (even if the e-mail changed there).
- **Not linked, e-mail not verified by the provider** (`email_verified` absent or false):
  refused, with "confirm the e-mail at the provider". Such an address finds no account and
  creates none.
- **Not linked, verified e-mail of an existing account:** linked and signed in **only when this
  system confirmed that address too** — the application declares `tenha confirmação de e-mail`
  (GEP 0031) and the account has `email_confirmado`. Otherwise refused, with the way out: sign in
  with the password and link the external account at `/conta-externa`.
  *Why both:* if only the provider's word counted, an attacker could pre-register here with the
  victim's address (no confirmation) and keep the password of the account the victim would later
  enter with the provider ("pre-account takeover"); if only this system's word counted, a
  provider that does not verify addresses would let anyone claim any account. With both, the
  account's owner and the provider's account owner proved control of the same address.
- **Not linked, verified e-mail, no account:** a new person is created **only when the system has
  `tenha cadastro`** (sign-up is open); a closed system refuses and says to ask an administrator.
  The person is created like a sign-up (the same validations, the same `quando criar` hook): the
  verified e-mail, the name from the provider, a login field such as `username` from the
  preferred name (letters, digits, `_ . -`, numbered when taken), a random password (password
  recovery creates a real one). With `tenha confirmação de e-mail`, `email_confirmado` starts
  true: the provider verified the address.
- **Linking while signed in** (`/conta-externa`, a browser session and its CSRF proof): the
  return must reach the same browser still signed in as the same person; the pair is then linked
  whatever the e-mail there (the person proved both accounts). A pair already linked to someone
  else is refused. Unlinking needs the session and CSRF proof.

### Signing in

- The session is the one of a password sign-in (the same cookie, the same CSRF proof); people who
  cannot sign in (`login exige …`) and people whose e-mail is waiting for confirmation are refused
  as with the password.
- **The second factor still applies** (GEP 0032): someone who turned it on gets the code page and
  a one-use challenge, exactly as after the password. The provider's own sign-in strength is not
  known to this system (an `amr` claim is optional and provider-defined), and the person chose the
  factor here.
- The lock of the login (wrong passwords) does not stop an external sign-in: nothing is guessed
  here, and refusing would let anyone who types wrong passwords lock the owner out of every way in.
- Requests count in the limit per address of the login (50 in 10 minutes, then `429`): every
  start, and every return refused as invalid (wrong state, used or expired, token refused).

### What it does not do (and why)

- **LDAP:** a directory protocol where the server receives the person's password and binds with
  it — password handling, a long-lived service account, TLS to the directory, group mapping; a
  separate capability with its own trust decisions.
- **SAML 2.0:** XML signatures (canonicalization, signature wrapping) are a notoriously dangerous
  surface and need a dedicated, audited implementation or dependency; a separate GEP.
- **WebAuthn / passkeys:** a credential of this system (like the second factor), not sign-in with
  another system; it needs attestation and authenticator decisions of its own.
- Several providers at once, group or role mapping from the provider, single logout, and
  refreshing provider tokens (they are not kept: only the sign-in is needed).

## Alternatives studied

- **A provider block in the program** (`provedor "Google"` + client id): provider knowledge in the
  domain, and secrets next to it. Rejected; the program says the intent, the host says the provider.
- **Plain OAuth 2.0 with a userinfo call** (no ID token): the identity would rest on a bearer
  token answer, with no signature, audience or nonce to check. Rejected.
- **Matching accounts by e-mail alone** (common in tutorials): account takeover in both directions,
  as explained above. Rejected.
- **Always creating a new account when the e-mail exists:** a second account for the same person,
  and the unique e-mail would refuse it anyway. Rejected in favor of the educational refusal and
  linking while signed in.
- **Skipping the second factor after an external sign-in:** convenient, but the person's choice
  here would depend on another system's policy. Rejected.
- **A JWT/OIDC library dependency:** the needed subset (RS256/ES256 verification, JWKS parsing)
  is small and the standard library covers it; a dependency would add surface and supply chain.
- **Do nothing:** every organization keeps one more password per person.

## Tests

`TestVerifyRefusals` (signature by another key, swapped content, `alg: none`, `HS256`, unknown
`kid`, wrong `iss`, wrong `aud`, several audiences without `azp`, foreign `azp`, expired, no
`exp`, `iat` and `nbf` in the future, wrong and empty nonce, no `sub`, malformed),
`TestChallengeS256` (the RFC 7636 vector), `TestDiscovery` (issuer must match exactly; https
outside development; the authorization address carries S256, state, nonce, scope),
`TestProvedorNaRedeLocalRecusado` (the SSRF guard), `TestES256ERotacao` (ES256 and key rotation),
`TestLoginComContaExterna` (parser: the phrase, the e-mail field, providers named in the program
refused), and with an in-process fake provider (`runtime/oidc/oidctest`: discovery, approval,
token endpoint checking client, PKCE and single use of codes, JWKS with a test RSA key):
`TestContaExterna` (the button; a new person from a verified e-mail with a free login and the same
session as a password sign-in; recognized by subject afterwards; tampered state and a return from
another browser refused; a replayed return and a replayed code refused; bad signature, wrong
`aud`, wrong `iss`, expired token and wrong nonce refused without creating anyone; an unverified
e-mail neither takes over an existing account nor creates one; a verified e-mail does not take an
account whose address this system did not confirm; verified and confirmed links; the second
factor is still asked; linking from a session needs CSRF and works whatever the e-mail; an
external account linked to someone else cannot be linked again), `TestContaExternaSistemaFechado`
(no sign-up and no confirmation: no account created, none linked by e-mail),
`TestContaExternaSemConfiguracao` (no button; starting says it is not available), and in GitLab
`TestContaExternaGitLab`.
