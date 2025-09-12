package httpserver

import (
    "encoding/json"
    "net/http"
)

type errorBody struct {
    ErrorCode string `json:"error_code"`
    Message   string `json:"message,omitempty"`
}

func writeJSON(w http.ResponseWriter, v any) {
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(errorBody{ErrorCode: code, Message: msg})
}

