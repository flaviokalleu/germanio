# GEP 0036: Mirrors of a repository (`espelham o repositório do projeto`)

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (GitLab stress test, inventory RP-10); decision by the maintainer
- **Level:** 1
- **Layer:** domain (one phrase on a data), core (Git mirrors, guarded transport, background
  tasks), adapter (external names only)

## Problem

Teams keep a copy of a repository in another Git server (a backup, the public mirror of an
internal project, the old server during a migration), or keep a local copy of a repository that
lives elsewhere. Done by hand this is a cron job with a URL, a password in a script, `git push
--mirror`, and nobody noticing when it stops working. The list of mirrors of each project is
chosen and changed by the people who manage it, so it is data, not source code — like webhooks
(`recebe eventos do projeto`) and protected branches (GEP 0016).

## Evidence

GitLab inventory RP-10 (push and pull mirrors). Any application hosting code needs the same:
a book publisher keeping its manuscripts in a second server, a school mirroring the public
repositories its students use.

## Proposal

```text
espelhos
    tem
        url obrigatório
    pertence a projeto
    espelham o repositório do projeto
    acesso
        maintainer
            administrar
```

Flat form: `espelhos espelham o repositório do projeto` (also `espelha`). The data may have any
name (`copias espelham o repositório do livro`).

## Semantics

- The data belongs to a data with `tem repositório` and has a `url`. The capability adds what
  every mirror needs, shown by `ge explain`:
  - `sentido`, chosen by the people: `enviar` (default) or `receber`;
  - `habilitado` (default true);
  - `credencial`, hidden: never returned by any listing, page or integration;
  - `situacao` (`nova`, `agendada`, `atualizando`, `atualizada`, `falhou`), `ultima_atualizacao`,
    `ultimo_sucesso`, `ultimo_erro`, maintained by the runtime.
- **enviar:** after every change of the code — git push, a file edited on the web, a merge, or a
  receiving mirror that brought something new — the whole repository goes to the url: every
  branch and tag, replacing what differs and removing what no longer exists (an exact copy).
- **receber:** every `GERMANIO_ESPELHO_MINUTOS` (default 30) the repository becomes an exact copy
  of the url's branches and tags. What arrives continues to the sending mirrors. A receiving
  mirror replaces the branches people push to; protected-branch rules and `antes de enviar código`
  do not apply to it (it is a copy, decided by whoever may manage mirrors), and executions are not
  started by it.
- Creating or editing a mirror schedules an update right away.
- **Update now** (amendment): the built-in action `atualizar_agora` (`POST …/<espelho>/
  atualizar_agora`; the vocabulary may rename it, `atualizar_agora é "sync"`) schedules an update
  without waiting for a change of the code or the period. Only whoever may edit the mirror (404
  for whoever does not see it); a disabled mirror is refused (400); a read-only owner (archived)
  refuses it like any action. It is the same background task: the same guarded transport and SSRF
  checks at connection time, retries and record of attempts; a mirror already waiting is not
  queued twice.
- **Background, never in the way:** updates are persistent background tasks queued in the same
  transaction as the change that caused them (a change undone queues nothing), run after it, and
  are tried again with the queue's backoff. A failing mirror never breaks a push; each attempt is
  recorded on the record. A mirror already waiting for its turn is not queued twice: when it runs,
  it sends the repository as it is then.
- **Address:** only `http` and `https`. `file`, `ssh`, `git` and git's `ext::` transports are
  refused, because they could read or run things on the server itself. Local addresses
  (loopback, private, link-local, and names that resolve to them) are refused when saved and again
  when connecting, unless `GERMANIO_PERMITIR_REDE_LOCAL=1` (development only).
- **Credentials:** user and password written in the url (`https://ana:token@exemplo.com/a.git`)
  are moved to `credencial`; the url keeps only the address. Changing the address forgets the
  credentials, so they never follow to another server; editing anything else keeps them.

## Errors

Resolver: the owner has no repository (the error says to declare `tem repositório`), the data
does not belong to the owner or has no `url` (the error lists what is missing), the data is the
owner itself, an incomplete phrase (the error shows the phrase to write). Runtime: an invalid,
non-http(s) or local address is a 400 on `url` with the reason; a failed update keeps the
reason in `ultimo_erro`, without credentials.

## Alternatives studied

- **Two phrases (`recebem o código do projeto` / `trazem o código para o projeto`):** two data
  for the same idea; the direction is something people choose per mirror, so it is a field.
- **Fields declared by the author (`sentido`, `credencial oculto`…):** every mirror needs them,
  with the same meaning; asking for them is ceremony, and forgetting `oculto` would leak
  credentials. They are added by the capability, as `runners executam jobs` adds its fields.
- **A level-3 hook with `git.*` and `chamar`:** mechanism in the domain, credentials in code,
  no SSRF protection, no retries, no record of failures.
- **Do nothing:** no mirrors.

## Performance and security

Git never opens a connection by itself: every byte goes through a proxy inside the server whose
dialer is the SSRF-protected one of `runtime/httpclient`, with redirects off, only http(s) allowed
(`GIT_ALLOW_PROTOCOL`), no credential helpers, objects checked (`transfer.fsckObjects`), a byte
budget of 2 GiB per update (the same bound as a push received), a stall limit and a 10-minute
timeout. Credentials reach git through its environment as an `Authorization` header — never in
the command line or the url — and are removed from error messages. The scheduler reads receiving
mirrors a page at a time. Pending: the task queue runs one task at a time, so a long mirror update
delays other background tasks (webhook deliveries) until it ends or times out; SSH mirrors; mirroring only protected branches; keeping divergent refs.

## Tests

`runtime/git/remoto_test.go` (`TestEspelhos`: push copies branches and tags and removals to an
independent Git server — git's own http-backend — with basic authentication, a wrong password
fails without appearing in the error, fetch makes another repository an exact copy and reports
the updates; `TestEspelhoRecusaRedeLocalEOutrosTransportes`: local addresses refused without
`GERMANIO_PERMITIR_REDE_LOCAL`, including a public name that resolves to 127.0.0.1, other
transports refused, credentials split out of the url), `compiler/parser/espelhos_test.go`
(block and flat forms are the same app; the added fields; resolver errors),
`runtime/lfs_espelhos_test.go` (`TestEspelhos`, a domain without GitLab: local address refused
without the permission, other transports refused, only managers create mirrors, credentials never
returned, a web edit reaches the mirror, a failing mirror is recorded without breaking the edit
and without the password, a new address forgets the credentials, a receiving mirror fills another
repository) and the GitLab E2E `examples/gitlab-foss/e2e/lfs_espelhos_test.go` (remote mirrors
through the GitLab API after a real `git push`). Update now: `runtime` `TestEspelhoAtualizarAgora`
(publishing house: a ref added behind the mirror's back is removed on demand, only by the editor,
refused when disabled) and the GitLab E2E `TestEspelhoAtualizarAgora` (`/sync`; a local address
fails at connection time without the permission; disabled; archived project).
