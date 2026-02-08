-- Walk.io Database Schema
-- Complete initial schema for AppFs building and Crutch (VM instance) runtime management

-- Applications: Container image definitions with runtime configuration
CREATE TABLE IF NOT EXISTS apps (
    id TEXT PRIMARY KEY,
    image_name TEXT NOT NULL,
    env_json TEXT,
    args_json TEXT,
    work_dir TEXT,
    
    -- Firecracker VM configuration for this app
    kernel_path TEXT NOT NULL,                           -- path to firecracker kernel binary
    state_fs_size_bytes INTEGER NOT NULL DEFAULT 1073741824,  -- 1GB default StateFS size
    
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Build Jobs: Tracks OCI image build progress and results
CREATE TABLE IF NOT EXISTS build_jobs (
    id TEXT PRIMARY KEY,
    app_id TEXT NOT NULL,
    image_name TEXT NOT NULL,
    status TEXT NOT NULL,           -- queued/building/completed/failed
    digest TEXT,                    -- OCI image digest after build
    block_device_path TEXT,         -- path to resulting .ext4 AppFs file
    error TEXT,
    started_at INTEGER,
    completed_at INTEGER,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_build_jobs_app_id ON build_jobs(app_id);
CREATE INDEX IF NOT EXISTS idx_build_jobs_status ON build_jobs(status);
CREATE INDEX IF NOT EXISTS idx_build_jobs_created_at_desc ON build_jobs(created_at DESC);

-- Crutches: Running VM instances (a Crutch is an instantiation of an App)
CREATE TABLE IF NOT EXISTS crutches (
    id TEXT PRIMARY KEY,                      -- UUID of this VM instance
    app_id TEXT NOT NULL,
    pid INTEGER NOT NULL,                     -- firecracker process PID
    socket_path TEXT NOT NULL,                -- firecracker control socket path
    state_fs_path TEXT NOT NULL,              -- path to StateFS block device
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_crutches_app_id ON crutches(app_id);
CREATE INDEX IF NOT EXISTS idx_crutches_created_at ON crutches(created_at DESC);

-- API Tokens: Authentication credentials for API access
CREATE TABLE IF NOT EXISTS api_tokens (
    id TEXT PRIMARY KEY,
    token_hash TEXT NOT NULL UNIQUE,
    name TEXT,
    created_at INTEGER NOT NULL
);
