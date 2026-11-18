// Package logging configures structured application, HTTP, and framework logs.
package logging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	kratoslog "github.com/go-kratos/kratos/v2/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/grpclog"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Config mirrors the logging section used by Heron Platform.
type Config struct {
	Level       string   `json:"level"`
	Format      string   `json:"format"`
	Outputs     []string `json:"outputs"`
	Environment string   `json:"environment"`
	Files       []File   `json:"files"`
}

// File configures one rotating file output.
type File struct {
	Path     string   `json:"path"`
	Level    string   `json:"level"`
	Rotation Rotation `json:"rotation"`
}

// Rotation defines the size and age limits for a log file.
type Rotation struct {
	Enabled    bool `json:"enabled"`
	MaxSize    int  `json:"max_size"`
	MaxAge     int  `json:"max_age"`
	MaxBackups int  `json:"max_backups"`
}

type contextKey struct{}

// New creates a Zap logger using the same fields and encodings as Heron Platform.
func New(cfg Config, serviceID, serviceVersion string) (*zap.Logger, error) {
	if cfg.Level == "" {
		cfg.Level = "info"
	}
	if cfg.Format == "" {
		cfg.Format = "console"
	}
	if cfg.Environment == "" {
		cfg.Environment = "local"
	}
	if len(cfg.Outputs) == 0 {
		cfg.Outputs = []string{"stdout"}
	}
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}
	if cfg.Format != "console" && cfg.Format != "json" {
		return nil, fmt.Errorf("logging.format must be console or json")
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.LevelKey = "level"
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.CallerKey = "caller"
	encoderConfig.EncodeCaller = zapcore.FullCallerEncoder
	encoderConfig.MessageKey = "msg"

	var cores []zapcore.Core
	for _, output := range cfg.Outputs {
		switch output {
		case "stdout":
			cores = append(cores, makeCore(cfg.Format, encoderConfig, os.Stdout, level))
		case "stderr":
			cores = append(cores, makeCore(cfg.Format, encoderConfig, os.Stderr, level))
		case "file":
			if len(cfg.Files) == 0 {
				return nil, fmt.Errorf("logging.files is required when outputs includes file")
			}
			for _, file := range cfg.Files {
				if strings.TrimSpace(file.Path) == "" {
					return nil, fmt.Errorf("logging.files[].path is required")
				}
				fileLevel := file.Level
				if fileLevel == "" {
					fileLevel = cfg.Level
				}
				parsedFileLevel, err := parseLevel(fileLevel)
				if err != nil {
					return nil, fmt.Errorf("logging file %q: %w", file.Path, err)
				}
				writer, err := fileWriter(file)
				if err != nil {
					return nil, fmt.Errorf("open log file %q: %w", file.Path, err)
				}
				cores = append(cores, makeCore(cfg.Format, encoderConfig, writer, parsedFileLevel))
			}
		default:
			return nil, fmt.Errorf("unsupported logging output %q", output)
		}
	}
	if len(cores) == 0 {
		return nil, fmt.Errorf("logging.outputs must contain stdout, stderr, or file")
	}

	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "unknown"
	}
	core := zapcore.NewTee(cores...)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)).With(
		zap.String("service_id", serviceID),
		zap.String("service_version", serviceVersion),
		zap.String("environment", cfg.Environment),
		zap.String("hostname", hostname),
		zap.Int("pid", os.Getpid()),
	)
	return logger, nil
}

// InitDefault installs a console logger before remote configuration is loaded.
func InitDefault() {
	logger, _ := New(Config{}, "stallink", "unknown")
	Set(logger)
}

// Set installs the application logger for both Zap and Kratos integrations.
func Set(logger *zap.Logger) {
	if logger == nil {
		return
	}
	zap.ReplaceGlobals(logger)
	kratoslog.SetLogger(KratosLogger())
	log.SetOutput(io.Discard)
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(io.Discard, io.Discard, io.Discard))
}

// Sync flushes buffered log entries. Some console writers return errors on Sync.
func Sync() {
	_ = zap.L().Sync()
}

// Logger returns the request-scoped logger when present, or the global logger.
func Logger(ctx context.Context) *zap.Logger {
	if ctx != nil {
		if logger, ok := ctx.Value(contextKey{}).(*zap.Logger); ok && logger != nil {
			return logger
		}
	}
	return zap.L()
}

// WithFields derives a context carrying a logger enriched with structured fields.
func WithFields(ctx context.Context, fields ...zap.Field) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, contextKey{}, Logger(ctx).With(fields...))
}

// ErrorContext writes an error with the caller's context fields.
func ErrorContext(ctx context.Context, message string, fields ...zap.Field) {
	Logger(ctx).WithOptions(zap.AddCallerSkip(1)).Error(message, fields...)
}

// WarnContext writes a warning with the caller's context fields.
func WarnContext(ctx context.Context, message string, fields ...zap.Field) {
	Logger(ctx).WithOptions(zap.AddCallerSkip(1)).Warn(message, fields...)
}

// InfoContext writes an informational record with the caller's context fields.
func InfoContext(ctx context.Context, message string, fields ...zap.Field) {
	Logger(ctx).WithOptions(zap.AddCallerSkip(1)).Info(message, fields...)
}

// Error writes an error through the global logger.
func Error(message string, fields ...zap.Field) {
	zap.L().WithOptions(zap.AddCallerSkip(1)).Error(message, fields...)
}

// RequestFilter adds a request ID to the response and logs one structured access record.
func RequestFilter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if !validRequestID(requestID) {
			requestID = newRequestID()
		}
		w.Header().Set("X-Request-ID", requestID)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		ctx := WithFields(r.Context(), zap.String("request_id", requestID))
		r = r.WithContext(ctx)
		started := time.Now()
		tracked := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(tracked, r)
		status := tracked.status
		if status == 0 {
			status = http.StatusOK
		}

		fields := []zap.Field{
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", status),
			zap.Duration("duration", time.Since(started)),
		}
		switch {
		case status >= http.StatusInternalServerError:
			ErrorContext(r.Context(), "request failed", fields...)
		case status >= http.StatusBadRequest:
			Logger(r.Context()).WithOptions(zap.AddCallerSkip(1)).Warn("request completed", fields...)
		default:
			InfoContext(r.Context(), "request completed", fields...)
		}
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *statusWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		if w.status == 0 {
			w.WriteHeader(http.StatusOK)
		}
		flusher.Flush()
	}
}

func validRequestID(value string) bool {
	if len(value) == 0 || len(value) > 128 {
		return false
	}
	for i := 0; i < len(value); i++ {
		if value[i] < 0x21 || value[i] > 0x7e {
			return false
		}
	}
	return true
}

func newRequestID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(value[:])
}

// KratosLogger adapts the shared Zap logger to Kratos' structured logger API.
func KratosLogger() kratoslog.Logger {
	return kratosAdapter{}
}

type kratosAdapter struct{}

func (kratosAdapter) Log(level kratoslog.Level, keyvals ...any) error {
	var message string
	fields := make([]zap.Field, 0, len(keyvals)/2)
	for i := 0; i+1 < len(keyvals); i += 2 {
		key, ok := keyvals[i].(string)
		if !ok {
			continue
		}
		value := keyvals[i+1]
		if key == "msg" {
			if msg, ok := value.(string); ok {
				message = msg
			}
			continue
		}
		fields = append(fields, zap.Any(key, value))
	}
	if message == "" {
		message = "framework log"
	}
	logger := zap.L().WithOptions(zap.AddCallerSkip(1))
	switch level {
	case kratoslog.LevelDebug:
		logger.Debug(message, fields...)
	case kratoslog.LevelInfo:
		logger.Info(message, fields...)
	case kratoslog.LevelWarn:
		logger.Warn(message, fields...)
	case kratoslog.LevelError:
		logger.Error(message, fields...)
	default:
		logger.Info(message, fields...)
	}
	return nil
}

func makeCore(format string, cfg zapcore.EncoderConfig, writer io.Writer, level zapcore.Level) zapcore.Core {
	var encoder zapcore.Encoder
	if format == "json" {
		encoder = zapcore.NewJSONEncoder(cfg)
	} else {
		encoder = zapcore.NewConsoleEncoder(cfg)
	}
	return zapcore.NewCore(encoder, zapcore.AddSync(writer), level)
}

func fileWriter(file File) (io.Writer, error) {
	dir := filepath.Dir(file.Path)
	if dir != "." && dir != "/" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	if file.Rotation.Enabled {
		return &lumberjack.Logger{
			Filename:   file.Path,
			MaxSize:    file.Rotation.MaxSize,
			MaxAge:     file.Rotation.MaxAge,
			MaxBackups: file.Rotation.MaxBackups,
			Compress:   true,
		}, nil
	}
	return os.OpenFile(file.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
}

func parseLevel(level string) (zapcore.Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn", "warning":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("logging level %q must be debug, info, warn, or error", level)
	}
}
