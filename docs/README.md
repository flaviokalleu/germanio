# Documentação do Germanio

## Ordem de leitura para pessoas e agentes

1. [INTENCAO.md](INTENCAO.md): especificação normativa da camada de intenção.
2. [Skill de simplicidade](../skills/germanio-simplicity/SKILL.md): como aplicar a filosofia.
3. [AGENTS.md](../AGENTS.md): obrigações de trabalho no repositório.
4. [ROADMAP.md](ROADMAP.md): prioridades e critérios de aceitação.
5. Código e testes da revisão em uso: comprovação do que funciona.

`GERMANIO_INTENT_LAYER.md` é o nome usado na solicitação externa para a especificação
que neste repositório permanece em `docs/INTENCAO.md`. Não criar uma cópia concorrente.

## Referências por finalidade

| Documento | Uso |
| --- | --- |
| [TUTORIAL](TUTORIAL.md), [CHEATSHEET](CHEATSHEET.md), [EXAMPLES](EXAMPLES.md), [FAQ](FAQ.md) | entrada pela intenção; seções antigas identificadas para consulta técnica |
| [SPEC da fundação](../SPEC.md) | gramática e execução do subconjunto da fundação |
| [SPEC técnica anterior](SPEC.md) | sintaxe de blocos anterior; não governa o nível padrão |
| [API](API.md), [INTEGRATIONS](INTEGRATIONS.md) | consumidores externos e adaptadores |
| [SECURITY](SECURITY.md), [DEPLOY](DEPLOY.md) | implementação de segurança e operação |
| [MCP](MCP.md) | ferramentas de autoria para agentes |
| [GAPS](../GERMANIO_GAPS.md), [EVOLUTION](../GERMANIO_EVOLUTION.md), [AGENT_STATE](../AGENT_STATE.md) | acompanhamento; conferir evidência antes de afirmar conclusão |
| [CHANGELOG](../CHANGELOG.md), [RELEASING](RELEASING.md), [PHASE1](PHASE1.md) | histórico de versões, processo de release, registro da fundação |
| [FEATURES-200](FEATURES-200.md), `research/` | ideias e pesquisa, sem força normativa |

## Regra de conflito

A implementação não altera silenciosamente a norma. Uma divergência deve ser classificada
como bug, lacuna da especificação ou decisão arquitetural pendente. Exemplos históricos,
contagens de testes, checklists antigos e aplicações de teste não são autoridade superior.
Não anuncie um requisito normativo como implementado sem evidência de execução.

## Limpeza documental

`FLANG_LEGACY.md` e `FLANG_ROADMAP_LEGACY.md` foram removidos por duplicarem documentação
antiga e planejamento superado. Permanecem recuperáveis pelo histórico Git. Capacidades
técnicas continuam nas referências específicas; a remoção não desativa recursos da linguagem.
