package exprgen

import (
    "math/rand"
    "testing"
)

func TestGenerateDeterministicWithSeed(t *testing.T) {
    rand.Seed(42)
    cfg := GenCfg{MaxDepth: 3, Mode: Mixed, MinBinaryOps: 1, AllowUnaryAtStart: true}
    e1 := GenerateExpression(cfg)
    rand.Seed(42)
    e2 := GenerateExpression(cfg)
    if e1 != e2 {
        t.Fatalf("expressions differ for same seed: %q vs %q", e1, e2)
    }
}

