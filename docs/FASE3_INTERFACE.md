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

(vazio)
