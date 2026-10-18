// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
package controller

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

// TestFakeClusterWatchReconcileStatus exercises the exact wiring main.go
// uses (claimFromUnstructured -> Reconcile -> writeStatus) against the
// client-go fake dynamic client: no cluster, no mocks of our own code.
func TestFakeClusterWatchReconcileStatus(t *testing.T) {
	scheme := runtime.NewScheme()
	gvr := schema.GroupVersionResource{Group: "sidekick.imbios.dev", Version: "v1", Resource: "emulatorclaims"}
	dc := dynamicfake.NewSimpleDynamicClient(scheme)

	seed := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "sidekick.imbios.dev/v1",
		"kind":       "EmulatorClaim",
		"metadata":   map[string]any{"name": "emu-abc", "namespace": "default"},
		"spec":       map[string]any{"profile": "Pixel_7_API_34", "tenant": "tn_123", "ttl": "30m"},
	}}
	if _, err := dc.Resource(gvr).Namespace("default").Create(context.Background(), seed, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	r := &Reconciler{CP: newFakeCP(), Clock: frozenClock{now}}
	ctx := context.Background()

	// Simulate one informer event: read, convert, reconcile, write status.
	u, err := dc.Resource(gvr).Namespace("default").Get(ctx, "emu-abc", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	// Fake client stamps no creationTimestamp; set one so TTL math is real.
	u.SetCreationTimestamp(metav1.NewTime(now))
	claim := ClaimFromUnstructured(KindEmulator, u)
	if claim.Spec.Profile != "Pixel_7_API_34" || claim.Spec.Tenant != "tn_123" {
		t.Fatalf("conversion dropped spec: %+v", claim.Spec)
	}
	st, err := r.Reconcile(ctx, claim)
	if err != nil || st.Phase != PhaseProvisioning || st.ResourceID == "" {
		t.Fatalf("reconcile: %+v %v", st, err)
	}
	if err := WriteStatus(ctx, dc, gvr, u, st); err != nil {
		t.Fatal(err)
	}

	got, err := dc.Resource(gvr).Namespace("default").Get(ctx, "emu-abc", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	phase, _, _ := unstructured.NestedString(got.Object, "status", "phase")
	rid, _, _ := unstructured.NestedString(got.Object, "status", "resourceID")
	if phase != PhaseProvisioning || rid == "" {
		t.Fatalf("status not persisted: phase=%q resourceID=%q", phase, rid)
	}
}
