package main

import (
	"github.com/utvecklat/slogformat"
	"log/slog"
	"os"
)

type basket struct {
	contentType string
	itemCount   int
}

func (b basket) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("contentType", b.contentType),
		slog.Int("itemCount", b.itemCount),
	)
}

func main() {
	logger := slog.New(slogformat.New(os.Stdout, &slogformat.HandlerOptions{
		AddSource:         true,
		AddSourceFullPath: false,
		Level:             slog.LevelDebug,
		BoldMessage:       true,
		ColorSeverity:     true,
		Date:              true,
	}))

	logger.Debug("connecting to database", "host", "localhost", "port", 5432)
	logger.Info("new user created", "username", "acme")
	logger.Warn("database connection closed", "host", "localhost", "database", "customers")
	logger.Error("unexpected error", "host", "localhost", "database", "customers")

	logger.Debug("test", "key", "val", "key2", "val2")

	logger = logger.With("fruit", "apple", "drink", "coffee")

	logger.Info("test", "key", "val", "key2", "val2")
	logger.Warn("test", "key", "val", "key2", "val2")

	logger = logger.WithGroup("group1")
	logger.Error("test", "key", "val", "key2", "val2")
	logger.Info("basket", "first", basket{contentType: "fruit", itemCount: 32})
}
