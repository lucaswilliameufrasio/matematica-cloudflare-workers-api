package config

import (
    "fmt"
    "os"

    "github.com/kelseyhightower/envconfig"
)

type Config struct {
    HTTPAddr    string `envconfig:"HTTP_ADDR" default:":8080"`
    DatabaseURL string `envconfig:"DATABASE_URL" required:"true"`
    RedisAddr   string `envconfig:"REDIS_ADDR" default:"localhost:6379"`
    JWTSecret   string `envconfig:"JWT_SECRET" default:"dev-secret-change"`
    AutoMigrate bool   `envconfig:"AUTO_MIGRATE" default:"true"`
    FirebaseProjectID string `envconfig:"FIREBASE_PROJECT_ID"`
    FirebaseJWKSURL   string `envconfig:"FIREBASE_JWKS_URL" default:"https://www.googleapis.com/service_accounts/v1/jwk/securetoken@system.gserviceaccount.com"`
    AvatarBaseURL     string `envconfig:"AVATAR_BASE_URL" default:""`
}

func Load() (*Config, error) {
    var c Config
    if err := envconfig.Process("APP", &c); err != nil {
        return nil, err
    }
    // Fallbacks without APP_ prefix for common envs
    if v := os.Getenv("HTTP_ADDR"); v != "" { c.HTTPAddr = v }
    if v := os.Getenv("DATABASE_URL"); v != "" { c.DatabaseURL = v }
    if v := os.Getenv("REDIS_ADDR"); v != "" { c.RedisAddr = v }
    if v := os.Getenv("JWT_SECRET"); v != "" { c.JWTSecret = v }
    if v := os.Getenv("AUTO_MIGRATE"); v != "" { if v == "0" || v == "false" { c.AutoMigrate = false } else { c.AutoMigrate = true } }
    if v := os.Getenv("FIREBASE_PROJECT_ID"); v != "" { c.FirebaseProjectID = v }
    if v := os.Getenv("FIREBASE_JWKS_URL"); v != "" { c.FirebaseJWKSURL = v }
    if v := os.Getenv("AVATAR_BASE_URL"); v != "" { c.AvatarBaseURL = v }

    if c.DatabaseURL == "" {
        return nil, fmt.Errorf("DATABASE_URL not set")
    }
    return &c, nil
}
