package httpserver

import (
    "encoding/json"
    "net/http"
    "regexp"
    "strings"

    "github.com/go-chi/chi/v5"
    "github.com/rs/zerolog/log"

    "matematica-api/internal/repo"
)

var usernameRe = regexp.MustCompile(`^[a-z0-9_]{3,20}$`)

type profileResp struct {
    DisplayName string `json:"display_name"`
    AvatarURL   string `json:"avatar_url"`
    Username    string `json:"username"`
}

type profilePutReq struct {
    DisplayName string `json:"display_name"`
    AvatarKey   string `json:"avatar_key"`
}

type profilePostReq struct {
    DisplayName string `json:"display_name"`
    AvatarKey   string `json:"avatar_key"`
    Username    string `json:"username"`
}

func MountProfile(r chi.Router, rp *repo.Repo, avatarBase string) {
    // Username availability (expects auth; adjust to public if desired)
    r.Get("/v1/profile/username-availability", func(w http.ResponseWriter, r *http.Request) {
        uname := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("username")))
        if !usernameRe.MatchString(uname) {
            writeError(w, http.StatusBadRequest, "INVALID_USERNAME", "username must be 3-20 chars [a-z0-9_]")
            return
        }
        exists, err := rp.UsernameExists(r.Context(), uname)
        if err != nil { log.Error().Err(err).Msg("username exists check failed"); writeError(w, http.StatusInternalServerError, "USERNAME_CHECK_FAILED", "failed to check username"); return }
        writeJSON(w, map[string]any{"username": uname, "available": !exists})
    })
    r.Get("/v1/profile", func(w http.ResponseWriter, r *http.Request) {
        u := UserFromContext(r.Context())
        if u == nil { writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid token"); return }
        p, err := rp.GetProfile(r.Context(), u.ID)
        display := ""
        key := ""
        username := ""
        if err == nil && p != nil {
            display = p.DisplayName
            key = p.AvatarKey
            username = p.Username
        }
        // if not found, return empty fields
        url := buildAvatarURL(avatarBase, key)
        writeJSON(w, profileResp{DisplayName: display, AvatarURL: url, Username: username})
    })

    r.Put("/v1/profile", func(w http.ResponseWriter, r *http.Request) {
        u := UserFromContext(r.Context())
        if u == nil { writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid token"); return }
        var req profilePutReq
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json body"); return }
        req.DisplayName = strings.TrimSpace(req.DisplayName)
        req.AvatarKey = strings.TrimSpace(req.AvatarKey)
        if len(req.DisplayName) > 80 { writeError(w, http.StatusBadRequest, "INVALID_DISPLAY_NAME", "display_name too long"); return }
        if err := rp.UpsertProfile(r.Context(), u.ID, req.DisplayName, req.AvatarKey); err != nil {
            log.Error().Err(err).Msg("upsert profile failed")
            writeError(w, http.StatusInternalServerError, "PROFILE_UPDATE_FAILED", "failed to update profile")
            return
        }
        writeJSON(w, profileResp{DisplayName: req.DisplayName, AvatarURL: buildAvatarURL(avatarBase, req.AvatarKey)})
    })

    r.Post("/v1/profile", func(w http.ResponseWriter, r *http.Request) {
        u := UserFromContext(r.Context())
        if u == nil { writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid token"); return }
        var req profilePostReq
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json body"); return }
        req.DisplayName = strings.TrimSpace(req.DisplayName)
        req.AvatarKey = strings.TrimSpace(req.AvatarKey)
        req.Username = strings.ToLower(strings.TrimSpace(req.Username))
        if !usernameRe.MatchString(req.Username) { writeError(w, http.StatusBadRequest, "INVALID_USERNAME", "username must be 3-20 chars [a-z0-9_]"); return }
        if len(req.DisplayName) > 80 { writeError(w, http.StatusBadRequest, "INVALID_DISPLAY_NAME", "display_name too long"); return }
        // Try create
        if err := rp.CreateProfileFirstRegistration(r.Context(), u.ID, req.DisplayName, req.AvatarKey, req.Username); err != nil {
            if err == repo.ErrUsernameTaken {
                writeError(w, http.StatusConflict, "USERNAME_ALREADY_TAKEN", "username already taken")
                return
            }
            if err == repo.ErrProfileExists {
                writeError(w, http.StatusConflict, "PROFILE_ALREADY_EXISTS", "profile already exists")
                return
            }
            log.Error().Err(err).Msg("create profile failed")
            writeError(w, http.StatusInternalServerError, "PROFILE_CREATE_FAILED", "failed to create profile")
            return
        }
        writeJSON(w, profileResp{DisplayName: req.DisplayName, AvatarURL: buildAvatarURL(avatarBase, req.AvatarKey), Username: req.Username})
    })
}

func buildAvatarURL(base, key string) string {
    if base == "" || key == "" { return "" }
    base = strings.TrimRight(base, "/")
    key = strings.TrimLeft(key, "/")
    return base + "/" + key
}
