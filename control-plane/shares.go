// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
package main

// Signed guest shares: login-free read-only snapshot views of one sim/emu.
//
// A share is a random `sk_share_...` secret bound to exactly one resource.
// Anyone holding the URL `GET /s/:token` sees the latest screenshot
// (auto-refreshed), AX/UI tree, logs tail, and recorded video when present.
// No auth header, no org key, no WebRTC: this is explicitly NOT a live media
// stream — there is no SFU in this system (see README comparison "Live share
// links"). Mint/revoke are org-key-only; instance tokens are rejected (403).

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"html/template"
	"net/http"
	"time"
)

const sharePrefix = "sk_share_"

// GuestShare is one signed read-only view of a sim or emu. The secret is
// returned once at mint and never stored; only its sha256 hash persists.
type GuestShare struct {
	ID         string    `json:"id"`
	Org        OrgID     `json:"-"`
	Kind       string    `json:"kind"` // "sim" | "emu"
	ResourceID string    `json:"resource_id"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
	Revoked    bool      `json:"-"`
}

func mintShareSecret() (secret, hash string, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	secret = sharePrefix + hex.EncodeToString(raw)
	sum := sha256.Sum256([]byte(secret))
	return secret, hex.EncodeToString(sum[:]), nil
}

// --- in-memory Store ---

// CreateGuestShare mints a share for one resource the caller already proved
// it owns (handlers check GetSim/GetEmu first). TTL bounds mirror instance
// tokens: (0, 720h].
func (s *Store) CreateGuestShare(org OrgID, kind, resourceID string, ttl time.Duration) (*GuestShare, string, bool) {
	if (kind != "sim" && kind != "emu") || resourceID == "" || ttl <= 0 || ttl > 30*24*time.Hour {
		return nil, "", false
	}
	secret, hash, err := mintShareSecret()
	if err != nil {
		return nil, "", false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	sh := &GuestShare{
		ID:         s.nextID("shr"),
		Org:        org,
		Kind:       kind,
		ResourceID: resourceID,
		ExpiresAt:  now.Add(ttl),
		CreatedAt:  now,
	}
	if s.shares == nil {
		s.shares = map[string]*GuestShare{}
	}
	s.shares[hash] = sh
	return sh, secret, true
}

// LookupGuestShare resolves a presented share secret. Expired or revoked
// shares resolve as missing (callers 404 uniformly — no oracle).
func (s *Store) LookupGuestShare(secret string) (*GuestShare, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sh, ok := s.shares[hashSecret(secret)]
	if !ok || sh.Revoked || time.Now().UTC().After(sh.ExpiresAt) {
		return nil, false
	}
	return sh, true
}

// RevokeGuestShare kills a share immediately; org-scoped like everything else.
func (s *Store) RevokeGuestShare(org OrgID, id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sh := range s.shares {
		if sh.ID == id && sh.Org == org {
			sh.Revoked = true
			return true
		}
	}
	return false
}

// GetSimAny / GetEmuAny fetch without org scoping for public share serving.
// The share itself is the capability: it was minted by the owning org.
func (s *Store) GetSimAny(id string) (*Sim, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sim, ok := s.sims[id]
	return sim, ok
}

// GetEmuAny fetches an emu without org scoping (see GetSimAny).
func (s *Store) GetEmuAny(id string) (*Emu, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	emu, ok := s.emus[id]
	return emu, ok
}

// --- handlers (org-key-only mint/revoke) ---

type shareReq struct {
	TTL string `json:"ttl"`
}

func (s *Server) shareTTL(req shareReq) (time.Duration, bool) {
	if req.TTL == "" {
		return 24 * time.Hour, true
	}
	ttl, err := time.ParseDuration(req.TTL)
	if err != nil {
		return 0, false
	}
	return ttl, true
}

// instanceBlocked rejects sandboxed instance tokens from share management.
func instanceBlocked(w http.ResponseWriter, r *http.Request) bool {
	if az, ok := r.Context().Value(authzKey{}).(authz); ok && az.instance {
		writeErr(w, http.StatusForbidden, "instance tokens cannot manage shares")
		return true
	}
	return false
}

// POST /v1/sims/{id}/shares — mint a guest share (org key only).
func (s *Server) handleCreateSimShare(w http.ResponseWriter, r *http.Request) {
	if instanceBlocked(w, r) {
		return
	}
	org := orgFromContext(r)
	simID := r.PathValue("id")
	if _, ok := s.store.GetSim(org, simID); !ok {
		writeErr(w, http.StatusNotFound, "sim not found")
		return
	}
	var req shareReq
	if !decodeJSONBody(w, r, 64<<10, &req) {
		return
	}
	ttl, ok := s.shareTTL(req)
	if !ok {
		writeErr(w, http.StatusBadRequest, `ttl must parse like "24h" (default 24h, max 720h)`)
		return
	}
	sh, secret, ok := s.store.CreateGuestShare(org, "sim", simID, ttl)
	if !ok {
		writeErr(w, http.StatusBadRequest, "ttl must be positive and at most 720h")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id": sh.ID, "kind": "sim", "resource_id": simID,
		"share_token": secret, "expires_at": sh.ExpiresAt,
	})
}

// POST /v1/emus/{id}/shares — mint a guest share (org key only).
func (s *Server) handleCreateEmuShare(w http.ResponseWriter, r *http.Request) {
	if instanceBlocked(w, r) {
		return
	}
	org := orgFromContext(r)
	emuID := r.PathValue("id")
	if _, ok := s.store.GetEmu(org, emuID); !ok {
		writeErr(w, http.StatusNotFound, "emu not found")
		return
	}
	var req shareReq
	if !decodeJSONBody(w, r, 64<<10, &req) {
		return
	}
	ttl, ok := s.shareTTL(req)
	if !ok {
		writeErr(w, http.StatusBadRequest, `ttl must parse like "24h" (default 24h, max 720h)`)
		return
	}
	sh, secret, ok := s.store.CreateGuestShare(org, "emu", emuID, ttl)
	if !ok {
		writeErr(w, http.StatusBadRequest, "ttl must be positive and at most 720h")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id": sh.ID, "kind": "emu", "resource_id": emuID,
		"share_token": secret, "expires_at": sh.ExpiresAt,
	})
}

// DELETE /v1/sims/{id}/shares/{sid} + emu twin — revoke immediately.
func (s *Server) handleRevokeSimShare(w http.ResponseWriter, r *http.Request) {
	if instanceBlocked(w, r) {
		return
	}
	if _, ok := s.store.GetSim(orgFromContext(r), r.PathValue("id")); !ok {
		writeErr(w, http.StatusNotFound, "sim not found")
		return
	}
	if !s.store.RevokeGuestShare(orgFromContext(r), r.PathValue("sid")) {
		writeErr(w, http.StatusNotFound, "share not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleRevokeEmuShare revokes an emu share (see handleRevokeSimShare).
func (s *Server) handleRevokeEmuShare(w http.ResponseWriter, r *http.Request) {
	if instanceBlocked(w, r) {
		return
	}
	if _, ok := s.store.GetEmu(orgFromContext(r), r.PathValue("id")); !ok {
		writeErr(w, http.StatusNotFound, "emu not found")
		return
	}
	if !s.store.RevokeGuestShare(orgFromContext(r), r.PathValue("sid")) {
		writeErr(w, http.StatusNotFound, "share not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- public guest surface (no auth) ---

// resolveShare maps the path token to a live share or 404s uniformly.
func (s *Server) resolveShare(w http.ResponseWriter, r *http.Request) (*GuestShare, bool) {
	sh, ok := s.store.LookupGuestShare(r.PathValue("token"))
	if !ok {
		writeErr(w, http.StatusNotFound, "share not found or expired")
		return nil, false
	}
	return sh, true
}

type shareSnapshot struct {
	Kind      string
	ID        string
	Status    string
	Recording bool
	HasShot   bool
	HasTree   bool
	HasVideo  bool
	ExpiresAt time.Time
}

// snapshotForShare reads the resource behind a share. False when the
// resource was deleted after mint (renders as a tombstone, not a leak).
func (s *Server) snapshotForShare(sh *GuestShare) (*shareSnapshot, []byte, []string, bool) {
	if sh.Kind == "sim" {
		sim, ok := s.store.GetSimAny(sh.ResourceID)
		if !ok {
			return nil, nil, nil, false
		}
		return &shareSnapshot{
			Kind: "sim", ID: sim.ID, Status: sim.Status,
			Recording: sim.Recording, HasShot: len(sim.Shot) > 0,
			HasTree: len(sim.Tree) > 0, HasVideo: len(sim.Video) > 0,
			ExpiresAt: sh.ExpiresAt,
		}, sim.Tree, sim.SimLog, true
	}
	emu, ok := s.store.GetEmuAny(sh.ResourceID)
	if !ok {
		return nil, nil, nil, false
	}
	return &shareSnapshot{
		Kind: "emu", ID: emu.ID, Status: emu.Status,
		HasShot: len(emu.Shot) > 0, HasTree: len(emu.Tree) > 0,
		ExpiresAt: sh.ExpiresAt,
	}, emu.Tree, emu.EmuLog, true
}

var sharePage = template.Must(template.New("share").Parse(`<!doctype html>
<html><head><meta charset="utf-8">
<title>sidekick shared {{.Kind}} {{.ID}}</title>
<meta http-equiv="refresh" content="5">
<style>body{font-family:system-ui,sans-serif;max-width:900px;margin:2rem auto;padding:0 1rem;color:#111}
img{max-width:100%;border:1px solid #ddd;border-radius:8px}pre{background:#f6f6f6;padding:1rem;overflow:auto;max-height:40vh;border-radius:8px}
.meta{color:#555}.warn{background:#fff7e6;border:1px solid #f0c36d;padding:.75rem 1rem;border-radius:8px}</style>
</head><body>
<h1>shared {{.Kind}} <code>{{.ID}}</code></h1>
<p class="meta">status: <b>{{.Status}}</b>{{if .Recording}} · ● recording{{end}} · link expires {{.ExpiresAt}} · auto-refreshes every 5s</p>
{{if .Deleted}}<p class="warn">This session was deleted by its owner. The link is kept but there is nothing to show.</p>{{end}}
{{if .HasShot}}<h2>screenshot</h2><img src="screenshot" alt="latest screenshot">{{else}}<p class="meta">no screenshot yet</p>{{end}}
{{if .HasVideo}}<p><a href="video">download recording (mp4)</a></p>{{end}}
{{if .Tree}}<h2>UI tree</h2><pre>{{.Tree}}</pre>{{end}}
{{if .Logs}}<h2>logs (last {{.NLogs}})</h2><pre>{{.Logs}}</pre>{{end}}
<p class="meta">Read-only guest view. No login. Live video is not streamed — this page shows snapshots the host posts every refresh interval.</p>
</body></html>`))

type sharePageData struct {
	Kind      string
	ID        string
	Status    string
	Recording bool
	HasShot   bool
	HasVideo  bool
	ExpiresAt time.Time
	Deleted   bool
	Tree      string
	Logs      string
	NLogs     int
}

func truncateForPage(b []byte, n int) string {
	if len(b) > n {
		return string(b[:n]) + "\n…[truncated]"
	}
	return string(b)
}

func tailLines(lines []string, n int) ([]string, int) {
	if len(lines) > n {
		return lines[len(lines)-n:], n
	}
	return lines, len(lines)
}

// GET /s/{token} — public guest page (HTML), or JSON with ?format=json.
func (s *Server) handleGuestPage(w http.ResponseWriter, r *http.Request) {
	sh, ok := s.resolveShare(w, r)
	if !ok {
		return
	}
	snap, tree, logs, ok := s.snapshotForShare(sh)
	if !ok {
		// Minted, then the resource was deleted: tombstone, same shape.
		snap = &shareSnapshot{Kind: sh.Kind, ID: sh.ResourceID, Status: "deleted", ExpiresAt: sh.ExpiresAt}
	}
	if r.URL.Query().Get("format") == "json" {
		writeJSON(w, http.StatusOK, map[string]any{
			"kind": sh.Kind, "id": sh.ResourceID, "status": snap.Status,
			"recording": snap.Recording, "has_shot": snap.HasShot,
			"has_tree": snap.HasTree, "has_video": snap.HasVideo,
			"expires_at": sh.ExpiresAt,
		})
		return
	}
	tail, n := tailLines(logs, 50)
	data := sharePageData{
		Kind: sh.Kind, ID: sh.ResourceID, Status: snap.Status,
		Recording: snap.Recording, HasShot: snap.HasShot, HasVideo: snap.HasVideo,
		ExpiresAt: sh.ExpiresAt.UTC(), Deleted: !ok,
		Tree: truncateForPage(tree, 8192),
	}
	for _, l := range tail {
		data.Logs += l + "\n"
	}
	data.NLogs = n
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = sharePage.Execute(w, data)
}

// GET /s/{token}/screenshot — latest PNG (404 until the host posts one).
func (s *Server) handleGuestShot(w http.ResponseWriter, r *http.Request) {
	sh, ok := s.resolveShare(w, r)
	if !ok {
		return
	}
	if sh.Kind == "sim" {
		if sim, ok := s.store.GetSimAny(sh.ResourceID); ok && len(sim.Shot) > 0 {
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(sim.Shot)
			return
		}
	} else if emu, ok := s.store.GetEmuAny(sh.ResourceID); ok && len(emu.Shot) > 0 {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(emu.Shot)
		return
	}
	writeErr(w, http.StatusNotFound, "no screenshot yet")
}

// GET /s/{token}/tree — latest AX/UI tree JSON.
func (s *Server) handleGuestTree(w http.ResponseWriter, r *http.Request) {
	sh, ok := s.resolveShare(w, r)
	if !ok {
		return
	}
	var tree []byte
	if sh.Kind == "sim" {
		if sim, ok := s.store.GetSimAny(sh.ResourceID); ok {
			tree = sim.Tree
		}
	} else if emu, ok := s.store.GetEmuAny(sh.ResourceID); ok {
		tree = emu.Tree
	}
	if len(tree) == 0 {
		writeErr(w, http.StatusNotFound, "no UI tree yet")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(tree)
}

// GET /s/{token}/logs?tail=N — device log tail (JSON {"lines":[...]}).
func (s *Server) handleGuestLogs(w http.ResponseWriter, r *http.Request) {
	sh, ok := s.resolveShare(w, r)
	if !ok {
		return
	}
	var lines []string
	if sh.Kind == "sim" {
		if sim, ok := s.store.GetSimAny(sh.ResourceID); ok {
			lines = sim.SimLog
		}
	} else if emu, ok := s.store.GetEmuAny(sh.ResourceID); ok {
		lines = emu.EmuLog
	}
	if lines == nil {
		lines = []string{}
	}
	if n := atoiOr(r.URL.Query().Get("tail"), len(lines)); n < len(lines) {
		lines = lines[len(lines)-n:]
	}
	writeJSON(w, http.StatusOK, map[string]any{"lines": lines})
}

// GET /s/{token}/video — finished MP4 (sims only; emus 404 — their record
// path downloads over adb, never through the plane).
func (s *Server) handleGuestVideo(w http.ResponseWriter, r *http.Request) {
	sh, ok := s.resolveShare(w, r)
	if !ok {
		return
	}
	if sh.Kind != "sim" {
		writeErr(w, http.StatusNotFound, "emus have no plane-hosted video")
		return
	}
	sim, ok := s.store.GetSimAny(sh.ResourceID)
	if !ok || len(sim.Video) == 0 {
		writeErr(w, http.StatusNotFound, "no recording yet")
		return
	}
	w.Header().Set("Content-Type", "video/mp4")
	_, _ = w.Write(sim.Video)
}
