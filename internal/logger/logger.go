// Package logger handles the logging
// Mailpit now uses slog for logging, but this package provides a logrus-compatible API and formatting to avoid changing all existing log calls
// and provide backwards compatibility with logrus formatting and features like log levels and file output.
package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sync"
)

// Logger wraps slog.Logger providing a logrus-compatible API
type Logger struct {
	sl *slog.Logger
}

var (
	log *Logger
	// VerboseLogging for verbose logging
	VerboseLogging bool
	// QuietLogging shows only errors
	QuietLogging bool
	// NoLogging disables all logging (tests)
	NoLogging bool
	// LogFile sets a log file
	LogFile string
)

// Log returns the logger instance, initialising it on first call. The level and
// output destination are determined once from VerboseLogging, QuietLogging,
// NoLogging, and LogFile at the time of first use.
func Log() *Logger {
	if log == nil {
		level := slog.LevelInfo
		switch {
		case VerboseLogging:
			level = slog.LevelDebug
		case QuietLogging:
			level = slog.LevelError
		case NoLogging:
			level = slog.Level(100) // above all real levels — silences all output
		}

		out := os.Stdout
		if LogFile != "" {
			file, err := os.OpenFile(filepath.Clean(LogFile), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0664) // #nosec
			if err == nil {
				out = file
			} else {
				fmt.Fprintln(os.Stderr, "failed to log to file, using default stdout")
			}
		}

		log = &Logger{
			sl: slog.New(&logrusHandler{
				out:   out,
				level: level,
				color: isTerminal(out),
			}),
		}
	}

	return log
}

// logrusHandler is a slog.Handler that formats output to match logrus TextFormatter.
// TTY output:  INFO[2006/01/02 15:04:05] message
// File output: time="2006/01/02 15:04:05" level=info msg="message"
type logrusHandler struct {
	mu    sync.Mutex
	out   *os.File
	level slog.Level
	color bool
}

// Enabled reports whether the handler will emit a record at the given level.
func (h *logrusHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle formats and writes a log record. TTY output is coloured; file output
// uses the logrus key=value text format.
func (h *logrusHandler) Handle(_ context.Context, r slog.Record) error {
	label, name, code := logrusLevel(r.Level)
	ts := r.Time.Format("2006/01/02 15:04:05")

	var line string
	if h.color {
		line = fmt.Sprintf("\x1b[%dm%s\x1b[0m[%s] %s\n", code, label, ts, r.Message)
	} else {
		line = fmt.Sprintf("time=%q level=%s msg=%q\n", ts, name, r.Message)
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := fmt.Fprint(h.out, line)
	return err
}

// WithAttrs returns the handler unchanged; structured attributes are not used.
func (h *logrusHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }

// WithGroup returns the handler unchanged; groups are not used.
func (h *logrusHandler) WithGroup(_ string) slog.Handler { return h }

// logrusLevel maps slog levels to the 4-char TTY label, lowercase file label, and ANSI colour code.
func logrusLevel(level slog.Level) (string, string, int) {
	switch {
	case level < slog.LevelInfo:
		return "DEBU", "debug", 37 // gray
	case level < slog.LevelWarn:
		return "INFO", "info", 36 // cyan
	case level < slog.LevelError:
		return "WARN", "warning", 33 // yellow
	default:
		return "ERRO", "error", 31 // red
	}
}

// isTerminal reports whether f is connected to a terminal.
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// Info logs a message at INFO level.
func (l *Logger) Info(args ...any) { l.sl.Info(fmt.Sprint(args...)) }

// Infof logs a formatted message at INFO level.
func (l *Logger) Infof(format string, args ...any) { l.sl.Info(fmt.Sprintf(format, args...)) }

// Debug logs a message at DEBUG level.
func (l *Logger) Debug(args ...any) { l.sl.Debug(fmt.Sprint(args...)) }

// Debugf logs a formatted message at DEBUG level.
func (l *Logger) Debugf(format string, args ...any) { l.sl.Debug(fmt.Sprintf(format, args...)) }

// Warn logs a message at WARN level.
func (l *Logger) Warn(args ...any) { l.sl.Warn(fmt.Sprint(args...)) }

// Warnf logs a formatted message at WARN level.
func (l *Logger) Warnf(format string, args ...any) { l.sl.Warn(fmt.Sprintf(format, args...)) }

// Error logs a message at ERROR level.
func (l *Logger) Error(args ...any) { l.sl.Error(fmt.Sprint(args...)) }

// Errorf logs a formatted message at ERROR level.
func (l *Logger) Errorf(format string, args ...any) { l.sl.Error(fmt.Sprintf(format, args...)) }

// Printf logs a formatted message at INFO level.
func (l *Logger) Printf(format string, args ...any) { l.sl.Info(fmt.Sprintf(format, args...)) }

// Fatal logs a message at ERROR level then exits with status 1.
func (l *Logger) Fatal(args ...any) { l.sl.Error(fmt.Sprint(args...)); os.Exit(1) }

// Fatalf logs a formatted message at ERROR level then exits with status 1.
func (l *Logger) Fatalf(format string, args ...any) {
	l.sl.Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}

// PrettyPrint prints any value as indented JSON to stdout, for debugging.
func PrettyPrint(i any) {
	s, _ := json.MarshalIndent(i, "", "\t")
	fmt.Println(string(s))
}

// CleanHTTPIP returns a human-readable address for log output.
// It translates [::]:<port> to localhost:<port>.
func CleanHTTPIP(s string) string {
	re := regexp.MustCompile(`^\[\:\:\]\:\d+`)
	if re.MatchString(s) {
		return "localhost:" + s[5:]
	}

	return s
}
