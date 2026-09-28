package interpreter

import (
	"strings"
	"testing"
)

func TestMarkdownSeguro(t *testing.T) {
	out := Markdown("# Título\n\n**forte** e `código`\n\n- [x] feito\n\n| a | b |\n|---|---|\n| 1 | 2 |\n")
	for _, want := range []string{"<h1>Título</h1>", "<strong>forte</strong>", "<code>código</code>", "<table>", `type="checkbox"`} {
		if !strings.Contains(out, want) {
			t.Errorf("faltou %q em %s", want, out)
		}
	}
	ataques := []string{
		"<script>alert(1)</script>",
		`<img src=x onerror="alert(1)">`,
		"[clique](javascript:alert(1))",
		"[clique](JaVaScRiPt:alert(1))",
		"![x](javascript:alert(1))",
		"[x](data:text/html;base64,PHNjcmlwdD4=)",
		`<a href="javascript:alert(1)">x</a>`,
		"<iframe src=//evil></iframe>",
	}
	for _, a := range ataques {
		low := strings.ToLower(Markdown(a))
		for _, bad := range []string{"<script", "onerror", "javascript:", "<iframe", "data:text/html", "<img src=x"} {
			if strings.Contains(low, bad) {
				t.Errorf("%q gerou %q", a, low)
			}
		}
	}
}
