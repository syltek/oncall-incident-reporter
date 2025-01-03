package router

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/gorilla/mux"
	"github.com/syltek/oncall-incident-reporter/internal/config"
	handlers "github.com/syltek/oncall-incident-reporter/internal/handlers/slack"
	"github.com/syltek/oncall-incident-reporter/internal/middleware"
	"github.com/syltek/oncall-incident-reporter/internal/responsewriter"
	"github.com/syltek/oncall-incident-reporter/pkg/logutil"
	"go.uber.org/zap"
)

type Router struct {
    *mux.Router
    cfg          *config.Config
    handler Handler
}

// Handler defines the interface for request handlers
type Handler interface {
    HandleCommand(w http.ResponseWriter, r *http.Request)
    HandleModalSubmission(w http.ResponseWriter, r *http.Request)
}

func NewRouter(cfg *config.Config) *Router {
    r := &Router{
        Router:       mux.NewRouter(),
        cfg:         cfg,
        handler: handlers.NewSlackHandler(cfg),
    }
    r.setupRoutes()
    return r
}

func (r *Router) setupRoutes() {
    r.Use(middleware.Recovery)
    r.Use(middleware.Logging)

    r.HandleFunc(r.cfg.Endpoints.SlackCommand, r.handler.HandleCommand).
        Methods(http.MethodPost)
    r.HandleFunc(r.cfg.Endpoints.SlackModalParser, r.handler.HandleModalSubmission).
        Methods(http.MethodPost)
}
// LambdaHandler processes AWS Lambda requests and routes them appropriately.
// It converts API Gateway requests into HTTP requests that can be handled by the router.
func (r *Router) LambdaHandler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    logutil.Debug("Starting LambdaHandler", 
        zap.String("path", request.Path),
        zap.String("method", request.HTTPMethod),
        zap.String("resource", request.Resource))

    if !r.isValidEndpoint(request.Resource) {
        return r.notFoundResponse()
    }

    return r.adaptAPIGatewayRequest(request)
}

func (r *Router) isValidEndpoint(resource string) bool {
    return resource == r.cfg.Endpoints.SlackCommand || 
           resource == r.cfg.Endpoints.SlackModalParser
}

func (r *Router) notFoundResponse() (events.APIGatewayProxyResponse, error) {
    return events.APIGatewayProxyResponse{
        StatusCode: http.StatusNotFound,
        Body:       "Endpoint not found",
    }, nil
}

// adaptAPIGatewayRequest converts API Gateway requests to HTTP requests and processes them
func (r *Router) adaptAPIGatewayRequest(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    logutil.Debug("Starting adaptAPIGatewayRequest", 
        zap.Bool("isBase64Encoded", request.IsBase64Encoded))

    body, err := r.getRequestBody(request)
    if err != nil {
        return r.errorResponse(http.StatusBadRequest, "Invalid request body encoding", err)
    }

    req, err := r.createHTTPRequest(request, body)
    if err != nil {
        return r.errorResponse(http.StatusInternalServerError, "Failed to create request", err)
    }

    return r.handleRequest(req)
}

func (r *Router) getRequestBody(request events.APIGatewayProxyRequest) (string, error) {
    if !request.IsBase64Encoded {
        return request.Body, nil
    }

    decoded, err := base64.StdEncoding.DecodeString(request.Body)
    if err != nil {
        logutil.Error("Failed to decode base64 body", zap.Error(err))
        return "", err
    }
    return string(decoded), nil
}

func (r *Router) createHTTPRequest(request events.APIGatewayProxyRequest, body string) (*http.Request, error) {
    req, err := http.NewRequest(request.HTTPMethod, request.Resource, strings.NewReader(body))
    if err != nil {
        return nil, err
    }

    for k, v := range request.Headers {
        req.Header.Set(k, v)
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    return req, nil
}

func (r *Router) handleRequest(req *http.Request) (events.APIGatewayProxyResponse, error) {
    w := responsewriter.NewResponseWriter()
    r.ServeHTTP(w, req)

    headers := r.prepareResponseHeaders(w)
    
    return events.APIGatewayProxyResponse{
        StatusCode: w.StatusCode,
        Headers:    headers,
        Body:       string(w.Body),
    }, nil
}

func (r *Router) prepareResponseHeaders(w *responsewriter.ResponseWriter) map[string]string {
    headers := make(map[string]string)
    for k, v := range w.Header() {
        if len(v) > 0 {
            headers[k] = v[0]
        }
    }
    headers["Content-Type"] = "application/json"
    return headers
}

func (r *Router) errorResponse(statusCode int, message string, err error) (events.APIGatewayProxyResponse, error) {
    return events.APIGatewayProxyResponse{
        StatusCode: statusCode,
        Body:       message,
    }, err
}
