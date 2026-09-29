# Roadmap do Germanio

O contrato da camada de intenção está em [INTENCAO.md](INTENCAO.md). Este roadmap
separa capacidades encontradas no código de critérios ainda exigidos; não é um relatório
de testes executados nesta revisão. A [SPEC da fundação](../SPEC.md) cobre seu subconjunto,
não todo o full-stack. Não há data de lançamento anunciada.

## Estado e evidências

| Área | Estado documental | Onde conferir |
| --- | --- | --- |
| Fundação, tipos e módulos | subconjunto implementado, evolução em andamento | `../SPEC.md`, `compiler/semantic/` |
| Camada de intenção full-stack | existe; não está limitada a HTML estático | `compiler/parser/intencao.go`, `resolver.go`, `runtime/servidor/intencao.go`, `paginas.go`, `runtime/intencao_test.go` |
| Relações, login, papéis, estados | mecanismos presentes; cobertura deve ser validada por caso | `compiler/ast/intencao.go`, `runtime/interpreter/intencao.go`, testes do runtime |
| Migração do banco e efeitos externos | renames só explícitos (`renomeie`, `descarte`; G93); efeitos externos depois do commit, fora da trava (G86); outbox durável ainda não | `runtime/banco/migracao.go`, `runtime/interpreter/efeitos.go`, `runtime/servidor/transacao.go`, `runtime/efeitos_test.go` |
| Trabalho remoto genérico | mecanismo e testes presentes | `runtime/servidor/trabalho_remoto.go`, `runtime/trabalho_remoto_test.go` |
| `ge explain` e `ge check` | suporte parcial ao contrato normativo ampliado | `tooling/explicar/explicar.go`, `tooling/gecli/` |
| HIR/MIR, FFI, WASM, JIT, SIMD e GPU | não anunciados como entregues por este documento | exigir evidência por backend |

## Fases de prova (ordem obrigatória)

| Fase | Pergunta | Estado |
| --- | --- | --- |
| 1 — GitLab FOSS | Germanio constrói software empresarial profundo? | pronta para encerramento, aguardando decisões — [FASE1_GITLAB.md](FASE1_GITLAB.md) |
| 2 — tempo real pesado | lida com sistemas vivos e concorrentes? | preparada, não iniciada — [FASE2_TEMPO_REAL.md](FASE2_TEMPO_REAL.md) |
| 3 — UI altamente interativa | constrói experiências de interface complexas? | não iniciada |
| 4 — escala | continua funcionando quando a carga deixa de ser confortável? | não iniciada |
| 5 — amplitude | é realmente geral? | não iniciada |

Uma fase termina implementada, testada, documentada, generalizada e auditada. Ideias de fases
futuras são registradas aqui e não desviam a fase atual.

### Adiado da FASE 1 (não descartado)

| Item (inventário) | Destino | Por quê |
| --- | --- | --- |
| confirmação de e-mail (ID-06) | FASE 5 | amplitude de identidade; precisa de decisão de produto (bloquear login até confirmar?) |
| fork (PR-05) | FASE 5 | amplitude |
| estrelas, tópicos, avatar (PR-06) | FASE 5 | amplitude; avatar usa os arquivos da GEP 0014 |
| edição de arquivo pela web (RP-08) | **FASE 3** | editor: estado local, pré-visualização, commit |
| boards (IS-09) | **FASE 3** | arrastar e soltar, estado derivado; a sincronização entre pessoas usa a FASE 2 |
| uploads dentro de Markdown (UP-01) | FASE 3 | editor de texto com imagens; usa a GEP 0014 |
| regras avançadas de merge request (MR-07) | BACKLOG | regras de aprovação e merge automático: capability futura com GEP própria |
| DAG avançado de CI (CI-09) | BACKLOG | capability de fluxo de execução (dependências, regras, manual) |
| isolamento de CI em containers (CI-10) | FASE 4 | infraestrutura de execução |

## Prioridade: conformidade da intenção

1. Completar proveniência e motivos das inferências em `ge explain`.
2. Expandir `ge check` para contradições, ambiguidades e capabilities incompletas.
3. Comprovar invariantes de papel mínimo em criação, herança, concorrência e alterações indiretas.
4. Testar transições, metadados, permissões e generalidade entre domínios.
5. Manter protocolos de produtos externos nos adaptadores e refatorar boilerplate antigo.

As etapas abaixo preservam frentes de evolução; não autorizam reimplementar capacidades
que já existem nem trocar intenção por uma API manual.

## Próxima sequência de trabalho

### 2A — Segurança e fluxo de tipos

- Concluir refinamento de opcionais para atribuições e junções entre ramos e
  loops; rejeitar acesso a valores que podem voltar a ser `nulo`.
- Verificar inferência e coerção numérica em funções, listas e módulos, incluindo
  chamadas recursivas e imports compartilhados.
- Ampliar genéricos fundamentais (outras restrições, inferência em casos complexos),
  definir contratos verificáveis para funções e erros tipados, sempre com
  semântica de execução e diagnósticos.
- Critério de saída: para cada regra, executar exemplos que passam e falham,
  testar os caminhos do verificador e do interpretador e registrar limites na SPEC.

### 2B — Módulos e interface semântica

- Consolidar visibilidade, resolução de nomes, tratamento de ciclos e erros de
  import com origem no arquivo e na linha corretos.
- Separar API exportada de código de entrada e manter compatibilidade com `.ge`.
- Critério de saída: programa multifile executável, dependências inválidas
  rejeitadas e nenhuma mudança de comportamento silenciosa para `.ge`.

### 3 — Consolidar aplicações full-stack por intenção

- Consolidar a separação já existente entre `backend/`, `frontend/` e `integracoes/`.
- Verificar isolamento, relações, permissões, transições e invariantes em todas as entradas.
- Evoluir páginas e eventos sem exigir rotas ou CRUD manuais do autor do domínio.
- Garantir que interface e integrações usem as mesmas regras de negócio e autorização.
- Critério de saída por capability: exemplo real, teste genérico em outro domínio,
  teste de erro, inspeção, documentação e refatoração do código redundante.

### 4 — Ferramentas de desenvolvimento

- Introduzir `germanio.toml` e lockfile apenas junto com resolução reproduzível
  de dependências. Completar formatter e linter com saída determinística.
- Ampliar testes nativos com cobertura de ramos, benchmarks/fuzzing pela CLI e LSP com diagnósticos e
  correções verificadas no editor.
- Critério de saída: projeto criado, formatado, testado e instalado em ambiente
  limpo; builds reproduzíveis e testes de integração na CI.

### 5 — Compilação e desempenho

- Introduzir HIR/MIR preservando a semântica atual; comparar saída com o
  interpretador usando os mesmos exemplos.
- Acrescentar cache incremental e builds de release somente com medição de
  tempo/memória e invalidação correta de módulos.
- Critério de saída: equivalência semântica, benchmarks repetíveis e regressões
  de compilação detectáveis na CI.

### 6 — Interoperabilidade e recursos avançados

- Planejar APIs de FFI C/Go, WASM, JIT, SIMD, GPU, memória e metaprogramação
  conforme casos de uso e isolamento das fronteiras do runtime.
- Implementar cada backend separadamente, com exemplos executáveis e testes no
  ambiente de destino. Nenhuma sigla conta como entregue por ter uma pasta ou flag.

## Regra para marcar uma entrega como pronta

Uma funcionalidade só entra na SPEC como disponível depois de passar por lexer,
parser, verificador, execução real e documentação, quando essas camadas se
aplicarem. A CI deve compilar `ge` e `germanio`, rodar testes válidos e inválidos,
`go vet`, detector de corrida, exemplos `.ge`, formatter e exemplos `.ge`.
Falhas devem identificar arquivo, posição, motivo e correção. A sintaxe natural
continua determinística: nenhuma LLM decide o significado de um programa.
