-- +goose Up
ALTER TABLE users ADD COLUMN IF NOT EXISTS external_id TEXT UNIQUE;
ALTER TABLE users ALTER COLUMN email DROP NOT NULL;
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS external_id;
-- Not reverting nullability to avoid data issues

