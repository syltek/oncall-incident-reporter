// Package middleware provides HTTP middleware functions for the application.
package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	apperrors "github.com/syltek/oncall-incident-reporter/pkg/errors"
	"github.com/syltek/oncall-incident-reporter/pkg/logutil"
	"go.uber.org/zap"
)

// ResponseWriter wraps http.ResponseWriter to capture the status code
type ResponseWriter struct {
    http.ResponseWriter
    statusCode int
}

// WriteHeader captures the status code and delegates to the underlying ResponseWriter
func (rw *ResponseWriter) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}

// Logging middleware logs HTTP request details including method, path, status code,
// and duration. It uses structured logging via zap.
func Logging(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // Wrap ResponseWriter to capture status code
        rw := &ResponseWriter{
            ResponseWriter: w,
            statusCode:    http.StatusOK, // Default to 200 if WriteHeader is never called
        }
        
        next.ServeHTTP(rw, r)
        
        logutil.Info("Request processed",
            zap.String("method", r.Method),
            zap.String("path", r.URL.Path),
            zap.Int("status", rw.statusCode),
            zap.Duration("duration", time.Since(start)),
            zap.String("remote_addr", r.RemoteAddr),
            zap.String("user_agent", r.UserAgent()),
        )
    })
}

// Recovery middleware recovers from panics and logs the error.
// It ensures the application continues running even if an individual
// request handler panics.
func Recovery(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if rec := recover(); rec != nil {
                err := apperrors.New(
                    http.StatusInternalServerError,
                    "Internal server error",
                    apperrors.CategoryServer,
                    fmt.Errorf("panic recovered: %v", rec),
                )
                logutil.Error("Panic recovered in request handler",
                    zap.Error(err),
                    zap.String("path", r.URL.Path),
                    zap.String("method", r.Method),
                )
                http.Error(w, err.Message, err.Code)
            }
        }()
        next.ServeHTTP(w, r)
    })
}

// ValidateSlackSignature middleware validates the X-Slack-Signature header
// to ensure requests are coming from Slack
func ValidateSlackSignature(next http.Handler) http.Handler {
    signingSecret := os.Getenv("SLACK_SIGNING_SECRET")
    if signingSecret == "" {
        logutil.Error("SLACK_SIGNING_SECRET environment variable is not set")
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            http.Error(w, "Server configuration error", http.StatusInternalServerError)
        })
    }

    logutil.Debug("Validating Slack signature. Using it from SLACK_SIGNING_SECRET environment variable")

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Get required headers
        timestamp := r.Header.Get("X-Slack-Request-Timestamp")
        signature := r.Header.Get("X-Slack-Signature")

        logutil.Debug("Timestamp", zap.String("timestamp", timestamp))
        logutil.Debug("Signature", zap.String("signature", signature))

        if timestamp == "" || signature == "" {
            logutil.Error("Missing required Slack headers",
                zap.String("path", r.URL.Path),
                zap.String("method", r.Method),
            )
            http.Error(w, "Invalid request", http.StatusBadRequest)
            return
        }

        // Read the request body
        body, err := io.ReadAll(r.Body)
        if err != nil {
            logutil.Error("Failed to read request body",
                zap.Error(err),
                zap.String("path", r.URL.Path),
            )
            http.Error(w, "Invalid request", http.StatusBadRequest)
            return
        }

        // Important: Restore the body for subsequent middleware/handlers
        r.Body = io.NopCloser(bytes.NewBuffer(body))

        // Create the signature base string
        baseString := fmt.Sprintf("v0:%s:%s", timestamp, string(body))

        // Create the signature using HMAC-SHA256
        mac := hmac.New(sha256.New, []byte(signingSecret))
        mac.Write([]byte(baseString))
        expectedSignature := "v0=" + hex.EncodeToString(mac.Sum(nil))

        // Compare signatures using a constant-time comparison
        if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
            logutil.Error("Invalid Slack signature",
                zap.String("path", r.URL.Path),
                zap.String("method", r.Method),
            )
            http.Error(w, "Invalid signature", http.StatusUnauthorized)
            return
        }

        logutil.Debug("Signature is valid")

        next.ServeHTTP(w, r)
    })
}
