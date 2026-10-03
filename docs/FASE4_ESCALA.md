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

### Passo — segredos (obstáculos 4 e 7; critério 5)

**Segredos guardados** ([GEP 0049](gep/0049-segredos-guardados.md), em teste, sem frase nova): o
que a aplicação precisa ler de volta para falar com outro sistema — a credencial dos espelhos, o
`token oculto` dos webhooks, o `valor texto oculto` das variáveis de CI, o segredo dos dois fatores
— fica cifrado no banco pelo core, sem mudar o domínio: `oculto` num texto escrito pelas pessoas já
diz "só o sistema usa". AES-256-GCM da biblioteca padrão, nonce aleatório, chave derivada com HKDF
de `GERMANIO_SEGREDO`, formato versionado `ge1:<id da chave>:…` preso à coluna
(`runtime/cofre`, `runtime/banco/segredos.go`). Na partida, o que estava em claro é cifrado e o que
foi cifrado com uma chave anterior (`GERMANIO_SEGREDO_ANTERIOR`) é cifrado de novo, em lotes, linha
a linha só se ainda tem o valor lido (sem parar o serviço, idempotente); no SQLite o arquivo é
refeito uma vez para o texto antigo não sobrar em páginas livres. Produção (`GERMANIO_PRODUCAO=1`)
com segredos e sem chave não parte; chave desconhecida também não. Filtrar, pesquisar, ordenar ou
tornar único um campo cifrado é recusado (na compilação e no banco), com o motivo. Medido nos
testes: o arquivo do banco (e o WAL) não contém nenhum dos segredos, nem depois da migração de
linhas antigas.

**Soma que nunca fica negativa** ([GEP 0050](gep/0050-soma-nunca-negativa.md), em teste;
`regras` › `não pode ficar com tempo gasto negativo`): o `add_spent_time` do adaptador lia o total
e depois gravava; agora a regra é do domínio e o core a confere na mesma transação da mudança, com
o registro travado antes de gravar. Medido: oito descontos de 30 minutos ao mesmo tempo com 1 hora
gasta — antes passavam os oito; agora passam dois e seis são recusados (E2E
`TestDescontarTempoAoMesmoTempo`; no runtime, estoque com 5 e oito saídas simultâneas: cinco
passam).

Pendente: envelope com KMS/Vault para instalações grandes; ligar o segredo à linha (hoje à coluna);
em PostgreSQL as versões antigas das linhas somem só com o vacuum do próprio banco; a trava
`FOR UPDATE` da GEP 0050 testada contra PostgreSQL real depende da suíte em PostgreSQL (obstáculo 2).
