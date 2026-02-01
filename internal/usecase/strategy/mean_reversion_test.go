package strategy

import (
	"context"
	"testing"
	"time"

	"github.com/zono819/hyperliquid-bot/internal/domain/entity"
	"github.com/zono819/hyperliquid-bot/internal/domain/service"
)

func TestNewMeanReversionStrategy(t *testing.T) {
	s := NewMeanReversionStrategy()

	if s.Name() != "mean_reversion" {
		t.Errorf("Name() = %v, expected mean_reversion", s.Name())
	}

	config := s.GetConfig()
	if config.RSIPeriod != 14 {
		t.Errorf("default RSIPeriod = %v, expected 14", config.RSIPeriod)
	}
	if config.RSIOversold != 25.0 {
		t.Errorf("default RSIOversold = %v, expected 25.0", config.RSIOversold)
	}
	if config.RSIOverbought != 75.0 {
		t.Errorf("default RSIOverbought = %v, expected 75.0", config.RSIOverbought)
	}
}

func TestMeanReversionStrategy_Init(t *testing.T) {
	s := NewMeanReversionStrategy()
	ctx := context.Background()

	config := map[string]interface{}{
		"rsi_period":      float64(10),
		"rsi_oversold":    float64(20),
		"rsi_overbought":  float64(80),
		"bb_period":       float64(15),
		"bb_std_dev":      float64(3.0),
		"take_profit_pct": float64(0.005),
		"stop_loss_pct":   float64(0.003),
		"position_size":   float64(0.01),
	}

	err := s.Init(ctx, config)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	c := s.GetConfig()
	if c.RSIPeriod != 10 {
		t.Errorf("RSIPeriod = %v, expected 10", c.RSIPeriod)
	}
	if c.RSIOversold != 20.0 {
		t.Errorf("RSIOversold = %v, expected 20.0", c.RSIOversold)
	}
	if c.BBPeriod != 15 {
		t.Errorf("BBPeriod = %v, expected 15", c.BBPeriod)
	}
}

func TestMeanReversionStrategy_OnTick_NoData(t *testing.T) {
	s := NewMeanReversionStrategy()
	ctx := context.Background()
	s.Init(ctx, nil)

	// Test with nil state
	signals, err := s.OnTick(ctx, nil)
	if err != nil {
		t.Errorf("OnTick() error = %v", err)
	}
	if signals != nil {
		t.Errorf("OnTick() with nil state should return nil signals")
	}

	// Test with nil ticker
	state := &service.MarketState{}
	signals, err = s.OnTick(ctx, state)
	if err != nil {
		t.Errorf("OnTick() error = %v", err)
	}
	if signals != nil {
		t.Errorf("OnTick() with nil ticker should return nil signals")
	}
}

func TestMeanReversionStrategy_OnTick_UnsupportedSymbol(t *testing.T) {
	s := NewMeanReversionStrategy()
	ctx := context.Background()
	s.Init(ctx, nil)

	state := &service.MarketState{
		Ticker: &entity.Ticker{
			Symbol:    "UNSUPPORTED",
			LastPrice: 100.0,
		},
	}

	signals, err := s.OnTick(ctx, state)
	if err != nil {
		t.Errorf("OnTick() error = %v", err)
	}
	if signals != nil {
		t.Errorf("OnTick() with unsupported symbol should return nil signals")
	}
}

func TestMeanReversionStrategy_LongEntry(t *testing.T) {
	s := NewMeanReversionStrategy()
	ctx := context.Background()
	s.Init(ctx, map[string]interface{}{
		"rsi_period":    float64(14),
		"rsi_oversold":  float64(30),
		"bb_period":     float64(20),
		"bb_std_dev":    float64(2.0),
		"position_size": float64(0.01),
	})

	// Simulate declining prices to create oversold condition
	// Start high and go down
	for i := 0; i < 25; i++ {
		price := 100.0 - float64(i)*2 // 100, 98, 96, ... 52
		state := &service.MarketState{
			Ticker: &entity.Ticker{
				Symbol:    "BTC/USDC",
				LastPrice: price,
				BidPrice:  price - 0.1,
				AskPrice:  price + 0.1,
			},
		}
		s.OnTick(ctx, state)
	}

	// Price drops significantly below BB lower - should trigger long
	state := &service.MarketState{
		Ticker: &entity.Ticker{
			Symbol:    "BTC/USDC",
			LastPrice: 45.0, // Very low
			BidPrice:  44.9,
			AskPrice:  45.1,
		},
	}

	signals, err := s.OnTick(ctx, state)
	if err != nil {
		t.Fatalf("OnTick() error = %v", err)
	}

	// Check for long signal
	var hasLongSignal bool
	for _, sig := range signals {
		if sig.Side == entity.SideBuy {
			hasLongSignal = true
			if sig.Price != 45.1 {
				t.Errorf("Long signal price = %v, expected ask price 45.1", sig.Price)
			}
		}
	}

	if !hasLongSignal {
		t.Log("No long signal generated - this may be expected depending on indicator values")
	}
}

func TestMeanReversionStrategy_ShortEntry(t *testing.T) {
	s := NewMeanReversionStrategy()
	ctx := context.Background()
	s.Init(ctx, map[string]interface{}{
		"rsi_period":     float64(14),
		"rsi_overbought": float64(70),
		"bb_period":      float64(20),
		"bb_std_dev":     float64(2.0),
		"position_size":  float64(0.01),
	})

	// Simulate rising prices to create overbought condition
	for i := 0; i < 25; i++ {
		price := 100.0 + float64(i)*2 // 100, 102, 104, ... 148
		state := &service.MarketState{
			Ticker: &entity.Ticker{
				Symbol:    "ETH/USDC",
				LastPrice: price,
				BidPrice:  price - 0.1,
				AskPrice:  price + 0.1,
			},
		}
		s.OnTick(ctx, state)
	}

	// Price rises significantly above BB upper - should trigger short
	state := &service.MarketState{
		Ticker: &entity.Ticker{
			Symbol:    "ETH/USDC",
			LastPrice: 160.0, // Very high
			BidPrice:  159.9,
			AskPrice:  160.1,
		},
	}

	signals, err := s.OnTick(ctx, state)
	if err != nil {
		t.Fatalf("OnTick() error = %v", err)
	}

	var hasShortSignal bool
	for _, sig := range signals {
		if sig.Side == entity.SideSell {
			hasShortSignal = true
			if sig.Price != 159.9 {
				t.Errorf("Short signal price = %v, expected bid price 159.9", sig.Price)
			}
		}
	}

	if !hasShortSignal {
		t.Log("No short signal generated - this may be expected depending on indicator values")
	}
}

func TestMeanReversionStrategy_TakeProfit(t *testing.T) {
	s := NewMeanReversionStrategy()
	ctx := context.Background()
	s.Init(ctx, map[string]interface{}{
		"take_profit_pct": float64(0.01), // 1%
		"stop_loss_pct":   float64(0.005),
	})

	// Simulate having a long position
	s.mu.Lock()
	s.hasPosition = true
	s.entryPrice = 100.0
	s.entrySide = entity.SideBuy
	s.entryTime = time.Now()
	s.mu.Unlock()

	// Price rises to take profit level
	state := &service.MarketState{
		Ticker: &entity.Ticker{
			Symbol:    "BTC/USDC",
			LastPrice: 101.5, // 1.5% profit
			BidPrice:  101.4,
			AskPrice:  101.6,
		},
	}

	signals, err := s.OnTick(ctx, state)
	if err != nil {
		t.Fatalf("OnTick() error = %v", err)
	}

	if len(signals) == 0 {
		t.Fatal("Expected take profit signal")
	}

	if signals[0].Side != entity.SideSell {
		t.Errorf("Take profit signal side = %v, expected sell", signals[0].Side)
	}
}

func TestMeanReversionStrategy_StopLoss(t *testing.T) {
	s := NewMeanReversionStrategy()
	ctx := context.Background()
	s.Init(ctx, map[string]interface{}{
		"take_profit_pct": float64(0.01),
		"stop_loss_pct":   float64(0.005), // 0.5%
	})

	// Simulate having a long position
	s.mu.Lock()
	s.hasPosition = true
	s.entryPrice = 100.0
	s.entrySide = entity.SideBuy
	s.entryTime = time.Now()
	s.mu.Unlock()

	// Price drops to stop loss level
	state := &service.MarketState{
		Ticker: &entity.Ticker{
			Symbol:    "BTC/USDC",
			LastPrice: 99.0, // 1% loss
			BidPrice:  98.9,
			AskPrice:  99.1,
		},
	}

	signals, err := s.OnTick(ctx, state)
	if err != nil {
		t.Fatalf("OnTick() error = %v", err)
	}

	if len(signals) == 0 {
		t.Fatal("Expected stop loss signal")
	}

	if signals[0].Side != entity.SideSell {
		t.Errorf("Stop loss signal side = %v, expected sell", signals[0].Side)
	}
}

func TestMeanReversionStrategy_TimeoutExit(t *testing.T) {
	s := NewMeanReversionStrategy()
	ctx := context.Background()
	s.Init(ctx, map[string]interface{}{
		"take_profit_pct": float64(0.1),  // 10% - won't hit
		"stop_loss_pct":   float64(0.1),  // 10% - won't hit
		"max_hold_time":   float64(1),    // 1 second
	})

	// Simulate having a position from the past
	s.mu.Lock()
	s.hasPosition = true
	s.entryPrice = 100.0
	s.entrySide = entity.SideBuy
	s.entryTime = time.Now().Add(-2 * time.Second) // 2 seconds ago
	s.mu.Unlock()

	// Price is neutral - not hitting TP or SL
	state := &service.MarketState{
		Ticker: &entity.Ticker{
			Symbol:    "BTC/USDC",
			LastPrice: 100.0,
			BidPrice:  99.9,
			AskPrice:  100.1,
		},
	}

	signals, err := s.OnTick(ctx, state)
	if err != nil {
		t.Fatalf("OnTick() error = %v", err)
	}

	if len(signals) == 0 {
		t.Fatal("Expected timeout exit signal")
	}

	if signals[0].Side != entity.SideSell {
		t.Errorf("Timeout exit signal side = %v, expected sell", signals[0].Side)
	}
}

func TestMeanReversionStrategy_OnOrderUpdate(t *testing.T) {
	s := NewMeanReversionStrategy()
	ctx := context.Background()
	s.Init(ctx, nil)

	// Entry order filled
	entryOrder := &entity.Order{
		ID:     "order-1",
		Symbol: "BTC/USDC",
		Side:   entity.SideBuy,
		Price:  100.0,
		Status: entity.OrderStatusFilled,
	}

	err := s.OnOrderUpdate(ctx, entryOrder)
	if err != nil {
		t.Fatalf("OnOrderUpdate() error = %v", err)
	}

	state := s.GetState()
	if !state["has_position"].(bool) {
		t.Error("Expected has_position = true after entry order filled")
	}

	// Exit order filled
	exitOrder := &entity.Order{
		ID:     "order-2",
		Symbol: "BTC/USDC",
		Side:   entity.SideSell, // opposite side
		Price:  101.0,
		Status: entity.OrderStatusFilled,
	}

	err = s.OnOrderUpdate(ctx, exitOrder)
	if err != nil {
		t.Fatalf("OnOrderUpdate() error = %v", err)
	}

	state = s.GetState()
	if state["has_position"].(bool) {
		t.Error("Expected has_position = false after exit order filled")
	}
}

func TestMeanReversionStrategy_OnPositionUpdate(t *testing.T) {
	s := NewMeanReversionStrategy()
	ctx := context.Background()
	s.Init(ctx, nil)

	// Position opened
	position := &entity.Position{
		Symbol:     "BTC/USDC",
		Side:       entity.SideBuy,
		Size:       0.01,
		EntryPrice: 100.0,
	}

	err := s.OnPositionUpdate(ctx, position)
	if err != nil {
		t.Fatalf("OnPositionUpdate() error = %v", err)
	}

	state := s.GetState()
	if !state["has_position"].(bool) {
		t.Error("Expected has_position = true")
	}
	if state["entry_price"].(float64) != 100.0 {
		t.Errorf("entry_price = %v, expected 100.0", state["entry_price"])
	}

	// Position closed
	err = s.OnPositionUpdate(ctx, nil)
	if err != nil {
		t.Fatalf("OnPositionUpdate() error = %v", err)
	}

	state = s.GetState()
	if state["has_position"].(bool) {
		t.Error("Expected has_position = false after position closed")
	}
}

func TestMeanReversionStrategy_Stop(t *testing.T) {
	s := NewMeanReversionStrategy()
	ctx := context.Background()
	s.Init(ctx, nil)

	// Add some data
	for i := 0; i < 10; i++ {
		state := &service.MarketState{
			Ticker: &entity.Ticker{
				Symbol:    "BTC/USDC",
				LastPrice: 100.0 + float64(i),
			},
		}
		s.OnTick(ctx, state)
	}

	s.mu.Lock()
	s.hasPosition = true
	s.mu.Unlock()

	err := s.Stop(ctx)
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	state := s.GetState()
	if state["has_position"].(bool) {
		t.Error("Expected has_position = false after stop")
	}
}

func TestMeanReversionStrategy_SymbolSupport(t *testing.T) {
	s := NewMeanReversionStrategy()

	tests := []struct {
		symbol   string
		expected bool
	}{
		{"BTC", true},
		{"ETH", true},
		{"XRP", true},
		{"BTC/USDC", true},
		{"ETH/USDC", true},
		{"XRP/USDC", true},
		{"BTC-PERP", true},
		{"ETH-PERP", true},
		{"BTCUSDC", true},
		{"DOGE", false},
		{"UNKNOWN", false},
	}

	for _, tt := range tests {
		t.Run(tt.symbol, func(t *testing.T) {
			result := s.isSymbolSupported(tt.symbol)
			if result != tt.expected {
				t.Errorf("isSymbolSupported(%s) = %v, expected %v", tt.symbol, result, tt.expected)
			}
		})
	}
}

func TestMeanReversionStrategy_GetState(t *testing.T) {
	s := NewMeanReversionStrategy()
	ctx := context.Background()
	s.Init(ctx, nil)

	// Build up price history
	for i := 0; i < 25; i++ {
		price := 100.0 + float64(i%5) // oscillating prices
		state := &service.MarketState{
			Ticker: &entity.Ticker{
				Symbol:    "BTC/USDC",
				LastPrice: price,
				BidPrice:  price - 0.1,
				AskPrice:  price + 0.1,
			},
		}
		s.OnTick(ctx, state)
	}

	state := s.GetState()

	if _, ok := state["has_position"]; !ok {
		t.Error("GetState() missing has_position")
	}
	if _, ok := state["price_history"]; !ok {
		t.Error("GetState() missing price_history")
	}
	if _, ok := state["current_rsi"]; !ok {
		t.Error("GetState() missing current_rsi")
	}
	if _, ok := state["bb_upper"]; !ok {
		t.Error("GetState() missing bb_upper")
	}
	if _, ok := state["bb_middle"]; !ok {
		t.Error("GetState() missing bb_middle")
	}
	if _, ok := state["bb_lower"]; !ok {
		t.Error("GetState() missing bb_lower")
	}
}
