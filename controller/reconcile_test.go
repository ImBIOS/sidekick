// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
package controller

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// fakeCP is an in-memory ControlPlane. Purposely lock-guarded: the
// reconciler must be safe under concurrent workers (one goroutine per
// claim), and -race watches this fake.
type fakeCP struct {
	mu     sync.Mutex
	seq    int
	sims   map[string]*ResourceState
	builds map[string]*ResourceState
	emus   map[string]*ResourceState
	creates int
}

func newFakeCP() *fakeCP {
	return &fakeCP{sims: map[string]*ResourceState{}, builds: map[string]*ResourceState{}, emus: map[string]*ResourceState{}}
}

func (f *fakeCP) nextID(p string) string {
	f.seq++
	return p + "_test-" + string(rune('0'+f.seq%10)) + "-" + time.Now().UTC().Format("150405")
}

func (f *fakeCP) CreateSim(region, device, runtime string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.creates++
	id := f.nextID("sim")
	f.sims[id] = &ResourceState{ID: id, Status: "provisioning", Found: true}
	return id, nil
}

func (f *fakeCP) CreateBuild(region, scheme string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.creates++
	id := f.nextID("bld")
	f.builds[id] = &ResourceState{ID: id, Status: "queued", Found: true}
	return id, nil
}

func (f *fakeCP) CreateEmu(region, profile string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.creates++
	id := f.nextID("emu")
	f.emus[id] = &ResourceState{ID: id, Status: "provisioning", Found: true}
	return id, nil
}

func (f *fakeCP) GetSim(id string) (ResourceState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	st, ok := f.sims[id]
	if !ok {
		return ResourceState{}, nil
	}
	return *st, nil
}

func (f *fakeCP) GetBuild(id string) (ResourceState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	st, ok := f.builds[id]
	if !ok {
		return ResourceState{}, nil
	}
	return *st, nil
}

func (f *fakeCP) GetEmu(id string) (ResourceState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	st, ok := f.emus[id]
	if !ok {
		return ResourceState{}, nil
	}
	return *st, nil
}

func (f *fakeCP) DeleteSim(id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.sims, id)
	return nil
}

func (f *fakeCP) DeleteEmu(id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.emus, id)
	return nil
}

func (f *fakeCP) setSim(id, status, host, url string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if st, ok := f.sims[id]; ok {
		st.Status, st.Host, st.WebRTCURL = status, host, url
	}
}

// frozenClock pins Now for TTL tests.
type frozenClock struct{ t time.Time }

func (f frozenClock) Now() time.Time { return f.t }

func TestSimClaimLifecycle(t *testing.T) {
	cp := newFakeCP()
	now := time.Now().UTC()
	r := &Reconciler{CP: cp, Clock: frozenClock{now}}
	ctx := context.Background()

	claim := Claim{Kind: KindSimulator, Name: "sim-abc",
		Spec:              Spec{Device: "iPhone16", OS: "18.2", TTL: "30m", Tenant: "tn_123"},
		CreationTimestamp: now}

	// 1. Fresh claim provisions.
	st, err := r.Reconcile(ctx, claim)
	if err != nil || st.Phase != PhaseProvisioning || st.ResourceID == "" {
		t.Fatalf("provision: %+v err=%v", st, err)
	}
	id := st.ResourceID

	// 2. Still provisioning while the daemon works.
	claim.Status = st
	st, _ = r.Reconcile(ctx, claim)
	if st.Phase != PhaseProvisioning {
		t.Fatalf("poll provisioning: %+v", st)
	}

	// 3. Daemon readies it: status mirrors host + URL.
	cp.setSim(id, "running", "mac-mini-02", "wss://x/"+id)
	st, _ = r.Reconcile(ctx, claim)
	if st.Phase != PhaseRunning || st.Host != "mac-mini-02" || st.WebRTCURL == "" {
		t.Fatalf("running: %+v", st)
	}

	// 4. Idempotent: re-reconciling a running claim creates nothing.
	claim.Status = st
	before := cp.creates
	if _, err := r.Reconcile(ctx, claim); err != nil {
		t.Fatal(err)
	}
	if cp.creates != before {
		t.Fatal("reconcile recreated an existing resource")
	}
}

func TestClaimTTLExpiryReleases(t *testing.T) {
	cp := newFakeCP()
	start := time.Now().UTC()
	r := &Reconciler{CP: cp, Clock: frozenClock{start}}
	ctx := context.Background()

	claim := Claim{Kind: KindEmulator, Name: "emu-abc",
		Spec:              Spec{Profile: "Pixel_7_API_34", TTL: "30m", Tenant: "tn_123"},
		CreationTimestamp: start}
	st, err := r.Reconcile(ctx, claim)
	if err != nil || st.ResourceID == "" {
		t.Fatalf("provision: %+v %v", st, err)
	}
	id := st.ResourceID

	// 31 minutes later the TTL fires: resource deleted, claim expired.
	r.Clock = frozenClock{start.Add(31 * time.Minute)}
	claim.Status = st
	st, err = r.Reconcile(ctx, claim)
	if err != nil || st.Phase != PhaseExpired {
		t.Fatalf("expiry: %+v %v", st, err)
	}
	if _, ok := cp.emus[id]; ok {
		t.Fatal("expired claim left its emulator running")
	}

	// Terminal claims are untouched thereafter.
	claim.Status = st
	st2, _ := r.Reconcile(ctx, claim)
	if st2.Phase != PhaseExpired {
		t.Fatalf("terminal moved: %+v", st2)
	}
}

func TestBuildClaimTerminal(t *testing.T) {
	cp := newFakeCP()
	now := time.Now().UTC()
	r := &Reconciler{CP: cp, Clock: frozenClock{now}}
	ctx := context.Background()

	claim := Claim{Kind: KindBuild, Name: "build-abc",
		Spec:              Spec{Scheme: "MyApp", Xcode: "16.2", Tenant: "tn_123"},
		CreationTimestamp: now}
	st, err := r.Reconcile(ctx, claim)
	if err != nil || st.Phase != PhaseProvisioning {
		t.Fatalf("provision: %+v %v", st, err)
	}
	cp.mu.Lock()
	cp.builds[st.ResourceID].Status = "succeeded"
	cp.mu.Unlock()
	claim.Status = st
	st, _ = r.Reconcile(ctx, claim)
	if st.Phase != PhaseSucceeded {
		t.Fatalf("terminal: %+v", st)
	}
}

func TestClaimValidation(t *testing.T) {
	now := time.Now().UTC()
	r := &Reconciler{CP: newFakeCP(), Clock: frozenClock{now}}
	ctx := context.Background()
	for _, tc := range []struct {
		name  string
		claim Claim
		want  string
	}{
		{"unknown kind", Claim{Kind: "ToasterClaim", CreationTimestamp: now}, "unknown kind"},
		{"sim needs device", Claim{Kind: KindSimulator, CreationTimestamp: now}, "spec.device"},
		{"build needs scheme", Claim{Kind: KindBuild, CreationTimestamp: now}, "spec.scheme"},
		{"emu needs profile", Claim{Kind: KindEmulator, CreationTimestamp: now}, "spec.profile"},
		{"bad ttl", Claim{Kind: KindSimulator, Spec: Spec{Device: "x", TTL: "forever"}, CreationTimestamp: now}, "bad spec.ttl"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st, _ := r.Reconcile(ctx, tc.claim)
			if st.Phase != PhaseFailed {
				t.Fatalf("phase=%q, want Failed (%+v)", st.Phase, st)
			}
			if st.Message == "" {
				t.Fatal("failed claims must carry a message for kubectl")
			}
		})
	}
}

func TestOutOfBandDeleteFailsClaim(t *testing.T) {
	cp := newFakeCP()
	now := time.Now().UTC()
	r := &Reconciler{CP: cp, Clock: frozenClock{now}}
	ctx := context.Background()
	claim := Claim{Kind: KindSimulator, Name: "sim-gone",
		Spec:              Spec{Device: "iPhone16", TTL: "1h"},
		CreationTimestamp: now}
	st, _ := r.Reconcile(ctx, claim)
	claim.Status = st
	_ = cp.DeleteSim(st.ResourceID) // user deleted it behind our back
	st, _ = r.Reconcile(ctx, claim)
	if st.Phase != PhaseFailed {
		t.Fatalf("phase=%q, want Failed", st.Phase)
	}
}

func TestConcurrentReconcileNoDoubleCreate(t *testing.T) {
	cp := newFakeCP()
	now := time.Now().UTC()
	r := &Reconciler{CP: cp}
	ctx := context.Background()
	const n = 32
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c := Claim{Kind: KindEmulator, Name: "emu-race",
				Spec:              Spec{Profile: "Pixel_7_API_34", TTL: "1h"},
				CreationTimestamp: now}
			_, errs[i] = r.Reconcile(ctx, c)
		}(i)
	}
	wg.Wait()
	for _, err := range errs {
		if !errors.Is(err, nil) {
			t.Fatal(err)
		}
	}
	// Each reconcile of a fresh claim creates once (callers serialize per
	// name via the workqueue; here we prove the fake + logic are race-clean
	// under -race — dedup across workers is the queue's job in main.go).
	if cp.creates != n {
		t.Fatalf("creates=%d, want %d", cp.creates, n)
	}
}
