package entity

import (
	"time"
)

// Position represents a trading position (exchange-agnostic)
type Position struct {
	Symbol        string
	Side          Side
	Size          float64
	EntryPrice    float64
	MarkPrice     float64
	Leverage      float64
	UnrealizedPnL float64
	RealizedPnL   float64
	UpdatedAt     time.Time
}

// IsLong returns true if position is long
func (p *Position) IsLong() bool {
	return p.Side == SideBuy
}

// IsShort returns true if position is short
func (p *Position) IsShort() bool {
	return p.Side == SideSell
}

// Value returns position value
func (p *Position) Value() float64 {
	return p.Size * p.MarkPrice
}
