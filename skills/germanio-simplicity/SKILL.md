---
name: germanio-simplicity
description: Obrigatória antes de criar, alterar ou revisar qualquer .ge, ou de evoluir parser/compiler/AST/runtime/capabilities do Germanio. Garante que .ge descreva intenção (o que existe, quem pode, o que acontece, o que aparece) e que Go implemente apenas mecanismos genéricos.
---

# Germanio Simplicity Skill

Leia antes de escrever `.ge`. Releia antes de finalizar e revise o código produzido.
Nenhum `.ge` está concluído sem passar pelo checklist da seção 35.

## 1. O que é Germanio

Não é Go/Ruby/Python em português, pseudocódigo, DSL de HTTP, ORM simplificado nem YAML.
É uma linguagem orientada à **intenção**. O autor diz: o que existe, quem pode fazer,
o que pode acontecer, o que deve aparecer e quais regras especiais existem.
Germanio resolve como armazenar, consultar, atualizar, validar, autenticar, autorizar,
comunicar, serializar, paginar, proteger e executar.

## 2. Público do nível padrão

Assuma que quem escreve `.ge` não sabe programar. Ele entende
`projeto tem issues`, `usuario pode criar issues`, `somente maintainer pode excluir issues`.
Não deve precisar de HTTP, verbos HTTP, REST, JSON, SQL, ORM, chave estrangeira, migration,
controller, service, repository, middleware, serializer, JWT, bcrypt, hash, WebSocket, fila,
worker, transação ou status code.

## 3. Teste mais importante

Para cada linha: *uma pessoa que nunca programou precisaria aprender um conceito técnico
só para entender ou escrever esta linha?* Se sim, pare e descubra a intenção por trás dela.

## 4–6. Capacidades de domínio, não implementação

Errado: `quando fechar issue` + `issue.atualizar(issue.id, {state: "closed", closed_at: …})`.
Certo: `issue pode fechar` — Germanio conhece estados e transições.
Campos de infraestrutura (id, criado_em, atualizado_em, estado, fechado_em, fechado_por)
são implícitos ou criados pela capability; `ge explain` os mostra.

## 7–9. Relações e tipos humanos

Prefira `issue tem autor, comentarios, labels, responsaveis` a `autor_id`, `project_id`,
`lista de usuarios`. IDs são representação interna. Peça tipo só quando houver ambiguidade real.

## 10–11. Padrões e ações com semântica

`issue começa aberta`, `issue pode ser confidencial` em vez de `state começa com "opened"`.
Ações declaradas são transições; o autor não chama `atualizar` para executá-las.

## 12–14. Permissões e políticas declarativas

`autor ou planner pode editar issues`, `reporter pode administrar labels`,
`issue confidencial pode ser vista por autor, responsaveis, reporter ou superior`.
Não programe políticas comuns com `!=`, `e`, `nao`, `contem`, níveis numéricos.

## 15–17. Pesquisa, filtros, integração, compatibilidade

`permita pesquisar issues`, `permita filtrar issues por estado, autor, labels e responsaveis`,
`disponibilize issues para integração`. Rotas e métodos são derivados. Compatibilidade com uma
API existente (nomes, caminhos, valores) fica em configuração separada, fora do domínio.

## 18–19. Nível de referência (issues)

```ge
projeto tem
    issues
    labels

issue tem
    titulo obrigatório até 255
    descrição
    autor
    labels
    responsaveis
    comentarios

issue começa aberta

issue pode
    fechar
    reabrir
    ser confidencial

usuario pode
    criar issues
    comentar issues

autor ou planner pode
    editar issues
    fechar issues

issue confidencial pode ser vista por
    autor
    responsaveis
    reporter ou superior
```

Referência de simplicidade, não sintaxe a copiar cegamente.

## 20–22. Compiler e runtime

O compiler deriva modelos, relações, índices, CRUD, busca, filtros, paginação, transições,
autorização, validação, APIs, serialização, handlers, erros. O runtime Go implementa
**mecanismos genéricos** (ex.: máquina de estados para pedido, ticket, tarefa, pipeline,
fatura). Nunca crie keyword de entidade da aplicação (`issue`, `merge_request`).

## 23–29. Testes de revisão

- **80%**: cada bloco deve ser majoritariamente domínio e intenção.
- **Boilerplate**: buscar+erro repetido → runtime; atualizar estado+data+usuário → transição;
  checar papel+recusar → policy; listar+paginar+serializar → coleção/API.
- **Leigo**: alguém que nunca programou entende a intenção de cada linha?
- **Explicação**: se a linha precisa de explicação técnica longa, está baixa demais.
- **Tamanho**: menos conceitos técnicos e mais clareza, não menos caracteres.
- **Determinismo**: nada vago; cada frase tem semântica definida.
- **Generalização**: a capability serviria a CRM, ERP, e-commerce?

## 30. Escape hatch (último recurso)

Ordem: 1. intenção → 2. declaração → 3. configuração → 4. regra de domínio →
5. lógica explícita → 6. primitive/interop. Não pule para 5 ou 6.

## 31–32. Quando alterar o Germanio

Pare a feature → conceito geral → abstração → compiler/runtime → testes → expor em `.ge`
→ voltar à aplicação → refatorar o `.ge` antigo. Não aceite workaround permanente quando o
padrão já é generalizável.

## 33–34. `ge explain` e `ge check`

Toda inferência relevante deve ser inspecionável (`ge explain issue`: campos declarados e
internos, relações, ações, permissões, integrações). `ge check` detecta ação sem
implementação, permissão impossível, relação inválida, entidade desconhecida, referência
ambígua, regra contraditória, integração sem capability, campo impossível de inferir.

## 35. Checklist obrigatório

```text
[ ] Isto descreve intenção?
[ ] Há detalhe técnico vazando?
[ ] Há IDs que poderiam ser relações?
[ ] Há atualização manual que poderia ser ação?
[ ] Há condição booleana que poderia ser policy?
[ ] Há CRUD manual que poderia ser inferido?
[ ] Há HTTP explícito desnecessário?
[ ] Há SQL/ORM explícito desnecessário?
[ ] Há segurança que deveria ser automática?
[ ] Existe capability genérica faltando?
[ ] Um leigo consegue aproximadamente ler?
[ ] Um profissional consegue inspecionar/personalizar?
[ ] É determinístico?
```

Se houver problema, não finalize: refatore Germanio ou o `.ge`.

## 36. Regra para agentes

Nunca responda a "Germanio não suporta" implementando a feature da aplicação em Go.
Pergunte qual capability genérica falta. Go implementa a capability; `.ge` a usa.

## Regra suprema

Germanio não deve ensinar o iniciante a pensar como o computador; deve ensinar o computador
a entender uma descrição humana, formal e determinística do software. Na dúvida entre duas
sintaxes, escolha a que exige menos conhecimento técnico sem introduzir ambiguidade.
