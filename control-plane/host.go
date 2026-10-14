// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
package main

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
)

// Host API: trusted fleet daemons (build-daemon, sim-daemon, Linux emulator
// hosts) authenticate with the shared SIDEKICK_HOST_TOKEN bearer token —
// never with org API keys. These endpoints mutate cross-org resources by ID,
// which is exactly the daemons' job and exactly what org keys must not do.

const (
	maxTreeBytes  = 1 << 20  // 1MB AX/UI tree JSON
	maxShotBytes  = 4 << 20  // 4MB screenshot PNG
	maxVideoBytes = 32 << 20 // 32MB recording MP4
)

// withHost enforces the shared host token. Empty configured token rejects
// everything: an operator must opt into fleet traffic explicitly.
func (s *Server) withHost(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.hostToken == "" || subtle.ConstantTimeCompare([]byte(bearerKey(r.Header.Get("Authorization"))), []byte(s.hostToken)) != 1 {
			writeErr(w, http.StatusUnauthorized, "invalid host token")
			return
		}
		next(w, r)
	}
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, limit int64, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

type claimReq struct {
	Pool  string   `json:"pool"`
	Kinds []string `json:"kinds"`
}

// POST /v1/host/claims/next — oldest actionable claim for pool, 204 if none.
func (s *Server) handleHostClaim(w http.ResponseWriter, r *http.Request) {
	var req claimReq
	if !decodeJSONBody(w, r, 64<<10, &req) {
		return
	}
	if strings.TrimSpace(req.Pool) == "" {
		writeErr(w, http.StatusBadRequest, "pool is required")
		return
	}
	claim := s.store.ClaimNext(req.Pool, req.Kinds)
	if claim == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeJSON(w, http.StatusOK, claim)
}

type hostLinesReq struct {
	Lines []string `json:"lines"`
}

// POST /v1/host/builds/{id}/logs — daemon streams xcodebuild output.
func (s *Server) handleHostBuildLogs(w http.ResponseWriter, r *http.Request) {
	var req hostLinesReq
	if !decodeJSONBody(w, r, 1<<20, &req) {
		return
	}
	id := r.PathValue("id")
	if !s.store.HostAppendBuildLog(id, req.Lines) {
		writeErr(w, http.StatusNotFound, "build not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type hostFinishReq struct {
	Status string `json:"status"`
}

// POST /v1/host/builds/{id}/finish — mark a build succeeded/failed.
func (s *Server) handleHostBuildFinish(w http.ResponseWriter, r *http.Request) {
	var req hostFinishReq
	if !decodeJSONBody(w, r, 64<<10, &req) {
		return
	}
	if req.Status != StatusSucceeded && req.Status != StatusFailed {
		writeErr(w, http.StatusBadRequest, "status must be succeeded or failed")
		return
	}
	if !s.store.HostFinishBuild(r.PathValue("id"), req.Status) {
		writeErr(w, http.StatusNotFound, "build not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": req.Status})
}

type hostSimReadyReq struct {
	Host      string `json:"host"`
	WebRTCURL string `json:"webrtc_url"`
}

// POST /v1/host/sims/{id}/ready — daemon provisioned the simulator.
func (s *Server) handleHostSimReady(w http.ResponseWriter, r *http.Request) {
	var req hostSimReadyReq
	if !decodeJSONBody(w, r, 64<<10, &req) {
		return
	}
	if strings.TrimSpace(req.Host) == "" {
		writeErr(w, http.StatusBadRequest, "host is required")
		return
	}
	if !s.store.HostSimReady(r.PathValue("id"), req.Host, req.WebRTCURL) {
		writeErr(w, http.StatusNotFound, "sim not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type hostEmuReadyReq struct {
	Host      string `json:"host"`
	ADBTarget string `json:"adb_target"`
}

// POST /v1/host/emus/{id}/ready — Linux host booted the emulator.
func (s *Server) handleHostEmuReady(w http.ResponseWriter, r *http.Request) {
	var req hostEmuReadyReq
	if !decodeJSONBody(w, r, 64<<10, &req) {
		return
	}
	if strings.TrimSpace(req.Host) == "" || strings.TrimSpace(req.ADBTarget) == "" {
		writeErr(w, http.StatusBadRequest, "host and adb_target are required")
		return
	}
	if !s.store.HostEmuReady(r.PathValue("id"), req.Host, req.ADBTarget) {
		writeErr(w, http.StatusNotFound, "emu not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// putRawBody reads a JSON {data_b64} envelope with a byte cap and returns the
// decoded payload. Used for tree JSON, PNG screenshots, and MP4 recordings.
func putRawBody(w http.ResponseWriter, r *http.Request, capBytes int64) ([]byte, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, capBytes*4/3+1024)
	var req struct {
		DataB64 string `json:"data_b64"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return nil, false
	}
	raw, err := base64.StdEncoding.DecodeString(req.DataB64)
	if err != nil || int64(len(raw)) > capBytes {
		writeErr(w, http.StatusBadRequest, "invalid or oversized data_b64")
		return nil, false
	}
	return raw, true
}

// PUT /v1/host/sims/{id}/tree — daemon posts the AX tree JSON.
func (s *Server) handleHostSimTree(w http.ResponseWriter, r *http.Request) {
	raw, ok := putRawBody(w, r, maxTreeBytes)
	if !ok {
		return
	}
	if !json.Valid(raw) {
		writeErr(w, http.StatusBadRequest, "data_b64 is not valid JSON")
		return
	}
	if !s.store.SetSimTree(r.PathValue("id"), raw) {
		writeErr(w, http.StatusNotFound, "sim not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// PUT /v1/host/sims/{id}/screenshot — daemon posts the latest PNG.
func (s *Server) handleHostSimShot(w http.ResponseWriter, r *http.Request) {
	raw, ok := putRawBody(w, r, maxShotBytes)
	if !ok {
		return
	}
	if !s.store.SetSimShot(r.PathValue("id"), raw) {
		writeErr(w, http.StatusNotFound, "sim not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "bytes": len(raw)})
}

// POST /v1/host/sims/{id}/logs — daemon streams device logs.
func (s *Server) handleHostSimLogs(w http.ResponseWriter, r *http.Request) {
	var req hostLinesReq
	if !decodeJSONBody(w, r, 1<<20, &req) {
		return
	}
	if !s.store.AppendSimLog(r.PathValue("id"), req.Lines) {
		writeErr(w, http.StatusNotFound, "sim not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// POST /v1/host/sims/{id}/recording — daemon posts the finished MP4.
func (s *Server) handleHostSimVideo(w http.ResponseWriter, r *http.Request) {
	raw, ok := putRawBody(w, r, maxVideoBytes)
	if !ok {
		return
	}
	if !s.store.SetSimVideo(r.PathValue("id"), raw) {
		writeErr(w, http.StatusNotFound, "sim not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "bytes": len(raw)})
}

type hostActionResultReq struct {
	Result string `json:"result"`
}

// POST /v1/host/actions/{id}/result — daemon reports an applied verb.
func (s *Server) handleHostActionResult(w http.ResponseWriter, r *http.Request) {
	var req hostActionResultReq
	if !decodeJSONBody(w, r, 64<<10, &req) {
		return
	}
	if !s.store.CompleteSimAction(r.PathValue("id"), req.Result) {
		writeErr(w, http.StatusNotFound, "action not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// PUT /v1/host/emus/{id}/tree — Linux host posts the UI tree JSON.
func (s *Server) handleHostEmuTree(w http.ResponseWriter, r *http.Request) {
	raw, ok := putRawBody(w, r, maxTreeBytes)
	if !ok {
		return
	}
	if !json.Valid(raw) {
		writeErr(w, http.StatusBadRequest, "data_b64 is not valid JSON")
		return
	}
	if !s.store.SetEmuTree(r.PathValue("id"), raw) {
		writeErr(w, http.StatusNotFound, "emu not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// PUT /v1/host/emus/{id}/screenshot — Linux host posts the latest PNG.
func (s *Server) handleHostEmuShot(w http.ResponseWriter, r *http.Request) {
	raw, ok := putRawBody(w, r, maxShotBytes)
	if !ok {
		return
	}
	if !s.store.SetEmuShot(r.PathValue("id"), raw) {
		writeErr(w, http.StatusNotFound, "emu not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "bytes": len(raw)})
}

// POST /v1/host/emus/{id}/logs — Linux host streams logcat lines.
func (s *Server) handleHostEmuLogs(w http.ResponseWriter, r *http.Request) {
	var req hostLinesReq
	if !decodeJSONBody(w, r, 1<<20, &req) {
		return
	}
	if !s.store.AppendEmuLog(r.PathValue("id"), req.Lines) {
		writeErr(w, http.StatusNotFound, "emu not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
