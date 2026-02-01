package entity

import (
	"time"
)

// Side represents order side (buy or sell)
type Side string

const (
	SideBuy  Side = "buy"
	SideSell Side = "sell"
)

// OrderType represents order type
type OrderType string

const (
	OrderTypeLimit  OrderType = "limit"
	OrderTypeMarket OrderType = "market"
)

// OrderStatus represents order status
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusOpen      OrderStatus = "open"
	OrderStatusFilled    OrderStatus = "filled"
	OrderStatusCanceled  OrderStatus = "canceled"
	OrderStatusRejected  OrderStatus = "rejected"
)

// Order represents a trading order (exchange-agnostic)
type Order struct {
	ID            string
	Symbol        string
	Side          Side
	Type          OrderType
	Price         float64
	Quantity      float64
	FilledQty     float64
	Status        OrderStatus
	ClientOrderID string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// IsFilled returns true if order is completely filled
func (o *Order) IsFilled() bool {
	return o.Status == OrderStatusFilled
}

// RemainingQty returns unfilled quantity
func (o *Order) RemainingQty() float64 {
	return o.Quantity - o.FilledQty
}
