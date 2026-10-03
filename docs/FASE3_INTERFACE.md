# FASE 3 — Interface altamente interativa

> **Estado: INICIADA em 2026-10-02**, depois do encerramento da FASE 2
> ([FASE2_TEMPO_REAL.md](FASE2_TEMPO_REAL.md)).

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
| 4 | edição no próprio cartão, painel sem sair do quadro, seleção, desfazer, atalhos | capabilities faltantes | próximos passos |

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
