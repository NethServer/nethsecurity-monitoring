package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
)

type Handler struct {
	out   io.Writer
	level slog.Level
}

func New(out io.Writer, level slog.Level) slog.Handler {
	return &Handler{out: out, level: level}
}

func (h *Handler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level
}

func (h *Handler) Handle(_ context.Context, r slog.Record) error {
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

func (h *Handler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *Handler) WithGroup(_ string) slog.Handler      { return h }
