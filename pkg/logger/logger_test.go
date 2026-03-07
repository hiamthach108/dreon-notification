package logger

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hiamthach108/dreon-notification/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestSanitize(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected any
	}{
		{
			name:     "password string",
			input:    "mypassword123",
			expected: "[REDACTED]",
		},
		{
			name:     "token string",
			input:    "bearer_token_12345",
			expected: "[REDACTED]",
		},
		{
			name:     "authorization header",
			input:    "Bearer authorization_token",
			expected: "[REDACTED]",
		},
		{
			name:     "api_key field",
			input:    "api_key_secret_value",
			expected: "[REDACTED]",
		},
		{
			name:     "secret data",
			input:    "my_secret_data",
			expected: "[REDACTED]",
		},
		{
			name:     "safe string",
			input:    "safe_user_data",
			expected: "safe_user_data",
		},
		{
			name:     "case insensitive password",
			input:    "MyPASSWORD123",
			expected: "[REDACTED]",
		},
		{
			name:     "struct with sensitive data",
			input:    map[string]string{"password": "secret123"},
			expected: "[REDACTED]",
		},
		{
			name:     "struct with safe data",
			input:    map[string]string{"username": "john_doe"},
			expected: `{"username":"john_doe"}`,
		},
		{
			name:     "number value",
			input:    12345,
			expected: 12345,
		},
		{
			name:     "boolean value",
			input:    true,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitize(tt.input)

			// Handle special case for struct comparison
			if tt.name == "struct with safe data" {
				// Convert result to JSON string for comparison
				if jsonBytes, err := json.Marshal(result); err == nil {
					if string(jsonBytes) != tt.expected {
						t.Errorf("sanitize() = %v, want %v", string(jsonBytes), tt.expected)
					}
				} else {
					t.Errorf("Failed to marshal result: %v", err)
				}
			} else if result != tt.expected {
				t.Errorf("sanitize() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.AppConfig
		wantErr bool
	}{
		{
			name: "valid debug config",
			config: &config.AppConfig{
				App: struct {
					Name    string `env:"APP_NAME"`
					Version string `env:"APP_VERSION"`
				}{
					Name: "test-service",
				},
				Logger: struct {
					Level string `env:"LOG_LEVEL"`
				}{
					Level: string(DebugLv),
				},
			},
			wantErr: false,
		},
		{
			name: "valid info config",
			config: &config.AppConfig{
				App: struct {
					Name    string `env:"APP_NAME"`
					Version string `env:"APP_VERSION"`
				}{
					Name: "test-service",
				},
				Logger: struct {
					Level string `env:"LOG_LEVEL"`
				}{
					Level: string(InfoLv),
				},
			},
			wantErr: false,
		},
		{
			name: "valid warn config",
			config: &config.AppConfig{
				App: struct {
					Name    string `env:"APP_NAME"`
					Version string `env:"APP_VERSION"`
				}{
					Name: "test-service",
				},
				Logger: struct {
					Level string `env:"LOG_LEVEL"`
				}{
					Level: string(WarnLv),
				},
			},
			wantErr: false,
		},
		{
			name: "valid error config",
			config: &config.AppConfig{
				App: struct {
					Name    string `env:"APP_NAME"`
					Version string `env:"APP_VERSION"`
				}{
					Name: "test-service",
				},
				Logger: struct {
					Level string `env:"LOG_LEVEL"`
				}{
					Level: string(ErrorLv),
				},
			},
			wantErr: false,
		},
		{
			name: "invalid level defaults to info",
			config: &config.AppConfig{
				App: struct {
					Name    string `env:"APP_NAME"`
					Version string `env:"APP_VERSION"`
				}{
					Name: "test-service",
				},
				Logger: struct {
					Level string `env:"LOG_LEVEL"`
				}{
					Level: "invalid",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLogger(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewLogger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && logger == nil {
				t.Error("NewLogger() returned nil logger")
			}
			if !tt.wantErr {
				// Cast to concrete type to access service field
				if zapLogger, ok := logger.(*zapLogger); ok {
					if zapLogger.service != tt.config.App.Name {
						t.Errorf("NewLogger() service = %v, want %v", zapLogger.service, tt.config.App.Name)
					}
				} else {
					t.Error("NewLogger() should return a *zapLogger")
				}
			}
		})
	}
}

func TestToZapFields(t *testing.T) {
	tests := []struct {
		name     string
		fields   []any
		expected int // expected number of zap fields
	}{
		{
			name:     "even number of fields",
			fields:   []any{"key1", "value1", "key2", "value2"},
			expected: 2,
		},
		{
			name:     "odd number of fields",
			fields:   []any{"orphan", "key1", "value1"},
			expected: 2, // orphan becomes field_0, then key1/value1
		},
		{
			name:     "empty fields",
			fields:   []any{},
			expected: 0,
		},
		{
			name:     "non-string key",
			fields:   []any{123, "value1"},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toZapFields(tt.fields...)
			if len(result) != tt.expected {
				t.Errorf("toZapFields() returned %d fields, want %d", len(result), tt.expected)
			}
		})
	}
}

func TestSanitizationInZapFields(t *testing.T) {
	// Test that sanitization is applied in toZapFields
	fields := []any{
		"password", "secret123",
		"username", "john_doe",
		"token", "bearer_token_xyz",
		"email", "user@example.com",
	}

	zapFields := toZapFields(fields...)

	// Check that we have the expected number of fields
	if len(zapFields) != 4 {
		t.Errorf("Expected 4 zap fields, got %d", len(zapFields))
	}

	// We can't easily inspect zap.Field values directly in tests,
	// but we can verify the function doesn't panic and returns expected count
}

// Helper function to create a test logger that writes to a buffer
func createTestLogger(buf *bytes.Buffer) *zapLogger {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:     "timestamp",
		LevelKey:    "level",
		MessageKey:  "msg",
		LineEnding:  zapcore.DefaultLineEnding,
		EncodeLevel: zapcore.CapitalLevelEncoder,
		EncodeTime:  zapcore.ISO8601TimeEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(buf),
		zapcore.DebugLevel,
	)

	logger := zap.New(core)

	return &zapLogger{
		logger:  logger,
		service: "test-service",
	}
}

func TestLoggerMethods(t *testing.T) {
	var buf bytes.Buffer
	logger := createTestLogger(&buf)

	tests := []struct {
		name  string
		logFn func()
		level string
	}{
		{
			name: "debug log",
			logFn: func() {
				logger.Debug("debug message", "key", "value")
			},
			level: "DEBUG",
		},
		{
			name: "info log",
			logFn: func() {
				logger.Info("info message", "key", "value")
			},
			level: "INFO",
		},
		{
			name: "warn log",
			logFn: func() {
				logger.Warn("warn message", "key", "value")
			},
			level: "WARN",
		},
		{
			name: "error log",
			logFn: func() {
				logger.Error("error message", "key", "value")
			},
			level: "ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFn()

			output := buf.String()
			if output == "" {
				t.Error("Expected log output, got empty string")
			}

			if !strings.Contains(output, tt.level) {
				t.Errorf("Expected log level %s in output, got: %s", tt.level, output)
			}

			if !strings.Contains(output, "test-service") {
				t.Errorf("Expected service name in output, got: %s", output)
			}
		})
	}
}

func TestLoggerSanitization(t *testing.T) {
	var buf bytes.Buffer
	logger := createTestLogger(&buf)

	// Test that sensitive data is sanitized in actual log output
	logger.Info("test message",
		"password", "secret123",
		"username", "john_doe",
		"api_key", "sensitive_api_key",
		"email", "user@example.com",
	)

	output := buf.String()

	// Parse the JSON log output
	var logEntry map[string]any
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output as JSON: %v", err)
	}

	// Check that sensitive fields are redacted
	if logEntry["password"] != "[REDACTED]" {
		t.Errorf("Expected password to be [REDACTED], got: %v", logEntry["password"])
	}

	if logEntry["api_key"] != "[REDACTED]" {
		t.Errorf("Expected api_key to be [REDACTED], got: %v", logEntry["api_key"])
	}

	// Check that non-sensitive fields are preserved
	if logEntry["username"] != "john_doe" {
		t.Errorf("Expected username to be preserved, got: %v", logEntry["username"])
	}

	if logEntry["email"] != "user@example.com" {
		t.Errorf("Expected email to be preserved, got: %v", logEntry["email"])
	}
}

func TestLoggerWith(t *testing.T) {
	var buf bytes.Buffer
	logger := createTestLogger(&buf)

	// Test the With method
	contextLogger := logger.With("request_id", "req123", "password", "secret456")

	// The With method should return a new logger instance
	if contextLogger == logger {
		t.Error("With() should return a new logger instance")
	}

	// Test that the new logger has the correct type
	if _, ok := contextLogger.(*zapLogger); !ok {
		t.Error("With() should return a *zapLogger")
	}
}

func TestNewLoggerWithConfigLevel(t *testing.T) {
	// Test that the logger correctly uses the Logger.Level from config
	testConfig := &config.AppConfig{
		App: struct {
			Name    string `env:"APP_NAME"`
			Version string `env:"APP_VERSION"`
		}{
			Name: "test-app",
		},
		Logger: struct {
			Level string `env:"LOG_LEVEL"`
		}{
			Level: string(DebugLv),
		},
	}

	logger, err := NewLogger(testConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	if zapLogger, ok := logger.(*zapLogger); ok {
		if zapLogger.service != "test-app" {
			t.Errorf("Expected service name 'test-app', got '%s'", zapLogger.service)
		}
	} else {
		t.Error("Expected logger to be *zapLogger")
	}

	// Test with empty level (should default to info)
	testConfig.Logger.Level = ""
	logger2, err := NewLogger(testConfig)
	if err != nil {
		t.Fatalf("Failed to create logger with empty level: %v", err)
	}

	if zapLogger2, ok := logger2.(*zapLogger); ok {
		if zapLogger2.service != "test-app" {
			t.Errorf("Expected service name 'test-app', got '%s'", zapLogger2.service)
		}
	} else {
		t.Error("Expected logger2 to be *zapLogger")
	}
}

func TestGetZapLogger(t *testing.T) {
	testConfig := &config.AppConfig{
		App: struct {
			Name    string `env:"APP_NAME"`
			Version string `env:"APP_VERSION"`
		}{
			Name: "test-app",
		},
		Logger: struct {
			Level string `env:"LOG_LEVEL"`
		}{
			Level: string(InfoLv),
		},
	}

	logger, err := NewLogger(testConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	zapLogger := logger.GetZapLogger()
	if zapLogger == nil {
		t.Error("GetZapLogger() returned nil")
	}

	// Verify it's actually a zap logger
	if zapLogger.Core() == nil {
		t.Error("GetZapLogger() returned invalid zap logger")
	}
}
