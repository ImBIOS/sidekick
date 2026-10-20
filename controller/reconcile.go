// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.

// Package controller is the Kubernetes operator for crds/claims.yaml.
//
// SimulatorClaim / BuildClaim / EmulatorClaim let platform teams provision
// Sidekick capacity declaratively: `kubectl apply -f team-a-sims.yaml` and
// the controller creates the matching control-plane resource, mirrors its
// state into the claim's status, and enforces spec.ttl by deleting the
// resource when it expires.
//
// The reconcile core (Reconcile) is pure logic over two small interfaces
// (ControlPlane, Clock) so every transition is unit-tested without a
// cluster. main.go wires it to a dynamic client + the control-plane REST
// API; the fake in reconcile_test.go proves the transitions.
package controller

import (
	"context"
	"fmt"
	"time"
)

// Phases a claim moves through. Terminal: Succeeded, Failed, Expired.
const (
	PhasePending      = "Pending"
	PhaseProvisioning = "Provisioning"
	PhaseRunning      = "Running"
	PhaseSucceeded    = "Succeeded"
	PhaseFailed       = "Failed"
	PhaseExpired      = "Expired"
)

// Kinds mirrored from crds/claims.yaml.
const (
	KindSimulator = "SimulatorClaim"
	KindBuild     = "BuildClaim"
	KindEmulator  = "EmulatorClaim"
)

// DefaultTTL applies when spec.ttl is empty. Claims are cost (a booted
// simulator bills streaming minutes), so unbounded claims are rejected.
const DefaultTTL = 2 * time.Hour

// Spec is the desired state. Fields mirror crds/claims.yaml.
type Spec struct {
	// SimulatorClaim: device + os. BuildClaim: scheme + xcode.
	// EmulatorClaim: profile.
	Device  string `json:"device,omitempty"`
	OS      string `json:"os,omitempty"`
	Scheme  string `json:"scheme,omitempty"`
	Xcode   string `json:"xcode,omitempty"`
	Profile string `json:"profile,omitempty"`
	Region  string `json:"region,omitempty"`
	Tenant  string `json:"tenant,omitempty"`
	// TTL like "30m". Empty = DefaultTTL.
	TTL string `json:"ttl,omitempty"`
}

// Status is the observed state, written back to .status by the controller.
type Status struct {
	Phase      string `json:"phase,omitempty"`
	ResourceID string `json:"resourceID,omitempty"`
	Host       string `json:"host,omitempty"`
	// Sim: UDID + WebRTCURL. Emu: ADBTarget.
	UDID       string `json:"udid,omitempty"`
	WebRTCURL  string `json:"webrtcURL,omitempty"`
	ADBTarget  string `json:"adbTarget,omitempty"`
	Message    string `json:"message,omitempty"`
	ExpiresAt  string `json:"expiresAt,omitempty"`
}

// Claim is one custom resource instance.
type Claim struct {
	Kind              string
	Name              string
	Spec              Spec
	Status            Status
	CreationTimestamp time.Time
}

// ttlOf parses spec.ttl, falling back to DefaultTTL on empty.
func (c *Claim) ttlOf() (time.Duration, error) {
	if c.Spec.TTL == "" {
		return DefaultTTL, nil
	}
	return time.ParseDuration(c.Spec.TTL)
}

// ResourceState is what the control-plane reports for a resource.
type ResourceState struct {
	ID        string
	Status    string // queued|provisioning|running|succeeded|failed|stopped
	Host      string
	WebRTCURL string
	ADBTarget string
	Found     bool
}

// ControlPlane is the narrow surface the controller needs. The production
// implementation (cpClient in main.go) speaks the control-plane REST API;
// tests use fakeCP.
type ControlPlane interface {
	CreateSim(region, device, runtime string) (id string, err error)
	CreateBuild(region, scheme string) (id string, err error)
	CreateEmu(region, profile string) (id string, err error)
	GetSim(id string) (ResourceState, error)
	GetBuild(id string) (ResourceState, error)
	GetEmu(id string) (ResourceState, error)
	DeleteSim(id string) error
	DeleteEmu(id string) error
}

// Clock supplies time; tests freeze it to prove TTL transitions.
type Clock interface{ Now() time.Time }

// realClock is production time.
type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

// Reconciler holds dependencies; Reconcile is reentrant and idempotent.
type Reconciler struct {
	CP    ControlPlane
	Clock Clock
}

// clock returns the configured clock or wall time.
func (r *Reconciler) clock() Clock {
	if r.Clock != nil {
		return r.Clock
	}
	return realClock{}
}

// terminal reports whether phase needs no further work.
func terminal(phase string) bool {
	return phase == PhaseSucceeded || phase == PhaseFailed || phase == PhaseExpired
}

// Reconcile drives one claim toward its desired state and returns the
// status the controller must persist (empty ResourceID means "no status
// write needed beyond what is returned"). It never throws: unknown kinds
// and bad TTLs surface as PhaseFailed with a message, so `kubectl get`
// shows the problem instead of hiding it in logs.
func (r *Reconciler) Reconcile(ctx context.Context, c Claim) (Status, error) {
	now := r.clock().Now().UTC()
	next := c.Status

	ttl, err := c.ttlOf()
	if err != nil {
		next.Phase = PhaseFailed
		next.Message = fmt.Sprintf("bad spec.ttl %q: %v", c.Spec.TTL, err)
		return next, nil
	}
	expires := c.CreationTimestamp.Add(ttl)
	next.ExpiresAt = expires.UTC().Format(time.RFC3339)

	if terminal(c.Status.Phase) {
		return next, nil
	}
	if !now.Before(expires) {
		// TTL fired: best-effort delete, then mark expired. Best-effort
		// because the resource may already be gone (manual delete).
		r.deleteResource(ctx, c.Kind, c.Status.ResourceID)
		next.Phase = PhaseExpired
		next.Message = fmt.Sprintf("ttl %s elapsed; resource released", ttl)
		return next, nil
	}

	// No resource yet: create it (idempotent — we only get here when
	// status.resourceID is empty, and a retry after a crash re-lists
	// before creating; see main.go).
	if c.Status.ResourceID == "" {
		return r.provision(ctx, c, next)
	}
	return r.refresh(ctx, c, next)
}

// provision creates the control-plane resource for a fresh claim.
func (r *Reconciler) provision(_ context.Context, c Claim, next Status) (Status, error) {
	var id string
	var err error
	switch c.Kind {
	case KindSimulator:
		if c.Spec.Device == "" {
			return failed(next, "spec.device is required"), nil
		}
		id, err = r.CP.CreateSim(c.Spec.Region, c.Spec.Device, c.Spec.OS)
	case KindBuild:
		if c.Spec.Scheme == "" {
			return failed(next, "spec.scheme is required"), nil
		}
		id, err = r.CP.CreateBuild(c.Spec.Region, c.Spec.Scheme)
	case KindEmulator:
		if c.Spec.Profile == "" {
			return failed(next, "spec.profile is required"), nil
		}
		id, err = r.CP.CreateEmu(c.Spec.Region, c.Spec.Profile)
	default:
		return failed(next, fmt.Sprintf("unknown kind %q", c.Kind)), nil
	}
	if err != nil {
		next.Phase = PhasePending
		next.Message = fmt.Sprintf("create retry: %v", err)
		return next, err
	}
	next.ResourceID = id
	next.Phase = PhaseProvisioning
	next.Message = fmt.Sprintf("created %s", id)
	return next, nil
}

// refresh mirrors control-plane state into the claim status.
func (r *Reconciler) refresh(_ context.Context, c Claim, next Status) (Status, error) {
	var st ResourceState
	var err error
	switch c.Kind {
	case KindSimulator:
		st, err = r.CP.GetSim(c.Status.ResourceID)
	case KindBuild:
		st, err = r.CP.GetBuild(c.Status.ResourceID)
	case KindEmulator:
		st, err = r.CP.GetEmu(c.Status.ResourceID)
	default:
		return failed(next, fmt.Sprintf("unknown kind %q", c.Kind)), nil
	}
	if err != nil {
		next.Message = fmt.Sprintf("poll retry: %v", err)
		return next, err
	}
	if !st.Found {
		// Resource vanished out-of-band (manual delete): release the
		// claim rather than recreating spend behind the user's back.
		next.Phase = PhaseFailed
		next.Message = "resource deleted out-of-band; delete and re-apply the claim to recreate"
		return next, nil
	}
	next.Host = st.Host
	next.WebRTCURL = st.WebRTCURL
	next.ADBTarget = st.ADBTarget
	switch st.Status {
	case "running":
		next.Phase = PhaseRunning
		next.Message = ""
	case "succeeded":
		next.Phase = PhaseSucceeded
		next.Message = ""
	case "failed":
		next.Phase = PhaseFailed
		next.Message = "resource failed; see control-plane logs"
	default:
		next.Phase = PhaseProvisioning
	}
	return next, nil
}

// deleteResource best-effort releases sims/emus. Builds are terminal work,
// not capacity, so expiry just stops watching them.
func (r *Reconciler) deleteResource(_ context.Context, kind, id string) {
	if id == "" {
		return
	}
	switch kind {
	case KindSimulator:
		_ = r.CP.DeleteSim(id)
	case KindEmulator:
		_ = r.CP.DeleteEmu(id)
	}
}

func failed(next Status, msg string) Status {
	next.Phase = PhaseFailed
	next.Message = msg
	return next
}
