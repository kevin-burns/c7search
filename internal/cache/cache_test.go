package cache

import (
	"testing"
	"time"
)

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	c := &FS{Dir: dir, Now: time.Now}

	type Payload struct{ X int }
	if err := c.Put("k", 10*time.Second, Payload{X: 42}); err != nil {
		t.Fatal(err)
	}
	var p Payload
	hit, err := c.Get("k", &p)
	if err != nil {
		t.Fatal(err)
	}
	if !hit || p.X != 42 {
		t.Fatalf("hit=%v p=%+v", hit, p)
	}
}

func TestExpired(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	c := &FS{Dir: dir, Now: func() time.Time { return now }}

	if err := c.Put("k", 1*time.Second, "v"); err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Second)
	var s string
	hit, _ := c.Get("k", &s)
	if hit {
		t.Errorf("expected miss after TTL, got hit")
	}
}

func TestKey_Stable(t *testing.T) {
	a := Key("docs", "vercel/next.js", "routing", "5000")
	b := Key("docs", "vercel/next.js", "routing", "5000")
	if a != b {
		t.Error("Key not deterministic")
	}
	c := Key("docs", "vercel/next.js", "rendering", "5000")
	if a == c {
		t.Error("different inputs produced same key")
	}
}

func TestMissReturnsNoError(t *testing.T) {
	dir := t.TempDir()
	c := &FS{Dir: dir, Now: time.Now}
	var s string
	hit, err := c.Get("nope", &s)
	if err != nil || hit {
		t.Errorf("got hit=%v err=%v; expected miss", hit, err)
	}
}

// T3: if the wall clock moves backwards (NTP correction, VM resume),
// FetchedAt can be in the future relative to Now. The naive math then
// computes a negative duration that is "less than TTL" and yields a hit.
// We treat any future FetchedAt as a miss so the cache self-heals.
func TestClockRewind_TreatedAsMiss(t *testing.T) {
	dir := t.TempDir()

	// Put with Now=T+1h.
	future := time.Now().Add(1 * time.Hour)
	cWriter := &FS{Dir: dir, Now: func() time.Time { return future }}
	if err := cWriter.Put("k", 24*time.Hour, "v"); err != nil {
		t.Fatal(err)
	}

	// Get with Now=T (i.e. wall clock has been rewound by an hour).
	cReader := &FS{Dir: dir, Now: func() time.Time { return future.Add(-1 * time.Hour) }}
	var s string
	hit, _ := cReader.Get("k", &s)
	if hit {
		t.Errorf("future-dated entry must be treated as miss after clock rewind")
	}
}
