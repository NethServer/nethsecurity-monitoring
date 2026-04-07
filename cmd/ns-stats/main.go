package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	airRecover "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/nethserver/nethsecurity-monitoring/api"
	"github.com/nethserver/nethsecurity-monitoring/stats"
	_ "modernc.org/sqlite"
)

func main() {
	var addr string
	flag.StringVar(&addr, "addr", ":8081", "address to listen on")

	var dbPath string
	flag.StringVar(&dbPath, "db-path", ":memory:", "path to the SQLite database file")

	var debugLevel string
	flag.StringVar(&debugLevel, "log-level", "info", "Log level (debug, info, warn, error)")

	flag.Parse()

	var logLevel slog.Level
	switch debugLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		log.Fatalf("Invalid log level: %s", debugLevel)
	}

	slog.SetLogLoggerLevel(logLevel)

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("Failed to open SQLite database: %v", err)
	}
	defer db.Close() //nolint:errcheck

	store := stats.NewStore(db)
	if err := store.Init(context.Background()); err != nil {
		log.Fatalf("Failed to initialize SQLite schema: %v", err)
	}

	server := fiber.New(fiber.Config{})
	server.Use(airRecover.New())
	server.Use(logger.New())
	api.NewStatsApi(stats.NewReceiver(store)).Setup(server)
	if err := server.Listen(addr); err != nil {
		slog.Error("API server stopped", "error", err)
	}
}
