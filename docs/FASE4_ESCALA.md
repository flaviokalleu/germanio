# FASE 4 — Escala

> **Estado: EM ANDAMENTO desde 2026-10-03.** As GEPs da fase ficam **em teste**, AGUARDANDO
> DECISÃO do mantenedor, como nas fases anteriores.

**Pergunta da fase:** o mesmo `.ge` continua funcionando quando a carga deixa de ser confortável:
muitos dados, muitas pessoas, vários servidores, banco de produção?

**Regra:** escalar é trabalho do runtime. Nenhuma linha de `.ge` da aplicação de referência muda
para ela escalar: o autor não escreve cache, réplica, fila, partição nem pool. O que muda é
configuração de operação (banco, endereços dos servidores), nunca o domínio.

## Aplicações de referência

O GitLab (`examples/gitlab-foss/`) e a Conversa (`examples/conversa/`), sem mudar o domínio.

## Obstáculos conhecidos na abertura

Do fechamento do GitLab e das fases 2 e 3:

| # | Obstáculo | Tipo |
| --- | --- | --- |
| 1 | um processo só: páginas vivas, presença e "digitando…" não atravessam servidores | capability faltante (a central da fase) |
| 2 | PostgreSQL e MySQL suportados no código, mas a suíte roda só em SQLite | lacuna de evidência |
| 3 | a fila de tarefas roda uma por vez: um espelho lento atrasa webhooks | limite de escala |
| 4 | credenciais de saída (espelhos) guardadas sem cifra; não há mecanismo de segredo guardado | segurança |
| 5 | visibilidade conferida por (mudança, pessoa) no fan-out ao vivo | custo em audiências grandes |
| 6 | testes que dependem de tempo falham sob carga | determinismo |
| 7 | `add_spent_time` e operações parecidas leem e depois gravam | concorrência |

## Critérios mensuráveis de encerramento

| # | Critério | Como medir |
| --- | --- | --- |
| 1 | domínio intocado | `git diff` dos `.ge` de domínio do GitLab e da Conversa nesta fase: só correções de produto, nenhuma por escala |
| 2 | banco de produção | suíte do runtime e E2E do GitLab verdes contra PostgreSQL real |
| 3 | vários servidores | dois processos atrás do mesmo banco: uma mudança num aparece ao vivo para quem olha o outro; presença coerente |
| 4 | fila sem bloqueio | uma tarefa lenta não atrasa as outras além de um limite medido |
| 5 | segredos | credenciais de saída cifradas em repouso, com rotação da chave; nada em texto puro no banco |
| 6 | carga | benchmark com dados grandes (10⁵–10⁶ registros) e audiência grande: p99 medido e comparado a Go direto, sem regressão ≥ 1,5× |
| 7 | determinismo | a suíte completa verde 5 vezes seguidas sob carga paralela |

## Diário da fase

### Passo 0 — abertura (2026-10-03)

Plano registrado; obstáculos levantados do fechamento do GitLab.
