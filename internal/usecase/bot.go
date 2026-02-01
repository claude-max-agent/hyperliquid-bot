package usecase

import (
	"context"
	"fmt"
	"sync"

	"github.com/zono819/hyperliquid-bot/internal/adapter/gateway"
	"github.com/zono819/hyperliquid-bot/internal/domain/entity"
	"github.com/zono819/hyperliquid-bot/internal/domain/service"
)

// BotUseCase handles bot trading logic
type BotUseCase struct {
	exchange gateway.ExchangeGateway
	strategy service.Strategy
	symbol   string

	mu       sync.RWMutex
	running  bool
	position *entity.Position
	orders   []*entity.Order
}

// NewBotUseCase creates a new bot use case
func NewBotUseCase(exchange gateway.ExchangeGateway, strategy service.Strategy, symbol string) *BotUseCase {
	return &BotUseCase{
		exchange: exchange,
		strategy: strategy,
		symbol:   symbol,
	}
}

// Start starts the bot
func (b *BotUseCase) Start(ctx context.Context) error {
	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return fmt.Errorf("bot is already running")
	}
	b.running = true
	b.mu.Unlock()

	// Connect to exchange
	if err := b.exchange.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to exchange: %w", err)
	}

	// Subscribe to market data
	if err := b.subscribeMarketData(ctx); err != nil {
		return fmt.Errorf("failed to subscribe market data: %w", err)
	}

	return nil
}

// Stop stops the bot
func (b *BotUseCase) Stop(ctx context.Context) error {
	b.mu.Lock()
	if !b.running {
		b.mu.Unlock()
		return nil
	}
	b.running = false
	b.mu.Unlock()

	// Cancel all orders
	if err := b.exchange.CancelAllOrders(ctx, b.symbol); err != nil {
		return fmt.Errorf("failed to cancel orders: %w", err)
	}

	// Stop strategy
	if err := b.strategy.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop strategy: %w", err)
	}

	// Disconnect from exchange
	if err := b.exchange.Disconnect(ctx); err != nil {
		return fmt.Errorf("failed to disconnect: %w", err)
	}

	return nil
}

// IsRunning returns true if bot is running
func (b *BotUseCase) IsRunning() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.running
}

// subscribeMarketData subscribes to market data feeds
func (b *BotUseCase) subscribeMarketData(ctx context.Context) error {
	// Subscribe to ticker
	if err := b.exchange.SubscribeTicker(ctx, b.symbol, b.onTicker); err != nil {
		return err
	}

	// Subscribe to order updates
	if err := b.exchange.SubscribeOrders(ctx, b.onOrderUpdate); err != nil {
		return err
	}

	return nil
}

// onTicker handles ticker updates
func (b *BotUseCase) onTicker(ticker *entity.Ticker) {
	b.mu.RLock()
	if !b.running {
		b.mu.RUnlock()
		return
	}
	position := b.position
	orders := b.orders
	b.mu.RUnlock()

	ctx := context.Background()

	// Get current market state
	state := &service.MarketState{
		Ticker:   ticker,
		Position: position,
		Orders:   orders,
	}

	// Get signals from strategy
	signals, err := b.strategy.OnTick(ctx, state)
	if err != nil {
		// Log error
		return
	}

	// Execute signals
	for _, signal := range signals {
		b.executeSignal(ctx, signal)
	}
}

// onOrderUpdate handles order updates
func (b *BotUseCase) onOrderUpdate(order *entity.Order) {
	b.mu.Lock()
	// Update orders list
	found := false
	for i, o := range b.orders {
		if o.ID == order.ID {
			b.orders[i] = order
			found = true
			break
		}
	}
	if !found && order.Status == entity.OrderStatusOpen {
		b.orders = append(b.orders, order)
	}
	b.mu.Unlock()

	// Notify strategy
	ctx := context.Background()
	b.strategy.OnOrderUpdate(ctx, order)
}

// executeSignal executes a trading signal
func (b *BotUseCase) executeSignal(ctx context.Context, signal *service.Signal) {
	order := &entity.Order{
		Symbol:   signal.Symbol,
		Side:     signal.Side,
		Type:     entity.OrderTypeLimit,
		Price:    signal.Price,
		Quantity: signal.Quantity,
	}

	_, err := b.exchange.PlaceOrder(ctx, order)
	if err != nil {
		// Log error
		return
	}
}
