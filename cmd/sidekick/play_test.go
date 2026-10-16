// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 ImBIOS — licensed under Apache-2.0, see LICENSE-APACHE.md.
package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeAAB(t *testing.T, manifest bool) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "app.aab")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	if manifest {
		m, _ := w.Create("AndroidManifest.xml")
		m.Write([]byte(`<manifest package="com.x"/>`))
	}
	filler, _ := w.CreateHeader(&zip.FileHeader{Name: "filler.bin", Method: zip.Store})
	filler.Write(make([]byte, 4096))
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return p
}

func makeKey(t *testing.T, email string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "key.json")
	body := `{"type":"service_account","client_email":"` + email + `","private_key":"x","private_key_id":"1"}`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPlayPlanDryRun(t *testing.T) {
	aab := makeAAB(t, true)
	key := makeKey(t, "ci@proj.iam.gserviceaccount.com")
	plan, err := playPlan(aab, "com.x", "internal", key)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(plan.String(), "com.x") || !strings.Contains(plan.String(), "internal") {
		t.Errorf("plan: %s", plan)
	}
}

func TestPlayPlanRejects(t *testing.T) {
	aab := makeAAB(t, true)
	key := makeKey(t, "ci@proj.iam.gserviceaccount.com")
	cases := []struct {
		name string
		aab  string
		pkg  string
		key  string
	}{
		{"missing aab", "", "com.x", key},
		{"missing package", aab, "", key},
		{"missing key", aab, "com.x", ""},
		{"bad key path", aab, "com.x", filepath.Join(t.TempDir(), "nope.json")},
		{"no manifest", makeAAB(t, false), "com.x", key},
	}
	for _, tc := range cases {
		if _, err := playPlan(tc.aab, tc.pkg, "internal", tc.key); err == nil {
			t.Errorf("%s: expected error", tc.name)
		}
	}
	badKey := filepath.Join(t.TempDir(), "bad.json")
	os.WriteFile(badKey, []byte(`{"type":"oauth2"}`), 0o600)
	if _, err := playPlan(aab, "com.x", "internal", badKey); err == nil {
		t.Error("non-service-account key should fail")
	}
}
