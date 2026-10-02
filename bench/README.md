# Benchmarks do Germanio

Suíte permanente exigida por `docs/INTENCAO.md` › Eficiência. O objetivo não é vencer
benchmark: é saber **quanto custa a abstração Germanio** e perceber quando uma mudança a
encarece. Metodologia e fontes: `docs/research/performance/BENCHMARKING.md`.

## Como rodar

```bash
scripts/bench.sh            # MICRO + APLICAÇÃO, 6 repetições (COUNT=n para mudar)
scripts/bench.sh stress     # também o STRESS: 1k e 10k conexões, Germanio e Go direto
CONEXOES="1000" DURACAO=10s scripts/bench.sh stress   # stress reduzido
```

Para 10k conexões, aumente o limite de descritores (`ulimit -n 65536`). O resultado vai para
`bench/resultados/<data>-<commit>.txt` com o contexto obrigatório: data, commit (e se havia
alterações locais), versão do Germanio, versão do Go, SO, CPU, memória, governor da CPU,
dataset e comando. **Um número sem esse contexto não é publicado.** Para comparar duas
revisões, rode as duas na mesma máquina e use `benchstat antigo.txt novo.txt`.

## Categorias

| Categoria | Onde | O que mede |
| --- | --- | --- |
| MICRO | `micro_test.go` | lexer, parser e resolver em programas sintéticos de 10, 100 e 1000 dados (a curva de crescimento); o projeto inteiro (clientes e GitLab) do arquivo ao modelo resolvido, como `ge check`; o interpretador (recursão, aritmética, listas) |
| APLICAÇÃO | `app_test.go` | listar (página 3 de 1000 registros, com total), obter, criar (com validação e unicidade) e a página HTML, **no Germanio e no mesmo app em Go direto** (`baseline/`), chamando o handler sem rede |
| STRESS | `stress/` + `scripts/bench.sh stress` | muitas conexões simultâneas contra o servidor real: req/s, p50/p90/p99/máx, erros e memória do servidor (RSS antes, pico e depois, lida de `/proc`) |

O baseline (`baseline/`) é o app `bench/testdata/clientes.ge` escrito à mão com `net/http`,
`database/sql` e o mesmo driver SQLite, com o mesmo DSN, o mesmo pool (25 conexões), as
mesmas validações, a mesma ordenação e o mesmo tamanho de página. Uma diferença de
configuração entre os dois invalida a comparação; ao mudar `runtime/banco`, atualize o
baseline.

## Limites conhecidos da medição

- O gerador de stress trabalha em **ciclo fechado** (cada conexão espera a resposta antes da
  próxima). Sob sobrecarga isso subestima a latência de cauda (coordinated omission). Use-o
  para comparar Germanio e Go na mesma máquina, não como capacidade absoluta; latência de
  cauda confiável exige um gerador de taxa fixa (wrk2, vegeta).
- Servidor e gerador na mesma máquina disputam CPU.
- `criar` é dominado pela escrita síncrona do SQLite no disco; compare as alocações, não só o
  tempo.
- Máquina com outros processos ativos, governor `powersave` ou turbo variável produzem
  ruído grande: com o mesmo código, duas execuções curtas já diferiram por mais de 2x.
  Resultados oficiais saem de uma máquina ociosa, com `COUNT` ≥ 6 e `benchstat`.

## Ainda não coberto

WebSocket, upload e download de arquivos grandes (streaming), jobs e fila, renderização de
páginas com muitas seções, datasets grandes (milhões de linhas) e o startup do CLI. Entram
junto com a capability que medem, ou quando a auditoria
(`docs/research/performance/AUDITORIA.md`) indicar risco.

## Primeiro baseline (2026-09-28, commit e421c8e)

Arquivo completo: [`resultados/2026-09-28-e421c8e.txt`](resultados/2026-09-28-e421c8e.txt)
(i7-10750H, 12 núcleos, 15 GB, Linux 7.1, Go 1.27.1, governor `powersave`, 6 repetições;
servidor e gerador na mesma máquina). Algumas linhas desse arquivo saíram misturadas com as
mensagens de partida do runtime; os benchmarks silenciam essas mensagens a partir daqui.

| Medida | Germanio | Go direto | Razão |
| --- | --- | --- | --- |
| listar (página 3 de 1000) | ≈ 250 µs | ≈ 146 µs | 1,7× |
| página HTML | ≈ 590 µs | ≈ 220 µs | 2,7× |
| compilar o GitLab inteiro | ≈ 6,7 ms | — | — |
| lista com visibilidade por registro (50 mil linhas, membro de 1 de 50 projetos) | ≈ 32 ms | — | — |
| stress, 1 000 conexões | 18,8 mil req/s, p99 159 ms, 0 erros, RSS pico 129 MB | 31,8 mil req/s, p99 92 ms, 0 erros, 89 MB | 0,59× |
| stress, 10 000 conexões | 18,4 mil req/s, p99 1,75 s, 0 erros, RSS pico 809 MB | 32,2 mil req/s, p99 1,0 s, 0 erros, 540 MB | 0,57× |

## Budgets

Com o baseline medido, os limites iniciais são de regressão, não de meta: uma mudança que
piore uma medida acima de **1,5×** em relação ao último baseline (ou a razão Germanio/Go acima
de **3×** em listar e página) precisa ser registrada em `GERMANIO_GAPS.md` e decidida antes de
entrar (norma: Regressão consciente). Metas absolutas ficam para quando houver baselines em
mais de uma máquina.

## Tempo real (FASE 2)

```bash
go build -o /tmp/chat ./bench/baseline/chat && go build -o /tmp/tempo_real ./bench/tempo_real
/tmp/chat -addr 127.0.0.1:18090 -db /tmp/chat.db & PID=$!
/tmp/tempo_real -c 2000 -remetentes 4 -mensagens 400 -taxa 200 -pid $PID -rotulo go-chat
```

Cada assinante confere a ordem por canal: o relatório traz entregas, faltando, fora de ordem,
repetidas, ressincronizações, latência de entrega (p50/p90/p99/máximo) e memória do servidor. O
alvo e os endereços são flags, para medir a aplicação de referência em Germanio do mesmo jeito.
