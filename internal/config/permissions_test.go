package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCheckPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permissions don't apply on Windows")
	}

	dir := t.TempDir()

	tight := filepath.Join(dir, "tight")
	if err := os.WriteFile(tight, []byte("k"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := CheckPermissions(tight); len(got) != 0 {
		t.Errorf("0600 file produced warnings: %v", got)
	}

	loose := filepath.Join(dir, "loose")
	if err := os.WriteFile(loose, []byte("k"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := CheckPermissions(loose); len(got) != 1 {
		t.Errorf("0644 file: expected 1 warning, got %v", got)
	}

	if got := CheckPermissions(filepath.Join(dir, "missing")); got != nil {
		t.Errorf("missing file should return nil, got %v", got)
	}
}
