// Package main provides the entry point for the oncall-incident-reporter application.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/slack-go/slack"
	"github.com/syltek/oncall-incident-reporter/internal/config"
	handlers "github.com/syltek/oncall-incident-reporter/internal/handlers/slack"
	"github.com/syltek/oncall-incident-reporter/internal/router"
	"github.com/syltek/oncall-incident-reporter/internal/service"
	"github.com/syltek/oncall-incident-reporter/pkg/logutil"
	"go.uber.org/zap"
)

const (
    defaultPort     = "8080"
    shutdownTimeout = 10 * time.Second
)

var (
    cfg    *config.Config
    logger *zap.Logger
)

// initLogger initializes the application logger based on the DEBUG environment variable.
func initLogger() {
    debug := os.Getenv("DEBUG") == "true"
    logutil.InitLogger(debug)
    logger = logutil.Logger
}

// initConfig loads the application configuration and validates it.
func initConfig() {
    var err error
    cfg, err = config.LoadConfig()
    if err != nil {
        logger.Fatal("Failed to load configuration",
            zap.Error(err),
        )
    }
    logger.Debug("Configuration loaded successfully",
        zap.Any("endpoints", cfg.Endpoints),
        zap.Any("slack_config", cfg.SlackConfig),
        zap.Any("modal", cfg.Modal),
    )
}

func init() {
    initLogger()
    initConfig()
    logger.Info("Application initialization completed")
}

// startLocalServer starts the HTTP server for local development
func startLocalServer(r *router.Router) error {
    port := os.Getenv("PORT")
    if port == "" {
        port = defaultPort
    }

    srv := &http.Server{
        Addr:         ":" + port,
        Handler:      r,
        ReadTimeout:  5 * time.Second,
        WriteTimeout: 10 * time.Second,
        IdleTimeout:  120 * time.Second,
    }

    // Channel to listen for errors coming from the listener.
    serverErrors := make(chan error, 1)
    // Channel to listen for an interrupt or terminate signal from the OS.
    shutdown := make(chan os.Signal, 1)
    signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

    // Start the server
    go func() {
        logger.Info("Starting local server", zap.String("port", port))
        serverErrors <- srv.ListenAndServe()
    }()

    // Blocking main and waiting for shutdown.
    select {
    case err := <-serverErrors:
        return fmt.Errorf("server error: %w", err)

    case sig := <-shutdown:
        logger.Info("Starting shutdown", zap.String("signal", sig.String()))
        
        // Give outstanding requests a deadline for completion.
        ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
        defer cancel()

        // Asking listener to shut down and shed load.
        if err := srv.Shutdown(ctx); err != nil {
            // Error from closing listeners, or context timeout:
            return fmt.Errorf("graceful shutdown failed: %w", err)
        }
    }

    return nil
}

func main() {
    defer func() {
        _ = logger.Sync()
    }()

    // Initialize Datadog service
    configuration := datadog.NewConfiguration()
    apiClient := datadog.NewAPIClient(configuration)
    datadogService := service.NewDatadogService(datadogV1.NewEventsApi(apiClient))
    
    // Initialize Slack service
    slackClient := slack.New(cfg.SlackConfig.Token)
	slackService := service.NewSlackService(slackClient)

	handler := handlers.NewSlackHandler(slackService, datadogService, cfg)
    r := router.NewRouter(handler, cfg)

    if os.Getenv("LOCAL") == "true" {
        if err := startLocalServer(r); err != nil {
            logger.Fatal("Server error", zap.Error(err))
        }
        return
    }

    logger.Info("Starting lambda handler")
    lambda.Start(r.LambdaHandler)
}
