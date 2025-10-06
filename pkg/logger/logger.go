package logger

import (
	"log"
	"os"
	"sync"
)

// Level represents the log level
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

// Logger represents a simple logger
type Logger struct {
	mu    sync.Mutex
	level Level
	debug *log.Logger
	info  *log.Logger
	warn  *log.Logger
	error *log.Logger
	fatal *log.Logger
}

var (
	instance *Logger
	once     sync.Once
)

// Get returns the singleton logger instance
func Get() *Logger {
	once.Do(func() {
		instance = &Logger{
			level: INFO,
			debug: log.New(os.Stdout, "[DEBUG] ", log.LstdFlags|log.Lshortfile),
			info:  log.New(os.Stdout, "[INFO] ", log.LstdFlags),
			warn:  log.New(os.Stdout, "[WARN] ", log.LstdFlags),
			error: log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.Lshortfile),
			fatal: log.New(os.Stderr, "[FATAL] ", log.LstdFlags|log.Lshortfile),
		}
	})
	return instance
}

// SetLevel sets the logging level
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// Debug logs a debug message
func (l *Logger) Debug(v ...interface{}) {
	if l.level <= DEBUG {
		l.debug.Println(v...)
	}
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, v ...interface{}) {
	if l.level <= DEBUG {
		l.debug.Printf(format, v...)
	}
}

// Info logs an info message
func (l *Logger) Info(v ...interface{}) {
	if l.level <= INFO {
		l.info.Println(v...)
	}
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, v ...interface{}) {
	if l.level <= INFO {
		l.info.Printf(format, v...)
	}
}

// Warn logs a warning message
func (l *Logger) Warn(v ...interface{}) {
	if l.level <= WARN {
		l.warn.Println(v...)
	}
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, v ...interface{}) {
	if l.level <= WARN {
		l.warn.Printf(format, v...)
	}
}

// Error logs an error message
func (l *Logger) Error(v ...interface{}) {
	if l.level <= ERROR {
		l.error.Println(v...)
	}
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, v ...interface{}) {
	if l.level <= ERROR {
		l.error.Printf(format, v...)
	}
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(v ...interface{}) {
	l.fatal.Println(v...)
	os.Exit(1)
}

// Fatalf logs a formatted fatal message and exits
func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.fatal.Printf(format, v...)
	os.Exit(1)
}
