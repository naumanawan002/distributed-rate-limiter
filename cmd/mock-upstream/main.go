package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"sync/atomic"
)

func main() {
	port := flag.String("port", "8081", "Port to listen on")
	flag.Parse()

	var counter uint64

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		reqNum := atomic.AddUint64(&counter, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":         "ok",
			"request_number": reqNum,
			"path":           r.URL.Path,
			"method":         r.Method,
		})
	})

	addr := ":" + *port
	log.Printf("[INFO] Upstream mock service listening on %s\n", addr)
	_ = http.ListenAndServe(addr, nil)
}
