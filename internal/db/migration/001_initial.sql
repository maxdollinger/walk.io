CREATE TABLE IF NOT EXISTS apps (
    id TEXT PRIMARY KEY,
    image_name TEXT NOT NULL,
    env_json TEXT,
    args_json TEXT,
    work_dir TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS build_jobs (
    id TEXT PRIMARY KEY,
    app_id TEXT NOT NULL,
    image_name TEXT NOT NULL,
    status TEXT NOT NULL,
    digest TEXT,
    block_device_path TEXT,
    error TEXT,
    started_at INTEGER,
    completed_at INTEGER,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_build_jobs_app_id ON build_jobs(app_id);
CREATE INDEX IF NOT EXISTS idx_build_jobs_status ON build_jobs(status);
CREATE INDEX IF NOT EXISTS idx_build_jobs_created_at_desc ON build_jobs(created_at DESC);

CREATE TABLE IF NOT EXISTS api_tokens (
    id TEXT PRIMARY KEY,
    token_hash TEXT NOT NULL UNIQUE,
    name TEXT,
    created_at INTEGER NOT NULL
);
