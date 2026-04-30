// Package cache provides a small filesystem cache for search results and
// docs payloads. Cached entries carry a TTL; corrupt or expired files are
// treated as misses (lazy invalidation).
//
// The cache directory is created with mode 0700 and individual files with
// 0600. Even though no secrets live here today, defense-in-depth costs
// nothing and is consistent with the rest of the binary's posture.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Envelope wraps every cached value so we can evolve the schema without
// blowing up old caches — a v-mismatch is treated as a miss.
type Envelope struct {
	V         int             `json:"v"`
	FetchedAt time.Time       `json:"fetched_at"`
	TTL       int             `json:"ttl_seconds"`
	Body      json.RawMessage `json:"body"`
}

const schemaVersion = 1

// FS is a filesystem-backed cache rooted at Dir.
type FS struct {
	Dir string
	Now func() time.Time
}

// New returns an FS rooted at the user cache dir's c7search/ subdir.
// Falls back to os.TempDir() if UserCacheDir fails.
func New() (*FS, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	dir := filepath.Join(base, "c7search")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}
	return &FS{Dir: dir, Now: time.Now}, nil
}

// Key builds a deterministic, FS-safe cache key. Components are joined with
// '|' (cannot appear in URL components) and hashed so disparate inputs
// can't collide.
func Key(namespace string, parts ...string) string {
	h := sha256.New()
	h.Write([]byte(namespace))
	for _, p := range parts {
		h.Write([]byte("|"))
		h.Write([]byte(p))
	}
	return namespace + "_" + hex.EncodeToString(h.Sum(nil))[:32]
}

// Get reads a cached value into out (json.Unmarshal target). Returns
// (true, nil) on a fresh hit, (false, nil) on miss/stale/corrupt, and
// (false, err) only on unexpected I/O errors.
func (c *FS) Get(key string, out any) (bool, error) {
	if c == nil {
		return false, nil
	}
	now := c.now()

	path := filepath.Join(c.Dir, key+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil || env.V != schemaVersion {
		return false, nil // treat corrupt/old as miss
	}
	age := now.Sub(env.FetchedAt)
	// Negative age = FetchedAt is in the future (NTP correction, VM resume,
	// disk-restored from another machine). Treat as a miss; the cache will
	// self-heal on the next write.
	if age < 0 || age > time.Duration(env.TTL)*time.Second {
		return false, nil
	}
	if err := json.Unmarshal(env.Body, out); err != nil {
		return false, nil
	}
	return true, nil
}

// Put stores body under key with the given TTL. Errors are returned but the
// caller is expected to log + continue — a cache failure must never break a
// command.
func (c *FS) Put(key string, ttl time.Duration, body any) error {
	if c == nil {
		return nil
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	env := Envelope{
		V:         schemaVersion,
		FetchedAt: c.now(),
		TTL:       int(ttl.Seconds()),
		Body:      raw,
	}
	out, err := json.Marshal(env)
	if err != nil {
		return err
	}
	path := filepath.Join(c.Dir, key+".json")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	if _, err := f.Write(out); err != nil {
		return err
	}
	return nil
}

// Clear removes every entry under Dir. Returns the count removed.
func (c *FS) Clear() (int, error) {
	if c == nil {
		return 0, nil
	}
	entries, err := os.ReadDir(c.Dir)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		if err := os.Remove(filepath.Join(c.Dir, e.Name())); err == nil {
			n++
		}
	}
	return n, nil
}

func (c *FS) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}
