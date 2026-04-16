CREATE TABLE IF NOT EXISTS tasks (
  id           TEXT    PRIMARY KEY,
  title        TEXT    NOT NULL,
  note         TEXT    NOT NULL DEFAULT '',
  due          TEXT    NOT NULL,
  priority     TEXT    NOT NULL DEFAULT 'medium',
  status       TEXT    NOT NULL DEFAULT 'queued',
  sort_order   INTEGER NOT NULL DEFAULT 0,
  last_updated BIGINT  NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_tasks_status_sort_order
  ON tasks(status, sort_order, last_updated DESC);

CREATE TABLE IF NOT EXISTS archived_tasks (
  id          TEXT   PRIMARY KEY,
  title       TEXT   NOT NULL,
  note        TEXT   NOT NULL DEFAULT '',
  due         TEXT   NOT NULL,
  priority    TEXT   NOT NULL,
  status      TEXT   NOT NULL,
  archived_at BIGINT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_archived_tasks_archived_at
  ON archived_tasks(archived_at DESC);

CREATE TABLE IF NOT EXISTS users (
  username      TEXT PRIMARY KEY,
  password_hash TEXT NOT NULL,
  role          TEXT NOT NULL DEFAULT 'admin',
  created_at    BIGINT NOT NULL,
  updated_at    BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
  token        TEXT PRIMARY KEY,
  username     TEXT NOT NULL REFERENCES users(username) ON DELETE CASCADE,
  created_at   BIGINT NOT NULL,
  expires_at   BIGINT NOT NULL,
  revoked_at   BIGINT,
  last_seen_at BIGINT NOT NULL,
  client_ip    TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_sessions_username ON sessions(username);

CREATE TABLE IF NOT EXISTS auth_audit_logs (
  id         BIGSERIAL PRIMARY KEY,
  username   TEXT NOT NULL DEFAULT '',
  event_type TEXT NOT NULL,
  outcome    TEXT NOT NULL,
  client_ip  TEXT NOT NULL DEFAULT '',
  details    TEXT NOT NULL DEFAULT '',
  created_at BIGINT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_auth_audit_logs_created_at
  ON auth_audit_logs(created_at);
