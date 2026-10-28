-- Sidekick control-plane schema. Applied on boot (IF NOT EXISTS):
-- Postgres is the durable scheduler; without DATABASE_URL the server runs
-- the in-memory store instead. Blobs (screenshots, MP4s) live inline up to
-- their API caps; graduate to object storage when video minutes grow.
CREATE TABLE IF NOT EXISTS builds (
  id         TEXT PRIMARY KEY,
  org        TEXT NOT NULL,
  scheme     TEXT NOT NULL,
  ref        TEXT NOT NULL DEFAULT '',
  status     TEXT NOT NULL,
  host       TEXT NOT NULL DEFAULT '',
  region     TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL,
  log        TEXT[] NOT NULL DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS builds_org_status ON builds (org, status);

CREATE TABLE IF NOT EXISTS sims (
  id          TEXT PRIMARY KEY,
  org         TEXT NOT NULL,
  device      TEXT NOT NULL,
  runtime     TEXT NOT NULL DEFAULT '',
  status      TEXT NOT NULL,
  webrtc_url  TEXT NOT NULL DEFAULT '',
  ax_tree_url TEXT NOT NULL DEFAULT '',
  host        TEXT NOT NULL DEFAULT '',
  region      TEXT NOT NULL DEFAULT '',
  recording   BOOLEAN NOT NULL DEFAULT FALSE,
  created_at  TIMESTAMPTZ NOT NULL,
  tree        JSONB,
  shot        BYTEA,
  sim_log     TEXT[] NOT NULL DEFAULT '{}',
  video       BYTEA
);
CREATE INDEX IF NOT EXISTS sims_org ON sims (org);

CREATE TABLE IF NOT EXISTS actions (
  id         TEXT PRIMARY KEY,
  sim_id     TEXT NOT NULL REFERENCES sims (id) ON DELETE CASCADE,
  body       JSONB NOT NULL,
  done       BOOLEAN NOT NULL DEFAULT FALSE,
  result     TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS actions_sim ON actions (sim_id, done);

CREATE TABLE IF NOT EXISTS emus (
  id         TEXT PRIMARY KEY,
  org        TEXT NOT NULL,
  profile    TEXT NOT NULL,
  status     TEXT NOT NULL,
  adb_target TEXT NOT NULL DEFAULT '',
  host       TEXT NOT NULL DEFAULT '',
  region     TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL,
  tree       JSONB,
  shot       BYTEA,
  emu_log    TEXT[] NOT NULL DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS emus_org ON emus (org);

-- Scoped instance tokens: only the sha256 hash persists; the secret itself
-- is shown once at creation and never stored.
CREATE TABLE IF NOT EXISTS instance_tokens (
  id         TEXT PRIMARY KEY,
  org        TEXT NOT NULL,
  name       TEXT NOT NULL,
  scopes     TEXT[] NOT NULL,
  hash       TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked    BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS instance_tokens_org ON instance_tokens (org);

-- Evolve older databases: region pinning landed after the initial schema.
ALTER TABLE builds ADD COLUMN IF NOT EXISTS region TEXT NOT NULL DEFAULT '';
ALTER TABLE sims ADD COLUMN IF NOT EXISTS region TEXT NOT NULL DEFAULT '';
ALTER TABLE emus ADD COLUMN IF NOT EXISTS region TEXT NOT NULL DEFAULT '';
ALTER TABLE sims ADD COLUMN IF NOT EXISTS pool TEXT NOT NULL DEFAULT '';
ALTER TABLE emus ADD COLUMN IF NOT EXISTS pool TEXT NOT NULL DEFAULT '';

-- Host liveness: one row per reporting pool+region, refreshed on every
-- POST /v1/host/claims/next poll (hit or empty). GET /v1/fleet/regions folds
-- this into the live region map. Empty region normalizes to 'any' on write.
CREATE TABLE IF NOT EXISTS host_beats (
  pool      TEXT NOT NULL,
  region    TEXT NOT NULL DEFAULT 'any',
  last_seen TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (pool, region)
);

-- Signed read-only guest shares: one row per minted link. Only the sha256
-- hash of the secret persists; the `sk_share_...` secret itself is shown
-- once at mint and never stored. Lookup checks revoked/expiry; expired or
-- revoked secrets resolve as missing (404, no oracle). Resource rows are
-- read via GetSimAny/GetEmuAny (the share itself is the capability).
CREATE TABLE IF NOT EXISTS guest_shares (
  id          TEXT NOT NULL,
  org         TEXT NOT NULL,
  kind        TEXT NOT NULL, -- 'sim' | 'emu'
  resource_id TEXT NOT NULL,
  hash        TEXT NOT NULL UNIQUE,
  expires_at  TIMESTAMPTZ NOT NULL,
  revoked     BOOLEAN NOT NULL DEFAULT FALSE,
  created_at  TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (id, org)
);
CREATE INDEX IF NOT EXISTS guest_shares_org ON guest_shares (org);
