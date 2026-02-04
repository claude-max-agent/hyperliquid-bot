package lunarcrush

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/zono819/hyperliquid-bot/internal/domain/entity"
)

const (
	baseURL = "https://lunarcrush.com/api4"
)

// Client is a LunarCrush API v4 client
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new LunarCrush client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Connect validates API key
func (c *Client) Connect(ctx context.Context) error {
	_, err := c.GetSentiment(ctx, "bitcoin")
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

// TopicResponse represents LunarCrush topic API response
type TopicResponse struct {
	Data TopicData `json:"data"`
}

// TopicData represents topic details
type TopicData struct {
	Topic               string  `json:"topic"`
	TopicRank           int     `json:"topic_rank"`
	NumPosts            int     `json:"num_posts"`
	NumContributors     int     `json:"num_contributors"`
	Interactions24h     int64   `json:"interactions_24h"`
	InteractionsTotal   int64   `json:"interactions_total"`
	Sentiment           float64 `json:"sentiment"` // 0-100, 50 = neutral
	GalaxyScore         float64 `json:"galaxy_score"`
	AltRank             int     `json:"alt_rank"`
	MarketCap           float64 `json:"market_cap"`
	Price               float64 `json:"price"`
	PriceChange24h      float64 `json:"percent_change_24h"`
	Volume24h           float64 `json:"volume_24h"`
	TypesSentimentDetail SentimentDetail `json:"types_sentiment_detail"`
}

// SentimentDetail represents sentiment breakdown by platform
type SentimentDetail struct {
	Twitter  PlatformSentiment `json:"twitter"`
	Reddit   PlatformSentiment `json:"reddit"`
	YouTube  PlatformSentiment `json:"youtube"`
	TikTok   PlatformSentiment `json:"tiktok"`
	News     PlatformSentiment `json:"news"`
}

// PlatformSentiment represents sentiment for a specific platform
type PlatformSentiment struct {
	Positive int `json:"positive"`
	Neutral  int `json:"neutral"`
	Negative int `json:"negative"`
}

// GetSentiment retrieves sentiment data for a crypto topic
func (c *Client) GetSentiment(ctx context.Context, symbol string) (*entity.SocialSentiment, error) {
	topic := symbolToTopic(symbol)
	body, err := c.doRequest(ctx, "/public/topic/"+topic+"/v1")
	if err != nil {
		return nil, err
	}

	var resp TopicResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	data := resp.Data

	// Calculate aggregated sentiment from all platforms
	totalPositive := data.TypesSentimentDetail.Twitter.Positive +
		data.TypesSentimentDetail.Reddit.Positive +
		data.TypesSentimentDetail.YouTube.Positive +
		data.TypesSentimentDetail.News.Positive

	totalNegative := data.TypesSentimentDetail.Twitter.Negative +
		data.TypesSentimentDetail.Reddit.Negative +
		data.TypesSentimentDetail.YouTube.Negative +
		data.TypesSentimentDetail.News.Negative

	totalNeutral := data.TypesSentimentDetail.Twitter.Neutral +
		data.TypesSentimentDetail.Reddit.Neutral +
		data.TypesSentimentDetail.YouTube.Neutral +
		data.TypesSentimentDetail.News.Neutral

	total := totalPositive + totalNegative + totalNeutral
	if total == 0 {
		total = 1 // Avoid division by zero
	}

	return &entity.SocialSentiment{
		Symbol:           symbol,
		Source:           "lunarcrush",
		Sentiment:        data.Sentiment / 100.0, // Convert to 0-1 scale
		SentimentScore:   (data.Sentiment - 50) / 50.0, // Convert to -1 to 1 scale
		PositiveRatio:    float64(totalPositive) / float64(total),
		NegativeRatio:    float64(totalNegative) / float64(total),
		NeutralRatio:     float64(totalNeutral) / float64(total),
		SocialVolume:     int64(data.NumPosts),
		Interactions:     data.Interactions24h,
		Contributors:     int64(data.NumContributors),
		GalaxyScore:      data.GalaxyScore,
		AltRank:          data.AltRank,
		PlatformBreakdown: map[string]entity.PlatformMetrics{
			"twitter": {
				Positive: data.TypesSentimentDetail.Twitter.Positive,
				Neutral:  data.TypesSentimentDetail.Twitter.Neutral,
				Negative: data.TypesSentimentDetail.Twitter.Negative,
			},
			"reddit": {
				Positive: data.TypesSentimentDetail.Reddit.Positive,
				Neutral:  data.TypesSentimentDetail.Reddit.Neutral,
				Negative: data.TypesSentimentDetail.Reddit.Negative,
			},
			"youtube": {
				Positive: data.TypesSentimentDetail.YouTube.Positive,
				Neutral:  data.TypesSentimentDetail.YouTube.Neutral,
				Negative: data.TypesSentimentDetail.YouTube.Negative,
			},
			"news": {
				Positive: data.TypesSentimentDetail.News.Positive,
				Neutral:  data.TypesSentimentDetail.News.Neutral,
				Negative: data.TypesSentimentDetail.News.Negative,
			},
		},
		Timestamp: time.Now(),
	}, nil
}

// TimeSeriesResponse represents time series API response
type TimeSeriesResponse struct {
	Data []TimeSeriesPoint `json:"data"`
}

// TimeSeriesPoint represents a single time series data point
type TimeSeriesPoint struct {
	Time            int64   `json:"time"`
	Sentiment       float64 `json:"sentiment"`
	Interactions    int64   `json:"interactions"`
	NumPosts        int     `json:"num_posts"`
	NumContributors int     `json:"num_contributors"`
	Price           float64 `json:"price"`
	Volume          float64 `json:"volume"`
	MarketCap       float64 `json:"market_cap"`
}

// GetSentimentHistory retrieves historical sentiment data
func (c *Client) GetSentimentHistory(ctx context.Context, symbol string, interval string, limit int) ([]*entity.SocialSentiment, error) {
	topic := symbolToTopic(symbol)
	endpoint := fmt.Sprintf("/public/topic/%s/time-series/v2?interval=%s&limit=%d", topic, interval, limit)

	body, err := c.doRequest(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var resp TimeSeriesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	sentiments := make([]*entity.SocialSentiment, 0, len(resp.Data))
	for _, point := range resp.Data {
		sentiments = append(sentiments, &entity.SocialSentiment{
			Symbol:         symbol,
			Source:         "lunarcrush",
			Sentiment:      point.Sentiment / 100.0,
			SentimentScore: (point.Sentiment - 50) / 50.0,
			SocialVolume:   int64(point.NumPosts),
			Interactions:   point.Interactions,
			Contributors:   int64(point.NumContributors),
			Timestamp:      time.Unix(point.Time, 0),
		})
	}

	return sentiments, nil
}

// TrendingResponse represents trending topics response
type TrendingResponse struct {
	Data []TrendingTopic `json:"data"`
}

// TrendingTopic represents a trending topic
type TrendingTopic struct {
	Topic           string  `json:"topic"`
	TopicRank       int     `json:"topic_rank"`
	Sentiment       float64 `json:"sentiment"`
	Interactions24h int64   `json:"interactions_24h"`
	NumPosts        int     `json:"num_posts"`
}

// GetTrendingTopics retrieves trending crypto topics
func (c *Client) GetTrendingTopics(ctx context.Context, limit int) ([]*entity.TrendingTopic, error) {
	endpoint := fmt.Sprintf("/public/topics/list/v1?limit=%d", limit)

	body, err := c.doRequest(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var resp TrendingResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	topics := make([]*entity.TrendingTopic, 0, len(resp.Data))
	for _, t := range resp.Data {
		topics = append(topics, &entity.TrendingTopic{
			Topic:        t.Topic,
			Rank:         t.TopicRank,
			Sentiment:    t.Sentiment / 100.0,
			Interactions: t.Interactions24h,
			PostCount:    t.NumPosts,
			Timestamp:    time.Now(),
		})
	}

	return topics, nil
}

// SubscribeSentiment subscribes to sentiment updates (polling)
func (c *Client) SubscribeSentiment(ctx context.Context, symbol string, handler func(*entity.SocialSentiment)) error {
	go func() {
		ticker := time.NewTicker(60 * time.Second) // LunarCrush rate limits
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sentiment, err := c.GetSentiment(ctx, symbol)
				if err != nil {
					continue
				}
				handler(sentiment)
			}
		}
	}()

	return nil
}

// symbolToTopic converts trading symbol to LunarCrush topic
func symbolToTopic(symbol string) string {
	topicMap := map[string]string{
		"BTC":  "bitcoin",
		"ETH":  "ethereum",
		"SOL":  "solana",
		"XRP":  "xrp",
		"DOGE": "dogecoin",
		"ADA":  "cardano",
		"AVAX": "avalanche",
		"DOT":  "polkadot",
		"LINK": "chainlink",
		"MATIC": "polygon",
	}

	if topic, ok := topicMap[strings.ToUpper(symbol)]; ok {
		return topic
	}
	return strings.ToLower(symbol)
}

// GetSentimentBias analyzes sentiment and returns trading bias
func GetSentimentBias(sentiment *entity.SocialSentiment) (entity.SignalBias, float64) {
	if sentiment == nil {
		return entity.SignalBiasNeutral, 0
	}

	// SentimentScore is -1 to 1, where negative is bearish, positive is bullish
	score := sentiment.SentimentScore

	// Also consider social volume (high volume = stronger signal)
	volumeMultiplier := 1.0
	if sentiment.Interactions > 1000000 {
		volumeMultiplier = 1.2
	} else if sentiment.Interactions > 100000 {
		volumeMultiplier = 1.1
	}

	adjustedScore := score * volumeMultiplier
	if adjustedScore > 1 {
		adjustedScore = 1
	} else if adjustedScore < -1 {
		adjustedScore = -1
	}

	if adjustedScore > 0.2 {
		return entity.SignalBiasBullish, adjustedScore
	} else if adjustedScore < -0.2 {
		return entity.SignalBiasBearish, -adjustedScore
	}
	return entity.SignalBiasNeutral, 0
}
