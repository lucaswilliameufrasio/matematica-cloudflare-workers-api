package httpserver

import (
    "context"
    "net/http"
    "strings"

    "github.com/golang-jwt/jwt/v5"
    "github.com/rs/zerolog/log"

    "matematica-api/internal/auth"
    "matematica-api/internal/repo"
)

type ctxKey int

const userKey ctxKey = 1

type AuthedUser struct {
    UID    string
    ID     string
    Claims jwt.MapClaims
    Token  string
}

// FirebaseAuthMiddleware ensures requests carry a valid Firebase ID token.
func FirebaseAuthMiddleware(v *auth.FirebaseVerifier, rp *repo.Repo) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            authz := r.Header.Get("Authorization")
            if authz == "" { writeError(w, http.StatusUnauthorized, "MISSING_AUTHORIZATION", "missing authorization header"); return }
            parts := strings.SplitN(authz, " ", 2)
            if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") { writeError(w, http.StatusUnauthorized, "INVALID_AUTHORIZATION", "expected Bearer token"); return }
            uid, claims, err := v.Verify(r.Context(), parts[1])
            if err != nil { log.Debug().Err(err).Msg("firebase verify failed"); writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token"); return }
            // Ensure local user and attach internal ID
            var emailPtr *string
            if em, ok := claims["email"].(string); ok && em != "" { emailPtr = &em }
            userID, err := rp.EnsureUserByExternal(r.Context(), uid, emailPtr)
            if err != nil { log.Warn().Err(err).Msg("ensure user by external failed"); writeError(w, http.StatusUnauthorized, "USER_UPSERT_FAILED", "could not ensure user"); return }
            ctx := context.WithValue(r.Context(), userKey, &AuthedUser{UID: uid, ID: userID, Claims: claims, Token: parts[1]})
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// UserFromContext returns the authenticated user or nil.
func UserFromContext(ctx context.Context) *AuthedUser {
    v, _ := ctx.Value(userKey).(*AuthedUser)
    return v
}
