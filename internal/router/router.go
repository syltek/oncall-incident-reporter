// Package router provides HTTP routing and request handling functionality for both
// AWS Lambda and local server environments.
package router

import (
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/awslabs/aws-lambda-go-api-proxy/core"
	"github.com/awslabs/aws-lambda-go-api-proxy/gorillamux"
	"github.com/gorilla/mux"
	"github.com/syltek/oncall-incident-reporter/internal/config"
	"github.com/syltek/oncall-incident-reporter/internal/middleware"
	"github.com/syltek/oncall-incident-reporter/pkg/logutil"
	"go.uber.org/zap"
)

// Handler defines the interface for handling Slack commands and modal submissions.
type Handler interface {
    HandleCommand(w http.ResponseWriter, r *http.Request)
    HandleModalSubmission(w http.ResponseWriter, r *http.Request)
}

// Router wraps the mux.Router and provides additional functionality for
// handling both HTTP and Lambda requests.
type Router struct {
    *mux.Router
    config *config.Config
    handler Handler
    adapter *gorillamux.GorillaMuxAdapter
}

// NewRouter creates and configures a new Router instance with the provided configuration.
func NewRouter(handler Handler, config *config.Config) *Router {
    r := &Router{
        Router:  mux.NewRouter(),
        handler: handler,
        config: config,
    }
    
    r.setupRoutes()
    r.adapter = gorillamux.New(r.Router)
    
    return r
}

// setupRoutes configures all routes and middleware for the application.
func (r *Router) setupRoutes() {
    // Add middleware
    r.Use(middleware.Recovery)
    r.Use(middleware.Logging)
    r.Use(middleware.ValidateSlackSignature)
    
    // Configure routes from config
    if r.config.Endpoints == nil {
        logutil.Error("Endpoints configuration is missing")
        return
    }

    r.HandleFunc(r.config.Endpoints.SlackCommand, r.handler.HandleCommand).
        Methods(http.MethodPost)
    r.HandleFunc(r.config.Endpoints.SlackModalParser, r.handler.HandleModalSubmission).
        Methods(http.MethodPost)
}

// LambdaHandler handles requests from AWS Lambda.
func (r *Router) LambdaHandler(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    logutil.Debug("Starting LambdaHandler")
    logutil.Debug("Request", zap.Any("request", req))

    switchableReq := *core.NewSwitchableAPIGatewayRequestV1(&req)
    resp, err := r.adapter.Proxy(switchableReq)
    if err != nil {
        logutil.Error("Error proxying request", zap.Error(err))
        return events.APIGatewayProxyResponse{}, err
    }

    // Convert to Version1 before logging to see the actual response contents
    v1Response := resp.Version1()
    logutil.Info("Response", zap.Any("response", *v1Response))

    return *v1Response, nil
}
