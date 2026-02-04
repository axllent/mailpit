package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client is the Mailpit API client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	username   string
	password   string
}

// Config holds the client configuration.
type Config struct {
	BaseURL  string
	Username string
	Password string
	Timeout  time.Duration
}

// New creates a new Mailpit API client.
func New(cfg Config) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	return &Client{
		baseURL:  cfg.BaseURL,
		username: cfg.Username,
		password: cfg.Password,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// request performs an HTTP request and returns the response body.
func (c *Client) request(ctx context.Context, method, path string, query url.Values, body any) ([]byte, error) {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if query != nil {
		u.RawQuery = query.Encode()
	}

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// requestJSON performs a request and unmarshals the JSON response.
func (c *Client) requestJSON(ctx context.Context, method, path string, query url.Values, body, result any) error {
	respBody, err := c.request(ctx, method, path, query, body)
	if err != nil {
		return err
	}
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}
	return nil
}

// requestText performs a request and returns the text response.
func (c *Client) requestText(ctx context.Context, method, path string, query url.Values) (string, error) {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	if query != nil {
		u.RawQuery = query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return string(body), nil
}

// --- Messages ---

// ListMessages returns a paginated list of messages.
func (c *Client) ListMessages(ctx context.Context, start, limit int) (*MessagesSummary, error) {
	q := url.Values{}
	if start > 0 {
		q.Set("start", fmt.Sprintf("%d", start))
	}
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	var result MessagesSummary
	err := c.requestJSON(ctx, http.MethodGet, "/api/v1/messages", q, nil, &result)
	return &result, err
}

// SearchMessages searches for messages.
func (c *Client) SearchMessages(ctx context.Context, query string, start, limit int, timezone string) (*MessagesSummary, error) {
	q := url.Values{}
	q.Set("query", query)
	if start > 0 {
		q.Set("start", fmt.Sprintf("%d", start))
	}
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	if timezone != "" {
		q.Set("tz", timezone)
	}
	var result MessagesSummary
	err := c.requestJSON(ctx, http.MethodGet, "/api/v1/search", q, nil, &result)
	return &result, err
}

// GetMessage returns a message by ID.
func (c *Client) GetMessage(ctx context.Context, id string) (*Message, error) {
	var result Message
	err := c.requestJSON(ctx, http.MethodGet, "/api/v1/message/"+url.PathEscape(id), nil, nil, &result)
	return &result, err
}

// GetMessageHeaders returns the headers of a message.
func (c *Client) GetMessageHeaders(ctx context.Context, id string) (MessageHeaders, error) {
	var result MessageHeaders
	err := c.requestJSON(ctx, http.MethodGet, "/api/v1/message/"+url.PathEscape(id)+"/headers", nil, nil, &result)
	return result, err
}

// GetMessageSource returns the raw source of a message.
func (c *Client) GetMessageSource(ctx context.Context, id string) (string, error) {
	return c.requestText(ctx, http.MethodGet, "/api/v1/message/"+url.PathEscape(id)+"/raw", nil)
}

// DeleteMessages deletes messages by IDs. If ids is empty, all messages are deleted.
func (c *Client) DeleteMessages(ctx context.Context, ids []string) error {
	body := &DeleteMessagesRequest{IDs: ids}
	_, err := c.request(ctx, http.MethodDelete, "/api/v1/messages", nil, body)
	return err
}

// DeleteSearch deletes messages matching a search query.
func (c *Client) DeleteSearch(ctx context.Context, query, timezone string) error {
	q := url.Values{}
	q.Set("query", query)
	if timezone != "" {
		q.Set("tz", timezone)
	}
	_, err := c.request(ctx, http.MethodDelete, "/api/v1/search", q, nil)
	return err
}

// SetReadStatus sets the read status of messages.
func (c *Client) SetReadStatus(ctx context.Context, ids []string, read bool, search string) error {
	body := &SetReadStatusRequest{
		IDs:    ids,
		Read:   read,
		Search: search,
	}
	_, err := c.request(ctx, http.MethodPut, "/api/v1/messages", nil, body)
	return err
}

// --- Message Content ---

// GetMessageHTML returns the rendered HTML of a message.
func (c *Client) GetMessageHTML(ctx context.Context, id string) (string, error) {
	return c.requestText(ctx, http.MethodGet, "/view/"+url.PathEscape(id)+".html", nil)
}

// GetMessageText returns the text content of a message.
func (c *Client) GetMessageText(ctx context.Context, id string) (string, error) {
	return c.requestText(ctx, http.MethodGet, "/view/"+url.PathEscape(id)+".txt", nil)
}

// GetAttachment returns an attachment by message ID and part ID.
func (c *Client) GetAttachment(ctx context.Context, messageID, partID string) ([]byte, error) {
	return c.request(ctx, http.MethodGet, "/api/v1/message/"+url.PathEscape(messageID)+"/part/"+url.PathEscape(partID), nil, nil)
}

// --- Validation ---

// CheckHTML performs HTML compatibility checking.
func (c *Client) CheckHTML(ctx context.Context, id string) (*HTMLCheckResponse, error) {
	var result HTMLCheckResponse
	err := c.requestJSON(ctx, http.MethodGet, "/api/v1/message/"+url.PathEscape(id)+"/html-check", nil, nil, &result)
	return &result, err
}

// CheckLinks performs link validation.
func (c *Client) CheckLinks(ctx context.Context, id string, follow bool) (*LinkCheckResponse, error) {
	q := url.Values{}
	if follow {
		q.Set("follow", "true")
	}
	var result LinkCheckResponse
	err := c.requestJSON(ctx, http.MethodGet, "/api/v1/message/"+url.PathEscape(id)+"/link-check", q, nil, &result)
	return &result, err
}

// CheckSpam performs SpamAssassin checking.
func (c *Client) CheckSpam(ctx context.Context, id string) (*SpamAssassinResponse, error) {
	var result SpamAssassinResponse
	err := c.requestJSON(ctx, http.MethodGet, "/api/v1/message/"+url.PathEscape(id)+"/sa-check", nil, nil, &result)
	return &result, err
}

// --- Tags ---

// ListTags returns all tags.
func (c *Client) ListTags(ctx context.Context) ([]string, error) {
	var result []string
	err := c.requestJSON(ctx, http.MethodGet, "/api/v1/tags", nil, nil, &result)
	return result, err
}

// SetTags sets tags on messages.
func (c *Client) SetTags(ctx context.Context, ids, tags []string) error {
	body := &SetTagsRequest{IDs: ids, Tags: tags}
	_, err := c.request(ctx, http.MethodPut, "/api/v1/tags", nil, body)
	return err
}

// RenameTag renames a tag.
func (c *Client) RenameTag(ctx context.Context, oldName, newName string) error {
	body := &RenameTagRequest{Name: newName}
	_, err := c.request(ctx, http.MethodPut, "/api/v1/tags/"+url.PathEscape(oldName), nil, body)
	return err
}

// DeleteTag deletes a tag.
func (c *Client) DeleteTag(ctx context.Context, name string) error {
	_, err := c.request(ctx, http.MethodDelete, "/api/v1/tags/"+url.PathEscape(name), nil, nil)
	return err
}

// --- Testing ---

// SendMessage sends a message via the API.
func (c *Client) SendMessage(ctx context.Context, msg *SendMessageRequest) (*SendMessageResponse, error) {
	var result SendMessageResponse
	err := c.requestJSON(ctx, http.MethodPost, "/api/v1/send", nil, msg, &result)
	return &result, err
}

// ReleaseMessage releases a message to external recipients.
func (c *Client) ReleaseMessage(ctx context.Context, id string, to []string) error {
	body := &ReleaseRequest{To: to}
	_, err := c.request(ctx, http.MethodPost, "/api/v1/message/"+url.PathEscape(id)+"/release", nil, body)
	return err
}

// GetChaos returns the chaos triggers configuration.
func (c *Client) GetChaos(ctx context.Context) (*ChaosTriggers, error) {
	var result ChaosTriggers
	err := c.requestJSON(ctx, http.MethodGet, "/api/v1/chaos", nil, nil, &result)
	return &result, err
}

// SetChaos sets the chaos triggers configuration.
func (c *Client) SetChaos(ctx context.Context, triggers *ChaosTriggers) (*ChaosTriggers, error) {
	var result ChaosTriggers
	err := c.requestJSON(ctx, http.MethodPut, "/api/v1/chaos", nil, triggers, &result)
	return &result, err
}

// --- System ---

// GetInfo returns application information.
func (c *Client) GetInfo(ctx context.Context) (*AppInfo, error) {
	var result AppInfo
	err := c.requestJSON(ctx, http.MethodGet, "/api/v1/info", nil, nil, &result)
	return &result, err
}

// GetWebUIConfig returns the web UI configuration.
func (c *Client) GetWebUIConfig(ctx context.Context) (*WebUIConfig, error) {
	var result WebUIConfig
	err := c.requestJSON(ctx, http.MethodGet, "/api/v1/webui", nil, nil, &result)
	return &result, err
}
