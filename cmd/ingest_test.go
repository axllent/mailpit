package cmd

import (
	"net/mail"
	"reflect"
	"strings"
	"testing"
)

func TestIngestEnvelope(t *testing.T) {
	const def = "default@example.com"

	tests := []struct {
		name       string
		headers    string
		wantSender string
		wantRcpts  []string
	}{
		// a message with no From or Return-Path must still get a valid sender,
		// otherwise sendmail.Send rejects it and the message is skipped
		{
			name:       "no From or Return-Path uses default sender",
			headers:    "To: to@example.com\r\n",
			wantSender: def,
			wantRcpts:  []string{"to@example.com"},
		},
		// control: same shape with a From header
		{
			name:       "From used as sender",
			headers:    "From: from@example.com\r\nTo: to@example.com\r\n",
			wantSender: "from@example.com",
			wantRcpts:  []string{"to@example.com"},
		},
		{
			name:       "Return-Path preferred over From",
			headers:    "Return-Path: <bounce@example.com>\r\nFrom: from@example.com\r\nTo: to@example.com\r\n",
			wantSender: "bounce@example.com",
			wantRcpts:  []string{"to@example.com"},
		},
		{
			name:       "null Return-Path falls back to From",
			headers:    "Return-Path: <>\r\nFrom: from@example.com\r\nTo: to@example.com\r\n",
			wantSender: "from@example.com",
			wantRcpts:  []string{"to@example.com"},
		},
		{
			name:       "To, Cc and Bcc collected",
			headers:    "From: from@example.com\r\nTo: a@example.com, b@example.com\r\nCc: c@example.com\r\nBcc: d@example.com\r\n",
			wantSender: "from@example.com",
			wantRcpts:  []string{"a@example.com", "b@example.com", "c@example.com", "d@example.com"},
		},
		{
			name:       "no recipients falls back to From",
			headers:    "From: from@example.com\r\n",
			wantSender: "from@example.com",
			wantRcpts:  []string{"from@example.com"},
		},
		{
			name:       "no recipients or From falls back to default",
			headers:    "Subject: nothing\r\n",
			wantSender: def,
			wantRcpts:  []string{def},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg, err := mail.ReadMessage(strings.NewReader(tc.headers + "\r\nbody\r\n"))
			if err != nil {
				t.Fatal(err)
			}

			sender, rcpts := ingestEnvelope(msg.Header, def)

			if sender != tc.wantSender {
				t.Errorf("sender = %q, want %q", sender, tc.wantSender)
			}
			if !reflect.DeepEqual(rcpts, tc.wantRcpts) {
				t.Errorf("recipients = %q, want %q", rcpts, tc.wantRcpts)
			}
			// sendmail.Send rejects a sender that does not parse
			if _, err := mail.ParseAddress(sender); err != nil {
				t.Errorf("sender %q would be rejected by sendmail.Send: %v", sender, err)
			}
		})
	}
}
