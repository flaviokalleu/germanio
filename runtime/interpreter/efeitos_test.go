package interpreter

import "testing"

// An effect sends the values as they were when it was asked.
func TestEfeitoGuardaOsValoresDoPedido(t *testing.T) {
	dados := map[string]any{"status": "novo", "itens": []any{"a"}}
	saved := snapshot([]any{"https://x.example", dados}).([]any)
	dados["status"] = "mudou"
	dados["itens"].([]any)[0] = "b"
	got := saved[1].(map[string]any)
	if got["status"] != "novo" || got["itens"].([]any)[0] != "a" {
		t.Fatalf("o efeito viu a mudança posterior: %v", got)
	}
}
