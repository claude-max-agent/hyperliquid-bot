package strategy

import (
	"context"
	"testing"
	"time"

	"github.com/zono819/hyperliquid-bot/internal/domain/entity"
	"github.com/zono819/hyperliquid-bot/internal/domain/service"
)

func TestAISignalStrategy_Name(t *testing.T) {
	s := NewAISignalStrategy()
	if s.Name() != "ai_signal" {
		t.Errorf("Expected name 'ai_signal', got '%s'", s.Name())
	}
}

func TestAISignalStrategy_Init(t *testing.T) {
	s := NewAISignalStrategy()
	ctx := context.Background()

	config := map[string]interface{}{
		"max_position_size":   2000.0,
		"min_signal_strength": 0.4,
		"min_confidence":      0.5,
		"take_profit_percent": 0.03,
		"stop_loss_percent":   0.015,
	}

	err := s.Init(ctx, config)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if s.config.MaxPositionSize != 2000.0 {
		t.Errorf("MaxPositionSize not set correctly: %f", s.config.MaxPositionSize)
	}
	if s.config.MinSignalStrength != 0.4 {
		t.Errorf("MinSignalStrength not set correctly: %f", s.config.MinSignalStrength)
	}
}

func TestAISignalStrategy_OnTick_NoSignal(t *testing.T) {
	s := NewAISignalStrategy()
	ctx := context.Background()
	s.Init(ctx, nil)

	state := &service.MarketState{
		Ticker: &entity.Ticker{
			Symbol:    "BTC",
			LastPrice: 50000.0,
			Timestamp: time.Now(),
		},
	}

	signals, err := s.OnTick(ctx, state)
	if err != nil {
		t.Fatalf("OnTick failed: %v", err)
	}

	if len(signals) != 0 {
		t.Errorf("Expected no signals without market signal, got %d", len(signals))
	}
}

func TestAISignalStrategy_OnTick_BullishEntry(t *testing.T) {
	s := NewAISignalStrategy()
	ctx := context.Background()
	s.Init(ctx, nil)

	// Create a strong bullish signal
	marketSignal := &entity.MarketSignal{
		Symbol:     "BTC",
		Timestamp:  time.Now(),
		Bias:       entity.SignalBiasBullish,
		Strength:   0.6,
		Confidence: 0.7,
		FundingRate: &entity.FundingRate{
			Rate: -0.0003,
		},
		LongShortRatio: &entity.LongShortRatio{
			LongShortRatio: 0.6,
		},
	}

	state := &service.MarketState{
		Ticker: &entity.Ticker{
			Symbol:    "BTC",
			LastPrice: 50000.0,
			Timestamp: time.Now(),
		},
		MarketSignal: marketSignal,
	}

	signals, err := s.OnTick(ctx, state)
	if err != nil {
		t.Fatalf("OnTick failed: %v", err)
	}

	if len(signals) == 0 {
		t.Fatal("Expected entry signal for strong bullish market")
	}

	sig := signals[0]
	if sig.Side != entity.SideBuy {
		t.Errorf("Expected BUY side for bullish signal, got %s", sig.Side)
	}
	if sig.Quantity <= 0 {
		t.Errorf("Expected positive quantity, got %f", sig.Quantity)
	}

	t.Logf("Entry signal: %s %s @ %.2f x %.6f - %s",
		sig.Side, sig.Symbol, sig.Price, sig.Quantity, sig.Reason)
}

func TestAISignalStrategy_OnTick_BearishEntry(t *testing.T) {
	s := NewAISignalStrategy()
	ctx := context.Background()
	s.Init(ctx, nil)

	// Create a strong bearish signal
	marketSignal := &entity.MarketSignal{
		Symbol:     "BTC",
		Timestamp:  time.Now(),
		Bias:       entity.SignalBiasBearish,
		Strength:   0.5,
		Confidence: 0.6,
		FundingRate: &entity.FundingRate{
			Rate: 0.001,
		},
	}

	state := &service.MarketState{
		Ticker: &entity.Ticker{
			Symbol:    "BTC",
			LastPrice: 50000.0,
			Timestamp: time.Now(),
		},
		MarketSignal: marketSignal,
	}

	signals, err := s.OnTick(ctx, state)
	if err != nil {
		t.Fatalf("OnTick failed: %v", err)
	}

	if len(signals) == 0 {
		t.Fatal("Expected entry signal for strong bearish market")
	}

	sig := signals[0]
	if sig.Side != entity.SideSell {
		t.Errorf("Expected SELL side for bearish signal, got %s", sig.Side)
	}

	t.Logf("Entry signal: %s %s @ %.2f x %.6f - %s",
		sig.Side, sig.Symbol, sig.Price, sig.Quantity, sig.Reason)
}

func TestAISignalStrategy_OnTick_WeakSignalNoEntry(t *testing.T) {
	s := NewAISignalStrategy()
	ctx := context.Background()
	s.Init(ctx, nil)

	// Create a weak signal (below threshold)
	marketSignal := &entity.MarketSignal{
		Symbol:     "BTC",
		Timestamp:  time.Now(),
		Bias:       entity.SignalBiasBullish,
		Strength:   0.2, // Below default 0.3 threshold
		Confidence: 0.3, // Below default 0.4 threshold
	}

	state := &service.MarketState{
		Ticker: &entity.Ticker{
			Symbol:    "BTC",
			LastPrice: 50000.0,
			Timestamp: time.Now(),
		},
		MarketSignal: marketSignal,
	}

	signals, err := s.OnTick(ctx, state)
	if err != nil {
		t.Fatalf("OnTick failed: %v", err)
	}

	if len(signals) != 0 {
		t.Errorf("Expected no entry for weak signal, got %d signals", len(signals))
	}
}

func TestAISignalStrategy_TakeProfit(t *testing.T) {
	s := NewAISignalStrategy()
	ctx := context.Background()
	s.Init(ctx, nil)

	// Set up a long position
	position := &entity.Position{
		Symbol:     "BTC",
		Size:       0.01,
		EntryPrice: 50000.0,
		Side:       entity.SideBuy,
	}
	s.OnPositionUpdate(ctx, position)

	// Price increased beyond take profit (2%)
	state := &service.MarketState{
		Ticker: &entity.Ticker{
			Symbol:    "BTC",
			LastPrice: 51500.0, // +3% from entry
			Timestamp: time.Now(),
		},
		Position: position,
	}

	signals, err := s.OnTick(ctx, state)
	if err != nil {
		t.Fatalf("OnTick failed: %v", err)
	}

	if len(signals) == 0 {
		t.Fatal("Expected take profit signal")
	}

	sig := signals[0]
	if sig.Side != entity.SideSell {
		t.Errorf("Expected SELL for take profit on long, got %s", sig.Side)
	}
	if sig.Reason == "" || len(sig.Reason) < 5 {
		t.Errorf("Expected reason for exit, got '%s'", sig.Reason)
	}

	t.Logf("Take profit signal: %s @ %.2f - %s", sig.Side, sig.Price, sig.Reason)
}

func TestAISignalStrategy_StopLoss(t *testing.T) {
	s := NewAISignalStrategy()
	ctx := context.Background()
	s.Init(ctx, nil)

	// Set up a long position
	position := &entity.Position{
		Symbol:     "BTC",
		Size:       0.01,
		EntryPrice: 50000.0,
		Side:       entity.SideBuy,
	}
	s.OnPositionUpdate(ctx, position)

	// Price decreased beyond stop loss (1%)
	state := &service.MarketState{
		Ticker: &entity.Ticker{
			Symbol:    "BTC",
			LastPrice: 49000.0, // -2% from entry
			Timestamp: time.Now(),
		},
		Position: position,
	}

	signals, err := s.OnTick(ctx, state)
	if err != nil {
		t.Fatalf("OnTick failed: %v", err)
	}

	if len(signals) == 0 {
		t.Fatal("Expected stop loss signal")
	}

	sig := signals[0]
	if sig.Side != entity.SideSell {
		t.Errorf("Expected SELL for stop loss on long, got %s", sig.Side)
	}

	t.Logf("Stop loss signal: %s @ %.2f - %s", sig.Side, sig.Price, sig.Reason)
}

func TestAISignalStrategy_CalculatePositionSize(t *testing.T) {
	s := NewAISignalStrategy()
	ctx := context.Background()
	s.Init(ctx, nil)

	tests := []struct {
		name       string
		strength   float64
		confidence float64
		wantMin    float64
		wantMax    float64
	}{
		{
			name:       "Strong signal",
			strength:   0.8,
			confidence: 0.9,
			wantMin:    500,
			wantMax:    1000,
		},
		{
			name:       "Medium signal",
			strength:   0.5,
			confidence: 0.5,
			wantMin:    100,
			wantMax:    500,
		},
		{
			name:       "Weak signal",
			strength:   0.3,
			confidence: 0.4,
			wantMin:    0,
			wantMax:    200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signal := &entity.MarketSignal{
				Strength:   tt.strength,
				Confidence: tt.confidence,
			}
			size := s.calculatePositionSize(signal)

			if size < tt.wantMin || size > tt.wantMax {
				t.Errorf("Position size %f not in expected range [%f, %f]",
					size, tt.wantMin, tt.wantMax)
			}
			t.Logf("Strength=%.1f, Confidence=%.1f -> Size=$%.0f", tt.strength, tt.confidence, size)
		})
	}
}

func TestAISignalStrategy_BuildEntryReason(t *testing.T) {
	s := NewAISignalStrategy()

	signal := &entity.MarketSignal{
		Symbol:     "BTC",
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
			Sentiment:      0.7,
		},
		FedCutProb:  0.6,
		FedHikeProb: 0.1,
	}

	reason := s.buildEntryReason(signal, "LONG")

	if reason == "" {
		t.Error("Expected non-empty reason")
	}

	// Check that key components are mentioned
	if len(reason) < 50 {
		t.Errorf("Reason seems too short: %s", reason)
	}

	t.Logf("Entry reason:\n%s", reason)
}
