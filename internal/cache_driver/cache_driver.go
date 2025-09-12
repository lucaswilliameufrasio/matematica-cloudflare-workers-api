package cache_driver

import (
	"sync/atomic"
	"time"

	"github.com/dgraph-io/ristretto"
	"golang.org/x/sync/singleflight"

	"matematica-api/internal/set"
)

// CacheItem wraps a cached string plus an expiration timestamp.
type CacheItem struct {
	value     string
	expiresAt int64 // UnixNano; 0 means “never expires”
}

// isExpired reports whether now > expiresAt
func (ci *CacheItem) isExpired() bool {
	if atomic.LoadInt64(&ci.expiresAt) == 0 {
		return false
	}
	return time.Now().UnixNano() > atomic.LoadInt64(&ci.expiresAt)
}

// CacheDriver is your reusable cache + singleflight bundle.
type CacheDriver struct {
	cache *ristretto.Cache
	grp   singleflight.Group
	keys  set.Set[string]
}

// NewCacheDriver builds a new driver with reasonable defaults.
// numCounters controls tracking precision, maxCost is total weight,
// bufferItems is the write buffer. Tweak to taste.
func NewCacheDriver(numCounters int64, maxCost int64, bufferItems int64) (*CacheDriver, error) {
	cfg := &ristretto.Config{
		NumCounters: numCounters,
		MaxCost:     maxCost,
		BufferItems: bufferItems,
	}
	c, err := ristretto.NewCache(cfg)
	if err != nil {
		return nil, err
	}
	return &CacheDriver{cache: c, keys: set.New[string](), grp: singleflight.Group{}}, nil
}

// Get returns the cached value if present and unexpired.
// ok==false means “not found or expired”.
func (d *CacheDriver) Get(key string) (value string, ok bool) {
	result, err, _ := d.grp.Do(key, func() (interface{}, error) {
		if v, found := d.cache.Get(key); found {
			if item, _ := v.(CacheItem); item.isExpired() == false {
				return item.value, nil
			}
			// expired: remove and treat as miss
			d.cache.Del(key)
		}

		// not found or expired
		return "", nil
	})

	return result.(string), err == nil
}

// FetchOrGet runs fn() exactly once if key is missing/expired,
// otherwise returns the existing value.
// fn must return the value and an optional TTL (0 == no expiry).
func (d *CacheDriver) FetchOrGet(key string, fn func() (string, time.Duration, error)) (string, error) {
	// singleflight to dedupe concurrent fetches
	result, err, _ := d.grp.Do(key, func() (interface{}, error) {
		// double-check inside flight
		if v, ok := d.Get(key); ok {
			return v, nil
		}
		// actually fetch
		val, ttl, err := fn()
		if err != nil {
			return "", err
		}
		d.Set(key, val, ttl)
		return val, nil
	})
	if err != nil {
		return "", err
	}
	return result.(string), nil
}

// Set writes a value with optional TTL (0 == no expiry).
func (d *CacheDriver) Set(key, value string, ttl time.Duration) {
	var expiresAt int64
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl).UnixNano()
	}
	item := CacheItem{value: value, expiresAt: expiresAt}
	d.cache.SetWithTTL(key, item, 1, ttl)
	d.cache.Wait() // ensure it's settled before return
}

// Delete removes a single key.
func (d *CacheDriver) Delete(key string) {
	d.cache.Del(key)
	d.cache.Wait()
}

// DeletePrefix removes all keys starting with prefix.
// WARNING: this scans the entire cache under the hood.
func (d *CacheDriver) DeletePrefix(prefix string) {
	found := set.KeysWithPrefix(d.keys, prefix)
	for _, key := range found {
		d.Delete(key)
	}
	d.cache.Wait()
}
