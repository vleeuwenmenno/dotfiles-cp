package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var globalLogger zerolog.Logger

// Init initializes the global logger with the specified configuration
func Init(verbose, quiet bool) {
	var level zerolog.Level
	var output io.Writer

	// Set log level based on flags
	switch {
	case quiet:
		level = zerolog.ErrorLevel
	case verbose:
		level = zerolog.DebugLevel
	default:
		level = zerolog.InfoLevel
	}

	// Configure output with colors for console
	output = zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
		NoColor:    false,
	}

	// Create the global logger
	globalLogger = zerolog.New(output).
		Level(level).
		With().
		Timestamp().
		Logger()

	// Set the global logger
	log.Logger = globalLogger
}

// Get returns the global logger instance
func Get() *zerolog.Logger {
	return &globalLogger
}

// Debug creates a debug level log event
func Debug() *zerolog.Event {
	return globalLogger.Debug()
}

// Info creates an info level log event
func Info() *zerolog.Event {
	return globalLogger.Info()
}

// Warn creates a warn level log event
func Warn() *zerolog.Event {
	return globalLogger.Warn()
}

// Error creates an error level log event
func Error() *zerolog.Event {
	return globalLogger.Error()
}

// Fatal creates a fatal level log event
func Fatal() *zerolog.Event {
	return globalLogger.Fatal()
}

// WithField returns a logger with the specified field
func WithField(key string, value interface{}) zerolog.Logger {
	return globalLogger.With().Interface(key, value).Logger()
}

// WithFields returns a logger with the specified fields
func WithFields(fields map[string]interface{}) zerolog.Logger {
	ctx := globalLogger.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return ctx.Logger()
}
