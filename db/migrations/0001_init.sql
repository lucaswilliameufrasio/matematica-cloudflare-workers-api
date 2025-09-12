-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;
-- Create enums
DO $$ BEGIN
    CREATE TYPE operator_enum AS ENUM ('add','sub','mul','div');
EXCEPTION WHEN duplicate_object THEN null; END $$;

DO $$ BEGIN
    CREATE TYPE mode_enum AS ENUM ('operator','expression');
EXCEPTION WHEN duplicate_object THEN null; END $$;

DO $$ BEGIN
    CREATE TYPE match_mode_enum AS ENUM ('duel','practice');
EXCEPTION WHEN duplicate_object THEN null; END $$;

DO $$ BEGIN
    CREATE TYPE match_state_enum AS ENUM ('waiting','active','finished','cancelled');
EXCEPTION WHEN duplicate_object THEN null; END $$;

-- Users
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Profiles
CREATE TABLE IF NOT EXISTS profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    display_name TEXT,
    avatar TEXT,
    country TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Sessions
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    jwt_id TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);

-- Ratings
CREATE TABLE IF NOT EXISTS ratings (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    operator operator_enum NOT NULL,
    rating INT NOT NULL DEFAULT 1200,
    rd DOUBLE PRECISION NOT NULL DEFAULT 350,
    last_updated TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, operator)
);

-- Attempts
CREATE TABLE IF NOT EXISTS attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NULL REFERENCES users(id) ON DELETE CASCADE,
    mode mode_enum NOT NULL,
    operator operator_enum NULL,
    difficulty INT NOT NULL DEFAULT 0,
    a INT NULL,
    b INT NULL,
    op CHAR(1) NULL,
    expr TEXT NULL,
    answer TEXT NOT NULL,
    correct BOOLEAN NOT NULL,
    time_ms INT NOT NULL,
    error_type TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_attempts_user_id ON attempts(user_id);
CREATE INDEX IF NOT EXISTS idx_attempts_created_at ON attempts(created_at);

-- Matches
CREATE TABLE IF NOT EXISTS matches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mode match_mode_enum NOT NULL,
    state match_state_enum NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at TIMESTAMPTZ NULL,
    finished_at TIMESTAMPTZ NULL
);

CREATE TABLE IF NOT EXISTS match_participants (
    match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    team INT NOT NULL DEFAULT 0,
    score INT NOT NULL DEFAULT 0,
    mmr_snapshot JSONB,
    PRIMARY KEY (match_id, user_id)
);

CREATE TABLE IF NOT EXISTS match_rounds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    round_index INT NOT NULL,
    problem JSONB NOT NULL,
    started_at TIMESTAMPTZ NULL,
    ended_at TIMESTAMPTZ NULL
);

-- Leaderboards
CREATE TABLE IF NOT EXISTS seasons (
    id SERIAL PRIMARY KEY,
    label TEXT NOT NULL,
    starts_at DATE NOT NULL,
    ends_at DATE NOT NULL
);

CREATE TABLE IF NOT EXISTS leaderboards (
    id SERIAL PRIMARY KEY,
    season_id INT REFERENCES seasons(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    global_rating INT NOT NULL,
    add_rating INT NOT NULL,
    sub_rating INT NOT NULL,
    mul_rating INT NOT NULL,
    div_rating INT NOT NULL,
    week_of DATE NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_leaderboards_season_week ON leaderboards(season_id, week_of);

-- Drills
CREATE TABLE IF NOT EXISTS drills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    spec JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Purchases
CREATE TABLE IF NOT EXISTS purchases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sku TEXT NOT NULL,
    provider TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- AI Hints
CREATE TABLE IF NOT EXISTS ai_hints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    attempt_id UUID NOT NULL REFERENCES attempts(id) ON DELETE CASCADE,
    tokens_used INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS ai_hints;
DROP TABLE IF EXISTS purchases;
DROP TABLE IF EXISTS drills;
DROP TABLE IF EXISTS leaderboards;
DROP TABLE IF EXISTS seasons;
DROP TABLE IF EXISTS match_rounds;
DROP TABLE IF EXISTS match_participants;
DROP TABLE IF EXISTS matches;
DROP TABLE IF EXISTS attempts;
DROP TABLE IF EXISTS ratings;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS profiles;
DROP TABLE IF EXISTS users;
DROP TYPE IF EXISTS match_state_enum;
DROP TYPE IF EXISTS match_mode_enum;
DROP TYPE IF EXISTS mode_enum;
DROP TYPE IF EXISTS operator_enum;
