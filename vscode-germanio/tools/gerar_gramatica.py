#!/usr/bin/env python3
"""Generates syntaxes/germanio.tmLanguage.json.

Every keyword accepts the accented and unaccented spellings the Germanio
lexer accepts (único/unico, integração/integracao). Word boundaries are
Unicode aware so words such as "visível" are highlighted as a whole.
Run: python3 tools/gerar_gramatica.py
"""
import json, os

ACC = {"a": "[aáàâã]", "e": "[eéê]", "i": "[ií]", "o": "[oóôõ]", "u": "[uú]", "c": "[cç]"}

def w(word):
    """Accent-tolerant regex for one word (spaces become \\s+)."""
    out = []
    for ch in word:
        if ch == " ":
            out.append(r"\s+")
        else:
            out.append(ACC.get(ch, ch))
    return "".join(out)

def words(*ws):
    return "(?:" + "|".join(w(x) for x in sorted(ws, key=len, reverse=True)) + ")"

B = r"(?<![\p{L}\p{N}_])"   # start of word
E = r"(?![\p{L}\p{N}_])"    # end of word
NAME = r"[\p{L}_][\p{L}\p{N}_]*"
NAMES = NAME + r"(?:\s+" + NAME + r")*?"

def kw(scope, *ws):
    return {"name": scope, "match": B + words(*ws) + E}

types = ["texto_longo", "texto", "inteiro", "numero", "booleano", "email", "senha", "data_hora", "data",
         "url", "visibilidade", "branch", "enum", "dinheiro", "telefone", "imagem", "arquivo", "segredo", "decimal", "bool"]
modifiers = ["obrigatorio", "obrigatoria", "unico", "unica", "privado", "privada", "oculto", "oculta", "imutavel",
             "formatado", "formatada", "protegida", "protegido", "opcional", "revogavel", "prefixo", "formato",
             "valida", "min", "max", "ate", "padrao", "comeca com", "numero por", "expira em", "dias", "dia",
             "com papel", "indice", "soft_delete", "interno", "pertence_a", "tem_muitos"]
control = ["senao se", "senao", "se", "enquanto", "para cada", "para", "em", "retornar", "retorne", "definir", "tentar",
           "erro", "pare", "parar", "continue", "continuar", "repetir", "vezes", "mostrar", "mostre", "funcao", "usa"]
builtins = ["recuse", "falhar", "responder", "redirecionar", "obter", "tem", "adicionar", "tamanho", "contem",
            "agora_iso", "hoje", "texto", "numero", "inteiro"]
logical = ["nao", "e", "ou"]
ctxvars = ["atual", "entrada", "dados", "registro", "atualizacoes", "requisicao"]
modules = ["git", "cripto", "acesso", "tarefas", "markdown", "regex", "ambiente", "token", "sessao", "json", "sequencia"]
blocks = ["sistema", "dados", "telas", "logica", "rotas", "eventos", "tema", "banco", "autenticacao", "integracoes",
          "importar", "paginas", "sidebar", "acoes"]

role_subject = r"(?:" + NAME + r")(?:\s+ou\s+" + NAME + r")*"

grammar = {
    "$schema": "https://raw.githubusercontent.com/martinring/tmlanguage/master/tmlanguage.json",
    "name": "Germanio",
    "scopeName": "source.germanio",
    "patterns": [
        {"include": "#comment"},
        {"include": "#data_block"},
        {"include": "#intent"},
        {"include": "#expression"},
    ],
    "repository": {
        "comment": {"patterns": [
            {"name": "comment.line.number-sign.germanio", "match": r"(^|(?<=\s))#.*$"},
            {"name": "comment.line.double-slash.germanio", "match": r"(^|(?<=\s))//.*$"},
        ]},
        "string": {"patterns": [
            {"name": "string.quoted.triple.germanio", "begin": '"""', "end": '"""'},
            {"name": "string.quoted.double.germanio", "begin": '"', "end": '"', "patterns": [
                {"name": "constant.character.escape.germanio", "match": r'\\[nt"\\]'},
                {"name": "meta.interpolation.germanio", "match": r"(\{)(" + NAME + r")(\})",
                 "captures": {"1": {"name": "punctuation.section.interpolation.begin.germanio"},
                              "2": {"name": "variable.other.interpolated.germanio"},
                              "3": {"name": "punctuation.section.interpolation.end.germanio"}}},
            ]},
        ]},
        # Intent phrases start a line: they say what exists, who can do what,
        # what happens and what appears.
        # Blocks: a phrase that ends the line opens an indented list whose
        # lines have a known role (fields, actions, roles, data, events).
        "blocks": {"patterns": [
            {"begin": r"^\s*(?:(" + w("cada") + r")\s+)?(" + NAMES + r")\s+(" + w("tem") + r")\s*$", "end": r"^(?=\S)",
             "beginCaptures": {"1": {"name": "keyword.other.intent.germanio"}, "2": {"name": "entity.name.type.germanio"},
                               "3": {"name": "keyword.other.intent.germanio"}},
             "patterns": [{"include": "#comment"},
                          {"match": r"^\s+(membros)\s+(com\s+papel)", "captures": {"1": {"name": "entity.name.type.germanio"}, "2": {"name": "storage.modifier.germanio"}}},
                          {"match": r"^\s+(" + NAME + r")", "captures": {"1": {"name": "variable.other.property.field.germanio"}}},
                          {"include": "#expression"}]},
            {"begin": r"^\s*(" + w("tenha") + r")\s+(" + words("papeis", "papel") + r")\s*$", "end": r"^(?=\S)",
             "beginCaptures": {"1": {"name": "keyword.other.intent.germanio"}, "2": {"name": "keyword.other.intent.germanio"}},
             "patterns": [{"include": "#comment"}, {"name": "entity.name.class.role.germanio", "match": NAME},
                          {"name": "constant.numeric.germanio", "match": r"[0-9]+"}]},
            {"begin": r"^\s*(" + w("tenha") + r")\s*$", "end": r"^(?=\S)",
             "beginCaptures": {"1": {"name": "keyword.other.intent.germanio"}},
             "patterns": [{"include": "#comment"}, kw("keyword.other.intent.germanio", "papeis", "login", "cadastro", "recuperacao de senha", "avisos por e-mail"),
                          {"name": "entity.name.type.germanio", "match": NAME}]},
            {"begin": r"^\s*(" + NAMES + r")\s+(" + words("pode ser vista por", "pode ser visto por", "podem ser vistas por", "podem ser vistos por") + r")\s*$", "end": r"^(?=\S)",
             "beginCaptures": {"1": {"name": "entity.name.type.germanio"}, "2": {"name": "keyword.other.permission.germanio"}},
             "patterns": [{"include": "#comment"}, kw("keyword.other.permission.germanio", "ou superior", "ou"),
                          {"name": "entity.name.class.role.germanio", "match": NAME}]},
            {"begin": r"^\s*(?:(" + w("somente") + r")\s+)?(" + role_subject + r")\s+(" + words("podem", "pode") + r")\s*$", "end": r"^(?=\S)",
             "beginCaptures": {"1": {"name": "keyword.other.permission.germanio"},
                               "2": {"patterns": [kw("keyword.operator.logical.germanio", "ou"), {"name": "entity.name.class.role.germanio", "match": NAME}]},
                               "3": {"name": "keyword.other.permission.germanio"}},
             "patterns": [{"include": "#comment"}, {"include": "#action_line"}]},
            {"begin": r"^\s*(" + w("permita") + r")\s*$", "end": r"^(?=\S)",
             "beginCaptures": {"1": {"name": "keyword.other.permission.germanio"}},
             "patterns": [{"include": "#comment"}, {"include": "#action_line"}]},
            {"begin": r"^\s*(" + w("disponibilize para integracao") + r")\s*$", "end": r"^(?=\S)",
             "beginCaptures": {"1": {"name": "keyword.other.intent.germanio"}},
             "patterns": [{"include": "#comment"}, {"include": "#string"}, kw("keyword.other.intent.germanio", "como"),
                          {"name": "entity.name.type.germanio", "match": NAME}]},
            {"begin": r"^\s*(" + NAMES + r")\s+(" + words("recebe eventos do", "recebe eventos da") + r")\s+(" + NAMES + r")\s*$", "end": r"^(?=\S)",
             "beginCaptures": {"1": {"name": "entity.name.type.germanio"}, "2": {"name": "keyword.other.intent.germanio"}, "3": {"name": "entity.name.type.germanio"}},
             "patterns": [{"include": "#comment"}, {"match": r"^\s+(" + words("enviar codigo") + r"|" + NAMES + r")\s*$", "captures": {"1": {"name": "entity.name.function.action.germanio"}}}]},
        ]},
        # "fechar issues", "editar seu perfil", "enviar código para a branch padrão dos projetos"
        "action_line": {"patterns": [
            {"match": r"^\s+(ser)\s+(" + NAME + r")", "captures": {"1": {"name": "keyword.other.intent.germanio"}, "2": {"name": "storage.modifier.flag.germanio"}}},
            {"match": r"^\s+(" + words("baixar codigo", "enviar codigo") + r"|" + NAME + r")(?:\s+(" + words("seus", "suas", "seu", "sua") + r"))?(?:\s+(.+?))?\s*$",
             "captures": {"1": {"name": "entity.name.function.action.germanio"}, "2": {"name": "storage.modifier.possessive.germanio"},
                          "3": {"patterns": [kw("keyword.other.intent.germanio", "por", "para a", "para", "dos", "das", "do", "da", "de", "e"),
                                             {"name": "punctuation.separator.germanio", "match": ","},
                                             {"name": "entity.name.type.germanio", "match": NAME}]}}},
        ]},
        # Hierarchical data blocks (docs/INTENCAO.md › Sintaxe hierárquica): a name at
        # column 1, sections below it; a section ends at the first line that is not
        # more indented than the section itself (\\1 is the section's indentation).
        "data_block": {"patterns": [
            {"begin": r"^(" + w("pagina") + r"|page)\s+(" + NAMES + r")\s*$", "end": r"^(?=\S)",
             "beginCaptures": {"1": {"name": "keyword.other.intent.germanio"}, "2": {"name": "entity.name.section.germanio"}},
             "patterns": [{"include": "#comment"}, {"include": "#string"},
                          # page sections (docs/gep/0002-secoes-de-pagina.md)
                          {"match": r"^\s+(" + words("topo", "filtros", "colunas", "vazio", "acoes", "indicadores") + r")\s*$", "captures": {"1": {"name": "keyword.other.intent.germanio"}}},
                          {"match": r"^\s+(" + w("total de") + r")\s", "captures": {"1": {"name": "keyword.other.intent.germanio"}}},
                          {"match": r"^\s+(" + words("titulo", "texto", "acao") + r")\b", "captures": {"1": {"name": "keyword.other.intent.germanio"}}},
                          {"match": r"^\s+(" + w("mostre") + r")\s+(" + NAMES + r")\s*$", "captures": {"1": {"name": "keyword.other.intent.germanio"}, "2": {"name": "entity.name.type.germanio"}}},
                          {"match": r"^\s+(" + w("permita") + r")\s*$", "captures": {"1": {"name": "keyword.other.permission.germanio"}}},
                          {"match": r"^\s+([0-9]+)\s+(" + w("por pagina") + r")", "captures": {"1": {"name": "constant.numeric.germanio"}, "2": {"name": "keyword.other.intent.germanio"}}},
                          {"match": r"^\s+(" + NAME + r")\s*$", "captures": {"1": {"name": "entity.name.function.action.germanio"}}}]},
            {"begin": r"^(?!" + words("tenha", "crie", "permita", "disponibilize", "importar", "vocabulario", "login", "escopo", "quando", "antes", "ao", "traduza", "enderecos", "integracao", "mensagens", "logica", "rotas", "sistema", "dados", "telas", "tema", "cada", "quem", "todo", "somente") + E + r")(?!.*" + B + words("tem", "pode", "podem", "pertence", "comeca", "herda", "recebe", "executa", "executam", "precisa", "e", "vira") + E + r")(" + NAMES + r")\s*$", "end": r"^(?=\S)",
             "beginCaptures": {"1": {"name": "entity.name.type.data.germanio"}},
             "patterns": [{"include": "#comment"}, {"include": "#section"}]},
        ]},
        "section": {"patterns": [
            # tem + fields
            {"begin": r"^(\s+)(" + w("tem") + r")\s*$", "end": r"^(?!\1\s+\S|\s*$)",
             "beginCaptures": {"2": {"name": "keyword.other.intent.germanio"}},
             "patterns": [{"include": "#comment"},
                          {"match": r"^\s+(membros)\s+(com\s+papel)", "captures": {"1": {"name": "entity.name.type.germanio"}, "2": {"name": "storage.modifier.germanio"}}},
                          {"match": r"^\s+(" + NAME + r")", "captures": {"1": {"name": "variable.other.property.field.germanio"}}},
                          {"include": "#expression"}]},
            {"match": r"^\s+(" + w("tem") + r")\s+(" + NAMES + r")\s*$", "captures": {"1": {"name": "keyword.other.intent.germanio"}, "2": {"name": "entity.name.type.germanio"}}},
            # pode + capabilities
            {"begin": r"^(\s+)(" + w("pode") + r")\s*$", "end": r"^(?!\1\s+\S|\s*$)",
             "beginCaptures": {"2": {"name": "keyword.other.permission.germanio"}},
             "patterns": [{"include": "#comment"},
                          {"match": r"^\s+(ser)\s+(" + NAME + r")", "captures": {"1": {"name": "keyword.other.intent.germanio"}, "2": {"name": "storage.modifier.flag.germanio"}}},
                          {"match": r"^\s+(" + NAME + r")\s*$", "captures": {"1": {"name": "entity.name.function.action.germanio"}}}]},
            {"match": r"^\s+(" + w("pode") + r")\s+(" + NAME + r")\s*$", "captures": {"1": {"name": "keyword.other.permission.germanio"}, "2": {"name": "entity.name.function.action.germanio"}}},
            # pertence a
            {"begin": r"^(\s+)(" + w("pertence a") + r")\s*$", "end": r"^(?!\1\s+\S|\s*$)",
             "beginCaptures": {"2": {"name": "keyword.other.intent.germanio"}},
             "patterns": [{"include": "#comment"}, kw("storage.modifier.germanio", "opcional", "como"), {"name": "entity.name.type.germanio", "match": NAME}]},
            {"match": r"^\s+(" + w("pertence a") + r")\s+(" + NAME + r")(?:\s+(" + w("opcional") + r"))?", "captures": {"1": {"name": "keyword.other.intent.germanio"}, "2": {"name": "entity.name.type.germanio"}, "3": {"name": "storage.modifier.germanio"}}},
            # renomeie <antigo> para <novo>, descarte <campo> (migração explícita, G93)
            {"match": r"^\s+(renomeie)\s+(" + NAME + r")\s+(para)\s+(" + NAME + r")\s*$", "captures": {"1": {"name": "keyword.other.intent.germanio"}, "2": {"name": "variable.other.property.field.germanio"}, "3": {"name": "keyword.other.intent.germanio"}, "4": {"name": "variable.other.property.field.germanio"}}},
            {"match": r"^\s+(guarda\s+hist[oó]rico)\s*$", "captures": {"1": {"name": "keyword.other.intent.germanio"}}},
            {"match": r"^\s+(descarte)\s+(" + NAME + r")\s*$", "captures": {"1": {"name": "keyword.other.intent.germanio"}, "2": {"name": "variable.other.property.field.germanio"}}},
            # começa <estado>, singular <forma>
            {"match": r"^\s+(" + w("comeca") + r")\s+(" + NAME + r")\s*$", "captures": {"1": {"name": "keyword.other.intent.germanio"}, "2": {"name": "constant.other.state.germanio"}}},
            {"match": r"^\s+(singular)\s+(" + NAMES + r")\s*$", "captures": {"1": {"name": "keyword.other.intent.germanio"}, "2": {"name": "entity.name.type.germanio"}}},
            # regras
            {"begin": r"^(\s+)(regras)\s*$", "end": r"^(?!\1\s+\S|\s*$)",
             "beginCaptures": {"2": {"name": "keyword.other.intent.germanio"}},
             "patterns": [{"include": "#comment"},
                          {"match": r"(" + w("quem cria vira") + r")\s+(" + NAME + r")", "captures": {"1": {"name": "keyword.other.intent.germanio"}, "2": {"name": "entity.name.class.role.germanio"}}},
                          {"match": r"(" + w("precisa de pelo menos um") + r"|" + w("precisa ter pelo menos um") + r")\s+(" + NAME + r")", "captures": {"1": {"name": "keyword.other.intent.germanio"}, "2": {"name": "entity.name.class.role.germanio"}}},
                          {"match": r"(" + w("herda membros do") + r"|" + w("nao pode ser mais visivel que o") + r")\s+(" + NAMES + r")\s*$", "captures": {"1": {"name": "keyword.other.intent.germanio"}, "2": {"name": "entity.name.type.germanio"}}},
                          {"match": r"^\s+(" + NAME + r")\s+(" + words("e somente leitura", "e final", "pode ser vista por", "pode ser visto por") + r")\s*$", "captures": {"1": {"name": "constant.other.state.germanio"}, "2": {"name": "keyword.other.intent.germanio"}}},
                          kw("keyword.other.permission.germanio", "ou superior", "ou"),
                          {"match": r"^\s+(" + NAME + r")\s*$", "captures": {"1": {"name": "entity.name.class.role.germanio"}}}]},
            # acesso › papel › ações
            {"begin": r"^(\s+)(acesso)\s*$", "end": r"^(?!\1\s+\S|\s*$)",
             "beginCaptures": {"2": {"name": "keyword.other.permission.germanio"}},
             "patterns": [{"include": "#comment"},
                          {"begin": r"^(\s+)(?:(" + w("somente") + r")\s+)?(" + role_subject + r")\s*$", "end": r"^(?!\1\s+\S|\s*$)",
                           "beginCaptures": {"2": {"name": "keyword.other.permission.germanio"},
                                             "3": {"patterns": [kw("keyword.operator.logical.germanio", "ou"), {"name": "entity.name.class.role.germanio", "match": NAME}]}},
                           "patterns": [{"include": "#comment"}, {"include": "#action_line"}]}]},
            # permita
            {"begin": r"^(\s+)(" + w("permita") + r")\s*$", "end": r"^(?!\1\s+\S|\s*$)",
             "beginCaptures": {"2": {"name": "keyword.other.permission.germanio"}},
             "patterns": [{"include": "#comment"}, {"match": r"^\s+(" + NAME + r")(?:\s+(por)\s+(.+))?$",
                           "captures": {"1": {"name": "entity.name.function.action.germanio"}, "2": {"name": "keyword.other.intent.germanio"},
                                        "3": {"patterns": [kw("keyword.operator.logical.germanio", "e"), {"name": "variable.other.property.field.germanio", "match": NAME}]}}}]},
            # integração [+ nome "x"]
            {"begin": r"^(\s+)(" + w("integracao") + r")\s*$", "end": r"^(?!\1\s+\S|\s*$)",
             "beginCaptures": {"2": {"name": "keyword.other.intent.germanio"}},
             "patterns": [{"include": "#comment"}, {"include": "#string"}, kw("keyword.other.intent.germanio", "nome")]},
            # quando / antes de <verbo> + lógica
            {"begin": r"^(\s+)(" + words("quando", "antes de") + r")\s+(" + NAME + r")\s*$", "end": r"^(?!\1\s+\S|\s*$)",
             "beginCaptures": {"2": {"name": "keyword.other.event.germanio"}, "3": {"name": "entity.name.function.action.germanio"}},
             "patterns": [{"include": "#comment"}, {"include": "#expression"}]},
            # frases com sujeito implícito
            {"match": r"^\s+(" + words("recebe aprovacoes", "recebe eventos do", "recebe eventos da", "executam", "executa") + r")" + E + r"(.*)$",
             "captures": {"1": {"name": "keyword.other.intent.germanio"}, "2": {"patterns": [{"include": "#string"}, kw("keyword.other.intent.germanio", "a cada envio de codigo conforme"), {"name": "entity.name.type.germanio", "match": NAME}]}}},
            {"match": r"^\s+(" + words("repositorio pode comecar com") + r")\s+(.*)$",
             "captures": {"1": {"name": "keyword.other.intent.germanio"}, "2": {"patterns": [{"include": "#string"}, kw("keyword.other.intent.germanio", "contendo")]}}},
            {"include": "#expression"},
        ]},
        "intent": {"patterns": [
            # importar produtos e pedidos do backend
            {"match": r"^\s*(" + w("importar") + r")\s+((?:" + NAME + r")(?:\s*(?:,|\be\b)\s*" + NAME + r")*)\s+(" + words("do", "da", "de") + r")\s+(" + NAME + r")\s*$",
             "captures": {"1": {"name": "keyword.control.import.germanio"},
                          "2": {"patterns": [kw("keyword.operator.logical.germanio", "e"), {"name": "punctuation.separator.germanio", "match": ","},
                                             {"name": "entity.name.type.germanio", "match": NAME}]},
                          "3": {"name": "keyword.control.import.germanio"}, "4": {"name": "entity.name.namespace.germanio"}}},
            {"include": "#blocks"},
            {"begin": r"^\s*(" + w("crie") + r")\s+(" + w("pagina") + r")\s+(.+?)\s*$", "end": r"^(?=\S)",
             "beginCaptures": {"1": {"name": "keyword.other.intent.germanio"}, "2": {"name": "keyword.other.intent.germanio"}, "3": {"name": "entity.name.section.germanio"}},
             "patterns": [{"include": "#comment"},
                          {"match": r"^\s+(" + w("mostre") + r")\s+(" + NAMES + r")\s*$", "captures": {"1": {"name": "keyword.other.intent.germanio"}, "2": {"name": "entity.name.type.germanio"}}},
                          {"match": r"^\s+(" + w("permita") + r")\s*$", "captures": {"1": {"name": "keyword.other.permission.germanio"}}},
                          {"match": r"^\s+([0-9]+)\s+(" + w("por pagina") + r")", "captures": {"1": {"name": "constant.numeric.germanio"}, "2": {"name": "keyword.other.intent.germanio"}}},
                          {"match": r"^\s+(" + NAME + r")\s*$", "captures": {"1": {"name": "entity.name.function.action.germanio"}}}]},
            # crie sistema X / crie página X
            {"match": r"^\s*(" + w("crie") + r")\s+(" + words("sistema", "pagina", "app", "aplicacao") + r")\s+(?:(" + w("para gerenciar") + r")\s+)?(.+?)\s*$",
             "captures": {"1": {"name": "keyword.other.intent.germanio"}, "2": {"name": "keyword.other.intent.germanio"},
                          "3": {"name": "keyword.other.intent.germanio"}, "4": {"name": "entity.name.section.germanio"}}},
            # tenha papeis / tenha login / tenha X, Y
            {"begin": r"^\s*(" + w("tenha") + r")" + E, "end": r"$",
             "beginCaptures": {"1": {"name": "keyword.other.intent.germanio"}},
             "patterns": [kw("keyword.other.intent.germanio", "papeis", "papel", "login", "cadastro", "recuperacao de senha"),
                          {"name": "entity.name.type.germanio", "match": NAME}]},
            # somente X pode / X ou Y pode / X podem
            {"match": r"^\s*(?:(" + w("somente") + r")\s+)?(" + role_subject + r")\s+(" + words("podem", "pode") + r")" + E + r"(?:\s+(" + words("ser vista por", "ser visto por", "ser vistas por", "ser vistos por") + r")|\s+(ser)\s+(" + NAME + r")|\s+(" + words("baixar codigo", "enviar codigo") + r"|" + NAME + r")(?:\s+(" + words("seus", "suas", "seu", "sua") + r"))?(?:\s+(.+?))?)?\s*$",
             "captures": {"1": {"name": "keyword.other.permission.germanio"},
                          "2": {"patterns": [kw("keyword.operator.logical.germanio", "ou"), {"name": "entity.name.class.role.germanio", "match": NAME}]},
                          "3": {"name": "keyword.other.permission.germanio"},
                          "4": {"name": "keyword.other.permission.germanio"},
                          "5": {"name": "keyword.other.intent.germanio"}, "6": {"name": "storage.modifier.flag.germanio"},
                          "7": {"name": "entity.name.function.action.germanio"},
                          "8": {"name": "storage.modifier.possessive.germanio"},
                          "9": {"patterns": [kw("keyword.other.intent.germanio", "por", "para a", "para", "dos", "das", "do", "da", "de", "e"),
                                             {"name": "entity.name.type.germanio", "match": NAME}]}}},
            # cada X tem / X tem ...
            {"match": r"^\s*(?:(" + w("cada") + r")\s+)?(" + NAMES + r")\s+(" + w("tem") + r")" + E,
             "captures": {"1": {"name": "keyword.other.intent.germanio"}, "2": {"name": "entity.name.type.germanio"},
                          "3": {"name": "keyword.other.intent.germanio"}}},
            # X pertence a Y / X herda membros do Y
            {"match": r"^\s*(" + NAMES + r")\s+(" + words("pertence a", "herda membros do", "herda membros da", "herda membros de") + r")\s+(" + NAMES + r")(?:\s+(" + words("opcional") + r"))?(?:\s+(como)\s+(" + NAME + r"))?\s*$",
             "captures": {"1": {"name": "entity.name.type.germanio"}, "2": {"name": "keyword.other.intent.germanio"},
                          "3": {"name": "entity.name.type.germanio"}, "4": {"name": "storage.modifier.germanio"},
                          "5": {"name": "keyword.other.intent.germanio"}, "6": {"name": "variable.other.germanio"}}},
            # X começa aberta / X mesclado é final
            {"match": r"^\s*(" + NAMES + r")\s+(" + w("comeca") + r")\s+(" + NAME + r")\s*$",
             "captures": {"1": {"name": "entity.name.type.germanio"}, "2": {"name": "keyword.other.intent.germanio"},
                          "3": {"name": "constant.other.state.germanio"}}},
            {"match": r"^\s*(" + NAMES + r")\s+(" + NAME + r")\s+(" + words("e final") + r")\s*$",
             "captures": {"1": {"name": "entity.name.type.germanio"}, "2": {"name": "constant.other.state.germanio"},
                          "3": {"name": "keyword.other.intent.germanio"}}},
            # X não pode ser mais visível que Y
            {"match": r"^\s*(" + NAMES + r")\s+(" + w("nao pode ser mais visivel que") + r")\s+(.+?)\s*$",
             "captures": {"1": {"name": "entity.name.type.germanio"}, "2": {"name": "keyword.other.permission.germanio"},
                          "3": {"name": "entity.name.type.germanio"}}},
            # X recebe aprovações / X recebe eventos do Y / X executa Ys a cada envio de código conforme "f"
            {"match": r"^\s*(" + NAMES + r")\s+(" + words("recebe aprovacoes", "recebe eventos do", "recebe eventos da") + r")(?:\s+(" + NAMES + r"))?\s*$",
             "captures": {"1": {"name": "entity.name.type.germanio"}, "2": {"name": "keyword.other.intent.germanio"},
                          "3": {"name": "entity.name.type.germanio"}}},
            {"match": r"^\s*(" + NAMES + r")\s+(" + w("executa") + r")\s+(" + NAMES + r")\s+(" + w("a cada envio de codigo conforme") + r")",
             "captures": {"1": {"name": "entity.name.type.germanio"}, "2": {"name": "keyword.other.intent.germanio"},
                          "3": {"name": "entity.name.type.germanio"}, "4": {"name": "keyword.other.intent.germanio"}}},
            # quem cria X vira Y
            {"match": r"^\s*(" + w("quem cria") + r")\s+(" + NAMES + r")\s+(" + w("vira") + r")\s+(" + NAME + r")",
             "captures": {"1": {"name": "keyword.other.intent.germanio"}, "2": {"name": "entity.name.type.germanio"},
                          "3": {"name": "keyword.other.intent.germanio"}, "4": {"name": "entity.name.class.role.germanio"}}},
            # quando / antes de <ação> <dado>
            {"match": r"^\s*(" + words("antes de", "quando") + r")\s+(" + words("enviar codigo para", "enviar codigo") + r"|" + NAME + r")\s+(" + NAMES + r")\s*$",
             "captures": {"1": {"name": "keyword.other.event.germanio"}, "2": {"name": "entity.name.function.action.germanio"},
                          "3": {"name": "entity.name.type.germanio"}}},
            {"match": r"^\s*(" + w("ao iniciar") + r")\s*$", "captures": {"1": {"name": "keyword.other.event.germanio"}}},
            # permita / disponibilize / integração / vocabulário / mensagens / login / escopo
            {"match": r"^\s*(" + w("permita") + r")" + E, "captures": {"1": {"name": "keyword.other.permission.germanio"}}},
            {"match": r"^\s*(" + w("disponibilize") + r")" + E, "captures": {"1": {"name": "keyword.other.intent.germanio"}}},
            {"match": r"^\s*(" + words("vocabulario da integracao", "integracao em", "mensagens em") + r")" + E,
             "captures": {"1": {"name": "keyword.other.intent.germanio"}}},
            {"match": r"^\s*(login)\s+(" + words("usa", "aceita", "bloqueia", "exige") + r")" + E,
             "captures": {"1": {"name": "keyword.other.intent.germanio"}, "2": {"name": "keyword.other.intent.germanio"}}},
            {"match": r"^\s*(" + w("escopo") + r")\s+(\"[^\"]*\")\s+(" + w("permite") + r")",
             "captures": {"1": {"name": "keyword.other.intent.germanio"}, "2": {"name": "string.quoted.double.germanio"},
                          "3": {"name": "keyword.other.permission.germanio"}}},
            # vocabulary lines: nome é "externo"
            {"match": r"^\s+(" + NAME + r")\s+(" + w("e") + r")\s+(\"[^\"]*\")\s*$",
             "captures": {"1": {"name": "variable.other.property.germanio"}, "2": {"name": "keyword.operator.germanio"},
                          "3": {"name": "string.quoted.double.germanio"}}},
            # legacy blocks and routes
            {"match": r"^\s*(rota)\s+(GET|POST|PUT|PATCH|DELETE|HEAD)" + E,
             "captures": {"1": {"name": "keyword.other.block.germanio"}, "2": {"name": "constant.language.http-method.germanio"}}},
            {"match": r"^(" + words(*blocks) + r")" + E, "captures": {"1": {"name": "keyword.other.block.germanio"}}},
            {"match": r"^\s*(" + w("funcao") + r")\s+(" + NAME + r")",
             "captures": {"1": {"name": "storage.type.function.germanio"}, "2": {"name": "entity.name.function.germanio"}}},
        ]},
        "expression": {"patterns": [
            {"include": "#string"},
            {"include": "#comment"},
            {"name": "constant.numeric.germanio", "match": B + r"-?[0-9]+(?:\.[0-9]+)?" + E},
            kw("constant.language.germanio", "verdadeiro", "falso", "nulo", "sim"),
            kw("keyword.control.germanio", *control),
            kw("keyword.operator.logical.germanio", *logical),
            kw("support.type.germanio", *types),
            {"name": "support.type.germanio", "match": B + w("lista de") + E},
            kw("storage.modifier.germanio", *modifiers),
            kw("support.function.builtin.germanio", "recuse", "falhar", "responder", "redirecionar"),
            kw("variable.language.germanio", *ctxvars),
            kw("keyword.other.permission.germanio", "ou superior", "administrador", "todos"),
            {"match": B + r"(" + words(*modules) + r")(\.)(" + NAME + r")",
             "captures": {"1": {"name": "support.class.module.germanio"}, "2": {"name": "punctuation.accessor.germanio"},
                          "3": {"name": "support.function.germanio"}}},
            {"match": B + r"(" + NAME + r")(\.)(" + NAME + r")(?=\s*\()",
             "captures": {"1": {"name": "variable.other.object.germanio"}, "2": {"name": "punctuation.accessor.germanio"},
                          "3": {"name": "entity.name.function.germanio"}}},
            {"match": B + r"(" + NAME + r")(?=\s*\()", "captures": {"1": {"name": "entity.name.function.germanio"}}},
            {"match": B + r"(" + NAME + r")\s*(:)(?!:)", "captures": {"1": {"name": "variable.other.property.germanio"},
                                                                        "2": {"name": "punctuation.separator.key-value.germanio"}}},
            {"name": "keyword.operator.comparison.germanio", "match": r"==|!=|<=|>=|<|>"},
            {"name": "keyword.operator.arithmetic.germanio", "match": r"[+\-*/%]"},
            {"name": "keyword.operator.assignment.germanio", "match": r"="},
            {"name": "punctuation.separator.germanio", "match": r","},
        ]},
    },
}

here = os.path.dirname(os.path.abspath(__file__))
with open(os.path.join(here, "..", "syntaxes", "germanio.tmLanguage.json"), "w", encoding="utf-8") as f:
    json.dump(grammar, f, ensure_ascii=False, indent=2)
    f.write("\n")
print("ok")
