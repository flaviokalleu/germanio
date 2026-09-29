# Documentação do Germanio

## Ordem de leitura para pessoas e agentes

1. [INTENCAO.md](INTENCAO.md): especificação normativa da camada de intenção.
2. [Skill de simplicidade](../skills/germanio-simplicity/SKILL.md): como aplicar a filosofia.
3. [AGENTS.md](../AGENTS.md): obrigações de trabalho no repositório.
4. [ROADMAP.md](ROADMAP.md): prioridades e critérios de aceitação.
5. Código e testes da revisão em uso: comprovação do que funciona.

`GERMANIO_INTENT_LAYER.md` é o nome usado na solicitação externa para a especificação
que neste repositório permanece em `docs/INTENCAO.md`. Não criar uma cópia concorrente.

## Documentação pública (inglês, para quem chega ao projeto)

| Documento | Uso |
| --- | --- |
| [README](../README.md) · [README.pt-BR](../README.pt-BR.md) | o que é, por que existe, como experimentar |
| [getting-started](getting-started.md) | instalar, criar, rodar e ler a primeira aplicação |
| [language-tour](language-tour.md) | a linguagem por partes, cada trecho verificado por `tooling/doctest` |
| [comparisons](comparisons.md) | Germanio, Python, Go e JavaScript/TypeScript: filosofia e trade-offs |
| [ARCHITECTURE](ARCHITECTURE.md) | do `.ge` à aplicação: pacotes, testes e dívidas conhecidas |
| [examples](../examples/README.md) | exemplos executáveis, testados na CI |

A documentação pública ensina; a norma define. Se discordarem, vale a norma, e a
documentação pública é corrigida (o `tooling/doctest` recusa código `.ge` que não compila).

## Norma, processo e acompanhamento

| Documento | Uso |
| --- | --- |
| [INTENCAO](INTENCAO.md) | especificação normativa da camada de intenção |
| [SPEC da fundação](../SPEC.md) | gramática e execução do núcleo estrito |
| [GEPs](gep/README.md) | propostas de evolução da linguagem e decisões |
| [GAPS](../GERMANIO_GAPS.md), [EVOLUTION](../GERMANIO_EVOLUTION.md), [AGENT_STATE](../AGENT_STATE.md) | acompanhamento; conferir evidência antes de afirmar conclusão |
| [CHANGELOG](../CHANGELOG.md), [RELEASING](RELEASING.md) | versões e processo de release |
| [CONTRIBUTING](../CONTRIBUTING.md), [SECURITY](../SECURITY.md), [CODE_OF_CONDUCT](../CODE_OF_CONDUCT.md) | comunidade |

## Referências técnicas e guias anteriores (português)

| Documento | Uso |
| --- | --- |
| [TUTORIAL](TUTORIAL.md), [CHEATSHEET](CHEATSHEET.md), [EXAMPLES](EXAMPLES.md), [FAQ](FAQ.md) | guias escritos em boa parte na **sintaxe técnica anterior** (`dados`/`telas`, `campo: tipo`); para código novo use o tour e a norma |
| [SPEC técnica anterior](SPEC.md) | sintaxe de blocos anterior; não governa o nível padrão |
| [API](API.md), [INTEGRATIONS](INTEGRATIONS.md) | API da sintaxe anterior e integrações |
| [SECURITY técnica](SECURITY.md), [DEPLOY](DEPLOY.md) | implementação de segurança e operação |
| [PHASE1](PHASE1.md) | registro da entrega da fundação |
| [FASE1_GITLAB](FASE1_GITLAB.md) | auditoria de encerramento da FASE 1 (GitLab): critérios, capabilities, decisões pendentes |
| [FASE2_TEMPO_REAL](FASE2_TEMPO_REAL.md) | preparação da FASE 2: aplicação de referência, auditoria do tempo real, critérios mensuráveis |
| [FEATURES-200](FEATURES-200.md), [research](research/) | ideias e pesquisa, sem força normativa |

## Regra de conflito

A implementação não altera silenciosamente a norma. Uma divergência deve ser classificada
como bug, lacuna da especificação ou decisão arquitetural pendente. Exemplos históricos,
contagens de testes, checklists antigos e aplicações de teste não são autoridade superior.
Não anuncie um requisito normativo como implementado sem evidência de execução.

## Limpeza documental

`MCP.md` foi removido por documentar um comando (`ge mcp`) que não existe.
`FLANG_LEGACY.md` e `FLANG_ROADMAP_LEGACY.md` foram removidos por duplicarem documentação
antiga e planejamento superado. Permanecem recuperáveis pelo histórico Git. Capacidades
técnicas continuam nas referências específicas; a remoção não desativa recursos da linguagem.
