package ast

import (
	"testing"
)

func TestColorNameMap(t *testing.T) {
	expectedColors := []struct {
		name string
		hex  string
	}{
		// Portuguese
		{"azul", "#3b82f6"},
		{"verde", "#22c55e"},
		{"vermelho", "#ef4444"},
		{"roxo", "#8b5cf6"},
		{"laranja", "#f97316"},
		{"rosa", "#ec4899"},
		{"amarelo", "#eab308"},
		{"branco", "#ffffff"},
		{"preto", "#000000"},
		// English
		{"blue", "#3b82f6"},
		{"green", "#22c55e"},
		{"red", "#ef4444"},
		{"purple", "#8b5cf6"},
		{"orange", "#f97316"},
		{"pink", "#ec4899"},
		{"yellow", "#eab308"},
		{"white", "#ffffff"},
		{"black", "#000000"},
	}

	for _, c := range expectedColors {
		t.Run(c.name, func(t *testing.T) {
			hex, ok := ColorName[c.name]
			if !ok {
				t.Errorf("color %q not found in ColorName map", c.name)
				return
			}
			if hex != c.hex {
				t.Errorf("color %q: expected %q, got %q", c.name, c.hex, hex)
			}
		})
	}
}

func TestResolveColorKnownNames(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"azul", "#3b82f6"},
		{"blue", "#3b82f6"},
		{"vermelho", "#ef4444"},
		{"red", "#ef4444"},
		{"verde", "#22c55e"},
		{"green", "#22c55e"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ResolveColor(tt.input)
			if result != tt.expected {
				t.Errorf("ResolveColor(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestResolveColorHexPassthrough(t *testing.T) {
	hexValues := []string{"#ff0000", "#00ff00", "#123abc", "#ffffff"}
	for _, hex := range hexValues {
		result := ResolveColor(hex)
		if result != hex {
			t.Errorf("ResolveColor(%q) = %q, want %q (passthrough)", hex, result, hex)
		}
	}
}

func TestThemePresets(t *testing.T) {
	tests := []struct {
		name    string
		dark    bool
		style   string
		primary string
	}{
		{"moderno", true, "glassmorphism", "#6366f1"},
		{"modern", true, "glassmorphism", "#6366f1"},
		{"claro", false, "flat", "#3b82f6"},
		{"light", false, "flat", "#3b82f6"},
		{"simples", false, "minimal", "#2563eb"},
		{"simple", false, "minimal", "#2563eb"},
		{"elegante", true, "neumorphism", "#7c3aed"},
		{"elegant", true, "neumorphism", "#7c3aed"},
		{"corporativo", false, "flat", "#0f766e"},
		{"corporate", false, "flat", "#0f766e"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			theme := ThemePreset(tt.name)
			if theme.Dark != tt.dark {
				t.Errorf("preset %q: dark=%v, want %v", tt.name, theme.Dark, tt.dark)
			}
			if theme.Style != tt.style {
				t.Errorf("preset %q: style=%q, want %q", tt.name, theme.Style, tt.style)
			}
			if theme.Primary != tt.primary {
				t.Errorf("preset %q: primary=%q, want %q", tt.name, theme.Primary, tt.primary)
			}
		})
	}
}

func TestThemePresetUnknownReturnsDefault(t *testing.T) {
	theme := ThemePreset("nonexistent_preset")
	def := DefaultTheme()
	if theme.Primary != def.Primary {
		t.Errorf("unknown preset: expected default primary %q, got %q", def.Primary, theme.Primary)
	}
	if theme.Font != def.Font {
		t.Errorf("unknown preset: expected default font %q, got %q", def.Font, theme.Font)
	}
}

func TestDefaultTheme(t *testing.T) {
	theme := DefaultTheme()
	if theme.Primary == "" {
		t.Error("default theme primary is empty")
	}
	if theme.Font == "" {
		t.Error("default theme font is empty")
	}
	if theme.Radius == "" {
		t.Error("default theme radius is empty")
	}
	if theme.Style == "" {
		t.Error("default theme style is empty")
	}
	if theme.Background == "" {
		t.Error("default theme background is empty")
	}
	if theme.TextColor == "" {
		t.Error("default theme text color is empty")
	}
}

func TestProgramMerge(t *testing.T) {
	p1 := &Program{
		System: &System{Name: "app1"},
		Models: []*Model{{Name: "cliente"}},
	}
	p2 := &Program{
		Theme:     &Theme{Primary: "#ff0000"},
		Models:    []*Model{{Name: "produto"}},
		Screens:   []*Screen{{Name: "tela1"}},
		Functions: []*FuncDecl{{Name: "fn1"}},
		Events:    []*Event{{Trigger: "click"}},
	}

	p1.Merge(p2)

	// Theme should be adopted from p2 since p1 had none
	if p1.Theme == nil || p1.Theme.Primary != "#ff0000" {
		t.Error("merge should adopt theme from other when nil")
	}

	// Models should be concatenated
	if len(p1.Models) != 2 {
		t.Errorf("expected 2 models after merge, got %d", len(p1.Models))
	}
	if p1.Models[0].Name != "cliente" || p1.Models[1].Name != "produto" {
		t.Error("models not merged correctly")
	}

	// Screens should be concatenated
	if len(p1.Screens) != 1 || p1.Screens[0].Name != "tela1" {
		t.Error("screens not merged correctly")
	}

	// Functions should be concatenated
	if len(p1.Functions) != 1 || p1.Functions[0].Name != "fn1" {
		t.Error("functions not merged correctly")
	}

	// Events should be concatenated
	if len(p1.Events) != 1 || p1.Events[0].Trigger != "click" {
		t.Error("events not merged correctly")
	}
}

func TestProgramMergeDoesNotOverrideExistingTheme(t *testing.T) {
	p1 := &Program{
		Theme: &Theme{Primary: "#0000ff"},
	}
	p2 := &Program{
		Theme: &Theme{Primary: "#ff0000"},
	}

	p1.Merge(p2)

	// p1 already had a theme, so it should NOT be overridden
	if p1.Theme.Primary != "#0000ff" {
		t.Errorf("merge should not override existing theme; got primary=%q", p1.Theme.Primary)
	}
}

func TestFieldTypeSQLType(t *testing.T) {
	tests := []struct {
		ft       FieldType
		expected string
	}{
		{FieldNumero, "REAL"},
		{FieldDinheiro, "REAL"},
		{FieldBooleano, "INTEGER"},
		{FieldData, "DATE"},
		{FieldTexto, "TEXT"},
		{FieldEmail, "TEXT"},
		{FieldSenha, "TEXT"},
		{FieldImagem, "TEXT"},
		{FieldEnum, "TEXT"},
	}

	for _, tt := range tests {
		t.Run(string(tt.ft), func(t *testing.T) {
			result := tt.ft.SQLType()
			if result != tt.expected {
				t.Errorf("FieldType(%q).SQLType() = %q, want %q", tt.ft, result, tt.expected)
			}
		})
	}
}

func TestNodeTypes(t *testing.T) {
	// Verify NodeType() returns expected strings for various AST nodes
	nodes := []struct {
		node     Node
		expected string
	}{
		{&Program{}, "Program"},
		{&System{Name: "test"}, "System"},
		{&Import{}, "Import"},
		{&Theme{}, "Theme"},
		{&Model{Name: "m"}, "Model"},
		{&Field{Name: "f"}, "Field"},
		{&Screen{Name: "s"}, "Screen"},
		{&Event{}, "Event"},
		{&Action{}, "Action"},
		{&Expression{}, "Expression"},
		{&VarDecl{}, "VarDecl"},
		{&FuncDecl{}, "FuncDecl"},
		{&Statement{}, "Statement"},
	}

	for _, tt := range nodes {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.node.NodeType(); got != tt.expected {
				t.Errorf("NodeType() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// Phrases about the login may live in different files: merging keeps all of
// them (sign-up and password recovery included).
func TestMergeIntentKeepsLoginPhrases(t *testing.T) {
	a := &Intent{Login: &LoginDecl{Fields: []string{"email"}}}
	b := &Intent{Login: &LoginDecl{Signup: true, Recovery: true}, PendingItems: []*PendingRule{{Entity: "issues", Fields: []string{"responsaveis"}}}}
	m := MergeIntent(a, b)
	if !m.Login.Signup || !m.Login.Recovery || len(m.Login.Fields) != 1 || len(m.PendingItems) != 1 {
		t.Fatalf("fusão perdeu frases: %+v %+v", m.Login, m.PendingItems)
	}
}
