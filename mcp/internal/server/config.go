// Package server provides the MCP server setup and configuration.
package server

import (
	"os"
	"strconv"
	"time"

	"github.com/axllent/mailpit/mcp/internal/client"
)

// Config holds the server configuration.
type Config struct {
	// Mailpit connection settings
	MailpitURL string
	AuthUser   string
	AuthPass   string
	Timeout    time.Duration

	// MCP transport settings
	Transport string // "stdio" or "http"
	HTTPHost  string
	HTTPPort  int

	// Logging
	LogLevel string
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		MailpitURL: "http://localhost:8025",
		Timeout:    30 * time.Second,
		Transport:  "stdio",
		HTTPHost:   "0.0.0.0",
		HTTPPort:   3000,
		LogLevel:   "info",
	}
}

// LoadFromEnv loads configuration from environment variables.
func (c *Config) LoadFromEnv() {
	if v := os.Getenv("MAILPIT_URL"); v != "" {
		c.MailpitURL = v
	}
	if v := os.Getenv("MAILPIT_AUTH_USER"); v != "" {
		c.AuthUser = v
	}
	if v := os.Getenv("MAILPIT_AUTH_PASS"); v != "" {
		c.AuthPass = v
	}
	if v := os.Getenv("MAILPIT_TIMEOUT"); v != "" {
		if timeout, err := strconv.Atoi(v); err == nil {
			c.Timeout = time.Duration(timeout) * time.Second
		}
	}
	if v := os.Getenv("MCP_TRANSPORT"); v != "" {
		c.Transport = v
	}
	if v := os.Getenv("MCP_HTTP_HOST"); v != "" {
		c.HTTPHost = v
	}
	if v := os.Getenv("MCP_HTTP_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.HTTPPort = port
		}
	}
	if v := os.Getenv("MCP_LOG_LEVEL"); v != "" {
		c.LogLevel = v
	}
}

// NewMailpitClient creates a Mailpit API client from the config.
func (c *Config) NewMailpitClient() *client.Client {
	return client.New(client.Config{
		BaseURL:  c.MailpitURL,
		Username: c.AuthUser,
		Password: c.AuthPass,
		Timeout:  c.Timeout,
	})
}
