package entity

import "time"

// Liquidation represents a liquidation event
type Liquidation struct {
	Symbol    string    `json:"symbol"`
	Side      string    `json:"side"` // "long" or "short"
	Price     float64   `json:"price"`
	Quantity  float64   `json:"quantity"`
	Value     float64   `json:"value"` // USD value
	Exchange  string    `json:"exchange"`
	Timestamp time.Time `json:"timestamp"`
}

// OpenInterest represents open interest data
type OpenInterest struct {
	Symbol      string    `json:"symbol"`
	OpenInterest float64  `json:"open_interest"`
	Change24h   float64   `json:"change_24h"` // percentage
	Exchange    string    `json:"exchange"`
	Timestamp   time.Time `json:"timestamp"`
}

// FundingRate represents funding rate data
type FundingRate struct {
	Symbol          string    `json:"symbol"`
	Rate            float64   `json:"rate"`
	PredictedRate   float64   `json:"predicted_rate"`
	NextFundingTime time.Time `json:"next_funding_time"`
	Exchange        string    `json:"exchange"`
	Timestamp       time.Time `json:"timestamp"`
}

// LongShortRatio represents long/short position ratio
type LongShortRatio struct {
	Symbol        string    `json:"symbol"`
	LongRatio     float64   `json:"long_ratio"`
	ShortRatio    float64   `json:"short_ratio"`
	LongShortRatio float64  `json:"long_short_ratio"`
	Exchange      string    `json:"exchange"`
	Timestamp     time.Time `json:"timestamp"`
}

// WhaleAlert represents a large transaction alert
type WhaleAlert struct {
	ID          string    `json:"id"`
	Blockchain  string    `json:"blockchain"`
	Symbol      string    `json:"symbol"`
	Amount      float64   `json:"amount"`
	AmountUSD   float64   `json:"amount_usd"`
	FromAddress string    `json:"from_address"`
	ToAddress   string    `json:"to_address"`
	FromOwner   string    `json:"from_owner"` // e.g., "Binance", "unknown"
	ToOwner     string    `json:"to_owner"`
	TxHash      string    `json:"tx_hash"`
	Timestamp   time.Time `json:"timestamp"`
}

// WhaleAlertType categorizes whale movements
type WhaleAlertType string

const (
	WhaleAlertExchangeInflow  WhaleAlertType = "exchange_inflow"  // Deposit to exchange (bearish)
	WhaleAlertExchangeOutflow WhaleAlertType = "exchange_outflow" // Withdrawal from exchange (bullish)
	WhaleAlertWalletTransfer  WhaleAlertType = "wallet_transfer"  // Wallet to wallet
	WhaleAlertUnknown         WhaleAlertType = "unknown"
)

// GetAlertType determines the type of whale alert
func (w *WhaleAlert) GetAlertType() WhaleAlertType {
	exchanges := map[string]bool{
		"binance": true, "coinbase": true, "kraken": true,
		"bitfinex": true, "bybit": true, "okx": true,
		"huobi": true, "kucoin": true, "gate.io": true,
	}

	fromIsExchange := exchanges[w.FromOwner]
	toIsExchange := exchanges[w.ToOwner]

	switch {
	case !fromIsExchange && toIsExchange:
		return WhaleAlertExchangeInflow
	case fromIsExchange && !toIsExchange:
		return WhaleAlertExchangeOutflow
	case !fromIsExchange && !toIsExchange:
		return WhaleAlertWalletTransfer
	default:
		return WhaleAlertUnknown
	}
}

// SocialSentiment represents social media sentiment data
type SocialSentiment struct {
	Symbol            string                     `json:"symbol"`
	Source            string                     `json:"source"` // "lunarcrush", "messari", etc.
	Sentiment         float64                    `json:"sentiment"` // 0-1 scale, 0.5 = neutral
	SentimentScore    float64                    `json:"sentiment_score"` // -1 to 1, negative = bearish
	PositiveRatio     float64                    `json:"positive_ratio"`
	NegativeRatio     float64                    `json:"negative_ratio"`
	NeutralRatio      float64                    `json:"neutral_ratio"`
	SocialVolume      int64                      `json:"social_volume"` // Number of posts
	Interactions      int64                      `json:"interactions"` // Total interactions
	Contributors      int64                      `json:"contributors"` // Unique contributors
	GalaxyScore       float64                    `json:"galaxy_score,omitempty"` // LunarCrush proprietary
	AltRank           int                        `json:"alt_rank,omitempty"` // LunarCrush proprietary
	PlatformBreakdown map[string]PlatformMetrics `json:"platform_breakdown,omitempty"`
	Timestamp         time.Time                  `json:"timestamp"`
}

// PlatformMetrics represents sentiment metrics for a specific platform
type PlatformMetrics struct {
	Positive int `json:"positive"`
	Neutral  int `json:"neutral"`
	Negative int `json:"negative"`
}

// TrendingTopic represents a trending social topic
type TrendingTopic struct {
	Topic        string    `json:"topic"`
	Rank         int       `json:"rank"`
	Sentiment    float64   `json:"sentiment"` // 0-1 scale
	Interactions int64     `json:"interactions"`
	PostCount    int       `json:"post_count"`
	Timestamp    time.Time `json:"timestamp"`
}

// MarketSignal represents aggregated market signal for trading decisions
type MarketSignal struct {
	Symbol    string    `json:"symbol"`
	Timestamp time.Time `json:"timestamp"`

	// Derivatives data
	OpenInterest     *OpenInterest   `json:"open_interest,omitempty"`
	FundingRate      *FundingRate    `json:"funding_rate,omitempty"`
	LongShortRatio   *LongShortRatio `json:"long_short_ratio,omitempty"`
	RecentLiquidations []*Liquidation `json:"recent_liquidations,omitempty"`

	// Whale activity
	RecentWhaleAlerts []*WhaleAlert `json:"recent_whale_alerts,omitempty"`

	// Social sentiment
	SocialSentiment *SocialSentiment `json:"social_sentiment,omitempty"`

	// Aggregated signals
	Bias       SignalBias `json:"bias"`       // overall market bias
	Strength   float64    `json:"strength"`   // signal strength (0-1)
	Confidence float64    `json:"confidence"` // confidence level (0-1)
}

// SignalBias represents market direction bias
type SignalBias string

const (
	SignalBiasBullish SignalBias = "bullish"
	SignalBiasBearish SignalBias = "bearish"
	SignalBiasNeutral SignalBias = "neutral"
)

// AnalyzeSignal analyzes the market signal and sets bias, strength, confidence
func (s *MarketSignal) AnalyzeSignal() {
	var bullishScore, bearishScore float64
	var dataPoints int

	// Analyze funding rate
	if s.FundingRate != nil {
		dataPoints++
		if s.FundingRate.Rate > 0.0001 { // High positive = bearish (shorts pay longs)
			bearishScore += 0.3
		} else if s.FundingRate.Rate < -0.0001 { // Negative = bullish
			bullishScore += 0.3
		}
	}

	// Analyze long/short ratio
	if s.LongShortRatio != nil {
		dataPoints++
		if s.LongShortRatio.LongShortRatio > 1.5 { // Too many longs = bearish
			bearishScore += 0.2
		} else if s.LongShortRatio.LongShortRatio < 0.7 { // Too many shorts = bullish
			bullishScore += 0.2
		}
	}

	// Analyze whale alerts
	if len(s.RecentWhaleAlerts) > 0 {
		dataPoints++
		var inflowValue, outflowValue float64
		for _, alert := range s.RecentWhaleAlerts {
			switch alert.GetAlertType() {
			case WhaleAlertExchangeInflow:
				inflowValue += alert.AmountUSD
			case WhaleAlertExchangeOutflow:
				outflowValue += alert.AmountUSD
			}
		}
		if inflowValue > outflowValue*1.5 {
			bearishScore += 0.3
		} else if outflowValue > inflowValue*1.5 {
			bullishScore += 0.3
		}
	}

	// Analyze recent liquidations
	if len(s.RecentLiquidations) > 0 {
		dataPoints++
		var longLiqValue, shortLiqValue float64
		for _, liq := range s.RecentLiquidations {
			if liq.Side == "long" {
				longLiqValue += liq.Value
			} else {
				shortLiqValue += liq.Value
			}
		}
		// Cascade liquidations often continue
		if longLiqValue > shortLiqValue*2 {
			bearishScore += 0.2
		} else if shortLiqValue > longLiqValue*2 {
			bullishScore += 0.2
		}
	}

	// Analyze social sentiment
	if s.SocialSentiment != nil {
		dataPoints++
		score := s.SocialSentiment.SentimentScore // -1 to 1
		if score > 0.2 {
			bullishScore += 0.25 * score
		} else if score < -0.2 {
			bearishScore += 0.25 * (-score)
		}
	}

	// Calculate final signal
	totalScore := bullishScore + bearishScore
	if totalScore == 0 {
		s.Bias = SignalBiasNeutral
		s.Strength = 0
		s.Confidence = 0
		return
	}

	if bullishScore > bearishScore {
		s.Bias = SignalBiasBullish
		s.Strength = (bullishScore - bearishScore) / totalScore
	} else if bearishScore > bullishScore {
		s.Bias = SignalBiasBearish
		s.Strength = (bearishScore - bullishScore) / totalScore
	} else {
		s.Bias = SignalBiasNeutral
		s.Strength = 0
	}

	// Confidence based on data availability (5 possible data sources)
	s.Confidence = float64(dataPoints) / 5.0
}
