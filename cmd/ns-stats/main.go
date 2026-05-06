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

	var exportPath string
	flag.StringVar(&exportPath, "export-path", "", "path to export hourly stats (required)")

	var debugLevel string
	flag.StringVar(&debugLevel, "log-level", "info", "Log level (debug, info, warn, error)")

	flag.Parse()

	// Validate required flags
	if exportPath == "" {
		log.Fatalf("--export-path is required")
	}

	// Fixed retention and export window
	retention := 3 * time.Hour
	exportWindowHours := 2

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

	store, err := stats.NewStore(context.Background(), dbPath)
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
		slog.Info("Starting API server")
		if err := server.Listen(addr); err != nil {
			slog.Error("Failed to start API server", "error", err)
			stop()
		}
	}()

	// Pruner
	wg.Add(1)
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		prune := func() {
			cutoff := time.Now().Add(-retention).Unix()
			if err := store.DeleteOlderThan(ctx, cutoff); err != nil {
				slog.Error("Failed to delete expired stats", "error", err)
				return
			}
			slog.Debug("Pruned expired stats", "cutoff", time.Unix(cutoff, 0).Format(time.RFC3339))
		}

		slog.Info("Starting stats cleanup process")
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

	// Exporter
	exporter := stats.NewExporter(exportPath, exportWindowHours)
	wg.Add(1)
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		export := func() {
			timestamp := time.Now()
			err := exporter.ExportAll(ctx, store)
			if err != nil {
				slog.Error("Failed to export stats", "error", err)
				return
			}
			slog.Debug("Exported stats successfully", "duration", time.Since(timestamp))
		}

		slog.Info("Starting exporter process")
		export()

		for {
			select {
			case <-ticker.C:
				export()
			case <-ctx.Done():
				slog.Info("Stopping exporter process")
				return
			}
		}
	}()

	// IP Resolver (DNS reverse lookup with caching)
	dnsResolver := reverse_dns.New(net.DefaultResolver.LookupAddr, 5*time.Minute, 10000)
	wg.Add(1)
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		resolve := func() {
			ips, err := store.QueryUnresolvedIPs(ctx)
			if err != nil {
				slog.Error("Failed to query unresolved IPs", "error", err)
				return
			}

			slog.Debug("Resolving IPs", "count", len(ips))
			for _, ip := range ips {
				select {
				case <-ctx.Done():
					return
				default:
				}

				hostname := dnsResolver.Lookup(ctx, ip)
				if err := store.SaveResolvedHost(ctx, ip, hostname); err != nil {
					slog.Error("Failed to save resolved host", "ip", ip, "error", err)
				}
			}

			dnsStats := dnsResolver.Stats()
			slog.Debug(
				"IP resolver stats",
				"cache_size",
				dnsStats.Size,
				"cache_hits",
				dnsStats.Hits,
				"cache_misses",
				dnsStats.Misses,
				"cache_miss_rate",
				dnsStats.MissRate,
			)
		}

		slog.Info("Starting IP resolver")
		resolve()

		for {
			select {
			case <-ticker.C:
				resolve()
			case <-ctx.Done():
				slog.Info("Stopping IP resolver")
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
