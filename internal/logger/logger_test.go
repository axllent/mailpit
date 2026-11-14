package logger

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"
)

func TestSanitize(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		// controls: printable text is left exactly as it is
		{"control: plain", "hello world", "hello world"},
		{"control: punctuation and quotes", `a "quoted" 'text' [tag] {x}: 100%`, `a "quoted" 'text' [tag] {x}: 100%`},
		{"control: backslashes are not doubled", `C:\Users\x`, `C:\Users\x`},
		{"control: other languages", "héllo wörld 日本語 Привет", "héllo wörld 日本語 Привет"},
		{"control: emoji", "ok 😀 ✓", "ok 😀 ✓"},
		{"control: empty", "", ""},

		// exploits: anything that could act on a terminal or forge a log line is escaped
		{"escape sequence", "\x1b[2J\x1b[31mred", `\x1b[2J\x1b[31mred`},
		{"set window title", "\x1b]0;title\x07", `\x1b]0;title\a`},
		{"newline forges a line", "a\nINFO[2026] fake", `a\nINFO[2026] fake`},
		{"carriage return overwrites a line", "real\rfake", `real\rfake`},
		{"tab", "a\tb", `a\tb`},
		{"backspace", "abc\b\b\bxyz", `abc\b\b\bxyz`},
		{"nul", "a\x00b", `a\x00b`},
		{"delete", "a\x7fb", `a\x7fb`},
		{"C1 control (single byte CSI)", "a\u009bb", `a\u009bb`},
		{"right-to-left override", "a\u202eb", `a\u202eb`},
		{"line separator", "a\u2028b", `a\u2028b`},
		{"zero width space", "a\u200bb", `a\u200bb`},
		{"non-breaking space", "a\u00a0b", `a\u00a0b`},
		{"invalid UTF-8", "a\xffb\xc0", `a\xffb\xc0`},
		{"truncated multi-byte character", "a\xe2\x82", `a\xe2\x82`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitize(tt.in); got != tt.want {
				t.Fatalf("sanitize(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func FuzzSanitize(f *testing.F) {
	for _, s := range []string{"", "plain", "\x1b[2J", "a\nb", "\xff", "\u202e", "日本語", `C:\x`, "\x1b]0;x\x07", "a\u00a0b"} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		got := sanitize(s)

		if !utf8.ValidString(got) {
			t.Fatalf("invalid UTF-8 in output for %q: %q", s, got)
		}
		for _, r := range got {
			if r != ' ' && !unicode.IsPrint(r) {
				t.Fatalf("non-printable %U in output for %q: %q", r, s, got)
			}
		}
		if again := sanitize(got); again != got {
			t.Fatalf("not idempotent for %q: %q then %q", s, got, again)
		}
		if utf8.ValidString(s) && strings.IndexFunc(s, func(r rune) bool { return r != ' ' && !unicode.IsPrint(r) }) < 0 && got != s {
			t.Fatalf("printable input changed: %q became %q", s, got)
		}
	})
}

// handleToFile runs one record through a handler and returns what was written.
func handleToFile(t *testing.T, color bool, msg string) string {
	t.Helper()

	f, err := os.Create(filepath.Join(t.TempDir(), "out.log"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	// LogFile is "" and so is the handler's name for it, so the output is not switched
	h := &logrusHandler{out: f, color: color}
	r := slog.NewRecord(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), slog.LevelWarn, msg, 0)
	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}

	return string(b)
}

func TestTerminalOutputCannotBeInjected(t *testing.T) {
	const colourStart, colourEnd = "\x1b[33m", "\x1b[0m"

	// control: a normal message appears exactly as it did before
	got := handleToFile(t, true, `ignoring invalid before: date "yesterday-ish"`)
	if want := colourStart + "WARN" + colourEnd + `[2026/01/02 03:04:05] ignoring invalid before: date "yesterday-ish"` + "\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}

	// exploit: the same message shape with an escape sequence, a forged line and a title change
	got = handleToFile(t, true, "ignoring invalid before: date \"\x1b[2J\x1b[31mFORGED\nINFO[2026/01/01] fake line\x1b]0;PWNED\x07\"")

	body := strings.TrimSuffix(strings.TrimPrefix(got, colourStart+"WARN"+colourEnd), "\n")
	if strings.ContainsAny(body, "\x1b\n\r\a") {
		t.Fatalf("raw control characters reached the terminal: %q", got)
	}
	if strings.Count(got, "\n") != 1 || !strings.HasSuffix(got, "\n") {
		t.Fatalf("output is not a single line: %q", got)
	}
	if !strings.Contains(got, `\x1b[2J`) || !strings.Contains(got, `\nINFO[2026/01/01] fake line`) {
		t.Fatalf("expected the characters to be escaped, got %q", got)
	}
}

func TestFileOutputIsQuoted(t *testing.T) {
	got := handleToFile(t, false, "a \x1b[2J\nb")
	if want := `time="2026/01/02 03:04:05" level=warning msg="a \x1b[2J\nb"` + "\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// setLogging applies logging settings for a test and restores them (and closes any
// log file that was opened) afterwards.
func setLogging(t *testing.T, logFile string, verbose, quiet, none bool) {
	t.Helper()

	oFile, oVerbose, oQuiet, oNone := LogFile, VerboseLogging, QuietLogging, NoLogging
	t.Cleanup(func() {
		LogFile, VerboseLogging, QuietLogging, NoLogging = oFile, oVerbose, oQuiet, oNone
		Log().sl.Handler().(*logrusHandler).refreshOutput() // closes the test's log file
	})

	LogFile, VerboseLogging, QuietLogging, NoLogging = logFile, verbose, quiet, none
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	return string(b)
}

// Settings made after the logger has already been used (as command line flags are, when
// something logs during initialisation) must still take effect.
func TestSettingsAppliedAfterFirstUse(t *testing.T) {
	// the logger exists and has been used with the default settings
	setLogging(t, "", false, false, false)
	Log().Debug("early message")

	tests := []struct {
		name                 string
		verbose, quiet, none bool
		wantDebug, wantInfo  bool
		wantError            bool
	}{
		{"default level", false, false, false, false, true, true},
		{"verbose shows debug", true, false, false, true, true, true},
		{"quiet shows only errors", false, true, false, false, false, true},
		{"none shows nothing", false, false, true, false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "mailpit.log")
			setLogging(t, path, tt.verbose, tt.quiet, tt.none)

			Log().Debug("debug-message")
			Log().Info("info-message")
			Log().Error("error-message")

			got := readFile(t, path)
			for _, c := range []struct {
				msg  string
				want bool
			}{{"debug-message", tt.wantDebug}, {"info-message", tt.wantInfo}, {"error-message", tt.wantError}} {
				if strings.Contains(got, c.msg) != c.want {
					t.Errorf("%s logged = %v, want %v (output %q)", c.msg, !c.want, c.want, got)
				}
			}
		})
	}
}

func TestLogFileSwitchesAndAppends(t *testing.T) {
	dir := t.TempDir()
	first, second := filepath.Join(dir, "first.log"), filepath.Join(dir, "second.log")

	setLogging(t, first, false, false, false)
	Log().Info("one")

	// switching the file moves later messages and closes the first
	LogFile = second
	Log().Info("two")

	// switching back appends to the first file again
	LogFile = first
	Log().Info("three")

	if got := readFile(t, first); !strings.Contains(got, "one") || !strings.Contains(got, "three") || strings.Contains(got, "two") {
		t.Fatalf("first.log = %q", got)
	}
	if got := readFile(t, second); !strings.Contains(got, "two") || strings.Contains(got, "one") {
		t.Fatalf("second.log = %q", got)
	}
}

func TestUnwritableLogFileFallsBack(t *testing.T) {
	// a directory that does not exist, so the file cannot be created
	setLogging(t, filepath.Join(t.TempDir(), "missing", "dir", "x.log"), false, false, false)

	h := Log().sl.Handler().(*logrusHandler)
	h.refreshOutput()

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.out != os.Stdout {
		t.Fatal("expected to fall back to stdout")
	}
}

func TestCleanHTTPIP(t *testing.T) {
	tests := []struct{ in, want string }{
		{"[::]:8025", "localhost:8025"},
		{"0.0.0.0:8025", "0.0.0.0:8025"},
		{"127.0.0.1:8025", "127.0.0.1:8025"},
		{"[::1]:8025", "[::1]:8025"},
		{"[::]:", "[::]:"},
		{"", ""},
	}

	for _, tt := range tests {
		if got := CleanHTTPIP(tt.in); got != tt.want {
			t.Errorf("CleanHTTPIP(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
