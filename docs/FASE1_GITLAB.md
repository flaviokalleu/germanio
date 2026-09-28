# FASE 1 — GitLab FOSS (profundidade): auditoria de encerramento

> **Estado: pronta para encerramento, aguardando decisões do mantenedor** (lista no fim).
> A fase só é registrada como encerrada quando essas decisões forem tomadas. A FASE 2
> (tempo real pesado) não começa antes disso.

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

## Fora do escopo desta fase (proposta, para confirmar)

| Item | Por quê fica para depois |
| --- | --- |
| ID-06 confirmação de e-mail | pequeno, mas precisa de decisão de produto (bloquear login até confirmar?) |
| PR-05 fork, PR-06 estrelas/tópicos/avatar | amplitude, não profundidade; avatar depende da GEP 0014 aceita |
| RP-08 editar arquivo pela web | interface interativa (FASE 3) |
| IS-09 boards, MR-07 regras avançadas de merge | interface interativa (FASE 3) e regras de aprovação (GEP própria) |
| CI-09 DAG/rules/manual, CI-10 containers | execução avançada; isolamento é infraestrutura (FASE 4) |
| UP-01 uploads em markdown | usa a GEP 0014; falta a referência no texto formatado |
| ID-07, ID-08, RP-10 | já marcados fora do núcleo (2FA/SSO, SSH, LFS) |

## Decisões do mantenedor que encerram a fase

1. GEPs em teste: 0010 (`descarte`), 0011, 0012, 0013, 0014, 0015, 0016, 0017 — aceitar, mudar ou
   recusar.
2. Confirmar (ou mudar) a lista "fora do escopo" acima.
3. G112: regra do singular de nomes compostos (muda nomes de tabela).
4. G114: se vocabulário, prefixo e forma de token de integração pertencem ao adaptador.
