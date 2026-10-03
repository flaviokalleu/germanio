# GEP 0037: Git over SSH (no syntax; configuration of the host)

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (GitLab stress test, inventory ID-08 transport / RP-10); decision by the maintainer
- **Level:** none in `.ge` (operation of the server)
- **Layer:** core (the Git hosting capability gains a second transport)

## Problem

Git clients speak two transports: smart HTTP and SSH. Many people, and most automation (deploy
keys, CI machines, scripts), use SSH with a key instead of a password or a token. A program that
declares `tem repositório` and keeps people's public keys (`chave pública`, GEP 0032) already
says everything needed: who the people are, which key is whose, who may download or send code.
What was missing is the transport. Done by hand it is a classic hole: a shell behind the key
(`command=` tricks in `authorized_keys`), paths with `..`, forwarding left on, a second set of
authorization rules that drifts from the HTTP ones, a host key that changes at every start (or
is readable by anyone).

## Evidence

GitLab inventory ID-08 (SSH keys: stored since GEP 0032, no transport) and RP-10 (clone and push
over SSH). Any Git hosting (an internal forge, a documentation site kept in Git, a set of
configuration repositories) needs the same.

## Proposal

No phrase. SSH is offered by the host's configuration, like the HTTP port:

| Variável | Para quê |
| --- | --- |
| `GERMANIO_SSH_ENDERECO` | turns Git over SSH on, listening there (`:2222`, `0.0.0.0:22`); unset, no SSH server exists |
| `GERMANIO_SSH_CHAVE_HOST` | the file of the server's key (default `<GERMANIO_GIT_RAIZ>/.ssh/chave_host_ed25519`) |
| `GERMANIO_SSH_CONEXOES` | connections served at the same time (default 64) |

People clone with the record's repository address, the same one HTTP uses without `.git`
being required: `git clone ssh://git@servidor:2222/grupo/projeto.git`.

## Semantics

- **Who:** the person is found by the key's SHA-256 fingerprint in the data that has a `chave
  pública` field and belongs to the people who sign in (the stored key is then compared whole).
  Only public-key authentication exists: no password, no keyboard-interactive. The SSH user
  name is ignored (`git@` by habit). An unknown key is refused by SSH itself.
- **Who may not:** a person who is not active (`login exige estado "ativo"`, blocked) or has not
  confirmed the e-mail (GEP 0031) is refused with the reason. A second factor (GEP 0032) does not
  apply: the key is its own credential, like an access token.
- **What:** only `git-upload-pack '<endereço>'` (clone, fetch, pull) and `git-receive-pack
  '<endereço>'` (push). The command is read by a strict pattern and git is started with an
  argument vector, never a shell. The address takes letters, digits, `_ . -` and `/`; no empty,
  `.` or `..` segment, no segment starting with `.` or `-`, no spaces or quotes. It must be the
  repository key of a record; nothing else is served. Any other request (another command, a
  shell, `sftp`, environment variables, terminals, agent or port forwarding) is refused; a
  session without a command (`ssh -T`) only greets the person, so they can check their key.
- **Rules:** exactly those of smart HTTP, through the same functions: seeing the record,
  `baixar código` / `enviar código`, read-only records, protected branches (default branch and
  GEP 0016), `antes de enviar código` (its refusal is shown by `git push` as the server's
  reason), and after a push the event to integrations, the history, the executions, the push
  mirrors ([GEP 0036](0036-espelhos.md)) and `quando enviar código`, and LFS objects are
  not served over SSH (they keep using HTTP). A repository the person may not see answers "not found", never "forbidden".
- **Host key:** ed25519, created once (folder 0700, file 0600, never overwritten) and read
  afterwards; a file that others can read stops the start with the fix (`chmod 600`). It is
  never logged.
- **Limits:** 64 connections at once (configurable), 8 from one address, 15 seconds to finish the
  handshake and ask for a command, 2 minutes without traffic close the connection, a command
  runs at most as long as an HTTP clone or push (5 times the git timeout), 4 sessions per
  connection, 6 key attempts per connection, a push at most 2 GB.
- **Pay for what you use:** without `GERMANIO_SSH_ENDERECO` nothing starts. With it, a program
  without repositories or without people's public keys does not start, and the error says what to
  declare.
- **Messages:** refusals are written in Portuguese on the client's terminal (`germanio: o
  repositório "x" não foi encontrado, ou você não tem acesso a ele…`).

## Alternatives studied

- **A phrase (`repositórios aceitam ssh`)**: nothing in the domain changes with the transport;
  the same program serves HTTP and SSH. Where to listen is a decision of whoever hosts it.
- **Spawning OpenSSH with `AuthorizedKeysCommand` and a forced command:** depends on the system's
  sshd configuration and on a second program; the in-process server keeps one binary, one set of
  rules, and is tested end to end.
- **Accepting passwords over SSH:** passwords already have HTTP (with lockout and the second
  factor); keys are the reason SSH is wanted.
- **Do nothing:** people who rely on SSH keys cannot use the forge.

## Tests

`TestGitPorSSH` (no GitLab: a collection of music scores; the system `ssh` and `git` clients with
temporary keys: the owner clones and pushes, the history records the push; a member clones the
private collection, the protected default branch refuses their push with the rule's message,
leaving no trace, and another branch is accepted; someone without a role does not find the private
one, clones the public one and cannot push to it; an unknown key is refused; other commands, shell
tricks, `..`, options and oversized paths are refused; `ssh -T` greets; a blocked person is refused
with the reason; the host key is 0600), `TestParseSSHCommand`, `TestSkipPack` (a refused push's
pack is read to its end and no further), `TestHostKey` (created once, 0600, refused when readable
by others), `TestSSHServerRecusa` (forwarding, environment, terminal and `sftp` refused; a session
or a connection that stays silent is closed), `TestSSHServerLimite` (connections per address), and
in GitLab `TestGitPorSSHGitLab` (a key registered at `/user/keys` clones `ada/app.git` and pushes;
the commit appears in the repository API; a non-member is refused; a removed key no longer gets in).
