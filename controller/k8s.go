// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
package controller

import (
	"context"
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// ClaimFromUnstructured converts a watched object into a Claim. Exported
// so the operator binary and tests share the exact conversion.
func ClaimFromUnstructured(kind string, u *unstructured.Unstructured) Claim {
	c := Claim{Kind: kind, Name: u.GetName(), CreationTimestamp: u.GetCreationTimestamp().Time}
	spec, _, _ := unstructured.NestedMap(u.Object, "spec")
	raw, _ := json.Marshal(spec)
	_ = json.Unmarshal(raw, &c.Spec)
	status, _, _ := unstructured.NestedMap(u.Object, "status")
	raw, _ = json.Marshal(status)
	_ = json.Unmarshal(raw, &c.Status)
	return c
}

// WriteStatus persists the reconciled status onto the claim.
func WriteStatus(ctx context.Context, dc dynamic.Interface, gvr schema.GroupVersionResource, u *unstructured.Unstructured, st Status) error {
	raw, _ := json.Marshal(st)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	if err := unstructured.SetNestedMap(u.Object, m, "status"); err != nil {
		return err
	}
	_, err := dc.Resource(gvr).Namespace(u.GetNamespace()).UpdateStatus(ctx, u, metav1.UpdateOptions{})
	return err
}
