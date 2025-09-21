package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestInitialize(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		hasErr bool
	}{
		{
			name: "valid json config",
			config: Config{
				Level:  "info",
				Format: "json",
			},
			hasErr: false,
		},
		{
			name: "valid console config",
			config: Config{
				Level:  "debug",
				Format: "console",
			},
			hasErr: false,
		},
		{
			name: "invalid log level",
			config: Config{
				Level:  "invalid",
				Format: "json",
			},
			hasErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Initialize(tt.config)
			if tt.hasErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.hasErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  "debug",
		Format: "json",
		Output: &buf,
	}

	err := Initialize(config)
	if err != nil {
		t.Fatalf("failed to initialize logger: %v", err)
	}

	logger := GetLogger()

	tests := []struct {
		name     string
		logFunc  func()
		expected string
	}{
		{
			name:     "debug",
			logFunc:  func() { logger.Debug("debug message") },
			expected: "debug",
		},
		{
			name:     "info",
			logFunc:  func() { logger.Info("info message") },
			expected: "info",
		},
		{
			name:     "warn",
			logFunc:  func() { logger.Warn("warn message") },
			expected: "warn",
		},
		{
			name:     "error",
			logFunc:  func() { logger.Error("error message") },
			expected: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc()

			if buf.Len() == 0 {
				t.Error("expected log output but got none")
				return
			}

			var logEntry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
				t.Errorf("failed to parse log entry: %v", err)
				return
			}

			level, ok := logEntry["level"].(string)
			if !ok {
				t.Error("level field not found or not string")
				return
			}

			if level != tt.expected {
				t.Errorf("expected level %s, got %s", tt.expected, level)
			}
		})
	}
}

func TestContextLogging(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  "info",
		Format: "json",
		Output: &buf,
	}

	err := Initialize(config)
	if err != nil {
		t.Fatalf("failed to initialize logger: %v", err)
	}

	// Test with request ID
	requestID := "test-request-123"
	ctx := ContextWithRequestID(context.Background(), requestID)

	logger := GetLogger().WithContext(ctx)
	logger.Info("test message")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log entry: %v", err)
	}

	if logEntry["request_id"] != requestID {
		t.Errorf("expected request_id %s, got %v", requestID, logEntry["request_id"])
	}

	// Test with user ID
	buf.Reset()
	userID := "test-user-456"
	ctx = ContextWithUserID(ctx, userID)

	logger = GetLogger().WithContext(ctx)
	logger.Info("test message with user")

	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log entry: %v", err)
	}

	if logEntry["request_id"] != requestID {
		t.Errorf("expected request_id %s, got %v", requestID, logEntry["request_id"])
	}

	if logEntry["user_id"] != userID {
		t.Errorf("expected user_id %s, got %v", userID, logEntry["user_id"])
	}
}

func TestWithFields(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  "info",
		Format: "json",
		Output: &buf,
	}

	err := Initialize(config)
	if err != nil {
		t.Fatalf("failed to initialize logger: %v", err)
	}

	logger := GetLogger().WithFields(map[string]interface{}{
		"field1": "value1",
		"field2": 42,
		"field3": true,
	})

	logger.Info("test message")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log entry: %v", err)
	}

	if logEntry["field1"] != "value1" {
		t.Errorf("expected field1 'value1', got %v", logEntry["field1"])
	}

	if logEntry["field2"] != float64(42) { // JSON numbers are float64
		t.Errorf("expected field2 42, got %v", logEntry["field2"])
	}

	if logEntry["field3"] != true {
		t.Errorf("expected field3 true, got %v", logEntry["field3"])
	}
}

func TestWithError(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  "info",
		Format: "json",
		Output: &buf,
	}

	err := Initialize(config)
	if err != nil {
		t.Fatalf("failed to initialize logger: %v", err)
	}

	testError := errors.New("test error")
	logger := GetLogger().WithError(testError)
	logger.Error("error occurred")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log entry: %v", err)
	}

	if logEntry["error"] != "test error" {
		t.Errorf("expected error 'test error', got %v", logEntry["error"])
	}
}

func TestGlobalFunctions(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  "debug",
		Format: "json",
		Output: &buf,
	}

	err := Initialize(config)
	if err != nil {
		t.Fatalf("failed to initialize logger: %v", err)
	}

	// Test global Info function
	Info("global info message")

	if buf.Len() == 0 {
		t.Error("expected log output but got none")
		return
	}

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Errorf("failed to parse log entry: %v", err)
		return
	}

	if logEntry["level"] != "info" {
		t.Errorf("expected level 'info', got %v", logEntry["level"])
	}

	if logEntry["message"] != "global info message" {
		t.Errorf("expected message 'global info message', got %v", logEntry["message"])
	}
}

func TestSpecializedLogging(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  "info",
		Format: "json",
		Output: &buf,
	}

	err := Initialize(config)
	if err != nil {
		t.Fatalf("failed to initialize logger: %v", err)
	}

	ctx := ContextWithRequestID(context.Background(), "test-request")

	// Test LogRequest
	buf.Reset()
	LogRequest(ctx, "GET", "/api/test", 200, 100*time.Millisecond)

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log entry: %v", err)
	}

	if logEntry["method"] != "GET" {
		t.Errorf("expected method 'GET', got %v", logEntry["method"])
	}

	if logEntry["path"] != "/api/test" {
		t.Errorf("expected path '/api/test', got %v", logEntry["path"])
	}

	if logEntry["status_code"] != float64(200) {
		t.Errorf("expected status_code 200, got %v", logEntry["status_code"])
	}

	// Test LogError
	buf.Reset()
	testError := errors.New("test error")
	LogError(ctx, testError, "operation failed", map[string]interface{}{
		"operation": "test_op",
	})

	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log entry: %v", err)
	}

	if logEntry["error"] != "test error" {
		t.Errorf("expected error 'test error', got %v", logEntry["error"])
	}

	if logEntry["operation"] != "test_op" {
		t.Errorf("expected operation 'test_op', got %v", logEntry["operation"])
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected zerolog.Level
		hasError bool
	}{
		{"debug", zerolog.DebugLevel, false},
		{"info", zerolog.InfoLevel, false},
		{"warn", zerolog.WarnLevel, false},
		{"warning", zerolog.WarnLevel, false},
		{"error", zerolog.ErrorLevel, false},
		{"fatal", zerolog.FatalLevel, false},
		{"panic", zerolog.PanicLevel, false},
		{"disabled", zerolog.Disabled, false},
		{"invalid", zerolog.InfoLevel, true},
		{"DEBUG", zerolog.DebugLevel, false}, // Test case insensitivity
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level, err := parseLogLevel(tt.input)

			if tt.hasError && err == nil {
				t.Error("expected error but got none")
			}

			if !tt.hasError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if level != tt.expected {
				t.Errorf("expected level %v, got %v", tt.expected, level)
			}
		})
	}
}

func TestRequestIDGeneration(t *testing.T) {
	id1 := NewRequestID()
	id2 := NewRequestID()

	if id1 == id2 {
		t.Error("expected different request IDs")
	}

	if len(id1) == 0 {
		t.Error("expected non-empty request ID")
	}

	// UUID format check (basic)
	if !strings.Contains(id1, "-") {
		t.Error("expected UUID format with dashes")
	}
}

func TestContextHelpers(t *testing.T) {
	ctx := context.Background()
	requestID := "test-request-123"
	userID := "test-user-456"

	// Test request ID context
	ctx = ContextWithRequestID(ctx, requestID)
	retrievedRequestID, ok := GetRequestIDFromContext(ctx)
	if !ok {
		t.Error("failed to retrieve request ID from context")
	}
	if retrievedRequestID != requestID {
		t.Errorf("expected request ID %s, got %s", requestID, retrievedRequestID)
	}

	// Test user ID context
	ctx = ContextWithUserID(ctx, userID)
	retrievedUserID, ok := GetUserIDFromContext(ctx)
	if !ok {
		t.Error("failed to retrieve user ID from context")
	}
	if retrievedUserID != userID {
		t.Errorf("expected user ID %s, got %s", userID, retrievedUserID)
	}

	// Test missing values
	emptyCtx := context.Background()
	_, ok = GetRequestIDFromContext(emptyCtx)
	if ok {
		t.Error("expected no request ID in empty context")
	}

	_, ok = GetUserIDFromContext(emptyCtx)
	if ok {
		t.Error("expected no user ID in empty context")
	}
}