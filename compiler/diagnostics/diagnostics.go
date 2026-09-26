// Package diagnostics defines stable, source-located Germanio diagnostics.
package diagnostics

import (
	"fmt"
	"strings"
)

type Position struct {
	File         string
	Line, Column int
}
type Diagnostic struct {
	Code                                  string
	Position                              Position
	Source, Message, Reason, Fix, Example string
}

func (d *Diagnostic) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s — %s\n%s:%d:%d\n", d.Code, d.Message, d.Position.File, d.Position.Line, d.Position.Column)
	lines := strings.Split(d.Source, "\n")
	if d.Position.Line > 0 && d.Position.Line <= len(lines) {
		fmt.Fprintf(&b, "%d | %s\n    %s^\n", d.Position.Line, lines[d.Position.Line-1], strings.Repeat(" ", max(0, d.Position.Column-1)))
	}
	fmt.Fprintf(&b, "Por quê: %s\nComo corrigir: %s", d.Reason, d.Fix)
	if d.Example != "" {
		fmt.Fprintf(&b, "\nExemplo:\n%s", d.Example)
	}
	return b.String()
}

var Explanations = map[string]string{
	"GE1001": "Sintaxe inválida. Confira a palavra, os delimitadores e a expressão indicada. Exemplo: mostre \"Olá\".",
	"GE1002": "Indentação inválida. Use dois espaços por bloco e alinhe comandos do mesmo bloco.",
	"GE2001": "Nome desconhecido. Declare a variável ou função antes de utilizá-la.",
	"GE2002": "Valor imutável. Declare variavel contador = 0 antes de modificar contador. mut continua aceito.",
	"GE2003": "Nome não utilizado. Use o valor ou remova a declaração. Para descarte intencional, use _ = valor.",
	"GE2004": "Tipos incompatíveis. Germanio não converte texto em número implicitamente. Exemplo: numero(\"5\") + 2.",
	"GE2005": "Operação numérica inválida, divisão por zero ou overflow. Verifique os operandos e limites.",
	"GE2006": "Índice fora dos limites. Use um índice inteiro entre zero e quantidade(lista) - 1.",
	"GE2007": "Código inalcançável. Remova comandos depois de retorne, pare ou continue.",
	"GE3001": "Módulo inválido. Use um arquivo .ge relativo ao projeto, sem ciclos de importação.",
	"GE3002": "Função privada. Exporte uma função pública que a chame, por exemplo: publico(x) = interna(x).",
	"GE4102": "Unidade incorreta. Arredondamento usa comprimento, como 16px; rotação usa ângulo, como 45deg.",
	"GE9001": "Limite de execução excedido. Verifique a condição do loop ou da recursão.",
}
