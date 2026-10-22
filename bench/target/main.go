// Command target is a minimal, fixed-cost HTTP server used as the neutral
// benchmark target for bench/compare.sh. It serves a 1 KiB body on every path.
package main

import (
	"flag"
	"log"
	"net/http"
	"strings"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8098", "listen address")
	flag.Parse()

	body := []byte(strings.Repeat("x", 1024))
	log.Fatal(http.ListenAndServe(*addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	})))
}
