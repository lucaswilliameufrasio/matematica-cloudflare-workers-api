package cache_driver

import (
    "sync"
    "testing"
    "time"
)

func TestCacheExpiry(t *testing.T) {
    d, err := NewCacheDriver(1e4, 1e4, 64)
    if err != nil { t.Fatal(err) }
    d.Set("k", "v", 50*time.Millisecond)
    if v, ok := d.Get("k"); !ok || v != "v" { t.Fatalf("miss before expiry") }
    time.Sleep(60 * time.Millisecond)
    if _, ok := d.Get("k"); ok { t.Fatalf("should be expired") }
}

func TestSingleflight(t *testing.T) {
    d, err := NewCacheDriver(1e4, 1e4, 64)
    if err != nil { t.Fatal(err) }
    var n int
    fetch := func() (string, time.Duration, error) {
        n++
        time.Sleep(10 * time.Millisecond)
        return "value", 0, nil
    }
    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() { defer wg.Done(); _, _ = d.FetchOrGet("key", fetch) }()
    }
    wg.Wait()
    if n != 1 { t.Fatalf("expected single fetch, got %d", n) }
}

