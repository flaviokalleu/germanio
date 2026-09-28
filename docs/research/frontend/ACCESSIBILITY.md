# Acessibilidade derivada: o que o iniciante não precisa lembrar

Data: 2026-09-28 · Status: pesquisa (não normativa).

## Fontes consultadas

- WCAG 2.2, W3C Recommendation (versão de 12/12/2024): https://www.w3.org/TR/WCAG22/
- Understanding 2.5.8 Target Size (Minimum): https://www.w3.org/WAI/WCAG22/Understanding/target-size-minimum.html
- Understanding 4.1.3 Status Messages: https://www.w3.org/WAI/WCAG22/Understanding/status-messages.html
- Understanding 2.4.11 Focus Not Obscured (Minimum): https://www.w3.org/WAI/WCAG22/Understanding/focus-not-obscured-minimum.html
- Understanding 1.4.10 Reflow: https://www.w3.org/WAI/WCAG22/Understanding/reflow.html
- WAI-ARIA APG, Read Me First ("No ARIA is better than bad ARIA"): https://www.w3.org/WAI/ARIA/apg/practices/read-me-first/
- APG, Dialog (Modal): https://www.w3.org/WAI/ARIA/apg/patterns/dialog-modal/
- APG, Tabs: https://www.w3.org/WAI/ARIA/apg/patterns/tabs/
- APG, Menu Button: https://www.w3.org/WAI/ARIA/apg/patterns/menu-button/ · Menu/Menubar: https://www.w3.org/WAI/ARIA/apg/patterns/menubar/
- APG, Disclosure Navigation: https://www.w3.org/WAI/ARIA/apg/patterns/disclosure/examples/disclosure-navigation/
- HTML Living Standard, o elemento `dialog`: https://html.spec.whatwg.org/multipage/interactive-elements.html#the-dialog-element
- MDN, `<dialog>`: https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/dialog
- WebAIM Million 2025: https://webaim.org/projects/million/2025
- Svelte, compiler warnings (a11y_*): https://svelte.dev/docs/svelte/compiler-warnings
- Radix Primitives: https://www.radix-ui.com/primitives/docs/overview/introduction · React Aria: https://react-aria.adobe.com/ · React Aria forms: https://react-aria.adobe.com/forms
- Material Color Utilities, `hct.ts`: https://github.com/material-foundation/material-color-utilities/blob/main/typescript/hct/hct.ts

Código do Germanio lido: `runtime/servidor/paginas.go` (`layoutTpl`, `formTpl`, `searchTpl`,
`treeTpl`, `post`), `runtime/servidor/renderizador.go`, `runtime/servidor/renderizador_declarativo.go`.

## 1. Princípio: HTML nativo primeiro, ARIA só como promessa cumprida

- O APG: "ARIA incorreto deturpa a experiência visual, com efeitos potencialmente devastadores na
  experiência não visual"; "um role é uma promessa": `role="button"` obriga a implementar o
  teclado que o usuário espera, porque ARIA não fornece comportamento; ARIA é "um CSS para
  tecnologias assistivas" que tanto enriquece quanto encobre a semântica original
  (w3.org/WAI/ARIA/apg/practices/read-me-first).
- WebAIM Million 2025: páginas com ARIA tiveram em média 57 erros detectados, contra 27 sem ARIA;
  o relatório adverte que isso não prova causalidade (as páginas são mais complexas)
  (webaim.org/projects/million/2025).
- Consequência para um gerador: emitir `<button>`, `<a href>`, `<label>`, `<table>` com `<th>`,
  `<nav>`, `<main>`, `<form>`, `<dialog>` resolve a maior parte por construção. O Germanio hoje
  não emite nenhum atributo `aria-` em `paginas.go`, `renderizador.go` ou
  `renderizador_declarativo.go` e usa elementos nativos; isso é uma boa base, não uma lacuna.

## 2. O que o Germanio pode derivar sozinho

| Obrigação (critério WCAG 2.2) | Derivado de | Estado hoje (`paginas.go`) |
| --- | --- | --- |
| Estrutura e landmarks (1.3.1, 2.4.1) | layout fixo da página | `<header>`, `<nav>`, `<main>` existem; **falta** link "pular para o conteúdo" |
| Título da página (2.4.2) | nome da página + sistema | `<title>{{.Title}} · {{.System}}` ok |
| Hierarquia de títulos | Page → `h1`; seções → `h2`; formulário → `h3` | ok |
| Rótulos (1.3.1, 3.3.2, 4.1.2) | `Field.Label`/nome | campos do formulário ok (`<label>` envolvendo); **pesquisa e filtros usam só `placeholder`** (`searchTpl`) |
| Obrigatoriedade (3.3.2) | `Field.Required` | `required` só na criação |
| Identificação de erro (3.3.1) e sugestão (3.3.3) | validação do domínio | erro genérico no topo via `?erro=`, sem campo |
| Entrada redundante (3.3.7) | re-renderizar com valores | redirect descarta o que foi digitado |
| Mensagens de status (4.1.3) | resultado da ação | `div.aviso` sem `role="status"`/`"alert"` |
| Idioma (3.1.1) | idioma do sistema/mensagens | `lang="pt-BR"` fixo nos quatro renderizadores; sem `dir="rtl"` para árabe, embora o léxico aceite árabe |
| Tabelas de dados (1.3.1) | colunas | `<th>` ok; sem `<caption>` (o `h2` acima não é associado) |
| Ícones com texto (1.1.1) | tipo da entrada | `treeTpl` usa emojis de pasta/arquivo sem alternativa textual |
| Contraste (1.4.3, 1.4.11) | gerador de tokens | falhas medidas: ver `DESIGN_SYSTEMS.md` §3 (botões de `tema azul` 3.68:1; botão do modo escuro 3.35:1; borda de campo 1.45:1) |
| Foco visível (2.4.7) | tokens de foco | contorno padrão do navegador (não removido: aceitável); sem token de foco próprio |
| Foco não encoberto (2.4.11) | cabeçalho fixo | cabeçalho não é fixo: ok; se ficar fixo, `scroll-padding` (técnica suficiente citada no Understanding 2.4.11) |
| Tamanho de alvo 24×24 (2.5.8) | tokens de densidade | botões com ~36px de altura: ok; checar `densidade compacta` |
| Reflow 320px (1.4.10) | layout intrínseco | ver `RESPONSIVE.md`; tabelas de dados são exceção explícita |

Níveis: 1.4.3, 1.4.11, 1.4.10, 2.4.7, 2.4.11, 2.5.8 e 4.1.3 são AA; 3.3.1, 3.3.2 e 3.3.7 são A;
2.4.13 Focus Appearance é AAA; 4.1.1 Parsing foi removido no WCAG 2.2 (w3.org/TR/WCAG22 e
Understanding correspondentes).

### 2.1 Rótulos a partir dos nomes

O nome do campo já é o rótulo (`fieldLabel` preserva a grafia com acento de `Field.Label`). A
mesma fonte deve servir a: `<label>` do formulário, `<th>` da tabela, `<dt>` do detalhe, o
rótulo do filtro, a mensagem de erro ("Email é obrigatório") e o nome acessível de botões de
linha ("Editar cliente Maria", não só "Editar" repetido dez vezes; o nome do registro vem de
`titleOf`). Um rótulo nunca deve ser só `placeholder`.

### 2.2 Formulários

Proposta alinhada ao React Aria (validação nativa por restrições; erros de servidor por campo,
limpos quando o valor muda; react-aria.adobe.com/forms): cada erro é texto associado ao campo por
`aria-describedby` e o campo recebe `aria-invalid="true"` (ARIA justificado: não há equivalente
nativo para associar a mensagem do servidor); um resumo no topo com links para os campos, com o
foco movido para ele após o envio; valores preservados. A mensagem é a mesma do diagnóstico do
domínio, na língua de `mensagens em`.

## 3. Comportamentos interativos segundo o APG (e o que o HTML já dá)

| Padrão | Regras do APG | O que o nativo já cumpre | Quando o Germanio precisa |
| --- | --- | --- | --- |
| Dialog modal | foco vai para dentro; Tab/Shift+Tab circulam só dentro; Esc fecha; ao fechar o foco volta ao elemento que abriu; `aria-modal` e nome por `aria-labelledby` (apg/patterns/dialog-modal) | `<dialog>` com `showModal()`: resto da página inerte, Esc fecha (`closedby` padrão `closerequest`), foco no primeiro focável ou no `autofocus`, `aria-modal` implícito (MDN); o HTML Standard guarda o "previously focused element" e o refoca ao fechar (html.spec.whatwg.org) | confirmar exclusão, formulário rápido. **Usar `<dialog>`**, não reimplementar trap de foco |
| Tabs | `tablist`/`tab`/`tabpanel`; setas movem o foco (com volta), Home/End; ativação automática recomendada quando o painel aparece sem latência; `aria-selected`, `aria-controls` (apg/patterns/tabs) | não há elemento nativo | detalhe de um registro com várias coleções; em SSR sem JS, **links** para sub-rotas são mais simples e corretos |
| Menu button | Enter/Espaço abrem e focam o primeiro item; `aria-haspopup`, `aria-expanded`; Esc fecha e devolve o foco (apg/patterns/menu-button, menubar) | não há | ações extras de uma linha |
| Navegação do site | o APG usa **disclosure**, não `role="menu"`, porque a navegação "não oferece a funcionalidade complexa que as tecnologias assistivas esperam de um widget com role menu" (apg disclosure-navigation) | `<nav>` + `<details>`/botão com `aria-expanded` | sidebar/navbar no mobile |

Regra proposta: o renderizador tem **um** componente por padrão APG, testado uma vez (como os
Primitives do Radix e o React Aria, que tratam "aria attributes, role management, focus handling
e keyboard navigation"), e o autor nunca escolhe roles. Preferência: elemento nativo > padrão APG
implementado no runtime > nada. Arrastar (kanban) exige alternativa por teclado e por botão
(mover para o próximo estado), o que o modelo de estados já fornece.

## 4. Live regions para feedback

4.1.3: mensagens de status devem ser determináveis por role ou propriedades "para que sejam
apresentadas… sem receber foco"; técnicas: `role="status"` para sucesso e progresso,
`role="alert"` para erros e avisos, `aria-live` para atualizações sequenciais
(Understanding 4.1.3). Proposta:

- Sucesso (`Criado com sucesso`) → região `role="status"` presente no layout.
- Erro de ação → `role="alert"`; erro de formulário → resumo focado (o foco já anuncia).
- Atualização em tempo real (websocket) → `aria-live="polite"` só com resumo ("3 novos
  registros"), nunca a tabela inteira.
- A região existe no DOM desde o início (a inserção de uma região já com conteúdo nem sempre é
  anunciada; não verificado em fonte primária nesta pesquisa, é prática corrente).

## 5. "a11y check" no `ge check`

O Svelte emite 43 avisos `a11y_*` em tempo de compilação (por exemplo
`a11y_label_has_associated_control`, `a11y_no_redundant_roles`, `a11y_autofocus`,
`a11y_positive_tabindex`) e permite suprimir um aviso com um comentário `svelte-ignore <código>`
acima da linha, com justificativa opcional (svelte.dev/docs/svelte/compiler-warnings). A maioria
desses avisos policia HTML escrito à mão; no Germanio o HTML é gerado, então a classe de erros é
outra: **escolhas do autor que o gerador não pode consertar sozinho**. Candidatos,
todos estáticos:

| Aviso (candidato) | Condição | Como corrigir (formato 4 partes) |
| --- | --- | --- |
| contraste | par de tokens do tema abaixo de 4.5:1 (texto) ou 3:1 (UI) após o ajuste automático | escolher outro `destaque` ou `contraste alto` |
| imagem sem descrição | campo `imagem` sem campo de texto que a descreva | declarar a descrição ou marcar como decorativa |
| gráfico sem alternativa | `gráfico` sem tabela/resumo acessível derivável | o runtime gera a tabela; aviso só se não for derivável |
| rótulo de ação ambíguo | `ações` com rótulos repetidos ou genéricos ("Clique aqui", "Ver") | usar o verbo do domínio |
| títulos de página duplicados | duas páginas com o mesmo nome visível | renomear |
| texto só por cor | estado distinguido só por cor num bloco | o runtime já mostra o texto do estado (`selo`); aviso para variantes que o omitam |
| idioma/direção | `mensagens em` em idioma RTL sem suporte de layout | informativo até o runtime suportar `dir` |

Diagnósticos no formato normativo (o quê / onde / por quê / como corrigir). Supressão: com
motivo obrigatório e visível em `ge explain`, nunca global. O check não substitui teste com
tecnologia assistiva: o próprio APG diz que testar a interoperabilidade é essencial antes de usar
os exemplos em produção (apg read-me-first).

## Para o Germanio

**ADOTAR**

- HTML nativo por construção e ARIA só quando não há equivalente (APG, WebAIM). Resolve: manter
  a base boa de `paginas.go` enquanto cresce (G62), sem ARIA ruim.
- `<dialog>` + `showModal()` para confirmação e formulários rápidos. Resolve: trap de foco,
  Esc e retorno do foco sem código próprio.
- Live regions fixas no layout (`role="status"`, `role="alert"`). Resolve: `.aviso` mudo para
  leitor de tela em `layoutTpl` (4.1.3).
- Formulário que re-renderiza com valores e erro por campo (`aria-describedby`,
  `aria-invalid`), resumo focado. Resolve: 3.3.1 e 3.3.7 em `post()`.
- Rótulo sempre visível derivado do nome, também para pesquisa e filtros. Resolve: `searchTpl`
  só com `placeholder`.
- Contraste garantido pelo gerador de tokens e reverificado pela fórmula WCAG. Resolve: as
  falhas medidas em `DESIGN_SYSTEMS.md` §3.

**ADAPTAR**

- Avisos `a11y` estáticos no `ge check`, como os do Svelte, mas sobre escolhas do domínio e do
  tema (tabela da seção 5), com supressão justificada. Afeta `tooling/explicar` (onde
  `Verificar` já emite avisos) e `compiler/diagnostics`.
- Nome acessível contextual para ações de linha a partir de `titleOf`. Resolve: "Editar"
  repetido sem contexto (2.4.4/2.4.6).
- `lang` e `dir` a partir de `mensagens em`. Resolve: `lang="pt-BR"` fixo com léxico em 20
  idiomas.

**EVITAR**

- `role="menu"` para navegação do site (o APG usa disclosure).
- Reimplementar comportamento que o HTML já tem (trap de foco, Esc do dialog).
- Deixar o autor escrever roles, tabindex ou atributos ARIA no nível padrão.
- Emojis como único conteúdo de um ícone (`treeTpl`).
- Tratar APCA como conformidade WCAG 2.2.

**INVESTIGAR**

- Testes automáticos de acessibilidade das páginas geradas (por exemplo um verificador de regras
  rodando sobre o HTML de teste) versus só testes de unidade de templates.
- Tabs em SSR: links para sub-rotas ou padrão APG com JS progressivo.
- Anúncio de atualizações em tempo real sem ruído (limiar de agrupamento).
- Mensagens de validação nativas do navegador (localizadas pelo navegador) versus as do domínio
  (localizadas por `mensagens em`): qual prevalece e como evitar duas mensagens diferentes.
