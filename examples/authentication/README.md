# authentication

Sign in, sign up, roles and permissions per role.

## What it demonstrates

- `tenha login`, `tenha cadastro`, `login usa email`: sign-in, sign-up and sessions with CSRF
  protection. Passwords are hashed and never returned; sign-up never accepts `admin`.
- `login bloqueia após 5 tentativas por 15 minutos`: lockout after repeated failures.
- `tenha administrador inicial "admin@equipes.local"`: the first administrator, created only
  when nobody exists yet. The password comes from the server environment, never from the file.
- Roles (`tenha papeis` › `leitor 10`, `editor 30`, `dono 50`): a higher role inherits what the
  lower ones may do.
- Each team (`equipes`) has members with a role. Whoever creates a team becomes its owner
  (`quem cria vira dono`), and a team always keeps at least one owner
  (`precisa de pelo menos um dono`).
- Documents inside a team: readers see, editors create and edit, only owners delete.
  People outside the team get "not found".

## Run

```bash
go build -o ge ./cmd/ge
./ge check examples/authentication/app.ge
GERMANIO_ADMIN_SENHA='choose-a-password' ./ge rodar examples/authentication/app.ge 8080
```

Open http://localhost:8080, create an account and a team. Members are managed under
`/_ge/api/equipes/<id>/membros` (`{"pessoa_id": 2, "papel": "editor"}`).

Known limitation: deleting a person who belongs to a team fails with a database error
([G76](../../GERMANIO_GAPS.md)).

**Capabilities:** login, sign-up, lockout, initial administrator, roles, membership,
minimum-owner invariant, per-role permissions, pages.

**Keywords are Portuguese** (`acesso` = access, `somente` = only, `regras` = rules,
`papeis` = roles). See [the index](../README.md#a-note-on-language).
