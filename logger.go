package p2p

import (
	"fmt"
	"log/slog"
)

// SlogLogger implements the p2p-message-bus logger interface using slog.
// Log level filtering is handled by slog's handler configuration.
type SlogLogger struct {
	logger *slog.Logger
}

// NewSlogLogger creates a new SlogLogger with the given slog.Logger.
// If logger is nil, uses slog.Default().
func NewSlogLogger(logger *slog.Logger) *SlogLogger {
	if logger == nil {
		logger = slog.Default()
	}
	return &SlogLogger{logger: logger}
}

// Debugf logs a debug message.
func (l *SlogLogger) Debugf(format string, v ...any) {
	l.logger.Debug(fmt.Sprintf(format, v...))
}

// Infof logs an info message.
func (l *SlogLogger) Infof(format string, v ...any) {
	l.logger.Info(fmt.Sprintf(format, v...))
}

// Warnf logs a warning message.
func (l *SlogLogger) Warnf(format string, v ...any) {
	l.logger.Warn(fmt.Sprintf(format, v...))
}

// Errorf logs an error message.
func (l *SlogLogger) Errorf(format string, v ...any) {
	l.logger.Error(fmt.Sprintf(format, v...))
}
