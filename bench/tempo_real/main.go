// Command tempo_real loads a real-time server with persistent connections
// (docs/FASE2_TEMPO_REAL.md): N subscribers on one channel, a few senders
// posting messages at a fixed rate, and it reports what each subscriber saw —
// delivery latency (from the moment the message was sent to its arrival),
// messages missing, out of order or repeated — and the server's memory.
//
// It speaks the protocol of bench/baseline/chat; the addresses are flags so
// the Germanio reference application of FASE 2 can be loaded the same way.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

type message struct {
	Seq   int64  `json:"seq"`
	Texto string `json:"texto"`
}

type stats struct {
	mu        sync.Mutex
	lat       []time.Duration
	missing   int64
	outOfOrd  int64
	repeated  int64
	delivered int64
	resyncs   int64
}

func main() {
	base := flag.String("servidor", "http://127.0.0.1:18090", "endereço do servidor")
	wsPath := flag.String("ws", "/ws?canal=geral", "endereço das conexões (relativo ao servidor)")
	postPath := flag.String("envio", "/canais/geral/mensagens", "endereço do envio de mensagens")
	conns := flag.Int("c", 1000, "conexões assinantes")
	senders := flag.Int("remetentes", 4, "remetentes simultâneos")
	total := flag.Int("mensagens", 2000, "mensagens no total")
	rate := flag.Int("taxa", 200, "mensagens por segundo (todas somadas)")
	pid := flag.Int("pid", 0, "processo do servidor (memória em /proc)")
	label := flag.String("rotulo", "", "nome do alvo no relatório")
	flag.Parse()

	wsURL := strings.Replace(*base, "http", "ws", 1) + *wsPath
	st := &stats{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	before := rss(*pid)
	var ready, connected sync.WaitGroup
	var dialErr atomic.Int64
	last := make([]atomic.Int64, *conns)
	for i := range *conns {
		ready.Add(1)
		connected.Add(1)
		go func(i int) {
			defer connected.Done()
			c, _, err := websocket.Dial(ctx, wsURL, nil)
			ready.Done()
			if err != nil {
				dialErr.Add(1)
				return
			}
			c.SetReadLimit(1 << 20)
			defer c.CloseNow()
			var prev int64
			for {
				_, b, err := c.Read(ctx)
				if err != nil {
					if websocket.CloseStatus(err) == websocket.StatusTryAgainLater {
						atomic.AddInt64(&st.resyncs, 1)
					}
					return
				}
				var m message
				if json.Unmarshal(b, &m) != nil {
					continue
				}
				sent, _ := strconv.ParseInt(m.Texto, 10, 64)
				d := time.Duration(time.Now().UnixNano() - sent)
				st.mu.Lock()
				switch {
				case m.Seq == prev+1 || prev == 0:
					st.delivered++
					st.lat = append(st.lat, d)
				case m.Seq <= prev:
					st.repeated++
				default:
					st.missing += m.Seq - prev - 1
					st.outOfOrd++
					st.delivered++
					st.lat = append(st.lat, d)
				}
				st.mu.Unlock()
				prev = m.Seq
				last[i].Store(prev)
			}
		}(i)
	}
	ready.Wait()
	time.Sleep(500 * time.Millisecond) // let every subscription register
	peak := atomic.Int64{}
	go func() {
		for ctx.Err() == nil {
			if kb := rss(*pid); kb > peak.Load() {
				peak.Store(kb)
			}
			time.Sleep(200 * time.Millisecond)
		}
	}()

	client := &http.Client{Timeout: 30 * time.Second}
	var postErr atomic.Int64
	interval := time.Second / time.Duration(max(*rate, 1))
	per := *total / max(*senders, 1)
	start := time.Now()
	var wg sync.WaitGroup
	for s := range *senders {
		wg.Add(1)
		go func(s int) {
			defer wg.Done()
			tick := time.NewTicker(interval * time.Duration(*senders))
			defer tick.Stop()
			for range per {
				<-tick.C
				body, _ := json.Marshal(map[string]string{"texto": strconv.FormatInt(time.Now().UnixNano(), 10)})
				resp, err := client.Post(*base+*postPath, "application/json", bytes.NewReader(body))
				if err != nil || resp.StatusCode >= 300 {
					postErr.Add(1)
				}
				if resp != nil {
					resp.Body.Close()
				}
			}
		}(s)
	}
	wg.Wait()
	sendTime := time.Since(start)
	time.Sleep(2 * time.Second) // let the last deliveries arrive
	cancel()
	connected.Wait()

	sent := int64(per * *senders)
	expected := sent * int64(*conns-int(dialErr.Load()))
	slices.Sort(st.lat)
	pct := func(p float64) time.Duration {
		if len(st.lat) == 0 {
			return 0
		}
		return st.lat[min(len(st.lat)-1, int(float64(len(st.lat))*p))]
	}
	fmt.Printf("alvo=%s conexoes=%d falhas_conexao=%d remetentes=%d enviadas=%d falhas_envio=%d duracao_envio=%s\n", *label, *conns, dialErr.Load(), *senders, sent, postErr.Load(), sendTime.Round(time.Millisecond))
	fmt.Printf("entregas=%d esperadas=%d faltando=%d fora_de_ordem=%d repetidas=%d ressincronizacoes=%d\n", st.delivered, expected, st.missing, st.outOfOrd, st.repeated, st.resyncs)
	fmt.Printf("latencia_entrega p50=%s p90=%s p99=%s max=%s\n", pct(0.50), pct(0.90), pct(0.99), pct(1))
	if *pid > 0 {
		fmt.Printf("memoria_servidor rss_antes=%dKB rss_pico=%dKB\n", before, peak.Load())
	}
}

func rss(pid int) int64 {
	if pid <= 0 {
		return 0
	}
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			n, _ := strconv.ParseInt(strings.Fields(line)[1], 10, 64)
			return n
		}
	}
	return 0
}
