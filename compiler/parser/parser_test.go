package parser

import (
	"testing"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/lexer"
)

func parse(t *testing.T, source string) *ast.Program {
	t.Helper()
	lex := lexer.New(source)
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatal(err)
	}
	p := New(tokens)
	prog, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	return prog
}

func TestParseSystem(t *testing.T) {
	prog := parse(t, "sistema meuapp")
	if prog.System == nil {
		t.Fatal("system is nil")
	}
	if prog.System.Name != "meuapp" {
		t.Errorf("expected 'meuapp', got %q", prog.System.Name)
	}
}

func TestParseSystemEnglish(t *testing.T) {
	prog := parse(t, "system myapp")
	if prog.System == nil {
		t.Fatal("system is nil")
	}
	if prog.System.Name != "myapp" {
		t.Errorf("expected 'myapp', got %q", prog.System.Name)
	}
}

func TestParseModelWithFields(t *testing.T) {
	source := `dados
  produto
    nome: texto obrigatorio
    preco: dinheiro
    email_contato: email unico
`
	prog := parse(t, source)
	if len(prog.Models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(prog.Models))
	}
	m := prog.Models[0]
	if m.Name != "produto" {
		t.Errorf("expected model name 'produto', got %q", m.Name)
	}
	if len(m.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(m.Fields))
	}

	// Check first field
	if m.Fields[0].Name != "nome" {
		t.Errorf("field 0: expected 'nome', got %q", m.Fields[0].Name)
	}
	if m.Fields[0].Type != ast.FieldTexto {
		t.Errorf("field 0: expected type 'texto', got %q", m.Fields[0].Type)
	}
	if !m.Fields[0].Required {
		t.Error("field 0: expected required=true")
	}

	// Check preco field
	if m.Fields[1].Name != "preco" {
		t.Errorf("field 1: expected 'preco', got %q", m.Fields[1].Name)
	}
	if m.Fields[1].Type != ast.FieldDinheiro {
		t.Errorf("field 1: expected type 'dinheiro', got %q", m.Fields[1].Type)
	}

	// Check unique modifier
	if m.Fields[2].Name != "email_contato" {
		t.Errorf("field 2: expected 'email_contato', got %q", m.Fields[2].Name)
	}
	if !m.Fields[2].Unique {
		t.Error("field 2: expected unique=true")
	}
}

func TestParseModelWithBelongsTo(t *testing.T) {
	source := `dados
  pedido
    cliente_id: numero pertence_a cliente
    valor: dinheiro
`
	prog := parse(t, source)
	if len(prog.Models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(prog.Models))
	}
	f := prog.Models[0].Fields[0]
	if f.Reference != "cliente" {
		t.Errorf("expected reference 'cliente', got %q", f.Reference)
	}
}

func TestParseModelRelationships(t *testing.T) {
	source := `dados
  cliente
    nome: texto
    tem_muitos pedido
    muitos_para_muitos produto
`
	prog := parse(t, source)
	if len(prog.Models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(prog.Models))
	}
	m := prog.Models[0]
	if len(m.HasMany) != 1 || m.HasMany[0] != "pedido" {
		t.Errorf("expected has_many=['pedido'], got %v", m.HasMany)
	}
	if len(m.ManyToMany) != 1 || m.ManyToMany[0] != "produto" {
		t.Errorf("expected many_to_many=['produto'], got %v", m.ManyToMany)
	}
}

func TestParseEnumField(t *testing.T) {
	source := `dados
  pedido
    status: enum(ativo, inativo, pendente)
`
	prog := parse(t, source)
	if len(prog.Models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(prog.Models))
	}
	f := prog.Models[0].Fields[0]
	if f.Type != ast.FieldEnum {
		t.Errorf("expected enum type, got %q", f.Type)
	}
	if len(f.EnumValues) != 3 {
		t.Fatalf("expected 3 enum values, got %d: %v", len(f.EnumValues), f.EnumValues)
	}
	expected := []string{"ativo", "inativo", "pendente"}
	for i, exp := range expected {
		if f.EnumValues[i] != exp {
			t.Errorf("enum value %d: expected %q, got %q", i, exp, f.EnumValues[i])
		}
	}
}

func TestParseScreen(t *testing.T) {
	source := `telas
  tela produtos
    titulo "Produtos"
    lista produto
    botao "Novo Produto"
`
	prog := parse(t, source)
	if len(prog.Screens) != 1 {
		t.Fatalf("expected 1 screen, got %d", len(prog.Screens))
	}
	s := prog.Screens[0]
	if s.Name != "produtos" {
		t.Errorf("expected screen name 'produtos', got %q", s.Name)
	}
}

func TestParseChatScreenComponent(t *testing.T) {
	source := `telas
  tela atendimento
    titulo "Chat"
    chat ticket
      mensagens mensagem
      relacao ticket
      texto corpo
`
	prog := parse(t, source)
	if len(prog.Screens) != 1 {
		t.Fatalf("expected 1 screen, got %d", len(prog.Screens))
	}
	if len(prog.Screens[0].Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(prog.Screens[0].Components))
	}
	comp := prog.Screens[0].Components[0]
	if comp.Type != ast.CompChat {
		t.Fatalf("expected chat component, got %q", comp.Type)
	}
	if comp.Target != "ticket" {
		t.Fatalf("expected target ticket, got %q", comp.Target)
	}
	if comp.Properties["messages_model"] != "mensagem" {
		t.Fatalf("expected messages_model mensagem, got %q", comp.Properties["messages_model"])
	}
}

func TestParseEvents(t *testing.T) {
	source := `eventos
  quando clicar "salvar"
    criar produto
`
	prog := parse(t, source)
	if len(prog.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(prog.Events))
	}
	ev := prog.Events[0]
	if ev.Trigger != "clicar" {
		t.Errorf("expected trigger 'clicar', got %q", ev.Trigger)
	}
	if ev.Target != "salvar" {
		t.Errorf("expected target 'salvar', got %q", ev.Target)
	}
}

func TestParseThemePreset(t *testing.T) {
	source := `tema moderno escuro`
	prog := parse(t, source)
	if prog.Theme == nil {
		t.Fatal("theme is nil")
	}
	if !prog.Theme.Dark {
		t.Error("expected dark=true for 'moderno escuro'")
	}
	if prog.Theme.Primary != "#6366f1" {
		t.Errorf("expected primary '#6366f1', got %q", prog.Theme.Primary)
	}
}

func TestParseThemeWithProperties(t *testing.T) {
	source := `tema
  escuro
  fonte "Roboto"
`
	prog := parse(t, source)
	if prog.Theme == nil {
		t.Fatal("theme is nil")
	}
	if !prog.Theme.Dark {
		t.Error("expected dark=true")
	}
	if prog.Theme.Font != "Roboto" {
		t.Errorf("expected font 'Roboto', got %q", prog.Theme.Font)
	}
}

func TestParseWhatsAppConfig(t *testing.T) {
	source := `integracoes
  whatsapp
    provedor: whatsmeow
    multi_sessao: verdadeiro
    presenca: verdadeiro
    qr_code: verdadeiro
`
	prog := parse(t, source)
	if prog.WhatsApp == nil {
		t.Fatal("expected whatsapp config")
	}
	if prog.WhatsApp.Provider != "whatsmeow" {
		t.Fatalf("expected provider whatsmeow, got %q", prog.WhatsApp.Provider)
	}
	if !prog.WhatsApp.MultiSession || !prog.WhatsApp.Presence || !prog.WhatsApp.QRCodeFlow {
		t.Fatal("expected whatsapp flags to be true")
	}
}

func TestParseImport(t *testing.T) {
	source := `importar "models.ge"`
	prog := parse(t, source)
	if len(prog.Imports) != 1 {
		t.Fatalf("expected 1 import, got %d", len(prog.Imports))
	}
	imp := prog.Imports[0]
	if imp.What != "tudo" {
		t.Errorf("expected what='tudo', got %q", imp.What)
	}
	if imp.Path != "models.ge" {
		t.Errorf("expected path='models.ge', got %q", imp.Path)
	}
}

func TestParseImportSpecific(t *testing.T) {
	source := `importar dados de "models.ge"`
	prog := parse(t, source)
	if len(prog.Imports) != 1 {
		t.Fatalf("expected 1 import, got %d", len(prog.Imports))
	}
	imp := prog.Imports[0]
	if imp.What != "dados" {
		t.Errorf("expected what='dados', got %q", imp.What)
	}
	if imp.Path != "models.ge" {
		t.Errorf("expected path='models.ge', got %q", imp.Path)
	}
}

func TestParseLogicWithFunction(t *testing.T) {
	source := `logica
  funcao somar(a, b)
    retornar a + b
`
	prog := parse(t, source)
	if len(prog.Functions) != 1 {
		t.Fatalf("expected 1 function, got %d", len(prog.Functions))
	}
	fn := prog.Functions[0]
	if fn.Name != "somar" {
		t.Errorf("expected function name 'somar', got %q", fn.Name)
	}
	if len(fn.Params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(fn.Params))
	}
	if fn.Params[0] != "a" || fn.Params[1] != "b" {
		t.Errorf("expected params [a, b], got %v", fn.Params)
	}
}

func TestParseLogicWithVariable(t *testing.T) {
	source := `logica
  definir x = 10
`
	prog := parse(t, source)
	if len(prog.Scripts) != 1 {
		t.Fatalf("expected 1 script statement, got %d", len(prog.Scripts))
	}
	stmt := prog.Scripts[0]
	if stmt.Type != "var" {
		t.Errorf("expected statement type 'var', got %q", stmt.Type)
	}
	if stmt.VarDecl.Name != "x" {
		t.Errorf("expected var name 'x', got %q", stmt.VarDecl.Name)
	}
}

func TestParseMultipleModels(t *testing.T) {
	source := `dados
  cliente
    nome: texto obrigatorio
    email: email unico

  produto
    titulo: texto
    preco: dinheiro
`
	prog := parse(t, source)
	if len(prog.Models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(prog.Models))
	}
	if prog.Models[0].Name != "cliente" {
		t.Errorf("model 0: expected 'cliente', got %q", prog.Models[0].Name)
	}
	if prog.Models[1].Name != "produto" {
		t.Errorf("model 1: expected 'produto', got %q", prog.Models[1].Name)
	}
}

func TestParseFullProgram(t *testing.T) {
	source := `sistema loja

dados
  produto
    nome: texto obrigatorio
    preco: dinheiro

telas
  tela produtos
    titulo "Produtos"
    lista produto

eventos
  quando clicar "salvar"
    criar produto

tema moderno
`
	prog := parse(t, source)
	if prog.System == nil || prog.System.Name != "loja" {
		t.Error("system not parsed correctly")
	}
	if len(prog.Models) != 1 {
		t.Errorf("expected 1 model, got %d", len(prog.Models))
	}
	if len(prog.Screens) != 1 {
		t.Errorf("expected 1 screen, got %d", len(prog.Screens))
	}
	if len(prog.Events) != 1 {
		t.Errorf("expected 1 event, got %d", len(prog.Events))
	}
	if prog.Theme == nil {
		t.Error("theme not parsed")
	}
}
