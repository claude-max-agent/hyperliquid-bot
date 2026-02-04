package strategy

import (
	"context"
	"math"
	"sync"

	"github.com/zono819/hyperliquid-bot/internal/domain/entity"
	"github.com/zono819/hyperliquid-bot/internal/domain/service"
)

// MeanReversionStrategy implements a simple mean reversion trading strategy
type MeanReversionStrategy struct {
	mu       sync.RWMutex
	running  bool
	config   MeanReversionConfig
	prices   []float64
	position *entity.Position
}

// MeanReversionConfig holds strategy configuration
type MeanReversionConfig struct {
	WindowSize      int     // Number of periods for MA calculation
	EntryDeviation  float64 // Entry threshold (standard deviations)
	ExitDeviation   float64 // Exit threshold (standard deviations)
	PositionSize    float64 // Position size in base currency
	MaxPositionSize float64 // Maximum position size
}

// DefaultMeanReversionConfig returns default configuration
func DefaultMeanReversionConfig() MeanReversionConfig {
	return MeanReversionConfig{
		WindowSize:      20,
		EntryDeviation:  2.0,
		ExitDeviation:   0.5,
		PositionSize:    0.01,
		MaxPositionSize: 0.1,
	}
}

// NewMeanReversionStrategy creates a new mean reversion strategy
func NewMeanReversionStrategy() *MeanReversionStrategy {
	return &MeanReversionStrategy{
		config: DefaultMeanReversionConfig(),
		prices: make([]float64, 0),
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

	if v, ok := config["window_size"].(int); ok {
		s.config.WindowSize = v
	}
	if v, ok := config["entry_deviation"].(float64); ok {
		s.config.EntryDeviation = v
	}
	if v, ok := config["exit_deviation"].(float64); ok {
		s.config.ExitDeviation = v
	}
	if v, ok := config["position_size"].(float64); ok {
		s.config.PositionSize = v
	}
	if v, ok := config["max_position_size"].(float64); ok {
		s.config.MaxPositionSize = v
	}

	s.running = true
	return nil
}

// OnTick is called on each market tick
func (s *MeanReversionStrategy) OnTick(ctx context.Context, state *service.MarketState) ([]*service.Signal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running || state.Ticker == nil {
		return nil, nil
	}

	signals := make([]*service.Signal, 0)
	currentPrice := state.Ticker.LastPrice

	// Add price to history
	s.prices = append(s.prices, currentPrice)
	if len(s.prices) > s.config.WindowSize {
		s.prices = s.prices[1:]
	}

	// Need enough data for calculation
	if len(s.prices) < s.config.WindowSize {
		return nil, nil
	}

	// Calculate mean and standard deviation
	mean := s.calculateMean()
	stdDev := s.calculateStdDev(mean)

	if stdDev == 0 {
		return nil, nil
	}

	// Calculate z-score
	zScore := (currentPrice - mean) / stdDev

	hasPosition := state.Position != nil && state.Position.Size != 0
	s.position = state.Position

	if hasPosition {
		// Check exit conditions
		if s.position.Size > 0 && zScore >= -s.config.ExitDeviation {
			// Close long position (price returned to mean)
			signals = append(signals, &service.Signal{
				Symbol:   state.Ticker.Symbol,
				Side:     entity.SideSell,
				Price:    currentPrice,
				Quantity: math.Abs(s.position.Size),
				Reason:   "Mean reversion: price returned to mean (close long)",
			})
		} else if s.position.Size < 0 && zScore <= s.config.ExitDeviation {
			// Close short position
			signals = append(signals, &service.Signal{
				Symbol:   state.Ticker.Symbol,
				Side:     entity.SideBuy,
				Price:    currentPrice,
				Quantity: math.Abs(s.position.Size),
				Reason:   "Mean reversion: price returned to mean (close short)",
			})
		}
	} else {
		// Check entry conditions
		if zScore <= -s.config.EntryDeviation {
			// Price below mean - buy expecting reversion up
			signals = append(signals, &service.Signal{
				Symbol:   state.Ticker.Symbol,
				Side:     entity.SideBuy,
				Price:    currentPrice,
				Quantity: s.config.PositionSize,
				Reason:   "Mean reversion: price below lower band (enter long)",
			})
		} else if zScore >= s.config.EntryDeviation {
			// Price above mean - sell expecting reversion down
			signals = append(signals, &service.Signal{
				Symbol:   state.Ticker.Symbol,
				Side:     entity.SideSell,
				Price:    currentPrice,
				Quantity: s.config.PositionSize,
				Reason:   "Mean reversion: price above upper band (enter short)",
			})
		}
	}

	return signals, nil
}

// calculateMean calculates the simple moving average
func (s *MeanReversionStrategy) calculateMean() float64 {
	if len(s.prices) == 0 {
		return 0
	}

	sum := 0.0
	for _, p := range s.prices {
		sum += p
	}
	return sum / float64(len(s.prices))
}

// calculateStdDev calculates standard deviation
func (s *MeanReversionStrategy) calculateStdDev(mean float64) float64 {
	if len(s.prices) == 0 {
		return 0
	}

	variance := 0.0
	for _, p := range s.prices {
		diff := p - mean
		variance += diff * diff
	}
	variance /= float64(len(s.prices))

	return math.Sqrt(variance)
}

// OnOrderUpdate is called when order status changes
func (s *MeanReversionStrategy) OnOrderUpdate(ctx context.Context, order *entity.Order) error {
	return nil
}

// OnPositionUpdate is called when position changes
func (s *MeanReversionStrategy) OnPositionUpdate(ctx context.Context, position *entity.Position) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.position = position
	return nil
}

// Stop stops the strategy
func (s *MeanReversionStrategy) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running = false
	return nil
}
