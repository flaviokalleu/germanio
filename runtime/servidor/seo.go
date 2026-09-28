package servidor

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
)

// Technical SEO for every application, computed once per program load
// (hot reload re-executes the process): the meta tags of each declared page,
// /robots.txt, /sitemap.xml and a real 404 for addresses that do not exist.
// Nothing here needs new syntax; the sources are what the program already
// declares (page title, hero description, navbar logo, `tema icone`) plus the
// public address of the deployment (GERMANIO_URL_PUBLICA, docs/DEPLOY.md).

// EnvURLPublica names the environment variable with the public address of
// the application (for example https://germanio.dev). Without it canonical,
// og:url, og:image and the sitemap are omitted: an address cannot be guessed
// from the Host header of a request without trusting the client.
const EnvURLPublica = "GERMANIO_URL_PUBLICA"

type siteSEO struct {
	base    string            // public URL without the final slash; "" when unknown
	pages   map[string]string // declarative page path → rendered HTML
	robots  []byte            // generated /robots.txt; nil → not served
	sitemap []byte            // generated /sitemap.xml; nil → not served
	// files the author placed in the assets folder; they win over generated ones
	files map[string]string // "/robots.txt" → file on disk
	shell string            // meta tags of the generated interface at "/"
}

// publicBase validates GERMANIO_URL_PUBLICA: an absolute http(s) URL without
// query or fragment. Anything else is ignored with a warning.
func publicBase() string {
	raw := strings.TrimSpace(os.Getenv(EnvURLPublica))
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.RawQuery != "" || u.Fragment != "" {
		fmt.Printf("[germanio] %s ignorada: use um endereço completo como https://exemplo.com\n", EnvURLPublica)
		return ""
	}
	return strings.TrimRight(u.String(), "/")
}

// seo returns the SEO data of the program, building it on first use.
func (s *Servidor) seo() *siteSEO {
	s.seoOnce.Do(func() { s.seoData = s.buildSEO() })
	return s.seoData
}

func (s *Servidor) assetsDir() string {
	if s.AssetsDir != "" {
		return s.AssetsDir
	}
	return "assets"
}

func (s *Servidor) buildSEO() *siteSEO {
	d := &siteSEO{base: publicBase(), pages: map[string]string{}, files: map[string]string{}}
	for _, name := range []string{"robots.txt", "sitemap.xml", "favicon.ico"} {
		p := filepath.Join(s.assetsDir(), name)
		if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() {
			d.files["/"+name] = p
		}
	}
	system := ""
	if s.Program.System != nil {
		system = s.Program.System.Name
	}
	themeIcon := ""
	if s.Program.Theme != nil && isImageRef(s.Program.Theme.Icon) {
		themeIcon = s.Program.Theme.Icon
	}

	var public []string
	for _, pg := range s.Program.Pages {
		if pg.Path == "" {
			continue
		}
		public = append(public, pg.Path)
		if len(pg.Blocks) == 0 {
			continue
		}
		d.pages[pg.Path] = renderDeclarativePage(pg, d.metaFor(pg, system, themeIcon))
	}
	if app := s.Program.App; app != nil {
		for _, pg := range app.Pages {
			if pg.Show != "" && publicIntentPage(app, pg) {
				public = append(public, "/"+slug(pg.Name))
			}
		}
	}
	d.shell = metaHTML(pageMeta{Title: system, Canonical: d.abs("/"), Icon: themeIcon, Image: d.abs(themeIcon)})

	if len(public) == 0 {
		return d // no pages: nothing to index
	}
	var r strings.Builder
	r.WriteString("User-agent: *\nDisallow: /api/\nDisallow: /_ge/\n")
	if d.base != "" {
		r.WriteString("Sitemap: " + d.base + "/sitemap.xml\n")
		d.sitemap = sitemapXML(d.base, public)
	}
	d.robots = []byte(r.String())
	return d
}

// publicIntentPage: a page of the intent layer is listed only when anyone,
// signed in or not, may see what it shows.
func publicIntentPage(app *ast.App, pg *ast.PageDecl) bool {
	if app.Login == nil {
		return true
	}
	e := app.Entities[pg.Show]
	if e == nil || e.Visibility != "" || len(e.Restrictions) > 0 {
		return false
	}
	for _, r := range e.Rules["ver"] {
		if r.Anyone && !r.Own {
			return true
		}
	}
	return false
}

func sitemapXML(base string, paths []string) []byte {
	seen := map[string]bool{}
	var list []string
	for _, p := range paths {
		if !seen[p] {
			seen[p] = true
			list = append(list, p)
		}
	}
	sort.Strings(list)
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	for _, p := range list {
		b.WriteString("  <url><loc>")
		template.HTMLEscape(&b, []byte(base+p))
		b.WriteString("</loc></url>\n")
	}
	b.WriteString("</urlset>\n")
	return b.Bytes()
}

// abs turns a path declared by the program into an absolute public URL, or
// "" when that is not possible.
func (d *siteSEO) abs(ref string) string {
	switch {
	case ref == "":
		return ""
	case strings.HasPrefix(ref, "https://") || strings.HasPrefix(ref, "http://"):
		return ref
	case d.base == "" || !strings.HasPrefix(ref, "/"):
		return ""
	}
	return d.base + ref
}

func isImageRef(ref string) bool {
	return strings.HasPrefix(ref, "/") || strings.HasPrefix(ref, "https://") || strings.HasPrefix(ref, "http://")
}

func (d *siteSEO) metaFor(pg *ast.CustomPage, system, themeIcon string) pageMeta {
	m := pageMeta{Title: pageTitle(pg, system), Canonical: d.abs(pg.Path)}
	logo := themeIcon
	for _, bl := range pg.Blocks {
		switch x := bl.(type) {
		case *ast.PageHero:
			if m.Description == "" {
				m.Description = strings.TrimSpace(x.Description)
			}
		case *ast.PageNavbar:
			if isImageRef(x.Logo) {
				logo = x.Logo
			}
		}
	}
	m.Icon = logo
	m.Image = d.abs(logo)
	return m
}

func pageTitle(pg *ast.CustomPage, system string) string {
	if pg.Title != "" {
		return pg.Title
	}
	return system
}

type pageMeta struct {
	Title, Description, Canonical, Image, Icon string
}

func (m pageMeta) IconType() string {
	switch strings.ToLower(path.Ext(strings.SplitN(m.Icon, "?", 2)[0])) {
	case ".png":
		return "image/png"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	}
	return ""
}

var metaTpl = template.Must(template.New("meta").Parse(`{{with .Description}}<meta name="description" content="{{.}}">
{{end}}{{with .Canonical}}<link rel="canonical" href="{{.}}">
<meta property="og:url" content="{{.}}">
{{end}}<meta property="og:type" content="website">
{{with .Title}}<meta property="og:title" content="{{.}}">
{{end}}{{with .Description}}<meta property="og:description" content="{{.}}">
{{end}}{{with .Image}}<meta property="og:image" content="{{.}}">
<meta name="twitter:card" content="summary_large_image">
{{end}}{{with .Icon}}<link rel="icon" href="{{.}}"{{with $.IconType}} type="{{.}}"{{end}}>
{{end}}`))

// metaHTML renders the tags with html/template (every value is escaped).
func metaHTML(m pageMeta) string {
	var b bytes.Buffer
	if err := metaTpl.Execute(&b, m); err != nil {
		return ""
	}
	return b.String()
}

// serveSEO answers /robots.txt, /sitemap.xml and /favicon.ico. It reports
// false when the path is none of them.
func (s *Servidor) serveSEO(w http.ResponseWriter, r *http.Request) bool {
	p := r.URL.Path
	if p != "/robots.txt" && p != "/sitemap.xml" && p != "/favicon.ico" {
		return false
	}
	d := s.seo()
	if f, ok := d.files[p]; ok {
		w.Header().Del("Content-Type")
		http.ServeFile(w, r, f)
		return true
	}
	var body []byte
	switch p {
	case "/robots.txt":
		body = d.robots
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	case "/sitemap.xml":
		body = d.sitemap
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	}
	if body == nil {
		notFound(w)
		return true
	}
	_, _ = w.Write(body)
	return true
}

// notFound is the answer for an address the program does not declare.
func notFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(`<!DOCTYPE html><html lang="pt-BR"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="robots" content="noindex"><title>Página não encontrada</title><style>body{font-family:system-ui,sans-serif;margin:0;padding:48px 24px;text-align:center}</style></head><body><h1>Página não encontrada</h1><p>Este endereço não existe.</p><p><a href="/">Voltar ao início</a></p></body></html>`))
}
