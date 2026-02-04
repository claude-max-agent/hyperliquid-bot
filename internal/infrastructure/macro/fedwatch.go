package macro

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/zono819/hyperliquid-bot/internal/domain/entity"
)

const (
	fedWatchBaseURL = "https://markets.api.cmegroup.com/fedwatch/v1"
)

// FedWatchClient is a CME FedWatch API client
type FedWatchClient struct {
	apiKey     string
	httpClient *http.Client
}

// NewFedWatchClient creates a new FedWatch client
func NewFedWatchClient(apiKey string) *FedWatchClient {
	return &FedWatchClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Connect validates API connection
func (c *FedWatchClient) Connect(ctx context.Context) error {
	_, err := c.GetFedWatchData(ctx)
	return err
}

// Disconnect closes connection
func (c *FedWatchClient) Disconnect(ctx context.Context) error {
	return nil
}

// doRequest performs authenticated HTTP request
func (c *FedWatchClient) doRequest(ctx context.Context, endpoint string) ([]byte, error) {
	url := fedWatchBaseURL + endpoint

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
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

// ForecastResponse represents CME FedWatch API response
type ForecastResponse struct {
	Forecasts []Forecast `json:"forecasts"`
}

// Forecast represents a single meeting forecast
type Forecast struct {
	MeetingDate   string        `json:"meetingDate"`
	CurrentRate   float64       `json:"currentRate"`
	Probabilities []Probability `json:"probabilities"`
}

// Probability represents rate probability
type Probability struct {
	Rate        float64 `json:"rate"`
	Probability float64 `json:"probability"`
}

// GetFedWatchData retrieves current FedWatch data
func (c *FedWatchClient) GetFedWatchData(ctx context.Context) (*entity.FedWatchData, error) {
	body, err := c.doRequest(ctx, "/forecasts")
	if err != nil {
		return nil, err
	}

	var resp ForecastResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(resp.Forecasts) == 0 {
		return nil, fmt.Errorf("no forecast data available")
	}

	data := &entity.FedWatchData{
		CurrentRate:      resp.Forecasts[0].CurrentRate,
		UpcomingMeetings: make([]*entity.FOMCMeeting, 0, len(resp.Forecasts)),
		Timestamp:        time.Now(),
	}

	for _, forecast := range resp.Forecasts {
		meeting, err := parseForecast(forecast)
		if err != nil {
			continue
		}
		data.UpcomingMeetings = append(data.UpcomingMeetings, meeting)
	}

	// Sort by date and set next meeting
	sort.Slice(data.UpcomingMeetings, func(i, j int) bool {
		return data.UpcomingMeetings[i].MeetingDate.Before(data.UpcomingMeetings[j].MeetingDate)
	})

	// Find next meeting (first meeting after now)
	now := time.Now()
	for _, meeting := range data.UpcomingMeetings {
		if meeting.MeetingDate.After(now) {
			data.NextMeeting = meeting
			break
		}
	}

	return data, nil
}

// parseForecast converts API forecast to entity
func parseForecast(f Forecast) (*entity.FOMCMeeting, error) {
	meetingDate, err := time.Parse("2006-01-02", f.MeetingDate)
	if err != nil {
		return nil, err
	}

	meeting := &entity.FOMCMeeting{
		MeetingDate:   meetingDate,
		CurrentRate:   f.CurrentRate,
		Probabilities: make(map[float64]float64),
		Timestamp:     time.Now(),
	}

	var maxProb float64
	var maxProbRate float64

	for _, p := range f.Probabilities {
		meeting.Probabilities[p.Rate] = p.Probability

		if p.Probability > maxProb {
			maxProb = p.Probability
			maxProbRate = p.Rate
		}

		// Calculate hike/cut/hold probabilities
		if p.Rate > f.CurrentRate {
			meeting.HikeProb += p.Probability
		} else if p.Rate < f.CurrentRate {
			meeting.CutProb += p.Probability
		} else {
			meeting.HoldProb += p.Probability
		}
	}

	meeting.MostLikelyRate = maxProbRate
	meeting.MostLikelyProb = maxProb
	meeting.RateChangeProb = meeting.HikeProb + meeting.CutProb

	return meeting, nil
}

// GetNextMeetingProbabilities returns probabilities for the next FOMC meeting
func (c *FedWatchClient) GetNextMeetingProbabilities(ctx context.Context) (*entity.FOMCMeeting, error) {
	data, err := c.GetFedWatchData(ctx)
	if err != nil {
		return nil, err
	}

	if data.NextMeeting == nil {
		return nil, fmt.Errorf("no upcoming meeting found")
	}

	return data.NextMeeting, nil
}

// SubscribeFedWatch subscribes to FedWatch updates (polling)
func (c *FedWatchClient) SubscribeFedWatch(ctx context.Context, handler func(*entity.FedWatchData)) error {
	go func() {
		// FedWatch updates every 60 seconds for real-time, EOD at 01:45 UTC
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				data, err := c.GetFedWatchData(ctx)
				if err != nil {
					continue
				}
				handler(data)
			}
		}
	}()

	return nil
}

// FormatFedWatchSummary returns a human-readable summary
func FormatFedWatchSummary(data *entity.FedWatchData) string {
	if data == nil || data.NextMeeting == nil {
		return "FedWatch: No data available"
	}

	m := data.NextMeeting
	summary := fmt.Sprintf("FedWatch - Next FOMC: %s\n", m.MeetingDate.Format("Jan 2, 2006"))
	summary += fmt.Sprintf("  Current Rate: %.2f%%\n", data.CurrentRate*100)
	summary += fmt.Sprintf("  Most Likely: %.2f%% (%.1f%% prob)\n", m.MostLikelyRate*100, m.MostLikelyProb*100)
	summary += fmt.Sprintf("  Rate Cut: %.1f%% | Hold: %.1f%% | Hike: %.1f%%",
		m.CutProb*100, m.HoldProb*100, m.HikeProb*100)

	return summary
}
