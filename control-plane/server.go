package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Server wires auth + store into the REST API.
type Server struct {
	verifier Verifier
	store    *Store
}

// NewServer builds the API with the given key verifier and scheduler.
func NewServer(v Verifier, s *Store) *Server {
	return &Server{verifier: v, store: s}
}

type ctxKey struct{}

// withOrg stores the verified org in the request context.
func withOrg(ctx context.Context, org OrgID) context.Context {
	return context.WithValue(ctx, ctxKey{}, org)
}

// orgFromContext returns the verified org for the request.
func orgFromContext(r *http.Request) OrgID {
	org, _ := r.Context().Value(ctxKey{}).(OrgID)
	return org
}

// withAuth enforces `Authorization: Bearer <sk_live_...>` and resolves the
// org before any handler runs. 401 on missing/invalid keys.
func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := bearerKey(r.Header.Get("Authorization"))
		if key == "" {
			writeErr(w, http.StatusUnauthorized, "missing bearer API key")
			return
		}
		org, err := s.verifier.Verify(key)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "invalid API key")
			return
		}
		next(w, r.WithContext(withOrg(r.Context(), org)))
	}
}

func bearerKey(h string) string {
	if h == "" {
		return ""
	}
	scheme, token, ok := strings.Cut(h, " ")
	if !ok || !strings.EqualFold(scheme, "bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}

// Routes registers all endpoints on mux.
func (s *Server) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/whoami", s.withAuth(s.handleWhoami))

	mux.HandleFunc("POST /v1/builds", s.withAuth(s.handleCreateBuild))
	mux.HandleFunc("GET /v1/builds", s.withAuth(s.handleListBuilds))
	mux.HandleFunc("GET /v1/builds/{id}", s.withAuth(s.handleGetBuild))
	mux.HandleFunc("GET /v1/builds/{id}/logs", s.withAuth(s.handleBuildLogs))

	mux.HandleFunc("POST /v1/sims", s.withAuth(s.handleCreateSim))
	mux.HandleFunc("GET /v1/sims", s.withAuth(s.handleListSims))
	mux.HandleFunc("GET /v1/sims/{id}", s.withAuth(s.handleGetSim))
	mux.HandleFunc("DELETE /v1/sims/{id}", s.withAuth(s.handleDeleteSim))
	mux.HandleFunc("POST /v1/sims/{id}/actions", s.withAuth(s.handleSimAction))

	mux.HandleFunc("POST /v1/emus", s.withAuth(s.handleCreateEmu))
	mux.HandleFunc("GET /v1/emus", s.withAuth(s.handleListEmus))
	mux.HandleFunc("GET /v1/emus/{id}", s.withAuth(s.handleGetEmu))
	mux.HandleFunc("DELETE /v1/emus/{id}", s.withAuth(s.handleDeleteEmu))
}

// GET /v1/whoami — lets `sidekick login` verify a key and learn its org.
func (s *Server) handleWhoami(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"org_id": string(orgFromContext(r))})
}

type createBuildReq struct {
	Scheme string `json:"scheme"`
	Ref    string `json:"ref"`
}

// POST /v1/builds — queue a remote Xcode/Gradle build for the caller's org.
func (s *Server) handleCreateBuild(w http.ResponseWriter, r *http.Request) {
	var req createBuildReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(req.Scheme) == "" {
		writeErr(w, http.StatusBadRequest, "scheme is required")
		return
	}
	writeJSON(w, http.StatusCreated, s.store.CreateBuild(orgFromContext(r), req.Scheme, req.Ref))
}

// GET /v1/builds — list the caller's org builds.
func (s *Server) handleListBuilds(w http.ResponseWriter, r *http.Request) {
	builds := s.store.ListBuilds(orgFromContext(r))
	if builds == nil {
		builds = []*Build{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"builds": builds})
}

// GET /v1/builds/{id} — fetch one build (404 when missing or foreign).
func (s *Server) handleGetBuild(w http.ResponseWriter, r *http.Request) {
	b, ok := s.store.GetBuild(orgFromContext(r), r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "build not found")
		return
	}
	writeJSON(w, http.StatusOK, b)
}

// GET /v1/builds/{id}/logs — SSE stream of build log lines, or
// ?format=json for a single {"lines":[...],"status":...} document.
func (s *Server) handleBuildLogs(w http.ResponseWriter, r *http.Request) {
	b, ok := s.store.GetBuild(orgFromContext(r), r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "build not found")
		return
	}
	if r.URL.Query().Get("format") == "json" {
		lines := b.Log
		if lines == nil {
			lines = []string{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"lines": lines, "status": b.Status})
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	for _, line := range b.Log {
		fmt.Fprintf(w, "data: %s\n\n", line)
	}
	fmt.Fprintf(w, "event: status\ndata: %s\n\n", b.Status)
}

type createSimReq struct {
	Device  string `json:"device"`
	Runtime string `json:"runtime"`
}

// POST /v1/sims — provision an iOS simulator session.
func (s *Server) handleCreateSim(w http.ResponseWriter, r *http.Request) {
	var req createSimReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(req.Device) == "" {
		writeErr(w, http.StatusBadRequest, "device is required")
		return
	}
	writeJSON(w, http.StatusCreated, s.store.CreateSim(orgFromContext(r), req.Device, req.Runtime))
}

// GET /v1/sims — list the caller's org sims.
func (s *Server) handleListSims(w http.ResponseWriter, r *http.Request) {
	sims := s.store.ListSims(orgFromContext(r))
	if sims == nil {
		sims = []*Sim{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"sims": sims})
}

// GET /v1/sims/{id} — fetch one sim (404 when missing or foreign).
func (s *Server) handleGetSim(w http.ResponseWriter, r *http.Request) {
	sim, ok := s.store.GetSim(orgFromContext(r), r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "sim not found")
		return
	}
	writeJSON(w, http.StatusOK, sim)
}

// DELETE /v1/sims/{id} — stop a sim; streaming minutes bill until stopped.
func (s *Server) handleDeleteSim(w http.ResponseWriter, r *http.Request) {
	if !s.store.DeleteSim(orgFromContext(r), r.PathValue("id")) {
		writeErr(w, http.StatusNotFound, "sim not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type simActionReq struct {
	Tap   string         `json:"tap"`
	Type  string         `json:"type"`
	Swipe string         `json:"swipe"`
	Batch []simActionReq `json:"batch"`
}

// POST /v1/sims/{id}/actions — tap/type/swipe, or a batch for ms precision.
// The sim-daemon applies these against the AX tree; the control-plane records
// receipt here so agents get a synchronous ack.
func (s *Server) handleSimAction(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.store.GetSim(orgFromContext(r), r.PathValue("id")); !ok {
		writeErr(w, http.StatusNotFound, "sim not found")
		return
	}
	var req simActionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	n := len(req.Batch)
	if n == 0 {
		n = 1
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "applied": n})
}

type createEmuReq struct {
	Profile string `json:"profile"`
}

// POST /v1/emus — provision an Android emulator session.
func (s *Server) handleCreateEmu(w http.ResponseWriter, r *http.Request) {
	var req createEmuReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(req.Profile) == "" {
		writeErr(w, http.StatusBadRequest, "profile is required")
		return
	}
	writeJSON(w, http.StatusCreated, s.store.CreateEmu(orgFromContext(r), req.Profile))
}

// GET /v1/emus — list the caller's org emus.
func (s *Server) handleListEmus(w http.ResponseWriter, r *http.Request) {
	emus := s.store.ListEmus(orgFromContext(r))
	if emus == nil {
		emus = []*Emu{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"emus": emus})
}

// GET /v1/emus/{id} — fetch one emu (404 when missing or foreign).
func (s *Server) handleGetEmu(w http.ResponseWriter, r *http.Request) {
	emu, ok := s.store.GetEmu(orgFromContext(r), r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "emu not found")
		return
	}
	writeJSON(w, http.StatusOK, emu)
}

// DELETE /v1/emus/{id} — stop an emu.
func (s *Server) handleDeleteEmu(w http.ResponseWriter, r *http.Request) {
	if !s.store.DeleteEmu(orgFromContext(r), r.PathValue("id")) {
		writeErr(w, http.StatusNotFound, "emu not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
