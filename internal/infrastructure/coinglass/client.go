package coinglass

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
	baseURL = "https://open-api.coinglass.com/public/v2"
)

// Client is a CoinGlass API client
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new CoinGlass client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Connect establishes connection (validates API key)
func (c *Client) Connect(ctx context.Context) error {
	// Test API connection
	_, err := c.GetFundingRate(ctx, "BTC")
	return err
}

// Disconnect closes connection
func (c *Client) Disconnect(ctx context.Context) error {
	return nil
}

// doRequest performs HTTP request with authentication
func (c *Client) doRequest(ctx context.Context, endpoint string) ([]byte, error) {
	url := baseURL + endpoint

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("CG-API-KEY", c.apiKey)

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

	return body, nil
}

// FundingRateResponse represents CoinGlass funding rate API response
type FundingRateResponse struct {
	Code    string `json:"code"`
	Msg     string `json:"msg"`
	Success bool   `json:"success"`
	Data    []struct {
		Symbol         string  `json:"symbol"`
		UMarginList    []ExchangeRate `json:"uMarginList"`
	} `json:"data"`
}

// ExchangeRate represents funding rate for an exchange
type ExchangeRate struct {
	ExchangeName  string  `json:"exchangeName"`
	Rate          float64 `json:"rate"`
	PredictedRate float64 `json:"predictedRate"`
	NextFundingTime int64 `json:"nextFundingTime"`
}

// GetFundingRate retrieves funding rate for a symbol
func (c *Client) GetFundingRate(ctx context.Context, symbol string) (*entity.FundingRate, error) {
	body, err := c.doRequest(ctx, "/funding?symbol="+symbol)
	if err != nil {
		return nil, err
	}

	var resp FundingRateResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !resp.Success || len(resp.Data) == 0 {
		return nil, fmt.Errorf("no data available for %s", symbol)
	}

	// Find Binance or first available
	var rate *ExchangeRate
	for _, data := range resp.Data {
		if data.Symbol == symbol {
			for i := range data.UMarginList {
				if data.UMarginList[i].ExchangeName == "Binance" {
					rate = &data.UMarginList[i]
					break
				}
			}
			if rate == nil && len(data.UMarginList) > 0 {
				rate = &data.UMarginList[0]
			}
			break
		}
	}

	if rate == nil {
		return nil, fmt.Errorf("no funding rate data for %s", symbol)
	}

	return &entity.FundingRate{
		Symbol:          symbol,
		Rate:            rate.Rate,
		PredictedRate:   rate.PredictedRate,
		NextFundingTime: time.Unix(rate.NextFundingTime/1000, 0),
		Exchange:        rate.ExchangeName,
		Timestamp:       time.Now(),
	}, nil
}

// OpenInterestResponse represents CoinGlass OI API response
type OpenInterestResponse struct {
	Code    string `json:"code"`
	Msg     string `json:"msg"`
	Success bool   `json:"success"`
	Data    []struct {
		Symbol        string  `json:"symbol"`
		OpenInterest  float64 `json:"openInterest"`
		H24Change     float64 `json:"h24Change"`
		ExchangeName  string  `json:"exchangeName"`
	} `json:"data"`
}

// GetOpenInterest retrieves open interest for a symbol
func (c *Client) GetOpenInterest(ctx context.Context, symbol string) (*entity.OpenInterest, error) {
	body, err := c.doRequest(ctx, "/open_interest?symbol="+symbol)
	if err != nil {
		return nil, err
	}

	var resp OpenInterestResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !resp.Success || len(resp.Data) == 0 {
		return nil, fmt.Errorf("no data available for %s", symbol)
	}

	// Aggregate all exchanges
	var totalOI float64
	var avgChange float64
	for _, data := range resp.Data {
		totalOI += data.OpenInterest
		avgChange += data.H24Change
	}
	avgChange /= float64(len(resp.Data))

	return &entity.OpenInterest{
		Symbol:       symbol,
		OpenInterest: totalOI,
		Change24h:    avgChange,
		Exchange:     "aggregated",
		Timestamp:    time.Now(),
	}, nil
}

// LongShortRatioResponse represents CoinGlass L/S ratio API response
type LongShortRatioResponse struct {
	Code    string `json:"code"`
	Msg     string `json:"msg"`
	Success bool   `json:"success"`
	Data    []struct {
		Symbol     string  `json:"symbol"`
		LongRate   float64 `json:"longRate"`
		ShortRate  float64 `json:"shortRate"`
		LongShortRatio float64 `json:"longShortRatio"`
		ExchangeName string `json:"exchangeName"`
	} `json:"data"`
}

// GetLongShortRatio retrieves long/short ratio for a symbol
func (c *Client) GetLongShortRatio(ctx context.Context, symbol string) (*entity.LongShortRatio, error) {
	body, err := c.doRequest(ctx, "/long_short?symbol="+symbol)
	if err != nil {
		return nil, err
	}

	var resp LongShortRatioResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !resp.Success || len(resp.Data) == 0 {
		return nil, fmt.Errorf("no data available for %s", symbol)
	}

	// Find Binance or first available
	var data *struct {
		Symbol     string  `json:"symbol"`
		LongRate   float64 `json:"longRate"`
		ShortRate  float64 `json:"shortRate"`
		LongShortRatio float64 `json:"longShortRatio"`
		ExchangeName string `json:"exchangeName"`
	}
	for i := range resp.Data {
		if resp.Data[i].ExchangeName == "Binance" {
			data = &resp.Data[i]
			break
		}
	}
	if data == nil {
		data = &resp.Data[0]
	}

	return &entity.LongShortRatio{
		Symbol:         symbol,
		LongRatio:      data.LongRate,
		ShortRatio:     data.ShortRate,
		LongShortRatio: data.LongShortRatio,
		Exchange:       data.ExchangeName,
		Timestamp:      time.Now(),
	}, nil
}

// LiquidationResponse represents CoinGlass liquidation API response
type LiquidationResponse struct {
	Code    string `json:"code"`
	Msg     string `json:"msg"`
	Success bool   `json:"success"`
	Data    []struct {
		Symbol     string  `json:"symbol"`
		Side       string  `json:"side"` // 1=long, 2=short
		Price      float64 `json:"price"`
		Quantity   float64 `json:"quantity"`
		Amount     float64 `json:"amount"`
		ExchangeName string `json:"exchangeName"`
		CreateTime int64   `json:"createTime"`
	} `json:"data"`
}

// GetLiquidations retrieves recent liquidations for a symbol
func (c *Client) GetLiquidations(ctx context.Context, symbol string) ([]*entity.Liquidation, error) {
	body, err := c.doRequest(ctx, "/liquidation_history?symbol="+symbol)
	if err != nil {
		return nil, err
	}

	var resp LiquidationResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("API error: %s", resp.Msg)
	}

	liquidations := make([]*entity.Liquidation, 0, len(resp.Data))
	for _, data := range resp.Data {
		side := "short"
		if data.Side == "1" {
			side = "long"
		}
		liquidations = append(liquidations, &entity.Liquidation{
			Symbol:    data.Symbol,
			Side:      side,
			Price:     data.Price,
			Quantity:  data.Quantity,
			Value:     data.Amount,
			Exchange:  data.ExchangeName,
			Timestamp: time.Unix(data.CreateTime/1000, 0),
		})
	}

	return liquidations, nil
}

// SubscribeLiquidations subscribes to liquidation events (polling implementation)
func (c *Client) SubscribeLiquidations(ctx context.Context, symbol string, handler func(*entity.Liquidation)) error {
	// CoinGlass doesn't have WebSocket, use polling
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		var lastSeen time.Time

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				liqs, err := c.GetLiquidations(ctx, symbol)
				if err != nil {
					continue
				}
				for _, liq := range liqs {
					if liq.Timestamp.After(lastSeen) {
						handler(liq)
						if liq.Timestamp.After(lastSeen) {
							lastSeen = liq.Timestamp
						}
					}
				}
			}
		}
	}()

	return nil
}

// SubscribeWhaleAlerts is not supported by CoinGlass
func (c *Client) SubscribeWhaleAlerts(ctx context.Context, handler func(*entity.WhaleAlert)) error {
	return fmt.Errorf("whale alerts not supported by CoinGlass, use Whale Alert API")
}
