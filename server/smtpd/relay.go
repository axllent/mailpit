package smtpd

import (
	"strings"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
)

func autoRelayMessage(from string, to []string, data *[]byte) {
	if len(to) == 0 {
		return
	}

	if config.SMTPRelayAll {
		if err := Send(from, to, *data); err != nil {
			logger.Log().Errorf("[smtp] error relaying message: %s", err.Error())
		} else {
			logger.Log().Debugf("[smtp] auto-relay message to %s from %s via %s:%d",
				strings.Join(to, ", "), from, config.SMTPRelayConfig.Host, config.SMTPRelayConfig.Port)
		}
	} else if config.SMTPRelayMatchingRegexp != nil {
		filtered := []string{}
		for _, t := range to {
			if config.SMTPRelayMatchingRegexp.MatchString(t) {
				filtered = append(filtered, t)
			}
		}

		if len(filtered) == 0 {
			return
		}

		if err := Send(from, filtered, *data); err != nil {
			logger.Log().Errorf("[smtp] error relaying message: %s", err.Error())
		} else {
			logger.Log().Debugf("[smtp] auto-relay message to %s from %s via %s:%d",
				strings.Join(filtered, ", "), from, config.SMTPRelayConfig.Host, config.SMTPRelayConfig.Port)
		}
	}
}
