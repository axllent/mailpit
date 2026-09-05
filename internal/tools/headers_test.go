package tools

import (
	"bytes"
	"strings"
	"testing"
)

const overlongPad = 70 * 1024

func buildMessage(headers ...string) []byte {
	return []byte(strings.Join(headers, "\r\n") + "\r\n\r\nbody\r\n")
}

// TestOverrideFromHeader_OverlongPrecedingHeader guards the fix for the
// scanner-truncation bypass: a header line longer than bufio.Scanner's default
// 64 KiB token size before From used to make OverrideFromHeader silently return
// the original message with a nil error, leaving the attacker-controlled From
// in place for relay.
func TestOverrideFromHeader_OverlongPrecedingHeader(t *testing.T) {
	msg := buildMessage(
		"X-Pad: "+strings.Repeat("A", overlongPad),
		"From: Attacker <attacker@example.test>",
		"To: victim@example.test",
	)

	out, err := OverrideFromHeader(msg, "safe@example.test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bytes.Contains(out, []byte("attacker@example.test")) && !bytes.Contains(out, []byte("X-Original-From: attacker@example.test")) {
		t.Fatalf("attacker From leaked into output without being demoted to X-Original-From")
	}
	if !bytes.Contains(out, []byte("From: safe@example.test")) && !bytes.Contains(out, []byte("From: \"Attacker\" <safe@example.test>")) {
		t.Fatalf("From header was not rewritten to configured address")
	}
	if !bytes.Contains(out, []byte("X-Original-From: attacker@example.test")) {
		t.Fatalf("X-Original-From audit header was not inserted")
	}
}

func TestOverrideFromHeader_ShortControl(t *testing.T) {
	msg := buildMessage(
		"X-Pad: short",
		"From: Attacker <attacker@example.test>",
		"To: victim@example.test",
	)

	out, err := OverrideFromHeader(msg, "safe@example.test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Contains(out, []byte("safe@example.test")) {
		t.Fatalf("From was not rewritten in short-header baseline")
	}
	if !bytes.Contains(out, []byte("X-Original-From: attacker@example.test")) {
		t.Fatalf("X-Original-From was not inserted in short-header baseline")
	}
}

func TestOverrideFromHeader_OverlongFromLine(t *testing.T) {
	// The From line itself is oversized (long display name).
	name := strings.Repeat("A", overlongPad)
	msg := buildMessage(
		"From: \""+name+"\" <attacker@example.test>",
		"To: victim@example.test",
	)

	out, err := OverrideFromHeader(msg, "safe@example.test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Contains(out, []byte("safe@example.test")) {
		t.Fatalf("From was not rewritten when From line itself is oversized")
	}
}

func TestOverrideFromHeader_NoFromHeader(t *testing.T) {
	msg := buildMessage(
		"To: victim@example.test",
		"Subject: hi",
	)

	out, err := OverrideFromHeader(msg, "safe@example.test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.HasPrefix(out, []byte("From: safe@example.test\r\n")) {
		t.Fatalf("From header was not prepended when missing")
	}
}

func TestSetMessageHeader_OverlongPrecedingHeader(t *testing.T) {
	msg := buildMessage(
		"X-Pad: "+strings.Repeat("A", overlongPad),
		"Return-Path: <original@example.test>",
		"From: someone@example.test",
	)

	out, err := SetMessageHeader(msg, "Return-Path", "<envelope@example.test>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Contains(out, []byte("Return-Path: <envelope@example.test>")) {
		t.Fatalf("Return-Path was not updated")
	}
	if bytes.Contains(out, []byte("Return-Path: <original@example.test>")) {
		t.Fatalf("original Return-Path still present after update")
	}
	if bytes.Count(out, []byte("Return-Path:")) != 1 {
		t.Fatalf("expected exactly one Return-Path header, got %d", bytes.Count(out, []byte("Return-Path:")))
	}
}

func TestSetMessageHeader_ShortControl(t *testing.T) {
	msg := buildMessage(
		"Return-Path: <original@example.test>",
		"From: someone@example.test",
	)

	out, err := SetMessageHeader(msg, "Return-Path", "<envelope@example.test>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Contains(out, []byte("Return-Path: <envelope@example.test>")) {
		t.Fatalf("Return-Path was not updated in baseline")
	}
	if bytes.Count(out, []byte("Return-Path:")) != 1 {
		t.Fatalf("expected exactly one Return-Path header, got %d", bytes.Count(out, []byte("Return-Path:")))
	}
}

func TestSetMessageHeader_MissingHeader(t *testing.T) {
	msg := buildMessage(
		"From: someone@example.test",
		"To: victim@example.test",
	)

	out, err := SetMessageHeader(msg, "Return-Path", "<envelope@example.test>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.HasPrefix(out, []byte("Return-Path: <envelope@example.test>\r\n")) {
		t.Fatalf("Return-Path was not prepended when missing")
	}
}

func TestRemoveMessageHeaders_OverlongPrecedingHeader(t *testing.T) {
	// A silent no-op here would have leaked Bcc recipients through to storage.
	msg := buildMessage(
		"X-Pad: "+strings.Repeat("A", overlongPad),
		"Bcc: secret@example.test",
		"From: someone@example.test",
		"To: victim@example.test",
	)

	out, err := RemoveMessageHeaders(msg, []string{"Bcc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bytes.Contains(out, []byte("secret@example.test")) {
		t.Fatalf("Bcc recipient leaked after removal")
	}
	if bytes.Contains(out, []byte("Bcc:")) {
		t.Fatalf("Bcc header still present after removal")
	}
}

func TestRemoveMessageHeaders_ShortControl(t *testing.T) {
	msg := buildMessage(
		"Bcc: secret@example.test",
		"From: someone@example.test",
		"To: victim@example.test",
	)

	out, err := RemoveMessageHeaders(msg, []string{"Bcc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bytes.Contains(out, []byte("Bcc:")) {
		t.Fatalf("Bcc header still present after removal in baseline")
	}
}

func TestRemoveMessageHeaders_MissingHeader(t *testing.T) {
	msg := buildMessage(
		"From: someone@example.test",
		"To: victim@example.test",
	)

	out, err := RemoveMessageHeaders(msg, []string{"Bcc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(out, msg) {
		t.Fatalf("message was modified when target header was absent")
	}
}
