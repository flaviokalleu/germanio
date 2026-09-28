# Instruções para agentes no Germanio

Antes de trabalhar no repositório, leia integralmente `docs/INTENCAO.md` e
`skills/germanio-simplicity/SKILL.md`. Consulte `docs/README.md` para a autoridade de cada guia.
Essas leituras também se aplicam a alterações de documentação e revisão de exemplos.

- O nível padrão descreve intenção humana, formal e determinística.
- Domínio descreve produto; core implementa mecanismos genéricos; adaptador traduz protocolo.
- Antes de lógica manual, procure a abstração declarativa existente ou a capability genérica faltante.
- Não mova regra de aplicação ou protocolo específico para Go apenas para reduzir o `.ge`.
- Não altere a norma silenciosamente para justificar uma implementação divergente.
- Diferencie contrato normativo, exemplo ilustrativo, implementação comprovada e pendência.
- Novas capabilities exigem testes de generalização e refatoração dos exemplos obsoletos.
- Alterações exclusivamente documentais não autorizam implementar compiler/runtime.
- Verifique links, coerência e exemplos alterados; relate exatamente o que foi validado.

## Documentation Gate

A documentação faz parte do código. Antes de modificar comportamento do Germanio — sintaxe,
lexer, parser, AST, resolver, runtime, formatter, `ge check`, `ge explain`, LSP, capabilities,
integrações ou qualquer `.ge`:

1. Identifique os documentos que governam a área: sempre `docs/INTENCAO.md` e
   `skills/germanio-simplicity/SKILL.md`; além deles, a referência do subsistema
   (`SPEC.md` para o núcleo estrito, `docs/research/` para a base das decisões,
   `GERMANIO_GAPS.md` para lacunas conhecidas).
2. Leia as seções relevantes. A leitura é proporcional à mudança, mas nenhuma mudança
   semântica acontece sem consultar a especificação.
3. Compare a alteração proposta com a especificação.
4. Implemente.
5. Rode os testes (e os testes normativos da área).
6. Verifique de novo a documentação: ela continua verdadeira?
7. Se a semântica mudou de propósito, atualize a documentação na mesma unidade de trabalho.

Se implementação e documentação discordarem, **não escolha silenciosamente uma delas**.
Classifique (implementação errada, especificação incompleta ou decisão arquitetural nova),
registre em `GERMANIO_GAPS.md` ou `AGENT_STATE.md` e resolva conscientemente.
Uma mudança que altera a linguagem e deixa a documentação incorreta está incompleta;
uma mudança normativa sem implementação e testes alinhados também está.

Go constrói mecanismos. Germanio constrói produtos.
