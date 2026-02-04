package hyperliquid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ClientConfig holds configuration for the Hyperliquid API client
type ClientConfig struct {
	BaseURL   string
	APIKey    string
	APISecret string
	Testnet   bool
}

// Client is a Hyperliquid API client
type Client struct {
	config     ClientConfig
	httpClient *http.Client
}

// NewClient creates a new Hyperliquid API client
func NewClient(config ClientConfig) *Client {
	if config.BaseURL == "" {
		if config.Testnet {
			config.BaseURL = "https://api.hyperliquid-testnet.xyz"
		} else {
			config.BaseURL = "https://api.hyperliquid.xyz"
		}
	}

	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// InfoRequest represents an info API request
type InfoRequest struct {
	Type string      `json:"type"`
	User string      `json:"user,omitempty"`
	Coin string      `json:"coin,omitempty"`
}

// doRequest performs an HTTP request
func (c *Client) doRequest(ctx context.Context, endpoint string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+endpoint, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// GetMeta retrieves exchange metadata
func (c *Client) GetMeta(ctx context.Context) (map[string]interface{}, error) {
	req := InfoRequest{Type: "meta"}
	respBody, err := c.doRequest(ctx, "/info", req)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return result, nil
}

// GetAllMids retrieves mid prices for all assets
func (c *Client) GetAllMids(ctx context.Context) (map[string]string, error) {
	req := InfoRequest{Type: "allMids"}
	respBody, err := c.doRequest(ctx, "/info", req)
	if err != nil {
		return nil, err
	}

	var result map[string]string
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return result, nil
}

// GetUserState retrieves user account state
func (c *Client) GetUserState(ctx context.Context, user string) (map[string]interface{}, error) {
	req := InfoRequest{Type: "clearinghouseState", User: user}
	respBody, err := c.doRequest(ctx, "/info", req)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return result, nil
}

// GetOpenOrders retrieves user's open orders
func (c *Client) GetOpenOrders(ctx context.Context, user string) ([]map[string]interface{}, error) {
	req := InfoRequest{Type: "openOrders", User: user}
	respBody, err := c.doRequest(ctx, "/info", req)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return result, nil
}
