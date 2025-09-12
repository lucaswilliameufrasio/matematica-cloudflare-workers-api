package httpserver

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"github.com/valkey-io/valkey-go"
	"github.com/valkey-io/valkey-go/valkeycompat"

	"matematica-api/internal/rate"
)

var (
	reqDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "api",
		Subsystem: "http",
		Name:      "request_duration_seconds",
		Help:      "HTTP request duration in seconds",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method", "path", "status"})
)

func init() { prometheus.MustRegister(reqDuration) }

func LoggingMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			next.ServeHTTP(ww, r)
			dur := time.Since(start)
			log.Info().Str("method", r.Method).Str("path", r.URL.Path).Int("status", ww.Status()).Dur("dur", dur).Msg("request")
		})
	}
}

func PrometheusMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			next.ServeHTTP(ww, r)
			dur := time.Since(start).Seconds()
			path := r.URL.Path
			// collapse IDs
			path = sanitizePath(path)
			reqDuration.WithLabelValues(r.Method, path, http.StatusText(ww.Status())).Observe(dur)
		})
	}
}

func RateLimitMiddleware(rdb valkey.Client, window time.Duration, limit int) func(next http.Handler) http.Handler {
	valkeyCompatClient := valkeycompat.NewAdapter(rdb)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.Header.Get("X-Forwarded-For")
			if ip == "" {
				ip = strings.Split(r.RemoteAddr, ":")[0]
			}
			key := ip
			allowed, remaining, reset, err := rate.AllowSlidingWindow(valkeyCompatClient, key, window, limit)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "RATE_LIMIT_ERROR", "rate limiter failure")
				return
			}
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.Itoa(reset))
			if !allowed {
				writeError(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "too many requests")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func sanitizePath(p string) string {
	parts := strings.Split(p, "/")
	for i, s := range parts {
		if s == "" {
			continue
		}
		if len(s) > 24 {
			parts[i] = ":id"
		}
	}
	return strings.Join(parts, "/")
}

// no-op helpers
