// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"testing"
	"time"
)

// This file proves the scheduler is safe under concurrent fleet load:
// ClaimNext hands each unit of work to exactly one host, instance tokens
// stay valid under concurrent mint/lookup/revoke, and region pinning holds
// when hosts race. Run it with the race detector:
//
//	go test -race -run 'Concurrent|RegionPinE2E|LoadNumbers' ./control-plane/ -v
//
// Load numbers are printed by TestSchedulerLoadNumbers (and BenchmarkClaimNext)
// so CI logs keep a record per run. Numbers from the reference machine are
// in docs/ARCH.md; re-run on your fleet hardware before promising p95s.

// TestConcurrentClaimNextNoDoubleClaim seeds 200 builds and races 16 hosts.
// Every build must be claimed exactly once; afterwards the queue is empty.
func TestConcurrentClaimNextNoDoubleClaim(t *testing.T) {
	s := NewStore()
	const builds = 200
	for i := 0; i < builds; i++ {
		s.CreateBuild("org_a", "MyApp", "", "")
	}
	const workers = 16
	var wg sync.WaitGroup
	mu := sync.Mutex{}
	claimed := map[string]string{} // build ID -> pool
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			pool := fmt.Sprintf("mac-%d", w)
			for {
				c := s.ClaimNext(pool, "", []string{"build"})
				if c == nil {
					return
				}
				mu.Lock()
				if prev, dup := claimed[c.Build.ID]; dup {
					t.Errorf("build %s double-claimed by %s and %s", c.Build.ID, prev, pool)
				}
				claimed[c.Build.ID] = pool
				mu.Unlock()
			}
		}(w)
	}
	wg.Wait()
	if len(claimed) != builds {
		t.Fatalf("claimed %d builds, want %d", len(claimed), builds)
	}
	if c := s.ClaimNext("mac-99", "", []string{"build"}); c != nil {
		t.Fatalf("drained queue returned %+v", c)
	}
}

// TestConcurrentTokensRace mints tokens while readers look them up and a
// revoker kills them. Under -race any unsynchronized map access fails the
// run; the assertions below check the semantics held.
func TestConcurrentTokensRace(t *testing.T) {
	s := NewStore()
	const n = 64
	secrets := make([]string, n)
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		tok, secret, ok := s.CreateInstanceToken("org_a", fmt.Sprintf("run-%d", i), []string{"sims", "emus"}, time.Hour)
		if !ok {
			t.Fatal("mint failed")
		}
		secrets[i], ids[i] = secret, tok.ID
	}
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(3)
		go func(i int) { defer wg.Done(); _, _ = s.LookupInstanceToken(secrets[i]) }(i)
		go func(i int) { defer wg.Done(); _ = s.ListInstanceTokens("org_a") }(i)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				_ = s.RevokeInstanceToken("org_a", ids[i])
			}
		}(i)
	}
	wg.Wait()
	// Even-index tokens revoked, odd-index survivors must still resolve.
	for i := 0; i < n; i++ {
		_, ok := s.LookupInstanceToken(secrets[i])
		if i%2 == 1 && !ok {
			t.Errorf("surviving token %d stopped resolving", i)
		}
		if i%2 == 0 && ok {
			t.Errorf("revoked token %d still resolves", i)
		}
	}
}

// TestRegionPinE2EConcurrent is the full agent→host path over HTTP: an org
// key mints an instance token, the token creates pinned + unpinned emus,
// and racing hosts in two regions each get exactly what pinning allows.
// It exercises tokens + region pin + ClaimNext together, the combination
// Limrun documents as jurisdiction pinning.
func TestRegionPinE2EConcurrent(t *testing.T) {
	srv := testHostServer()

	// Mint a token scoped to emus.
	rec := authed(t, srv, http.MethodPost, "/v1/tokens", "key-a",
		`{"name":"e2e","scopes":["emus"],"ttl":"1h"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("mint: %d %s", rec.Code, rec.Body.String())
	}
	var minted struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &minted); err != nil {
		t.Fatal(err)
	}

	// Token creates 10 Tokyo-pinned + 10 unpinned emus.
	const perRegion = 10
	for i := 0; i < perRegion; i++ {
		if rec := authed(t, srv, http.MethodPost, "/v1/emus", minted.Token, `{"profile":"P","region":"asia-northeast1"}`); rec.Code != http.StatusCreated {
			t.Fatalf("pinned create: %d", rec.Code)
		}
		if rec := authed(t, srv, http.MethodPost, "/v1/emus", minted.Token, `{"profile":"P"}`); rec.Code != http.StatusCreated {
			t.Fatalf("unpinned create: %d", rec.Code)
		}
	}

	// Race 4 Singapore hosts against 4 Tokyo hosts.
	type result struct {
		region string
		emuID  string
		pinned string
	}
	results := make(chan result, 2*perRegion)
	var wg sync.WaitGroup
	claimLoop := func(pool, region string) {
		defer wg.Done()
		for {
			rec := hosted(t, srv, http.MethodPost, "/v1/host/claims/next", testHostToken,
				`{"pool":"`+pool+`","region":"`+region+`","kinds":["emu"]}`)
			if rec.Code == http.StatusNoContent {
				return
			}
			if rec.Code != http.StatusOK {
				t.Errorf("claim %s: %d %s", region, rec.Code, rec.Body.String())
				return
			}
			var c Claim
			if err := json.Unmarshal(rec.Body.Bytes(), &c); err != nil {
				t.Errorf("decode %s: %v", region, err)
				return
			}
			results <- result{region: region, emuID: c.Emu.ID, pinned: c.Emu.Region}
		}
	}
	for i := 0; i < 4; i++ {
		wg.Add(2)
		go claimLoop(fmt.Sprintf("sg-%d", i), "asia-southeast1")
		go claimLoop(fmt.Sprintf("tokyo-%d", i), "asia-northeast1")
	}
	wg.Wait()
	close(results)

	seen := map[string]bool{}
	sgPinned := 0
	for r := range results {
		if seen[r.emuID] {
			t.Fatalf("emu %s claimed twice", r.emuID)
		}
		seen[r.emuID] = true
		// Pinned emus are only claimable in Tokyo; unpinned ones stick to
		// whichever region claims them first (ClaimNext regionMatch), so an
		// unpinned emu claimed from Tokyo reads back as Tokyo-pinned.
		// The invariant is: every Tokyo-pinned emu was claimed in Tokyo.
		if r.region == "asia-southeast1" && r.pinned == "asia-northeast1" {
			sgPinned++
		}
	}
	if len(seen) != 2*perRegion {
		t.Fatalf("claimed %d emus, want %d", len(seen), 2*perRegion)
	}
	if sgPinned != 0 {
		t.Errorf("singapore hosts stole %d tokyo-pinned emus", sgPinned)
	}
}

// TestFleetRegionsHeartbeat proves the liveness map is honest: regions only
// appear after a host actually polls from them, empty polls (204) still
// count as alive, queued depth is org-scoped, and cross-org queues don't
// leak into another org's map.
func TestFleetRegionsHeartbeat(t *testing.T) {
	srv := testHostServer()

	// Empty fleet: no heartbeats, no rows — never fake regions.
	rec := authed(t, srv, http.MethodGet, "/v1/fleet/regions", "key-a", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("fleet: %d %s", rec.Code, rec.Body.String())
	}
	var empty struct {
		Regions []RegionStatus `json:"regions"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &empty); err != nil || len(empty.Regions) != 0 {
		t.Fatalf("empty fleet should list zero regions: %q %v", rec.Body.String(), err)
	}

	// One host poll per region, both 204 (nothing queued). Even misses record
	// liveness — a host that phones home is alive whether or not it got work.
	for _, tc := range []struct{ pool, region string }{
		{"sg-1", "asia-southeast1"},
		{"sg-2", "asia-southeast1"},
		{"tokyo-1", "asia-northeast1"},
	} {
		rec := hosted(t, srv, http.MethodPost, "/v1/host/claims/next", testHostToken,
			`{"pool":"`+tc.pool+`","region":"`+tc.region+`","kinds":["emu"]}`)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("empty poll %v: %d %s", tc, rec.Code, rec.Body.String())
		}
		if got := srv.store.FleetRegions("org_a"); len(got) == 0 {
			t.Fatal("direct heartbeat check: no regions after poll")
		}
	}

	// Queue org_a work: 1 tokyo-pinned + 1 unpinned emu; org_b queues 5 pinned
	// builds that must NOT show up in org_a's depth.
	rec = authed(t, srv, http.MethodPost, "/v1/emus", "key-a", `{"profile":"P","region":"asia-northeast1"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("pinned emu: %d", rec.Code)
	}
	rec = authed(t, srv, http.MethodPost, "/v1/emus", "key-a", `{"profile":"P"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("unpinned emu: %d", rec.Code)
	}
	for i := 0; i < 5; i++ {
		srv.store.CreateBuild("org_b", "Other", "", "asia-southeast1")
	}

	rec = authed(t, srv, http.MethodGet, "/v1/fleet/regions", "key-a", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("fleet: %d %s", rec.Code, rec.Body.String())
	}
	var fleet struct {
		Regions []RegionStatus `json:"regions"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &fleet); err != nil {
		t.Fatal(err)
	}
	if len(fleet.Regions) != 2 {
		t.Fatalf("want 2 live regions, got %+v", fleet.Regions)
	}
	byRegion := map[string]RegionStatus{}
	for _, r := range fleet.Regions {
		byRegion[r.Region] = r
		if r.LastSeen.IsZero() {
			t.Errorf("region %s has zero last_seen", r.Region)
		}
	}
	sg, ok := byRegion["asia-southeast1"]
	if !ok || sg.Hosts != 2 || len(sg.Pools) != 2 {
		t.Errorf("sg row: %+v (want 2 hosts, 2 pools)", sg)
	}
	tokyo, ok := byRegion["asia-northeast1"]
	if !ok || tokyo.Hosts != 1 {
		t.Errorf("tokyo row: %+v (want 1 host)", tokyo)
	}
	// org_a depth: tokyo row sees pinned + unpinned (2); sg row sees only the
	// unpinned emu (1). org_b's 5 builds must not leak in.
	if tokyo.Queued != 2 {
		t.Errorf("tokyo queued = %d, want 2", tokyo.Queued)
	}
	if sg.Queued != 1 {
		t.Errorf("sg queued = %d, want 1", sg.Queued)
	}

	// Org isolation at the HTTP layer: org_b sees its own 5 builds in sg.
	rec = authed(t, srv, http.MethodGet, "/v1/fleet/regions", "key-b", "")
	var bFleet struct {
		Regions []RegionStatus `json:"regions"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &bFleet); err != nil {
		t.Fatal(err)
	}
	for _, r := range bFleet.Regions {
		if r.Region == "asia-southeast1" && r.Queued != 5 {
			t.Errorf("org_b sg queued = %d, want 5", r.Queued)
		}
		if r.Region == "asia-northeast1" && r.Queued != 0 {
			t.Errorf("org_b tokyo queued = %d, want 0", r.Queued)
		}
	}

	// Unauthenticated and wrong-key callers get 401, never a fleet map.
	// (Host tokens authenticate /v1/host/* only; the fleet map is org-scoped.)
	for _, key := range []string{"", "nope", testHostToken} {
		r := hosted(t, srv, http.MethodGet, "/v1/fleet/regions", key, "")
		if r.Code != http.StatusUnauthorized {
			t.Errorf("key %q: got %d, want 401", key, r.Code)
		}
	}
}

// TestFleetRegionsConcurrency races heartbeat writers against fleet readers:
// under -race any unsynchronized beats-map access fails the run.
func TestFleetRegionsConcurrency(t *testing.T) {
	srv := testHostServer()
	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		wg.Add(2)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				srv.store.RecordHeartbeat(fmt.Sprintf("pool-%d", w), "asia-southeast1")
			}
		}(w)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				_ = srv.store.FleetRegions("org_a")
			}
		}()
	}
	wg.Wait()
	regs := srv.store.FleetRegions("org_a")
	if len(regs) != 1 || regs[0].Hosts != 8 {
		t.Errorf("fleet after race: %+v (want 1 region, 8 hosts)", regs)
	}
}

// percentile returns the p-th percentile of sorted nanos.
func percentile(sorted []int64, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	i := int(p / 100 * float64(len(sorted)-1))
	return time.Duration(sorted[i])
}

// TestSchedulerLoadNumbers measures what operators need for capacity
// planning: cold-start latency and ClaimNext scheduling latency (p50/p95)
// on this machine. It fails only on correctness (nil claim); the numbers
// go to the log for humans and CI history.
func TestSchedulerLoadNumbers(t *testing.T) {
	start := time.Now()
	s := NewStore()
	startup := time.Since(start)

	const n = 5000
	for i := 0; i < n; i++ {
		s.CreateBuild("org_a", "MyApp", "", "")
	}
	lats := make([]int64, 0, n)
	for i := 0; i < n; i++ {
		t0 := time.Now()
		if c := s.ClaimNext("mac-1", "", []string{"build"}); c == nil {
			t.Fatal("queue drained early")
		}
		lats = append(lats, int64(time.Since(t0)))
	}
	sort.Slice(lats, func(i, j int) bool { return lats[i] < lats[j] })
	t.Logf("startup (NewStore): %s", startup)
	t.Logf("ClaimNext over %d builds: p50=%s p95=%s max=%s", n,
		percentile(lats, 50), percentile(lats, 95), time.Duration(lats[len(lats)-1]))
}

// BenchmarkClaimNext keeps the same measurement in bench form for
// `go test -bench`.
func BenchmarkClaimNext(b *testing.B) {
	s := NewStore()
	for i := 0; i < b.N; i++ {
		s.CreateBuild("org_a", "MyApp", "", "")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if c := s.ClaimNext("mac-1", "", []string{"build"}); c == nil {
			b.Fatal("queue drained early")
		}
	}
}
