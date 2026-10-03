# GEP 0044: The place of a new record, named by its address (no syntax)

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (GitLab stress test: fork into a personal namespace, `namespace_path`);
  decision by the maintainer
- **Level:** 1 (no new phrase; a value of creating and copying)
- **Layer:** core (creating and copying a data with `endereço dentro de …`), adapter (the external
  name)

## Problem

`endereço dentro do grupo ou do criador` makes addresses like `empresa/web/app` and `ana/app`:
a project lives inside a group or in the space of the person who created it. People name places
by those addresses — "put the copy in `empresa/web`", "in my own space" — but creating or copying
only accepted the internal reference of the group, and nothing named "my own space" explicitly:
leaving the group out meant it, and naming another place by address was silently ignored.

## Evidence

GitLab `POST /projects/:id/fork` with `namespace_path` (a group's full path or the user's own
name); the same intent exists in any product with addresses: a document copied into a folder by
its path, a page created under a section by its path.

## Current state

`runtime/servidor/intencao.go` (create) and `copias.go` took the parent reference (`grupo_id`)
only; an unknown value in the request was ignored.

## Proposal

No new phrase. Creating or copying a record of a data with an address accepts `lugar`, the
address of where it goes:

```text
POST …/projetos            {"nome": "App", "caminho": "app", "lugar": "empresa/web"}
POST …/projetos/7/copiar   {"lugar": "ana"}        the person's own space
```

The integration vocabulary may rename it (`lugar é "namespace_path"`).

## Semantics

- `lugar` is looked up, in the order of the address declaration, among the containers'
  addresses (`grupo` → its `endereco`); the first container found is the place, and every other
  container of the address is cleared (the place wins over a reference sent with it).
- A container the person does not see answers 404 (it does not exist for them), like any hidden
  parent; one they see but may not create in answers 403, through the ordinary rules of creating.
- When the address may be the creator's (`ou do criador`) and `lugar` is a person's name: the
  person's own name means their own space (no container); another person's name is refused
  (403), even for an administrator, because whoever creates becomes owner (`quem cria vira
  owner`) and the record would not belong to the person whose space it is.
- An address that is nobody's answers 404.
- Nothing else changes: visibility ceilings, the address uniqueness, copies never more visible
  than the original (GEP 0029), history and events.

## Errors

404 "O lugar … não existe" (or the hidden container's "inexistente"); 403 "… é o espaço de outra
pessoa: crie no seu (…) ou num lugar onde você pode criar".

## Evaluation

The author of the domain writes nothing new; the place is named the way people already read it.
No reference (ID) has to be learned by whoever names the place.

## Impact

Runtime only: `runtime/servidor/lugar.go`, called by creating (`intencao.go`) and copying
(`copias.go`).

## Performance and security

One indexed lookup per container data (the address is unique and indexed), only when `lugar` is
given. Security: the place is checked with `ver` before anything, so naming a hidden group never
reveals it (404, the same as an address that does not exist); the rules of creating decide the
rest.

## Compatibility and migration

Additive: requests without `lugar` behave as before.

## Trade-offs and alternatives

- **Accept the address in the reference field (`grupo_id: "empresa/web"`):** mixes two kinds of
  value in one field and cannot name a person's space.
- **Let administrators create in someone else's space:** needs the person named to become owner
  instead of the creator, a change of `quem cria vira owner` left for a later proposal.
- **Do nothing:** the external `namespace_path` stays silently ignored.

## Tests

`runtime` `TestVariosValoresLugarECopias` (contracts in folders, no GitLab: created and copied
into a folder by its address, the person's own space wins over a reference, another person's
space 403, a private folder 404, an unknown address 404, a folder one may not create in 403);
`examples/gitlab-foss/e2e` `TestForkNoEspacoPessoal` (`namespace_path` with one's own name, a
subgroup, a group over `namespace_id`, another person 403, a private group 404, unknown 404, not
even the administrator in someone else's space, creating a project by the group's path).
