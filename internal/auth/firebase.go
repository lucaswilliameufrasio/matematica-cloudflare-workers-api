package auth

import (
    "context"
    "crypto/rsa"
    "encoding/base64"
    "encoding/json"
    "errors"
    "io"
    "math/big"
    "net/http"
    "strings"
    "time"

    "github.com/golang-jwt/jwt/v5"

    "matematica-api/internal/cache_driver"
)

var (
    ErrNoAuthHeader   = errors.New("authorization header missing")
    ErrInvalidBearer  = errors.New("invalid authorization header format")
    ErrInvalidToken   = errors.New("invalid token")
    ErrKeyNotFound    = errors.New("signing key not found")
    ErrProjectMismatch = errors.New("project mismatch")
)

type FirebaseVerifier struct {
    ProjectID string
    JWKSURL   string
    Cache     *cache_driver.CacheDriver
    Client    *http.Client
}

func NewFirebaseVerifier(projectID, jwksURL string, cache *cache_driver.CacheDriver) *FirebaseVerifier {
    return &FirebaseVerifier{
        ProjectID: projectID,
        JWKSURL:   jwksURL,
        Cache:     cache,
        Client:    &http.Client{Timeout: 5 * time.Second},
    }
}

// Verify extracts and validates a Firebase ID token returning uid and claims.
func (v *FirebaseVerifier) Verify(ctx context.Context, tokenString string) (string, jwt.MapClaims, error) {
    if strings.TrimSpace(tokenString) == "" {
        return "", nil, ErrInvalidToken
    }
    var claims jwt.MapClaims
    keyFunc := func(t *jwt.Token) (interface{}, error) {
        if t.Method.Alg() != jwt.SigningMethodRS256.Alg() {
            return nil, ErrInvalidToken
        }
        kid, _ := t.Header["kid"].(string)
        if kid == "" {
            return nil, ErrInvalidToken
        }
        // get jwk for kid (cache JWKS for 1h)
        pub, err := v.getKey(ctx, kid)
        if err != nil {
            return nil, err
        }
        return pub, nil
    }
    parser := jwt.NewParser(jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer("https://securetoken.google.com/"+v.ProjectID), jwt.WithAudience(v.ProjectID))
    tok, err := parser.ParseWithClaims(tokenString, &claims, keyFunc)
    if err != nil || !tok.Valid {
        return "", nil, ErrInvalidToken
    }
    // subject must be non-empty
    sub, _ := claims["sub"].(string)
    if sub == "" {
        return "", nil, ErrInvalidToken
    }
    return sub, claims, nil
}

// getKey fetches JWKS and returns the rsa.PublicKey for the given kid.
func (v *FirebaseVerifier) getKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
    // Cache the entire JWKS blob under a single key
    cacheKey := "firebase:jwks"
    blob, err := v.Cache.FetchOrGet(cacheKey, func() (string, time.Duration, error) {
        req, _ := http.NewRequestWithContext(ctx, http.MethodGet, v.JWKSURL, nil)
        resp, err := v.Client.Do(req)
        if err != nil {
            return "", 0, err
        }
        defer resp.Body.Close()
        if resp.StatusCode != 200 {
            return "", 0, errors.New("jwks fetch failed")
        }
        data, err := io.ReadAll(resp.Body)
        if err != nil { return "", 0, err }
        // Determine TTL from Cache-Control or Expires
        ttl := parseTTL(resp)
        if ttl <= 0 { ttl = time.Hour }
        return string(data), ttl, nil
    })
    if err != nil {
        return nil, err
    }
    // Parse JWKS
    var set jwks
    if err := json.Unmarshal([]byte(blob), &set); err != nil {
        return nil, err
    }
    for _, k := range set.Keys {
        if k.Kid == kid {
            return jwkToRSA(k)
        }
    }
    return nil, ErrKeyNotFound
}

// JWKS structures
type jwks struct { Keys []jwk `json:"keys"` }
type jwk struct {
    Kty string `json:"kty"`
    Kid string `json:"kid"`
    N   string `json:"n"`
    E   string `json:"e"`
    Alg string `json:"alg"`
    Use string `json:"use"`
}

func jwkToRSA(j jwk) (*rsa.PublicKey, error) {
    if j.Kty != "RSA" { return nil, ErrInvalidToken }
    nb, err := base64.RawURLEncoding.DecodeString(j.N)
    if err != nil { return nil, err }
    eb, err := base64.RawURLEncoding.DecodeString(j.E)
    if err != nil { return nil, err }
    n := new(big.Int).SetBytes(nb)
    var e int
    for _, b := range eb { e = e<<8 + int(b) }
    return &rsa.PublicKey{N: n, E: e}, nil
}

// parseTTL reads Cache-Control max-age or Expires header and returns a TTL.
func parseTTL(resp *http.Response) time.Duration {
    cc := resp.Header.Get("Cache-Control")
    if cc != "" {
        parts := strings.Split(cc, ",")
        for _, p := range parts {
            p = strings.TrimSpace(strings.ToLower(p))
            if strings.HasPrefix(p, "max-age=") {
                v := strings.TrimPrefix(p, "max-age=")
                if secs, err := time.ParseDuration(v+"s"); err == nil {
                    return secs
                }
            }
        }
    }
    if exp := resp.Header.Get("Expires"); exp != "" {
        if t, err := http.ParseTime(exp); err == nil {
            if t.After(time.Now()) {
                return time.Until(t)
            }
        }
    }
    return 0
}
