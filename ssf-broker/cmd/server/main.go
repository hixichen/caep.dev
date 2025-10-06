package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-broker/internal/broker"
	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-broker/internal/handlers"
	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-broker/internal/pubsub"
	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-broker/internal/registry"
	"github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-broker/pkg/api"
)

func main() {
	// Create structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	logger.Info("Starting SSF Broker Service", "version", "v1.0.0", "port", config.Server.Port)

	// Initialize Pub/Sub client
	pubsubClient, err := pubsub.NewClient(context.Background(), config.PubSub.ProjectID)
	if err != nil {
		log.Fatalf("Failed to create Pub/Sub client: %v", err)
	}
	defer pubsubClient.Close()

	// Initialize receiver registry
	receiverRegistry := registry.NewMemoryRegistry() // TODO: Replace with persistent storage

	// Initialize broker
	ssfBroker := broker.New(pubsubClient, receiverRegistry, logger)

	// Initialize handlers
	handlerConfig := &handlers.Config{
		Logger:   logger,
		Broker:   ssfBroker,
		Registry: receiverRegistry,
	}

	apiHandlers := handlers.New(handlerConfig)

	// Setup HTTP routes
	mux := http.NewServeMux()

	// SSF standard endpoints
	mux.HandleFunc("POST /events", apiHandlers.HandleEvents)
	mux.HandleFunc("GET /.well-known/ssf_configuration", apiHandlers.HandleSSFConfiguration)

	// Management endpoints
	mux.HandleFunc("GET /health", apiHandlers.HandleHealth)
	mux.HandleFunc("GET /ready", apiHandlers.HandleReady)
	mux.HandleFunc("GET /metrics", apiHandlers.HandleMetrics)

	// Receiver registration API
	mux.HandleFunc("POST /api/v1/receivers", apiHandlers.HandleRegisterReceiver)
	mux.HandleFunc("GET /api/v1/receivers", apiHandlers.HandleListReceivers)
	mux.HandleFunc("GET /api/v1/receivers/{id}", apiHandlers.HandleGetReceiver)
	mux.HandleFunc("PUT /api/v1/receivers/{id}", apiHandlers.HandleUpdateReceiver)
	mux.HandleFunc("DELETE /api/v1/receivers/{id}", apiHandlers.HandleUnregisterReceiver)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("HTTP server starting", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down SSF Broker Service...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}

	logger.Info("SSF Broker Service stopped")
}

// loadConfig loads configuration from environment variables and files
func loadConfig() (*api.Config, error) {
	config := &api.Config{
		Server: api.ServerConfig{
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: getEnvInt("SERVER_PORT", 8080),
		},
		PubSub: api.PubSubConfig{
			ProjectID:   getEnv("GCP_PROJECT_ID", ""),
			TopicPrefix: getEnv("PUBSUB_TOPIC_PREFIX", "ssf-events"),
		},
		Auth: api.AuthConfig{
			JWTSecret: getEnv("JWT_SECRET", "default-secret"), // TODO: Generate secure secret
		},
		Logging: api.LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
	}

	// Validate required configuration
	if config.PubSub.ProjectID == "" {
		return nil, fmt.Errorf("GCP_PROJECT_ID environment variable is required")
	}

	return config, nil
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets an environment variable as int with a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}