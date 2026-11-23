// Package output renders search results and docs to text/markdown/json.
//
// Discipline: status messages always go to stderr; only the requested
// payload goes to stdout. ANSI styling is only applied when stdout is a
// real terminal AND NO_COLOR is unset — otherwise raw output flows through
// pipes cleanly (`c7search docs ... | glow -`, `| jq`, `> file.md`).
package output

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/kevin-burns/c7search/internal/client"
)

// Format identifies which renderer to use.
type Format int

const (
	FormatText Format = iota
	FormatMarkdown
	FormatJSON
)

// ParseFormat maps a CLI flag value to a Format. Unknown → FormatText.
func ParseFormat(s string) Format {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "json":
		return FormatJSON
	case "md", "markdown":
		return FormatMarkdown
	default:
		return FormatText
	}
}

// IsTTY reports whether the given file is connected to a terminal.
func IsTTY(f *os.File) bool {
	if f == nil {
		return false
	}
	// File descriptors are POSIX small ints (kernel-managed, < RLIMIT_NOFILE
	// which itself fits comfortably in int32). The uintptr -> int cast is
	// safe in practice; gosec G115 flags it categorically.
	return term.IsTerminal(int(f.Fd())) //nolint:gosec
}

// useColor decides whether to emit ANSI; respects the NO_COLOR convention.
func useColor(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if f, ok := w.(*os.File); ok {
		return IsTTY(f)
	}
	return false
}

// RenderLibraries writes a list of search hits to w in the requested format.
func RenderLibraries(w io.Writer, libs []client.Library, f Format, limit int) error {
	if limit > 0 && len(libs) > limit {
		libs = libs[:limit]
	}
	switch f {
	case FormatJSON:
		return writeJSON(w, libs)
	default:
		return renderLibrariesText(w, libs, useColor(w))
	}
}

func renderLibrariesText(w io.Writer, libs []client.Library, color bool) error {
	if len(libs) == 0 {
		_, err := fmt.Fprintln(w, "no results")
		return err
	}
	for _, l := range libs {
		id := l.ID
		if color {
			id = "\x1b[1;36m" + id + "\x1b[0m"
		}
		// -1 is the API's "no stars data" sentinel (websites entries, etc.);
		// 0 is a legitimate value worth displaying.
		stars := ""
		if l.Stars >= 0 {
			stars = fmt.Sprintf("  ★%d", l.Stars)
		}
		if _, err := fmt.Fprintf(w, "%s  trust=%.1f  snippets=%d%s\n", id, l.TrustScore, l.TotalSnippets, stars); err != nil {
			return err
		}
		desc := strings.TrimSpace(l.Description)
		if desc != "" {
			if len(desc) > 200 {
				desc = desc[:197] + "..."
			}
			if _, err := fmt.Fprintf(w, "    %s\n", desc); err != nil {
				return err
			}
		}
	}
	return nil
}

// RenderDoc writes a docs payload to w in the requested format.
func RenderDoc(w io.Writer, doc client.Doc, f Format) error {
	switch f {
	case FormatJSON:
		// Prefer the structured snippets if we have them; otherwise wrap text.
		if len(doc.Snippets) > 0 {
			return writeJSON(w, map[string]any{
				"libraryId": doc.LibraryID,
				"format":    "json",
				"snippets":  doc.Snippets,
			})
		}
		return writeJSON(w, map[string]any{
			"libraryId": doc.LibraryID,
			"format":    doc.Format,
			"body":      doc.Body,
		})
	default:
		// Text and markdown share the wire body — the API returns markdown-ish
		// text either way. We just stream it through.
		_, err := fmt.Fprint(w, doc.Body)
		return err
	}
}

// writeJSON writes v as indented JSON plus a trailing newline, the shape
// json.Encoder produced before the move to encoding/json/v2.
//
// Two v1 behaviours are kept on purpose and one is not:
//   - Deterministic: v2 does not sort map keys by default; v1 did, and
//     RenderDoc encodes a map, so without it the key order would vary run
//     to run.
//   - v2 marshals a nil slice as [] instead of null (e.g. a library with no
//     versions). Accepted: both decode to "no items" and [] is what most
//     JSON consumers expect.
//   - v2 does not escape <, > and & as \u003c etc. Accepted: the output is
//     not embedded in HTML, and code snippets become readable.
func writeJSON(w io.Writer, v any) error {
	b, err := json.Marshal(v, jsontext.WithIndent("  "), json.Deterministic(true))
	if err != nil {
		return err
	}
	_, err = w.Write(append(b, '\n'))
	return err
}
