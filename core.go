package verbs

import (
	"log"
	"log/slog"
)

var (
	logger = slog.New(slog.NewJSONHandler(log.Writer(), &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
)
