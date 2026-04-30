package cli

import (
	"context"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/kevin-burns/c7search/internal/redact"
)

// newDebugLogger builds an slog.Logger that writes to the command's stderr
// when --debug is set, and discards everything otherwise. Every string
// attribute and the record message are passed through redact helpers so a
// caller cannot accidentally log a bearer token or a URL with embedded
// credentials.
func newDebugLogger(cmd *cobra.Command) *slog.Logger {
	if !debugFlag(cmd) {
		return nil // client.New treats nil as discard
	}
	base := slog.NewTextHandler(cmd.ErrOrStderr(), &slog.HandlerOptions{Level: slog.LevelDebug})
	return slog.New(redactHandler{inner: base})
}

// redactHandler scrubs bearer tokens and URL credentials from log messages
// and string attributes before delegating to its inner handler. Defense in
// depth: callers SHOULD already redact, but accidents happen.
type redactHandler struct {
	inner slog.Handler
}

func (h redactHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	return h.inner.Enabled(ctx, lvl)
}

func (h redactHandler) Handle(ctx context.Context, r slog.Record) error {
	r.Message = scrub(r.Message)
	clone := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	r.Attrs(func(a slog.Attr) bool {
		clone.AddAttrs(scrubAttr(a))
		return true
	})
	return h.inner.Handle(ctx, clone)
}

func (h redactHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clean := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		clean[i] = scrubAttr(a)
	}
	return redactHandler{inner: h.inner.WithAttrs(clean)}
}

func (h redactHandler) WithGroup(name string) slog.Handler {
	return redactHandler{inner: h.inner.WithGroup(name)}
}

func scrub(s string) string { return redact.RedactBearer(redact.RedactURL(s)) }

func scrubAttr(a slog.Attr) slog.Attr {
	if a.Value.Kind() == slog.KindString {
		return slog.String(a.Key, scrub(a.Value.String()))
	}
	return a
}
