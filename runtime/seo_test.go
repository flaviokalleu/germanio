package runtime

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// SEO técnico como capability genérica: 404 de verdade, robots.txt,
// sitemap.xml e metatags a partir do que o programa já declara.

func loadSEO(t *testing.T, file, publica string) string {
	t.Helper()
	t.Setenv("GERMANIO_SQLITE", filepath.Join(t.TempDir(), "app.db"))
	t.Setenv("GERMANIO_BCRYPT_RAPIDO", "1")
	t.Setenv("GERMANIO_URL_PUBLICA", publica)
	app, err := Carregar(file, "0")
	if err != nil {
		t.Fatalf("Carregar(%s): %v", file, err)
	}
	srv := httptest.NewServer(app.Handler)
	t.Cleanup(func() { srv.Close(); app.Fechar() })
	return srv.URL
}

func get(t *testing.T, url string) (int, string, string) {
	t.Helper()
	c := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := c.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, resp.Header.Get("Content-Type"), string(b)
}

func TestSEO404DeVerdade(t *testing.T) {
	base := loadSEO(t, "testdata/seo/site/inicio.ge", "")
	for _, p := range []string{"/naoexiste", "/sobre/extra", "/favicon.ico", "/sitemap.xml"} {
		if code, _, body := get(t, base+p); code != 404 || strings.Contains(body, "<script") {
			t.Fatalf("%s: status %d, esperado 404 sem o shell; corpo %.200s", p, code, body)
		}
	}
	for _, p := range []string{"/", "/sobre", "/assets/logo.png", "/health"} {
		if code, _, _ := get(t, base+p); code != 200 {
			t.Fatalf("%s: status %d, esperado 200", p, code)
		}
	}
	// A API continua respondendo JSON, não a página 404.
	if code, ct, _ := get(t, base+"/api/naoexiste"); code == 200 || strings.Contains(ct, "text/html") {
		t.Fatalf("/api/naoexiste: %d %s", code, ct)
	}
}

func TestSEO404NaoQuebraAInterfaceGerada(t *testing.T) {
	// Sem páginas: "/" continua sendo a interface gerada; o resto é 404.
	base := loadSEO(t, "testdata/fullstack/app.ge", "https://exemplo.com")
	if code, _, body := get(t, base+"/"); code != 200 || !strings.Contains(body, "<!DOCTYPE html>") {
		t.Fatalf("/: %d", code)
	}
	for _, p := range []string{"/qualquer", "/robots.txt", "/sitemap.xml"} {
		if code, _, _ := get(t, base+p); code != 404 {
			t.Fatalf("%s: status %d, esperado 404 (app sem páginas)", p, code)
		}
	}
}

func TestSEORobotsESitemap(t *testing.T) {
	base := loadSEO(t, "testdata/seo/site/inicio.ge", "https://vitrine.exemplo/")
	code, ct, robots := get(t, base+"/robots.txt")
	if code != 200 || !strings.HasPrefix(ct, "text/plain") {
		t.Fatalf("robots: %d %s", code, ct)
	}
	want := "User-agent: *\nDisallow: /api/\nDisallow: /_ge/\nSitemap: https://vitrine.exemplo/sitemap.xml\n"
	if robots != want {
		t.Fatalf("robots.txt:\n%s\nesperado:\n%s", robots, want)
	}
	code, ct, sm := get(t, base+"/sitemap.xml")
	if code != 200 || !strings.Contains(ct, "xml") {
		t.Fatalf("sitemap: %d %s", code, ct)
	}
	for _, loc := range []string{"<loc>https://vitrine.exemplo/</loc>", "<loc>https://vitrine.exemplo/sobre</loc>"} {
		if !strings.Contains(sm, loc) {
			t.Fatalf("sitemap sem %s:\n%s", loc, sm)
		}
	}
	if !strings.Contains(sm, `xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"`) {
		t.Fatalf("sitemap sem namespace:\n%s", sm)
	}
}

func TestSEOSemURLPublicaNaoInventaEndereco(t *testing.T) {
	base := loadSEO(t, "testdata/seo/site/inicio.ge", "")
	_, _, robots := get(t, base+"/robots.txt")
	if strings.Contains(robots, "Sitemap:") || !strings.Contains(robots, "Disallow: /api/") {
		t.Fatalf("robots sem URL pública:\n%s", robots)
	}
	_, _, page := get(t, base+"/")
	for _, tag := range []string{`rel="canonical"`, `og:url`, `og:image`, `twitter:card`} {
		if strings.Contains(page, tag) {
			t.Fatalf("sem URL pública a página não deve ter %s", tag)
		}
	}
	if !strings.Contains(page, `<link rel="icon" href="/assets/logo.png" type="image/png">`) {
		t.Fatalf("favicon do logo declarado ausente")
	}
}

func TestSEORobotsDoAutorVence(t *testing.T) {
	base := loadSEO(t, "testdata/seo/proprio/inicio.ge", "https://proprio.exemplo")
	if code, _, body := get(t, base+"/robots.txt"); code != 200 || body != "User-agent: *\nDisallow: /\n" {
		t.Fatalf("robots do autor: %d %q", code, body)
	}
	if code, _, _ := get(t, base+"/sitemap.xml"); code != 200 {
		t.Fatalf("sitemap gerado continua disponível: %d", code)
	}
}

func TestSEOMetatagsEscapadas(t *testing.T) {
	base := loadSEO(t, "testdata/seo/site/inicio.ge", "https://vitrine.exemplo")
	_, _, page := get(t, base+"/")
	for _, tag := range []string{
		`<meta name="description" content="Doces &#34;caseiros&#34; &amp; &lt;salgados&gt; desde 1990">`,
		`<link rel="canonical" href="https://vitrine.exemplo/">`,
		`<meta property="og:url" content="https://vitrine.exemplo/">`,
		`<meta property="og:type" content="website">`,
		`<meta property="og:title" content="Vitrine &amp; Cia">`,
		`<meta property="og:description" content="Doces &#34;caseiros&#34; &amp; &lt;salgados&gt; desde 1990">`,
		`<meta property="og:image" content="https://vitrine.exemplo/assets/logo.png">`,
		`<meta name="twitter:card" content="summary_large_image">`,
		`<link rel="icon" href="/assets/logo.png" type="image/png">`,
	} {
		if !strings.Contains(page, tag) {
			t.Fatalf("página sem %s", tag)
		}
	}
	if strings.Contains(page, "<salgados>") {
		t.Fatal("descrição sem escape")
	}
	if n := strings.Count(page, `class="header"`); n != 1 {
		t.Fatalf("a página desenha %d cabeçalhos, esperado só o declarado", n)
	}
	// Sem descrição declarada (e sem descrição do sistema), a tag é omitida.
	_, _, sobre := get(t, base+"/sobre")
	if strings.Contains(sobre, `name="description"`) || strings.Contains(sobre, "og:description") {
		t.Fatal("/sobre não declara descrição: a tag deve ser omitida")
	}
	if !strings.Contains(sobre, `<link rel="canonical" href="https://vitrine.exemplo/sobre">`) {
		t.Fatal("/sobre sem canonical")
	}
}

func TestSEOPaginasDeIntencao(t *testing.T) {
	base := loadSEO(t, "testdata/seo/intencao/app.ge", "https://loja.exemplo")
	_, _, sm := get(t, base+"/sitemap.xml")
	if !strings.Contains(sm, "<loc>https://loja.exemplo/produtos</loc>") {
		t.Fatalf("sitemap sem a página pública:\n%s", sm)
	}
	if strings.Contains(sm, "/pedidos") {
		t.Fatalf("sitemap lista página que exige login:\n%s", sm)
	}
	code, _, page := get(t, base+"/produtos")
	if code != 200 {
		t.Fatalf("/produtos: %d", code)
	}
	for _, tag := range []string{`<link rel="canonical" href="https://loja.exemplo/produtos">`, `<meta property="og:title" content="Produtos · Loja">`} {
		if !strings.Contains(page, tag) {
			t.Fatalf("página de intenção sem %s", tag)
		}
	}
	if code, _, _ := get(t, base+"/naoexiste"); code != 404 {
		t.Fatalf("/naoexiste numa app de intenção: %d", code)
	}
}
