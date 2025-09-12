-- +goose Up
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS username TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS ux_profiles_username ON profiles (username);

-- +goose Down
DROP INDEX IF EXISTS ux_profiles_username;
ALTER TABLE profiles DROP COLUMN IF EXISTS username;

