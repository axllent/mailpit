package apiv1

import (
    "encoding/json"
    "testing"
)

func TestEmailOrObjectUnmarshal(t *testing.T) {
    var a EmailOrObject

    // Test simple string
    err := json.Unmarshal([]byte(`"user@example.com"`), &a)
    if err != nil {
        t.Fatal(err)
    }
    if a.Name != "" || a.Email != "user@example.com" {
        t.Fatalf("expected email only, got %+v", a)
    }

    // Test full object
    err = json.Unmarshal([]byte(`{"name":"John Doe","email":"john@example.com"}`), &a)
    if err != nil {
        t.Fatal(err)
    }
    if a.Name != "John Doe" || a.Email != "john@example.com" {
        t.Fatalf("expected full struct, got %+v", a)
    }
}


func TestEmailOrObjectListUnmarshal(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []EmailOrObject
		wantErr  bool
	}{
		{
			name:  "single email string",
			input: `"bob@example.com"`,
			expected: []EmailOrObject{
				{Email: "bob@example.com"},
			},
		},
		{
			name:  "single object",
			input: `{"name":"Alice","email":"alice@example.com"}`,
			expected: []EmailOrObject{
				{Name: "Alice", Email: "alice@example.com"},
			},
		},
		{
			name:  "array of strings",
			input: `["bob@example.com", "jane@example.com"]`,
			expected: []EmailOrObject{
				{Email: "bob@example.com"},
				{Email: "jane@example.com"},
			},
		},
		{
			name:  "array of objects",
			input: `[{"name":"Alice","email":"alice@example.com"}, {"name":"Bob","email":"bob@example.com"}]`,
			expected: []EmailOrObject{
				{Name: "Alice", Email: "alice@example.com"},
				{Name: "Bob", Email: "bob@example.com"},
			},
		},
		{
			name:    "invalid format",
			input:   `12345`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result EmailOrObjectList
			err := json.Unmarshal([]byte(tt.input), &result)

			if (err != nil) != tt.wantErr {
				t.Fatalf("unexpected error state: %v", err)
			}
			if !tt.wantErr && len(result) != len(tt.expected) {
				t.Fatalf("expected %d items, got %d", len(tt.expected), len(result))
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("mismatch at index %d: got %+v, want %+v", i, result[i], tt.expected[i])
				}
			}
		})
	}
}