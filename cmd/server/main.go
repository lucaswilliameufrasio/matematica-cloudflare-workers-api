package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/valkey-io/valkey-go"

	"matematica-api/internal/auth"
	"matematica-api/internal/cache_driver"
	"matematica-api/internal/config"
	"matematica-api/internal/httpserver"
	"matematica-api/internal/repo"
	"matematica-api/internal/spec"
)

var healthCheckResponse = []byte("{\"message\": \"ok\"}")

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	// DB
	ctx := context.Background()
	dbpool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create pgx pool")
	}
	defer dbpool.Close()
	if err := dbpool.Ping(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to ping database")
	}

	// Migrations: run separately via goose CLI or docker-compose.

	// Redis / Valkey
	client, err := valkey.NewClient(valkey.ClientOption{InitAddress: []string{cfg.RedisAddr}})

	if err := client.Do(ctx, client.B().Ping().Build()).Error(); err != nil {
		log.Fatal().Err(err).Msg("failed to connect to redis")
	}
	defer client.Close()

	// In-memory cache driver
	cache, err := cache_driver.NewCacheDriver(1_000_000, 100_000, 64)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to init cache driver")
	}

	// Repository
	rp := repo.New(dbpool, &client, cache)

	// Router
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(httpserver.LoggingMiddleware())
	r.Use(httpserver.PrometheusMiddleware())
	r.Use(httpserver.RateLimitMiddleware(client, time.Minute, 120)) // 120 req/min per identity

	// Health
	r.Get("/health-check", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(healthCheckResponse)
	})

	// Metrics
	r.Handle("/metrics", promhttp.Handler())

	// Docs
	if os.Getenv("APP_ENV") != "prd" {
		r.Get("/docs", httpserver.RedocHandler("/api/openapi.yaml"))
		r.Get("/api/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/yaml")
			w.WriteHeader(200)
			_, _ = w.Write(spec.OpenAPI)
		})
	}

	// Auth
	var fbVerifier *auth.FirebaseVerifier
	if cfg.FirebaseProjectID != "" {
		fbVerifier = auth.NewFirebaseVerifier(cfg.FirebaseProjectID, cfg.FirebaseJWKSURL, cache)
	}

	// Expressions endpoints (public)
	httpserver.MountExpressions(r, rp)

	// Example protected route
	if fbVerifier != nil {
		r.Group(func(r chi.Router) {
			r.Use(httpserver.FirebaseAuthMiddleware(fbVerifier, rp))
			r.Get("/v1/auth/me", func(w http.ResponseWriter, r *http.Request) {
				u := httpserver.UserFromContext(r.Context())
				if u == nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(401)
					_, _ = w.Write([]byte(`{"error_code":"UNAUTHORIZED","message":"missing or invalid token"}`))
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(fmt.Sprintf(`{"id":"%s","uid":"%s"}`, u.ID, u.UID)))
			})
			// Profile endpoints
			httpserver.MountProfile(r, rp, cfg.AvatarBaseURL)
		})
	}

	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: r}

	go func() {
		log.Info().Str("addr", cfg.HTTPAddr).Msg("server listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("listen and serve failed")
		}
	}()

	// graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctxShut, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctxShut)
	log.Info().Msg("server stopped")
}
