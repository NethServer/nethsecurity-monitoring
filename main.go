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
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"syscall"
	"time"

	"github.com/nethserver/nethsecurity-monitoring/flows"
)

type FlowsResponse struct {
	Total int   `json:"total"`
	Flows []any `json:"flows"`
}

// getDigestFromEvent extracts the digest from a FlowEvent
func getDigestFromEvent(event any) string {
	if e, ok := event.(flows.FlowEvent); ok {
		if fb, ok := e.Flow.(flows.FlowBase); ok {
			return fb.Digest
		}
	}
	return ""
}

// sortFlowsByDigest sorts flows deterministically by their digest
func sortFlowsByDigest(flows []any) {
	sort.Slice(flows, func(i, j int) bool {
		return getDigestFromEvent(flows[i]) < getDigestFromEvent(flows[j])
	})
}

func handleFlows(processor *flows.FlowProcessor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		events := processor.GetEvents()
		currentFlows := make([]any, 0, len(events))

		for e := range maps.Values(events) {
			currentFlows = append(currentFlows, e)
		}

		// Sort flows deterministically by digest for consistent pagination
		sortFlowsByDigest(currentFlows)

		total := len(currentFlows)

		// Parse pagination parameters
		start := 0
		end := total

		if startParam := r.URL.Query().Get("start"); startParam != "" {
			if s, err := strconv.Atoi(startParam); err == nil && s >= 0 {
				start = s
			}
		}

		if endParam := r.URL.Query().Get("end"); endParam != "" {
			if e, err := strconv.Atoi(endParam); err == nil && e >= 0 {
				end = e
			}
		}

		// Apply bounds checking
		if start > total {
			start = total
		}
		if end > total {
			end = total
		}
		if start > end {
			start = end
		}

		// Slice the flows
		paginatedFlows := currentFlows[start:end]

		response := FlowsResponse{
			Total: total,
			Flows: paginatedFlows,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			slog.Error("Failed to encode flows response", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}

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

	var httpAddr string
	flag.StringVar(
		&httpAddr,
		"http-addr",
		"127.0.0.1:19000",
		"HTTP server address for API",
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

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/flows", handleFlows(processor))

	server := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("HTTP server shutdown error", "error", err)
		} else {
			slog.Info("HTTP server stopped")
		}
	}()

	slog.Info("Starting flow processing")
	slog.Info("Starting HTTP server", "address", httpAddr)

	// Start HTTP server
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "error", err)
		}
	}()

	NewTask(ctx, "save", 10*time.Second, func() {
		events := processor.GetEvents()
		currentFlows := make([]any, 0, len(events))

		for e := range maps.Values(events) {
			currentFlows = append(currentFlows, e)
		}

		// Sort flows deterministically by digest for consistency
		sortFlowsByDigest(currentFlows)

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

	NewTask(ctx, "prune", 10*time.Second, func() {
		processor.PurgeFlowsOlderThan(1 * time.Minute)
	}).Run()

	go func() {
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
}
