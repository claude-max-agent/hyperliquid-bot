package entity

import (
	"testing"
	"time"
)

func TestMarketSignal_AnalyzeSignal_Bullish(t *testing.T) {
	signal := &MarketSignal{
		Symbol:    "BTC",
		Timestamp: time.Now(),
		// Negative funding rate = bullish
		FundingRate: &FundingRate{
			Rate: -0.0005,
		},
		// Low long/short ratio = bullish (shorts overcrowded)
		LongShortRatio: &LongShortRatio{
			LongShortRatio: 0.5,
		},
		// Whale outflow > inflow = bullish
		RecentWhaleAlerts: []*WhaleAlert{
			{FromOwner: "binance", ToOwner: "unknown", AmountUSD: 50000000}, // Outflow
			{FromOwner: "unknown", ToOwner: "binance", AmountUSD: 10000000}, // Inflow
		},
		// Bullish sentiment
		SocialSentiment: &SocialSentiment{
			SentimentScore: 0.5,
		},
		// Fed cut probability high = bullish
		FedCutProb:  0.7,
		FedHikeProb: 0.1,
	}

	signal.AnalyzeSignal()

	if signal.Bias != SignalBiasBullish {
		t.Errorf("Expected bullish bias, got %s", signal.Bias)
	}
	if signal.Strength <= 0 {
		t.Errorf("Expected positive strength, got %f", signal.Strength)
	}
	if signal.Confidence <= 0 {
		t.Errorf("Expected positive confidence, got %f", signal.Confidence)
	}

	t.Logf("Bullish signal: Bias=%s, Strength=%.2f, Confidence=%.2f",
		signal.Bias, signal.Strength, signal.Confidence)
}

func TestMarketSignal_AnalyzeSignal_Bearish(t *testing.T) {
	signal := &MarketSignal{
		Symbol:    "BTC",
		Timestamp: time.Now(),
		// High positive funding rate = bearish
		FundingRate: &FundingRate{
			Rate: 0.001,
		},
		// High long/short ratio = bearish (longs overcrowded)
		LongShortRatio: &LongShortRatio{
			LongShortRatio: 2.0,
		},
		// Whale inflow > outflow = bearish
		RecentWhaleAlerts: []*WhaleAlert{
			{FromOwner: "unknown", ToOwner: "binance", AmountUSD: 100000000}, // Inflow
			{FromOwner: "binance", ToOwner: "unknown", AmountUSD: 20000000},  // Outflow
		},
		// Bearish sentiment
		SocialSentiment: &SocialSentiment{
			SentimentScore: -0.5,
		},
		// Fed hike probability high = bearish
		FedCutProb:  0.1,
		FedHikeProb: 0.6,
	}

	signal.AnalyzeSignal()

	if signal.Bias != SignalBiasBearish {
		t.Errorf("Expected bearish bias, got %s", signal.Bias)
	}
	if signal.Strength <= 0 {
		t.Errorf("Expected positive strength, got %f", signal.Strength)
	}

	t.Logf("Bearish signal: Bias=%s, Strength=%.2f, Confidence=%.2f",
		signal.Bias, signal.Strength, signal.Confidence)
}

func TestMarketSignal_AnalyzeSignal_Neutral(t *testing.T) {
	signal := &MarketSignal{
		Symbol:    "BTC",
		Timestamp: time.Now(),
	}

	signal.AnalyzeSignal()

	if signal.Bias != SignalBiasNeutral {
		t.Errorf("Expected neutral bias with no data, got %s", signal.Bias)
	}
	if signal.Strength != 0 {
		t.Errorf("Expected zero strength with no data, got %f", signal.Strength)
	}

	t.Logf("Neutral signal: Bias=%s, Strength=%.2f, Confidence=%.2f",
		signal.Bias, signal.Strength, signal.Confidence)
}

func TestWhaleAlert_GetAlertType(t *testing.T) {
	tests := []struct {
		name     string
		alert    *WhaleAlert
		expected WhaleAlertType
	}{
		{
			name: "Exchange inflow",
			alert: &WhaleAlert{
				FromOwner: "unknown",
				ToOwner:   "binance",
			},
			expected: WhaleAlertExchangeInflow,
		},
		{
			name: "Exchange outflow",
			alert: &WhaleAlert{
				FromOwner: "coinbase",
				ToOwner:   "unknown",
			},
			expected: WhaleAlertExchangeOutflow,
		},
		{
			name: "Wallet transfer",
			alert: &WhaleAlert{
				FromOwner: "unknown",
				ToOwner:   "unknown",
			},
			expected: WhaleAlertWalletTransfer,
		},
		{
			name: "Exchange to exchange",
			alert: &WhaleAlert{
				FromOwner: "binance",
				ToOwner:   "coinbase",
			},
			expected: WhaleAlertUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.alert.GetAlertType()
			if got != tt.expected {
				t.Errorf("GetAlertType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMacroSignal_AnalyzeMacroSignal(t *testing.T) {
	t.Run("Bullish macro (rate cut expected)", func(t *testing.T) {
		signal := &MacroSignal{
			Timestamp: time.Now(),
			FedWatch: &FedWatchData{
				NextMeeting: &FOMCMeeting{
					CutProb:  0.8,
					HikeProb: 0.05,
					HoldProb: 0.15,
				},
			},
			CPI: &EconomicIndicator{
				Value:    2.5,
				Forecast: 3.0,
			},
		}

		signal.AnalyzeMacroSignal()

		if signal.Bias != SignalBiasBullish {
			t.Errorf("Expected bullish bias, got %s", signal.Bias)
		}
		t.Logf("Macro signal: Bias=%s, Strength=%.2f, Confidence=%.2f",
			signal.Bias, signal.Strength, signal.Confidence)
	})

	t.Run("Bearish macro (rate hike expected)", func(t *testing.T) {
		signal := &MacroSignal{
			Timestamp: time.Now(),
			FedWatch: &FedWatchData{
				NextMeeting: &FOMCMeeting{
					CutProb:  0.1,
					HikeProb: 0.6,
					HoldProb: 0.3,
				},
			},
			CPI: &EconomicIndicator{
				Value:    4.0,
				Forecast: 3.0,
			},
		}

		signal.AnalyzeMacroSignal()

		if signal.Bias != SignalBiasBearish {
			t.Errorf("Expected bearish bias, got %s", signal.Bias)
		}
		t.Logf("Macro signal: Bias=%s, Strength=%.2f, Confidence=%.2f",
			signal.Bias, signal.Strength, signal.Confidence)
	})
}
