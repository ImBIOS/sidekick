// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
package main

import (
	"encoding/json"
	"time"
)

// Host-facing scheduler state. Hosts (build-daemon, sim-daemon, Linux
// emulator hosts) are trusted infrastructure: they see resource IDs across
// orgs but only through narrow claim/result calls. Agents stay org-scoped.

// --- claims ---

// Claim is one unit of work handed to a host poll.
type Claim struct {
	Kind   string        `json:"kind"` // "build" | "sim" | "emu" | "sim-action"
	Build  *Build        `json:"build,omitempty"`
	Sim    *Sim          `json:"sim,omitempty"`
	Emu    *Emu          `json:"emu,omitempty"`
	Action *QueuedAction `json:"action,omitempty"`
}

// ClaimNext returns the oldest actionable claim for a host in pool+region,
// preferring queued builds, then sims/emus awaiting provisioning, then
// pending sim actions. kinds filters (empty = all). A resource pinned to a
// region is only claimable there; unpinned resources go to whoever claims
// first and stick to that region. Claimed resources bind to pool.
func (s *Store) ClaimNext(pool, region string, kinds []string) *Claim {
	s.mu.Lock()
	defer s.mu.Unlock()
	want := func(k string) bool {
		if len(kinds) == 0 {
			return true
		}
		for _, w := range kinds {
			if w == k {
				return true
			}
		}
		return false
	}
	if want("build") {
		var oldest *Build
		for _, b := range s.builds {
			if b.Status != StatusQueued || !regionMatch(b.Region, region) {
				continue
			}
			if oldest == nil || b.CreatedAt.Before(oldest.CreatedAt) {
				oldest = b
			}
		}
		if oldest != nil {
			oldest.Status = StatusRunning
			oldest.Host = pool
			oldest.Region = region
			oldest.Log = append(oldest.Log, "[sidekick] build claimed by "+pool)
			return &Claim{Kind: "build", Build: oldest}
		}
	}
	if want("sim") {
		var oldest *Sim
		for _, sim := range s.sims {
			if sim.Status != StatusProvisioning || sim.Host != "" || !regionMatch(sim.Region, region) {
				continue
			}
			if oldest == nil || sim.CreatedAt.Before(oldest.CreatedAt) {
				oldest = sim
			}
		}
		if oldest != nil {
			oldest.Host = pool
			oldest.Region = region
			return &Claim{Kind: "sim", Sim: oldest}
		}
	}
	if want("emu") {
		var oldest *Emu
		for _, emu := range s.emus {
			if emu.Status != StatusProvisioning || emu.Host != "" || !regionMatch(emu.Region, region) {
				continue
			}
			if oldest == nil || emu.CreatedAt.Before(oldest.CreatedAt) {
				oldest = emu
			}
		}
		if oldest != nil {
			oldest.Host = pool
			oldest.Region = region
			return &Claim{Kind: "emu", Emu: oldest}
		}
	}
	if want("sim-action") {
		var oldest *QueuedAction
		for _, sim := range s.sims {
			if sim.Host != pool || !regionMatch(sim.Region, region) {
				continue
			}
			for _, a := range sim.Actions {
				if !a.Done && (oldest == nil || a.CreatedAt.Before(oldest.CreatedAt)) {
					oldest = a
				}
			}
		}
		if oldest != nil {
			return &Claim{Kind: "sim-action", Action: oldest}
		}
	}
	return nil
}

// HostSimReady marks a claimed sim running with its live URLs.
func (s *Store) HostSimReady(id, host, webrtcURL string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	sim, ok := s.sims[id]
	if !ok || (sim.Host != "" && sim.Host != host) {
		return false
	}
	sim.Host = host
	sim.Status = StatusRunning
	sim.WebRTCURL = webrtcURL
	return true
}

// HostEmuReady marks a claimed emu running with its adb target.
func (s *Store) HostEmuReady(id, host, adbTarget string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	emu, ok := s.emus[id]
	if !ok || (emu.Host != "" && emu.Host != host) {
		return false
	}
	emu.Host = host
	emu.Status = StatusRunning
	emu.ADBTarget = adbTarget
	return true
}

// --- sim actions ---

// EnqueueSimAction records an agent verb for the daemon owning the sim's pool.
func (s *Store) EnqueueSimAction(org OrgID, simID string, body map[string]string) (*QueuedAction, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sim, ok := s.sims[simID]
	if !ok || sim.Org != org {
		return nil, false
	}
	a := &QueuedAction{
		ID:        s.nextID("act"),
		SimID:     simID,
		Body:      body,
		CreatedAt: time.Now().UTC(),
	}
	sim.Actions = append(sim.Actions, a)
	return a, true
}

// CompleteSimAction records the daemon's result for one action.
func (s *Store) CompleteSimAction(id, result string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sim := range s.sims {
		for _, a := range sim.Actions {
			if a.ID == id {
				a.Done = true
				a.Result = result
				return true
			}
		}
	}
	return false
}

// GetSimAction returns an action only if its sim belongs to org.
func (s *Store) GetSimAction(org OrgID, id string) (*QueuedAction, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sim := range s.sims {
		if sim.Org != org {
			continue
		}
		for _, a := range sim.Actions {
			if a.ID == id {
				return a, true
			}
		}
	}
	return nil, false
}

// --- sim device state ---

// SetSimTree stores the daemon-reported AX tree (raw JSON, size-capped by caller).
func (s *Store) SetSimTree(id string, raw json.RawMessage) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	sim, ok := s.sims[id]
	if !ok {
		return false
	}
	sim.Tree = raw
	sim.TreeUpdated = time.Now().UTC()
	return true
}

// SetSimShot stores the latest screenshot PNG.
func (s *Store) SetSimShot(id string, png []byte) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	sim, ok := s.sims[id]
	if !ok {
		return false
	}
	sim.Shot = png
	sim.ShotUpdated = time.Now().UTC()
	return true
}

// AppendSimLog appends daemon log lines.
func (s *Store) AppendSimLog(id string, lines []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	sim, ok := s.sims[id]
	if !ok {
		return false
	}
	sim.SimLog = append(sim.SimLog, lines...)
	return true
}

// SetSimRecording flips the recording flag; daemons drain record actions.
func (s *Store) SetSimRecording(id string, rec bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	sim, ok := s.sims[id]
	if !ok {
		return false
	}
	sim.Recording = rec
	return true
}

// SetSimVideo stores the finished recording MP4.
func (s *Store) SetSimVideo(id string, mp4 []byte) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	sim, ok := s.sims[id]
	if !ok {
		return false
	}
	sim.Video = mp4
	sim.Recording = false
	return true
}

// --- emu device state ---

// SetEmuTree stores the daemon-reported UI tree (raw JSON).
func (s *Store) SetEmuTree(id string, raw json.RawMessage) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	emu, ok := s.emus[id]
	if !ok {
		return false
	}
	emu.Tree = raw
	return true
}

// SetEmuShot stores the latest screenshot PNG.
func (s *Store) SetEmuShot(id string, png []byte) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	emu, ok := s.emus[id]
	if !ok {
		return false
	}
	emu.Shot = png
	return true
}

// AppendEmuLog appends daemon log lines.
func (s *Store) AppendEmuLog(id string, lines []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	emu, ok := s.emus[id]
	if !ok {
		return false
	}
	emu.EmuLog = append(emu.EmuLog, lines...)
	return true
}

// regionMatch: unpinned resources ("") are claimable anywhere; pinned ones
// only in their region.
func regionMatch(resource, host string) bool {
	return resource == "" || resource == host
}
