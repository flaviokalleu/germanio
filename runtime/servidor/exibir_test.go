package servidor

import (
	"testing"
	"time"

	"github.com/flaviokalleu/germanio/compiler/ast"
)

// People read dates the way they write them; forms keep the browser's form.
func TestDatasParaPessoas(t *testing.T) {
	old := time.Local
	time.Local = time.UTC
	defer func() { time.Local = old }()
	for _, c := range []struct {
		tipo ast.FieldType
		v    any
		want string
	}{
		{ast.FieldData, "2026-10-03", "03/10/2026"},
		{"", "2026-09-29T00:44:28Z", "29/09/2026 00:44"},
		{ast.FieldTexto, "2026-10-03", "2026-10-03"}, // a text that looks like a date stays text
		{ast.FieldData, nil, "—"},
		{ast.FieldTexto, "aberta", "aberta"},
	} {
		if got := displayFor(c.tipo, c.v); got != c.want {
			t.Errorf("displayFor(%q, %v) = %q, esperado %q", c.tipo, c.v, got, c.want)
		}
	}
	if got := display("2026-10-03"); got != "2026-10-03" {
		t.Errorf("formulários mantêm a forma técnica: %q", got)
	}
}
