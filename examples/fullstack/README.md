# fullstack

There is no separate full-stack example here: the complete application written in the
current intent syntax is [`examples/gitlab-foss`](../gitlab-foss), a reimplementation of a
subset of GitLab FOSS. It is the reference for how a large Germanio application is organized.

## What it shows

```text
examples/gitlab-foss/
├── app.ge         crie sistema gitlab + importar "backend", "frontend", "integracoes"
├── backend/       what exists, who can do what, what happens (12 files)
│   ├── identidade/usuarios.ge     people, access tokens, login, scopes
│   ├── grupos/grupos.ge           groups and subgroups with members and roles
│   ├── projetos/projetos.ge       projects with a Git repository, visibility, owners
│   ├── issues/                    issues, comments, labels, milestones
│   ├── merge_requests/            merge requests with approvals
│   ├── ci/                        pipelines and runners (remote work)
│   ├── webhooks/                  project events
│   ├── busca.ge, enderecos.ge     global search, namespaced addresses
│   └── compatibilidade.ge         external names for the GitLab v4 API (configuration)
├── frontend/paginas.ge            the pages people use
├── integracoes/                   ADVANCED: adapters for the gitlab-runner protocol
└── e2e/                           Go end-to-end tests against the running application
```

- The domain (`backend/`, `frontend/`) is written in intent phrases: states and transitions
  (`issues` › `começa aberta`, `pode fechar`), roles and inherited membership, invariants
  (`precisa de pelo menos um owner`), restricted visibility, Git repositories, remote work.
- The GitLab API compatibility lives in configuration (`compatibilidade.ge`) and in adapters
  (`integracoes/`), which are marked as the advanced level and may contain protocol details.
  The domain never does.

## Run

```bash
go build -o ge ./cmd/ge
./ge check examples/gitlab-foss/app.ge
GERMANIO_ADMIN_SENHA='choose-a-password' GERMANIO_GIT_RAIZ=./repos \
  ./ge rodar examples/gitlab-foss/app.ge 8080
curl -u root:choose-a-password localhost:8080/api/v4/user
go test ./examples/gitlab-foss/e2e/
```

**Keywords are Portuguese.** See [the index](../README.md#a-note-on-language).
