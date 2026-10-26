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
	BaseURL       string
	Client        *http.Client
	password      string
	sessionID     string
	authenticated bool
	mu            sync.Mutex
}

type authResponse struct {
	Session struct {
		Valid bool   `json:"valid"`
		SID   string `json:"sid"`
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

// session returns the current session ID, authenticating first if there is none yet.
func (c *APIClient) session() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.authenticated {
		if err := c.authenticate(); err != nil {
			return "", err
		}
	}
	return c.sessionID, nil
}

// renewSession authenticates again after Pi-hole rejected staleSID. The lock is
// held during the request so that concurrent callers wait for a single
// authentication instead of each running the costly Pi-hole password check.
func (c *APIClient) renewSession(staleSID string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.authenticated && c.sessionID != staleSID {
		return c.sessionID, nil
	}
	if err := c.authenticate(); err != nil {
		return "", err
	}
	return c.sessionID, nil
}

// authenticate logs in and stores the session ID. c.mu must be held.
//
// Pi-hole extends a session each time it is used, so the session is kept
// until Pi-hole rejects it rather than renewed from its announced validity.
func (c *APIClient) authenticate() error {
	c.authenticated = false

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
	c.authenticated = true
	log.Debugf("Authentication successful")
	return nil
}

// FetchData makes a GET request to the specified endpoint and parses the response.
// When Pi-hole rejects the session, it authenticates again and retries once.
func (c *APIClient) FetchData(endpoint string, result interface{}) error {
	sid, err := c.session()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s%s", c.BaseURL, endpoint)
	log.Debugf("Fetching data from %s", url)

	statusCode, body, err := c.get(url, sid)
	if err != nil {
		return err
	}

	if statusCode == http.StatusUnauthorized {
		log.Debug("Session rejected, re-authenticating")
		if sid, err = c.renewSession(sid); err != nil {
			return err
		}
		if statusCode, body, err = c.get(url, sid); err != nil {
			return err
		}
	}

	if statusCode != http.StatusOK {
		return fmt.Errorf("non-200 status code: %d", statusCode)
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("failed to parse JSON response: %w", err)
	}

	log.Debugf("Successfully fetched data from endpoint: %s", endpoint)
	return nil
}

// get performs a GET request with the given session ID and returns the status code and body.
func (c *APIClient) get(url string, sid string) (int, []byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add security headers
	req.Header.Set("X-FTL-SID", sid)
	req.Header.Set("X-Content-Type-Options", "nosniff")

	ctx, cancel := context.WithTimeout(context.Background(), c.Client.Timeout)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := c.Client.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to fetch data from %s: %w", url, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Warnf("Failed to close response body: %v", err)
		}
	}()

	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseSize)) // prevent reading too much data
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read response body: %w", err)
	}
	return resp.StatusCode, body, nil
}

// logout ends the current session so it does not hold one of Pi-hole's
// limited API seats until it expires.
func (c *APIClient) logout() {
	c.mu.Lock()
	sid := c.sessionID
	c.sessionID = ""
	c.authenticated = false
	c.mu.Unlock()

	if sid == "" {
		return
	}

	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/auth", c.BaseURL), nil)
	if err != nil {
		log.Warnf("Failed to create logout request: %v", err)
		return
	}
	req.Header.Set("X-FTL-SID", sid)

	resp, err := c.Client.Do(req)
	if err != nil {
		log.Warnf("Logout request failed: %v", err)
		return
	}
	if err := resp.Body.Close(); err != nil {
		log.Warnf("Failed to close response body: %v", err)
	}
}

// Close cleans up resources used by the API client
func (c *APIClient) Close() {
	c.logout()

	// Close the transport to ensure no connection leaks
	if transport, ok := c.Client.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
}
