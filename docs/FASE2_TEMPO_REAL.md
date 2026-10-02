# FASE 2 — Tempo real pesado: preparação

> **Estado: preparada, NÃO iniciada.** Pela ordem obrigatória, nada disto é implementado antes
> do encerramento formal da FASE 1 ([FASE1_GITLAB.md](FASE1_GITLAB.md)). Este documento deixa
> a fase pronta para começar.

**Pergunta da fase:** Germanio consegue construir aplicações de tempo real complexas sem que
o autor `.ge` programe infraestrutura de tempo real?

**Regra:** WebSocket, SSE, polling, pub/sub, filas e goroutines são mecanismos. O `.ge` da
aplicação de referência não pode conter nenhum deles. A intenção é "esta informação precisa
permanecer atualizada", e o transporte é escolhido pelo runtime.

## A aplicação de referência: "Conversa"

Um chat de equipes com espaços, canais e mensagens diretas, complexo o bastante para pressionar
cada ponto da fase. O que cada comportamento exercita:

| Comportamento | O que pressiona |
| --- | --- |
| espaços (equipes) com membros e papéis; canais públicos e privados; conversas diretas | autorização em tempo real (quem recebe cada evento) |
| enviar, editar e apagar mensagens; respostas em fio | ordenação, entrega, atualização parcial |
| histórico rolável e pesquisa | histórico, paginação por cursor, busca |
| menções e avisos (GEP 0017 e a revisão da GEP 0013) | notificações, entrega a quem não está olhando |
| presença (online, ausente, offline) e "digitando…" | estado efêmero, estado online/offline |
| lido/não lido por canal e contadores | estado por pessoa, agregação incremental |
| anexos (GEP 0014) | arquivos no fluxo em tempo real |
| entrar e sair de um canal, ser removido de um espaço | autorização que muda com conexões abertas |
| sair do ar e voltar (celular no metrô) | reconexão, sincronização, recuperação após falha |
| milhares de pessoas num canal grande | milhares de conexões, fan-out, backpressure |
| muitas pessoas escrevendo ao mesmo tempo | concorrência, ordem total por canal |

## O que existe hoje (auditoria factual, 2026-09-28)

- `runtime/servidor/websocket.go` (215 linhas):
  - hub em memória com `Broadcast` global, sem salas nem destinatários (G66);
  - buffer de 64 mensagens por conexão; buffer cheio descarta em silêncio;
  - protocolo WebSocket escrito à mão;
  - um único processo.
- Segurança já garantida: `/ws` exige sessão e mesma origem; avisos de mudança não levam dados do
  registro.
- As páginas de intenção são desenhadas no servidor, sem atualização parcial, e o hub não está
  ligado a elas.
- O que ajuda: efeitos depois do commit (G86), fila persistente com novas tentativas,
  pendências e menções (GEP 0009/0017), histórico (GEP 0011), arquivos (GEP 0014), indicadores
  (GEP 0012).

## Capabilities que provavelmente faltam (hipóteses a confirmar escrevendo a aplicação)

1. **Informação que permanece atualizada**, como intenção em páginas e listas, com atualização
   parcial (sem redesenhar a página).
2. **Destinatários derivados das permissões:** um evento só chega a quem pode ver o registro,
   inclusive quando o acesso muda com a conexão aberta.
3. **Ordem por canal:** uma sequência por conversa, idêntica para todos.
4. **Entrega com retomada:** cursor por pessoa, reenvio do que faltou ao reconectar, sem perda
   nem duplicação visível.
5. **Presença e estado efêmero** (digitando), sem gravar no banco a cada tecla.
6. **Lido/não lido e contadores** mantidos de forma incremental, não recontados.
7. **Backpressure:** um cliente lento não atrasa os outros nem cresce a memória sem limite. Em
   vez de descartar em silêncio, o cliente recebe um pedido de ressincronização.
8. **Vários processos** (fan-out entre servidores): possivelmente FASE 4, a decidir.

Cada uma que se confirmar vira uma GEP antes de qualquer sintaxe.

## Critérios mensuráveis de encerramento (propostos, a confirmar)

Os valores de carga são alvos iniciais para a máquina de benchmark do projeto (12 núcleos). As
latências são comparadas a um servidor de chat equivalente escrito em Go, como no resto da
suíte, e nunca a números inventados.

| # | Critério | Como medir | Alvo inicial |
| --- | --- | --- | --- |
| 1 | conexões simultâneas | stress com N clientes conectados e ativos | 10 000 sem erro; memória por conexão medida e registrada |
| 2 | latência de entrega (do commit ao cliente) | p50/p99 a 1 000 e 10 000 conexões | p99 ≤ 2× o baseline em Go |
| 3 | ordem por canal | remetentes concorrentes; cada cliente compara a sequência | 100% das sequências iguais |
| 4 | retomada após queda | derrubar clientes e o servidor no meio do fluxo | 0 mensagens perdidas; 0 duplicadas mostradas |
| 5 | autorização em tempo real | teste de propriedade: membros entram e saem com conexões abertas | 0 eventos entregues a quem não vê o registro |
| 6 | backpressure | 1 cliente parado entre 1 000 ativos | latência dos ativos dentro do critério 2; memória limitada |
| 7 | presença | desconexão sem aviso | offline visível para os outros em até 2 intervalos de batimento |
| 8 | intenção pura | varredura do `.ge` da referência | nenhuma palavra de transporte (WebSocket, SSE, fila, canal técnico, JSON…) |
| 9 | regressão | suíte de benchmarks existente | nenhuma piora ≥ 1,5× sem investigação registrada |

## Preparação dos benchmarks (feita)

- `bench/tempo_real`: gerador de conexões persistentes. Abre N assinantes, dispara mensagens a
  uma taxa fixa e mede, em cada assinante, a latência de entrega, as mensagens faltando, fora de
  ordem ou repetidas, os pedidos de ressincronização e a memória do servidor.
- `bench/baseline/chat`: o mesmo chat escrito diretamente em Go:
  - sequência por canal e mensagens no SQLite;
  - fan-out por WebSocket e retomada por cursor (`?desde=N`);
  - buffer limitado: o cliente lento é fechado pedindo ressincronização, sem perda silenciosa.
- Primeiro baseline:
  [`bench/resultados/2026-10-02-tempo-real-baseline.txt`](../bench/resultados/2026-10-02-tempo-real-baseline.txt).
  Com 500 conexões, p99 de 20,7 ms e pico de 49 MB; com 2 000, p99 de 115 ms e pico de 146 MB;
  0 faltando, 0 fora de ordem e 0 repetidas nas duas medições. A aplicação Conversa em Germanio
  será medida com o mesmo gerador.

## Ordem de trabalho quando a fase começar

1. Escrever a Conversa em Germanio com o que existe, anotando cada parede. Nenhuma sintaxe nova
   nesta etapa.
2. Classificar as paredes (capability genérica? domínio? transporte?) e escrever as GEPs.
3. Implementar os mecanismos no core, com testes de propriedade para ordem, entrega e
   autorização.
4. Os benchmarks da tabela, contra o baseline em Go.
5. Auditoria e registro do encerramento, como na FASE 1.
