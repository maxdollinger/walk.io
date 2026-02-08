-- Initial schema for walk.io
-- Apps table: tracks applications with their base version
CREATE TABLE apps (
    id VARCHAR(255) PRIMARY KEY,
    digest VARCHAR(255) NOT NULL UNIQUE,
    base_version VARCHAR(255) NOT NULL,
    state_fs_size_bytes BIGINT NOT NULL DEFAULT 1073741824,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Crutches table: running VM instances
-- NOTE: state_fs_path is NOT stored; computed as /var/lib/walkio/state/{id}.ext4
CREATE TABLE crutches (
    id VARCHAR(255) PRIMARY KEY,
    app_id VARCHAR(255) NOT NULL,
    pid INT NOT NULL,
    socket_path VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (app_id) REFERENCES apps(id)
);

-- Build jobs table: tracks application builds
CREATE TABLE build_jobs (
    id VARCHAR(255) PRIMARY KEY,
    app_id VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (app_id) REFERENCES apps(id)
);
