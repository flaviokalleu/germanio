package parser

import (
	"fmt"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/lexer"
)

// Parser converts a stream of tokens into an AST.
type Parser struct {
	tokens  []lexer.Token
	pos     int
	program *ast.Program
}

// New creates a new Parser for the given tokens.
func New(tokens []lexer.Token) *Parser {
	return &Parser{
		tokens:  tokens,
		pos:     0,
		program: &ast.Program{},
	}
}

// Parse processes all tokens and returns the AST.
func (p *Parser) Parse() (*ast.Program, error) {
	p.skipWhitespace()

	for !p.isAtEnd() {
		tok := p.current()
		switch tok.Type {
		case lexer.TokenImportar:
			if err := p.parseImportar(); err != nil {
				return nil, err
			}
		case lexer.TokenSistema:
			if err := p.parseSistema(); err != nil {
				return nil, err
			}
		case lexer.TokenDados:
			if err := p.parseDados(); err != nil {
				return nil, err
			}
		case lexer.TokenTelas:
			if err := p.parseTelas(); err != nil {
				return nil, err
			}
		case lexer.TokenEventos:
			if err := p.parseEventos(); err != nil {
				return nil, err
			}
		case lexer.TokenAcoes:
			if err := p.parseAcoes(); err != nil {
				return nil, err
			}
		case lexer.TokenTema:
			if err := p.parseTema(); err != nil {
				return nil, err
			}
		case lexer.TokenLogica:
			if err := p.parseLogica(); err != nil {
				return nil, err
			}
		case lexer.TokenBanco:
			if err := p.parseBanco(); err != nil {
				return nil, err
			}
		case lexer.TokenAutenticacao:
			if err := p.parseAuth(); err != nil {
				return nil, err
			}
		case lexer.TokenIntegracoes:
			if err := p.parseIntegracoes(); err != nil {
				return nil, err
			}
		default:
			handled := false
			if tok.Type == lexer.TokenIdentifier {
				switch tok.Value {
				case "rotas", "routes":
					if err := p.parseRotas(); err != nil {
						return nil, err
					}
					handled = true
				case "paginas", "pages":
					if err := p.parsePaginas(); err != nil {
						return nil, err
					}
					handled = true
				case "sidebar", "menu":
					if err := p.parseSidebar(); err != nil {
						return nil, err
					}
					handled = true
				}
			}
			if !handled {
				p.advance()
			}
		}
		p.skipWhitespace()
	}

	return p.program, nil
}

func (p *Parser) current() lexer.Token {
	if p.pos >= len(p.tokens) {
		return lexer.Token{Type: lexer.TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() lexer.Token {
	tok := p.current()
	p.pos++
	return tok
}

func (p *Parser) expect(tt lexer.TokenType) (lexer.Token, error) {
	tok := p.current()
	if tok.Type != tt {
		return tok, fmt.Errorf("line %d: expected token type %d, got %d (%q)", tok.Line, tt, tok.Type, tok.Value)
	}
	p.advance()
	return tok, nil
}

func (p *Parser) isAtEnd() bool {
	return p.pos >= len(p.tokens) || p.tokens[p.pos].Type == lexer.TokenEOF
}

func (p *Parser) skipNewlines() {
	for !p.isAtEnd() && p.current().Type == lexer.TokenNewline {
		p.advance()
	}
}

func (p *Parser) skipWhitespace() {
	for !p.isAtEnd() {
		tt := p.current().Type
		if tt == lexer.TokenNewline || tt == lexer.TokenIndent {
			p.advance()
		} else {
			break
		}
	}
}

func (p *Parser) skipToNextLine() {
	for !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
		p.advance()
	}
	p.skipNewlines()
}

func (p *Parser) isBlockKeyword() bool {
	tt := p.current().Type
	if tt == lexer.TokenImportar {
		return true // imports break blocks
	}
	return lexer.IsBlockKeyword(tt)
}

// parseSistema parses: sistema <name>
func (p *Parser) parseSistema() error {
	p.advance() // consume 'sistema'
	p.skipIndent()

	name := p.advance()
	if name.Type != lexer.TokenIdentifier && name.Type != lexer.TokenString {
		return fmt.Errorf("line %d: expected system name after 'sistema'", name.Line)
	}

	p.program.System = &ast.System{Name: name.Value}
	p.skipToNextLine()
	return nil
}

func (p *Parser) skipIndent() {
	for !p.isAtEnd() && p.current().Type == lexer.TokenIndent {
		p.advance()
	}
}

// isNameToken returns true if the token can be used as an identifier/name
// (includes actual identifiers and type keywords that may be used as field/model names).
func (p *Parser) isNameToken(tok lexer.Token) bool {
	// A name token is anything that's not whitespace, punctuation, or a block keyword
	switch tok.Type {
	case lexer.TokenEOF, lexer.TokenNewline, lexer.TokenIndent,
		lexer.TokenColon, lexer.TokenDot, lexer.TokenComma,
		lexer.TokenString, lexer.TokenNumber,
		lexer.TokenEquals, lexer.TokenPlus, lexer.TokenMinus,
		lexer.TokenStar, lexer.TokenSlash, lexer.TokenLParen, lexer.TokenRParen:
		return false
	}
	if lexer.IsBlockKeyword(tok.Type) {
		return false
	}
	return true
}

// parseDados parses the data models block.
func (p *Parser) parseDados() error {
	p.advance() // consume 'dados'
	p.skipWhitespace()

	for !p.isAtEnd() && !p.isBlockKeyword() {
		tok := p.current()
		// A model name is any name token where the next meaningful token is NOT a colon
		// (if it were a colon, it would be a field definition, not a model)
		if p.isNameToken(tok) && p.peekNextMeaningful().Type != lexer.TokenColon {
			model, err := p.parseModel()
			if err != nil {
				return err
			}
			p.program.Models = append(p.program.Models, model)
		} else {
			p.advance()
		}
	}
	return nil
}

// parseModel parses a single model definition.
func (p *Parser) parseModel() (*ast.Model, error) {
	nameTok := p.advance()
	model := &ast.Model{Name: nameTok.Value}

	// Check for modifiers after model name on the same line (e.g. soft_delete)
	for !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
		if p.current().Type == lexer.TokenSoftDelete {
			model.SoftDelete = true
			p.advance()
		} else if p.current().Type == lexer.TokenIndent {
			p.advance()
		} else {
			break
		}
	}

	p.skipWhitespace()

	for !p.isAtEnd() && !p.isBlockKeyword() {
		tok := p.current()

		if tok.Type == lexer.TokenIndent {
			p.advance()
			continue
		}
		if tok.Type == lexer.TokenNewline {
			p.advance()
			continue
		}

		// Relationship keywords at model level
		if tok.Type == lexer.TokenTemMuitos {
			p.advance()
			p.skipIndent()
			if !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
				model.HasMany = append(model.HasMany, p.advance().Value)
			}
			continue
		}
		if tok.Type == lexer.TokenMuitosParaMuitos {
			p.advance()
			p.skipIndent()
			if !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
				model.ManyToMany = append(model.ManyToMany, p.advance().Value)
			}
			continue
		}

		// Any name token: check if it's a field (followed by colon) or a new model
		if p.isNameToken(tok) {
			if p.peekNextMeaningful().Type == lexer.TokenColon {
				// It's a field definition
				field, err := p.parseField()
				if err != nil {
					return nil, err
				}
				if field != nil {
					model.Fields = append(model.Fields, field)
				}
			} else {
				// It's a new model name — stop
				break
			}
		} else {
			p.advance()
		}
	}

	return model, nil
}

func (p *Parser) peekNextMeaningful() lexer.Token {
	saved := p.pos
	p.pos++
	for p.pos < len(p.tokens) {
		tok := p.tokens[p.pos]
		if tok.Type != lexer.TokenNewline && tok.Type != lexer.TokenIndent {
			p.pos = saved
			return tok
		}
		p.pos++
	}
	p.pos = saved
	return lexer.Token{Type: lexer.TokenEOF}
}

// parseField parses: <name>: <type>
func (p *Parser) parseField() (*ast.Field, error) {
	nameTok := p.advance() // consume field name (could be identifier or type keyword used as name)

	// Expect colon
	if p.current().Type != lexer.TokenColon {
		// Not a field definition
		p.pos--
		return nil, nil
	}
	p.advance() // consume ':'

	// Skip any indent tokens after colon
	p.skipIndent()

	typeTok := p.advance()
	fieldType, err := tokenToFieldType(typeTok)
	if err != nil {
		return nil, fmt.Errorf("line %d: %w", typeTok.Line, err)
	}

	field := &ast.Field{
		Name: nameTok.Value,
		Type: fieldType,
	}

	// Check for modifiers on the same line
	for !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
		switch p.current().Type {
		case lexer.TokenObrigatorio:
			field.Required = true
			p.advance()
		case lexer.TokenUnico:
			field.Unique = true
			p.advance()
		case lexer.TokenIndice:
			field.Index = true
			p.advance()
		case lexer.TokenPertenceA:
			p.advance()
			p.skipIndent()
			if !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
				field.Reference = p.advance().Value
			}
		case lexer.TokenLParen:
			// Parse enum values: enum(ativo, inativo, pendente)
			p.advance() // consume '('
			for !p.isAtEnd() && p.current().Type != lexer.TokenRParen {
				if p.current().Type == lexer.TokenComma || p.current().Type == lexer.TokenIndent {
					p.advance()
					continue
				}
				if p.current().Type == lexer.TokenString {
					field.EnumValues = append(field.EnumValues, p.advance().Value)
				} else if p.current().Type != lexer.TokenRParen {
					field.EnumValues = append(field.EnumValues, p.advance().Value)
				}
			}
			if p.current().Type == lexer.TokenRParen {
				p.advance()
			}
		default:
			p.advance()
		}
	}

	return field, nil
}

func tokenToFieldType(tok lexer.Token) (ast.FieldType, error) {
	switch tok.Type {
	case lexer.TokenTexto:
		return ast.FieldTexto, nil
	case lexer.TokenNumero:
		return ast.FieldNumero, nil
	case lexer.TokenData:
		return ast.FieldData, nil
	case lexer.TokenBooleano:
		return ast.FieldBooleano, nil
	case lexer.TokenEmail:
		return ast.FieldEmail, nil
	case lexer.TokenTelefone:
		return ast.FieldTelefone, nil
	case lexer.TokenImagem:
		return ast.FieldImagem, nil
	case lexer.TokenArquivo:
		return ast.FieldArquivo, nil
	case lexer.TokenUpload:
		return ast.FieldUpload, nil
	case lexer.TokenLink:
		return ast.FieldLink, nil
	case lexer.TokenStatus:
		return ast.FieldStatus, nil
	case lexer.TokenDinheiro:
		return ast.FieldDinheiro, nil
	case lexer.TokenSenha:
		return ast.FieldSenha, nil
	case lexer.TokenTextoLongo:
		return ast.FieldTextoLongo, nil
	case lexer.TokenEnum:
		return ast.FieldEnum, nil
	case lexer.TokenCPF:
		return ast.FieldCPF, nil
	case lexer.TokenCEP:
		return ast.FieldCEP, nil
	case lexer.TokenEstrelas:
		return ast.FieldEstrelas, nil
	case lexer.TokenDataHora:
		return ast.FieldDataHora, nil
	case lexer.TokenPercentual:
		return ast.FieldPercentual, nil
	case lexer.TokenTags:
		return ast.FieldTags, nil
	case lexer.TokenMoeda:
		return ast.FieldMoeda, nil
	}
	typeMap := map[string]ast.FieldType{
		"texto": ast.FieldTexto, "numero": ast.FieldNumero, "data": ast.FieldData,
		"booleano": ast.FieldBooleano, "email": ast.FieldEmail, "telefone": ast.FieldTelefone,
		"imagem": ast.FieldImagem, "arquivo": ast.FieldArquivo, "upload": ast.FieldUpload,
		"link": ast.FieldLink, "status": ast.FieldStatus, "dinheiro": ast.FieldDinheiro,
		"senha": ast.FieldSenha, "texto_longo": ast.FieldTextoLongo, "enum": ast.FieldEnum,
		// EN
		"text": ast.FieldTexto, "number": ast.FieldNumero, "date": ast.FieldData,
		"boolean": ast.FieldBooleano, "phone": ast.FieldTelefone, "image": ast.FieldImagem,
		"file": ast.FieldArquivo, "money": ast.FieldDinheiro, "password": ast.FieldSenha,
		"long_text": ast.FieldTextoLongo,
		// New types PT
		"cpf": ast.FieldCPF, "cnpj": ast.FieldCPF,
		"cep": ast.FieldCEP, "zipcode": ast.FieldCEP,
		"cor": ast.FieldCor,
		"estrelas": ast.FieldEstrelas, "rating": ast.FieldEstrelas, "stars": ast.FieldEstrelas,
		"hora": ast.FieldHora, "horario": ast.FieldHora, "time": ast.FieldHora,
		"data_hora": ast.FieldDataHora, "datetime": ast.FieldDataHora,
		"percentual": ast.FieldPercentual, "porcentagem": ast.FieldPercentual, "percentage": ast.FieldPercentual,
		"tags": ast.FieldTags, "etiquetas": ast.FieldTags, "chips": ast.FieldTags,
		"url": ast.FieldURL,
		"moeda": ast.FieldMoeda, "currency_type": ast.FieldMoeda,
	}
	if ft, ok := typeMap[tok.Value]; ok {
		return ft, nil
	}
	return "", fmt.Errorf("tipo desconhecido: %q", tok.Value)
}

// parseTelas parses the screens block.
func (p *Parser) parseTelas() error {
	p.advance() // consume 'telas'
	p.skipWhitespace()

	for !p.isAtEnd() && !p.isBlockKeyword() {
		tok := p.current()
		if tok.Type == lexer.TokenTela {
			screen, err := p.parseScreen()
			if err != nil {
				return err
			}
			p.program.Screens = append(p.program.Screens, screen)
		} else {
			p.advance()
		}
	}
	return nil
}

// parseScreen parses a single screen definition.
func (p *Parser) parseScreen() (*ast.Screen, error) {
	p.advance() // consume 'tela'
	p.skipIndent()

	nameTok := p.advance()
	screen := &ast.Screen{Name: nameTok.Value}
	p.skipWhitespace()

	for !p.isAtEnd() && !p.isBlockKeyword() {
		tok := p.current()

		// Check if we hit a new 'tela' keyword
		if tok.Type == lexer.TokenTela {
			break
		}

		if tok.Type == lexer.TokenNewline || tok.Type == lexer.TokenIndent {
			p.advance()
			continue
		}

		switch tok.Type {
		case lexer.TokenTitulo:
			p.advance()
			p.skipIndent()
			if p.current().Type == lexer.TokenString {
				screen.Title = p.advance().Value
			}
		case lexer.TokenLista:
			comp, err := p.parseListComponent()
			if err != nil {
				return nil, err
			}
			screen.Components = append(screen.Components, comp)
		case lexer.TokenBotao:
			comp, err := p.parseButtonComponent()
			if err != nil {
				return nil, err
			}
			screen.Components = append(screen.Components, comp)
		case lexer.TokenFormulario:
			comp, err := p.parseFormComponent()
			if err != nil {
				return nil, err
			}
			screen.Components = append(screen.Components, comp)
		case lexer.TokenChat:
			comp, err := p.parseChatComponent()
			if err != nil {
				return nil, err
			}
			screen.Components = append(screen.Components, comp)
		case lexer.TokenMostrar:
			comp := p.parseShowComponent()
			screen.Components = append(screen.Components, comp)
		case lexer.TokenGrafico:
			comp := &ast.Component{
				Type:       ast.CompChart,
				Properties: make(map[string]string),
			}
			p.advance()
			p.skipIndent()
			if !p.isAtEnd() && (p.current().Type == lexer.TokenIdentifier || lexer.IsTypeKeyword(p.current().Type)) {
				comp.Target = p.advance().Value
			}
			// Parse chart properties on following lines
			p.skipWhitespace()
			for !p.isAtEnd() && !p.isBlockKeyword() && p.current().Type != lexer.TokenTela {
				tok2 := p.current()
				if tok2.Type == lexer.TokenNewline || tok2.Type == lexer.TokenIndent {
					p.advance()
					continue
				}
				if tok2.Value == "tipo" || tok2.Value == "type" {
					p.advance()
					p.skipIndent()
					if p.current().Type == lexer.TokenString || p.current().Type == lexer.TokenIdentifier {
						comp.Properties["chart_type"] = p.advance().Value
					}
					continue
				}
				break
			}
			screen.Components = append(screen.Components, comp)
		case lexer.TokenBusca:
			comp := &ast.Component{
				Type:       ast.CompSearch,
				Properties: make(map[string]string),
			}
			p.advance()
			p.skipIndent()
			if !p.isAtEnd() && p.current().Type == lexer.TokenIdentifier {
				comp.Target = p.advance().Value
			}
			screen.Components = append(screen.Components, comp)
		case lexer.TokenSelecionar:
			comp := &ast.Component{
				Type:       ast.CompSelect,
				Properties: make(map[string]string),
			}
			p.advance()
			p.skipIndent()
			if !p.isAtEnd() && p.current().Type == lexer.TokenIdentifier {
				comp.Target = p.advance().Value
			}
			screen.Components = append(screen.Components, comp)
		case lexer.TokenAreaTexto:
			comp := &ast.Component{
				Type:       ast.CompTextarea,
				Properties: make(map[string]string),
			}
			p.advance()
			p.skipIndent()
			if !p.isAtEnd() && p.current().Type == lexer.TokenIdentifier {
				comp.Target = p.advance().Value
			}
			screen.Components = append(screen.Components, comp)
		case lexer.TokenDashboard:
			comp := &ast.Component{
				Type:       "dashboard",
				Properties: make(map[string]string),
			}
			p.advance()
			screen.Components = append(screen.Components, comp)
		case lexer.TokenTabela:
			comp, err := p.parseListComponent()
			if err != nil {
				return nil, err
			}
			comp.Type = "tabela"
			screen.Components = append(screen.Components, comp)
		case lexer.TokenRequer:
			p.advance()
			p.skipIndent()
			if !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
				screen.Requires = p.advance().Value
			}
		case lexer.TokenPublico:
			p.advance()
			screen.Public = true
		default:
			p.advance()
		}
	}

	return screen, nil
}

func (p *Parser) parseListComponent() (*ast.Component, error) {
	p.advance() // consume 'lista'
	p.skipIndent()

	comp := &ast.Component{
		Type:       ast.CompList,
		Properties: make(map[string]string),
	}

	if !p.isAtEnd() && (p.current().Type == lexer.TokenIdentifier || lexer.IsTypeKeyword(p.current().Type)) {
		comp.Target = p.advance().Value
	}

	p.skipWhitespace()

	// Parse children (mostrar fields)
	for !p.isAtEnd() && !p.isBlockKeyword() {
		tok := p.current()
		if tok.Type == lexer.TokenNewline || tok.Type == lexer.TokenIndent {
			p.advance()
			continue
		}
		if tok.Type == lexer.TokenMostrar {
			child := p.parseShowComponent()
			comp.Children = append(comp.Children, child)
			continue
		}
		// Any other token means we're done with list children
		break
	}

	return comp, nil
}

func (p *Parser) parseShowComponent() *ast.Component {
	p.advance() // consume 'mostrar'
	p.skipIndent()

	comp := &ast.Component{
		Type:       ast.CompShow,
		Properties: make(map[string]string),
	}

	if !p.isAtEnd() {
		tok := p.current()
		if tok.Type == lexer.TokenIdentifier || lexer.IsTypeKeyword(tok.Type) {
			comp.Target = p.advance().Value
		}
	}

	return comp
}

func (p *Parser) parseButtonComponent() (*ast.Component, error) {
	p.advance() // consume 'botao'
	p.skipIndent()

	comp := &ast.Component{
		Type:       ast.CompButton,
		Properties: make(map[string]string),
	}

	// Optional color
	if p.current().Type == lexer.TokenIdentifier {
		comp.Properties["color"] = p.advance().Value
	}

	p.skipWhitespace()

	// Parse button properties
	for !p.isAtEnd() && !p.isBlockKeyword() {
		tok := p.current()
		if tok.Type == lexer.TokenNewline || tok.Type == lexer.TokenIndent {
			p.advance()
			continue
		}
		if tok.Type == lexer.TokenTexto || tok.Value == "texto" {
			p.advance()
			p.skipIndent()
			if p.current().Type == lexer.TokenString {
				comp.Properties["text"] = p.advance().Value
			}
			continue
		}
		break
	}

	return comp, nil
}

func (p *Parser) parseFormComponent() (*ast.Component, error) {
	p.advance() // consume 'formulario'
	p.skipIndent()

	comp := &ast.Component{
		Type:       ast.CompForm,
		Properties: make(map[string]string),
	}

	if !p.isAtEnd() && p.current().Type == lexer.TokenIdentifier {
		comp.Target = p.advance().Value
	}

	return comp, nil
}

func (p *Parser) parseChatComponent() (*ast.Component, error) {
	p.advance() // consume 'chat'
	p.skipIndent()

	comp := &ast.Component{
		Type:       ast.CompChat,
		Properties: make(map[string]string),
	}

	if !p.isAtEnd() && p.current().Type == lexer.TokenIdentifier {
		comp.Target = p.advance().Value
	}

	comp.Properties["messages_model"] = "mensagem"
	comp.Properties["relation_field"] = strings.ToLower(comp.Target)
	comp.Properties["text_field"] = "corpo"
	comp.Properties["media_field"] = "media_url"
	comp.Properties["author_field"] = "de_mim"
	comp.Properties["timestamp_field"] = "criado_em"
	comp.Properties["type_field"] = "tipo"

	p.skipWhitespace()
	for !p.isAtEnd() && !p.isBlockKeyword() {
		tok := p.current()
		if tok.Type == lexer.TokenNewline || tok.Type == lexer.TokenIndent {
			p.advance()
			continue
		}
		if tok.Type == lexer.TokenTela {
			break
		}

		key := tok.Value
		p.advance()
		p.skipIndent()
		if p.isAtEnd() || p.current().Type == lexer.TokenNewline {
			continue
		}
		value := p.advance().Value
		switch key {
		case "mensagens", "messages":
			comp.Properties["messages_model"] = value
		case "relacao", "relation":
			comp.Properties["relation_field"] = value
		case "texto", "text":
			comp.Properties["text_field"] = value
		case "media", "media_field":
			comp.Properties["media_field"] = value
		case "autor", "author":
			comp.Properties["author_field"] = value
		case "data", "timestamp":
			comp.Properties["timestamp_field"] = value
		case "tipo", "message_type", "type":
			comp.Properties["type_field"] = value
		case "titulo", "title":
			comp.Properties["title"] = value
		default:
			comp.Properties[key] = value
		}
	}

	return comp, nil
}

// parseEventos parses the events block.
func (p *Parser) parseEventos() error {
	p.advance() // consume 'eventos'
	p.skipWhitespace()

	for !p.isAtEnd() && !p.isBlockKeyword() {
		tok := p.current()
		if tok.Type == lexer.TokenQuando {
			event, err := p.parseEvent()
			if err != nil {
				return err
			}
			p.program.Events = append(p.program.Events, event)
		} else {
			p.advance()
		}
	}
	return nil
}

func (p *Parser) parseEvent() (*ast.Event, error) {
	p.advance() // consume 'quando'
	p.skipIndent()

	event := &ast.Event{}

	// Parse trigger: clicar, enviar, etc.
	if !p.isAtEnd() {
		event.Trigger = p.advance().Value
	}

	// Parse target (quoted string)
	p.skipIndent()
	if p.current().Type == lexer.TokenString {
		event.Target = p.advance().Value
	}

	p.skipWhitespace()

	// Parse action reference (next line typically)
	var actionParts []string
	for !p.isAtEnd() && p.current().Type != lexer.TokenNewline && !p.isBlockKeyword() && p.current().Type != lexer.TokenQuando {
		if p.current().Type == lexer.TokenIndent {
			p.advance()
			continue
		}
		actionParts = append(actionParts, p.advance().Value)
	}
	event.ActionRef = strings.Join(actionParts, " ")

	return event, nil
}

// parseAcoes parses the actions block.
func (p *Parser) parseAcoes() error {
	p.advance() // consume 'acoes'
	p.skipWhitespace()

	for !p.isAtEnd() && !p.isBlockKeyword() {
		p.advance()
	}
	return nil
}

// parseTemaValue reads a string or identifier and resolves color names.
func (p *Parser) parseTemaValue() string {
	if p.current().Type == lexer.TokenString {
		return p.advance().Value
	}
	if p.current().Type == lexer.TokenIdentifier || p.isNameToken(p.current()) {
		return ast.ResolveColor(p.advance().Value)
	}
	return ""
}

// parseTema parses the theme block.
// Supports: tema moderno escuro     (preset + modifier)
//
//	tema azul               (color preset)
//	tema                    (block with properties)
func (p *Parser) parseTema() error {
	p.advance() // consume 'tema'
	p.skipIndent()

	theme := ast.DefaultTheme()

	// Check for inline preset: "tema moderno escuro" or "tema azul"
	if !p.isAtEnd() && p.current().Type != lexer.TokenNewline && !p.isBlockKeyword() {
		first := p.current()
		if first.Type == lexer.TokenIdentifier || first.Type == lexer.TokenEscuro {
			word := first.Value
			// Check if it's a preset name
			preset := ast.ThemePreset(word)
			if word != "cor" && word != "fonte" && word != "estilo" && word != "fundo" &&
				word != "borda" && word != "icone" && word != "cartao" && word != "css" &&
				word != "texto_cor" && word != "card" && word != "raio" {
				if _, isColor := ast.ColorName[word]; isColor {
					// "tema azul" — apply color as primary
					theme.Primary = ast.ResolveColor(word)
					p.advance()
				} else if word == "escuro" || word == "dark" {
					theme.Dark = true
					p.advance()
				} else {
					// It's a preset name: "tema moderno"
					theme = preset
					p.advance()
				}
				p.skipIndent()
				// Check for second word: "tema moderno escuro" or "tema azul escuro"
				if !p.isAtEnd() && p.current().Type != lexer.TokenNewline && !p.isBlockKeyword() {
					second := p.current().Value
					if second == "escuro" || second == "dark" {
						theme.Dark = true
						p.advance()
					} else if _, isColor := ast.ColorName[second]; isColor {
						theme.Primary = ast.ResolveColor(second)
						p.advance()
					}
				}
			}
		}
	}

	p.skipWhitespace()

	for !p.isAtEnd() && !p.isBlockKeyword() {
		tok := p.current()

		if tok.Type == lexer.TokenNewline || tok.Type == lexer.TokenIndent {
			p.advance()
			continue
		}

		switch {
		case tok.Value == "cor" || tok.Type == lexer.TokenCor:
			p.advance()
			p.skipIndent()
			if p.current().Type == lexer.TokenIdentifier || p.current().Type == lexer.TokenCor {
				which := p.advance().Value
				p.skipIndent()
				val := p.parseTemaValue()
				if val != "" {
					switch which {
					case "primaria", "primary":
						theme.Primary = val
					case "secundaria", "secondary":
						theme.Secondary = val
					case "destaque", "accent":
						theme.Accent = val
					case "sidebar":
						theme.Sidebar = val
					}
				}
			} else {
				val := p.parseTemaValue()
				if val != "" {
					theme.Primary = val
				}
			}
		case tok.Value == "escuro" || tok.Value == "dark" || tok.Type == lexer.TokenEscuro:
			p.advance()
			theme.Dark = true
		case tok.Value == "claro" || tok.Value == "light":
			p.advance()
			theme.Dark = false
		case tok.Value == "icone" || tok.Type == lexer.TokenIcone:
			p.advance()
			p.skipIndent()
			if p.current().Type == lexer.TokenString {
				theme.Icon = p.advance().Value
			}
		case tok.Value == "fonte" || tok.Value == "font":
			p.advance()
			p.skipIndent()
			val := p.parseTemaValue()
			if val != "" {
				theme.Font = val
			}
		case tok.Value == "borda" || tok.Value == "radius" || tok.Value == "raio":
			p.advance()
			p.skipIndent()
			val := p.parseTemaValue()
			if val != "" {
				theme.Radius = val
			}
		case tok.Value == "fundo" || tok.Value == "background":
			p.advance()
			p.skipIndent()
			val := p.parseTemaValue()
			if val != "" {
				theme.Background = val
			}
		case tok.Value == "card" || tok.Value == "cartao":
			p.advance()
			p.skipIndent()
			val := p.parseTemaValue()
			if val != "" {
				theme.CardBg = val
			}
		case tok.Value == "texto_cor" || tok.Value == "text_color":
			p.advance()
			p.skipIndent()
			val := p.parseTemaValue()
			if val != "" {
				theme.TextColor = val
			}
		case tok.Value == "estilo" || tok.Value == "style":
			p.advance()
			p.skipIndent()
			if p.current().Type == lexer.TokenString {
				theme.Style = p.advance().Value
			} else if p.current().Type == lexer.TokenIdentifier || p.isNameToken(p.current()) {
				theme.Style = p.advance().Value
			}
		case tok.Value == "css":
			p.advance()
			p.skipIndent()
			if p.current().Type == lexer.TokenString {
				theme.CustomCSS = p.advance().Value
			}
		default:
			p.advance()
		}
	}

	p.program.Theme = theme
	return nil
}

// parseImportar parses: importar "arquivo.ge"
//
//	importar dados de "arquivo.ge"
//	importar tela de "arquivo.ge"
func (p *Parser) parseImportar() error {
	p.advance() // consume 'importar'
	p.skipIndent()

	imp := &ast.Import{}

	tok := p.current()

	// importar "file.ge" (import everything)
	if tok.Type == lexer.TokenString {
		imp.What = "tudo"
		imp.Path = p.advance().Value
		p.program.Imports = append(p.program.Imports, imp)
		return nil
	}

	// importar <what> de "file.ge"
	imp.What = p.advance().Value
	p.skipIndent()

	// expect 'de'
	if p.current().Type == lexer.TokenDe {
		p.advance()
		p.skipIndent()
	}

	if p.current().Type == lexer.TokenString {
		imp.Path = p.advance().Value
	} else {
		return fmt.Errorf("line %d: esperado caminho do arquivo após 'importar %s de'", p.current().Line, imp.What)
	}

	p.program.Imports = append(p.program.Imports, imp)
	return nil
}

// parseLogica parses the logic block with rules.
func (p *Parser) parseLogica() error {
	p.advance() // consume 'logica'
	p.skipWhitespace()

	for !p.isAtEnd() && !p.isBlockKeyword() {
		tok := p.current()

		if tok.Type == lexer.TokenNewline || tok.Type == lexer.TokenIndent {
			p.advance()
			continue
		}

		// Try new scripting constructs first
		switch tok.Type {
		case lexer.TokenFuncao:
			fn, err := p.parseFuncDecl()
			if err != nil {
				return err
			}
			p.program.Functions = append(p.program.Functions, fn)
			continue
		case lexer.TokenDefinir:
			stmt, err := p.parseStatement(0)
			if err != nil {
				return err
			}
			if stmt != nil {
				p.program.Scripts = append(p.program.Scripts, stmt)
			}
			continue
		case lexer.TokenSe:
			stmt, err := p.parseStatement(0)
			if err != nil {
				return err
			}
			if stmt != nil {
				p.program.Scripts = append(p.program.Scripts, stmt)
			}
			continue
		case lexer.TokenParaCada, lexer.TokenPara:
			stmt, err := p.parseStatement(0)
			if err != nil {
				return err
			}
			if stmt != nil {
				p.program.Scripts = append(p.program.Scripts, stmt)
			}
			continue
		case lexer.TokenEnquanto:
			stmt, err := p.parseStatement(0)
			if err != nil {
				return err
			}
			if stmt != nil {
				p.program.Scripts = append(p.program.Scripts, stmt)
			}
			continue
		case lexer.TokenRepetir:
			stmt, err := p.parseStatement(0)
			if err != nil {
				return err
			}
			if stmt != nil {
				p.program.Scripts = append(p.program.Scripts, stmt)
			}
			continue
		case lexer.TokenMostrar:
			stmt, err := p.parseStatement(0)
			if err != nil {
				return err
			}
			if stmt != nil {
				p.program.Scripts = append(p.program.Scripts, stmt)
			}
			continue
		case lexer.TokenTentar:
			stmt, err := p.parseStatement(0)
			if err != nil {
				return err
			}
			if stmt != nil {
				p.program.Scripts = append(p.program.Scripts, stmt)
			}
			continue
		case lexer.TokenRetornar:
			stmt, err := p.parseStatement(0)
			if err != nil {
				return err
			}
			if stmt != nil {
				p.program.Scripts = append(p.program.Scripts, stmt)
			}
			continue
		case lexer.TokenPausar, lexer.TokenContinuar, lexer.TokenParar:
			stmt, err := p.parseStatement(0)
			if err != nil {
				return err
			}
			if stmt != nil {
				p.program.Scripts = append(p.program.Scripts, stmt)
			}
			continue
		case lexer.TokenValidar:
			if err := p.parseValidacao(); err != nil {
				return err
			}
			continue
		case lexer.TokenIdentifier:
			// Could be assignment (x = ...) or function call (func(...))
			stmt, err := p.parseStatement(0)
			if err != nil {
				return err
			}
			if stmt != nil {
				p.program.Scripts = append(p.program.Scripts, stmt)
			}
			continue
		default:
			p.advance()
		}
	}
	return nil
}

// parseRule parses: se <field> igual/maior/menor <value> \n <action> <arg>
func (p *Parser) parseRule() error {
	p.advance() // consume 'se' or 'quando'
	p.skipIndent()

	rule := &ast.Rule{}

	// field name
	if !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
		rule.Field = p.advance().Value
	}
	p.skipIndent()

	// operator
	if !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
		rule.Operator = p.advance().Value
	}
	p.skipIndent()

	// value
	if !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
		if p.current().Type == lexer.TokenString {
			rule.Value = p.advance().Value
		} else {
			rule.Value = p.advance().Value
		}
	}

	p.skipWhitespace()

	// action line: mudar/validar/calcular/definir <arg>
	if !p.isAtEnd() && !p.isBlockKeyword() {
		tok := p.current()
		if tok.Type != lexer.TokenSe && tok.Type != lexer.TokenQuando && tok.Type != lexer.TokenValidar {
			rule.Action = p.advance().Value
			p.skipIndent()

			// Collect rest of line as arg
			var parts []string
			for !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
				if p.current().Type == lexer.TokenIndent {
					p.advance()
					continue
				}
				if p.current().Type == lexer.TokenString {
					parts = append(parts, p.advance().Value)
				} else {
					parts = append(parts, p.advance().Value)
				}
			}
			rule.ActionArg = strings.Join(parts, " ")
		}
	}

	p.program.Rules = append(p.program.Rules, rule)
	return nil
}

// parseValidacao parses: validar <field> <condition>
func (p *Parser) parseValidacao() error {
	p.advance() // consume 'validar'
	p.skipIndent()

	rule := &ast.Rule{Action: "validar"}

	// field
	if !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
		rule.Field = p.advance().Value
	}
	p.skipIndent()

	// operator
	if !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
		rule.Operator = p.advance().Value
	}
	p.skipIndent()

	// value
	if !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
		if p.current().Type == lexer.TokenString {
			rule.Value = p.advance().Value
		} else {
			rule.Value = p.advance().Value
		}
	}

	p.program.Rules = append(p.program.Rules, rule)
	return nil
}

// parseBanco parses database configuration block.
func (p *Parser) parseBanco() error {
	p.advance() // consume 'banco'
	p.skipWhitespace()

	db := ast.DefaultDatabase()

	// Check if driver name is on same line: banco postgres
	if !p.isAtEnd() && p.current().Type == lexer.TokenIdentifier {
		db.Driver = p.advance().Value
	}
	p.skipWhitespace()

	for !p.isAtEnd() && !p.isBlockKeyword() {
		tok := p.current()

		if tok.Type == lexer.TokenNewline || tok.Type == lexer.TokenIndent {
			p.advance()
			continue
		}

		if tok.Type == lexer.TokenIdentifier || lexer.IsTypeKeyword(tok.Type) {
			key := p.advance().Value
			p.skipIndent()

			if p.current().Type == lexer.TokenColon {
				p.advance()
				p.skipIndent()
			}

			val := ""
			if p.current().Type == lexer.TokenString {
				val = p.advance().Value
			} else if !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
				val = p.advance().Value
			}

			switch key {
			case "driver", "tipo", "type":
				db.Driver = val
			case "host", "servidor", "server":
				db.Host = val
			case "port", "porta":
				db.Port = val
			case "nome", "name", "database", "db":
				db.Name = val
			case "usuario", "user", "username":
				db.User = val
			case "senha", "password", "pass":
				db.Password = val
			}
			continue
		}

		p.advance()
	}

	p.program.Database = db
	return nil
}

// parseIntegracoes parses the integrations block (whatsapp, etc).
func (p *Parser) parseIntegracoes() error {
	p.advance() // consume 'integracoes'
	p.skipWhitespace()

	for !p.isAtEnd() && !p.isBlockKeyword() {
		tok := p.current()

		if tok.Type == lexer.TokenNewline || tok.Type == lexer.TokenIndent {
			p.advance()
			continue
		}

		if tok.Type == lexer.TokenWhatsapp {
			if err := p.parseWhatsApp(); err != nil {
				return err
			}
			continue
		}

		if tok.Type == lexer.TokenEmailInteg || tok.Value == "email" {
			if err := p.parseEmailInteg(); err != nil {
				return err
			}
			continue
		}

		if tok.Type == lexer.TokenCron {
			if err := p.parseCron(); err != nil {
				return err
			}
			continue
		}

		p.advance()
	}
	return nil
}

// parseWhatsApp parses the whatsapp sub-block inside integracoes.
// whatsapp
//
//	quando criar pedido
//	  enviar mensagem para cliente.telefone
//	    texto "Seu pedido foi recebido!"
func (p *Parser) parseWhatsApp() error {
	p.advance() // consume 'whatsapp'
	p.skipWhitespace()

	// Enable WhatsApp
	if p.program.WhatsApp == nil {
		p.program.WhatsApp = &ast.WhatsAppConfig{Enabled: true, DBPath: "whatsapp.db", Provider: "whatsmeow"}
	}

	for !p.isAtEnd() && !p.isBlockKeyword() {
		tok := p.current()

		if tok.Type == lexer.TokenNewline || tok.Type == lexer.TokenIndent {
			p.advance()
			continue
		}

		if tok.Type == lexer.TokenIdentifier || tok.Type == lexer.TokenSessao || tok.Type == lexer.TokenQRCode || tok.Type == lexer.TokenPresenca || tok.Type == lexer.TokenFilaJobs {
			key := tok.Value
			p.advance()
			p.skipIndent()
			if p.current().Type == lexer.TokenColon {
				p.advance()
				p.skipIndent()
			}
			if p.isAtEnd() || p.current().Type == lexer.TokenNewline {
				continue
			}
			value := p.advance().Value
			switch key {
			case "provedor", "provider":
				p.program.WhatsApp.Provider = value
			case "db_path", "banco", "database":
				p.program.WhatsApp.DBPath = value
			case "multi_sessao", "multi_session":
				p.program.WhatsApp.MultiSession = value == "verdadeiro" || value == "true" || value == "sim"
			case "presenca", "presence":
				p.program.WhatsApp.Presence = value == "verdadeiro" || value == "true" || value == "sim"
			case "qr_code", "qrcode":
				p.program.WhatsApp.QRCodeFlow = value == "verdadeiro" || value == "true" || value == "sim"
			}
			continue
		}

		// quando <trigger> <model>
		if tok.Type == lexer.TokenQuando {
			notif, err := p.parseNotifier("whatsapp")
			if err != nil {
				return err
			}
			if notif != nil {
				p.program.Notifiers = append(p.program.Notifiers, notif)
			}
			continue
		}

		// Exit if we hit another integration section
		if tok.Type == lexer.TokenWhatsapp {
			break
		}

		p.advance()
	}

	return nil
}

// parseNotifier parses a notification trigger.
// quando criar pedido
//
//	enviar mensagem para cliente.telefone
//	  texto "Mensagem aqui"
func (p *Parser) parseNotifier(channel string) (*ast.Notifier, error) {
	p.advance() // consume 'quando'
	p.skipIndent()

	notif := &ast.Notifier{Channel: channel}

	// trigger: criar, atualizar, deletar, or field condition
	if !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
		triggerTok := p.advance()
		notif.Trigger = triggerTok.Value
	}
	p.skipIndent()

	// model name or field condition
	if !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
		notif.Model = p.advance().Value
	}

	// Optional condition: igual "value"
	p.skipIndent()
	if !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
		if p.current().Type == lexer.TokenIgual || p.current().Value == "igual" || p.current().Value == "equals" {
			p.advance() // skip 'igual'
			p.skipIndent()
			// The model was actually the field, trigger was condition context
			notif.Field = notif.Model
			notif.Model = ""
			if p.current().Type == lexer.TokenString {
				notif.Value = p.advance().Value
			} else if !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
				notif.Value = p.advance().Value
			}
		}
	}

	p.skipWhitespace()

	// Parse body lines: enviar mensagem para <dest> / texto "..."
	for !p.isAtEnd() && !p.isBlockKeyword() {
		tok := p.current()

		if tok.Type == lexer.TokenNewline || tok.Type == lexer.TokenIndent {
			p.advance()
			continue
		}

		// Stop at next 'quando'
		if tok.Type == lexer.TokenQuando {
			break
		}

		switch tok.Value {
		case "enviar", "send":
			p.advance()
			p.skipIndent()
			// skip 'mensagem' / 'message'
			if p.current().Type == lexer.TokenMensagem {
				p.advance()
			}
			p.skipIndent()
			// skip 'para' / 'to'
			if p.current().Value == "para" || p.current().Type == lexer.TokenPara {
				p.advance()
			}
			p.skipIndent()
			// destination
			if !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
				notif.SendTo = p.advance().Value
				// Check for dotted access like cliente.telefone
				if p.current().Type == lexer.TokenDot {
					p.advance()
					if !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
						notif.SendTo += "." + p.advance().Value
					}
				}
			}

		case "texto", "text":
			p.advance()
			p.skipIndent()
			if p.current().Type == lexer.TokenString {
				notif.Message = p.advance().Value
			}

		default:
			p.advance()
		}
	}

	return notif, nil
}

// parseAuth parses the authentication block.
// autenticacao
//
//	usuario: email
//	senha: senha
//	roles: admin, usuario
func (p *Parser) parseAuth() error {
	p.advance() // consume 'autenticacao'
	p.skipWhitespace()

	auth := &ast.AuthConfig{
		Enabled:    true,
		UserModel:  "usuario",
		LoginField: "email",
		PassField:  "senha",
		JWTSecret:  "flang-secret-change-me",
	}

	for !p.isAtEnd() && !p.isBlockKeyword() {
		tok := p.current()

		if tok.Type == lexer.TokenNewline || tok.Type == lexer.TokenIndent {
			p.advance()
			continue
		}

		// Accept any token as key (keywords can be used as config keys)
		if tok.Type != lexer.TokenNewline && tok.Type != lexer.TokenIndent && tok.Type != lexer.TokenEOF {
			key := p.advance().Value
			p.skipIndent()

			if p.current().Type == lexer.TokenColon {
				p.advance()
				p.skipIndent()
			}

			val := ""
			if p.current().Type == lexer.TokenString {
				val = p.advance().Value
			} else if !p.isAtEnd() && p.current().Type != lexer.TokenNewline && p.current().Type != lexer.TokenComma {
				val = p.advance().Value
			}

			switch key {
			case "usuario", "user", "modelo", "model":
				auth.UserModel = val
			case "login", "campo_login", "login_field":
				auth.LoginField = val
			case "senha", "password", "campo_senha", "password_field":
				auth.PassField = val
			case "secret", "segredo", "jwt_secret":
				auth.JWTSecret = val
			case "roles", "permissoes":
				auth.Roles = append(auth.Roles, val)
				for !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
					if p.current().Type == lexer.TokenComma || p.current().Type == lexer.TokenIndent {
						p.advance()
						continue
					}
					auth.Roles = append(auth.Roles, p.advance().Value)
				}
			}
			continue
		}

		p.advance()
	}

	p.program.Auth = auth
	return nil
}

// parseEmailInteg parses the email sub-block inside integracoes.
// email
//
//	servidor: "smtp.gmail.com"
//	porta: "587"
//	usuario: "me@gmail.com"
//	senha: "app-password"
//	quando criar pedido
//	  enviar email para cliente.email
//	    assunto "Pedido recebido"
//	    texto "Olá {cliente}, seu pedido..."
func (p *Parser) parseEmailInteg() error {
	p.advance() // consume 'email'
	p.skipWhitespace()

	// Initialize email config
	if p.program.Email == nil {
		p.program.Email = &ast.EmailConfig{}
	}

	for !p.isAtEnd() && !p.isBlockKeyword() {
		tok := p.current()

		if tok.Type == lexer.TokenNewline || tok.Type == lexer.TokenIndent {
			p.advance()
			continue
		}

		// quando <trigger> <model> — parse as email notifier
		if tok.Type == lexer.TokenQuando {
			notif, err := p.parseEmailNotifier()
			if err != nil {
				return err
			}
			if notif != nil {
				p.program.Notifiers = append(p.program.Notifiers, notif)
			}
			continue
		}

		// Exit if we hit another integration keyword
		if tok.Type == lexer.TokenWhatsapp || tok.Type == lexer.TokenCron || tok.Type == lexer.TokenEmailInteg {
			break
		}

		// Config key: value pairs (servidor, porta, usuario, senha)
		if tok.Type == lexer.TokenIdentifier || lexer.IsTypeKeyword(tok.Type) ||
			tok.Type == lexer.TokenSenha || tok.Type == lexer.TokenUsuario {
			key := p.advance().Value
			p.skipIndent()

			if p.current().Type == lexer.TokenColon {
				p.advance()
				p.skipIndent()
			}

			val := ""
			if p.current().Type == lexer.TokenString {
				val = p.advance().Value
			} else if !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
				val = p.advance().Value
			}

			switch key {
			case "servidor", "server", "host":
				p.program.Email.Host = val
			case "porta", "port":
				p.program.Email.Port = val
			case "usuario", "user", "username":
				p.program.Email.User = val
			case "senha", "password", "pass":
				p.program.Email.Password = val
			case "de", "from", "remetente":
				p.program.Email.From = val
			}
			continue
		}

		p.advance()
	}

	return nil
}

// parseEmailNotifier parses an email notification trigger.
// quando criar pedido
//
//	enviar email para cliente.email
//	  assunto "Pedido recebido"
//	  texto "Mensagem aqui"
func (p *Parser) parseEmailNotifier() (*ast.Notifier, error) {
	p.advance() // consume 'quando'
	p.skipIndent()

	notif := &ast.Notifier{Channel: "email"}

	// trigger: criar, atualizar, deletar
	if !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
		notif.Trigger = p.advance().Value
	}
	p.skipIndent()

	// model name
	if !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
		notif.Model = p.advance().Value
	}

	p.skipWhitespace()

	// Parse body: enviar email para <dest> / assunto "..." / texto "..."
	for !p.isAtEnd() && !p.isBlockKeyword() {
		tok := p.current()

		if tok.Type == lexer.TokenNewline || tok.Type == lexer.TokenIndent {
			p.advance()
			continue
		}

		// Stop at next 'quando' or integration keyword
		if tok.Type == lexer.TokenQuando || tok.Type == lexer.TokenWhatsapp ||
			tok.Type == lexer.TokenCron || tok.Type == lexer.TokenEmailInteg {
			break
		}

		switch tok.Value {
		case "enviar", "send":
			p.advance()
			p.skipIndent()
			// skip 'email' / 'mensagem' / 'message'
			if p.current().Value == "email" || p.current().Type == lexer.TokenMensagem {
				p.advance()
			}
			p.skipIndent()
			// skip 'para' / 'to'
			if p.current().Value == "para" || p.current().Type == lexer.TokenPara {
				p.advance()
			}
			p.skipIndent()
			// destination
			if !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
				notif.SendTo = p.advance().Value
				if p.current().Type == lexer.TokenDot {
					p.advance()
					if !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
						notif.SendTo += "." + p.advance().Value
					}
				}
			}

		case "assunto", "subject":
			p.advance()
			p.skipIndent()
			if p.current().Type == lexer.TokenString {
				notif.Subject = p.advance().Value
			}

		case "texto", "text":
			p.advance()
			p.skipIndent()
			if p.current().Type == lexer.TokenString {
				notif.Message = p.advance().Value
			}

		default:
			p.advance()
		}
	}

	return notif, nil
}

// parseCron parses the cron sub-block inside integracoes.
// cron
//
//	cada 5 minutos
//	  chamar api "https://example.com/webhook"
//	cada 1 hora
//	  limpar sessoes
func (p *Parser) parseCron() error {
	p.advance() // consume 'cron'
	p.skipWhitespace()

	for !p.isAtEnd() && !p.isBlockKeyword() {
		tok := p.current()

		if tok.Type == lexer.TokenNewline || tok.Type == lexer.TokenIndent {
			p.advance()
			continue
		}

		// cada/every <N> <unit>
		if tok.Type == lexer.TokenCada {
			job, err := p.parseCronJob()
			if err != nil {
				return err
			}
			if job != nil {
				p.program.Crons = append(p.program.Crons, job)
			}
			continue
		}

		// Exit if we hit another integration keyword
		if tok.Type == lexer.TokenWhatsapp || tok.Type == lexer.TokenEmailInteg || tok.Type == lexer.TokenCron {
			break
		}

		p.advance()
	}

	return nil
}

// parseCronJob parses a single cron job definition.
// cada 5 minutos
//
//	chamar api "https://example.com/webhook"
func (p *Parser) parseCronJob() (*ast.CronJob, error) {
	p.advance() // consume 'cada' / 'every'
	p.skipIndent()

	job := &ast.CronJob{}

	// Parse interval: <number> <unit>
	var intervalParts []string
	for !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
		if p.current().Type == lexer.TokenIndent {
			p.advance()
			continue
		}
		intervalParts = append(intervalParts, p.advance().Value)
	}
	job.Every = strings.Join(intervalParts, " ")

	p.skipWhitespace()

	// Parse action line(s)
	for !p.isAtEnd() && !p.isBlockKeyword() {
		tok := p.current()

		if tok.Type == lexer.TokenNewline || tok.Type == lexer.TokenIndent {
			p.advance()
			continue
		}

		// Stop at next 'cada' or integration keyword
		if tok.Type == lexer.TokenCada || tok.Type == lexer.TokenWhatsapp ||
			tok.Type == lexer.TokenEmailInteg || tok.Type == lexer.TokenCron {
			break
		}

		// chamar api "url" / chamar "url"
		if tok.Type == lexer.TokenChamar || tok.Value == "chamar" || tok.Value == "call" {
			p.advance()
			p.skipIndent()
			job.Action = "chamar"

			// skip optional 'api'
			if p.current().Type == lexer.TokenApi || p.current().Value == "api" {
				p.advance()
				p.skipIndent()
			}

			// URL
			if p.current().Type == lexer.TokenString {
				job.Target = p.advance().Value
			}
			break
		}

		// Generic action: collect remaining tokens on the line as action + target
		var actionParts []string
		for !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
			if p.current().Type == lexer.TokenIndent {
				p.advance()
				continue
			}
			actionParts = append(actionParts, p.advance().Value)
		}
		if len(actionParts) > 0 {
			job.Action = actionParts[0]
			if len(actionParts) > 1 {
				job.Target = strings.Join(actionParts[1:], " ")
			}
		}
		break
	}

	return job, nil
}

// ==================== Scripting Parser ====================

// parseStatement parses a single statement at a given indentation level.
func (p *Parser) parseStatement(minIndent int) (*ast.Statement, error) {
	tok := p.current()

	switch tok.Type {
	case lexer.TokenDefinir:
		return p.parseVarDecl()
	case lexer.TokenSe:
		return p.parseIfStmt(minIndent)
	case lexer.TokenParaCada:
		return p.parseForEachStmt(minIndent)
	case lexer.TokenPara:
		// "para cada" or "para" as for_each
		return p.parseForStmt(minIndent)
	case lexer.TokenEnquanto:
		return p.parseWhileStmt(minIndent)
	case lexer.TokenRepetir:
		return p.parseRepeatStmt(minIndent)
	case lexer.TokenMostrar:
		return p.parsePrintStmt()
	case lexer.TokenTentar:
		return p.parseTryStmt(minIndent)
	case lexer.TokenRetornar:
		return p.parseReturnStmt()
	case lexer.TokenPausar:
		p.advance()
		return &ast.Statement{Type: "pause"}, nil
	case lexer.TokenContinuar:
		p.advance()
		return &ast.Statement{Type: "continue"}, nil
	case lexer.TokenParar:
		p.advance()
		return &ast.Statement{Type: "break"}, nil
	case lexer.TokenIdentifier:
		return p.parseIdentStmt()
	default:
		p.advance()
		return nil, nil
	}
}

// parseVarDecl: definir x = expression
func (p *Parser) parseVarDecl() (*ast.Statement, error) {
	p.advance() // consume 'definir'/'set'
	p.skipIndent()

	nameTok := p.current()
	if nameTok.Type != lexer.TokenIdentifier {
		return nil, fmt.Errorf("line %d: expected variable name after 'definir', got %q", nameTok.Line, nameTok.Value)
	}
	p.advance()
	p.skipIndent()

	// Expect '='
	if p.current().Type != lexer.TokenEquals {
		return nil, fmt.Errorf("line %d: expected '=' after variable name", p.current().Line)
	}
	p.advance()
	p.skipIndent()

	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	return &ast.Statement{
		Type: "var",
		VarDecl: &ast.VarDecl{
			Name:  nameTok.Value,
			Value: *expr,
		},
	}, nil
}

// parseIdentStmt parses assignment (x = ..., x.field = ...) or function call (fn(...))
func (p *Parser) parseIdentStmt() (*ast.Statement, error) {
	nameTok := p.advance() // consume identifier
	p.skipIndent()

	// Check for dot (field access assignment)
	if p.current().Type == lexer.TokenDot {
		p.advance() // consume '.'
		p.skipIndent()
		fieldTok := p.current()
		if fieldTok.Type != lexer.TokenIdentifier && !p.isNameToken(fieldTok) {
			return nil, fmt.Errorf("line %d: expected field name after '.'", fieldTok.Line)
		}
		p.advance()
		p.skipIndent()

		if p.current().Type == lexer.TokenEquals {
			p.advance() // consume '='
			p.skipIndent()
			expr, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			return &ast.Statement{
				Type: "assign",
				Assign: &ast.Assignment{
					Target: nameTok.Value,
					Field:  fieldTok.Value,
					Value:  *expr,
				},
			}, nil
		}

		// It's a method call: obj.method(args)
		if p.current().Type == lexer.TokenLParen {
			args, err := p.parseCallArgs()
			if err != nil {
				return nil, err
			}
			return &ast.Statement{
				Type: "call",
				Call: &ast.FuncCall{
					Name:   fieldTok.Value,
					Object: nameTok.Value,
					Args:   args,
				},
			}, nil
		}

		return nil, fmt.Errorf("line %d: expected '=' or '(' after field access", p.current().Line)
	}

	// Check for '=' (assignment)
	if p.current().Type == lexer.TokenEquals {
		p.advance() // consume '='
		p.skipIndent()
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		return &ast.Statement{
			Type: "assign",
			Assign: &ast.Assignment{
				Target: nameTok.Value,
				Value:  *expr,
			},
		}, nil
	}

	// Check for '(' (function call)
	if p.current().Type == lexer.TokenLParen {
		args, err := p.parseCallArgs()
		if err != nil {
			return nil, err
		}
		return &ast.Statement{
			Type: "call",
			Call: &ast.FuncCall{
				Name: nameTok.Value,
				Args: args,
			},
		}, nil
	}

	// Bare identifier — treat as call with no args (like a command)
	return &ast.Statement{
		Type: "call",
		Call: &ast.FuncCall{
			Name: nameTok.Value,
		},
	}, nil
}

// parseCallArgs parses (arg1, arg2, ...)
func (p *Parser) parseCallArgs() ([]*ast.Expression, error) {
	p.advance() // consume '('
	p.skipIndent()

	var args []*ast.Expression
	for !p.isAtEnd() && p.current().Type != lexer.TokenRParen {
		if p.current().Type == lexer.TokenComma {
			p.advance()
			p.skipIndent()
			continue
		}
		if p.current().Type == lexer.TokenNewline || p.current().Type == lexer.TokenIndent {
			p.advance()
			continue
		}
		arg, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		p.skipIndent()
	}
	if p.current().Type == lexer.TokenRParen {
		p.advance() // consume ')'
	}
	return args, nil
}

// parsePrintStmt: mostrar expression
func (p *Parser) parsePrintStmt() (*ast.Statement, error) {
	p.advance() // consume 'mostrar'/'print'/'show'
	p.skipIndent()

	// In screens context, mostrar is handled elsewhere.
	// In logic context, parse as print statement.
	if p.current().Type == lexer.TokenNewline || p.current().Type == lexer.TokenEOF || p.isBlockKeyword() {
		return &ast.Statement{
			Type:  "print",
			Print: &ast.Expression{Type: "literal", Value: ""},
		}, nil
	}

	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	return &ast.Statement{
		Type:  "print",
		Print: expr,
	}, nil
}

// parseReturnStmt: retornar expression
func (p *Parser) parseReturnStmt() (*ast.Statement, error) {
	p.advance() // consume 'retornar'/'return'
	p.skipIndent()

	if p.current().Type == lexer.TokenNewline || p.current().Type == lexer.TokenEOF {
		return &ast.Statement{
			Type:   "return",
			Return: &ast.Expression{Type: "literal", Value: nil},
		}, nil
	}

	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	return &ast.Statement{
		Type:   "return",
		Return: expr,
	}, nil
}

// parseIfStmt: se condition \n body \n senao se condition \n body \n senao \n body
func (p *Parser) parseIfStmt(minIndent int) (*ast.Statement, error) {
	p.advance() // consume 'se'/'if'
	p.skipIndent()

	cond, err := p.parseExpression()
	if err != nil {
		return nil, fmt.Errorf("line %d: error parsing if condition: %w", p.current().Line, err)
	}

	ifStmt := &ast.IfStmt{
		Condition: *cond,
	}

	// Parse body
	body, err := p.parseBlock(minIndent)
	if err != nil {
		return nil, err
	}
	ifStmt.Body = body

	// Parse else-if and else clauses
	for {
		p.skipNewlinesAndIndent()
		if p.isAtEnd() || p.isBlockKeyword() {
			break
		}

		if p.current().Type == lexer.TokenSenao {
			p.advance() // consume 'senao'
			p.skipIndent()

			// senao se = else if
			if p.current().Type == lexer.TokenSe {
				p.advance() // consume 'se'
				p.skipIndent()

				elseIfCond, err := p.parseExpression()
				if err != nil {
					return nil, err
				}
				elseIfBody, err := p.parseBlock(minIndent)
				if err != nil {
					return nil, err
				}
				ifStmt.ElseIfs = append(ifStmt.ElseIfs, &ast.ElseIfClause{
					Condition: *elseIfCond,
					Body:      elseIfBody,
				})
				continue
			}

			// plain else
			elseBody, err := p.parseBlock(minIndent)
			if err != nil {
				return nil, err
			}
			ifStmt.Else = elseBody
			break
		}
		break
	}

	return &ast.Statement{
		Type: "if",
		If:   ifStmt,
	}, nil
}

// parseForEachStmt: para_cada x em collection \n body
func (p *Parser) parseForEachStmt(minIndent int) (*ast.Statement, error) {
	p.advance() // consume 'para_cada'/'for_each'
	p.skipIndent()

	varTok := p.current()
	if varTok.Type != lexer.TokenIdentifier {
		return nil, fmt.Errorf("line %d: expected variable name after 'para_cada'", varTok.Line)
	}
	p.advance()
	p.skipIndent()

	// Expect 'em'/'in'
	if p.current().Type == lexer.TokenEm {
		p.advance()
	}
	p.skipIndent()

	collExpr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	body, err := p.parseBlock(minIndent)
	if err != nil {
		return nil, err
	}

	return &ast.Statement{
		Type: "for_each",
		ForEach: &ast.ForEachStmt{
			VarName:    varTok.Value,
			Collection: *collExpr,
			Body:       body,
		},
	}, nil
}

// parseForStmt: para cada x em collection OR just use for_each semantics
func (p *Parser) parseForStmt(minIndent int) (*ast.Statement, error) {
	p.advance() // consume 'para'/'for'
	p.skipIndent()

	// 'para cada' = for each
	if p.current().Type == lexer.TokenCada {
		p.advance() // consume 'cada'/'each'
		p.skipIndent()

		varTok := p.current()
		if varTok.Type != lexer.TokenIdentifier {
			return nil, fmt.Errorf("line %d: expected variable name after 'para cada'", varTok.Line)
		}
		p.advance()
		p.skipIndent()

		if p.current().Type == lexer.TokenEm {
			p.advance()
		}
		p.skipIndent()

		collExpr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		body, err := p.parseBlock(minIndent)
		if err != nil {
			return nil, err
		}

		return &ast.Statement{
			Type: "for_each",
			ForEach: &ast.ForEachStmt{
				VarName:    varTok.Value,
				Collection: *collExpr,
				Body:       body,
			},
		}, nil
	}

	// Bare 'para' — skip for now
	p.skipToNextLine()
	return nil, nil
}

// parseWhileStmt: enquanto condition \n body
func (p *Parser) parseWhileStmt(minIndent int) (*ast.Statement, error) {
	p.advance() // consume 'enquanto'/'while'
	p.skipIndent()

	cond, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	body, err := p.parseBlock(minIndent)
	if err != nil {
		return nil, err
	}

	return &ast.Statement{
		Type: "while",
		While: &ast.WhileStmt{
			Condition: *cond,
			Body:      body,
		},
	}, nil
}

// parseRepeatStmt: repetir N vezes \n body
func (p *Parser) parseRepeatStmt(minIndent int) (*ast.Statement, error) {
	p.advance() // consume 'repetir'/'repeat'
	p.skipIndent()

	countExpr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	// Optional 'vezes'/'times'
	if p.current().Type == lexer.TokenVezes {
		p.advance()
	}

	body, err := p.parseBlock(minIndent)
	if err != nil {
		return nil, err
	}

	return &ast.Statement{
		Type: "repeat",
		Repeat: &ast.RepeatStmt{
			Count: *countExpr,
			Body:  body,
		},
	}, nil
}

// parseTryStmt: tentar \n body \n erro [varname] \n body
func (p *Parser) parseTryStmt(minIndent int) (*ast.Statement, error) {
	p.advance() // consume 'tentar'/'try'

	tryBody, err := p.parseBlock(minIndent)
	if err != nil {
		return nil, err
	}

	tryStmt := &ast.TryStmt{
		Body: tryBody,
	}

	// Look for 'erro'/'error'
	p.skipNewlinesAndIndent()
	if p.current().Type == lexer.TokenErro {
		p.advance() // consume 'erro'
		p.skipIndent()

		// Optional error variable name
		if p.current().Type == lexer.TokenIdentifier {
			tryStmt.ErrVar = p.advance().Value
		}

		catchBody, err := p.parseBlock(minIndent)
		if err != nil {
			return nil, err
		}
		tryStmt.Catch = catchBody
	}

	return &ast.Statement{
		Type: "try",
		Try:  tryStmt,
	}, nil
}

// parseFuncDecl: funcao name(param1, param2) \n body
func (p *Parser) parseFuncDecl() (*ast.FuncDecl, error) {
	p.advance() // consume 'funcao'/'function'
	p.skipIndent()

	nameTok := p.current()
	if nameTok.Type != lexer.TokenIdentifier {
		return nil, fmt.Errorf("line %d: expected function name after 'funcao'", nameTok.Line)
	}
	p.advance()
	p.skipIndent()

	// Parse parameters
	var params []string
	if p.current().Type == lexer.TokenLParen {
		p.advance() // consume '('
		for !p.isAtEnd() && p.current().Type != lexer.TokenRParen {
			if p.current().Type == lexer.TokenComma || p.current().Type == lexer.TokenIndent {
				p.advance()
				continue
			}
			params = append(params, p.advance().Value)
		}
		if p.current().Type == lexer.TokenRParen {
			p.advance() // consume ')'
		}
	}

	body, err := p.parseBlock(0)
	if err != nil {
		return nil, err
	}

	return &ast.FuncDecl{
		Name:   nameTok.Value,
		Params: params,
		Body:   body,
	}, nil
}

// parseBlock parses indented statements as a block.
// A block is a set of statements that are indented more than minIndent.
func (p *Parser) parseBlock(minIndent int) ([]*ast.Statement, error) {
	var stmts []*ast.Statement

	// Skip to next line to start the block
	for !p.isAtEnd() && p.current().Type != lexer.TokenNewline && p.current().Type != lexer.TokenEOF {
		// If there's content on the same line after condition, skip it
		if p.current().Type == lexer.TokenIndent {
			p.advance()
			continue
		}
		break
	}

	// Find the block indentation level
	blockIndent := -1

	for !p.isAtEnd() {
		tok := p.current()

		if tok.Type == lexer.TokenNewline {
			p.advance()
			continue
		}

		if tok.Type == lexer.TokenIndent {
			indent := tok.Indent
			p.advance()

			if blockIndent == -1 {
				// First indented line establishes block indent
				if indent > minIndent {
					blockIndent = indent
				} else {
					// Not indented enough — no block body
					p.pos-- // put indent back
					return stmts, nil
				}
			}

			if indent < blockIndent {
				// De-indented — block is over
				p.pos-- // put indent back
				return stmts, nil
			}

			if indent >= blockIndent {
				// Parse statement at this indent level
				if p.isAtEnd() || p.current().Type == lexer.TokenNewline {
					continue
				}
				// Check for block keywords that end logic blocks
				if p.isBlockKeyword() {
					p.pos-- // put indent back
					return stmts, nil
				}
				// Check for senao at same level (belongs to parent if)
				if p.current().Type == lexer.TokenSenao && indent == minIndent {
					p.pos--
					return stmts, nil
				}
				if p.current().Type == lexer.TokenErro && indent == minIndent {
					p.pos--
					return stmts, nil
				}

				stmt, err := p.parseStatement(blockIndent)
				if err != nil {
					return nil, err
				}
				if stmt != nil {
					stmts = append(stmts, stmt)
				}
			}
			continue
		}

		// Non-indent, non-newline token at start — check if it's a block boundary
		if p.isBlockKeyword() {
			break
		}
		if tok.Type == lexer.TokenSenao || tok.Type == lexer.TokenErro {
			break
		}

		// If we haven't established block indent yet, this is at the base level — not a block
		if blockIndent == -1 {
			break
		}

		// Parse inline content
		stmt, err := p.parseStatement(blockIndent)
		if err != nil {
			return nil, err
		}
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
	}

	return stmts, nil
}

// skipNewlinesAndIndent skips newlines and indent tokens.
func (p *Parser) skipNewlinesAndIndent() {
	for !p.isAtEnd() {
		tt := p.current().Type
		if tt == lexer.TokenNewline || tt == lexer.TokenIndent {
			p.advance()
		} else {
			break
		}
	}
}

// ==================== Expression Parser ====================
// Operator precedence (low to high):
// 1. ou/or
// 2. e/and
// 3. ==, !=, >, <, >=, <=, igual, maior, menor
// 4. +, -
// 5. *, /
// 6. unary (nao/not, -)
// 7. primary (literals, variables, calls, field access, parenthesized)

func (p *Parser) parseExpression() (*ast.Expression, error) {
	return p.parseOr()
}

func (p *Parser) parseOr() (*ast.Expression, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.current().Type == lexer.TokenOu {
		p.advance()
		p.skipIndent()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &ast.Expression{
			Type:     "binary",
			Left:     left,
			Right:    right,
			Operator: "ou",
		}
	}
	return left, nil
}

func (p *Parser) parseAnd() (*ast.Expression, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}

	for p.current().Type == lexer.TokenE {
		p.advance()
		p.skipIndent()
		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		left = &ast.Expression{
			Type:     "binary",
			Left:     left,
			Right:    right,
			Operator: "e",
		}
	}
	return left, nil
}

func (p *Parser) parseComparison() (*ast.Expression, error) {
	left, err := p.parseAddition()
	if err != nil {
		return nil, err
	}

	for {
		tok := p.current()
		var op string
		switch tok.Type {
		case lexer.TokenEqualEqual:
			op = "=="
		case lexer.TokenDiferente:
			op = "!="
		case lexer.TokenMaiorQue:
			op = ">"
		case lexer.TokenMenorQue:
			op = "<"
		case lexer.TokenMaiorIgual:
			op = ">="
		case lexer.TokenMenorIgual:
			op = "<="
		case lexer.TokenIgual:
			op = "=="
		case lexer.TokenMaior:
			op = ">"
		case lexer.TokenMenor:
			op = "<"
		default:
			return left, nil
		}

		p.advance()
		p.skipIndent()
		right, err := p.parseAddition()
		if err != nil {
			return nil, err
		}
		left = &ast.Expression{
			Type:     "binary",
			Left:     left,
			Right:    right,
			Operator: op,
		}
	}
}

func (p *Parser) parseAddition() (*ast.Expression, error) {
	left, err := p.parseMultiplication()
	if err != nil {
		return nil, err
	}

	for p.current().Type == lexer.TokenPlus || p.current().Type == lexer.TokenMinus {
		op := "+"
		if p.current().Type == lexer.TokenMinus {
			op = "-"
		}
		p.advance()
		p.skipIndent()
		right, err := p.parseMultiplication()
		if err != nil {
			return nil, err
		}
		left = &ast.Expression{
			Type:     "binary",
			Left:     left,
			Right:    right,
			Operator: op,
		}
	}
	return left, nil
}

func (p *Parser) parseMultiplication() (*ast.Expression, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for p.current().Type == lexer.TokenStar || p.current().Type == lexer.TokenSlash {
		op := "*"
		if p.current().Type == lexer.TokenSlash {
			op = "/"
		}
		p.advance()
		p.skipIndent()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &ast.Expression{
			Type:     "binary",
			Left:     left,
			Right:    right,
			Operator: op,
		}
	}
	return left, nil
}

func (p *Parser) parseUnary() (*ast.Expression, error) {
	if p.current().Type == lexer.TokenNao {
		p.advance()
		p.skipIndent()
		expr, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &ast.Expression{
			Type:     "unary",
			Operator: "nao",
			Right:    expr,
		}, nil
	}
	if p.current().Type == lexer.TokenMinus {
		p.advance()
		p.skipIndent()
		expr, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &ast.Expression{
			Type:     "unary",
			Operator: "-",
			Right:    expr,
		}, nil
	}
	return p.parsePrimary()
}

func (p *Parser) parsePrimary() (*ast.Expression, error) {
	tok := p.current()

	switch tok.Type {
	case lexer.TokenNumber:
		p.advance()
		return &ast.Expression{Type: "literal", Value: tok.Value}, nil

	case lexer.TokenString:
		p.advance()
		return &ast.Expression{Type: "literal", Value: tok.Value}, nil

	case lexer.TokenVerdadeiro:
		p.advance()
		return &ast.Expression{Type: "literal", Value: true}, nil

	case lexer.TokenFalso:
		p.advance()
		return &ast.Expression{Type: "literal", Value: false}, nil

	case lexer.TokenNulo:
		p.advance()
		return &ast.Expression{Type: "literal", Value: nil}, nil

	case lexer.TokenLParen:
		p.advance() // consume '('
		p.skipIndent()
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		p.skipIndent()
		if p.current().Type == lexer.TokenRParen {
			p.advance() // consume ')'
		}
		return expr, nil

	case lexer.TokenLBracket:
		// List literal: [1, 2, 3]
		p.advance() // consume '['
		p.skipIndent()
		var elements []*ast.Expression
		for !p.isAtEnd() && p.current().Type != lexer.TokenRBracket {
			if p.current().Type == lexer.TokenComma || p.current().Type == lexer.TokenNewline || p.current().Type == lexer.TokenIndent {
				p.advance()
				continue
			}
			elem, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			elements = append(elements, elem)
			p.skipIndent()
		}
		if p.current().Type == lexer.TokenRBracket {
			p.advance()
		}
		return &ast.Expression{Type: "list", Elements: elements}, nil

	case lexer.TokenIdentifier:
		name := tok.Value
		p.advance()

		// Check for function call: name(...)
		if p.current().Type == lexer.TokenLParen {
			args, err := p.parseCallArgs()
			if err != nil {
				return nil, err
			}
			return &ast.Expression{
				Type: "call",
				Name: name,
				Args: args,
			}, nil
		}

		// Check for array indexing: name[index]
		if p.current().Type == lexer.TokenLBracket {
			p.advance() // consume '['
			p.skipIndent()
			indexExpr, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			p.skipIndent()
			if p.current().Type == lexer.TokenRBracket {
				p.advance() // consume ']'
			}
			return &ast.Expression{
				Type:  "index",
				Name:  name,
				Index: indexExpr,
			}, nil
		}

		// Check for field access: name.field
		if p.current().Type == lexer.TokenDot {
			p.advance() // consume '.'
			fieldTok := p.current()
			fieldName := fieldTok.Value
			p.advance()

			// Check for method call: name.field(...)
			if p.current().Type == lexer.TokenLParen {
				args, err := p.parseCallArgs()
				if err != nil {
					return nil, err
				}
				return &ast.Expression{
					Type:   "call",
					Name:   fieldName,
					Object: name,
					Args:   args,
				}, nil
			}

			// Check for array index after field access: obj.field[0]
			if p.current().Type == lexer.TokenLBracket {
				p.advance()
				p.skipIndent()
				indexExpr, err := p.parseExpression()
				if err != nil {
					return nil, err
				}
				p.skipIndent()
				if p.current().Type == lexer.TokenRBracket {
					p.advance()
				}
				return &ast.Expression{
					Type:   "index",
					Object: name,
					Field:  fieldName,
					Index:  indexExpr,
				}, nil
			}

			return &ast.Expression{
				Type:   "field_access",
				Object: name,
				Field:  fieldName,
			}, nil
		}

		return &ast.Expression{Type: "variable", Name: name}, nil

	default:
		// For keywords used as identifiers/functions in expression context
		// e.g. texto(x), numero(x), tamanho(x), maiusculo(x)
		if p.isNameToken(tok) {
			name := tok.Value
			p.advance()

			// Check for function call: name(...)
			if p.current().Type == lexer.TokenLParen {
				args, err := p.parseCallArgs()
				if err != nil {
					return nil, err
				}
				return &ast.Expression{Type: "call", Name: name, Args: args}, nil
			}

			// Check for array indexing: name[index]
			if p.current().Type == lexer.TokenLBracket {
				p.advance() // consume '['
				p.skipIndent()
				indexExpr, err := p.parseExpression()
				if err != nil {
					return nil, err
				}
				p.skipIndent()
				if p.current().Type == lexer.TokenRBracket {
					p.advance()
				}
				return &ast.Expression{Type: "index", Name: name, Index: indexExpr}, nil
			}

			// Check for field access: name.field
			if p.current().Type == lexer.TokenDot {
				p.advance()
				fieldTok := p.current()
				fieldName := fieldTok.Value
				p.advance()
				if p.current().Type == lexer.TokenLParen {
					args, err := p.parseCallArgs()
					if err != nil {
						return nil, err
					}
					return &ast.Expression{Type: "call", Name: fieldName, Object: name, Args: args}, nil
				}
				// Check for array index after field access: obj.field[0]
				if p.current().Type == lexer.TokenLBracket {
					p.advance()
					p.skipIndent()
					indexExpr, err := p.parseExpression()
					if err != nil {
						return nil, err
					}
					p.skipIndent()
					if p.current().Type == lexer.TokenRBracket {
						p.advance()
					}
					return &ast.Expression{Type: "index", Object: name, Field: fieldName, Index: indexExpr}, nil
				}
				return &ast.Expression{Type: "field_access", Object: name, Field: fieldName}, nil
			}

			return &ast.Expression{Type: "variable", Name: name}, nil
		}
		return &ast.Expression{Type: "literal", Value: nil}, nil
	}
}

// parseRotas parses custom route definitions.
// rotas
//
//	rota GET "/api/relatorio"
//	  retornar "dados"
func (p *Parser) parseRotas() error {
	p.advance() // consume 'rotas'
	p.skipWhitespace()

	for !p.isAtEnd() && !p.isBlockKeyword() {
		tok := p.current()
		if tok.Type == lexer.TokenNewline || tok.Type == lexer.TokenIndent {
			p.advance()
			continue
		}
		// Break on other top-level custom blocks
		if tok.Type == lexer.TokenIdentifier && (tok.Value == "rotas" || tok.Value == "routes" ||
			tok.Value == "paginas" || tok.Value == "pages" || tok.Value == "sidebar" || tok.Value == "menu") {
			break
		}

		if tok.Value == "rota" || tok.Value == "route" {
			p.advance()
			p.skipIndent()

			route := &ast.CustomRoute{}

			// Method: GET, POST, PUT, DELETE
			if !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
				route.Method = strings.ToUpper(p.advance().Value)
			}
			p.skipIndent()

			// Path
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				route.Path = p.advance().Value
			}

			// Parse body as statements
			body, err := p.parseBlock(0)
			if err != nil {
				return err
			}
			route.Handler = body

			p.program.Routes = append(p.program.Routes, route)
			continue
		}

		p.advance()
	}
	return nil
}

// parsePaginas parses custom HTML pages.
// paginas
//
//	pagina "/sobre"
//	  titulo "Sobre nós"
//	  html "<div>Conteúdo</div>"
func (p *Parser) parsePaginas() error {
	p.advance() // consume 'paginas'
	p.skipWhitespace()

	for !p.isAtEnd() && !p.isBlockKeyword() {
		tok := p.current()
		if tok.Type == lexer.TokenNewline || tok.Type == lexer.TokenIndent {
			p.advance()
			continue
		}
		if tok.Type == lexer.TokenIdentifier && (tok.Value == "rotas" || tok.Value == "routes" ||
			tok.Value == "paginas" || tok.Value == "pages" || tok.Value == "sidebar" || tok.Value == "menu") {
			break
		}

		if tok.Value == "pagina" || tok.Value == "page" {
			p.advance()
			p.skipIndent()

			page := &ast.CustomPage{}

			// Path
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				page.Path = p.advance().Value
			}
			p.skipWhitespace()

			// Parse properties
			for !p.isAtEnd() && !p.isBlockKeyword() {
				inner := p.current()
				if inner.Type == lexer.TokenNewline || inner.Type == lexer.TokenIndent {
					p.advance()
					continue
				}
				if inner.Value == "pagina" || inner.Value == "page" {
					break
				}
				if inner.Type == lexer.TokenIdentifier && (inner.Value == "rotas" || inner.Value == "routes" ||
					inner.Value == "paginas" || inner.Value == "pages" || inner.Value == "sidebar" || inner.Value == "menu") {
					break
				}
				switch inner.Value {
				case "titulo", "title":
					p.advance()
					p.skipIndent()
					if !p.isAtEnd() && p.current().Type == lexer.TokenString {
						page.Title = p.advance().Value
					}
				case "html", "conteudo", "content":
					p.advance()
					p.skipIndent()
					if !p.isAtEnd() && p.current().Type == lexer.TokenString {
						page.Content = p.advance().Value
					}
				default:
					p.advance()
				}
			}

			p.program.Pages = append(p.program.Pages, page)
			continue
		}

		p.advance()
	}
	return nil
}

// parseSidebarBlock parses sidebar customization.
// sidebar
//
//	item "Dashboard" icone "home" link "dashboard"
//	item "Relatórios" icone "chart" link "/relatorios"
func (p *Parser) parseSidebar() error {
	p.advance() // consume 'sidebar'
	p.skipWhitespace()

	order := 0
	for !p.isAtEnd() && !p.isBlockKeyword() {
		tok := p.current()
		if tok.Type == lexer.TokenNewline || tok.Type == lexer.TokenIndent {
			p.advance()
			continue
		}
		if tok.Type == lexer.TokenIdentifier && (tok.Value == "rotas" || tok.Value == "routes" ||
			tok.Value == "paginas" || tok.Value == "pages" || tok.Value == "sidebar" || tok.Value == "menu") {
			break
		}

		if tok.Value == "item" {
			p.advance()
			p.skipIndent()

			item := &ast.SidebarItem{Order: order}
			order++

			// Label
			if !p.isAtEnd() && p.current().Type == lexer.TokenString {
				item.Label = p.advance().Value
			}
			p.skipIndent()

			// Parse optional properties on same line
			for !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
				if p.current().Type == lexer.TokenIndent {
					p.advance()
					continue
				}
				prop := p.current().Value
				p.advance()
				p.skipIndent()
				switch prop {
				case "icone", "icon":
					if !p.isAtEnd() && p.current().Type == lexer.TokenString {
						item.Icon = p.advance().Value
					}
				case "link", "href":
					if !p.isAtEnd() && p.current().Type == lexer.TokenString {
						item.Link = p.advance().Value
					} else if !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
						// Read link value, joining tokens with - (e.g. screen-tickets)
						item.Link = p.advance().Value
						for !p.isAtEnd() && p.current().Type == lexer.TokenMinus {
							p.advance() // consume '-'
							if !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
								item.Link += "-" + p.advance().Value
							}
						}
					}
				default:
					// skip unknown props
				}
			}

			p.program.SidebarItems = append(p.program.SidebarItems, item)
			continue
		}

		p.advance()
	}
	return nil
}
