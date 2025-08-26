package logging

import (
	"log/slog"

	"github.com/toyz/axon/examples/simple-app/internal/config"
)

// axon::logger
type AppLogger struct {
	//axon::inject
	Config *config.Config
	//axon::init
	logger *slog.Logger
}

// Info logs an info message
func (l *AppLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

// Error logs an error message
func (l *AppLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

// Debug logs a debug message
func (l *AppLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

// Warn logs a warning message
func (l *AppLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

// Logger returns the underlying slog.Logger for advanced usage
func (l *AppLogger) Logger() *slog.Logger {
	return l.logger
}
