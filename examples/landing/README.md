# Landing page do Germanio

A página de apresentação do projeto, escrita inteira em `.ge` com as palavras em português da
[GEP 0059](../../docs/gep/0059-pagina-de-apresentacao.md) (em teste): `capa`, `seção`, `cartão`,
`fundo`, `imagem`, `ponto`, `rodapé`.

```bash
cd examples/landing
ge run app.ge        # http://localhost:8080
```

As imagens de fundo (`assets/montanhas.png`, `assets/editor.png`) são ligações para os arquivos
em `examples/germanio/`, que não vão para o repositório. Sem elas a página continua funcionando,
com o fundo escuro padrão.
