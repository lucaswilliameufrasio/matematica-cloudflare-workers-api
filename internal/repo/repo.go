package repo

import (
	"context"
	"errors"
	"matematica-api/internal/cache_driver"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valkey-io/valkey-go"
)

type Repo struct {
	db    *pgxpool.Pool
	rdb   *valkey.Client
	cache *cache_driver.CacheDriver
}

func New(db *pgxpool.Pool, rdb *valkey.Client, cache *cache_driver.CacheDriver) *Repo {
	return &Repo{db: db, rdb: rdb, cache: cache}
}

// SaveExpressionAttempt records an attempt for expressions mode. userID optional.
func (r *Repo) SaveExpressionAttempt(ctx context.Context, userID *string, expr, expected string, correct bool, dur time.Duration) error {
	// difficulty not used for expressions for now
	_, err := r.db.Exec(ctx, `
        INSERT INTO attempts (user_id, mode, operator, difficulty, a, b, op, expr, answer, correct, time_ms, error_type)
        VALUES ($1, 'expression', NULL, 0, NULL, NULL, NULL, $2, $3, $4, $5, NULL)
    `, userID, expr, expected, correct, int(dur.Milliseconds()))
	return err
}

// EnsureUserByExternal ensures a user exists with the given external_id and returns the internal user id.
// If email is provided and available, it is stored on initial insert.
func (r *Repo) EnsureUserByExternal(ctx context.Context, externalID string, email *string) (string, error) {
	// Try fast path
	var id string
	err := r.db.QueryRow(ctx, `SELECT id FROM users WHERE external_id=$1`, externalID).Scan(&id)
	if err == nil {
		return id, nil
	}
	// Insert or get existing (UPSERT on external_id)
	if email != nil {
		err = r.db.QueryRow(ctx, `
            INSERT INTO users (external_id, email)
            VALUES ($1, $2)
            ON CONFLICT (external_id) DO UPDATE SET external_id=EXCLUDED.external_id
            RETURNING id
        `, externalID, *email).Scan(&id)
	} else {
		err = r.db.QueryRow(ctx, `
            INSERT INTO users (external_id)
            VALUES ($1)
            ON CONFLICT (external_id) DO UPDATE SET external_id=EXCLUDED.external_id
            RETURNING id
        `, externalID).Scan(&id)
	}
	return id, err
}

type Profile struct {
	UserID      string
	DisplayName string
	AvatarKey   string
	Username    string
}

func (r *Repo) GetProfile(ctx context.Context, userID string) (*Profile, error) {
	var p Profile
	err := r.db.QueryRow(ctx, `SELECT user_id, COALESCE(display_name,''), COALESCE(avatar,''), COALESCE(username,'') FROM profiles WHERE user_id=$1`, userID).
		Scan(&p.UserID, &p.DisplayName, &p.AvatarKey, &p.Username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

// UpsertProfile updates display name and avatar key, creating the row if needed.
func (r *Repo) UpsertProfile(ctx context.Context, userID, displayName, avatarKey string) error {
	_, err := r.db.Exec(ctx, `
        INSERT INTO profiles (user_id, display_name, avatar)
        VALUES ($1, $2, $3)
        ON CONFLICT (user_id) DO UPDATE SET display_name=EXCLUDED.display_name, avatar=EXCLUDED.avatar
    `, userID, displayName, avatarKey)
	return err
}

// CreateProfileFirstRegistration inserts a new profile with username; conflicts on username or user_id return pg error 23505.
func (r *Repo) CreateProfileFirstRegistration(ctx context.Context, userID, displayName, avatarKey, username string) error {
	_, err := r.db.Exec(ctx, `
        INSERT INTO profiles (user_id, display_name, avatar, username)
        VALUES ($1, $2, $3, $4)
    `, userID, displayName, avatarKey, username)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			switch pgErr.ConstraintName {
			case "ux_profiles_username":
				return ErrUsernameTaken
			case "profiles_pkey":
				return ErrProfileExists
			default:
				return ErrProfileExists
			}
		}
	}
	return err
}

var ErrNotFound = errors.New("not found")
var ErrUsernameTaken = errors.New("username_taken")
var ErrProfileExists = errors.New("profile_exists")

// UsernameExists returns true if a profile with the given username exists.
func (r *Repo) UsernameExists(ctx context.Context, username string) (bool, error) {
	var one int
	err := r.db.QueryRow(ctx, `SELECT 1 FROM profiles WHERE username=$1 LIMIT 1`, username).Scan(&one)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
