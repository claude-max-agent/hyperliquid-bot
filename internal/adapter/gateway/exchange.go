package gateway

import (
	"context"

	"github.com/zono819/hyperliquid-bot/internal/domain/entity"
)

// ExchangeGateway defines exchange interaction interface
type ExchangeGateway interface {
	// Connect establishes connection to exchange
	Connect(ctx context.Context) error

	// Disconnect closes connection
	Disconnect(ctx context.Context) error

	// PlaceOrder places a new order
	PlaceOrder(ctx context.Context, order *entity.Order) (*entity.Order, error)

	// CancelOrder cancels an order
	CancelOrder(ctx context.Context, orderID string) error

	// CancelAllOrders cancels all orders for a symbol
	CancelAllOrders(ctx context.Context, symbol string) error

	// GetOrder retrieves order by ID
	GetOrder(ctx context.Context, orderID string) (*entity.Order, error)

	// GetOpenOrders retrieves all open orders
	GetOpenOrders(ctx context.Context, symbol string) ([]*entity.Order, error)

	// GetPosition retrieves current position
	GetPosition(ctx context.Context, symbol string) (*entity.Position, error)

	// GetTicker retrieves current ticker
	GetTicker(ctx context.Context, symbol string) (*entity.Ticker, error)

	// GetOrderBook retrieves order book
	GetOrderBook(ctx context.Context, symbol string, depth int) (*entity.OrderBook, error)

	// SubscribeTicker subscribes to ticker updates
	SubscribeTicker(ctx context.Context, symbol string, handler func(*entity.Ticker)) error

	// SubscribeOrderBook subscribes to order book updates
	SubscribeOrderBook(ctx context.Context, symbol string, handler func(*entity.OrderBook)) error

	// SubscribeOrders subscribes to order updates
	SubscribeOrders(ctx context.Context, handler func(*entity.Order)) error
}
