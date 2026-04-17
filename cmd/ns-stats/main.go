package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"net"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	airRecover "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/nethserver/nethsecurity-monitoring/api"
	"github.com/nethserver/nethsecurity-monitoring/internal/logger"
	"github.com/nethserver/nethsecurity-monitoring/reverse_dns"
	"github.com/nethserver/nethsecurity-monitoring/stats"
)

func main() {
	// CLI setup
	var addr string
	flag.StringVar(&addr, "addr", ":8081", "address to listen on")

	var dbPath string
	flag.StringVar(&dbPath, "db-path", ":memory:", "path to the SQLite database file")

	var debugLevel string
	flag.StringVar(&debugLevel, "log-level", "info", "Log level (debug, info, warn, error)")

	var retention time.Duration
	flag.DurationVar(&retention, "retention", 24*time.Hour, "delete stats older than this duration")

	flag.Parse()

	// slog setup
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

	cache := reverse_dns.NewResolver(
		func(ctx context.Context, ip string) ([]string, error) {
			return net.DefaultResolver.LookupAddr(ctx, ip)
		},
		10*time.Minute,
		10000,
	)
	store, err := stats.NewStore(context.Background(), dbPath, cache)
	if err != nil {
		log.Fatalf("Failed to initialize SQLite schema: %v", err)
	}
	defer store.Close() //nolint:errcheck

	// Concurrent managers
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup

	// API Server
	server := fiber.New(fiber.Config{
		AppName:               "ns-stats",
		DisableStartupMessage: true,
	})
	if logLevel == slog.LevelDebug {
		server.Use(fiberlogger.New(fiberlogger.Config{
			Format:     "${method} ${path} ${status} ${latency}\n",
			TimeFormat: "15:04:05",
			Output:     &logger.FiberWriter{},
		}))
	}
	server.Use(airRecover.New())
	api.NewStatsApi(store).Setup(server)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.Listen(addr); err != nil {
			slog.Debug("API server stopped", "error", err)
		}
	}()

	// Pruner
	wg.Add(1)
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		prune := func() {
			cutoff := time.Now().Add(-retention).Unix()
			slog.Debug("Pruning expired stats", "cutoff", cutoff)
			if err := store.DeleteOlderThan(ctx, cutoff); err != nil {
				slog.Error("Failed to delete expired stats", "error", err)
				return
			}
			slog.Debug("Pruned expired stats", "cutoff", cutoff)
		}

		prune()

		for {
			select {
			case <-ticker.C:
				prune()
			case <-ctx.Done():
				slog.Info("Stopping stats cleanup")
				return
			}
		}
	}()

	<-ctx.Done()

	slog.Info("Shutting down API server")
	if err := server.Shutdown(); err != nil {
		slog.Error("API server shutdown error", "error", err)
	}

	wg.Wait()
	slog.Info("All processes completed, exiting")
}
