package exprgen

import (
    "math/rand"
    "strings"
    "time"
)

type NumMode int

const (
    Mixed NumMode = iota
    IntegersOnly
    FloatsOnly
)

type GenCfg struct {
    MaxDepth         int
    Mode             NumMode
    MinBinaryOps     int  // minimum + / - at the top level
    AllowUnaryAtStart bool // allow leading unary minus at expression start
}

func init() {
    rand.Seed(time.Now().UnixNano())
}

// GenerateExpression returns a single randomized expression string.
func GenerateExpression(cfg GenCfg) string {
    return strings.TrimSpace(genExpression(cfg, 0, true))
}

// ---- Internal helpers (unchanged behavior) ----

func genExpression(c GenCfg, d int, allowUnaryStart bool) string {
    return genExprWithMinOps(c, d, c.MinBinaryOps, allowUnaryStart)
}

func genExprWithMinOps(c GenCfg, d, need int, allowUnaryStart bool) string {
    if need > 0 || (d < c.MaxDepth && !bernoulli(0.55)) {
        left := genExprWithMinOps(c, d+1, max(0, need-1), allowUnaryStart)
        return left + " " + addOp() + " " + genTerm(c, d+1, false)
    }
    return genTerm(c, d+1, allowUnaryStart)
}

func genTerm(c GenCfg, d int, allowUnaryStart bool) string {
    if d >= c.MaxDepth || bernoulli(0.6) {
        return genFactor(c, d+1, allowUnaryStart, false, false)
    }
    if bernoulli(0.5) {
        return genTerm(c, d+1, allowUnaryStart) + " * " + genFactor(c, d+1, false, false, false)
    }
    return genTerm(c, d+1, allowUnaryStart) + " / " + genFactor(c, d+1, false, true, true)
}

func genFactor(c GenCfg, d int, allowUnaryStart, requireNumeric, requireNonZero bool) string {
    if d >= c.MaxDepth {
        return genNumeric(c, requireNonZero)
    }
    if requireNumeric {
        return genNumeric(c, requireNonZero)
    }
    choices := []int{0, 1, 2}
    if allowUnaryStart {
        choices = append(choices, 3)
    }
    switch choices[rand.Intn(len(choices))] {
    case 0:
        return genNumeric(c, false)
    case 1:
        return "(" + genExpression(c, d+1, true) + ")"
    case 2:
        return "sqrt(" + genExpression(c, d+1, true) + ")"
    default:
        return "-" + genNumeric(c, false)
    }
}

func genNumeric(c GenCfg, nonZero bool) string {
    switch c.Mode {
    case IntegersOnly:
        if nonZero {
            return genIntNonZero()
        }
        return genInt()
    case FloatsOnly:
        if nonZero {
            return genFloatNonZero()
        }
        return genFloat()
    default:
        if bernoulli(0.65) {
            if nonZero {
                return genIntNonZero()
            }
            return genInt()
        }
        if nonZero {
            return genFloatNonZero()
        }
        return genFloat()
    }
}

func genInt() string {
    nDigits := 1 + rand.Intn(3)
    if nDigits == 1 {
        return string('0' + rune(rand.Intn(10)))
    }
    first := '1' + rune(rand.Intn(9))
    var b strings.Builder
    b.WriteRune(first)
    for i := 1; i < nDigits; i++ {
        b.WriteByte(byte('0' + rand.Intn(10)))
    }
    return b.String()
}

func genIntNonZero() string {
    s := genInt()
    for s == "0" {
        s = genInt()
    }
    return s
}

func genFloat() string {
    if bernoulli(0.7) {
        return genIntNonZeroOrZero() + "." + digits(1+rand.Intn(3))
    }
    return "." + digits(1+rand.Intn(3))
}

func genFloatNonZero() string {
    for {
        s := genFloat()
        if s == "0.0" || strings.TrimLeft(s, "0.") == "" {
            continue
        }
        allZeros := true
        for _, r := range s {
            if r != '0' && r != '.' {
                allZeros = false
                break
            }
        }
        if !allZeros {
            return s
        }
    }
}

func genIntNonZeroOrZero() string {
    if bernoulli(0.25) {
        return "0"
    }
    return genInt()
}

func digits(n int) string {
    var b strings.Builder
    for i := 0; i < n; i++ {
        b.WriteByte(byte('0' + rand.Intn(10)))
    }
    return b.String()
}

func addOp() string { if bernoulli(0.5) { return "+" }; return "-" }

func bernoulli(p float64) bool { return rand.Float64() < p }

func max(a, b int) int {
    if a > b { return a }
    return b
}

