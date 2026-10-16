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
