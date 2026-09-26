# Roadmap do Germanio

Este é o plano de entregas da linguagem `.ge`, baseado no prompt mestre do
projeto. [SPEC.md](../SPEC.md) descreve a gramática e o comportamento que
funcionam hoje. O [roadmap original do Germanio](GERMANIO_ROADMAP_LEGACY.md) permanece
como histórico do modo `.ge`; suas funcionalidades não passam automaticamente
para `.ge`. Não há data de lançamento anunciada para as fases futuras.

## Estado verificado

| Fase | Estado | Entrega comprovável |
| --- | --- | --- |
| 1. Fundação | Implementada no escopo da SPEC | `ge`, `.ge`, lexer/parser/AST, análise inicial, interpretador, exemplos, formatter inicial e compatibilidade `.ge` |
| 2. Tipos e semântica | Em andamento | Opcionais com junção de fluxo, genéricos explícitos com restrição `numero`, funções privadas, inferência monomórfica fora de genéricos e imports locais |
| 3. Full-stack `.ge` | Pendente | Apenas HTML estático limitado; servidor/banco/auth do Germanio continuam em `.ge` |
| 4. Toolchain | Em andamento | CLI/formatter/CI iniciais, testes nativos `ge testar` e cobertura de instruções; manifesto, dependências e LSP ainda não |
| 5. Performance | Pendente | Limites de execução existem; HIR/MIR e compilação incremental ainda não |
| 6. Poder avançado | Pendente | Nenhum backend FFI/WASM/JIT/SIMD/GPU `.ge` anunciado |

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

### 3 — Aplicações full-stack em `.ge`

- Definir domínios `cliente`, `servidor` e `compartilhado` no compilador, com
  checagem de acessos proibidos e tipos serializáveis na fronteira.
- Ligar rotas, APIs, modelos, banco e autenticação a programas `.ge` reais,
  reaproveitando servidor e componentes Germanio quando fizer sentido.
- Evoluir a UI natural com eventos e navegação executáveis; testar no navegador
  a ida e a volta entre cliente e servidor, incluindo permissões e dados secretos.
- Critério de saída: exemplo `.ge` completo com frontend e backend, testes de
  autorização e isolamento, API funcionando e nenhum botão decorativo anunciado
  como interativo.

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
