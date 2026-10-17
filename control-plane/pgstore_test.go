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
	simCols := []string{"id", "org", "device", "runtime", "status", "webrtc_url", "ax_tree_url", "host", "region", "recording", "created_at", "tree", "shot", "sim_log", "video"}
	mock.ExpectQuery("FROM sims WHERE id=\\$1 AND org=\\$2").
		WithArgs("sim_1", "org_a").
		WillReturnRows(pgxmock.NewRows(simCols).AddRow(
			"sim_1", "org_a", "iPhone 16", "", StatusRunning, "", "/v1/sims/sim_1/tree", "mac-1", "", false, testTime,
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
