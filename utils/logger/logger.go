// Package logger handles the logging
package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/axllent/mailpit/config"
	"github.com/sirupsen/logrus"
)

var (
	log *logrus.Logger
)

// Log returns the logger instance
func Log() *logrus.Logger {
	if log == nil {
		log = logrus.New()
		log.SetLevel(logrus.InfoLevel)
		if config.VerboseLogging {
			// verbose logging (debug)
			log.SetLevel(logrus.DebugLevel)
		} else if config.QuietLogging {
			// show errors only
			log.SetLevel(logrus.ErrorLevel)
		} else if config.NoLogging {
			// disable all logging (tests)
			log.SetLevel(logrus.PanicLevel)
		}

		log.Out = os.Stdout
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006/01/02 15:04:05",
			ForceColors:     true,
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
// when starting services. It translates [::]:<port> to "localhost:<port>"
func CleanIP(s string) string {
	re := regexp.MustCompile(`^\[\:\:\]\:\d+`)
	if re.MatchString(s) {
		return "0.0.0.0:" + s[5:]
	}

	return s
}
