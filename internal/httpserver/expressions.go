package httpserver

import (
    "encoding/json"
    "net/http"
    "strings"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/rs/zerolog/log"

    "matematica-api/internal/eval"
    "matematica-api/internal/exprgen"
    "matematica-api/internal/repo"
)

type exprNextReq struct {
    Mode            string `json:"mode"`
    MaxDepth        int    `json:"max_depth"`
    MinBinaryOps    int    `json:"min_binary_ops"`
    AllowUnaryStart bool   `json:"allow_unary_start"`
}

type exprNextResp struct {
    Expr   string           `json:"expr"`
    Result string           `json:"result"`
    Cfg    exprNextRespCfg  `json:"cfg"`
}

type exprNextRespCfg struct {
    Mode            string `json:"mode"`
    MaxDepth        int    `json:"max_depth"`
    MinBinaryOps    int    `json:"min_binary_ops"`
    AllowUnaryStart bool   `json:"allow_unary_start"`
}

type exprAnswerReq struct {
    Expr   string `json:"expr"`
    Answer string `json:"answer"`
}

type exprAnswerResp struct {
    Correct bool   `json:"correct"`
    Expected string `json:"expected"`
    TimeMS  int64  `json:"time_ms"`
}

func MountExpressions(r chi.Router, rp *repo.Repo) {
    r.Route("/v1/expressions", func(r chi.Router) {
        r.Post("/next", func(w http.ResponseWriter, r *http.Request) {
            var req exprNextReq
            if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                writeError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json body")
                return
            }
            mode := strings.ToLower(req.Mode)
            var nm exprgen.NumMode
            switch mode {
            case "int", "ints", "integer", "integers":
                nm = exprgen.IntegersOnly
                mode = "int"
            case "float", "floats":
                nm = exprgen.FloatsOnly
                mode = "float"
            default:
                nm = exprgen.Mixed
                mode = "mixed"
            }
            if req.MaxDepth <= 0 { req.MaxDepth = 4 }
            cfg := exprgen.GenCfg{MaxDepth: req.MaxDepth, Mode: nm, MinBinaryOps: req.MinBinaryOps, AllowUnaryAtStart: req.AllowUnaryStart}
            expr := exprgen.GenerateExpression(cfg)
            res, err := eval.Evaluate(expr)
            if err != nil { writeError(w, http.StatusBadRequest, "EVALUATION_ERROR", "failed to evaluate expression"); return }
            writeJSON(w, exprNextResp{Expr: expr, Result: res, Cfg: exprNextRespCfg{Mode: mode, MaxDepth: req.MaxDepth, MinBinaryOps: req.MinBinaryOps, AllowUnaryStart: req.AllowUnaryStart}})
        })
        r.Post("/answer", func(w http.ResponseWriter, r *http.Request) {
            var req exprAnswerReq
            if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                writeError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json body")
                return
            }
            start := time.Now()
            expected, err := eval.Evaluate(req.Expr)
            if err != nil { writeError(w, http.StatusBadRequest, "INVALID_EXPRESSION", "expression cannot be evaluated"); return }
            correct := normalize(req.Answer) == normalize(expected)
            // Write attempt (anon allowed)
            if err := rp.SaveExpressionAttempt(r.Context(), nil, req.Expr, expected, correct, time.Since(start)); err != nil {
                log.Warn().Err(err).Msg("failed to save attempt")
            }
            writeJSON(w, exprAnswerResp{Correct: correct, Expected: expected, TimeMS: time.Since(start).Milliseconds()})
        })
    })
}

func normalize(s string) string {
    s = strings.TrimSpace(s)
    s = strings.TrimLeft(s, "+")
    if s == "-0" { s = "0" }
    // trim trailing zeros if decimal
    if strings.Contains(s, ".") {
        s = strings.TrimRight(s, "0")
        s = strings.TrimRight(s, ".")
    }
    return s
}
