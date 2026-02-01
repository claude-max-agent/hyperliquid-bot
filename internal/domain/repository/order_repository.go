package repository

import (
	"context"

	"github.com/zono819/hyperliquid-bot/internal/domain/entity"
)

// OrderRepository defines order data access interface
type OrderRepository interface {
	// Create creates a new order
	Create(ctx context.Context, order *entity.Order) error

	// GetByID retrieves order by ID
	GetByID(ctx context.Context, id string) (*entity.Order, error)

	// GetByClientOrderID retrieves order by client order ID
	GetByClientOrderID(ctx context.Context, clientOrderID string) (*entity.Order, error)

	// List retrieves orders with filters
	List(ctx context.Context, filter OrderFilter) ([]*entity.Order, error)

	// Update updates order
	Update(ctx context.Context, order *entity.Order) error

	// Delete deletes order
	Delete(ctx context.Context, id string) error
}

// OrderFilter represents filter for listing orders
type OrderFilter struct {
	Symbol string
	Status entity.OrderStatus
	Side   entity.Side
	Limit  int
}
