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
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// Logger wraps slog.Logger providing a logrus-compatible API
type Logger struct {
	sl *slog.Logger
}

var (
	log     *Logger
	logOnce sync.Once
	// VerboseLogging for verbose logging
	VerboseLogging bool
	// QuietLogging shows only errors
	QuietLogging bool
	// NoLogging disables all logging (tests)
	NoLogging bool
	// LogFile sets a log file
	LogFile string
)

// Log returns the logger instance, initialising it on first call.
//
// The level (VerboseLogging, QuietLogging, NoLogging) and the output destination
// (LogFile) are read each time a message is logged, not when the logger is first
// created. Settings made after an early log message (for example command line flags
// parsed after a warning logged during initialisation) therefore still take effect.
// They should be set before messages are logged concurrently.
func Log() *Logger {
	logOnce.Do(func() {
		log = &Logger{
			sl: slog.New(&logrusHandler{
				out:   os.Stdout,
				color: isTerminal(os.Stdout),
			}),
		}
	})

	return log
}

// currentLevel returns the minimum level to log, from the current settings.
func currentLevel() slog.Level {
	switch {
	case VerboseLogging:
		return slog.LevelDebug
	case QuietLogging:
		return slog.LevelError
	case NoLogging:
		return slog.Level(100) // above all real levels, silences all output
	}

	return slog.LevelInfo
}

// logrusHandler is a slog.Handler that formats output to match logrus TextFormatter.
// TTY output:  INFO[2006/01/02 15:04:05] message
// File output: time="2006/01/02 15:04:05" level=info msg="message"
type logrusHandler struct {
	mu      sync.Mutex
	out     *os.File
	outName string // the LogFile that out was opened for ("" is stdout)
	color   bool
}

// Enabled reports whether the handler will emit a record at the given level.
func (h *logrusHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= currentLevel()
}

// Handle formats and writes a log record. TTY output is coloured; file output
// uses the logrus key=value text format.
//
// The message is never written as-is to a terminal: it may contain text from a remote
// client, and raw control characters could clear the screen, forge log lines or change
// the terminal title. File output is quoted, which escapes them already.
func (h *logrusHandler) Handle(_ context.Context, r slog.Record) error {
	label, name, code := logrusLevel(r.Level)
	ts := r.Time.Format("2006/01/02 15:04:05")

	h.mu.Lock()
	defer h.mu.Unlock()

	h.refreshOutputLocked()

	var line string
	if h.color {
		line = fmt.Sprintf("\x1b[%dm%s\x1b[0m[%s] %s\n", code, label, ts, sanitize(r.Message))
	} else {
		line = fmt.Sprintf("time=%q level=%s msg=%q\n", ts, name, r.Message)
	}

	_, err := fmt.Fprint(h.out, line)

	return err
}

// refreshOutput switches the output if LogFile has changed since it was last opened.
func (h *logrusHandler) refreshOutput() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.refreshOutputLocked()
}

// refreshOutputLocked switches the output if LogFile has changed since it was last
// opened. The caller must hold h.mu.
func (h *logrusHandler) refreshOutputLocked() {
	if LogFile == h.outName {
		return
	}

	// record this first so that a file which cannot be opened is not retried (and
	// reported) for every message
	h.outName = LogFile

	out := os.Stdout
	if LogFile != "" {
		file, err := os.OpenFile(filepath.Clean(LogFile), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0664) // #nosec
		if err == nil {
			out = file
		} else {
			fmt.Fprintln(os.Stderr, "failed to log to file, using default stdout")
		}
	}

	if h.out != nil && h.out != os.Stdout && h.out != out {
		_ = h.out.Close()
	}

	h.out = out
	h.color = isTerminal(out)
}

// sanitize makes s safe to write to a terminal. Anything that is not printable (control
// characters, DEL, C1 controls, line and paragraph separators, bidirectional and other
// format characters, non-ASCII spaces and invalid UTF-8) is replaced by an escape such as
// \x1b or \u202e, as Go quoting does. Printable text, including other languages, is unchanged.
func sanitize(s string) string {
	var b strings.Builder

	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])

		switch {
		case r == utf8.RuneError && size == 1:
			fmt.Fprintf(&b, `\x%02x`, s[i])
		case r == ' ' || unicode.IsPrint(r):
			b.WriteString(s[i : i+size])
		default:
			q := strconv.QuoteRune(r) // 'x', quoted with escapes for non-printable runes
			b.WriteString(q[1 : len(q)-1])
		}

		i += size
	}

	return b.String()
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
