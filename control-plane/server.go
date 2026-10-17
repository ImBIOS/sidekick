// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Server wires auth + store into the REST API.
type Server struct {
	verifier  Verifier
	store     Scheduler
	hostToken string
}

// Scheduler is the persistence surface both backends implement: the
// in-memory Store (dev, tests) and PGStore (DATABASE_URL set).
type Scheduler interface {
	CreateBuild(org OrgID, scheme, ref, region string) *Build
	GetBuild(org OrgID, id string) (*Build, bool)
	ListBuilds(org OrgID) []*Build
	AppendBuildLog(org OrgID, id string, lines []string) bool
	FinishBuild(org OrgID, id, status string) bool
	HostAppendBuildLog(id string, lines []string) bool
	HostFinishBuild(id, status string) bool
	CreateSim(org OrgID, device, runtime, region string) *Sim
	GetSim(org OrgID, id string) (*Sim, bool)
	ListSims(org OrgID) []*Sim
	DeleteSim(org OrgID, id string) bool
	HostSimReady(id, host, webrtcURL string) bool
	SetSimTree(id string, raw json.RawMessage) bool
	SetSimShot(id string, png []byte) bool
	AppendSimLog(id string, lines []string) bool
	SetSimRecording(id string, rec bool) bool
	SetSimVideo(id string, mp4 []byte) bool
	EnqueueSimAction(org OrgID, simID string, body map[string]string) (*QueuedAction, bool)
	CompleteSimAction(id, result string) bool
	GetSimAction(org OrgID, id string) (*QueuedAction, bool)
	CreateEmu(org OrgID, profile, region string) *Emu
	GetEmu(org OrgID, id string) (*Emu, bool)
	ListEmus(org OrgID) []*Emu
	DeleteEmu(org OrgID, id string) bool
	HostEmuReady(id, host, adbTarget string) bool
	SetEmuTree(id string, raw json.RawMessage) bool
	SetEmuShot(id string, png []byte) bool
	AppendEmuLog(id string, lines []string) bool
	ClaimNext(pool, region string, kinds []string) *Claim
	CreateInstanceToken(org OrgID, name string, scopes []string, ttl time.Duration) (*InstanceToken, string, bool)
	LookupInstanceToken(secret string) (*InstanceToken, bool)
	ListInstanceTokens(org OrgID) []*InstanceToken
	RevokeInstanceToken(org OrgID, id string) bool
}

// NewServer builds the API with the given key verifier and scheduler.
func NewServer(v Verifier, s Scheduler) *Server {
	return &Server{verifier: v, store: s}
}

// WithHostToken sets the shared fleet token for /v1/host/* endpoints.
// Empty (default) rejects all host traffic.
func (s *Server) WithHostToken(tok string) *Server {
	s.hostToken = tok
	return s
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
	mux.HandleFunc("GET /v1/whoami", s.withAuthz(s.handleWhoami))

	mux.HandleFunc("POST /v1/tokens", s.withAuthz(s.handleCreateToken))
	mux.HandleFunc("GET /v1/tokens", s.withAuthz(s.handleListTokens))
	mux.HandleFunc("DELETE /v1/tokens/{id}", s.withAuthz(s.handleRevokeToken))

	mux.HandleFunc("POST /v1/builds", s.withAuthz(s.handleCreateBuild))
	mux.HandleFunc("GET /v1/builds", s.withAuthz(s.handleListBuilds))
	mux.HandleFunc("GET /v1/builds/{id}", s.withAuthz(s.handleGetBuild))
	mux.HandleFunc("GET /v1/builds/{id}/logs", s.withAuthz(s.handleBuildLogs))

	mux.HandleFunc("POST /v1/sims", s.withAuthz(s.handleCreateSim))
	mux.HandleFunc("GET /v1/sims", s.withAuthz(s.handleListSims))
	mux.HandleFunc("GET /v1/sims/{id}", s.withAuthz(s.handleGetSim))
	mux.HandleFunc("DELETE /v1/sims/{id}", s.withAuthz(s.handleDeleteSim))
	mux.HandleFunc("POST /v1/sims/{id}/actions", s.withAuthz(s.handleSimAction))
	mux.HandleFunc("GET /v1/sims/{id}/actions/{aid}", s.withAuthz(s.handleGetSimAction))
	mux.HandleFunc("GET /v1/sims/{id}/tree", s.withAuthz(s.handleGetSimTree))
	mux.HandleFunc("GET /v1/sims/{id}/screenshot", s.withAuthz(s.handleGetSimShot))
	mux.HandleFunc("GET /v1/sims/{id}/logs", s.withAuthz(s.handleGetSimLogs))
	mux.HandleFunc("POST /v1/sims/{id}/record", s.withAuthz(s.handleSimRecord))
	mux.HandleFunc("GET /v1/sims/{id}/recording", s.withAuthz(s.handleGetSimRecording))
	mux.HandleFunc("GET /v1/sims/{id}/recording/download", s.withAuthz(s.handleGetSimVideo))

	mux.HandleFunc("POST /v1/emus", s.withAuthz(s.handleCreateEmu))
	mux.HandleFunc("GET /v1/emus", s.withAuthz(s.handleListEmus))
	mux.HandleFunc("GET /v1/emus/{id}", s.withAuthz(s.handleGetEmu))
	mux.HandleFunc("DELETE /v1/emus/{id}", s.withAuthz(s.handleDeleteEmu))
	mux.HandleFunc("GET /v1/emus/{id}/tree", s.withAuthz(s.handleGetEmuTree))
	mux.HandleFunc("GET /v1/emus/{id}/screenshot", s.withAuthz(s.handleGetEmuShot))
	mux.HandleFunc("GET /v1/emus/{id}/logs", s.withAuthz(s.handleGetEmuLogs))

	mux.HandleFunc("POST /v1/host/claims/next", s.withHost(s.handleHostClaim))
	mux.HandleFunc("POST /v1/host/builds/{id}/logs", s.withHost(s.handleHostBuildLogs))
	mux.HandleFunc("POST /v1/host/builds/{id}/finish", s.withHost(s.handleHostBuildFinish))
	mux.HandleFunc("POST /v1/host/sims/{id}/ready", s.withHost(s.handleHostSimReady))
	mux.HandleFunc("PUT /v1/host/sims/{id}/tree", s.withHost(s.handleHostSimTree))
	mux.HandleFunc("PUT /v1/host/sims/{id}/screenshot", s.withHost(s.handleHostSimShot))
	mux.HandleFunc("POST /v1/host/sims/{id}/logs", s.withHost(s.handleHostSimLogs))
	mux.HandleFunc("POST /v1/host/sims/{id}/recording", s.withHost(s.handleHostSimVideo))
	mux.HandleFunc("POST /v1/host/actions/{id}/result", s.withHost(s.handleHostActionResult))
	mux.HandleFunc("POST /v1/host/emus/{id}/ready", s.withHost(s.handleHostEmuReady))
	mux.HandleFunc("PUT /v1/host/emus/{id}/tree", s.withHost(s.handleHostEmuTree))
	mux.HandleFunc("PUT /v1/host/emus/{id}/screenshot", s.withHost(s.handleHostEmuShot))
	mux.HandleFunc("POST /v1/host/emus/{id}/logs", s.withHost(s.handleHostEmuLogs))
}

// GET /v1/whoami — lets `sidekick login` verify a key and learn its org.
func (s *Server) handleWhoami(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"org_id": string(orgFromContext(r))})
}

type createBuildReq struct {
	Scheme string `json:"scheme"`
	Ref    string `json:"ref"`
	Region string `json:"region"`
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
	writeJSON(w, http.StatusCreated, s.store.CreateBuild(orgFromContext(r), req.Scheme, req.Ref, req.Region))
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
	Region  string `json:"region"`
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
	writeJSON(w, http.StatusCreated, s.store.CreateSim(orgFromContext(r), req.Device, req.Runtime, req.Region))
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

// POST /v1/sims/{id}/actions — queue tap/type/swipe (or a batch) for the
// sim-daemon owning the sim's pool. Returns the queued action IDs; agents
// poll GET /v1/sims/{id}/actions/{aid} for the daemon's result.
func (s *Server) handleSimAction(w http.ResponseWriter, r *http.Request) {
	org := orgFromContext(r)
	simID := r.PathValue("id")
	var req simActionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	bodies := []map[string]string{}
	if len(req.Batch) > 0 {
		for _, b := range req.Batch {
			m := map[string]string{}
			if b.Tap != "" {
				m["tap"] = b.Tap
			}
			if b.Type != "" {
				m["type"] = b.Type
			}
			if b.Swipe != "" {
				m["swipe"] = b.Swipe
			}
			if len(m) > 0 {
				bodies = append(bodies, m)
			}
		}
	} else {
		m := map[string]string{}
		if req.Tap != "" {
			m["tap"] = req.Tap
		}
		if req.Type != "" {
			m["type"] = req.Type
		}
		if req.Swipe != "" {
			m["swipe"] = req.Swipe
		}
		if len(m) > 0 {
			bodies = append(bodies, m)
		}
	}
	if len(bodies) == 0 {
		writeErr(w, http.StatusBadRequest, "give one of tap, type, swipe, or batch")
		return
	}
	ids := make([]string, 0, len(bodies))
	for _, body := range bodies {
		a, ok := s.store.EnqueueSimAction(org, simID, body)
		if !ok {
			writeErr(w, http.StatusNotFound, "sim not found")
			return
		}
		ids = append(ids, a.ID)
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true, "queued": len(ids), "action_ids": ids})
}

// GET /v1/sims/{id}/actions/{aid} — poll one queued verb for completion.
func (s *Server) handleGetSimAction(w http.ResponseWriter, r *http.Request) {
	a, ok := s.store.GetSimAction(orgFromContext(r), r.PathValue("aid"))
	if !ok {
		writeErr(w, http.StatusNotFound, "action not found")
		return
	}
	writeJSON(w, http.StatusOK, a)
}

// GET /v1/sims/{id}/tree — latest daemon-reported AX tree (404 until posted).
func (s *Server) handleGetSimTree(w http.ResponseWriter, r *http.Request) {
	sim, ok := s.store.GetSim(orgFromContext(r), r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "sim not found")
		return
	}
	if len(sim.Tree) == 0 {
		writeErr(w, http.StatusNotFound, "no AX tree yet; the sim may still be provisioning")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(sim.Tree)
}

// GET /v1/sims/{id}/screenshot — latest daemon-posted PNG (404 until posted).
func (s *Server) handleGetSimShot(w http.ResponseWriter, r *http.Request) {
	sim, ok := s.store.GetSim(orgFromContext(r), r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "sim not found")
		return
	}
	if len(sim.Shot) == 0 {
		writeErr(w, http.StatusNotFound, "no screenshot yet")
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Write(sim.Shot)
}

// GET /v1/sims/{id}/logs?format=json&tail=N — daemon device logs.
func (s *Server) handleGetSimLogs(w http.ResponseWriter, r *http.Request) {
	sim, ok := s.store.GetSim(orgFromContext(r), r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "sim not found")
		return
	}
	lines := sim.SimLog
	if lines == nil {
		lines = []string{}
	}
	if n := atoiOr(r.URL.Query().Get("tail"), len(lines)); n < len(lines) {
		lines = lines[len(lines)-n:]
	}
	writeJSON(w, http.StatusOK, map[string]any{"lines": lines})
}

type recordReq struct {
	Op string `json:"op"`
}

// POST /v1/sims/{id}/record {"op":"start"|"stop"} — toggle session recording.
// Stop returns the download path; the MP4 lands there once the daemon posts it.
func (s *Server) handleSimRecord(w http.ResponseWriter, r *http.Request) {
	var req recordReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	org := orgFromContext(r)
	simID := r.PathValue("id")
	switch req.Op {
	case "start":
		if _, ok := s.store.EnqueueSimAction(org, simID, map[string]string{"record": "start"}); !ok {
			writeErr(w, http.StatusNotFound, "sim not found")
			return
		}
		s.store.SetSimRecording(simID, true)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "recording": true})
	case "stop":
		if _, ok := s.store.EnqueueSimAction(org, simID, map[string]string{"record": "stop"}); !ok {
			writeErr(w, http.StatusNotFound, "sim not found")
			return
		}
		s.store.SetSimRecording(simID, false)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "recording": false, "video_url": "/v1/sims/" + simID + "/recording/download"})
	default:
		writeErr(w, http.StatusBadRequest, `op must be "start" or "stop"`)
	}
}

// GET /v1/sims/{id}/recording — recording flag + video availability.
func (s *Server) handleGetSimRecording(w http.ResponseWriter, r *http.Request) {
	sim, ok := s.store.GetSim(orgFromContext(r), r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "sim not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"recording": sim.Recording,
		"has_video": len(sim.Video) > 0,
		"video_url": "/v1/sims/" + sim.ID + "/recording/download",
	})
}

// GET /v1/sims/{id}/recording/download — the finished MP4 (404 until posted).
func (s *Server) handleGetSimVideo(w http.ResponseWriter, r *http.Request) {
	sim, ok := s.store.GetSim(orgFromContext(r), r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "sim not found")
		return
	}
	if len(sim.Video) == 0 {
		writeErr(w, http.StatusNotFound, "no recording yet")
		return
	}
	w.Header().Set("Content-Type", "video/mp4")
	w.Write(sim.Video)
}

type createEmuReq struct {
	Profile string `json:"profile"`
	Region  string `json:"region"`
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
	writeJSON(w, http.StatusCreated, s.store.CreateEmu(orgFromContext(r), req.Profile, req.Region))
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

// GET /v1/emus/{id}/tree — latest daemon-reported UI tree (404 until posted).
func (s *Server) handleGetEmuTree(w http.ResponseWriter, r *http.Request) {
	emu, ok := s.store.GetEmu(orgFromContext(r), r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "emu not found")
		return
	}
	if len(emu.Tree) == 0 {
		writeErr(w, http.StatusNotFound, "no UI tree yet; the emu may still be provisioning")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(emu.Tree)
}

// GET /v1/emus/{id}/screenshot — latest daemon-posted PNG (404 until posted).
func (s *Server) handleGetEmuShot(w http.ResponseWriter, r *http.Request) {
	emu, ok := s.store.GetEmu(orgFromContext(r), r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "emu not found")
		return
	}
	if len(emu.Shot) == 0 {
		writeErr(w, http.StatusNotFound, "no screenshot yet")
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Write(emu.Shot)
}

// GET /v1/emus/{id}/logs?tail=N — daemon logcat lines.
func (s *Server) handleGetEmuLogs(w http.ResponseWriter, r *http.Request) {
	emu, ok := s.store.GetEmu(orgFromContext(r), r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "emu not found")
		return
	}
	lines := emu.EmuLog
	if lines == nil {
		lines = []string{}
	}
	if n := atoiOr(r.URL.Query().Get("tail"), len(lines)); n < len(lines) {
		lines = lines[len(lines)-n:]
	}
	writeJSON(w, http.StatusOK, map[string]any{"lines": lines})
}

func atoiOr(s string, dflt int) int {	if s == "" {
		return dflt
	}
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return dflt
		}
		n = n*10 + int(ch-'0')
	}
	return n
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
