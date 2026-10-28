// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
package main

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

// These tests prove PGStore's SQL shapes, argument order, and row scanning
// against a mocked pgx pool — no live Postgres needed. The same statements
// run unmodified against real Postgres (docker-compose.selfhost.yml).

// mockAdapter reshapes pgxmock's pool (CommandTag Exec) to Querier.
type mockAdapter struct {
	mock pgxmock.PgxPoolIface
}

func (a mockAdapter) Exec(ctx context.Context, sql string, args ...any) (int64, error) {
	tag, err := a.mock.Exec(ctx, sql, args...)
	return tag.RowsAffected(), err
}

func (a mockAdapter) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return a.mock.Query(ctx, sql, args...)
}

func (a mockAdapter) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return a.mock.QueryRow(ctx, sql, args...)
}

func newMockStore(t *testing.T) (*PGStore, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { mock.Close() })
	return &PGStore{q: mockAdapter{mock}}, mock
}

func qm(sql string) string { return regexp.QuoteMeta(sql) }

var testTime = time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)

func TestPGCreateGetBuild(t *testing.T) {
	s, mock := newMockStore(t)
	mock.ExpectExec(qm(`INSERT INTO builds (id, org, scheme, ref, status, region, created_at, log) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`)).
		WithArgs(pgxmock.AnyArg(), "org_a", "MyApp", "", StatusQueued, "", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	b := s.CreateBuild("org_a", "MyApp", "", "")
	if b.Status != StatusQueued {
		t.Errorf("status = %q", b.Status)
	}
	rows := pgxmock.NewRows([]string{"id", "org", "scheme", "ref", "status", "host", "region", "created_at", "log"}).
		AddRow(b.ID, "org_a", "MyApp", "", StatusQueued, "", "", testTime, []string{"l1"})
	mock.ExpectQuery(qm(`SELECT id, org, scheme, ref, status, host, region, created_at, log FROM builds WHERE id=$1 AND org=$2`)).
		WithArgs(b.ID, "org_a").WillReturnRows(rows)
	got, ok := s.GetBuild("org_a", b.ID)
	if !ok || got.Scheme != "MyApp" || len(got.Log) != 1 {
		t.Errorf("get: %+v %v", got, ok)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestPGClaimBuildAtomic(t *testing.T) {
	s, mock := newMockStore(t)
	rows := pgxmock.NewRows([]string{"id", "org", "scheme", "ref", "status", "host", "region", "created_at", "log"}).
		AddRow("bld_x", "org_a", "MyApp", "", StatusRunning, "mac-1", "", testTime, []string{})
	mock.ExpectQuery("UPDATE builds SET status").
		WithArgs("mac-1", "").WillReturnRows(rows)
	claim := s.ClaimNext("mac-1", "", []string{"build"})
	if claim == nil || claim.Kind != "build" || claim.Build.Host != "mac-1" {
		t.Fatalf("claim: %+v", claim)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestPGSimActionRoundtrip(t *testing.T) {
	s, mock := newMockStore(t)
	simCols := []string{"id", "org", "device", "runtime", "status", "webrtc_url", "ax_tree_url", "host", "region", "pool", "recording", "created_at", "tree", "shot", "sim_log", "video"}
	mock.ExpectQuery("FROM sims WHERE id=\\$1 AND org=\\$2").
		WithArgs("sim_1", "org_a").
		WillReturnRows(pgxmock.NewRows(simCols).AddRow(
			"sim_1", "org_a", "iPhone 16", "", StatusRunning, "", "/v1/sims/sim_1/tree", "mac-1", "", "", false, testTime,
			[]byte(`{}`), []byte{}, []string{}, []byte{}))
	mock.ExpectQuery("INSERT INTO actions").
		WithArgs(pgxmock.AnyArg(), "sim_1", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"created_at"}).AddRow(testTime))
	a, ok := s.EnqueueSimAction("org_a", "sim_1", map[string]string{"tap": "@e1"})
	if !ok || a.Body["tap"] != "@e1" {
		t.Fatalf("enqueue: %+v %v", a, ok)
	}
	mock.ExpectExec(qm(`UPDATE actions SET done=true, result=$2 WHERE id=$1`)).
		WithArgs(a.ID, "tapped").WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	if !s.CompleteSimAction(a.ID, "tapped") {
		t.Error("complete returned false")
	}
	mock.ExpectQuery("FROM actions a JOIN sims s").
		WithArgs(a.ID, "org_a").
		WillReturnRows(pgxmock.NewRows([]string{"id", "sim_id", "body", "done", "result", "created_at"}).
			AddRow(a.ID, "sim_1", []byte(`{"tap":"@e1"}`), true, "tapped", testTime))
	got, ok := s.GetSimAction("org_a", a.ID)
	if !ok || !got.Done || got.Result != "tapped" {
		t.Errorf("get action: %+v %v", got, ok)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestPGSetGetSimTree(t *testing.T) {
	s, mock := newMockStore(t)
	mock.ExpectExec(qm(`UPDATE sims SET tree=$2 WHERE id=$1`)).
		WithArgs("sim_1", pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	if !s.SetSimTree("sim_1", []byte(`{"root":"@e1"}`)) {
		t.Error("set tree returned false")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestPGHeartbeatUpsertShape(t *testing.T) {
	s, mock := newMockStore(t)
	// RecordHeartbeat normalizes "" -> "any" and upserts on (pool, region).
	mock.ExpectExec("INTO host_beats").
		WithArgs("sg-1", "any").WillReturnResult(pgxmock.NewResult("INSERT", 1))
	s.RecordHeartbeat("sg-1", "")
	mock.ExpectExec("INTO host_beats").
		WithArgs("sg-1", "asia-southeast1").WillReturnResult(pgxmock.NewResult("INSERT", 1))
	s.RecordHeartbeat("sg-1", "asia-southeast1")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestPGFleetRegionsShape(t *testing.T) {
	s, mock := newMockStore(t)
	beats := pgxmock.NewRows([]string{"region", "pool", "last_seen"}).
		AddRow("asia-southeast1", "emu-sg-1", testTime).
		AddRow("asia-southeast1", "emu-sg-2", testTime).
		AddRow("asia-northeast1", "emu-jp-1", testTime)
	mock.ExpectQuery("FROM host_beats").WillReturnRows(beats)
	mock.ExpectQuery("FROM builds").
		WithArgs("org_a").
		WillReturnRows(pgxmock.NewRows([]string{"region", "count"}).
			AddRow("asia-northeast1", 2).AddRow("", 1))
	mock.ExpectQuery("FROM sims").
		WithArgs("org_a").
		WillReturnRows(pgxmock.NewRows([]string{"region", "count"}))
	mock.ExpectQuery("FROM emus").
		WithArgs("org_a").
		WillReturnRows(pgxmock.NewRows([]string{"region", "count"}).
			AddRow("asia-southeast1", 3))
	regs := s.FleetRegions("org_a")
	if len(regs) != 2 {
		t.Fatalf("regions = %+v, want 2", regs)
	}
	byRegion := map[string]RegionStatus{}
	for _, r := range regs {
		byRegion[r.Region] = r
	}
	// tokyo: 2 pinned + 1 unpinned; sg: 3 pinned + 1 unpinned.
	if byRegion["asia-northeast1"].Queued != 3 {
		t.Errorf("tokyo queued = %d, want 3", byRegion["asia-northeast1"].Queued)
	}
	if byRegion["asia-southeast1"].Queued != 4 {
		t.Errorf("sg queued = %d, want 4", byRegion["asia-southeast1"].Queued)
	}
	if byRegion["asia-southeast1"].Hosts != 2 {
		t.Errorf("sg hosts = %d, want 2", byRegion["asia-southeast1"].Hosts)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestPGShareRoundtrip(t *testing.T) {
	s, mock := newMockStore(t)
	mock.ExpectExec(qm(`INSERT INTO guest_shares (id, org, kind, resource_id, hash, expires_at, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`)).
		WithArgs(pgxmock.AnyArg(), "org_a", "sim", "sim_1", pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	sh, secret, ok := s.CreateGuestShare("org_a", "sim", "sim_1", time.Hour)
	if !ok || sh == nil || secret == "" {
		t.Fatalf("mint: %+v %v", sh, ok)
	}
	future := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("FROM guest_shares WHERE hash=").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"id", "org", "kind", "resource_id", "expires_at", "revoked", "created_at"}).
			AddRow(sh.ID, "org_a", "sim", "sim_1", future, false, testTime))
	got, ok := s.LookupGuestShare(secret)
	if !ok || got.ID != sh.ID || got.ResourceID != "sim_1" {
		t.Errorf("lookup: %+v %v", got, ok)
	}
	mock.ExpectExec(qm(`UPDATE guest_shares SET revoked=true WHERE id=$1 AND org=$2`)).
		WithArgs(sh.ID, "org_a").WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	if !s.RevokeGuestShare("org_a", sh.ID) {
		t.Error("revoke returned false")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}
