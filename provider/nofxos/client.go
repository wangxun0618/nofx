// Package nofxos provides data access to the NofxOS API (https://nofxos.ai)
// for per-coin quantitative data (open interest, fund flow, price change).
//
// The formerly published ranking endpoints (AI500 score list, OI increase /
// decrease rankings) are no longer served; the equivalent selection is done
// locally from Hyperliquid data (see kernel/local_coin_source.go).
package nofxos

import (
	"errors"
	"io/ioutil"
	"net/http"
	"nofx/security"
	"strings"
	"sync"
	"time"
)

// Default configuration
const (
	DefaultBaseURL = "https://nofxos.ai"
	DefaultTimeout = 30 * time.Second
	DefaultAuthKey = "cm_568c67eae410d912c54c"
)

// Client is the NofxOS API client
type Client struct {
	BaseURL string
	AuthKey string
	Timeout time.Duration
	mu      sync.RWMutex
}

var (
	defaultClient *Client
	clientOnce    sync.Once
)

// DefaultClient returns the singleton default client
func DefaultClient() *Client {
	clientOnce.Do(func() {
		defaultClient = &Client{
			BaseURL: DefaultBaseURL,
			AuthKey: DefaultAuthKey,
			Timeout: DefaultTimeout,
		}
	})
	return defaultClient
}

// NewClient creates a new NofxOS API client
func NewClient(baseURL, authKey string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if authKey == "" {
		authKey = DefaultAuthKey
	}
	return &Client{
		BaseURL: baseURL,
		AuthKey: authKey,
		Timeout: DefaultTimeout,
	}
}

// SetConfig updates client configuration
func (c *Client) SetConfig(baseURL, authKey string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if baseURL != "" {
		c.BaseURL = baseURL
	}
	if authKey != "" {
		c.AuthKey = authKey
	}
}

// GetBaseURL returns the current base URL
func (c *Client) GetBaseURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.BaseURL
}

// doRequest performs an HTTP GET request with authentication.
func (c *Client) doRequest(endpoint string) ([]byte, error) {
	c.mu.RLock()
	baseURL := c.BaseURL
	authKey := c.AuthKey
	timeout := c.Timeout
	c.mu.RUnlock()

	url := baseURL + endpoint
	if !strings.Contains(url, "auth=") {
		if strings.Contains(url, "?") {
			url += "&auth=" + authKey
		} else {
			url += "?auth=" + authKey
		}
	}

	resp, err := security.SafeGet(url, timeout)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return body, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	return body, nil
}

// APIError represents an API error response
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return e.Message
}

// IsSubscriptionUnavailable reports whether an error means this account is not
// entitled to the endpoint, as opposed to a transient failure.
//
// The distinction matters because retrying cannot fix it: the ranking endpoints
// this client used to call now answer 402, and a caller that keeps asking once
// per symbol per cycle is retrying a decision rather than a connection.
func IsSubscriptionUnavailable(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	switch apiErr.StatusCode {
	case http.StatusUnauthorized, http.StatusPaymentRequired, http.StatusForbidden,
		http.StatusNotFound, http.StatusGone:
		return true
	default:
		return false
	}
}
