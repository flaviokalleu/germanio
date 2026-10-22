package servidor

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
)

// This file renders the declarative "elementos" tree (see ast.PageElement)
// into plain HTML + hand-written CSS, with zero JavaScript. It is the
// pure-Germanio alternative to a `pagina`'s raw `conteudo` HTML string —
// the whole point being that a landing page can be built "apenas
// digitando", without the author ever touching HTML/CSS/JS.
//
// Interactivity (the navbar's mobile menu, the tabbed code/demo section) is
// done with the CSS-only checkbox/radio "hack" (:checked ~ selector), never
// <script>.

type pageRenderer struct {
	theme      *ast.Theme
	elCounter  int
	navCounter int
	tabCounter int
	css        strings.Builder
}

// RenderPageElements renders a page's declarative element tree into body
// HTML and accompanying CSS (meant to be injected into the page's
// <style>). theme may be nil, in which case ast.DefaultTheme() is used.
func RenderPageElements(elements []*ast.PageElement, theme *ast.Theme) (string, string) {
	if theme == nil {
		theme = ast.DefaultTheme()
	}
	r := &pageRenderer{theme: theme}
	r.writeBaseCSS()

	var body strings.Builder
	for _, el := range elements {
		body.WriteString(r.renderElement(el))
	}
	return body.String(), r.css.String()
}

// ==================== base CSS ====================

func (r *pageRenderer) writeBaseCSS() {
	t := r.theme
	font := t.Font
	if font == "" {
		font = "Inter, system-ui, sans-serif"
	} else {
		font = "'" + font + "', system-ui, sans-serif"
	}
	radius := t.Radius
	if radius == "" {
		radius = "12px"
	}
	fmt.Fprintf(&r.css, `
*{box-sizing:border-box}
body{margin:0;padding:0;font-family:%s;background:%s;color:%s;line-height:1.5;overflow-x:hidden}
a{color:inherit;text-decoration:none}
.ge-container{max-width:1200px;margin:0 auto;padding:0 24px}
.ge-secao{padding:80px 24px}
.ge-secao-inner{max-width:1200px;margin:0 auto}
.ge-titulo{font-weight:800;margin:0 0 16px;letter-spacing:-0.02em}
.ge-titulo-grande{font-size:clamp(2.2rem,5vw,3.6rem)}
.ge-titulo-media{font-size:clamp(1.5rem,3vw,2.2rem)}
.ge-titulo-pequena{font-size:1.25rem}
.ge-destaque{background:linear-gradient(90deg,%s,%s);-webkit-background-clip:text;background-clip:text;color:transparent;display:inline-block;font-weight:800}
.ge-texto{opacity:.85;font-size:1.05rem;max-width:640px}
.ge-selo{display:inline-flex;align-items:center;gap:6px;padding:6px 14px;border-radius:999px;background:%s;border:1px solid rgba(255,255,255,.12);font-size:.85rem;font-weight:600;margin-bottom:16px}
.ge-botao{display:inline-flex;align-items:center;justify-content:center;gap:8px;padding:13px 26px;border-radius:%s;font-weight:700;font-size:.95rem;border:1px solid transparent;transition:transform .15s,filter .15s;cursor:pointer}
.ge-botao:hover{filter:brightness(1.08)}
.ge-botao-contorno{background:transparent;border-color:currentColor}
.ge-botao-fantasma{background:transparent;opacity:.85}
.ge-icone{width:24px;height:24px;display:inline-flex}
.ge-navbar{position:sticky;top:0;z-index:50;display:flex;align-items:center;justify-content:space-between;padding:16px 24px;background:%s;border-bottom:1px solid rgba(255,255,255,.08)}
.ge-nav-logo{display:flex;align-items:center;gap:10px;font-weight:800;font-size:1.1rem}
.ge-nav-logo svg{width:28px;height:28px}
.ge-nav-links{display:flex;align-items:center;gap:28px}
.ge-nav-link{opacity:.85;font-weight:500}
.ge-nav-link:hover{opacity:1}
.ge-nav-check{display:none}
.ge-nav-burger{display:none;cursor:pointer;font-size:1.5rem;line-height:1}
.ge-grade{display:grid;gap:24px;margin-top:40px}
.ge-item{padding:28px;border-radius:%s;background:%s;border:1px solid rgba(255,255,255,.08)}
.ge-item .ge-icone{margin-bottom:14px;color:%s}
.ge-item h3,.ge-item .ge-titulo{font-size:1.1rem;margin-bottom:8px}
.ge-item p,.ge-item .ge-texto{font-size:.92rem;opacity:.75}
.ge-cartao{border-radius:%s;background:%s;border:1px solid rgba(255,255,255,.08);padding:20px;overflow:hidden}
.ge-cartao-destacado{border-color:%s;box-shadow:0 0 0 1px %s}
.ge-bloco-codigo{margin:0;padding:20px;border-radius:10px;background:rgba(0,0,0,.35);font-family:'Fira Code',monospace;font-size:.85rem;overflow-x:auto;white-space:pre}
.tok-kw{color:%s;font-weight:600}
.tok-str{color:#a3e635}
.tok-com{color:#6b7280;font-style:italic}
.tok-num{color:%s}
.ge-abas{margin-top:32px}
.ge-abas-nav{display:flex;gap:8px;margin-bottom:0;flex-wrap:wrap}
.ge-aba-radio{display:none}
.ge-aba-label{padding:10px 20px;border-radius:10px 10px 0 0;background:rgba(255,255,255,.05);cursor:pointer;font-weight:600;font-size:.9rem;opacity:.6}
.ge-aba-painel{display:none;padding:24px;border-radius:0 10px 10px 10px;background:rgba(255,255,255,.03);border:1px solid rgba(255,255,255,.08)}
.ge-cadeia-passos{display:flex;flex-wrap:wrap;gap:0;margin-top:40px}
.ge-passo{flex:1;min-width:160px;text-align:center;padding:20px;position:relative}
.ge-passo-num{width:36px;height:36px;border-radius:50%%;background:linear-gradient(135deg,%s,%s);display:flex;align-items:center;justify-content:center;font-weight:800;margin:0 auto 12px}
.ge-diagrama{display:flex;align-items:center;gap:24px;margin-top:40px;flex-wrap:wrap;justify-content:center}
.ge-lado{flex:1;min-width:220px;padding:24px;border-radius:%s;background:%s;border:1px solid rgba(255,255,255,.08)}
.ge-centro{font-size:1.8rem;opacity:.5}
.ge-imagem{border-radius:%s;background:rgba(255,255,255,.04);border:1px dashed rgba(255,255,255,.18);display:flex;flex-direction:column;align-items:center;justify-content:center;padding:48px 24px;text-align:center;gap:8px;color:inherit;opacity:.8}
.ge-links{display:flex;gap:20px;flex-wrap:wrap}
.ge-rodape{padding:48px 24px;border-top:1px solid rgba(255,255,255,.08);opacity:.75;font-size:.9rem}
`,
		font, t.Background, t.TextColor,
		t.Primary, t.Accent,
		t.CardBg,
		radius,
		t.Sidebar,
		radius, t.CardBg, t.Primary,
		radius, t.CardBg,
		t.Primary, t.Primary,
		t.Accent,
		t.Accent,
		t.Primary, t.Secondary,
		radius, t.CardBg,
		radius,
	)
	// Sensible default columns for .ge-grade when an explicit inline
	// style isn't wide enough (small screens): always fall back to 1 col.
	r.css.WriteString(`
@media (max-width:640px){.ge-grade{grid-template-columns:1fr!important}.ge-diagrama{flex-direction:column}.ge-cadeia-passos{flex-direction:column}}
`)
	if t.CustomCSS != "" {
		r.css.WriteString("\n" + t.CustomCSS + "\n")
	}
}

// ==================== dispatch ====================

// classFor allocates a synthetic "ge-el-N" class for elements that carry a
// hover or responsive rule, and emits the CSS for those rules against it.
// Elements with neither get no synthetic class (keeps generated markup and
// CSS minimal).
func (r *pageRenderer) classFor(el *ast.PageElement) string {
	if el.Hover == nil && len(el.Responsive) == 0 {
		return ""
	}
	r.elCounter++
	cls := fmt.Sprintf("ge-el-%d", r.elCounter)
	if el.Hover != nil {
		r.emitHoverCSS(cls, el.Hover)
	}
	for _, resp := range el.Responsive {
		r.emitResponsiveCSS(cls, resp, el.Kind == ast.PageElNavbar)
	}
	return cls
}

func (r *pageRenderer) emitHoverCSS(cls string, hv *ast.HoverRule) {
	scale := hv.Scale
	if scale == "" {
		scale = "1.03"
	}
	dur := hv.Duration
	if dur == "" {
		dur = "200ms"
	}
	if hv.Scope == "itens" {
		fmt.Fprintf(&r.css, ".%s>*{transition:transform %s}.%s>*:hover{transform:scale(%s)}\n", cls, dur, cls, scale)
	} else {
		fmt.Fprintf(&r.css, ".%s{transition:transform %s}.%s:hover{transform:scale(%s)}\n", cls, dur, cls, scale)
	}
}

func (r *pageRenderer) emitResponsiveCSS(cls string, resp *ast.ResponsiveRule, isNavbar bool) {
	var b strings.Builder
	for _, h := range resp.Hide {
		switch h {
		case "menu":
			if isNavbar {
				fmt.Fprintf(&b, ".%s .ge-nav-links{display:none}\n", cls)
			}
		case "":
			fmt.Fprintf(&b, ".%s{display:none}\n", cls)
		}
	}
	for _, s := range resp.Show {
		switch s {
		case "menu hamburguer", "menu":
			if isNavbar {
				fmt.Fprintf(&b, ".%s .ge-nav-burger{display:flex}.%s .ge-nav-check:checked~.ge-nav-links{display:flex;position:absolute;top:100%%;left:0;right:0;flex-direction:column;background:inherit;padding:16px 24px;border-bottom:1px solid rgba(255,255,255,.08)}\n", cls, cls)
			}
		case "":
			fmt.Fprintf(&b, ".%s{display:block}\n", cls)
		}
	}
	if b.Len() > 0 {
		fmt.Fprintf(&r.css, "@media (max-width:%s){\n%s}\n", resp.MaxWidth, b.String())
	}
}

func esc(s string) string { return html.EscapeString(s) }

func classAttr(base, extra string) string {
	if extra == "" {
		return base
	}
	return base + " " + extra
}

func (r *pageRenderer) renderChildren(children []*ast.PageElement) string {
	var b strings.Builder
	for _, c := range children {
		b.WriteString(r.renderElement(c))
	}
	return b.String()
}

func (r *pageRenderer) renderElement(el *ast.PageElement) string {
	cls := r.classFor(el)
	switch el.Kind {
	case ast.PageElNavbar:
		return r.renderNavbar(el, cls)
	case ast.PageElSection:
		return r.renderSection(el, cls)
	case ast.PageElGrid:
		return r.renderGrid(el, cls)
	case ast.PageElItem:
		return r.renderItem(el, cls)
	case ast.PageElCard:
		return r.renderCard(el, cls)
	case ast.PageElTabs:
		return r.renderTabs(el, cls)
	case ast.PageElSteps:
		return r.renderSteps(el, cls)
	case ast.PageElUnify:
		return r.renderUnify(el, cls)
	case ast.PageElLinks:
		return fmt.Sprintf(`<div class="%s">%s</div>`, classAttr("ge-links", cls), r.renderChildren(el.Children))
	case ast.PageElLink:
		return r.renderLink(el, cls)
	case ast.PageElFooter:
		return fmt.Sprintf(`<footer class="%s">%s</footer>`, classAttr("ge-rodape", cls), r.renderChildren(el.Children))
	case ast.PageElTitle:
		return r.renderTitle(el, cls)
	case ast.PageElHighlight:
		return fmt.Sprintf(`<span class="%s">%s</span>`, classAttr("ge-destaque", cls), esc(el.Text))
	case ast.PageElText:
		return fmt.Sprintf(`<p class="%s">%s</p>`, classAttr("ge-texto", cls), esc(el.Text))
	case ast.PageElBadge:
		return fmt.Sprintf(`<span class="%s">%s</span>`, classAttr("ge-selo", cls), esc(el.Text))
	case ast.PageElButton:
		return r.renderButton(el, cls)
	case ast.PageElIcon:
		return fmt.Sprintf(`<span class="%s">%s</span>`, classAttr("ge-icone", cls), svgIcon(el.Text))
	case ast.PageElImage:
		return fmt.Sprintf(`<div class="%s">%s<span>%s</span></div>`, classAttr("ge-imagem", cls), svgIcon("layout"), esc(el.Text))
	case ast.PageElLogo:
		return r.renderLogo(el, cls)
	case ast.PageElCode:
		return r.renderCodeBlock(el, cls)
	case ast.PageElCenter:
		return fmt.Sprintf(`<div class="%s">%s</div>`, classAttr("ge-centro", cls), r.renderChildren(el.Children))
	default:
		// item/aba/passo/lado are handled by their parent containers
		// (grade/abas/cadeia_passos/diagrama_unificacao) directly, so they
		// never reach here in a well-formed tree; render children as a
		// harmless fallback if they do.
		return r.renderChildren(el.Children)
	}
}

// ==================== containers ====================

func (r *pageRenderer) renderNavbar(el *ast.PageElement, cls string) string {
	r.navCounter++
	checkID := fmt.Sprintf("ge-nav-%d", r.navCounter)

	var logo, links strings.Builder
	for _, c := range el.Children {
		switch c.Kind {
		case ast.PageElLogo:
			logo.WriteString(r.renderElement(c))
		case ast.PageElLinks:
			links.WriteString(r.renderChildren(c.Children))
		case ast.PageElLink, ast.PageElButton:
			links.WriteString(r.renderElement(c))
		default:
			links.WriteString(r.renderElement(c))
		}
	}

	return fmt.Sprintf(`<nav class="%s">
<input type="checkbox" id="%s" class="ge-nav-check">
%s
<div class="ge-nav-links">%s</div>
<label for="%s" class="ge-nav-burger">%s</label>
</nav>`,
		classAttr("ge-navbar", cls), checkID, logo.String(), links.String(), checkID, svgIcon("menu"))
}

func (r *pageRenderer) renderSection(el *ast.PageElement, cls string) string {
	style := ""
	if fundo, ok := el.Properties["fundo"]; ok {
		style += "background:" + fundo + ";"
	}
	if align, ok := el.Properties["alinhamento"]; ok {
		switch align {
		case "centro", "center":
			style += "text-align:center;"
		case "direita", "right":
			style += "text-align:right;"
		}
	}
	styleAttr := ""
	if style != "" {
		styleAttr = ` style="` + style + `"`
	}
	return fmt.Sprintf(`<section class="%s"%s><div class="ge-secao-inner">%s</div></section>`,
		classAttr("ge-secao", cls), styleAttr, r.renderChildren(el.Children))
}

func (r *pageRenderer) renderGrid(el *ast.PageElement, cls string) string {
	cols := el.Properties["colunas"]
	n, err := strconv.Atoi(cols)
	if err != nil || n <= 0 {
		n = 3
	}
	style := fmt.Sprintf(` style="grid-template-columns:repeat(%d,1fr)"`, n)
	return fmt.Sprintf(`<div class="%s"%s>%s</div>`, classAttr("ge-grade", cls), style, r.renderChildren(el.Children))
}

func (r *pageRenderer) renderItem(el *ast.PageElement, cls string) string {
	return fmt.Sprintf(`<div class="%s">%s</div>`, classAttr("ge-item", cls), r.renderChildren(el.Children))
}

func (r *pageRenderer) renderCard(el *ast.PageElement, cls string) string {
	base := "ge-cartao"
	if el.Properties["estilo"] == "destacado" {
		base += " ge-cartao-destacado"
	}
	return fmt.Sprintf(`<div class="%s">%s</div>`, classAttr(base, cls), r.renderChildren(el.Children))
}

func (r *pageRenderer) renderTabs(el *ast.PageElement, cls string) string {
	r.tabCounter++
	groupName := fmt.Sprintf("ge-abas-%d", r.tabCounter)
	bodyClass := fmt.Sprintf("ge-abas-body-%d", r.tabCounter)

	// The radios must be direct children of .ge-abas — siblings of both
	// the label row and the panel body — so the "~" general-sibling
	// combinator below can reach across from a checked radio into the
	// (unrelated, later) panel body. Nesting the radios inside the label
	// row would put them in a different subtree and break that link.
	var radios, labels, panels strings.Builder
	tabIndex := 0
	for _, c := range el.Children {
		if c.Kind != ast.PageElTab {
			continue
		}
		id := fmt.Sprintf("%s-tab-%d", groupName, tabIndex)
		checked := ""
		if tabIndex == 0 {
			checked = " checked"
		}
		fmt.Fprintf(&radios, `<input type="radio" name="%s" id="%s" class="ge-aba-radio"%s>`, groupName, id, checked)
		fmt.Fprintf(&labels, `<label for="%s" class="ge-aba-label">%s</label>`, id, esc(c.Text))
		panels.WriteString(`<div class="ge-aba-painel">` + r.renderChildren(c.Children) + `</div>`)

		fmt.Fprintf(&r.css, "#%s:checked~.%s .ge-aba-painel:nth-of-type(%d){display:block}\n",
			id, bodyClass, tabIndex+1)
		fmt.Fprintf(&r.css, "#%s:checked~.ge-abas-nav-%d label[for=%s]{background:%s;color:#fff;opacity:1}\n",
			id, r.tabCounter, id, r.theme.Primary)
		tabIndex++
	}

	navClass := fmt.Sprintf("ge-abas-nav ge-abas-nav-%d", r.tabCounter)
	return fmt.Sprintf(`<div class="%s">%s<div class="%s">%s</div><div class="%s">%s</div></div>`,
		classAttr("ge-abas", cls), radios.String(), navClass, labels.String(), bodyClass, panels.String())
}

func (r *pageRenderer) renderSteps(el *ast.PageElement, cls string) string {
	var b strings.Builder
	n := 0
	for _, c := range el.Children {
		if c.Kind != ast.PageElStep {
			continue
		}
		n++
		fmt.Fprintf(&b, `<div class="ge-passo"><div class="ge-passo-num">%d</div>%s</div>`, n, r.renderChildren(c.Children))
	}
	return fmt.Sprintf(`<div class="%s">%s</div>`, classAttr("ge-cadeia-passos", cls), b.String())
}

func (r *pageRenderer) renderUnify(el *ast.PageElement, cls string) string {
	var b strings.Builder
	for _, c := range el.Children {
		switch c.Kind {
		case ast.PageElSide:
			b.WriteString(fmt.Sprintf(`<div class="ge-lado"><h4>%s</h4>%s</div>`, esc(c.Text), r.renderChildren(c.Children)))
		default:
			b.WriteString(r.renderElement(c))
		}
	}
	return fmt.Sprintf(`<div class="%s">%s</div>`, classAttr("ge-diagrama", cls), b.String())
}

// ==================== leaves ====================

func (r *pageRenderer) renderTitle(el *ast.PageElement, cls string) string {
	tag := "h2"
	sizeClass := "ge-titulo-media"
	switch el.Properties["tamanho"] {
	case "grande":
		tag, sizeClass = "h1", "ge-titulo-grande"
	case "pequena":
		tag, sizeClass = "h3", "ge-titulo-pequena"
	}
	return fmt.Sprintf(`<%s class="%s">%s</%s>`, tag, classAttr("ge-titulo "+sizeClass, cls), esc(el.Text), tag)
}

func (r *pageRenderer) renderButton(el *ast.PageElement, cls string) string {
	base := "ge-botao"
	style := ""
	switch el.Properties["estilo"] {
	case "contorno":
		base += " ge-botao-contorno"
		if c, ok := el.Properties["cor"]; ok {
			style = fmt.Sprintf(` style="color:%s;border-color:%s"`, c, c)
		}
	case "fantasma":
		base += " ge-botao-fantasma"
	default:
		c := el.Properties["cor"]
		if c == "" {
			c = r.theme.Primary
		}
		style = fmt.Sprintf(` style="background:%s;color:#fff"`, c)
	}
	href := el.Properties["para"]
	if href == "" {
		href = "#"
	}
	return fmt.Sprintf(`<a href="%s" class="%s"%s>%s</a>`, esc(href), classAttr(base, cls), style, esc(el.Text))
}

func (r *pageRenderer) renderLink(el *ast.PageElement, cls string) string {
	style := ""
	if c, ok := el.Properties["cor"]; ok {
		style = fmt.Sprintf(` style="color:%s"`, c)
	}
	href := el.Properties["para"]
	if href == "" {
		href = "#"
	}
	return fmt.Sprintf(`<a href="%s" class="%s"%s>%s</a>`, esc(href), classAttr("ge-nav-link", cls), style, esc(el.Text))
}

func (r *pageRenderer) renderLogo(el *ast.PageElement, cls string) string {
	gem := `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M6 3h12l4 6-10 12L2 9z"/><path d="M11 3 8 9l4 12 4-12-3-6"/><path d="M2 9h20"/></svg>`
	return fmt.Sprintf(`<div class="%s">%s<span>%s</span></div>`, classAttr("ge-nav-logo", cls), gem, esc(el.Text))
}

func (r *pageRenderer) renderCodeBlock(el *ast.PageElement, cls string) string {
	return fmt.Sprintf(`<pre class="%s"><code>%s</code></pre>`, classAttr("ge-bloco-codigo", cls), highlightCode(el.Text))
}

// ==================== deterministic fake syntax highlighting ====================

// codeKeywords is intentionally small and generic: Germanio's own
// vocabulary plus a handful of common words that show up in the example
// snippets landing pages tend to display. This is cosmetic highlighting,
// not a real lexer — good enough for a <pre><code> demo block, not meant
// to be a language server.
var codeKeywords = map[string]bool{
	"app": true, "sistema": true, "dados": true, "telas": true, "tela": true,
	"acoes": true, "ações": true, "eventos": true, "tema": true, "banco": true,
	"modelo": true, "modelos": true, "campo": true, "funcao": true, "função": true,
	"retorne": true, "retornar": true, "se": true, "senao": true, "senão": true,
	"para": true, "cada": true, "em": true, "enquanto": true, "definir": true,
	"variavel": true, "variável": true, "mostre": true, "texto": true, "numero": true,
	"número": true, "booleano": true, "verdadeiro": true, "falso": true,
	"import": true, "const": true, "let": true, "function": true, "return": true,
	"if": true, "else": true, "for": true, "while": true, "class": true, "def": true,
	"true": true, "false": true, "null": true, "nulo": true,
}

var codeTokenRe = regexp.MustCompile(`(?s)//[^\n]*|#[^\n]*|"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|` + "`" + `[^` + "`" + `]*` + "`" + `|\b\d+(?:\.\d+)?\b|[A-Za-zÀ-ÿ_][A-Za-zÀ-ÿ0-9_]*`)

func highlightCode(src string) string {
	var b strings.Builder
	last := 0
	for _, m := range codeTokenRe.FindAllStringIndex(src, -1) {
		start, end := m[0], m[1]
		b.WriteString(esc(src[last:start]))
		tok := src[start:end]
		switch {
		case strings.HasPrefix(tok, "//") || strings.HasPrefix(tok, "#"):
			b.WriteString(`<span class="tok-com">` + esc(tok) + `</span>`)
		case strings.HasPrefix(tok, `"`) || strings.HasPrefix(tok, "'") || strings.HasPrefix(tok, "`"):
			b.WriteString(`<span class="tok-str">` + esc(tok) + `</span>`)
		case tok[0] >= '0' && tok[0] <= '9':
			b.WriteString(`<span class="tok-num">` + esc(tok) + `</span>`)
		case codeKeywords[strings.ToLower(tok)]:
			b.WriteString(`<span class="tok-kw">` + esc(tok) + `</span>`)
		default:
			b.WriteString(esc(tok))
		}
		last = end
	}
	b.WriteString(esc(src[last:]))
	return b.String()
}
