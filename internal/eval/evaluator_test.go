package eval

import "testing"

func TestEvaluateBasic(t *testing.T) {
    tests := []struct{ in, out string }{
        {"1+2", "3"},
        {"2*3+4", "10"},
        {"2*(3+4)", "14"},
        {"-3+5", "2"},
        {"sqrt(9)", "3"},
        {"3/2", "1.5"},
        {".5 + .25", "0.75"},
    }
    for _, tt := range tests {
        got, err := Evaluate(tt.in)
        if err != nil { t.Fatalf("%s: %v", tt.in, err) }
        if got != tt.out { t.Fatalf("%s: got %s want %s", tt.in, got, tt.out) }
    }
}

func TestEvaluateDivideByZero(t *testing.T) {
    _, err := Evaluate("1/0")
    if err == nil { t.Fatal("expected error") }
}

