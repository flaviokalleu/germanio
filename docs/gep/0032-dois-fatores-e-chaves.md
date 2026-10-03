# GEP 0032: Credentials beyond the password (public keys; two-factor authentication)

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (GitLab stress test, inventory ID-07 and ID-08); decision by the maintainer
- **Level:** 1
- **Layer:** domain (one field type; one phrase), core (checking keys; the second factor)

## Problem

People prove who they are with more than a password: a public key registered for a machine
(SSH, signing, device pairing), a code from an authenticator app. Done by hand, each one is a
trap: parsing keys with string functions, accepting obsolete or weak keys, comparing keys by
their text (the same key with another comment passes as new), letting a client write the
fingerprint; for the second factor, secrets stored in clear, codes compared in variable time,
codes accepted twice, no limit on guesses, a password-only path (API, Git) that skips the
second factor.

## Evidence

GitLab inventory ID-07 (2FA, MISSING) and ID-08 (SSH keys, MISSING). Servers holding deploy
keys, device registries and signing services have the same need as GitLab's `/user/keys`.

## Proposal

### Public keys

A field type, written like the other types after the field name:

```text
chaves ssh
    singular chave ssh
    tem
        titulo obrigatório até 255
        conteúdo chave pública obrigatório e único
    pertence a
        usuario
```

### Two-factor authentication

One phrase next to the other login phrases:

```text
tenha login
tenha autenticação em dois fatores
```

## Semantics

### Public keys

- The value is a key in the `authorized_keys` form (`tipo base64 [comentário]`), checked by
  the SSH library (`golang.org/x/crypto/ssh`), never by string functions. Options before the
  key (`command="…"`), more than one line, or more than 8 KB are refused.
- Weak kinds are refused by default: DSA, and RSA below 2048 bits, with a message that says why.
- The key is stored in one canonical form (`tipo base64 comentário`, spaces trimmed).
- The data gains `impressao_digital` (`SHA256:…`), derived from the key, kept by the system:
  a client never sets it (it is ignored on input).
- `único` on the key means unique **by fingerprint**: the same key with another comment is the
  same key and is refused ("já está em uso" on `impressao_digital`).
- The key never changes once saved (it is removed and another one added); other fields of the
  record may be edited, if the rules allow it.
- Two public keys in one data are an error that tells to make one record per key.
- Using the key (an SSH server, signatures) is not part of this GEP: the field records and
  checks keys; a transport that authenticates with them is a separate capability (Git over
  SSH: [GEP 0037](0037-git-por-ssh.md)).

### Two-factor authentication

- **Opt-in per person.** The phrase offers the second factor; each person decides. The people
  gain the system field `dois_fatores` (false until the person turns it on), shown only to the
  person and to administrators, never set by hand.
- **The factor is TOTP** (RFC 6238 over RFC 4226, HMAC-SHA1, 6 digits, 30-second steps),
  what every authenticator app speaks, built from the standard library (`crypto/hmac`,
  `crypto/sha1`); the RFC test vectors are part of the tests.
- **Turning it on** needs a browser session (with its CSRF proof) and the password again; the
  server creates a 20-byte secret and shows it (and the `otpauth://` address) while it is being
  set up; the first right code turns the factor on and shows **10 recovery codes once**. After
  that the secret is never shown again. Access tokens, OAuth and Git clients cannot change the
  second factor.
- **Storage:** the secret is encrypted with AES-256-GCM, the key derived with HKDF-SHA256 from
  `GERMANIO_SEGREDO`, the person's id authenticated with it (a row copied to someone else does
  not open). Without a lasting `GERMANIO_SEGREDO` (32 characters or more) turning the factor
  on is refused (`503`, with the reason): a secret saved with a key that dies with the process
  would lock the person out after a restart. Recovery codes (80 random bits each) are stored
  only as SHA-256.
- **Signing in:** the right password of someone with the factor on starts no session; it gives
  a challenge kept on the server for 5 minutes (`202` with `desafio`; the login form puts it in
  a short HttpOnly cookie and shows the code page). The challenge and a code at
  `/entrar/codigo` start the session. The challenge works once, even sent twice at once. It is
  a random value checked against the server's table, never a signed token that could pass for
  a session cookie.
- **Codes:** compared in constant time, one step of clock drift each way; a code is never
  accepted twice (the last accepted step is kept and moved in the same statement that checks
  it, so two requests with the same code cannot both pass). A recovery code works once.
- **Guessing:** wrong codes count toward the lock of the login (`login bloqueia após N
  tentativas por M minutos`, 10 in 10 minutes by default), and while the factor is on, a right
  password no longer clears that count — only a right code does, so retyping the password buys
  no extra guesses. Every wrong code also counts in the limit per address (50 failures in
  10 minutes, then `429`).
- **The password alone opens nothing:** OAuth with the password and HTTP Basic with the
  password (Git clients) are refused for people with the factor on; access tokens keep working,
  as their own credential (the same rule as GitLab).
- **Turning it off** needs the session, the password and a code (or a recovery code). **New
  recovery codes** need the session, the password and an app code; the old ones stop working.
- **Pages:** `/dois-fatores` (status, set up, new codes, turn off) and `/entrar/codigo`, plain
  forms that work without JavaScript; the same addresses answer JSON.
- **Out of scope here:** WebAuthn/passkeys, OAuth/OpenID sign-in with other providers, LDAP and
  SAML (each a separate capability with its own protocol and trust decisions), SMS or e-mail
  codes (weaker, and they need a sending channel), an administrator turning someone's factor
  off, and a QR code image (the page shows the secret and the `otpauth://` address).

## Alternatives studied

- **`formato "regex"`**: a regular expression cannot check a key (base64 of a structure, sizes
  of RSA moduli) nor compute its fingerprint.
- **A type inferred from the name (`chave`)**: `chave` already means other things (variables,
  settings); an explicit type is the safer default.
- **Do nothing:** every application parses keys by hand.
- **Second factor by SMS or e-mail:** weaker (SIM swap, shared mailboxes) and dependent on a
  sending channel; TOTP works offline with any authenticator app.
- **Storing the TOTP secret hashed:** impossible, the server needs it to compute codes; it is
  encrypted instead, and never shown after setup.
- **A signed challenge cookie instead of a server table:** a signed value with the person's id
  could be replayed as a session by the same signing key; a random value in a table cannot.

## Tests

Second factor: `TestRFC6238` and `TestRFC4226` (the RFC test vectors), `TestCheck` (window,
single use, wrong codes), `TestDoisFatores` (session and CSRF required; password again; the
secret encrypted in the database and absent from the page and the person's data; recovery codes
stored as hashes; the password gives a challenge and no session; HTTP Basic with the password
refused; wrong, reused and setup codes refused; a recovery code works once; new codes kill the
old ones; turning off needs password and code), `TestDoisFatoresBloqueio` (ten wrong codes
lock the login although the password was right each time), `TestDoisFatoresUsoUnicoConcorrente`
(the same code and challenge sent eight times at once sign in once), `TestDoisFatoresPelaPagina`
(the form flow, challenge never in the address), `TestDoisFatoresSemChave` (refused without a
lasting key), `TestFrasesDeIdentidade` (parser), and in GitLab `TestDoisFatoresGitLab` (OAuth and
HTTP Basic with the password refused, the access token still works, `two_factor_enabled`).

Public keys: `TestChavePublicaValidacao` (canonical form, fingerprint, ed25519/ECDSA/RSA 2048 accepted;
RSA 1024, options, two lines, junk, oversized refused), `TestChavePublicaNoApp` (in a non-GitLab
app: the fingerprint cannot be forged, the same key with another comment is refused, the key
cannot be edited while the label can), `TestChavePublicaTipo` (parser: type, derived field,
two keys refused) and the GitLab end-to-end test `TestChavesSSH` (`/user/keys`).
