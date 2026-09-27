package main

import (
	"fmt"
	"net/http"
)

// Control plane: org API keys -> instance claims -> per-instance tokens.
// Fronted by web/ (TanStack Start + tRPC) for console/billing;
// this Go service owns scheduling + Mac/Linux dispatch.
func main() {
	http.HandleFunc("/v1/builds", func(w http.ResponseWriter, r *http.Request) {
		// TODO: auth org key, pick least-loaded Mac host with warm NVMe cache, return SSE
		fmt.Fprintln(w, `{"build_id":"bld_123","status":"queued"}`)
	})
	http.HandleFunc("/v1/sims", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"id":"sim_abc","status":"provisioning"}`)
	})
	http.HandleFunc("/v1/emus", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"id":"emu_abc","status":"provisioning"}`)
	})
	fmt.Println("openlim control-plane :8080")
	_ = http.ListenAndServe(":8080", nil)
}
