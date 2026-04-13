package logger

import (
	"log/slog"
	"strings"
)

type FiberWriter struct{}

func (w *FiberWriter) Write(p []byte) (int, error) {
	line := strings.TrimSpace(string(p))
	if line != "" {
		slog.Debug(line)
	}

	return len(p), nil
}
