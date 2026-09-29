// Command chat is the Go-direct baseline of FASE 2 (docs/FASE2_TEMPO_REAL.md):
// the smallest correct real-time chat, written the way a Go developer would,
// to compare Germanio against. Channels, one sequence per channel, messages
// stored in SQLite, fan-out over WebSocket, resumption from a cursor
// (?desde=N) and a bounded buffer per connection: a client too slow to keep up
// is closed with a "ressincronize" reason instead of losing messages in
// silence.
//
//	POST /canais/{canal}/mensagens  {"texto": "..."}  → 201 {"seq": N}
//	GET  /ws?canal=C&desde=N                           → the history after N, then live
//
// Every message sent to clients is {"canal":…, "seq":…, "texto":…}.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/coder/websocket"
	_ "modernc.org/sqlite"
)

const bufferPerConn = 256

type message struct {
	Canal string `json:"canal"`
	Seq   int64  `json:"seq"`
	Texto string `json:"texto"`
}

type sub struct {
	send chan []byte
	slow chan struct{} // closed when the buffer overflows: resync
	once sync.Once
}

type channel struct {
	mu   sync.Mutex
	seq  int64
	subs map[*sub]bool
}

type server struct {
	db       *sql.DB
	mu       sync.Mutex
	channels map[string]*channel
}

func (s *server) channel(name string) (*channel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c := s.channels[name]; c != nil {
		return c, nil
	}
	c := &channel{subs: map[*sub]bool{}}
	if err := s.db.QueryRow(`SELECT COALESCE(MAX(seq), 0) FROM mensagem WHERE canal = ?`, name).Scan(&c.seq); err != nil {
		return nil, err
	}
	s.channels[name] = c
	return c, nil
}

func (s *server) post(w http.ResponseWriter, r *http.Request) {
	var body struct{ Texto string }
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&body); err != nil {
		http.Error(w, `{"message":"JSON inválido"}`, http.StatusBadRequest)
		return
	}
	c, err := s.channel(r.PathValue("canal"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// the channel lock gives one total order: sequence, storage and fan-out
	c.mu.Lock()
	m := message{Canal: r.PathValue("canal"), Seq: c.seq + 1, Texto: body.Texto}
	if _, err := s.db.Exec(`INSERT INTO mensagem (canal, seq, texto) VALUES (?, ?, ?)`, m.Canal, m.Seq, m.Texto); err != nil {
		c.mu.Unlock()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	c.seq = m.Seq
	b, _ := json.Marshal(m)
	for su := range c.subs {
		select {
		case su.send <- b:
		default:
			su.once.Do(func() { close(su.slow) })
			delete(c.subs, su)
		}
	}
	c.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int64{"seq": m.Seq})
}

func (s *server) ws(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("canal")
	since, _ := strconv.ParseInt(r.URL.Query().Get("desde"), 10, 64)
	c, err := s.channel(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer conn.CloseNow()
	su := &sub{send: make(chan []byte, bufferPerConn), slow: make(chan struct{})}
	// history and registration under the channel lock: nothing is missed or
	// repeated between the backlog and the live stream
	c.mu.Lock()
	rows, err := s.db.Query(`SELECT seq, texto FROM mensagem WHERE canal = ? AND seq > ? ORDER BY seq`, name, since)
	var backlog [][]byte
	if err == nil {
		for rows.Next() {
			m := message{Canal: name}
			rows.Scan(&m.Seq, &m.Texto)
			b, _ := json.Marshal(m)
			backlog = append(backlog, b)
		}
		rows.Close()
	}
	c.subs[su] = true
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.subs, su)
		c.mu.Unlock()
	}()
	ctx := conn.CloseRead(r.Context())
	for _, b := range backlog {
		if conn.Write(ctx, websocket.MessageText, b) != nil {
			return
		}
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-su.slow:
			conn.Close(websocket.StatusTryAgainLater, "ressincronize")
			return
		case b := <-su.send:
			wctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := conn.Write(wctx, websocket.MessageText, b)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

func main() {
	addr := flag.String("addr", "127.0.0.1:18090", "endereço")
	path := flag.String("db", "chat.db", "arquivo SQLite")
	flag.Parse()
	db, err := sql.Open("sqlite", "file:"+*path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_txlock=immediate")
	if err != nil {
		log.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS mensagem (id INTEGER PRIMARY KEY, canal TEXT NOT NULL, seq INTEGER NOT NULL, texto TEXT NOT NULL, UNIQUE (canal, seq))`); err != nil {
		log.Fatal(err)
	}
	s := &server{db: db, channels: map[string]*channel{}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /canais/{canal}/mensagens", s.post)
	mux.HandleFunc("GET /ws", s.ws)
	srv := &http.Server{Addr: *addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	log.Fatal(srv.ListenAndServe())
}
