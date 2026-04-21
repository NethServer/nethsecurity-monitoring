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

const RETENTION = 2 * time.Hour

func main() {
	// CLI setup
	var addr string
	flag.StringVar(&addr, "addr", ":8081", "address to listen on")

	var dbPath string
	flag.StringVar(&dbPath, "db-path", ":memory:", "path to the SQLite database file")

	var exportPath string
	flag.StringVar(&exportPath, "export-path", "./exports", "path to write hourly json exports")

	var debugLevel string
	flag.StringVar(&debugLevel, "log-level", "info", "Log level (debug, info, warn, error)")

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

	store, err := stats.NewStore(context.Background(), dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize SQLite schema: %v", err)
	}
	defer store.Close() //nolint:errcheck

	resolver := reverse_dns.New(func(ctx context.Context, ip string) ([]string, error) {
		addrs, err := net.DefaultResolver.LookupAddr(ctx, ip)
		if err != nil {
			return nil, err
		}
		return addrs, nil
	}, 5*time.Minute, 10000)

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

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		prune := func() {
			cutoff := time.Now().Add(-RETENTION).Unix()
			if err := store.DeleteOlderThan(ctx, cutoff); err != nil {
				slog.Error("Failed to delete expired stats", "error", err)
				return
			}
			slog.Debug("Pruned expired stats", "cutoff", time.Unix(cutoff, 0).Format(time.RFC3339))
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

	// Host resolver
	wg.Add(1)
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		resolve := func() {
			duration := time.Now()
			ips, err := store.ListUnresolvedIPs(ctx)
			if err != nil {
				slog.Error("Failed to list unresolved stats IPs", "error", err)
				return
			}

			for _, ip := range ips {
				name := resolver.Lookup(ctx, ip)
				if err := store.ResolveIP(ctx, ip, name); err != nil {
					slog.Error("Failed to resolve stats IP", "ip", ip, "error", err)
				}
			}
			resolverStats := resolver.Stats()
			slog.Debug(
				"Resolved hostnames for stats",
				"count",
				len(ips),
				"duration",
				time.Since(duration).String(),
				"cache_size",
				resolverStats.Size,
				"miss_rate",
				resolverStats.MissRate,
			)
		}

		resolve()

		for {
			select {
			case <-ticker.C:
				resolve()
			case <-ctx.Done():
				slog.Info("Stopping stats host resolver")
				return
			}
		}
	}()

	// Exporter
	wg.Add(1)
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		export := func() {
			duration := time.Now()
			endHour := time.Now().Truncate(time.Hour).Unix()
			startHour := endHour - int64((48 * time.Hour).Seconds())
			targets, err := store.ListExportTargets(ctx, startHour, endHour)
			if err != nil {
				slog.Error("Failed to list export targets", "error", err)
				return
			}

			for _, target := range targets {
				summary, err := store.BuildSummary(ctx, target.HourBucket, target.LocalIP)
				if err != nil {
					slog.Error(
						"Failed to build hourly summary",
						"hour_bucket",
						target.HourBucket,
						"local_ip",
						target.LocalIP,
						"error",
						err,
					)
					continue
				}
				if err := stats.WriteHourSummary(
					exportPath,
					time.Unix(target.HourBucket, 0),
					target.LocalIP,
					summary,
				); err != nil {
					slog.Error(
						"Failed to write hourly summary",
						"hour_bucket",
						target.HourBucket,
						"local_ip",
						target.LocalIP,
						"error",
						err,
					)
				}
			}
			slog.Debug(
				"Exported hourly summary",
				"startHour", time.Unix(startHour, 0).Format(time.RFC3339),
				"endHour", time.Unix(endHour, 0).Format(time.RFC3339),
				"duration", time.Since(duration).String(),
			)
		}

		export()

		for {
			select {
			case <-ticker.C:
				export()
			case <-ctx.Done():
				slog.Info("Stopping stats exporter")
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
