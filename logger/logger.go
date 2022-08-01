package logger

import (
	"encoding/json"
	"fmt"
	"os"

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
			log.SetLevel(logrus.DebugLevel)
		}
		if config.NoLogging {
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
