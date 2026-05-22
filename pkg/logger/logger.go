package logger

import (
	"log"
	"os"
)

// Logger wraps the standard logger with level support
type Logger struct {
	info  *log.Logger
	warn  *log.Logger
	error *log.Logger
	debug *log.Logger
}

var defaultLogger *Logger

func init() {
	defaultLogger = NewLogger(os.Stdout, "[UBAX-Pilot] ")
}

// NewLogger creates a new Logger instance
func NewLogger(prefix string) *Logger {
	return NewLogger(os.Stdout, prefix)
}

// NewLogger creates a logger writing to the given writer
func NewLogger(w *os.File, prefix string) *Logger {
	return &Logger{
		info:  log.New(w, prefix+"INFO:  ", log.Ldate|log.Ltime|log.Lshortfile),
		warn:  log.New(w, prefix+"WARN:  ", log.Ldate|log.Ltime|log.Lshortfile),
		error: log.New(w, prefix+"ERROR: ", log.Ldate|log.Ltime|log.Lshortfile),
		debug: log.New(w, prefix+"DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

// Info logs an informational message
func Info(v ...interface{}) {
	defaultLogger.info.Println(v...)
}

// Warn logs a warning message
func Warn(v ...interface{}) {
	defaultLogger.warn.Println(v...)
}

// Error logs an error message
func Error(v ...interface{}) {
	defaultLogger.error.Println(v...)
}

// Debug logs a debug message
func Debug(v ...interface{}) {
	defaultLogger.debug.Println(v...)
}

// GetLogger returns the default logger instance
func GetLogger() *Logger {
	return defaultLogger
}
