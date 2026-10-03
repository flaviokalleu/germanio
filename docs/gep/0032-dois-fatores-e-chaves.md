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
  checks keys; a transport that authenticates with them is a separate capability.

## Alternatives studied

- **`formato "regex"`**: a regular expression cannot check a key (base64 of a structure, sizes
  of RSA moduli) nor compute its fingerprint.
- **A type inferred from the name (`chave`)**: `chave` already means other things (variables,
  settings); an explicit type is the safer default.
- **Do nothing:** every application parses keys by hand.

## Tests

`TestChavePublicaValidacao` (canonical form, fingerprint, ed25519/ECDSA/RSA 2048 accepted;
RSA 1024, options, two lines, junk, oversized refused), `TestChavePublicaNoApp` (in a non-GitLab
app: the fingerprint cannot be forged, the same key with another comment is refused, the key
cannot be edited while the label can), `TestChavePublicaTipo` (parser: type, derived field,
two keys refused) and the GitLab end-to-end test `TestChavesSSH` (`/user/keys`).
