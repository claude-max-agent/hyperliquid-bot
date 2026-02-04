package whalealert

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/zono819/hyperliquid-bot/internal/domain/entity"
)

const (
	baseURL = "https://api.whale-alert.io/v1"
)

// Client is a Whale Alert API client
type Client struct {
	apiKey     string
	httpClient *http.Client
	minValue   float64 // Minimum USD value to track
}

// NewClient creates a new Whale Alert client
func NewClient(apiKey string, minValue float64) *Client {
	if minValue == 0 {
		minValue = 500000 // Default $500k minimum
	}
	return &Client{
		apiKey:   apiKey,
		minValue: minValue,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Connect establishes connection (validates API key)
func (c *Client) Connect(ctx context.Context) error {
	// Test API connection with a simple status check
	_, err := c.GetRecentTransactions(ctx, "bitcoin", time.Now().Add(-1*time.Hour))
	return err
}

// Disconnect closes connection
func (c *Client) Disconnect(ctx context.Context) error {
	return nil
}

// TransactionResponse represents Whale Alert API response
type TransactionResponse struct {
	Result       string `json:"result"`
	Cursor       string `json:"cursor"`
	Count        int    `json:"count"`
	Transactions []Transaction `json:"transactions"`
}

// Transaction represents a single whale transaction
type Transaction struct {
	ID          string  `json:"id"`
	Blockchain  string  `json:"blockchain"`
	Symbol      string  `json:"symbol"`
	Hash        string  `json:"hash"`
	Timestamp   int64   `json:"timestamp"`
	Amount      float64 `json:"amount"`
	AmountUSD   float64 `json:"amount_usd"`
	From        Owner   `json:"from"`
	To          Owner   `json:"to"`
}

// Owner represents transaction owner
type Owner struct {
	Address     string `json:"address"`
	Owner       string `json:"owner"`
	OwnerType   string `json:"owner_type"`
}

// GetRecentTransactions retrieves recent whale transactions
func (c *Client) GetRecentTransactions(ctx context.Context, blockchain string, since time.Time) ([]*entity.WhaleAlert, error) {
	url := fmt.Sprintf("%s/transactions?api_key=%s&min_value=%d&start=%d",
		baseURL, c.apiKey, int(c.minValue), since.Unix())

	if blockchain != "" {
		url += "&blockchain=" + blockchain
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var txResp TransactionResponse
	if err := json.Unmarshal(body, &txResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if txResp.Result != "success" {
		return nil, fmt.Errorf("API error: %s", txResp.Result)
	}

	alerts := make([]*entity.WhaleAlert, 0, len(txResp.Transactions))
	for _, tx := range txResp.Transactions {
		alerts = append(alerts, &entity.WhaleAlert{
			ID:          tx.ID,
			Blockchain:  tx.Blockchain,
			Symbol:      tx.Symbol,
			Amount:      tx.Amount,
			AmountUSD:   tx.AmountUSD,
			FromAddress: tx.From.Address,
			ToAddress:   tx.To.Address,
			FromOwner:   normalizeOwner(tx.From.Owner),
			ToOwner:     normalizeOwner(tx.To.Owner),
			TxHash:      tx.Hash,
			Timestamp:   time.Unix(tx.Timestamp, 0),
		})
	}

	return alerts, nil
}

// normalizeOwner normalizes owner names to lowercase for comparison
func normalizeOwner(owner string) string {
	if owner == "" {
		return "unknown"
	}
	// Map common variations
	ownerMap := map[string]string{
		"Binance":     "binance",
		"Coinbase":    "coinbase",
		"Kraken":      "kraken",
		"Bitfinex":    "bitfinex",
		"Bybit":       "bybit",
		"OKX":         "okx",
		"OKEx":        "okx",
		"Huobi":       "huobi",
		"KuCoin":      "kucoin",
		"Gate.io":     "gate.io",
		"unknown":     "unknown",
	}
	if normalized, ok := ownerMap[owner]; ok {
		return normalized
	}
	return owner
}

// GetLiquidations is not supported by Whale Alert
func (c *Client) GetLiquidations(ctx context.Context, symbol string) ([]*entity.Liquidation, error) {
	return nil, fmt.Errorf("liquidations not supported by Whale Alert, use CoinGlass")
}

// GetOpenInterest is not supported by Whale Alert
func (c *Client) GetOpenInterest(ctx context.Context, symbol string) (*entity.OpenInterest, error) {
	return nil, fmt.Errorf("open interest not supported by Whale Alert, use CoinGlass")
}

// GetFundingRate is not supported by Whale Alert
func (c *Client) GetFundingRate(ctx context.Context, symbol string) (*entity.FundingRate, error) {
	return nil, fmt.Errorf("funding rate not supported by Whale Alert, use CoinGlass")
}

// GetLongShortRatio is not supported by Whale Alert
func (c *Client) GetLongShortRatio(ctx context.Context, symbol string) (*entity.LongShortRatio, error) {
	return nil, fmt.Errorf("long/short ratio not supported by Whale Alert, use CoinGlass")
}

// SubscribeLiquidations is not supported by Whale Alert
func (c *Client) SubscribeLiquidations(ctx context.Context, symbol string, handler func(*entity.Liquidation)) error {
	return fmt.Errorf("liquidations not supported by Whale Alert, use CoinGlass")
}

// SubscribeWhaleAlerts subscribes to whale transaction alerts (polling implementation)
func (c *Client) SubscribeWhaleAlerts(ctx context.Context, handler func(*entity.WhaleAlert)) error {
	go func() {
		ticker := time.NewTicker(60 * time.Second) // Whale Alert has rate limits
		defer ticker.Stop()

		lastCheck := time.Now().Add(-5 * time.Minute)
		seenIDs := make(map[string]bool)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Get transactions for major blockchains
				blockchains := []string{"bitcoin", "ethereum", "tron"}
				for _, bc := range blockchains {
					alerts, err := c.GetRecentTransactions(ctx, bc, lastCheck)
					if err != nil {
						continue
					}
					for _, alert := range alerts {
						if !seenIDs[alert.ID] {
							seenIDs[alert.ID] = true
							handler(alert)
						}
					}
				}
				lastCheck = time.Now().Add(-1 * time.Minute) // Overlap to avoid missing
			}
		}
	}()

	return nil
}

// FilterBySymbol filters alerts for specific crypto symbols
func FilterBySymbol(alerts []*entity.WhaleAlert, symbols ...string) []*entity.WhaleAlert {
	symbolMap := make(map[string]bool)
	for _, s := range symbols {
		symbolMap[s] = true
	}

	filtered := make([]*entity.WhaleAlert, 0)
	for _, alert := range alerts {
		if symbolMap[alert.Symbol] {
			filtered = append(filtered, alert)
		}
	}
	return filtered
}

// FilterExchangeFlows filters alerts for exchange inflows/outflows only
func FilterExchangeFlows(alerts []*entity.WhaleAlert) []*entity.WhaleAlert {
	filtered := make([]*entity.WhaleAlert, 0)
	for _, alert := range alerts {
		alertType := alert.GetAlertType()
		if alertType == entity.WhaleAlertExchangeInflow || alertType == entity.WhaleAlertExchangeOutflow {
			filtered = append(filtered, alert)
		}
	}
	return filtered
}
