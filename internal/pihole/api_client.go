package pihole

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"crypto/tls"

	log "github.com/sirupsen/logrus"
)

type APIClient struct {
	BaseURL   string
	Client    *http.Client
	password  string
	sessionID string
	validity  time.Time
	mu        sync.Mutex
}

type authResponse struct {
	Session struct {
		Valid    bool   `json:"valid"`
		SID      string `json:"sid"`
		Validity int    `json:"validity"`
	} `json:"session"`
}

const (
	MaxResponseSize = 1 * 1024 * 1024 // 1MB (for DoS protection)
)

// NewAPIClient initializes and returns a new APIClient.
func NewAPIClient(baseURL string, password string, timeout time.Duration, skipTLSVerification bool) *APIClient {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: skipTLSVerification,
		},
	}

	return &APIClient{
		BaseURL:  baseURL,
		password: password,
		Client: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}
}

// Authenticate logs in and stores the session ID.
func (c *APIClient) Authenticate() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	url := fmt.Sprintf("%s/api/auth", c.BaseURL)
	payload := map[string]string{"password": c.password}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal authentication payload: %w", err)
	}

	log.Debugf("Authenticating to %s", c.BaseURL)

	resp, err := c.Client.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Errorf("Authentication request failed: %v", err)
		return fmt.Errorf("authentication request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Warnf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed, status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseSize)) // Prevent
	if err != nil {
		return fmt.Errorf("failed to read authentication response: %w", err)
	}

	var authResp authResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return fmt.Errorf("failed to parse authentication response: %w", err)
	}

	if !authResp.Session.Valid {
		return fmt.Errorf("authentication unsuccessful")
	}

	c.sessionID = authResp.Session.SID
	c.validity = time.Now().Add(time.Duration(authResp.Session.Validity) * time.Second)
	log.Debugf("Authentication successful")
	return nil
}

// ensureAuth ensures the session is valid before making a request.
func (c *APIClient) ensureAuth() error {
	c.mu.Lock()
	// Check if authentication is needed
	needsAuth := time.Now().After(c.validity)
	// Always unlock the mutex before calling Authenticate
	c.mu.Unlock()

	if needsAuth {
		log.Debug("Session expired, re-authenticating")
		return c.Authenticate()
	}
	return nil
}

// FetchData makes a GET request to the specified endpoint and parses the response.
func (c *APIClient) FetchData(endpoint string, result interface{}) error {
	if err := c.ensureAuth(); err != nil {
		return err
	}

	url := fmt.Sprintf("%s%s", c.BaseURL, endpoint)
	log.Debugf("Fetching data from %s\n", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add security headers
	req.Header.Set("X-FTL-SID", c.sessionID)
	req.Header.Set("X-Content-Type-Options", "nosniff")

	ctx, cancel := context.WithTimeout(context.Background(), c.Client.Timeout)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch data from %s: %w", url, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Warnf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non-200 status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseSize)) // prevent reading too much data
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("failed to parse JSON response: %w", err)
	}

	log.Debugf("Successfully fetched data from endpoint: %s\n", endpoint)
	return nil
}

// Close cleans up resources used by the API client
func (c *APIClient) Close() {
	// Close the transport to ensure no connection leaks
	if transport, ok := c.Client.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
}
