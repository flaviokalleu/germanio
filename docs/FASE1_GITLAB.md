# FASE 1 — GitLab FOSS (profundidade): auditoria de encerramento

> **Estado: ENCERRADA em 2026-10-02.** O mantenedor aprovou o encerramento formal sem aprovar
> mudanças normativas no escuro: as GEPs 0010–0017 continuam **em teste** (implementadas, não
> normativas) e as GEPs 0018/0019 continuam **rascunho**, todas AGUARDANDO DECISÃO. O status
> delas é independente do encerramento da fase. Evidência final, no commit do encerramento:
> `go test ./...` e `go vet ./...` verdes; E2E do GitLab com o `gitlab-runner` oficial 19.4.1,
> 25 testes aprovados.

Data: 2026-09-28. Referência: `examples/gitlab-foss/`, inventório em
[`research/gitlab/FEATURE_INVENTORY.md`](../research/gitlab/FEATURE_INVENTORY.md).

## Os dez critérios do mandato

| # | Critério | Situação | Evidência |
| --- | --- | --- | --- |
| 1 | Inventário completo revisado | feito | inventário: 42 PASS, 5 PARTIAL, 8 NOT_STARTED, 3 BLOCKED (fora do núcleo); a tabela abaixo diz o que ficou fora e por quê |
| 2 | Itens do escopo resolvidos | feito para o escopo proposto (abaixo) | núcleo v1 `[N]`: todos PASS; NT-01, NT-02, NT-03, AD-01, CI-07, CI-08, RP-06, RP-07 nesta fase |
| 3 | Testes | verdes | `go test ./...`, `go vet ./...` |
| 4 | E2E | verdes | `examples/gitlab-foss/e2e`: fluxos 1–5, issues privadas, labels, milestones, busca, merge requests, webhooks, interface web, atividade, painel, avisos, tags, variáveis, branches protegidas |
| 5 | `gitlab-runner` real | verde | `TestRunnerOficial`: jobs, log, falha, artefatos (zip), variável mascarada |
| 6 | Lacunas revisadas | feito | [`GERMANIO_GAPS.md`](../GERMANIO_GAPS.md): G86, G93, G105–G111, G113 resolvidas; G112 e G114 abertas para decisão |
| 7 | Acoplamento do GitLab no core | nenhum comportamento | busca por `gitlab`, `/api/v4`, `CI_JOB`, `JOB-TOKEN`, `PRIVATE-TOKEN`, `merge_request` no core: só comentários de exemplo. Ponto a observar: a recusa em inglês de push para branch protegida usa a frase do GitLab, por compatibilidade com clientes git |
| 8 | Simplicidade dos `.ge` | feito | domínio (`backend/`, `frontend/`) sem verbos HTTP, JSON, SQL, cabeçalhos, `chamar`, `ambiente.ler` ou lógica; lógica e rotas só nos adaptadores marcados de `integracoes/`. Exceção registrada: G114 |
| 9 | Documentação | feito | norma com as construções em teste marcadas, GEPs 0010–0017, CHANGELOG, inventário, lacunas; `tooling/doctest` e verificador de links (906 links, 0 quebrados) |
| 10 | Encerramento registrado | este documento | aguarda as decisões abaixo |

## Capabilities genéricas desta fase

Cada uma nasceu de um obstáculo do GitLab e serve a outros domínios.

| Capability | Frase | GEP | Onde mais serve |
| --- | --- | --- | --- |
| Efeitos externos depois do commit | (nenhuma: é o comportamento) | G86 | qualquer integração |
| Migração só com rename explícito | `renomeie a para b` | G93 (+ `descarte`, GEP 0010) | toda aplicação com banco |
| Histórico | `guarda histórico` | 0011 | auditoria de ERP/CRM, chat |
| Indicadores | `indicadores` › `total de issues abertas` | 0012 | dashboards |
| Avisos por e-mail | `tenha avisos por e-mail` | 0013 | atendimento, CRM |
| Arquivos de um registro | `anexo arquivo`, `foto imagem` | 0014 | e-commerce, conteúdo, CRM |
| Variáveis das execuções | `pipelines usam as variaveis do projeto` | 0015 | deploys, processamento |
| Branches protegidas por dado | `enviar código para as branches protegidas` | 0016 | hospedagem de código |
| Menções | `pendência para` › `mencionados` | 0017 | chat, documentos |

## Falhas de segurança encontradas e corrigidas nesta fase

- Um link de recuperação de senha era aceito várias vezes sob pedidos simultâneos.
- Listas de dados com `confidencial pode ser vista por` devolviam registros restritos (G106).
- Pendências revelavam o título de registros que a pessoa não podia ver (G107).

## Adiado da FASE 1 (aprovado pelo mantenedor; não descartado)

Registrado no [roadmap](ROADMAP.md) e no inventário com o destino de cada item.

| Item | Destino |
| --- | --- |
| confirmação de e-mail (ID-06) | FASE 5 |
| fork (PR-05); estrelas, tópicos, avatar (PR-06) | FASE 5 |
| edição de arquivo pela web (RP-08) | FASE 3 |
| boards (IS-09) | FASE 3 (sincronização entre pessoas usa a FASE 2) |
| uploads dentro de Markdown (UP-01) | FASE 3 |
| regras avançadas de merge request (MR-07) | BACKLOG |
| DAG avançado de CI (CI-09) | BACKLOG |
| isolamento de CI em containers (CI-10) | FASE 4 |
| 2FA/SSO, SSH, LFS (ID-07, ID-08, RP-10) | fora do núcleo (já marcados) |

## Decisões

| Decisão | Estado |
| --- | --- |
| escopo adiado | **decidido** pelo mantenedor (acima) |
| GEPs 0010–0017 | **AGUARDANDO DECISÃO** — revisão individual com recomendação em [gep/REVISAO_0010_0017.md](gep/REVISAO_0010_0017.md): ACEITAR 0011, 0012, 0014, 0016, 0017; REVISAR 0010, 0013, 0015 |
| G112 (singular de nomes compostos) | **AGUARDANDO DECISÃO** — tratado como identidade de esquema na [GEP 0018](gep/0018-identidade-do-esquema.md) (rascunho); a regra de singular não foi mudada |
| G114 (domínio × protocolo) | **AGUARDANDO DECISÃO** — fronteira proposta na [GEP 0019](gep/0019-fronteira-dos-adaptadores.md) (rascunho); nenhuma mudança da regra de camadas |
| mensagem de protocolo (branch protegida em inglês) | mantida por compatibilidade observável de protocolo; registrada como G115 para ir ao adaptador quando a GEP 0019 for decidida. As mensagens da linguagem continuam em português |

## Evidências preservadas

`go test ./...` e `go vet ./...` verdes; `-race` sem data races no runtime, compilador e
ferramentas; E2E do GitLab verdes, inclusive com o `gitlab-runner` oficial; auditoria das
camadas; nenhum conhecimento específico do GitLab no core (só comentários de exemplo e a frase
de protocolo da G115).
