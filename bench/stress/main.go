// Command stress puts a running server under many simultaneous connections
// and reports throughput, latency percentiles, errors and (with -pid) the
// server's memory. It is a closed-loop generator: each connection sends the
// next request when the previous one answers, so latency under overload is
// underestimated (coordinated omission) — use it to compare Germanio with Go
// direct on the same machine, not as an absolute capacity number.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	url := flag.String("url", "http://127.0.0.1:18080/_ge/api/clientes", "endereço a carregar")
	conns := flag.Int("c", 1000, "conexões simultâneas")
	dur := flag.Duration("d", 20*time.Second, "duração")
	pid := flag.Int("pid", 0, "processo do servidor (para medir a memória em /proc)")
	label := flag.String("rotulo", "", "nome do alvo no relatório")
	flag.Parse()

	tr := &http.Transport{
		MaxIdleConns:        *conns,
		MaxIdleConnsPerHost: *conns,
		MaxConnsPerHost:     *conns,
		IdleConnTimeout:     time.Minute,
		DialContext:         (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
	}
	client := &http.Client{Transport: tr, Timeout: 30 * time.Second}

	ctx, stop := context.WithTimeout(context.Background(), *dur)
	defer stop()
	var ok, failed atomic.Int64
	errs := sync.Map{}
	lat := make([][]time.Duration, *conns)
	var peak atomic.Int64
	go func() { // sample the server's resident memory
		for ctx.Err() == nil {
			if kb := rss(*pid, "VmRSS"); kb > peak.Load() {
				peak.Store(kb)
			}
			time.Sleep(200 * time.Millisecond)
		}
	}()
	before := rss(*pid, "VmRSS")
	start := time.Now()
	var wg sync.WaitGroup
	for i := range *conns {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ctx.Err() == nil {
				t := time.Now()
				resp, err := client.Get(*url)
				if err != nil {
					if ctx.Err() == nil {
						failed.Add(1)
						errs.Store(short(err), true)
					}
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					failed.Add(1)
					errs.Store(fmt.Sprint("status ", resp.StatusCode), true)
					continue
				}
				ok.Add(1)
				lat[i] = append(lat[i], time.Since(t))
			}
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	var all []time.Duration
	for _, l := range lat {
		all = append(all, l...)
	}
	slices.Sort(all)
	p := func(q float64) time.Duration {
		if len(all) == 0 {
			return 0
		}
		return all[min(len(all)-1, int(q*float64(len(all))))]
	}
	fmt.Printf("alvo=%s url=%s conexoes=%d duracao=%s go=%s cpus=%d\n", *label, *url, *conns, *dur, runtime.Version(), runtime.NumCPU())
	fmt.Printf("respostas_ok=%d erros=%d req_por_s=%.0f\n", ok.Load(), failed.Load(), float64(ok.Load())/elapsed.Seconds())
	fmt.Printf("latencia p50=%s p90=%s p99=%s max=%s\n", p(0.50), p(0.90), p(0.99), p(1))
	if *pid != 0 {
		fmt.Printf("memoria_servidor rss_antes=%dKB rss_pico=%dKB rss_depois=%dKB\n", before, peak.Load(), rss(*pid, "VmRSS"))
	}
	errs.Range(func(k, _ any) bool { fmt.Println("erro:", k); return true })
}

// rss reads a memory line (VmRSS, VmHWM) of /proc/<pid>/status in kB.
func rss(pid int, key string) int64 {
	if pid == 0 {
		return 0
	}
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, key+":") {
			var kb int64
			fmt.Sscan(strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, key+":"), "kB")), &kb)
			return kb
		}
	}
	return 0
}

func short(err error) string {
	s := err.Error()
	if i := strings.LastIndex(s, ": "); i >= 0 {
		return s[i+2:]
	}
	return s
}
