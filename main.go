package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nethserver/nethsecurity-monitoring/api"
	"github.com/nethserver/nethsecurity-monitoring/flows"
)

func main() {
	var socketPath string
	flag.StringVar(
		&socketPath,
		"socket",
		"/var/run/netifyd/flows.sock",
		"Path to the netifyd Unix socket to read flow events from",
	)

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

	logger := slog.New(&BasicLogger{out: os.Stderr, level: logLevel})
	slog.SetDefault(logger)

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		log.Fatalf("Failed to connect to netifyd socket: %v", err)
	}
	defer conn.Close() //nolint:errcheck

	processor := flows.NewFlowProcessor()

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	api.NewFlowApi(processor).Setup(app)

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
			// app.Shutdown() causes Listen to return a non-nil error; ignore it.
			slog.Debug("API server stopped", "error", err)
		}
	}()

	NewTask(ctx, &wg, "prune", 10*time.Second, func() {
		processor.PurgeFlowsOlderThan(expiredPersistence)
	}).Run()

	slog.Info("Starting flow processing")

	wg.Add(1)
	go func() {
		defer wg.Done()
		decoder := json.NewDecoder(conn)
		for {
			select {
			case <-ctx.Done():
				slog.Info("Stopping flow processing")
				return
			default:
				var event flows.FlowEvent
				if err := decoder.Decode(&event); err != nil {
					if ctx.Err() != nil {
						slog.Info("Stopping flow processing")
						return
					}
					if err == io.EOF {
						log.Fatalf("Socket closed: %v", err)
					}
					log.Fatalf("Failed to decode flow event: %v", err)
				}
				processor.Process(event)
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
