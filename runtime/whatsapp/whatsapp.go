package whatsapp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

type Event struct {
	Type      string `json:"type"`
	Session   string `json:"session"`
	Status    string `json:"status,omitempty"`
	QRCode    string `json:"qr_code,omitempty"`
	Phone     string `json:"phone,omitempty"`
	Error     string `json:"error,omitempty"`
	Connected bool   `json:"connected"`
}

type SessionInfo struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	QRCode    string `json:"qr_code,omitempty"`
	Phone     string `json:"phone,omitempty"`
	Connected bool   `json:"connected"`
}

type sessionState struct {
	id        string
	dbPath    string
	client    *whatsmeow.Client
	connected bool
	status    string
	qrCode    string
	phone     string
}

// Client wraps multiple whatsmeow sessions for Germanio.
type Client struct {
	mu             sync.RWMutex
	sessions       map[string]*sessionState
	defaultSession string
	baseDBPath     string
	onEvent        func(Event)
}

// Novo creates a new WhatsApp client manager.
func Novo(dbPath string) *Client {
	if dbPath == "" {
		dbPath = "whatsapp.db"
	}
	c := &Client{
		sessions:       make(map[string]*sessionState),
		defaultSession: "default",
		baseDBPath:     dbPath,
	}
	c.sessions[c.defaultSession] = &sessionState{id: c.defaultSession, dbPath: dbPath, status: "desconectado"}
	return c
}

func (c *Client) SetEventHandler(fn func(Event)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onEvent = fn
}

func (c *Client) emit(evt Event) {
	c.mu.RLock()
	fn := c.onEvent
	c.mu.RUnlock()
	if fn != nil {
		fn(evt)
	}
}

func (c *Client) Conectar() error {
	return c.ConectarSessao(c.defaultSession)
}

func (c *Client) ConectarSessao(sessionID string) error {
	if sessionID == "" {
		sessionID = c.defaultSession
	}

	s := c.ensureSession(sessionID)
	ctx := context.Background()

	dbLog := waLog.Stdout("WA-DB", "WARN", true)
	container, err := sqlstore.New(ctx, "sqlite3", "file:"+s.dbPath+"?_foreign_keys=on", dbLog)
	if err != nil {
		c.setStatus(sessionID, "erro", "", "")
		return fmt.Errorf("erro ao criar store WhatsApp: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		c.setStatus(sessionID, "erro", "", "")
		return fmt.Errorf("erro ao obter device: %w", err)
	}

	clientLog := waLog.Stdout("WA", "WARN", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)
	c.mu.Lock()
	s.client = client
	c.mu.Unlock()

	if client.Store.ID == nil {
		c.setStatus(sessionID, "aguardando_qr", "", "")
		qrChan, _ := client.GetQRChannel(ctx)
		if err := client.Connect(); err != nil {
			c.setStatus(sessionID, "erro", "", "")
			c.emit(Event{Type: "whatsapp_erro", Session: sessionID, Status: "erro", Error: err.Error()})
			return fmt.Errorf("erro ao conectar WhatsApp: %w", err)
		}

		for evt := range qrChan {
			switch evt.Event {
			case "code":
				c.setStatus(sessionID, "aguardando_qr", evt.Code, "")
				printQR(evt.Code)
				c.emit(Event{Type: "qr", Session: sessionID, Status: "aguardando_qr", QRCode: evt.Code})
			case "success":
				phone := ""
				if client.Store.ID != nil {
					phone = client.Store.ID.User
				}
				c.setStatus(sessionID, "conectado", "", phone)
				c.emit(Event{Type: "whatsapp_status", Session: sessionID, Status: "conectado", Connected: true, Phone: phone})
				return nil
			case "timeout":
				c.setStatus(sessionID, "erro", "", "")
				c.emit(Event{Type: "whatsapp_erro", Session: sessionID, Status: "erro", Error: "timeout ao esperar QR code"})
				return fmt.Errorf("timeout ao esperar QR code")
			}
		}
		return nil
	}

	if err := client.Connect(); err != nil {
		c.setStatus(sessionID, "erro", "", "")
		c.emit(Event{Type: "whatsapp_erro", Session: sessionID, Status: "erro", Error: err.Error()})
		return fmt.Errorf("erro ao conectar WhatsApp: %w", err)
	}
	phone := ""
	if client.Store.ID != nil {
		phone = client.Store.ID.User
	}
	c.setStatus(sessionID, "conectado", "", phone)
	c.emit(Event{Type: "whatsapp_status", Session: sessionID, Status: "conectado", Connected: true, Phone: phone})
	return nil
}

func (c *Client) EnviarMensagem(telefone string, mensagem string) error {
	return c.EnviarMensagemSessao(c.defaultSession, telefone, mensagem)
}

func (c *Client) EnviarMensagemSessao(sessionID string, telefone string, mensagem string) error {
	if sessionID == "" {
		sessionID = c.defaultSession
	}
	c.mu.RLock()
	s := c.sessions[sessionID]
	c.mu.RUnlock()
	if s == nil || !s.connected || s.client == nil {
		return fmt.Errorf("WhatsApp não conectado")
	}

	phone := limparTelefone(telefone)
	if phone == "" {
		return fmt.Errorf("telefone inválido: %s", telefone)
	}

	jid := types.NewJID(phone, types.DefaultUserServer)
	msg := &waE2E.Message{Conversation: proto.String(mensagem)}
	_, err := s.client.SendMessage(context.Background(), jid, msg)
	if err != nil {
		return fmt.Errorf("erro ao enviar mensagem: %w", err)
	}
	return nil
}

func (c *Client) Desconectar() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, s := range c.sessions {
		if s.client != nil {
			s.client.Disconnect()
			s.connected = false
			s.status = "desconectado"
		}
	}
}

func (c *Client) DesconectarSessao(sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if s := c.sessions[sessionID]; s != nil && s.client != nil {
		s.client.Disconnect()
		s.connected = false
		s.status = "desconectado"
		s.qrCode = ""
	}
	if sessionID != "" {
		c.emit(Event{Type: "whatsapp_status", Session: sessionID, Status: "desconectado", Connected: false})
	}
}

func (c *Client) IsConnected() bool {
	return c.IsConnectedSessao(c.defaultSession)
}

func (c *Client) IsConnectedSessao(sessionID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	s := c.sessions[sessionID]
	return s != nil && s.connected
}

func (c *Client) QRCode(sessionID string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if s := c.sessions[sessionID]; s != nil {
		return s.qrCode
	}
	return ""
}

func (c *Client) ListarSessoes() []SessionInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	items := make([]SessionInfo, 0, len(c.sessions))
	for _, s := range c.sessions {
		items = append(items, SessionInfo{ID: s.id, Status: s.status, QRCode: s.qrCode, Phone: s.phone, Connected: s.connected})
	}
	return items
}

func (c *Client) SessionInfo(sessionID string) SessionInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if s := c.sessions[sessionID]; s != nil {
		return SessionInfo{ID: s.id, Status: s.status, QRCode: s.qrCode, Phone: s.phone, Connected: s.connected}
	}
	return SessionInfo{ID: sessionID, Status: "desconhecida"}
}

func (c *Client) ensureSession(sessionID string) *sessionState {
	c.mu.Lock()
	defer c.mu.Unlock()
	if s, ok := c.sessions[sessionID]; ok {
		if s.status == "" {
			s.status = "desconectado"
		}
		return s
	}
	s := &sessionState{id: sessionID, dbPath: c.sessionDBPath(sessionID), status: "desconectado"}
	c.sessions[sessionID] = s
	return s
}

func (c *Client) sessionDBPath(sessionID string) string {
	if sessionID == "" || sessionID == c.defaultSession {
		return c.baseDBPath
	}
	clean := sanitizeSessionID(sessionID)
	ext := filepath.Ext(c.baseDBPath)
	if ext == "" {
		ext = ".db"
	}
	base := strings.TrimSuffix(c.baseDBPath, filepath.Ext(c.baseDBPath))
	return base + "-" + clean + ext
}

func (c *Client) setStatus(sessionID, status, qrCode, phone string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	s, ok := c.sessions[sessionID]
	if !ok {
		s = &sessionState{id: sessionID, dbPath: c.sessionDBPath(sessionID)}
		c.sessions[sessionID] = s
	}
	s.status = status
	s.qrCode = qrCode
	if phone != "" {
		s.phone = phone
	}
	s.connected = status == "conectado"
}

func sanitizeSessionID(sessionID string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(sessionID) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "session"
	}
	return b.String()
}

// limparTelefone normalizes a phone number for WhatsApp.
func limparTelefone(tel string) string {
	// Remove everything that's not a digit
	var clean strings.Builder
	for _, r := range tel {
		if r >= '0' && r <= '9' {
			clean.WriteRune(r)
		}
	}
	phone := clean.String()

	if len(phone) < 10 {
		return ""
	}

	// If starts with 0, assume local (Brazil) and prepend 55
	if strings.HasPrefix(phone, "0") {
		phone = "55" + phone[1:]
	}

	// If doesn't start with country code, assume Brazil
	if !strings.HasPrefix(phone, "55") && len(phone) <= 11 {
		phone = "55" + phone
	}

	return phone
}

// printQR renders a basic QR representation for terminal.
func printQR(code string) {
	// Simple text-based QR display
	short := code
	if len(short) > 20 {
		short = short[:20] + "..."
	}
	fmt.Println("┌─────────────────────────────────────┐")
	fmt.Println("│                                     │")
	fmt.Printf("│  QR Code: %-26s │\n", short)
	fmt.Println("│                                     │")
	fmt.Println("│  Use seu celular para escanear:     │")
	fmt.Println("│  WhatsApp > Dispositivos Conectados │")
	fmt.Println("│  > Conectar Dispositivo             │")
	fmt.Println("│                                     │")
	fmt.Printf("│  Ou acesse: wa.me/qr/%-15s │\n", "")
	fmt.Println("│                                     │")
	fmt.Println("└─────────────────────────────────────┘")
	fmt.Println()
	// Also write the full code for QR generators
	fmt.Fprintf(os.Stderr, "[whatsapp] QR raw: %s\n", code)
}
