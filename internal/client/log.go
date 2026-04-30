package client

import (
	"context"
	"log/slog"
)

// discardHandler is a slog.Handler that drops every record. Used as the
// zero-value logger so callers don't have to nil-check.
type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (h discardHandler) WithAttrs([]slog.Attr) slog.Handler      { return h }
func (h discardHandler) WithGroup(string) slog.Handler           { return h }
