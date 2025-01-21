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

var (
    cfg    *config.Config
    logger *zap.Logger
)

// initLogger initializes the application logger based on the DEBUG environment variable.
func initLogger() {
    debug := cfg.LogLevel == config.DEBUG_LOG_LEVEL
    logutil.InitLogger(debug)
    logger = logutil.Logger
}

// initConfig loads the application configuration and validates it.
func initConfig() {
    var err error
    cfg, err = config.LoadConfig()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
        os.Exit(1)
    }
    if cfg.LogLevel == config.DEBUG_LOG_LEVEL {
        fmt.Printf("Configuration loaded successfully:\n"+
            "  Log Level: %s\n"+
            "  Local Port: %d\n"+
            "  Slack Config:\n"+
            "    Token: %s\n"+
            "    Channel ID: %s\n"+
            "  Endpoints:\n"+
            "    Base URL: %s\n",
            cfg.LogLevel,
            cfg.Local.Port,
            maskToken(cfg.SlackConfig.Token),
            cfg.SlackConfig.ChannelID,
            cfg.Endpoints,
        )
    }
}

// maskToken masks a token string for secure logging
func maskToken(token string) string {
    if len(token) <= 8 {
        return "****"
    }
    return token[:4] + "..." + token[len(token)-4:]
}

func init() {
    initConfig()
    initLogger()
    logger.Info("Application initialization completed")
}

// startLocalServer starts the HTTP server for local development
func startLocalServer(r *router.Router) error {
    port := cfg.Local.Port

    srv := &http.Server{
        Addr:         ":" + fmt.Sprintf("%d", port),
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
        logger.Info("Starting local server", zap.Int("port", port))
        serverErrors <- srv.ListenAndServe()
    }()

    // Blocking main and waiting for shutdown.
    select {
    case err := <-serverErrors:
        return fmt.Errorf("server error: %w", err)

    case sig := <-shutdown:
        logger.Info("Starting shutdown", zap.String("signal", sig.String()))
        
        // Give outstanding requests a deadline for completion.
        ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Local.ShutdownTimeout) * time.Second)
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

    if cfg.Local.Enabled {
        if err := startLocalServer(r); err != nil {
            logger.Fatal("Server error", zap.Error(err))
        }
        return
    }

    logger.Info("Starting lambda handler")
    lambda.Start(r.LambdaHandler)
}
