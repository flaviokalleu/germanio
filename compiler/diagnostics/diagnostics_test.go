package diagnostics

import (
	"strings"
	"testing"
)

func TestEducationalDiagnostic(t *testing.T) {
	d := &Diagnostic{Code: "GE2004", Position: Position{File: "teste.ge", Line: 2, Column: 3}, Source: "# exemplo\n  mostre x", Message: "Tipos incompatíveis", Reason: "texto não é inteiro", Fix: "use numero", Example: `mostre numero("5")`}
	for _, part := range []string{"GE2004", "teste.ge:2:3", "2 |   mostre x", "^", "Por quê:", "Como corrigir:", "Exemplo:"} {
		if !strings.Contains(d.Error(), part) {
			t.Errorf("missing %s", part)
		}
	}
}
