package logging

import (
	"io"
	"log/slog"
)

func New(w io.Writer, debug bool) *slog.Logger {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: level,
	}))
}
