# Solid: reatividade fina, signals e o que o Germanio pode derivar sozinho

Data: 2026-09-28
Status: pesquisa concluída; nada implementado a partir dela.

## Fontes consultadas

- Repositório: https://github.com/solidjs/solid (arquivo `packages/solid/src/reactive/signal.ts`)
- Reatividade fina: https://docs.solidjs.com/advanced-concepts/fine-grained-reactivity
- Stores: https://docs.solidjs.com/concepts/stores
- Busca de dados (`createResource`): https://docs.solidjs.com/guides/fetching-data
- `renderToStream`: https://docs.solidjs.com/reference/rendering/render-to-stream
- Release 1.3 "Spice Must Flow" (streaming de HTML): https://github.com/solidjs/solid/releases/tag/v1.3.0
- Solid Router, revalidação: https://docs.solidjs.com/solid-router/data-fetching/revalidation
- dom-expressions: https://github.com/ryansolid/dom-expressions
- Convex, consultas reativas por conjunto de leitura: https://stack.convex.dev/how-convex-works
- Noria (OSDI 2018): https://pdos.csail.mit.edu/papers/noria:osdi18.pdf

Código do Germanio lido: `runtime/servidor/websocket.go`, `runtime/servidor/servidor.go`
(chamadas a `Broadcast`), `runtime/servidor/eventos.go` (`emit`), `runtime/servidor/transacao.go`
(`afterCommit`), `runtime/servidor/paginas.go`, `runtime/servidor/renderizador.go`
(`connectWS`), `runtime/auth/auth.go` (`/ws` sem token), `compiler/ast/autorizacao.go`.

## 1. O modelo do Solid

- **Signals e observadores.** Um signal guarda um valor com leitura e escrita separadas; uma
  computação (efeito, memo) que lê um signal é registrada como assinante; a escrita notifica
  os assinantes. Um `currentSubscriber` global indica quem está executando
  (https://docs.solidjs.com/advanced-concepts/fine-grained-reactivity).
- **Componentes rodam uma vez.** O componente é uma função de montagem, não de renderização;
  depois disso só as computações ligadas aos signals alterados reexecutam: "no Solid as
  atualizações são feitas no atributo que precisa mudar ... o React reexecutaria o componente
  inteiro" (mesma fonte).
- **O algoritmo** (`signal.ts`): globais `Owner` (escopo de posse, para limpeza) e `Listener`
  (quem rastreia agora); estados `STALE` e `PENDING`; ao escrever, marca-se toda a cadeia a
  jusante e depois executa-se de cima para baixo, o que evita *glitches* (leitura de valor
  intermediário inconsistente); `createMemo` aceita `equals` e não propaga se o valor não
  mudou; `batch` junta várias escritas numa só propagação; a cada execução as fontes antigas
  são limpas (`cleanNode`), de modo que as dependências são **dinâmicas**: dependem do caminho
  que a execução tomou (https://github.com/solidjs/solid/blob/main/packages/solid/src/reactive/signal.ts).
- **Stores.** Proxies com signals criados preguiçosamente por propriedade lida dentro de um
  escopo rastreado; setter por caminho (`setStore("users", 0, "username", ...)`); `reconcile`
  compara dados novos com os existentes e "só dispara atualizações onde os valores mudaram"
  (https://docs.solidjs.com/concepts/stores). `reconcile` é a peça que liga o servidor ao
  cliente: chega um JSON novo e só o que mudou se propaga.
- **Resources.** `createResource` é um signal assíncrono com `loading`, `error`, `state`
  (`unresolved`, `pending`, `ready`, `refreshing`, `errored`), `latest`, `refetch` e `mutate`
  (atualização otimista); `Suspense` mostra o fallback do limite mais próximo
  (https://docs.solidjs.com/guides/fetching-data).
- **Revalidação no router.** Consultas têm chaves (`query.key`, `query.keyFor(args)`);
  "depois de uma action concluir com sucesso, o Solid Router revalida automaticamente todas as
  queries ativas na página" (https://docs.solidjs.com/solid-router/data-fetching/revalidation).
  Como no SvelteKit, a revalidação depois de uma mutação é grossa por padrão.
- **SSR e streaming.** `renderToStream` renderiza de forma síncrona o que pode, incluindo os
  fallbacks de `Suspense`, e continua transmitindo dados e HTML de cada recurso assíncrono
  quando conclui (https://docs.solidjs.com/reference/rendering/render-to-stream); desde a 1.3
  o Solid transmite também o HTML, não só os dados, de modo que o conteúdo aparece antes de o
  JavaScript carregar (https://github.com/solidjs/solid/releases/tag/v1.3.0).
- **dom-expressions.** O runtime de renderização é separado da reatividade (funciona com S.js,
  MobX, Knockout...); o plugin de Babel compila o JSX analisando o template inteiro, porque
  "a pré-compilação permite o melhor desempenho"; escapa inserções e atributos contra XSS
  (https://github.com/ryansolid/dom-expressions). O desenho em camadas (compilação do template
  / runtime de inserção / núcleo reativo) é o ponto; o JSX não.

## 2. Onde o Solid e o Germanio divergem

O Solid resolve a reatividade **dentro de um processo** (o navegador), com dados que já estão
na memória. O problema do Germanio é outro: os dados estão no banco do servidor, cada pessoa
vê um subconjunto diferente (grants, visibilidade, `seu/seus`), e a mudança vem de outra
pessoa. O grafo que importa não é "signal → nó do DOM", e sim "linha do banco → consulta da
página de cada espectador → região do HTML". Três consequências:

1. **Dependências dinâmicas não são necessárias.** O Solid reconstrói as fontes a cada
   execução porque o código JavaScript pode ler coisas diferentes em caminhos diferentes. Uma
   região de página do Germanio lê um conjunto fixo, conhecido na compilação. Um grafo
   estático basta e é explicável.
2. **A granularidade útil é a região, não o nó do DOM.** Reexecutar a consulta de "vendas do
   mês" e trocar o número custa o mesmo que trocar a região inteira; o custo dominante é a
   consulta, não o DOM.
3. **A autorização entra no grafo.** No Solid todo assinante pode ler o valor. No Germanio,
   o valor enviado a cada espectador precisa ser calculado com as permissões dele.

## 3. A pergunta: quanto da reatividade pode ser derivado?

Exemplo:

```ge
página Dashboard
    mostre
        total de clientes
        vendas do mês
        receita do mês
```

Supondo que cada linha resolva para um agregado sobre uma entidade (as formas exatas ainda não
existem: G62 "indicadores" está OPEN em `GERMANIO_GAPS.md`), o compiler conhece, sem nada
escrito pelo usuário:

| Região | Lê | Filtro | Função |
|--------|----|--------|--------|
| total de clientes | `cliente` | visibilidade da pessoa | contagem |
| vendas do mês | `venda` | visibilidade + `criado_em` no mês corrente | contagem |
| receita do mês | `venda.valor` | idem | soma |

A resposta é: **toda** a reatividade de leitura pode ser derivada, porque a página é uma
função pura de (dados do banco, pessoa, parâmetros da URL, relógio). O que não pode ser
derivado é a política de frescor quando o custo passa do limite, e isso deve ser decidido pelo
runtime por medição, não pelo usuário (seção 5).

### 3.1 O grafo estático

Para cada região *r* da página *p*: `leituras(r) = {(entidade, campos, filtro)}`. As
dependências indiretas também são estáticas e vêm do `ast.App`:

- Referências mostradas por nome (`nameOf` em `paginas.go`): a lista de projetos lê também
  `grupo.nome`; renomear um grupo invalida a lista.
- Visibilidade e grants: a lista depende das tabelas de membros (`MemberModel`) e dos campos
  de visibilidade (`Entity.Visibility`, tetos `VisibilityCeiling`); adicionar alguém como
  membro muda o que essa pessoa vê.
- Estados (`começa ativo`, transições) e somente-leitura (`ReadOnly`) mudam as ações
  disponíveis, que também são parte da região.
- O relógio: "do mês" depende da virada do mês; é uma dependência temporal conhecida, que o
  runtime agenda.

`ge explain` pode mostrar, para cada região, "atualiza quando: cliente é criado ou excluído;
alguém ganha ou perde acesso a clientes".

### 3.2 A invalidação por entidade

O ponto de corte já existe no código: `intentAPI.emit` (em `eventos.go`) é chamado em
`criar`, `editar`, `excluir` e nas transições, dentro da transação da requisição; e
`afterCommit` (em `transacao.go`) executa trabalho só depois do commit. O mecanismo:

1. Cada mutação produz, depois do commit, um fato `(entidade, ação, id, campos alterados,
   valores relevantes antes/depois)`.
2. Um índice invertido `entidade → regiões assinadas` (montado a partir do grafo estático e
   das conexões abertas) seleciona as regiões candidatas.
3. Um filtro barato elimina falsos positivos quando possível: se a região lê só
   `venda` do mês corrente e a venda alterada é de outro mês, nada acontece; se a região lê só
   `nome` e mudou `status`, nada acontece.
4. Para cada região restante, o servidor recalcula **com a sessão do espectador** (a mesma
   chamada interna que `paginas.go` já faz com o cookie) e compara com o último valor enviado
   (a igualdade de `createMemo`/`$derived`); só envia se mudou.

O passo 3 é o equivalente estático da "sobreposição do conjunto de escrita com o conjunto de
leitura" que o Convex faz em runtime: o Convex registra o *read set* de cada consulta e, a
cada entrada no log de transações, verifica se ela se sobrepõe a alguma assinatura ativa,
reexecuta a consulta e mantém todas as consultas de um cliente "no mesmo timestamp"; e isso só
é preciso porque o resultado da consulta é determinado pelos argumentos e pelas leituras do
banco (https://stack.convex.dev/how-convex-works). O Germanio obtém a mesma propriedade sem
instrumentar consultas, porque a página não tem código arbitrário.

### 3.3 O servidor empurra

Duas formas, com custos diferentes:

- **Sinal de invalidação** ("a região `vendas-do-mes` mudou"), e o cliente busca o HTML da
  região com os próprios cookies. Permissões aplicadas naturalmente pela requisição normal;
  nenhum dado trafega no WebSocket; custo de uma requisição por cliente afetado.
- **Conteúdo pronto** (o HTML ou o valor da região, calculado pelo servidor por espectador).
  Menos idas e voltas, mas o servidor calcula e guarda o último valor por conexão.

A primeira é a certa para começar (simples, segura, e é o que o Turbo faz com "refresh", ver
REACTIVITY.md). A segunda é otimização quando a medição mostrar necessidade.

### 3.4 Consistência

- **Só depois do commit.** Um aviso antes do commit pode mostrar dado que será desfeito. O
  `afterCommit` já existe para isso.
- **Mesmo instante para a página inteira.** Se "vendas do mês" e "receita do mês" forem
  recalculadas em momentos diferentes, podem mostrar números incoerentes entre si (o
  *glitch* do Solid, entre regiões). Solução simples: invalidar juntas as regiões que leem a
  mesma entidade e recalculá-las numa mesma leitura (uma transação de leitura, ou uma
  requisição que devolve as várias regiões).
- **Ordem e perdas.** O cliente pode perder mensagens (reconexão, buffer cheio: hoje
  `Broadcast` descarta a mensagem quando o canal de 64 está cheio). Com sinal de invalidação
  isso é inofensivo se a reconexão invalidar tudo; com conteúdo pronto é preciso versão por
  região.
- **O próprio autor.** Quem fez a mudança já recebe a página nova pelo PRG; o Turbo evita
  mandar o refresh a quem originou a requisição (ver REACTIVITY.md).

### 3.5 Custo

- Uma mutação em `venda` com N espectadores do Dashboard custa até N recálculos de dois
  agregados. Com agrupamento por "mesma consulta com a mesma visibilidade" (por exemplo,
  todos os administradores veem o mesmo total) o custo cai para o número de classes de
  visibilidade distintas, não de pessoas. Com dados privados por pessoa (`seus pedidos`) não
  há compartilhamento.
- **Debounce/coalescência.** Uma importação de 10 mil vendas não pode gerar 10 mil rodadas;
  juntar invalidações numa janela curta (o Turbo faz debounce dos refreshes, ver
  REACTIVITY.md).
- **Agregados caros.** Recalcular `soma` do mês a cada venda é O(vendas do mês). A
  alternativa é manutenção incremental (somar o delta), que é o que o Noria faz em geral para
  consultas parametrizadas, com estado parcial e evicção
  (https://pdos.csail.mit.edu/papers/noria:osdi18.pdf). Para contagem e soma o delta é trivial
  e exato; para média, mínimo, máximo com exclusão, e filtros de visibilidade, não é. Começar
  por recálculo com debounce; incremental só com medição.
- **Estado por conexão.** Guardar o último valor enviado custa memória por conexão × região.
  O sinal de invalidação não guarda nada.

## 4. O que já existe no Germanio (confirmado no código)

- `runtime/servidor/websocket.go`: um hub em memória, handshake manual, `Broadcast` para
  **todos** os clientes, sem salas, sem filtro por pessoa; o leitor descarta tudo o que chega;
  não há ping/pong nem tratamento de frame de fechamento.
- `runtime/auth/auth.go` deixa `/ws` passar sem token; não há verificação de `Origin` no
  handshake.
- `runtime/servidor/servidor.go` faz `Broadcast` de `criar`, `atualizar`, `deletar` e
  `restaurar` **com o registro inteiro** em `Data`, na API legada (`/api/`), mais presença e
  eventos do WhatsApp. Como o socket não é autenticado nem filtrado, qualquer pessoa conectada
  recebe o conteúdo de qualquer registro alterado, independentemente das permissões. Na API de
  intenção (`/_ge/api/`, `intencao.go`) não há `Broadcast`: as páginas de intenção não
  recebem nada.
- O cliente legado (`connectWS` em `renderizador.go`) recarrega a lista inteira do modelo e
  refaz todos os gráficos a cada mensagem com `model` (invalidação por entidade, sem filtro);
  reconecta a cada 2 s.
- `eventos.go`/`emit` e `transacao.go`/`afterCommit` já dão o ponto certo e transacional para
  publicar mudanças da API de intenção.

## Para o Germanio

### ADOTAR

- **Grafo de leitura estático por região de página.** Problema: páginas de intenção não se
  atualizam e o usuário não deve configurar o que atualizar. Solução: derivar
  `leituras(região)` do `ast.App` (entidade, campos, filtros, referências por nome, membros,
  visibilidade, relógio) numa fase de análise; mostrar em `ge explain`. Afeta
  `compiler/ast/intencao.go` (`PageDecl` ganha regiões quando G62 for resolvido) e um novo
  passo em `compiler/`. Fonte: o contraste com as dependências dinâmicas de `signal.ts`
  (https://github.com/solidjs/solid/blob/main/packages/solid/src/reactive/signal.ts).
- **Publicar mudanças só depois do commit, a partir de `emit`.** Problema: a API de intenção
  não avisa ninguém; a legada avisa todos os sockets, com o registro inteiro e sem filtro de permissão. Afeta
  `runtime/servidor/eventos.go`, `transacao.go`. Fonte: consistência do Convex
  (https://stack.convex.dev/how-convex-works).
- **Não propagar se o valor não mudou.** Igualdade como em `createMemo` `equals`; evita
  tráfego e redesenho. Afeta o futuro mecanismo de regiões.

### ADAPTAR

- **`reconcile` como troca de região por HTML do servidor.** Em vez de um store no cliente,
  o cliente substitui a região (idealmente com morph, ver REACTIVITY.md). Mesmo efeito:
  só o que mudou muda. Fonte: https://docs.solidjs.com/concepts/stores.
- **Estados de resource (`pending`, `refreshing`, `errored`) como estados padrão de região.**
  Toda região que atualiza deve ter a aparência de "atualizando" e de "falhou, tentar de
  novo" gerada pelo Germanio, sem declaração. Fonte: https://docs.solidjs.com/guides/fetching-data.
- **Streaming de regiões lentas.** Renderizar o esqueleto e as regiões rápidas, transmitir as
  lentas depois (como `renderToStream`). Útil para dashboards com agregados caros; só depois
  de medir. Fonte: https://docs.solidjs.com/reference/rendering/render-to-stream.
- **Batch.** Juntar invalidações da mesma transação e de uma janela curta numa só rodada.

### EVITAR

- **Signals, stores ou resources na linguagem.** São mecanismo do cliente; no Germanio o
  usuário declara o que aparece.
- **Rastreamento dinâmico de dependências no servidor.** Desnecessário com páginas fechadas e
  mais difícil de explicar.
- **Enviar registros pelo WebSocket sem autorização por espectador.** É o que a API legada faz
  hoje: vazamento de dados para qualquer conexão. Afeta `servidor.go` e `websocket.go`
  (problema independente desta pesquisa; merece item em `GERMANIO_GAPS.md`).
- **JSX ou compilação de template por aplicação no cliente.** dom-expressions é ótimo para
  autores de framework; o Germanio precisa de um script genérico de troca de regiões.

### INVESTIGAR

- **Agrupar espectadores por classe de visibilidade.** Quantas classes distintas existem na
  prática (GitLab: membros por projeto) e se a deduplicação compensa.
- **Manutenção incremental de contagem/soma.** Exata para contagem e soma sem visibilidade;
  medir contra recálculo com debounce antes de adotar. Fonte:
  https://pdos.csail.mit.edu/papers/noria:osdi18.pdf.
- **Dependências temporais ("do mês", "hoje").** Agendar a invalidação na virada do período,
  no fuso de quem vê.
- **Várias instâncias do servidor.** O hub é em memória; com mais de um processo, a
  invalidação precisa passar pelo banco (ou pela fila de tarefas existente) para chegar a
  todos.
