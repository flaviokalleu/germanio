package runtime

import (
	"encoding/json"
	"reflect"
	"testing"
)

// meaningOf is the resolved application without positions: two sources
// that say the same thing resolve to equal values here.
func meaningOf(t *testing.T, file string) any {
	t.Helper()
	prog, err := Compilar(file)
	if err != nil {
		t.Fatalf("%s: %v", file, err)
	}
	b, err := json.Marshal(prog.App)
	if err != nil {
		t.Fatal(err)
	}
	var v any
	json.Unmarshal(b, &v)
	var strip func(any)
	strip = func(x any) {
		switch m := x.(type) {
		case map[string]any:
			for _, k := range []string{"Pos", "File", "Line", "Column", "Context", "from", "to"} {
				delete(m, k)
			}
			for _, val := range m {
				strip(val)
			}
		case []any:
			for _, val := range m {
				strip(val)
			}
		}
	}
	strip(v)
	return v
}

// A program split in files means exactly what the same program in one file
// means: login, sign-up, password recovery and the lock, roles, membership,
// permissions, states, relations, pending items and page sections.
func TestProjetoDivididoSignificaOMesmo(t *testing.T) {
	one := meaningOf(t, "testdata/composicao/unico.ge")
	split := meaningOf(t, "testdata/composicao/organizado/app.ge")
	if !reflect.DeepEqual(one, split) {
		a, _ := json.MarshalIndent(one, "", " ")
		b, _ := json.MarshalIndent(split, "", " ")
		t.Fatalf("o projeto dividido significa outra coisa:\nunico: %.3000s\n\ndividido: %.3000s", a, b)
	}
	app := one.(map[string]any)
	login := app["Login"].(map[string]any)
	if login["Recovery"] != true || login["Signup"] != true || login["LockAttempts"] != float64(5) {
		t.Fatalf("login incompleto: %v", login)
	}
	if app["PendingEntity"] != "pendencia" {
		t.Fatalf("pendências ausentes: %v", app["PendingEntity"])
	}
}
