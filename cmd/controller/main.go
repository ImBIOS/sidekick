// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.

// Command controller runs the Sidekick claim operator: it watches
// SimulatorClaim / BuildClaim / EmulatorClaim resources (crds/claims.yaml)
// and reconciles each one against the control-plane REST API.
//
// Usage:
//
//	controller --kubeconfig ~/.kube/config \
//	  --control https://api.example.com --token $SIDEKICK_HOST_TOKEN
//
// RBAC needed: get/list/watch/update on sidekick.imbios.dev claims plus
// update on their /status. See deploy/controller.yaml.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ImBIOS/sidekick/controller"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

// claimGVRs are the three watched resources.
var claimGVRs = []schema.GroupVersionResource{
	{Group: "sidekick.imbios.dev", Version: "v1", Resource: "simulatorclaims"},
	{Group: "sidekick.imbios.dev", Version: "v1", Resource: "buildclaims"},
	{Group: "sidekick.imbios.dev", Version: "v1", Resource: "emulatorclaims"},
}

var kindForResource = map[string]string{
	"simulatorclaims": controller.KindSimulator,
	"buildclaims":     controller.KindBuild,
	"emulatorclaims":  controller.KindEmulator,
}

// cpClient implements controller.ControlPlane over the control-plane REST
// API. Claims carry a tenant but the controller is fleet infrastructure:
// it authenticates with the host token like the daemons, and passes the
// claim's tenant through as the org header the control-plane trusts from
// fleet callers.
type cpClient struct {
	base  string
	token string
	http  *http.Client
}

func (c *cpClient) post(path string, body any, out any) error {
	raw, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, c.base+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("control-plane %s: %d", path, resp.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (c *cpClient) get(path string, out any) (bool, error) {
	req, err := http.NewRequest(http.MethodGet, c.base+path, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, fmt.Errorf("control-plane %s: %d", path, resp.StatusCode)
	}
	return true, json.NewDecoder(resp.Body).Decode(out)
}

func idOf(v map[string]any) string {
	for _, k := range []string{"id", "build_id"} {
		if s, _ := v[k].(string); s != "" {
			return s
		}
	}
	return ""
}

func (c *cpClient) CreateSim(region, device, runtime string) (string, error) {
	var out map[string]any
	if err := c.post("/v1/sims", map[string]string{"device": device, "runtime": runtime, "region": region}, &out); err != nil {
		return "", err
	}
	return idOf(out), nil
}

func (c *cpClient) CreateBuild(region, scheme string) (string, error) {
	var out map[string]any
	if err := c.post("/v1/builds", map[string]string{"scheme": scheme, "region": region}, &out); err != nil {
		return "", err
	}
	return idOf(out), nil
}

func (c *cpClient) CreateEmu(region, profile string) (string, error) {
	var out map[string]any
	if err := c.post("/v1/emus", map[string]string{"profile": profile, "region": region}, &out); err != nil {
		return "", err
	}
	return idOf(out), nil
}

func toState(v map[string]any, found bool) controller.ResourceState {
	if !found {
		return controller.ResourceState{}
	}
	st := controller.ResourceState{Found: true}
	st.ID = idOf(v)
	st.Status, _ = v["status"].(string)
	st.Host, _ = v["host"].(string)
	st.WebRTCURL, _ = v["webrtc_url"].(string)
	st.ADBTarget, _ = v["adb_target"].(string)
	return st
}

func (c *cpClient) GetSim(id string) (controller.ResourceState, error) {
	var out map[string]any
	found, err := c.get("/v1/sims/"+id, &out)
	return toState(out, found), err
}

func (c *cpClient) GetBuild(id string) (controller.ResourceState, error) {
	var out map[string]any
	found, err := c.get("/v1/builds/"+id, &out)
	return toState(out, found), err
}

func (c *cpClient) GetEmu(id string) (controller.ResourceState, error) {
	var out map[string]any
	found, err := c.get("/v1/emus/"+id, &out)
	return toState(out, found), err
}

func (c *cpClient) DeleteSim(id string) error {
	req, _ := http.NewRequest(http.MethodDelete, c.base+"/v1/sims/"+id, nil)
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *cpClient) DeleteEmu(id string) error {
	req, _ := http.NewRequest(http.MethodDelete, c.base+"/v1/emus/"+id, nil)
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func main() {
	kubeconfig := flag.String("kubeconfig", "", "path to kubeconfig (or KUBECONFIG)")
	control := flag.String("control", "", "control-plane URL (or SIDEKICK_CONTROL)")
	token := flag.String("token", "", "fleet host token (or SIDEKICK_HOST_TOKEN)")
	resync := flag.Duration("resync", 30*time.Second, "informer resync / TTL recheck interval")
	flag.Parse()

	if *control == "" {
		*control = os.Getenv("SIDEKICK_CONTROL")
	}
	if *token == "" {
		*token = os.Getenv("SIDEKICK_HOST_TOKEN")
	}
	if *control == "" || *token == "" {
		fmt.Fprintln(os.Stderr, "controller: need --control and --token (or SIDEKICK_CONTROL / SIDEKICK_HOST_TOKEN)")
		os.Exit(1)
	}
	kcfg := *kubeconfig
	if kcfg == "" {
		kcfg = os.Getenv("KUBECONFIG")
	}
	cfg, err := clientcmd.BuildConfigFromFlags("", kcfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "controller: kubeconfig:", err)
		os.Exit(1)
	}
	dc, err := dynamic.NewForConfig(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "controller: dynamic client:", err)
		os.Exit(1)
	}

	r := &controller.Reconciler{CP: &cpClient{base: *control, token: *token, http: &http.Client{Timeout: 30 * time.Second}}}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(dc, *resync, metav1.NamespaceAll, nil)
	for _, gvr := range claimGVRs {
		gvr := gvr
		kind := kindForResource[gvr.Resource]
		inf := factory.ForResource(gvr)
		_, _ = inf.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj any) { handleClaim(ctx, dc, gvr, kind, r, obj) },
			UpdateFunc: func(_, newObj any) {
				handleClaim(ctx, dc, gvr, kind, r, newObj)
			},
		})
	}
	fmt.Printf("controller: watching %d claim kinds via %s (resync %s)\n", len(claimGVRs), *control, *resync)
	factory.Start(ctx.Done())
	<-ctx.Done()
	fmt.Println("controller: shutting down")
}

func handleClaim(ctx context.Context, dc dynamic.Interface, gvr schema.GroupVersionResource, kind string, r *controller.Reconciler, obj any) {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return
	}
	before := u.DeepCopy()
	claim := controller.ClaimFromUnstructured(kind, u)
	st, err := r.Reconcile(ctx, claim)
	if err != nil {
		fmt.Printf("controller: %s/%s reconcile error: %v\n", kind, u.GetName(), err)
	}
	// Skip the status write when nothing changed (avoids hot-looping).
	rawBefore, _ := json.Marshal(before.Object["status"])
	rawAfter, _ := json.Marshal(st)
	if string(rawBefore) == string(rawAfter) && err == nil {
		return
	}
	u = before
	if err := controller.WriteStatus(ctx, dc, gvr, u, st); err != nil {
		fmt.Printf("controller: %s/%s status write: %v\n", kind, u.GetName(), err)
		return
	}
	fmt.Printf("controller: %s/%s -> %s (%s)\n", kind, u.GetName(), st.Phase, st.Message)
}
