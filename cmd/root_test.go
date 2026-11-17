package cmd

import (
	"strings"
	"testing"

	"github.com/axllent/mailpit/config"
)

// resetEnvState restores the globals touched by these tests.
func resetEnvState(t *testing.T) {
	t.Helper()
	origSize, origErrs := config.MaxMessageSize, config.EnvErrors
	t.Cleanup(func() {
		config.MaxMessageSize, config.EnvErrors = origSize, origErrs
	})
}

func TestSetIntFromEnv(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    int
		wantErr bool
	}{
		{"unset keeps default", "", 50, false},
		{"valid integer", "10", 10, false},
		{"valid zero", "0", 0, false},
		{"unit suffix", "10MB", 50, true},
		{"leading space", " 10", 50, true},
		{"exponent", "1e3", 50, true},
		{"overflow", "99999999999999999999", 50, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetEnvState(t)
			config.EnvErrors = nil
			got := 50
			t.Setenv("MP_TEST_INT", tc.value)

			setIntFromEnv("MP_TEST_INT", &got)

			if got != tc.want {
				t.Errorf("value = %d, want %d", got, tc.want)
			}
			if hasErr := len(config.EnvErrors) > 0; hasErr != tc.wantErr {
				t.Errorf("recorded error = %v, want %v (%v)", hasErr, tc.wantErr, config.EnvErrors)
			}
		})
	}
}

// An invalid MP_MAX_MESSAGE_SIZE must not silently become 0 (unlimited), and must stop
// startup. The control uses the same variable with a valid value.
func TestInvalidMaxMessageSizeFailsClosed(t *testing.T) {
	t.Run("invalid value", func(t *testing.T) {
		resetEnvState(t)
		config.EnvErrors = nil
		config.MaxMessageSize = 50
		t.Setenv("MP_MAX_MESSAGE_SIZE", "10MB")

		initConfigFromEnv()

		if config.MaxMessageSize != 50 {
			t.Errorf("MaxMessageSize = %d, want default 50 retained", config.MaxMessageSize)
		}
		err := config.VerifyConfig()
		if err == nil || !strings.Contains(err.Error(), "MP_MAX_MESSAGE_SIZE") {
			t.Errorf("VerifyConfig() error = %v, want error naming MP_MAX_MESSAGE_SIZE", err)
		}
	})

	t.Run("valid value", func(t *testing.T) {
		resetEnvState(t)
		config.EnvErrors = nil
		config.MaxMessageSize = 50
		t.Setenv("MP_MAX_MESSAGE_SIZE", "10")

		initConfigFromEnv()

		if config.MaxMessageSize != 10 {
			t.Errorf("MaxMessageSize = %d, want 10", config.MaxMessageSize)
		}
		if len(config.EnvErrors) != 0 {
			t.Errorf("EnvErrors = %v, want none", config.EnvErrors)
		}
	})
}
