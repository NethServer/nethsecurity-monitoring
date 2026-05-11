package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nethserver/nethsecurity-monitoring/api"
	"github.com/nethserver/nethsecurity-monitoring/flows"
	"github.com/nethserver/nethsecurity-monitoring/internal/logger"
)

func main() {
	var debugLevel string
	flag.StringVar(&debugLevel, "log-level", "info", "Log level (debug, info, warn, error)")

	var apiPort string
	flag.StringVar(
		&apiPort,
		"api-port",
		"8080",
		"TCP port the HTTP API server listens on (bound to 127.0.0.1)",
	)

	var expiredPersistence time.Duration
	flag.DurationVar(
		&expiredPersistence,
		"expired-persistence",
		60*time.Second,
		"Purge expired flows older than this duration",
	)

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

	loggerHandler := logger.New(os.Stderr, logLevel)
	slog.SetDefault(slog.New(loggerHandler))

	processor := flows.NewFlowProcessor()

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	api.NewFlowApi(processor, processor).Setup(app)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup

	// Start the HTTP API server on 127.0.0.1 only.
	wg.Add(1)
	go func() {
		defer wg.Done()
		addr := "127.0.0.1:" + apiPort
		slog.Info("API server listening", "addr", addr)
		if err := app.Listen(addr); err != nil {
			slog.Error("Failed to start API server", "error", err)
			stop()
		}
	}()

	// Flow cleanup (purge flows older than expiredPersistence)
	wg.Add(1)
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		slog.Info("Starting flow cleanup process")
		for {
			select {
			case <-ticker.C:
				processor.PurgeFlowsOlderThan(expiredPersistence)
			case <-ctx.Done():
				slog.Info("Stopping flow cleanup")
				return
			}
		}
	}()

	<-ctx.Done()
	stop()

	slog.Info("Shutting down API server")
	if err := app.Shutdown(); err != nil {
		slog.Error("API server shutdown error", "error", err)
	}

	wg.Wait()
	slog.Info("All processes completed, exiting")
}
