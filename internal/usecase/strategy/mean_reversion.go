package strategy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/zono819/hyperliquid-bot/internal/domain/entity"
	"github.com/zono819/hyperliquid-bot/internal/domain/service"
)

// MeanReversionConfig holds strategy configuration
type MeanReversionConfig struct {
	// RSI settings
	RSIPeriod     int     `json:"rsi_period"`
	RSIOversold   float64 `json:"rsi_oversold"`
	RSIOverbought float64 `json:"rsi_overbought"`

	// Bollinger Bands settings
	BBPeriod    int     `json:"bb_period"`
	BBStdDev    float64 `json:"bb_std_dev"`

	// Position settings
	TakeProfitPct float64 `json:"take_profit_pct"` // e.g., 0.004 = 0.4%
	StopLossPct   float64 `json:"stop_loss_pct"`   // e.g., 0.0025 = 0.25%
	MaxHoldTime   int     `json:"max_hold_time"`   // seconds

	// Risk settings
	PositionSize float64 `json:"position_size"` // quantity per trade
	MaxPositions int     `json:"max_positions"` // max concurrent positions
}

// DefaultMeanReversionConfig returns default configuration
func DefaultMeanReversionConfig() MeanReversionConfig {
	return MeanReversionConfig{
		RSIPeriod:     14,
		RSIOversold:   25.0,
		RSIOverbought: 75.0,
		BBPeriod:      20,
		BBStdDev:      2.5,
		TakeProfitPct: 0.004,  // 0.4%
		StopLossPct:   0.0025, // 0.25%
		MaxHoldTime:   1800,   // 30 minutes
		PositionSize:  0.001,  // default position size
		MaxPositions:  1,
	}
}

// MeanReversionStrategy implements mean reversion trading strategy
type MeanReversionStrategy struct {
	config MeanReversionConfig

	mu           sync.RWMutex
	priceHistory []float64
	maxHistory   int

	// Current position tracking
	entryPrice float64
	entryTime  time.Time
	entrySide  entity.Side
	hasPosition bool

	// Supported symbols
	symbols map[string]bool
}

// NewMeanReversionStrategy creates a new mean reversion strategy
func NewMeanReversionStrategy() *MeanReversionStrategy {
	return &MeanReversionStrategy{
		config:     DefaultMeanReversionConfig(),
		maxHistory: 100,
		symbols: map[string]bool{
			"BTC":  true,
			"ETH":  true,
			"XRP":  true,
		},
	}
}

// Name returns strategy name
func (s *MeanReversionStrategy) Name() string {
	return "mean_reversion"
}

// Init initializes strategy with config
func (s *MeanReversionStrategy) Init(ctx context.Context, config map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Parse config
	if v, ok := config["rsi_period"].(float64); ok {
		s.config.RSIPeriod = int(v)
	}
	if v, ok := config["rsi_oversold"].(float64); ok {
		s.config.RSIOversold = v
	}
	if v, ok := config["rsi_overbought"].(float64); ok {
		s.config.RSIOverbought = v
	}
	if v, ok := config["bb_period"].(float64); ok {
		s.config.BBPeriod = int(v)
	}
	if v, ok := config["bb_std_dev"].(float64); ok {
		s.config.BBStdDev = v
	}
	if v, ok := config["take_profit_pct"].(float64); ok {
		s.config.TakeProfitPct = v
	}
	if v, ok := config["stop_loss_pct"].(float64); ok {
		s.config.StopLossPct = v
	}
	if v, ok := config["max_hold_time"].(float64); ok {
		s.config.MaxHoldTime = int(v)
	}
	if v, ok := config["position_size"].(float64); ok {
		s.config.PositionSize = v
	}
	if v, ok := config["max_positions"].(float64); ok {
		s.config.MaxPositions = int(v)
	}

	s.priceHistory = make([]float64, 0, s.maxHistory)

	return nil
}

// OnTick is called on each market tick
func (s *MeanReversionStrategy) OnTick(ctx context.Context, state *service.MarketState) ([]*service.Signal, error) {
	if state == nil || state.Ticker == nil {
		return nil, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if symbol is supported
	if !s.isSymbolSupported(state.Ticker.Symbol) {
		return nil, nil
	}

	// Update price history
	currentPrice := state.Ticker.LastPrice
	s.updatePriceHistory(currentPrice)

	// Check for timeout exit
	if s.hasPosition {
		signals := s.checkExitConditions(state)
		if len(signals) > 0 {
			return signals, nil
		}
	}

	// Check for entry signals
	if !s.hasPosition {
		return s.checkEntryConditions(state)
	}

	return nil, nil
}

// OnOrderUpdate is called when order status changes
func (s *MeanReversionStrategy) OnOrderUpdate(ctx context.Context, order *entity.Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if order.Status == entity.OrderStatusFilled {
		// If we had a position and order filled, we've exited
		if s.hasPosition {
			// Check if this is an exit order (opposite side)
			if order.Side != s.entrySide {
				s.hasPosition = false
				s.entryPrice = 0
				s.entryTime = time.Time{}
			}
		} else {
			// New entry filled
			s.hasPosition = true
			s.entryPrice = order.Price
			s.entryTime = time.Now()
			s.entrySide = order.Side
		}
	}

	return nil
}

// OnPositionUpdate is called when position changes
func (s *MeanReversionStrategy) OnPositionUpdate(ctx context.Context, position *entity.Position) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if position == nil || position.Size == 0 {
		s.hasPosition = false
		s.entryPrice = 0
		s.entryTime = time.Time{}
	} else {
		s.hasPosition = true
		s.entryPrice = position.EntryPrice
		s.entrySide = position.Side
	}

	return nil
}

// Stop stops the strategy
func (s *MeanReversionStrategy) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.priceHistory = nil
	s.hasPosition = false

	return nil
}

// updatePriceHistory adds price to history and maintains max size
func (s *MeanReversionStrategy) updatePriceHistory(price float64) {
	s.priceHistory = append(s.priceHistory, price)
	if len(s.priceHistory) > s.maxHistory {
		s.priceHistory = s.priceHistory[1:]
	}
}

// isSymbolSupported checks if symbol is in supported list
func (s *MeanReversionStrategy) isSymbolSupported(symbol string) bool {
	// Check various symbol formats (BTC, BTC/USDC, BTC-PERP, etc.)
	for sym := range s.symbols {
		if symbol == sym ||
		   symbol == sym+"/USDC" ||
		   symbol == sym+"-PERP" ||
		   symbol == sym+"USDC" {
			return true
		}
	}
	return false
}

// checkEntryConditions checks for entry signals
func (s *MeanReversionStrategy) checkEntryConditions(state *service.MarketState) ([]*service.Signal, error) {
	if len(s.priceHistory) < s.config.BBPeriod {
		return nil, nil // Not enough data
	}

	currentPrice := state.Ticker.LastPrice
	rsi := RSI(s.priceHistory, s.config.RSIPeriod)
	bb := CalculateBollingerBands(s.priceHistory, s.config.BBPeriod, s.config.BBStdDev)

	var signals []*service.Signal

	// Long entry: RSI oversold + price below lower BB
	if rsi < s.config.RSIOversold && currentPrice < bb.Lower {
		signals = append(signals, &service.Signal{
			Symbol:   state.Ticker.Symbol,
			Side:     entity.SideBuy,
			Price:    state.Ticker.AskPrice, // Use ask for buy
			Quantity: s.config.PositionSize,
			Reason:   fmt.Sprintf("Long entry: RSI=%.1f (<%.1f), Price=%.2f (<BB_Lower=%.2f)", rsi, s.config.RSIOversold, currentPrice, bb.Lower),
		})
	}

	// Short entry: RSI overbought + price above upper BB
	if rsi > s.config.RSIOverbought && currentPrice > bb.Upper {
		signals = append(signals, &service.Signal{
			Symbol:   state.Ticker.Symbol,
			Side:     entity.SideSell,
			Price:    state.Ticker.BidPrice, // Use bid for sell
			Quantity: s.config.PositionSize,
			Reason:   fmt.Sprintf("Short entry: RSI=%.1f (>%.1f), Price=%.2f (>BB_Upper=%.2f)", rsi, s.config.RSIOverbought, currentPrice, bb.Upper),
		})
	}

	return signals, nil
}

// checkExitConditions checks for exit signals
func (s *MeanReversionStrategy) checkExitConditions(state *service.MarketState) []*service.Signal {
	currentPrice := state.Ticker.LastPrice

	var signals []*service.Signal
	var shouldExit bool
	var reason string

	if s.entrySide == entity.SideBuy {
		// Long position exit conditions
		takeProfitPrice := s.entryPrice * (1 + s.config.TakeProfitPct)
		stopLossPrice := s.entryPrice * (1 - s.config.StopLossPct)

		if currentPrice >= takeProfitPrice {
			shouldExit = true
			reason = fmt.Sprintf("Take profit: entry=%.2f, current=%.2f, target=%.2f", s.entryPrice, currentPrice, takeProfitPrice)
		} else if currentPrice <= stopLossPrice {
			shouldExit = true
			reason = fmt.Sprintf("Stop loss: entry=%.2f, current=%.2f, stop=%.2f", s.entryPrice, currentPrice, stopLossPrice)
		}
	} else {
		// Short position exit conditions
		takeProfitPrice := s.entryPrice * (1 - s.config.TakeProfitPct)
		stopLossPrice := s.entryPrice * (1 + s.config.StopLossPct)

		if currentPrice <= takeProfitPrice {
			shouldExit = true
			reason = fmt.Sprintf("Take profit: entry=%.2f, current=%.2f, target=%.2f", s.entryPrice, currentPrice, takeProfitPrice)
		} else if currentPrice >= stopLossPrice {
			shouldExit = true
			reason = fmt.Sprintf("Stop loss: entry=%.2f, current=%.2f, stop=%.2f", s.entryPrice, currentPrice, stopLossPrice)
		}
	}

	// Timeout exit
	if !shouldExit && time.Since(s.entryTime) > time.Duration(s.config.MaxHoldTime)*time.Second {
		shouldExit = true
		reason = fmt.Sprintf("Timeout exit: held for %v, max=%ds", time.Since(s.entryTime), s.config.MaxHoldTime)
	}

	if shouldExit {
		exitSide := entity.SideSell
		exitPrice := state.Ticker.BidPrice
		if s.entrySide == entity.SideSell {
			exitSide = entity.SideBuy
			exitPrice = state.Ticker.AskPrice
		}

		signals = append(signals, &service.Signal{
			Symbol:   state.Ticker.Symbol,
			Side:     exitSide,
			Price:    exitPrice,
			Quantity: s.config.PositionSize,
			Reason:   reason,
		})
	}

	return signals
}

// GetConfig returns current configuration
func (s *MeanReversionStrategy) GetConfig() MeanReversionConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// GetState returns current strategy state (for monitoring)
func (s *MeanReversionStrategy) GetState() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state := map[string]interface{}{
		"has_position":  s.hasPosition,
		"price_history": len(s.priceHistory),
	}

	if s.hasPosition {
		state["entry_price"] = s.entryPrice
		state["entry_side"] = s.entrySide
		state["entry_time"] = s.entryTime
		state["hold_duration"] = time.Since(s.entryTime).String()
	}

	if len(s.priceHistory) >= s.config.RSIPeriod {
		state["current_rsi"] = RSI(s.priceHistory, s.config.RSIPeriod)
	}
	if len(s.priceHistory) >= s.config.BBPeriod {
		bb := CalculateBollingerBands(s.priceHistory, s.config.BBPeriod, s.config.BBStdDev)
		state["bb_upper"] = bb.Upper
		state["bb_middle"] = bb.Middle
		state["bb_lower"] = bb.Lower
	}

	return state
}
