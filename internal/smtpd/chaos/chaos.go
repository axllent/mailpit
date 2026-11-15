// Package chaos is used to simulate Chaos engineering (random failures) in the SMTPD server.
// See https://en.wikipedia.org/wiki/Chaos_engineering
// See https://mailpit.axllent.org/docs/integration/chaos/
package chaos

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/axllent/mailpit/internal/logger"
)

var (
	// Enabled is a flag to enable or disable support for chaos. It is set when Mailpit starts (and by Set, only
	// if it is not already set), as it is read by every SMTP session.
	Enabled = false

	// mu protects triggers, which the API changes while SMTP sessions read it
	mu sync.RWMutex

	// triggers is the global Chaos configuration, read with Get
	triggers = Triggers{
		Sender:         Trigger{ErrorCode: 451, Probability: 0},
		Recipient:      Trigger{ErrorCode: 451, Probability: 0},
		Authentication: Trigger{ErrorCode: 535, Probability: 0},
	}
)

// Get returns a copy of the current Chaos configuration.
func Get() Triggers {
	mu.RLock()
	defer mu.RUnlock()

	return triggers
}

// Triggers for the Chaos configuration
//
// swagger:model ChaosTriggers
type Triggers struct {
	// Sender trigger to fail on From, Sender
	Sender Trigger
	// Recipient trigger to fail on To, Cc, Bcc
	Recipient Trigger
	// Authentication trigger to fail while authenticating (auth must be configured)
	Authentication Trigger
}

// Trigger for Chaos
//
// swagger:model ChaosTrigger
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

// SetFromStruct will set a whole map of chaos configurations (ie: API).
// All of it is valid or none of it is used, and it is all changed at once.
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

	for _, t := range []Trigger{c.Sender, c.Recipient, c.Authentication} {
		if err := validate(t.ErrorCode, t.Probability); err != nil {
			return err
		}
	}

	setEnabled()

	mu.Lock()
	triggers = c
	mu.Unlock()

	logger.Log().Infof("[chaos] Sender to return %d error with %d%% probability", c.Sender.ErrorCode, c.Sender.Probability)
	logger.Log().Infof("[chaos] Recipient to return %d error with %d%% probability", c.Recipient.ErrorCode, c.Recipient.Probability)
	logger.Log().Infof("[chaos] Authentication to return %d error with %d%% probability", c.Authentication.ErrorCode, c.Authentication.Probability)

	return nil
}

// setEnabled enables chaos. Enabled is only written if it is not set, which it already is when this is called
// from the API (that is only possible when it is), as every SMTP session reads it.
func setEnabled() {
	if !Enabled {
		Enabled = true
	}
}

func validate(errorCode int, probability int) error {
	if errorCode < 400 || errorCode > 599 {
		return fmt.Errorf("error code must be between 400 and 599")
	}

	if probability > 100 || probability < 0 {
		return fmt.Errorf("probability must be between 0 and 100")
	}

	return nil
}

// Set will set the chaos configuration for the given key (CLI & setMap())
func Set(key string, errorCode int, probability int) error {
	setEnabled()

	if err := validate(errorCode, probability); err != nil {
		return err
	}

	key = strings.ToLower(key)

	mu.Lock()
	defer mu.Unlock()

	switch key {
	case "sender":
		triggers.Sender = Trigger{ErrorCode: errorCode, Probability: probability}
		logger.Log().Infof("[chaos] Sender to return %d error with %d%% probability", errorCode, probability)
	case "recipient", "recipients":
		triggers.Recipient = Trigger{ErrorCode: errorCode, Probability: probability}
		logger.Log().Infof("[chaos] Recipient to return %d error with %d%% probability", errorCode, probability)
	case "auth", "authentication":
		triggers.Authentication = Trigger{ErrorCode: errorCode, Probability: probability}
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
