package main

import (
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/syltek/oncall-incident-reporter/internal/config"
	"github.com/syltek/oncall-incident-reporter/internal/router"
	"github.com/syltek/oncall-incident-reporter/pkg/logutil"
	"go.uber.org/zap"
)

var (
	cfg    *config.Config
	logger *zap.Logger
)

func initLogger() {
	debug := os.Getenv("DEBUG") == "true"
	logutil.InitLogger(debug)
	logger = logutil.Logger
}

func initConfig() {
	var err error
	cfg, err = config.LoadConfig()
	if err != nil {
		logger.Error("Failed to load configuration", zap.Error(err))
		os.Exit(1)
	}
	logger.Debug("Configuration loaded successfully",
		zap.Any("endpoints", cfg.Endpoints),
		zap.Any("slack_config", cfg.SlackConfig),
		zap.Any("modal", cfg.Modal),
		zap.Any("monitor_config", cfg.MonitorConfig))
}

func init() {
	initLogger()
	initConfig()
	logger.Info("Lambda initialization completed")
}

func main() {
	defer func() { _ = logger.Sync() }()

	logger.Debug("Creating new router")
	r := router.NewRouter(cfg)

	if os.Getenv("LOCAL") != "" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		logger.Info("Starting local server", zap.String("port", port))
		if err := http.ListenAndServe(":"+port, r); err != nil {
			logger.Error("Server error", zap.Error(err))
		}
		return
	}

	logger.Debug("Starting lambda handler")
	lambda.Start(r.LambdaHandler)
}
