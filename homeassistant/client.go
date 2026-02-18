package homeassistant

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var Devices = []string{
	"mobile_app_mikael",
}

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) do(method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	return c.http.Do(req)
}

// Ping checks connectivity by calling GET /api/
func (c *Client) Ping() error {
	resp, err := c.do("GET", "/api/", nil)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	var result struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if result.Message != "API running." {
		return fmt.Errorf("unexpected response: %q", result.Message)
	}
	return nil
}

// Notify sends a push notification to all devices in Devices.
func (c *Client) Notify(title, message string) error {
	payload, _ := json.Marshal(map[string]string{
		"title":   title,
		"message": message,
	})
	var errs []error
	for _, device := range Devices {
		resp, err := c.do("POST", "/api/services/notify/"+device, bytes.NewReader(payload))
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", device, err))
			continue
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			errs = append(errs, fmt.Errorf("%s: status %d", device, resp.StatusCode))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("notify errors: %v", errs)
	}
	return nil
}
