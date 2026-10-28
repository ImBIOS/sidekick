// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schemaSQL string

// Querier is the pgx surface PGStore needs. *pgxpool.Pool satisfies it in
// production; pgxmock satisfies it in tests.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (int64, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// PGStore is the durable Scheduler: same method set as the in-memory Store,
// backed by Postgres. ClaimNext uses SELECT ... FOR UPDATE SKIP LOCKED so
// concurrent hosts in one pool never double-claim.
type PGStore struct {
	q   Querier
	seq atomic.Uint64
}

// Open connects, applies schema.sql, and returns the store.
func Open(ctx context.Context, databaseURL string) (*PGStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	store := &PGStore{q: poolAdapter{pool}}
	if err := Migrate(ctx, store.q); err != nil {
		pool.Close()
		return nil, err
	}
	return store, nil
}

// Migrate applies schema.sql. Open already runs it; exported for tooling.
func Migrate(ctx context.Context, q Querier) error {
	_, err := q.Exec(ctx, schemaSQL)
	return err
}

// poolAdapter reshapes *pgxpool.Pool Exec (CommandTag) to Querier (int64).
type poolAdapter struct {
	pool *pgxpool.Pool
}

func (a poolAdapter) Exec(ctx context.Context, sql string, args ...any) (int64, error) {
	tag, err := a.pool.Exec(ctx, sql, args...)
	return tag.RowsAffected(), err
}

func (a poolAdapter) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return a.pool.Query(ctx, sql, args...)
}

func (a poolAdapter) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return a.pool.QueryRow(ctx, sql, args...)
}

func (s *PGStore) nextID(prefix string) string {
	n := s.seq.Add(1)
	return fmt.Sprintf("%s_%s-%d", prefix, time.Now().UTC().Format("20060102"), n)
}

// --- builds ---

func (s *PGStore) CreateBuild(org OrgID, scheme, ref, region string) *Build {
	b := &Build{
		ID: s.nextID("bld"), Org: org, Scheme: scheme, Ref: ref,
		Status: StatusQueued, Region: region, CreatedAt: time.Now().UTC(),
		Log: []string{"[sidekick] queued; waiting for a build host"},
	}
	_, _ = s.q.Exec(context.Background(),
		`INSERT INTO builds (id, org, scheme, ref, status, region, created_at, log) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		b.ID, string(org), scheme, ref, b.Status, b.Region, b.CreatedAt, b.Log)
	return b
}

func scanBuild(row pgx.Row) (*Build, error) {
	var b Build
	var org string
	err := row.Scan(&b.ID, &org, &b.Scheme, &b.Ref, &b.Status, &b.Host, &b.Region, &b.CreatedAt, &b.Log)
	if err != nil {
		return nil, err
	}
	b.Org = OrgID(org)
	return &b, nil
}

func (s *PGStore) GetBuild(org OrgID, id string) (*Build, bool) {
	b, err := scanBuild(s.q.QueryRow(context.Background(),
		`SELECT id, org, scheme, ref, status, host, region, created_at, log FROM builds WHERE id=$1 AND org=$2`, id, string(org)))
	if err != nil {
		return nil, false
	}
	return b, true
}

func (s *PGStore) ListBuilds(org OrgID) []*Build {
	rows, err := s.q.Query(context.Background(),
		`SELECT id, org, scheme, ref, status, host, region, created_at, log FROM builds WHERE org=$1 ORDER BY created_at DESC`, string(org))
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*Build
	for rows.Next() {
		if b, err := scanBuild(rows); err == nil {
			out = append(out, b)
		}
	}
	return out
}

func (s *PGStore) AppendBuildLog(org OrgID, id string, lines []string) bool {
	n, err := s.q.Exec(context.Background(),
		`UPDATE builds SET log = log || $3 WHERE id=$1 AND org=$2`, id, string(org), lines)
	return err == nil && n == 1
}

func (s *PGStore) FinishBuild(org OrgID, id, status string) bool {
	n, err := s.q.Exec(context.Background(),
		`UPDATE builds SET status=$3 WHERE id=$1 AND org=$2`, id, string(org), status)
	return err == nil && n == 1
}

// HostAppendBuildLog appends daemon lines without org scoping (trusted host).
func (s *PGStore) HostAppendBuildLog(id string, lines []string) bool {
	n, err := s.q.Exec(context.Background(),
		`UPDATE builds SET log = log || $2 WHERE id=$1`, id, lines)
	return err == nil && n == 1
}

// HostFinishBuild marks a build terminal without org scoping (trusted host).
func (s *PGStore) HostFinishBuild(id, status string) bool {
	n, err := s.q.Exec(context.Background(),
		`UPDATE builds SET status=$2 WHERE id=$1`, id, status)
	return err == nil && n == 1
}

// --- sims ---

func (s *PGStore) CreateSim(org OrgID, device, runtime, region, pool string) *Sim {
	sim := &Sim{
		ID: s.nextID("sim"), Org: org, Device: device, Runtime: runtime,
		Status: StatusProvisioning, Region: region, Pool: pool, CreatedAt: time.Now().UTC(),
	}
	sim.AXTreeURL = "/v1/sims/" + sim.ID + "/tree"
	_, _ = s.q.Exec(context.Background(),
		`INSERT INTO sims (id, org, device, runtime, status, region, pool, ax_tree_url, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		sim.ID, string(org), device, runtime, sim.Status, sim.Region, sim.Pool, sim.AXTreeURL, sim.CreatedAt)
	return sim
}

func scanSim(row pgx.Row) (*Sim, error) {
	var sim Sim
	var org string
	var tree []byte
	err := row.Scan(&sim.ID, &org, &sim.Device, &sim.Runtime, &sim.Status,
		&sim.WebRTCURL, &sim.AXTreeURL, &sim.Host, &sim.Region, &sim.Pool, &sim.Recording, &sim.CreatedAt,
		&tree, &sim.Shot, &sim.SimLog, &sim.Video)
	if err != nil {
		return nil, err
	}
	sim.Org = OrgID(org)
	sim.Tree = tree
	return &sim, nil
}

const simCols = `id, org, device, runtime, status, webrtc_url, ax_tree_url, host, region, pool, recording, created_at, tree, shot, sim_log, video`

func (s *PGStore) GetSim(org OrgID, id string) (*Sim, bool) {
	sim, err := scanSim(s.q.QueryRow(context.Background(),
		`SELECT `+simCols+` FROM sims WHERE id=$1 AND org=$2`, id, string(org)))
	if err != nil {
		return nil, false
	}
	return sim, true
}

func (s *PGStore) ListSims(org OrgID) []*Sim {
	rows, err := s.q.Query(context.Background(),
		`SELECT `+simCols+` FROM sims WHERE org=$1 ORDER BY created_at DESC`, string(org))
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*Sim
	for rows.Next() {
		if sim, err := scanSim(rows); err == nil {
			out = append(out, sim)
		}
	}
	return out
}

func (s *PGStore) DeleteSim(org OrgID, id string) bool {
	n, err := s.q.Exec(context.Background(), `DELETE FROM sims WHERE id=$1 AND org=$2`, id, string(org))
	return err == nil && n == 1
}

func (s *PGStore) HostSimReady(id, host, webrtcURL string) bool {
	n, err := s.q.Exec(context.Background(),
		`UPDATE sims SET host=$2, status=$3, webrtc_url=$4 WHERE id=$1 AND (host='' OR host=$2)`,
		id, host, StatusRunning, webrtcURL)
	return err == nil && n == 1
}

func (s *PGStore) SetSimTree(id string, raw json.RawMessage) bool {
	n, err := s.q.Exec(context.Background(), `UPDATE sims SET tree=$2 WHERE id=$1`, id, []byte(raw))
	return err == nil && n == 1
}

func (s *PGStore) SetSimShot(id string, png []byte) bool {
	n, err := s.q.Exec(context.Background(), `UPDATE sims SET shot=$2 WHERE id=$1`, id, png)
	return err == nil && n == 1
}

func (s *PGStore) AppendSimLog(id string, lines []string) bool {
	n, err := s.q.Exec(context.Background(), `UPDATE sims SET sim_log = sim_log || $2 WHERE id=$1`, id, lines)
	return err == nil && n == 1
}

func (s *PGStore) SetSimRecording(id string, rec bool) bool {
	n, err := s.q.Exec(context.Background(), `UPDATE sims SET recording=$2 WHERE id=$1`, id, rec)
	return err == nil && n == 1
}

func (s *PGStore) SetSimVideo(id string, mp4 []byte) bool {
	n, err := s.q.Exec(context.Background(), `UPDATE sims SET video=$2, recording=false WHERE id=$1`, id, mp4)
	return err == nil && n == 1
}

// --- sim actions ---

func (s *PGStore) EnqueueSimAction(org OrgID, simID string, body map[string]string) (*QueuedAction, bool) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, false
	}
	// Org check first: never queue into a foreign sim.
	if _, ok := s.GetSim(org, simID); !ok {
		return nil, false
	}
	a := &QueuedAction{ID: s.nextID("act"), SimID: simID, Body: body, CreatedAt: time.Now().UTC()}
	err = s.q.QueryRow(context.Background(),
		`INSERT INTO actions (id, sim_id, body, created_at) VALUES ($1,$2,$3,$4) RETURNING created_at`,
		a.ID, simID, raw, a.CreatedAt).Scan(&a.CreatedAt)
	if err != nil {
		return nil, false
	}
	return a, true
}

func scanAction(row pgx.Row) (*QueuedAction, error) {
	var a QueuedAction
	var body []byte
	err := row.Scan(&a.ID, &a.SimID, &body, &a.Done, &a.Result, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(body, &a.Body); err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *PGStore) CompleteSimAction(id, result string) bool {
	n, err := s.q.Exec(context.Background(),
		`UPDATE actions SET done=true, result=$2 WHERE id=$1`, id, result)
	return err == nil && n == 1
}

func (s *PGStore) GetSimAction(org OrgID, id string) (*QueuedAction, bool) {
	a, err := scanAction(s.q.QueryRow(context.Background(),
		`SELECT a.id, a.sim_id, a.body, a.done, a.result, a.created_at FROM actions a JOIN sims s ON s.id=a.sim_id WHERE a.id=$1 AND s.org=$2`, id, string(org)))
	if err != nil {
		return nil, false
	}
	return a, true
}

// --- emus ---

func (s *PGStore) CreateEmu(org OrgID, profile, region, pool string) *Emu {
	emu := &Emu{ID: s.nextID("emu"), Org: org, Profile: profile, Status: StatusProvisioning, Region: region, Pool: pool, CreatedAt: time.Now().UTC()}
	_, _ = s.q.Exec(context.Background(),
		`INSERT INTO emus (id, org, profile, status, region, pool, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		emu.ID, string(org), profile, emu.Status, emu.Region, emu.Pool, emu.CreatedAt)
	return emu
}

func scanEmu(row pgx.Row) (*Emu, error) {
	var emu Emu
	var org string
	var tree []byte
	err := row.Scan(&emu.ID, &org, &emu.Profile, &emu.Status, &emu.ADBTarget,
		&emu.Host, &emu.Region, &emu.Pool, &emu.CreatedAt, &tree, &emu.Shot, &emu.EmuLog)
	if err != nil {
		return nil, err
	}
	emu.Org = OrgID(org)
	emu.Tree = tree
	return &emu, nil
}

const emuCols = `id, org, profile, status, adb_target, host, region, pool, created_at, tree, shot, emu_log`

func (s *PGStore) GetEmu(org OrgID, id string) (*Emu, bool) {
	emu, err := scanEmu(s.q.QueryRow(context.Background(),
		`SELECT `+emuCols+` FROM emus WHERE id=$1 AND org=$2`, id, string(org)))
	if err != nil {
		return nil, false
	}
	return emu, true
}

func (s *PGStore) ListEmus(org OrgID) []*Emu {
	rows, err := s.q.Query(context.Background(),
		`SELECT `+emuCols+` FROM emus WHERE org=$1 ORDER BY created_at DESC`, string(org))
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*Emu
	for rows.Next() {
		if emu, err := scanEmu(rows); err == nil {
			out = append(out, emu)
		}
	}
	return out
}

func (s *PGStore) DeleteEmu(org OrgID, id string) bool {
	n, err := s.q.Exec(context.Background(), `DELETE FROM emus WHERE id=$1 AND org=$2`, id, string(org))
	return err == nil && n == 1
}

func (s *PGStore) HostEmuReady(id, host, adbTarget string) bool {
	n, err := s.q.Exec(context.Background(),
		`UPDATE emus SET host=$2, status=$3, adb_target=$4 WHERE id=$1 AND (host='' OR host=$2)`,
		id, host, StatusRunning, adbTarget)
	return err == nil && n == 1
}

func (s *PGStore) SetEmuTree(id string, raw json.RawMessage) bool {
	n, err := s.q.Exec(context.Background(), `UPDATE emus SET tree=$2 WHERE id=$1`, id, []byte(raw))
	return err == nil && n == 1
}

func (s *PGStore) SetEmuShot(id string, png []byte) bool {
	n, err := s.q.Exec(context.Background(), `UPDATE emus SET shot=$2 WHERE id=$1`, id, png)
	return err == nil && n == 1
}

func (s *PGStore) AppendEmuLog(id string, lines []string) bool {
	n, err := s.q.Exec(context.Background(), `UPDATE emus SET emu_log = emu_log || $2 WHERE id=$1`, id, lines)
	return err == nil && n == 1
}

// --- instance tokens ---

func (s *PGStore) CreateInstanceToken(org OrgID, name string, scopes []string, ttl time.Duration) (*InstanceToken, string, bool) {
	if name == "" || len(scopes) == 0 || ttl <= 0 || ttl > 30*24*time.Hour {
		return nil, "", false
	}
	seen := map[string]bool{}
	for _, sc := range scopes {
		if !validScope(sc) || seen[sc] {
			return nil, "", false
		}
		seen[sc] = true
	}
	secret, hash, err := mintSecret()
	if err != nil {
		return nil, "", false
	}
	now := time.Now().UTC()
	tok := &InstanceToken{
		ID: s.nextID("tok"), Org: org, Name: name, Scopes: scopes,
		ExpiresAt: now.Add(ttl), CreatedAt: now,
	}
	_, err = s.q.Exec(context.Background(),
		`INSERT INTO instance_tokens (id, org, name, scopes, hash, expires_at, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		tok.ID, string(org), name, scopes, hash, tok.ExpiresAt, tok.CreatedAt)
	if err != nil {
		return nil, "", false
	}
	return tok, secret, true
}

func (s *PGStore) LookupInstanceToken(secret string) (*InstanceToken, bool) {
	var tok InstanceToken
	var org string
	err := s.q.QueryRow(context.Background(),
		`SELECT id, org, name, scopes, expires_at, revoked, created_at FROM instance_tokens WHERE hash=$1`,
		hashSecret(secret)).Scan(&tok.ID, &org, &tok.Name, &tok.Scopes, &tok.ExpiresAt, &tok.Revoked, &tok.CreatedAt)
	if err != nil {
		return nil, false
	}
	tok.Org = OrgID(org)
	if tok.Revoked || time.Now().UTC().After(tok.ExpiresAt) {
		return nil, false
	}
	return &tok, true
}

func (s *PGStore) ListInstanceTokens(org OrgID) []*InstanceToken {
	rows, err := s.q.Query(context.Background(),
		`SELECT id, name, scopes, expires_at, revoked, created_at FROM instance_tokens WHERE org=$1 ORDER BY created_at DESC`, string(org))
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*InstanceToken
	for rows.Next() {
		var tok InstanceToken
		if err := rows.Scan(&tok.ID, &tok.Name, &tok.Scopes, &tok.ExpiresAt, &tok.Revoked, &tok.CreatedAt); err == nil {
			tok.Org = org
			out = append(out, &tok)
		}
	}
	return out
}

func (s *PGStore) RevokeInstanceToken(org OrgID, id string) bool {
	n, err := s.q.Exec(context.Background(),
		`UPDATE instance_tokens SET revoked=true WHERE id=$1 AND org=$2`, id, string(org))
	return err == nil && n == 1
}

// --- guest shares (PGStore) ---

// CreateGuestShare mints a signed read-only share (see Store.CreateGuestShare
// for the model). The org-ownership check happens in the HTTP handler (it
// GetSim/GetEmu's first); the row write is unconditional.
func (s *PGStore) CreateGuestShare(org OrgID, kind, resourceID string, ttl time.Duration) (*GuestShare, string, bool) {
	if (kind != "sim" && kind != "emu") || resourceID == "" || ttl <= 0 || ttl > 30*24*time.Hour {
		return nil, "", false
	}
	secret, hash, err := mintShareSecret()
	if err != nil {
		return nil, "", false
	}
	now := time.Now().UTC()
	sh := &GuestShare{
		ID: s.nextID("shr"), Org: org, Kind: kind, ResourceID: resourceID,
		ExpiresAt: now.Add(ttl), CreatedAt: now,
	}
	_, err = s.q.Exec(context.Background(),
		`INSERT INTO guest_shares (id, org, kind, resource_id, hash, expires_at, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		sh.ID, string(org), kind, resourceID, hash, sh.ExpiresAt, sh.CreatedAt)
	if err != nil {
		return nil, "", false
	}
	return sh, secret, true
}

func scanShare(row pgx.Row) (*GuestShare, error) {
	var sh GuestShare
	var org string
	err := row.Scan(&sh.ID, &org, &sh.Kind, &sh.ResourceID, &sh.ExpiresAt, &sh.Revoked, &sh.CreatedAt)
	if err != nil {
		return nil, err
	}
	sh.Org = OrgID(org)
	return &sh, nil
}

// LookupGuestShare resolves a share secret; expired/revoked read as missing.
func (s *PGStore) LookupGuestShare(secret string) (*GuestShare, bool) {
	sh, err := scanShare(s.q.QueryRow(context.Background(),
		`SELECT id, org, kind, resource_id, expires_at, revoked, created_at FROM guest_shares WHERE hash=$1`,
		hashSecret(secret)))
	if err != nil {
		return nil, false
	}
	if sh.Revoked || time.Now().UTC().After(sh.ExpiresAt) {
		return nil, false
	}
	return sh, true
}

// RevokeGuestShare kills a share immediately (org-scoped).
func (s *PGStore) RevokeGuestShare(org OrgID, id string) bool {
	n, err := s.q.Exec(context.Background(),
		`UPDATE guest_shares SET revoked=true WHERE id=$1 AND org=$2`, id, string(org))
	return err == nil && n == 1
}

// GetSimAny fetches a sim without org scoping (the share is the capability).
func (s *PGStore) GetSimAny(id string) (*Sim, bool) {
	sim, err := scanSim(s.q.QueryRow(context.Background(),
		`SELECT `+simCols+` FROM sims WHERE id=$1`, id))
	if err != nil {
		return nil, false
	}
	return sim, true
}

// GetEmuAny fetches an emu without org scoping (see GetSimAny).
func (s *PGStore) GetEmuAny(id string) (*Emu, bool) {
	emu, err := scanEmu(s.q.QueryRow(context.Background(),
		`SELECT `+emuCols+` FROM emus WHERE id=$1`, id))
	if err != nil {
		return nil, false
	}
	return emu, true
}

// --- claims ---

// ClaimNext mirrors Store.ClaimNext with SKIP LOCKED atomicity: concurrent
// hosts in one pool can never double-claim the same build.
// Every poll also refreshes the pool+region heartbeat backing
// GET /v1/fleet/regions, so the fleet map reflects hosts that phone home.
func (s *PGStore) ClaimNext(pool, region string, kinds []string) *Claim {
	ctx := context.Background()
	want := func(k string) bool {
		if len(kinds) == 0 {
			return true
		}
		for _, w := range kinds {
			if w == k {
				return true
			}
		}
		return false
	}
	if want("build") {
		var b Build
		var org string
		err := s.q.QueryRow(ctx, `UPDATE builds SET status='running', host=$1, region=$2,
			log = log || ('[sidekick] build claimed by ' || $1)
			WHERE id = (SELECT id FROM builds WHERE status='queued' AND (region='' OR region=$2) ORDER BY created_at LIMIT 1 FOR UPDATE SKIP LOCKED)
			RETURNING id, org, scheme, ref, status, host, region, created_at, log`, pool, region).
			Scan(&b.ID, &org, &b.Scheme, &b.Ref, &b.Status, &b.Host, &b.Region, &b.CreatedAt, &b.Log)
		if err == nil {
			b.Org = OrgID(org)
			_, _ = s.q.Exec(ctx, `INSERT INTO host_beats (pool, region, last_seen) VALUES ($1,$2,now())
				ON CONFLICT (pool, region) DO UPDATE SET last_seen=excluded.last_seen`, pool, region)
			return &Claim{Kind: "build", Build: &b}
		}
	}
	if want("sim") {
		var sim Sim
		var org string
		var tree []byte
		err := s.q.QueryRow(ctx, `UPDATE sims SET host=$1
			WHERE id = (SELECT id FROM sims WHERE status='provisioning' AND host='' AND (region='' OR region=$2) AND (pool='' OR pool=$3) ORDER BY created_at LIMIT 1 FOR UPDATE SKIP LOCKED)
			RETURNING `+simCols, pool, region, pool).Scan(&sim.ID, &org, &sim.Device, &sim.Runtime, &sim.Status,
			&sim.WebRTCURL, &sim.AXTreeURL, &sim.Host, &sim.Region, &sim.Pool, &sim.Recording, &sim.CreatedAt,
			&tree, &sim.Shot, &sim.SimLog, &sim.Video)
		if err == nil {
			sim.Org = OrgID(org)
			sim.Tree = tree
			_, _ = s.q.Exec(ctx, `INSERT INTO host_beats (pool, region, last_seen) VALUES ($1,$2,now())
				ON CONFLICT (pool, region) DO UPDATE SET last_seen=excluded.last_seen`, pool, region)
			return &Claim{Kind: "sim", Sim: &sim}
		}
	}
	if want("emu") {
		var emu Emu
		var org string
		var tree []byte
		err := s.q.QueryRow(ctx, `UPDATE emus SET host=$1
			WHERE id = (SELECT id FROM emus WHERE status='provisioning' AND host='' AND (region='' OR region=$2) AND (pool='' OR pool=$3) ORDER BY created_at LIMIT 1 FOR UPDATE SKIP LOCKED)
			RETURNING `+emuCols, pool, region, pool).Scan(&emu.ID, &org, &emu.Profile, &emu.Status, &emu.ADBTarget,
			&emu.Host, &emu.Region, &emu.Pool, &emu.CreatedAt, &tree, &emu.Shot, &emu.EmuLog)
		if err == nil {
			emu.Org = OrgID(org)
			emu.Tree = tree
			_, _ = s.q.Exec(ctx, `INSERT INTO host_beats (pool, region, last_seen) VALUES ($1,$2,now())
				ON CONFLICT (pool, region) DO UPDATE SET last_seen=excluded.last_seen`, pool, region)
			return &Claim{Kind: "emu", Emu: &emu}
		}
	}
	if want("sim-action") {
		var a QueuedAction
		var body []byte
		var simID string
		err := s.q.QueryRow(ctx, `SELECT a.id, a.sim_id, a.body, a.done, a.result, a.created_at
			FROM actions a JOIN sims s ON s.id = a.sim_id
			WHERE s.host=$1 AND (s.region='' OR s.region=$2) AND a.done=false ORDER BY a.created_at LIMIT 1`, pool, region).
			Scan(&a.ID, &simID, &body, &a.Done, &a.Result, &a.CreatedAt)
		if err == nil {
			a.SimID = simID
			_ = json.Unmarshal(body, &a.Body)
			_, _ = s.q.Exec(ctx, `INSERT INTO host_beats (pool, region, last_seen) VALUES ($1,$2,now())
				ON CONFLICT (pool, region) DO UPDATE SET last_seen=excluded.last_seen`, pool, region)
			return &Claim{Kind: "sim-action", Action: &a}
		}
	}
	return nil
}

// RecordHeartbeat stamps pool+region liveness (upsert; empty region reads as
// "any" so unpinned hosts still show in the fleet map).
func (s *PGStore) RecordHeartbeat(pool, region string) {
	if region == "" {
		region = "any"
	}
	_, _ = s.q.Exec(context.Background(), `INSERT INTO host_beats (pool, region, last_seen) VALUES ($1,$2,now())
		ON CONFLICT (pool, region) DO UPDATE SET last_seen=excluded.last_seen`, pool, region)
}

// FleetRegions folds host_beats by region for org: pools reporting, host
// count, last-seen, plus the org's queued (unclaimed) depth per region.
// Unpinned queued resources count toward every live region (any host could
// claim them); region-pinned ones only toward their own live row.
func (s *PGStore) FleetRegions(org OrgID) []RegionStatus {
	ctx := context.Background()
	rows, err := s.q.Query(ctx, `SELECT region, pool, last_seen FROM host_beats`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	type beat struct {
		region   string
		pool     string
		lastSeen time.Time
	}
	var beats []beat
	for rows.Next() {
		var b beat
		if err := rows.Scan(&b.region, &b.pool, &b.lastSeen); err == nil {
			beats = append(beats, b)
		}
	}
	byRegion := map[string]*RegionStatus{}
	poolSet := map[string]map[string]bool{}
	for _, b := range beats {
		st, ok := byRegion[b.region]
		if !ok {
			st = &RegionStatus{Region: b.region, LastSeen: b.lastSeen}
			byRegion[b.region] = st
		}
		st.Hosts++
		if st.LastSeen.Before(b.lastSeen) {
			st.LastSeen = b.lastSeen
		}
		if poolSet[b.region] == nil {
			poolSet[b.region] = map[string]bool{}
		}
		poolSet[b.region][b.pool] = true
	}
	for r, st := range byRegion {
		for p := range poolSet[r] {
			st.Pools = append(st.Pools, p)
		}
	}
	regions := make([]string, 0, len(byRegion))
	for r := range byRegion {
		regions = append(regions, r)
	}
	queued := map[string]int{}
	count := func(sql string) {
		qrows, err := s.q.Query(ctx, sql, string(org))
		if err != nil {
			return
		}
		defer qrows.Close()
		for qrows.Next() {
			var r string
			var n int
			if err := qrows.Scan(&r, &n); err == nil {
				if r == "" {
					r = "any"
				}
				queued[r] += n
			}
		}
	}
	count(`SELECT region, count(*) FROM builds WHERE org=$1 AND status='queued' GROUP BY region`)
	count(`SELECT region, count(*) FROM sims WHERE org=$1 AND status='provisioning' AND host='' GROUP BY region`)
	count(`SELECT region, count(*) FROM emus WHERE org=$1 AND status='provisioning' AND host='' GROUP BY region`)
	// Pinned depth only shows on its own live row; unpinned ('' or 'any')
	// depth is claimable from every live region. Regions with no beats never
	// appear — liveness, not configuration.
	unpinned := queued[""] + queued["any"]
	out := make([]RegionStatus, 0, len(byRegion))
	for _, r := range regions {
		st := byRegion[r]
		st.Queued = queued[r] + unpinned
		out = append(out, *st)
	}
	return out
}
