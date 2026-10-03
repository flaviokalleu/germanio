package servidor

import (
	"strings"
	"testing"

	"github.com/flaviokalleu/germanio/compiler/ast"
)

func TestPaginaImagemSegura(t *testing.T) {
	for _, u := range []string{"/assets/montanhas.png", "/a/b-c_d.webp"} {
		if _, ok := pageImageURL(u); !ok {
			t.Errorf("recusou %q", u)
		}
	}
	for _, u := range []string{"", "assets/x.png", "//evil.example/x.png", "https://evil.example/x.png",
		"javascript:alert(1)", "/x.png') ; background:url('//evil", "/x\".png", "/a b.png", "/x.png)\n"} {
		if _, ok := pageImageURL(u); ok {
			t.Errorf("aceitou %q", u)
		}
	}
}

func TestPaginaFundosPontosEFaixas(t *testing.T) {
	page := &ast.CustomPage{Title: "T", Blocks: []ast.PageUIBlock{
		&ast.PageHero{Title: "Oi", Background: "/assets/m.png", Points: []string{"<b>um</b>"}},
		&ast.PageSection{Title: "S", Light: true, Image: "/assets/e.png", Columns: 4, Cards: []*ast.PageCard{{Title: "c"}}},
		&ast.PageSection{Title: "C", CodeBlocks: []*ast.PageCodeBlock{{Title: "app.ge", Code: "crie sistema X"}}},
		&ast.PageHero{Title: "Mau", Background: "/x.png') ; x:url('//evil"},
	}}
	out := RenderDeclarativePage(page)
	for _, want := range []string{"hero-photo", "url('/assets/m.png')", "&lt;b&gt;um&lt;/b&gt;", "section section-light",
		`src="/assets/e.png"`, `class="grid-4"`, "section-split", "editor-card section-editor"} {
		if !strings.Contains(out, want) {
			t.Errorf("faltou %q", want)
		}
	}
	if strings.Contains(out, "evil") {
		t.Error("endereço inseguro chegou à página")
	}
}
