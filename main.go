package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"log"
	"log/slog"
	"maps"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/nethserver/nethsecurity-monitoring/flows"
)

func main() {
	var socketPath string
	flag.StringVar(
		&socketPath,
		"socket",
		"/var/run/netifyd/flows.sock",
		"Path to the socket to listen on",
	)

	var debugLevel string
	flag.StringVar(&debugLevel, "log-level", "info", "Log level")

	var outFile string
	flag.StringVar(
		&outFile,
		"outfile",
		"/var/run/netifyd/flows.json",
		"Path to the output file for flows",
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
		log.Fatalf("Failed to connect to socket: %v", err)
	}
	defer conn.Close() //nolint:errcheck

	processor := flows.NewFlowProcessor()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup

	slog.Info("Starting flow processing")

	NewTask(ctx, &wg, "save", 10*time.Second, func() {
		events := processor.GetEvents()
		currentFlows := make([]any, 0, len(events))

		for e := range maps.Values(events) {
			currentFlows = append(currentFlows, e)
		}
		data, err := json.Marshal(currentFlows)
		if err != nil {
			slog.Error("Failed to marshal flows", "error", err)
			return
		}
		err = os.WriteFile(outFile, data, 0o644)
		if err != nil {
			slog.Error("Failed to save flows", "error", err)
		}
	}).Run()

	NewTask(ctx, &wg, "prune", 10*time.Second, func() {
		processor.PurgeFlowsOlderThan(expiredPersistence)
	}).Run()

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
	wg.Wait()
	slog.Info("All processes completed, exiting")
}
