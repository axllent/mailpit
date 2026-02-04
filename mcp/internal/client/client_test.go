package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestClient(serverURL string) *Client {
	return New(Config{
		BaseURL: serverURL,
		Timeout: 10 * time.Second,
	})
}

func newTestClientWithAuth(serverURL, user, pass string) *Client {
	return New(Config{
		BaseURL:  serverURL,
		Username: user,
		Password: pass,
		Timeout:  10 * time.Second,
	})
}

func TestNewClient(t *testing.T) {
	c := New(Config{
		BaseURL: "http://localhost:8025",
		Timeout: 30 * time.Second,
	})
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	if c.baseURL != "http://localhost:8025" {
		t.Errorf("expected baseURL http://localhost:8025, got %s", c.baseURL)
	}
}

func TestGetInfo(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/info" {
			t.Errorf("expected path /api/v1/info, got %s", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}

		info := &AppInfo{
			Version:      "1.0.0",
			Database:     "memory",
			DatabaseSize: 1024,
			Messages:     10,
			Unread:       5,
		}
		json.NewEncoder(w).Encode(info)
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	info, err := c.GetInfo(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", info.Version)
	}
	if info.Messages != 10 {
		t.Errorf("expected 10 messages, got %d", info.Messages)
	}
	if info.Unread != 5 {
		t.Errorf("expected 5 unread, got %d", info.Unread)
	}
}

func TestListMessages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/messages" {
			t.Errorf("expected path /api/v1/messages, got %s", r.URL.Path)
		}

		// Check query params
		if r.URL.Query().Get("limit") != "10" {
			t.Errorf("expected limit=10, got %s", r.URL.Query().Get("limit"))
		}

		response := &MessagesSummary{
			Total:         100,
			Unread:        50,
			MessagesCount: 10,
			Messages: []*MessageSummary{
				{ID: "abc123", Subject: "Test Email"},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	result, err := c.ListMessages(context.Background(), 0, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 100 {
		t.Errorf("expected total 100, got %d", result.Total)
	}
	if len(result.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(result.Messages))
	}
	if result.Messages[0].ID != "abc123" {
		t.Errorf("expected message ID abc123, got %s", result.Messages[0].ID)
	}
}

func TestSearchMessages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/search" {
			t.Errorf("expected path /api/v1/search, got %s", r.URL.Path)
		}

		// Check query params
		if r.URL.Query().Get("query") != "from:test@example.com" {
			t.Errorf("expected query from:test@example.com, got %s", r.URL.Query().Get("query"))
		}

		response := &MessagesSummary{
			Total:         5,
			MessagesCount: 5,
			Messages: []*MessageSummary{
				{ID: "search1", Subject: "Search Result"},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	result, err := c.SearchMessages(context.Background(), "from:test@example.com", 0, 50, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 5 {
		t.Errorf("expected total 5, got %d", result.Total)
	}
}

func TestGetMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/message/msg123" {
			t.Errorf("expected path /api/v1/message/msg123, got %s", r.URL.Path)
		}

		msg := &Message{
			ID:      "msg123",
			Subject: "Test Subject",
			Text:    "Hello World",
		}
		json.NewEncoder(w).Encode(msg)
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	msg, err := c.GetMessage(context.Background(), "msg123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.ID != "msg123" {
		t.Errorf("expected ID msg123, got %s", msg.ID)
	}
	if msg.Subject != "Test Subject" {
		t.Errorf("expected subject 'Test Subject', got %s", msg.Subject)
	}
}

func TestListTags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/tags" {
			t.Errorf("expected path /api/v1/tags, got %s", r.URL.Path)
		}

		tags := []string{"inbox", "important", "work"}
		json.NewEncoder(w).Encode(tags)
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	tags, err := c.ListTags(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(tags))
	}
	if tags[0] != "inbox" {
		t.Errorf("expected first tag 'inbox', got %s", tags[0])
	}
}

func TestAuthentication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "testuser" || pass != "testpass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		info := &AppInfo{Version: "1.0.0"}
		json.NewEncoder(w).Encode(info)
	}))
	defer server.Close()

	// Without auth
	c := newTestClient(server.URL)
	_, err := c.GetInfo(context.Background())
	if err == nil {
		t.Error("expected error without auth")
	}

	// With auth
	c = newTestClientWithAuth(server.URL, "testuser", "testpass")
	info, err := c.GetInfo(context.Background())
	if err != nil {
		t.Fatalf("unexpected error with auth: %v", err)
	}
	if info.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", info.Version)
	}
}

func TestErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("message not found"))
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	_, err := c.GetMessage(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for 404 response")
	}
}

func TestSendMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/send" {
			t.Errorf("expected path /api/v1/send, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}

		var req SendMessageRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Subject != "Test Subject" {
			t.Errorf("expected subject 'Test Subject', got %s", req.Subject)
		}

		resp := &SendMessageResponse{ID: "newmsg123"}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	msg := &SendMessageRequest{
		From:    &SendAddress{Email: "test@example.com"},
		To:      []*SendAddress{{Email: "recipient@example.com"}},
		Subject: "Test Subject",
		Text:    "Hello World",
	}

	resp, err := c.SendMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "newmsg123" {
		t.Errorf("expected ID newmsg123, got %s", resp.ID)
	}
}

func TestDeleteMessages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/messages" {
			t.Errorf("expected path /api/v1/messages, got %s", r.URL.Path)
		}
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE method, got %s", r.Method)
		}

		var req DeleteMessagesRequest
		json.NewDecoder(r.Body).Decode(&req)

		if len(req.IDs) != 2 {
			t.Errorf("expected 2 IDs, got %d", len(req.IDs))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	err := c.DeleteMessages(context.Background(), []string{"msg1", "msg2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetTags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/tags" {
			t.Errorf("expected path /api/v1/tags, got %s", r.URL.Path)
		}
		if r.Method != "PUT" {
			t.Errorf("expected PUT method, got %s", r.Method)
		}

		var req SetTagsRequest
		json.NewDecoder(r.Body).Decode(&req)

		if len(req.IDs) != 1 || req.IDs[0] != "msg1" {
			t.Errorf("expected IDs [msg1], got %v", req.IDs)
		}
		if len(req.Tags) != 2 {
			t.Errorf("expected 2 tags, got %d", len(req.Tags))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	err := c.SetTags(context.Background(), []string{"msg1"}, []string{"important", "work"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
