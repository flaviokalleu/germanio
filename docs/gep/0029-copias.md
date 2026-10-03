# GEP 0029: Copies of a record (`copiar`)

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (GitLab stress test, inventory PR-05 fork); decision by the maintainer
- **Level:** 1 (one built-in verb in `acesso`, like `sair` and `revogar`)
- **Layer:** domain (who may copy), core (what a copy is), adapter (the external names)

## Problem

Forking a repository (GitLab, GitHub), duplicating a template (a contract, a recipe, a project
plan, a quote in a CRM) or cloning a product in a shop are the same intent: make a new record that
starts as a copy of one I can see, in my name, somewhere I am allowed to create it. By hand this is
a create endpoint that reads the original, decides which fields to carry, re-checks where the copy
may go, copies the files or the repository, remembers the origin, and must not let the copy be more
visible than the original. Each of those steps is a place to leak something or to forget a rule.

## Proposal

```text
projetos
    acesso
        usuario
            copiar
        reporter
            copiar
```

Flat form: `usuario pode copiar projetos`. `copiar` becomes a built-in verb (it needs no `quando
copiar` hook), the way `sair` and `revogar` already are. A program that defines its own `quando
copiar` keeps it: the built-in meaning only applies when the program says nothing else.

## Semantics

- **Who.** Whoever passes a `copiar` rule on the original *and* sees it. Inside something with
  members the generic rules (`usuario`) only count when the original is public or internal, as for
  every other verb; a private original is copied only by the roles listed. When the data has a
  repository, copying also requires `baixar código`: copying code is reading it.
- **What the copy is.** A new record created by whoever copies, through the same path as `criar`:
  where it may be created (the parent given, or none), `antes de criar` / `quando criar`,
  `quem cria vira owner`, the visibility ceilings, the minimum role, the history and the events.
  Someone who may copy but may not create where they asked gets 403 (404 if that place is hidden).
- **Which values.** The plain values of the original: texts, numbers, dates, flags and lists of
  texts. Not copied: what places the original (its parents, its address, its number inside the
  parent), what the runtime keeps (state, counts, the repository path), the read-only flag, files,
  secrets, hidden fields and references to other data. The request may change any value people may
  set (a new name, a new path, the parent, a lower visibility).
- **The origin.** The copy keeps `copiado_de_id`, a system field nobody sets or edits. The record
  shows `copiado_de` (the original as anyone allowed would see it) only to someone who still sees
  the original; to anyone else the copy does not say where it came from. Deleting the original
  keeps the copies, without the link.
- **Visibility.** A copy is never more visible than its original: by default it starts as visible
  as the original, lowered to the visibility of where it is created; asking for more is refused
  (400 with the reason), now and when the copy is edited later.
- **Repository.** The copy gets its own repository with every branch and tag of the original and
  the same default branch (`git clone --bare`, objects hard linked when possible, no hooks, no
  configuration and no link back). Pushing to one never changes the other. The initial file
  (`repositório pode começar com`) is not added to a copy.
- **Read-only originals.** An archived (read-only) original may still be copied: copying reads it.
- **Address.** `POST …/<registro>/copiar`; the integration vocabulary may rename it
  (`copiar é "fork"`). The answer is 201 with the copy. Pages show a short "Copiar" form to whoever
  may copy, asking the values that place the copy: its title, its address segment (the path) and
  any value that must be unique, filled with the original's (a value unique everywhere starts
  empty, since the original's would be refused). A copy next to its original therefore gets a new
  path instead of failing; a refusal comes back on the page with the reason. The page opens the
  copy afterwards.

## Amendment: the copies of one original

The listing of a data with copies accepts `?copiado_de_id=<original>` (the original by its
reference or unique field; the vocabulary renames the parameter like the field,
`forked_from_project_id`). It keeps the copies of that original that the person sees — each copy
still passes `ver` — and only when the person also sees the original; otherwise the list is empty,
so a copy never tells someone where it came from (the rule of `copiado_de` above). Pagination,
search and the other filters work as on any listing. The GitLab adapter answers
`GET /projects/:id/forks` with it (404 when the project is hidden). Creating the copy in a place
named by its address is GEP 0044. Tests: `runtime` `TestVariosValoresLugarECopias`, GitLab E2E
`TestForksDoProjeto`.

## Alternatives studied

- **A capability section (`pode` › `ser copiado`).** `ser X` already means a yes/no condition
  (`ser arquivado`, `ser confidencial`); reusing it for an operation would make one phrase mean
  two things. And a capability without saying *who* would still need an access rule.
- **Ordinary data plus a hook.** A `quando copiar` that creates the record, copies fields and
  clones the repository is exactly the mechanism the domain must not write (and it could not copy
  a repository without technical primitives).
- **Copy everything that belongs to the record (issues, members).** GitLab forks do not, and a
  deep copy is a different, much more expensive intent; left for a later proposal if needed.
- **Do nothing.** Forking is one of the main flows of a code forge; without it the reference
  application cannot express it.

## Performance and security

One record read, one create; the repository copy is a local clone with hard links (no object is
rewritten), done inside the change and removed if the change is undone. The copy never comes from
a path given by the client. Visibility can only go down. The origin link is hidden from whoever
cannot see the original. Lists show the origin with one read per copy shown (bounded by the page).

## Tests

`runtime/git` `TestCopy` (branches, tags, default branch, empty repository, no link back,
independent commits, refused destinations); `runtime` `TestCopiasDeReceitas` (a domain without
repository, members or GitLab: plain values, the copier as author, the origin shown and never set
by hand, deleting the original); `examples/gitlab-foss/e2e` `TestFork` (fork with `git clone` of
the copy, the origin in `forked_from_project`, owner, same path refused, never more visible on
create and edit, lowered inside a private group, someone else's group refused, a private project
not forked by a non-member nor by a guest, by a reporter yes, the origin hidden once it is no longer
visible, deleting the original keeps the fork); the page form in `TestPaginasAcoes` (runtime) and
`TestPaginasRestantes` (GitLab: forking one's own project with a new path).
