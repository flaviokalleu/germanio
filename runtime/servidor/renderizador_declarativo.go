package servidor

import (
	"fmt"
	"html"
	"strings"
	"unicode"

	"github.com/flaviokalleu/germanio/compiler/ast"
)

// RenderDeclarativePage compiles a pure Germanio AST page into modern, self-contained HTML/CSS
func RenderDeclarativePage(page *ast.CustomPage) string {
	return renderDeclarativePage(page, pageMeta{Title: page.Title})
}

// renderDeclarativePage renders the page with its meta tags (seo.go). The
// server renders each page once per program load.
func renderDeclarativePage(page *ast.CustomPage, meta pageMeta) string {
	var b strings.Builder

	title := page.Title
	if title == "" {
		title = meta.Title
	}

	b.WriteString(`<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>` + html.EscapeString(title) + `</title>
` + metaHTML(meta) + `  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500;600&display=swap" rel="stylesheet">
  <style>
    *, *::before, *::after {
      box-sizing: border-box;
      margin: 0;
      padding: 0;
    }
    :root {
      --bg-main: #05070A;
      --bg-surface: #080B10;
      --bg-card: #0D1520;
      --bg-card-hover: #121D2C;
      --border-subtle: rgba(255, 255, 255, 0.08);
      --border-card: rgba(255, 255, 255, 0.12);
      --border-hover: rgba(0, 217, 255, 0.5);
      --text-primary: #F4F8FC;
      --text-secondary: #94A3B8;
      --text-muted: #64748B;
      --accent-cyan: #00D9FF;
      --accent-blue: #168BFF;
      --accent-emerald: #00E58B;
      --font-sans: 'Plus Jakarta Sans', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      --font-mono: 'JetBrains Mono', ui-monospace, SFMono-Regular, Menlo, Monaco, monospace;
    }
    html, body {
      background-color: var(--bg-main) !important;
      background-image: radial-gradient(rgba(255, 255, 255, 0.04) 1px, transparent 1px);
      background-size: 28px 28px;
      color: var(--text-primary) !important;
      font-family: var(--font-sans);
      line-height: 1.6;
      min-height: 100vh;
      display: flex;
      flex-direction: column;
      -webkit-font-smoothing: antialiased;
    }
    a {
      color: inherit;
      text-decoration: none;
    }
    .code-font {
      font-family: var(--font-mono);
    }
    .container {
      width: 100%;
      max-width: 1240px;
      margin: 0 auto;
      padding: 0 24px;
    }
    /* Header / Navbar */
    .header {
      position: sticky;
      top: 0;
      z-index: 50;
      background: rgba(8, 11, 16, 0.92);
      backdrop-filter: blur(16px);
      -webkit-backdrop-filter: blur(16px);
      border-bottom: 1px solid var(--border-subtle);
    }
    .nav-wrapper {
      display: flex;
      align-items: center;
      justify-content: space-between;
      height: 72px;
    }
    .brand {
      display: flex;
      align-items: center;
      gap: 12px;
      font-size: 1.25rem;
      font-weight: 800;
      letter-spacing: -0.02em;
      color: var(--text-primary);
    }
    .brand-logo {
      width: 32px;
      height: 32px;
      object-fit: contain;
      filter: drop-shadow(0 0 10px rgba(0, 217, 255, 0.5));
    }
    .nav-links {
      display: flex;
      align-items: center;
      gap: 24px;
      font-size: 0.925rem;
      font-weight: 600;
      color: var(--text-secondary);
    }
    .nav-links a:hover {
      color: var(--accent-cyan);
      transition: color 0.15s ease;
    }
    .btn-install {
      background: var(--accent-cyan);
      color: #03101C !important;
      font-weight: 700;
      padding: 9px 20px;
      border-radius: 9px;
      font-size: 0.875rem;
      transition: all 0.2s ease;
      box-shadow: 0 0 20px rgba(0, 217, 255, 0.25);
    }
    .btn-install:hover {
      background: #38bdf8;
      transform: translateY(-1px);
      box-shadow: 0 4px 25px rgba(0, 217, 255, 0.45);
    }
    /* Hero */
    .hero-section {
      padding: 70px 0 90px 0;
      position: relative;
      background: radial-gradient(ellipse 80% 60% at 50% -10%, rgba(0, 217, 255, 0.12) 0%, rgba(22, 139, 255, 0.04) 50%, transparent 100%);
    }
    .hero-grid {
      display: grid;
      grid-template-columns: 1.15fr 0.85fr;
      gap: 48px;
      align-items: center;
    }
    @media (max-width: 960px) {
      .hero-grid {
        grid-template-columns: 1fr;
      }
      .nav-links {
        display: none;
      }
    }
    .badge {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      padding: 6px 14px;
      background: rgba(0, 217, 255, 0.1);
      border: 1px solid rgba(0, 217, 255, 0.3);
      border-radius: 9999px;
      font-size: 0.75rem;
      font-weight: 700;
      color: var(--accent-cyan);
      letter-spacing: 0.06em;
      margin-bottom: 24px;
      box-shadow: 0 0 15px rgba(0, 217, 255, 0.15);
    }
    .badge-dot {
      width: 8px;
      height: 8px;
      border-radius: 50%;
      background: var(--accent-cyan);
      box-shadow: 0 0 8px var(--accent-cyan);
    }
    .hero-title {
      font-size: 3.5rem;
      font-weight: 800;
      line-height: 1.12;
      letter-spacing: -0.035em;
      margin-bottom: 20px;
      color: #FFFFFF;
    }
    .gradient-text {
      color: var(--accent-cyan);
      background: linear-gradient(135deg, #00D9FF 0%, #38BDF8 50%, #168BFF 100%);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
      background-clip: text;
      display: block;
      margin-top: 4px;
    }
    .hero-desc {
      font-size: 1.15rem;
      color: var(--text-secondary);
      line-height: 1.7;
      margin-bottom: 36px;
      max-width: 580px;
    }
    .hero-actions {
      display: flex;
      flex-wrap: wrap;
      align-items: center;
      gap: 16px;
    }
    .btn-primary {
      background: var(--accent-cyan);
      color: #03101C !important;
      font-weight: 800;
      padding: 15px 30px;
      border-radius: 12px;
      font-size: 1rem;
      transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
      box-shadow: 0 4px 25px rgba(0, 217, 255, 0.3);
      display: inline-flex;
      align-items: center;
      gap: 8px;
    }
    .btn-primary:hover {
      background: #38bdf8;
      transform: translateY(-2px);
      box-shadow: 0 8px 35px rgba(0, 217, 255, 0.45);
    }
    .btn-secondary {
      background: var(--bg-card);
      color: var(--text-primary);
      border: 1px solid var(--border-card);
      font-weight: 600;
      padding: 15px 26px;
      border-radius: 12px;
      font-size: 1rem;
      transition: all 0.2s ease;
    }
    .btn-secondary:hover {
      border-color: var(--border-hover);
      background: var(--bg-card-hover);
    }
    .cli-box {
      display: flex;
      align-items: center;
      gap: 10px;
      background: #080B10;
      border: 1px solid var(--border-card);
      border-radius: 12px;
      padding: 13px 20px;
      font-size: 0.875rem;
      color: var(--text-secondary);
      width: 100%;
      margin-top: 8px;
    }
    .cli-box code {
      color: var(--accent-cyan);
      font-family: var(--font-mono);
      font-size: 0.85rem;
    }
    /* Code Editor Card */
    .editor-card {
      background: #080B10;
      border: 1px solid rgba(0, 217, 255, 0.25);
      border-radius: 18px;
      overflow: hidden;
      box-shadow: 0 25px 60px rgba(0, 0, 0, 0.8), 0 0 30px rgba(0, 217, 255, 0.1);
    }
    .editor-header {
      background: #0D1520;
      padding: 14px 18px;
      border-bottom: 1px solid var(--border-subtle);
      display: flex;
      align-items: center;
      justify-content: space-between;
    }
    .editor-dots {
      display: flex;
      gap: 7px;
    }
    .dot {
      width: 11px;
      height: 11px;
      border-radius: 50%;
    }
    .dot-red { background: #ff5f56; }
    .dot-yellow { background: #ffbd2e; }
    .dot-green { background: #27c93f; }
    .editor-filename {
      font-size: 0.85rem;
      color: var(--text-secondary);
      font-family: var(--font-mono);
      font-weight: 600;
      margin-left: 12px;
    }
    .editor-badge {
      font-size: 0.725rem;
      font-weight: 700;
      background: rgba(0, 217, 255, 0.12);
      color: var(--accent-cyan);
      border: 1px solid rgba(0, 217, 255, 0.3);
      padding: 3px 8px;
      border-radius: 5px;
      font-family: var(--font-mono);
    }
    .editor-code {
      padding: 24px;
      font-family: var(--font-mono);
      font-size: 0.9rem;
      line-height: 1.75;
      color: #E2E8F0;
      overflow-x: auto;
      background: #05070A;
    }
    .kw { color: #00D9FF; font-weight: 600; }
    .str { color: #00E58B; }
    .tp { color: #38BDF8; }
    .fn { color: #F59E0B; }
    .cm { color: #64748B; font-style: italic; }

    /* Section & Cards */
    .section {
      padding: 80px 0;
      border-top: 1px solid var(--border-subtle);
      background: rgba(8, 11, 16, 0.4);
    }
    .section-header {
      max-width: 680px;
      margin-bottom: 44px;
    }
    .section-title {
      font-size: 2.25rem;
      font-weight: 800;
      letter-spacing: -0.025em;
      margin-bottom: 12px;
      color: #FFFFFF;
    }
    .section-sub {
      color: var(--text-secondary);
      font-size: 1.1rem;
    }
    .grid-3 {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
      gap: 28px;
    }
    .grid-2 {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(420px, 1fr));
      gap: 28px;
    }
    .card {
      background: var(--bg-card);
      border: 1px solid var(--border-card);
      border-radius: 18px;
      padding: 30px;
      display: flex;
      flex-direction: column;
      justify-content: space-between;
      transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
      box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
    }
    .card:hover {
      border-color: var(--border-hover);
      background: var(--bg-card-hover);
      transform: translateY(-3px);
      box-shadow: 0 16px 40px rgba(0, 217, 255, 0.1);
    }
    .card-top {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 18px;
    }
    .card-icon {
      font-size: 1.8rem;
    }
    .card-tag {
      font-size: 0.725rem;
      font-weight: 800;
      font-family: var(--font-mono);
      background: rgba(0, 217, 255, 0.1);
      color: var(--accent-cyan);
      border: 1px solid rgba(0, 217, 255, 0.25);
      padding: 3px 10px;
      border-radius: 5px;
    }
    .card-title {
      font-size: 1.35rem;
      font-weight: 700;
      margin-bottom: 10px;
      letter-spacing: -0.015em;
      color: #FFFFFF;
    }
    .card-text {
      font-size: 0.95rem;
      color: var(--text-secondary);
      line-height: 1.65;
      margin-bottom: 18px;
    }
    .card-code {
      background: #05070A;
      border: 1px solid var(--border-subtle);
      border-radius: 12px;
      padding: 16px;
      font-family: var(--font-mono);
      font-size: 0.825rem;
      color: #A5B4FC;
      overflow-x: auto;
      margin-bottom: 18px;
      line-height: 1.65;
    }
    .card-link {
      color: var(--accent-cyan);
      font-size: 0.875rem;
      font-weight: 700;
      display: inline-flex;
      align-items: center;
      gap: 5px;
    }
    .card-link:hover {
      text-decoration: underline;
    }
    /* Footer */
    .footer {
      margin-top: auto;
      border-top: 1px solid var(--border-subtle);
      background: #080B10;
      padding: 48px 0;
    }
    .footer-wrapper {
      display: flex;
      flex-wrap: wrap;
      align-items: center;
      justify-content: space-between;
      gap: 24px;
      font-size: 0.9rem;
      color: var(--text-muted);
    }
    .footer-left {
      display: flex;
      align-items: center;
      gap: 12px;
    }
    .footer-links {
      display: flex;
      gap: 28px;
      font-weight: 500;
    }
    .footer-links a:hover {
      color: var(--text-primary);
    }
  </style>
</head>
<body>
`)

	for _, block := range page.Blocks {
		switch bl := block.(type) {
		case *ast.PageNavbar:
			renderNavbar(&b, bl)
		case *ast.PageHero:
			renderHero(&b, bl)
		case *ast.PageSection:
			renderSection(&b, bl)
		case *ast.PageFooter:
			renderFooter(&b, bl)
		}
	}

	b.WriteString(`
</body>
</html>`)

	return b.String()
}

// codeKeywords are highlighted in code previews, as whole words only.
var codeKeywords = map[string]bool{
	"crie": true, "sistema": true, "tenha": true, "tem": true, "acesso": true, "pode": true,
	"permita": true, "página": true, "mostre": true, "pertence": true, "começa": true,
	"importar": true, "login": true, "usa": true, "todos": true, "somente": true, "regras": true,
	"integração": true, "antes": true, "quando": true, "se": true, "recuse": true,
	"app": true, "banco": true, "tabela": true, "pagina": true, "navbar": true, "hero": true,
	"secao": true, "card": true, "rodape": true,
}

func highlightGermanioCode(code string) string {
	var b strings.Builder
	runes := []rune(code)
	for i := 0; i < len(runes); {
		if !unicode.IsLetter(runes[i]) {
			b.WriteString(html.EscapeString(string(runes[i])))
			i++
			continue
		}
		j := i
		for j < len(runes) && (unicode.IsLetter(runes[j]) || runes[j] == '_') {
			j++
		}
		word := string(runes[i:j])
		if codeKeywords[word] {
			b.WriteString(`<span class="kw">` + html.EscapeString(word) + `</span>`)
		} else {
			b.WriteString(html.EscapeString(word))
		}
		i = j
	}
	return b.String()
}

func renderNavbar(b *strings.Builder, nav *ast.PageNavbar) {
	brand := nav.Brand
	if brand == "" {
		brand = "Germanio"
	}
	logo := nav.Logo
	if logo == "" {
		logo = "/assets/germanio.png"
	}
	b.WriteString(`
  <div class="header">
    <div class="container nav-wrapper">
      <a href="/" class="brand">`)
	b.WriteString(fmt.Sprintf(`<img src="%s" alt="%s" class="brand-logo" onerror="this.style.display='none'"><span>%s</span></a>`, html.EscapeString(logo), html.EscapeString(brand), html.EscapeString(brand)))
	b.WriteString(`<div class="nav-links">`)
	for _, link := range nav.Links {
		b.WriteString(fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(link.URL), html.EscapeString(link.Label)))
	}
	for _, button := range nav.Buttons {
		class := "btn-secondary"
		if button.Primary {
			class = "btn-install"
		}
		b.WriteString(fmt.Sprintf(`<a href="%s" class="%s">%s</a>`, html.EscapeString(button.URL), class, html.EscapeString(button.Label)))
	}
	b.WriteString(`</div></div></div>`)
}

func renderHero(b *strings.Builder, hero *ast.PageHero) {
	b.WriteString(`
  <div class="hero-section">
    <div class="container hero-grid">
      <div>`)

	if hero.Badge != "" {
		b.WriteString(fmt.Sprintf(`
        <div class="badge">
          <div class="badge-dot"></div> %s
        </div>`, html.EscapeString(hero.Badge)))
	}

	if hero.Title != "" {
		b.WriteString(fmt.Sprintf(`
        <h1 class="hero-title">%s`, html.EscapeString(hero.Title)))
		if hero.Highlight != "" {
			b.WriteString(fmt.Sprintf(`
          <span class="gradient-text">%s</span>`, html.EscapeString(hero.Highlight)))
		}
		b.WriteString(`
        </h1>`)
	}

	if hero.Description != "" {
		b.WriteString(fmt.Sprintf(`
        <p class="hero-desc">%s</p>`, html.EscapeString(hero.Description)))
	}

	b.WriteString(`
        <div class="hero-actions">`)
	for _, btn := range hero.Buttons {
		if btn.Primary {
			b.WriteString(fmt.Sprintf(`
          <a href="%s" class="btn-primary">%s</a>`, html.EscapeString(btn.URL), html.EscapeString(btn.Label)))
		} else {
			b.WriteString(fmt.Sprintf(`
          <a href="%s" class="btn-secondary">%s</a>`, html.EscapeString(btn.URL), html.EscapeString(btn.Label)))
		}
	}

	if hero.Command != "" {
		b.WriteString(fmt.Sprintf(`
          <div class="cli-box">
            <span style="color: var(--accent-cyan); font-weight: 800;">$</span>
            <code>%s</code>
          </div>`, html.EscapeString(hero.Command)))
	}
	b.WriteString(`
        </div>
      </div>

      <div>`)

	if hero.CodePreview != nil {
		filename := hero.CodePreview.Filename
		if filename == "" {
			filename = "app.ge"
		}
		highlighted := highlightGermanioCode(hero.CodePreview.Code)
		b.WriteString(fmt.Sprintf(`
        <div class="editor-card">
          <div class="editor-header">
            <div style="display: flex; align-items: center;">
              <div class="editor-dots">
                <div class="dot dot-red"></div>
                <div class="dot dot-yellow"></div>
                <div class="dot dot-green"></div>
              </div>
              <span class="editor-filename">%s</span>
            </div>
            <span class="editor-badge">.GE</span>
          </div>
          <pre class="editor-code"><code>%s</code></pre>
        </div>`, html.EscapeString(filename), highlighted))
	} else if hero.Mascot != "" {
		b.WriteString(fmt.Sprintf(`
        <div style="text-align: center;">
          <img src="%s" alt="Mascote" style="max-width: 320px; width: 100%%; filter: drop-shadow(0 0 30px rgba(0, 217, 255, 0.3));" onerror="this.style.display='none'">
        </div>`, html.EscapeString(hero.Mascot)))
	}
	b.WriteString(`
      </div>
    </div>
  </div>
`)
}

func renderSection(b *strings.Builder, sec *ast.PageSection) {
	b.WriteString(`
  <div class="section">
    <div class="container">`)

	if sec.Title != "" {
		b.WriteString(`
      <div class="section-header">`)
		b.WriteString(fmt.Sprintf(`
        <h2 class="section-title">%s</h2>`, html.EscapeString(sec.Title)))
		if sec.Subtitle != "" {
			b.WriteString(fmt.Sprintf(`
        <p class="section-sub">%s</p>`, html.EscapeString(sec.Subtitle)))
		}
		b.WriteString(`
      </div>`)
	}

	if len(sec.Cards) > 0 {
		gridClass := "grid-3"
		if sec.Columns == 2 {
			gridClass = "grid-2"
		}
		b.WriteString(fmt.Sprintf(`
      <div class="%s">`, gridClass))

		for _, card := range sec.Cards {
			b.WriteString(`
        <div class="card">
          <div>
            <div class="card-top">`)
			if card.Icon != "" {
				b.WriteString(fmt.Sprintf(`<span class="card-icon">%s</span>`, html.EscapeString(card.Icon)))
			} else {
				b.WriteString(`<span></span>`)
			}
			if card.Tag != "" {
				b.WriteString(fmt.Sprintf(`<span class="card-tag">%s</span>`, html.EscapeString(card.Tag)))
			}
			b.WriteString(`
            </div>`)

			if card.Title != "" {
				b.WriteString(fmt.Sprintf(`
            <h3 class="card-title">%s</h3>`, html.EscapeString(card.Title)))
			}

			if card.Text != "" {
				b.WriteString(fmt.Sprintf(`
            <p class="card-text">%s</p>`, html.EscapeString(card.Text)))
			}

			if card.Code != "" {
				highlighted := highlightGermanioCode(card.Code)
				b.WriteString(fmt.Sprintf(`
            <pre class="card-code"><code>%s</code></pre>`, highlighted))
			}
			b.WriteString(`
          </div>`)

			if card.Link != "" {
				b.WriteString(fmt.Sprintf(`
				  <a href="%s" class="card-link">Saiba mais →</a>`, html.EscapeString(card.Link)))
			}
			if card.ButtonURL != "" {
				class := "btn-secondary"
				if card.ButtonPrimary {
					class = "btn-primary"
				}
				b.WriteString(fmt.Sprintf(`
				  <a href="%s" class="%s">%s</a>`, html.EscapeString(card.ButtonURL), class, html.EscapeString(card.ButtonLabel)))
			}

			b.WriteString(`
        </div>`)
		}
		b.WriteString(`
      </div>`)
	}

	b.WriteString(`
    </div>
  </div>
`)
}

func renderFooter(b *strings.Builder, foot *ast.PageFooter) {
	b.WriteString(`
  <div class="footer">
    <div class="container footer-wrapper">
      <div class="footer-left">
        <img src="/assets/germanio.png" alt="Germanio" style="width: 22px; height: 22px; opacity: 0.8;" onerror="this.style.display='none'">
        <span>`)

	copyright := foot.Copyright
	if copyright == "" {
		copyright = "Germanio © 2026 — Desenvolvido por Flávio Kalleu e contribuidores."
	}
	b.WriteString(html.EscapeString(copyright))
	b.WriteString(`</span>
      </div>
      <div class="footer-links">`)

	for _, l := range foot.Links {
		b.WriteString(fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(l.URL), html.EscapeString(l.Label)))
	}

	b.WriteString(`
      </div>
    </div>
  </div>
`)
}
