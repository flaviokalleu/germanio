package runtime

import "testing"

// Guessing passwords must be stopped by default: an application that only
// says `tenha login` locks an account after 10 wrong passwords, and the
// right password is refused while the lock lasts (docs/INTENCAO.md › Login).
// `login bloqueia após N tentativas por M minutos` only changes the numbers.
func TestLoginBloqueiaPorPadrao(t *testing.T) {
	for _, tc := range []struct {
		file     string
		attempts int
	}{
		{"testdata/login/padrao.ge", 10},
		{"testdata/login/ajustado.ge", 3},
	} {
		t.Run(tc.file, func(t *testing.T) {
			app, c := loadApp(t, tc.file)
			if app.Program.App.Login.LockAttempts != tc.attempts {
				t.Fatalf("tentativas até bloquear = %d, esperado %d", app.Program.App.Login.LockAttempts, tc.attempts)
			}
			c.expect("POST", "/cadastro", map[string]any{"nome": "Ana", "email": "ana@x.com", "senha": "senha-certa-1"}, 201)
			c.expect("POST", "/sair", nil, 204)
			for i := 1; i < tc.attempts; i++ {
				c.expect("POST", "/entrar", map[string]any{"login": "ana@x.com", "senha": "errada"}, 401)
			}
			c.expect("POST", "/entrar", map[string]any{"login": "ana@x.com", "senha": "senha-certa-1"}, 200)
			c.expect("POST", "/sair", nil, 204)
			for i := 0; i < tc.attempts; i++ {
				c.expect("POST", "/entrar", map[string]any{"login": "ana@x.com", "senha": "errada"}, 401)
			}
			c.expect("POST", "/entrar", map[string]any{"login": "ana@x.com", "senha": "senha-certa-1"}, 401)
		})
	}
}
