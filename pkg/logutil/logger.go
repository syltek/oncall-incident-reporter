package logutil

import (
	"fmt"
	"regexp"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func InitLogger(debug bool) {
	var config zap.Config
	if debug {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}

	// Configure log level dynamically
	level := zapcore.InfoLevel
	if debug {
		level = zapcore.DebugLevel
	}
	config.Level = zap.NewAtomicLevelAt(level)

	logger, err := config.Build()
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	Logger = logger
}

// Sync ensures all logs are flushed before the app exits.
func Sync() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}

// RedactSensitiveInfo masks sensitive information in log messages.
func RedactSensitiveInfo(msg string) string {
	sensitivePatterns := []string{
		`(?i)Bearer\s+[\w\.-]+`,                    // Bearer tokens
		`(?i)password\s*[=:]\s*["']?[\w\.-]+["']?`, // password=secret or "password": "secret"
		`(?i)token\s*[=:]\s*["']?[\w\.-]+["']?`,    // token=secret or "token": "secret"
		`(?i)"Token"\s*:\s*"(xoxb-[0-9A-Za-z-]+)"`, // Slack tokens in config JSON
		`(?i)xoxb-[0-9A-Za-z-]+`,                   // Raw Slack bot tokens
		`(?i)"SlackToken"\s*:\s*"[^"]*"`,           // Slack tokens in JSON format
		`(?i)"access_token"\s*:\s*"[^"]*"`,         // Access tokens in JSON
		`(?i)"refresh_token"\s*:\s*"[^"]*"`,        // Refresh tokens in JSON
		`(?i)"signing_secret"\s*:\s*"[^"]*"`,       // Signing secrets in JSON
		`(?i)"X-Slack-Signature"\s*:\s*"[^"]*"`,    // Slack signature in JSON
	}

	for _, pattern := range sensitivePatterns {
		re := regexp.MustCompile(pattern)
		msg = re.ReplaceAllString(msg, `"[REDACTED]"`)
	}

	return msg
}

// RedactFields processes zap.Fields to redact sensitive data.
func RedactFields(fields []zap.Field) []zap.Field {
	redactedFields := make([]zap.Field, len(fields))
	for i, field := range fields {
		switch field.Type {
		case zapcore.StringType:
			field.Interface = RedactSensitiveInfo(field.String)
		case zapcore.BinaryType:
			// Handle byte strings by converting to string first
			field.Interface = RedactSensitiveInfo(string(field.Interface.([]byte)))
		case zapcore.ReflectType, zapcore.ObjectMarshalerType:
			// Handle complex objects by converting them to string representation
			if str, ok := field.Interface.(string); ok {
				field.Interface = RedactSensitiveInfo(str)
			} else {
				// Convert complex objects to string for redaction
				str := fmt.Sprintf("%+v", field.Interface)
				field.Interface = RedactSensitiveInfo(str)
			}
		}
		redactedFields[i] = field
	}
	return redactedFields
}

// Helper functions for convenience
func Info(msg string, fields ...zap.Field) {
	msg = RedactSensitiveInfo(msg)
	fields = RedactFields(fields)
	// AddCallerSkip(1) skips the first caller, which is the logutil package itself
	Logger.WithOptions(zap.AddCallerSkip(1)).Info(msg, fields...)
}

func Debug(msg string, fields ...zap.Field) {
	msg = RedactSensitiveInfo(msg)
	fields = RedactFields(fields)
	// AddCallerSkip(1) skips the first caller, which is the logutil package itself
	Logger.WithOptions(zap.AddCallerSkip(1)).Debug(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	msg = RedactSensitiveInfo(msg)
	fields = RedactFields(fields)
	// AddCallerSkip(1) skips the first caller, which is the logutil package itself
	Logger.WithOptions(zap.AddCallerSkip(1)).Error(msg, fields...)
}
