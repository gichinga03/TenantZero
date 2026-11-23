-- users, sessions, progress, journal

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE sessions (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    lab_id TEXT NOT NULL,
    state JSONB NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    last_active TIMESTAMPTZ NOT NULL,
    completed BOOLEAN DEFAULT FALSE
);

CREATE TABLE journal (
    id BIGSERIAL PRIMARY KEY,
    session_id UUID REFERENCES sessions(id),
    sequence INT NOT NULL,
    input TEXT NOT NULL,
    output TEXT NOT NULL,
    machine TEXT NOT NULL,
    username TEXT NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL
);