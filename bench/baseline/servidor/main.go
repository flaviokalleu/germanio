// Command servidor serves the Go-direct baseline over HTTP so that the stress
// tool can load it exactly like a Germanio application.
package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/flaviokalleu/germanio/bench/baseline"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:18081", "endereço")
	db := flag.String("db", "baseline.db", "arquivo SQLite")
	flag.Parse()
	h, closeDB, err := baseline.New(*db)
	if err != nil {
		log.Fatal(err)
	}
	defer closeDB()
	srv := &http.Server{Addr: *addr, Handler: h, ReadHeaderTimeout: 10 * time.Second}
	log.Fatal(srv.ListenAndServe())
}
