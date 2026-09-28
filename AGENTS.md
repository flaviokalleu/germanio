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

Go constrói mecanismos. Germanio constrói produtos.
