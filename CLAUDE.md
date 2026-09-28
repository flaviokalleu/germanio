## CONTRATO INVIOLÁVEL DO GERMANIO

Estas regras têm prioridade sobre conveniência de implementação, exemplos antigos, APIs já existentes no runtime e sintaxe técnica legada.

Germanio NÃO é uma linguagem tradicional traduzida para português.

Germanio NÃO deve ensinar o usuário a pensar como Go, Python, JavaScript, Rust, Java ou C.

Germanio permite que o usuário descreva um software pela intenção:

- o que existe;
- quem pode fazer;
- o que pode acontecer;
- quais regras existem;
- o que aparece.

O compilador e o runtime resolvem o mecanismo.

**Go constrói mecanismos. Germanio constrói produtos.**

### REGRA FUNDAMENTAL

A existência de uma função, keyword, primitive ou API no parser/runtime NÃO significa que ela deve ser usada no nível padrão.

Sempre diferencie:

```text
SUPORTADO PELO RUNTIME ≠ APROPRIADO PARA O DOMÍNIO
SINTAXE LEGADA ≠ SINTAXE RECOMENDADA
COMPATIBILIDADE ≠ DESIGN ATUAL
PRIMITIVA DISPONÍVEL ≠ AUTORIZAÇÃO PARA USÁ-LA
```

Nunca escolha uma construção apenas porque ela já existe.

---

## ORDEM OBRIGATÓRIA DE ABSTRAÇÃO

Antes de escrever qualquer solução `.ge`, tente resolver nesta ordem:

1. intenção;
2. declaração;
3. configuração;
4. regra de domínio;
5. lógica explícita;
6. primitive técnica;
7. adaptador externo.

Nunca pule diretamente para lógica ou primitive porque é mais fácil implementar.

Se uma necessidade comum só puder ser resolvida nos níveis 5 ou 6, considere primeiro que pode existir uma capability faltante no Germanio.

---

## PROIBIÇÃO DE VAZAMENTO TÉCNICO

Código normal de aplicação em `backend/` e `frontend/` NÃO DEVE expor detalhes técnicos quando Germanio puder derivá-los.

Considere um alerta arquitetural encontrar no domínio:

```text
GET
POST
PUT
PATCH
DELETE
HTTP
JSON
SQL
header
status code
payload
URL montada manualmente
ID usado como relação
CRUD manual
serialização
JWT
bcrypt
WebSocket
fila
worker
goroutine
mutex
transação
controller
service
repository
middleware
```

Também considere suspeitas construções como:

```ge
chamar(...)
ambiente.ler(...)
chamar_async(...)
consultar_paralelo(...)
```

Elas podem existir por compatibilidade ou níveis avançados.

Sua existência NÃO autoriza seu uso no domínio.

---

## TESTE DA INTENÇÃO

Sempre que estiver prestes a escrever uma linha técnica, pergunte:

> Qual é a intenção humana por trás desta operação?

Exemplo inadequado para domínio:

```ge
quando criar pedido
    chamar(ambiente.ler("DESTINO") + "/um", "POST", pedido.cliente)
```

Isso descreve COMO executar.

A intenção real pode ser algo conceitualmente equivalente a:

```text
quando um pedido for criado
    envie o cliente ao destino
```

ATENÇÃO:

O exemplo acima NÃO autoriza inventar uma nova sintaxe `envie`.

Primeiro procure uma construção normativa existente.

Se ela não existir, trate isso como uma possível capability faltante.

---

## NÃO INVENTE SINTAXE

Claude NÃO tem autorização para inventar silenciosamente sintaxe Germanio.

Antes de utilizar uma construção `.ge`:

1. procure em `docs/INTENCAO.md`;
2. procure nos GEPs aceitos;
3. procure nos testes normativos;
4. confira parser/compiler/runtime;
5. confirme que a construção pertence ao nível correto.

Se não existir, NÃO apresente como Germanio válido.

Uma ideia de sintaxe ainda inexistente deve ser identificada explicitamente como:

```text
PROPOSTA DE SINTAXE — NÃO IMPLEMENTADA
```

Nunca misture proposta e sintaxe implementada sem deixar a diferença explícita.

---

## QUANDO FALTAR UMA CAPABILITY

Se a intenção for válida e genérica, mas Germanio não conseguir expressá-la de maneira simples:

NÃO:

- coloque regra específica da aplicação em Go;
- esconda mecanismo dentro de uma função `.ge`;
- use primitive técnica apenas para terminar rapidamente;
- copie uma solução de outra linguagem;
- invente sintaxe e continue como se estivesse implementada.

Faça:

```text
necessidade da aplicação
        ↓
identificar intenção
        ↓
verificar capability existente
        ↓
não existe?
        ↓
verificar se é generalizável
        ↓
definir semântica
        ↓
GEP quando necessário
        ↓
implementar mecanismo genérico
        ↓
parser/compiler/runtime
        ↓
testes
        ↓
ge check / ge explain
        ↓
documentação
        ↓
usar sintaxe simples na aplicação
```

A capability deve servir potencialmente a mais de um domínio.

CRM, ERP, e-commerce, projetos, atendimento ou qualquer outro sistema devem poder reutilizar o mecanismo quando fizer sentido.

---

## PRINCÍPIO DA COMPLEXIDADE PROGRESSIVA

Germanio deve ser simples para iniciantes sem limitar profissionais.

O caso comum deve exigir poucos conceitos.

Recursos avançados devem aparecer somente quando necessários.

Não remova poder para obter simplicidade.

Esconda complexidade até ela ser necessária.

Um iniciante não deve precisar aprender infraestrutura para criar software comum.

Um profissional deve conseguir descer aos níveis avançados quando realmente precisar.

---

## HIERARQUIA ANTES DE REPETIÇÃO

A indentação fornece contexto.

Prefira:

```ge
pedidos
    tem
        cliente obrigatório

    acesso
        usuario
            criar
            ver
```

em vez de repetir continuamente:

```ge
pedido tem cliente obrigatório
usuario pode criar pedidos
usuario pode ver pedidos
```

quando a estrutura hierárquica puder representar a mesma intenção com clareza.

Não busque o menor número de caracteres.

Busque o menor número de conceitos necessários.

---

## SEGURANÇA É PADRÃO

Simplicidade nunca justifica comportamento inseguro.

Germanio deve preferir automaticamente:

- validação;
- autorização;
- escaping;
- limites;
- paginação;
- proteção contra operações perigosas;
- concorrência limitada;
- timeouts;
- backpressure;
- transações seguras;
- tratamento previsível de erros.

O usuário não deve precisar conhecer a vulnerabilidade para estar protegido dela.

Operações realmente perigosas devem exigir intenção explícita.

---

## EFICIÊNCIA É RESPONSABILIDADE DO CORE

Uma frase simples não autoriza implementação ingênua.

```ge
mostre clientes
```

não significa:

```text
SELECT * FROM clientes
carregar tudo na memória
```

Germanio deve resolver paginação, projeção, índices e limites adequados.

A regra permanece:

**simples para o humano, eficiente para a máquina.**

---

## DIAGNÓSTICOS FAZEM PARTE DA LINGUAGEM

Quando o usuário errar, não apenas rejeite.

Explique:

1. o que está errado;
2. onde está errado;
3. por que está errado;
4. como corrigir;
5. mostre uma forma válida quando apropriado.

O objetivo é permitir que uma pessoa aprenda Germanio pelo próprio compilador.

---

## GATE OBRIGATÓRIO ANTES DE FINALIZAR

Antes de concluir qualquer alteração envolvendo Germanio, responda internamente:

```text
[ ] Estou descrevendo intenção e não mecanismo?
[ ] Usei o nível mais alto possível?
[ ] Introduzi algum conceito técnico desnecessário?
[ ] Usei primitive simplesmente porque ela já existe?
[ ] Copiei um padrão de outra linguagem sem necessidade?
[ ] Existe uma capability declarativa para isso?
[ ] Estou inventando sintaxe?
[ ] Toda sintaxe apresentada como válida realmente funciona neste commit?
[ ] Um iniciante consegue aproximadamente entender o código?
[ ] Um profissional ainda consegue obter controle quando necessário?
[ ] A solução é determinística?
[ ] O padrão é seguro?
[ ] O mecanismo é genérico?
[ ] Está na camada correta?
[ ] docs/INTENCAO.md continua sendo respeitado?
[ ] ge check passa?
[ ] testes relevantes passam?
```

Se qualquer resposta importante for negativa, a tarefa NÃO está concluída.

---

## REGRA SUPREMA

Quando houver duas soluções corretas, prefira a que exige menos conhecimento técnico do usuário sem sacrificar:

- determinismo;
- segurança;
- desempenho;
- capacidade;
- clareza;
- previsibilidade.

Não faça o humano aprender como o computador trabalha quando o compilador puder aprender como o humano descreve o problema.