// Package config (in v0.1) provides only the filesystem-permission warning
// helper. A TOML config-file loader is planned for v1.1; this package is the
// natural home for it when the time comes.
package config

import (
	"fmt"
	"os"
	"runtime"
)

// CheckPermissions inspects path's mode and returns a slice of human-readable
// warnings if the file is readable by group or other (perm & 0o077 != 0).
// Returns nil on Windows (POSIX bits don't apply) and on stat errors —
// callers handle missing files separately.
//
// Warnings are intended for stderr at startup; they are never fatal. A
// too-permissive config file should nudge the user, not break their flow.
func CheckPermissions(path string) []string {
	if runtime.GOOS == "windows" {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	perm := info.Mode().Perm()
	if perm&0o077 != 0 {
		return []string{fmt.Sprintf(
			"config file %s has permissions %o (should be 0600) — your API key may be exposed to other local users",
			path, perm,
		)}
	}
	return nil
}
