package config

import (
	"errors"
	"fmt"
	"math"
	"net/mail"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/smtpd/chaos"
	"github.com/goccy/go-yaml"
)

// Parse the --max-age value (if set)
func parseMaxAge() error {
	if MaxAge == "" {
		return nil
	}

	re := regexp.MustCompile(`^\d+(h|d)$`)
	if !re.MatchString(MaxAge) {
		return fmt.Errorf("max-age must be either <int>h for hours or <int>d for days: %s", MaxAge)
	}

	// the largest age that can be represented as a time.Duration (about 292 years). Larger
	// values overflow when the prune cutoff is calculated, which can put the cutoff in the
	// future and delete every message.
	const maxHours = math.MaxInt64 / int64(time.Hour)
	tooLarge := fmt.Errorf("max-age is too large: %s (maximum is %dh or %dd)", MaxAge, maxHours, maxHours/24)

	if before, ok := strings.CutSuffix(MaxAge, "h"); ok {
		hours, err := strconv.Atoi(before)
		if err != nil {
			return tooLarge // the regexp allows only digits, so this can only be out of range
		}
		if int64(hours) > maxHours {
			return tooLarge
		}

		MaxAgeInHours = hours

		return nil
	}

	days, err := strconv.Atoi(strings.TrimSuffix(MaxAge, "d"))
	if err != nil {
		return tooLarge // the regexp allows only digits, so this can only be out of range
	}
	// checked before multiplying, so days * 24 cannot overflow
	if int64(days) > maxHours/24 {
		return tooLarge
	}

	logger.Log().Debugf("[db] auto-deleting messages older than %s", MaxAge)

	MaxAgeInHours = days * 24
	return nil
}

// Parse the SMTPRelayConfigFile (if set)
func parseRelayConfig(c string) error {
	if c == "" {
		return nil
	}

	c = filepath.Clean(c)

	if !isFile(c) {
		return fmt.Errorf("[relay] configuration not found or readable: %s", c)
	}

	data, err := os.ReadFile(c)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, &SMTPRelayConfig); err != nil {
		return err
	}

	// Removed keys are refused rather than ignored, as the YAML parser silently skips keys it does not know, which
	// for recipient-allowlist would lift the restriction on who messages can be released to.
	var keys map[string]any
	if err := yaml.Unmarshal(data, &keys); err != nil {
		return err
	}

	if _, ok := keys["recipient-allowlist"]; ok {
		return errors.New("[relay] 'recipient-allowlist' is no longer supported, use 'allowed-recipients' instead")
	}

	if SMTPRelayConfig.Host == "" {
		return errors.New("[relay] host not set")
	}

	return nil
}

// Validate the SMTPRelayConfig (if Host is set)
func validateRelayConfig() error {
	if SMTPRelayConfig.Host == "" {
		return nil
	}

	if SMTPRelayConfig.Port == 0 {
		SMTPRelayConfig.Port = 25 // default
	}
	if SMTPRelayConfig.Port < 1 || SMTPRelayConfig.Port > 65535 {
		return fmt.Errorf("[relay] port must be between 1 and 65535: %d", SMTPRelayConfig.Port)
	}

	SMTPRelayConfig.Auth = strings.ToLower(SMTPRelayConfig.Auth)

	if SMTPRelayConfig.Auth == "" || SMTPRelayConfig.Auth == "none" || SMTPRelayConfig.Auth == "false" {
		SMTPRelayConfig.Auth = "none"
	} else if SMTPRelayConfig.Auth == "plain" {
		if SMTPRelayConfig.Username == "" || SMTPRelayConfig.Password == "" {
			return fmt.Errorf("[relay] host username or password not set for PLAIN authentication")
		}
	} else if SMTPRelayConfig.Auth == "login" {
		SMTPRelayConfig.Auth = "login"
		if SMTPRelayConfig.Username == "" || SMTPRelayConfig.Password == "" {
			return fmt.Errorf("[relay] host username or password not set for LOGIN authentication")
		}
	} else if strings.HasPrefix(SMTPRelayConfig.Auth, "cram") {
		SMTPRelayConfig.Auth = "cram-md5"
		if SMTPRelayConfig.Username == "" || SMTPRelayConfig.Secret == "" {
			return fmt.Errorf("[relay] host username or secret not set for CRAM-MD5 authentication")
		}
	} else {
		return fmt.Errorf("[relay] authentication method not supported: %s", SMTPRelayConfig.Auth)
	}

	if SMTPRelayConfig.AllowedRecipients != "" {
		re, err := regexp.Compile(SMTPRelayConfig.AllowedRecipients)
		if err != nil {
			return fmt.Errorf("[relay] failed to compile recipient allowlist regexp: %s", err.Error())
		}

		SMTPRelayConfig.AllowedRecipientsRegexp = re
		logger.Log().Infof("[relay] recipient allowlist is active with the following regexp: %s", SMTPRelayConfig.AllowedRecipients)
	}

	if SMTPRelayConfig.BlockedRecipients != "" {
		re, err := regexp.Compile(SMTPRelayConfig.BlockedRecipients)
		if err != nil {
			return fmt.Errorf("[relay] failed to compile recipient blocklist regexp: %s", err.Error())
		}

		SMTPRelayConfig.BlockedRecipientsRegexp = re
		logger.Log().Infof("[relay] recipient blocklist is active with the following regexp: %s", SMTPRelayConfig.BlockedRecipients)
	}

	if SMTPRelayConfig.OverrideFrom != "" {
		m, err := mail.ParseAddress(SMTPRelayConfig.OverrideFrom)
		if err != nil {
			return fmt.Errorf("[relay] override-from is not a valid email address: %s", SMTPRelayConfig.OverrideFrom)
		}

		SMTPRelayConfig.OverrideFrom = m.Address
	}

	if SMTPRelayConfig.ReturnPath != "" {
		m, err := mail.ParseAddress(SMTPRelayConfig.ReturnPath)
		if err != nil {
			return fmt.Errorf("[relay] return-path is not a valid email address: %s", SMTPRelayConfig.ReturnPath)
		}

		SMTPRelayConfig.ReturnPath = m.Address
	}

	SMTPRelayConfig.SubjectPrefix = strings.TrimSpace(SMTPRelayConfig.SubjectPrefix)
	if strings.ContainsAny(SMTPRelayConfig.SubjectPrefix, "\r\n") {
		return fmt.Errorf("[relay] subject-prefix must not contain line breaks")
	}

	if SMTPRelayConfig.STARTTLS && SMTPRelayConfig.TLS {
		return fmt.Errorf("[relay] TLS & STARTTLS cannot be required together")
	}

	ReleaseEnabled = true

	logger.Log().Infof("[relay] enabling message relaying via %s:%d", SMTPRelayConfig.Host, SMTPRelayConfig.Port)

	return nil
}

// Parse the SMTPForwardConfigFile (if set)
func parseForwardConfig(c string) error {
	if c == "" {
		return nil
	}

	c = filepath.Clean(c)

	if !isFile(c) {
		return fmt.Errorf("[forward] configuration not found or readable: %s", c)
	}

	data, err := os.ReadFile(c)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, &SMTPForwardConfig); err != nil {
		return err
	}

	if SMTPForwardConfig.Host == "" {
		return errors.New("[forward] host not set")
	}

	return nil
}

// Validate the SMTPForwardConfig (if Host is set)
func validateForwardConfig() error {
	if SMTPForwardConfig.Host == "" {
		return nil
	}

	if SMTPForwardConfig.Port == 0 {
		SMTPForwardConfig.Port = 25 // default
	}
	if SMTPForwardConfig.Port < 1 || SMTPForwardConfig.Port > 65535 {
		return fmt.Errorf("[forward] port must be between 1 and 65535: %d", SMTPForwardConfig.Port)
	}

	SMTPForwardConfig.Auth = strings.ToLower(SMTPForwardConfig.Auth)

	if SMTPForwardConfig.Auth == "" || SMTPForwardConfig.Auth == "none" || SMTPForwardConfig.Auth == "false" {
		SMTPForwardConfig.Auth = "none"
	} else if SMTPForwardConfig.Auth == "plain" {
		if SMTPForwardConfig.Username == "" || SMTPForwardConfig.Password == "" {
			return fmt.Errorf("[forward] host username or password not set for PLAIN authentication")
		}
	} else if SMTPForwardConfig.Auth == "login" {
		SMTPForwardConfig.Auth = "login"
		if SMTPForwardConfig.Username == "" || SMTPForwardConfig.Password == "" {
			return fmt.Errorf("[forward] host username or password not set for LOGIN authentication")
		}
	} else if strings.HasPrefix(SMTPForwardConfig.Auth, "cram") {
		SMTPForwardConfig.Auth = "cram-md5"
		if SMTPForwardConfig.Username == "" || SMTPForwardConfig.Secret == "" {
			return fmt.Errorf("[forward] host username or secret not set for CRAM-MD5 authentication")
		}
	} else {
		return fmt.Errorf("[forward] authentication method not supported: %s", SMTPForwardConfig.Auth)
	}

	if SMTPForwardConfig.To == "" {
		return errors.New("[forward] To addresses missing")
	}

	to := []string{}
	addresses := strings.SplitSeq(SMTPForwardConfig.To, ",")
	for a := range addresses {
		a = strings.TrimSpace(a)
		m, err := mail.ParseAddress(a)
		if err != nil {
			return fmt.Errorf("[forward] To address is not a valid email address: %s", a)
		}
		to = append(to, m.Address)
	}

	if len(to) == 0 {
		return errors.New("[forward] no valid To addresses found")
	}

	// overwrite the To field with the cleaned up list
	SMTPForwardConfig.To = strings.Join(to, ",")

	if SMTPForwardConfig.OverrideFrom != "" {
		m, err := mail.ParseAddress(SMTPForwardConfig.OverrideFrom)
		if err != nil {
			return fmt.Errorf("[forward] override-from is not a valid email address: %s", SMTPForwardConfig.OverrideFrom)
		}

		SMTPForwardConfig.OverrideFrom = m.Address
	}

	if SMTPForwardConfig.ReturnPath != "" {
		m, err := mail.ParseAddress(SMTPForwardConfig.ReturnPath)
		if err != nil {
			return fmt.Errorf("[forward] return-path is not a valid email address: %s", SMTPForwardConfig.ReturnPath)
		}

		SMTPForwardConfig.ReturnPath = m.Address
	}

	SMTPForwardConfig.SubjectPrefix = strings.TrimSpace(SMTPForwardConfig.SubjectPrefix)
	if strings.ContainsAny(SMTPForwardConfig.SubjectPrefix, "\r\n") {
		return fmt.Errorf("[forward] subject-prefix must not contain line breaks")
	}

	if SMTPForwardConfig.STARTTLS && SMTPForwardConfig.TLS {
		return fmt.Errorf("[forward] TLS & STARTTLS cannot be required together")
	}

	logger.Log().Infof("[forward] enabling message forwarding to %s via %s:%d", SMTPForwardConfig.To, SMTPForwardConfig.Host, SMTPForwardConfig.Port)

	return nil
}

func parseChaosTriggers() error {
	if ChaosTriggers == "" {
		return nil
	}

	re := regexp.MustCompile(`^([a-zA-Z0-9]+):(\d\d\d):(\d+)$`)

	parts := strings.SplitSeq(ChaosTriggers, ",")
	for p := range parts {
		p = strings.TrimSpace(p)
		if !re.MatchString(p) {
			return fmt.Errorf("invalid argument: %s", p)
		}

		matches := re.FindAllStringSubmatch(p, 1)
		key := matches[0][1]
		errorCode, err := strconv.Atoi(matches[0][2])
		if err != nil {
			return err
		}
		probability, err := strconv.Atoi(matches[0][3])
		if err != nil {
			return err
		}

		if err := chaos.Set(key, errorCode, probability); err != nil {
			return err
		}
	}

	return nil
}
