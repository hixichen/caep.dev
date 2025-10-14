package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/internal/controller"
	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/internal/handlers"
	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/internal/pubsub"
	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/internal/registry"
	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/pkg/api"
)

func main() {
	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting SSF Hub with Mock Pub/Sub",
		"version", "1.0.0",
		"mode", "mock")

	// Create mock Pub/Sub client
	ctx := context.Background()
	mockProjectID := "mock-local-project"

	pubsubClient, err := pubsub.NewMockClient(ctx, mockProjectID)
	if err != nil {
		logger.Error("Failed to create mock Pub/Sub client", "error", err)
		os.Exit(1)
	}
	pubsubClient.SetLogger(logger)

	// Create receiver registry
	receiverRegistry := registry.NewMemoryRegistry()

	// Create broker/controller
	broker := controller.New(pubsubClient, receiverRegistry, logger)

	// Start the broker
	if err := broker.Start(ctx); err != nil {
		logger.Error("Failed to start broker", "error", err)
		os.Exit(1)
	}

	// Create HTTP handlers
	config := &api.Config{
		ServerPort: getEnvOrDefault("SERVER_PORT", "8080"),
		LogLevel:   getEnvOrDefault("LOG_LEVEL", "debug"),
		JWTSecret:  getEnvOrDefault("JWT_SECRET", "mock-jwt-secret-change-in-production"),
	}

	httpHandlers := handlers.New(broker, receiverRegistry, logger)

	// Setup HTTP routes
	mux := http.NewServeMux()

	// SSF endpoints
	mux.HandleFunc("POST /events", httpHandlers.ReceiveEvent)
	mux.HandleFunc("GET /.well-known/ssf_configuration", httpHandlers.GetSSFConfiguration)

	// Management API
	mux.HandleFunc("GET /api/v1/receivers", httpHandlers.ListReceivers)
	mux.HandleFunc("POST /api/v1/receivers", httpHandlers.RegisterReceiver)
	mux.HandleFunc("PUT /api/v1/receivers/{id}", httpHandlers.UpdateReceiver)
	mux.HandleFunc("DELETE /api/v1/receivers/{id}", httpHandlers.UnregisterReceiver)
	mux.HandleFunc("GET /api/v1/receivers/{id}", httpHandlers.GetReceiver)
	mux.HandleFunc("GET /api/v1/stats", httpHandlers.GetStats)

	// Health and debug endpoints
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /ready", readyHandler)
	mux.HandleFunc("GET /debug/mock/stats", mockStatsHandler(pubsubClient))
	mux.HandleFunc("POST /debug/mock/clear", mockClearHandler(pubsubClient))

	// Start HTTP server
	server := &http.Server{
		Addr:         ":" + config.ServerPort,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Starting HTTP server",
			"port", config.ServerPort,
			"endpoints", []string{
				"POST /events",
				"GET /.well-known/ssf_configuration",
				"GET /api/v1/receivers",
				"POST /api/v1/receivers",
				"GET /health",
				"GET /debug/mock/stats",
			})

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Print helpful startup info
	fmt.Printf("\n🚀 SSF Hub Mock Server Started Successfully!\n\n")
	fmt.Printf("📡 Server: http://localhost:%s\n", config.ServerPort)
	fmt.Printf("🔧 Management API: http://localhost:%s/api/v1/receivers\n", config.ServerPort)
	fmt.Printf("💡 Health Check: http://localhost:%s/health\n", config.ServerPort)
	fmt.Printf("🐛 Mock Stats: http://localhost:%s/debug/mock/stats\n", config.ServerPort)
	fmt.Printf("📋 SSF Config: http://localhost:%s/.well-known/ssf_configuration\n", config.ServerPort)
	fmt.Printf("\n💻 No GCP service account needed - everything runs in memory!\n")
	fmt.Printf("📖 Check local_development.md for testing examples\n\n")

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down SSF Hub Mock Server...")

	// Gracefully shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}

	// Stop broker
	if err := broker.Stop(shutdownCtx); err != nil {
		logger.Error("Failed to stop broker gracefully", "error", err)
	}

	// Close Pub/Sub client
	if err := pubsubClient.Close(); err != nil {
		logger.Error("Failed to close Pub/Sub client", "error", err)
	}

	logger.Info("SSF Hub Mock Server stopped")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"mode":      "mock",
		"message":   "SSF Hub Mock is running in memory",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func readyHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status": "ready",
		"mode":   "mock",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func mockStatsHandler(client *pubsub.MockClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats := client.GetMockStats()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	}
}

func mockClearHandler(client *pubsub.MockClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		client.ClearAllMessages()

		response := map[string]interface{}{
			"status":  "success",
			"message": "All mock messages cleared",
			"timestamp": time.Now(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}