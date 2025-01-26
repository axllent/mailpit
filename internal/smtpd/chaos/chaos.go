// Package chaos is used to simulate Chaos engineering (random failures) in the SMTPD server.
// See https://en.wikipedia.org/wiki/Chaos_engineering
// See https://mailpit.axllent.org/docs/integration/chaos/
package chaos

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	"github.com/axllent/mailpit/internal/logger"
)

var (
	// Enabled is a flag to enable or disable support for chaos
	Enabled = false

	// Config is the global Chaos configuration
	Config = Triggers{
		Sender:         Trigger{ErrorCode: 451, Probability: 0},
		Recipient:      Trigger{ErrorCode: 451, Probability: 0},
		Authentication: Trigger{ErrorCode: 535, Probability: 0},
	}
)

// Triggers for the Chaos configuration
//
// swagger:model Triggers
type Triggers struct {
	// Sender trigger to fail on From, Sender
	Sender Trigger
	// Recipient trigger to fail on To, Cc, Bcc
	Recipient Trigger
	// Authentication trigger to fail while authenticating (auth must be configured)
	Authentication Trigger
}

// Trigger for Chaos
type Trigger struct {
	// SMTP error code to return. The value must range from 400 to 599.
	// required: true
	// example: 451
	ErrorCode int

	// Probability (chance) of triggering the error. The value must range from 0 to 100.
	// required: true
	// example: 5
	Probability int
}

// SetFromStruct will set a whole map of chaos configurations (ie: API)
func SetFromStruct(c Triggers) error {
	if c.Sender.ErrorCode == 0 {
		c.Sender.ErrorCode = 451 // default
	}

	if c.Recipient.ErrorCode == 0 {
		c.Recipient.ErrorCode = 451 // default
	}

	if c.Authentication.ErrorCode == 0 {
		c.Authentication.ErrorCode = 535 // default
	}

	if err := Set("Sender", c.Sender.ErrorCode, c.Sender.Probability); err != nil {
		return err
	}
	if err := Set("Recipient", c.Recipient.ErrorCode, c.Recipient.Probability); err != nil {
		return err
	}
	if err := Set("Authentication", c.Authentication.ErrorCode, c.Authentication.Probability); err != nil {
		return err
	}

	return nil
}

// Set will set the chaos configuration for the given key (CLI & setMap())
func Set(key string, errorCode int, probability int) error {
	Enabled = true
	if errorCode < 400 || errorCode > 599 {
		return fmt.Errorf("error code must be between 400 and 599")
	}

	if probability > 100 || probability < 0 {
		return fmt.Errorf("probability must be between 0 and 100")
	}

	key = strings.ToLower(key)

	switch key {
	case "sender":
		Config.Sender = Trigger{ErrorCode: errorCode, Probability: probability}
		logger.Log().Infof("[chaos] Sender to return %d error with %d%% probability", errorCode, probability)
	case "recipient", "recipients":
		Config.Recipient = Trigger{ErrorCode: errorCode, Probability: probability}
		logger.Log().Infof("[chaos] Recipient to return %d error with %d%% probability", errorCode, probability)
	case "auth", "authentication":
		Config.Authentication = Trigger{ErrorCode: errorCode, Probability: probability}
		logger.Log().Infof("[chaos] Authentication to return %d error with %d%% probability", errorCode, probability)
	default:
		return fmt.Errorf("unknown key %s", key)
	}

	return nil
}

// Trigger will return whether the Chaos rule is triggered based on the configuration
// and a randomly-generated percentage value.
func (c Trigger) Trigger() (bool, int) {
	if !Enabled || c.Probability == 0 {
		return false, 0
	}

	nBig, _ := rand.Int(rand.Reader, big.NewInt(100))

	// rand.IntN(100) will return 0-99, whereas probability is 1-100,
	// so value must be less than (not <=) to the probability to trigger
	return int(nBig.Int64()) < c.Probability, c.ErrorCode
}
