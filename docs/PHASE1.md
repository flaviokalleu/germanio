# Entrega da fundação Germanio

## Auditoria inicial

Base: `36ef40513d25f1398faada3ce8e7f652bbcab79b`.
O projeto já possuía lexer multilíngue, parser declarativo, AST de scripts e web,
interpretador, geradores, CLI, banco, autenticação, integrações e extensão VS Code.
O ambiente exigiu instalar Go 1.26.1, conforme `go.mod`.

Os testes originais do compilador e interpretador passaram. A suíte `go test ./...`
falhava por dois problemas anteriores: instalador Windows dependente de payload
ausente e `fmt.Println` com newline redundante detectado pelo vet. Ambos foram
corrigidos sem mudar a semântica Flang.

O parser Flang tolerava tokens desconhecidos e o runtime fazia coerções. Para não
quebrar programas existentes, o modo `.fg` foi mantido; `.ge` usa entrada estrita,
análise de tipos e execução separada, compartilhando os nós do AST e a estrutura
de varredura léxica. Nenhum recurso web foi reescrito por aparência.

## Implementação real

- Módulo Go corrigido para `github.com/flaviokalleu/germanio`.
- CLI `ge`: rodar, check, fmt/--check, novo, explicar, ajuda e versão.
- Entry point Flang preservado, inclusive build da raiz com nome `flang`.
- Lexer `.ge` sem tradução de keywords, com operadores, tipos opcionais e posições.
- Parser determinístico, blocos de dois espaços, funções e rejeição de lixo sintático.
- AST ampliado sem remover estruturas legadas; origem em arquivo/linha/coluna.
- Variáveis imutáveis, `mut`, `const`, inferência inicial monomórfica e anotações.
- Texto, inteiro, decimal, bool, listas homogêneas, listas vazias tipadas e opcionais.
- Funções, retorno implícito/explicito, recursão, condicionais e loops com controle.
- Entrada/saída, interpolação, conversões explícitas, quantidade e indexação segura.
- Imports locais com namespace, diagnóstico de ciclos, duplicatas e acesso externo.
- Diagnósticos GE com motivo/correção; tipos inválidos, nomes desconhecidos,
  variáveis/imports sem uso, imutabilidade, overflow e divisão por zero.
- Formatter inicial idempotente, que preserva comentários e recusa código inválido.
- UI natural mínima executável: navbar e botão estáticos, escape de texto,
  arredondamento com unidades verificadas. Não conecta ações nem inicia servidor.
- Seis exemplos `.ge`, documentação atual, logo fornecida pelo usuário e referência
  histórica Flang preservada em `docs/FLANG_LEGACY.md`.
- Highlighting `.ge` na extensão existente, mantendo `.fg`. Sem alegar suporte LSP.
- CI com testes, vet, race checks, builds e exemplos de ambos os modos.

## Validação

- `go test ./...`: testes originais e novos, incluindo casos válidos e inválidos.
- `go vet ./...` e `go test -race ./compiler/... ./runtime/germanio ./tooling/...`.
- Builds da CLI Germanio e da CLI Flang.
- Execução real de entrada pelo terminal, exemplos de fundamentos e imports.
- `ge check` nos seis exemplos `.ge` e nos demos plano/organizado `.fg`.
- `ge fmt examples/germanio --check`.
- Fuzzing do parser por 15 segundos: 670.904 entradas, sem falha naquela execução.
- JSON de metadados e gramática da extensão validado sintaticamente. A extensão
  não foi instalada/interativamente validada no VS Code neste ambiente.

Os testes não equivalem a auditoria completa das integrações externas herdadas.
Não houve deploy, publicação de pacote, criação de release ou alteração de banco.

## Reaproveitamento e limites

Foram preservados lexer/parser `.fg`, AST web, interpretador legado, servidor,
modelos, CRUD, auth, banco, templates e integrações. Distribuições antigas em
`dist/` e o backup binário histórico foram identificados e mantidos nesta entrega;
não foram apresentados como releases Germanio. A logo original também permanece
para documentação histórica. O novo asset é `assets/germanio.png`.

Esta entrega é a fundação executável da Fase 1, com suporte limitado descrito na
SPEC. `.ge` roda por interpretação AST em Go. Não há compilação nativa de `.ge`,
stdlib remota, mapas, tipos financeiros/de data, captura de globais por funções,
refinamento de opcionais, generics ou ponte automática frontend/backend.

Fase 2: aprofundar inferência, null safety com refinamento, escopos e módulos;
contratos, generics fundamentais, erros tipados e tipos especializados. Fase 3:
UI completa, transporte seguro, geração full-stack `.ge` e análise de segredos.
LSP, package manager, toolchain de testes e otimizações pertencem às fases seguintes.
GPU, SIMD, WASM, JIT e FFI não foram simulados nem marcados como implementados.
