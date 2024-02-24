// Package logger handles the logging
package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/sirupsen/logrus"
)

var (
	log *logrus.Logger
	// VerboseLogging for verbose logging
	VerboseLogging bool
	// QuietLogging shows only errors
	QuietLogging bool
	// NoLogging shows only fatal errors
	NoLogging bool
	// LogFile sets a log file
	LogFile string
)

// Log returns the logger instance
func Log() *logrus.Logger {
	if log == nil {
		log = logrus.New()
		log.SetLevel(logrus.InfoLevel)
		if VerboseLogging {
			// verbose logging (debug)
			log.SetLevel(logrus.DebugLevel)
		} else if QuietLogging {
			// show errors only
			log.SetLevel(logrus.ErrorLevel)
		} else if NoLogging {
			// disable all logging (tests)
			log.SetLevel(logrus.PanicLevel)
		}

		if LogFile != "" {
			file, err := os.OpenFile(filepath.Clean(LogFile), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0664) // #nosec
			if err == nil {
				log.Out = file
			} else {
				log.Out = os.Stdout
				log.Warn("Failed to log to file, using default stderr")
			}
		} else {
			log.Out = os.Stdout
		}

		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006/01/02 15:04:05",
		})
	}

	return log
}

// PrettyPrint for debugging
func PrettyPrint(i interface{}) {
	s, _ := json.MarshalIndent(i, "", "\t")
	fmt.Println(string(s))
}

// CleanIP returns a human-readable IP for the logging interface
// when starting services. It translates [::]:<port> to "0.0.0.0:<port>"
func CleanIP(s string) string {
	re := regexp.MustCompile(`^\[\:\:\]\:\d+`)
	if re.MatchString(s) {
		return "0.0.0.0:" + s[5:]
	}

	return s
}

// CleanHTTPIP returns a human-readable IP for the logging interface
// when starting services. It translates [::]:<port> to "localhost:<port>"
func CleanHTTPIP(s string) string {
	re := regexp.MustCompile(`^\[\:\:\]\:\d+`)
	if re.MatchString(s) {
		return "localhost:" + s[5:]
	}

	return s
}
