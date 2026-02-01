package service

import (
	"context"

	"github.com/zono819/hyperliquid-bot/internal/domain/entity"
)

// Signal represents a trading signal from strategy
type Signal struct {
	Symbol   string
	Side     entity.Side
	Price    float64
	Quantity float64
	Reason   string
}

// MarketState represents current market state for strategy
type MarketState struct {
	Ticker    *entity.Ticker
	OrderBook *entity.OrderBook
	Position  *entity.Position
	Orders    []*entity.Order
}

// Strategy defines trading strategy interface
type Strategy interface {
	// Name returns strategy name
	Name() string

	// Init initializes strategy with config
	Init(ctx context.Context, config map[string]interface{}) error

	// OnTick is called on each market tick
	OnTick(ctx context.Context, state *MarketState) ([]*Signal, error)

	// OnOrderUpdate is called when order status changes
	OnOrderUpdate(ctx context.Context, order *entity.Order) error

	// OnPositionUpdate is called when position changes
	OnPositionUpdate(ctx context.Context, position *entity.Position) error

	// Stop stops the strategy
	Stop(ctx context.Context) error
}

// StrategyFactory creates strategy instances
type StrategyFactory interface {
	// Create creates a new strategy instance by name
	Create(name string) (Strategy, error)

	// List returns available strategy names
	List() []string
}
