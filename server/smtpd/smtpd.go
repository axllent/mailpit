package smtpd

import (
	"bytes"
	"net"
	"net/mail"
	"regexp"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/storage"
	"github.com/axllent/mailpit/utils/logger"
	"github.com/mhale/smtpd"
)

func mailHandler(origin net.Addr, from string, to []string, data []byte) error {
	msg, err := mail.ReadMessage(bytes.NewReader(data))
	if err != nil {
		logger.Log().Errorf("error parsing message: %s", err.Error())
		return err
	}

	if _, err := storage.Store(data); err != nil {
		// Value with size 4800709 exceeded 1048576 limit
		re := regexp.MustCompile(`(Value with size \d+ exceeded \d+ limit)`)
		tooLarge := re.FindStringSubmatch(err.Error())
		if len(tooLarge) > 0 {
			logger.Log().Errorf("[db] error storing message: %s", tooLarge[0])
		} else {
			logger.Log().Errorf("[db] error storing message")
			logger.Log().Errorf(err.Error())
		}
		return err
	}

	subject := msg.Header.Get("Subject")
	logger.Log().Debugf("[smtp] received mail from %s for %s with subject %s", from, to[0], subject)
	return nil
}

func authHandler(remoteAddr net.Addr, mechanism string, username []byte, password []byte, shared []byte) (bool, error) {
	return config.SMTPAuth.Match(string(username), string(password)), nil
}

func authHandlerAny(remoteAddr net.Addr, mechanism string, username []byte, password []byte, shared []byte) (bool, error) {
	// Allow any username and password
	logger.Log().Debugf("[smtp] Allow login with username %q and password %q", string(username), string(password))
	return true, nil
}

// Listen starts the SMTPD server
func Listen() error {
	if config.SMTPSSLCert != "" {
		logger.Log().Info("[smtp] enabling TLS")
	}
	if config.SMTPAuthFile != "" {
		logger.Log().Info("[smtp] enabling authentication")
	}

	logger.Log().Infof("[smtp] starting on %s", config.SMTPListen)

	return listenAndServe(config.SMTPListen, mailHandler, authHandler)
}

func listenAndServe(addr string, handler smtpd.Handler, authHandler smtpd.AuthHandler) error {
	srv := &smtpd.Server{
		Addr:         addr,
		Handler:      handler,
		Appname:      "Mailpit",
		Hostname:     "",
		AuthHandler:  nil,
		AuthRequired: false,
	}

	if config.AllowLoginsInsecure {
		srv.AuthMechs = map[string]bool{"LOGIN": true, "PLAIN": true}
	}

	if config.SMTPAuthFile != "" {
		srv.AuthHandler = authHandler
		srv.AuthRequired = true
	} else if config.AllowLoginsAny {
		srv.AuthHandler = authHandlerAny
	}

	if config.SMTPSSLCert != "" {
		err := srv.ConfigureTLS(config.SMTPSSLCert, config.SMTPSSLKey)
		if err != nil {
			return err
		}
	}

	return srv.ListenAndServe()
}
