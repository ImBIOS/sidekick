// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
package main

import (
	"fmt"
	"net/http"
	"os"
)

// Control plane: org API keys -> instance claims -> per-instance dispatch.
// Fronted by web/ (TanStack Start + tRPC) for console/billing; this Go
// service owns scheduling + Mac/Linux dispatch.
//
// Env:
//
//	SIDEKICK_API_KEYS     static "org:key,..." pairs (self-host/dev)
//	SIDEKICK_AUTH_URL     web server base URL for hosted key verification
//	CONTROL_PLANE_SECRET  shared secret for /internal/verify-key
//	SIDEKICK_HOST_TOKEN   shared bearer token for fleet daemons (/v1/host/*)
//	PORT                  listen port (default 8080)
func main() {
	srv := NewServer(verifierFromEnv(), NewStore()).WithHostToken(os.Getenv("SIDEKICK_HOST_TOKEN"))
	mux := http.NewServeMux()
	srv.Routes(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Println("sidekick control-plane :" + port)
	_ = http.ListenAndServe(":"+port, mux)
}
