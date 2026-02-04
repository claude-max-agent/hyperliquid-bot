package strategy

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/zono819/hyperliquid-bot/internal/domain/entity"
	"github.com/zono819/hyperliquid-bot/internal/domain/service"
)

// AISignalConfig holds AI signal strategy configuration
type AISignalConfig struct {
	// Position sizing
	MaxPositionSize  float64 `yaml:"max_position_size"`   // Max position size in USD
	PositionSizeStep float64 `yaml:"position_size_step"`  // Position adjustment step

	// Entry thresholds
	MinSignalStrength  float64 `yaml:"min_signal_strength"`  // Minimum signal strength to enter (0-1)
	MinConfidence      float64 `yaml:"min_confidence"`       // Minimum confidence level (0-1)

	// Exit thresholds
	TakeProfitPercent float64 `yaml:"take_profit_percent"`   // Take profit %
	StopLossPercent   float64 `yaml:"stop_loss_percent"`     // Stop loss %
	TrailingStop      bool    `yaml:"trailing_stop"`         // Enable trailing stop
	TrailingPercent   float64 `yaml:"trailing_percent"`      // Trailing stop %

	// Risk management
	MaxDrawdown       float64 `yaml:"max_drawdown"`          // Max drawdown before stopping
	CooldownPeriod    time.Duration `yaml:"cooldown_period"` // Cooldown after loss

	// Signal weights (should sum to 1.0)
	WeightDerivatives float64 `yaml:"weight_derivatives"`    // CoinGlass weight
	WeightWhale       float64 `yaml:"weight_whale"`          // Whale Alert weight
	WeightSentiment   float64 `yaml:"weight_sentiment"`      // LunarCrush weight
	WeightMacro       float64 `yaml:"weight_macro"`          // FedWatch/TE weight
}

// DefaultAISignalConfig returns default configuration
func DefaultAISignalConfig() AISignalConfig {
	return AISignalConfig{
		MaxPositionSize:    1000,    // $1000 max
		PositionSizeStep:   100,     // $100 steps
		MinSignalStrength:  0.3,     // 30% minimum strength
		MinConfidence:      0.4,     // 40% minimum confidence
		TakeProfitPercent:  0.02,    // 2% take profit
		StopLossPercent:    0.01,    // 1% stop loss
		TrailingStop:       true,
		TrailingPercent:    0.005,   // 0.5% trailing
		MaxDrawdown:        0.05,    // 5% max drawdown
		CooldownPeriod:     30 * time.Minute,
		WeightDerivatives:  0.30,
		WeightWhale:        0.20,
		WeightSentiment:    0.25,
		WeightMacro:        0.25,
	}
}

// AISignalStrategy implements AI-driven trading strategy
type AISignalStrategy struct {
	config AISignalConfig

	mu            sync.RWMutex
	running       bool
	entryPrice    float64
	highestPrice  float64   // For trailing stop
	lastSignal    *entity.MarketSignal
	lastTradeTime time.Time
	totalPnL      float64
	peakEquity    float64
}

// NewAISignalStrategy creates a new AI signal strategy
func NewAISignalStrategy() *AISignalStrategy {
	return &AISignalStrategy{
		config: DefaultAISignalConfig(),
	}
}

// Name returns strategy name
func (s *AISignalStrategy) Name() string {
	return "ai_signal"
}

// Init initializes strategy with config
func (s *AISignalStrategy) Init(ctx context.Context, config map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Parse config
	if v, ok := config["max_position_size"].(float64); ok {
		s.config.MaxPositionSize = v
	}
	if v, ok := config["min_signal_strength"].(float64); ok {
		s.config.MinSignalStrength = v
	}
	if v, ok := config["min_confidence"].(float64); ok {
		s.config.MinConfidence = v
	}
	if v, ok := config["take_profit_percent"].(float64); ok {
		s.config.TakeProfitPercent = v
	}
	if v, ok := config["stop_loss_percent"].(float64); ok {
		s.config.StopLossPercent = v
	}

	s.running = true
	return nil
}

// OnTick is called on each market tick
func (s *AISignalStrategy) OnTick(ctx context.Context, state *service.MarketState) ([]*service.Signal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running || state.Ticker == nil {
		return nil, nil
	}

	signals := make([]*service.Signal, 0)

	// Update market signal
	if state.MarketSignal != nil {
		s.lastSignal = state.MarketSignal
	}

	// Check cooldown
	if time.Since(s.lastTradeTime) < s.config.CooldownPeriod && s.totalPnL < 0 {
		return nil, nil
	}

	currentPrice := state.Ticker.LastPrice
	hasPosition := state.Position != nil && state.Position.Size != 0

	if hasPosition {
		// Manage existing position
		exitSignals := s.managePosition(state, currentPrice)
		signals = append(signals, exitSignals...)
	} else {
		// Look for entry opportunities
		entrySignal := s.evaluateEntry(state, currentPrice)
		if entrySignal != nil {
			signals = append(signals, entrySignal)
		}
	}

	return signals, nil
}

// evaluateEntry evaluates entry opportunity based on aggregated signals
func (s *AISignalStrategy) evaluateEntry(state *service.MarketState, currentPrice float64) *service.Signal {
	if s.lastSignal == nil {
		return nil
	}

	signal := s.lastSignal

	// Check minimum thresholds
	if signal.Strength < s.config.MinSignalStrength {
		return nil
	}
	if signal.Confidence < s.config.MinConfidence {
		return nil
	}

	// Determine position size based on signal strength and confidence
	positionSize := s.calculatePositionSize(signal)
	if positionSize <= 0 {
		return nil
	}

	// Generate trading signal based on bias
	var side entity.Side
	var reason string

	switch signal.Bias {
	case entity.SignalBiasBullish:
		side = entity.SideBuy
		reason = s.buildEntryReason(signal, "LONG")
	case entity.SignalBiasBearish:
		side = entity.SideSell
		reason = s.buildEntryReason(signal, "SHORT")
	default:
		return nil
	}

	quantity := positionSize / currentPrice

	return &service.Signal{
		Symbol:   state.Ticker.Symbol,
		Side:     side,
		Price:    currentPrice,
		Quantity: quantity,
		Reason:   reason,
	}
}

// calculatePositionSize calculates position size based on signal
func (s *AISignalStrategy) calculatePositionSize(signal *entity.MarketSignal) float64 {
	// Base size scaled by strength and confidence
	baseSize := s.config.MaxPositionSize * signal.Strength * signal.Confidence

	// Round to step size
	steps := math.Floor(baseSize / s.config.PositionSizeStep)
	size := steps * s.config.PositionSizeStep

	// Apply max limit
	if size > s.config.MaxPositionSize {
		size = s.config.MaxPositionSize
	}

	return size
}

// buildEntryReason builds human-readable entry reason
func (s *AISignalStrategy) buildEntryReason(signal *entity.MarketSignal, direction string) string {
	reason := fmt.Sprintf("%s Entry | Strength: %.0f%% | Confidence: %.0f%%\n",
		direction, signal.Strength*100, signal.Confidence*100)

	// Add data source contributions
	reasons := []string{}

	if signal.FundingRate != nil {
		if signal.FundingRate.Rate > 0 {
			reasons = append(reasons, fmt.Sprintf("FR: +%.4f%% (bearish pressure)", signal.FundingRate.Rate*100))
		} else {
			reasons = append(reasons, fmt.Sprintf("FR: %.4f%% (bullish pressure)", signal.FundingRate.Rate*100))
		}
	}

	if signal.LongShortRatio != nil {
		reasons = append(reasons, fmt.Sprintf("L/S Ratio: %.2f", signal.LongShortRatio.LongShortRatio))
	}

	if len(signal.RecentWhaleAlerts) > 0 {
		var inflow, outflow float64
		for _, a := range signal.RecentWhaleAlerts {
			switch a.GetAlertType() {
			case entity.WhaleAlertExchangeInflow:
				inflow += a.AmountUSD
			case entity.WhaleAlertExchangeOutflow:
				outflow += a.AmountUSD
			}
		}
		reasons = append(reasons, fmt.Sprintf("Whale: $%.0fM in / $%.0fM out", inflow/1e6, outflow/1e6))
	}

	if signal.SocialSentiment != nil {
		sentimentStr := "neutral"
		if signal.SocialSentiment.SentimentScore > 0.2 {
			sentimentStr = "bullish"
		} else if signal.SocialSentiment.SentimentScore < -0.2 {
			sentimentStr = "bearish"
		}
		reasons = append(reasons, fmt.Sprintf("Sentiment: %s (%.0f%%)", sentimentStr, signal.SocialSentiment.Sentiment*100))
	}

	if signal.FedCutProb > 0 || signal.FedHikeProb > 0 {
		reasons = append(reasons, fmt.Sprintf("Fed: Cut %.0f%% / Hike %.0f%%", signal.FedCutProb*100, signal.FedHikeProb*100))
	}

	for _, r := range reasons {
		reason += "  • " + r + "\n"
	}

	return reason
}

// managePosition manages existing position (take profit, stop loss, trailing)
func (s *AISignalStrategy) managePosition(state *service.MarketState, currentPrice float64) []*service.Signal {
	signals := make([]*service.Signal, 0)
	position := state.Position

	if position == nil || position.Size == 0 {
		return signals
	}

	isLong := position.Size > 0
	entryPrice := position.EntryPrice

	// Update highest price for trailing stop
	if isLong && currentPrice > s.highestPrice {
		s.highestPrice = currentPrice
	} else if !isLong && (s.highestPrice == 0 || currentPrice < s.highestPrice) {
		s.highestPrice = currentPrice
	}

	// Calculate PnL percentage
	var pnlPercent float64
	if isLong {
		pnlPercent = (currentPrice - entryPrice) / entryPrice
	} else {
		pnlPercent = (entryPrice - currentPrice) / entryPrice
	}

	// Check take profit
	if pnlPercent >= s.config.TakeProfitPercent {
		signals = append(signals, s.createExitSignal(state, position, currentPrice,
			fmt.Sprintf("Take Profit: %.2f%% gain", pnlPercent*100)))
		return signals
	}

	// Check stop loss
	if pnlPercent <= -s.config.StopLossPercent {
		signals = append(signals, s.createExitSignal(state, position, currentPrice,
			fmt.Sprintf("Stop Loss: %.2f%% loss", pnlPercent*100)))
		return signals
	}

	// Check trailing stop
	if s.config.TrailingStop && s.highestPrice > 0 {
		var trailingPnL float64
		if isLong {
			trailingPnL = (currentPrice - s.highestPrice) / s.highestPrice
		} else {
			trailingPnL = (s.highestPrice - currentPrice) / s.highestPrice
		}

		if trailingPnL <= -s.config.TrailingPercent {
			signals = append(signals, s.createExitSignal(state, position, currentPrice,
				fmt.Sprintf("Trailing Stop: %.2f%% from high", trailingPnL*100)))
			return signals
		}
	}

	// Check signal reversal
	if s.lastSignal != nil {
		if isLong && s.lastSignal.Bias == entity.SignalBiasBearish && s.lastSignal.Strength > 0.5 {
			signals = append(signals, s.createExitSignal(state, position, currentPrice,
				"Signal Reversal: Strong bearish signal detected"))
			return signals
		}
		if !isLong && s.lastSignal.Bias == entity.SignalBiasBullish && s.lastSignal.Strength > 0.5 {
			signals = append(signals, s.createExitSignal(state, position, currentPrice,
				"Signal Reversal: Strong bullish signal detected"))
			return signals
		}
	}

	return signals
}

// createExitSignal creates an exit signal
func (s *AISignalStrategy) createExitSignal(state *service.MarketState, position *entity.Position, price float64, reason string) *service.Signal {
	var side entity.Side
	if position.Size > 0 {
		side = entity.SideSell // Close long
	} else {
		side = entity.SideBuy // Close short
	}

	return &service.Signal{
		Symbol:   state.Ticker.Symbol,
		Side:     side,
		Price:    price,
		Quantity: math.Abs(position.Size),
		Reason:   "EXIT: " + reason,
	}
}

// OnOrderUpdate is called when order status changes
func (s *AISignalStrategy) OnOrderUpdate(ctx context.Context, order *entity.Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if order.Status == entity.OrderStatusFilled {
		s.lastTradeTime = time.Now()

		// Track PnL for drawdown calculation
		if order.Side == entity.SideSell && s.entryPrice > 0 {
			pnl := (order.Price - s.entryPrice) * order.Quantity
			s.totalPnL += pnl
			if s.totalPnL > s.peakEquity {
				s.peakEquity = s.totalPnL
			}
		}
	}

	return nil
}

// OnPositionUpdate is called when position changes
func (s *AISignalStrategy) OnPositionUpdate(ctx context.Context, position *entity.Position) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if position.Size != 0 {
		s.entryPrice = position.EntryPrice
		s.highestPrice = position.EntryPrice
	} else {
		// Position closed
		s.entryPrice = 0
		s.highestPrice = 0
	}

	return nil
}

// Stop stops the strategy
func (s *AISignalStrategy) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running = false
	return nil
}

// GetStats returns strategy statistics
func (s *AISignalStrategy) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	drawdown := 0.0
	if s.peakEquity > 0 {
		drawdown = (s.peakEquity - s.totalPnL) / s.peakEquity
	}

	return map[string]interface{}{
		"total_pnl":      s.totalPnL,
		"peak_equity":    s.peakEquity,
		"current_drawdown": drawdown,
		"running":        s.running,
	}
}
