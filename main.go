package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
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

// reconnectTimeout is the maximum time spent retrying a lost netifyd connection.
const reconnectTimeout = 5 * time.Second

// retryInterval is the pause between successive reconnection attempts.
const retryInterval = 100 * time.Millisecond

// dialWithRetry tries to open a Unix socket connection to socketPath.  It keeps
// retrying every retryInterval until either the connection succeeds, the context
// is cancelled, or the reconnectTimeout deadline is exceeded.  All retry
// attempts are logged at debug level so that brief netifyd reloads don't
// produce noise in the default log level.
func dialWithRetry(ctx context.Context, socketPath string) (net.Conn, error) {
	deadline := time.Now().Add(reconnectTimeout)
	for {
		conn, err := net.Dial("unix", socketPath)
		if err == nil {
			return conn, nil
		}
		slog.Debug(
			"Failed to connect to netifyd socket, retrying",
			"error",
			err,
			"retry_in",
			retryInterval,
		)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(retryInterval):
			if time.Now().After(deadline) {
				return nil, fmt.Errorf(
					"could not reconnect to netifyd socket within %s: %w",
					reconnectTimeout, err,
				)
			}
		}
	}
}

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

	conn, err := dialWithRetry(context.Background(), socketPath)
	if err != nil {
		log.Fatalf("Failed to connect to netifyd socket: %v", err)
	}

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
		defer conn.Close() //nolint:errcheck

		decoder := json.NewDecoder(conn)
		for {
			// Check for shutdown before every decode attempt.
			select {
			case <-ctx.Done():
				slog.Info("Stopping flow processing")
				return
			default:
			}

			var event flows.FlowEvent
			if err := decoder.Decode(&event); err != nil {
				if ctx.Err() != nil {
					slog.Info("Stopping flow processing")
					return
				}

				if err != io.EOF {
					slog.Debug("Failed to decode flow event, continuing", "error", err)
					continue
				}

				slog.Debug("Netifyd socket disconnected, attempting to reconnect",
					"timeout", reconnectTimeout)
				conn.Close() //nolint:errcheck

				newConn, dialErr := dialWithRetry(ctx, socketPath)
				if dialErr != nil {
					log.Fatalf("Netifyd socket unavailable: %v", dialErr)
				}

				slog.Debug("Reconnected to netifyd socket")
				conn = newConn
				decoder = json.NewDecoder(conn)
				continue
			}

			processor.Process(event)
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
