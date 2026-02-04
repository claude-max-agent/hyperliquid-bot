package signal

import (
	"context"
	"testing"
	"time"

	"github.com/zono819/hyperliquid-bot/internal/domain/entity"
)

func TestNewProvider(t *testing.T) {
	cfg := Config{
		Symbols: []string{"BTC", "ETH"},
	}

	provider := NewProvider(cfg)

	if provider == nil {
		t.Fatal("Expected provider to be created")
	}
	if len(provider.symbols) != 2 {
		t.Errorf("Expected 2 symbols, got %d", len(provider.symbols))
	}
}

func TestProvider_GetMarketSignal_NoDataSources(t *testing.T) {
	cfg := Config{
		Symbols: []string{"BTC"},
	}

	provider := NewProvider(cfg)
	ctx := context.Background()

	signal, err := provider.GetMarketSignal(ctx, "BTC")
	if err != nil {
		t.Fatalf("GetMarketSignal failed: %v", err)
	}

	if signal == nil {
		t.Fatal("Expected signal to be returned")
	}
	if signal.Symbol != "BTC" {
		t.Errorf("Expected symbol BTC, got %s", signal.Symbol)
	}
	if signal.Bias != entity.SignalBiasNeutral {
		t.Errorf("Expected neutral bias without data, got %s", signal.Bias)
	}

	t.Logf("Signal without data sources: Bias=%s, Strength=%.2f, Confidence=%.2f",
		signal.Bias, signal.Strength, signal.Confidence)
}

func TestProvider_onLiquidation(t *testing.T) {
	cfg := Config{
		Symbols: []string{"BTC"},
	}
	provider := NewProvider(cfg)

	liq := &entity.Liquidation{
		Symbol:    "BTC",
		Side:      "long",
		Price:     50000.0,
		Quantity:  1.5,
		Value:     75000.0,
		Exchange:  "binance",
		Timestamp: time.Now(),
	}

	provider.onLiquidation("BTC", liq)

	provider.mu.RLock()
	liqs := provider.recentLiquidations["BTC"]
	provider.mu.RUnlock()

	if len(liqs) != 1 {
		t.Errorf("Expected 1 liquidation, got %d", len(liqs))
	}
}

func TestProvider_onWhaleAlert(t *testing.T) {
	cfg := Config{
		Symbols: []string{"BTC"},
	}
	provider := NewProvider(cfg)

	alert := &entity.WhaleAlert{
		ID:         "test-123",
		Blockchain: "bitcoin",
		Symbol:     "BTC",
		Amount:     100.0,
		AmountUSD:  5000000.0,
		FromOwner:  "unknown",
		ToOwner:    "binance",
		Timestamp:  time.Now(),
	}

	provider.onWhaleAlert(alert)

	provider.mu.RLock()
	alerts := provider.recentWhaleAlerts["BTC"]
	provider.mu.RUnlock()

	if len(alerts) != 1 {
		t.Errorf("Expected 1 whale alert, got %d", len(alerts))
	}
}

func TestProvider_onSentimentUpdate(t *testing.T) {
	cfg := Config{
		Symbols: []string{"BTC"},
	}
	provider := NewProvider(cfg)

	sentiment := &entity.SocialSentiment{
		Symbol:         "BTC",
		SentimentScore: 0.5,
		SocialVolume:   10000,
		Interactions:   50000,
		Timestamp:      time.Now(),
	}

	provider.onSentimentUpdate("BTC", sentiment)

	provider.mu.RLock()
	cached := provider.recentSentiment["BTC"]
	provider.mu.RUnlock()

	if cached == nil {
		t.Fatal("Expected sentiment to be cached")
	}
	if cached.SentimentScore != 0.5 {
		t.Errorf("Expected sentiment score 0.5, got %f", cached.SentimentScore)
	}
}

func TestProvider_onMacroUpdate(t *testing.T) {
	cfg := Config{
		Symbols: []string{"BTC"},
	}
	provider := NewProvider(cfg)

	macroSignal := &entity.MacroSignal{
		Timestamp: time.Now(),
		FedWatch: &entity.FedWatchData{
			NextMeeting: &entity.FOMCMeeting{
				CutProb:  0.7,
				HikeProb: 0.1,
			},
		},
		Bias:       entity.SignalBiasBullish,
		Strength:   0.5,
		Confidence: 0.6,
	}

	provider.onMacroUpdate(macroSignal)

	provider.mu.RLock()
	cached := provider.cachedMacro
	provider.mu.RUnlock()

	if cached == nil {
		t.Fatal("Expected macro signal to be cached")
	}
	if cached.Bias != entity.SignalBiasBullish {
		t.Errorf("Expected bullish bias, got %s", cached.Bias)
	}
}

func TestProvider_GetMarketSignal_WithCachedData(t *testing.T) {
	cfg := Config{
		Symbols: []string{"BTC"},
	}
	provider := NewProvider(cfg)
	ctx := context.Background()

	// Add cached data
	provider.mu.Lock()
	provider.recentWhaleAlerts["BTC"] = []*entity.WhaleAlert{
		{FromOwner: "binance", ToOwner: "unknown", AmountUSD: 50000000, Timestamp: time.Now()},
	}
	provider.recentSentiment["BTC"] = &entity.SocialSentiment{
		SentimentScore: 0.4,
		Timestamp:      time.Now(),
	}
	provider.cachedMacro = &entity.MacroSignal{
		FedWatch: &entity.FedWatchData{
			NextMeeting: &entity.FOMCMeeting{
				CutProb:  0.6,
				HikeProb: 0.1,
			},
		},
		Bias:       entity.SignalBiasBullish,
		Strength:   0.4,
		Confidence: 0.5,
	}
	provider.mu.Unlock()

	signal, err := provider.GetMarketSignal(ctx, "BTC")
	if err != nil {
		t.Fatalf("GetMarketSignal failed: %v", err)
	}

	if signal.FedCutProb != 0.6 {
		t.Errorf("Expected FedCutProb 0.6, got %f", signal.FedCutProb)
	}
	if len(signal.RecentWhaleAlerts) != 1 {
		t.Errorf("Expected 1 whale alert, got %d", len(signal.RecentWhaleAlerts))
	}
	if signal.SocialSentiment == nil {
		t.Error("Expected social sentiment to be included")
	}

	t.Logf("Signal with cached data: Bias=%s, Strength=%.2f, Confidence=%.2f, FedCut=%.0f%%",
		signal.Bias, signal.Strength, signal.Confidence, signal.FedCutProb*100)
}

func TestMapBlockchainToSymbol(t *testing.T) {
	tests := []struct {
		blockchain string
		expected   string
	}{
		{"bitcoin", "BTC"},
		{"ethereum", "ETH"},
		{"solana", "SOL"},
		{"tron", "TRX"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.blockchain, func(t *testing.T) {
			got := mapBlockchainToSymbol(tt.blockchain)
			if got != tt.expected {
				t.Errorf("mapBlockchainToSymbol(%s) = %s, want %s",
					tt.blockchain, got, tt.expected)
			}
		})
	}
}

func TestGetSignalSummary(t *testing.T) {
	signal := &entity.MarketSignal{
		Symbol:     "BTC",
		Timestamp:  time.Now(),
		Bias:       entity.SignalBiasBullish,
		Strength:   0.6,
		Confidence: 0.7,
		FundingRate: &entity.FundingRate{
			Rate: -0.0003,
		},
		LongShortRatio: &entity.LongShortRatio{
			LongShortRatio: 0.8,
		},
		RecentWhaleAlerts: []*entity.WhaleAlert{
			{FromOwner: "binance", ToOwner: "unknown", AmountUSD: 50000000},
		},
		SocialSentiment: &entity.SocialSentiment{
			SentimentScore: 0.4,
			SocialVolume:   10000,
			Interactions:   50000,
		},
	}

	summary := GetSignalSummary(signal)

	if summary == "" {
		t.Error("Expected non-empty summary")
	}
	if len(summary) < 50 {
		t.Errorf("Summary seems too short: %s", summary)
	}

	t.Logf("Signal summary:\n%s", summary)
}

func TestGetSignalSummary_Nil(t *testing.T) {
	summary := GetSignalSummary(nil)
	if summary != "No signal available" {
		t.Errorf("Expected 'No signal available', got '%s'", summary)
	}
}

func TestProvider_SubscribeSignals(t *testing.T) {
	cfg := Config{
		Symbols: []string{"BTC"},
	}
	provider := NewProvider(cfg)
	ctx := context.Background()

	received := make(chan *entity.MarketSignal, 1)
	handler := func(signal *entity.MarketSignal) {
		received <- signal
	}

	err := provider.SubscribeSignals(ctx, handler)
	if err != nil {
		t.Fatalf("SubscribeSignals failed: %v", err)
	}

	// Manually broadcast a signal
	testSignal := &entity.MarketSignal{
		Symbol:    "BTC",
		Timestamp: time.Now(),
		Bias:      entity.SignalBiasBullish,
	}
	provider.broadcastSignal(testSignal)

	select {
	case sig := <-received:
		if sig.Symbol != "BTC" {
			t.Errorf("Expected BTC signal, got %s", sig.Symbol)
		}
	case <-time.After(time.Second):
		t.Error("Did not receive signal within timeout")
	}
}
