# FASE 3 — Interface altamente interativa

> **Estado: ENCERRADA em 2026-10-02.** GEP da fase (0023, por estado) em teste, AGUARDANDO
> DECISÃO. Critérios 1–7 com evidência; regressão: nenhuma piora ≥ 1,5×
> ([`2026-10-02-494cfa0.txt`](../bench/resultados/2026-10-02-494cfa0.txt)). Os boards também
> estão no GitLab (as issues de cada projeto por estado).

**Pergunta da fase:** Germanio constrói experiências de interface complexas, e não só sistemas
administrativos, sem virar React traduzido?

**Regra:** o autor descreve o comportamento da interface (o que se move, o que se seleciona, o
que se desfaz), nunca a manipulação de DOM, componentes com estado à mão ou eventos de teclado.
O toolchain pode gerar JavaScript; o autor não escreve JavaScript.

## A aplicação de referência: "Quadro"

Um quadro de tarefas em colunas (o que a FASE 1 adiou como *boards*), com:

| Comportamento | O que pressiona |
| --- | --- |
| colunas que são os estados de um dado (`começa`, `pode`) | estado derivado do domínio |
| arrastar um cartão de uma coluna para outra = fazer a transição | drag-and-drop ligado a regras (quem pode mover) |
| reordenar cartões dentro da coluna | ordem persistida, atualização local |
| abrir um cartão num painel sem sair do quadro | modais e overlays |
| editar o título no próprio cartão | edição em linha, formulário curto |
| selecionar vários cartões e mover juntos | seleção |
| desfazer e refazer a última ação | undo/redo |
| atalhos de teclado (novo cartão, mover, desfazer) | atalhos |
| filtros por responsável e etiqueta, sem recarregar | estado local derivado |
| outras pessoas movendo cartões ao mesmo tempo | sincronização (usa as páginas vivas da FASE 2) |
| celular e teclado/leitor de tela | responsividade e acessibilidade |

Também da FASE 1: editar arquivo pela web (RP-08) e uploads dentro do Markdown (UP-01); e da
FASE 2: "digitando…" e o teste de reconexão num navegador real.

## Critérios mensuráveis de encerramento (propostos)

| # | Critério | Como medir |
| --- | --- | --- |
| 1 | intenção pura | o `.ge` do Quadro sem DOM, evento de teclado, estado de componente ou JavaScript |
| 2 | interação funciona num navegador real | testes de navegador (arrastar, painel, atalhos, desfazer) |
| 3 | regras valem na interface | quem não pode fazer a transição não consegue arrastar; o servidor recusa mesmo se o cliente tentar |
| 4 | acessibilidade | navegação só por teclado; papéis ARIA; leitor de tela anuncia o movimento |
| 5 | atualização local | mover um cartão não recarrega a página; outras pessoas veem pela página viva |
| 6 | desfazer | desfazer devolve o estado anterior também no servidor, respeitando as regras |
| 7 | peso | JavaScript gerado medido e limitado; carregamento comparado a uma página equivalente |

## Diário da fase

### Passo 1 — o Quadro com o que existe (2026-10-02)

O domínio coube na linguagem atual:
- quadros com membros e papéis;
- cartões com estados e transições (`começa pendente`, `pode comecar/concluir/reabrir`, com quem
  e quando);
- responsáveis, menções, histórico e presença.

Obstáculos:

| # | Obstáculo | Tipo | Encaminhamento |
| --- | --- | --- | --- |
| 1 | um estado não pode ter duas palavras (`a fazer`) | limite da linguagem | registrado; contornado com `pendente` |
| 2 | não há como mostrar um dado em colunas por estado nem mover por arraste | capability faltante (a central da fase) | GEP 0023, em teste: feita (`TestPorEstado`) |
| 3 | os títulos das seções eram o singular mais "s" ("Cartaos") | bug de interface | corrigido: o plural do dado como o autor escreveu |
| 4 | edição no próprio cartão, painel sem sair do quadro, seleção, desfazer, atalhos | capabilities faltantes | desfazer e atalhos de movimento: feitos (passo 4); painel com edição: feito (passo 5); seleção e filtro local: feitos (passo 6) |

### Passo 2 — por estado (2026-10-02)

GEP 0023: `cartoes por estado` abaixo de `mostre`, onde já fica `20 por página`.
- As colunas são os estados: o inicial, depois os destinos das transições.
- Cada cartão tem como botões só os movimentos que aquela pessoa pode fazer a partir do estado
  dele. Isso vale para teclado, leitor de tela e páginas sem JavaScript.
- Arrastar envia o mesmo botão do movimento. A coluna não aceita um cartão sem aquele movimento.
- O servidor confere de novo, e um movimento forjado é recusado.
- O quadro se atualiza pela página viva.

### Passo 3 — navegador de verdade (2026-10-02)

`TestNavegador` dirige um Chromium sem tela (`playwright-core`). É opcional: roda com
`GERMANIO_NAVEGADOR` e `GERMANIO_NAVEGADOR_MODULOS`, e é pulado sem eles, como o do
`gitlab-runner`. Ele cobre, só pelas páginas:
- cadastrar-se, criar um quadro e um cartão;
- **arrastar** o cartão para "concluido";
- outra aba ver a mudança **sem recarregar**;
- uma aba **offline** durante uma mudança recuperá-la ao voltar.

O teste encontrou um bug: quando a nova busca falhava (offline), a atualização era descartada.
Agora ela é tentada de novo, e voltar a ficar online dispara uma atualização.

```bash
GERMANIO_NAVEGADOR=<chromium headless> GERMANIO_NAVEGADOR_MODULOS=<node_modules com playwright-core> \
  go test -run TestNavegador ./runtime/
```

### Passo 4 — teclado e desfazer (2026-10-02)

Sem sintaxe nova, como parte da GEP 0023:
- as setas movem o cartão em foco para a coluna vizinha, se aquele movimento é um dos botões
  dele;
- todo movimento é anunciado numa região `aria-live`;
- Ctrl+Z, ou o botão "Desfazer", faz o movimento de volta, e o servidor confere como qualquer
  outro movimento.

Provado no navegador (`TestNavegador`): a seta move, o anúncio sai, o botão aparece e Ctrl+Z
devolve o cartão, com a outra aba vendo cada passo.

### Passo 5 — painel (2026-10-02)

O link do cartão abre o registro num `<dialog>` nativo sobre o quadro: o foco fica dentro e Esc
fecha. Os formulários de dentro (editar, ações) enviam sem sair do quadro: o painel se recarrega e
o quadro se atualiza. O link continua sendo um link, sem JavaScript ou com Ctrl/clique do meio.
Provado no navegador: abre, edita o título (o quadro mostra o novo) e Esc fecha.

### Passo 6 — seleção e filtro local (2026-10-02)

- Espaço ou Ctrl+clique seleciona; mover um cartão selecionado move todos os selecionados que têm
  aquele movimento; desfazer devolve o grupo.
- Um campo "Filtrar" esconde os cartões que não batem, sem pedir nada ao servidor, e o filtro
  sobrevive às atualizações ao vivo.
- Provado no navegador, inclusive que o filtro não faz nenhuma requisição.

### Passo 7 — critérios (2026-10-02)

| # | Critério | Situação | Evidência |
| --- | --- | --- | --- |
| 1 | intenção pura | **cumprido** | o `.ge` do Quadro não tem DOM, evento, estado de componente nem JavaScript; a única linha de interface é `cartoes por estado` |
| 2 | interação num navegador real | **cumprido** | `TestNavegador`: arrastar, painel, teclado, seleção, desfazer, filtro, outra aba |
| 3 | regras valem na interface | **cumprido** | `TestPorEstado`: só os movimentos permitidos aparecem; movimento forjado é recusado |
| 4 | acessibilidade | **cumprido** | botões sem mouse nem JavaScript; setas; anúncios `aria-live`; `axe-core` sem violação grave ou crítica no quadro |
| 5 | atualização local | **cumprido** | mover não recarrega; a outra aba vê pela página viva; offline recupera |
| 6 | desfazer | **cumprido** | Ctrl+Z e botão; volta pelo movimento inverso, conferido pelo servidor; também em grupo |
| 7 | peso | **cumprido** | todo o JavaScript gerado: 10,4 KB (3,3 KB comprimido), sem framework |

Responsividade: num celular (375 px) a página não rola para o lado, e o quadro rola dentro dele.

Os itens adiados para esta fase estão feitos (passos 8 a 10).

### Passo 8 — "digitando…" (2026-10-02)

Parte da presença (GEP 0021): quem digita num formulário da página aparece como "<nome> está
digitando…" para as outras pessoas que olham a mesma página. O aviso é efêmero e passa por sessão
e CSRF. Provado no servidor (`TestDigitando`: a mesma página sim, a própria pessoa e outra página
não, recusa sem sessão ou CSRF) e no navegador (Ana digita no quadro, Bia vê).

### Passo 9 — editar arquivo pela web (RP-08, 2026-10-02)

Sem sintaxe nova, como parte da capability de repositório:
- a página de um arquivo mostra um formulário de edição a quem pode enviar código;
- salvar é um commit da pessoa, com a mensagem dela;
- vale a mesma checagem de um `git push`, branches protegidas incluídas, e as execuções começam
  como depois de um push.

Provado pelo GitLab, na API (`PUT …/repository/files/…`) e na página (`TestEditarArquivoPelaWeb`).

### Passo 10 — imagens no texto (UP-01, 2026-10-02)

Na GEP 0014, sem sintaxe nova:
- num texto `formatado`, colar ou soltar uma imagem a envia ao registro da página e insere o
  Markdown;
- só imagens de verdade, com o tipo detectado pelo conteúdo;
- vista por quem vê o registro, e excluída junto com ele;
- o envio fica fora de qualquer transação.

Achado na revisão: o envio passaria pela transação da requisição, com a trava presa enquanto os
bytes chegam. Corrigido antes do commit.
Provado no servidor (`TestImagensNoTexto`) e no navegador (soltar uma imagem vira Markdown).
