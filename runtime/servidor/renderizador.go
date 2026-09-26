package servidor

import (
	"fmt"
	"html"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
)

// renderHTML generates the full single-page application HTML.
// Uses Tailwind CSS via CDN for a modern, clean design inspired by Flowise/Material UI.
func (s *Servidor) renderHTML() string {
	theme := s.Program.Theme
	if theme == nil {
		theme = ast.DefaultTheme()
	}
	applyThemeDefaults(theme)

	darkClass := ""
	if theme.Dark {
		darkClass = " dark"
	}

	var b strings.Builder

	// --- Head ---
	b.WriteString(`<!DOCTYPE html><html lang="pt-BR" class="` + darkClass + `"><head><meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1.0">
<title>` + html.EscapeString(cap(s.Program.System.Name)) + `</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link href="https://fonts.googleapis.com/css2?family=` + strings.ReplaceAll(theme.Font, " ", "+") + `:wght@300;400;500;600;700;800&display=swap" rel="stylesheet">
<script src="https://cdn.jsdelivr.net/npm/chart.js@4/dist/chart.umd.min.js"></script>
<script src="https://cdn.tailwindcss.com"></script>
<script>
tailwind.config = {
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        primary: '` + theme.Primary + `',
        secondary: '` + theme.Secondary + `',
        accent: '` + theme.Accent + `',
      },
      fontFamily: {
        sans: ['` + theme.Font + `', 'system-ui', '-apple-system', 'sans-serif'],
      }
    }
  }
}
</script>
<style>`)
	b.WriteString(s.generateCSS(theme))
	b.WriteString(`</style></head><body class="bg-gray-50 dark:bg-gray-950 text-gray-900 dark:text-gray-100 font-sans flex min-h-screen transition-colors duration-300">`)

	// --- Sidebar ---
	b.WriteString(s.renderSidebar(theme))

	// --- Main area ---
	b.WriteString(`<main class="flex-1 ml-64 min-h-screen transition-all duration-300" id="main">`)
	b.WriteString(s.renderTopbar())
	b.WriteString(`<div class="p-6" id="content">`)

	// Dashboard section
	b.WriteString(s.renderDashboard(theme))

	// Model sections - respect custom screens
	if len(s.Program.Screens) > 0 {
		s.renderCustomScreens(&b)
	}
	// Always generate model sections (custom screens are shown via nav)
	for _, model := range s.Program.Models {
		s.renderModelSection(&b, model, theme)
	}

	b.WriteString(`</div></main>`) // content, main

	// Modal forms - rendered OUTSIDE content/main so they're never hidden by display:none parents
	for _, model := range s.Program.Models {
		s.renderModelModal(&b, model)
	}

	// Toast container
	b.WriteString(`<div id="toast" class="fixed bottom-4 right-4 z-[10000] bg-gray-800 dark:bg-gray-800 border border-gray-700 text-white px-4 py-3 rounded-xl shadow-lg transform translate-y-20 opacity-0 transition-all duration-300 pointer-events-none"></div>`)

	// Auth modal (login/register)
	if s.Auth != nil {
		loginField := "email"
		passField := "senha"
		if s.Program.Auth != nil {
			if s.Program.Auth.LoginField != "" {
				loginField = s.Program.Auth.LoginField
			}
			if s.Program.Auth.PassField != "" {
				passField = s.Program.Auth.PassField
			}
		}
		b.WriteString(`<div id="auth-modal" class="fixed inset-0 bg-black/60 backdrop-blur-sm z-[9999] hidden items-center justify-center p-4">`)
		b.WriteString(`<div class="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-2xl w-full max-w-md shadow-2xl p-8">`)
		b.WriteString(`<h2 id="auth-title" class="text-xl font-bold text-center mb-6">Entrar</h2>`)
		b.WriteString(`<form id="auth-form" onsubmit="authSubmit(event)">`)
		b.WriteString(`<div id="auth-extra-fields"></div>`)
		b.WriteString(fmt.Sprintf(`<div class="mb-4"><label class="block text-sm font-medium text-gray-500 dark:text-gray-400 mb-1.5">%s</label><input type="text" id="auth-login" required class="w-full bg-gray-100 dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-xl px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary" placeholder="%s"></div>`, cap(loginField), loginField))
		b.WriteString(fmt.Sprintf(`<div class="mb-4"><label class="block text-sm font-medium text-gray-500 dark:text-gray-400 mb-1.5">%s</label><input type="password" id="auth-pass" required class="w-full bg-gray-100 dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-xl px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary" placeholder="%s" minlength="6"></div>`, cap(passField), passField))
		b.WriteString(`<div id="auth-error" class="text-red-500 text-sm my-2 hidden"></div>`)
		b.WriteString(`<button type="submit" class="w-full bg-primary hover:bg-primary/80 text-white py-2.5 rounded-xl font-medium transition-all mt-3">Entrar</button>`)
		b.WriteString(`<p class="text-center mt-4 text-sm text-gray-500">`)
		b.WriteString(`<span id="auth-toggle-text">Nao tem conta?</span> <a href="#" onclick="toggleAuthMode()" id="auth-toggle-link" class="text-primary hover:underline">Criar conta</a></p>`)
		b.WriteString(`<button type="button" onclick="fecharAuth()" class="w-full mt-2 px-6 py-2.5 rounded-xl border border-gray-300 dark:border-gray-700 text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 transition-all text-sm">Cancelar</button>`)
		b.WriteString(`</form></div></div>`)
	}

	// --- JavaScript ---
	b.WriteString(`<script>`)
	b.WriteString(s.generateJS(theme))
	b.WriteString(`</script></body></html>`)

	return b.String()
}

// applyThemeDefaults fills empty theme fields with sensible defaults.
func applyThemeDefaults(t *ast.Theme) {
	if t.Primary == "" {
		t.Primary = "#6366f1"
	}
	if t.Secondary == "" {
		t.Secondary = "#8b5cf6"
	}
	if t.Accent == "" {
		t.Accent = "#f59e0b"
	}
	if t.Sidebar == "" {
		t.Sidebar = "#1e1b4b"
	}
	if t.Font == "" {
		t.Font = "Inter"
	}
	if t.Radius == "" {
		t.Radius = "12px"
	}
	if t.Style == "" {
		t.Style = "glassmorphism"
	}
	if t.Background == "" {
		if t.Dark {
			t.Background = "#0c0a1d"
		} else {
			t.Background = "#f8fafc"
		}
	}
	if t.CardBg == "" {
		if t.Dark {
			t.CardBg = "rgba(30,27,75,0.6)"
		} else {
			t.CardBg = "rgba(255,255,255,0.85)"
		}
	}
	if t.TextColor == "" {
		if t.Dark {
			t.TextColor = "#e2e8f0"
		} else {
			t.TextColor = "#0f172a"
		}
	}
}

// ============================================================
// Sidebar
// ============================================================

func (s *Servidor) renderSidebar(theme *ast.Theme) string {
	var b strings.Builder
	b.WriteString(`<aside class="w-64 bg-white dark:bg-gray-900 border-r border-gray-200 dark:border-gray-800 flex flex-col fixed h-full z-50 transition-all duration-300" id="sidebar">`)

	// Brand
	b.WriteString(`<div class="p-5 border-b border-gray-200 dark:border-gray-800">`)
	b.WriteString(`<div class="flex items-center gap-3">`)
	if theme.Icon != "" {
		b.WriteString(`<div class="w-9 h-9 rounded-lg bg-primary/10 flex items-center justify-center flex-shrink-0"><img src="` + theme.Icon + `" alt="logo" class="w-6 h-6 object-contain rounded"></div>`)
	} else {
		b.WriteString(`<div class="w-9 h-9 rounded-lg bg-primary/10 flex items-center justify-center flex-shrink-0 text-primary">` + svgIcon("zap") + `</div>`)
	}
	b.WriteString(`<span class="font-bold text-lg tracking-tight truncate">` + html.EscapeString(cap(s.Program.System.Name)) + `</span>`)
	b.WriteString(`</div></div>`)

	// Nav
	b.WriteString(`<nav class="flex-1 p-2 space-y-1 overflow-y-auto">`)
	if len(s.Program.SidebarItems) > 0 {
		for _, item := range s.Program.SidebarItems {
			label := item.Label
			if label == "" {
				continue
			}
			icon := item.Icon
			if icon == "" {
				icon = "grid"
			}
			link := item.Link
			if link == "" {
				link = lo(label)
			}
			b.WriteString(fmt.Sprintf(`<a class="nav-item flex items-center gap-3 px-3 py-2.5 rounded-lg text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 hover:text-gray-900 dark:hover:text-white transition-all cursor-pointer text-sm font-medium" onclick="irPara('%s',this)" href="#">`, html.EscapeString(link)))
			b.WriteString(`<span class="w-5 h-5 flex-shrink-0">` + svgIcon(icon) + `</span><span>` + html.EscapeString(cap(label)) + `</span></a>`)
		}
	} else {
		// Dashboard link
		b.WriteString(`<a class="nav-item flex items-center gap-3 px-3 py-2.5 rounded-lg bg-primary/10 text-primary transition-all cursor-pointer text-sm font-medium" onclick="irPara('dashboard',this)" href="#">`)
		b.WriteString(`<span class="w-5 h-5 flex-shrink-0">` + svgIcon("layout") + `</span><span>Dashboard</span></a>`)

		if len(s.Program.Screens) > 0 {
			for _, scr := range s.Program.Screens {
				name := lo(scr.Name)
				title := scr.Title
				if title == "" {
					title = cap(scr.Name)
				}
				icon := "grid"
				if pm := s.primaryScreenModel(scr); pm != nil {
					icon = modelIcon(lo(pm.Name))
				}
				b.WriteString(fmt.Sprintf(`<a class="nav-item flex items-center gap-3 px-3 py-2.5 rounded-lg text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 hover:text-gray-900 dark:hover:text-white transition-all cursor-pointer text-sm font-medium" onclick="irPara('screen-%s',this)" href="#">`, name))
				b.WriteString(`<span class="w-5 h-5 flex-shrink-0">` + svgIcon(icon) + `</span><span>` + html.EscapeString(title) + `</span></a>`)
			}
			// Models without custom screen
			for _, model := range s.Program.Models {
				mName := lo(model.Name)
				hasScreen := false
				for _, scr := range s.Program.Screens {
					if pm := s.primaryScreenModel(scr); pm != nil && lo(pm.Name) == mName {
						hasScreen = true
						break
					}
				}
				if !hasScreen {
					icon := modelIcon(mName)
					b.WriteString(fmt.Sprintf(`<a class="nav-item flex items-center gap-3 px-3 py-2.5 rounded-lg text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 hover:text-gray-900 dark:hover:text-white transition-all cursor-pointer text-sm font-medium" onclick="irPara('%s',this)" href="#">`, mName))
					b.WriteString(`<span class="w-5 h-5 flex-shrink-0">` + svgIcon(icon) + `</span><span>` + html.EscapeString(cap(model.Name)) + `</span></a>`)
				}
			}
		} else {
			for _, model := range s.Program.Models {
				name := lo(model.Name)
				icon := modelIcon(name)
				if model.Icon != "" {
					icon = model.Icon
				}
				b.WriteString(fmt.Sprintf(`<a class="nav-item flex items-center gap-3 px-3 py-2.5 rounded-lg text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 hover:text-gray-900 dark:hover:text-white transition-all cursor-pointer text-sm font-medium" onclick="irPara('%s',this)" href="#">`, name))
				b.WriteString(`<span class="w-5 h-5 flex-shrink-0">` + svgIcon(icon) + `</span><span>` + html.EscapeString(cap(model.Name)) + `</span></a>`)
			}
		}
	}
	b.WriteString(`</nav>`)

	// Footer
	b.WriteString(`<div class="p-3 border-t border-gray-200 dark:border-gray-800">`)
	b.WriteString(`<button class="flex items-center gap-3 px-3 py-2 rounded-lg text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 hover:text-gray-900 dark:hover:text-white transition-all cursor-pointer text-sm w-full" onclick="toggleDark()"><span class="w-5 h-5 flex-shrink-0">` + svgIcon("moon") + `</span><span>Tema</span></button>`)
	b.WriteString(`<div class="text-xs text-gray-400 dark:text-gray-600 text-center mt-2">Germanio v0.5</div>`)
	b.WriteString(`</div></aside>`)
	return b.String()
}

// ============================================================
// Topbar
// ============================================================

func (s *Servidor) renderTopbar() string {
	var b strings.Builder
	b.WriteString(`<div class="flex items-center justify-between mb-6 px-2 pt-2">`)

	// Mobile hamburger
	b.WriteString(`<button class="md:hidden text-gray-600 dark:text-gray-400 p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800" onclick="toggleSidebar()">` + svgIcon("menu") + `</button>`)

	// Title
	b.WriteString(`<div>`)
	b.WriteString(`<h1 class="text-2xl font-bold tracking-tight" id="page-title">Dashboard</h1>`)
	b.WriteString(`<p class="text-gray-400 dark:text-gray-500 text-sm">Gerenciamento</p>`)
	b.WriteString(`</div>`)

	// Status chips
	b.WriteString(`<div class="hidden md:flex items-center gap-2 flex-1 justify-center">`)
	b.WriteString(`<div class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-gray-100 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 text-xs font-medium text-gray-500 dark:text-gray-400"><span class="w-2 h-2 rounded-full bg-green-500 shadow-[0_0_0_3px_rgba(34,197,94,0.15)]"></span><span id="tb-sockets">0 conexoes</span></div>`)
	b.WriteString(`<div class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-gray-100 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 text-xs font-medium text-gray-500 dark:text-gray-400"><span class="w-2 h-2 rounded-full bg-amber-500 shadow-[0_0_0_3px_rgba(245,158,11,0.12)]"></span><span id="tb-jobs">jobs 0/0</span></div>`)
	b.WriteString(`<div class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-gray-100 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 text-xs font-medium text-gray-500 dark:text-gray-400"><span class="w-2 h-2 rounded-full bg-cyan-500 shadow-[0_0_0_3px_rgba(6,182,212,0.12)]"></span><span id="tb-wa">whatsapp offline</span></div>`)
	b.WriteString(`</div>`)

	// Right: search + auth
	b.WriteString(`<div class="flex items-center gap-3">`)
	b.WriteString(`<div class="relative"><input type="text" placeholder="Buscar..." id="global-search" oninput="buscaGlobal(this.value)" class="bg-gray-100 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg px-4 py-2 pl-9 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 w-48 focus:w-64 transition-all"><span class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 pointer-events-none">` + svgIcon("search") + `</span></div>`)
	// Auth button
	if s.Auth != nil {
		b.WriteString(`<div id="auth-area" class="flex items-center gap-2">`)
		b.WriteString(`<button class="bg-primary hover:bg-primary/80 text-white px-3 py-2 rounded-lg text-sm font-medium transition-all flex items-center gap-2" id="btn-login" onclick="mostrarLogin()"><span class="w-4 h-4 inline-flex">` + svgIcon("user") + `</span><span>Entrar</span></button>`)
		b.WriteString(`<span id="user-info" class="hidden text-sm text-gray-500 dark:text-gray-400"></span>`)
		b.WriteString(`<button class="hidden text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-gray-800 px-3 py-2 rounded-lg text-sm transition-all" id="btn-logout" onclick="sair()">Sair</button>`)
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)

	b.WriteString(`</div>`)
	return b.String()
}

// ============================================================
// Dashboard
// ============================================================

func (s *Servidor) renderDashboard(theme *ast.Theme) string {
	var b strings.Builder
	b.WriteString(`<div id="secao-dashboard" class="section">`)

	// Bento stat cards
	numCols := len(s.Program.Models)
	gridCols := "grid-cols-2 md:grid-cols-4"
	if numCols <= 2 {
		gridCols = "grid-cols-1 md:grid-cols-2"
	} else if numCols == 3 {
		gridCols = "grid-cols-1 md:grid-cols-3"
	}
	b.WriteString(`<div class="grid ` + gridCols + ` gap-4 mb-6">`)
	for _, model := range s.Program.Models {
		name := lo(model.Name)
		icon := modelIcon(name)
		b.WriteString(fmt.Sprintf(`<div class="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-xl p-5 hover:border-primary/50 transition-all cursor-pointer group" onclick="irParaNav('%s')">`, name))
		b.WriteString(`<div class="flex items-center justify-between mb-3">`)
		b.WriteString(`<div class="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center text-primary"><span class="w-5 h-5 inline-flex">` + svgIcon(icon) + `</span></div>`)
		b.WriteString(fmt.Sprintf(`<span class="text-2xl font-bold" id="stat-%s">0</span>`, name))
		b.WriteString(`</div>`)
		b.WriteString(`<p class="text-gray-500 dark:text-gray-400 text-sm font-medium">` + html.EscapeString(cap(model.Name)) + `</p>`)
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)

	// Chart.js canvas for records per model
	b.WriteString(`<div class="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-xl overflow-hidden mb-6">`)
	b.WriteString(`<div class="flex items-center gap-2.5 px-5 py-4 border-b border-gray-200 dark:border-gray-800"><span class="w-4 h-4 text-primary">` + svgIcon("activity") + `</span><h3 class="text-sm font-semibold">Registros por Modelo</h3></div>`)
	b.WriteString(`<div class="p-5"><canvas id="chart-models" height="260"></canvas></div></div>`)

	// Render any user-defined grafico components from screens
	for _, scr := range s.Program.Screens {
		for _, comp := range scr.Components {
			if comp.Type == ast.CompChart {
				s.renderChartComponent(&b, comp)
			}
		}
	}

	// Status chart for models with status fields
	hasStatus := false
	for _, model := range s.Program.Models {
		for _, f := range model.Fields {
			if f.Type == ast.FieldStatus {
				hasStatus = true
				break
			}
		}
	}
	if hasStatus {
		b.WriteString(`<div class="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-xl overflow-hidden mb-6">`)
		b.WriteString(`<div class="flex items-center gap-2.5 px-5 py-4 border-b border-gray-200 dark:border-gray-800"><span class="w-4 h-4 text-primary">` + svgIcon("tag") + `</span><h3 class="text-sm font-semibold">Status por Modelo</h3></div>`)
		b.WriteString(`<div class="p-5"><canvas id="chart-status" height="260"></canvas></div></div>`)
	}

	// Activity feed + info
	b.WriteString(`<div class="grid grid-cols-1 md:grid-cols-3 gap-4">`)

	// Activity
	b.WriteString(`<div class="md:col-span-2 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-xl overflow-hidden">`)
	b.WriteString(`<div class="flex items-center gap-2.5 px-5 py-4 border-b border-gray-200 dark:border-gray-800"><span class="w-4 h-4 text-primary">` + svgIcon("activity") + `</span><h3 class="text-sm font-semibold">Atividade Recente</h3></div>`)
	b.WriteString(`<div id="atividade" class="max-h-80 overflow-y-auto">`)
	b.WriteString(`<div class="flex flex-col items-center justify-center py-12 text-gray-400 dark:text-gray-600"><span class="w-10 h-10 mb-2 opacity-40">` + svgIcon("inbox") + `</span><p class="text-sm">Nenhuma atividade</p></div>`)
	b.WriteString(`</div></div>`)

	// Info
	b.WriteString(`<div class="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-xl overflow-hidden">`)
	b.WriteString(`<div class="flex items-center gap-2.5 px-5 py-4 border-b border-gray-200 dark:border-gray-800"><span class="w-4 h-4 text-primary">` + svgIcon("info") + `</span><h3 class="text-sm font-semibold">Informacoes</h3></div>`)
	b.WriteString(`<div class="divide-y divide-gray-100 dark:divide-gray-800">`)
	b.WriteString(fmt.Sprintf(`<div class="flex justify-between px-5 py-3 text-sm"><span class="text-gray-500 dark:text-gray-400">Sistema</span><span class="font-medium">%s</span></div>`, html.EscapeString(cap(s.Program.System.Name))))
	b.WriteString(fmt.Sprintf(`<div class="flex justify-between px-5 py-3 text-sm"><span class="text-gray-500 dark:text-gray-400">Modelos</span><span class="font-medium">%d</span></div>`, len(s.Program.Models)))
	b.WriteString(fmt.Sprintf(`<div class="flex justify-between px-5 py-3 text-sm"><span class="text-gray-500 dark:text-gray-400">Telas</span><span class="font-medium">%d</span></div>`, len(s.Program.Screens)))
	b.WriteString(`<div class="flex justify-between px-5 py-3 text-sm"><span class="text-gray-500 dark:text-gray-400">Engine</span><span class="font-medium">Germanio v0.5</span></div>`)
	b.WriteString(`</div></div>`)

	b.WriteString(`</div>`) // grid
	b.WriteString(`</div>`) // dashboard section
	return b.String()
}

// ============================================================
// Chart component (user-defined grafico blocks)
// ============================================================

func (s *Servidor) renderChartComponent(b *strings.Builder, comp *ast.Component) {
	chartType := comp.Properties["tipo"]
	if chartType == "" {
		chartType = "bar"
	}
	target := comp.Target
	title := comp.Properties["titulo"]
	if title == "" {
		title = cap(target) + " - Grafico"
	}
	chartID := "chart-custom-" + lo(target) + "-" + lo(chartType)
	b.WriteString(`<div class="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-xl overflow-hidden mb-6">`)
	b.WriteString(`<div class="flex items-center gap-2.5 px-5 py-4 border-b border-gray-200 dark:border-gray-800"><span class="w-4 h-4 text-primary">` + svgIcon("activity") + `</span><h3 class="text-sm font-semibold">` + title + `</h3></div>`)
	b.WriteString(fmt.Sprintf(`<div class="p-5"><canvas id="%s" height="260" data-chart-type="%s" data-chart-model="%s"></canvas></div></div>`, chartID, chartType, lo(target)))
}

// ============================================================
// Custom screens
// ============================================================

func (s *Servidor) renderCustomScreens(b *strings.Builder) {
	for _, scr := range s.Program.Screens {
		name := lo(scr.Name)
		title := scr.Title
		primaryModel := s.primaryScreenModel(scr)
		if title == "" {
			title = cap(scr.Name)
		}
		b.WriteString(fmt.Sprintf(`<div id="secao-screen-%s" class="section hidden">`, name))
		b.WriteString(`<div class="mb-6"><h2 class="text-xl font-bold">` + html.EscapeString(title) + `</h2></div>`)
		for _, comp := range scr.Components {
			s.renderScreenComponent(b, comp, primaryModel)
		}
		b.WriteString(`</div>`)
	}
}

func (s *Servidor) primaryScreenModel(screen *ast.Screen) *ast.Model {
	if screen == nil {
		return nil
	}
	for _, comp := range screen.Components {
		if model := s.inferModelForTarget(comp.Target); model != nil {
			return model
		}
	}
	return nil
}

func (s *Servidor) findModelByName(name string) *ast.Model {
	target := lo(name)
	for _, model := range s.Program.Models {
		if lo(model.Name) == target {
			return model
		}
	}
	return nil
}

func (s *Servidor) inferModelForTarget(target string) *ast.Model {
	if target == "" {
		return nil
	}
	if model := s.findModelByName(target); model != nil {
		return model
	}
	lookup := lo(target)
	if strings.HasSuffix(lookup, "s") {
		return s.findModelByName(strings.TrimSuffix(lookup, "s"))
	}
	return nil
}

func (s *Servidor) renderInlineListComponent(b *strings.Builder, model *ast.Model) {
	if model == nil {
		return
	}

	name := lo(model.Name)
	b.WriteString(fmt.Sprintf(`<div class="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-xl overflow-hidden mb-4" data-list-model="%s">`, name))
	b.WriteString(`<table class="w-full"><thead><tr class="border-b border-gray-200 dark:border-gray-800">`)
	b.WriteString(`<th class="text-left px-4 py-3 text-xs font-medium text-gray-500 uppercase tracking-wider w-12">#</th>`)
	for _, f := range model.Fields {
		if f.Type == ast.FieldSenha {
			continue
		}
		b.WriteString(`<th class="text-left px-4 py-3 text-xs font-medium text-gray-500 uppercase tracking-wider">` + html.EscapeString(cap(f.Name)) + `</th>`)
	}
	b.WriteString(`<th class="text-right px-4 py-3 text-xs font-medium text-gray-500 uppercase tracking-wider w-24"></th>`)
	b.WriteString(`</tr></thead><tbody class="divide-y divide-gray-100 dark:divide-gray-800"></tbody></table>`)
	b.WriteString(`<div class="flex flex-col items-center justify-center py-12 text-gray-400 dark:text-gray-600">`)
	b.WriteString(`<span class="w-10 h-10 mb-2 opacity-40">` + svgIcon("inbox") + `</span><p class="text-sm">Nenhum registro</p></div></div>`)
}

func (s *Servidor) renderScreenComponent(b *strings.Builder, comp *ast.Component, primaryModel *ast.Model) {
	switch comp.Type {
	case ast.CompList:
		s.renderInlineListComponent(b, s.inferModelForTarget(comp.Target))
	case ast.CompChart:
		s.renderChartComponent(b, comp)
	case ast.CompText:
		text := comp.Properties["conteudo"]
		if text == "" {
			text = comp.Properties["valor"]
		}
		b.WriteString(`<div class="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-xl p-5 mb-4"><p class="text-sm">` + text + `</p></div>`)
	case ast.CompButton:
		label := comp.Properties["texto"]
		if label == "" {
			label = comp.Properties["text"]
		}
		if label == "" {
			label = comp.Properties["label"]
		}
		action := comp.Properties["acao"]
		if action == "" {
			if primaryModel != nil {
				action = fmt.Sprintf("abrirForm('%s')", lo(primaryModel.Name))
			} else if model := s.inferModelForTarget(comp.Target); model != nil {
				action = fmt.Sprintf("abrirForm('%s')", lo(model.Name))
			}
		}
		b.WriteString(fmt.Sprintf(`<button class="bg-primary hover:bg-primary/80 text-white px-4 py-2.5 rounded-xl text-sm font-medium transition-all flex items-center gap-2 mb-4" onclick="%s">%s</button>`, action, html.EscapeString(label)))
	case ast.CompChat:
		s.renderChatComponent(b, comp)
	case ast.CompForm:
		target := lo(comp.Target)
		b.WriteString(fmt.Sprintf(`<div class="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-xl p-5 mb-4"><h3 class="text-base font-semibold mb-4">Formulario - %s</h3>`, cap(target)))
		b.WriteString(fmt.Sprintf(`<form onsubmit="salvar('%s',event)" class="space-y-4">`, target))
		b.WriteString(fmt.Sprintf(`<input type="hidden" id="%s-id">`, target))
		for _, m := range s.Program.Models {
			if lo(m.Name) == target {
				for _, f := range m.Fields {
					s.renderFormField(b, m, f)
				}
				break
			}
		}
		b.WriteString(`<button type="submit" class="bg-primary hover:bg-primary/80 text-white px-4 py-2.5 rounded-xl text-sm font-medium transition-all flex items-center gap-2"><span class="w-4 h-4 inline-flex">` + svgIcon("check") + `</span><span>Salvar</span></button>`)
		b.WriteString(`</form></div>`)
	default:
		for _, child := range comp.Children {
			s.renderScreenComponent(b, child, primaryModel)
		}
	}
}

func (s *Servidor) renderChatComponent(b *strings.Builder, comp *ast.Component) {
	target := lo(comp.Target)
	messagesModel := lo(comp.Properties["messages_model"])
	if messagesModel == "" {
		messagesModel = "mensagem"
	}
	relationField := lo(comp.Properties["relation_field"])
	if relationField == "" {
		relationField = target
	}
	title := comp.Properties["title"]
	if title == "" {
		title = "Chat"
	}
	b.WriteString(fmt.Sprintf(`<div class="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-xl overflow-hidden grid grid-cols-[340px_1fr] min-h-[720px]" data-chat-target="%s" data-chat-messages="%s" data-chat-relation="%s" data-chat-text="%s" data-chat-media="%s" data-chat-author="%s" data-chat-time="%s" data-chat-type="%s">`,
		target, messagesModel, relationField, lo(comp.Properties["text_field"]), lo(comp.Properties["media_field"]), lo(comp.Properties["author_field"]), lo(comp.Properties["timestamp_field"]), lo(comp.Properties["type_field"])))

	// Chat sidebar
	b.WriteString(`<div class="border-r border-gray-200 dark:border-gray-800 flex flex-col">`)
	b.WriteString(`<div class="p-4 border-b border-gray-200 dark:border-gray-800 flex flex-col gap-2">`)
	b.WriteString(`<h3 class="font-semibold text-sm">` + html.EscapeString(title) + `</h3>`)
	b.WriteString(`<input class="w-full bg-gray-100 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50" placeholder="Buscar conversa" oninput="chatFilter('` + target + `',this.value)">`)
	b.WriteString(`</div>`)
	b.WriteString(`<div id="chat-conv-` + target + `" class="flex-1 overflow-auto flex flex-col"></div>`)
	b.WriteString(`</div>`)

	// Chat main
	b.WriteString(`<div class="flex flex-col min-w-0">`)
	b.WriteString(`<div class="p-4 border-b border-gray-200 dark:border-gray-800 flex justify-between items-center gap-3">`)
	b.WriteString(`<div><strong class="text-sm" id="chat-title-` + target + `">Selecione uma conversa</strong>`)
	b.WriteString(`<div id="chat-presence-` + target + `" class="text-xs text-gray-500 dark:text-gray-400 mt-1"></div></div>`)
	b.WriteString(`<button class="text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-gray-800 px-3 py-1.5 rounded-lg text-xs transition-all" type="button" onclick="refreshChat('` + target + `')">Atualizar</button>`)
	b.WriteString(`</div>`)
	b.WriteString(`<div id="chat-msg-` + target + `" class="flex-1 p-6 overflow-auto flex flex-col gap-3 bg-gray-50 dark:bg-gray-950/50">`)
	b.WriteString(`<div class="flex flex-col items-center justify-center py-12 text-gray-400"><p class="text-sm">Nenhuma conversa selecionada</p></div>`)
	b.WriteString(`</div>`)
	b.WriteString(`<div id="chat-typing-` + target + `" class="min-h-[20px] px-4 py-1 text-xs text-gray-500"></div>`)
	b.WriteString(`<form class="flex gap-2 p-4 border-t border-gray-200 dark:border-gray-800 items-center" onsubmit="chatSend('` + target + `',event)">`)
	b.WriteString(`<input type="file" id="chat-file-` + target + `" accept="image/*,audio/*,video/*,.pdf,.doc,.docx,.txt" onchange="chatUpload('` + target + `',this)" class="max-w-[160px] text-xs">`)
	b.WriteString(`<input type="text" id="chat-input-` + target + `" placeholder="Digite uma mensagem" oninput="chatTyping('` + target + `',true)" onblur="chatTyping('` + target + `',false)" class="flex-1 bg-gray-100 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-full px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50">`)
	b.WriteString(`<button class="bg-primary hover:bg-primary/80 text-white px-4 py-2 rounded-xl text-sm font-medium transition-all" type="submit">Enviar</button>`)
	b.WriteString(`</form></div>`)

	b.WriteString(`</div>`) // chat shell
}

// ============================================================
// Model section (auto-generated CRUD)
// ============================================================

func (s *Servidor) renderModelSection(b *strings.Builder, model *ast.Model, theme *ast.Theme) {
	name := lo(model.Name)
	capName := cap(model.Name)

	b.WriteString(fmt.Sprintf(`<div id="secao-%s" class="section hidden">`, name))

	// Section header
	b.WriteString(`<div class="flex items-center justify-between gap-4 mb-5 flex-wrap">`)
	// Search
	b.WriteString(`<div class="flex-1 max-w-sm">`)
	b.WriteString(fmt.Sprintf(`<div class="relative"><input type="text" placeholder="Buscar em %s..." oninput="filtrar('%s',this.value)" class="w-full bg-gray-100 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg px-4 py-2 pl-9 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50">`, html.EscapeString(capName), name))
	b.WriteString(`<span class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 pointer-events-none">` + svgIcon("search") + `</span></div></div>`)
	// Actions
	b.WriteString(`<div class="flex items-center gap-2">`)
	b.WriteString(fmt.Sprintf(`<button class="text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-gray-800 px-3 py-2 rounded-lg text-sm transition-all" onclick="exportar('%s','csv')"><span class="w-4 h-4 inline-block align-middle mr-1">%s</span>CSV</button>`, name, svgIcon("file")))
	b.WriteString(fmt.Sprintf(`<button class="text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-gray-800 px-3 py-2 rounded-lg text-sm transition-all" onclick="exportar('%s','json')"><span class="w-4 h-4 inline-block align-middle mr-1">%s</span>JSON</button>`, name, svgIcon("file")))
	b.WriteString(fmt.Sprintf(`<button class="bg-primary hover:bg-primary/80 text-white px-4 py-2 rounded-xl text-sm font-medium transition-all flex items-center gap-2" onclick="abrirForm('%s')"><span class="w-4 h-4 inline-flex">%s</span><span>Novo %s</span></button>`, name, svgIcon("plus"), html.EscapeString(capName)))
	b.WriteString(`</div></div>`)

	// Status/Enum tabs for filtering
	hasStatusTabs := false
	var tabField string
	var tabValues []string
	for _, f := range model.Fields {
		if f.Type == ast.FieldStatus {
			hasStatusTabs = true
			tabField = lo(f.Name)
			tabValues = []string{"todos", "ativo", "inativo", "pendente", "concluido"}
			break
		}
		if f.Type == ast.FieldEnum && len(f.EnumValues) > 0 {
			hasStatusTabs = true
			tabField = lo(f.Name)
			tabValues = append([]string{"todos"}, f.EnumValues...)
			break
		}
	}
	if hasStatusTabs {
		b.WriteString(fmt.Sprintf(`<div class="flex gap-1 mb-4 bg-gray-100 dark:bg-gray-800 rounded-xl p-1" data-tabs="%s" data-tab-field="%s">`, name, tabField))
		for i, val := range tabValues {
			active := ""
			if i == 0 {
				active = " bg-white dark:bg-gray-700 shadow-sm text-gray-900 dark:text-white"
			} else {
				active = " text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300"
			}
			b.WriteString(fmt.Sprintf(`<button class="px-4 py-2 rounded-lg text-sm font-medium transition-all%s" onclick="filtrarTab('%s','%s','%s',this)">%s</button>`,
				active, name, tabField, val, html.EscapeString(cap(val))))
		}
		b.WriteString(`</div>`)
	}

	// Table
	b.WriteString(`<div class="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-xl overflow-hidden">`)
	b.WriteString(`<div class="overflow-x-auto">`)
	b.WriteString(`<table class="w-full"><thead><tr class="border-b border-gray-200 dark:border-gray-800 bg-gray-50 dark:bg-gray-800/50">`)
	b.WriteString(`<th class="text-left px-4 py-3 text-xs font-medium text-gray-500 uppercase tracking-wider w-12">#</th>`)
	for _, f := range model.Fields {
		if f.Type == ast.FieldSenha {
			continue
		}
		b.WriteString(`<th class="text-left px-4 py-3 text-xs font-medium text-gray-500 uppercase tracking-wider">` + html.EscapeString(cap(f.Name)) + `</th>`)
	}
	b.WriteString(`<th class="text-right px-4 py-3 text-xs font-medium text-gray-500 uppercase tracking-wider w-24"></th>`)
	b.WriteString(`</tr></thead>`)
	b.WriteString(fmt.Sprintf(`<tbody id="tabela-%s" class="divide-y divide-gray-100 dark:divide-gray-800"></tbody></table></div>`, name))
	b.WriteString(fmt.Sprintf(`<div id="paginacao-%s" class="flex items-center justify-center gap-1 p-3 border-t border-gray-200 dark:border-gray-800"></div>`, name))
	b.WriteString(fmt.Sprintf(`<div id="vazio-%s" class="flex flex-col items-center justify-center py-12 text-gray-400 dark:text-gray-600">`, name))
	b.WriteString(`<span class="w-10 h-10 mb-2 opacity-40">` + svgIcon("inbox") + `</span><p class="text-sm">Nenhum registro</p></div></div>`)

	b.WriteString(`</div>`) // section
}

// renderModelModal generates the modal form for a model, rendered at body level (not inside any section).
func (s *Servidor) renderModelModal(b *strings.Builder, model *ast.Model) {
	name := lo(model.Name)
	capName := cap(model.Name)

	b.WriteString(fmt.Sprintf(`<div id="modal-%s" class="fixed inset-0 bg-black/60 backdrop-blur-sm z-[9999] hidden items-center justify-center p-4" onclick="if(event.target===this)fecharForm('%s')">`, name, name))
	b.WriteString(`<div class="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-2xl w-full max-w-lg max-h-[85vh] overflow-y-auto shadow-2xl">`)
	// Header
	b.WriteString(fmt.Sprintf(`<div class="flex items-center justify-between p-5 border-b border-gray-200 dark:border-gray-800"><h2 class="text-lg font-semibold" id="titulo-form-%s">Novo %s</h2>`, name, capName))
	b.WriteString(fmt.Sprintf(`<button onclick="fecharForm('%s')" class="text-gray-400 hover:text-gray-900 dark:hover:text-white p-1.5 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 transition-all"><span class="w-5 h-5 inline-flex">`, name) + svgIcon("x") + `</span></button></div>`)
	// Form
	b.WriteString(fmt.Sprintf(`<form onsubmit="salvar('%s',event)" class="p-5 space-y-4"><input type="hidden" id="%s-id">`, name, name))

	for _, f := range model.Fields {
		s.renderFormField(b, model, f)
	}

	b.WriteString(`<div class="flex gap-3 pt-4 border-t border-gray-200 dark:border-gray-800">`)
	b.WriteString(`<button type="submit" class="flex-1 bg-primary hover:bg-primary/80 text-white py-2.5 rounded-xl font-medium transition-all flex items-center justify-center gap-2"><span class="w-4 h-4 inline-flex">` + svgIcon("check") + `</span>Salvar</button>`)
	b.WriteString(fmt.Sprintf(`<button type="button" onclick="fecharForm('%s')" class="px-6 py-2.5 rounded-xl border border-gray-300 dark:border-gray-700 text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 transition-all text-sm">Cancelar</button>`, name))
	b.WriteString(`</div></form></div></div>`)
}

// renderFormField generates the correct form input element based on field type.
func (s *Servidor) renderFormField(b *strings.Builder, model *ast.Model, f *ast.Field) {
	name := lo(model.Name)
	fname := lo(f.Name)
	req := ""
	if f.Required {
		req = " required"
	}

	inputClass := `w-full bg-gray-100 dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-xl px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary placeholder-gray-400 dark:placeholder-gray-600 transition-all`

	b.WriteString(`<div>`)
	b.WriteString(fmt.Sprintf(`<label for="%s-%s" class="block text-sm font-medium text-gray-500 dark:text-gray-400 mb-1.5">%s</label>`, name, fname, html.EscapeString(cap(f.Name))))

	switch {
	// FK dropdown
	case f.Reference != "":
		refModel := lo(f.Reference)
		b.WriteString(fmt.Sprintf(`<select id="%s-%s" data-ref="%s" class="%s"%s>`,
			name, fname, refModel, inputClass, req))
		b.WriteString(`<option value="">Selecione...</option></select>`)

	// Enum dropdown
	case f.Type == ast.FieldEnum && len(f.EnumValues) > 0:
		b.WriteString(fmt.Sprintf(`<select id="%s-%s" class="%s"%s>`, name, fname, inputClass, req))
		b.WriteString(`<option value="">Selecione...</option>`)
		for _, v := range f.EnumValues {
			b.WriteString(fmt.Sprintf(`<option value="%s">%s</option>`, html.EscapeString(v), html.EscapeString(cap(v))))
		}
		b.WriteString(`</select>`)

	// Status dropdown
	case f.Type == ast.FieldStatus:
		b.WriteString(fmt.Sprintf(`<select id="%s-%s" class="%s"%s>`, name, fname, inputClass, req))
		b.WriteString(`<option value="">Selecione...</option>`)
		for _, v := range []string{"ativo", "inativo", "pendente", "concluido"} {
			b.WriteString(fmt.Sprintf(`<option value="%s">%s</option>`, v, cap(v)))
		}
		b.WriteString(`</select>`)

	// Long text
	case f.Type == ast.FieldTextoLongo:
		b.WriteString(fmt.Sprintf(`<textarea id="%s-%s" placeholder="%s" rows="4" class="%s resize-none"%s></textarea>`,
			name, fname, html.EscapeString(cap(f.Name)), inputClass, req))

	// File/image upload
	case f.Type == ast.FieldImagem || f.Type == ast.FieldUpload || f.Type == ast.FieldArquivo:
		b.WriteString(fmt.Sprintf(`<input type="hidden" id="%s-%s">`, name, fname))
		b.WriteString(fmt.Sprintf(`<input type="file" id="%s-%s-file" onchange="uploadFile('%s','%s',this)" class="%s cursor-pointer">`,
			name, fname, name, fname, inputClass))
		b.WriteString(fmt.Sprintf(`<div id="%s-%s-preview" class="mt-2"></div>`, name, fname))

	// Boolean checkbox
	case f.Type == ast.FieldBooleano:
		b.WriteString(fmt.Sprintf(`<label class="relative inline-flex items-center cursor-pointer"><input type="checkbox" id="%s-%s" class="sr-only peer"%s><div class="w-11 h-6 bg-gray-300 dark:bg-gray-700 rounded-full peer peer-checked:bg-primary peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all"></div></label>`,
			name, fname, req))

	case f.Type == ast.FieldCPF:
		b.WriteString(fmt.Sprintf(`<input type="text" id="%s-%s" placeholder="000.000.000-00" maxlength="18" oninput="maskCPF(this)" class="%s"%s>`, name, fname, inputClass, req))

	case f.Type == ast.FieldCEP:
		b.WriteString(fmt.Sprintf(`<input type="text" id="%s-%s" placeholder="00000-000" maxlength="9" oninput="maskCEP(this)" onblur="buscaCEP(this)" class="%s"%s>`, name, fname, inputClass, req))

	case f.Type == ast.FieldCor:
		b.WriteString(fmt.Sprintf(`<input type="color" id="%s-%s" value="#6366f1" style="height:40px;padding:4px" class="%s"%s>`, name, fname, inputClass, req))

	case f.Type == ast.FieldEstrelas:
		b.WriteString(fmt.Sprintf(`<div class="flex gap-1" id="%s-%s-wrap">`, name, fname))
		for i := 1; i <= 5; i++ {
			b.WriteString(fmt.Sprintf(`<button type="button" class="text-2xl text-gray-600 hover:text-yellow-400 transition-colors" onclick="setStars('%s-%s',%d)">★</button>`, name, fname, i))
		}
		b.WriteString(fmt.Sprintf(`<input type="hidden" id="%s-%s" value="0">`, name, fname))
		b.WriteString(`</div>`)

	case f.Type == ast.FieldHora:
		b.WriteString(fmt.Sprintf(`<input type="time" id="%s-%s" class="%s"%s>`, name, fname, inputClass, req))

	case f.Type == ast.FieldDataHora:
		b.WriteString(fmt.Sprintf(`<input type="datetime-local" id="%s-%s" class="%s"%s>`, name, fname, inputClass, req))

	case f.Type == ast.FieldPercentual:
		b.WriteString(fmt.Sprintf(`<div class="relative"><input type="number" id="%s-%s" min="0" max="100" step="0.1" placeholder="0" class="%s"%s><span class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 text-sm">%%</span></div>`, name, fname, inputClass, req))

	case f.Type == ast.FieldTags:
		b.WriteString(fmt.Sprintf(`<input type="text" id="%s-%s" placeholder="Digite e pressione Enter para adicionar" onkeydown="if(event.key==='Enter'){event.preventDefault();addTag('%s-%s',this)}" class="%s"%s>`, name, fname, name, fname, inputClass, req))
		b.WriteString(fmt.Sprintf(`<div id="%s-%s-tags" class="flex flex-wrap gap-1 mt-1"></div>`, name, fname))

	case f.Type == ast.FieldURL:
		b.WriteString(fmt.Sprintf(`<input type="url" id="%s-%s" placeholder="https://..." pattern="https?://.+" class="%s"%s>`, name, fname, inputClass, req))

	case f.Type == ast.FieldMoeda:
		b.WriteString(fmt.Sprintf(`<select id="%s-%s" class="%s"%s>`, name, fname, inputClass, req))
		b.WriteString(`<option value="BRL">R$ - Real Brasileiro</option>`)
		b.WriteString(`<option value="USD">$ - Dolar Americano</option>`)
		b.WriteString(`<option value="EUR">€ - Euro</option>`)
		b.WriteString(`<option value="GBP">£ - Libra</option>`)
		b.WriteString(`</select>`)

	// All other input types
	default:
		inputType := tipoInput(f.Type)
		extra := ""
		if f.Type == ast.FieldNumero || f.Type == ast.FieldDinheiro {
			extra = ` step="any"`
		}
		if f.Type == ast.FieldSenha {
			inputType = "password"
		}
		b.WriteString(fmt.Sprintf(`<input type="%s" id="%s-%s" placeholder="%s" class="%s"%s%s>`,
			inputType, name, fname, html.EscapeString(cap(f.Name)), inputClass, extra, req))
	}

	b.WriteString(`</div>`)
}

// ============================================================
// CSS Generation - minimal, mostly Tailwind handles it
// ============================================================

func (s *Servidor) generateCSS(theme *ast.Theme) string {
	css := `
*{margin:0;padding:0;box-sizing:border-box}

/* Smooth scrollbar */
::-webkit-scrollbar{width:6px;height:6px}
::-webkit-scrollbar-track{background:transparent}
::-webkit-scrollbar-thumb{background:rgba(128,128,128,0.2);border-radius:3px}
::-webkit-scrollbar-thumb:hover{background:rgba(128,128,128,0.4)}

/* Animations */
@keyframes fadeUp{from{opacity:0;transform:translateY(12px)}to{opacity:1;transform:translateY(0)}}
.section{animation:fadeUp .3s ease-out}

/* SVG sizing inside spans */
span > svg, button > svg, div > svg, a > svg {width:100%;height:100%}

/* Toast show state */
.toast-show{opacity:1!important;transform:translateY(0)!important;pointer-events:auto!important}

/* Pagination buttons */
.pg-btn{min-width:34px;height:34px;display:inline-flex;align-items:center;justify-content:center;
  border:1px solid;border-radius:8px;cursor:pointer;font-size:.8rem;font-weight:600;transition:all .2s}

/* Status pills */
.pill{display:inline-flex;align-items:center;gap:4px;padding:3px 12px;border-radius:99px;font-size:.78rem;font-weight:600;text-transform:capitalize}
.pill::before{content:'';width:6px;height:6px;border-radius:50%;flex-shrink:0}
.pill-green{background:rgba(22,163,74,.1);color:#16a34a}.pill-green::before{background:#16a34a}
.dark .pill-green{background:rgba(22,163,74,.15);color:#4ade80}
.pill-red{background:rgba(239,68,68,.1);color:#ef4444}.pill-red::before{background:#ef4444}
.dark .pill-red{background:rgba(239,68,68,.15);color:#fca5a5}
.pill-yellow{background:rgba(245,158,11,.1);color:#d97706}.pill-yellow::before{background:#f59e0b}
.dark .pill-yellow{background:rgba(245,158,11,.15);color:#fde047}
.pill-blue{background:rgba(59,130,246,.1);color:#3b82f6}.pill-blue::before{background:#3b82f6}
.dark .pill-blue{background:rgba(59,130,246,.15);color:#93c5fd}

/* Chat bubbles */
.chat-bubble{max-width:min(72%,680px);padding:13px 15px;border-radius:18px;position:relative}
.chat-bubble.mine{align-self:flex-end;border-bottom-right-radius:8px}
.chat-bubble.other{align-self:flex-start}
.chat-bubble-meta{font-size:.72rem;margin-top:6px}
.chat-media img,.chat-media video{max-width:100%;border-radius:12px}
.chat-media audio{width:100%}

/* Chat conversation item */
.chat-conv{padding:14px 16px;cursor:pointer;transition:background .2s;display:grid;grid-template-columns:44px 1fr auto;gap:12px;align-items:center}

/* Responsive sidebar */
@media(max-width:768px){
  #sidebar{transform:translateX(-100%);z-index:100}
  #sidebar.open{transform:translateX(0)}
  #main{margin-left:0!important}
  .chat-grid-responsive{grid-template-columns:1fr!important;min-height:unset!important}
}
`

	// Inject user custom CSS
	if theme.CustomCSS != "" {
		css += "\n/* === User Custom CSS === */\n" + theme.CustomCSS + "\n"
	}

	return css
}

// styleVariantCSS returns extra CSS for the chosen style variant.
// With Tailwind, this is simplified since most styling is in classes.
func (s *Servidor) styleVariantCSS(style string) string {
	return ""
}

// ============================================================
// JavaScript Generation
// ============================================================

func (s *Servidor) generateJS(theme *ast.Theme) string {
	var b strings.Builder

	// Icon strings for table action buttons
	editIcon := strings.ReplaceAll(strings.ReplaceAll(svgIcon("edit"), `"`, `'`), "\n", "")
	trashIcon := strings.ReplaceAll(strings.ReplaceAll(svgIcon("trash"), `"`, `'`), "\n", "")
	b.WriteString(fmt.Sprintf("var ICO_E=%q,ICO_D=%q;\n", editIcon, trashIcon))

	// Model metadata for JS
	b.WriteString("var M={\n")
	for _, model := range s.Program.Models {
		name := lo(model.Name)
		b.WriteString(fmt.Sprintf("'%s':[", name))
		for i, f := range model.Fields {
			ft := fieldTypeCode(f)
			ref := ""
			if f.Reference != "" {
				ref = fmt.Sprintf(",r:'%s'", lo(f.Reference))
			}
			enumVals := ""
			if f.Type == ast.FieldEnum && len(f.EnumValues) > 0 {
				enumVals = fmt.Sprintf(",ev:%s", enumJSArray(f.EnumValues))
			}
			if i > 0 {
				b.WriteString(",")
			}
			b.WriteString(fmt.Sprintf("{n:'%s',t:'%s'%s%s}", lo(f.Name), ft, ref, enumVals))
		}
		b.WriteString("],\n")
	}
	b.WriteString("};\n")

	// Pagination state
	b.WriteString("var PAGE_SIZE=20,pages={};\n")

	// Chart.js theme colors
	b.WriteString(fmt.Sprintf("var THEME_PRI='%s',THEME_SEC='%s',THEME_ACC='%s';\n", theme.Primary, theme.Secondary, theme.Accent))
	b.WriteString("var chartColors=[THEME_PRI,THEME_SEC,THEME_ACC,'#10b981','#3b82f6','#ef4444','#06b6d4','#ec4899'];\n")

	b.WriteString(`
var ativs=[];
function $(id){return document.getElementById(id);}
function esc(v){if(v==null)return'';var d=document.createElement('div');d.textContent=String(v);return d.innerHTML;}

// ===== Navigation =====
function irPara(n,el){
  document.querySelectorAll('.section').forEach(function(s){s.classList.add('hidden');});
  var sec=$('secao-'+n);
  if(sec){sec.classList.remove('hidden');}
  // Update sidebar active state
  document.querySelectorAll('.nav-item').forEach(function(a){
    a.classList.remove('bg-primary/10','text-primary');
    a.classList.add('text-gray-500','dark:text-gray-400');
  });
  if(el){
    el.classList.add('bg-primary/10','text-primary');
    el.classList.remove('text-gray-500','dark:text-gray-400');
  }
  var title=n==='dashboard'?'Dashboard':n.replace('screen-','').charAt(0).toUpperCase()+n.replace('screen-','').slice(1);
  $('page-title').textContent=title;
  if(innerWidth<768)$('sidebar').classList.remove('open');
  // Load data for inline lists in custom screens
  if(sec){carregarListasInline();}
  // Also load data for model section
  if(n!=='dashboard'&&!n.startsWith('screen-')){carregar(n);}
}
function irParaNav(n){
  var links=document.querySelectorAll('.nav-item');
  for(var i=0;i<links.length;i++){
    var sp=links[i].querySelector('span:last-child');
    if(sp&&sp.textContent.toLowerCase()===n){irPara(n,links[i]);return;}
  }
  irPara(n,null);
}

function toggleSidebar(){$('sidebar').classList.toggle('open');}
function toggleDark(){document.documentElement.classList.toggle('dark');}

function toast(msg,t){
  var e=$('toast');e.textContent=msg;
  e.className='fixed bottom-4 right-4 z-[10000] px-4 py-3 rounded-xl shadow-lg transition-all duration-300 toast-show text-white font-medium text-sm';
  if(t==='erro'){e.classList.add('bg-red-600');}else{e.classList.add('bg-green-600');}
  setTimeout(function(){e.className='fixed bottom-4 right-4 z-[10000] px-4 py-3 rounded-xl shadow-lg transform translate-y-20 opacity-0 transition-all duration-300 pointer-events-none';},3000);
}

// ===== Form open/close =====
function abrirForm(m){
  var modal=$('modal-'+m);
  modal.classList.remove('hidden');
  modal.classList.add('flex');
  $(m+'-id').value='';
  modal.querySelector('form').reset();
  $('titulo-form-'+m).textContent='Novo '+m.charAt(0).toUpperCase()+m.slice(1);
  M[m].forEach(function(c){
    if(c.t==='f'){var prev=$(m+'-'+c.n+'-preview');if(prev)prev.innerHTML='';}
  });
  carregarSelects(m);
}
function fecharForm(m){
  var modal=$('modal-'+m);
  modal.classList.add('hidden');
  modal.classList.remove('flex');
}

// ===== Tab Filter =====
function filtrarTab(model,field,value,btn){
  var container=btn.parentElement;
  container.querySelectorAll('button').forEach(function(b){
    b.className=b.className.replace(/bg-white|dark:bg-gray-700|shadow-sm|text-gray-900|dark:text-white/g,'');
    if(b.className.indexOf('text-gray-500')<0){
      b.className+=' text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300';
    }
  });
  btn.className=btn.className.replace(/text-gray-500|dark:text-gray-400|hover:text-gray-700|dark:hover:text-gray-300/g,'');
  btn.className+=' bg-white dark:bg-gray-700 shadow-sm text-gray-900 dark:text-white';
  var rows=document.querySelectorAll('#tabela-'+model+' tr');
  rows.forEach(function(row){
    if(value==='todos'){row.style.display='';return;}
    var text=row.textContent.toLowerCase();
    row.style.display=text.indexOf(value.toLowerCase())>=0?'':'none';
  });
}

// ===== Search / Filter =====
function filtrar(m,q){
  q=q.toLowerCase();
  document.querySelectorAll('#tabela-'+m+' tr').forEach(function(r){
    r.style.display=r.textContent.toLowerCase().includes(q)?'':'none';
  });
}
function buscaGlobal(q){
  if(!q){document.querySelectorAll('table tr').forEach(function(r){r.style.display='';});return;}
  q=q.toLowerCase();
  document.querySelectorAll('table tbody tr').forEach(function(r){r.style.display=r.textContent.toLowerCase().includes(q)?'':'none';});
}

// ===== File upload =====
function uploadFile(m,fname,input){
  if(!input.files||!input.files[0])return;
  var fd=new FormData();fd.append('file',input.files[0]);
  var prev=$(m+'-'+fname+'-preview');
  if(prev)prev.innerHTML='<span class="text-gray-500 text-sm">Enviando...</span>';
  fetch('/upload',{method:'POST',body:fd})
    .then(function(r){if(!r.ok)throw new Error('Upload falhou');return r.json();})
    .then(function(d){
      $(m+'-'+fname).value=d.path;
      if(prev){
        if(d.path.match(/\.(jpg|jpeg|png|gif|webp|svg)$/i)){
          prev.innerHTML='<img src="'+esc(d.path)+'" class="max-w-full max-h-28 rounded-lg mt-1">';
        }else{
          prev.innerHTML='<span class="text-primary text-sm mt-1 block">'+esc(d.name)+' &#10003;</span>';
        }
      }
    })
    .catch(function(err){toast('Erro upload: '+err.message,'erro');if(prev)prev.innerHTML='';});
}

// ===== FK & Enum select population =====
function carregarSelects(m){
  M[m].forEach(function(c){
    if(!c.r)return;
    var sel=$(m+'-'+c.n);if(!sel||sel.tagName!=='SELECT')return;
    fetch('/api/'+c.r).then(function(r){return r.json();}).then(function(items){
      var val=sel.value;
      sel.innerHTML='<option value="">Selecione...</option>';
      if(!items||!items.length)return;
      var labelKey=null;
      if(M[c.r]){for(var i=0;i<M[c.r].length;i++){if(M[c.r][i].t==='t'||M[c.r][i].t==='e'){labelKey=M[c.r][i].n;break;}}}
      items.forEach(function(it){
        var label=labelKey?it[labelKey]:(it.nome||it.name||it.titulo||it.title||'#'+it.id);
        var o=document.createElement('option');o.value=it.id;o.textContent=label;
        if(String(it.id)===String(val))o.selected=true;
        sel.appendChild(o);
      });
    });
  });
}

// ===== Table cell formatting =====
function fmtCell(v,t){
  var s=esc(v);
  if(!s||s==='-')return'<span class="text-gray-400">&#8212;</span>';
  if(t==='s')return'<span class="pill pill-'+pillColor(v)+'">'+s+'</span>';
  if(t==='d'){var n=parseFloat(v);return'<span class="font-semibold text-primary tabular-nums">R$&nbsp;'+n.toFixed(2)+'</span>';}
  if(t==='e')return'<a class="text-primary hover:underline font-medium" href="mailto:'+s+'">'+s+'</a>';
  if(t==='en')return'<span class="pill pill-blue">'+s+'</span>';
  if(t==='b'){return v?'<span class="pill pill-green">Sim</span>':'<span class="pill pill-red">N&atilde;o</span>';}
  if(t==='f'){
    if(String(v).match(/\.(jpg|jpeg|png|gif|webp|svg)$/i))return'<img src="'+s+'" class="max-h-10 rounded">';
    return'<a class="text-primary hover:underline font-medium" href="'+s+'" target="_blank">'+s.split('/').pop()+'</a>';
  }
  if(t==='tl'){return s.length>60?s.substring(0,60)+'...':s;}
  return s;
}

function pillColor(v){
  if(!v)return'blue';v=v.toLowerCase();
  if('ativo,livre,aberto,ok,sim,disponivel,pronto,entregue,pago,aprovado,online,concluido'.indexOf(v)>=0)return'green';
  if('inativo,ocupado,fechado,nao,cancelado,bloqueado,offline,reprovado'.indexOf(v)>=0)return'red';
  if('pendente,aguardando,em andamento,reservado,preparando,analise'.indexOf(v)>=0)return'yellow';
  return'blue';
}

// ===== Activity feed =====
function addAtiv(tipo,mod,nome){
  var labs={c:'Criado',e:'Editado',d:'Exclu\u00eddo'};
  var now=new Date();var h=String(now.getHours()).padStart(2,'0')+':'+String(now.getMinutes()).padStart(2,'0');
  ativs.unshift({t:tipo,m:mod,n:nome,h:h,l:labs[tipo]});
  if(ativs.length>15)ativs.pop();
  renderAtiv();
}
function renderAtiv(){
  var el=$('atividade');
  if(!ativs.length)return;
  var h='';ativs.forEach(function(a){
    var tagClass=a.t==='c'?'bg-green-600':a.t==='e'?'bg-primary':'bg-red-600';
    h+='<div class="flex items-center gap-3 px-5 py-3 text-sm hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors">';
    h+='<span class="text-[10px] px-2 py-0.5 rounded-full font-bold text-white uppercase tracking-wide '+tagClass+'">'+a.l+'</span>';
    h+='<span class="flex-1 truncate"><b>'+esc(a.m)+'</b>';
    if(a.n)h+=' \u2014 '+esc(a.n);
    h+='</span><span class="text-gray-400 text-xs tabular-nums flex-shrink-0">'+a.h+'</span></div>';
  });
  el.innerHTML=h;
}

function renderTableRows(tb,m,items){
  var cs=M[m];
  if(!tb||!cs)return;
  tb.innerHTML='';
  (items||[]).forEach(function(item){
    var tr=document.createElement('tr');
    tr.className='hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors';
    var h='<td class="px-4 py-3 text-sm font-bold text-gray-400 w-12">'+item.id+'</td>';
    cs.forEach(function(c){
      if(c.t==='pw')return;
      h+='<td class="px-4 py-3 text-sm">'+fmtCell(item[c.n],c.t)+'</td>';
    });
    h+='<td class="px-4 py-3 text-right whitespace-nowrap">';
    h+='<button class="w-8 h-8 inline-flex items-center justify-center rounded-lg text-primary hover:bg-primary/10 transition-all" onclick="editar(\''+m+'\','+item.id+')"><span class="w-4 h-4 inline-flex">'+ICO_E+'</span></button>';
    h+='<button class="w-8 h-8 inline-flex items-center justify-center rounded-lg text-red-500 hover:bg-red-500/10 transition-all" onclick="excluir(\''+m+'\','+item.id+')"><span class="w-4 h-4 inline-flex">'+ICO_D+'</span></button>';
    h+='</td>';
    tr.innerHTML=h;tb.appendChild(tr);
  });
}

function carregarListasInline(modelo){
  var selector=modelo?'[data-list-model="'+modelo+'"]':'[data-list-model]';
  document.querySelectorAll(selector).forEach(function(card){
    var m=card.getAttribute('data-list-model');
    var tb=card.querySelector('tbody');
    var vazio=card.querySelector('.flex.flex-col.items-center');
    var table=card.querySelector('table');
    if(!m||!tb||!table)return;
    fetch('/api/'+m).then(function(r){return r.json();}).then(function(items){
      items=items||[];
      console.log('carregarListasInline',m,'items:',items.length);
      renderTableRows(tb,m,items);
      if(!items.length){
        if(vazio)vazio.style.display='flex';
        table.style.display='none';
        return;
      }
      if(vazio)vazio.style.display='none';
      table.style.display='';
    }).catch(function(err){
      console.error('carregarListasInline error',m,err);
      if(vazio)vazio.style.display='flex';
      table.style.display='none';
    });
  });
}

// ===== Data loading with pagination =====
function carregar(m,page){
  if(!page)page=1;
  pages[m]=page;
  fetch('/api/'+m).then(function(r){return r.json();}).then(function(items){
    var tb=$('tabela-'+m),vz=$('vazio-'+m),st=$('stat-'+m);
    if(!tb)return;
    tb.innerHTML='';
    var total=items?items.length:0;
    if(st)st.textContent=total;
    if(!items||!items.length){
      if(vz)vz.style.display='flex';if(tb.closest('table'))tb.closest('table').style.display='none';
      var pg=$('paginacao-'+m);if(pg)pg.innerHTML='';
      return;
    }
    if(vz)vz.style.display='none';if(tb.closest('table'))tb.closest('table').style.display='';

    // Pagination
    var totalPages=Math.ceil(total/PAGE_SIZE);
    var start=(page-1)*PAGE_SIZE;
    var end=Math.min(start+PAGE_SIZE,total);
    var pageItems=items.slice(start,end);

    renderTableRows(tb,m,pageItems);

    // Render pagination controls
    var pg=$('paginacao-'+m);
    if(pg&&totalPages>1){
      var btnBase='min-w-[34px] h-[34px] inline-flex items-center justify-center border rounded-lg cursor-pointer text-xs font-semibold transition-all ';
      var btnNormal=btnBase+'border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 hover:border-primary hover:text-primary';
      var btnActive=btnBase+'bg-primary text-white border-primary';
      var btnDisabled=btnBase+'border-gray-200 dark:border-gray-700 bg-gray-100 dark:bg-gray-800 text-gray-400 cursor-default opacity-40';
      var ph='<button class="'+(page<=1?btnDisabled:btnNormal)+'" '+(page<=1?'disabled':'')+' onclick="carregar(\''+m+'\','+(page-1)+')">&laquo;</button>';
      for(var i=1;i<=totalPages;i++){
        if(totalPages>7&&Math.abs(i-page)>2&&i!==1&&i!==totalPages){
          if(i===2||i===totalPages-1)ph+='<button class="'+btnDisabled+'" disabled>...</button>';
          continue;
        }
        ph+='<button class="'+(i===page?btnActive:btnNormal)+'" onclick="carregar(\''+m+'\','+i+')">'+i+'</button>';
      }
      ph+='<button class="'+(page>=totalPages?btnDisabled:btnNormal)+'" '+(page>=totalPages?'disabled':'')+' onclick="carregar(\''+m+'\','+(page+1)+')">&raquo;</button>';
      pg.innerHTML=ph;
    }else if(pg){pg.innerHTML='';}
  });
}

// ===== CRUD operations =====
function salvar(m,e){
  e.preventDefault();var id=$(m+'-id').value;var d={};
  M[m].forEach(function(c){
    var el=$(m+'-'+c.n);if(!el)return;
    if(c.t==='b'){d[c.n]=el.checked;return;}
    var v=el.value;
    d[c.n]=(c.t==='n'||c.t==='d')?parseFloat(v)||0:v;
  });
  fetch(id?'/api/'+m+'/'+id:'/api/'+m,{method:id?'PUT':'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(d)})
    .then(function(r){if(!r.ok)return r.json().then(function(e){throw new Error(e.erro||e.error||'Erro');});return r.json();})
    .then(function(){fecharForm(m);carregar(m);carregarListasInline(m);addAtiv(id?'e':'c',m,d[M[m][0].n]||'');toast(id?'Atualizado!':'Criado!');renderCharts();})
    .catch(function(err){toast('Erro: '+err.message,'erro');});
}

function editar(m,id){
  fetch('/api/'+m+'/'+id).then(function(r){return r.json();}).then(function(item){
    $(m+'-id').value=item.id;
    carregarSelects(m);
    M[m].forEach(function(c){
      var el=$(m+'-'+c.n);
      if(!el)return;
      if(c.t==='b'){el.checked=!!item[c.n];return;}
      el.value=item[c.n]||'';
      if(c.t==='f'){
        var prev=$(m+'-'+c.n+'-preview');
        if(prev&&item[c.n]){
          if(String(item[c.n]).match(/\.(jpg|jpeg|png|gif|webp|svg)$/i)){
            prev.innerHTML='<img src="'+esc(item[c.n])+'" class="max-w-full max-h-28 rounded-lg mt-1">';
          }else{
            prev.innerHTML='<span class="text-primary text-sm mt-1 block">'+esc(item[c.n])+'</span>';
          }
        }
      }
      if(c.r&&el){setTimeout(function(){el.value=item[c.n]||'';},300);}
    });
    $('titulo-form-'+m).textContent='Editar';
    var modal=$('modal-'+m);
    modal.classList.remove('hidden');
    modal.classList.add('flex');
  });
}

function excluir(m,id){
  if(!confirm('Excluir #'+id+'?'))return;
  var tb=$('tabela-'+m),rows=tb.querySelectorAll('tr'),label='';
  rows.forEach(function(r){var td=r.querySelector('td');if(td&&td.textContent==id){label=r.children[1]?r.children[1].textContent:'';}});
  fetch('/api/'+m+'/'+id,{method:'DELETE'}).then(function(){carregar(m);carregarListasInline(m);addAtiv('d',m,label);toast('Excluido!');renderCharts();});
}

function exportar(m,fmt){
  var a=document.createElement('a');a.href='/api/'+m+'/export/'+fmt;a.download='';document.body.appendChild(a);a.click();document.body.removeChild(a);
}

// ===== Chart.js rendering =====
var chartInstances={};
var CHATS={};

function refreshSystemStatus(){
  fetch('/api/_jobs/status').then(function(r){return r.json();}).then(function(d){
    var queued=(d&&d.queued)||0, running=(d&&d.running)||0;
    if($('tb-jobs')) $('tb-jobs').textContent='jobs '+queued+'/'+running;
  }).catch(function(){});
  fetch('/api/whatsapp/sessions').then(function(r){if(!r.ok) throw new Error(); return r.json();}).then(function(items){
    var connected=(items||[]).filter(function(it){return !!it.connected;}).length;
    if($('tb-wa')) $('tb-wa').textContent=connected>0?('whatsapp '+connected+' online'):('whatsapp offline');
  }).catch(function(){ if($('tb-wa')) $('tb-wa').textContent='whatsapp offline'; });
  if($('tb-sockets') && typeof ws!=='undefined' && ws){ $('tb-sockets').textContent='tempo real ativo'; }
}

function renderCharts(){
  fetch('/api/_stats').then(function(r){return r.json();}).then(function(stats){
    var models=Object.keys(stats);
    var isDark=document.documentElement.classList.contains('dark');
    var textColor=isDark?'#9ca3af':'#6b7280';

    // Bar chart - records per model
    var el=$('chart-models');
    if(el){
      var labels=models.map(function(m){return m.charAt(0).toUpperCase()+m.slice(1);});
      var data=models.map(function(m){return stats[m].count||0;});
      var bgColors=models.map(function(_,i){return chartColors[i%chartColors.length];});
      if(chartInstances['models'])chartInstances['models'].destroy();
      chartInstances['models']=new Chart(el,{
        type:'bar',
        data:{labels:labels,datasets:[{label:'Registros',data:data,backgroundColor:bgColors,borderRadius:6,borderSkipped:false}]},
        options:{responsive:true,maintainAspectRatio:false,plugins:{legend:{display:false}},
          scales:{y:{beginAtZero:true,grid:{color:isDark?'rgba(128,128,128,0.1)':'rgba(0,0,0,0.06)'},ticks:{color:textColor}},
                  x:{grid:{display:false},ticks:{color:textColor}}}}
      });
    }

    // Status doughnut chart
    var sel=$('chart-status');
    if(sel){
      var statusData={};
      models.forEach(function(m){
        var st=stats[m].statuses;
        if(!st)return;
        Object.keys(st).forEach(function(k){statusData[k]=(statusData[k]||0)+st[k];});
      });
      var sKeys=Object.keys(statusData);
      if(sKeys.length){
        var sColors=sKeys.map(function(k){var pc=pillColor(k);return{green:'#16a34a',red:'#ef4444',yellow:'#f59e0b',blue:'#3b82f6'}[pc]||THEME_PRI;});
        if(chartInstances['status'])chartInstances['status'].destroy();
        chartInstances['status']=new Chart(sel,{
          type:'doughnut',
          data:{labels:sKeys.map(function(k){return k.charAt(0).toUpperCase()+k.slice(1);}),
                datasets:[{data:sKeys.map(function(k){return statusData[k];}),backgroundColor:sColors,borderWidth:0}]},
          options:{responsive:true,maintainAspectRatio:false,cutout:'60%',
            plugins:{legend:{position:'bottom',labels:{color:textColor,padding:16,usePointStyle:true,pointStyle:'circle'}}}}
        });
      }
    }

    // Custom chart canvases
    document.querySelectorAll('canvas[data-chart-model]').forEach(function(canvas){
      var cid=canvas.id;
      var ctype=canvas.getAttribute('data-chart-type')||'bar';
      var cmodel=canvas.getAttribute('data-chart-model');
      if(!stats[cmodel])return;
      fetch('/api/'+cmodel).then(function(r){return r.json();}).then(function(items){
        if(!items||!items.length)return;
        if(chartInstances[cid])chartInstances[cid].destroy();
        var cs=M[cmodel];if(!cs)return;
        var numField=null,labelField=null;
        cs.forEach(function(c){if(!numField&&(c.t==='n'||c.t==='d'))numField=c.n;if(!labelField&&c.t==='t')labelField=c.n;});
        if(!numField)return;
        if(!labelField)labelField=cs[0].n;
        var labels=items.map(function(it){return it[labelField]||'#'+it.id;});
        var data=items.map(function(it){return parseFloat(it[numField])||0;});
        chartInstances[cid]=new Chart(canvas,{
          type:ctype,
          data:{labels:labels,datasets:[{label:numField,data:data,
            backgroundColor:items.map(function(_,i){return chartColors[i%chartColors.length];}),
            borderColor:ctype==='line'?THEME_PRI:undefined,
            borderWidth:ctype==='line'?2:0,borderRadius:ctype==='bar'?6:0,
            fill:ctype==='line'?false:undefined,tension:0.3}]},
          options:{responsive:true,maintainAspectRatio:false,
            plugins:{legend:{display:false}},
            scales:ctype==='doughnut'||ctype==='pie'?{}:{
              y:{beginAtZero:true,grid:{color:isDark?'rgba(128,128,128,0.1)':'rgba(0,0,0,0.06)'}},
              x:{grid:{display:false}}}}
        });
      });
    });
  }).catch(function(){});
}

// ===== WebSocket for real-time updates =====
var ws;
function connectWS(){
  var proto=location.protocol==='https:'?'wss:':'ws:';
  ws=new WebSocket(proto+'//'+location.host+'/ws');
  ws.onmessage=function(e){
    try{
      var msg=JSON.parse(e.data);
      if(msg.model){carregar(msg.model);carregarListasInline(msg.model);renderCharts();chatHandleUpdate(msg);}
      if(msg.type==='presenca'||msg.type==='digitando'||msg.type==='presenca_socket'){chatHandlePresence(msg);}
      if(msg.type==='qr'){toast('QR code atualizado para sessao '+(msg.session||'default'));}
      if(msg.type==='whatsapp_status'&&msg.data&&msg.data.status){toast('WhatsApp '+msg.data.status);}
      if(msg.type==='presenca_socket'&&$('tb-sockets')&&msg.data){$('tb-sockets').textContent=(msg.data.connections||0)+' conexoes';}
      if(msg.type==='whatsapp_status'){refreshSystemStatus();}
    }catch(ex){}
  };
  ws.onclose=function(){setTimeout(connectWS,2000);};
  ws.onerror=function(){ws.close();};
}

function initChats(){
  document.querySelectorAll('[data-chat-target]').forEach(function(el){
    var target=el.getAttribute('data-chat-target');
    CHATS[target]={
      target:target,
      messages:el.getAttribute('data-chat-messages')||'mensagem',
      relation:el.getAttribute('data-chat-relation')||target,
      textField:el.getAttribute('data-chat-text')||'corpo',
      mediaField:el.getAttribute('data-chat-media')||'media_url',
      authorField:el.getAttribute('data-chat-author')||'de_mim',
      timeField:el.getAttribute('data-chat-time')||'criado_em',
      typeField:el.getAttribute('data-chat-type')||'tipo',
      active:null,
      items:[]
    };
    loadChatConversations(target);
  });
}

function loadChatConversations(target){
  var c=CHATS[target]; if(!c) return;
  fetch('/api/'+target).then(function(r){return r.json();}).then(function(items){
    c.items=items||[];
    var box=$('chat-conv-'+target); if(!box) return;
    box.innerHTML='';
    if(!items||!items.length){ box.innerHTML='<div class="flex flex-col items-center justify-center py-12 text-gray-400"><p class="text-sm">Sem conversas</p></div>'; return; }
    items.forEach(function(item){
      var name=item.nome||item.titulo||item.numero||item.id;
      var last=item.ultima_mensagem||item.last_message||'';
      var unread=item.nao_lidas||0;
      var initials=String(name).trim().split(/\s+/).slice(0,2).map(function(part){return part.charAt(0).toUpperCase();}).join('');
      var div=document.createElement('div');
      div.className='chat-conv border-b border-gray-100 dark:border-gray-800 hover:bg-gray-50 dark:hover:bg-gray-800/50'+(String(c.active)===String(item.id)?' bg-primary/5':'');
      div.innerHTML='<div class="w-11 h-11 rounded-xl bg-gradient-to-br from-secondary to-primary flex items-center justify-center text-white font-bold text-sm">'+esc(initials||'#')+'</div><div class="min-w-0"><div class="font-semibold text-sm">'+esc(name)+'</div><div class="text-xs text-gray-500 mt-1 truncate">'+esc(last||'Sem mensagens')+'</div><div class="flex justify-between gap-2 mt-1 text-xs text-gray-400"><span>'+esc(item.status||'')+'</span><span>'+esc(item.numero||'')+'</span></div></div><div>'+(unread?('<span class="inline-flex items-center justify-center min-w-[24px] h-6 px-2 rounded-full bg-accent text-white text-xs font-bold">'+unread+'</span>'):'')+'</div>';
      div.onclick=function(){openChat(target,item.id,name);};
      box.appendChild(div);
    });
    if(!c.active&&items[0]) openChat(target,items[0].id,items[0].nome||items[0].titulo||items[0].numero||items[0].id);
  }).catch(function(){ var box=$('chat-conv-'+target); if(box) box.innerHTML='<div class="flex flex-col items-center justify-center py-12 text-gray-400"><p class="text-sm">Erro ao carregar conversas</p></div>'; });
}

function refreshChat(target){
  loadChatConversations(target);
  var c=CHATS[target];
  if(c&&c.active){ loadChatMessages(target,c.active); }
}

function openChat(target,id,title){
  var c=CHATS[target]; if(!c) return;
  c.active=id;
  $('chat-title-'+target).textContent=title||('Conversa #'+id);
  loadChatConversations(target);
  loadChatMessages(target,id);
}

function loadChatMessages(target,id){
  var c=CHATS[target]; if(!c) return;
  fetch('/api/'+c.messages+'?'+encodeURIComponent(c.relation)+'='+encodeURIComponent(id)).then(function(r){return r.json();}).then(function(items){
    var box=$('chat-msg-'+target); if(!box) return;
    box.innerHTML='';
    if(!items||!items.length){ box.innerHTML='<div class="flex flex-col items-center justify-center py-12 text-gray-400"><p class="text-sm">Sem mensagens</p></div>'; return; }
    items.forEach(function(item){
      var mine=!!item[c.authorField];
      var text=item[c.textField]||'';
      var media=item[c.mediaField]||'';
      var type=(item[c.typeField]||'').toLowerCase();
      var div=document.createElement('div');
      div.className='chat-bubble '+(mine?'mine bg-primary/10 dark:bg-primary/20 border border-primary/20':'other bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700');
      var html='';
      if(text) html+='<div class="text-sm">'+esc(text)+'</div>';
      if(media){ html+='<div class="chat-media mt-2">'+chatMediaHTML(media,type)+'</div>'; }
      html+='<div class="chat-bubble-meta text-gray-400">'+esc(item[c.timeField]||'')+'</div>';
      div.innerHTML=html;
      box.appendChild(div);
    });
    box.scrollTop=box.scrollHeight;
  });
}

function chatMediaHTML(path,type){
  var src='/media/stream?path='+encodeURIComponent(path);
  if((type||'').indexOf('audio')>=0||String(path).match(/\.(mp3|wav|ogg|m4a|aac)$/i)) return '<audio controls src="'+src+'"></audio>';
  if((type||'').indexOf('video')>=0||String(path).match(/\.(mp4|webm|mov|avi)$/i)) return '<video controls src="'+src+'"></video>';
  if(String(path).match(/\.(jpg|jpeg|png|gif|webp|svg)$/i)) return '<img src="'+esc(path)+'" alt="midia">';
  return '<a class="text-primary hover:underline" target="_blank" href="'+esc(path)+'">Abrir arquivo</a>';
}

function chatSend(target,e){
  e.preventDefault();
  var c=CHATS[target]; if(!c||!c.active) return;
  var input=$('chat-input-'+target); var text=input.value.trim();
  var payload={};
  payload[c.relation]=c.active;
  payload[c.textField]=text;
  payload[c.authorField]=true;
  payload[c.typeField]='chat';
  fetch('/api/'+c.messages,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(payload)})
    .then(function(r){if(!r.ok) throw new Error('Erro ao enviar'); return r.json();})
    .then(function(){ input.value=''; chatTyping(target,false); loadChatMessages(target,c.active); loadChatConversations(target); })
    .catch(function(err){ toast(err.message,'erro'); });
}

function chatUpload(target,input){
  var c=CHATS[target]; if(!c||!c.active||!input.files||!input.files[0]) return;
  var fd=new FormData(); fd.append('file', input.files[0]);
  fetch('/upload',{method:'POST',body:fd}).then(function(r){if(!r.ok) throw new Error('Upload falhou'); return r.json();}).then(function(d){
    var payload={};
    payload[c.relation]=c.active;
    payload[c.textField]=input.files[0].name;
    payload[c.mediaField]=d.path;
    payload[c.authorField]=true;
    payload[c.typeField]=input.files[0].type||'arquivo';
    return fetch('/api/'+c.messages,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(payload)});
  }).then(function(){ input.value=''; loadChatMessages(target,c.active); loadChatConversations(target); }).catch(function(err){ toast(err.message,'erro'); });
}

function chatFilter(target,q){
  q=(q||'').toLowerCase();
  document.querySelectorAll('#chat-conv-'+target+' .chat-conv').forEach(function(el){
    el.style.display=el.textContent.toLowerCase().includes(q)?'':'none';
  });
}

function chatTyping(target,typing){
  var c=CHATS[target]; if(!c||!c.active) return;
  fetch('/api/_presence',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({user:'local',screen:target,ticket_id:c.active,typing:typing,status:typing?'digitando':'online'})}).catch(function(){});
}

function chatHandlePresence(msg){
  var data=msg.data||{};
  var ticketId=String(data.ticket_id||'');
  Object.keys(CHATS).forEach(function(target){
    var c=CHATS[target];
    if(!c||String(c.active)!==ticketId) return;
    if(msg.type==='digitando'){
      $('chat-typing-'+target).textContent=(data.user||'Alguem')+' esta digitando...';
      setTimeout(function(){ if($('chat-typing-'+target).textContent.indexOf('digitando')>=0) $('chat-typing-'+target).textContent=''; }, 1800);
    }else if(msg.type==='presenca'){
      $('chat-presence-'+target).textContent=(data.user||'')+' '+(data.status||'');
    }else if(msg.type==='presenca_socket'){
      $('chat-presence-'+target).textContent='Conexoes ativas: '+((data&&data.connections)||0);
    }
  });
}

function chatHandleUpdate(msg){
  Object.keys(CHATS).forEach(function(target){
    var c=CHATS[target];
    if(!c) return;
    if(msg.model===c.target){ loadChatConversations(target); }
    if(msg.model===c.messages&&c.active){ loadChatMessages(target,c.active); }
  });
}

// ===== New field type helpers =====
function maskCPF(el){var v=el.value.replace(/\D/g,'');if(v.length<=11){v=v.replace(/(\d{3})(\d)/,'$1.$2');v=v.replace(/(\d{3})(\d)/,'$1.$2');v=v.replace(/(\d{3})(\d{1,2})$/,'$1-$2');}else{v=v.replace(/^(\d{2})(\d)/,'$1.$2');v=v.replace(/^(\d{2})\.(\d{3})(\d)/,'$1.$2.$3');v=v.replace(/\.(\d{3})(\d)/,'.$1/$2');v=v.replace(/(\d{4})(\d)/,'$1-$2');}el.value=v;}
function maskCEP(el){var v=el.value.replace(/\D/g,'');v=v.replace(/^(\d{5})(\d)/,'$1-$2');el.value=v;}
function buscaCEP(el){var cep=el.value.replace(/\D/g,'');if(cep.length!==8)return;fetch('https://viacep.com.br/ws/'+cep+'/json/').then(r=>r.json()).then(function(d){if(d.erro)return;var form=el.closest('form');if(!form)return;var inputs=form.querySelectorAll('input');inputs.forEach(function(inp){var n=inp.id.toLowerCase();if(n.includes('cidade')||n.includes('city'))inp.value=d.localidade;if(n.includes('estado')||n.includes('state')||n.includes('uf'))inp.value=d.uf;if(n.includes('rua')||n.includes('endereco')||n.includes('logradouro')||n.includes('street'))inp.value=d.logradouro;if(n.includes('bairro')||n.includes('neighborhood'))inp.value=d.bairro;});}).catch(function(){});}
function setStars(id,n){document.getElementById(id).value=n;var wrap=document.getElementById(id+'-wrap');if(!wrap)return;var btns=wrap.querySelectorAll('button');btns.forEach(function(b,i){b.style.color=i<n?'#eab308':'#4b5563';});}
function addTag(id,input){var val=input.value.trim();if(!val)return;input.value='';var hidden=document.getElementById(id);var container=document.getElementById(id+'-tags');var current=hidden.value?hidden.value.split(','):[];if(current.indexOf(val)>=0)return;current.push(val);hidden.value=current.join(',');var chip=document.createElement('span');chip.className='inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs bg-primary/10 text-primary';chip.innerHTML=val+'<button type="button" onclick="removeTag(\''+id+'\',\''+val+'\',this.parentElement)" class="ml-1 hover:text-red-400">&times;</button>';container.appendChild(chip);}
function removeTag(id,val,el){var hidden=document.getElementById(id);var current=hidden.value.split(',').filter(function(v){return v!==val;});hidden.value=current.join(',');el.remove();}

// ===== Init =====
document.addEventListener('DOMContentLoaded',function(){
  connectWS();
  renderCharts();
  initChats();
  carregarListasInline();
  refreshSystemStatus();
  setInterval(refreshSystemStatus,8000);
`)
	for _, model := range s.Program.Models {
		b.WriteString(fmt.Sprintf("  carregar('%s');\n", lo(model.Name)))
	}
	b.WriteString("});\n")

	// Auth JS
	if s.Auth != nil {
		loginField := "email"
		passField := "senha"
		if s.Program.Auth != nil {
			if s.Program.Auth.LoginField != "" {
				loginField = s.Program.Auth.LoginField
			}
			if s.Program.Auth.PassField != "" {
				passField = s.Program.Auth.PassField
			}
		}
		b.WriteString(fmt.Sprintf(`
// ===== Auth =====
var authToken=localStorage.getItem('germanio_token')||'';
var authMode='login';
var AUTH_LOGIN='%s',AUTH_PASS='%s';

function authHeaders(){
  var h={'Content-Type':'application/json'};
  if(authToken)h['Authorization']='Bearer '+authToken;
  return h;
}

// Override fetch to inject auth token
var _fetch=window.fetch;
window.fetch=function(url,opts){
  opts=opts||{};
  if(authToken&&url.startsWith('/api/')){
    opts.headers=opts.headers||{};
    if(typeof opts.headers==='object'&&!opts.headers['Authorization']){
      opts.headers['Authorization']='Bearer '+authToken;
    }
  }
  return _fetch(url,opts);
};

function mostrarLogin(){
  var modal=$('auth-modal');
  modal.classList.remove('hidden');
  modal.classList.add('flex');
  authMode='login';
  $('auth-title').textContent='Entrar';
  $('auth-toggle-text').textContent='Nao tem conta?';
  $('auth-toggle-link').textContent='Criar conta';
  $('auth-error').classList.add('hidden');
  $('auth-extra-fields').innerHTML='';
}
function fecharAuth(){
  var modal=$('auth-modal');
  modal.classList.add('hidden');
  modal.classList.remove('flex');
}
function toggleAuthMode(){
  if(authMode==='login'){
    authMode='register';
    $('auth-title').textContent='Criar Conta';
    $('auth-toggle-text').textContent='Ja tem conta?';
    $('auth-toggle-link').textContent='Entrar';
    $('auth-extra-fields').innerHTML='<div class="mb-4"><label class="block text-sm font-medium text-gray-500 dark:text-gray-400 mb-1.5">Nome</label><input type="text" id="auth-nome" required class="w-full bg-gray-100 dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-xl px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50" placeholder="Seu nome"></div>';
  } else {
    authMode='login';
    $('auth-title').textContent='Entrar';
    $('auth-toggle-text').textContent='Nao tem conta?';
    $('auth-toggle-link').textContent='Criar conta';
    $('auth-extra-fields').innerHTML='';
  }
  $('auth-error').classList.add('hidden');
}
function authSubmit(e){
  e.preventDefault();
  var login=$('auth-login').value;
  var pass=$('auth-pass').value;
  var url=authMode==='login'?'/api/login':'/api/registro';
  var body={};
  body[AUTH_LOGIN]=login;
  body[AUTH_PASS]=pass;
  if(authMode==='register'){
    var nome=$('auth-nome');
    if(nome)body['nome']=nome.value;
  }
  _fetch(url,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)})
    .then(function(r){return r.json().then(function(d){return{ok:r.ok,data:d};});})
    .then(function(res){
      if(!res.ok){
        $('auth-error').textContent=res.data.erro||'Erro';
        $('auth-error').classList.remove('hidden');
        return;
      }
      authToken=res.data.token;
      localStorage.setItem('germanio_token',authToken);
      fecharAuth();
      updateAuthUI(res.data.login||login,res.data.role||'usuario');
      toast(authMode==='login'?'Bem-vindo!':'Conta criada!');
      // Reload all data with auth
      Object.keys(M).forEach(function(m){carregar(m);});
      carregarListasInline();
      renderCharts();
    });
}
function sair(){
  authToken='';
  localStorage.removeItem('germanio_token');
  $('btn-login').style.display='';
  $('btn-login').classList.remove('hidden');
  $('user-info').classList.add('hidden');
  $('btn-logout').classList.add('hidden');
  toast('Desconectado');
}
function updateAuthUI(login,role){
  $('btn-login').classList.add('hidden');
  $('user-info').classList.remove('hidden');
  $('user-info').textContent=login+' ('+role+')';
  $('btn-logout').classList.remove('hidden');
}
// Check stored token on load
if(authToken){
  _fetch('/api/me',{headers:{'Authorization':'Bearer '+authToken}})
    .then(function(r){return r.json();})
    .then(function(d){if(d&&d.login)updateAuthUI(d.login,d.role||'usuario');else sair();})
    .catch(function(){sair();});
}
`, loginField, passField))
	}

	return b.String()
}

// ============================================================
// Helpers
// ============================================================

// fieldTypeCode returns a short code for JS metadata.
func fieldTypeCode(f *ast.Field) string {
	switch f.Type {
	case ast.FieldNumero:
		return "n"
	case ast.FieldDinheiro:
		return "d"
	case ast.FieldStatus:
		return "s"
	case ast.FieldEmail:
		return "e"
	case ast.FieldTextoLongo:
		return "tl"
	case ast.FieldImagem, ast.FieldUpload, ast.FieldArquivo:
		return "f"
	case ast.FieldEnum:
		return "en"
	case ast.FieldBooleano:
		return "b"
	case ast.FieldSenha:
		return "pw"
	case ast.FieldCPF:
		return "cpf"
	case ast.FieldCEP:
		return "cep"
	case ast.FieldCor:
		return "cor"
	case ast.FieldEstrelas:
		return "star"
	case ast.FieldHora:
		return "hr"
	case ast.FieldDataHora:
		return "dh"
	case ast.FieldPercentual:
		return "pct"
	case ast.FieldTags:
		return "tag"
	case ast.FieldURL:
		return "url"
	case ast.FieldMoeda:
		return "cur"
	default:
		return "t"
	}
}

// enumJSArray builds a JS array literal from enum values.
func enumJSArray(vals []string) string {
	var parts []string
	for _, v := range vals {
		parts = append(parts, fmt.Sprintf("'%s'", v))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// ============================================================
// SVG Icons
// ============================================================

func svgIcon(name string) string {
	// All SVGs use w-full h-full so they inherit size from parent container
	const svgAttrs = `viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="w-full h-full"`
	icons := map[string]string{
		"zap":      `<svg ` + svgAttrs + `><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>`,
		"layout":   `<svg ` + svgAttrs + `><rect x="3" y="3" width="18" height="18" rx="2"/><line x1="3" y1="9" x2="21" y2="9"/><line x1="9" y1="21" x2="9" y2="9"/></svg>`,
		"menu":     `<svg ` + svgAttrs + `><line x1="4" y1="12" x2="20" y2="12"/><line x1="4" y1="6" x2="20" y2="6"/><line x1="4" y1="18" x2="20" y2="18"/></svg>`,
		"search":   `<svg ` + svgAttrs + `><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>`,
		"plus":     `<svg ` + svgAttrs + `><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>`,
		"edit":     `<svg ` + svgAttrs + `><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.12 2.12 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>`,
		"trash":    `<svg ` + svgAttrs + `><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>`,
		"x":        `<svg ` + svgAttrs + `><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>`,
		"check":    `<svg ` + svgAttrs + `><polyline points="20 6 9 17 4 12"/></svg>`,
		"moon":     `<svg ` + svgAttrs + `><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>`,
		"chevleft": `<svg ` + svgAttrs + `><polyline points="15 18 9 12 15 6"/></svg>`,
		"activity": `<svg ` + svgAttrs + `><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>`,
		"info":     `<svg ` + svgAttrs + `><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>`,
		"inbox":    `<svg ` + svgAttrs + `><polyline points="22 12 16 12 14 15 10 15 8 12 2 12"/><path d="M5.45 5.11L2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z"/></svg>`,
		"box":      `<svg ` + svgAttrs + `><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/><polyline points="3.27 6.96 12 12.01 20.73 6.96"/><line x1="12" y1="22.08" x2="12" y2="12"/></svg>`,
		"users":    `<svg ` + svgAttrs + `><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>`,
		"user":     `<svg ` + svgAttrs + `><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>`,
		"grid":     `<svg ` + svgAttrs + `><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/></svg>`,
		"list":     `<svg ` + svgAttrs + `><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>`,
		"clip":     `<svg ` + svgAttrs + `><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"/><rect x="8" y="2" width="8" height="4" rx="1"/></svg>`,
		"dollar":   `<svg ` + svgAttrs + `><line x1="12" y1="1" x2="12" y2="23"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>`,
		"utensils": `<svg ` + svgAttrs + `><path d="M3 2v7c0 1.1.9 2 2 2h4a2 2 0 0 0 2-2V2"/><path d="M7 2v20"/><path d="M21 15V2a5 5 0 0 0-5 5v6c0 1.1.9 2 2 2h3zm0 0v7"/></svg>`,
		"tag":      `<svg ` + svgAttrs + `><path d="M20.59 13.41l-7.17 7.17a2 2 0 0 1-2.83 0L2 12V2h10l8.59 8.59a2 2 0 0 1 0 2.82z"/><line x1="7" y1="7" x2="7.01" y2="7"/></svg>`,
		"file":     `<svg ` + svgAttrs + `><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>`,
		"settings": `<svg ` + svgAttrs + `><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09a1.65 1.65 0 0 0-1.08-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09a1.65 1.65 0 0 0 1.51-1.08 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9c.26.604.852.997 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>`,
	}
	if svg, ok := icons[name]; ok {
		return svg
	}
	return icons["box"]
}

func modelIcon(name string) string {
	m := map[string]string{
		"produto": "box", "produtos": "box", "prato": "utensils", "pratos": "utensils",
		"cliente": "user", "clientes": "users", "usuario": "user", "usuarios": "users",
		"funcionario": "users", "funcionarios": "users", "equipe": "users",
		"mesa": "grid", "mesas": "grid", "pedido": "clip", "pedidos": "clip",
		"venda": "dollar", "vendas": "dollar", "pagamento": "dollar",
		"categoria": "tag", "categorias": "tag", "item": "list", "itens": "list",
		"configuracao": "settings", "config": "settings", "arquivo": "file",
	}
	if icon, ok := m[name]; ok {
		return icon
	}
	return "box"
}

func tipoInput(ft ast.FieldType) string {
	switch ft {
	case ast.FieldEmail:
		return "email"
	case ast.FieldTelefone:
		return "tel"
	case ast.FieldNumero, ast.FieldDinheiro:
		return "number"
	case ast.FieldData:
		return "date"
	case ast.FieldBooleano:
		return "checkbox"
	case ast.FieldImagem, ast.FieldUpload, ast.FieldArquivo:
		return "file"
	case ast.FieldLink:
		return "url"
	case ast.FieldSenha:
		return "password"
	default:
		return "text"
	}
}

func cap(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func lo(s string) string { return strings.ToLower(s) }
