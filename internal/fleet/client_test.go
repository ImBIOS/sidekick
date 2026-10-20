// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
package fleet

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Regression test for a live bug (18 Oct 2026): the control-plane answers
// idle polls with 204 + empty body, which do() reports as success. ClaimNext
// used to check err first, so every idle poll decoded as an *empty claim*
// and daemons spun on phantom work ("claimed <empty>", refuse, repeat).
func TestClaimNext204IsIdle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	c := New(srv.URL, "tok")
	claim, ok, err := c.ClaimNext("linux-pool", "", []string{"emu"})
	if err != nil || ok || claim != nil {
		t.Fatalf("idle poll: claim=%v ok=%v err=%v", claim, ok, err)
	}
}

func TestClaimNext200Decodes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"kind":"emu","emu":{"id":"emu_1","profile":"T3"}}`))
	}))
	defer srv.Close()
	c := New(srv.URL, "tok")
	claim, ok, err := c.ClaimNext("linux-pool", "", []string{"emu"})
	if err != nil || !ok || claim.EmuID() != "emu_1" {
		t.Fatalf("claim poll: %+v ok=%v err=%v", claim, ok, err)
	}
}
