package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
)

type BasicLogger struct {
	out   io.Writer
	level slog.Level
}

func (h *BasicLogger) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level
}

func (h *BasicLogger) Handle(_ context.Context, r slog.Record) error {
	// Format: LEVEL Message
	if _, err := fmt.Fprintf(h.out, "%s %s", r.Level, r.Message); err != nil {
		return err
	}
	// Append attributes as key=value
	r.Attrs(func(a slog.Attr) bool {
		fmt.Fprintf(h.out, " %s=%v", a.Key, a.Value) //nolint:errcheck
		return true
	})
	fmt.Fprintln(h.out) //nolint:errcheck
	return nil
}

func (h *BasicLogger) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *BasicLogger) WithGroup(_ string) slog.Handler      { return h }
