// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

var errUnauthorized = errors.New("unauthorized: invalid or missing API key")

// Resource lifecycle states shared by builds, sims, and emus.
const (
	StatusQueued       = "queued"
	StatusProvisioning = "provisioning"
	StatusRunning      = "running"
	StatusSucceeded    = "succeeded"
	StatusFailed       = "failed"
	StatusStopped      = "stopped"
)

// Build is a remote xcodebuild/gradle run claimed by a Mac/Linux host.
type Build struct {
	ID        string    `json:"build_id"`
	Org       OrgID     `json:"-"`
	Scheme    string    `json:"scheme"`
	Ref       string    `json:"ref,omitempty"`
	Status    string    `json:"status"`
	Host      string    `json:"host,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Log       []string  `json:"-"`
}

// Sim is a cloud iOS simulator session.
type Sim struct {
	ID        string    `json:"id"`
	Org       OrgID     `json:"-"`
	Device    string    `json:"device"`
	Runtime   string    `json:"runtime,omitempty"`
	Status    string    `json:"status"`
	WebRTCURL string    `json:"webrtc_url,omitempty"`
	AXTreeURL string    `json:"ax_tree_url,omitempty"`
	Host      string    `json:"host,omitempty"`
	Recording bool      `json:"recording"`
	CreatedAt time.Time `json:"created_at"`
	// Daemon-reported state, served via dedicated endpoints (never inlined).
	Tree        json.RawMessage `json:"-"`
	TreeUpdated time.Time       `json:"-"`
	Shot        []byte          `json:"-"`
	ShotUpdated time.Time       `json:"-"`
	SimLog      []string        `json:"-"`
	Video       []byte          `json:"-"`
	Actions     []*QueuedAction `json:"-"`
}

// QueuedAction is one agent verb awaiting the sim-daemon that owns the sim's
// pool. The control-plane never executes verbs; it queues, and the daemon
// posts back a result the agent polls for.
type QueuedAction struct {
	ID        string            `json:"action_id"`
	SimID     string            `json:"sim_id"`
	Body      map[string]string `json:"body"`
	Done      bool              `json:"done"`
	Result    string            `json:"result,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

// Emu is a cloud Android emulator session with an adb tunnel.
type Emu struct {
	ID        string    `json:"id"`
	Org       OrgID     `json:"-"`
	Profile   string    `json:"profile"`
	Status    string    `json:"status"`
	ADBTarget string    `json:"adb_target,omitempty"`
	Host      string    `json:"host,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	// Daemon-reported state, served via dedicated endpoints (never inlined).
	Tree   json.RawMessage `json:"-"`
	Shot   []byte          `json:"-"`
	EmuLog []string        `json:"-"`
}

// Store is the in-memory scheduler state. Every lookup is org-scoped: a key
// from org A can never see, mutate, or delete org B's resources.
// Persistence (Postgres) is a follow-up; the HTTP contract is stable.
type Store struct {
	mu     sync.Mutex
	seq    uint64
	builds map[string]*Build
	sims   map[string]*Sim
	emus   map[string]*Emu
	tokens map[string]*InstanceToken // by secret hash
}

// NewStore returns an empty scheduler.
func NewStore() *Store {
	return &Store{
		builds: map[string]*Build{},
		sims:   map[string]*Sim{},
		emus:   map[string]*Emu{},
		tokens: map[string]*InstanceToken{},
	}
}

func (s *Store) nextID(prefix string) string {
	s.seq++
	return prefix + "_" + time.Now().UTC().Format("20060102") + "-" + itoa(s.seq)
}

// --- builds ---

// CreateBuild queues a build for org. A build-daemon claims it via
// /v1/host/claims/next, streams logs, and marks it terminal.
func (s *Store) CreateBuild(org OrgID, scheme, ref string) *Build {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := &Build{
		ID:        s.nextID("bld"),
		Org:       org,
		Scheme:    scheme,
		Ref:       ref,
		Status:    StatusQueued,
		CreatedAt: time.Now().UTC(),
		Log:       []string{"[sidekick] queued; waiting for a build host"},
	}
	s.builds[b.ID] = b
	return b
}

// GetBuild returns a build only if it belongs to org.
func (s *Store) GetBuild(org OrgID, id string) (*Build, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.builds[id]
	if !ok || b.Org != org {
		return nil, false
	}
	return b, true
}

// ListBuilds returns org's builds, newest first.
func (s *Store) ListBuilds(org OrgID) []*Build {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*Build
	for _, b := range s.builds {
		if b.Org == org {
			out = append(out, b)
		}
	}
	return out
}

// AppendBuildLog appends host log lines (build-daemon calls this).
func (s *Store) AppendBuildLog(org OrgID, id string, lines []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.builds[id]
	if !ok || b.Org != org {
		return false
	}
	b.Log = append(b.Log, lines...)
	return true
}

// FinishBuild marks a build terminal.
func (s *Store) FinishBuild(org OrgID, id, status string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.builds[id]
	if !ok || b.Org != org {
		return false
	}
	b.Status = status
	return true
}

// HostAppendBuildLog appends daemon lines without org scoping (trusted host).
func (s *Store) HostAppendBuildLog(id string, lines []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.builds[id]
	if !ok {
		return false
	}
	b.Log = append(b.Log, lines...)
	return true
}

// HostFinishBuild marks a build terminal without org scoping (trusted host).
func (s *Store) HostFinishBuild(id, status string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.builds[id]
	if !ok {
		return false
	}
	b.Status = status
	return true
}

// --- sims ---

// CreateSim provisions a simulator session for org. It starts provisioning;
// the sim-daemon binds it to a host and marks it running with live URLs.
func (s *Store) CreateSim(org OrgID, device, runtime string) *Sim {
	s.mu.Lock()
	defer s.mu.Unlock()
	sim := &Sim{
		ID:        s.nextID("sim"),
		Org:       org,
		Device:    device,
		Runtime:   runtime,
		Status:    StatusProvisioning,
		CreatedAt: time.Now().UTC(),
	}
	sim.AXTreeURL = "/v1/sims/" + sim.ID + "/tree"
	s.sims[sim.ID] = sim
	return sim
}

// GetSim returns a sim only if it belongs to org.
func (s *Store) GetSim(org OrgID, id string) (*Sim, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sim, ok := s.sims[id]
	if !ok || sim.Org != org {
		return nil, false
	}
	return sim, true
}

// ListSims returns org's sims.
func (s *Store) ListSims(org OrgID) []*Sim {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*Sim
	for _, sim := range s.sims {
		if sim.Org == org {
			out = append(out, sim)
		}
	}
	return out
}

// DeleteSim stops and removes a sim; returns false when missing or foreign.
func (s *Store) DeleteSim(org OrgID, id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	sim, ok := s.sims[id]
	if !ok || sim.Org != org {
		return false
	}
	delete(s.sims, id)
	return true
}

// --- emus ---

// CreateEmu provisions an emulator session for org. It starts provisioning;
// the Linux host binds it and reports the adb target when running.
func (s *Store) CreateEmu(org OrgID, profile string) *Emu {
	s.mu.Lock()
	defer s.mu.Unlock()
	emu := &Emu{
		ID:        s.nextID("emu"),
		Org:       org,
		Profile:   profile,
		Status:    StatusProvisioning,
		CreatedAt: time.Now().UTC(),
	}
	s.emus[emu.ID] = emu
	return emu
}

// GetEmu returns an emu only if it belongs to org.
func (s *Store) GetEmu(org OrgID, id string) (*Emu, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	emu, ok := s.emus[id]
	if !ok || emu.Org != org {
		return nil, false
	}
	return emu, true
}

// ListEmus returns org's emus.
func (s *Store) ListEmus(org OrgID) []*Emu {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*Emu
	for _, emu := range s.emus {
		if emu.Org == org {
			out = append(out, emu)
		}
	}
	return out
}

// DeleteEmu stops and removes an emu; returns false when missing or foreign.
func (s *Store) DeleteEmu(org OrgID, id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	emu, ok := s.emus[id]
	if !ok || emu.Org != org {
		return false
	}
	delete(s.emus, id)
	return true
}

// --- instance tokens ---

func hashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// CreateInstanceToken mints a scoped, expiring token. The secret is returned
// once and never stored; only its hash persists.
func (s *Store) CreateInstanceToken(org OrgID, name string, scopes []string, ttl time.Duration) (*InstanceToken, string, bool) {
	if name == "" || len(scopes) == 0 || ttl <= 0 || ttl > 30*24*time.Hour {
		return nil, "", false
	}
	seen := map[string]bool{}
	for _, sc := range scopes {
		if !validScope(sc) || seen[sc] {
			return nil, "", false
		}
		seen[sc] = true
	}
	secret, hash, err := mintSecret()
	if err != nil {
		return nil, "", false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	tok := &InstanceToken{
		ID:        s.nextID("tok"),
		Org:       org,
		Name:      name,
		Scopes:    scopes,
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
	}
	s.tokens[hash] = tok
	return tok, secret, true
}

// LookupInstanceToken resolves a presented secret to its token.
func (s *Store) LookupInstanceToken(secret string) (*InstanceToken, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tok, ok := s.tokens[hashSecret(secret)]
	if !ok || tok.Revoked || time.Now().UTC().After(tok.ExpiresAt) {
		return nil, false
	}
	return tok, true
}

// ListInstanceTokens returns org's tokens (secrets never leave Create).
func (s *Store) ListInstanceTokens(org OrgID) []*InstanceToken {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*InstanceToken
	for _, tok := range s.tokens {
		if tok.Org == org {
			out = append(out, tok)
		}
	}
	return out
}

// RevokeInstanceToken kills a token immediately.
func (s *Store) RevokeInstanceToken(org OrgID, id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, tok := range s.tokens {
		if tok.ID == id && tok.Org == org {
			tok.Revoked = true
			return true
		}
	}
	return false
}

func itoa(n uint64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
