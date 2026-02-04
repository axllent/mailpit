package tools

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestTextResult(t *testing.T) {
	result := textResult("Hello, World!")

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.IsError {
		t.Error("expected IsError to be false")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent type")
	}
	if textContent.Text != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got %s", textContent.Text)
	}
}

func TestErrorResult(t *testing.T) {
	err := &testError{msg: "something went wrong"}
	result := errorResult(err)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.IsError {
		t.Error("expected IsError to be true")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent type")
	}
	if textContent.Text != "Error: something went wrong" {
		t.Errorf("expected 'Error: something went wrong', got %s", textContent.Text)
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		input    uint64
		expected string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{1048576, "1.0 MiB"},
		{1073741824, "1.0 GiB"},
		{1099511627776, "1.0 TiB"},
	}

	for _, tc := range tests {
		result := formatSize(tc.input)
		if result != tc.expected {
			t.Errorf("formatSize(%d): expected %s, got %s", tc.input, tc.expected, result)
		}
	}
}

func TestFormatAddress(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{"", "test@example.com", "test@example.com"},
		{"John Doe", "john@example.com", "John Doe <john@example.com>"},
		{"Jane", "jane@example.com", "Jane <jane@example.com>"},
	}

	for _, tc := range tests {
		result := formatAddress(tc.name, tc.email)
		if result != tc.expected {
			t.Errorf("formatAddress(%q, %q): expected %s, got %s", tc.name, tc.email, tc.expected, result)
		}
	}
}
