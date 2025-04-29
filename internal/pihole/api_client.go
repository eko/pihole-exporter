package pihole

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

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

// NewAPIClient initializes and returns a new APIClient with optional TLS verification disabling.
func NewAPIClient(baseURL string, password string, timeout time.Duration, disableTLSVerification bool) *APIClient {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: skipTLSmatchSNI,
	}
	
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
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
	jsonPayload, _ := json.Marshal(payload)

	log.Info("Authenticating", url)

	resp, err := c.Client.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Error("Authentication failed", err)
		return fmt.Errorf("authentication request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed, status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
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
	log.Info("Authentication successful", c.sessionID)
	return nil
}

// ensureAuth ensures the session is valid before making a request.
func (c *APIClient) ensureAuth() error {
	c.mu.Lock()
	if time.Now().After(c.validity) {
		log.Info("Session expired, re-authenticating")
		c.mu.Unlock()
		return c.Authenticate()
	}
	c.mu.Unlock()
	return nil
}

// FetchData makes a GET request to the specified endpoint and parses the response.
func (c *APIClient) FetchData(endpoint string, result interface{}) error {
	if err := c.ensureAuth(); err != nil {
		return err
	}

	url := fmt.Sprintf("%s%s", c.BaseURL, endpoint)
	log.Info("Fetching data", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-FTL-SID", c.sessionID)

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch data from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non-200 status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("failed to parse JSON response: %w", err)
	}

	log.Info("Successfully fetched data", url)
	return nil
}
