package macro

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/zono819/hyperliquid-bot/internal/domain/entity"
)

const (
	tradingEconomicsBaseURL = "https://api.tradingeconomics.com"
)

// TradingEconomicsClient is a Trading Economics API client
type TradingEconomicsClient struct {
	apiKey     string
	httpClient *http.Client
}

// NewTradingEconomicsClient creates a new Trading Economics client
func NewTradingEconomicsClient(apiKey string) *TradingEconomicsClient {
	return &TradingEconomicsClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Connect validates API connection
func (c *TradingEconomicsClient) Connect(ctx context.Context) error {
	_, err := c.GetIndicator(ctx, "united states", "inflation rate")
	return err
}

// Disconnect closes connection
func (c *TradingEconomicsClient) Disconnect(ctx context.Context) error {
	return nil
}

// doRequest performs authenticated HTTP request
func (c *TradingEconomicsClient) doRequest(ctx context.Context, endpoint string) ([]byte, error) {
	// Add API key to URL
	separator := "?"
	if len(endpoint) > 0 && endpoint[len(endpoint)-1] != '?' {
		if containsQuery(endpoint) {
			separator = "&"
		}
	}
	fullURL := tradingEconomicsBaseURL + endpoint + separator + "c=" + c.apiKey

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

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

func containsQuery(s string) bool {
	for _, c := range s {
		if c == '?' {
			return true
		}
	}
	return false
}

// IndicatorResponse represents Trading Economics indicator response
type IndicatorResponse []struct {
	Country          string  `json:"Country"`
	Category         string  `json:"Category"`
	Title            string  `json:"Title"`
	LatestValue      float64 `json:"LatestValue"`
	LatestValueDate  string  `json:"LatestValueDate"`
	PreviousValue    float64 `json:"PreviousValue"`
	PreviousValueDate string `json:"PreviousValueDate"`
	Frequency        string  `json:"Frequency"`
	Unit             string  `json:"Unit"`
	Source           string  `json:"Source"`
	HistoricalDataSymbol string `json:"HistoricalDataSymbol"`
}

// GetIndicator retrieves a specific economic indicator
func (c *TradingEconomicsClient) GetIndicator(ctx context.Context, country, indicator string) (*entity.EconomicIndicator, error) {
	endpoint := fmt.Sprintf("/country/%s/%s", url.PathEscape(country), url.PathEscape(indicator))

	body, err := c.doRequest(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var resp IndicatorResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(resp) == 0 {
		return nil, fmt.Errorf("no data found for %s %s", country, indicator)
	}

	data := resp[0]
	lastUpdate, _ := time.Parse("2006-01-02T15:04:05", data.LatestValueDate)

	return &entity.EconomicIndicator{
		Country:    data.Country,
		Category:   data.Category,
		Name:       data.Title,
		Value:      data.LatestValue,
		Previous:   data.PreviousValue,
		Unit:       data.Unit,
		Frequency:  data.Frequency,
		LastUpdate: lastUpdate,
		Timestamp:  time.Now(),
	}, nil
}

// GetUSInflation retrieves US CPI/Inflation data
func (c *TradingEconomicsClient) GetUSInflation(ctx context.Context) (*entity.EconomicIndicator, error) {
	indicator, err := c.GetIndicator(ctx, "united states", "inflation rate")
	if err != nil {
		return nil, err
	}
	indicator.Importance = "high"
	return indicator, nil
}

// GetUSGDP retrieves US GDP data
func (c *TradingEconomicsClient) GetUSGDP(ctx context.Context) (*entity.EconomicIndicator, error) {
	indicator, err := c.GetIndicator(ctx, "united states", "gdp growth rate")
	if err != nil {
		return nil, err
	}
	indicator.Importance = "high"
	return indicator, nil
}

// GetUSUnemployment retrieves US unemployment data
func (c *TradingEconomicsClient) GetUSUnemployment(ctx context.Context) (*entity.EconomicIndicator, error) {
	indicator, err := c.GetIndicator(ctx, "united states", "unemployment rate")
	if err != nil {
		return nil, err
	}
	indicator.Importance = "high"
	return indicator, nil
}

// GetUSPCE retrieves US PCE (Fed's preferred inflation measure)
func (c *TradingEconomicsClient) GetUSPCE(ctx context.Context) (*entity.EconomicIndicator, error) {
	indicator, err := c.GetIndicator(ctx, "united states", "core pce price index annual change")
	if err != nil {
		return nil, err
	}
	indicator.Importance = "high"
	return indicator, nil
}

// CalendarResponse represents economic calendar response
type CalendarResponse []struct {
	ID          string  `json:"CalendarId"`
	Date        string  `json:"Date"`
	Country     string  `json:"Country"`
	Category    string  `json:"Category"`
	Event       string  `json:"Event"`
	Actual      *float64 `json:"Actual"`
	Previous    float64 `json:"Previous"`
	Forecast    float64 `json:"Forecast"`
	Importance  int     `json:"Importance"` // 1=low, 2=medium, 3=high
}

// GetEconomicCalendar retrieves upcoming economic events
func (c *TradingEconomicsClient) GetEconomicCalendar(ctx context.Context, country string, days int) ([]*entity.EconomicEvent, error) {
	startDate := time.Now().Format("2006-01-02")
	endDate := time.Now().AddDate(0, 0, days).Format("2006-01-02")

	endpoint := fmt.Sprintf("/calendar/country/%s/%s/%s",
		url.PathEscape(country), startDate, endDate)

	body, err := c.doRequest(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var resp CalendarResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	events := make([]*entity.EconomicEvent, 0, len(resp))
	for _, item := range resp {
		eventDate, _ := time.Parse("2006-01-02T15:04:05", item.Date)

		importance := "low"
		if item.Importance == 2 {
			importance = "medium"
		} else if item.Importance == 3 {
			importance = "high"
		}

		// Determine impact based on actual vs forecast
		impact := "neutral"
		if item.Actual != nil && item.Forecast != 0 {
			if *item.Actual > item.Forecast {
				impact = "positive"
			} else if *item.Actual < item.Forecast {
				impact = "negative"
			}
		}

		events = append(events, &entity.EconomicEvent{
			ID:         item.ID,
			Country:    item.Country,
			Category:   item.Category,
			Event:      item.Event,
			Date:       eventDate,
			Actual:     item.Actual,
			Previous:   item.Previous,
			Forecast:   item.Forecast,
			Importance: importance,
			Impact:     impact,
		})
	}

	return events, nil
}

// GetHighImpactEvents returns only high-importance events
func (c *TradingEconomicsClient) GetHighImpactEvents(ctx context.Context, days int) ([]*entity.EconomicEvent, error) {
	events, err := c.GetEconomicCalendar(ctx, "united states", days)
	if err != nil {
		return nil, err
	}

	highImpact := make([]*entity.EconomicEvent, 0)
	for _, event := range events {
		if event.Importance == "high" {
			highImpact = append(highImpact, event)
		}
	}

	return highImpact, nil
}

// SubscribeIndicators subscribes to indicator updates (polling)
func (c *TradingEconomicsClient) SubscribeIndicators(ctx context.Context, handler func(*entity.MacroSignal)) error {
	go func() {
		// Economic data updates infrequently, check every 15 minutes
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				signal := c.buildMacroSignal(ctx)
				if signal != nil {
					handler(signal)
				}
			}
		}
	}()

	return nil
}

// buildMacroSignal builds a macro signal from all indicators
func (c *TradingEconomicsClient) buildMacroSignal(ctx context.Context) *entity.MacroSignal {
	signal := &entity.MacroSignal{
		Timestamp: time.Now(),
	}

	// Get key indicators
	if cpi, err := c.GetUSInflation(ctx); err == nil {
		signal.CPI = cpi
	}
	if gdp, err := c.GetUSGDP(ctx); err == nil {
		signal.GDP = gdp
	}
	if unemp, err := c.GetUSUnemployment(ctx); err == nil {
		signal.Unemployment = unemp
	}
	if pce, err := c.GetUSPCE(ctx); err == nil {
		signal.PCE = pce
	}

	// Get upcoming events
	if events, err := c.GetHighImpactEvents(ctx, 7); err == nil {
		signal.UpcomingEvents = events
	}

	signal.AnalyzeMacroSignal()

	return signal
}

// FormatIndicatorSummary returns a human-readable summary
func FormatIndicatorSummary(indicator *entity.EconomicIndicator) string {
	if indicator == nil {
		return "No data"
	}

	change := ""
	if indicator.Previous != 0 {
		diff := indicator.Value - indicator.Previous
		if diff > 0 {
			change = fmt.Sprintf(" (+%.2f)", diff)
		} else if diff < 0 {
			change = fmt.Sprintf(" (%.2f)", diff)
		}
	}

	return fmt.Sprintf("%s: %.2f%s%s (prev: %.2f)",
		indicator.Name, indicator.Value, indicator.Unit, change, indicator.Previous)
}
